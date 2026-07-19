package card

import (
	"context"
	"math"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
)

var gradeCredit = map[string]float64{
	GradeLGTM:     0.0,
	GradeDigested: 0.5,
	GradeGrilled:  1.0,
}

// normalizedWeight maps a raw catalog weight to [0.5, 1.0] scaled across
// the catalog's own min/max. A single-tier catalog (or catalog where all
// entries share the same weight) degenerates to 1.0.
func normalizedWeight(w float64, catalog []entity.LevelEntry) float64 {
	if len(catalog) == 0 {
		return 1.0
	}
	wMin, wMax := catalog[0].Weight, catalog[0].Weight
	for _, e := range catalog[1:] {
		if e.Weight < wMin {
			wMin = e.Weight
		}
		if e.Weight > wMax {
			wMax = e.Weight
		}
	}
	if wMax == wMin {
		return 1.0
	}
	return 0.5 + 0.5*(w-wMin)/(wMax-wMin)
}

// catalogWeight returns the raw weight for the given level entry ID within
// the group's catalog. Returns (0, false) if the ID isn't in the catalog.
func catalogWeight(catalog []entity.LevelEntry, entryID string) (float64, bool) {
	for _, e := range catalog {
		if e.ID == entryID {
			return e.Weight, true
		}
	}
	return 0, false
}

// computeReviewScore returns the entity's review_score:
//   - nil for any card with parent_card_id != nil (nested; parent surfaces it)
//   - CREDIT[grade] * 100 for a top-level leaf (no children AND has content)
//   - nil for an empty container (no live scoreable children)
//   - weighted recursive walk for a top-level container
func (m *manager) computeReviewScore(ctx context.Context, card *entity.Card) (*float64, error) {
	if card.ParentCardID != nil {
		return nil, nil
	}
	return m.scoreForCard(ctx, card)
}

// scoreForCard is the internal recursive helper. Called for the top-level
// card AND for every nested container we recurse into.
func (m *manager) scoreForCard(ctx context.Context, card *entity.Card) (*float64, error) {
	children, err := m.repo.ListChildren(ctx, card.ID, false /* includeTrashed */)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	live := make([]*entity.Card, 0, len(children))
	for _, c := range children {
		if c.DeletedAt == nil && c.LevelEntryID != nil {
			live = append(live, c)
		}
	}
	if len(live) == 0 {
		// Top-level leaf: score by own grade × 100.
		// Empty container (was decomposed, content cleared): nil.
		if card.Content == "" {
			return nil, nil
		}
		v := gradeCredit[card.ReviewGrade] * 100.0
		return &v, nil
	}
	group, err := m.repo.GetGroup(ctx, card.GroupID)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	var totalWeight, totalContrib float64
	for _, c := range live {
		rawW, ok := catalogWeight(group.LevelCatalog, *c.LevelEntryID)
		if !ok {
			continue
		}
		nw := normalizedWeight(rawW, group.LevelCatalog)

		// Peek at children to decide leaf vs nested container.
		grandchildren, err := m.repo.ListChildren(ctx, c.ID, false)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		var contrib float64
		if len(grandchildren) == 0 {
			// Leaf child — score its own grade.
			contrib = nw * gradeCredit[c.ReviewGrade]
		} else {
			// Nested container — recurse.
			childScore, err := m.scoreForCard(ctx, c)
			if err != nil {
				return nil, errors.Wrap(err)
			}
			if childScore == nil {
				continue
			}
			contrib = nw * (*childScore / 100.0)
		}
		totalWeight += nw
		totalContrib += contrib
	}
	if totalWeight == 0 {
		return nil, nil
	}
	v := math.Round(totalContrib/totalWeight*100.0*10) / 10
	return &v, nil
}
