//go:build sqlite_fts5

package zenbackend_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/xhanio/errors"

	client "github.com/xhanio/zen/pkg/components/client/zen-backend"
	server "github.com/xhanio/zen/pkg/components/server/zen-backend"
	"github.com/xhanio/zen/pkg/types/api"
)

// realBackendMigrationsDir resolves the absolute path to the production
// migrations dir, regardless of cwd at test run time.
func realBackendMigrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// pkg/components/client/zen-backend/integration_test.go → 4 levels up to repo root
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	dir, err := filepath.Abs(filepath.Join(repoRoot, "env", "default", "config", "zen-backend", "migrations"))
	if err != nil {
		t.Fatalf("abs migrations dir: %v", err)
	}
	return dir
}

func bootBackend(t *testing.T) (base string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "zen-client-it.db")
	logPath := filepath.Join(dir, "app.log")
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := fmt.Sprintf(`
log:
  level: 0
  file: %s
  rotation:
    max_size: 1
    max_backups: 1
    max_age: 1

db:
  type: sqlite
  source:
    dbname: %s
  migration:
    dir: %s
  connection:
    max_open: 1
    max_idle: 1
    max_lifetime: 1h
    exec_timeout: 30s

api:
  http:
    host: 127.0.0.1
    port: %d
    prefix: /api/v1
`, logPath, dbPath, realBackendMigrationsDir(t), port)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	srv := server.New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	if err := srv.Init(ctx); err != nil {
		cancel()
		t.Fatalf("Init: %v", err)
	}
	go func() {
		_ = srv.Start(ctx)
	}()

	base = fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/healthz") //nolint:noctx
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				stop = func() {
					_ = srv.Stop(true)
					cancel()
				}
				return base, stop
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	t.Fatalf("backend did not become ready in 5s")
	return "", nil
}

func TestClient_GroupTagDocumentCard_Roundtrip(t *testing.T) {
	base, stop := bootBackend(t)
	defer stop()

	c := client.New(base)
	ctx := context.Background()

	// 1. Create a group.
	g, err := c.Group().Create(ctx, api.CreateGroupRequest{Name: "work"})
	if err != nil {
		t.Fatalf("Group.Create: %v", err)
	}
	if g.Name != "work" {
		t.Fatalf("got %q", g.Name)
	}

	// 2. Duplicate group → 409 (Conflict mapping).
	_, err = c.Group().Create(ctx, api.CreateGroupRequest{Name: "work"})
	if !errors.Is(err, errors.Conflict) {
		t.Fatalf("expected Conflict on duplicate, got: %v", err)
	}

	// 3. Create a card with a tag — the tag is created in the group implicitly
	//    (there is no direct tag-create endpoint under the per-group model).
	card, err := c.Card().Create(ctx, api.CreateCardRequest{
		Title: "Card", GroupID: g.ID, Tags: []string{"go"},
	})
	if err != nil {
		t.Fatalf("Card.Create: %v", err)
	}
	if len(card.Tags) != 1 || card.Tags[0] != "go" {
		t.Fatalf("expected tag attached, got %+v", card.Tags)
	}

	// 5. Get the card back via the client.
	got, err := c.Card().Get(ctx, card.ID)
	if err != nil {
		t.Fatalf("Card.Get: %v", err)
	}
	if got.ID != card.ID || len(got.Tags) != 1 {
		t.Fatalf("Card.Get mismatch: %+v", got)
	}

	// 6. Recursive delete of the group.
	if err := c.Group().Delete(ctx, g.ID, true); err != nil {
		t.Fatalf("Group.Delete recursive: %v", err)
	}

	// 9. Card now 404 (NotFound mapping).
	_, err = c.Card().Get(ctx, card.ID)
	if !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected NotFound after cascade, got: %v", err)
	}
}

func TestClient_CardUpdate_ReplacesTags(t *testing.T) {
	base, stop := bootBackend(t)
	defer stop()

	c := client.New(base)
	ctx := context.Background()

	g, err := c.Group().Create(ctx, api.CreateGroupRequest{Name: "tagwork"})
	if err != nil {
		t.Fatalf("Group.Create: %v", err)
	}
	created, err := c.Card().Create(ctx, api.CreateCardRequest{
		Title: "T", GroupID: g.ID, Tags: []string{"one", "two"},
	})
	if err != nil {
		t.Fatalf("Card.Create: %v", err)
	}

	tags := []string{"two", "three"}
	updated, err := c.Card().Update(ctx, created.ID, api.UpdateCardRequest{Tags: &tags})
	if err != nil {
		t.Fatalf("Card.Update: %v", err)
	}
	got := map[string]bool{}
	for _, n := range updated.Tags {
		got[n] = true
	}
	if !got["two"] || !got["three"] || len(updated.Tags) != 2 {
		t.Fatalf("expected {two,three}, got %v", updated.Tags)
	}
}
