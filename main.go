package main

import (
	"context"
	"flag"
	"os"

	"github.com/cludden/terraform-provider-get/get"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debuggable", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	log := hclog.New(&hclog.LoggerOptions{
		Name:   "terraform-provider-get",
		Output: os.Stderr,
		Color:  hclog.ColorOff,
	})
	hclog.SetDefault(log)

	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/cludden/get",
			&plugin.ServeOpts{
				ProviderFunc: get.Provider,
			})
		if err != nil {
			log.Error(err.Error())
		}
	} else {
		plugin.Serve(&plugin.ServeOpts{
			ProviderFunc: get.Provider,
			Logger:       log,
		})
	}
}
