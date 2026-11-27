package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*preferredAssistPipelineResource)(nil)
	_ resource.ResourceWithImportState = (*preferredAssistPipelineResource)(nil)
)

const preferredAssistPipelineID = "preferred"

// NewPreferredAssistPipeline returns the `homeassistant_preferred_assist_pipeline` resource.
func NewPreferredAssistPipeline() resource.Resource {
	return &preferredAssistPipelineResource{}
}

type preferredAssistPipelineResource struct {
	pipelines client.AssistPipelines
}

type preferredAssistPipelineModel struct {
	ID         types.String `tfsdk:"id"`
	PipelineID types.String `tfsdk:"pipeline_id"`
}

func (r *preferredAssistPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_preferred_assist_pipeline"
}

func (r *preferredAssistPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Which Assist pipeline is preferred: the one voice assistants set to \"preferred\" use, " +
			"and the default in the UI. There is exactly one per Home Assistant instance, so there is " +
			"nothing to create: the resource sets it. Destroying the resource only forgets it; the " +
			"preferred pipeline stays as it is.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Always `" + preferredAssistPipelineID + "`. Import with this ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pipeline_id": schema.StringAttribute{
				Description: "ID of the preferred pipeline, e.g. `homeassistant_assist_pipeline.main.id`.",
				Required:    true,
			},
		},
	}
}

func (r *preferredAssistPipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c := haClient(req, resp)
	if c == nil {
		return
	}
	r.pipelines = client.NewAssistPipelines(c.WS)
}

func (r *preferredAssistPipelineResource) set(ctx context.Context, plan preferredAssistPipelineModel, state *preferredAssistPipelineModel) error {
	if err := r.pipelines.SetPreferred(ctx, plan.PipelineID.ValueString()); err != nil {
		return err
	}
	*state = preferredAssistPipelineModel{
		ID:         types.StringValue(preferredAssistPipelineID),
		PipelineID: plan.PipelineID,
	}
	return nil
}

func (r *preferredAssistPipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, state preferredAssistPipelineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.set(ctx, plan, &state); err != nil {
		resp.Diagnostics.AddError("Setting the preferred Assist pipeline", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *preferredAssistPipelineResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	_, preferred, err := r.pipelines.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading the preferred Assist pipeline", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, preferredAssistPipelineModel{
		ID:         types.StringValue(preferredAssistPipelineID),
		PipelineID: types.StringValue(preferred),
	})...)
}

func (r *preferredAssistPipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state preferredAssistPipelineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.set(ctx, plan, &state); err != nil {
		resp.Diagnostics.AddError("Setting the preferred Assist pipeline", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Delete forgets the resource. Home Assistant always has a preferred pipeline, so there is
// nothing to undo.
func (r *preferredAssistPipelineResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *preferredAssistPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != preferredAssistPipelineID {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("The preferred Assist pipeline is imported with the ID %q, got %q.", preferredAssistPipelineID, req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
