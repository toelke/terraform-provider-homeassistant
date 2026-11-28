package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// schedulerWeekdays are the values of a schedule's `weekdays`.
var schedulerWeekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun", "daily", "workday", "weekend"}

var (
	// schedulerFixedTime is a time of day as the card writes it.
	schedulerFixedTime = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`)
	// schedulerSunTime is a time relative to sunrise or sunset.
	schedulerSunTime = regexp.MustCompile(`^(sunrise|sunset)[+-]([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`)
	// jsonNumber is a number as JSON writes it, which the integration takes as a number.
	jsonNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
	// dottedID is an entity ID or an action, `<domain>.<name>`.
	dottedID = regexp.MustCompile(`^[a-z0-9_]+\.[a-z0-9_]+$`)
	// noOuterSpace is a string that does not start or end with whitespace. Scheduler strips the
	// name when it edits a schedule, but not when it creates one.
	noOuterSpace = regexp.MustCompile(`(?s)^(\S.*\S|\S)$`)
	// schedulerConditionType is the type of one entry of a timeslot's `conditions`.
	schedulerConditionType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"entity_id":  types.StringType,
		"attribute":  types.StringType,
		"match_type": types.StringType,
		"value":      types.StringType,
	}}
)

func schedulerTimeValidator() validator.String {
	return stringvalidator.Any(
		stringvalidator.RegexMatches(schedulerFixedTime, "must have the format HH:MM:SS, e.g. 06:30:00"),
		stringvalidator.RegexMatches(schedulerSunTime, "or be relative to the sun, e.g. sunrise+00:30:00 or sunset-01:00:00"),
	)
}

// dateValidator checks that a string is a date `YYYY-MM-DD`, the format the integration stores.
type dateValidator struct{}

func (dateValidator) Description(context.Context) string {
	return "must be a date YYYY-MM-DD, e.g. 2026-12-24"
}

func (v dateValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v dateValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.Parse(time.DateOnly, req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			fmt.Sprintf("%s %s, got %q.", req.Path, v.Description(ctx), req.ConfigValue.ValueString()))
	}
}

// repeatTypeValidator allows `repeat` and `pause`, and explains why `single` is not.
type repeatTypeValidator struct{}

func (repeatTypeValidator) Description(context.Context) string {
	return `must be "repeat" or "pause"`
}

func (v repeatTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v repeatTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	switch val := req.ConfigValue.ValueString(); val {
	case "repeat", "pause":
	case "single":
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			`repeat_type "single" is not supported: the integration deletes such a schedule after it ran, `+
				`and the next apply would create it again, so it would run again. Use "pause", which turns `+
				`the schedule off after it ran, or remove the resource.`)
	default:
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			fmt.Sprintf("repeat_type %s, got %q.", v.Description(ctx), val))
	}
}

// jsonObjectValidator checks that a string is a JSON object.
type jsonObjectValidator struct{}

func (jsonObjectValidator) Description(context.Context) string {
	return "must be a JSON object, e.g. jsonencode({ temperature = 21 })"
}

func (v jsonObjectValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonObjectValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &obj); err != nil || obj == nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			fmt.Sprintf("%s %s, got %q.", req.Path, v.Description(ctx), req.ConfigValue.ValueString()))
	}
}

// ValidateConfig checks what spans several attributes: that end_date is not before start_date,
// that conditions comparing with `below` or `above` have a number, and that timeslots with fixed
// times do not overlap.
func (r *schedulerScheduleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var start, end types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("start_date"), &start)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("end_date"), &end)...)
	if isSet(start) && isSet(end) && end.ValueString() < start.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("end_date"), "Invalid Attribute Value",
			fmt.Sprintf("end_date must not be before start_date, got %s and %s.", start.ValueString(), end.ValueString()))
	}

	var slots types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeslots"), &slots)...)
	if slots.IsNull() || slots.IsUnknown() {
		return
	}
	var parsed []slotTimes
	for i, e := range slots.Elements() {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		validateConditions(path.Root("timeslots").AtListIndex(i), obj, resp)
		if st, ok := newSlotTimes(i, obj); ok {
			parsed = append(parsed, st)
		}
	}
	for i := range parsed {
		for j := range i {
			if msg := parsed[i].overlap(parsed[j]); msg != "" {
				resp.Diagnostics.AddAttributeError(path.Root("timeslots").AtListIndex(parsed[i].index),
					"Overlapping timeslots", msg)
			}
		}
	}
}

func isSet(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }

// validateConditions checks that the conditions of the timeslot obj at p that compare with
// `below` or `above` compare with a number.
func validateConditions(p path.Path, obj types.Object, resp *resource.ValidateConfigResponse) {
	conds, ok := obj.Attributes()["conditions"].(types.List)
	if !ok || conds.IsNull() || conds.IsUnknown() {
		return
	}
	for i, e := range conds.Elements() {
		c, ok := e.(types.Object)
		if !ok || c.IsNull() || c.IsUnknown() {
			continue
		}
		mt, _ := c.Attributes()["match_type"].(types.String)
		val, _ := c.Attributes()["value"].(types.String)
		if !isSet(mt) || !isSet(val) || (mt.ValueString() != "below" && mt.ValueString() != "above") {
			continue
		}
		if !jsonNumber.MatchString(val.ValueString()) {
			resp.Diagnostics.AddAttributeError(p.AtName("conditions").AtListIndex(i).AtName("value"),
				"Invalid Attribute Value",
				fmt.Sprintf("value must be a number like 20 or -0.5 with match_type %q, got %q.", mt.ValueString(), val.ValueString()))
		}
	}
}

// slotTimes are the fixed times of a timeslot, as seconds of the day: a single time (point), or
// ranges. A range that ends at or before its start continues past midnight, and is split in two.
type slotTimes struct {
	index  int
	label  string
	point  int
	ranges [][2]int
}

// newSlotTimes returns the times of the timeslot obj at index i, and false if they are not all
// fixed times (relative to the sun, unknown, or invalid, which the attribute validators report).
func newSlotTimes(i int, obj types.Object) (slotTimes, bool) {
	start, _ := obj.Attributes()["start"].(types.String)
	stop, _ := obj.Attributes()["stop"].(types.String)
	if !isSet(start) || stop.IsUnknown() {
		return slotTimes{}, false
	}
	s, ok := secondsOfDay(start.ValueString())
	if !ok {
		return slotTimes{}, false
	}
	st := slotTimes{index: i, label: start.ValueString(), point: -1}
	if stop.IsNull() {
		st.point = s
		return st, true
	}
	e, ok := secondsOfDay(stop.ValueString())
	if !ok {
		return slotTimes{}, false
	}
	st.label += "-" + stop.ValueString()
	switch {
	case e == 0:
		st.ranges = [][2]int{{s, 24 * 3600}}
	case e > s:
		st.ranges = [][2]int{{s, e}}
	default:
		st.ranges = [][2]int{{s, 24 * 3600}, {0, e}}
	}
	return st, true
}

// overlap returns why a and b overlap, or "".
func (a slotTimes) overlap(b slotTimes) string {
	if a.point >= 0 && b.point >= 0 {
		if a.point == b.point {
			return fmt.Sprintf("timeslots %d and %d both start at %s.", b.index, a.index, a.label)
		}
		return ""
	}
	for _, x := range a.ranges {
		for _, y := range b.ranges {
			if x[0] < y[1] && y[0] < x[1] {
				return fmt.Sprintf("timeslot %d (%s) overlaps timeslot %d (%s).", a.index, a.label, b.index, b.label)
			}
		}
	}
	return ""
}

// secondsOfDay parses a fixed time HH:MM:SS.
func secondsOfDay(s string) (int, bool) {
	if !schedulerFixedTime.MatchString(s) {
		return 0, false
	}
	parts := strings.Split(s, ":")
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	sec, _ := strconv.Atoi(parts[2])
	return h*3600 + m*60 + sec, true
}
