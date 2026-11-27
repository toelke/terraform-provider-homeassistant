package client

import (
	"context"
	"errors"
)

// AssistPipeline is an Assist pipeline (`assist_pipeline/pipeline/*`): the conversation agent,
// speech-to-text, text-to-speech, and wake word a voice assistant uses. Engines and languages
// that are not set are nil.
type AssistPipeline struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Language             string  `json:"language"`
	ConversationEngine   string  `json:"conversation_engine"`
	ConversationLanguage string  `json:"conversation_language"`
	STTEngine            *string `json:"stt_engine"`
	STTLanguage          *string `json:"stt_language"`
	TTSEngine            *string `json:"tts_engine"`
	TTSLanguage          *string `json:"tts_language"`
	TTSVoice             *string `json:"tts_voice"`
	WakeWordEntity       *string `json:"wake_word_entity"`
	WakeWordID           *string `json:"wake_word_id"`
	PreferLocalIntents   *bool   `json:"prefer_local_intents"`
}

// AssistPipelines manages Assist pipelines over the WebSocket `assist_pipeline/pipeline/*` API.
// Unlike the registries, its list reply is an object that also names the preferred pipeline.
type AssistPipelines struct {
	ws WSCommander
}

// NewAssistPipelines returns the Assist pipeline API on ws.
func NewAssistPipelines(ws WSCommander) AssistPipelines {
	return AssistPipelines{ws: ws}
}

const assistPipelinePrefix = "assist_pipeline/pipeline"

// List returns every pipeline and the ID of the preferred one.
func (p AssistPipelines) List(ctx context.Context) ([]AssistPipeline, string, error) {
	var res struct {
		Pipelines []AssistPipeline `json:"pipelines"`
		Preferred string           `json:"preferred_pipeline"`
	}
	err := p.ws.Command(ctx, assistPipelinePrefix+"/list", nil, &res)
	return res.Pipelines, res.Preferred, err
}

// Get returns the pipeline with the given ID. ok is false if there is none.
func (p AssistPipelines) Get(ctx context.Context, id string) (AssistPipeline, bool, error) {
	pipelines, _, err := p.List(ctx)
	if err != nil {
		return AssistPipeline{}, false, err
	}
	for _, pl := range pipelines {
		if pl.ID == id {
			return pl, true, nil
		}
	}
	return AssistPipeline{}, false, nil
}

// Create adds a pipeline. Home Assistant requires every field: send unset ones as nil.
func (p AssistPipelines) Create(ctx context.Context, fields map[string]any) (AssistPipeline, error) {
	var pl AssistPipeline
	err := p.ws.Command(ctx, assistPipelinePrefix+"/create", fields, &pl)
	return pl, err
}

// Update replaces a pipeline's fields. Like Create, Home Assistant requires every field.
func (p AssistPipelines) Update(ctx context.Context, id string, fields map[string]any) (AssistPipeline, error) {
	params := map[string]any{"pipeline_id": id}
	for k, v := range fields {
		params[k] = v
	}
	var pl AssistPipeline
	err := p.ws.Command(ctx, assistPipelinePrefix+"/update", params, &pl)
	return pl, err
}

// Delete removes a pipeline. A pipeline that is already gone is not an error. Home Assistant
// refuses to delete the preferred pipeline.
func (p AssistPipelines) Delete(ctx context.Context, id string) error {
	err := p.ws.Command(ctx, assistPipelinePrefix+"/delete", map[string]any{"pipeline_id": id}, nil)
	var wsErr *WSError
	if errors.As(err, &wsErr) && wsErr.Code == "not_found" {
		return nil
	}
	return err
}

// SetPreferred makes the pipeline with the given ID the preferred one, which voice assistants
// set to "preferred" use.
func (p AssistPipelines) SetPreferred(ctx context.Context, id string) error {
	return p.ws.Command(ctx, assistPipelinePrefix+"/set_preferred", map[string]any{"pipeline_id": id}, nil)
}
