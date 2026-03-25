package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockBlacklistUserHandler(usersJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/blacklist/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, usersJSON)
	})
	return mux
}

const blacklistUsersListJSON = `[
  {"userid": "alice", "name": "Alice Smith",  "email": "alice@example.com"},
  {"userid": "bob",   "name": "Bob Jones",    "email": "bob@example.com"}
]`

func TestBlacklistUserDataSource_FindByEmail(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockBlacklistUserHandler(blacklistUsersListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_blacklist_user" "test" {
  email = "alice@example.com"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_blacklist_user.test", "email", "alice@example.com"),
					resource.TestCheckResourceAttr("data.remscontent_blacklist_user.test", "id", "alice"),
				),
			},
		},
	})
}

func TestBlacklistUserDataSource_APIWrongEmail(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockBlacklistUserHandler(blacklistUsersListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_blacklist_user" "test" {
  email = "wrong-email@example.com"
}`,
				ExpectError: regexp.MustCompile(`User Not Found for Blacklist`),
			},
		},
	})
}

func TestBlacklistUserDataSource_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/blacklist/users", func(w http.ResponseWriter, r *http.Request) {
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
data "remscontent_blacklist_user" "test" {
  email = "alice@example.com"
}`,
				ExpectError: regexp.MustCompile(`Error fetching user  available for blacklist`),
			},
		},
	})
}
