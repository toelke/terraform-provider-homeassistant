package datasources_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// entitiesFixture is what TestAccEntitiesDataSource sets up through the API.
type entitiesFixture struct {
	areaID, areaName   string
	labelID, labelName string
}

// setUpEntities creates two input booleans, a label, and an area whose name needs escaping in a
// template, and assigns them: input_boolean.acc_ents_one is in the area and has the label,
// input_boolean.acc_ents_two only has the label. It also sets the state of a sensor that exists
// only in the state machine, sensor.acc_ents_temp, with a device class.
func setUpEntities(t *testing.T) entitiesFixture {
	t.Helper()
	c := entitiesClient(t)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	booleans := client.Helpers(c.WS, "input_boolean")
	for _, name := range []string{"Acc Ents One", "Acc Ents Two"} {
		h, err := booleans.Create(ctx, map[string]any{"name": name})
		must(err)
		t.Cleanup(func() { _ = booleans.Delete(context.Background(), h.HelperID()) })
	}

	labels := client.Labels(c.WS)
	label, err := labels.Create(ctx, map[string]any{"name": "Acc Ents Label"})
	must(err)
	t.Cleanup(func() { _ = labels.Delete(context.Background(), label.LabelID) })

	var area struct {
		AreaID string `json:"area_id"`
		Name   string `json:"name"`
	}
	must(c.WS.Command(ctx, "config/area_registry/create", map[string]any{"name": `Acc "Ents" Room's {{ x }}`}, &area))
	t.Cleanup(func() {
		_ = c.WS.Command(context.Background(), "config/area_registry/delete", map[string]any{"area_id": area.AreaID}, nil)
	})

	updateEntity(t, c, map[string]any{
		"entity_id": "input_boolean.acc_ents_one", "area_id": area.AreaID, "labels": []string{label.LabelID},
	})
	updateEntity(t, c, map[string]any{"entity_id": "input_boolean.acc_ents_two", "labels": []string{label.LabelID}})

	must(c.REST.Do(ctx, http.MethodPost, "states/sensor.acc_ents_temp", map[string]any{
		"state": "21.5",
		"attributes": map[string]any{
			"friendly_name": "Acc Ents Temp", "device_class": "temperature", "unit_of_measurement": "°C",
		},
	}, nil))
	t.Cleanup(func() {
		_ = c.REST.Do(context.Background(), http.MethodDelete, "states/sensor.acc_ents_temp", nil, nil)
	})

	return entitiesFixture{areaID: area.AreaID, areaName: area.Name, labelID: label.LabelID, labelName: label.Name}
}

// entitiesClient returns a client for the shared HA instance.
func entitiesClient(t *testing.T) *client.HAClient {
	t.Helper()
	ha := acctest.SharedInstance(t)
	u, err := url.Parse(ha.URL)
	if err != nil {
		t.Fatal(err)
	}
	return client.New(client.Config{URL: u, Token: ha.Token, Timeout: 30 * time.Second})
}

// updateEntity updates an entity registry entry. A new helper's entity may reach the entity
// registry only shortly after the create returns, so it retries for a while.
func updateEntity(t *testing.T, c *client.HAClient, fields map[string]any) {
	t.Helper()
	var err error
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		if err = c.WS.Command(context.Background(), "config/entity_registry/update", fields, nil); err == nil {
			return
		}
	}
	t.Fatal(err)
}

// setUpHiddenEntities creates an area with two input booleans that both have the friendly name
// "Acc Hidden Lamp": input_boolean.acc_hidden_old is hidden by the user,
// input_boolean.acc_hidden_new is not. It returns the area ID.
func setUpHiddenEntities(t *testing.T) string {
	t.Helper()
	c := entitiesClient(t)
	ctx := context.Background()

	var area struct {
		AreaID string `json:"area_id"`
	}
	if err := c.WS.Command(ctx, "config/area_registry/create", map[string]any{"name": "Acc Hidden Room"}, &area); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.WS.Command(context.Background(), "config/area_registry/delete", map[string]any{"area_id": area.AreaID}, nil)
	})

	booleans := client.Helpers(c.WS, "input_boolean")
	for _, name := range []string{"Acc Hidden Old", "Acc Hidden New"} {
		h, err := booleans.Create(ctx, map[string]any{"name": name})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = booleans.Delete(context.Background(), h.HelperID()) })
	}
	updateEntity(t, c, map[string]any{
		"entity_id": "input_boolean.acc_hidden_old", "area_id": area.AreaID, "name": "Acc Hidden Lamp", "hidden_by": "user",
	})
	updateEntity(t, c, map[string]any{
		"entity_id": "input_boolean.acc_hidden_new", "area_id": area.AreaID, "name": "Acc Hidden Lamp",
	})
	return area.AreaID
}

