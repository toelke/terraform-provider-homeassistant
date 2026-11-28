package datasources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
)

// The Scheduler integration is not installed in this package's Home Assistant. The data source's
// own tests are in internal/scheduleracc.
func TestAccSchedulerSchedules_IntegrationMissing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      acctest.ProviderConfig + `data "homeassistant_scheduler_schedules" "all" {}`,
			ExpectError: regexp.MustCompile(`(?s)Scheduler integration missing.*scheduler-component`),
		}},
	})
}
