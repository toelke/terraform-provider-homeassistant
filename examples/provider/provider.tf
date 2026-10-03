terraform {
  required_providers {
    homeassistant = {
      source = "toelke/homeassistant"
    }
  }
}

# url and token can also come from HOMEASSISTANT_URL and HOMEASSISTANT_TOKEN.
provider "homeassistant" {
  url   = "http://homeassistant.local:8123"
  token = var.ha_token # long-lived access token of an admin user
}

variable "ha_token" {
  type      = string
  sensitive = true
}
