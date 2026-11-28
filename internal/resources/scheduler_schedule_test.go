package resources

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

// schedulerItem is a schedule as Scheduler 3.3.9 returns it from `scheduler/item`.
const schedulerItem = `{"schedule_id":"3d22e6","weekdays":["mon","tue"],"start_date":null,"end_date":"2026-12-31",
"timeslots":[{"start":"06:30:00","stop":"08:00:00","conditions":[],"condition_type":null,"track_conditions":false,
"actions":[{"service":"climate.set_temperature","entity_id":"climate.office","service_data":{"temperature":21}}]},
{"start":"sunset-00:30:00","stop":null,"conditions":[{"entity_id":"sensor.outside","attribute":null,"value":20,"match_type":"above"},
{"entity_id":"binary_sensor.window","attribute":"device_class","value":"window","match_type":"is"}],
"condition_type":"and","track_conditions":true,
"actions":[{"service":"light.turn_on","entity_id":null,"service_data":{}}]}],
"repeat_type":"repeat","name":"Office","enabled":false,"next_entries":[0,1],
"timestamps":["2026-11-30T06:30:00+00:00","2026-11-30T17:21:00+00:00"],"entity_id":"switch.schedule_office","tags":["a","b"]}`

func parseSchedulerItem(t *testing.T) client.SchedulerSchedule {
	t.Helper()
	var s client.SchedulerSchedule
	if err := json.Unmarshal([]byte(schedulerItem), &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSchedulerScheduleModelRoundTrip(t *testing.T) {
	s := parseSchedulerItem(t)
	m := newSchedulerScheduleModel(s, types.BoolNull())
	if !m.Enabled.IsNull() {
		t.Errorf("enabled = %v, want null when unmanaged", m.Enabled)
	}
	if got := newSchedulerScheduleModel(s, types.BoolValue(true)).Enabled; got != types.BoolValue(false) {
		t.Errorf("managed enabled = %v, want false", got)
	}
	ts := m.Timeslots[1]
	if ts.Stop.IsNull() != true || ts.ConditionType.ValueString() != "and" || !ts.TrackConditions.ValueBool() {
		t.Errorf("timeslot = %+v", ts)
	}
	if c := ts.Conditions[0]; c.Value.ValueString() != "20" || !c.Attribute.IsNull() {
		t.Errorf("number condition = %+v", c)
	}
	if c := ts.Conditions[1]; c.Value.ValueString() != "window" || c.Attribute.ValueString() != "device_class" {
		t.Errorf("string condition = %+v", c)
	}
	if a := ts.Actions[0]; !a.EntityID.IsNull() || a.Data.ValueString() != "{}" {
		t.Errorf("action = %+v", a)
	}

	fields, err := m.fields()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(fields)
	want := `{"end_date":"2026-12-31","name":"Office","repeat_type":"repeat","start_date":null,"tags":["a","b"],` +
		`"timeslots":[{"actions":[{"entity_id":"climate.office","service":"climate.set_temperature","service_data":{"temperature":21}}],` +
		`"start":"06:30:00","stop":"08:00:00","track_conditions":false},` +
		`{"actions":[{"service":"light.turn_on","service_data":{}}],"condition_type":"and",` +
		`"conditions":[{"entity_id":"sensor.outside","match_type":"above","value":20},` +
		`{"attribute":"device_class","entity_id":"binary_sensor.window","match_type":"is","value":"window"}],` +
		`"start":"sunset-00:30:00","track_conditions":true}],"weekdays":["mon","tue"]}`
	if string(got) != want {
		t.Errorf("fields =\n%s\nwant\n%s", got, want)
	}
	if !sameFields(fields, m) {
		t.Error("sameFields(fields of m, m) = false")
	}
}

func TestSchedulerSameFieldsIgnoresJSONSpelling(t *testing.T) {
	m := newSchedulerScheduleModel(parseSchedulerItem(t), types.BoolNull())
	written, _ := m.fields()
	m.Timeslots[0].Actions[0].Data = dyntype.NewJSONString(`{ "temperature": 21.0 }`)
	if !sameFields(written, m) {
		t.Error("data spelled differently should be the same")
	}
	m.Weekdays = []string{"daily"}
	if sameFields(written, m) {
		t.Error("other weekdays should differ")
	}
}

func TestConditionValue(t *testing.T) {
	for _, tc := range []struct {
		matchType, value string
		want             any
	}{
		{"above", "20", json.Number("20")},
		{"below", "-1.5", json.Number("-1.5")},
		{"is", "20", "20"},
		{"not", "off", "off"},
	} {
		if got := conditionValue(tc.matchType, tc.value); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("conditionValue(%q, %q) = %#v, want %#v", tc.matchType, tc.value, got, tc.want)
		}
	}
}

// validateScheduler runs ValidateConfig on the configuration m.
func validateScheduler(t *testing.T, m schedulerScheduleModel) resource.ValidateConfigResponse {
	t.Helper()
	r := &schedulerScheduleResource{}
	state := stateOf(t, r, m)
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(t.Context(), resource.ValidateConfigRequest{Config: tfsdk.Config(state)}, &resp)
	return resp
}

func slot(start, stop string) schedulerTimeslotModel {
	s := schedulerTimeslotModel{
		Start:           types.StringValue(start),
		Stop:            types.StringNull(),
		ConditionType:   types.StringNull(),
		TrackConditions: types.BoolValue(false),
		Conditions:      []schedulerConditionModel{},
		Actions: []schedulerActionModel{{
			Action: types.StringValue("light.turn_on"), EntityID: types.StringNull(), Data: dyntype.NewJSONString("{}"),
		}},
	}
	if stop != "" {
		s.Stop = types.StringValue(stop)
	}
	return s
}

func TestValidateSchedulerSchedule(t *testing.T) {
	base := func(slots ...schedulerTimeslotModel) schedulerScheduleModel {
		return schedulerScheduleModel{
			ScheduleID: types.StringNull(), EntityID: types.StringNull(), Name: types.StringValue("x"),
			Weekdays: []string{"daily"}, StartDate: types.StringNull(), EndDate: types.StringNull(),
			RepeatType: types.StringValue("repeat"), Tags: []string{}, Enabled: types.BoolNull(), Timeslots: slots,
			Timeouts: timeouts.Value{Object: types.ObjectNull(schedulerTimeoutsType)},
		}
	}
	cond := func(mt, v string) []schedulerConditionModel {
		return []schedulerConditionModel{{
			EntityID: types.StringValue("sensor.t"), Attribute: types.StringNull(),
			MatchType: types.StringValue(mt), Value: types.StringValue(v),
		}}
	}
	withCond := func(s schedulerTimeslotModel, c []schedulerConditionModel) schedulerTimeslotModel {
		s.Conditions = c
		return s
	}
	dates := func(m schedulerScheduleModel, start, end string) schedulerScheduleModel {
		m.StartDate, m.EndDate = types.StringValue(start), types.StringValue(end)
		return m
	}
	for _, tc := range []struct {
		name string
		m    schedulerScheduleModel
		want string
	}{
		{"ranges that meet", base(slot("06:00:00", "08:00:00"), slot("08:00:00", "00:00:00")), ""},
		{"points", base(slot("06:00:00", ""), slot("07:00:00", "")), ""},
		{"sun relative skipped", base(slot("sunrise+00:00:00", "09:00:00"), slot("08:00:00", "10:00:00")), ""},
		{"overlap", base(slot("06:00:00", "09:00:00"), slot("08:00:00", "10:00:00")), "timeslot 1 (08:00:00-10:00:00) overlaps timeslot 0"},
		{"point in end-of-day range", base(slot("22:00:00", "00:00:00"), slot("23:00:00", "")), "lies within"},
		{"past midnight overlap", base(slot("22:00:00", "02:00:00"), slot("01:00:00", "03:00:00")), "overlaps"},
		{"same point", base(slot("06:00:00", ""), slot("06:00:00", "")), "both start at 06:00:00"},
		{"point in range", base(slot("06:00:00", "08:00:00"), slot("06:30:00", "")), "timeslot 1 (06:30:00) lies within timeslot 0 (06:00:00-08:00:00)"},
		{"point at range start", base(slot("06:30:00", ""), slot("06:30:00", "08:00:00")), "lies within"},
		{"point at range end", base(slot("06:00:00", "08:00:00"), slot("08:00:00", "")), ""},
		{"numeric condition", base(withCond(slot("06:00:00", ""), cond("above", "20.5"))), ""},
		{"text condition", base(withCond(slot("06:00:00", ""), cond("is", "on"))), ""},
		{"non-numeric below", base(withCond(slot("06:00:00", ""), cond("below", "warm"))), `value must be a number like 20 or -0.5 with match_type "below"`},
		{"no leading digit", base(withCond(slot("06:00:00", ""), cond("below", ".5"))), `value must be a number`},
		{"NaN", base(withCond(slot("06:00:00", ""), cond("above", "NaN"))), `value must be a number`},
		{"negative fraction", base(withCond(slot("06:00:00", ""), cond("above", "-0.5"))), ""},
		{"dates in order", dates(base(slot("06:00:00", "")), "2026-01-01", "2026-01-01"), ""},
		{"end before start", dates(base(slot("06:00:00", "")), "2026-02-01", "2026-01-31"), "end_date must not be before start_date"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkDiags(t, validateScheduler(t, tc.m).Diagnostics, tc.want)
		})
	}
}

