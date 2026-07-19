package zenbackend

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

func TestConversationClient_Create_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodPost, "/conversations", func(w http.ResponseWriter, r *http.Request) {
		var req api.CreateConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Title != "hi" {
			t.Fatalf("title not propagated: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(entity.Conversation{
			ID:            "01ABC",
			Title:         "hi",
			CreatedAt:     time.Now(),
			LastMessageAt: time.Now(),
		})
	})

	got, err := srv.client().Conversation().Create(t.Context(),
		api.CreateConversationRequest{Title: "hi"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "01ABC" || got.Title != "hi" {
		t.Fatalf("bad result: %+v", got)
	}
}

func TestConversationClient_AppendMessage_HappyPath(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodPost, "/conversations/01CONV/messages", func(w http.ResponseWriter, r *http.Request) {
		var req api.AppendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Role != "user" || req.Content != "hello" {
			t.Fatalf("bad req: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(entity.Message{
			ID:             "01MSG",
			ConversationID: "01CONV",
			Role:           "user",
			Content:        "hello",
			CreatedAt:      time.Now(),
		})
	})

	msg, err := srv.client().Conversation().AppendMessage(t.Context(), "01CONV",
		api.AppendMessageRequest{Role: "user", Content: "hello"})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if msg.ID != "01MSG" || msg.Content != "hello" {
		t.Fatalf("bad msg: %+v", msg)
	}
}

func TestConversationClient_List_EncodesFilters(t *testing.T) {
	srv := newStubBackend(t)
	srv.on(http.MethodGet, "/conversations", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("anchor_kind") != "card" {
			t.Fatalf("anchor_kind not propagated: %q", q.Get("anchor_kind"))
		}
		if q.Get("anchor_id") != "01CARD" {
			t.Fatalf("anchor_id not propagated: %q", q.Get("anchor_id"))
		}
		if q.Get("pending") != "true" {
			t.Fatalf("pending not propagated: %q", q.Get("pending"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"conversations":[],"unanswered_counts":[]}`))
	})

	kind := "card"
	id := "01CARD"
	got, err := srv.client().Conversation().List(t.Context(), &kind, &id, true, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Fatalf("nil response")
	}
}
