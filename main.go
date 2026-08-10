package main

import (
	"github.com/espressocomputing/terraform-provider-espresso/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() { plugin.Serve(&plugin.ServeOpts{ProviderFunc: provider.New}) }
