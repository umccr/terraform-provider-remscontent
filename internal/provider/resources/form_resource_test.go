package resources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================
// Integration tests — full CRUD with mock HTTP server
// ============================================================

// mockFormHandler returns a handler covering all form CRUD endpoints.
// readJSON is the JSON returned by GET /api/forms/{id}.
func mockFormHandler(readJSON string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/forms/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})

	mux.HandleFunc("PUT /api/forms/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/forms/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/forms/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("GET /api/forms/42/editable", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("GET /api/forms/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, readJSON)
	})

	return mux
}

const minimalFormReadJSON = `{
  "form/id": 42,
  "form/internal-name": "My Form",
  "form/external-title": {"en": "My Form"},
  "form/fields": [
    {
      "field/id": "1",
      "field/type": "text",
      "field/title": {"en": "Full Name"},
      "field/optional": false
    }
  ],
  "enabled": true,
  "archived": false,
  "organization": {
    "organization/id": "test-org",
    "organization/name": {},
    "organization/short-name": {}
  }
}`

func TestFormResource_CreateMinimal(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockFormHandler(minimalFormReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Full Name", type = "text" }
  ]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_form.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_form.test", "internal_name", "My Form"),
					resource.TestCheckResourceAttr("remscontent_form.test", "external_title", "My Form"),
					resource.TestCheckResourceAttr("remscontent_form.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_form.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_form.test", "archived", "false"),
					resource.TestCheckResourceAttr("remscontent_form.test", "fields.#", "1"),
					resource.TestCheckResourceAttr("remscontent_form.test", "fields.0.id", "1"),
					resource.TestCheckResourceAttr("remscontent_form.test", "fields.0.type", "text"),
					resource.TestCheckResourceAttr("remscontent_form.test", "fields.0.title", "Full Name"),
				),
			},
		},
	})
}

func TestFormResource_UpdateTitle(t *testing.T) {
	step1ReadJSON := `{
  "form/id": 42, "form/internal-name": "Original Form", "form/external-title": {"en": "Original Form"},
  "form/fields": [{"field/id": "1", "field/type": "text", "field/title": {"en": "Name"}, "field/optional": false}],
  "enabled": true, "archived": false,
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}}
}`
	step2ReadJSON := `{
  "form/id": 42, "form/internal-name": "Updated Form", "form/external-title": {"en": "Updated Form"},
  "form/fields": [{"field/id": "1", "field/type": "text", "field/title": {"en": "Name"}, "field/optional": false}],
  "enabled": true, "archived": false,
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}}
}`

	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/forms/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})
	mux.HandleFunc("PUT /api/forms/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/forms/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/forms/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/forms/42/editable", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/forms/42", func(w http.ResponseWriter, r *http.Request) {
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
resource "remscontent_form" "test" {
  internal_name   = "Original Form"
  external_title  = "Original Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Name", type = "text" }
  ]
}`,
				Check: resource.TestCheckResourceAttr("remscontent_form.test", "internal_name", "Original Form"),
			},
			{
				Config: `
provider "remscontent" {}
resource "remscontent_form" "test" {
  internal_name   = "Updated Form"
  external_title  = "Updated Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Name", type = "text" }
  ]
}`,
				Check: resource.TestCheckResourceAttr("remscontent_form.test", "internal_name", "Updated Form"),
			},
		},
	})
}

func TestFormResource_DeleteOnDestroy(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockFormHandler(minimalFormReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Full Name", type = "text" }
  ]
}`,
				Check: resource.TestCheckResourceAttr("remscontent_form.test", "id", "42"),
			},
			{
				Destroy: true,
				Config:  `provider "remscontent" {}`,
			},
		},
	})
}

func TestFormResource_ReadRemovesStateWhen404(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/forms/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 42}`)
	})
	mux.HandleFunc("PUT /api/forms/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/forms/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		// First read (post-create refresh): form exists.
		// Second read (plan/refresh for step 2): form is gone.
		if callCount <= 1 {
			fmt.Fprintln(w, minimalFormReadJSON)
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
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Full Name", type = "text" }
  ]
}`,
				Check: resource.TestCheckResourceAttr("remscontent_form.test", "id", "42"),
			},
			{
				// When the form no longer exists on the server, the provider removes it
				// from state and Terraform plans a recreate (non-empty plan).
				Config: `
provider "remscontent" {}
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Full Name", type = "text" }
  ]
}`,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestFormResource_ImportState(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockFormHandler(minimalFormReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Full Name", type = "text" }
  ]
}`,
			},
			{
				ResourceName:      "remscontent_form.test",
				ImportState:       true,
				ImportStateId:     "42",
				ImportStateVerify: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_form.test", "id", "42"),
					resource.TestCheckResourceAttr("remscontent_form.test", "internal_name", "My Form"),
					resource.TestCheckResourceAttr("remscontent_form.test", "external_title", "My Form"),
					resource.TestCheckResourceAttr("remscontent_form.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_form.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_form.test", "archived", "false"),
				),
			},
		},
	})
}

func TestFormResource_APIError_OnCreate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/forms/create", func(w http.ResponseWriter, r *http.Request) {
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
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Full Name", type = "text" }
  ]
}`,
				ExpectError: regexp.MustCompile(`Error Creating Form`),
			},
		},
	})
}

