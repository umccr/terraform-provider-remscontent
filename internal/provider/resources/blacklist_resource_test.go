package resources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockBlacklistHandler(readJSON string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/blacklist/add", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("POST /api/blacklist/remove", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})

	mux.HandleFunc("GET /api/blacklist", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, readJSON)
	})

	return mux
}

const blacklistEntryReadJSON = `[
  {
    "blacklist/resource": {"resource/ext-id": "urn:example:dataset1"},
    "blacklist/user":     {"userid": "alice"},
    "blacklist/comment":  "test comment",
    "blacklist/added-at": "2024-01-01T00:00:00Z",
    "blacklist/added-by": {"userid": "admin"}
  }
]`

func TestBlacklistResource_Create(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockBlacklistHandler(blacklistEntryReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_blacklist" "test" {
  resource_ext_id = "urn:example:dataset1"
  user_id         = "alice"
  comment         = "test comment"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_blacklist.test", "resource_ext_id", "urn:example:dataset1"),
					resource.TestCheckResourceAttr("remscontent_blacklist.test", "user_id", "alice"),
					resource.TestCheckResourceAttr("remscontent_blacklist.test", "comment", "test comment"),
				),
			},
		},
	})
}

func TestBlacklistResource_DeleteOnDestroy(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockBlacklistHandler(blacklistEntryReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_blacklist" "test" {
  resource_ext_id = "urn:example:dataset1"
  user_id         = "alice"
  comment         = "test comment"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_blacklist.test", "resource_ext_id", "urn:example:dataset1"),
				),
			},
			{
				Destroy: true,
				Config:  `provider "remscontent" {}`,
			},
		},
	})
}

func TestBlacklistResource_ReadRemovesStateWhenEmpty(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/blacklist/add", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("POST /api/blacklist/remove", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"success": true}`)
	})
	mux.HandleFunc("GET /api/blacklist", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		// First read (post-create refresh): entry exists
		// Second read (plan/refresh for step 2): entry is gone
		if callCount <= 1 {
			fmt.Fprintln(w, blacklistEntryReadJSON)
		} else {
			fmt.Fprintln(w, `[]`)
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
resource "remscontent_blacklist" "test" {
  resource_ext_id = "urn:example:dataset1"
  user_id         = "alice"
  comment         = "test comment"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_blacklist.test", "resource_ext_id", "urn:example:dataset1"),
				),
			},
			{
				// When the entry no longer exists on the server, the provider should
				// recreate it (plan detects drift and shows a non-empty plan).
				Config: `
provider "remscontent" {}
resource "remscontent_blacklist" "test" {
  resource_ext_id = "urn:example:dataset1"
  user_id         = "alice"
  comment         = "test comment"
}`,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestBlacklistResource_ImportState(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockBlacklistHandler(blacklistEntryReadJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
resource "remscontent_blacklist" "test" {
  resource_ext_id = "urn:example:dataset1"
  user_id         = "alice"
  comment         = "test comment"
}`,
			},
			{
				ResourceName:      "remscontent_blacklist.test",
				ImportState:       true,
				ImportStateId:     "urn:example:dataset1|alice",
				ImportStateVerify: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("remscontent_blacklist.test", "resource_ext_id", "urn:example:dataset1"),
					resource.TestCheckResourceAttr("remscontent_blacklist.test", "user_id", "alice"),
				),
			},
		},
	})
}

func TestBlacklistResource_ImportState_InvalidFormat(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "remscontent_blacklist" "test" {
  resource_ext_id = "urn:example:dataset1"
  user_id         = "alice"
}`,
				ResourceName:  "remscontent_blacklist.test",
				ImportState:   true,
				ImportStateId: "invalid-no-pipe",
				ExpectError:   regexp.MustCompile(`Expected import ID in the format`),
			},
		},
	})
}
