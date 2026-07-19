package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func newConvForMsgTest(t *testing.T, r repository.Repository) string {
	t.Helper()
	ctx := context.Background()
	c := &entity.Conversation{
		ID:            ulidutil.New(),
		Title:         "x",
		CreatedAt:     time.Now(),
		LastMessageAt: time.Now(),
	}
	if err := r.CreateConversation(ctx, c); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	return c.ID
}

func TestCreateMessage_BasicRoundtrip(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)

	sel := "selected text"
	msg := &entity.Message{
		ID:             ulidutil.New(),
		ConversationID: convID,
		Role:           "user",
		Content:        "hello",
		SelectionText:  &sel,
		CreatedAt:      time.Now(),
	}
	if err := r.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	got, err := r.GetMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got.Content != "hello" || got.SelectionText == nil || *got.SelectionText != "selected text" {
		t.Fatalf("bad roundtrip: %+v", got)
	}
}

func TestCreateMessage_PersistsSession(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)

	sid, cwd := "sess-A", "/home/x/zen"
	msg := &entity.Message{
		ID:             ulidutil.New(),
		ConversationID: convID,
		Role:           "user",
		Content:        "hi",
		SessionID:      &sid,
		SessionCwd:     &cwd,
		CreatedAt:      time.Now(),
	}
	if err := r.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	got, err := r.GetMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got.SessionID == nil || *got.SessionID != sid {
		t.Fatalf("session_id = %v, want %q", got.SessionID, sid)
	}
	if got.SessionCwd == nil || *got.SessionCwd != cwd {
		t.Fatalf("session_cwd = %v, want %q", got.SessionCwd, cwd)
	}
}

func TestSetMessageSession_SetOnce(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)

	msg := &entity.Message{
		ID: ulidutil.New(), ConversationID: convID, Role: "user",
		Content: "hi", CreatedAt: time.Now(),
	}
	if err := r.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	if err := r.SetMessageSession(ctx, msg.ID, "sess-A", "/repo-a"); err != nil {
		t.Fatalf("first SetMessageSession: %v", err)
	}
	got, _ := r.GetMessage(ctx, msg.ID)
	if got.SessionID == nil || *got.SessionID != "sess-A" {
		t.Fatalf("after first set: %v", got.SessionID)
	}

	// A message that already has a session is never re-pointed.
	if err := r.SetMessageSession(ctx, msg.ID, "sess-B", "/repo-b"); err != nil {
		t.Fatalf("second SetMessageSession: %v", err)
	}
	got, _ = r.GetMessage(ctx, msg.ID)
	if *got.SessionID != "sess-A" {
		t.Fatalf("re-pointed to %q, want sess-A", *got.SessionID)
	}
}

func TestCreateMessage_RejectsBadRole(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)
	msg := &entity.Message{
		ID:             ulidutil.New(),
		ConversationID: convID,
		Role:           "bogus",
		Content:        "x",
		CreatedAt:      time.Now(),
	}
	if err := r.CreateMessage(ctx, msg); err == nil {
		t.Fatalf("expected error on bad role, got nil")
	}
}

func TestCreateMessage_RejectsSelectionOnAssistant(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)
	sel := "should not be allowed"
	msg := &entity.Message{
		ID:             ulidutil.New(),
		ConversationID: convID,
		Role:           "assistant",
		Content:        "x",
		SelectionText:  &sel,
		CreatedAt:      time.Now(),
	}
	if err := r.CreateMessage(ctx, msg); err == nil {
		t.Fatalf("expected CHECK constraint error, got nil")
	}
}

func TestListMessages_OrdersByCreatedAt(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)
	base := time.Now()
	for i, txt := range []string{"first", "second", "third"} {
		_ = r.CreateMessage(ctx, &entity.Message{
			ID:             ulidutil.New(),
			ConversationID: convID,
			Role:           "user",
			Content:        txt,
			CreatedAt:      base.Add(time.Duration(i) * time.Second),
		})
	}
	got, err := r.ListMessages(ctx, convID, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(got) != 3 || got[0].Content != "first" || got[2].Content != "third" {
		t.Fatalf("bad order: %+v", got)
	}
}

func TestListMessagesAfter_ReturnsOnlyNewerMessages(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)

	base := time.Now()
	ids := make([]string, 3)
	for i := range ids {
		m := &entity.Message{
			ID: ulidutil.New(), ConversationID: convID,
			Role: "user", Content: "m", CreatedAt: base.Add(time.Duration(i) * time.Second),
		}
		if err := r.CreateMessage(ctx, m); err != nil {
			t.Fatalf("CreateMessage: %v", err)
		}
		ids[i] = m.ID
	}

	got, err := r.ListMessagesAfter(ctx, convID, ids[0], 0)
	if err != nil {
		t.Fatalf("ListMessagesAfter: %v", err)
	}
	if len(got) != 2 || got[0].ID != ids[1] || got[1].ID != ids[2] {
		t.Fatalf("got %d messages, want the two after the cursor", len(got))
	}
}

// An empty cursor means "from the beginning".
func TestListMessagesAfter_EmptyCursorReturnsEverything(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)

	base := time.Now()
	for i := 0; i < 2; i++ {
		_ = r.CreateMessage(ctx, &entity.Message{
			ID: ulidutil.New(), ConversationID: convID, Role: "user", Content: "m",
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		})
	}

	got, err := r.ListMessagesAfter(ctx, convID, "", 0)
	if err != nil {
		t.Fatalf("ListMessagesAfter: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
}

// A cursor at the newest message yields nothing — the caller is caught up.
func TestListMessagesAfter_CursorAtHeadReturnsNothing(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convID := newConvForMsgTest(t, r)

	base := time.Now()
	last := ""
	for i := 0; i < 2; i++ {
		m := &entity.Message{
			ID: ulidutil.New(), ConversationID: convID, Role: "user", Content: "m",
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		}
		_ = r.CreateMessage(ctx, m)
		last = m.ID
	}

	got, err := r.ListMessagesAfter(ctx, convID, last, 0)
	if err != nil {
		t.Fatalf("ListMessagesAfter: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d, want 0", len(got))
	}
}

// The cursor is scoped to one conversation.
func TestListMessagesAfter_IsScopedToTheConversation(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	convA := newConvForMsgTest(t, r)
	convB := newConvForMsgTest(t, r)

	inA := &entity.Message{ID: ulidutil.New(), ConversationID: convA, Role: "user", Content: "a", CreatedAt: time.Now()}
	_ = r.CreateMessage(ctx, inA)
	inB := &entity.Message{ID: ulidutil.New(), ConversationID: convB, Role: "user", Content: "b", CreatedAt: time.Now()}
	_ = r.CreateMessage(ctx, inB)

	got, err := r.ListMessagesAfter(ctx, convB, inA.ID, 0)
	if err != nil {
		t.Fatalf("ListMessagesAfter: %v", err)
	}
	if len(got) != 1 || got[0].ID != inB.ID {
		t.Fatalf("got %+v, want only b's message", got)
	}
}
