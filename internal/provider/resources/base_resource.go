// internal/provider/resources/base_resource.go
package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/umccr/terraform-provider-remscontent/internal/provider/shared"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// BaseRemsResource contains common fields and methods for all REMS resources
type BaseRemsResource struct {
	client   *remsclient.ClientWithResponses
	language string
}

// Configure sets up the API client (called by all resources)
func (r *BaseRemsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cfg, ok := req.ProviderData.(*shared.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *shared.ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	r.client = cfg.Client
	r.language = cfg.Language
}
