package resources_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// assistPipelines returns the Assist pipeline API of the shared instance, for setting up and
// checking what the provider does.
func assistPipelines(t *testing.T) client.AssistPipelines {
	t.Helper()
	return client.NewAssistPipelines(haClient(t).WS)
}

// preferredAssistPipeline returns the ID of the shared instance's preferred pipeline: onboarding
// creates one and makes it preferred.
func preferredAssistPipeline(t *testing.T) string {
	t.Helper()
	_, preferred, err := assistPipelines(t).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return preferred
}

// checkAssistPipelineGone fails if HA has a pipeline with the given name.
func checkAssistPipelineGone(t *testing.T, name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		pipelines, _, err := assistPipelines(t).List(context.Background())
		if err != nil {
			return err
		}
		for _, p := range pipelines {
			if p.Name == name {
				t.Errorf("pipeline %q (%s) still exists; want it gone", name, p.ID)
			}
		}
		return nil
	}
}

// checkPreferredAssistPipeline fails unless HA's preferred pipeline is the one addr has.
func checkPreferredAssistPipeline(t *testing.T, addr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not in state", addr)
		}
		if got := preferredAssistPipeline(t); got != rs.Primary.ID {
			return fmt.Errorf("preferred pipeline = %q, want %s (%q)", got, addr, rs.Primary.ID)
		}
		return nil
	}
}

func TestAccAssistPipeline(t *testing.T) {
	const addr = "homeassistant_assist_pipeline.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             func(s *terraform.State) error { return checkAssistPipelineGone(t, "Acc Pipeline Renamed")(s) },
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "homeassistant_assist_pipeline" "test" {
  name                  = "Acc Pipeline"
  language              = "en"
  conversation_engine   = "conversation.home_assistant"
  conversation_language = "en"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "id"),
					resource.TestCheckResourceAttr(addr, "name", "Acc Pipeline"),
					resource.TestCheckResourceAttr(addr, "prefer_local_intents", "false"),
					resource.TestCheckNoResourceAttr(addr, "stt_engine"),
					resource.TestCheckNoResourceAttr(addr, "tts_engine"),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Updated in place; the ID stays.
				Config: acctest.ProviderConfig + `
resource "homeassistant_assist_pipeline" "test" {
  name                  = "Acc Pipeline Renamed"
  language              = "en"
  conversation_engine   = "conversation.home_assistant"
  conversation_language = "*"
  prefer_local_intents  = true
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "Acc Pipeline Renamed"),
					resource.TestCheckResourceAttr(addr, "conversation_language", "*"),
					resource.TestCheckResourceAttr(addr, "prefer_local_intents", "true"),
				),
			},
		},
	})
}

func TestAccPreferredAssistPipeline(t *testing.T) {
	const (
		addr     = "homeassistant_preferred_assist_pipeline.test"
		pipeline = "homeassistant_assist_pipeline.test"
	)
	original := preferredAssistPipeline(t)
	pipelineConfig := acctest.ProviderConfig + `
resource "homeassistant_assist_pipeline" "test" {
  name                  = "Acc Preferred Pipeline"
  language              = "en"
  conversation_engine   = "conversation.home_assistant"
  conversation_language = "en"
}
`
	preferredConfig := pipelineConfig + `
resource "homeassistant_preferred_assist_pipeline" "test" {
  pipeline_id = homeassistant_assist_pipeline.test.id
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             func(s *terraform.State) error { return checkAssistPipelineGone(t, "Acc Preferred Pipeline")(s) },
		Steps: []resource.TestStep{
			{
				Config: preferredConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", "preferred"),
					resource.TestCheckResourceAttrPair(addr, "pipeline_id", pipeline, "id"),
					checkPreferredAssistPipeline(t, pipeline),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     "preferred",
				ImportStateVerify: true,
			},
			{
				ResourceName:  addr,
				ImportState:   true,
				ImportStateId: "default",
				ExpectError:   regexp.MustCompile(`imported with the ID "preferred"`),
			},
			{
				// The pipeline preferred elsewhere, e.g. in the UI, shows up as drift, and the apply
				// takes it back.
				PreConfig: func() {
					if err := assistPipelines(t).SetPreferred(context.Background(), original); err != nil {
						t.Fatal(err)
					}
				},
				Config: preferredConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
				},
				Check: checkPreferredAssistPipeline(t, pipeline),
			},
			{
				// HA refuses to delete the preferred pipeline: hand "preferred" back to the original
				// one before the test pipeline is destroyed. Destroying the preferred resource itself
				// changes nothing in HA.
				Config: pipelineConfig + fmt.Sprintf(`
resource "homeassistant_preferred_assist_pipeline" "test" {
  pipeline_id = %q
}
`, original),
				Check: resource.TestCheckResourceAttr(addr, "pipeline_id", original),
			},
		},
	})
}

// The resources are new: no released provider has them yet, so there is no state to upgrade
// from. Enable these with the first release that has them.
func TestAccAssistPipeline_upgrade(t *testing.T) {
	t.Skip("homeassistant_assist_pipeline is not in a release yet")
	acctest.UpgradeTest(t, acctest.ProviderConfig+`
resource "homeassistant_assist_pipeline" "test" {
  name                  = "Acc Pipeline"
  language              = "en"
  conversation_engine   = "conversation.home_assistant"
  conversation_language = "en"
}
`,
		resource.TestCheckResourceAttr("homeassistant_assist_pipeline.test", "name", "Acc Pipeline"),
	)
}

func TestAccPreferredAssistPipeline_upgrade(t *testing.T) {
	t.Skip("homeassistant_preferred_assist_pipeline is not in a release yet")
	acctest.UpgradeTest(t, acctest.ProviderConfig+`
resource "homeassistant_preferred_assist_pipeline" "test" {
  pipeline_id = "`+preferredAssistPipeline(t)+`"
}
`,
		resource.TestCheckResourceAttr("homeassistant_preferred_assist_pipeline.test", "id", "preferred"),
	)
}
