package zenbackend

import (
	"context"
	"net/url"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

type referenceClient struct{ c *client }

func (k *referenceClient) Create(ctx context.Context, req api.CreateReferenceRequest) (*entity.Reference, error) {
	var out entity.Reference
	if err := k.c.doJSON(ctx, "POST", "/references", req, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *referenceClient) Get(ctx context.Context, id string) (*entity.Reference, error) {
	var out entity.Reference
	if err := k.c.doJSON(ctx, "GET", "/references/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}

func (k *referenceClient) List(ctx context.Context, f api.ListReferencesRequest) ([]*entity.Reference, error) {
	q := url.Values{}
	if f.SourceCardID != nil {
		q.Set("source_card_id", *f.SourceCardID)
	}
	if f.DerivedCardID != nil {
		q.Set("derived_card_id", *f.DerivedCardID)
	}
	if f.ConversationID != nil {
		q.Set("conversation_id", *f.ConversationID)
	}
	path := "/references"
	if enc := q.Encode(); enc != "" {
		path = path + "?" + enc
	}
	var out api.ListReferencesResponse
	if err := k.c.doJSON(ctx, "GET", path, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return out.References, nil
}

func (k *referenceClient) Delete(ctx context.Context, id string) error {
	if err := k.c.doNoContent(ctx, "DELETE", "/references/"+url.PathEscape(id), nil); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
