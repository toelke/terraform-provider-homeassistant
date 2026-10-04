package datasources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
)

func TestAccTemplateDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "homeassistant_template" "sum" {
  template = "{{ 1 + 1 }}"
}

data "homeassistant_template" "json" {
  template = "{{ {'sun': states('sun.sun') != 'unknown'} | tojson }}"
}

output "sun_known" {
  value = jsondecode(data.homeassistant_template.json.result).sun
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_template.sum", "result", "2"),
					resource.TestCheckOutput("sun_known", "true"),
				),
			},
		},
	})
}

func TestAccTemplateDataSourceRenderError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "homeassistant_template" "broken" {
  template = "{{ 1 + }}"
}
`,
				ExpectError: regexp.MustCompile(`Rendering template failed`),
			},
		},
	})
}
