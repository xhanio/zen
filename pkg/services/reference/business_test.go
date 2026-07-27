package reference_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/services/card"
	"github.com/xhanio/zen/pkg/services/conversation"
	"github.com/xhanio/zen/pkg/services/reference"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func newRefSvc(t *testing.T) (svc model.Reference, src, der, conv string) {
	t.Helper()
	svc, _, src, der, conv = newRefSvcWithRepo(t)
	return
}

// newRefSvcWithRepo also hands back the repository so a test can seed the
// message a reference will inherit its range from.
func newRefSvcWithRepo(t *testing.T) (svc model.Reference, repo repository.Repository, src, der, conv string) {
	t.Helper()
	repo = repository.New(testutil.NewDB(t))
	ctx := context.Background()
	g := &entity.Group{ID: ulidutil.New(), Name: "g"}
	_ = repo.CreateGroup(ctx, g)
	tagSvc := tag.New(repo)
	cardSvc := card.New(repo, tagSvc, nil)
	convSvc := conversation.New(repo)
	a, err := cardSvc.Create(ctx, "a", "x", g.ID, nil, nil, nil, nil, nil, nil, nil, nil, entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("cardSvc.Create a: %v", err)
	}
	b, err := cardSvc.Create(ctx, "b", "y", g.ID, nil, nil, nil, nil, nil, nil, nil, nil, entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("cardSvc.Create b: %v", err)
	}
	c, err := convSvc.Create(ctx, "", nil, nil)
	if err != nil {
		t.Fatalf("convSvc.Create: %v", err)
	}
	src, der, conv = a.ID, b.ID, c.ID
	svc = reference.New(repo, cardSvc, convSvc)
	return
}

func TestReference_Create_HappyPath(t *testing.T) {
	svc, src, der, conv := newRefSvc(t)
	r, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv, SelectionText: "hello",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID == "" || r.SourceCardID != src || r.DerivedCardID != der || r.ConversationID == nil || *r.ConversationID != conv || r.SelectionText != "hello" {
		t.Fatalf("bad reference: %+v", r)
	}
}

func TestReference_Create_RejectsSelfReference(t *testing.T) {
	svc, src, _, conv := newRefSvc(t)
	_, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: src, ConversationID: conv, SelectionText: "x",
	})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestReference_Create_RejectsMissingSource(t *testing.T) {
	svc, _, der, conv := newRefSvc(t)
	_, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: ulidutil.New(), DerivedCardID: der, ConversationID: conv, SelectionText: "x",
	})
	if err == nil {
		t.Fatalf("expected error for missing source card")
	}
}

func TestReference_Create_RejectsEmptySelection(t *testing.T) {
	svc, src, der, conv := newRefSvc(t)
	_, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv, SelectionText: "",
	})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for empty selection, got %v", err)
	}
}

func TestReference_Create_RejectsTooLongSelection(t *testing.T) {
	svc, src, der, conv := newRefSvc(t)
	_, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv,
		SelectionText: strings.Repeat("x", 5001),
	})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for too-long selection, got %v", err)
	}
}

func TestReference_List_RequiresAtLeastOneFilter(t *testing.T) {
	svc, _, _, _ := newRefSvc(t)
	_, err := svc.List(context.Background(), api.ListReferencesRequest{})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest with no filters, got %v", err)
	}
}

func TestReference_List_FiltersBySource(t *testing.T) {
	svc, src, der, conv := newRefSvc(t)
	ctx := context.Background()
	_, _ = svc.Create(ctx, api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv, SelectionText: "a",
	})
	got, err := svc.List(ctx, api.ListReferencesRequest{SourceCardID: &src})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
}

func TestReference_Delete_RemovesRow(t *testing.T) {
	svc, src, der, conv := newRefSvc(t)
	ctx := context.Background()
	r, _ := svc.Create(ctx, api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv, SelectionText: "x",
	})
	if err := svc.Delete(ctx, r.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(ctx, r.ID); !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected NotFound after delete, got %v", err)
	}
}

