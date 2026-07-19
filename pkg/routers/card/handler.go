package card

import (
	"net/http"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) CreateCard(c api.Context) error {
	var req api.CreateCardRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	card, err := r.svc.Create(c,
		req.Title, req.Content, req.GroupID, req.Tags,
		req.ParentCardID, req.SourceConversationID,
		req.Format, req.LevelEntryID, req.Genesis,
		req.Reference, req.Summary,
	)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusCreated, card)
}

func (r *router) ListCards(c api.Context) error {
	var groupID *string
	if g := c.QueryParam("group_id"); g != "" {
		groupID = &g
	}
	includeTrashed := c.QueryParam("include_trashed") == "true"
	cards, err := r.svc.List(c, groupID, includeTrashed)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, cards)
}

func (r *router) GetCard(c api.Context) error {
	id := c.Param("id")
	card, err := r.svc.Get(c, id)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, card)
}

func (r *router) UpdateCard(c api.Context) error {
	id := c.Param("id")
	var req api.UpdateCardRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	card, err := r.svc.Update(c, id,
		req.Title, req.Content, req.GroupID,
		req.Position, req.Tags,
		req.Format, req.LevelEntryID, req.ClearLevelEntry, req.Genesis,
		req.Summary,
	)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, card)
}

func (r *router) DeleteCard(c api.Context) error {
	id := c.Param("id")
	// Cascade defaults to true so DELETE /cards/:id keeps its "move
	// folder to trash" behavior. Pass ?cascade=false to trash just this
	// card and leave its descendants live.
	cascade := c.QueryParam("cascade") != "false"
	if err := r.svc.Delete(c, id, cascade); err != nil {
		return errors.Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (r *router) DecomposeCard(c api.Context) error {
	var req api.DecomposeRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	resp, err := r.svc.Decompose(c, req)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (r *router) ComposeCard(c api.Context) error {
	var req api.ComposeRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	resp, err := r.svc.Compose(c, req)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (r *router) RestoreCard(c api.Context) error {
	id := c.Param("id")
	card, err := r.svc.Restore(c, id)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, card)
}

func (r *router) PurgeCard(c api.Context) error {
	id := c.Param("id")
	if err := r.svc.Purge(c, id); err != nil {
		return errors.Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (r *router) ListChildren(c api.Context) error {
	id := c.Param("id")
	includeTrashed := c.QueryParam("include_trashed") == "true"
	cards, err := r.svc.Children(c, id, includeTrashed)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, map[string]any{"cards": cards})
}

func (r *router) ReorderCard(c api.Context) error {
	id := c.Param("id")
	var req api.ReorderCardRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	card, err := r.svc.Reorder(c, id, req.Position)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, card)
}

func (r *router) ReviewCard(c api.Context) error {
	id := c.Param("id")
	var req api.ReviewCardRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	card, err := r.svc.Review(c, id, req.Grade)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, card)
}
