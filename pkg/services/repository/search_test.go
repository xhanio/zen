//go:build sqlite_fts5

package repository_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

func seedSearchableContent(t *testing.T, r repository.Repository) (groupID, cardID string) {
	t.Helper()
	ctx := context.Background()
	g := &entity.Group{ID: ulidutil.New(), Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	groupID = g.ID
	c := &entity.Card{
		ID: ulidutil.New(), Title: "Hover flight",
		Content:   "Hummingbirds can hover by flapping their wings figure-eight style.",
		GroupID:   groupID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	cardID = c.ID
	return
}

func TestSearchCards_MatchesContent(t *testing.T) {
	r := repository.New(testutil.NewDBWithFTS5(t))
	_, cardID := seedSearchableContent(t, r)
	hits, err := r.SearchCards(context.Background(), "hover", 10)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	found := false
	for _, h := range hits {
		if h.ID == cardID {
			found = true
			if !strings.Contains(h.Snippet, "<mark>") {
				t.Fatalf("snippet missing <mark>: %q", h.Snippet)
			}
		}
	}
	if !found {
		t.Fatalf("expected card %q in hits, got %+v", cardID, hits)
	}
}

func TestSearchCards_NoMatch(t *testing.T) {
	r := repository.New(testutil.NewDBWithFTS5(t))
	seedSearchableContent(t, r)
	hits, err := r.SearchCards(context.Background(), "nonexistent", 10)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("want 0 hits for unknown term, got %d", len(hits))
	}
}

func TestSearchCards_LimitHonored(t *testing.T) {
	r := repository.New(testutil.NewDBWithFTS5(t))
	groupID := ulidutil.New()
	g := &entity.Group{ID: groupID, Name: "work", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = r.CreateGroup(context.Background(), g)
	for i := 0; i < 5; i++ {
		c := &entity.Card{
			ID: ulidutil.New(), Title: "hover card",
			Content:   "hover hover hover",
			GroupID:   groupID,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := r.CreateCard(context.Background(), c); err != nil {
			t.Fatalf("CreateCard: %v", err)
		}
	}
	hits, err := r.SearchCards(context.Background(), "hover", 3)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("limit=3 but got %d hits", len(hits))
	}
}

func TestCard_HtmlFormat_SearchHintStripsTags(t *testing.T) {
	r := repository.New(testutil.NewDBWithFTS5(t))
	groupID := mustCreateGroupRow(t, r, "viz")
	ctx := context.Background()

	c := &entity.Card{
		ID: ulidutil.New(), Title: "chart",
		Content: `<svg><text>banana</text></svg><div>apple</div>`,
		Format:  "html",
		GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	hits, err := r.SearchCards(ctx, "apple", 10)
	if err != nil {
		t.Fatalf("Search apple: %v", err)
	}
	if !hasCardID(hits, c.ID) {
		t.Fatalf("expected apple search to hit card %s; got %+v", c.ID, hits)
	}
	hits, err = r.SearchCards(ctx, "banana", 10)
	if err != nil {
		t.Fatalf("Search banana: %v", err)
	}
	if !hasCardID(hits, c.ID) {
		t.Fatalf("expected banana (svg text) search to hit card %s; got %+v", c.ID, hits)
	}

	hits, err = r.SearchCards(ctx, "svg", 10)
	if err != nil {
		t.Fatalf("Search svg: %v", err)
	}
	if hasCardID(hits, c.ID) {
		t.Fatalf("svg tag name should not match HTML card %s; got %+v", c.ID, hits)
	}
}

// The snippet must be built from the stripped search_hint that was indexed —
// not the raw HTML content. With an external-content FTS5 table, snippet()
// reconstructs from the cards column whose name matches the FTS column, so if
// the index is fed from search_hint but the FTS column is named "content",
// snippet() returns raw HTML with <mark> offsets that don't line up.
func TestSearchCards_SnippetIsStrippedHint_NotRawHTML(t *testing.T) {
	r := repository.New(testutil.NewDBWithFTS5(t))
	groupID := mustCreateGroupRow(t, r, "viz")
	ctx := context.Background()

	c := &entity.Card{
		ID: ulidutil.New(), Title: "chart",
		Content: `<style>.x{color:crimson}</style><div class="x">the apple is ripe</div>`,
		Format:  "html",
		GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(ctx, c); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	hits, err := r.SearchCards(ctx, "apple", 10)
	if err != nil {
		t.Fatalf("Search apple: %v", err)
	}
	var snip string
	for _, h := range hits {
		if h.ID == c.ID {
			snip = h.Snippet
		}
	}
	if snip == "" {
		t.Fatalf("apple search did not hit card %s; hits=%+v", c.ID, hits)
	}
	if strings.Contains(snip, "<div") || strings.Contains(snip, "<style") || strings.Contains(snip, "crimson") {
		t.Fatalf("snippet leaks raw HTML/CSS (reading content, not search_hint): %q", snip)
	}
	if !strings.Contains(snip, "<mark>apple</mark>") {
		t.Fatalf("snippet must highlight the matched term; got %q", snip)
	}
}

// A document (a decomposed container) has an empty body, so it matches a query
// only via its title. A title match is a strong relevance signal, so such a card
// must surface within the result limit even when many cards match the same term
// in their bodies. Guards the bm25 title weighting in cardSearchSQL.
func TestSearchCards_TitleMatchRanksWithinLimit(t *testing.T) {
	r := repository.New(testutil.NewDBWithFTS5(t))
	groupID := mustCreateGroupRow(t, r, "docs")
	ctx := context.Background()

	doc := &entity.Card{
		ID: ulidutil.New(), Title: "Aardvark design doc",
		Content: "", Format: "html",
		GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := r.CreateCard(ctx, doc); err != nil {
		t.Fatalf("CreateCard doc: %v", err)
	}
	// 25 cards matching "aardvark" in their bodies — more than the limit, so
	// without title weighting the title-only doc is crowded out past rank 20.
	for i := 0; i < 25; i++ {
		c := &entity.Card{
			ID: ulidutil.New(), Title: "section",
			Content: strings.Repeat("aardvark ", 8),
			GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := r.CreateCard(ctx, c); err != nil {
			t.Fatalf("CreateCard body %d: %v", i, err)
		}
	}

	hits, err := r.SearchCards(ctx, "aardvark", 20)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if !hasCardID(hits, doc.ID) {
		t.Fatalf("title-only document should rank within the top 20; body matches crowded it out")
	}
}

func TestSearchCards_TitlePathIsRootFirstAncestors(t *testing.T) {
	r := repository.New(testutil.NewDBWithFTS5(t))
	groupID := mustCreateGroupRow(t, r, "docs")
	ctx := context.Background()

	mk := func(title, content string, parent *string) *entity.Card {
		c := &entity.Card{
			ID: ulidutil.New(), Title: title, Content: content, ParentCardID: parent,
			GroupID: groupID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := r.CreateCard(ctx, c); err != nil {
			t.Fatalf("CreateCard %q: %v", title, err)
		}
		return c
	}
	root := mk("Roadmap", "root filler", nil)
	mid := mk("v0.14 Design", "mid filler", &root.ID)
	leaf := mk("Conversation UX", "hummingbird unique term", &mid.ID)

	hits, err := r.SearchCards(ctx, "hummingbird", 10)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	var hit *entity.SearchHit
	for _, h := range hits {
		if h.ID == leaf.ID {
			hit = h
		}
	}
	if hit == nil {
		t.Fatalf("leaf card not in hits")
	}
	want := []string{"Roadmap", "v0.14 Design"} // root-first, excludes the card itself
	if !reflect.DeepEqual(hit.TitlePath, want) {
		t.Fatalf("TitlePath = %v, want %v", hit.TitlePath, want)
	}

	// A top-level card has no ancestors → empty path.
	rootHits, err := r.SearchCards(ctx, "root", 10)
	if err != nil {
		t.Fatalf("SearchCards root: %v", err)
	}
	for _, h := range rootHits {
		if h.ID == root.ID && len(h.TitlePath) != 0 {
			t.Fatalf("top-level card should have empty TitlePath, got %v", h.TitlePath)
		}
	}
}

func hasCardID(hits []*entity.SearchHit, id string) bool {
	for _, h := range hits {
		if h.ID == id {
			return true
		}
	}
	return false
}
