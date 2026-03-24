package data_sources_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func mockActorHandler(actorsJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workflows/actors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, actorsJSON)
	})
	return mux
}

const actorsListJSON = `[
  {"userid": "alice", "name": "Alice Smith",  "email": "alice@example.com"},
  {"userid": "bob",   "name": "Bob Jones",    "email": "bob@example.com"}
]`

func TestActorDataSource_FindByEmail(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockActorHandler(actorsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_actor" "test" {
  email = "bob@example.com"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.remscontent_actor.test", "email", "bob@example.com"),
					resource.TestCheckResourceAttr("data.remscontent_actor.test", "id", "bob"),
				),
			},
		},
	})
}

func TestActorDataSource_WrongEmail(t *testing.T) {
	factories, cleanup := testProviderWithMockServer(t, mockActorHandler(actorsListJSON))
	defer cleanup()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "remscontent" {}
data "remscontent_actor" "test" {
  email = "wrong-email@example.com"
}`,
				ExpectError: regexp.MustCompile(`No actor found with email: wrong-email@example.com`),
			},
		},
	})
}

func TestActorDataSource_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workflows/actors", func(w http.ResponseWriter, r *http.Request) {
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
data "remscontent_actor" "test" {
  email = "alice@example.com"
}`,
				ExpectError: regexp.MustCompile(`Error fetching actors list`),
			},
		},
	})
}
