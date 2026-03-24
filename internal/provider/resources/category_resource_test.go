package resources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockResourceHandler returns a handler covering all resource CRUD endpoints.
// readJSON is the JSON returned by GET /api/categories/{id}.
func mockCategoriesHandler(readJSON string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/categories/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "category/id": 10}`)
	})

	mux.HandleFunc("POST /api/categories/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("GET /api/categories/{categoryId}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, readJSON)
	})

	return mux
}

const minimalCategoryReadJSON = `{
    "category/id": 10,
    "category/title": {"en": "category-01"},
    "category/description": {"en": "description for category 1"}
}`

func TestCategoryResource_CreateMinimal(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCategoriesHandler(minimalCategoryReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_category" "test" {
  description = "description for category 1"
  title       = "category-01"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_category.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_category.test", "title", "category-01"),
					resource.TestCheckResourceAttr("remscontent_category.test", "description", "description for category 1"),
				),
			},
		},
	})
}

const category2ReadJSON = `{
    "category/id": 10,
    "category/title": {"en": "category-02"},
    "category/description": {"en": "a different description"}
}`

func TestCategoryResource_CreateWithDifferentTitle(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCategoriesHandler(category2ReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_category" "test" {
  description = "a different description"
  title       = "category-02"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_category.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_category.test", "title", "category-02"),
					resource.TestCheckResourceAttr("remscontent_category.test", "description", "a different description"),
				),
			},
		},
	})
}

func TestCategoryResource_UpdateTitle(t *testing.T) {
	updatedJSON := `{"category/id": 10, "category/title": {"en": "category-01-updated"}, "category/description": {"en": "description for category 1"}}`
	factories, cleanup := testProviderWithMockServer(t, mockCategoriesHandler(updatedJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_category" "test" {
  description = "description for category 1"
  title       = "category-01-updated"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_category.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_category.test", "title", "category-01-updated"),
				),
			},
		},
	})
}

func TestCategoryResource_DeleteOnDestroy(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCategoriesHandler(minimalCategoryReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_category" "test" {
  description = "description for category 1"
  title       = "category-01"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_category.test", "id", "10"),
				),
			},
			{
				Destroy: true,
				Config:  `provider "remscontent" {}`,
			},
		},
	})
}

func TestCategoryResource_ReadRemovesStateWhenDeleted(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/categories/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "category/id": 10}`)
	})
	mux.HandleFunc("POST /api/categories/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/categories/{categoryId}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		// First read (post-create refresh): category exists.
		// Second read (plan/refresh for step 2): category is gone.
		if callCount <= 1 {
			fmt.Fprintln(w, minimalCategoryReadJSON)
		} else {
			http.NotFound(w, r)
		}
	})

	factories, cleanup := testProviderWithMockServer(t, mux)
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_category" "test" {
  description = "description for category 1"
  title       = "category-01"
}`,
				Check: resource.TestCheckResourceAttr("remscontent_category.test", "id", "10"),
			},
			{
				// When the category no longer exists on the server, the provider removes
				// it from state and Terraform plans a recreate (non-empty plan).
				Config: `
provider "remscontent" {}
resource "remscontent_category" "test" {
  description = "description for category 1"
  title       = "category-01"
}`,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestCategoryResource_ImportState(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCategoriesHandler(minimalCategoryReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_category" "test" {
  description = "description for category 1"
  title       = "category-01"
}`,
			},
			{
				ResourceName:      "remscontent_category.test",
				ImportState:       true,
				ImportStateId:     "10",
				ImportStateVerify: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_category.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_category.test", "title", "category-01"),
					resource.TestCheckResourceAttr("remscontent_category.test", "description", "description for category 1"),
				),
			},
		},
	})
}

func TestCategoryResource_APIError_OnCreate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/categories/create", func(w http.ResponseWriter, r *http.Request) {
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
resource "remscontent_category" "test" {
  description = "description for category 1"
  title       = "category-01"
}`,
				ExpectError: regexp.MustCompile(`Error Adding Category Entry`),
			},
		},
	})
}
