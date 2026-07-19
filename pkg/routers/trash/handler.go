package trash

import (
	"net/http"
	"strconv"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) ListTrash(c api.Context) error {
	limit := 100
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	cards, err := r.svc.Trash(c, limit)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, api.TrashResponse{Cards: cards})
}

func (r *router) EmptyTrash(c api.Context) error {
	n, err := r.svc.EmptyTrash(c)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, api.EmptyTrashResponse{Purged: n})
}