func TestSchedulerAttributeValidators(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	(&schedulerScheduleResource{}).Schema(ctx, resource.SchemaRequest{}, &resp)
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatal(diags)
	}
	timeV := schedulerTimeValidator()
	for v, ok := range map[string]bool{
		"06:30:00": true, "23:59:59": true, "sunrise+00:30:00": true, "sunset-01:00:00": true,
		"6:30:00": false, "06:30": false, "24:00:00": false, "noon+00:00:00": false, "sunset+1:00:00": false,
	} {
		if got := !validateString(t, timeV, v).HasError(); got != ok {
			t.Errorf("time %q valid = %v, want %v", v, got, ok)
		}
	}
	for v, ok := range map[string]bool{"2026-12-24": true, "2026-2-1": false, "2026-02-30": false, "tomorrow": false} {
		if got := !validateString(t, dateValidator{}, v).HasError(); got != ok {
			t.Errorf("date %q valid = %v, want %v", v, got, ok)
		}
	}
	for v, ok := range map[string]bool{`{}`: true, `{"a":[1]}`: true, `[]`: false, `null`: false, `1`: false, `{`: false} {
		if got := !validateString(t, jsonObjectValidator{}, v).HasError(); got != ok {
			t.Errorf("data %q valid = %v, want %v", v, got, ok)
		}
	}
	single := validateString(t, repeatTypeValidator{}, "single")
	if !single.HasError() || !strings.Contains(single.Errors()[0].Detail(), "pause") {
		t.Errorf("single: %v", single)
	}
	for _, v := range []string{"repeat", "pause"} {
		if d := validateString(t, repeatTypeValidator{}, v); d.HasError() {
			t.Errorf("%s: %v", v, d)
		}
	}
	for v, ok := range map[string]bool{"Office": true, "a": true, " Office": false, "Office ": false} {
		if got := noOuterSpace.MatchString(v); got != ok {
			t.Errorf("name %q valid = %v, want %v", v, got, ok)
		}
	}
}

