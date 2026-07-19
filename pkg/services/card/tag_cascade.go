package card

import (
	"context"
	"strings"

	"github.com/xhanio/errors"
)

// cascadeAttachTags walks the live subtree rooted at parentID (excluding
// parentID itself — the caller has already attached there) and ensures each
// tag NAME on every descendant, IN THAT DESCENDANT'S OWN GROUP. Working by
// name (not tag id) keeps the tag.group_id == card.group_id invariant even
// when a descendant lives in a different group than the container.
//
// Idempotent per descendant: we skip names the descendant already has, so the
// primary-key constraint on (card_id, tag_id) never fires. Removals are NOT
// cascaded — a container that drops a tag leaves descendants with what they had.
func (m *manager) cascadeAttachTags(ctx context.Context, parentID string, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}
	// BFS over live descendants.
	queue := []string{parentID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		children, err := m.repo.ListChildren(ctx, id, false /* includeTrashed */)
		if err != nil {
			return errors.Wrap(err)
		}
		for _, child := range children {
			existing, err := m.repo.ListTagsForCard(ctx, child.ID)
			if err != nil {
				return errors.Wrap(err)
			}
			have := make(map[string]struct{}, len(existing))
			for _, name := range existing {
				have[name] = struct{}{}
			}
			for _, name := range tagNames {
				if _, already := have[strings.ToLower(strings.TrimSpace(name))]; already {
					continue
				}
				tag, err := m.tags.EnsureByName(ctx, child.GroupID, name)
				if err != nil {
					return errors.Wrap(err)
				}
				if err := m.repo.AttachTag(ctx, child.ID, tag.ID); err != nil {
					return errors.Wrap(err)
				}
			}
			queue = append(queue, child.ID)
		}
	}
	return nil
}
