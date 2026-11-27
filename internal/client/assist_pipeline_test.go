package client

import (
	"context"
	"errors"
	"testing"
)

func TestAssistPipelinesList(t *testing.T) {
	p := NewAssistPipelines(fakeWS{"assist_pipeline/pipeline/list": fixed(`{
		"pipelines": [
			{"id": "01home", "name": "Home Assistant", "language": "en",
			 "conversation_engine": "conversation.home_assistant", "conversation_language": "en",
			 "stt_engine": "stt.faster_whisper", "stt_language": "en", "tts_engine": null,
			 "tts_language": null, "tts_voice": null, "wake_word_entity": null, "wake_word_id": null,
			 "prefer_local_intents": true},
			{"id": "01other", "name": "Other", "language": "de",
			 "conversation_engine": "conversation.home_assistant", "conversation_language": "de",
			 "stt_engine": null, "stt_language": null, "tts_engine": null, "tts_language": null,
			 "tts_voice": null, "wake_word_entity": null, "wake_word_id": null}
		],
		"preferred_pipeline": "01home"
	}`)})

	pipelines, preferred, err := p.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pipelines) != 2 || preferred != "01home" {
		t.Fatalf("List = %d pipelines, preferred %q", len(pipelines), preferred)
	}

	home, ok, err := p.Get(context.Background(), "01home")
	if err != nil || !ok {
		t.Fatalf("Get(01home) = %+v, %v, %v", home, ok, err)
	}
	if home.STTEngine == nil || *home.STTEngine != "stt.faster_whisper" || home.TTSEngine != nil ||
		home.PreferLocalIntents == nil || !*home.PreferLocalIntents {
		t.Errorf("01home = %+v", home)
	}

	// A pipeline without prefer_local_intents leaves it nil.
	other, ok, err := p.Get(context.Background(), "01other")
	if err != nil || !ok || other.PreferLocalIntents != nil {
		t.Errorf("Get(01other) = %+v, %v, %v", other, ok, err)
	}

	if _, ok, err := p.Get(context.Background(), "missing"); err != nil || ok {
		t.Errorf("Get(missing) = %v, %v; want not found", ok, err)
	}
}

func TestAssistPipelinesWrite(t *testing.T) {
	sent := map[string]map[string]any{}
	record := func(typ, reply string) func(map[string]any) (any, error) {
		return func(params map[string]any) (any, error) {
			sent[typ] = params
			return fixed(reply)(params)
		}
	}
	pl := `{"id": "01new", "name": "New", "language": "en", "conversation_engine": "conversation.home_assistant",
		"conversation_language": "en"}`
	p := NewAssistPipelines(fakeWS{
		"assist_pipeline/pipeline/create":        record("create", pl),
		"assist_pipeline/pipeline/update":        record("update", pl),
		"assist_pipeline/pipeline/set_preferred": record("set_preferred", `null`),
		"assist_pipeline/pipeline/delete": func(params map[string]any) (any, error) {
			sent["delete"] = params
			switch params["pipeline_id"] {
			case "gone":
				return nil, &WSError{Code: "not_found", Message: "Unable to find pipeline_id gone"}
			case "preferred":
				return nil, &WSError{Code: "not_allowed", Message: "Item 01home preferred."}
			}
			return nil, nil
		},
	})
	ctx := context.Background()
	fields := map[string]any{"name": "New", "stt_engine": nil}

	created, err := p.Create(ctx, fields)
	if err != nil || created.ID != "01new" {
		t.Fatalf("Create = %+v, %v", created, err)
	}
	if v, ok := sent["create"]["stt_engine"]; !ok || v != nil {
		t.Errorf("create sent stt_engine = %v, %v; want it present and nil", v, ok)
	}

	if _, err := p.Update(ctx, "01new", fields); err != nil {
		t.Fatal(err)
	}
	if sent["update"]["pipeline_id"] != "01new" || sent["update"]["name"] != "New" {
		t.Errorf("update sent %v", sent["update"])
	}
	if v, ok := sent["update"]["stt_engine"]; !ok || v != nil {
		t.Errorf("update sent stt_engine = %v, %v; want it present and nil", v, ok)
	}
	if _, ok := fields["pipeline_id"]; ok {
		t.Error("Update changed the caller's fields")
	}

	if err := p.SetPreferred(ctx, "01new"); err != nil || sent["set_preferred"]["pipeline_id"] != "01new" {
		t.Errorf("SetPreferred: err %v, sent %v", err, sent["set_preferred"])
	}

	if err := p.Delete(ctx, "01new"); err != nil {
		t.Errorf("Delete(01new) = %v", err)
	}
	if err := p.Delete(ctx, "gone"); err != nil {
		t.Errorf("Delete(gone) = %v; want nil for a pipeline that is already gone", err)
	}
	var wsErr *WSError
	if err := p.Delete(ctx, "preferred"); !errors.As(err, &wsErr) || wsErr.Code != "not_allowed" {
		t.Errorf("Delete(preferred) = %v; want HA's not_allowed", err)
	}
}
