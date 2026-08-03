package zenbackend

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/services/api/server"
	"github.com/xhanio/framingo/pkg/services/db"
	_ "github.com/xhanio/framingo/pkg/services/db/drivers/sqlite" // registers the sqlite driver with the db service
	"github.com/xhanio/framingo/pkg/services/messagebus"
	"github.com/xhanio/framingo/pkg/services/pubsub"
	"github.com/xhanio/framingo/pkg/services/supervisor"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/sliceutil"

	"github.com/xhanio/zen/pkg/services/card"
	"github.com/xhanio/zen/pkg/services/conversation"
	"github.com/xhanio/zen/pkg/services/delivery"
	"github.com/xhanio/zen/pkg/services/group"
	"github.com/xhanio/zen/pkg/services/presence"
	"github.com/xhanio/zen/pkg/services/reference"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/search"
	"github.com/xhanio/zen/pkg/services/snapshot"
	"github.com/xhanio/zen/pkg/services/tag"
	"github.com/xhanio/zen/pkg/utils/busutil"
	"github.com/xhanio/zen/pkg/utils/infra"
)

// defaultMonitorInterval is how often the supervisor re-probes every service
// when supervisor.monitor.interval is unset. It sets the staleness ceiling on
// /readyz, since the roll-up reads what the last sweep recorded.
const defaultMonitorInterval = 15 * time.Second

func (m *manager) initServices() error {
	m.log = log.New(
		log.WithLevel(m.config.GetInt("log.level")),
		log.WithFileWriter(
			m.config.GetString("log.file"),
			m.config.GetInt("log.rotation.max_size"),
			m.config.GetInt("log.rotation.max_backups"),
			m.config.GetInt("log.rotation.max_age"),
		),
	)
	infra.Debug = (m.log.Level() == zapcore.DebugLevel)

	// Without a monitor interval the supervisor never re-probes: every stat
	// keeps whatever Init left it, so /readyz would answer "ready" against a
	// database that died after startup. The restart policy stays at its zero
	// value (no in-process restarts), so a liveness failure escalates to
	// whatever supervises the process rather than being retried here.
	monitorInterval := m.config.GetDuration("supervisor.monitor.interval")
	if monitorInterval <= 0 {
		monitorInterval = defaultMonitorInterval
	}
	m.services = supervisor.New(m.config,
		supervisor.WithLogger(m.log),
		supervisor.WithMonitorInterval(monitorInterval),
	)

	// The migrations directory is the single source of truth for the schema
	// version. db.migration.version is absent from every config, so this
	// resolves to 0, which makes framingo run Up() — migrate to the newest
	// migration it ships with.
	//
	// Never pin a literal here. golang-migrate walks DOWN as readily as up, so
	// a target below the DB's current version silently runs the
	// down-migrations. That is how a fixed 15 dropped cards.review_grade on
	// every restart of a v0.13 deployment.
	m.db = db.New(
		db.WithType(m.config.GetString("db.type")),
		db.WithDataSource(db.Source{
			DBName: sliceutil.First(
				m.config.GetString("db.source.dbname"),
				m.config.GetString("DB_DBNAME"),
			),
		}),
		db.WithMigration(
			m.config.GetString("db.migration.dir"),
			m.config.GetUint("db.migration.version"),
		),
		db.WithConnection(
			m.config.GetInt("db.connection.max_open"),
			m.config.GetInt("db.connection.max_idle"),
			m.config.GetDuration("db.connection.max_lifetime"),
			m.config.GetDuration("db.connection.max_idle_time"),
			m.config.GetDuration("db.connection.exec_timeout"),
		),
		db.WithLogger(m.log),
	)

	m.repository = repository.New(
		m.db,
		repository.WithLogger(m.log),
	)

	m.pubsub = pubsub.New(
		busutil.NewDriver(m.log),
		pubsub.WithLogger(m.log),
	)

	m.bus = messagebus.New(
		m.pubsub,
		messagebus.WithLogger(m.log),
	)

	m.presence = presence.New(
		presence.WithLogger(m.log),
	)

	m.delivery = delivery.New(
		delivery.WithLogger(m.log),
	)

	m.conversation = conversation.New(
		m.repository,
		conversation.WithLogger(m.log),
		conversation.WithMessageBus(m.bus),
	)

	m.group = group.New(
		m.repository,
		m.conversation,
		group.WithLogger(m.log),
	)

	m.tag = tag.New(
		m.repository,
		tag.WithLogger(m.log),
	)

	m.card = card.New(
		m.repository,
		m.tag,
		m.conversation,
		card.WithLogger(m.log),
	)

	m.search = search.New(
		m.repository,
		search.WithLogger(m.log),
	)

	m.reference = reference.New(
		m.repository,
		m.card,
		m.conversation,
		reference.WithLogger(m.log),
	)

	m.snapshot = snapshot.New(
		m.repository,
		snapshot.WithLogger(m.log),
	)

	m.api = server.New(
		server.WithLogger(m.log),
	)

	for name := range m.config.GetStringMap("api") {
		opts := []server.ServerOption{
			server.WithEndpoint(
				m.config.GetString(fmt.Sprintf("api.%s.host", name)),
				m.config.GetUint(fmt.Sprintf("api.%s.port", name)),
				m.config.GetString(fmt.Sprintf("api.%s.prefix", name)),
			),
		}
		// Per-server middleware configs: a plain mapping of middleware name to
		// its default config, unordered - order only matters in router.yaml,
		// where attachment lives. Each block reaches its middleware wherever
		// nothing more specific is configured.
		if mws := m.config.GetStringMap(fmt.Sprintf("api.%s.middlewares", name)); len(mws) > 0 {
			raw, err := yaml.Marshal(mws)
			if err != nil {
				return errors.Wrap(err)
			}
			opts = append(opts, server.WithMiddlewareConfigs(raw))
		}
		if err := m.api.Add(name, opts...); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}
