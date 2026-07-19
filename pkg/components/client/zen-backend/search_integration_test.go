//go:build sqlite_fts5

package zenbackend_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	client "github.com/xhanio/zen/pkg/components/client/zen-backend"
	server "github.com/xhanio/zen/pkg/components/server/zen-backend"
	"github.com/xhanio/zen/pkg/types/api"
)

func bootBackendFTS5(t *testing.T) (base string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "zen-client-fts5.db")
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
	go func() { _ = srv.Start(ctx) }()
	base = fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/healthz") //nolint:noctx
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base, func() { _ = srv.Stop(true); cancel() }
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	t.Fatalf("backend not ready")
	return "", nil
}

func TestClient_Search_Roundtrip(t *testing.T) {
	base, stop := bootBackendFTS5(t)
	defer stop()
	c := client.New(base)
	ctx := context.Background()

	g, err := c.Group().Create(ctx, api.CreateGroupRequest{Name: "search-client-it"})
	if err != nil {
		t.Fatalf("Group.Create: %v", err)
	}
	_, err = c.Card().Create(ctx, api.CreateCardRequest{
		Title: "Hover", Content: "Hummingbirds hover by flapping figure-eight wings.", GroupID: g.ID,
	})
	if err != nil {
		t.Fatalf("Card.Create: %v", err)
	}

	resp, err := c.Search().Search(ctx, "hover", "all", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Cards) != 1 || !strings.Contains(resp.Cards[0].Snippet, "<mark>") {
		t.Fatalf("expected one card hit with <mark> snippet, got: %+v", resp.Cards)
	}
}
