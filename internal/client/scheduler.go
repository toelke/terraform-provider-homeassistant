package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

// ErrSchedulerMissing means Home Assistant has no Scheduler integration: it is not installed, or
// it has no config entry, so its commands and services do not exist.
var ErrSchedulerMissing = errors.New("the Scheduler integration is not set up in Home Assistant")

// SchedulerHint tells the user how to get the Scheduler integration, for diagnostics about
// ErrSchedulerMissing.
const SchedulerHint = "Install the Scheduler integration (https://github.com/nielsfaber/scheduler-component, " +
	"version 3.3, e.g. through HACS), restart Home Assistant, and add it under Settings > Devices & " +
	"services. A `homeassistant_integration` with `domain = \"scheduler\"` can add it, once it is installed."

// SchedulerSchedule is one schedule of the Scheduler integration, as `scheduler/item` returns it.
type SchedulerSchedule struct {
	ScheduleID string `json:"schedule_id"`
	// Name is null for a schedule created without one.
	Name *string `json:"name"`
	// EntityID is the schedule's switch, `switch.schedule_<slug of name>`.
	EntityID string `json:"entity_id"`
	// Weekdays holds `mon` … `sun`, `daily`, `workday`, or `weekend`.
	Weekdays  []string            `json:"weekdays"`
	StartDate *string             `json:"start_date"`
	EndDate   *string             `json:"end_date"`
	Timeslots []SchedulerTimeslot `json:"timeslots"`
	// RepeatType is `repeat`, `pause`, or `single`.
	RepeatType string   `json:"repeat_type"`
	Enabled    bool     `json:"enabled"`
	Tags       []string `json:"tags"`
}

// SchedulerTimeslot is a time, or a range of time, in which a schedule runs its actions.
type SchedulerTimeslot struct {
	Start           string               `json:"start"`
	Stop            *string              `json:"stop"`
	Conditions      []SchedulerCondition `json:"conditions"`
	ConditionType   *string              `json:"condition_type"`
	TrackConditions bool                 `json:"track_conditions"`
	Actions         []SchedulerAction    `json:"actions"`
}

// SchedulerAction is an action of a timeslot. The integration calls it a service.
type SchedulerAction struct {
	Service     string          `json:"service"`
	EntityID    *string         `json:"entity_id"`
	ServiceData json.RawMessage `json:"service_data"`
}

// SchedulerCondition is a condition of a timeslot.
type SchedulerCondition struct {
	EntityID  string  `json:"entity_id"`
	Attribute *string `json:"attribute"`
	// Value is a JSON string or number.
	Value     json.RawMessage `json:"value"`
	MatchType string          `json:"match_type"`
}

// Schedulers is the Scheduler integration's API: the WebSocket commands `scheduler` and
// `scheduler/item` to read, and the services `scheduler.add`, `scheduler.edit` and
// `scheduler.remove` to write. The services name a schedule by its switch's entity ID.
type Schedulers struct {
	ws WSCommander
}

// NewSchedulers returns the Scheduler API on ws.
func NewSchedulers(ws WSCommander) Schedulers {
	return Schedulers{ws: ws}
}

// schedulerAdds serializes Add: it finds the new schedule's ID by comparing the list before and
// after, so two adds from this process must not overlap.
var schedulerAdds sync.Mutex

// List returns every schedule.
func (s Schedulers) List(ctx context.Context) ([]SchedulerSchedule, error) {
	var all []SchedulerSchedule
	if err := s.command(ctx, "scheduler", nil, &all); err != nil {
		return nil, err
	}
	return all, nil
}

// Get returns the schedule with the ID id, and false if there is none.
func (s Schedulers) Get(ctx context.Context, id string) (SchedulerSchedule, bool, error) {
	var raw json.RawMessage
	if err := s.command(ctx, "scheduler/item", map[string]any{"schedule_id": id}, &raw); err != nil {
		return SchedulerSchedule{}, false, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return SchedulerSchedule{}, false, nil
	}
	var sched SchedulerSchedule
	if err := json.Unmarshal(raw, &sched); err != nil {
		return SchedulerSchedule{}, false, fmt.Errorf("scheduler/item: %w", err)
	}
	return sched, true, nil
}

// Add creates a schedule from fields, as `scheduler.add` takes them, and returns its ID. The
// service returns nothing, so Add compares the IDs before and after the call. A schedule that
// someone else creates at the same moment makes that ambiguous, which is an error.
//
// Once the service has succeeded, the schedule exists: Add retries the list a few times, and if it
// still cannot tell the ID, the error says how to find and import the schedule. It never deletes
// a schedule it cannot identify.
func (s Schedulers) Add(ctx context.Context, fields map[string]any) (string, error) {
	schedulerAdds.Lock()
	defer schedulerAdds.Unlock()

	before, err := s.ids(ctx)
	if err != nil {
		return "", err
	}
	if err := s.call(ctx, "add", fields); err != nil {
		return s.reconcileFailedAdd(ctx, fields, before, err)
	}
	var after map[string]bool
	for attempt := range schedulerListAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(schedulerListRetryDelay):
			}
		}
		if after, err = s.ids(ctx); err == nil || ctx.Err() != nil {
			break
		}
	}
	if err != nil {
		return "", addedUnidentified(fields, true, fmt.Errorf("listing the schedules failed: %w", err))
	}
	added := newIDs(before, after)
	switch len(added) {
	case 1:
		return added[0], nil
	case 0:
		return "", addedUnidentified(fields, true, errors.New("no new schedule appeared in the list"))
	default:
		return "", addedUnidentified(fields, true, fmt.Errorf("several schedules appeared at once (%s)",
			strings.Join(added, ", ")))
	}
}

