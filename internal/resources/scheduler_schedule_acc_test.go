package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
)

// The Scheduler integration is not installed in this package's Home Assistant. Its own tests are
// in internal/scheduleracc.
const schedulerMissingConfig = acctest.ProviderConfig + `
resource "homeassistant_scheduler_schedule" "test" {
  name      = "Acc Missing"
  timeslots = [{ start = "06:30:00", actions = [{ action = "light.turn_on" }] }]
}
`

var schedulerMissing = regexp.MustCompile(`(?s)Scheduler integration missing.*scheduler-component`)

func TestAccSchedulerSchedule_IntegrationMissing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// The plan fails, not just the apply.
				Config:      schedulerMissingConfig,
				PlanOnly:    true,
				ExpectError: schedulerMissing,
			},
			{
				Config:        schedulerMissingConfig,
				ResourceName:  "homeassistant_scheduler_schedule.test",
				ImportState:   true,
				ImportStateId: "abcdef",
				ExpectError:   schedulerMissing,
			},
			{
				// Everything else keeps working.
				Config: acctest.ProviderConfig + `
resource "homeassistant_label" "test" {
  name = "Acc Without Scheduler"
}
`,
			},
		},
	})
}
