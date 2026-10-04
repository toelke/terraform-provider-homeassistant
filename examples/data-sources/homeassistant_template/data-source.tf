data "homeassistant_template" "areas" {
  template = "{{ areas() | tojson }}"
}

locals {
  area_ids = jsondecode(data.homeassistant_template.areas.result)
}

output "area_ids" {
  value = local.area_ids
}
