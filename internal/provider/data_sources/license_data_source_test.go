package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockLicenseHandler(licensesJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/licenses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, licensesJSON)
	})
	return mux
}

const licensesListJSON = `[
  {
    "id": 1, "archived": false, "enabled": true, "licensetype": "link",
    "localizations": {"en": {"title": "First License", "textcontent": "http://example.com/1"}},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  },
  {
    "id": 2, "archived": false, "enabled": true, "licensetype": "link",
    "localizations": {"en": {"title": "Second License", "textcontent": "http://example.com/2"}},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  }
]`

func TestLicenseDataSource_FindByTitle(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(licensesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_license" "test" {
  title = "First License"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_license.test", "title", "First License"),
					resource.TestCheckResourceAttr("data.remscontent_license.test", "id", "1"),
				),
			},
		},
	})
}

func TestLicenseDataSource_FindSecondLicense(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(licensesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_license" "test" {
  title = "Second License"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_license.test", "title", "Second License"),
					resource.TestCheckResourceAttr("data.remscontent_license.test", "id", "2"),
				),
			},
		},
	})
}

func TestLicenseDataSource_NotFound(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(licensesListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_license" "test" {
  title = "Nonexistent License"
}`,
				ExpectError: regexp.MustCompile(`License Not Found`),
			},
		},
	})
}

func TestLicenseDataSource_MultipleLicensesError(t *testing.T) {
	duplicatesJSON := `[
  {
    "id": 1, "archived": false, "enabled": true, "licensetype": "link",
    "localizations": {"en": {"title": "Duplicate License", "textcontent": "http://example.com/1"}},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  },
  {
    "id": 2, "archived": false, "enabled": true, "licensetype": "link",
    "localizations": {"en": {"title": "Duplicate License", "textcontent": "http://example.com/2"}},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  }
]`

	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(duplicatesJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_license" "test" {
  title = "Duplicate License"
}`,
				ExpectError: regexp.MustCompile(`Multiple Licenses Found`),
			},
		},
	})
}

func TestLicenseDataSource_NoEnglishLocalization(t *testing.T) {
	// A license with only a Finnish localization should not match an English title lookup.
	fiOnlyJSON := `[
  {
    "id": 1, "archived": false, "enabled": true, "licensetype": "link",
    "localizations": {"fi": {"title": "Suomalainen Lisenssi", "textcontent": "http://example.fi"}},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  },
  {
    "id": 2, "archived": false, "enabled": true, "licensetype": "link",
    "localizations": {"en": {"title": "English License", "textcontent": "http://example.com"}},
    "organization": {"organization/id": "org1", "organization/name": {"en": "Org 1"}, "organization/short-name": {"en": "O1"}}
  }
]`

	factories, cleanup := testProviderWithMockServer(t, mockLicenseHandler(fiOnlyJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_license" "test" {
  title = "English License"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_license.test", "id", "2"),
				),
			},
		},
	})
}
