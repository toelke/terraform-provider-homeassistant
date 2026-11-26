package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*assistPipelineResource)(nil)
	_ resource.ResourceWithImportState = (*assistPipelineResource)(nil)
)

// NewAssistPipeline returns the `homeassistant_assist_pipeline` resource.
func NewAssistPipeline() resource.Resource {
	return &assistPipelineResource{}
}

type assistPipelineResource struct {
	pipelines client.AssistPipelines
}

type assistPipelineModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Language             types.String `tfsdk:"language"`
	ConversationEngine   types.String `tfsdk:"conversation_engine"`
	ConversationLanguage types.String `tfsdk:"conversation_language"`
	STTEngine            types.String `tfsdk:"stt_engine"`
	STTLanguage          types.String `tfsdk:"stt_language"`
	TTSEngine            types.String `tfsdk:"tts_engine"`
	TTSLanguage          types.String `tfsdk:"tts_language"`
	TTSVoice             types.String `tfsdk:"tts_voice"`
	WakeWordEntity       types.String `tfsdk:"wake_word_entity"`
	WakeWordID           types.String `tfsdk:"wake_word_id"`
	PreferLocalIntents   types.Bool   `tfsdk:"prefer_local_intents"`
}

func (r *assistPipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assist_pipeline"
}

func (r *assistPipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optional := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Description: description, Optional: true}
	}
	resp.Schema = schema.Schema{
		Description: "An Assist pipeline (Settings > Voice assistants): the conversation agent, " +
			"speech-to-text, text-to-speech, and wake word that a voice assistant uses. Which pipeline is " +
			"preferred is `homeassistant_preferred_assist_pipeline`. Home Assistant refuses to delete " +
			"the preferred pipeline.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Pipeline ID, assigned by Home Assistant. Import with this ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name, e.g. `Home Assistant`.",
				Required:    true,
			},
			"language": schema.StringAttribute{
				Description: "The pipeline's language, e.g. `en`.",
				Required:    true,
			},
			"conversation_engine": schema.StringAttribute{
				Description: "Conversation agent: an entity such as `conversation.home_assistant` (the built-in " +
					"agent) or an LLM's conversation entity.",
				Required: true,
			},
			"conversation_language": schema.StringAttribute{
				Description: "Language the conversation agent uses, e.g. `en`, or `*` for an agent that " +
					"handles every language.",
				Required: true,
			},
			"stt_engine": optional("Speech-to-text engine, e.g. the `stt.` entity of a Wyoming " +
				"faster-whisper server. Unset, the pipeline takes no speech."),
			"stt_language":     optional("Language for speech-to-text, e.g. `en`."),
			"tts_engine":       optional("Text-to-speech engine, e.g. `tts.piper`. Unset, the pipeline doesn't speak."),
			"tts_language":     optional("Language for text-to-speech, e.g. `en_US`."),
			"tts_voice":        optional("Text-to-speech voice, e.g. `en_US-lessac-medium`."),
			"wake_word_entity": optional("Wake word engine for streaming wake word detection in Home Assistant, e.g. a `wake_word.` entity. Satellites that detect their wake word on the device don't use it."),
			"wake_word_id":     optional("Wake word the streaming engine listens for, e.g. `ok_nabu`."),
			"prefer_local_intents": schema.BoolAttribute{
				Description: "Whether commands Home Assistant understands itself (\"turn on the lights\") are " +
					"handled locally before the conversation agent is asked. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *assistPipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c := haClient(req, resp)
	if c == nil {
		return
	}
	r.pipelines = client.NewAssistPipelines(c.WS)
}

// fields are every field of the pipeline: Home Assistant requires all of them on create and
// update, an unset one as null.
func (m assistPipelineModel) fields() map[string]any {
	return map[string]any{
		"name":                  m.Name.ValueString(),
		"language":              m.Language.ValueString(),
		"conversation_engine":   m.ConversationEngine.ValueString(),
		"conversation_language": m.ConversationLanguage.ValueString(),
		"stt_engine":            m.STTEngine.ValueStringPointer(),
		"stt_language":          m.STTLanguage.ValueStringPointer(),
		"tts_engine":            m.TTSEngine.ValueStringPointer(),
		"tts_language":          m.TTSLanguage.ValueStringPointer(),
		"tts_voice":             m.TTSVoice.ValueStringPointer(),
		"wake_word_entity":      m.WakeWordEntity.ValueStringPointer(),
		"wake_word_id":          m.WakeWordID.ValueStringPointer(),
		"prefer_local_intents":  m.PreferLocalIntents.ValueBool(),
	}
}

func assistPipelineToModel(pl client.AssistPipeline) assistPipelineModel {
	prefer := false
	if pl.PreferLocalIntents != nil {
		prefer = *pl.PreferLocalIntents
	}
	return assistPipelineModel{
		ID:                   types.StringValue(pl.ID),
		Name:                 types.StringValue(pl.Name),
		Language:             types.StringValue(pl.Language),
		ConversationEngine:   types.StringValue(pl.ConversationEngine),
		ConversationLanguage: types.StringValue(pl.ConversationLanguage),
		STTEngine:            types.StringPointerValue(pl.STTEngine),
		STTLanguage:          types.StringPointerValue(pl.STTLanguage),
		TTSEngine:            types.StringPointerValue(pl.TTSEngine),
		TTSLanguage:          types.StringPointerValue(pl.TTSLanguage),
		TTSVoice:             types.StringPointerValue(pl.TTSVoice),
		WakeWordEntity:       types.StringPointerValue(pl.WakeWordEntity),
		WakeWordID:           types.StringPointerValue(pl.WakeWordID),
		PreferLocalIntents:   types.BoolValue(prefer),
	}
}

func (r *assistPipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan assistPipelineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pl, err := r.pipelines.Create(ctx, plan.fields())
	if err != nil {
		resp.Diagnostics.AddError("Creating Assist pipeline", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, assistPipelineToModel(pl))...)
}

func (r *assistPipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state assistPipelineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pl, ok, err := r.pipelines.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading Assist pipeline", client.ErrorDetail(err))
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, assistPipelineToModel(pl))...)
}

func (r *assistPipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan assistPipelineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pl, err := r.pipelines.Update(ctx, plan.ID.ValueString(), plan.fields())
	if err != nil {
		resp.Diagnostics.AddError("Updating Assist pipeline", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, assistPipelineToModel(pl))...)
}

func (r *assistPipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state assistPipelineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.pipelines.Delete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting Assist pipeline", client.ErrorDetail(err))
	}
}

func (r *assistPipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
