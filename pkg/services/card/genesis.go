package card

import (
	"context"
	"strings"

	"github.com/xhanio/zen/pkg/types/entity"
)

// maxAncestryDepth caps how many ancestor titles the auto-generated genesis
// walks upward. Keeps the string tractable for deeply-nested trees and puts
// a hard limit on repo reads.
const maxAncestryDepth = 5

// ancestryTitleChain returns the ancestor titles of parent, root-first:
//
//	[root_title, grandparent_title, parent_title]
//
// Excludes the current card (the one being derived). Walks up via
// parent_card_id, capped at maxAncestryDepth levels. Errors on a missing
// parent link are swallowed silently so genesis generation never blocks
// the surrounding transaction.
func (m *manager) ancestryTitleChain(ctx context.Context, parent *entity.Card) []string {
	if parent == nil {
		return nil
	}
	titles := []string{strings.TrimSpace(parent.Title)}
	current := parent
	for i := 0; i < maxAncestryDepth && current.ParentCardID != nil; i++ {
		next, err := m.repo.GetCard(ctx, *current.ParentCardID)
		if err != nil {
			break
		}
		titles = append([]string{strings.TrimSpace(next.Title)}, titles...)
		current = next
	}
	return titles
}

// defaultDecomposeGenesis builds the auto-genesis for a child of parent
// using a title breadcrumb instead of raw IDs. Falls back to the parent
// title alone (and finally to a plain placeholder) if the chain is empty.
func (m *manager) defaultDecomposeGenesis(ctx context.Context, parent *entity.Card) string {
	chain := m.ancestryTitleChain(ctx, parent)
	if len(chain) == 0 {
		return "Decomposed from an untitled card"
	}
	return "Decomposed from " + strings.Join(chain, " - ")
}

// defaultComposeGenesis builds the auto-genesis for a compose target using
// the source cards' titles instead of their IDs. Sources are listed in
// caller-provided order.
func defaultComposeGenesis(sources []*entity.Card) string {
	titles := make([]string, 0, len(sources))
	for _, s := range sources {
		t := strings.TrimSpace(s.Title)
		if t == "" {
			t = "(untitled)"
		}
		titles = append(titles, t)
	}
	return "Composed from " + strings.Join(titles, ", ")
}
