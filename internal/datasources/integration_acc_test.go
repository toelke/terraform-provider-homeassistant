package datasources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
)

// calendars creates two local_calendar entries, which need no hardware.
const calendars = `
resource "homeassistant_integration" "chores" {
  domain = "local_calendar"
  steps  = { user = { calendar_name = "Acc DS Chores" } }
}

resource "homeassistant_integration" "trash" {
  domain = "local_calendar"
  steps  = { user = { calendar_name = "Acc DS Trash" } }
}
`

func TestAccIntegrationDataSource(t *testing.T) {
	const addr = "data.homeassistant_integration.trash"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + calendars + `
data "homeassistant_integration" "trash" {
  domain = "local_calendar"
  title  = homeassistant_integration.trash.title
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(addr, "id", "homeassistant_integration.trash", "id"),
					resource.TestCheckResourceAttrPair(addr, "entry_id", "homeassistant_integration.trash", "id"),
					resource.TestCheckResourceAttr(addr, "title", "Acc DS Trash"),
					resource.TestCheckResourceAttr(addr, "state", "loaded"),
					resource.TestCheckNoResourceAttr(addr, "disabled_by"),
				),
			},
			{
				// Two entries match the domain alone.
				Config: acctest.ProviderConfig + calendars + `
data "homeassistant_integration" "any" {
  domain = "local_calendar"
}
`,
				ExpectError: regexp.MustCompile(`2 config entries have domain "local_calendar"`),
			},
			{
				Config: acctest.ProviderConfig + `
data "homeassistant_integration" "none" {
  domain = "local_calendar"
  title  = "Acc DS Missing"
}
`,
				ExpectError: regexp.MustCompile(`no config entry has domain "local_calendar" and title "Acc DS Missing"`),
			},
		},
	})
}
