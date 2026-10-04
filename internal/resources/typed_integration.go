package resources

import (
	"context"
	"fmt"
	"maps"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// typedIntegrationSpec describes a typed integration resource: a config entry of a fixed domain
// whose flows are answered from typed attributes (spec/integrations.md, "Typed integrations").
type typedIntegrationSpec struct {
	domain      string
	description string
	// attributes are the inputs; the computed entry attributes are added to them.
	attributes map[string]schema.Attribute
	// validate checks the configuration across attributes; optional.
	validate func(ctx context.Context, cfg resource.ValidateConfigRequest) diag.Diagnostics
	// answer wraps the field answer of the config and reconfigure flows; optional.
	answer func(base client.FlowAnswer, values map[string]any, files map[string]string) client.FlowAnswer
}

// typedModel is the pointer to a typed integration's model.
type typedModel[M any] interface {
	*M
	// entry returns the computed attributes of the config entry.
	entry() *entryModel
	// imported reports whether the inputs are null, which they are only after an import.
	imported() bool
	// flowInput returns the field values and file contents for the config and reconfigure flows.
	flowInput(ctx context.Context) (values map[string]any, files map[string]string, diags diag.Diagnostics)
	// optionInput returns the field values for the options flow.
	optionInput() map[string]any
}

// entryModel holds the computed attributes every config entry resource has.
type entryModel struct {
	ID         types.String `tfsdk:"id"`
	Title      types.String `tfsdk:"title"`
	State      types.String `tfsdk:"state"`
	DisabledBy types.String `tfsdk:"disabled_by"`
}

func (m *entryModel) set(e client.ConfigEntry) {
	m.ID = types.StringValue(e.EntryID)
	m.Title = types.StringValue(e.Title)
	m.State = types.StringValue(e.State)
	m.DisabledBy = types.StringPointerValue(e.DisabledBy)
}

// unread marks the entry as created but not read.
func (m *entryModel) unread(id string) {
	m.ID = types.StringValue(id)
	m.Title, m.State, m.DisabledBy = types.StringNull(), types.StringNull(), types.StringNull()
}

var (
	_ resource.ResourceWithConfigure      = (*typedIntegration[esphomeModel, *esphomeModel])(nil)
	_ resource.ResourceWithImportState    = (*typedIntegration[esphomeModel, *esphomeModel])(nil)
	_ resource.ResourceWithValidateConfig = (*typedIntegration[esphomeModel, *esphomeModel])(nil)
)

// typedIntegration implements a typed integration resource from its spec.
type typedIntegration[M any, P typedModel[M]] struct {
	spec    typedIntegrationSpec
	rest    *client.RESTClient
	flows   client.ConfigFlows
	options client.OptionsFlows
	entries client.ConfigEntries
}

func (r *typedIntegration[M, P]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.domain
}

func (r *typedIntegration[M, P]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computed := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			Description:   desc,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	attrs := map[string]schema.Attribute{
		"id":          computed("The config entry's `entry_id`, assigned by Home Assistant. Import with this ID."),
		"title":       computed("Title of the config entry, chosen by the integration."),
		"state":       computed("State of the config entry when it was last read, e.g. `loaded` or `setup_error`."),
		"disabled_by": computed("Who disabled the config entry, e.g. `user`; null if it is enabled."),
	}
	maps.Copy(attrs, r.spec.attributes)
	resp.Schema = schema.Schema{Description: r.spec.description, Attributes: attrs}
}

func (r *typedIntegration[M, P]) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if r.spec.validate != nil {
		resp.Diagnostics.Append(r.spec.validate(ctx, req)...)
	}
}

func (r *typedIntegration[M, P]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.rest = c.REST
	r.flows = client.NewConfigFlows(c.REST)
	r.options = client.NewOptionsFlows(c.REST)
	r.entries = client.NewConfigEntries(c.WS, c.REST)
}

// answer returns the answer to the config or reconfigure flow for m.
func (r *typedIntegration[M, P]) answer(ctx context.Context, m P) (client.FlowAnswer, diag.Diagnostics) {
	values, files, diags := m.flowInput(ctx)
	answer := client.FieldAnswer(r.rest, values, files)
	if r.spec.answer != nil {
		answer = r.spec.answer(answer, values, files)
	}
	return answer, diags
}

