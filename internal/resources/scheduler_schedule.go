package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

var (
	_ resource.ResourceWithConfigure      = (*schedulerScheduleResource)(nil)
	_ resource.ResourceWithImportState    = (*schedulerScheduleResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*schedulerScheduleResource)(nil)
	_ resource.ResourceWithValidateConfig = (*schedulerScheduleResource)(nil)
)

// defaultSchedulerWriteTimeout bounds the wait for a schedule to read back as written, unless
// `timeouts` says otherwise: the integration refreshes its switch, which serves the reads, in a
// task of its own after each write.
const defaultSchedulerWriteTimeout = 60 * time.Second

// schedulerTimeoutsType is the type of `timeouts`.
var schedulerTimeoutsType = map[string]attr.Type{"create": types.StringType, "update": types.StringType}

// NewSchedulerSchedule returns the `homeassistant_scheduler_schedule` resource.
func NewSchedulerSchedule() resource.Resource {
	return &schedulerScheduleResource{}
}

type schedulerScheduleResource struct {
	schedulers client.Schedulers
	configured bool
}

func newSchedulerScheduleResource(ws client.WSCommander) *schedulerScheduleResource {
	return &schedulerScheduleResource{schedulers: client.NewSchedulers(ws), configured: true}
}

type schedulerScheduleModel struct {
	ScheduleID types.String             `tfsdk:"schedule_id"`
	EntityID   types.String             `tfsdk:"entity_id"`
	Name       types.String             `tfsdk:"name"`
	Weekdays   []string                 `tfsdk:"weekdays"`
	StartDate  types.String             `tfsdk:"start_date"`
	EndDate    types.String             `tfsdk:"end_date"`
	RepeatType types.String             `tfsdk:"repeat_type"`
	Tags       []string                 `tfsdk:"tags"`
	Enabled    types.Bool               `tfsdk:"enabled"`
	Timeouts   timeouts.Value           `tfsdk:"timeouts"`
	Timeslots  []schedulerTimeslotModel `tfsdk:"timeslots"`
}

type schedulerTimeslotModel struct {
	Start           types.String              `tfsdk:"start"`
	Stop            types.String              `tfsdk:"stop"`
	Actions         []schedulerActionModel    `tfsdk:"actions"`
	Conditions      []schedulerConditionModel `tfsdk:"conditions"`
	ConditionType   types.String              `tfsdk:"condition_type"`
	TrackConditions types.Bool                `tfsdk:"track_conditions"`
}

type schedulerActionModel struct {
	Action   types.String       `tfsdk:"action"`
	EntityID types.String       `tfsdk:"entity_id"`
	Data     dyntype.JSONString `tfsdk:"data"`
}

type schedulerConditionModel struct {
	EntityID  types.String `tfsdk:"entity_id"`
	Attribute types.String `tfsdk:"attribute"`
	Value     types.String `tfsdk:"value"`
	MatchType types.String `tfsdk:"match_type"`
}

func (r *schedulerScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduler_schedule"
}

