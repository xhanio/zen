package zenmcp

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/xhanio/errors"

	server "github.com/xhanio/zen/pkg/components/server/zen-mcp"
)

var (
	configPath string
)

func NewDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "daemon",
		RunE:         runDaemon,
		SilenceUsage: true,
	}
	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to the YAML config file")
	return cmd
}

func runDaemon(cmd *cobra.Command, args []string) error {
	m := server.New(configPath)
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		return errors.Wrap(err)
	}
	if err := m.Start(ctx); err != nil {
		return errors.Wrap(err)
	}
	return nil
}
