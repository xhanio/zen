package card_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func snapshotsFor(t *testing.T, repo repository.Repository, cardID string) []*entity.CardSnapshot {
	t.Helper()
	got, err := repo.ListCardSnapshots(context.Background(),
		api.ListSnapshotsRequest{CardID: &cardID})
	if err != nil {
		t.Fatalf("ListCardSnapshots: %v", err)
	}
	return got
}

func TestSnapshot_CreateWritesBaseline(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	c, err := svc.Create(context.Background(), "t", "body", groupID, nil, nil, nil,
		nil, nil, nil, nil, nil, entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := snapshotsFor(t, repo, c.ID)
	if len(got) != 1 || got[0].Seq != 1 || got[0].ChangeKind != "create" {
		t.Fatalf("want one seq=1 create snapshot, got %+v", got)
	}
	if got[0].Actor != "user" {
		t.Fatalf("zero attribution must default to user, got %q", got[0].Actor)
	}
}

func TestSnapshot_ContentUpdateWritesExactlyOne(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	c, err := svc.Create(ctx, "t", "before", groupID, nil, nil, nil, nil, nil, nil, nil, nil,
		entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := "after"
	conv := ulidutil.New()
	if _, err := svc.Update(ctx, c.ID, nil, &body, nil, nil, nil, nil, nil, false, nil, nil,
		entity.SnapshotAttribution{Actor: "agent", ConversationID: &conv}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got := snapshotsFor(t, repo, c.ID)
	if len(got) != 2 {
		t.Fatalf("want 2 snapshots, got %d", len(got))
	}
	latest := got[0] // card-filtered lists are newest-first
	if latest.Seq != 2 || latest.ChangeKind != "update" || latest.Actor != "agent" {
		t.Fatalf("unexpected snapshot: %+v", latest)
	}
	if latest.ConversationID == nil || *latest.ConversationID != conv {
		t.Fatalf("conversation not recorded: %v", latest.ConversationID)
	}
	if latest.Content != "after" {
		t.Fatalf("snapshot stores post-state; content = %q", latest.Content)
	}
	if latest.LinesAdded == 0 && latest.LinesRemoved == 0 {
		t.Fatalf("diff counts not computed")
	}
	if latest.Diff == "" {
		t.Fatalf("diff payload not stored")
	}
}

func TestSnapshot_NonContentUpdateWritesNone(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	c, err := svc.Create(ctx, "t", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil,
		entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	tags := []string{"alpha"}
	if _, err := svc.Update(ctx, c.ID, nil, nil, nil, nil, &tags, nil, nil, false, nil, nil,
		entity.SnapshotAttribution{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := snapshotsFor(t, repo, c.ID); len(got) != 1 {
		t.Fatalf("tag-only update must not snapshot; got %d", len(got))
	}
}

func TestSnapshot_DecomposeCapturesParentBodyBeforeClear(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, err := svc.Create(ctx, "doc", "## A\nalpha\n\n## B\nbeta", groupID, nil, nil, nil,
		nil, nil, nil, nil, nil, entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards: []api.CardSpec{
			{Title: "A", Content: "alpha"},
			{Title: "B", Content: "beta"},
		},
	}, entity.SnapshotAttribution{Actor: "agent"}); err != nil {
		t.Fatalf("Decompose: %v", err)
	}

	got := snapshotsFor(t, repo, parent.ID)
	if len(got) != 2 {
		t.Fatalf("want baseline + decompose, got %d", len(got))
	}
	if got[0].ChangeKind != "decompose" {
		t.Fatalf("change_kind = %q, want decompose", got[0].ChangeKind)
	}
	// The pre-clear body is the point of this snapshot: decompose is the one
	// path that destroys prose without a trace.
	if got[1].Content == "" {
		t.Fatalf("the pre-clear body must survive in the baseline snapshot")
	}
	if got[0].Content != "" {
		t.Fatalf("post-decompose parent body should be empty, got %q", got[0].Content)
	}
}

func TestSnapshot_PurgeDeletesSnapshots(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	c, err := svc.Create(ctx, "t", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil,
		entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, c.ID, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := svc.Purge(ctx, c.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if got := snapshotsFor(t, repo, c.ID); len(got) != 0 {
		t.Fatalf("purge must remove snapshots; got %d", len(got))
	}
}

func TestSnapshot_EmptyTrashDeletesSnapshots(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	c, err := svc.Create(ctx, "t", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil,
		entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, c.ID, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.EmptyTrash(ctx); err != nil {
		t.Fatalf("EmptyTrash: %v", err)
	}
	if got := snapshotsFor(t, repo, c.ID); len(got) != 0 {
		t.Fatalf("empty trash must remove snapshots; got %d", len(got))
	}
}

// The inline reference on card.create inherits from the message exactly as
// reference.create does — the agent passes an id, never a retyped excerpt.
func TestInlineReference_InheritsFromMessage(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()

	source, err := svc.Create(ctx, "source", "alpha beta gamma", groupID, nil, nil, nil,
		nil, nil, nil, nil, nil, entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create source: %v", err)
	}
	conv := &entity.Conversation{ID: ulidutil.New(), Title: "c", CreatedAt: time.Now(), LastMessageAt: time.Now()}
	if err := repo.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	start, end, seq := 6, 10, 1
	captured := "beta"
	msg := &entity.Message{
		ID: ulidutil.New(), ConversationID: conv.ID, Role: "user", Content: "what is this?",
		SelectionText: &captured, CreatedAt: time.Now(),
		SelectionStart: &start, SelectionEnd: &end, SelectionSeq: &seq,
	}
	if err := repo.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	derived, err := svc.Create(ctx, "derived", "spun off", groupID, nil, &source.ID, &conv.ID,
		nil, nil, nil, &api.ReferenceSpec{MessageID: &msg.ID}, nil, entity.SnapshotAttribution{})
	if err != nil {
		t.Fatalf("Create derived: %v", err)
	}

	refs, err := repo.ListReferences(ctx, api.ListReferencesRequest{DerivedCardID: &derived.ID})
	if err != nil {
		t.Fatalf("ListReferences: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("want 1 reference, got %d", len(refs))
	}
	r := refs[0]
	if r.SelectionText != "beta" {
		t.Fatalf("selection_text = %q, want the message's copy", r.SelectionText)
	}
	if r.SelectionStart == nil || *r.SelectionStart != 6 || r.SelectionEnd == nil || *r.SelectionEnd != 10 {
		t.Fatalf("range not inherited: %+v %+v", r.SelectionStart, r.SelectionEnd)
	}
}
