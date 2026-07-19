package zenbackend

import (
	"context"
	"fmt"
	"net/url"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type cardClient struct{ c *client }

func (k *cardClient) Create(ctx context.Context, req api.CreateCardRequest) (*entity.Card, error) {
	var out entity.Card
	if err := k.c.doJSON(ctx, "POST", "/cards", req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) List(ctx context.Context, groupID *string, includeTrashed bool) ([]*entity.Card, error) {
	path := "/cards"
	q := url.Values{}
	if groupID != nil {
		q.Set("group_id", *groupID)
	}
	if includeTrashed {
		q.Set("include_trashed", "true")
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out []*entity.Card
	if err := k.c.doJSON(ctx, "GET", path, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return out, nil
}

func (k *cardClient) Get(ctx context.Context, id string) (*entity.Card, error) {
	var out entity.Card
	if err := k.c.doJSON(ctx, "GET", "/cards/"+id, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) Update(ctx context.Context, id string, req api.UpdateCardRequest) (*entity.Card, error) {
	var out entity.Card
	if err := k.c.doJSON(ctx, "PUT", "/cards/"+id, req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) Delete(ctx context.Context, id string, cascade bool) error {
	path := "/cards/" + id
	if !cascade {
		path += "?cascade=false"
	}
	if err := k.c.doNoContent(ctx, "DELETE", path, nil); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func (k *cardClient) Decompose(ctx context.Context, req api.DecomposeRequest) (*api.DecomposeResponse, error) {
	var out api.DecomposeResponse
	if err := k.c.doJSON(ctx, "POST", "/cards/decompose", req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) Compose(ctx context.Context, req api.ComposeRequest) (*api.ComposeResponse, error) {
	var out api.ComposeResponse
	if err := k.c.doJSON(ctx, "POST", "/cards/compose", req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) Restore(ctx context.Context, id string) (*entity.Card, error) {
	var out entity.Card
	if err := k.c.doJSON(ctx, "POST", "/cards/"+id+"/restore", nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) Purge(ctx context.Context, id string) error {
	if err := k.c.doNoContent(ctx, "DELETE", "/cards/"+id+"/purge", nil); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func (k *cardClient) Trash(ctx context.Context, limit int) (*api.TrashResponse, error) {
	path := "/trash"
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}
	var out api.TrashResponse
	if err := k.c.doJSON(ctx, "GET", path, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) EmptyTrash(ctx context.Context) (*api.EmptyTrashResponse, error) {
	var out api.EmptyTrashResponse
	if err := k.c.doJSON(ctx, "DELETE", "/trash", nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) Children(ctx context.Context, parentID string, includeTrashed bool) ([]*entity.Card, error) {
	path := "/cards/" + parentID + "/children"
	if includeTrashed {
		path += "?include_trashed=true"
	}
	var out struct {
		Cards []*entity.Card `json:"cards"`
	}
	if err := k.c.doJSON(ctx, "GET", path, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return out.Cards, nil
}

func (k *cardClient) Reorder(ctx context.Context, id string, position int) (*entity.Card, error) {
	var out entity.Card
	if err := k.c.doJSON(ctx, "POST", "/cards/"+id+"/reorder", api.ReorderCardRequest{Position: position}, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *cardClient) Review(ctx context.Context, id string, grade string) (*entity.Card, error) {
	var out entity.Card
	if err := k.c.doJSON(ctx, "POST", "/cards/"+id+"/review", api.ReviewCardRequest{Grade: grade}, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}
