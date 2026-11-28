package datasources

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var _ datasource.DataSourceWithConfigure = (*schedulerSchedulesDataSource)(nil)

// NewSchedulerSchedules returns the `homeassistant_scheduler_schedules` data source.
func NewSchedulerSchedules() datasource.DataSource {
	return &schedulerSchedulesDataSource{}
}

type schedulerSchedulesDataSource struct {
	schedulers client.Schedulers
}

type schedulerSchedulesModel struct {
	Schedules []schedulerScheduleSummary `tfsdk:"schedules"`
}

type schedulerScheduleSummary struct {
	ScheduleID types.String `tfsdk:"schedule_id"`
	Name       types.String `tfsdk:"name"`
	EntityID   types.String `tfsdk:"entity_id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	Weekdays   []string     `tfsdk:"weekdays"`
	Tags       types.Set    `tfsdk:"tags"`
	Entities   types.Set    `tfsdk:"entities"`
}

func newSchedulerScheduleSummary(s client.SchedulerSchedule) schedulerScheduleSummary {
	var entities []string
	for _, ts := range s.Timeslots {
		for _, a := range ts.Actions {
			if a.EntityID != nil && !slices.Contains(entities, *a.EntityID) {
				entities = append(entities, *a.EntityID)
			}
		}
	}
	weekdays := s.Weekdays
	if weekdays == nil {
		weekdays = []string{}
	}
	return schedulerScheduleSummary{
		ScheduleID: types.StringValue(s.ScheduleID),
		Name:       types.StringPointerValue(s.Name),
		EntityID:   types.StringValue(s.EntityID),
		Enabled:    types.BoolValue(s.Enabled),
		Weekdays:   weekdays,
		Tags:       stringSet(s.Tags),
		Entities:   stringSet(entities),
	}
}

func (d *schedulerSchedulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduler_schedules"
}

func (d *schedulerSchedulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists every schedule of the Scheduler integration (`scheduler`), e.g. to find the " +
			"`schedule_id` to import into `homeassistant_scheduler_schedule`. The Scheduler integration " +
			"(https://github.com/nielsfaber/scheduler-component, version 3.3) must be installed and set up; " +
			"without it, reading fails with an error that says so.",
		Attributes: map[string]schema.Attribute{
			"schedules": schema.ListNestedAttribute{
				Description: "The schedules, sorted by `name`, then by `schedule_id`.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"schedule_id": schema.StringAttribute{Description: "ID of the schedule, the import ID.", Computed: true},
					"name":        schema.StringAttribute{Description: "Name; null if it has none.", Computed: true},
					"entity_id":   schema.StringAttribute{Description: "Entity ID of the schedule's switch.", Computed: true},
					"enabled":     schema.BoolAttribute{Description: "Whether the schedule is enabled.", Computed: true},
					"weekdays": schema.SetAttribute{
						Description: "Days on which it runs, e.g. `[\"mon\", \"fri\"]` or `[\"daily\"]`.",
						ElementType: types.StringType, Computed: true,
					},
					"tags": schema.SetAttribute{Description: "Tags.", ElementType: types.StringType, Computed: true},
					"entities": schema.SetAttribute{
						Description: "Entities that the schedule's actions act on.",
						ElementType: types.StringType, Computed: true,
					},
				}},
			},
		},
	}
}

func (d *schedulerSchedulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := haClient(req, resp); c != nil {
		d.schedulers = client.NewSchedulers(c.WS)
	}
}

func (d *schedulerSchedulesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	all, err := d.schedulers.List(ctx)
	if err != nil {
		if errors.Is(err, client.ErrSchedulerMissing) {
			resp.Diagnostics.AddError("Scheduler integration missing", err.Error()+"\n\n"+client.SchedulerHint)
		} else {
			resp.Diagnostics.AddError("Reading Scheduler schedules", client.ErrorDetail(err))
		}
		return
	}
	slices.SortFunc(all, func(a, b client.SchedulerSchedule) int {
		return cmp.Or(cmp.Compare(deref(a.Name), deref(b.Name)), cmp.Compare(a.ScheduleID, b.ScheduleID))
	})
	out := schedulerSchedulesModel{Schedules: make([]schedulerScheduleSummary, len(all))}
	for i, s := range all {
		out.Schedules[i] = newSchedulerScheduleSummary(s)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