// ============================================================
// Unit tests — ValidateConfig
// These use resource.UnitTest (no TF_ACC required) and PlanOnly
// so they never reach Apply and never make real HTTP calls.
// ============================================================

func TestUnitFormResource_OptionField_MissingOptions(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Choose one", type = "option" }
  ]
}`,
				ExpectError: regexp.MustCompile("`options` is required when field type is `option`"),
			},
		},
	})
}

func TestUnitFormResource_MultiSelectField_MissingOptions(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Pick many", type = "multiselect" }
  ]
}`,
				ExpectError: regexp.MustCompile("`options` is required when field type is `multiselect`"),
			},
		},
	})
}

func TestUnitFormResource_TextField_UnexpectedOptions(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    {
      id    = "1"
      title = "Name"
      type  = "text"
      options = [{ key = "a", label = "A" }]
    }
  ]
}`,
				ExpectError: regexp.MustCompile("`options` should not be set when field type is `text`"),
			},
		},
	})
}

func TestUnitFormResource_TableField_MissingColumns(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Grid", type = "table" }
  ]
}`,
				ExpectError: regexp.MustCompile("`columns` is required when field type is `table`"),
			},
		},
	})
}

func TestUnitFormResource_TextField_UnexpectedColumns(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    {
      id      = "1"
      title   = "Name"
      type    = "text"
      columns = [{ key = "c1", label = "Col 1" }]
    }
  ]
}`,
				ExpectError: regexp.MustCompile("`columns` should only be set when field type is `table`"),
			},
		},
	})
}

func TestUnitFormResource_Visibility_OnlyIf_MissingFieldId(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    {
      id    = "1"
      title = "Name"
      type  = "text"
      visibility = {
        visibility_type = "only-if"
        has_value       = ["y"]
      }
    }
  ]
}`,
				ExpectError: regexp.MustCompile("`field_id` is required when `visibility_type` is `only-if`"),
			},
		},
	})
}

func TestUnitFormResource_Visibility_OnlyIf_MissingHasValue(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    {
      id    = "q1"
      title = "Attach?"
      type  = "option"
      options = [
        { key = "y", label = "Yes" },
        { key = "n", label = "No" },
      ]
    },
    {
      id    = "q2"
      title = "File"
      type  = "attachment"
      visibility = {
        visibility_type = "only-if"
        field_id        = "q1"
      }
    }
  ]
}`,
				ExpectError: regexp.MustCompile("`has_value` is required when `visibility_type` is `only-if`"),
			},
		},
	})
}

func TestUnitFormResource_Visibility_OnlyIf_InvalidFieldId(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    {
      id    = "1"
      title = "File"
      type  = "attachment"
      visibility = {
        visibility_type = "only-if"
        field_id        = "nonexistent-field"
        has_value       = ["y"]
      }
    }
  ]
}`,
				ExpectError: regexp.MustCompile("`field_id` `nonexistent-field` does not reference any field in this form"),
			},
		},
	})
}

func TestUnitFormResource_Visibility_OnlyIf_InvalidHasValue(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    {
      id    = "q1"
      title = "Attach?"
      type  = "option"
      options = [
        { key = "y", label = "Yes" },
        { key = "n", label = "No" },
      ]
    },
    {
      id    = "1"
      title = "File"
      type  = "attachment"
      visibility = {
        visibility_type = "only-if"
        field_id        = "q1"
        has_value       = ["maybe"]
      }
    }
  ]
}`,
				ExpectError: regexp.MustCompile("`maybe` is not a valid option key in field `q1`"),
			},
		},
	})
}

func TestUnitFormResource_Visibility_Always_UnexpectedFieldId(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Test"
  external_title  = "Test"
  organization_id = "test-org"
  fields = [
    {
      id    = "1"
      title = "Name"
      type  = "text"
      visibility = {
        visibility_type = "always"
        field_id        = "some-field"
      }
    }
  ]
}`,
				ExpectError: regexp.MustCompile("`field_id` should not be set when `visibility_type` is `always`"),
			},
		},
	})
}

// Valid config — PlanOnly so it never reaches Apply (no real HTTP call needed).
func TestUnitFormResource_ValidConfig_DuplicateFieldId(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "A", type = "text" },
    { id = "1", title = "B", type = "email" },
  ]
}`,
				ExpectError: regexp.MustCompile("Field ID `1` must be unique within the form."),
			},
		},
	})
}

func TestUnitFormResource_ValidConfig_WithVisibility(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "Form With Visibility"
  external_title  = "Form With Visibility"
  organization_id = "test-org"
  fields = [
    {
      id    = "attach-q"
      title = "Attach a file?"
      type  = "option"
      options = [
        { key = "y", label = "Yes" },
        { key = "n", label = "No" },
      ]
    },
    {
      id    = "attachment-file"
      title = "Attachment"
      type  = "attachment"
      visibility = {
        visibility_type = "only-if"
        field_id        = "attach-q"
        has_value       = ["y"]
      }
    },
  ]
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitFormResource_ValidConfig_Basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_form" "test" {
  internal_name   = "My Form"
  external_title  = "My Form"
  organization_id = "test-org"
  fields = [
    { id = "1", title = "Full Name", type = "text" },
    { id = "2", title = "Email",     type = "email" },
    { id = "3", title = "Notes",     type = "texta", optional = true },
  ]
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
