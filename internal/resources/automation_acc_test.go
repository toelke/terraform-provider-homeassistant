package resources_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
	"github.com/toelke/terraform-provider-homeassistant/internal/client"
	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

func automations(t *testing.T) client.ConfigItems {
	t.Helper()
	return client.NewAutomations(haClient(t).REST)
}

// checkAutomationConfig fails unless HA's stored config of the automation, without its id, means
// the same as want.
func checkAutomationConfig(t *testing.T, id, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got, err := automations(t).Get(context.Background(), id)
		if err != nil {
			return err
		}
		eq, err := dyntype.SemanticallyEqualJSON(got, []byte(want))
		if err != nil {
			return err
		}
		if !eq {
			return fmt.Errorf("automation %q config = %s, want %s", id, got, want)
		}
		return nil
	}
}

// checkAutomationAlias fails unless HA's stored config of the automation has the given alias.
func checkAutomationAlias(t *testing.T, id, alias string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		raw, err := automations(t).Get(context.Background(), id)
		if err != nil {
			return err
		}
		var config struct {
			Alias string `json:"alias"`
		}
		if err := json.Unmarshal(raw, &config); err != nil {
			return err
		}
		if config.Alias != alias {
			return fmt.Errorf("automation %q has alias %q, want %q", id, config.Alias, alias)
		}
		return nil
	}
}

func checkAutomationGone(t *testing.T, id string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if _, err := automations(t).Get(context.Background(), id); !errors.Is(err, client.ErrNotFound) {
			return fmt.Errorf("automation %q: err = %w; want it gone", id, err)
		}
		return nil
	}
}

const accAutomation = `
locals {
  notify = {
    action = "persistent_notification.create"
    data   = { title = "Acc", message = "Motion" }
  }
}

resource "homeassistant_automation" "test" {
  id = "acc_motion"
  config = {
    alias = %q
    mode  = "restart"
    triggers = [{
      trigger   = "state"
      entity_id = "sun.sun"
      to        = "below_horizon"
    }]
    conditions = [{
      condition = "time"
      after     = "22:00:00"
      before    = "08:00:00"
    }]
    actions = [local.notify, { delay = { seconds = 1.5 } }]
  }
}
`

func TestAccAutomation(t *testing.T) {
	const addr = "homeassistant_automation.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             checkAutomationGone(t, "acc_motion"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(accAutomation, "Acc Motion"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", "acc_motion"),
					resource.TestCheckResourceAttr(addr, "entity_id", "automation.acc_motion"),
					checkAutomationConfig(t, "acc_motion", `{"alias":"Acc Motion","mode":"restart",
						"triggers":[{"trigger":"state","entity_id":"sun.sun","to":"below_horizon"}],
						"conditions":[{"condition":"time","after":"22:00:00","before":"08:00:00"}],
						"actions":[
							{"action":"persistent_notification.create","data":{"title":"Acc","message":"Motion"}},
							{"delay":{"seconds":1.5}}]}`),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     "acc_motion",
				ImportStateVerify: true,
			},
			{
				// A new alias updates in place and keeps the entity ID.
				Config: acctest.ProviderConfig + fmt.Sprintf(accAutomation, "Acc Motion Renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "entity_id", "automation.acc_motion"),
					checkAutomationAlias(t, "acc_motion", "Acc Motion Renamed"),
				),
			},
			{
				// A change saved in the automation editor shows up as drift, and the apply puts
				// the config back.
				PreConfig: func() {
					if err := automations(t).Save(context.Background(), "acc_motion",
						json.RawMessage(`{"alias":"Edited in the UI","triggers":[],"actions":[]}`)); err != nil {
						t.Fatal(err)
					}
				},
				Config: acctest.ProviderConfig + fmt.Sprintf(accAutomation, "Acc Motion Renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
				},
				Check: checkAutomationConfig(t, "acc_motion", `{"alias":"Acc Motion Renamed","mode":"restart",
					"triggers":[{"trigger":"state","entity_id":"sun.sun","to":"below_horizon"}],
					"conditions":[{"condition":"time","after":"22:00:00","before":"08:00:00"}],
					"actions":[
						{"action":"persistent_notification.create","data":{"title":"Acc","message":"Motion"}},
						{"delay":{"seconds":1.5}}]}`),
			},
			{
				// Deleted outside Tofu: the automation drops out of state and is created again.
				PreConfig: func() {
					if err := automations(t).Delete(context.Background(), "acc_motion"); err != nil {
						t.Fatal(err)
					}
				},
				Config: acctest.ProviderConfig + fmt.Sprintf(accAutomation, "Acc Motion Renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate)},
				},
				Check: resource.TestCheckResourceAttrSet(addr, "entity_id"),
			},
		},
	})
}

// accAutomationYAML is an automation as the automation editor's "Edit in YAML" shows it, minus
// its id line.
const accAutomationYAML = `alias: Acc Away
description: ""
triggers:
  - trigger: state
    entity_id: sun.sun
    to: below_horizon
    for:
      minutes: 5
conditions: []
actions:
  - action: persistent_notification.create
    metadata: {}
    data:
      title: Away
      message: "Sun set at {{ now().strftime('%H:%M') }}"
  - delay: 2.5
mode: single
initial_state: false
`

func TestAccAutomation_YAMLRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "away.yaml")
	if err := os.WriteFile(file, []byte(accAutomationYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	config := acctest.ProviderConfig + fmt.Sprintf(`
resource "homeassistant_automation" "test" {
  id     = "acc_away"
  config = yamldecode(file(%q))
}
`, file)
	const addr = "homeassistant_automation.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             checkAutomationGone(t, "acc_away"),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(addr, "entity_id", "automation.acc_away"),
			},
			{
				// After a refresh, the plan is still empty.
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccAutomation_InvalidConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             checkAutomationGone(t, "acc_invalid"),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "homeassistant_automation" "test" {
  id = "acc_invalid"
  config = {
    alias    = "Acc Invalid"
    triggers = [{ trigger = "no_such_trigger" }]
    actions  = []
  }
}
`,
				ExpectError: regexp.MustCompile(`(?s)HTTP 400.*no_such_trigger`),
			},
		},
	})
}

func TestAccAutomation_IDInConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "homeassistant_automation" "test" {
  id     = "acc_id_in_config"
  config = { id = "other", alias = "Acc", triggers = [], actions = [] }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must not contain\s+an\s+.id.\s+key`),
			},
		},
	})
}
