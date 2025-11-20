package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/iilei/terraform-provider-jsonschema/internal/provider"
)

// These variables are set by GoReleaser via ldflags
var (
	version = "dev"  // Default value, gets overwritten by -ldflags
	commit  = "none" // Default value, gets overwritten by -ldflags
)

func main() {
	var debugMode bool
	var showVersion bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.Parse()

	if showVersion {
		log.Printf("terraform-provider-jsonschema version %s (commit: %s)", version, commit)
		return
	}

	opts := &plugin.ServeOpts{ProviderFunc: provider.New(version)}

	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/providers/iilei/jsonschema", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
