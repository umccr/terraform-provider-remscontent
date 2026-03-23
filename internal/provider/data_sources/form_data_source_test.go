package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockFormHandler(formsJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/forms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, formsJSON)
	})
	return mux
}

const formsListJSON = `[
  {
    "form/id": 1, "form/internal-name": "first-form", "archived": false, "enabled": true,
    "form/external-title": {"en": "First Form"},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  },
  {
    "form/id": 2, "form/internal-name": "second-form", "archived": false, "enabled": true,
    "form/external-title": {"en": "Second Form"},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  }
]`

func TestFormDataSource_FindByInternalName(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockFormHandler(formsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_form" "test" {
  internal_name = "second-form"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_form.test", "internal_name", "second-form"),
					resource.TestCheckResourceAttr("data.remscontent_form.test", "id", "2"),
				),
			},
		},
	})
}

func TestFormDataSource_MultipleFormsError(t *testing.T) {
	duplicatesJSON := `[
  {
    "form/id": 1, "form/internal-name": "duplicate-form", "archived": false, "enabled": true,
    "form/external-title": {"en": "Form"},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  },
  {
    "form/id": 2, "form/internal-name": "duplicate-form", "archived": false, "enabled": true,
    "form/external-title": {"en": "Form"},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  }
]`

	factories, cleanup := testProviderWithMockServer(t, mockFormHandler(duplicatesJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_form" "test" {
  internal_name = "duplicate-form"
}`,
				ExpectError: regexp.MustCompile(`Multiple Form Found`),
			},
		},
	})
}

func TestFormDataSource_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/forms", func(w http.ResponseWriter, r *http.Request) {
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
data "remscontent_form" "test" {
  internal_name = "any-form"
}`,
				ExpectError: regexp.MustCompile(`Error fetching forms list`),
			},
		},
	})
}

func TestFormDataSource_NotFound(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockFormHandler(formsListJSON))
	defer cleanup()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_form" "test" {
  internal_name = "invalid-name"
}`,
				ExpectError: regexp.MustCompile(`Form Not Found`),
			},
		},
	})
}
