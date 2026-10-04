# Each key of `steps` answers the config-flow form with that step_id.
resource "homeassistant_integration" "shelly" {
  domain = "shelly"
  steps = {
    user = { host = "192.168.1.50" }
  }
}

# Secrets go into `sensitive_steps`, which is hidden in plan output. Its fields are merged into
# the step of the same step_id.
resource "homeassistant_integration" "shelly_gen2" {
  domain = "shelly"
  steps = {
    user        = { host = "192.168.1.51" }
    credentials = { username = "admin" }
  }
  sensitive_steps = {
    credentials = { password = var.shelly_password }
  }
}

variable "shelly_password" {
  type      = string
  sensitive = true
}
