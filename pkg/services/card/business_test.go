package card_test

import (
	"context"
	"testing"
	"strings"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/services/card"
	"github.com/xhanio/zen/pkg/services/conversation"
	"github.com/xhanio/zen/pkg/services/group"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func newCardCtx(t *testing.T) (svc model.Card, repo repository.Repository, groupID string) {
	t.Helper()
	repo = repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	groupID = g.ID
	tagSvc := tag.New(repo)
	svc = card.New(repo, tagSvc, nil)
	return
}

func TestUpdate_GroupMove_ClearsLevelAndRehomesTags(t *testing.T) {
	svc, repo, _ := newCardCtx(t)
	gsvc := group.New(repo, nil)
	ctx := context.Background()

	gA, _ := gsvc.Create(ctx, "A", []entity.LevelEntry{{Weight: 0, Name: "principle"}}, "")
	gB, _ := gsvc.Create(ctx, "B", nil, "")
	entryID := gA.LevelCatalog[0].ID

	// card in A, leveled + tagged
	card, err := svc.Create(ctx, "c", "", gA.ID, []string{"draft"}, nil, nil, nil, &entryID, nil, nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// move to B
	moved, err := svc.Update(ctx, card.ID, nil, nil, &gB.ID, nil, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.GroupID != gB.ID {
		t.Fatalf("group not changed: %s", moved.GroupID)
	}
	if moved.LevelEntryID != nil {
		t.Fatalf("level should be cleared on move, got %v", *moved.LevelEntryID)
	}
	found := false
	for _, tg := range moved.Tags {
		if tg == "draft" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tag should survive the move, got %v", moved.Tags)
	}
	// "draft" now belongs to B and is attached to the card there.
	if _, err := repo.GetTagByNameInGroup(ctx, gB.ID, "draft"); err != nil {
		t.Fatalf("draft not re-homed into B: %v", err)
	}
	listB, _ := repo.ListTags(ctx, gB.ID)
	var bCount int
	for _, x := range listB {
		if x.Name == "draft" {
			bCount = x.CardCount
		}
	}
	if bCount != 1 {
		t.Fatalf("B draft card_count = %d, want 1 (card re-homed)", bCount)
	}
	// The card is no longer attached to A's draft tag (A's tag may linger
	// orphaned — tags aren't auto-deleted, matching the detach-only model).
	listA, _ := repo.ListTags(ctx, gA.ID)
	for _, x := range listA {
		if x.Name == "draft" && x.CardCount != 0 {
			t.Fatalf("A draft still attached to a card (count %d)", x.CardCount)
		}
	}
}

func TestCard_Create_NoTags(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	c, err := svc.Create(context.Background(), "first", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == "" || c.Title != "first" {
		t.Fatalf("bad card: %+v", c)
	}
	if c.Tags == nil {
		t.Fatalf("Tags should be non-nil empty slice, got nil")
	}
	if len(c.Tags) != 0 {
		t.Fatalf("expected zero tags, got %v", c.Tags)
	}
}

func TestCard_Create_WithTags(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	c, err := svc.Create(context.Background(), "polyglot", "", groupID, []string{"Go", "RUST"}, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(c.Tags) != 2 {
		t.Fatalf("want 2 tags, got %v", c.Tags)
	}
}

func TestCard_Create_MissingGroup(t *testing.T) {
	svc, _, _ := newCardCtx(t)
	_, err := svc.Create(context.Background(), "x", "", ulidutil.New(), nil, nil, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}


func TestCard_Get_LoadsTags(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	created, _ := svc.Create(context.Background(), "x", "", groupID, []string{"go"}, nil, nil, nil, nil, nil, nil, nil)
	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "go" {
		t.Fatalf("expected one tag 'go', got %v", got.Tags)
	}
}

func TestCard_Create_WithProvenance(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	parent, err := svc.Create(context.Background(), "parent", "", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	conv := &entity.Conversation{
		ID:            ulidutil.New(),
		Title:         "x",
		CreatedAt:     time.Now(),
		LastMessageAt: time.Now(),
	}
	if err := repo.CreateConversation(context.Background(), conv); err != nil {
		t.Fatalf("create conv: %v", err)
	}
	c, err := svc.Create(context.Background(), "derived", "from chat", groupID, nil, &parent.ID, &conv.ID, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ParentCardID == nil || *c.ParentCardID != parent.ID {
		t.Fatalf("ParentCardID lost: %+v", c.ParentCardID)
	}
	if c.SourceConversationID == nil || *c.SourceConversationID != conv.ID {
		t.Fatalf("SourceConversationID lost: %+v", c.SourceConversationID)
	}
}

func TestCard_Delete_SoftDeletes(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	c, _ := svc.Create(context.Background(), "x", "", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err := svc.Delete(context.Background(), c.ID, true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err := svc.Get(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Get after soft-delete: %v", err)
	}
	if got.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set after soft-delete")
	}
}

func TestCard_Update_RetainsTags(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	c, _ := svc.Create(context.Background(), "old", "", groupID, []string{"go"}, nil, nil, nil, nil, nil, nil, nil)
	newTitle := "new"
	updated, err := svc.Update(context.Background(), c.ID, &newTitle, nil, nil, nil, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "new" {
		t.Fatalf("title not updated: %q", updated.Title)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "go" {
		t.Fatalf("expected tag retained, got %v", updated.Tags)
	}
}

func TestCard_Update_AddsTags(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	created, err := svc.Create(context.Background(), "T", "", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	tags := []string{"alpha", "beta"}
	updated, err := svc.Update(context.Background(), created.ID, nil, nil, nil, nil, &tags, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := map[string]bool{}
	for _, n := range updated.Tags {
		got[n] = true
	}
	if !got["alpha"] || !got["beta"] || len(updated.Tags) != 2 {
		t.Fatalf("expected {alpha,beta}, got %v", updated.Tags)
	}
}

func TestCard_Update_RemovesTags(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	created, err := svc.Create(context.Background(), "T", "", groupID, []string{"keep", "drop"}, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	tags := []string{"keep"}
	updated, err := svc.Update(context.Background(), created.ID, nil, nil, nil, nil, &tags, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "keep" {
		t.Fatalf("expected [keep], got %v", updated.Tags)
	}
}

func TestCard_Update_EmptyTagsRemovesAll(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	created, err := svc.Create(context.Background(), "T", "", groupID, []string{"a", "b"}, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	empty := []string{}
	updated, err := svc.Update(context.Background(), created.ID, nil, nil, nil, nil, &empty, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.Tags) != 0 {
		t.Fatalf("expected [], got %v", updated.Tags)
	}
}

func TestCard_Update_NilTagsLeavesUnchanged(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	created, err := svc.Create(context.Background(), "T", "", groupID, []string{"keep1", "keep2"}, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	newTitle := "T2"
	updated, err := svc.Update(context.Background(), created.ID, &newTitle, nil, nil, nil, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "T2" {
		t.Fatalf("expected title=T2, got %q", updated.Title)
	}
	got := map[string]bool{}
	for _, n := range updated.Tags {
		got[n] = true
	}
	if !got["keep1"] || !got["keep2"] || len(updated.Tags) != 2 {
		t.Fatalf("expected {keep1,keep2}, got %v", updated.Tags)
	}
}

func TestCard_Purge_CascadesAnchoredConversations(t *testing.T) {
	repo := repository.New(testutil.NewDB(t))
	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	conv := conversation.New(repo)
	svc := card.New(repo, tag.New(repo), conv)
	ctx := context.Background()

	c, err := svc.Create(ctx, "x", "", g.ID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	kind := "card"
	conversationRec, err := conv.Create(ctx, "linked", &kind, &c.ID)
	if err != nil {
		t.Fatalf("Create conversation: %v", err)
	}
	if err := svc.Delete(ctx, c.ID, true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := svc.Purge(ctx, c.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if _, err := conv.Get(ctx, conversationRec.ID); !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected anchored conversation gone, got: %v", err)
	}
}

func TestCard_Create_DefaultsToMarkdown(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	c, err := svc.Create(context.Background(), "x", "hello", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.Format != "markdown" {
		t.Fatalf("Format = %q want markdown", c.Format)
	}
}

func TestCard_Create_AcceptsHtml(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	f := "html"
	c, err := svc.Create(context.Background(), "x", "<p>hi</p>", groupID, nil, nil, nil, &f, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.Format != "html" {
		t.Fatalf("Format = %q want html", c.Format)
	}
}

func TestCard_Create_RejectsUnknownFormat(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	f := "bbcode"
	_, err := svc.Create(context.Background(), "x", "", groupID, nil, nil, nil, &f, nil, nil, nil, nil)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestCard_Create_AttachesToKnownEntry(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	// Seed a catalog entry via the group repo (in v0.10, cards can only
	// point at pre-existing catalog entries; new entries are added via
	// group.update).
	g, _ := repo.GetGroup(context.Background(), groupID)
	entryID := ulidutil.New()
	g.LevelCatalog = []entity.LevelEntry{{ID: entryID, Weight: 0, Name: "原则"}}
	if err := repo.UpdateGroup(context.Background(), g); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	c, err := svc.Create(context.Background(), "x", "hi", groupID, nil, nil, nil, nil, &entryID, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.LevelEntryID == nil || *c.LevelEntryID != entryID {
		t.Fatalf("LevelEntryID = %v want %s", c.LevelEntryID, entryID)
	}
}

func TestCard_Create_RejectsUnknownEntry(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	unknown := ulidutil.New()
	_, err := svc.Create(context.Background(), "x", "", groupID, nil, nil, nil, nil, &unknown, nil, nil, nil)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestCard_Update_ClearLevelEntry(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	g, _ := repo.GetGroup(context.Background(), groupID)
	entryID := ulidutil.New()
	g.LevelCatalog = []entity.LevelEntry{{ID: entryID, Weight: 0, Name: "原则"}}
	if err := repo.UpdateGroup(context.Background(), g); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	c, _ := svc.Create(context.Background(), "a", "", groupID, nil, nil, nil, nil, &entryID, nil, nil, nil)
	got, err := svc.Update(context.Background(), c.ID, nil, nil, nil, nil, nil, nil, nil, true, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.LevelEntryID != nil {
		t.Fatalf("expected nil LevelEntryID after ClearLevelEntry, got %v", *got.LevelEntryID)
	}
}

func TestCard_Compose_SoftDeletesSourcesAndCreatesTarget(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	b, _ := svc.Create(ctx, "B", "beta", groupID, nil, nil, nil, nil, nil, nil, nil, nil)

	gid := groupID
	resp, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, b.ID},
		Target: api.CardSpec{
			Title:   "Merged",
			Content: "alpha + beta",
			GroupID: &gid,
		},
	})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if resp.Card == nil || resp.Card.Title != "Merged" || resp.Card.GroupID != groupID {
		t.Fatalf("bad target: %+v", resp.Card)
	}

	got, _ := svc.Get(ctx, a.ID)
	if got.DeletedAt == nil {
		t.Fatalf("source A should be soft-deleted, got %+v", got)
	}
	got, _ = svc.Get(ctx, b.ID)
	if got.DeletedAt == nil {
		t.Fatalf("source B should be soft-deleted, got %+v", got)
	}
}

func TestCard_Compose_RejectsLessThan2Sources(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID},
		Target:        api.CardSpec{Title: "T", Content: "x"},
	})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestCard_Compose_RejectsDuplicateSourceIDs(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, a.ID},
		Target:        api.CardSpec{Title: "T", Content: "x"},
	})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for dup IDs, got %v", err)
	}
}

func TestCard_Compose_RejectsAlreadySoftDeletedSource(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	b, _ := svc.Create(ctx, "B", "beta", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err := svc.Delete(ctx, a.ID, true); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, b.ID},
		Target:        api.CardSpec{Title: "T", Content: "x"},
	})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for soft-deleted source, got %v", err)
	}
}

func TestCard_Compose_RejectsCrossGroupSources(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	g2 := &entity.Group{ID: ulidutil.New(), Name: "other", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g2); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	b, _ := svc.Create(ctx, "B", "beta", g2.ID, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, b.ID},
		Target:        api.CardSpec{Title: "T", Content: "x"},
	})
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for cross-group sources, got %v", err)
	}
}

func TestCard_Compose_DefaultsGenesisToComposedFromTitles(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	b, _ := svc.Create(ctx, "B", "beta", groupID, nil, nil, nil, nil, nil, nil, nil, nil)

	resp, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, b.ID},
		Target:        api.CardSpec{Title: "T", Content: "x"},
	})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	// Genesis now uses source titles, not IDs — human-readable and
	// stable across re-runs. Order matches source_card_ids input.
	want := "Composed from A, B"
	if resp.Card.Genesis != want {
		t.Fatalf("genesis mismatch: want %q, got %q", want, resp.Card.Genesis)
	}
}

func TestCard_Compose_AllowsTargetGenesisOverride(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	b, _ := svc.Create(ctx, "B", "beta", groupID, nil, nil, nil, nil, nil, nil, nil, nil)

	gen := "custom provenance"
	resp, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, b.ID},
		Target:        api.CardSpec{Title: "T", Content: "x", Genesis: &gen},
	})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if resp.Card.Genesis != gen {
		t.Fatalf("genesis override ignored: got %q", resp.Card.Genesis)
	}
}

func TestCard_Compose_AllowsTargetGroupOverride(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	g2 := &entity.Group{ID: ulidutil.New(), Name: "elsewhere", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g2); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	b, _ := svc.Create(ctx, "B", "beta", groupID, nil, nil, nil, nil, nil, nil, nil, nil)

	other := g2.ID
	resp, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, b.ID},
		Target:        api.CardSpec{Title: "T", Content: "x", GroupID: &other},
	})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if resp.Card.GroupID != g2.ID {
		t.Fatalf("group override ignored: got %q", resp.Card.GroupID)
	}
}

func TestCard_Compose_RollsBackOnTargetCreationFailure(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, "A", "alpha", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	b, _ := svc.Create(ctx, "B", "beta", groupID, nil, nil, nil, nil, nil, nil, nil, nil)

	// Empty title fails CardSpec validation inside createInTx.
	_, err := svc.Compose(ctx, api.ComposeRequest{
		SourceCardIDs: []string{a.ID, b.ID},
		Target:        api.CardSpec{Title: "", Content: "x"},
	})
	if err == nil {
		t.Fatalf("expected error on empty target title")
	}
	got, _ := svc.Get(ctx, a.ID)
	if got.DeletedAt != nil {
		t.Fatalf("source A soft-delete should have rolled back, got %+v", got)
	}
	got, _ = svc.Get(ctx, b.ID)
	if got.DeletedAt != nil {
		t.Fatalf("source B soft-delete should have rolled back, got %+v", got)
	}
}

func TestCard_Get_AttachesReferencesWhereCardIsSource(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	src, _ := svc.Create(ctx, "src", "x", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	der, _ := svc.Create(ctx, "der", "y", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	conv := &entity.Conversation{ID: ulidutil.New(), Title: "", CreatedAt: time.Now(), LastMessageAt: time.Now()}
	if err := repo.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	cid := conv.ID
	r := &entity.Reference{
		ID: ulidutil.New(), SourceCardID: src.ID, DerivedCardID: der.ID,
		ConversationID: &cid, SelectionText: "hello", CreatedAt: time.Now(),
	}
	if err := repo.CreateReference(ctx, r); err != nil {
		t.Fatalf("CreateReference: %v", err)
	}
	got, err := svc.Get(ctx, src.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.References) != 1 || got.References[0].ID != r.ID {
		t.Fatalf("expected 1 reference attached, got %+v", got.References)
	}
}

func TestCard_Create_WithInlineReference_HappyPath(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "parent", "x", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	conv := &entity.Conversation{ID: ulidutil.New(), Title: "", CreatedAt: time.Now(), LastMessageAt: time.Now()}
	if err := repo.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	pid := parent.ID
	cid := conv.ID
	child, err := svc.Create(ctx, "child", "y", groupID, nil, &pid, &cid, nil, nil, nil,
		&api.ReferenceSpec{SelectionText: "quick"}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	refs, err := repo.ListReferences(ctx, api.ListReferencesRequest{SourceCardID: &pid})
	if err != nil {
		t.Fatalf("ListReferences: %v", err)
	}
	if len(refs) != 1 || refs[0].DerivedCardID != child.ID || refs[0].SelectionText != "quick" {
		t.Fatalf("expected one reference linking parent→child, got %+v", refs)
	}
	if refs[0].ConversationID == nil || *refs[0].ConversationID != conv.ID {
		t.Fatalf("expected conversation_id %q, got %v", conv.ID, refs[0].ConversationID)
	}
}

func TestCard_Create_WithInlineReference_AllowsNullConversation(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "parent", "x", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	pid := parent.ID
	child, err := svc.Create(ctx, "child", "y", groupID, nil, &pid, nil, nil, nil, nil,
		&api.ReferenceSpec{SelectionText: "anchor"}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	refs, err := repo.ListReferences(ctx, api.ListReferencesRequest{SourceCardID: &pid})
	if err != nil {
		t.Fatalf("ListReferences: %v", err)
	}
	if len(refs) != 1 || refs[0].DerivedCardID != child.ID {
		t.Fatalf("expected one reference, got %+v", refs)
	}
	if refs[0].ConversationID != nil {
		t.Fatalf("expected ConversationID nil, got %v", *refs[0].ConversationID)
	}
}

func TestCard_Create_WithInlineReference_RejectsWithoutParent(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	_, err := svc.Create(context.Background(), "child", "y", groupID, nil, nil, nil, nil, nil, nil,
		&api.ReferenceSpec{SelectionText: "x"}, nil)
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestCard_Create_WithInlineReference_RejectsEmptySelection(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "parent", "x", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	pid := parent.ID
	_, err := svc.Create(ctx, "child", "y", groupID, nil, &pid, nil, nil, nil, nil,
		&api.ReferenceSpec{SelectionText: ""}, nil)
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestCard_Create_WithInlineReference_RejectsTooLongSelection(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "parent", "x", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	pid := parent.ID
	_, err := svc.Create(ctx, "child", "y", groupID, nil, &pid, nil, nil, nil, nil,
		&api.ReferenceSpec{SelectionText: strings.Repeat("x", 5001)}, nil)
	if err == nil || !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestCard_Children_HydratesTagsAndOrdersByPosition(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "P", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	pid := parent.ID
	tags := []string{"alpha"}
	svc.Create(ctx, "B", "b", groupID, tags, &pid, nil, nil, nil, nil, nil, nil)
	svc.Create(ctx, "A", "a", groupID, nil, &pid, nil, nil, nil, nil, nil, nil)

	got, err := svc.Children(ctx, pid, false)
	if err != nil {
		t.Fatalf("Children: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 children, got %d", len(got))
	}
	for _, c := range got {
		if c.Tags == nil {
			t.Fatalf("Tags should be a non-nil slice, got nil for %s", c.ID)
		}
	}
}

func TestDecompose_ClearsParentContentAndSkipsReferences(t *testing.T) {
	svc, repo, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "P", "original body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards: []api.CardSpec{
			{Title: "S1", Content: "s1"},
			{Title: "S2", Content: "s2"},
		},
	})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}
	if len(resp.Cards) != 2 {
		t.Fatalf("want 2 children, got %d", len(resp.Cards))
	}
	got, _ := svc.Get(ctx, parent.ID)
	if got.Content != "" {
		t.Fatalf("expected parent content cleared, got %q", got.Content)
	}
	refs, _ := repo.ListReferences(ctx, api.ListReferencesRequest{SourceCardID: &parent.ID})
	if len(refs) != 0 {
		t.Fatalf("expected 0 parent-side references after decompose, got %d", len(refs))
	}
}

func TestDecompose_RejectsWhenParentAlreadyHasLiveChildren(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "P", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards:        []api.CardSpec{{Title: "S1", Content: "s1"}},
	})
	if err != nil {
		t.Fatalf("first Decompose: %v", err)
	}
	_, err = svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards:        []api.CardSpec{{Title: "S2", Content: "s2"}},
	})
	if err == nil {
		t.Fatalf("expected error on re-decompose, got nil")
	}
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest kind, got: %v", err)
	}
}

