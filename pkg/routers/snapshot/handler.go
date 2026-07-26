package snapshot

import (
	"net/http"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) ListSnapshots(c api.Context) error {
	var f api.ListSnapshotsRequest
	if s := c.QueryParam("card_id"); s != "" {
		f.CardID = &s
	}
	if s := c.QueryParam("conversation_id"); s != "" {
		f.ConversationID = &s
	}
	snaps, err := r.svc.List(c, f)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, api.ListSnapshotsResponse{Snapshots: snaps})
}

func (r *router) GetSnapshot(c api.Context) error {
	resp, err := r.svc.Get(c, c.Param("id"))
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, resp)
}
