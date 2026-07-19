package zenbackend

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

func TestCardClient_Create_WithTags(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodPost, "/cards", func(w http.ResponseWriter, r *http.Request) {
		var got api.CreateCardRequest
		_ = json.NewDecoder(r.Body).Decode(&got)
		if len(got.Tags) != 2 || got.Tags[0] != "go" || got.Tags[1] != "rust" {
			t.Fatalf("bad tags: %+v", got.Tags)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(entity.Card{
			ID: "c1", Title: "x", GroupID: "g1",
			Tags: []string{"go", "rust"},
		})
	})

	c, err := srv.client().Card().Create(context.Background(), api.CreateCardRequest{
		Title: "x", GroupID: "g1", Tags: []string{"go", "rust"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(c.Tags) != 2 {
		t.Fatalf("bad tags: %+v", c.Tags)
	}
}

func TestCardClient_List_WithGroupFilter(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodGet, "/cards", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("group_id") != "g1" {
			t.Fatalf("missing query param: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]entity.Card{{ID: "c1"}})
	})

	gid := "g1"
	cs, err := srv.client().Card().List(context.Background(), &gid, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("got %d", len(cs))
	}
}

func TestCardClient_Children_HitsCorrectPath(t *testing.T) {
	srv := newStubBackend(t)
	var gotQuery string
	srv.on(http.MethodGet, "/cards/01PID/children", func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cards":[{"id":"01ABC","title":"c1"}]}`))
	})

	got, err := srv.client().Card().Children(context.Background(), "01PID", true)
	if err != nil {
		t.Fatalf("Children: %v", err)
	}
	if gotQuery != "include_trashed=true" {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
	if len(got) != 1 || got[0].ID != "01ABC" {
		t.Fatalf("unexpected result: %+v", got)
	}
}
