package resources_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockCatalogueItemHandler returns an HTTP handler covering all catalogue item CRUD endpoints.
// readJSON is the JSON returned by GET /api/catalogue-items/42.
func mockCatalogueItemHandler(readJSON string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/catalogue-items/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})

	mux.HandleFunc("PUT /api/catalogue-items/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/catalogue-items/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/catalogue-items/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("GET /api/catalogue-items/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, readJSON)
	})

	return mux
}

// minimalCatalogueItemReadJSON is the baseline GET response used by most tests.
// formid must be non-null to avoid a nil-pointer dereference in Read.
const minimalCatalogueItemReadJSON = `{
  "id": 42,
  "resid": "urn:example:resource1",
  "resource-id": 10,
  "wfid": 20,
  "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {
    "en": {"id": 1, "langcode": "en", "title": "Test Item", "infourl": null}
  },
  "enabled": true,
  "archived": false,
  "expired": false,
  "start": "2025-01-01T00:00:00Z",
  "end": null
}`

func TestCatalogueItemResource_CreateMinimal(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(minimalCatalogueItemReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "resource_id", "10"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "workflow_id", "20"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "form_id", "5"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "localizations.title", "Test Item"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "archived", "false"),
				),
			},
		},
	})
}

func TestCatalogueItemResource_CreateWithInfourl(t *testing.T) {
	readJSON := `{
  "id": 42,
  "resid": "urn:example:resource1",
  "resource-id": 10,
  "wfid": 20,
  "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {
    "en": {"id": 1, "langcode": "en", "title": "Test Item", "infourl": "https://example.com/info"}
  },
  "enabled": true,
  "archived": false,
  "expired": false,
  "start": "2025-01-01T00:00:00Z",
  "end": null
}`

	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title   = "Test Item"
    infourl = "https://example.com/info"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "localizations.title", "Test Item"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "localizations.infourl", "https://example.com/info"),
				),
			},
		},
	})
}

func TestCatalogueItemResource_CreateWithCategories(t *testing.T) {
	readJSON := `{
  "id": 42,
  "resid": "urn:example:resource1",
  "resource-id": 10,
  "wfid": 20,
  "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {
    "en": {"id": 1, "langcode": "en", "title": "Test Item", "infourl": null}
  },
  "enabled": true,
  "archived": false,
  "expired": false,
  "start": "2025-01-01T00:00:00Z",
  "end": null,
  "categories": [
    {"category/id": 1},
    {"category/id": 2}
  ]
}`

	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
  categories = [1, 2]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "categories.#", "2"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "categories.0", "1"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "categories.1", "2"),
				),
			},
		},
	})
}

func TestCatalogueItemResource_CreateDisabled(t *testing.T) {
	readJSON := `{
  "id": 42,
  "resid": "urn:example:resource1",
  "resource-id": 10,
  "wfid": 20,
  "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {
    "en": {"id": 1, "langcode": "en", "title": "Test Item", "infourl": null}
  },
  "enabled": false,
  "archived": false,
  "expired": false,
  "start": "2025-01-01T00:00:00Z",
  "end": null
}`

	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
  enabled = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "enabled", "false"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "archived", "false"),
				),
			},
		},
	})
}

func TestCatalogueItemResource_CreateArchived(t *testing.T) {
	readJSON := `{
  "id": 42,
  "resid": "urn:example:resource1",
  "resource-id": 10,
  "wfid": 20,
  "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {
    "en": {"id": 1, "langcode": "en", "title": "Test Item", "infourl": null}
  },
  "enabled": true,
  "archived": true,
  "expired": false,
  "start": "2025-01-01T00:00:00Z",
  "end": null
}`

	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
  archived = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "archived", "true"),
				),
			},
		},
	})
}

