# A voice pipeline: Home Assistant's own intents first, then an LLM conversation agent, with
# speech-to-text and text-to-speech from Wyoming servers (Whisper and Piper).
resource "homeassistant_assist_pipeline" "home" {
  name                  = "Home Assistant"
  language              = "en"
  conversation_engine   = "conversation.ollama"
  conversation_language = "en"
  prefer_local_intents  = true
  stt_engine            = "stt.faster_whisper"
  stt_language          = "en"
  tts_engine            = "tts.piper"
  tts_language          = "en_US"
  tts_voice             = "en_US-lessac-high"
}
