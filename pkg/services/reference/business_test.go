package reference_test

import (
	"context"
	"strings"
	"testing"

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
	repo := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	g := &entity.Group{ID: ulidutil.New(), Name: "g"}
	_ = repo.CreateGroup(ctx, g)
	tagSvc := tag.New(repo)
	cardSvc := card.New(repo, tagSvc, nil)
	convSvc := conversation.New(repo)
	a, err := cardSvc.Create(ctx, "a", "x", g.ID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("cardSvc.Create a: %v", err)
	}
	b, err := cardSvc.Create(ctx, "b", "y", g.ID, nil, nil, nil, nil, nil, nil, nil, nil)
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
