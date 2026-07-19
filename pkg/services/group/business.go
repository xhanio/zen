package group

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

const maxGroupNameLen = 100

// sortByWeight returns a stably-sorted copy of the catalog, ascending by
// Weight. It does not dedupe or validate — callers do that via
// validateAndAssignIDs before persisting.
func sortByWeight(in []entity.LevelEntry) []entity.LevelEntry {
	if in == nil {
		return []entity.LevelEntry{}
	}
	out := make([]entity.LevelEntry, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Weight < out[j].Weight })
	return out
}

// validateAndAssignIDs enforces name uniqueness and assigns fresh ULIDs
// to new entries (those with empty ID). Existing entries must reference
// an ID present in the existing catalog. Returns a shallow-copied slice
// safe to store back on the group.
func validateAndAssignIDs(existing []entity.LevelEntry, next []entity.LevelEntry) ([]entity.LevelEntry, error) {
	existingByID := make(map[string]bool, len(existing))
	for _, e := range existing {
		existingByID[e.ID] = true
	}
	seenNames := make(map[string]bool, len(next))
	out := make([]entity.LevelEntry, len(next))
	for i, e := range next {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			return nil, errors.BadRequest.Newf("catalog entry name cannot be empty")
		}
		if seenNames[name] {
			return nil, errors.Conflict.Newf("catalog entry name %q is not unique", name)
		}
		seenNames[name] = true
		if e.ID != "" && !existingByID[e.ID] {
			return nil, errors.BadRequest.Newf("catalog entry id %q not found in existing catalog", e.ID)
		}
		if e.ID == "" {
			e.ID = ulidutil.New()
		}
		e.Name = name
		out[i] = e
	}
	return out, nil
}

// deletedEntryIDs returns the ids present in old but absent in new — the
// ones that need the unfile cascade.
func deletedEntryIDs(oldCatalog, newCatalog []entity.LevelEntry) []string {
	newByID := make(map[string]bool, len(newCatalog))
	for _, e := range newCatalog {
		newByID[e.ID] = true
	}
	var deleted []string
	for _, e := range oldCatalog {
		if !newByID[e.ID] {
			deleted = append(deleted, e.ID)
		}
	}
	return deleted
}

func (m *manager) Create(ctx context.Context, name string, levelCatalog []entity.LevelEntry, rule string) (*entity.Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.BadRequest.Newf("group name is required")
	}
	if len(name) > maxGroupNameLen {
		return nil, errors.BadRequest.Newf("group name must be %d chars or fewer", maxGroupNameLen)
	}
	validated, err := validateAndAssignIDs(nil, levelCatalog)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	now := time.Now()
	g := &entity.Group{
		ID:           ulidutil.New(),
		Name:         name,
		Rule:         rule,
		LevelCatalog: sortByWeight(validated),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := m.repo.CreateGroup(ctx, g); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, errors.Conflict.Newf("a group named %q already exists", name)
		}
		return nil, errors.Wrap(err)
	}
	return g, nil
}

func (m *manager) Get(ctx context.Context, id string) (*entity.Group, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}
	return m.repo.GetGroup(ctx, id)
}

func (m *manager) List(ctx context.Context) ([]*entity.Group, error) {
	return m.repo.ListGroups(ctx)
}

func (m *manager) Update(ctx context.Context, id string, name *string, position *int, levelCatalog *[]entity.LevelEntry, rule *string) (*entity.Group, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}
	var updated *entity.Group
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		g, err := m.repo.GetGroup(txCtx, id)
		if err != nil {
			return errors.Wrap(err)
		}
		if name != nil {
			trimmed := strings.TrimSpace(*name)
			if trimmed == "" {
				return errors.BadRequest.Newf("group name cannot be empty")
			}
			if len(trimmed) > maxGroupNameLen {
				return errors.BadRequest.Newf("group name must be %d chars or fewer", maxGroupNameLen)
			}
			g.Name = trimmed
		}
		if position != nil {
			g.Position = *position
		}
		if rule != nil {
			g.Rule = *rule
		}
		if levelCatalog != nil {
			next, err := validateAndAssignIDs(g.LevelCatalog, *levelCatalog)
			if err != nil {
				return errors.Wrap(err)
			}
			for _, entryID := range deletedEntryIDs(g.LevelCatalog, next) {
				if _, err := m.repo.UnfileCardsForEntry(txCtx, id, entryID); err != nil {
					return errors.Wrap(err)
				}
			}
			g.LevelCatalog = sortByWeight(next)
		}
		if err := m.repo.UpdateGroup(txCtx, g); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return errors.Conflict.Newf("a group named %q already exists", g.Name)
			}
			return errors.Wrap(err)
		}
		updated = g
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return updated, nil
}

func (m *manager) Delete(ctx context.Context, id string, recursive bool) error {
	if err := ulidutil.Parse(id); err != nil {
		return errors.Wrap(err)
	}
	if _, err := m.repo.GetGroup(ctx, id); err != nil {
		return errors.Wrap(err)
	}
	hasContent, err := m.repo.GroupHasContent(ctx, id)
	if err != nil {
		return errors.Wrap(err)
	}
	if hasContent && !recursive {
		return errors.Conflict.Newf("group %q is not empty; pass recursive=true to delete contents", id)
	}
	if recursive {
		return m.deleteRecursive(ctx, id)
	}
	return m.repo.Transaction(ctx, func(txCtx context.Context) error {
		if m.conv != nil {
			if err := m.conv.DeleteByAnchor(txCtx, "group", id); err != nil {
				return errors.Wrap(err)
			}
		}
		return m.repo.DeleteGroup(txCtx, id)
	})
}

func (m *manager) deleteRecursive(ctx context.Context, id string) error {
	return m.repo.Transaction(ctx, func(txCtx context.Context) error {
		return m.deleteRecursiveTx(txCtx, id)
	})
}

func (m *manager) deleteRecursiveTx(ctx context.Context, id string) error {
	// v0.6.0: cards are the only contained entity; cascade includes
	// soft-deleted cards too (include_trashed=true).
	if m.conv != nil {
		cards, err := m.repo.ListCards(ctx, &id, true)
		if err != nil {
			return errors.Wrap(err)
		}
		for _, c := range cards {
			if err := m.conv.DeleteByAnchor(ctx, "card", c.ID); err != nil {
				return errors.Wrap(err)
			}
		}
		if err := m.conv.DeleteByAnchor(ctx, "group", id); err != nil {
			return errors.Wrap(err)
		}
	}
	if err := m.repo.DeleteCardsByGroup(ctx, id); err != nil {
		return errors.Wrap(err)
	}
	if err := m.repo.DeleteGroup(ctx, id); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
