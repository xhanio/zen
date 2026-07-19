package zenbackend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

// stubBackend returns a server that serves canned responses keyed by
// "<method> <path>".
type stubBackend struct {
	t        *testing.T
	handlers map[string]http.HandlerFunc
	server   *httptest.Server
}

func newStubBackend(t *testing.T) *stubBackend {
	t.Helper()
	s := &stubBackend{t: t, handlers: map[string]http.HandlerFunc{}}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if h, ok := s.handlers[key]; ok {
			h(w, r)
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *stubBackend) on(method, path string, h http.HandlerFunc) {
	s.handlers[method+" "+path] = h
}

func (s *stubBackend) client() Client {
	return New(s.server.URL)
}

func TestGroupClient_Create_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodPost, "/groups", func(w http.ResponseWriter, r *http.Request) {
		var got api.CreateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if got.Name != "work" {
			t.Fatalf("got name %q, want %q", got.Name, "work")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(entity.Group{ID: "01H_FAKE_ULID", Name: "work"})
	})

	g, err := srv.client().Group().Create(context.Background(), api.CreateGroupRequest{Name: "work"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.ID != "01H_FAKE_ULID" || g.Name != "work" {
		t.Fatalf("bad response: %+v", g)
	}
}

func TestGroupClient_Get_NotFound(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodGet, "/groups/missing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"source":"zen-backend","status":404,"kind":"NotFound","message":"not here"}`))
	})

	_, err := srv.client().Group().Get(context.Background(), "missing")
	if !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestGroupClient_List_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodGet, "/groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]entity.Group{
			{ID: "a", Name: "alpha"},
			{ID: "b", Name: "beta"},
		})
	})

	gs, err := srv.client().Group().List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(gs) != 2 || gs[0].Name != "alpha" || gs[1].Name != "beta" {
		t.Fatalf("bad list: %+v", gs)
	}
}

func TestGroupClient_Delete_RecursiveQueryParam(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodDelete, "/groups/abc", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "recursive=true" {
			t.Fatalf("expected query recursive=true, got %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := srv.client().Group().Delete(context.Background(), "abc", true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestGroupClient_Update_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodPut, "/groups/abc", func(w http.ResponseWriter, r *http.Request) {
		var got api.UpdateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if got.Name == nil || *got.Name != "renamed" {
			t.Fatalf("bad request body: %+v", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entity.Group{ID: "abc", Name: *got.Name})
	})

	name := "renamed"
	g, err := srv.client().Group().Update(context.Background(), "abc", api.UpdateGroupRequest{Name: &name})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if g.Name != "renamed" {
		t.Fatalf("got %q", g.Name)
	}
	if !strings.HasPrefix(g.ID, "abc") {
		t.Fatalf("bad id: %q", g.ID)
	}
}
