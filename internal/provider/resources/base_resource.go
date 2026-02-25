// internal/provider/resources/base_resource.go
package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/remsclient"
)

// BaseRemsResource contains common fields and methods for all REMS resources
type BaseRemsResource struct {
	client *remsclient.APIClient
}

// Configure sets up the API client (called by all resources)
func (r *BaseRemsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*remsclient.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *remsclient.APIClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}