func seedSelectionMessage(t *testing.T, repo repository.Repository, convID, text string, start, end, seq *int) string {
	t.Helper()
	m := &entity.Message{
		ID: ulidutil.New(), ConversationID: convID, Role: "user", Content: "tighten this",
		SelectionText: &text, CreatedAt: time.Now(),
		SelectionStart: start, SelectionEnd: end, SelectionSeq: seq,
	}
	if err := repo.CreateMessage(context.Background(), m); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	return m.ID
}

func TestReference_Create_InheritsRangeAndTextFromMessage(t *testing.T) {
	svc, repo, src, der, conv := newRefSvcWithRepo(t)
	start, end, seq := 12, 20, 4
	msgID := seedSelectionMessage(t, repo, conv, "the exact words", &start, &end, &seq)

	got, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv,
		// A retyped excerpt that differs from what the SPA captured. The
		// message's copy must win — that is the point of inheriting.
		SelectionText: "the  exact words",
		MessageID:     &msgID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.SelectionText != "the exact words" {
		t.Fatalf("selection_text = %q, want the message's copy", got.SelectionText)
	}
	if got.SelectionStart == nil || *got.SelectionStart != 12 || got.SelectionEnd == nil || *got.SelectionEnd != 20 {
		t.Fatalf("range not inherited: %+v %+v", got.SelectionStart, got.SelectionEnd)
	}
	if got.SelectionSeq == nil || *got.SelectionSeq != 4 {
		t.Fatalf("seq not inherited: %+v", got.SelectionSeq)
	}
}

func TestReference_Create_WithoutMessageIDStoresNoRange(t *testing.T) {
	svc, _, src, der, conv := newRefSvcWithRepo(t)
	got, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv,
		SelectionText: "back-filled by hand",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.SelectionText != "back-filled by hand" {
		t.Fatalf("selection_text = %q", got.SelectionText)
	}
	if got.SelectionStart != nil || got.SelectionEnd != nil {
		t.Fatalf("expected no range, got %+v %+v", got.SelectionStart, got.SelectionEnd)
	}
}

// A message that predates this feature carries text but no offsets: the text
// still wins over the caller's retype, the range is simply absent.
func TestReference_Create_MessageWithoutRangeStillInheritsText(t *testing.T) {
	svc, repo, src, der, conv := newRefSvcWithRepo(t)
	msgID := seedSelectionMessage(t, repo, conv, "older selection", nil, nil, nil)

	got, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv,
		SelectionText: "retyped", MessageID: &msgID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.SelectionText != "older selection" {
		t.Fatalf("selection_text = %q, want the message's copy", got.SelectionText)
	}
	if got.SelectionStart != nil {
		t.Fatalf("expected no range, got %v", *got.SelectionStart)
	}
}

func TestReference_Create_RejectsMessageFromAnotherConversation(t *testing.T) {
	svc, repo, src, der, conv := newRefSvcWithRepo(t)
	other := &entity.Conversation{ID: ulidutil.New(), Title: "other", CreatedAt: time.Now(), LastMessageAt: time.Now()}
	if err := repo.CreateConversation(context.Background(), other); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	start, end := 1, 3
	msgID := seedSelectionMessage(t, repo, other.ID, "elsewhere", &start, &end, nil)

	// SelectionText is supplied so this cannot pass via the empty-text guard —
	// only the conversation mismatch can reject it.
	if _, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv,
		SelectionText: "elsewhere", MessageID: &msgID,
	}); err == nil {
		t.Fatalf("expected an error: the message belongs to a different conversation")
	}
}

func TestReference_Create_RejectsNoTextAndNoMessage(t *testing.T) {
	svc, _, src, der, conv := newRefSvcWithRepo(t)
	if _, err := svc.Create(context.Background(), api.CreateReferenceRequest{
		SourceCardID: src, DerivedCardID: der, ConversationID: conv,
	}); err == nil {
		t.Fatalf("expected an error: nothing to anchor")
	}
}
