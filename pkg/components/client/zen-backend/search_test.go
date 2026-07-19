package zenbackend

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

func TestSearchClient_Search_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodGet, "/search", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "hover" {
			t.Fatalf("expected q=hover, got %q", r.URL.RawQuery)
		}
		if r.URL.Query().Get("scope") != "all" {
			t.Fatalf("expected scope=all, got %q", r.URL.RawQuery)
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Fatalf("expected limit=10, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.SearchResponse{
			Query: "hover",
			Scope: "all",
			Cards: []*entity.SearchHit{{Kind: "card", ID: "c1", Title: "Hover", Snippet: "..."}},
		})
	})

	resp, err := srv.client().Search().Search(context.Background(), "hover", "all", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Cards) != 1 || resp.Cards[0].Title != "Hover" {
		t.Fatalf("bad response: %+v", resp)
	}
}
