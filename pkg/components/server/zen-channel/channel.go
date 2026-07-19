package zenchannel

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/xhanio/framingo/pkg/utils/log"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// ChannelOptions configures RunChannel. Zero values pick reasonable defaults
// for production stdio mode.
type ChannelOptions struct {
	// BackendURL is the zen-backend root WITHOUT the /api/v1 prefix
	// (e.g. "http://127.0.0.1:38000"). Defaults to $ZEN_BACKEND_URL or
	// http://127.0.0.1:38000 — 38000 is the host port Zen's compose publishes
	// (nginx :80 → 38000).
	BackendURL string

	// In/Out are the JSON-RPC framing endpoints. Default to os.Stdin/Stdout.
	In  io.Reader
	Out io.Writer

	// InstanceID identifies this channel process. Multiple channels can run
	// at once (one per Claude Code session), so each gets a distinct id that
	// SIGUSR1 surfaces for operators. Defaults to a fresh ulid.
	InstanceID string

	// LogDir, when non-empty, is a directory in which the channel writes a
	// per-instance log file named "channel-{InstanceID}.log" capturing all
	// dispatch info — every stdio JSON-RPC frame in/out and every WS event
	// handled, plus per-frame debug traces. Empty leaves the channel silent
	// (stdout must stay clean).
	LogDir string
}

// claudeSessionEnv is the environment variable Claude Code sets on the
// processes it spawns, including this MCP server. It is undocumented, so a
// future release may rename it; resolveSessionID degrades instead of failing.
const claudeSessionEnv = "CLAUDE_CODE_SESSION_ID"

// resolveSessionID returns the Claude Code session id this channel belongs to,
// falling back to instanceID when the env var is absent. The fallback keeps the
// channel registrable — the picker then shows a ulid prefix instead of a label,
// which is a degraded picker rather than an invisible session.
func resolveSessionID(instanceID string, logger log.Logger) string {
	if id := os.Getenv(claudeSessionEnv); id != "" {
		return id
	}
	if logger != nil {
		logger.Warnf("%s is unset; falling back to instance id %s for routing", claudeSessionEnv, instanceID)
	}
	return instanceID
}

// RunChannel is the entry point used by both `zen-mcp channel` (real stdio)
// and TestE2E_ChannelFlow (driven over net.Pipe). It builds the backend
// client, wires the stdio server and the WS subscriber together, and runs
// both until ctx is cancelled or either side errors.
func RunChannel(ctx context.Context, opts ChannelOptions) error {
	if opts.BackendURL == "" {
		opts.BackendURL = os.Getenv("ZEN_BACKEND_URL")
	}
	if opts.BackendURL == "" {
		opts.BackendURL = "http://127.0.0.1:38000"
	}
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.InstanceID == "" {
		opts.InstanceID = ulidutil.New()
	}

	// framingo's logger writes its console core to os.Stdout, but stdout is the
	// JSON-RPC framing pipe and must stay clean — so we build the channel
	// logger with NoStdout and let records flow only to a per-instance file
	// under LogDir. Without a LogDir there is no file core, so the channel
	// stays silent.
	var logFile string
	if opts.LogDir != "" {
		logFile = filepath.Join(opts.LogDir, "channel-"+opts.InstanceID+".log")
	}
	logger := log.New(
		log.WithLevel(int(zapcore.DebugLevel)),
		log.NoStdout(),
		log.WithFileWriter(logFile, 50, 3, 7),
	).With("instance", opts.InstanceID)

	logger.Infof("instance %s starting (backend %s)", opts.InstanceID, opts.BackendURL)

	// The backend client must share the channel's NoStdout logger. Left on its
	// default it writes request traces to stdout, interleaving console-format
	// lines into the JSON-RPC framing pipe.
	backend := zenbackend.New(
		strings.TrimRight(opts.BackendURL, "/")+"/api/v1",
		zenbackend.WithLogger(logger),
	)

	cwd, err := os.Getwd()
	if err != nil {
		logger.Warnf("cannot determine cwd: %v", err)
		cwd = ""
	}
	sessionID := resolveSessionID(opts.InstanceID, logger)

	server := &StdioServer{
		In:    opts.In,
		Out:   opts.Out,
		Reply: NewReply(backend, sessionID, cwd),
		Log:   logger,
	}

	sub := &Subscriber{
		BaseURL: opts.BackendURL,
		Backend: backend,
		Push:    server.PushChannelNotification,
		Log:     logger,
		Registration: entity.ChannelRegistration{
			Kind:       entity.ChannelRegistrationKind,
			InstanceID: opts.InstanceID,
			SessionID:  sessionID,
			Cwd:        cwd,
			StartedAt:  time.Now(),
		},
		// Filled at dial time; sub.Run only dials after server.Ready().
		ClientInfo: server.ClientInfo,
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	go func() { errCh <- server.Run(runCtx) }()
	go func() {
		// Wait for the MCP handshake before dialing. Until `initialize` lands
		// there is no client to receive notifications, and a frame written to
		// stdout early corrupts the JSON-RPC stream.
		select {
		case <-server.Ready():
		case <-runCtx.Done():
			errCh <- runCtx.Err()
			return
		}
		logger.Infof("MCP handshake complete; subscribing to backend")
		errCh <- sub.Run(runCtx)
	}()

	// Either goroutine returning tears down the other so RunChannel exits
	// promptly. First error wins; second is drained.
	err = <-errCh
	cancel()
	<-errCh
	if err == context.Canceled {
		return nil
	}
	return err
}
