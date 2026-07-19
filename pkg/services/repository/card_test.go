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

func mustCreateGroupRow(t *testing.T, r repository.Repository, name string) string {
	t.Helper()
	g := &entity.Group{ID: ulidutil.New(), Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.CreateGroup(context.Background(), g); err != nil {
		t.Fatalf("CreateGroup %q: %v", name, err)
	}
	return g.ID
}

func mustCreateCardSetup(t *testing.T, r repository.Repository) string {
	t.Helper()
	return mustCreateGroupRow(t, r, "work")
}

func TestCreateAndGetCard(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	c := &entity.Card{
		ID: ulidutil.New(), Title: "first", Content: "body",
		GroupID:   groupID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	got, err := r.GetCard(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.Title != "first" || got.GroupID != groupID {
		t.Fatalf("bad card: %+v", got)
	}
}

func TestListCards_ByGroup(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()

	for _, name := range []string{"a", "b", "c"} {
		_ = r.CreateCard(ctx, &entity.Card{ID: ulidutil.New(), Title: name, GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	}

	all, err := r.ListCards(ctx, nil, false)
	if err != nil {
		t.Fatalf("ListCards: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("want 3 cards total, got %d", len(all))
	}

	group, err := r.ListCards(ctx, &groupID, false)
	if err != nil {
		t.Fatalf("ListCards: %v", err)
	}
	if len(group) != 3 {
		t.Fatalf("want 3 cards by group, got %d", len(group))
	}
}

func TestDeleteCardsByGroup(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	c := &entity.Card{ID: ulidutil.New(), Title: "c1", GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = r.CreateCard(ctx, c)

	if err := r.DeleteCardsByGroup(ctx, groupID); err != nil {
		t.Fatalf("DeleteCardsByGroup: %v", err)
	}
	if _, err := r.GetCard(ctx, c.ID); err == nil {
		t.Fatalf("expected NotFound after group cascade delete")
	}
}

func TestCardTag_AttachListDetach(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()

	tagA := &entity.Tag{ID: ulidutil.New(), GroupID: groupID, Name: "go"}
	tagB := &entity.Tag{ID: ulidutil.New(), GroupID: groupID, Name: "rust"}
	_ = r.CreateTag(ctx, tagA)
	_ = r.CreateTag(ctx, tagB)

	card := &entity.Card{ID: ulidutil.New(), Title: "polyglot", GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = r.CreateCard(ctx, card)

	if err := r.AttachTag(ctx, card.ID, tagA.ID); err != nil {
		t.Fatalf("AttachTag a: %v", err)
	}
	if err := r.AttachTag(ctx, card.ID, tagB.ID); err != nil {
		t.Fatalf("AttachTag b: %v", err)
	}

	names, err := r.ListTagsForCard(ctx, card.ID)
	if err != nil {
		t.Fatalf("ListTagsForCard: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("want 2 tags, got %d: %v", len(names), names)
	}

	if err := r.DetachTag(ctx, card.ID, tagA.ID); err != nil {
		t.Fatalf("DetachTag: %v", err)
	}
	names, _ = r.ListTagsForCard(ctx, card.ID)
	if len(names) != 1 || names[0] != "rust" {
		t.Fatalf("want only rust remaining, got %v", names)
	}
}

func TestCardTag_DeletingCardCascadesCardTags(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	tag := &entity.Tag{ID: ulidutil.New(), GroupID: groupID, Name: "x"}
	_ = r.CreateTag(ctx, tag)
	card := &entity.Card{ID: ulidutil.New(), Title: "c", GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = r.CreateCard(ctx, card)
	_ = r.AttachTag(ctx, card.ID, tag.ID)

	if err := r.DeleteCard(ctx, card.ID); err != nil {
		t.Fatalf("DeleteCard: %v", err)
	}
	cardIDs, _ := r.ListCardsForTag(ctx, "x")
	if len(cardIDs) != 0 {
		t.Fatalf("expected card_tags to cascade-delete, got %v", cardIDs)
	}
}

func TestCard_RoundTrip_PreservesFormat(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()

	now := time.Now()
	for _, format := range []string{"markdown", "html", "text"} {
		c := &entity.Card{
			ID: ulidutil.New(), Title: "t", Content: "<p>hi</p>", Format: format,
			GroupID: groupID, CreatedAt: now, UpdatedAt: now,
		}
		if err := r.CreateCard(ctx, c); err != nil {
			t.Fatalf("CreateCard %s: %v", format, err)
		}
		got, err := r.GetCard(ctx, c.ID)
		if err != nil {
			t.Fatalf("GetCard %s: %v", format, err)
		}
		if got.Format != format {
			t.Fatalf("format: got %q want %q", got.Format, format)
		}
	}
}

func TestCard_RoundTrip_PreservesLevelEntryID(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()

	now := time.Now()
	entryID := ulidutil.New()
	c := &entity.Card{
		ID: ulidutil.New(), Title: "t", Content: "body",
		LevelEntryID: &entryID,
		GroupID:      groupID, CreatedAt: now, UpdatedAt: now,
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	got, err := r.GetCard(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.LevelEntryID == nil || *got.LevelEntryID != entryID {
		t.Fatalf("LevelEntryID: got %v want %s", got.LevelEntryID, entryID)
	}
}

func TestCard_NilLevelEntryIDRoundTrips(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()

	now := time.Now()
	c := &entity.Card{
		ID: ulidutil.New(), Title: "t", Content: "body",
		GroupID: groupID, CreatedAt: now, UpdatedAt: now,
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	got, _ := r.GetCard(ctx, c.ID)
	if got.LevelEntryID != nil {
		t.Fatalf("expected nil LevelEntryID, got %v", *got.LevelEntryID)
	}
}

func TestCard_RoundTrip_PreservesGenesisAndDeletedAt(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()

	now := time.Now()
	deletedAt := now.Add(-time.Hour)
	c := &entity.Card{
		ID:        ulidutil.New(),
		Title:     "t",
		Content:   "body",
		Genesis:   "Decomposed from card 01ABC",
		DeletedAt: &deletedAt,
		GroupID:   groupID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	got, err := r.GetCard(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.Genesis != "Decomposed from card 01ABC" {
		t.Fatalf("Genesis: got %q", got.Genesis)
	}
	if got.DeletedAt == nil || !got.DeletedAt.Equal(deletedAt) {
		t.Fatalf("DeletedAt: got %v want %v", got.DeletedAt, deletedAt)
	}
}

func TestCard_ListCards_ExcludesTrashedByDefault(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	now := time.Now()

	live := &entity.Card{ID: ulidutil.New(), Title: "live", GroupID: groupID, CreatedAt: now, UpdatedAt: now}
	deletedAt := now
	dead := &entity.Card{ID: ulidutil.New(), Title: "dead", DeletedAt: &deletedAt, GroupID: groupID, CreatedAt: now, UpdatedAt: now}
	_ = r.CreateCard(ctx, live)
	_ = r.CreateCard(ctx, dead)

	got, err := r.ListCards(ctx, &groupID, false)
	if err != nil {
		t.Fatalf("ListCards: %v", err)
	}
	if len(got) != 1 || got[0].Title != "live" {
		t.Fatalf("got %d cards (titles=%v), want only live", len(got), titles(got))
	}

	all, err := r.ListCards(ctx, &groupID, true)
	if err != nil {
		t.Fatalf("ListCards include_trashed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("include_trashed got %d, want 2", len(all))
	}
}

func TestCard_RestoreCard_ClearsDeletedAt(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	now := time.Now()

	c := &entity.Card{ID: ulidutil.New(), Title: "x", DeletedAt: &now, GroupID: groupID, CreatedAt: now, UpdatedAt: now}
	_ = r.CreateCard(ctx, c)

	if err := r.RestoreCard(ctx, c.ID); err != nil {
		t.Fatalf("RestoreCard: %v", err)
	}
	got, _ := r.GetCard(ctx, c.ID)
	if got.DeletedAt != nil {
		t.Fatalf("DeletedAt: got %v want nil", got.DeletedAt)
	}
}

func TestCard_RestoreCard_RejectsLiveCard(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	now := time.Now()

	c := &entity.Card{ID: ulidutil.New(), Title: "x", GroupID: groupID, CreatedAt: now, UpdatedAt: now}
	_ = r.CreateCard(ctx, c)

	err := r.RestoreCard(ctx, c.ID)
	if err == nil {
		t.Fatal("expected NotFound for live card; got nil")
	}
}

func TestCard_PurgeCard_RemovesRow(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	now := time.Now()

	c := &entity.Card{ID: ulidutil.New(), Title: "x", DeletedAt: &now, GroupID: groupID, CreatedAt: now, UpdatedAt: now}
	_ = r.CreateCard(ctx, c)

	if err := r.PurgeCard(ctx, c.ID); err != nil {
		t.Fatalf("PurgeCard: %v", err)
	}
	if _, err := r.GetCard(ctx, c.ID); err == nil {
		t.Fatal("expected card to be gone after purge")
	}
}

func TestCard_PurgeCard_RejectsLiveCard(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	now := time.Now()

	c := &entity.Card{ID: ulidutil.New(), Title: "x", GroupID: groupID, CreatedAt: now, UpdatedAt: now}
	_ = r.CreateCard(ctx, c)

	err := r.PurgeCard(ctx, c.ID)
	if err == nil {
		t.Fatal("expected error purging live card; got nil")
	}
}

func TestCard_ListTrashedCards_OrderByDeletedAtDesc(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	t1 := time.Now()
	t2 := t1.Add(time.Hour)

	older := &entity.Card{ID: ulidutil.New(), Title: "older", DeletedAt: &t1, GroupID: groupID, CreatedAt: t1, UpdatedAt: t1}
	newer := &entity.Card{ID: ulidutil.New(), Title: "newer", DeletedAt: &t2, GroupID: groupID, CreatedAt: t2, UpdatedAt: t2}
	_ = r.CreateCard(ctx, older)
	_ = r.CreateCard(ctx, newer)

	got, err := r.ListTrashedCards(ctx, 10)
	if err != nil {
		t.Fatalf("ListTrashedCards: %v", err)
	}
	if len(got) != 2 || got[0].Title != "newer" {
		t.Fatalf("ordering: %+v", titles(got))
	}
}

func titles(cs []*entity.Card) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Title
	}
	return out
}

func TestListChildren_OrdersByPositionAndFiltersTrashed(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	ctx := context.Background()
	groupID := mustCreateCardSetup(t, r)
	now := time.Now()
	parent := &entity.Card{ID: ulidutil.New(), Title: "P", GroupID: groupID, CreatedAt: now, UpdatedAt: now}
	if err := r.CreateCard(ctx, parent); err != nil {
		t.Fatalf("CreateCard parent: %v", err)
	}
	mk := func(title string, pos int, trashed bool) string {
		id := ulidutil.New()
		c := &entity.Card{
			ID:           id,
			Title:        title,
			GroupID:      groupID,
			ParentCardID: &parent.ID,
			Position:     pos,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if trashed {
			t2 := now
			c.DeletedAt = &t2
		}
		if err := r.CreateCard(ctx, c); err != nil {
			t.Fatalf("CreateCard child: %v", err)
		}
		return id
	}
	b := mk("B", 1, false)
	a := mk("A", 0, false)
	c := mk("C", 2, true) // trashed

	live, err := r.ListChildren(ctx, parent.ID, false)
	if err != nil {
		t.Fatalf("ListChildren live: %v", err)
	}
	if len(live) != 2 || live[0].ID != a || live[1].ID != b {
		t.Fatalf("expected [A, B] in position order, got: %+v", live)
	}

	all, err := r.ListChildren(ctx, parent.ID, true)
	if err != nil {
		t.Fatalf("ListChildren all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 with include_trashed, got %d", len(all))
	}
	if all[2].ID != c {
		t.Fatalf("expected trashed C last at position 2, got %+v", all)
	}
}

func TestCardRepo_ReviewGrade_RoundTrip(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	reviewedAt := time.Now().UTC().Truncate(time.Second)
	c := &entity.Card{
		ID:          ulidutil.New(),
		Title:       "review round-trip",
		Content:     "body",
		Format:      "html",
		GroupID:     groupID,
		ReviewGrade: "GRILLED",
		ReviewedAt:  &reviewedAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	got, err := r.GetCard(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "GRILLED" {
		t.Fatalf("expected GRILLED, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at non-nil")
	}
	if diff := got.ReviewedAt.Sub(reviewedAt).Abs(); diff > time.Second {
		t.Fatalf("expected reviewed_at close to %v, got %v (diff %v)", reviewedAt, *got.ReviewedAt, diff)
	}
}

func TestCardRepo_ReviewGrade_DefaultsToLGTM(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	groupID := mustCreateCardSetup(t, r)
	ctx := context.Background()
	c := &entity.Card{
		ID:        ulidutil.New(),
		Title:     "default grade",
		Content:   "body",
		Format:    "html",
		GroupID:   groupID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	got, err := r.GetCard(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if got.ReviewGrade != "LGTM" {
		t.Fatalf("expected LGTM default, got %q", got.ReviewGrade)
	}
	if got.ReviewedAt != nil {
		t.Fatalf("expected reviewed_at nil, got %v", *got.ReviewedAt)
	}
}
