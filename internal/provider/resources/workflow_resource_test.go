package resources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================
// Unit tests — schema validation (PlanOnly, no HTTP calls)
// ============================================================

func TestUnitWorkflowResource_InvalidType(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_workflow" "test" {
  title           = "Test"
  organization_id = "test-org"
  type            = "workflow/invalid"
}`,
				ExpectError: regexp.MustCompile(`value must be one of`),
				PlanOnly:    true,
			},
		},
	})
}

// ============================================================
// Integration tests — full CRUD with mock HTTP server
// ============================================================

// mockWorkflowHandler returns a handler that covers all workflow CRUD endpoints.
// readJSON is the JSON returned by GET /api/workflows/{id} (Read).
func mockWorkflowHandler(readJSON string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/workflows/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})

	mux.HandleFunc("PUT /api/workflows/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/workflows/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/workflows/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	// Read — GET /api/workflows/42
	mux.HandleFunc("GET /api/workflows/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, readJSON)
	})

	return mux
}

const minimalWorkflowReadJSON = `{
  "id": 42,
  "title": "My Workflow",
  "organization": {
    "organization/id": "test-org",
    "organization/name": {},
    "organization/short-name": {}
  },
  "enabled": true,
  "archived": false,
  "workflow": {
    "type": "workflow/default",
    "anonymize-handling": false,
    "handlers": [],
    "forms": [],
    "licenses": []
  }
}`

func TestWorkflowResource_CreateMinimal(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(minimalWorkflowReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}

resource "remscontent_workflow" "test" {
  title           = "My Workflow"
  organization_id = "test-org"
  type            = "workflow/default"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_workflow.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "title", "My Workflow"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "type", "workflow/default"),
					// computed defaults
					resource.TestCheckResourceAttr("remscontent_workflow.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "archived", "false"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "anonymize_handling", "false"),
				),
			},
		},
	})
}

func TestWorkflowResource_CreateWithHandlers(t *testing.T) {
	readJSON := `{
  "id": 42,
  "title": "Handler Workflow",
  "organization": {
    "organization/id": "test-org",
    "organization/name": {},
    "organization/short-name": {}
  },
  "enabled": true,
  "archived": false,
  "workflow": {
    "type": "workflow/default",
    "anonymize-handling": true,
    "handlers": [
      {"userid": "handler1@example.com"},
      {"userid": "handler2@example.com"}
    ],
    "forms": [],
    "licenses": []
  }
}`

	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}

resource "remscontent_workflow" "test" {
  title              = "Handler Workflow"
  organization_id    = "test-org"
  type               = "workflow/default"
  anonymize_handling = true
  handlers           = ["handler1@example.com", "handler2@example.com"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_workflow.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "anonymize_handling", "true"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "handlers.#", "2"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "handlers.0", "handler1@example.com"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "handlers.1", "handler2@example.com"),
				),
			},
		},
	})
}

func TestWorkflowResource_UpdateTitle(t *testing.T) {
	// Step 1: initial state returned by Read
	step1ReadJSON := `{
  "id": 42, "title": "Original Title",
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "enabled": true, "archived": false,
  "workflow": {"type": "workflow/default", "anonymize-handling": false, "handlers": [], "forms": [], "licenses": []}
}`
	// Step 2: updated title returned by Read after update
	step2ReadJSON := `{
  "id": 42, "title": "Updated Title",
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "enabled": true, "archived": false,
  "workflow": {"type": "workflow/default", "anonymize-handling": false, "handlers": [], "forms": [], "licenses": []}
}`

	// Use a counter to serve different responses per step
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/workflows/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})
	mux.HandleFunc("PUT /api/workflows/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/workflows/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/workflows/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/workflows/42", func(w http.ResponseWriter, r *http.Request) {
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
resource "remscontent_workflow" "test" {
  title           = "Original Title"
  organization_id = "test-org"
  type            = "workflow/default"
}`,
				Check: resource.TestCheckResourceAttr("remscontent_workflow.test", "title", "Original Title"),
			},
			{
				Config: `
provider "remscontent" {}
resource "remscontent_workflow" "test" {
  title           = "Updated Title"
  organization_id = "test-org"
  type            = "workflow/default"
}`,
				Check: resource.TestCheckResourceAttr("remscontent_workflow.test", "title", "Updated Title"),
			},
		},
	})
}

