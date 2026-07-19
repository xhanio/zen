package conversation_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/services/messagebus"
	"github.com/xhanio/framingo/pkg/services/pubsub"
	framodel "github.com/xhanio/framingo/pkg/types/model"
	"github.com/xhanio/framingo/pkg/utils/log"

	"github.com/xhanio/zen/pkg/services/conversation"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/busutil"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func newConvCtx(t *testing.T) (svc conversation.Manager, repo repository.Repository, groupID string) {
	t.Helper()
	repo = repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	svc = conversation.New(repo)
	return svc, repo, g.ID
}

func TestAppendMessage_PersistsSession(t *testing.T) {
	svc, repo, _ := newConvCtx(t)
	ctx := context.Background()
	c, err := svc.Create(ctx, "x", nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	msg, err := svc.AppendMessage(ctx, c.ID, "user", "hi", nil,
		model.WithSession("sess-A", "/home/x/zen"))
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if msg.SessionID == nil || *msg.SessionID != "sess-A" {
		t.Fatalf("returned session_id = %v", msg.SessionID)
	}
	if msg.SessionCwd == nil || *msg.SessionCwd != "/home/x/zen" {
		t.Fatalf("returned session_cwd = %v", msg.SessionCwd)
	}
	got, err := repo.GetMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got.SessionID == nil || *got.SessionID != "sess-A" {
		t.Fatalf("persisted session_id = %v", got.SessionID)
	}
}

func TestConv_Create_Standalone(t *testing.T) {
	svc, _, _ := newConvCtx(t)
	c, err := svc.Create(context.Background(), "hello", nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.Title != "hello" || c.AnchorKind != nil || c.AnchorID != nil {
		t.Fatalf("bad standalone: %+v", c)
	}
}

func TestConv_Create_HalfAnchorRejected(t *testing.T) {
	svc, _, _ := newConvCtx(t)
	kind := "card"
	_, err := svc.Create(context.Background(), "x", &kind, nil)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}

func TestConv_Create_BadAnchorKind(t *testing.T) {
	svc, _, _ := newConvCtx(t)
	kind := "tag"
	id := ulidutil.New()
	_, err := svc.Create(context.Background(), "x", &kind, &id)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}

func TestConv_Create_AnchorMustExist(t *testing.T) {
	svc, _, _ := newConvCtx(t)
	kind := "card"
	id := ulidutil.New()
	_, err := svc.Create(context.Background(), "x", &kind, &id)
	if !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestConv_AppendMessage_SetsTitleFromFirstUserMsg(t *testing.T) {
	svc, _, _ := newConvCtx(t)
	c, _ := svc.Create(context.Background(), "", nil, nil)
	msg, err := svc.AppendMessage(context.Background(), c.ID, "user", "What is the meaning of X?", nil)
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if msg.Content != "What is the meaning of X?" {
		t.Fatalf("bad content roundtrip")
	}
	got, _ := svc.Get(context.Background(), c.ID)
	if got.Title != "What is the meaning of X?" {
		t.Fatalf("title should auto-set from first user msg, got %q", got.Title)
	}
}

func TestConv_AppendMessage_SelectionOnAssistantRejected(t *testing.T) {
	svc, _, _ := newConvCtx(t)
	c, _ := svc.Create(context.Background(), "x", nil, nil)
	sel := "nope"
	_, err := svc.AppendMessage(context.Background(), c.ID, "assistant", "reply", &sel)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}

func TestConv_List_PendingOnlyReturnsUnanswered(t *testing.T) {
	svc, _, _ := newConvCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "a", nil, nil)
	b, _ := svc.Create(ctx, "b", nil, nil)
	_, _ = svc.AppendMessage(ctx, a.ID, "user", "q1", nil)
	_, _ = svc.AppendMessage(ctx, b.ID, "user", "q1", nil)
	_, _ = svc.AppendMessage(ctx, b.ID, "assistant", "a1", nil)
	cs, counts, err := svc.List(ctx, nil, nil, true, 0)
	if err != nil {
		t.Fatalf("List pending: %v", err)
	}
	if len(cs) != 1 || cs[0].ID != a.ID {
		t.Fatalf("expected only conversation a pending, got %d (%v)", len(cs), cs)
	}
	if len(counts) != 1 || counts[0] != 1 {
		t.Fatalf("expected unanswered count 1, got %v", counts)
	}
}

func TestConv_DeleteByAnchor_CascadesAllConversationsOnAnchor(t *testing.T) {
	svc, _, groupID := newConvCtx(t)
	ctx := context.Background()
	kind := "group"
	for i := 0; i < 2; i++ {
		_, err := svc.Create(ctx, "x", &kind, &groupID)
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	_, _ = svc.Create(ctx, "standalone", nil, nil)

	if err := svc.DeleteByAnchor(ctx, "group", groupID); err != nil {
		t.Fatalf("DeleteByAnchor: %v", err)
	}
	all, _, _ := svc.List(ctx, nil, nil, false, 0)
	if len(all) != 1 {
		t.Fatalf("expected 1 standalone left, got %d", len(all))
	}
}

func TestConv_AppendMessage_PublishesEvent(t *testing.T) {
	repo := repository.New(testutil.NewDB(t))

	bus := messagebus.New(pubsub.New(busutil.NewDriver(log.Default)))
	if err := bus.Start(context.Background()); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(func() { _ = bus.Stop(true) })

	subscriber, err := bus.NewMessenger("test-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	t.Cleanup(subscriber.Close)

	svc := conversation.New(repo, conversation.WithMessageBus(bus))
	c, err := svc.Create(context.Background(), "x", nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "ping", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	select {
	case msg := <-subscriber.Ch():
		if msg.Kind != entity.ConversationEventKind {
			t.Fatalf("expected kind=%q, got %q", entity.ConversationEventKind, msg.Kind)
		}
		ev, ok := msg.Payload.(*entity.ConversationEvent)
		if !ok {
			t.Fatalf("payload type = %T, want *entity.ConversationEvent", msg.Payload)
		}
		if ev.ConversationID != c.ID || ev.Role != "user" || ev.Content != "ping" {
			t.Fatalf("bad event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("no event received within 2s")
	}
}

func TestConv_AppendMessage_WithoutBus_SilentNoPublish(t *testing.T) {
	repo := repository.New(testutil.NewDB(t))
	svc := conversation.New(repo)
	c, _ := svc.Create(context.Background(), "x", nil, nil)
	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "ping", nil); err != nil {
		t.Fatalf("AppendMessage with nil bus: %v", err)
	}
}

// newManagerWithBus builds a conversation service wired to a live in-memory
// messagebus, so tests can observe the events it publishes.
func newManagerWithBus(t *testing.T) (conversation.Manager, framodel.MessageBus) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))

	bus := messagebus.New(pubsub.New(busutil.NewDriver(log.Default)))
	if err := bus.Start(context.Background()); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(func() { _ = bus.Stop(true) })

	return conversation.New(repo, conversation.WithMessageBus(bus)), bus
}

// The target rides on the published event so the channel can filter on it.
func TestAppendMessage_StampsTargetOnEvent(t *testing.T) {
	m, bus := newManagerWithBus(t)
	sub, err := bus.NewMessenger("test-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	defer sub.Close()

	c, err := m.Create(context.Background(), "x", nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := m.AppendMessage(context.Background(), c.ID, "user", "hi", nil,
		model.WithTargetSession("sess-A")); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	select {
	case msg := <-sub.Ch():
		ev, ok := msg.Payload.(*entity.ConversationEvent)
		if !ok {
			t.Fatalf("payload type %T", msg.Payload)
		}
		if ev.TargetSessionID != "sess-A" {
			t.Fatalf("TargetSessionID = %q, want sess-A", ev.TargetSessionID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
}

// Without the option the event carries no target: nobody is addressed.
func TestAppendMessage_NoTargetByDefault(t *testing.T) {
	m, bus := newManagerWithBus(t)
	sub, err := bus.NewMessenger("test-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	defer sub.Close()

	c, _ := m.Create(context.Background(), "x", nil, nil)
	if _, err := m.AppendMessage(context.Background(), c.ID, "user", "hi", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	select {
	case msg := <-sub.Ch():
		ev := msg.Payload.(*entity.ConversationEvent)
		if ev.TargetSessionID != "" {
			t.Fatalf("TargetSessionID = %q, want empty", ev.TargetSessionID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
}

// The messages table must never learn the target existed.
func TestAppendMessage_TargetIsNotPersisted(t *testing.T) {
	m, _ := newManagerWithBus(t)
	c, _ := m.Create(context.Background(), "x", nil, nil)

	msg, err := m.AppendMessage(context.Background(), c.ID, "user", "hi", nil,
		model.WithTargetSession("sess-A"))
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	got, err := m.GetMessages(context.Background(), c.ID, 10)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(got) != 1 || got[0].ID != msg.ID {
		t.Fatalf("messages = %+v", got)
	}
	if got[0].Content != "hi" {
		t.Fatalf("content = %q", got[0].Content)
	}
}

func TestDispatchMessage_RepublishesStoredMessage(t *testing.T) {
	m, bus := newManagerWithBus(t)
	c, _ := m.Create(context.Background(), "x", nil, nil)
	msg, err := m.AppendMessage(context.Background(), c.ID, "user", "queued", nil)
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	sub, err := bus.NewMessenger("dispatch-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	defer sub.Close()

	if err := m.DispatchMessage(context.Background(), c.ID, msg.ID, "sess-A", "/repo"); err != nil {
		t.Fatalf("DispatchMessage: %v", err)
	}

	select {
	case bm := <-sub.Ch():
		ev := bm.Payload.(*entity.ConversationEvent)
		if ev.MessageID != msg.ID || ev.TargetSessionID != "sess-A" || ev.Content != "queued" {
			t.Fatalf("event = %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
}

// Dispatch republishes; it never inserts.
func TestDispatchMessage_DoesNotDuplicateTheRow(t *testing.T) {
	m, _ := newManagerWithBus(t)
	c, _ := m.Create(context.Background(), "x", nil, nil)
	msg, _ := m.AppendMessage(context.Background(), c.ID, "user", "queued", nil)

	if err := m.DispatchMessage(context.Background(), c.ID, msg.ID, "sess-A", "/repo"); err != nil {
		t.Fatalf("DispatchMessage: %v", err)
	}

	got, _ := m.GetMessages(context.Background(), c.ID, 10)
	if len(got) != 1 {
		t.Fatalf("messages = %d, want 1", len(got))
	}
}

func TestDispatchMessage_RejectsUnknownMessage(t *testing.T) {
	m, _ := newManagerWithBus(t)
	c, _ := m.Create(context.Background(), "x", nil, nil)

	err := m.DispatchMessage(context.Background(), c.ID, ulidutil.New(), "sess-A", "/repo")
	if err == nil {
		t.Fatal("expected an error for an unknown message id")
	}
}

// Only a user message is dispatched to a session; an assistant reply is not.
func TestDispatchMessage_RejectsNonUserMessage(t *testing.T) {
	m, _ := newManagerWithBus(t)
	c, _ := m.Create(context.Background(), "x", nil, nil)
	msg, _ := m.AppendMessage(context.Background(), c.ID, "assistant", "answer", nil)

	if err := m.DispatchMessage(context.Background(), c.ID, msg.ID, "sess-A", "/repo"); err == nil {
		t.Fatal("expected an error dispatching an assistant message")
	}
}

func TestDispatchMessage_RejectsMessageFromAnotherConversation(t *testing.T) {
	m, _ := newManagerWithBus(t)
	a, _ := m.Create(context.Background(), "a", nil, nil)
	b, _ := m.Create(context.Background(), "b", nil, nil)
	msg, _ := m.AppendMessage(context.Background(), a.ID, "user", "hi", nil)

	if err := m.DispatchMessage(context.Background(), b.ID, msg.ID, "sess-A", "/repo"); err == nil {
		t.Fatal("expected an error dispatching across conversations")
	}
}

// The first message in a conversation has no predecessor. LatestMessage returns
// NotFound there, and that must not fail the append.
func TestAppendMessage_FirstMessageHasEmptyPrev(t *testing.T) {
	m, bus := newManagerWithBus(t)
	sub, err := bus.NewMessenger("prev-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	defer sub.Close()

	c, _ := m.Create(context.Background(), "x", nil, nil)
	if _, err := m.AppendMessage(context.Background(), c.ID, "user", "first", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	select {
	case bm := <-sub.Ch():
		ev := bm.Payload.(*entity.ConversationEvent)
		if ev.PrevMessageID != "" {
			t.Fatalf("PrevMessageID = %q, want empty for the first message", ev.PrevMessageID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
}

// Each event names the message before it, forming a chain the consumer can
// check its cursor against.
func TestAppendMessage_PrevPointsAtThePreviousMessage(t *testing.T) {
	m, bus := newManagerWithBus(t)
	c, _ := m.Create(context.Background(), "x", nil, nil)

	first, err := m.AppendMessage(context.Background(), c.ID, "user", "one", nil)
	if err != nil {
		t.Fatalf("AppendMessage 1: %v", err)
	}

	sub, err := bus.NewMessenger("prev-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	defer sub.Close()

	second, err := m.AppendMessage(context.Background(), c.ID, "assistant", "two", nil)
	if err != nil {
		t.Fatalf("AppendMessage 2: %v", err)
	}

	select {
	case bm := <-sub.Ch():
		ev := bm.Payload.(*entity.ConversationEvent)
		if ev.MessageID != second.ID {
			t.Fatalf("MessageID = %q, want %q", ev.MessageID, second.ID)
		}
		if ev.PrevMessageID != first.ID {
			t.Fatalf("PrevMessageID = %q, want %q", ev.PrevMessageID, first.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
}

// prev is scoped to one conversation: a message in another thread is not a
// predecessor.
func TestAppendMessage_PrevIsScopedToTheConversation(t *testing.T) {
	m, bus := newManagerWithBus(t)
	a, _ := m.Create(context.Background(), "a", nil, nil)
	b, _ := m.Create(context.Background(), "b", nil, nil)

	if _, err := m.AppendMessage(context.Background(), a.ID, "user", "in a", nil); err != nil {
		t.Fatalf("append a: %v", err)
	}

	sub, err := bus.NewMessenger("prev-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	defer sub.Close()

	if _, err := m.AppendMessage(context.Background(), b.ID, "user", "in b", nil); err != nil {
		t.Fatalf("append b: %v", err)
	}

	select {
	case bm := <-sub.Ch():
		ev := bm.Payload.(*entity.ConversationEvent)
		if ev.PrevMessageID != "" {
			t.Fatalf("PrevMessageID = %q; a's message is not b's predecessor", ev.PrevMessageID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
}

// Dispatch re-publishes an old message. The browser already has it, so a prev
// from the past would look like a gap. It claims none.
func TestDispatchMessage_LeavesPrevEmpty(t *testing.T) {
	m, bus := newManagerWithBus(t)
	c, _ := m.Create(context.Background(), "x", nil, nil)
	_, _ = m.AppendMessage(context.Background(), c.ID, "user", "one", nil)
	target, _ := m.AppendMessage(context.Background(), c.ID, "user", "two", nil)

	sub, err := bus.NewMessenger("dispatch-sub")
	if err != nil {
		t.Fatalf("NewMessenger: %v", err)
	}
	defer sub.Close()

	if err := m.DispatchMessage(context.Background(), c.ID, target.ID, "sess-A", "/repo"); err != nil {
		t.Fatalf("DispatchMessage: %v", err)
	}

	select {
	case bm := <-sub.Ch():
		ev := bm.Payload.(*entity.ConversationEvent)
		if ev.PrevMessageID != "" {
			t.Fatalf("dispatch claimed prev %q; it must claim none", ev.PrevMessageID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event published")
	}
}
