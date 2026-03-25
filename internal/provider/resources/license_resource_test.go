package resources_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockLicenseHandler returns an HTTP handler covering all license CRUD endpoints.
// readJSON is the JSON returned by GET /api/licenses/{licenseId}.
func mockLicenseHandler(readJSON string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/licenses/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 10}`)
	})

	mux.HandleFunc("PUT /api/licenses/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/licenses/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("GET /api/licenses/10", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, readJSON)
	})

	return mux
}

const minimalLinkLicenseReadJSON = `{
  "id": 10,
  "licensetype": "link",
  "enabled": true,
  "archived": false,
  "organization": {
    "organization/id": "test-org",
    "organization/name": {},
    "organization/short-name": {}
  },
  "localizations": {
    "en": {
      "title": "My Link License",
      "textcontent": "https://example.com/license"
    }
  }
}`

const minimalTextLicenseReadJSON = `{
  "id": 10,
  "licensetype": "text",
  "enabled": true,
  "archived": false,
  "organization": {
    "organization/id": "test-org",
    "organization/name": {},
    "organization/short-name": {}
  },
  "localizations": {
    "en": {
      "title": "My Text License",
      "textcontent": "This is the license text."
    }
  }
}`

// ============================================================
// Unit tests — ValidateConfig
// These never reach Apply and never make real HTTP calls.
// ============================================================

func TestUnitLicenseResource_AttachmentType_MissingPath(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type  = "attachment"
  title = "My Attachment"
}`,
				ExpectError: regexp.MustCompile("path must be set when type is attachment"),
			},
		},
	})
}

func TestUnitLicenseResource_TextType_MissingContent(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type  = "text"
  title = "My Text License"
}`,
				ExpectError: regexp.MustCompile("content must be set when type is text or link"),
			},
		},
	})
}

func TestUnitLicenseResource_LinkType_MissingContent(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type  = "link"
  title = "My Link License"
}`,
				ExpectError: regexp.MustCompile("content must be set when type is text or link"),
			},
		},
	})
}

// ============================================================
// Integration tests — using mock HTTP server
// ============================================================

func TestLicenseResource_CreateLinkLicense(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(minimalLinkLicenseReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type    = "link"
  title   = "My Link License"
  content = "https://example.com/license"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_license.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_license.test", "type", "link"),
					resource.TestCheckResourceAttr("remscontent_license.test", "title", "My Link License"),
					resource.TestCheckResourceAttr("remscontent_license.test", "content", "https://example.com/license"),
					resource.TestCheckResourceAttr("remscontent_license.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_license.test", "archived", "false"),
				),
			},
		},
	})
}

func TestLicenseResource_CreateTextLicense(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(minimalTextLicenseReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type    = "text"
  title   = "My Text License"
  content = "This is the license text."
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_license.test", "type", "text"),
					resource.TestCheckResourceAttr("remscontent_license.test", "title", "My Text License"),
					resource.TestCheckResourceAttr("remscontent_license.test", "content", "This is the license text."),
					resource.TestCheckResourceAttr("remscontent_license.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_license.test", "archived", "false"),
				),
			},
		},
	})
}

func TestLicenseResource_Disabled(t *testing.T) {
	readJSON := `{
  "id": 10, "licensetype": "link", "enabled": false, "archived": false,
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "localizations": {"en": {"title": "My Link License", "textcontent": "https://example.com/license"}}
}`

	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type    = "link"
  title   = "My Link License"
  content = "https://example.com/license"
  enabled = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_license.test", "enabled", "false"),
					resource.TestCheckResourceAttr("remscontent_license.test", "archived", "false"),
				),
			},
		},
	})
}

func TestLicenseResource_UpdateEnabledArchived(t *testing.T) {
	step1ReadJSON := `{
  "id": 10, "licensetype": "link", "enabled": true, "archived": false,
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "localizations": {"en": {"title": "My Link License", "textcontent": "https://example.com/license"}}
}`
	step2ReadJSON := `{
  "id": 10, "licensetype": "link", "enabled": false, "archived": true,
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "localizations": {"en": {"title": "My Link License", "textcontent": "https://example.com/license"}}
}`

	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/licenses/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 10}`)
	})
	mux.HandleFunc("PUT /api/licenses/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/licenses/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/licenses/10", func(w http.ResponseWriter, r *http.Request) {
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
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type     = "link"
  title    = "My Link License"
  content  = "https://example.com/license"
  enabled  = true
  archived = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_license.test", "archived", "false"),
				),
			},
			{
				Config: `
provider "remscontent" {}
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type     = "link"
  title    = "My Link License"
  content  = "https://example.com/license"
  enabled  = false
  archived = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "enabled", "false"),
					resource.TestCheckResourceAttr("remscontent_license.test", "archived", "true"),
				),
			},
		},
	})
}

func TestLicenseResource_DeleteArchives(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(minimalLinkLicenseReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type    = "link"
  title   = "My Link License"
  content = "https://example.com/license"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "id", "10"),
				),
			},
			{
				Destroy: true,
				Config:  `provider "remscontent" {}`,
			},
		},
	})
}

func TestLicenseResource_ImportState(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(minimalLinkLicenseReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type    = "link"
  title   = "My Link License"
  content = "https://example.com/license"
}`,
			},
			{
				ResourceName:      "remscontent_license.test",
				ImportState:       true,
				ImportStateId:     "10",
				ImportStateVerify: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_license.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_license.test", "type", "link"),
					resource.TestCheckResourceAttr("remscontent_license.test", "title", "My Link License"),
					resource.TestCheckResourceAttr("remscontent_license.test", "content", "https://example.com/license"),
					resource.TestCheckResourceAttr("remscontent_license.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_license.test", "archived", "false"),
				),
			},
		},
	})
}

func TestLicenseResource_CreateAttachmentLicense(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "license.txt")
	if err := os.WriteFile(filePath, []byte("license content"), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	attachmentReadJSON := fmt.Sprintf(`{
  "id": 10,
  "licensetype": "attachment",
  "enabled": true,
  "archived": false,
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "localizations": {"en": {"title": "My Attachment License", "textcontent": %q, "attachment-id": 5}}
}`, filepath.Base(filePath))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/licenses/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 10}`)
	})
	mux.HandleFunc("PUT /api/licenses/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/licenses/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("POST /api/licenses/add_attachment", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 5}`)
	})
	mux.HandleFunc("GET /api/licenses/10", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, attachmentReadJSON)
	})

	factories, cleanup := testProviderWithMockServer(t, mux)
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "remscontent" {}
resource "remscontent_license" "test" {
  organization_id = "test-org"
  type  = "attachment"
  title = "My Attachment License"
  path  = %q
}`, filePath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_license.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_license.test", "type", "attachment"),
					resource.TestCheckResourceAttr("remscontent_license.test", "title", "My Attachment License"),
					resource.TestCheckResourceAttr("remscontent_license.test", "path", filePath),
					resource.TestCheckResourceAttr("remscontent_license.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_license.test", "archived", "false"),
				),
			},
		},
	})
}
