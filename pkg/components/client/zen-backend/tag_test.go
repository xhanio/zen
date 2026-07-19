package zenbackend

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

func TestTagClient_List_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodGet, "/groups/G1/tags", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]entity.Tag{{ID: "t1", GroupID: "G1", Name: "spec"}})
	})
	tags, err := srv.client().Tag().List(context.Background(), "G1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "spec" || tags[0].GroupID != "G1" {
		t.Fatalf("got %+v", tags)
	}
}

func TestTagClient_Rename_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodPut, "/groups/G1/tags/old", func(w http.ResponseWriter, r *http.Request) {
		var got api.RenameTagRequest
		_ = json.NewDecoder(r.Body).Decode(&got)
		if got.NewName != "new" {
			t.Fatalf("got new_name %q", got.NewName)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entity.Tag{ID: "tag", GroupID: "G1", Name: "new"})
	})

	tag, err := srv.client().Tag().Rename(context.Background(), "G1", "old", api.RenameTagRequest{NewName: "new"})
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if tag.Name != "new" {
		t.Fatalf("got %q", tag.Name)
	}
}

func TestTagClient_Delete_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodDelete, "/groups/G1/tags/old", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if err := srv.client().Tag().Delete(context.Background(), "G1", "old"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