func (r *typedIntegration[M, P]) setOptions(ctx context.Context, m P) error {
	return r.options.Run(ctx, m.entry().ID.ValueString(), client.FieldAnswer(r.rest, m.optionInput(), nil))
}

// readEntry reads the config entry into m. A missing entry is an error.
func (r *typedIntegration[M, P]) readEntry(ctx context.Context, m P) error {
	id := m.entry().ID.ValueString()
	e, ok, err := r.entries.Get(ctx, r.spec.domain, id)
	if err == nil && !ok {
		err = fmt.Errorf("config entry %s of %s is gone", id, r.spec.domain)
	}
	if err != nil {
		return err
	}
	m.entry().set(e)
	return nil
}

func (r *typedIntegration[M, P]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := P(&plan)
	answer, diags := r.answer(ctx, p)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	entryID, err := r.flows.RunAnswer(ctx, r.spec.domain, "", answer)
	if err != nil {
		resp.Diagnostics.AddError("Creating config entry", client.ErrorDetail(err))
		return
	}
	// From here on the entry exists: keep it in state even on errors, so it is tainted and
	// replaced, not orphaned.
	p.entry().unread(entryID)
	if err := r.setOptions(ctx, p); err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Setting the options of the new config entry", client.ErrorDetail(err))
		return
	}
	if err := r.readEntry(ctx, p); err != nil {
		p.entry().unread(entryID)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Reading the new config entry", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *typedIntegration[M, P]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := P(&state)

	e, ok, err := r.entries.Get(ctx, r.spec.domain, p.entry().ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading config entry", client.ErrorDetail(err))
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	// The inputs cannot be read back, so they stay as they are.
	p.entry().set(e)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update runs the reconfigure flow if the connection inputs changed, and the options flow if the
// options changed. Right after an import it only records the inputs.
func (r *typedIntegration[M, P]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, s := P(&plan), P(&state)
	*p.entry() = *s.entry()
	if s.imported() {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return
	}

	planValues, planFiles, diags := p.flowInput(ctx)
	resp.Diagnostics.Append(diags...)
	stateValues, stateFiles, diags := s.flowInput(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !reflect.DeepEqual(planValues, stateValues) || !reflect.DeepEqual(planFiles, stateFiles) {
		answer, diags := r.answer(ctx, p)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if _, err := r.flows.RunAnswer(ctx, r.spec.domain, p.entry().ID.ValueString(), answer); err != nil {
			resp.Diagnostics.AddError("Reconfiguring config entry", client.ErrorDetail(err))
			return
		}
	}
	if !reflect.DeepEqual(p.optionInput(), s.optionInput()) {
		if err := r.setOptions(ctx, p); err != nil {
			// State keeps the old inputs, so the next apply runs both flows again.
			resp.Diagnostics.AddError("Setting the options of the config entry", client.ErrorDetail(err))
			return
		}
	}
	// The computed attributes keep their planned values: a reconfigure reloads the entry, so
	// `state` changes during apply. The next refresh reads it.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *typedIntegration[M, P]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.entries.Delete(ctx, P(&state).entry().ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting config entry", client.ErrorDetail(err))
	}
}

func (r *typedIntegration[M, P]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// replaceUnlessImported requires replacement when the attribute changes, except right after an
// import, which leaves the attribute key null in state.
func replaceUnlessImported(key string) planmodifier.String {
	desc := "Changing it replaces the config entry, except right after an import."
	return stringplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
		var prior types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(key), &prior)...)
		resp.RequiresReplace = !prior.IsNull()
	}, desc, desc)
}

// int64Between accepts integers of at least min, and at most max if max > min.
type int64Between struct{ min, max int64 }

var _ validator.Int64 = int64Between{}

func (v int64Between) Description(context.Context) string {
	if v.max > v.min {
		return fmt.Sprintf("must be between %d and %d", v.min, v.max)
	}
	return fmt.Sprintf("must be at least %d", v.min)
}

func (v int64Between) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int64Between) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	n := req.ConfigValue.ValueInt64()
	if n < v.min || (v.max > v.min && n > v.max) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value",
			fmt.Sprintf("Attribute %s %s, got: %d", req.Path, v.Description(ctx), n))
	}
}
