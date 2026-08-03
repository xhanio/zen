package zenbackend

import (
	"context"

	"github.com/spf13/viper"
	"github.com/xhanio/framingo/pkg/services/api/server"
	"github.com/xhanio/framingo/pkg/services/db"
	"github.com/xhanio/framingo/pkg/services/pubsub"
	"github.com/xhanio/framingo/pkg/services/supervisor"
	framodel "github.com/xhanio/framingo/pkg/types/model"
	"github.com/xhanio/framingo/pkg/utils/log"

	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/types/model"
)

type manager struct {
	name   string
	config *viper.Viper

	log log.Logger

	db         db.Manager
	repository repository.Repository
	pubsub     pubsub.Manager
	bus        framodel.MessageBus

	group        model.Group
	tag          model.Tag
	card         model.Card
	search       model.Search
	conversation model.Conversation
	reference    model.Reference
	snapshot     model.Snapshot
	presence     model.Presence
	delivery     model.Delivery

	api server.Manager

	services supervisor.Manager

	ctx    context.Context
	cancel context.CancelFunc
}

func New(configPath string) Server {
	return &manager{
		config: newConfig(configPath),
	}
}

func (m *manager) Name() string {
	return m.name
}