func TestDecompose_ChildPositionDefaultsToSpecIndex(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "P", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards: []api.CardSpec{
			{Title: "S0", Content: "s0"},
			{Title: "S1", Content: "s1"},
			{Title: "S2", Content: "s2"},
		},
	})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}
	for i, c := range resp.Cards {
		if c.Position != i {
			t.Fatalf("child %d: expected position %d, got %d", i, i, c.Position)
		}
	}
}

func TestDecompose_KeepsContainerContentWhenSet(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "P", "meta\n\nS1\n\nS2", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	meta := "Date: 2026-07-11\nStatus: draft"
	resp, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID:     parent.ID,
		ContainerContent: &meta,
		Cards: []api.CardSpec{
			{Title: "S1", Content: "s1"},
			{Title: "S2", Content: "s2"},
		},
	})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}
	if len(resp.Cards) != 2 {
		t.Fatalf("want 2 children, got %d", len(resp.Cards))
	}
	got, _ := svc.Get(ctx, parent.ID)
	if got.Content != meta {
		t.Fatalf("expected parent content %q, got %q", meta, got.Content)
	}
}

func TestDecompose_RejectsContainerContentOnNestedParent(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, "P", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards:        []api.CardSpec{{Title: "S1", Content: "s1"}},
	})
	if err != nil {
		t.Fatalf("first Decompose: %v", err)
	}
	child := resp.Cards[0] // nested: parent_card_id == parent.ID
	meta := "Owner: x"
	_, err = svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID:     child.ID,
		ContainerContent: &meta,
		Cards:            []api.CardSpec{{Title: "G1", Content: "g1"}},
	})
	if err == nil {
		t.Fatalf("expected error setting container_content on nested parent, got nil")
	}
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest kind, got: %v", err)
	}
}
