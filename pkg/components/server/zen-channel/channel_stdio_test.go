package zenchannel

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestStdioServer_ReadyClosesAfterInitialize(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()

	s := &StdioServer{In: serverSide, Out: serverSide}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	t.Cleanup(func() {
		serverSide.Close()
		<-done
	})

	select {
	case <-s.Ready():
		t.Fatal("Ready() closed before initialize")
	case <-time.After(100 * time.Millisecond):
	}

	enc := json.NewEncoder(clientSide)
	scan := bufio.NewScanner(clientSide)
	scan.Buffer(make([]byte, 1<<20), 1<<20)
	_ = enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{"name": "claude-code", "version": "2.1.205"},
		},
	})
	if !scan.Scan() {
		t.Fatal("no response to initialize")
	}

	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("Ready() did not close after initialize")
	}

	name, version := s.ClientInfo()
	if name != "claude-code" || version != "2.1.205" {
		t.Fatalf("ClientInfo() = (%q, %q)", name, version)
	}
}

func TestStdioServer_ReadyIsIdempotent(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()

	s := &StdioServer{In: serverSide, Out: serverSide}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	t.Cleanup(func() {
		serverSide.Close()
		<-done
	})

	enc := json.NewEncoder(clientSide)
	scan := bufio.NewScanner(clientSide)
	scan.Buffer(make([]byte, 1<<20), 1<<20)

	// A second initialize must not panic on a double close.
	for i := 0; i < 2; i++ {
		_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": i, "method": "initialize"})
		if !scan.Scan() {
			t.Fatalf("no response to initialize #%d", i)
		}
	}

	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("Ready() did not close")
	}
}

func TestStdioServer_InitToolsListAndCall(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()

	s := &StdioServer{
		In:  serverSide,
		Out: serverSide,
		Reply: func(_ context.Context, convID, content string) (string, error) {
			if convID != "01CONV" || content != "hi" {
				t.Errorf("reply got conv=%s content=%s", convID, content)
			}
			return "sent (01MSG)", nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	t.Cleanup(func() {
		serverSide.Close()
		<-done
	})

	enc := json.NewEncoder(clientSide)
	scan := bufio.NewScanner(clientSide)
	scan.Buffer(make([]byte, 1<<20), 1<<20)

	send := func(v any) { _ = enc.Encode(v) }
	recv := func() map[string]any {
		ch := make(chan map[string]any, 1)
		go func() {
			if scan.Scan() {
				var m map[string]any
				_ = json.Unmarshal(scan.Bytes(), &m)
				ch <- m
				return
			}
			ch <- nil
		}()
		select {
		case m := <-ch:
			if m == nil {
				t.Fatal("EOF before frame")
			}
			return m
		case <-time.After(2 * time.Second):
			t.Fatal("recv timeout")
			return nil
		}
	}

	send(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "t", "version": "0"},
		},
	})
	initResp := recv()
	if initResp["id"].(float64) != 1 {
		t.Fatalf("init id: %v", initResp["id"])
	}
	result := initResp["result"].(map[string]any)
	caps := result["capabilities"].(map[string]any)
	if _, ok := caps["tools"]; !ok {
		t.Fatal("no tools cap")
	}
	exp, ok := caps["experimental"].(map[string]any)
	if !ok || exp["claude/channel"] == nil {
		t.Fatal("no experimental.claude/channel cap")
	}

	send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})

	send(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	listResp := recv()
	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "reply" {
		t.Fatalf("tools/list result: %v", tools)
	}

	send(map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]any{
			"name": "reply",
			"arguments": map[string]any{"conversation_id": "01CONV", "content": "hi"},
		},
	})
	callResp := recv()
	content := callResp["result"].(map[string]any)["content"].([]any)
	if content[0].(map[string]any)["text"] != "sent (01MSG)" {
		t.Fatalf("tools/call result: %v", content)
	}

	pushErrCh := make(chan error, 1)
	go func() {
		pushErrCh <- s.PushChannelNotification(ctx, ChannelNotification{Content: "x", Meta: map[string]string{"k": "v"}})
	}()
	note := recv()
	if err := <-pushErrCh; err != nil {
		t.Fatalf("push: %v", err)
	}
	if note["method"] != "notifications/claude/channel" {
		t.Fatalf("notification method: %v", note["method"])
	}
	if _, hasID := note["id"]; hasID {
		t.Fatalf("notification must have no id; got envelope %v", note)
	}
	params := note["params"].(map[string]any)
	if params["content"] != "x" || params["meta"].(map[string]any)["k"] != "v" {
		t.Fatalf("notification params: %v", params)
	}
}

func TestStdioServer_UnknownTool_ReturnsIsError(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()

	s := &StdioServer{In: serverSide, Out: serverSide, Reply: func(context.Context, string, string) (string, error) { return "", nil }}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	t.Cleanup(func() { serverSide.Close(); <-done })

	enc := json.NewEncoder(clientSide)
	scan := bufio.NewScanner(clientSide)

	_ = enc.Encode(map[string]any{
		"jsonrpc": "2.0", "id": 9, "method": "tools/call",
		"params": map[string]any{"name": "bogus", "arguments": map[string]any{}},
	})
	scanCh := make(chan map[string]any, 1)
	go func() {
		if scan.Scan() {
			var m map[string]any
			_ = json.Unmarshal(scan.Bytes(), &m)
			scanCh <- m
		}
	}()
	select {
	case m := <-scanCh:
		if m["result"].(map[string]any)["isError"] != true {
			t.Fatalf("expected isError; got %v", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
