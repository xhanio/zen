package zenmcp

import (
	"context"
	"path"

	"github.com/spf13/viper"
	"github.com/xhanio/framingo/pkg/services/api/server"
	"github.com/xhanio/framingo/pkg/services/supervisor"
	"github.com/xhanio/framingo/pkg/utils/log"
	"github.com/xhanio/framingo/pkg/utils/reflectutil"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
	"github.com/xhanio/zen/pkg/types/model"
)

type manager struct {
	name   string
	config *viper.Viper

	log log.Logger

	backend zenbackend.Client
	mcpSvc  model.MCP

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
	if m.name == "" {
		m.name = path.Join(reflectutil.Locate(m))
	}
	return m.name
}
