package zenbackend

import (
	"context"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type groupClient struct{ c *client }

func (g *groupClient) Create(ctx context.Context, req api.CreateGroupRequest) (*entity.Group, error) {
	var out entity.Group
	if err := g.c.doJSON(ctx, "POST", "/groups", req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (g *groupClient) List(ctx context.Context) ([]*entity.Group, error) {
	var out []*entity.Group
	if err := g.c.doJSON(ctx, "GET", "/groups", nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return out, nil
}

func (g *groupClient) Get(ctx context.Context, id string) (*entity.Group, error) {
	var out entity.Group
	if err := g.c.doJSON(ctx, "GET", "/groups/"+id, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (g *groupClient) Update(ctx context.Context, id string, req api.UpdateGroupRequest) (*entity.Group, error) {
	var out entity.Group
	if err := g.c.doJSON(ctx, "PUT", "/groups/"+id, req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (g *groupClient) Delete(ctx context.Context, id string, recursive bool) error {
	path := "/groups/" + id
	if recursive {
		path += "?recursive=true"
	}
	if err := g.c.doNoContent(ctx, "DELETE", path, nil); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
