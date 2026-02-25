package data_sources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/remsclient"
)

// BaseRemsDataSource contains common fields and methods for all REMS data sources
type BaseRemsDataSource struct {
	client *remsclient.APIClient
}

// Configure sets up the API client (called by all data sources)
func (d *BaseRemsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*remsclient.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *remsclient.APIClient, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}
