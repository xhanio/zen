//go:build sqlite_fts5

package zenbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// migrationsDir returns the absolute path to the production migrations
// directory, computed from this test file's location so tests don't depend
// on the working directory at run time.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// pkg/components/server/zen-backend/manager_test.go → 4 levels up to repo root
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	dir, err := filepath.Abs(filepath.Join(repoRoot, "env", "default", "config", "zen-backend", "migrations"))
	if err != nil {
		t.Fatalf("abs migrations dir: %v", err)
	}
	return dir
}

func writeTestConfig(t *testing.T, dbPath string, httpPort int) string {
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

func waitForOK(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for 200 from %s", url)
}

func TestDaemonBoot_HealthAndReady(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-test.db")
	cfgPath := writeTestConfig(t, dbPath, port)

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
	waitForOK(t, base+"/readyz", 5*time.Second)

	// POST /api/v1/groups creates a group.
	createReq, _ := http.NewRequest(http.MethodPost, base+"/groups",
		strings.NewReader(`{"name":"integration-test-group"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("POST /groups: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		createResp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", createResp.StatusCode, body)
	}
	var created struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	createResp.Body.Close()
	if created.ID == "" || created.Name != "integration-test-group" {
		t.Fatalf("bad created group: %+v", created)
	}

	// GET /api/v1/groups returns the new group.
	listResp, err := http.Get(base + "/groups") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /groups: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		listResp.Body.Close()
		t.Fatalf("expected 200, got %d", listResp.StatusCode)
	}
	var list []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	listResp.Body.Close()
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected one group, got: %+v", list)
	}

	// Sibling-name conflict: POST a duplicate group → 409.
	dupReq, _ := http.NewRequest(http.MethodPost, base+"/groups",
		strings.NewReader(`{"name":"integration-test-group"}`))
	dupReq.Header.Set("Content-Type", "application/json")
	dupResp, err := http.DefaultClient.Do(dupReq)
	if err != nil {
		t.Fatalf("duplicate POST /groups: %v", err)
	}
	if dupResp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(dupResp.Body)
		dupResp.Body.Close()
		t.Fatalf("expected 409 on sibling conflict, got %d: %s",
			dupResp.StatusCode, body)
	}
	dupResp.Body.Close()

	// Tags are group-scoped and created implicitly by card writes. Seed
	// "original" by tagging a card in the group, then rename it within the
	// group and list the group's tags.
	seedBody := `{"title":"Seed","group_id":"` + created.ID + `","tags":["original"]}`
	seedReq, _ := http.NewRequest(http.MethodPost, base+"/cards", strings.NewReader(seedBody))
	seedReq.Header.Set("Content-Type", "application/json")
	seedResp, err := http.DefaultClient.Do(seedReq)
	if err != nil {
		t.Fatalf("POST /cards (seed tag): %v", err)
	}
	if seedResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(seedResp.Body)
		seedResp.Body.Close()
		t.Fatalf("expected 201 seeding tag via card, got %d: %s", seedResp.StatusCode, body)
	}
	seedResp.Body.Close()

	tagRename, _ := http.NewRequest(http.MethodPut, base+"/groups/"+created.ID+"/tags/original",
		strings.NewReader(`{"new_name":"renamed"}`))
	tagRename.Header.Set("Content-Type", "application/json")
	tagRenameResp, err := http.DefaultClient.Do(tagRename)
	if err != nil {
		t.Fatalf("PUT /groups/:id/tags/original: %v", err)
	}
	if tagRenameResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tagRenameResp.Body)
		tagRenameResp.Body.Close()
		t.Fatalf("expected 200 on rename, got %d: %s",
			tagRenameResp.StatusCode, body)
	}
	tagRenameResp.Body.Close()

	tagListResp, err := http.Get(base + "/groups/" + created.ID + "/tags") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /groups/:id/tags: %v", err)
	}
	var tagList []struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(tagListResp.Body).Decode(&tagList)
	tagListResp.Body.Close()
	if len(tagList) != 1 || tagList[0].Name != "renamed" {
		t.Fatalf("expected one tag named 'renamed', got: %+v", tagList)
	}

	// === M3 scenarios ===

	parentGroupID := created.ID

	// Create a card in the group with a tag.
	cardBody := `{"title":"Card","group_id":"` + parentGroupID + `","tags":["renamed"]}`
	cardReq, _ := http.NewRequest(http.MethodPost, base+"/cards", strings.NewReader(cardBody))
	cardReq.Header.Set("Content-Type", "application/json")
	cardResp, err := http.DefaultClient.Do(cardReq)
	if err != nil {
		t.Fatalf("POST /cards: %v", err)
	}
	if cardResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(cardResp.Body)
		cardResp.Body.Close()
		t.Fatalf("expected 201 on card create, got %d: %s", cardResp.StatusCode, body)
	}
	var card struct {
		ID   string   `json:"id"`
		Tags []string `json:"tags"`
	}
	_ = json.NewDecoder(cardResp.Body).Decode(&card)
	cardResp.Body.Close()
	if len(card.Tags) != 1 || card.Tags[0] != "renamed" {
		t.Fatalf("expected card tag 'renamed', got %v", card.Tags)
	}

	// Recursive delete of the group: card should be gone too.
	delGroupReq, _ := http.NewRequest(http.MethodDelete, base+"/groups/"+parentGroupID+"?recursive=true", nil)
	delGroupResp, err := http.DefaultClient.Do(delGroupReq)
	if err != nil {
		t.Fatalf("DELETE /groups recursive: %v", err)
	}
	if delGroupResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(delGroupResp.Body)
		delGroupResp.Body.Close()
		t.Fatalf("expected 204 on recursive group delete, got %d: %s",
			delGroupResp.StatusCode, body)
	}
	delGroupResp.Body.Close()

	// Card must now be 404.
	cardGoneResp, _ := http.Get(base + "/cards/" + card.ID) //nolint:noctx
	if cardGoneResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(cardGoneResp.Body)
		cardGoneResp.Body.Close()
		t.Fatalf("expected 404 for card after cascade, got %d: %s",
			cardGoneResp.StatusCode, body)
	}
	cardGoneResp.Body.Close()
}

func TestDaemonBoot_DecomposeAndComposeBodyEndpoints(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-decompose-compose.db")
	cfgPath := writeTestConfig(t, dbPath, port)

	srv := New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop(true) })

	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	waitForOK(t, base+"/healthz", 5*time.Second)

	// Seed: group + parent card.
	gResp, err := http.Post(base+"/groups", "application/json", //nolint:noctx
		strings.NewReader(`{"name":"decomp"}`))
	if err != nil {
		t.Fatalf("POST /groups: %v", err)
	}
	var g struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&g)
	gResp.Body.Close()

	pResp, err := http.Post(base+"/cards", "application/json", //nolint:noctx
		strings.NewReader(fmt.Sprintf(`{"title":"parent","content":"x","group_id":%q}`, g.ID)))
	if err != nil {
		t.Fatalf("POST /cards: %v", err)
	}
	var parent struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(pResp.Body).Decode(&parent)
	pResp.Body.Close()

	// POST /cards/decompose with parent_card_id in body.
	decBody := fmt.Sprintf(`{"parent_card_id":%q,"cards":[
		{"title":"c1","content":"a","group_id":%q},
		{"title":"c2","content":"b","group_id":%q}
	]}`, parent.ID, g.ID, g.ID)
	dResp, err := http.Post(base+"/cards/decompose", "application/json", strings.NewReader(decBody)) //nolint:noctx
	if err != nil {
		t.Fatalf("decompose POST: %v", err)
	}
	if dResp.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(dResp.Body)
		dResp.Body.Close()
		t.Fatalf("decompose status %d, body %s", dResp.StatusCode, bb)
	}
	var dOut struct {
		Cards []struct {
			ID string `json:"id"`
		} `json:"cards"`
	}
	_ = json.NewDecoder(dResp.Body).Decode(&dOut)
	dResp.Body.Close()
	if len(dOut.Cards) != 2 {
		t.Fatalf("want 2 children, got %d", len(dOut.Cards))
	}

	// POST /cards/compose merging the two new children.
	cBody := fmt.Sprintf(`{"source_card_ids":[%q,%q],"target":{
		"title":"merged","content":"merged body","group_id":%q
	}}`, dOut.Cards[0].ID, dOut.Cards[1].ID, g.ID)
	cResp, err := http.Post(base+"/cards/compose", "application/json", strings.NewReader(cBody)) //nolint:noctx
	if err != nil {
		t.Fatalf("compose POST: %v", err)
	}
	if cResp.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(cResp.Body)
		cResp.Body.Close()
		t.Fatalf("compose status %d, body %s", cResp.StatusCode, bb)
	}
	var cOut struct {
		Card struct {
			ID      string `json:"id"`
			Genesis string `json:"genesis"`
		} `json:"card"`
	}
	_ = json.NewDecoder(cResp.Body).Decode(&cOut)
	cResp.Body.Close()
	// Genesis names the sources by title, never by ID.
	if cOut.Card.ID == "" || cOut.Card.Genesis != "Composed from c1, c2" {
		t.Fatalf("bad composed card: %+v", cOut.Card)
	}
	for _, id := range []string{dOut.Cards[0].ID, dOut.Cards[1].ID} {
		if strings.Contains(cOut.Card.Genesis, id) {
			t.Fatalf("genesis leaks a source ID: %q", cOut.Card.Genesis)
		}
	}

	// Sources should be in Trash.
	tResp, err := http.Get(base + "/trash") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /trash: %v", err)
	}
	var trash struct {
		Cards []struct {
			ID string `json:"id"`
		} `json:"cards"`
	}
	_ = json.NewDecoder(tResp.Body).Decode(&trash)
	tResp.Body.Close()
	seen := map[string]bool{}
	for _, c := range trash.Cards {
		seen[c.ID] = true
	}
	if !seen[dOut.Cards[0].ID] || !seen[dOut.Cards[1].ID] {
		t.Fatalf("expected both source IDs in /trash, got %+v", trash.Cards)
	}
}

func TestDaemonBoot_References_RoundTrip(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-references.db")
	cfgPath := writeTestConfig(t, dbPath, port)

	srv := New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop(true) })

	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	waitForOK(t, base+"/healthz", 5*time.Second)

	mkJSON := func(path, body string) *http.Response {
		resp, err := http.Post(base+path, "application/json", strings.NewReader(body)) //nolint:noctx
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		return resp
	}
	getID := func(r *http.Response) string {
		var out struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&out)
		r.Body.Close()
		return out.ID
	}

	gID := getID(mkJSON("/groups", `{"name":"refs"}`))
	srcID := getID(mkJSON("/cards", fmt.Sprintf(`{"title":"src","content":"hello world","group_id":%q}`, gID)))
	derID := getID(mkJSON("/cards", fmt.Sprintf(`{"title":"der","content":"about hello","group_id":%q}`, gID)))
	convID := getID(mkJSON("/conversations", `{"title":""}`))

	body := fmt.Sprintf(`{"source_card_id":%q,"derived_card_id":%q,"conversation_id":%q,"selection_text":"hello"}`, srcID, derID, convID)
	refResp := mkJSON("/references", body)
	if refResp.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(refResp.Body)
		refResp.Body.Close()
		t.Fatalf("create ref: status %d, body %s", refResp.StatusCode, bb)
	}
	refID := getID(refResp)

	cardResp, _ := http.Get(base + "/cards/" + srcID) //nolint:noctx
	var cardOut struct {
		References []struct {
			ID            string `json:"id"`
			SelectionText string `json:"selection_text"`
		} `json:"references"`
	}
	_ = json.NewDecoder(cardResp.Body).Decode(&cardOut)
	cardResp.Body.Close()
	if len(cardOut.References) != 1 || cardOut.References[0].ID != refID || cardOut.References[0].SelectionText != "hello" {
		t.Fatalf("expected one reference attached to src card, got %+v", cardOut.References)
	}

	delReq, _ := http.NewRequest(http.MethodDelete, base+"/references/"+refID, nil)
	delResp, _ := http.DefaultClient.Do(delReq)
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete ref: status %d", delResp.StatusCode)
	}
	delResp.Body.Close()

	cardResp2, _ := http.Get(base + "/cards/" + srcID) //nolint:noctx
	var cardOut2 struct {
		References []any `json:"references"`
	}
	_ = json.NewDecoder(cardResp2.Body).Decode(&cardOut2)
	cardResp2.Body.Close()
	if len(cardOut2.References) != 0 {
		t.Fatalf("expected empty references after delete, got %+v", cardOut2.References)
	}
}

func TestDaemonBoot_CardCreate_WithInlineReference(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-inline-ref.db")
	cfgPath := writeTestConfig(t, dbPath, port)

	srv := New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop(true) })

	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	waitForOK(t, base+"/healthz", 5*time.Second)

	post := func(path, body string) *http.Response {
		resp, _ := http.Post(base+path, "application/json", strings.NewReader(body)) //nolint:noctx
		return resp
	}
	getID := func(r *http.Response) string {
		var out struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&out)
		r.Body.Close()
		return out.ID
	}

	gID := getID(post("/groups", `{"name":"inline-ref"}`))
	parentID := getID(post("/cards", fmt.Sprintf(`{"title":"src","content":"hello","group_id":%q}`, gID)))
	convID := getID(post("/conversations", `{"title":""}`))

	body := fmt.Sprintf(`{
        "title":"der","content":"about hello","group_id":%q,
        "parent_card_id":%q,"source_conversation_id":%q,
        "reference":{"selection_text":"hello"}
    }`, gID, parentID, convID)
	cardResp := post("/cards", body)
	if cardResp.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(cardResp.Body)
		cardResp.Body.Close()
		t.Fatalf("inline ref card.create: status %d, body %s", cardResp.StatusCode, bb)
	}
	cardResp.Body.Close()

	getResp, _ := http.Get(base + "/cards/" + parentID) //nolint:noctx
	var got struct {
		References []struct {
			SelectionText  string  `json:"selection_text"`
			ConversationID *string `json:"conversation_id"`
		} `json:"references"`
	}
	_ = json.NewDecoder(getResp.Body).Decode(&got)
	getResp.Body.Close()
	if len(got.References) != 1 || got.References[0].SelectionText != "hello" {
		t.Fatalf("expected one reference with selection_text=hello, got %+v", got.References)
	}
	if got.References[0].ConversationID == nil || *got.References[0].ConversationID != convID {
		t.Fatalf("expected conversation_id=%q, got %v", convID, got.References[0].ConversationID)
	}

	body2 := fmt.Sprintf(`{
        "title":"der2","content":"x","group_id":%q,
        "parent_card_id":%q,
        "reference":{"selection_text":"hel"}
    }`, gID, parentID)
	cardResp2 := post("/cards", body2)
	if cardResp2.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(cardResp2.Body)
		cardResp2.Body.Close()
		t.Fatalf("inline ref null-conv card.create: status %d, body %s", cardResp2.StatusCode, bb)
	}
	cardResp2.Body.Close()
	getResp2, _ := http.Get(base + "/cards/" + parentID) //nolint:noctx
	var got2 struct {
		References []struct {
			SelectionText  string  `json:"selection_text"`
			ConversationID *string `json:"conversation_id"`
		} `json:"references"`
	}
	_ = json.NewDecoder(getResp2.Body).Decode(&got2)
	getResp2.Body.Close()
	if len(got2.References) != 2 {
		t.Fatalf("expected 2 references after null-conv create, got %d", len(got2.References))
	}
	nulls := 0
	for _, r := range got2.References {
		if r.ConversationID == nil {
			nulls++
		}
	}
	if nulls != 1 {
		t.Fatalf("expected exactly 1 null-conv reference, got %d", nulls)
	}
}

func TestDaemonBoot_ChildrenEndpoint(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	dbPath := filepath.Join(t.TempDir(), "zen-children.db")
	cfgPath := writeTestConfig(t, dbPath, port)
	srv := New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop(true) })

	base := fmt.Sprintf("http://127.0.0.1:%d/api/v1", port)
	waitForOK(t, base+"/healthz", 5*time.Second)

	gResp, _ := http.Post(base+"/groups", "application/json", //nolint:noctx
		strings.NewReader(`{"name":"kids"}`))
	var g struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&g)
	gResp.Body.Close()

	pResp, _ := http.Post(base+"/cards", "application/json", //nolint:noctx
		strings.NewReader(fmt.Sprintf(`{"title":"P","content":"x","group_id":%q}`, g.ID)))
	var parent struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(pResp.Body).Decode(&parent)
	pResp.Body.Close()

	dec := fmt.Sprintf(`{"parent_card_id":%q,"cards":[
		{"title":"c1","content":"a"},{"title":"c2","content":"b"}
	]}`, parent.ID)
	dResp, _ := http.Post(base+"/cards/decompose", "application/json", strings.NewReader(dec)) //nolint:noctx
	if dResp.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(dResp.Body)
		dResp.Body.Close()
		t.Fatalf("decompose status %d, body %s", dResp.StatusCode, bb)
	}
	dResp.Body.Close()

	cResp, err := http.Get(base + "/cards/" + parent.ID + "/children") //nolint:noctx
	if err != nil {
		t.Fatalf("GET children: %v", err)
	}
	if cResp.StatusCode != http.StatusOK {
		bb, _ := io.ReadAll(cResp.Body)
		cResp.Body.Close()
		t.Fatalf("children status %d, body %s", cResp.StatusCode, bb)
	}
	var out struct {
		Cards []struct {
			Title string `json:"title"`
		} `json:"cards"`
	}
	_ = json.NewDecoder(cResp.Body).Decode(&out)
	cResp.Body.Close()
	if len(out.Cards) != 2 || out.Cards[0].Title != "c1" || out.Cards[1].Title != "c2" {
		t.Fatalf("unexpected children payload: %+v", out)
	}
}
