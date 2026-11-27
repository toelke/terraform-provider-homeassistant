# Assist

Assist pipelines (Settings > Voice assistants), over the WebSocket `assist_pipeline/pipeline/*`
API. The list reply has every pipeline and the ID of the preferred one, so both resources read
everything back and drift detection works.

## `homeassistant_assist_pipeline`

One pipeline: the conversation agent, speech-to-text, text-to-speech, and wake word that a voice
assistant runs a request through. HA assigns the `id`, a ULID, which is also the import ID. HA
requires every field on `create` and `update`, so the provider sends all of them, unset ones as
`null`. HA refuses to delete the preferred pipeline: make another one preferred first.

```hcl
resource "homeassistant_assist_pipeline" "home" {
  name                  = "Home Assistant"
  language              = "en"
  conversation_engine   = "conversation.home_assistant"
  conversation_language = "en"
  prefer_local_intents  = true
  stt_engine            = "stt.faster_whisper"
  stt_language          = "en"
  tts_engine            = "tts.piper"
  tts_language          = "en_US"
  tts_voice             = "en_US-lessac-high"
}
```

| Argument | Type | |
|---|---|---|
| `name` | string, required | |
| `language` | string, required | e.g. `en` |
| `conversation_engine` | string, required | e.g. `conversation.home_assistant`, HA's own agent |
| `conversation_language` | string, required | `*` for an agent that handles every language |
| `stt_engine`, `stt_language` | string | Unset: the pipeline takes no speech |
| `tts_engine`, `tts_language`, `tts_voice` | string | Unset: the pipeline doesn't speak |
| `wake_word_entity`, `wake_word_id` | string | Streaming wake word detection in HA |
| `prefer_local_intents` | bool, default `false` | HA's own intents before the conversation agent |

## `homeassistant_preferred_assist_pipeline`

A singleton: `id` is always `preferred`, which is also the only import ID. `pipeline_id` names the
pipeline that voice assistants set to "preferred" use (`set_preferred`). HA always has a
preferred pipeline, so destroying the resource changes nothing in HA.

```hcl
resource "homeassistant_preferred_assist_pipeline" "this" {
  pipeline_id = homeassistant_assist_pipeline.home.id
}
```
