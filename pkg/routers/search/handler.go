package search

import (
	"net/http"
	"strconv"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) Search(c api.Context) error {
	query := c.QueryParam("q")
	scope := c.QueryParam("scope")

	var limit int
	if lstr := c.QueryParam("limit"); lstr != "" {
		n, err := strconv.Atoi(lstr)
		if err != nil {
			return errors.BadRequest.Newf("limit must be an integer")
		}
		limit = n
	}

	cards, msgs, err := r.svc.Search(c, query, scope, limit)
	if err != nil {
		return errors.Wrap(err)
	}

	respScope := scope
	if respScope == "" {
		respScope = "all"
	}

	return c.JSON(http.StatusOK, api.SearchResponse{
		Query:    query,
		Scope:    respScope,
		Cards:    cards,
		Messages: msgs,
	})
}
