package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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