func TestWorkflowResource_Disabled(t *testing.T) {
	readJSON := `{
  "id": 42, "title": "Disabled Workflow",
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "enabled": false, "archived": false,
  "workflow": {"type": "workflow/default", "anonymize-handling": false, "handlers": [], "forms": [], "licenses": []}
}`

	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_workflow" "test" {
  title           = "Disabled Workflow"
  organization_id = "test-org"
  type            = "workflow/default"
  enabled         = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_workflow.test", "enabled", "false"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "archived", "false"),
				),
			},
		},
	})
}

func TestWorkflowResource_ImportState(t *testing.T) {
	readJson := `
{
  "id": 42,
  "organization": {
    "organization/id": "nbn",
    "organization/short-name": {
      "en": "NBN"
    },
    "organization/name": {
      "en": "NBN"
    }
  },
  "title": "With workflow form",
  "workflow": {
    "type": "workflow/default",
    "handlers": [
      {
        "userid": "developer",
        "name": "Developer",
        "email": "developer@example.com"
      },
      {
        "userid": "handler",
        "name": "Hannah Handler",
        "email": "handler@example.com"
      },
      {
        "userid": "rejecter-bot",
        "name": "Rejecter Bot",
        "email": null
      }
    ],
    "licenses": [
      {
        "licensetype": "link",
        "organization": {
          "organization/id": "nbn",
          "organization/name": {
            "en": "NBN"
          },
          "organization/short-name": {
            "en": "NBN"
          }
        },
        "enabled": true,
        "archived": false,
        "localizations": {
          "en": {
            "textcontent": "https://creativecommons.org/licenses/by/4.0/legalcode",
            "title": "CC Attribution 4.0"
          }
        },
        "license/id": 7
      },
      {
        "licensetype": "text",
        "organization": {
          "organization/id": "nbn",
          "organization/name": {
            "en": "NBN"
          },
          "organization/short-name": {
            "en": "NBN"
          }
        },
        "enabled": true,
        "archived": false,
        "localizations": {
          "en": {
            "textcontent": "License text in English. License text in English. License text in English. License text in English. License text in English. License text in English. License text in English. License text in English. License text in English. License text in English. ",
            "title": "General Terms of Use"
          }
        },
        "license/id": 8
      }
    ],
    "forms": [
      {
        "form/id": 1,
        "form/internal-name": "Workflow form",
        "form/external-title": {
          "en": "Workflow form"
        }
      }
    ],
    "anonymize-handling": true
  },
  "enabled": true,
  "archived": false
}`

	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(readJson))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create the resource first
			{
				Config: `
provider "remscontent" {}
resource "remscontent_workflow" "test" {
  title              = "With workflow form"
  organization_id    = "nbn"
  type               = "workflow/default"
  anonymize_handling = true
  handlers           = ["developer", "handler", "rejecter-bot"]
  forms              = [1]
  licenses           = [7, 8]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_workflow.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "title", "With workflow form"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "organization_id", "nbn"),
				),
			},
			// Import by ID, verify full state is restored from API
			{
				ResourceName:      "remscontent_workflow.test",
				ImportState:       true,
				ImportStateId:     "42",
				ImportStateVerify: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_workflow.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "title", "With workflow form"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "organization_id", "nbn"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "type", "workflow/default"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "archived", "false"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "anonymize_handling", "true"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "handlers.#", "3"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "handlers.0", "developer"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "handlers.1", "handler"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "handlers.2", "rejecter-bot"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "forms.#", "1"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "forms.0", "1"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "licenses.#", "2"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "licenses.0", "7"),
					resource.TestCheckResourceAttr("remscontent_workflow.test", "licenses.1", "8"),
				),
			},
		},
	})
}