// fakeScheduler serves the Scheduler integration's commands and services from memory, or, if
// missing, answers like Home Assistant without the integration.
type fakeScheduler struct {
	missing   bool
	schedules map[string]json.RawMessage
	calls     []string
}

func (f *fakeScheduler) Command(_ context.Context, typ string, params map[string]any, result any) error {
	if f.missing && typ != "call_service" {
		return &client.WSError{Code: "unknown_command", Message: "Unknown command."}
	}
	reply := func(v any) error {
		b, _ := json.Marshal(v)
		return json.Unmarshal(b, result)
	}
	switch typ {
	case "scheduler":
		var all []json.RawMessage
		for _, s := range f.schedules {
			all = append(all, s)
		}
		return reply(all)
	case "scheduler/item":
		return reply(f.schedules[params["schedule_id"].(string)])
	case "call_service":
		f.calls = append(f.calls, params["domain"].(string)+"."+params["service"].(string))
		if f.missing {
			return &client.WSError{Code: "not_found", Message: "Service scheduler.remove not found."}
		}
		data, _ := params["service_data"].(map[string]any)
		switch params["service"] {
		case "remove":
			for id := range f.schedules {
				delete(f.schedules, id)
			}
		case "edit", "turn_on", "turn_off":
			for id, raw := range f.schedules {
				var item map[string]any
				_ = json.Unmarshal(raw, &item)
				if params["service"] == "edit" {
					b, _ := json.Marshal(data)
					_ = json.Unmarshal(b, &item) // through JSON, as Home Assistant gets it
					item["entity_id"] = data["entity_id"]
				} else {
					item["enabled"] = params["service"] == "turn_on"
				}
				f.schedules[id], _ = json.Marshal(item)
			}
		}
		return nil
	}
	return &client.WSError{Code: "unknown_command", Message: "Unknown command."}
}

