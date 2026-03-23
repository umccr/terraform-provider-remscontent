package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockWorkflowHandler(workflowsJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workflows", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, workflowsJSON)
	})
	return mux
}

const workflowsListJSON = `[
  {
    "id": 1, "title": "First Workflow", "archived": false, "enabled": true,
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}},
    "workflow": {}
  },
  {
    "id": 2, "title": "Second Workflow", "archived": false, "enabled": true,
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}},
    "workflow": {}
  }
]`

func TestWorkflowDataSource_FindByTitle(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(workflowsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_workflow" "test" {
  title = "First Workflow"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_workflow.test", "title", "First Workflow"),
					resource.TestCheckResourceAttr("data.remscontent_workflow.test", "id", "1"),
				),
			},
		},
	})
}

func TestWorkflowDataSource_FindSecondWorkflow(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(workflowsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_workflow" "test" {
  title = "Second Workflow"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_workflow.test", "title", "Second Workflow"),
					resource.TestCheckResourceAttr("data.remscontent_workflow.test", "id", "2"),
				),
			},
		},
	})
}

func TestWorkflowDataSource_MultipleWorkflowsError(t *testing.T) {
	duplicatesJSON := `[
  {
    "id": 1, "title": "Duplicate Workflow", "archived": false, "enabled": true,
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}},
    "workflow": {}
  },
  {
    "id": 2, "title": "Duplicate Workflow", "archived": false, "enabled": true,
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}},
    "workflow": {}
  }
]`

	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(duplicatesJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_workflow" "test" {
  title = "Duplicate Workflow"
}`,
				ExpectError: regexp.MustCompile(`Multiple Workflow Found`),
			},
		},
	})
}

func TestWorkflowDataSource_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workflows", func(w http.ResponseWriter, r *http.Request) {
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
data "remscontent_workflow" "test" {
  title = "Any Workflow"
}`,
				ExpectError: regexp.MustCompile(`Error fetching workflows list`),
			},
		},
	})
}

func TestWorkflowDataSource_NotFound(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockWorkflowHandler(workflowsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_workflow" "test" {
  title = "Invalid title"
}`,
				ExpectError: regexp.MustCompile(`Workflow Not Found`),
			},
		},
	})
}
