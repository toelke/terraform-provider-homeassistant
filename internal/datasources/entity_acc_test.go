package datasources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
)

func TestAccEntityDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// `==` is false across types, so these outputs prove that attributes keep their
				// JSON types instead of becoming strings.
				Config: acctest.ProviderConfig + `
data "homeassistant_entity" "sun" {
  entity_id = "sun.sun"
}

locals {
  sun = data.homeassistant_entity.sun.attributes
}

output "elevation_is_number" {
  value = local.sun.elevation == tonumber(local.sun.elevation)
}

output "rising_is_bool" {
  value = local.sun.rising == tobool(local.sun.rising)
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.homeassistant_entity.sun", "state",
						regexp.MustCompile(`^(above|below)_horizon$`)),
					resource.TestCheckResourceAttr("data.homeassistant_entity.sun", "friendly_name", "Sun"),
					resource.TestCheckNoResourceAttr("data.homeassistant_entity.sun", "unit_of_measurement"),
					resource.TestCheckResourceAttrSet("data.homeassistant_entity.sun", "last_changed"),
					resource.TestCheckResourceAttrSet("data.homeassistant_entity.sun", "last_updated"),
					resource.TestCheckOutput("elevation_is_number", "true"),
					resource.TestCheckOutput("rising_is_bool", "true"),
				),
			},
		},
	})
}

func TestAccEntityDataSourceMissing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "homeassistant_entity" "missing" {
  entity_id = "light.does_not_exist"
}
`,
				ExpectError: regexp.MustCompile(`Entity not found(.|\n)*Check the entity ID`),
			},
		},
	})
}
