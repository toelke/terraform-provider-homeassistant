package datasources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
)

func TestMain(m *testing.M) {
	acctest.Main(m)
}

func TestAccConfigDataSource(t *testing.T) {
	ha := acctest.SharedInstance(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "homeassistant_config" "this" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_config.this", "version", ha.ImageTag),
					resource.TestCheckResourceAttr("data.homeassistant_config.this", "time_zone", "UTC"),
					resource.TestCheckResourceAttrSet("data.homeassistant_config.this", "unit_system.temperature"),
					resource.TestCheckTypeSetElemAttr("data.homeassistant_config.this", "components.*", "api"),
				),
			},
		},
	})
}
