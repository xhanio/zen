package health

import (
	"net/http"

	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
)

func (r *router) HealthZ(c api.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (r *router) ReadyZ(c api.Context) error {
	if err := r.db.DB().Ping(); err != nil {
		return errors.Unavailable.Wrap(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}
