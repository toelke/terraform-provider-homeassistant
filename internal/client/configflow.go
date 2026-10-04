package client

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// FlowStep is the input for one form of a config flow, keyed by the form's step_id.
type FlowStep struct {
	StepID string
	Data   map[string]any
	// SensitiveData is merged over Data when the step is submitted (ADR-0014).
	SensitiveData map[string]any
}

// input returns the data submitted for the step: Data ∪ SensitiveData, SensitiveData winning on a
// shared key.
func (s FlowStep) input() map[string]any {
	in := make(map[string]any, len(s.Data)+len(s.SensitiveData))
	maps.Copy(in, s.Data)
	maps.Copy(in, s.SensitiveData)
	return in
}

// ReasonReconfigureSuccessful is the abort reason that ends a successful reconfigure flow.
const ReasonReconfigureSuccessful = "reconfigure_successful"

// FlowAbortedError means Home Assistant aborted a config flow, e.g. `already_configured`.
type FlowAbortedError struct {
	Handler string
	Reason  string
}

func (e *FlowAbortedError) Error() string {
	return fmt.Sprintf("config flow for %s aborted: %s", e.Handler, e.Reason)
}

// FlowFormError means Home Assistant rejected the data of a step, e.g. `{"base": "cannot_connect"}`.
type FlowFormError struct {
	Handler string
	StepID  string
	Errors  map[string]string
}

func (e *FlowFormError) Error() string {
	fields := slices.Sorted(maps.Keys(e.Errors))
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = f + ": " + e.Errors[f]
	}
	return fmt.Sprintf("config flow for %s rejected step %q: %s", e.Handler, e.StepID, strings.Join(parts, ", "))
}

// FlowStepMissingError means Home Assistant asked for a step the config does not provide.
type FlowStepMissingError struct {
	Handler string
	StepID  string
	// Provided are the step_ids the config provides.
	Provided []string
	// Repeated means the step was provided but HA asked for it a second time.
	Repeated bool
}

func (e *FlowStepMissingError) Error() string {
	if e.Repeated {
		return fmt.Sprintf("config flow for %s asked for step %q a second time", e.Handler, e.StepID)
	}
	return fmt.Sprintf("config flow for %s asked for step %q, but the config only has %s",
		e.Handler, e.StepID, quoteList(e.Provided))
}

// FlowUnsupportedError means the flow returned a result the driver cannot handle, e.g. a menu,
// an external step, or a progress step.
type FlowUnsupportedError struct {
	Handler string
	Type    string
	StepID  string
}

func (e *FlowUnsupportedError) Error() string {
	return fmt.Sprintf("config flow for %s returned unsupported result %q (step %q)", e.Handler, e.Type, e.StepID)
}

// flowResult is the subset of a FlowResult the driver reads.
type flowResult struct {
	Type   string            `json:"type"`
	FlowID string            `json:"flow_id"`
	StepID string            `json:"step_id"`
	Errors map[string]string `json:"errors"`
	Reason string            `json:"reason"`
	Result *struct {
		EntryID string `json:"entry_id"`
	} `json:"result"`
}

// ConfigFlows drives config flows over REST (`/api/config/config_entries/flow`, ADR-0004).
type ConfigFlows struct {
	rest *RESTClient
}

// NewConfigFlows returns the config-flow API on rest.
func NewConfigFlows(rest *RESTClient) ConfigFlows {
	return ConfigFlows{rest: rest}
}

const flowPath = "config/config_entries/flow"

// Run starts a config flow for handler, answers each form with the step of the same step_id, and
// returns the entry_id of the config entry.
//
// A non-empty entryID starts a reconfigure flow for that entry instead; it returns entryID once HA
// aborts with ReasonReconfigureSuccessful.
//
// A missing step, form errors, or an unsupported result abort the flow (`DELETE`) before the error
// is returned. Each step is submitted at most once, so a flow that asks for a step again fails
// instead of looping.
func (f ConfigFlows) Run(ctx context.Context, handler, entryID string, steps []FlowStep) (string, error) {
	byID := make(map[string]FlowStep, len(steps))
	for _, s := range steps {
		byID[s.StepID] = s
	}

	init := map[string]any{"handler": handler, "show_advanced_options": true}
	if entryID != "" {
		init["entry_id"] = entryID
	}
	var res flowResult
	if err := f.rest.Do(ctx, http.MethodPost, flowPath, init, &res); err != nil {
		return "", fmt.Errorf("starting config flow for %s: %w", handler, err)
	}

	submitted := map[string]bool{}
	for {
		switch res.Type {
		case "create_entry":
			if res.Result == nil || res.Result.EntryID == "" {
				return "", fmt.Errorf("config flow for %s created an entry without entry_id", handler)
			}
			return res.Result.EntryID, nil

		case "abort":
			if entryID != "" && res.Reason == ReasonReconfigureSuccessful {
				return entryID, nil
			}
			return "", &FlowAbortedError{Handler: handler, Reason: res.Reason}

		case "form":
			if len(res.Errors) > 0 {
				return "", f.abort(ctx, res.FlowID, &FlowFormError{Handler: handler, StepID: res.StepID, Errors: res.Errors})
			}
			step, ok := byID[res.StepID]
			if !ok || submitted[res.StepID] {
				return "", f.abort(ctx, res.FlowID, &FlowStepMissingError{
					Handler: handler, StepID: res.StepID, Provided: slices.Sorted(maps.Keys(byID)), Repeated: ok,
				})
			}
			submitted[res.StepID] = true
			flowID := res.FlowID
			res = flowResult{}
			if err := f.rest.Do(ctx, http.MethodPost, flowPath+"/"+flowID, step.input(), &res); err != nil {
				return "", f.abort(ctx, flowID, fmt.Errorf("submitting step %q of config flow for %s: %w", step.StepID, handler, err))
			}

		default:
			return "", f.abort(ctx, res.FlowID, &FlowUnsupportedError{Handler: handler, Type: res.Type, StepID: res.StepID})
		}
	}
}

// abort deletes the flow and returns cause, joined with the abort's own failure if any. It runs
// even if ctx is cancelled, so a flow is not left open in HA.
func (f ConfigFlows) abort(ctx context.Context, flowID string, cause error) error {
	if flowID == "" {
		return cause
	}
	err := f.rest.Do(context.WithoutCancel(ctx), http.MethodDelete, flowPath+"/"+flowID, nil, nil)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return errors.Join(cause, fmt.Errorf("aborting config flow %s: %w", flowID, err))
	}
	return cause
}

func quoteList(ss []string) string {
	if len(ss) == 0 {
		return "none"
	}
	q := make([]string, len(ss))
	for i, s := range ss {
		q[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(q, ", ")
}
