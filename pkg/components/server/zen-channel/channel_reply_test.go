package zenchannel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
)

func TestReply_PostsAssistantMessage(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"01MSG","conversation_id":"01CONV","role":"assistant","content":"hi","created_at":"2026-06-27T00:00:00Z"}`))
	}))
	defer srv.Close()

	be := zenbackend.New(srv.URL)
	reply := NewReply(be, "sess-A", "/repo")
	out, err := reply(context.Background(), "01CONV", "hi")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if gotPath != "/conversations/01CONV/messages" {
		t.Fatalf("path: %s", gotPath)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(gotBody), &body)
	if body["role"] != "assistant" || body["content"] != "hi" {
		t.Fatalf("body: %s", gotBody)
	}
	if body["session_id"] != "sess-A" || body["session_cwd"] != "/repo" {
		t.Fatalf("session not sent on reply: %s", gotBody)
	}
	if !strings.Contains(out, "01MSG") {
		t.Fatalf("result missing message id: %s", out)
	}
}

func TestReply_ErrorsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"kind":"NotFound","message":"gone"}`))
	}))
	defer srv.Close()

	be := zenbackend.New(srv.URL)
	reply := NewReply(be, "sess-A", "/repo")
	if _, err := reply(context.Background(), "01CONV", "hi"); err == nil {
		t.Fatal("expected error for 404")
	}
}
