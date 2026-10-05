package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*deviceSettingsResource)(nil)
	_ resource.ResourceWithImportState = (*deviceSettingsResource)(nil)
)

// NewDeviceSettings returns the `homeassistant_device_settings` resource.
func NewDeviceSettings() resource.Resource {
	return &deviceSettingsResource{}
}

type deviceSettingsResource struct {
	devices client.Registry[client.Device]
}

type deviceSettingsModel struct {
	DeviceID      types.String   `tfsdk:"device_id"`
	NameByUser    types.String   `tfsdk:"name_by_user"`
	AreaID        types.String   `tfsdk:"area_id"`
	Labels        types.Set      `tfsdk:"labels"`
	Disabled      types.Bool     `tfsdk:"disabled"`
	Name          types.String   `tfsdk:"name"`
	Manufacturer  types.String   `tfsdk:"manufacturer"`
	Model         types.String   `tfsdk:"model"`
	SWVersion     types.String   `tfsdk:"sw_version"`
	HWVersion     types.String   `tfsdk:"hw_version"`
	Identifiers   types.List     `tfsdk:"identifiers"`
	ConfigEntries types.List     `tfsdk:"config_entries"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func (r *deviceSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_settings"
}

func (r *deviceSettingsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computed := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Description:   description,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	computedList := func(description string, elem attr.Type) schema.ListAttribute {
		return schema.ListAttribute{
			Description:   description,
			ElementType:   elem,
			Computed:      true,
			PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		Description: "Settings of a device that an integration owns, as in the device settings " +
			"dialog: name, area, labels, disabled. Only the attributes you set are managed; " +
			"changes made elsewhere to the others cause no diff. Destroying the resource resets " +
			"the attributes you set to Home Assistant's defaults and leaves the device in place. " +
			"Values from before Tofu managed the device are not restored. Import with the device " +
			"ID; the first plan after an import sets the configured attributes.",
		Attributes: map[string]schema.Attribute{
			"device_id": schema.StringAttribute{
				Description: "Device ID, e.g. from the `homeassistant_device` data source. Changing " +
					"it replaces the resource. Create waits for the device to appear in the registry.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name_by_user": schema.StringAttribute{
				Description: "Name that overrides the integration's. Unsetting it goes back to " +
					"the integration's name.",
				Optional: true,
			},
			"area_id": schema.StringAttribute{
				Description: "ID of the device's area, e.g. `homeassistant_area.bedroom.id`. " +
					"Its entities without an area of their own follow it.",
				Optional: true,
			},
			"labels": schema.SetAttribute{
				Description: "IDs of the labels on the device. When set, this is the complete " +
					"list: labels added elsewhere are removed.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Whether the user disables the device, which also disables its " +
					"entities. Reads `false` when it is enabled or disabled by its integration.",
				Optional: true,
			},
			"name":         computed("Name the integration gives the device."),
			"manufacturer": computed("Manufacturer, as the integration reports it."),
			"model":        computed("Model, as the integration reports it."),
			"sw_version":   computed("Software or firmware version."),
			"hw_version":   computed("Hardware version."),
			"identifiers": computedList("Identifiers of the device, each a list such as "+
				"`[\"hue\", \"00:17:88:01:...\"]`: the integration's domain, then its ID.",
				types.ListType{ElemType: types.StringType}),
			"config_entries": computedList("IDs of the config entries that provide the device.", types.StringType),
			"timeouts":       timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		},
	}
}

func (r *deviceSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.devices = client.Devices(c.WS)
}

// changes returns the registry fields that turn from into to: those set in to that differ from
// from, and those set only in from, reset to their defaults. Unset attributes are left out, so
// only configured fields are ever written (ADR-0010).
func (to deviceSettingsModel) changes(ctx context.Context, from deviceSettingsModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	fields := map[string]any{}
	set := func(key string, toV, fromV attr.Value, value func() any, reset any) {
		switch {
		case !toV.IsNull() && !toV.Equal(fromV):
			fields[key] = value()
		case toV.IsNull() && !fromV.IsNull():
			fields[key] = reset
		}
	}
	set("name_by_user", to.NameByUser, from.NameByUser, func() any { return to.NameByUser.ValueStringPointer() }, nil)
	set("area_id", to.AreaID, from.AreaID, func() any { return to.AreaID.ValueStringPointer() }, nil)
	set("labels", to.Labels, from.Labels, func() any {
		labels := []string{}
		diags.Append(to.Labels.ElementsAs(ctx, &labels, false)...)
		return labels
	}, []string{})
	set("disabled_by", to.Disabled, from.Disabled, func() any { return byUser(to.Disabled.ValueBool()) }, nil)
	return fields, diags
}

