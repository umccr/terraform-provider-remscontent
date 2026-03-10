package resources_test

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/umccr/terraform-provider-remscontent/internal/provider"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"remscontent": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// providerConfig is a minimal valid provider block used in all test configs.
// The fake endpoint is intentional — ValidateConfig tests never reach Apply.
const providerConfig = `
provider "remscontent" {
  endpoint = "rems.fake.example.com"
  api_user = "test-user"
  api_key  = "test-api-key"
}
`
