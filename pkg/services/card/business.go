package card

import (
	"context"
	"strings"
	"time"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/htmltext"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

const (
	maxTitleLen   = 200
	maxContentLen = 1 << 20 // 1 MB
	maxSummaryLen = 500
)

// validateSummary trims the summary and enforces the length cap. Empty is
// fine — callers fall back to the content-derived preview at display time.
// The frontend nudges users toward ~30 words for tile display, but that's a
// soft UI hint, not a server rule.
func validateSummary(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) > maxSummaryLen {
		return "", errors.BadRequest.Newf("card summary must be %d chars or fewer", maxSummaryLen)
	}
	return s, nil
}

func (m *manager) Create(ctx context.Context, title, content, groupID string, tagNames []string, parentCardID, sourceConversationID *string, format *string, levelEntryID *string, genesis *string, reference *api.ReferenceSpec, summary *string) (*entity.Card, error) {
	if reference != nil {
		if parentCardID == nil {
			return nil, errors.BadRequest.Newf("reference requires parent_card_id")
		}
		if reference.SelectionText == "" {
			return nil, errors.BadRequest.Newf("reference.selection_text is required")
		}
		if len(reference.SelectionText) > 5000 {
			return nil, errors.BadRequest.Newf("reference.selection_text exceeds 5000 chars")
		}
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.BadRequest.Newf("card title is required")
	}
	if len(title) > maxTitleLen {
		return nil, errors.BadRequest.Newf("card title must be %d chars or fewer", maxTitleLen)
	}
	if len(content) > maxContentLen {
		return nil, errors.BadRequest.Newf("card content exceeds 1 MB limit")
	}
	f := formatMarkdown
	if format != nil {
		f = *format
	}
	if err := validateFormat(f); err != nil {
		return nil, err
	}
	if err := ulidutil.Parse(groupID); err != nil {
		return nil, errors.Wrap(err)
	}
	if _, err := m.repo.GetGroup(ctx, groupID); err != nil {
		return nil, errors.Wrap(err)
	}
	if parentCardID != nil {
		if err := ulidutil.Parse(*parentCardID); err != nil {
			return nil, errors.Wrap(err)
		}
		if _, err := m.repo.GetCard(ctx, *parentCardID); err != nil {
			return nil, errors.Wrap(err)
		}
	}
	if sourceConversationID != nil {
		if err := ulidutil.Parse(*sourceConversationID); err != nil {
			return nil, errors.Wrap(err)
		}
		if _, err := m.repo.GetConversation(ctx, *sourceConversationID); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	g := ""
	if genesis != nil {
		g = *genesis
	}
	sm := ""
	if summary != nil {
		v, err := validateSummary(*summary)
		if err != nil {
			return nil, err
		}
		sm = v
	}

	now := time.Now()
	card := &entity.Card{
		ID:                   ulidutil.New(),
		Title:                title,
		Summary:              sm,
		Content:              content,
		Format:               f,
		Genesis:              g,
		GroupID:              groupID,
		Position:             0,
		Tags:                 []string{},
		ParentCardID:         parentCardID,
		SourceConversationID: sourceConversationID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		if levelEntryID != nil {
			grp, err := m.repo.GetGroup(txCtx, groupID)
			if err != nil {
				return errors.Wrap(err)
			}
			if err := ensureCatalogEntry(grp.LevelCatalog, *levelEntryID); err != nil {
				return errors.Wrap(err)
			}
			card.LevelEntryID = levelEntryID
		}
		if err := m.repo.CreateCard(txCtx, card); err != nil {
			return errors.Wrap(err)
		}
		for _, name := range tagNames {
			tag, err := m.tags.EnsureByName(txCtx, card.GroupID, name)
			if err != nil {
				return errors.Wrap(err)
			}
			if err := m.repo.AttachTag(txCtx, card.ID, tag.ID); err != nil {
				return errors.Wrap(err)
			}
			card.Tags = append(card.Tags, tag.Name)
		}
		if reference != nil {
			ref := &entity.Reference{
				ID:             ulidutil.New(),
				SourceCardID:   *parentCardID,
				DerivedCardID:  card.ID,
				ConversationID: sourceConversationID,
				SelectionText:  reference.SelectionText,
				CreatedAt:      time.Now(),
			}
			if err := m.repo.CreateReference(txCtx, ref); err != nil {
				return errors.Wrap(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return card, nil
}

func (m *manager) Get(ctx context.Context, id string) (*entity.Card, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}
	card, err := m.repo.GetCard(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	names, err := m.repo.ListTagsForCard(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	card.Tags = names
	if card.Tags == nil {
		card.Tags = []string{}
	}
	refs, err := m.repo.ListReferences(ctx, api.ListReferencesRequest{SourceCardID: &id})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if refs == nil {
		refs = []*entity.Reference{}
	}
	card.References = refs
	score, err := m.computeReviewScore(ctx, card)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	card.ReviewScore = score
	return card, nil
}

func (m *manager) List(ctx context.Context, groupID *string, includeTrashed bool) ([]*entity.Card, error) {
	if groupID != nil {
		if err := ulidutil.Parse(*groupID); err != nil {
			return nil, errors.Wrap(err)
		}
	}
	cards, err := m.repo.ListCards(ctx, groupID, includeTrashed)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	for _, c := range cards {
		names, err := m.repo.ListTagsForCard(ctx, c.ID)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		c.Tags = names
		if c.Tags == nil {
			c.Tags = []string{}
		}
		score, err := m.computeReviewScore(ctx, c)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		c.ReviewScore = score
	}
	return cards, nil
}

func (m *manager) Children(ctx context.Context, parentID string, includeTrashed bool) ([]*entity.Card, error) {
	if err := ulidutil.Parse(parentID); err != nil {
		return nil, errors.Wrap(err)
	}
	cards, err := m.repo.ListChildren(ctx, parentID, includeTrashed)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	for _, c := range cards {
		names, err := m.repo.ListTagsForCard(ctx, c.ID)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		c.Tags = names
		if c.Tags == nil {
			c.Tags = []string{}
		}
		// Children are always nested (parent_card_id != nil), so
		// computeReviewScore short-circuits to nil. Cheap to call.
		score, err := m.computeReviewScore(ctx, c)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		c.ReviewScore = score
	}
	return cards, nil
}

func (m *manager) Update(ctx context.Context, id string, title, content, groupID *string, position *int, tagNames *[]string, format *string, levelEntryID *string, clearLevelEntry bool, genesis *string, summary *string) (*entity.Card, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}

	var updated *entity.Card
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		card, err := m.repo.GetCard(txCtx, id)
		if err != nil {
			return errors.Wrap(err)
		}
		if title != nil {
			trimmed := strings.TrimSpace(*title)
			if trimmed == "" {
				return errors.BadRequest.Newf("card title cannot be empty")
			}
			if len(trimmed) > maxTitleLen {
				return errors.BadRequest.Newf("card title must be %d chars or fewer", maxTitleLen)
			}
			card.Title = trimmed
		}
		if content != nil {
			if len(*content) > maxContentLen {
				return errors.BadRequest.Newf("card content exceeds 1 MB limit")
			}
			card.Content = *content
		}
		groupChanged := false
		oldGroupID := card.GroupID
		if groupID != nil {
			if err := ulidutil.Parse(*groupID); err != nil {
				return errors.Wrap(err)
			}
			if _, err := m.repo.GetGroup(txCtx, *groupID); err != nil {
				return errors.Wrap(err)
			}
			if *groupID != card.GroupID {
				card.GroupID = *groupID
				groupChanged = true
			}
		}
		if position != nil {
			card.Position = *position
		}
		if format != nil {
			if err := validateFormat(*format); err != nil {
				return err
			}
			card.Format = *format
		}
		if genesis != nil {
			card.Genesis = *genesis
		}
		if summary != nil {
			v, err := validateSummary(*summary)
			if err != nil {
				return err
			}
			card.Summary = v
		}
		switch {
		case groupChanged && levelEntryID == nil:
			// The old level_entry_id points into the previous group's catalog
			// and is meaningless in the new one — land the card unfiled (level)
			// in the destination unless the caller sets a new level explicitly.
			card.LevelEntryID = nil
		case clearLevelEntry:
			card.LevelEntryID = nil
		case levelEntryID != nil:
			grp, err := m.repo.GetGroup(txCtx, card.GroupID)
			if err != nil {
				return errors.Wrap(err)
			}
			if err := ensureCatalogEntry(grp.LevelCatalog, *levelEntryID); err != nil {
				return errors.Wrap(err)
			}
			card.LevelEntryID = levelEntryID
		}
		if err := m.repo.UpdateCard(txCtx, card); err != nil {
			return errors.Wrap(err)
		}
		if groupChanged {
			// Re-home the card's tags into the destination group by name: the
			// old tags belong to the previous group's namespace. Keeps the
			// tag.group_id == card.group_id invariant across a move.
			names, err := m.repo.ListTagsForCard(txCtx, id)
			if err != nil {
				return errors.Wrap(err)
			}
			for _, name := range names {
				oldTag, err := m.repo.GetTagByNameInGroup(txCtx, oldGroupID, name)
				if err != nil {
					return errors.Wrap(err)
				}
				if err := m.repo.DetachTag(txCtx, id, oldTag.ID); err != nil {
					return errors.Wrap(err)
				}
				newTag, err := m.tags.EnsureByName(txCtx, card.GroupID, name)
				if err != nil {
					return errors.Wrap(err)
				}
				if err := m.repo.AttachTag(txCtx, id, newTag.ID); err != nil {
					return errors.Wrap(err)
				}
			}
		}
		if tagNames != nil {
			existing, err := m.repo.ListTagsForCard(txCtx, id)
			if err != nil {
				return errors.Wrap(err)
			}
			existingSet := make(map[string]struct{}, len(existing))
			for _, name := range existing {
				existingSet[name] = struct{}{}
			}
			desiredSet := make(map[string]struct{}, len(*tagNames))
			for _, name := range *tagNames {
				desiredSet[name] = struct{}{}
			}
			// Removed tags: detach from the target only. Removal is NOT
			// cascaded — descendants keep the tag if they had it.
			for _, name := range existing {
				if _, keep := desiredSet[name]; keep {
					continue
				}
				t, err := m.repo.GetTagByNameInGroup(txCtx, card.GroupID, name)
				if err != nil {
					return errors.Wrap(err)
				}
				if err := m.repo.DetachTag(txCtx, id, t.ID); err != nil {
					return errors.Wrap(err)
				}
			}
			// Added tags: attach to the target and cascade to every live
			// descendant (containers spread their tags downward). Attach
			// is idempotent — skip descendants that already have the tag.
			addedTagNames := make([]string, 0)
			for _, name := range *tagNames {
				t, err := m.tags.EnsureByName(txCtx, card.GroupID, name)
				if err != nil {
					return errors.Wrap(err)
				}
				if _, already := existingSet[name]; !already {
					if err := m.repo.AttachTag(txCtx, id, t.ID); err != nil {
						return errors.Wrap(err)
					}
					addedTagNames = append(addedTagNames, t.Name)
				}
			}
			if len(addedTagNames) > 0 {
				if err := m.cascadeAttachTags(txCtx, id, addedTagNames); err != nil {
					return errors.Wrap(err)
				}
			}
		}
		names, err := m.repo.ListTagsForCard(txCtx, id)
		if err != nil {
			return errors.Wrap(err)
		}
		card.Tags = names
		if card.Tags == nil {
			card.Tags = []string{}
		}
		updated = card
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return updated, nil
}

// Delete soft-deletes a card (sets deleted_at). When cascade=true, every
// live descendant reached via parent_card_id is trashed in the same
// statement — matching "move folder to trash" intuition. Rows are
// retained for restore, which stays per-card (the user walks the
// subtree back out manually if they want it all).
func (m *manager) Delete(ctx context.Context, id string, cascade bool) error {
	if err := ulidutil.Parse(id); err != nil {
		return errors.Wrap(err)
	}
	return m.repo.Transaction(ctx, func(txCtx context.Context) error {
		card, err := m.repo.GetCard(txCtx, id)
		if err != nil {
			return errors.Wrap(err)
		}
		if card.DeletedAt != nil {
			return nil
		}
		_, err = m.repo.SoftDelete(txCtx, id, cascade)
		return errors.Wrap(err)
	})
}

func (m *manager) Restore(ctx context.Context, id string) (*entity.Card, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}
	if err := m.repo.RestoreCard(ctx, id); err != nil {
		return nil, errors.Wrap(err)
	}
	return m.Get(ctx, id)
}

func (m *manager) Purge(ctx context.Context, id string) error {
	if err := ulidutil.Parse(id); err != nil {
		return errors.Wrap(err)
	}
	return m.repo.Transaction(ctx, func(txCtx context.Context) error {
		if m.conv != nil {
			if err := m.conv.DeleteByAnchor(txCtx, "card", id); err != nil {
				return errors.Wrap(err)
			}
		}
		// Prod SQLite runs without foreign_keys pragma, so the schema-level
		// ON DELETE CASCADE on card_references never fires. Drop references
		// pointing at this card explicitly before the row goes.
		if err := m.repo.DeleteReferencesForCard(txCtx, id); err != nil {
			return errors.Wrap(err)
		}
		return errors.Wrap(m.repo.PurgeCard(txCtx, id))
	})
}

// EmptyTrash hard-deletes every soft-deleted card across all groups.
// Returns the count of cards purged.
func (m *manager) EmptyTrash(ctx context.Context) (int, error) {
	var n int
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		// Same reason as Purge: drop dangling references before the cards go.
		if err := m.repo.DeleteReferencesForTrashedCards(txCtx); err != nil {
			return errors.Wrap(err)
		}
		var perr error
		n, perr = m.repo.PurgeAllTrashedCards(txCtx)
		return errors.Wrap(perr)
	})
	if err != nil {
		return 0, errors.Wrap(err)
	}
	return n, nil
}

func (m *manager) Trash(ctx context.Context, limit int) ([]*entity.Card, error) {
	cards, err := m.repo.ListTrashedCards(ctx, limit)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	for _, c := range cards {
		names, err := m.repo.ListTagsForCard(ctx, c.ID)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		c.Tags = names
		if c.Tags == nil {
			c.Tags = []string{}
		}
	}
	return cards, nil
}

func (m *manager) Decompose(ctx context.Context, req api.DecomposeRequest) (*api.DecomposeResponse, error) {
	if err := ulidutil.Parse(req.ParentCardID); err != nil {
		return nil, errors.Wrap(err)
	}
	if len(req.Cards) == 0 {
		return nil, errors.BadRequest.Newf("decompose requires at least one card spec")
	}

	var created []*entity.Card
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		parent, err := m.repo.GetCard(txCtx, req.ParentCardID)
		if err != nil {
			return errors.Wrap(err)
		}
		if parent.DeletedAt != nil {
			return errors.BadRequest.Newf("card %q already soft-deleted", req.ParentCardID)
		}
		existing, err := m.repo.ListChildren(txCtx, parent.ID, false)
		if err != nil {
			return errors.Wrap(err)
		}
		if len(existing) > 0 {
			return errors.BadRequest.Newf("card %q already decomposed; has %d live children", parent.ID, len(existing))
		}

		// A preamble (non-empty container_content) is only supported on a
		// top-level document. A container rendered inline inside another
		// document still relies on the empty-content heuristic in
		// CardBody.maybeLoad to discover its children, so a nested container
		// with a preamble could not display its sections. Reject it here so
		// that state is never persisted. Empty/whitespace container_content
		// is a no-op and stays allowed (parent content ends empty).
		if req.ContainerContent != nil && strings.TrimSpace(*req.ContainerContent) != "" && parent.ParentCardID != nil {
			return errors.BadRequest.Newf("container_content is only supported on top-level documents")
		}

		defaultGenesis := m.defaultDecomposeGenesis(txCtx, parent)
		created = make([]*entity.Card, 0, len(req.Cards))
		for i, spec := range req.Cards {
			groupID := parent.GroupID
			if spec.GroupID != nil {
				groupID = *spec.GroupID
			}
			f := parent.Format
			if spec.Format != nil {
				if err := validateFormat(*spec.Format); err != nil {
					return errors.Wrapf(err, "card %d", i)
				}
				f = *spec.Format
			}
			gen := defaultGenesis
			if spec.Genesis != nil {
				gen = *spec.Genesis
			}
			parentRef := parent.ID
			// A section's title lives in the title field; strip it from the body
			// so it isn't duplicated as a leading heading (idempotent — a no-op
			// when the content doesn't lead with the title).
			content := htmltext.StripLeadingHeading(spec.Content, f, spec.Title)
			child, err := m.createInTx(txCtx, spec.Title, content, groupID,
				spec.Tags, &parentRef, nil,
				&f, spec.LevelEntryID, &gen, spec.Summary,
			)
			if err != nil {
				return errors.Wrapf(err, "card %d", i)
			}
			pos := i
			if spec.Position != nil {
				pos = *spec.Position
			}
			if child.Position != pos {
				child.Position = pos
				if err := m.repo.UpdateCard(txCtx, child); err != nil {
					return errors.Wrap(err)
				}
			}
			created = append(created, child)
		}
		parent.Content = ""
		if req.ContainerContent != nil {
			parent.Content = *req.ContainerContent
		}
		if err := m.repo.UpdateCard(txCtx, parent); err != nil {
			return errors.Wrap(err)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return &api.DecomposeResponse{Cards: created}, nil
}

func (m *manager) Compose(ctx context.Context, req api.ComposeRequest) (*api.ComposeResponse, error) {
	if len(req.SourceCardIDs) < 2 {
		return nil, errors.BadRequest.Newf("compose requires at least 2 source cards")
	}
	if !idsUnique(req.SourceCardIDs) {
		return nil, errors.BadRequest.Newf("source_card_ids must be unique")
	}

	var target *entity.Card
	err := m.repo.Transaction(ctx, func(txCtx context.Context) error {
		sources, sharedGroupID, err := m.loadSourcesForCompose(txCtx, req.SourceCardIDs)
		if err != nil {
			return err
		}

		spec := req.Target
		groupID := sharedGroupID
		if spec.GroupID != nil {
			groupID = *spec.GroupID
		}

		gen := defaultComposeGenesis(sources)
		if spec.Genesis != nil && *spec.Genesis != "" {
			gen = *spec.Genesis
		}

		var format *string
		if spec.Format != nil {
			if err := validateFormat(*spec.Format); err != nil {
				return errors.Wrap(err)
			}
			format = spec.Format
		}

		created, err := m.createInTx(txCtx, spec.Title, spec.Content, groupID,
			spec.Tags, nil, nil,
			format, spec.LevelEntryID, &gen, spec.Summary,
		)
		if err != nil {
			return errors.Wrap(err)
		}
		if spec.Position != nil {
			created.Position = *spec.Position
			if err := m.repo.UpdateCard(txCtx, created); err != nil {
				return errors.Wrap(err)
			}
		}
		target = created

		now := time.Now()
		for _, src := range sources {
			src.DeletedAt = &now
			if err := m.repo.UpdateCard(txCtx, src); err != nil {
				return errors.Wrap(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return &api.ComposeResponse{Card: target}, nil
}

func (m *manager) loadSourcesForCompose(ctx context.Context, ids []string) ([]*entity.Card, string, error) {
	sources := make([]*entity.Card, 0, len(ids))
	var groupID string
	for i, id := range ids {
		if err := ulidutil.Parse(id); err != nil {
			return nil, "", errors.Wrap(err)
		}
		c, err := m.repo.GetCard(ctx, id)
		if err != nil {
			return nil, "", errors.Wrap(err)
		}
		if c.DeletedAt != nil {
			return nil, "", errors.BadRequest.Newf("source card %q is already soft-deleted", id)
		}
		if i == 0 {
			groupID = c.GroupID
		} else if c.GroupID != groupID {
			return nil, "", errors.BadRequest.Newf("compose sources must all share the same group")
		}
		sources = append(sources, c)
	}
	return sources, groupID, nil
}

func idsUnique(ids []string) bool {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

// createInTx runs repo.CreateCard + tag attachment inside an existing
// transaction. Returns the created card. Validates level_entry_id (when
// non-nil) against the target group's catalog.
func (m *manager) createInTx(txCtx context.Context, title, content, groupID string, tagNames []string, parentCardID, sourceConversationID *string, format *string, levelEntryID *string, genesis *string, summary *string) (*entity.Card, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.BadRequest.Newf("card title is required")
	}
	if len(title) > maxTitleLen {
		return nil, errors.BadRequest.Newf("card title must be %d chars or fewer", maxTitleLen)
	}
	if len(content) > maxContentLen {
		return nil, errors.BadRequest.Newf("card content exceeds 1 MB limit")
	}
	f := formatMarkdown
	if format != nil {
		f = *format
	}
	if err := validateFormat(f); err != nil {
		return nil, err
	}

	g := ""
	if genesis != nil {
		g = *genesis
	}
	sm := ""
	if summary != nil {
		v, err := validateSummary(*summary)
		if err != nil {
			return nil, err
		}
		sm = v
	}

	now := time.Now()
	card := &entity.Card{
		ID:                   ulidutil.New(),
		Title:                title,
		Summary:              sm,
		Content:              content,
		Format:               f,
		Genesis:              g,
		GroupID:              groupID,
		Position:             0,
		Tags:                 []string{},
		ParentCardID:         parentCardID,
		SourceConversationID: sourceConversationID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if levelEntryID != nil {
		grp, err := m.repo.GetGroup(txCtx, groupID)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if err := ensureCatalogEntry(grp.LevelCatalog, *levelEntryID); err != nil {
			return nil, errors.Wrap(err)
		}
		card.LevelEntryID = levelEntryID
	}
	if err := m.repo.CreateCard(txCtx, card); err != nil {
		return nil, errors.Wrap(err)
	}
	for _, name := range tagNames {
		tag, err := m.tags.EnsureByName(txCtx, groupID, name)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if err := m.repo.AttachTag(txCtx, card.ID, tag.ID); err != nil {
			return nil, errors.Wrap(err)
		}
		card.Tags = append(card.Tags, tag.Name)
	}
	return card, nil
}

// ensureCatalogEntry returns nil when entryID matches one of catalog's
// entries; otherwise a BadRequest error. Used by card create/update to
// validate incoming level_entry_id against the target group's catalog.
func ensureCatalogEntry(catalog []entity.LevelEntry, entryID string) error {
	for _, e := range catalog {
		if e.ID == entryID {
			return nil
		}
	}
	return errors.BadRequest.Newf("level_entry_id %q not found in group catalog", entryID)
}
