package scheduleracc_test

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

const scheduleAddr = "homeassistant_scheduler_schedule.test"

// scheduleConfig is a heating schedule with a fixed and a sun-relative timeslot. extra goes into
// the resource block.
func scheduleConfig(name, weekdays, temperature, extra string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
resource "homeassistant_scheduler_schedule" "test" {
  name     = %q
  weekdays = %s
  timeslots = [
    {
      start = "06:30:00"
      stop  = "08:00:00"
      actions = [{
        action    = "climate.set_temperature"
        entity_id = "climate.office"
        data      = jsonencode({ temperature = %s })
      }]
    },
    {
      start          = "sunset-00:30:00"
      condition_type = "and"
      conditions = [{
        entity_id  = "sun.sun"
        attribute  = "elevation"
        match_type = "below"
        value      = "5"
      }]
      actions = [{ action = "light.turn_on" }]
    },
  ]
%s
}
`, name, weekdays, temperature, extra)
}

// checkSchedule checks the schedule named name in Home Assistant, once it has caught up.
func checkSchedule(t *testing.T, name string, check func(client.SchedulerSchedule) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		acctest.Eventually(t, 10*time.Second, "schedule "+name, func() (bool, error) {
			s, ok := scheduleNamed(t, name)
			if !ok {
				return false, fmt.Errorf("no schedule named %q", name)
			}
			return check(s) == nil, check(s)
		})
		return nil
	}
}

func TestAccSchedulerSchedule(t *testing.T) {
	const (
		name    = "Acc Heating"
		renamed = "Acc Heating Office"
	)
	var scheduleID string
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			for _, n := range []string{name, renamed} {
				if _, ok := scheduleNamed(t, n); ok {
					return fmt.Errorf("schedule %q still exists", n)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: scheduleConfig(name, `["mon", "tue", "wed", "thu", "fri"]`, "21", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(scheduleAddr, "schedule_id", regexp.MustCompile(`^[0-9a-f]{6}$`)),
					resource.TestCheckResourceAttr(scheduleAddr, "entity_id", "switch.schedule_acc_heating"),
					resource.TestCheckResourceAttr(scheduleAddr, "repeat_type", "repeat"),
					resource.TestCheckResourceAttr(scheduleAddr, "tags.#", "0"),
					resource.TestCheckNoResourceAttr(scheduleAddr, "enabled"),
					resource.TestCheckResourceAttr(scheduleAddr, "timeslots.0.actions.0.data", `{"temperature":21}`),
					resource.TestCheckResourceAttr(scheduleAddr, "timeslots.1.track_conditions", "false"),
					func(s *terraform.State) error {
						scheduleID = s.RootModule().Resources[scheduleAddr].Primary.Attributes["schedule_id"]
						return nil
					},
					checkSchedule(t, name, func(s client.SchedulerSchedule) error {
						ts := s.Timeslots
						if len(ts) != 2 || ts[1].Start != "sunset-00:30:00" || ts[0].Actions[0].Service != "climate.set_temperature" ||
							string(ts[1].Conditions[0].Value) != "5" || !s.Enabled {
							return fmt.Errorf("in Home Assistant: %+v", s)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:                         scheduleAddr,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "schedule_id",
				ImportStateIdFunc:                    func(*terraform.State) (string, error) { return scheduleID, nil },
			},
			{
				// Everything but the name changes in place, and enabled is managed now.
				Config: scheduleConfig(name, `["daily"]`, "20.5", `
  tags        = ["heating", "office"]
  repeat_type = "pause"
  start_date  = "2026-01-01"
  end_date    = "2099-12-31"
  enabled     = false
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(scheduleAddr, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(scheduleAddr, "schedule_id", sameAs(&scheduleID)),
					resource.TestCheckResourceAttr(scheduleAddr, "entity_id", "switch.schedule_acc_heating"),
					resource.TestCheckResourceAttr(scheduleAddr, "enabled", "false"),
					checkSchedule(t, name, func(s client.SchedulerSchedule) error {
						if !slices.Equal(s.Weekdays, []string{"daily"}) || !slices.Equal(s.Tags, []string{"heating", "office"}) ||
							s.Enabled || s.RepeatType != "pause" || s.EndDate == nil || *s.EndDate != "2099-12-31" ||
							!strings.Contains(string(s.Timeslots[0].Actions[0].ServiceData), "20.5") {
							return fmt.Errorf("in Home Assistant: %+v", s)
						}
						return nil
					}),
				),
			},
			{
				// Edits in the card, and turning the switch on, are drift that apply reverts.
				PreConfig: func() {
					s, ok := scheduleNamed(t, name)
					if !ok {
						t.Fatal("schedule missing")
					}
					ctx := context.Background()
					if err := schedulers(t).Edit(ctx, s.EntityID, map[string]any{"weekdays": []string{"weekend"}}); err != nil {
						t.Fatal(err)
					}
					if err := schedulers(t).SetEnabled(ctx, s.EntityID, true); err != nil {
						t.Fatal(err)
					}
					acctest.Eventually(t, 10*time.Second, "edit in Home Assistant", func() (bool, error) {
						s, _ := scheduleNamed(t, name)
						return s.Enabled && slices.Equal(s.Weekdays, []string{"weekend"}), nil
					})
				},
				Config: scheduleConfig(name, `["daily"]`, "20.5", `
  tags        = ["heating", "office"]
  repeat_type = "pause"
  start_date  = "2026-01-01"
  end_date    = "2099-12-31"
  enabled     = false
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(scheduleAddr, plancheck.ResourceActionUpdate)},
				},
				Check: checkSchedule(t, name, func(s client.SchedulerSchedule) error {
					if s.Enabled || !slices.Equal(s.Weekdays, []string{"daily"}) {
						return fmt.Errorf("in Home Assistant: %+v", s)
					}
					return nil
				}),
			},
			{
				// Without enabled, switching it is no drift.
				PreConfig: func() {
					s, _ := scheduleNamed(t, name)
					if err := schedulers(t).SetEnabled(context.Background(), s.EntityID, true); err != nil {
						t.Fatal(err)
					}
				},
				Config: scheduleConfig(name, `["daily"]`, "20.5", `
  tags        = ["heating", "office"]
  repeat_type = "pause"
  start_date  = "2026-01-01"
  end_date    = "2099-12-31"
`),
				Check: resource.TestCheckNoResourceAttr(scheduleAddr, "enabled"),
			},
			{
				// A new name replaces the schedule.
				Config: scheduleConfig(renamed, `["daily"]`, "20.5", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(scheduleAddr, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(scheduleAddr, "entity_id", "switch.schedule_acc_heating_office"),
					func(*terraform.State) error {
						if _, ok := scheduleNamed(t, name); ok {
							return fmt.Errorf("old schedule %q still exists", name)
						}
						return nil
					},
				),
			},
			{
				// Deleted outside Tofu: the schedule drops out of state and is created again.
				PreConfig: func() {
					s, ok := scheduleNamed(t, renamed)
					if !ok {
						t.Fatal("schedule missing")
					}
					if err := schedulers(t).Remove(context.Background(), s.EntityID); err != nil {
						t.Fatal(err)
					}
				},
				Config: scheduleConfig(renamed, `["daily"]`, "20.5", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(scheduleAddr, plancheck.ResourceActionCreate)},
				},
			},
		},
	})
}

// sameAs checks that an attribute still has the value *want.
func sameAs(want *string) resource.CheckResourceAttrWithFunc {
	return func(v string) error {
		if v != *want {
			return fmt.Errorf("= %q, want %q", v, *want)
		}
		return nil
	}
}

func TestAccSchedulerSchedule_Invalid(t *testing.T) {
	for _, tc := range []struct{ extra, want string }{
		{`repeat_type = "single"`, `repeat_type "single" is not supported`},
		{`start_date = "2026-02-01"` + "\n" + `end_date = "2026-01-01"`, `end_date must not be before start_date`},
	} {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { preCheck(t) },
			ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
			Steps: []resource.TestStep{{
				Config:      scheduleConfig("Acc Invalid", `["daily"]`, "21", tc.extra),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(tc.want)),
			}},
		})
	}
}

func TestAccSchedulerSchedules(t *testing.T) {
	const ds = "data.homeassistant_scheduler_schedules.all"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: scheduleConfig("Acc Listed", `["sat", "sun"]`, "19", `  tags = ["listed"]`) + `
data "homeassistant_scheduler_schedules" "all" {
  depends_on = [homeassistant_scheduler_schedule.test]
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckTypeSetElemNestedAttrs(ds, "schedules.*", map[string]string{
					"name":       "Acc Listed",
					"entity_id":  "switch.schedule_acc_listed",
					"enabled":    "true",
					"weekdays.#": "2",
					"weekdays.0": "sat",
					"tags.#":     "1",
					"tags.0":     "listed",
					"entities.#": "1",
					"entities.0": "climate.office",
				}),
				resource.TestCheckResourceAttrPair(ds, "schedules.0.schedule_id", scheduleAddr, "schedule_id"),
			),
		}},
	})
}