func schedulerState(t *testing.T) tfsdk.State {
	t.Helper()
	m := newSchedulerScheduleModel(parseSchedulerItem(t), types.BoolNull())
	return stateOf(t, &schedulerScheduleResource{}, m)
}

func TestSchedulerScheduleRead(t *testing.T) {
	t.Run("missing integration", func(t *testing.T) {
		state := schedulerState(t)
		resp := resource.ReadResponse{State: state}
		newSchedulerScheduleResource(&fakeScheduler{missing: true}).Read(t.Context(), resource.ReadRequest{State: state}, &resp)
		errs := resp.Diagnostics.Errors()
		if len(errs) != 1 || errs[0].Summary() != "Scheduler integration missing" ||
			!strings.Contains(errs[0].Detail(), "scheduler-component") {
			t.Errorf("diagnostics = %v", resp.Diagnostics)
		}
	})
	t.Run("gone", func(t *testing.T) {
		state := schedulerState(t)
		resp := resource.ReadResponse{State: state}
		newSchedulerScheduleResource(&fakeScheduler{}).Read(t.Context(), resource.ReadRequest{State: state}, &resp)
		if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
			t.Errorf("diagnostics = %v, state = %v", resp.Diagnostics, resp.State.Raw)
		}
	})
	t.Run("found", func(t *testing.T) {
		state := schedulerState(t)
		resp := resource.ReadResponse{State: state}
		f := &fakeScheduler{schedules: map[string]json.RawMessage{"3d22e6": json.RawMessage(schedulerItem)}}
		newSchedulerScheduleResource(f).Read(t.Context(), resource.ReadRequest{State: state}, &resp)
		var m schedulerScheduleModel
		resp.Diagnostics.Append(resp.State.Get(t.Context(), &m)...)
		if resp.Diagnostics.HasError() || m.EntityID.ValueString() != "switch.schedule_office" {
			t.Errorf("diagnostics = %v, model = %+v", resp.Diagnostics, m)
		}
	})
}

func TestSchedulerScheduleDelete(t *testing.T) {
	f := &fakeScheduler{schedules: map[string]json.RawMessage{"3d22e6": json.RawMessage(schedulerItem)}}
	state := schedulerState(t)
	var resp resource.DeleteResponse
	newSchedulerScheduleResource(f).Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() || !reflect.DeepEqual(f.calls, []string{"scheduler.remove"}) {
		t.Errorf("diagnostics = %v, calls = %v", resp.Diagnostics, f.calls)
	}
	// Already gone: nothing to do.
	f.calls = nil
	newSchedulerScheduleResource(f).Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() || len(f.calls) != 0 {
		t.Errorf("diagnostics = %v, calls = %v", resp.Diagnostics, f.calls)
	}
}

// validateString runs v on the configured value s.
func validateString(t *testing.T, v validator.String, s string) diag.Diagnostics {
	t.Helper()
	var resp validator.StringResponse
	v.ValidateString(t.Context(), validator.StringRequest{Path: path.Root("x"), ConfigValue: types.StringValue(s)}, &resp)
	return resp.Diagnostics
}

