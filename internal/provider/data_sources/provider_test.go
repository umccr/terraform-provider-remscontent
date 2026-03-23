package data_sources_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/umccr/terraform-provider-remscontent/internal/provider"
	remsclient "github.com/umccr/terraform-provider-remscontent/internal/rems-client"
)

// providerConfig is a minimal valid provider block used in all test configs.
// The fake endpoint is intentional — ValidateConfig tests never reach Apply.
const providerConfig = `
provider "remscontent" {
  endpoint = "rems.fake.example.com"
  api_user = "test-user"
  api_key  = "test-api-key"
}
`

func testProviderWithMockServer(t *testing.T, handler http.Handler) (providerFactories map[string]func() (tfprotov6.ProviderServer, error), cleanup func()) {
	t.Helper()
	srv := httptest.NewServer(handler)

	client, err := remsclient.NewClientWithResponses(
		srv.URL,
		remsclient.WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"remscontent": providerserver.NewProtocol6WithError(
			provider.NewWithClient("test", client)(),
		),
	}
	return factories, srv.Close
}
