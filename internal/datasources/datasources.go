// Package datasources holds the provider's data sources.
package datasources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var entityIDPath = path.Root("entity_id")

// restClient returns the REST client from the provider data, or nil before the provider is
// configured or on error.
func restClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.RESTClient {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return nil
	}
	return c.REST
}
