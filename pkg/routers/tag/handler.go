package tag

import (
	"net/http"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) ListTags(c api.Context) error {
	gid := c.Param("id")
	tags, err := r.svc.List(c, gid)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, tags)
}

func (r *router) RenameTag(c api.Context) error {
	gid := c.Param("id")
	old := c.Param("name")
	var req api.RenameTagRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	t, err := r.svc.Rename(c, gid, old, req.NewName)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, t)
}

func (r *router) DeleteTag(c api.Context) error {
	gid := c.Param("id")
	name := c.Param("name")
	if err := r.svc.Delete(c, gid, name); err != nil {
		return errors.Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}
