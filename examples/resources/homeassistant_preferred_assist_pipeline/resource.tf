# The pipeline that voice assistants set to "preferred" use.
resource "homeassistant_assist_pipeline" "home" {
  name                  = "Home Assistant"
  language              = "en"
  conversation_engine   = "conversation.home_assistant"
  conversation_language = "en"
}

resource "homeassistant_preferred_assist_pipeline" "this" {
  pipeline_id = homeassistant_assist_pipeline.home.id
}
