package resources_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/toelke/terraform-provider-homeassistant/internal/acctest"
	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

func TestMain(m *testing.M) {
	acctest.Main(m)
}

// haClient returns a client for the shared instance, for setting up and checking what the
// provider does.
func haClient(t *testing.T) *client.HAClient {
	t.Helper()
	ha := acctest.SharedInstance(t)
	u, err := url.Parse(ha.URL)
	if err != nil {
		t.Fatal(err)
	}
	return client.New(client.Config{URL: u, Token: ha.Token, Timeout: 30 * time.Second})
}
