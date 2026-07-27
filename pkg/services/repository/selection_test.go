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

func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

// newSelCtx builds a group + card and returns the repo and the card id.
func newSelCtx(t *testing.T) (repository.Repository, string) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	g := &entity.Group{ID: ulidutil.New(), Name: "g", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	c := &entity.Card{
		ID: ulidutil.New(), Title: "card", Content: "hello brave world", Format: "markdown",
		GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	return repo, c.ID
}

// mkConv creates a conversation anchored to (kind, id).
func mkConv(t *testing.T, repo repository.Repository, kind, id string) string {
	t.Helper()
	c := &entity.Conversation{
		ID: ulidutil.New(), Title: "conv",
		AnchorKind: ptrStr(kind), AnchorID: ptrStr(id),
		CreatedAt: time.Now(), LastMessageAt: time.Now(),
	}
	if err := repo.CreateConversation(context.Background(), c); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	return c.ID
}

// mkMsg appends a message, optionally carrying a selection range.
func mkMsg(t *testing.T, repo repository.Repository, convID, text string, start, end *int) string {
	t.Helper()
	m := &entity.Message{
		ID: ulidutil.New(), ConversationID: convID, Role: "user", Content: "q",
		SelectionStart: start, SelectionEnd: end, CreatedAt: time.Now(),
	}
	if text != "" {
		m.SelectionText = &text
	}
	if err := repo.CreateMessage(context.Background(), m); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	return m.ID
}

func TestListCardSelections_ReturnsRangedMessagesAnchoredToCard(t *testing.T) {
	repo, cardID := newSelCtx(t)
	ctx := context.Background()
	conv := mkConv(t, repo, "card", cardID)
	want := mkMsg(t, repo, conv, "brave", ptrInt(6), ptrInt(11))

	got, err := repo.ListCardSelections(ctx, cardID)
	if err != nil {
		t.Fatalf("ListCardSelections: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 selection, got %d", len(got))
	}
	if got[0].MessageID != want {
		t.Errorf("message id = %q, want %q", got[0].MessageID, want)
	}
	if got[0].SelectionStart != 6 || got[0].SelectionEnd != 11 {
		t.Errorf("range = [%d,%d), want [6,11)", got[0].SelectionStart, got[0].SelectionEnd)
	}
	if got[0].SelectionText != "brave" {
		t.Errorf("text = %q, want %q", got[0].SelectionText, "brave")
	}
	if got[0].ConversationID != conv {
		t.Errorf("conversation id = %q, want %q", got[0].ConversationID, conv)
	}
}

// Decision 3: no offsets, no underline — those rows must not even leave the DB.
func TestListCardSelections_ExcludesMessagesWithoutRange(t *testing.T) {
	repo, cardID := newSelCtx(t)
	conv := mkConv(t, repo, "card", cardID)
	mkMsg(t, repo, conv, "brave", nil, nil)

	got, err := repo.ListCardSelections(context.Background(), cardID)
	if err != nil {
		t.Fatalf("ListCardSelections: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 selections, got %d", len(got))
	}
}

func TestListCardSelections_ExcludesOtherCards(t *testing.T) {
	repo, cardID := newSelCtx(t)
	other := mkConv(t, repo, "card", ulidutil.New())
	mkMsg(t, repo, other, "brave", ptrInt(0), ptrInt(5))

	got, err := repo.ListCardSelections(context.Background(), cardID)
	if err != nil {
		t.Fatalf("ListCardSelections: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("selections leaked from another card: %d", len(got))
	}
}

func TestListCardSelections_EmptyForCardWithNoConversations(t *testing.T) {
	repo, cardID := newSelCtx(t)
	got, err := repo.ListCardSelections(context.Background(), cardID)
	if err != nil {
		t.Fatalf("want no error for a card with no conversations, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 selections, got %d", len(got))
	}
}

// Conversations have no soft-delete column and production runs without the
// foreign_keys pragma, so a deleted conversation leaves orphaned messages.
// The join is what excludes them.
func TestListCardSelections_ExcludesOrphansOfDeletedConversations(t *testing.T) {
	repo, cardID := newSelCtx(t)
	ctx := context.Background()
	conv := mkConv(t, repo, "card", cardID)
	mkMsg(t, repo, conv, "brave", ptrInt(6), ptrInt(11))

	if err := repo.DeleteConversation(ctx, conv); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}
	got, err := repo.ListCardSelections(ctx, cardID)
	if err != nil {
		t.Fatalf("ListCardSelections: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("orphaned messages leaked: %d", len(got))
	}
}

// Paint order must be deterministic.
func TestListCardSelections_OrderedOldestFirst(t *testing.T) {
	repo, cardID := newSelCtx(t)
	conv := mkConv(t, repo, "card", cardID)
	first := mkMsg(t, repo, conv, "a", ptrInt(0), ptrInt(1))
	second := mkMsg(t, repo, conv, "b", ptrInt(2), ptrInt(3))

	got, err := repo.ListCardSelections(context.Background(), cardID)
	if err != nil {
		t.Fatalf("ListCardSelections: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[0].MessageID != first || got[1].MessageID != second {
		t.Errorf("out of order: %q then %q", got[0].MessageID, got[1].MessageID)
	}
}
