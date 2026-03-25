package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockOrganizationHandler(orgJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/organizations/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, orgJSON)
	})
	return mux
}

const organizationJSON = `{
  "organization/id": "test-org",
  "organization/name": {"en": "Test Organization"},
  "organization/short-name": {"en": "TestOrg"}
}`

func TestOrganizationDataSource_Found(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockOrganizationHandler(organizationJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_organization" "test" {
  id = "test-org"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_organization.test", "id", "test-org"),
				),
			},
		},
	})
}

func TestOrganizationDataSource_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/organizations/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	factories, cleanup := testProviderWithMockServer(t, mux)
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_organization" "test" {
  id = "nonexistent-org"
}`,
				ExpectError: regexp.MustCompile(`Error fetching organization list`),
			},
		},
	})
}
