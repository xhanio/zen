package health

import (
	"database/sql"
	_ "embed"
	"path"

	"github.com/xhanio/framingo/pkg/services/db"
	fapi "github.com/xhanio/framingo/pkg/types/api"
	"github.com/xhanio/framingo/pkg/types/common"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/reflectutil"

	"github.com/xhanio/zen/pkg/types/api"
)

var _ fapi.Router = (*router)(nil)

//go:embed router.yaml
var config []byte

// pinger is the narrow surface this router needs from a database manager.
// db.Manager satisfies it via DB() *sql.DB.
type pinger interface {
	DB() *sql.DB
}

type router struct {
	name string
	log  log.Logger
	db   pinger
}

func New(database db.Manager, logger log.Logger) fapi.Router {
	return &router{
		log: logger,
		db:  database,
	}
}

func (r *router) Name() string {
	if r.name == "" {
		r.name = path.Join(reflectutil.Locate(r))
	}
	return r.name
}

func (r *router) Dependencies() []common.Service {
	// The narrow pinger interface won't satisfy common.Service; we still
	// need the real db.Manager from initServices for ordering. Cast back.
	if svc, ok := r.db.(common.Service); ok {
		return []common.Service{svc}
	}
	return nil
}

func (r *router) Config() []byte {
	return config
}

func (r *router) Handlers() map[string]any {
	handlers := api.DiscoverHandlers(r)
	r.log.Debugf("router %s parsed %d handler(s)", r.Name(), len(handlers))
	return handlers
}
