package get

import (
	"context"

	"github.com/hashicorp/go-getter/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		ConfigureContextFunc: configure,
		ResourcesMap: map[string]*schema.Resource{
			"get_artifact": resourceArtifact(),
		},
	}
}

func configure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	client := getter.DefaultClient

	getters := getter.Getters
	client.Getters = getters

	return client, nil
}
