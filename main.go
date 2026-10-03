package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/toelke/terraform-provider-homeassistant/internal/provider"
)

//go:generate go tool tfplugindocs generate --provider-name homeassistant

// version is set by the release build.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.opentofu.org/toelke/homeassistant",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
