package data_sources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/umccr/terraform-provider-remscontent/internal/provider/shared"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// BaseRemsDataSource contains common fields and methods for all REMS data sources.
type BaseRemsDataSource struct {
	client   *remsclient.ClientWithResponses
	language string
}

// Configure sets up the API client (called by all data sources).
func (d *BaseRemsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cfg, ok := req.ProviderData.(*shared.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *shared.ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	d.client = cfg.Client
	d.language = cfg.Language
}
