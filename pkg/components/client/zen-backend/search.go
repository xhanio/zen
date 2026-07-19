package zenbackend

import (
	"context"
	"net/url"
	"strconv"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

type searchClient struct{ c *client }

func (s *searchClient) Search(ctx context.Context, query, scope string, limit int) (*api.SearchResponse, error) {
	q := url.Values{}
	q.Set("q", query)
	if scope != "" {
		q.Set("scope", scope)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/search?" + q.Encode()
	var out api.SearchResponse
	if err := s.c.doJSON(ctx, "GET", path, nil, &out); err != nil {
		return nil, errors.Wrap(err)
	}
	return &out, nil
}