func (r *schedulerScheduleResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	timeHint := "`HH:MM:SS` (24 hours, e.g. `06:30:00`), or relative to the sun as `sunrise` or `sunset`, " +
		"`+` or `-`, and an offset `HH:MM:SS` (e.g. `sunset-00:30:00`)"
	resp.Schema = schema.Schema{
		Description: "A schedule of the Scheduler integration (`scheduler`), the backend of the scheduler " +
			"card: on the chosen days, it runs actions at fixed times or relative to the sun, e.g. a weekly " +
			"heating plan. The Scheduler integration is a custom integration (" +
			"https://github.com/nielsfaber/scheduler-component, version 3.3, e.g. installed through HACS) and " +
			"must be installed and set up in Home Assistant; without it, this resource fails with an error that " +
			"says so, and everything else keeps working. Changes update the schedule in place, except a new " +
			"`name`, which replaces it. Edits made in the card show up as drift.",
		Attributes: map[string]schema.Attribute{
			"schedule_id": schema.StringAttribute{
				Description: "ID that the integration assigned to the schedule (6 hex digits). Import with this ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"entity_id": schema.StringAttribute{
				Description: "Entity ID of the schedule's switch, `switch.schedule_<name as slug>`. Turning it " +
					"off disables the schedule.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the schedule. The switch's entity ID is derived from it. Changing it " +
					"replaces the schedule, with a new `schedule_id`: Scheduler 3.3 loses the switch of a " +
					"schedule renamed in place.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(noOuterSpace, "must not start or end with whitespace"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"weekdays": schema.ListAttribute{
				Description: "Days on which the schedule runs: any of `mon`, `tue`, `wed`, `thu`, `fri`, " +
					"`sat`, `sun`, or `daily`, `workday`, `weekend`. `workday` follows " +
					"`binary_sensor.workday_sensor`. Defaults to `[\"daily\"]`.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType,
					[]attr.Value{types.StringValue("daily")})),
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.OneOf(schedulerWeekdays...)),
				},
			},
			"start_date": schema.StringAttribute{
				Description: "First day on which the schedule runs, as `YYYY-MM-DD`. Unset, it starts right away.",
				Optional:    true,
				Validators:  []validator.String{dateValidator{}},
			},
			"end_date": schema.StringAttribute{
				Description: "Last day on which the schedule runs, as `YYYY-MM-DD`, not before `start_date`. " +
					"Unset, it does not end.",
				Optional:   true,
				Validators: []validator.String{dateValidator{}},
			},
			"repeat_type": schema.StringAttribute{
				Description: "What happens after the last timeslot of a day: `repeat` (run again next time) or " +
					"`pause` (turn the switch off; it must be turned on again). Defaults to `repeat`. The " +
					"integration's `single`, which deletes the schedule after it ran, is not supported: the next " +
					"apply would create it again.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("repeat"),
				Validators: []validator.String{
					repeatTypeValidator{},
				},
			},
			"tags": schema.SetAttribute{
				Description: "Tags of the schedule, which the card filters by. Defaults to none.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, nil)),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the schedule is enabled, i.e. its switch is on. Unset, the provider " +
					"neither changes nor reads it, so turning the switch on and off in Home Assistant is no " +
					"drift. Set, it is enforced (ADR-0031). A `pause` schedule turns itself off after it ran, " +
					"so `enabled = true` turns it on again on the next apply.",
				Optional: true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Update: true}),
			"timeslots": schema.ListNestedAttribute{
				Description: "When the schedule runs and what it does: at a time (`start` only), or in a time " +
					"range (`start` and `stop`), whose actions run at the start, and again when Home " +
					"Assistant starts within the range. Timeslots with fixed times must not overlap.",
				Required:   true,
				Validators: []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Description: "Start, as " + timeHint + ".",
							Required:    true,
							Validators:  []validator.String{schedulerTimeValidator()},
						},
						"stop": schema.StringAttribute{
							Description: "End of the range, in the format of `start`. `00:00:00` is the end of " +
								"the day. Unset, the timeslot is a single time.",
							Optional:   true,
							Validators: []validator.String{schedulerTimeValidator()},
						},
						"actions": schema.ListNestedAttribute{
							Description: "Actions to run, in order.",
							Required:    true,
							Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"action": schema.StringAttribute{
										Description: "Action to call, as `<domain>.<action>`, e.g. " +
											"`climate.set_temperature`.",
										Required: true,
										Validators: []validator.String{stringvalidator.RegexMatches(dottedID,
											"must have the format <domain>.<name>, e.g. climate.set_temperature")},
									},
									"entity_id": schema.StringAttribute{
										Description: "Entity to call the action on. Unset, the action gets no target.",
										Optional:    true,
										Validators: []validator.String{stringvalidator.RegexMatches(dottedID,
											"must be an entity ID, e.g. climate.living_room")},
									},
									"data": schema.StringAttribute{
										Description: "Data of the action, as a JSON object, e.g. " +
											"`jsonencode({ temperature = 21 })`. The order of keys and the spelling " +
											"of numbers do not matter. Defaults to `{}`.",
										Optional:   true,
										Computed:   true,
										CustomType: dyntype.JSONStringType{},
										Default:    stringdefault.StaticString("{}"),
										Validators: []validator.String{jsonObjectValidator{}},
									},
								},
							},
						},
						"conditions": schema.ListNestedAttribute{
							Description: "Conditions on the state of entities that must hold for the actions to " +
								"run. The card gives every timeslot the same conditions. Defaults to none.",
							Optional: true,
							Computed: true,
							Default:  listdefault.StaticValue(types.ListValueMust(schedulerConditionType, nil)),
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"entity_id": schema.StringAttribute{
										Description: "Entity whose state is compared.",
										Required:    true,
										Validators: []validator.String{stringvalidator.RegexMatches(dottedID,
											"must be an entity ID, e.g. binary_sensor.window")},
									},
									"attribute": schema.StringAttribute{
										Description: "Attribute to compare instead of the state.",
										Optional:    true,
									},
									"match_type": schema.StringAttribute{
										Description: "Comparison: `is`, `not`, `below`, or `above`.",
										Required:    true,
										Validators:  []validator.String{stringvalidator.OneOf("is", "not", "below", "above")},
									},
									"value": schema.StringAttribute{
										Description: "Value to compare with, e.g. `on` or `20`. With `below` and " +
											"`above`, it must be a number.",
										Required: true,
									},
								},
							},
						},
						"condition_type": schema.StringAttribute{
							Description: "How the conditions combine: `and` (all must hold) or `or` (one must " +
								"hold). Unset, the integration uses `or`.",
							Optional:   true,
							Validators: []validator.String{stringvalidator.OneOf("and", "or")},
						},
						"track_conditions": schema.BoolAttribute{
							Description: "Whether to run the actions later in the range, when the conditions start " +
								"to hold. Defaults to `false`.",
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func (r *schedulerScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := haClient(req, resp); c != nil {
		*r = *newSchedulerScheduleResource(c.WS)
	}
}

// ModifyPlan checks, when a schedule is to be created, that the Scheduler integration is set up,
// so that the plan rather than the apply fails without it.
func (r *schedulerScheduleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if !r.configured || !req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if _, err := r.schedulers.List(ctx); err != nil {
		addSchedulerError(&resp.Diagnostics, "Reading Scheduler schedules", err)
	}
}

func (r *schedulerScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan schedulerScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, err := plan.fields()
	if err != nil {
		resp.Diagnostics.AddError("Creating Scheduler schedule", err.Error())
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, defaultSchedulerWriteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.schedulers.Add(ctx, fields)
	if err != nil {
		addSchedulerError(&resp.Diagnostics, "Creating Scheduler schedule", err)
		return
	}
	// Store the ID right away, so a failure below leaves the schedule in state, not orphaned.
	plan.ScheduleID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schedule_id"), id)...)
	r.finishWrite(ctx, plan, fields, timeout, &resp.Diagnostics, &resp.State)
}

func (r *schedulerScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan schedulerScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, err := plan.fields()
	if err != nil {
		resp.Diagnostics.AddError("Updating Scheduler schedule", err.Error())
		return
	}
	timeout, diags := plan.Timeouts.Update(ctx, defaultSchedulerWriteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	current, ok := r.get(ctx, plan.ScheduleID.ValueString(), "Updating Scheduler schedule", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.schedulers.Edit(ctx, current.EntityID, fields); err != nil {
		addSchedulerError(&resp.Diagnostics, "Updating Scheduler schedule", err)
		return
	}
	r.finishWrite(ctx, plan, fields, timeout, &resp.Diagnostics, &resp.State)
}

// get returns the schedule with the ID id, and adds an error if it is missing.
func (r *schedulerScheduleResource) get(ctx context.Context, id, summary string, diags *diag.Diagnostics) (client.SchedulerSchedule, bool) {
	s, ok, err := r.schedulers.Get(ctx, id)
	if err != nil {
		addSchedulerError(diags, summary, err)
		return s, false
	}
	if !ok {
		diags.AddError(summary, fmt.Sprintf("Scheduler schedule %s does not exist anymore.", id))
	}
	return s, ok
}

// finishWrite waits until the schedule reads back as written, sets `enabled` if the plan manages
// it, and stores the schedule in state. Each wait lasts at most timeout.
func (r *schedulerScheduleResource) finishWrite(ctx context.Context, plan schedulerScheduleModel, fields map[string]any, timeout time.Duration, diags *diag.Diagnostics, state stateSetter) {
	if diags.HasError() {
		return
	}
	id := plan.ScheduleID.ValueString()
	s, ok := r.awaitSchedule(ctx, id, timeout, func(s client.SchedulerSchedule) bool {
		return sameFields(fields, newSchedulerScheduleModel(s, types.BoolNull()))
	}, diags)
	if !ok {
		return
	}
	if !plan.Enabled.IsNull() && s.Enabled != plan.Enabled.ValueBool() {
		if err := r.schedulers.SetEnabled(ctx, s.EntityID, plan.Enabled.ValueBool()); err != nil {
			addSchedulerError(diags, "Setting enabled of Scheduler schedule", err)
			return
		}
		want := plan.Enabled.ValueBool()
		if s, ok = r.awaitSchedule(ctx, id, timeout, func(s client.SchedulerSchedule) bool { return s.Enabled == want }, diags); !ok {
			return
		}
	}
	m := newSchedulerScheduleModel(s, plan.Enabled)
	// The values read back equal the plan's up to JSON spelling; keep the plan's spelling.
	m.Timeslots = plan.Timeslots
	m.Timeouts = plan.Timeouts
	diags.Append(state.Set(ctx, m)...)
}

// stateSetter is the part of tfsdk.State that finishWrite needs.
type stateSetter interface {
	Set(ctx context.Context, val any) diag.Diagnostics
}

// awaitSchedule reads the schedule until done holds for it, and returns the last read. If done
// never holds within timeout, it adds an error.
func (r *schedulerScheduleResource) awaitSchedule(ctx context.Context, id string, timeout time.Duration, done func(client.SchedulerSchedule) bool, diags *diag.Diagnostics) (client.SchedulerSchedule, bool) {
	var last client.SchedulerSchedule
	wctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := waitFor(wctx, func(ctx context.Context) (bool, error) {
		s, ok, err := r.schedulers.Get(ctx, id)
		if err != nil || !ok {
			return false, err
		}
		last = s
		return done(s), nil
	})
	switch {
	case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
		diags.AddError("Scheduler schedule not updated",
			fmt.Sprintf("Scheduler schedule %s did not read back as written within %s. Run `tofu apply` again, "+
				"or raise `timeouts`; if this persists, the Scheduler integration may store the values "+
				"differently, please report it.", id, timeout))
		return last, false
	case err != nil:
		addSchedulerError(diags, "Reading Scheduler schedule", err)
		return last, false
	}
	return last, true
}

func (r *schedulerScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state schedulerScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, ok, err := r.schedulers.Get(ctx, state.ScheduleID.ValueString())
	if err != nil {
		addSchedulerError(&resp.Diagnostics, "Reading Scheduler schedule", err)
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	m := newSchedulerScheduleModel(s, state.Enabled)
	m.Timeouts = state.Timeouts
	keepNumberSpelling(m.Timeslots, state.Timeslots)
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *schedulerScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state schedulerScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, ok, err := r.schedulers.Get(ctx, state.ScheduleID.ValueString())
	if err != nil {
		addSchedulerError(&resp.Diagnostics, "Deleting Scheduler schedule", err)
		return
	}
	if !ok {
		return
	}
	if err := r.schedulers.Remove(ctx, s.EntityID); err != nil {
		addSchedulerError(&resp.Diagnostics, "Deleting Scheduler schedule", err)
	}
}

// ImportState takes the schedule_id.
func (r *schedulerScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("schedule_id"), req, resp)
}

// addSchedulerError adds err as an error, with how to install the integration if it is missing.
func addSchedulerError(diags *diag.Diagnostics, summary string, err error) {
	if errors.Is(err, client.ErrSchedulerMissing) {
		diags.AddError("Scheduler integration missing", err.Error()+"\n\n"+client.SchedulerHint)
		return
	}
	diags.AddError(summary, client.ErrorDetail(err))
}

// newSchedulerScheduleModel turns s into a model. enabled is null if the schedule's `enabled` is
// not managed, and anything else if it is.
func newSchedulerScheduleModel(s client.SchedulerSchedule, enabled types.Bool) schedulerScheduleModel {
	m := schedulerScheduleModel{
		ScheduleID: types.StringValue(s.ScheduleID),
		EntityID:   types.StringValue(s.EntityID),
		Name:       types.StringPointerValue(s.Name),
		Weekdays:   nonNil(s.Weekdays),
		StartDate:  types.StringPointerValue(s.StartDate),
		EndDate:    types.StringPointerValue(s.EndDate),
		RepeatType: types.StringValue(s.RepeatType),
		Tags:       nonNil(s.Tags),
		Enabled:    types.BoolNull(),
		Timeouts:   timeouts.Value{Object: types.ObjectNull(schedulerTimeoutsType)},
		Timeslots:  make([]schedulerTimeslotModel, 0, len(s.Timeslots)),
	}
	if !enabled.IsNull() {
		m.Enabled = types.BoolValue(s.Enabled)
	}
	for _, ts := range s.Timeslots {
		t := schedulerTimeslotModel{
			Start:           types.StringValue(ts.Start),
			Stop:            types.StringPointerValue(ts.Stop),
			ConditionType:   types.StringPointerValue(ts.ConditionType),
			TrackConditions: types.BoolValue(ts.TrackConditions),
			Actions:         make([]schedulerActionModel, 0, len(ts.Actions)),
			Conditions:      make([]schedulerConditionModel, 0, len(ts.Conditions)),
		}
		for _, a := range ts.Actions {
			data := "{}"
			if len(a.ServiceData) > 0 && string(a.ServiceData) != "null" {
				data = string(a.ServiceData)
			}
			t.Actions = append(t.Actions, schedulerActionModel{
				Action:   types.StringValue(a.Service),
				EntityID: types.StringPointerValue(a.EntityID),
				Data:     dyntype.NewJSONString(data),
			})
		}
		for _, c := range ts.Conditions {
			t.Conditions = append(t.Conditions, schedulerConditionModel{
				EntityID:  types.StringValue(c.EntityID),
				Attribute: types.StringPointerValue(c.Attribute),
				Value:     types.StringValue(conditionValueString(c.Value)),
				MatchType: types.StringValue(c.MatchType),
			})
		}
		m.Timeslots = append(m.Timeslots, t)
	}
	return m
}

// conditionValueString returns a condition's value, a JSON string or number, as a string.
func conditionValueString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// fields returns the schedule as `scheduler.add` and `scheduler.edit` take it. Optional fields
// that are unset are left out, because the integration rejects them as null.
func (m schedulerScheduleModel) fields() (map[string]any, error) {
	tags := slices.Sorted(slices.Values(nonNil(m.Tags)))
	f := map[string]any{
		"name":        m.Name.ValueString(),
		"weekdays":    nonNil(m.Weekdays),
		"repeat_type": m.RepeatType.ValueString(),
		"tags":        tags,
		"start_date":  m.StartDate.ValueStringPointer(),
		"end_date":    m.EndDate.ValueStringPointer(),
	}
	slots := make([]map[string]any, 0, len(m.Timeslots))
	for _, t := range m.Timeslots {
		slot := map[string]any{
			"start":            t.Start.ValueString(),
			"track_conditions": t.TrackConditions.ValueBool(),
		}
		if !t.Stop.IsNull() {
			slot["stop"] = t.Stop.ValueString()
		}
		if !t.ConditionType.IsNull() {
			slot["condition_type"] = t.ConditionType.ValueString()
		}
		actions := make([]map[string]any, 0, len(t.Actions))
		for _, a := range t.Actions {
			action := map[string]any{"service": a.Action.ValueString()}
			if !a.EntityID.IsNull() {
				action["entity_id"] = a.EntityID.ValueString()
			}
			data := json.RawMessage("{}")
			if !a.Data.IsNull() && a.Data.ValueString() != "" {
				data = json.RawMessage(a.Data.ValueString())
			}
			if !json.Valid(data) {
				return nil, fmt.Errorf("data of action %s is not valid JSON", a.Action.ValueString())
			}
			action["service_data"] = data
			actions = append(actions, action)
		}
		slot["actions"] = actions
		if len(t.Conditions) > 0 {
			conds := make([]map[string]any, 0, len(t.Conditions))
			for _, c := range t.Conditions {
				cond := map[string]any{
					"entity_id":  c.EntityID.ValueString(),
					"match_type": c.MatchType.ValueString(),
					"value":      conditionValue(c.MatchType.ValueString(), c.Value.ValueString()),
				}
				if !c.Attribute.IsNull() {
					cond["attribute"] = c.Attribute.ValueString()
				}
				conds = append(conds, cond)
			}
			slot["conditions"] = conds
		}
		slots = append(slots, slot)
	}
	f["timeslots"] = slots
	return f, nil
}

// conditionValue is the value of a condition as the integration takes it: a number for `below`
// and `above`, which compare numbers, and the string otherwise.
func conditionValue(matchType, value string) any {
	if isNumericMatch(matchType) {
		if jsonNumber.MatchString(value) {
			return json.Number(value)
		}
	}
	return value
}

// sameFields reports whether the schedule m holds the fields that were written.
func sameFields(written map[string]any, m schedulerScheduleModel) bool {
	got, err := m.fields()
	if err != nil {
		return false
	}
	a, errA := json.Marshal(written)
	b, errB := json.Marshal(got)
	if errA != nil || errB != nil {
		return false
	}
	eq, err := dyntype.SemanticallyEqualJSON(a, b)
	return err == nil && eq
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// keepNumberSpelling keeps the prior value of each `below` or `above` condition whose number
// reads back spelled differently, e.g. `20.50` as `20.5`: those values are sent as numbers, and
// the integration stores them as numbers. `is` and `not` values are strings and compared as
// strings, so they are left alone: a changed string is drift even if it is the same number.
func keepNumberSpelling(read, prior []schedulerTimeslotModel) {
	for i := range min(len(read), len(prior)) {
		for j := range min(len(read[i].Conditions), len(prior[i].Conditions)) {
			r, p := &read[i].Conditions[j], prior[i].Conditions[j]
			if !isNumericMatch(r.MatchType.ValueString()) || !isNumericMatch(p.MatchType.ValueString()) {
				continue
			}
			a, errA := strconv.ParseFloat(r.Value.ValueString(), 64)
			b, errB := strconv.ParseFloat(p.Value.ValueString(), 64)
			if errA == nil && errB == nil && a == b && jsonNumber.MatchString(p.Value.ValueString()) {
				r.Value = p.Value
			}
		}
	}
}

// isNumericMatch reports whether a condition's match_type compares numbers.
func isNumericMatch(matchType string) bool {
	return matchType == "below" || matchType == "above"
}
