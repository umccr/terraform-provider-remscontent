package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockCategoryHandler(categoriesJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/categories", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, categoriesJSON)
	})
	return mux
}

const categoriesListJSON = `[
  {"category/id": 1, "category/title": {"en": "First Category"}},
  {"category/id": 2, "category/title": {"en": "Second Category"}}
]`

func TestCategoryDataSource_FindByTitle(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCategoryHandler(categoriesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_category" "test" {
  title = "Second Category"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_category.test", "title", "Second Category"),
					resource.TestCheckResourceAttr("data.remscontent_category.test", "id", "2"),
				),
			},
		},
	})
}

func TestCategoryDataSource_MultipleTitlesError(t *testing.T) {
	duplicatesJSON := `[
  {"category/id": 1, "category/title": {"en": "Duplicate Category"}},
  {"category/id": 2, "category/title": {"en": "Duplicate Category"}}
]`

	factories, cleanup := testProviderWithMockServer(t, mockCategoryHandler(duplicatesJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_category" "test" {
  title = "Duplicate Category"
}`,
				ExpectError: regexp.MustCompile(`Multiple Category Found`),
			},
		},
	})
}

func TestCategoryDataSource_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/categories", func(w http.ResponseWriter, r *http.Request) {
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
data "remscontent_category" "test" {
  title = "Any Category"
}`,
				ExpectError: regexp.MustCompile(`Error fetching category list`),
			},
		},
	})
}

func TestCategoryDataSource_NotFound(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCategoryHandler(categoriesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_category" "test" {
  title = "Not Valid Title"
}`,
				ExpectError: regexp.MustCompile(`Category Not Found`),
			},
		},
	})
}
