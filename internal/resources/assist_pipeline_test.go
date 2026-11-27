package resources

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// Home Assistant requires every field on create and update: unset ones go out as null.
func TestAssistPipelineFieldsSendsEveryField(t *testing.T) {
	m := assistPipelineModel{
		Name:                 types.StringValue("Home"),
		Language:             types.StringValue("en"),
		ConversationEngine:   types.StringValue("conversation.home_assistant"),
		ConversationLanguage: types.StringValue("en"),
		STTEngine:            types.StringValue("stt.faster_whisper"),
		STTLanguage:          types.StringNull(),
		TTSEngine:            types.StringNull(),
		TTSLanguage:          types.StringNull(),
		TTSVoice:             types.StringNull(),
		WakeWordEntity:       types.StringNull(),
		WakeWordID:           types.StringNull(),
		PreferLocalIntents:   types.BoolValue(true),
	}
	b, err := json.Marshal(m.fields())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"name":                  "Home",
		"language":              "en",
		"conversation_engine":   "conversation.home_assistant",
		"conversation_language": "en",
		"stt_engine":            "stt.faster_whisper",
		"stt_language":          nil,
		"tts_engine":            nil,
		"tts_language":          nil,
		"tts_voice":             nil,
		"wake_word_entity":      nil,
		"wake_word_id":          nil,
		"prefer_local_intents":  true,
	}
	if len(got) != len(want) {
		t.Errorf("fields sent %d keys, want %d: %s", len(got), len(want), b)
	}
	for k, v := range want {
		if g, ok := got[k]; !ok || g != v {
			t.Errorf("%s = %v (present %v), want %v", k, g, ok, v)
		}
	}
}

func TestAssistPipelineToModel(t *testing.T) {
	stt := "stt.faster_whisper"
	m := assistPipelineToModel(client.AssistPipeline{
		ID: "01home", Name: "Home", Language: "en", ConversationEngine: "conversation.home_assistant",
		ConversationLanguage: "en", STTEngine: &stt,
	})
	if m.ID.ValueString() != "01home" || m.STTEngine.ValueString() != stt || !m.TTSEngine.IsNull() {
		t.Errorf("model = %+v", m)
	}
	// A pipeline without prefer_local_intents reads as the default, false.
	if m.PreferLocalIntents.IsNull() || m.PreferLocalIntents.ValueBool() {
		t.Errorf("prefer_local_intents = %v, want false", m.PreferLocalIntents)
	}
}
