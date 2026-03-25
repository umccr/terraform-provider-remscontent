package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockResourceHandler(resourcesJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, resourcesJSON)
	})
	return mux
}

const resourcesListJSON = `[
  {
    "id": 1, "resid": "urn:test:001", "archived": false, "enabled": true, "licenses": [],
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  },
  {
    "id": 2, "resid": "urn:test:002", "archived": false, "enabled": true, "licenses": [],
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  }
]`

func TestResourceDataSource_FindByResid(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(resourcesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_resource" "test" {
  resource_ext_id = "urn:test:001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_resource.test", "resource_ext_id", "urn:test:001"),
					resource.TestCheckResourceAttr("data.remscontent_resource.test", "id", "1"),
				),
			},
		},
	})
}

func TestResourceDataSource_FindSecondResource(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(resourcesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_resource" "test" {
  resource_ext_id = "urn:test:002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_resource.test", "resource_ext_id", "urn:test:002"),
					resource.TestCheckResourceAttr("data.remscontent_resource.test", "id", "2"),
				),
			},
		},
	})
}

func TestResourceDataSource_MultipleResidsError(t *testing.T) {
	duplicatesJSON := `[
  {
    "id": 1, "resid": "urn:test:dup", "archived": false, "enabled": true, "licenses": [],
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  },
  {
    "id": 2, "resid": "urn:test:dup", "archived": false, "enabled": true, "licenses": [],
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  }
]`

	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(duplicatesJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_resource" "test" {
  resource_ext_id = "urn:test:dup"
}`,
				ExpectError: regexp.MustCompile(`Multiple Resource Found`),
			},
		},
	})
}

func TestResourceDataSource_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resources", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})

	factories, cleanup := testProviderWithMockServer(t, mux)
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_resource" "test" {
  resource_ext_id = "urn:test:001"
}`,
				ExpectError: regexp.MustCompile(`Error fetching resources list`),
			},
		},
	})
}

func TestResourceDataSource_NotFound(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(resourcesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_resource" "test" {
  resource_ext_id = "invalid:ext:id"
}`,
				ExpectError: regexp.MustCompile(`Resource Not Found`),
			},
		},
	})
}
