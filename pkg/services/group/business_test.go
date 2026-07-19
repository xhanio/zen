package group_test

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/services/conversation"
	"github.com/xhanio/zen/pkg/services/group"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func TestGroup_Create_Success(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, err := svc.Create(context.Background(), "work", nil, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.ID == "" {
		t.Fatalf("expected ulid, got empty")
	}
	if g.Name != "work" {
		t.Fatalf("got name %q", g.Name)
	}
}

func TestGroup_Create_WithRule(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, err := svc.Create(context.Background(), "design", nil, "Translate into Chinese; HTML only.")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.Rule != "Translate into Chinese; HTML only." {
		t.Fatalf("got rule %q", g.Rule)
	}
}

func TestGroup_Update_SetsRule(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, err := svc.Create(context.Background(), "design", nil, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	rule := "Must be HTML."
	got, err := svc.Update(context.Background(), g.ID, nil, nil, nil, &rule)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Rule != rule {
		t.Fatalf("got rule %q, want %q", got.Rule, rule)
	}
	// omitting rule (nil) leaves it unchanged
	name := "design2"
	got2, err := svc.Update(context.Background(), g.ID, &name, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update2: %v", err)
	}
	if got2.Rule != rule {
		t.Fatalf("rule changed on nil update: got %q", got2.Rule)
	}
}

func TestGroup_Create_EmptyName(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	_, err := svc.Create(context.Background(), "", nil, "")
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}

func TestGroup_Create_NameTooLong(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	_, err := svc.Create(context.Background(), string(long), nil, "")
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got: %v", err)
	}
}

func TestGroup_Create_DuplicateName_Conflict(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	if _, err := svc.Create(context.Background(), "design", nil, ""); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(context.Background(), "design", nil, "")
	if !errors.Is(err, errors.Conflict) {
		t.Fatalf("expected Conflict on duplicate, got %v", err)
	}
}

func TestGroup_Delete_NonEmpty_RefusesByDefault(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, err := svc.Create(context.Background(), "work", nil, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	card := &entity.Card{
		ID:        ulidutil.New(),
		Title:     "c",
		GroupID:   g.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(context.Background(), card); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	err = svc.Delete(context.Background(), g.ID, false)
	if !errors.Is(err, errors.Conflict) {
		t.Fatalf("expected Conflict (non-empty group), got: %v", err)
	}
}

func TestGroup_Delete_NonEmpty_RecursiveAllowed(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "work", nil, "")

	card := &entity.Card{
		ID:        ulidutil.New(),
		Title:     "card",
		GroupID:   g.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(context.Background(), card); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	if err := svc.Delete(context.Background(), g.ID, true); err != nil {
		t.Fatalf("recursive Delete: %v", err)
	}

	if _, err := svc.Get(context.Background(), g.ID); !errors.Is(err, errors.NotFound) {
		t.Fatalf("group still present: %v", err)
	}
	if _, err := r.GetCard(context.Background(), card.ID); err == nil {
		t.Fatalf("card still present after cascade")
	}
}

func TestGroup_Delete_Recursive_CascadesSoftDeletedCards(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	ctx := context.Background()
	g, _ := svc.Create(ctx, "work", nil, "")

	now := time.Now()
	live := &entity.Card{ID: ulidutil.New(), Title: "live", GroupID: g.ID, CreatedAt: now, UpdatedAt: now}
	dead := &entity.Card{ID: ulidutil.New(), Title: "dead", GroupID: g.ID, DeletedAt: &now, CreatedAt: now, UpdatedAt: now}
	_ = r.CreateCard(ctx, live)
	_ = r.CreateCard(ctx, dead)

	if err := svc.Delete(ctx, g.ID, true); err != nil {
		t.Fatalf("recursive Delete: %v", err)
	}
	if _, err := r.GetCard(ctx, live.ID); err == nil {
		t.Fatal("live card still present")
	}
	if _, err := r.GetCard(ctx, dead.ID); err == nil {
		t.Fatal("trashed card still present")
	}
}

func TestGroup_Delete_EmptyGroup_Succeeds(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "work", nil, "")
	if err := svc.Delete(context.Background(), g.ID, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestGroup_Delete_CascadesAnchoredConversations(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	conv := conversation.New(r)
	svc := group.New(r, conv)
	ctx := context.Background()

	g, err := svc.Create(ctx, "work", nil, "")
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}
	kind := "group"
	c, err := conv.Create(ctx, "linked", &kind, &g.ID)
	if err != nil {
		t.Fatalf("Create conversation: %v", err)
	}

	if err := svc.Delete(ctx, g.ID, false); err != nil {
		t.Fatalf("Delete group: %v", err)
	}
	if _, err := conv.Get(ctx, c.ID); !errors.Is(err, errors.NotFound) {
		t.Fatalf("expected anchored conversation gone, got: %v", err)
	}
}

func TestGroup_Delete_Recursive_CascadesConversationsForCards(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	conv := conversation.New(r)
	svc := group.New(r, conv)
	ctx := context.Background()

	g, _ := svc.Create(ctx, "work", nil, "")
	card := &entity.Card{
		ID: ulidutil.New(), Title: "card", GroupID: g.ID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(ctx, card); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	groupKind, cardKind := "group", "card"
	cg, _ := conv.Create(ctx, "g", &groupKind, &g.ID)
	cc, _ := conv.Create(ctx, "c", &cardKind, &card.ID)

	if err := svc.Delete(ctx, g.ID, true); err != nil {
		t.Fatalf("recursive Delete: %v", err)
	}
	for _, id := range []string{cg.ID, cc.ID} {
		if _, err := conv.Get(ctx, id); !errors.Is(err, errors.NotFound) {
			t.Fatalf("expected conversation %s gone, got: %v", id, err)
		}
	}
}

func TestGroup_Create_WithCatalog(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	cat := []entity.LevelEntry{{Weight: 0, Name: "原则"}, {Weight: 1, Name: "决策"}}
	g, err := svc.Create(context.Background(), "design", cat, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(g.LevelCatalog) != 2 || g.LevelCatalog[0].Name != "原则" {
		t.Fatalf("catalog: %+v", g.LevelCatalog)
	}
}

func TestGroup_Update_ReplacesCatalog(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "design", []entity.LevelEntry{{Weight: 0, Name: "原则"}}, "")
	newCat := []entity.LevelEntry{{Weight: 0, Name: "原则"}, {Weight: 1, Name: "细节"}}
	updated, err := svc.Update(context.Background(), g.ID, nil, nil, &newCat, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.LevelCatalog) != 2 || updated.LevelCatalog[1].Name != "细节" {
		t.Fatalf("catalog: %+v", updated.LevelCatalog)
	}
}

func TestGroup_Update_EmptyCatalogClears(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "design", []entity.LevelEntry{{Weight: 0, Name: "原则"}}, "")
	empty := []entity.LevelEntry{}
	updated, _ := svc.Update(context.Background(), g.ID, nil, nil, &empty, nil)
	if len(updated.LevelCatalog) != 0 {
		t.Fatalf("expected empty catalog, got %+v", updated.LevelCatalog)
	}
}

func TestGroup_Create_AssignsIDsToEntries(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	cat := []entity.LevelEntry{{Weight: 0, Name: "原则"}, {Weight: 1, Name: "决策"}}
	g, err := svc.Create(context.Background(), "design", cat, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, e := range g.LevelCatalog {
		if e.ID == "" {
			t.Fatalf("entry %q missing ID", e.Name)
		}
		if err := ulidutil.Parse(e.ID); err != nil {
			t.Fatalf("entry %q ID not ULID: %v", e.Name, err)
		}
	}
}

func TestGroup_Update_RejectsDuplicateName(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "design", nil, "")
	dupes := []entity.LevelEntry{
		{Weight: 0, Name: "x"},
		{Weight: 1, Name: "x"},
	}
	_, err := svc.Update(context.Background(), g.ID, nil, nil, &dupes, nil)
	if !errors.Is(err, errors.Conflict) {
		t.Fatalf("expected Conflict, got %v", err)
	}
}

func TestGroup_Update_RejectsWhitespaceName(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "design", nil, "")
	bad := []entity.LevelEntry{{Weight: 0, Name: "   "}}
	_, err := svc.Update(context.Background(), g.ID, nil, nil, &bad, nil)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest, got %v", err)
	}
}

func TestGroup_Update_RejectsInventedID(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "design", nil, "")
	fake := []entity.LevelEntry{{ID: ulidutil.New(), Weight: 0, Name: "x"}}
	_, err := svc.Update(context.Background(), g.ID, nil, nil, &fake, nil)
	if !errors.Is(err, errors.BadRequest) {
		t.Fatalf("expected BadRequest for invented id, got %v", err)
	}
}

