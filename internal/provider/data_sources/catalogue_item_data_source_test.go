package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockCatalogueItemHandler(itemsJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/catalogue-items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, itemsJSON)
	})
	return mux
}

const catalogueItemsListJSON = `[
  {
    "id": 1, "resid": "urn:test:001", "archived": false, "enabled": true, "expired": false,
    "start": "2024-01-01T00:00:00Z", "end": null, "wfid": 10, "resource-id": 100,
    "organization": {"organization/id": "org1"},
    "localizations": {"en": {"id": 1, "langcode": "en", "title": "First Item", "infourl": null}}
  },
  {
    "id": 2, "resid": "urn:test:002", "archived": false, "enabled": true, "expired": false,
    "start": "2024-01-01T00:00:00Z", "end": null, "wfid": 10, "resource-id": 101,
    "organization": {"organization/id": "org1"},
    "localizations": {"en": {"id": 2, "langcode": "en", "title": "Second Item", "infourl": null}}
  }
]`

func TestCatalogueItemDataSource_FindByTitle(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(catalogueItemsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_catalogue_item" "test" {
  title = "Second Item"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_catalogue_item.test", "title", "Second Item"),
					resource.TestCheckResourceAttr("data.remscontent_catalogue_item.test", "id", "2"),
				),
			},
		},
	})
}

func TestCatalogueItemDataSource_MultipleTitlesError(t *testing.T) {
	duplicatesJSON := `[
  {
    "id": 1, "resid": "urn:test:001", "archived": false, "enabled": true, "expired": false,
    "start": "2024-01-01T00:00:00Z", "end": null, "wfid": 10, "resource-id": 100,
    "organization": {"organization/id": "org1"},
    "localizations": {"en": {"id": 1, "langcode": "en", "title": "Duplicate Item", "infourl": null}}
  },
  {
    "id": 2, "resid": "urn:test:002", "archived": false, "enabled": true, "expired": false,
    "start": "2024-01-01T00:00:00Z", "end": null, "wfid": 10, "resource-id": 101,
    "organization": {"organization/id": "org1"},
    "localizations": {"en": {"id": 2, "langcode": "en", "title": "Duplicate Item", "infourl": null}}
  }
]`

	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(duplicatesJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_catalogue_item" "test" {
  title = "Duplicate Item"
}`,
				ExpectError: regexp.MustCompile(`Multiple CatalogueItem Found`),
			},
		},
	})
}

func TestCatalogueItemDataSource_NotFound(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(catalogueItemsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_catalogue_item" "test" {
  title = "Non existence item"
}`,
				ExpectError: regexp.MustCompile(`Catalogue Item Not Found`),
			},
		},
	})
}
