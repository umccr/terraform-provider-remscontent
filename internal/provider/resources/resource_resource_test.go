// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mockResourceHandler returns a handler covering all resource CRUD endpoints.
// readJSON is the JSON returned by GET /api/resources/{id}.
func mockResourceHandler(readJSON string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/resources/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 10}`)
	})

	mux.HandleFunc("PUT /api/resources/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("PUT /api/resources/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("GET /api/resources/10", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, readJSON)
	})

	return mux
}

const minimalResourceReadJSON = `{
  "id": 10,
  "resid": "urn:example:dataset1",
  "organization": {
    "organization/id": "test-org",
    "organization/name": {},
    "organization/short-name": {}
  },
  "licenses": [],
  "enabled": true,
  "archived": false
}`

func TestResourceResource_CreateMinimal(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(minimalResourceReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_resource" "test" {
  resid           = "urn:example:dataset1"
  organization_id = "test-org"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_resource.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "resid", "urn:example:dataset1"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "organization_id", "test-org"),
					// computed defaults
					resource.TestCheckResourceAttr("remscontent_resource.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "archived", "false"),
				),
			},
		},
	})
}

func TestResourceResource_CreateWithLicenses(t *testing.T) {
	readJSON := `{
  "id": 10,
  "resid": "urn:example:dataset1",
  "organization": {
    "organization/id": "test-org",
    "organization/name": {},
    "organization/short-name": {}
  },
  "licenses": [
    {"id": 7, "licensetype": "link", "enabled": true, "archived": false,
     "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
     "localizations": {}},
    {"id": 8, "licensetype": "text", "enabled": true, "archived": false,
     "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
     "localizations": {}}
  ],
  "enabled": true,
  "archived": false
}`

	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_resource" "test" {
  resid           = "urn:example:dataset1"
  organization_id = "test-org"
  licenses        = [7, 8]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_resource.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "licenses.#", "2"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "licenses.0", "7"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "licenses.1", "8"),
				),
			},
		},
	})
}

func TestResourceResource_Disabled(t *testing.T) {
	readJSON := `{
  "id": 10,
  "resid": "urn:example:dataset1",
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "licenses": [],
  "enabled": false,
  "archived": false
}`

	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_resource" "test" {
  resid           = "urn:example:dataset1"
  organization_id = "test-org"
  enabled         = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_resource.test", "enabled", "false"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "archived", "false"),
				),
			},
		},
	})
}

func TestResourceResource_UpdateEnabledArchived(t *testing.T) {
	step1ReadJSON := `{
  "id": 10, "resid": "urn:example:dataset1",
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "licenses": [], "enabled": true, "archived": false
}`
	step2ReadJSON := `{
  "id": 10, "resid": "urn:example:dataset1",
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "licenses": [], "enabled": false, "archived": true
}`

	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/resources/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true, "id": 10}`)
	})
	mux.HandleFunc("PUT /api/resources/enabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("PUT /api/resources/archived", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/resources/10", func(w http.ResponseWriter, r *http.Request) {
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
resource "remscontent_resource" "test" {
  resid           = "urn:example:dataset1"
  organization_id = "test-org"
  enabled         = true
  archived        = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_resource.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "archived", "false"),
				),
			},
			{
				Config: `
provider "remscontent" {}
resource "remscontent_resource" "test" {
  resid           = "urn:example:dataset1"
  organization_id = "test-org"
  enabled         = false
  archived        = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_resource.test", "enabled", "false"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "archived", "true"),
				),
			},
		},
	})
}

func TestResourceResource_ImportState(t *testing.T) {
	readJSON := `{
  "id": 10,
  "resid": "urn:example:dataset1",
  "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
  "licenses": [
    {"id": 7, "licensetype": "link", "enabled": true, "archived": false,
     "organization": {"organization/id": "test-org", "organization/name": {}, "organization/short-name": {}},
     "localizations": {}}
  ],
  "enabled": true,
  "archived": false
}`

	factories, cleanup := testProviderWithMockServer(t, mockResourceHandler(readJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_resource" "test" {
  resid           = "urn:example:dataset1"
  organization_id = "test-org"
  licenses        = [7]
}`,
			},
			{
				ResourceName:      "remscontent_resource.test",
				ImportState:       true,
				ImportStateId:     "10",
				ImportStateVerify: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_resource.test", "id", "10"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "resid", "urn:example:dataset1"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "organization_id", "test-org"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "enabled", "true"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "archived", "false"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "licenses.#", "1"),
					resource.TestCheckResourceAttr("remscontent_resource.test", "licenses.0", "7"),
				),
			},
		},
	})
}