func TestGroup_Update_PreservesIDOnEdit(t *testing.T) {
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)
	g, _ := svc.Create(context.Background(), "design", []entity.LevelEntry{{Weight: 0, Name: "原则"}}, "")
	origID := g.LevelCatalog[0].ID
	edit := []entity.LevelEntry{{ID: origID, Weight: 5, Name: "renamed"}}
	updated, err := svc.Update(context.Background(), g.ID, nil, nil, &edit, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.LevelCatalog) != 1 || updated.LevelCatalog[0].ID != origID {
		t.Fatalf("id changed: %+v", updated.LevelCatalog)
	}
	if updated.LevelCatalog[0].Weight != 5 || updated.LevelCatalog[0].Name != "renamed" {
		t.Fatalf("edit did not apply: %+v", updated.LevelCatalog[0])
	}
}

func TestGroup_Update_CascadesUnfileOnDelete(t *testing.T) {
	ctx := context.Background()
	r := repository.New(testutil.NewDB(t))
	svc := group.New(r, nil)

	// Seed a group with one catalog entry.
	g, _ := svc.Create(ctx, "design", []entity.LevelEntry{{Weight: 0, Name: "原则"}}, "")
	entryID := g.LevelCatalog[0].ID

	// Insert a card pointing at that entry, via the repo directly.
	cardID := ulidutil.New()
	if err := r.CreateCard(ctx, &entity.Card{
		ID:           cardID,
		Title:        "c",
		GroupID:      g.ID,
		LevelEntryID: &entryID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	// Update the catalog to drop the entry.
	empty := []entity.LevelEntry{}
	if _, err := svc.Update(ctx, g.ID, nil, nil, &empty, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Card's LevelEntryID should be nil now.
	card, err := r.GetCard(ctx, cardID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if card.LevelEntryID != nil {
		t.Fatalf("expected LevelEntryID nil after cascade, got %v", *card.LevelEntryID)
	}
}