func TestKeepNumberSpelling(t *testing.T) {
	withValue := func(matchType, v string) []schedulerTimeslotModel {
		s := slot("06:00:00", "")
		s.Conditions = []schedulerConditionModel{{
			EntityID: types.StringValue("sensor.t"), Attribute: types.StringNull(),
			MatchType: types.StringValue(matchType), Value: types.StringValue(v),
		}}
		return []schedulerTimeslotModel{s}
	}
	for _, tc := range []struct{ readType, read, priorType, prior, want string }{
		{"above", "20.5", "above", "20.50", "20.50"}, // same number: keep how it was written
		{"below", "1000.0", "below", "1e3", "1e3"},
		{"above", "21", "above", "20.50", "21"}, // another number: drift
		// is and not compare strings, so another spelling is drift.
		{"is", "20.5", "is", "20.50", "20.5"},
		{"not", "20.5", "not", "20.50", "20.5"},
		// The match type changed too: drift either way.
		{"is", "20.5", "above", "20.50", "20.5"},
		{"above", "20.5", "is", "20.50", "20.5"},
	} {
		read := withValue(tc.readType, tc.read)
		keepNumberSpelling(read, withValue(tc.priorType, tc.prior))
		if got := read[0].Conditions[0].Value.ValueString(); got != tc.want {
			t.Errorf("read %s %q, prior %s %q: got %q, want %q", tc.readType, tc.read, tc.priorType, tc.prior, got, tc.want)
		}
	}
}

// updateScheduler runs Update from the stored schedule to the plan that change makes of it, and
// returns the services called.
func updateScheduler(t *testing.T, change func(*schedulerScheduleModel)) []string {
	t.Helper()
	f := &fakeScheduler{schedules: map[string]json.RawMessage{"3d22e6": json.RawMessage(schedulerItem)}}
	r := &schedulerScheduleResource{}
	prior := newSchedulerScheduleModel(parseSchedulerItem(t), types.BoolNull())
	planned := newSchedulerScheduleModel(parseSchedulerItem(t), types.BoolNull())
	change(&planned)
	state, plan := stateOf(t, r, prior), stateOf(t, r, planned)
	resp := resource.UpdateResponse{State: state}
	newSchedulerScheduleResource(f).Update(t.Context(), resource.UpdateRequest{
		State: state, Plan: tfsdk.Plan(plan),
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return f.calls
}

func TestSchedulerScheduleUpdate(t *testing.T) {
	// Every edit runs the active timeslot's actions again, so enabled alone must not edit.
	got := updateScheduler(t, func(m *schedulerScheduleModel) { m.Enabled = types.BoolValue(true) })
	if !reflect.DeepEqual(got, []string{"switch.turn_on"}) {
		t.Errorf("enabled only: calls = %v", got)
	}
	// The same days in another order are no change either.
	got = updateScheduler(t, func(m *schedulerScheduleModel) { m.Weekdays = []string{"tue", "mon"} })
	if len(got) != 0 {
		t.Errorf("weekdays reordered: calls = %v", got)
	}
	got = updateScheduler(t, func(m *schedulerScheduleModel) { m.Weekdays = []string{"daily"} })
	if !reflect.DeepEqual(got, []string{"scheduler.edit"}) {
		t.Errorf("weekdays changed: calls = %v", got)
	}
}

func TestSchedulerScheduleModifyPlanWarnsWithoutIntegration(t *testing.T) {
	r := &schedulerScheduleResource{}
	m := newSchedulerScheduleModel(parseSchedulerItem(t), types.BoolNull())
	plan := stateOf(t, r, m)
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan(plan)}
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Schema: plan.Schema, Raw: tftypes.NewValue(plan.Raw.Type(), nil)},
		Plan:  tfsdk.Plan(plan),
	}
	newSchedulerScheduleResource(&fakeScheduler{missing: true}).ModifyPlan(t.Context(), req, &resp)
	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 ||
		!strings.Contains(resp.Diagnostics.Warnings()[0].Detail(), "homeassistant_integration") {
		t.Errorf("diagnostics = %v", resp.Diagnostics)
	}
}
