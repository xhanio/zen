package reference

import (
	"net/http"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) CreateReference(c api.Context) error {
	var req api.CreateReferenceRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	ref, err := r.svc.Create(c, req)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusCreated, ref)
}

func (r *router) GetReference(c api.Context) error {
	id := c.Param("id")
	ref, err := r.svc.Get(c, id)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, ref)
}

func (r *router) ListReferences(c api.Context) error {
	var f api.ListReferencesRequest
	if s := c.QueryParam("source_card_id"); s != "" {
		f.SourceCardID = &s
	}
	if s := c.QueryParam("derived_card_id"); s != "" {
		f.DerivedCardID = &s
	}
	if s := c.QueryParam("conversation_id"); s != "" {
		f.ConversationID = &s
	}
	refs, err := r.svc.List(c, f)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, api.ListReferencesResponse{References: refs})
}

func (r *router) DeleteReference(c api.Context) error {
	id := c.Param("id")
	if err := r.svc.Delete(c, id); err != nil {
		return errors.Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}
