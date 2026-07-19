//go:build sqlite_fts5

package zenmcp_test

import (
	"bufio"
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

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	backend "github.com/xhanio/zen/pkg/components/server/zen-backend"
	mcp "github.com/xhanio/zen/pkg/components/server/zen-mcp"
	channel "github.com/xhanio/zen/pkg/components/server/zen-channel"
)

func realBackendMigrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	dir, err := filepath.Abs(filepath.Join(repoRoot, "env", "default", "config", "zen-backend", "migrations"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return dir
}

func grabPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func waitForOK(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
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
	t.Fatalf("timed out waiting for %s", url)
}

func bootBackend(t *testing.T) (port int, stop func()) {
	t.Helper()
	port = grabPort(t)
	dir := t.TempDir()
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
`, filepath.Join(dir, "be.log"), filepath.Join(dir, "be.db"), realBackendMigrationsDir(t), port)
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write be cfg: %v", err)
	}
	srv := backend.New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	if err := srv.Init(ctx); err != nil {
		cancel()
		t.Fatalf("backend Init: %v", err)
	}
	go func() { _ = srv.Start(ctx) }()
	waitForOK(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1/healthz", port))
	return port, func() { _ = srv.Stop(true); cancel() }
}

func bootMCP(t *testing.T, backendURL string) (port int, stop func()) {
	t.Helper()
	port = grabPort(t)
	dir := t.TempDir()
	cfg := fmt.Sprintf(`
log:
  level: 0
  file: %s
  rotation:
    max_size: 1
    max_backups: 1
    max_age: 1
backend:
  url: %s
api:
  http:
    host: 127.0.0.1
    port: %d
    prefix: /api/v1
`, filepath.Join(dir, "mcp.log"), backendURL, port)
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write mcp cfg: %v", err)
	}
	srv := mcp.New(cfgPath)
	ctx, cancel := context.WithCancel(context.Background())
	if err := srv.Init(ctx); err != nil {
		cancel()
		t.Fatalf("mcp Init: %v", err)
	}
	go func() { _ = srv.Start(ctx) }()

	// MCP daemon has no /healthz; poll the MCP endpoint until any non-network
	// error comes back (handler is responding with framingo's normal pipeline).
	deadline := time.Now().Add(5 * time.Second)
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", port)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodPost, url, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			return port, func() { _ = srv.Stop(true); cancel() }
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	t.Fatalf("mcp daemon not ready")
	return 0, nil
}

func TestMCP_EndToEnd_GroupCreateAndList(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()

	mcpPort, stopMCP := bootMCP(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort))
	defer stopMCP()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	createResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "group.create",
		Arguments: map[string]any{"name": "from-mcp"},
	})
	if err != nil {
		t.Fatalf("call group.create: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("group.create returned IsError=true: %+v", createResult)
	}

	listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "group.list",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call group.list: %v", err)
	}
	if listResult.IsError {
		t.Fatalf("group.list returned IsError=true: %+v", listResult)
	}
	body := fmt.Sprintf("%v", listResult.StructuredContent)
	if !contains(body, "from-mcp") {
		t.Fatalf("expected listed groups to include from-mcp, got: %s", body)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMCP_TagList_ScopedToGroup(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	mcpPort, stopMCP := bootMCP(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort))
	defer stopMCP()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	gc, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "group.create", Arguments: map[string]any{"name": "design"},
	})
	if err != nil || gc.IsError {
		t.Fatalf("group.create: %v %+v", err, gc)
	}
	gid := gc.StructuredContent.(map[string]any)["group"].(map[string]any)["id"].(string)

	if _, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "card.create", Arguments: map[string]any{"title": "c", "group_id": gid, "tags": []any{"spec"}},
	}); err != nil {
		t.Fatalf("card.create: %v", err)
	}

	lr, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "tag.list", Arguments: map[string]any{"group_id": gid},
	})
	if err != nil || lr.IsError {
		t.Fatalf("tag.list: %v %+v", err, lr)
	}
	if body := fmt.Sprintf("%v", lr.StructuredContent); !contains(body, "spec") {
		t.Fatalf("tag.list did not return the scoped tag, got: %s", body)
	}
}

func TestMCP_GroupGet_ReturnsRule(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	mcpPort, stopMCP := bootMCP(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort))
	defer stopMCP()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	createResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "group.create",
		Arguments: map[string]any{"name": "design", "rule": "Chinese + HTML."},
	})
	if err != nil || createResult.IsError {
		t.Fatalf("group.create: %v %+v", err, createResult)
	}
	gid := createResult.StructuredContent.(map[string]any)["group"].(map[string]any)["id"].(string)

	getResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "group.get",
		Arguments: map[string]any{"id": gid},
	})
	if err != nil || getResult.IsError {
		t.Fatalf("group.get: %v %+v", err, getResult)
	}
	if body := fmt.Sprintf("%v", getResult.StructuredContent); !contains(body, "Chinese + HTML.") {
		t.Fatalf("group.get did not return the rule, got: %s", body)
	}
}

func TestMCP_CardCreate_EchoesGroupRule(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	mcpPort, stopMCP := bootMCP(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort))
	defer stopMCP()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	mkGroup := func(args map[string]any) string {
		res, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: "group.create", Arguments: args})
		if err != nil || res.IsError {
			t.Fatalf("group.create: %v %+v", err, res)
		}
		return res.StructuredContent.(map[string]any)["group"].(map[string]any)["id"].(string)
	}

	// group WITH a rule → card.create echoes it
	gid := mkGroup(map[string]any{"name": "design", "rule": "Translate into Chinese."})
	res, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "card.create",
		Arguments: map[string]any{"title": "Spec", "content": "hi", "group_id": gid},
	})
	if err != nil || res.IsError {
		t.Fatalf("card.create: %v %+v", err, res)
	}
	if body := fmt.Sprintf("%v", res.StructuredContent); !contains(body, "Translate into Chinese.") {
		t.Fatalf("card.create did not echo group_rule, got: %s", body)
	}

	// group WITHOUT a rule → no group_rule key
	pgid := mkGroup(map[string]any{"name": "notes"})
	res2, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "card.create",
		Arguments: map[string]any{"title": "n", "content": "x", "group_id": pgid},
	})
	if err != nil || res2.IsError {
		t.Fatalf("card.create (plain): %v %+v", err, res2)
	}
	if body := fmt.Sprintf("%v", res2.StructuredContent); contains(body, "group_rule") {
		t.Fatalf("expected no group_rule for a ruleless group, got: %s", body)
	}
}

func TestMCP_EndToEnd_ConversationListAndGet(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()

	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	// Seed a standalone conversation with one user message via REST.
	convResp, err := http.Post(beURL+"/conversations", "application/json",
		strings.NewReader(`{"title":""}`))
	if err != nil {
		t.Fatalf("create conv: %v", err)
	}
	var convOut struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(convResp.Body).Decode(&convOut)
	convResp.Body.Close()

	msgResp, err := http.Post(beURL+"/conversations/"+convOut.ID+"/messages",
		"application/json", strings.NewReader(`{"role":"user","content":"hello"}`))
	if err != nil {
		t.Fatalf("append msg: %v", err)
	}
	msgResp.Body.Close()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "conversation.list",
		Arguments: map[string]any{"pending": true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if listResult.IsError {
		t.Fatalf("list IsError: %+v", listResult)
	}
	body := fmt.Sprintf("%v", listResult.StructuredContent)
	if !contains(body, convOut.ID) {
		t.Fatalf("expected pending list to include %s, got: %s", convOut.ID, body)
	}

	getResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "conversation.get",
		Arguments: map[string]any{"id": convOut.ID},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getResult.IsError {
		t.Fatalf("get IsError: %+v", getResult)
	}
	if !contains(fmt.Sprintf("%v", getResult.StructuredContent), "hello") {
		t.Fatalf("get missing user message")
	}
}

func TestMCP_EndToEnd_CardWithProvenance(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()

	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	// Seed a group + parent card + conversation via REST.
	gResp, _ := http.Post(beURL+"/groups", "application/json",
		strings.NewReader(`{"name":"prov"}`))
	var gOut struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&gOut)
	gResp.Body.Close()

	pResp, _ := http.Post(beURL+"/cards", "application/json",
		strings.NewReader(`{"title":"parent","group_id":"`+gOut.ID+`"}`))
	var pOut struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(pResp.Body).Decode(&pOut)
	pResp.Body.Close()

	cResp, _ := http.Post(beURL+"/conversations", "application/json",
		strings.NewReader(`{"title":"src"}`))
	var cOut struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(cResp.Body).Decode(&cOut)
	cResp.Body.Close()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-prov-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "card.create",
		Arguments: map[string]any{
			"title":                  "derived",
			"group_id":               gOut.ID,
			"parent_card_id":         pOut.ID,
			"source_conversation_id": cOut.ID,
		},
	})
	if err != nil {
		t.Fatalf("card.create: %v", err)
	}
	if result.IsError {
		t.Fatalf("card.create IsError: %+v", result)
	}
	body := fmt.Sprintf("%v", result.StructuredContent)
	if !contains(body, pOut.ID) {
		t.Fatalf("expected parent_card_id in result, got %s", body)
	}
	if !contains(body, cOut.ID) {
		t.Fatalf("expected source_conversation_id in result, got %s", body)
	}
}

func TestE2E_ChannelFlow(t *testing.T) {
	// Since v0.13 an event reaches a channel only if it is addressed to that
	// channel's session, so the flow under test needs a session id it can
	// name. Pinning the env var is what gives the test a stable one.
	const sessionID = "sess-e2e-channel-flow"
	t.Setenv("CLAUDE_CODE_SESSION_ID", sessionID)

	bePort, stopBE := bootBackend(t)
	defer stopBE()

	beURL := fmt.Sprintf("http://127.0.0.1:%d", bePort)

	clientSide, serverSide := net.Pipe()
	t.Cleanup(func() { clientSide.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- channel.RunChannel(ctx, channel.ChannelOptions{
			BackendURL: beURL,
			In:         serverSide,
			Out:        serverSide,
		})
	}()
	t.Cleanup(func() {
		cancel()
		serverSide.Close()
		<-runDone
	})

	send := func(payload map[string]any) {
		b, _ := json.Marshal(payload)
		clientSide.Write(b)
		clientSide.Write([]byte{'\n'})
	}
	scanner := bufio.NewScanner(clientSide)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	readJSON := func(timeout time.Duration) (map[string]any, error) {
		ch := make(chan map[string]any, 1)
		go func() {
			if scanner.Scan() {
				var m map[string]any
				_ = json.Unmarshal(scanner.Bytes(), &m)
				ch <- m
			}
			close(ch)
		}()
		select {
		case m, ok := <-ch:
			if !ok {
				return nil, fmt.Errorf("EOF")
			}
			return m, nil
		case <-time.After(timeout):
			return nil, fmt.Errorf("timeout")
		}
	}

	send(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "zen-it", "version": "0"},
		},
	})
	if _, err := readJSON(3 * time.Second); err != nil {
		t.Fatalf("initialize response: %v", err)
	}
	send(map[string]any{
		"jsonrpc": "2.0", "method": "notifications/initialized",
	})

	beAPI := beURL + "/api/v1"
	convResp, err := http.Post(beAPI+"/conversations", "application/json",
		strings.NewReader(`{"title":""}`))
	if err != nil {
		t.Fatalf("create conv: %v", err)
	}
	var convOut struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(convResp.Body).Decode(&convOut)
	convResp.Body.Close()

	// The channel registers over its own WS after the MCP handshake, and the
	// backend answers 409 until it has. Retry rather than sleep: the append is
	// idempotent only in the sense that a rejected one stores nothing.
	appendBody := fmt.Sprintf(`{"role":"user","content":"hi via channel","target_session_id":%q}`, sessionID)
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Post(beAPI+"/conversations/"+convOut.ID+"/messages",
			"application/json", strings.NewReader(appendBody))
		if err != nil {
			t.Fatalf("append: %v", err)
		}
		status := resp.StatusCode
		resp.Body.Close()
		if status == http.StatusCreated || status == http.StatusOK {
			break
		}
		if status != http.StatusConflict {
			t.Fatalf("append status %d", status)
		}
		if time.Now().After(deadline) {
			t.Fatalf("channel never registered session %q", sessionID)
		}
		time.Sleep(50 * time.Millisecond)
	}

	noteDeadline := time.Now().Add(5 * time.Second)
	var note map[string]any
	for time.Now().Before(noteDeadline) {
		m, err := readJSON(time.Until(noteDeadline))
		if err != nil {
			t.Fatalf("read notification: %v", err)
		}
		if m["method"] == "notifications/claude/channel" {
			note = m
			break
		}
	}
	if note == nil {
		t.Fatal("no channel notification received within 5s")
	}
	params, _ := note["params"].(map[string]any)
	if content, _ := params["content"].(string); !contains(content, "hi via channel") {
		t.Fatalf("notification content missing message: %v", content)
	}

	send(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{
			"name": "reply",
			"arguments": map[string]any{
				"conversation_id": convOut.ID,
				"content":         "got it",
			},
		},
	})
	if _, err := readJSON(3 * time.Second); err != nil {
		t.Fatalf("reply response: %v", err)
	}

	listResp, err := http.Get(beAPI + "/conversations/" + convOut.ID + "/messages")
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	var listOut struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&listOut)
	listResp.Body.Close()

	var sawAssistant bool
	for _, m := range listOut.Messages {
		if m.Role == "assistant" && m.Content == "got it" {
			sawAssistant = true
			break
		}
	}
	if !sawAssistant {
		t.Fatalf("assistant reply not persisted: got %+v", listOut.Messages)
	}
}

func TestMCP_CardCreate_AcceptsFormatHtml(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	gResp, _ := http.Post(beURL+"/groups", "application/json",
		strings.NewReader(`{"name":"fmt"}`))
	var gOut struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&gOut)
	gResp.Body.Close()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-fmt-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "card.create",
		Arguments: map[string]any{
			"title":    "viz",
			"content":  "<svg></svg>",
			"format":   "html",
			"group_id": gOut.ID,
		},
	})
	if err != nil {
		t.Fatalf("card.create: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError: %+v", result)
	}
	body := fmt.Sprintf("%v", result.StructuredContent)
	if !contains(body, `"format":"html"`) && !contains(body, "format:html") {
		t.Fatalf("expected format=html in result, got: %s", body)
	}
}

func TestE2E_HtmlCardSearch(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	beAPI := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)

	gResp, err := http.Post(beAPI+"/groups", "application/json",
		strings.NewReader(`{"name":"viz"}`))
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	var grp struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&grp)
	gResp.Body.Close()

	body := fmt.Sprintf(`{"title":"sales-chart","content":"<svg><text>banana</text></svg><p>apple</p>","format":"html","group_id":%q}`, grp.ID)
	cardResp, err := http.Post(beAPI+"/cards", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	if cardResp.StatusCode != http.StatusCreated {
		buf := new(strings.Builder)
		_, _ = io.Copy(buf, cardResp.Body)
		t.Fatalf("create card status %d: %s", cardResp.StatusCode, buf.String())
	}
	var card struct {
		ID     string `json:"id"`
		Format string `json:"format"`
	}
	_ = json.NewDecoder(cardResp.Body).Decode(&card)
	cardResp.Body.Close()
	if card.Format != "html" {
		t.Fatalf("Format = %q", card.Format)
	}

	// Search "banana" (SVG <text> label) — must hit the card.
	{
		sresp, err := http.Get(beAPI + "/search?q=banana")
		if err != nil {
			t.Fatalf("search banana: %v", err)
		}
		var sres struct {
			Cards []struct {
				ID string `json:"id"`
			} `json:"cards"`
		}
		_ = json.NewDecoder(sresp.Body).Decode(&sres)
		sresp.Body.Close()
		found := false
		for _, h := range sres.Cards {
			if h.ID == card.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected banana search to find card %s; got hits=%+v", card.ID, sres.Cards)
		}
	}

	// Search "svg" (tag name) — must NOT hit the card.
	{
		sresp, err := http.Get(beAPI + "/search?q=svg")
		if err != nil {
			t.Fatalf("search svg: %v", err)
		}
		var sres struct {
			Cards []struct {
				ID string `json:"id"`
			} `json:"cards"`
		}
		_ = json.NewDecoder(sresp.Body).Decode(&sres)
		sresp.Body.Close()
		for _, h := range sres.Cards {
			if h.ID == card.ID {
				t.Fatalf("tag-name search 'svg' should not match HTML card %s", card.ID)
			}
		}
	}
}

// A card attaches to a level by naming a catalog entry of its group. The old
// (level, level_name) pair was dropped in migration 016; the entry id is the
// only handle now, and group.create is what mints it.
func TestMCP_CardCreate_AcceptsLevelEntryID(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	gResp, _ := http.Post(beURL+"/groups", "application/json",
		strings.NewReader(`{"name":"lvl","level_catalog":[{"weight":0,"name":"原则"}]}`))
	var gOut struct {
		ID           string `json:"id"`
		LevelCatalog []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"level_catalog"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&gOut)
	gResp.Body.Close()
	if len(gOut.LevelCatalog) != 1 {
		t.Fatalf("want 1 seeded catalog entry, got %d", len(gOut.LevelCatalog))
	}
	entryID := gOut.LevelCatalog[0].ID

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-lvl-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "card.create",
		Arguments: map[string]any{
			"title":          "p1",
			"group_id":       gOut.ID,
			"level_entry_id": entryID,
		},
	})
	if err != nil {
		t.Fatalf("card.create: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError: %+v", result)
	}
	body := fmt.Sprintf("%v", result.StructuredContent)
	if !contains(body, entryID) {
		t.Fatalf("expected level_entry_id %s in result body, got: %s", entryID, body)
	}
}

func TestE2E_MCPCompose(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	gResp, err := http.Post(beURL+"/groups", "application/json", //nolint:noctx
		strings.NewReader(`{"name":"mcp-compose"}`))
	if err != nil {
		t.Fatalf("POST /groups: %v", err)
	}
	var g struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&g)
	gResp.Body.Close()

	mkCard := func(title string) string {
		body := fmt.Sprintf(`{"title":%q,"content":"x","group_id":%q}`, title, g.ID)
		resp, _ := http.Post(beURL+"/cards", "application/json", strings.NewReader(body)) //nolint:noctx
		var out struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		return out.ID
	}
	a, b := mkCard("A"), mkCard("B")

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-compose-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "compose",
		Arguments: map[string]any{
			"source_card_ids": []string{a, b},
			"target": map[string]any{
				"title":    "merged",
				"content":  "merged",
				"group_id": g.ID,
			},
		},
	})
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if result.IsError {
		t.Fatalf("compose IsError: %+v", result)
	}
	// Genesis names the sources by title, never by ID. (The stringified body
	// carries the source ids in other fields, so assert on the genesis text
	// itself rather than on the absence of an id anywhere.)
	body := fmt.Sprintf("%v", result.StructuredContent)
	if !contains(body, "genesis:Composed from A, B") {
		t.Fatalf("expected title-based genesis, got %s", body)
	}
}

func TestE2E_MCPReferenceCreateAndList(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	mkJSON := func(path, body string) string {
		resp, _ := http.Post(beURL+path, "application/json", strings.NewReader(body)) //nolint:noctx
		var out struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		return out.ID
	}
	gID := mkJSON("/groups", `{"name":"mcp-refs"}`)
	srcID := mkJSON("/cards", fmt.Sprintf(`{"title":"src","content":"x","group_id":%q}`, gID))
	derID := mkJSON("/cards", fmt.Sprintf(`{"title":"der","content":"y","group_id":%q}`, gID))
	convID := mkJSON("/conversations", `{"title":""}`)

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-refs-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	createResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "reference.create",
		Arguments: map[string]any{
			"source_card_id":  srcID,
			"derived_card_id": derID,
			"conversation_id": convID,
			"selection_text":  "hello",
		},
	})
	if err != nil {
		t.Fatalf("reference.create: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("reference.create IsError: %+v", createResult)
	}
	body := fmt.Sprintf("%v", createResult.StructuredContent)
	if !contains(body, "hello") || !contains(body, srcID) {
		t.Fatalf("expected reference with hello+src in result, got %s", body)
	}

	listResult, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "reference.list",
		Arguments: map[string]any{"source_card_id": srcID},
	})
	if err != nil {
		t.Fatalf("reference.list: %v", err)
	}
	if listResult.IsError {
		t.Fatalf("reference.list IsError: %+v", listResult)
	}
	body2 := fmt.Sprintf("%v", listResult.StructuredContent)
	if !contains(body2, "hello") || !contains(body2, "references") {
		t.Fatalf("expected list with hello in result, got %s", body2)
	}
}

func TestE2E_MCPCardCreate_WithInlineReference(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	post := func(path, body string) string {
		resp, _ := http.Post(beURL+path, "application/json", strings.NewReader(body)) //nolint:noctx
		var out struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		return out.ID
	}
	gID := post("/groups", `{"name":"inline"}`)
	parentID := post("/cards", fmt.Sprintf(`{"title":"src","content":"hello","group_id":%q}`, gID))
	convID := post("/conversations", `{"title":""}`)

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-inline-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "card.create",
		Arguments: map[string]any{
			"title":                  "der",
			"content":                "about hello",
			"group_id":               gID,
			"parent_card_id":         parentID,
			"source_conversation_id": convID,
			"reference":              map[string]any{"selection_text": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("card.create: %v", err)
	}
	if result.IsError {
		t.Fatalf("card.create IsError: %+v", result)
	}

	parentResp, _ := http.Get(beURL + "/cards/" + parentID) //nolint:noctx
	body, _ := io.ReadAll(parentResp.Body)
	parentResp.Body.Close()
	if !contains(string(body), "hello") {
		t.Fatalf("expected parent card to carry reference with selection_text=hello, got %s", body)
	}
}

func TestE2E_MCPCardChildren(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	beURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort)
	mcpPort, stopMCP := bootMCP(t, beURL)
	defer stopMCP()

	gResp, _ := http.Post(beURL+"/groups", "application/json", //nolint:noctx
		strings.NewReader(`{"name":"kids"}`))
	var g struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(gResp.Body).Decode(&g)
	gResp.Body.Close()

	pResp, _ := http.Post(beURL+"/cards", "application/json", //nolint:noctx
		strings.NewReader(fmt.Sprintf(`{"title":"P","content":"body","group_id":%q}`, g.ID)))
	var parent struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(pResp.Body).Decode(&parent)
	pResp.Body.Close()

	dResp, _ := http.Post(beURL+"/cards/decompose", "application/json", //nolint:noctx
		strings.NewReader(fmt.Sprintf(`{"parent_card_id":%q,"cards":[
			{"title":"c1","content":"a"},{"title":"c2","content":"b"}
		]}`, parent.ID)))
	dResp.Body.Close()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-kids-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "card.children",
		Arguments: map[string]any{"id": parent.ID},
	})
	if err != nil {
		t.Fatalf("card.children: %v", err)
	}
	if result.IsError {
		t.Fatalf("card.children IsError: %+v", result)
	}
	body := fmt.Sprintf("%v", result.StructuredContent)
	if !contains(body, "c1") || !contains(body, "c2") {
		t.Fatalf("expected both child titles in payload, got %s", body)
	}
}

func TestMCP_Decompose_ContainerContent(t *testing.T) {
	bePort, stopBE := bootBackend(t)
	defer stopBE()
	mcpPort, stopMCP := bootMCP(t, fmt.Sprintf("http://127.0.0.1:%d/api/v1", bePort))
	defer stopMCP()

	mcpURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/mcp", mcpPort)
	transport := &mcpsdk.StreamableClientTransport{Endpoint: mcpURL}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "zen-it", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	gRes, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "group.create", Arguments: map[string]any{"name": "notes"},
	})
	if err != nil || gRes.IsError {
		t.Fatalf("group.create: %v %+v", err, gRes)
	}
	gid := gRes.StructuredContent.(map[string]any)["group"].(map[string]any)["id"].(string)

	cRes, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "card.create", Arguments: map[string]any{"title": "Doc", "content": "meta\n\nA\n\nB", "group_id": gid},
	})
	if err != nil || cRes.IsError {
		t.Fatalf("card.create: %v %+v", err, cRes)
	}
	pid := cRes.StructuredContent.(map[string]any)["card"].(map[string]any)["id"].(string)

	dRes, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "decompose",
		Arguments: map[string]any{
			"parent_card_id":    pid,
			"container_content": "Date: 2026-07-11",
			"cards": []any{
				map[string]any{"title": "A", "content": "A"},
				map[string]any{"title": "B", "content": "B"},
			},
		},
	})
	if err != nil || dRes.IsError {
		t.Fatalf("decompose: %v %+v", err, dRes)
	}

	gcRes, err := session.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "card.get", Arguments: map[string]any{"id": pid},
	})
	if err != nil || gcRes.IsError {
		t.Fatalf("card.get: %v %+v", err, gcRes)
	}
	content := gcRes.StructuredContent.(map[string]any)["card"].(map[string]any)["content"].(string)
	if content != "Date: 2026-07-11" {
		t.Fatalf("expected parent to keep container_content, got %q", content)
	}
}
