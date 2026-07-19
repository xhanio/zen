package zenbackend

import (
	"context"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type tagClient struct{ c *client }

func (t *tagClient) List(ctx context.Context, groupID string) ([]*entity.Tag, error) {
	var out []*entity.Tag
	if err := t.c.doJSON(ctx, "GET", "/groups/"+groupID+"/tags", nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return out, nil
}

func (t *tagClient) Rename(ctx context.Context, groupID, name string, req api.RenameTagRequest) (*entity.Tag, error) {
	var out entity.Tag
	if err := t.c.doJSON(ctx, "PUT", "/groups/"+groupID+"/tags/"+name, req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (t *tagClient) Delete(ctx context.Context, groupID, name string) error {
	if err := t.c.doNoContent(ctx, "DELETE", "/groups/"+groupID+"/tags/"+name, nil); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
