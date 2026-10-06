// Package resources holds the provider's resources.
package resources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// haClient returns the client from the provider data, or nil before the provider is configured
// or on error.
func haClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.HAClient {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return nil
	}
	return c
}