func TestAccEntitiesDataSourceHidden(t *testing.T) {
	areaID := setUpHiddenEntities(t)
	// sensor.acc_hidden_temp exists only in the state machine, so it has no registry entry.
	c := entitiesClient(t)
	if err := c.REST.Do(context.Background(), http.MethodPost, "states/sensor.acc_hidden_temp", map[string]any{
		"state": "1", "attributes": map[string]any{"friendly_name": "Acc Hidden Temp"},
	}, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.REST.Do(context.Background(), http.MethodDelete, "states/sensor.acc_hidden_temp", nil, nil)
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
data "homeassistant_entities" "visible" {
  area         = %[1]q
  name_pattern = "Acc Hidden Lamp"
  hidden       = false
}

data "homeassistant_entities" "hidden" {
  area         = %[1]q
  name_pattern = "Acc Hidden Lamp"
  hidden       = true
}

data "homeassistant_entities" "both" {
  area         = %[1]q
  name_pattern = "Acc Hidden Lamp"
}

data "homeassistant_entities" "unregistered" {
  name_pattern = "Acc Hidden Temp"
  hidden       = false
}

output "old_hidden" {
  value = data.homeassistant_entities.both.entities["input_boolean.acc_hidden_old"].hidden
}

output "new_hidden" {
  value = data.homeassistant_entities.both.entities["input_boolean.acc_hidden_new"].hidden
}

output "unregistered_hidden" {
  value = data.homeassistant_entities.unregistered.entities["sensor.acc_hidden_temp"].hidden
}
`, areaID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkEntityIDs("visible", "input_boolean.acc_hidden_new"),
					checkEntityIDs("hidden", "input_boolean.acc_hidden_old"),
					checkEntityIDs("both", "input_boolean.acc_hidden_new", "input_boolean.acc_hidden_old"),
					checkEntityIDs("unregistered", "sensor.acc_hidden_temp"),
					resource.TestCheckOutput("old_hidden", "true"),
					resource.TestCheckOutput("new_hidden", "false"),
					resource.TestCheckOutput("unregistered_hidden", "false"),
				),
			},
		},
	})
}

func TestAccEntitiesDataSource(t *testing.T) {
	f := setUpEntities(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
data "homeassistant_entities" "by_domain_and_name" {
  domain       = "input_boolean"
  name_pattern = "Acc Ents *"
}

data "homeassistant_entities" "area_by_name" {
  area = %[1]q
}

data "homeassistant_entities" "area_by_id" {
  area = %[2]q
}

data "homeassistant_entities" "label_by_name" {
  label = %[3]q
}

data "homeassistant_entities" "label_by_id" {
  label  = %[4]q
  domain = "input_boolean"
}

data "homeassistant_entities" "device_class" {
  device_class = "temperature"
  name_pattern = "Acc*"
}

data "homeassistant_entities" "unknown_area" {
  area = "no such area"
}

locals {
  temp = data.homeassistant_entities.device_class.entities["sensor.acc_ents_temp"]
}

output "temp_state" {
  value = local.temp.state
}

output "temp_friendly_name" {
  value = local.temp.friendly_name
}

output "temp_unit" {
  value = local.temp.attributes.unit_of_measurement
}

output "entity_count" {
  value = length(data.homeassistant_entities.label_by_name.entities)
}
`, f.areaName, f.areaID, f.labelName, f.labelID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkEntityIDs("by_domain_and_name", "input_boolean.acc_ents_one", "input_boolean.acc_ents_two"),
					checkEntityIDs("area_by_name", "input_boolean.acc_ents_one"),
					checkEntityIDs("area_by_id", "input_boolean.acc_ents_one"),
					checkEntityIDs("label_by_name", "input_boolean.acc_ents_one", "input_boolean.acc_ents_two"),
					checkEntityIDs("label_by_id", "input_boolean.acc_ents_one", "input_boolean.acc_ents_two"),
					checkEntityIDs("device_class", "sensor.acc_ents_temp"),
					checkEntityIDs("unknown_area"),
					resource.TestCheckOutput("temp_state", "21.5"),
					resource.TestCheckOutput("temp_friendly_name", "Acc Ents Temp"),
					resource.TestCheckOutput("temp_unit", "°C"),
					resource.TestCheckOutput("entity_count", "2"),
				),
			},
		},
	})
}

// checkEntityIDs checks that data.homeassistant_entities.<name> found exactly ids, in order.
func checkEntityIDs(name string, ids ...string) resource.TestCheckFunc {
	address := "data.homeassistant_entities." + name
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(address, "entity_ids.#", fmt.Sprint(len(ids))),
	}
	for i, id := range ids {
		checks = append(checks, resource.TestCheckResourceAttr(address, fmt.Sprintf("entity_ids.%d", i), id))
	}
	return resource.ComposeAggregateTestCheckFunc(checks...)
}
