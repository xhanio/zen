package group

import (
	"net/http"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) CreateGroup(c api.Context) error {
	var req api.CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	g, err := r.svc.Create(c, req.Name, req.LevelCatalog, req.Rule)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusCreated, g)
}

func (r *router) ListGroups(c api.Context) error {
	gs, err := r.svc.List(c)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, gs)
}

func (r *router) GetGroup(c api.Context) error {
	id := c.Param("id")
	g, err := r.svc.Get(c, id)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, g)
}

func (r *router) UpdateGroup(c api.Context) error {
	id := c.Param("id")
	var req api.UpdateGroupRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	g, err := r.svc.Update(c, id, req.Name, req.Position, req.LevelCatalog, req.Rule)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, g)
}

func (r *router) DeleteGroup(c api.Context) error {
	id := c.Param("id")
	recursive := c.QueryParam("recursive") == "true"
	if err := r.svc.Delete(c, id, recursive); err != nil {
		return errors.Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}
