package card_test

import (
	"context"
	"testing"

	"github.com/xhanio/zen/pkg/types/api"
)

// The auto-generated genesis for decompose children must use a title
// breadcrumb rooted at the top-most ancestor, joined with " - ". IDs
// must NEVER appear.
func TestDecompose_DefaultGenesis_UsesTitleChainNotIDs(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()

	// Build a 3-level ancestry via decompose:
	//   grandparent "GP"  →  parent "PA"  →  (decompose here, children under PA)
	grandparent, err := svc.Create(ctx, "GP", "gp body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create GP: %v", err)
	}
	// Decompose grandparent into one child that will become PA.
	respGP, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: grandparent.ID,
		Cards: []api.CardSpec{{
			Title: "PA", Content: "pa body",
		}},
	})
	if err != nil {
		t.Fatalf("Decompose GP: %v", err)
	}
	parent := respGP.Cards[0]

	// Now decompose PA. Its children should get genesis:
	//   "Decomposed from GP - PA"
	respPA, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards: []api.CardSpec{{
			Title: "child", Content: "child body",
		}},
	})
	if err != nil {
		t.Fatalf("Decompose PA: %v", err)
	}
	got := respPA.Cards[0].Genesis
	want := "Decomposed from GP - PA"
	if got != want {
		t.Fatalf("child genesis: want %q, got %q", want, got)
	}
	// Sanity: no ULID substring leaked into genesis.
	if len(got) > 0 && (containsULIDLike(got, grandparent.ID) || containsULIDLike(got, parent.ID)) {
		t.Fatalf("genesis leaks an ID: %q", got)
	}
}

func TestDecompose_DefaultGenesis_TopLevelParent_UsesParentTitleAlone(t *testing.T) {
	svc, _, groupID := newCardCtx(t)
	ctx := context.Background()
	parent, err := svc.Create(ctx, "Solo parent", "body", groupID, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	resp, err := svc.Decompose(ctx, api.DecomposeRequest{
		ParentCardID: parent.ID,
		Cards: []api.CardSpec{{Title: "kid", Content: "body"}},
	})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}
	got := resp.Cards[0].Genesis
	want := "Decomposed from Solo parent"
	if got != want {
		t.Fatalf("genesis: want %q, got %q", want, got)
	}
}

func containsULIDLike(s, id string) bool {
	if id == "" || len(id) < 6 {
		return false
	}
	// A ULID substring of at least 8 chars is enough to be a leak signal.
	return len(s) >= len(id) && indexOf(s, id) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
