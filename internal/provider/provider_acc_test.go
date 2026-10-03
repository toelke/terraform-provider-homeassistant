package provider_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
)

func TestMain(m *testing.M) {
	acctest.Main(m)
}

// TestAccHarness_TokenIsAccepted checks the harness itself: the onboarded instance runs the
// requested image and accepts the minted long-lived token.
func TestAccHarness_TokenIsAccepted(t *testing.T) {
	ha := acctest.SharedInstance(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ha.URL+"/api/config", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+ha.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/config: HTTP %d", resp.StatusCode)
	}
	var cfg struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Version != ha.ImageTag {
		t.Errorf("version = %q, want image tag %q", cfg.Version, ha.ImageTag)
	}
}

func TestAccProvider_ConfiguresFromEnvironment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// url and token come from the environment the harness sets.
				Config: acctest.ProviderConfig + `
provider "homeassistant" {}
`,
			},
		},
	})
}
