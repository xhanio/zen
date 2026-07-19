package zenchannel

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xhanio/errors"

	server "github.com/xhanio/zen/pkg/components/server/zen-channel"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// runChannel serves the Claude Code channel until the context is cancelled.
func runChannel(backendURL, logDir string) error {
	// Each channel process gets its own id — several can run at once
	// (one per Claude Code session). SIGUSR1 prints it so operators can
	// tell instances apart; SIGINT/SIGTERM shut down gracefully.
	instanceID := ulidutil.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go listenChannelSignals(ctx, cancel, instanceID)

	if err := server.RunChannel(ctx, server.ChannelOptions{
		BackendURL: backendURL,
		LogDir:     logDir,
		InstanceID: instanceID,
	}); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// listenChannelSignals runs the channel's signal loop until ctx is cancelled.
// Modeled on the zen-backend manager's handler: SIGINT/SIGTERM trigger a
// graceful shutdown, SIGUSR1 prints this instance's id. Everything goes to
// stderr — stdout is the JSON-RPC framing channel and must stay clean.
func listenChannelSignals(ctx context.Context, cancel context.CancelFunc, instanceID string) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	defer signal.Stop(signalCh)
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-signalCh:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				fmt.Fprintf(os.Stderr, "zen-channel: received %s, shutting down instance %s...\n", sig, instanceID)
				cancel()
				return
			case syscall.SIGUSR1:
				fmt.Fprintf(os.Stderr, "zen-channel: instance %s\n", instanceID)
			}
		}
	}
}
