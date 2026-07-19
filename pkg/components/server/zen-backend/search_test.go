//go:build sqlite_fts5

package zenbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeFTS5TestConfig is like writeTestConfig but targets migration version 2
// so the FTS5 schema is applied. Build-tagged because version 2 requires the
// sqlite_fts5 module.
func writeFTS5TestConfig(t *testing.T, dbPath string, httpPort int) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	logPath := filepath.Join(dir, "app.log")
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
`, logPath, dbPath, migrationsDir(t), httpPort)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return cfgPath
}

func TestDaemonBoot_Search(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-fts5-test.db")
	cfgPath := writeFTS5TestConfig(t, dbPath, port)

	srv := New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	go func() {
		if err := srv.Start(ctx); err != nil {
			t.Logf("Start returned: %v", err)
		}
	}()
	t.Cleanup(func() { _ = srv.Stop(true) })

	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	waitForOK(t, base+"/healthz", 5*time.Second)

	// Seed: create a group and a card with searchable content.
	groupID := postJSON(t, base+"/groups", `{"name":"search-test"}`).String("id")
	postJSON(t, base+"/cards", fmt.Sprintf(
		`{"title":"Hover flight","content":"Hummingbirds can hover by flapping their wings figure-eight style.","group_id":%q}`,
		groupID,
	))

	// Search for "hover".
	q := url.Values{}
	q.Set("q", "hover")
	q.Set("scope", "all")
	q.Set("limit", "10")
	resp, err := http.Get(base + "/search?" + q.Encode()) //nolint:noctx
	if err != nil {
		t.Fatalf("GET /search: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	var got struct {
		Cards []struct {
			Snippet string `json:"snippet"`
			Title   string `json:"title"`
		} `json:"cards"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resp.Body.Close()
	if len(got.Cards) != 1 {
		t.Fatalf("want 1 card hit, got %d: %+v", len(got.Cards), got.Cards)
	}
	if !strings.Contains(got.Cards[0].Snippet, "<mark>") {
		t.Fatalf("expected snippet with <mark>, got %q", got.Cards[0].Snippet)
	}
}

func TestDaemonBoot_SearchMessages(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-fts5-msg.db")
	cfgPath := writeFTS5TestConfig(t, dbPath, port)

	srv := New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	go func() {
		if err := srv.Start(ctx); err != nil {
			t.Logf("Start returned: %v", err)
		}
	}()
	t.Cleanup(func() { _ = srv.Stop(true) })

	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	waitForOK(t, base+"/healthz", 5*time.Second)

	convID := postJSON(t, base+"/conversations", `{"title":""}`).String("id")
	postJSON(t, base+"/conversations/"+convID+"/messages",
		`{"role":"user","content":"Tell me about rainbows in the sky"}`)

	q := url.Values{}
	q.Set("q", "rainbows")
	q.Set("scope", "messages")
	q.Set("limit", "10")
	resp, err := http.Get(base + "/search?" + q.Encode()) //nolint:noctx
	if err != nil {
		t.Fatalf("GET /search: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	var got struct {
		Messages []struct {
			Kind           string  `json:"kind"`
			ID             string  `json:"id"`
			Title          string  `json:"title"`
			Snippet        string  `json:"snippet"`
			ConversationID *string `json:"conversation_id"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resp.Body.Close()
	if len(got.Messages) != 1 {
		t.Fatalf("want 1 message hit, got %d: %+v", len(got.Messages), got.Messages)
	}
	h := got.Messages[0]
	if h.Kind != "message" {
		t.Fatalf("expected kind=message, got %q", h.Kind)
	}
	if h.ConversationID == nil || *h.ConversationID != convID {
		t.Fatalf("expected conversation_id=%q, got %v", convID, h.ConversationID)
	}
	if !strings.Contains(h.Snippet, "<mark>") {
		t.Fatalf("expected snippet with <mark>, got %q", h.Snippet)
	}
	// Title carries the auto-derived conversation title (first user message)
	if !strings.Contains(strings.ToLower(h.Title), "rainbows") {
		t.Fatalf("expected conversation title to mention rainbows, got %q", h.Title)
	}
}

// jsonObj wraps a decoded JSON map for ergonomic test field extraction.
type jsonObj map[string]any

func (j jsonObj) String(key string) string {
	v, _ := j[key].(string)
	return v
}

func postJSON(t *testing.T, url, body string) jsonObj {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s: status %d body %s", url, resp.StatusCode, b)
	}
	var out jsonObj
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}