// refresh reads d into m: the computed attributes, and the settable ones that m manages.
func (m *deviceSettingsModel) refresh(ctx context.Context, d client.Device) diag.Diagnostics {
	m.DeviceID = types.StringValue(d.ID)
	diags := m.computedFrom(ctx, d)
	if !m.NameByUser.IsNull() {
		m.NameByUser = types.StringPointerValue(d.NameByUser)
	}
	if !m.AreaID.IsNull() {
		m.AreaID = types.StringPointerValue(d.AreaID)
	}
	if !m.Labels.IsNull() {
		var labelDiags diag.Diagnostics
		m.Labels, labelDiags = types.SetValueFrom(ctx, types.StringType, append([]string{}, d.Labels...))
		diags.Append(labelDiags...)
	}
	if !m.Disabled.IsNull() {
		m.Disabled = types.BoolValue(d.DisabledBy != nil && *d.DisabledBy == "user")
	}
	return diags
}

// computedFrom sets m's computed attributes from d and leaves the settable ones alone.
func (m *deviceSettingsModel) computedFrom(ctx context.Context, d client.Device) diag.Diagnostics {
	var diags, listDiags diag.Diagnostics
	m.Name = types.StringPointerValue(d.Name)
	m.Manufacturer = types.StringPointerValue(d.Manufacturer)
	m.Model = types.StringPointerValue(d.Model)
	m.SWVersion = types.StringPointerValue(d.SWVersion)
	m.HWVersion = types.StringPointerValue(d.HWVersion)
	m.Identifiers, listDiags = types.ListValueFrom(ctx, types.ListType{ElemType: types.StringType},
		append([][]string{}, d.Identifiers...))
	diags.Append(listDiags...)
	m.ConfigEntries, listDiags = types.ListValueFrom(ctx, types.StringType, append([]string{}, d.ConfigEntries...))
	diags.Append(listDiags...)
	return diags
}

// awaitDevice polls the registry until deviceID is in it, for at most timeout.
func (r *deviceSettingsResource) awaitDevice(ctx context.Context, deviceID string, timeout time.Duration) (client.Device, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var device client.Device
	err := waitFor(waitCtx, func(ctx context.Context) (bool, error) {
		var ok bool
		var err error
		device, ok, err = r.devices.Get(ctx, deviceID)
		return ok, err
	})
	if err != nil && errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
		return device, fmt.Errorf("device %s did not appear in the device registry within %s.\n\n"+
			"Check that the device ID is correct and that the integration providing it is set up. "+
			"If the integration needs longer, raise `timeouts.create`", deviceID, timeout)
	}
	return device, err
}

func (r *deviceSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, defaultSettingsCreateTimeout)
	resp.Diagnostics.Append(diags...)
	fields, diags := plan.changes(ctx, deviceSettingsModel{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	device, err := r.awaitDevice(ctx, deviceID, timeout)
	if err != nil {
		resp.Diagnostics.AddError("Waiting for device", client.ErrorDetail(err))
		return
	}
	if len(fields) > 0 {
		if device, err = r.devices.Update(ctx, deviceID, fields); err != nil {
			resp.Diagnostics.AddError("Updating device settings", client.ErrorDetail(err))
			return
		}
	}

	resp.Diagnostics.Append(plan.computedFrom(ctx, device)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *deviceSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device, ok, err := r.devices.Get(ctx, state.DeviceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading device settings", client.ErrorDetail(err))
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.refresh(ctx, device)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *deviceSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state deviceSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, diags := plan.changes(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(fields) > 0 {
		device, err := r.devices.Update(ctx, plan.DeviceID.ValueString(), fields)
		if err != nil {
			resp.Diagnostics.AddError("Updating device settings", client.ErrorDetail(err))
			return
		}
		resp.Diagnostics.Append(plan.computedFrom(ctx, device)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *deviceSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deviceSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, diags := deviceSettingsModel{}.changes(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || len(fields) == 0 {
		return
	}

	deviceID := state.DeviceID.ValueString()
	_, err := r.devices.Update(ctx, deviceID, fields)
	var wsErr *client.WSError
	if errors.As(err, &wsErr) {
		// A device that is gone has nothing left to reset.
		if _, ok, getErr := r.devices.Get(ctx, deviceID); getErr == nil && !ok {
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Resetting device settings", client.ErrorDetail(err))
	}
}

func (r *deviceSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("device_id"), req, resp)
}
