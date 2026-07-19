package tag

import (
	"context"
	"strings"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

const maxTagNameLen = 50

func normalizeTagName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (m *manager) EnsureByName(ctx context.Context, groupID, name string) (*entity.Tag, error) {
	norm := normalizeTagName(name)
	if norm == "" {
		return nil, errors.BadRequest.Newf("tag name is required")
	}
	if len(norm) > maxTagNameLen {
		return nil, errors.BadRequest.Newf("tag name must be %d chars or fewer", maxTagNameLen)
	}
	existing, err := m.repo.GetTagByNameInGroup(ctx, groupID, norm)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, errors.NotFound) {
		return nil, errors.Wrap(err)
	}
	t := &entity.Tag{
		ID:      ulidutil.New(),
		GroupID: groupID,
		Name:    norm,
	}
	if err := m.repo.CreateTag(ctx, t); err != nil {
		return nil, errors.Wrap(err)
	}
	return t, nil
}

func (m *manager) Get(ctx context.Context, id string) (*entity.Tag, error) {
	if err := ulidutil.Parse(id); err != nil {
		return nil, errors.Wrap(err)
	}
	return m.repo.GetTag(ctx, id)
}

func (m *manager) List(ctx context.Context, groupID string) ([]*entity.Tag, error) {
	return m.repo.ListTags(ctx, groupID)
}

func (m *manager) Rename(ctx context.Context, groupID, oldName, newName string) (*entity.Tag, error) {
	oldNorm := normalizeTagName(oldName)
	newNorm := normalizeTagName(newName)
	if newNorm == "" {
		return nil, errors.BadRequest.Newf("new tag name is required")
	}
	if len(newNorm) > maxTagNameLen {
		return nil, errors.BadRequest.Newf("new tag name must be %d chars or fewer", maxTagNameLen)
	}
	if oldNorm == newNorm {
		return m.repo.GetTagByNameInGroup(ctx, groupID, oldNorm)
	}
	src, err := m.repo.GetTagByNameInGroup(ctx, groupID, oldNorm)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	target, err := m.repo.GetTagByNameInGroup(ctx, groupID, newNorm)
	switch {
	case err == nil:
		// Target exists: merge. Delete src; return target. M3's Card service
		// will be responsible for re-pointing card_tags rows; in M2 there
		// are no Go-managed cards so the merge is purely a tag-table op.
		if err := m.repo.DeleteTag(ctx, src.ID); err != nil {
			return nil, errors.Wrap(err)
		}
		return target, nil
	case errors.Is(err, errors.NotFound):
		src.Name = newNorm
		if err := m.repo.UpdateTag(ctx, src); err != nil {
			return nil, errors.Wrap(err)
		}
		return src, nil
	default:
		return nil, errors.Wrap(err)
	}
}

func (m *manager) Delete(ctx context.Context, groupID, name string) error {
	norm := normalizeTagName(name)
	if norm == "" {
		return errors.BadRequest.Newf("tag name is required")
	}
	t, err := m.repo.GetTagByNameInGroup(ctx, groupID, norm)
	if err != nil {
		return errors.Wrap(err)
	}
	return m.repo.DeleteTag(ctx, t.ID)
}