func TestCatalogueItemResource_UpdateLocalizations(t *testing.T) {
	step1ReadJSON := `{
  "id": 42, "resid": "urn:example:resource1", "resource-id": 10, "wfid": 20, "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {"en": {"id": 1, "langcode": "en", "title": "Original Title", "infourl": null}},
  "enabled": true, "archived": false, "expired": false, "start": "2025-01-01T00:00:00Z", "end": null
}`
	step2ReadJSON := `{
  "id": 42, "resid": "urn:example:resource1", "resource-id": 10, "wfid": 20, "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {"en": {"id": 1, "langcode": "en", "title": "Updated Title", "infourl": null}},
  "enabled": true, "archived": false, "expired": false, "start": "2025-01-01T00:00:00Z", "end": null
}`

	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/catalogue-items/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})
	mux.HandleFunc("PUT /api/catalogue-items/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/catalogue-items/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/catalogue-items/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/catalogue-items/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount <= 1 {
			fmt.Fprintln(w, step1ReadJSON)
		} else {
			fmt.Fprintln(w, step2ReadJSON)
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
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Original Title"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "localizations.title", "Original Title"),
				),
			},
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Updated Title"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "localizations.title", "Updated Title"),
				),
			},
		},
	})
}

func TestCatalogueItemResource_UpdateEnabledArchived(t *testing.T) {
	step1ReadJSON := `{
  "id": 42, "resid": "urn:example:resource1", "resource-id": 10, "wfid": 20, "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {"en": {"id": 1, "langcode": "en", "title": "Test Item", "infourl": null}},
  "enabled": true, "archived": false, "expired": false, "start": "2025-01-01T00:00:00Z", "end": null
}`
	step2ReadJSON := `{
  "id": 42, "resid": "urn:example:resource1", "resource-id": 10, "wfid": 20, "formid": 5,
  "organization": {"organization/id": "test-org"},
  "localizations": {"en": {"id": 1, "langcode": "en", "title": "Test Item", "infourl": null}},
  "enabled": false, "archived": true, "expired": false, "start": "2025-01-01T00:00:00Z", "end": null
}`

	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/catalogue-items/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})
	mux.HandleFunc("PUT /api/catalogue-items/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/catalogue-items/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/catalogue-items/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/catalogue-items/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount <= 1 {
			fmt.Fprintln(w, step1ReadJSON)
		} else {
			fmt.Fprintln(w, step2ReadJSON)
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
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
  enabled  = true
  archived = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "archived", "false"),
				),
			},
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
  enabled  = false
  archived = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "enabled", "false"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "archived", "true"),
				),
			},
		},
	})
}

func TestCatalogueItemResource_DeleteOnDestroy(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(minimalCatalogueItemReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
				),
			},
			{
				Destroy: true,
				Config:  `provider "remscontent" {}`,
			},
		},
	})
}

func TestCatalogueItemResource_ReadRemovesStateWhen404(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/catalogue-items/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})
	mux.HandleFunc("PUT /api/catalogue-items/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/catalogue-items/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		// First read (post-create refresh): item exists.
		// Second read (plan/refresh for step 2): item is gone.
		if callCount <= 1 {
			fmt.Fprintln(w, minimalCatalogueItemReadJSON)
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
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
				),
			},
			{
				// When the item no longer exists on the server, the provider removes it
				// from state and Terraform plans a recreate (non-empty plan).
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
}`,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestCatalogueItemResource_ImportState(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockCatalogueItemHandler(minimalCatalogueItemReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_catalogue_item" "test" {
  organization_id = "test-org"
  resource_id     = 10
  workflow_id     = 20
  form_id         = 5
  localizations = {
    title = "Test Item"
  }
}`,
			},
			{
				ResourceName:      "remscontent_catalogue_item.test",
				ImportState:       true,
				ImportStateId:     "42",
				ImportStateVerify: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "resource_id", "10"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "workflow_id", "20"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "form_id", "5"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "localizations.title", "Test Item"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_catalogue_item.test", "archived", "false"),
				),
			},
		},
	})
}
