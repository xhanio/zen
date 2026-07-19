package card_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/card"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// scoreCtx holds the pieces a score test needs.
type scoreCtx struct {
	svc          model.Card
	repo         repository.Repository
	groupID      string
	catalog      []entity.LevelEntry
	principleLE  string // weight 0
	decisionLE   string // weight 1
	patternLE    string // weight 2
	detailLE     string // weight 3
	containerID  string
	sectionIDs   []string
	topLeafID    string
	nestedLeafID string // child of container — for "nested returns nil" test
}

// newScoreCtx builds a fresh in-memory service with:
//   - one 4-tier catalog (weights 0..3)
//   - one top-level leaf card
//   - one top-level container with 4 leaf sections (one per level)
func newScoreCtx(t *testing.T) *scoreCtx {
	t.Helper()
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	catalog := []entity.LevelEntry{
		{ID: ulidutil.New(), Weight: 0, Name: "Principle"},
		{ID: ulidutil.New(), Weight: 1, Name: "Decision"},
		{ID: ulidutil.New(), Weight: 2, Name: "Pattern"},
		{ID: ulidutil.New(), Weight: 3, Name: "Detail"},
	}
	g := &entity.Group{ID: ulidutil.New(), Name: "g", LevelCatalog: catalog, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	leaf := &entity.Card{
		ID: ulidutil.New(), Title: "top-level leaf", Content: "body",
		GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, leaf); err != nil {
		t.Fatalf("CreateCard leaf: %v", err)
	}
	container := &entity.Card{
		ID: ulidutil.New(), Title: "container", Content: "",
		GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, container); err != nil {
		t.Fatalf("CreateCard container: %v", err)
	}
	pid := container.ID
	var sectionIDs []string
	for i, lvl := range catalog {
		le := lvl.ID
		c := &entity.Card{
			ID: ulidutil.New(), Title: lvl.Name, Content: "section body",
			GroupID: g.ID, ParentCardID: &pid, Position: i, LevelEntryID: &le,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := repo.CreateCard(ctx, c); err != nil {
			t.Fatalf("CreateCard section %d: %v", i, err)
		}
		sectionIDs = append(sectionIDs, c.ID)
	}
	return &scoreCtx{
		svc:          card.New(repo, tag.New(repo), nil),
		repo:         repo,
		groupID:      g.ID,
		catalog:      catalog,
		principleLE:  catalog[0].ID,
		decisionLE:   catalog[1].ID,
		patternLE:    catalog[2].ID,
		detailLE:     catalog[3].ID,
		containerID:  container.ID,
		sectionIDs:   sectionIDs,
		topLeafID:    leaf.ID,
		nestedLeafID: sectionIDs[0],
	}
}

func TestScore_NestedCardReturnsNil(t *testing.T) {
	c := newScoreCtx(t)
	got, err := c.svc.Get(context.Background(), c.nestedLeafID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore != nil {
		t.Fatalf("expected nil ReviewScore on nested card, got %v", *got.ReviewScore)
	}
}

func TestScore_TopLevelLeaf_LGTM_ReturnsZero(t *testing.T) {
	c := newScoreCtx(t)
	got, err := c.svc.Get(context.Background(), c.topLeafID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore == nil {
		t.Fatalf("expected non-nil ReviewScore on top-level leaf")
	}
	if *got.ReviewScore != 0.0 {
		t.Fatalf("expected 0.0, got %v", *got.ReviewScore)
	}
}

func TestScore_TopLevelLeaf_Digested_ReturnsFifty(t *testing.T) {
	c := newScoreCtx(t)
	if _, err := c.svc.Review(context.Background(), c.topLeafID, "DIGESTED"); err != nil {
		t.Fatalf("Review: %v", err)
	}
	got, err := c.svc.Get(context.Background(), c.topLeafID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore == nil || *got.ReviewScore != 50.0 {
		t.Fatalf("expected 50.0, got %v", got.ReviewScore)
	}
}

func TestScore_TopLevelLeaf_Grilled_ReturnsHundred(t *testing.T) {
	c := newScoreCtx(t)
	if _, err := c.svc.Review(context.Background(), c.topLeafID, "GRILLED"); err != nil {
		t.Fatalf("Review: %v", err)
	}
	got, err := c.svc.Get(context.Background(), c.topLeafID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore == nil || *got.ReviewScore != 100.0 {
		t.Fatalf("expected 100.0, got %v", got.ReviewScore)
	}
}

func TestScore_LeafOnlyContainer_AllLGTM_ReturnsZero(t *testing.T) {
	c := newScoreCtx(t)
	got, err := c.svc.Get(context.Background(), c.containerID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore == nil || *got.ReviewScore != 0.0 {
		t.Fatalf("expected 0.0, got %v", got.ReviewScore)
	}
}

func TestScore_LeafOnlyContainer_AllGrilled_ReturnsHundred(t *testing.T) {
	c := newScoreCtx(t)
	for _, id := range c.sectionIDs {
		if _, err := c.svc.Review(context.Background(), id, "GRILLED"); err != nil {
			t.Fatalf("Review %s: %v", id, err)
		}
	}
	got, err := c.svc.Get(context.Background(), c.containerID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore == nil || *got.ReviewScore != 100.0 {
		t.Fatalf("expected 100.0, got %v", got.ReviewScore)
	}
}

func TestScore_EmptyContainer_ReturnsNil(t *testing.T) {
	// Delete all live children of the container; content is already empty.
	c := newScoreCtx(t)
	ctx := context.Background()
	for _, id := range c.sectionIDs {
		if err := c.svc.Delete(ctx, id, false); err != nil {
			t.Fatalf("Delete %s: %v", id, err)
		}
	}
	got, err := c.svc.Get(ctx, c.containerID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore != nil {
		t.Fatalf("expected nil ReviewScore on empty container, got %v", *got.ReviewScore)
	}
}

func TestScore_AllTrashed_ReturnsNil(t *testing.T) {
	c := newScoreCtx(t)
	ctx := context.Background()
	for _, id := range c.sectionIDs {
		if err := c.svc.Delete(ctx, id, false); err != nil {
			t.Fatalf("Delete %s: %v", id, err)
		}
	}
	got, err := c.svc.Get(ctx, c.containerID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore != nil {
		t.Fatalf("expected nil when all children trashed, got %v", *got.ReviewScore)
	}
}

func TestScore_AbstractCardStillCountsAtHalfCredit(t *testing.T) {
	// Container with 2 children: GRILLED 原则 (raw weight 0 → nw 0.5) +
	// LGTM 细节 (raw weight 3 → nw 1.0).
	// numerator   = 0.5*1.0 + 1.0*0.0 = 0.5
	// denominator = 0.5 + 1.0         = 1.5
	// score       = 0.5/1.5 * 100     = 33.333... → 33.3
	c := newScoreCtx(t)
	ctx := context.Background()
	// Trash decision + pattern, keep principle (idx 0) + detail (idx 3).
	if err := c.svc.Delete(ctx, c.sectionIDs[1], false); err != nil {
		t.Fatalf("Delete decision: %v", err)
	}
	if err := c.svc.Delete(ctx, c.sectionIDs[2], false); err != nil {
		t.Fatalf("Delete pattern: %v", err)
	}
	if _, err := c.svc.Review(ctx, c.sectionIDs[0], "GRILLED"); err != nil {
		t.Fatalf("Review principle: %v", err)
	}
	got, err := c.svc.Get(ctx, c.containerID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore == nil {
		t.Fatalf("expected non-nil score")
	}
	if math.Abs(*got.ReviewScore-33.3) > 0.05 {
		t.Fatalf("expected ~33.3, got %v", *got.ReviewScore)
	}
}

func TestScore_SingleTierCatalog_FlatWeighting(t *testing.T) {
	// Group with a single catalog entry. Container with 2 children at
	// that level. One GRILLED, one LGTM. nw = 1.0 for both.
	// score = (1.0*1.0 + 1.0*0.0) / (1.0+1.0) * 100 = 50.0
	ctx := context.Background()
	repo := repository.New(testutil.NewDB(t))
	catalog := []entity.LevelEntry{{ID: ulidutil.New(), Weight: 1, Name: "Flat"}}
	g := &entity.Group{ID: ulidutil.New(), Name: "flat", LevelCatalog: catalog, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	container := &entity.Card{
		ID: ulidutil.New(), Title: "flat container", Content: "",
		GroupID: g.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, container); err != nil {
		t.Fatalf("CreateCard container: %v", err)
	}
	pid := container.ID
	le := catalog[0].ID
	child1 := &entity.Card{
		ID: ulidutil.New(), Title: "A", Content: "body",
		GroupID: g.ID, ParentCardID: &pid, Position: 0, LevelEntryID: &le,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	child2 := &entity.Card{
		ID: ulidutil.New(), Title: "B", Content: "body",
		GroupID: g.ID, ParentCardID: &pid, Position: 1, LevelEntryID: &le,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.CreateCard(ctx, child1); err != nil {
		t.Fatalf("CreateCard child1: %v", err)
	}
	if err := repo.CreateCard(ctx, child2); err != nil {
		t.Fatalf("CreateCard child2: %v", err)
	}
	svc := card.New(repo, tag.New(repo), nil)
	if _, err := svc.Review(ctx, child1.ID, "GRILLED"); err != nil {
		t.Fatalf("Review child1: %v", err)
	}
	got, err := svc.Get(ctx, container.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ReviewScore == nil || *got.ReviewScore != 50.0 {
		t.Fatalf("expected 50.0, got %v", got.ReviewScore)
	}
}
