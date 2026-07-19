package zenchannel

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	help    bool
	verbose bool
)

// NewRootCmd builds the zen-channel command tree. Unlike zen-mcp, the channel
// is the binary's whole job, so it runs from the root rather than a subcommand
// — Claude Code spawns it as a bare `zen-channel` (see plugins/zen/.mcp.json).
func NewRootCmd() *cobra.Command {
	var backendURL string
	var logDir string
	root := &cobra.Command{
		Use:          "zen-channel",
		Short:        "Run the Claude Code channel server (stdio JSON-RPC + backend WS subscriber)",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if help {
				_ = cmd.Help()
				os.Exit(0)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChannel(backendURL, logDir)
		},
	}
	root.PersistentFlags().BoolVar(&help, "help", false, "")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
	root.Flags().StringVar(&backendURL, "backend-url", "", "Zen backend URL without /api/v1 (default: $ZEN_BACKEND_URL or http://127.0.0.1:38000)")
	root.Flags().StringVar(&logDir, "log-dir", "", "directory for per-instance channel-{id}.log dispatch logs (stdio frames + WS events); empty disables logging")
	root.AddCommand(NewVersionCmd())
	return root
}
