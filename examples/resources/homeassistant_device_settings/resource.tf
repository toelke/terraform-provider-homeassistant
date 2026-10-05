resource "homeassistant_area" "hall" {
  name = "Hall"
}

resource "homeassistant_label" "managed" {
  name = "Managed"
}

data "homeassistant_device" "hall_sensor" {
  name = "Hall Motion Sensor"
}

# Only these attributes are managed. Disabling the device, for example, can still be done in the
# UI.
resource "homeassistant_device_settings" "hall_sensor" {
  device_id    = data.homeassistant_device.hall_sensor.device_id
  name_by_user = "Hall Sensor"
  area_id      = homeassistant_area.hall.id
  labels       = [homeassistant_label.managed.id]
}