// reconcileFailedAdd handles an error of the `scheduler.add` call itself. Home Assistant's own
// rejection, a missing integration, or a connection that never came up mean nothing was created.
// Otherwise, e.g. after a timeout or a lost connection, the schedule may exist: it lists once more,
// with a context that the caller's cancel does not end, and returns the one new ID if there is
// one.
func (s Schedulers) reconcileFailedAdd(ctx context.Context, fields map[string]any, before map[string]bool, err error) (string, error) {
	var wsErr *WSError
	if errors.As(err, &wsErr) || errors.Is(err, ErrSchedulerMissing) || errors.Is(err, ErrUnreachable) ||
		errors.Is(err, ErrUnauthorized) {
		return "", err
	}
	lctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), schedulerReconcileTimeout)
	defer cancel()
	after, listErr := s.ids(lctx)
	if listErr == nil {
		switch added := newIDs(before, after); len(added) {
		case 0:
			return "", err // not created
		case 1:
			return added[0], nil
		}
	}
	return "", addedUnidentified(fields, false, err)
}

// schedulerReconcileTimeout bounds the list after a failed add.
var schedulerReconcileTimeout = 30 * time.Second

// newIDs returns the IDs in after that are not in before, sorted.
func newIDs(before, after map[string]bool) []string {
	var added []string
	for id := range after {
		if !before[id] {
			added = append(added, id)
		}
	}
	slices.Sort(added)
	return added
}

// Retries of the list after `scheduler.add`. Tests shorten the delay.
var (
	schedulerListAttempts   = 3
	schedulerListRetryDelay = time.Second
)

// addedUnidentified is the error of an add that created, or if not certain may have created, a
// schedule without the provider learning its ID.
func addedUnidentified(fields map[string]any, certain bool, cause error) error {
	verb := "created"
	if !certain {
		verb = "may have created"
	}
	return fmt.Errorf("scheduler.add %s the schedule %q, but the provider could not tell its "+
		"schedule_id: %w. Look it up, e.g. with the data source homeassistant_scheduler_schedules, and "+
		"import it with `tofu import <resource address> <schedule_id>`, or delete it in Home Assistant, "+
		"before applying again; otherwise the next apply creates it a second time", verb, fields["name"], cause)
}

// Edit sets the fields of the schedule whose switch is entityID, as `scheduler.edit` takes them.
func (s Schedulers) Edit(ctx context.Context, entityID string, fields map[string]any) error {
	data := make(map[string]any, len(fields)+1)
	maps.Copy(data, fields)
	data["entity_id"] = entityID
	return s.call(ctx, "edit", data)
}

// Remove deletes the schedule whose switch is entityID.
func (s Schedulers) Remove(ctx context.Context, entityID string) error {
	return s.call(ctx, "remove", map[string]any{"entity_id": entityID})
}

// SetEnabled turns the schedule's switch on or off. The integration stores this as the schedule's
// `enabled`.
func (s Schedulers) SetEnabled(ctx context.Context, entityID string, enabled bool) error {
	service := "turn_off"
	if enabled {
		service = "turn_on"
	}
	return CallService(ctx, s.ws, "switch", service, map[string]any{"entity_id": entityID})
}

func (s Schedulers) ids(ctx context.Context) (map[string]bool, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(all))
	for _, sched := range all {
		ids[sched.ScheduleID] = true
	}
	return ids, nil
}

func (s Schedulers) command(ctx context.Context, typ string, params map[string]any, result any) error {
	return schedulerError(s.ws.Command(ctx, typ, params, result))
}

func (s Schedulers) call(ctx context.Context, service string, data map[string]any) error {
	return schedulerError(CallService(ctx, s.ws, "scheduler", service, data))
}

// schedulerError turns HA's errors for a missing command or service into ErrSchedulerMissing.
func schedulerError(err error) error {
	var wsErr *WSError
	if errors.As(err, &wsErr) && (wsErr.Code == "unknown_command" || wsErr.Code == "not_found" && isServiceNotFound(wsErr)) {
		return fmt.Errorf("%w (%s)", ErrSchedulerMissing, wsErr.Message)
	}
	return err
}

// isServiceNotFound reports whether a `not_found` error of call_service is about the service
// itself, rather than e.g. an entity.
func isServiceNotFound(err *WSError) bool {
	m := strings.ToLower(err.Message)
	return strings.Contains(m, "service scheduler.") || strings.Contains(m, "action scheduler.")
}

// CallService calls the action (service) domain.service with data and waits until it is done
// (`call_service`, which blocks by default).
func CallService(ctx context.Context, ws WSCommander, domain, service string, data map[string]any) error {
	return ws.Command(ctx, "call_service", map[string]any{
		"domain":       domain,
		"service":      service,
		"service_data": data,
	}, nil)
}
