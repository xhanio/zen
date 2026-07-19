package conversation_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/labstack/echo/v4"
	frmessagebus "github.com/xhanio/framingo/pkg/services/messagebus"
	frpubsub "github.com/xhanio/framingo/pkg/services/pubsub"
	frlog "github.com/xhanio/framingo/pkg/utils/log"

	conversationrouter "github.com/xhanio/zen/pkg/routers/conversation"
	conversationsvc "github.com/xhanio/zen/pkg/services/conversation"
	presencesvc "github.com/xhanio/zen/pkg/services/presence"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/utils/busutil"
)

// newStreamEchoLogging mounts /_stream/ws against a router whose logger writes
// to a file the test can read back.
func newStreamEchoLogging(t *testing.T, timeout time.Duration) (*echo.Echo, string) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "app.log")
	logger := frlog.New(
		frlog.WithLevel(0),
		frlog.WithFileWriter(logPath, 1, 1, 1),
		frlog.NoStdout(),
	)

	repo := repository.New(testutil.NewDB(t))
	bus := frmessagebus.New(frpubsub.New(busutil.NewDriver(frlog.Default)))
	if err := bus.Start(context.Background()); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(func() { _ = bus.Stop(true) })

	r := conversationrouter.NewForTestWithPresence(
		conversationsvc.New(repo, conversationsvc.WithMessageBus(bus)),
		bus, presencesvc.New(),
	)
	r.SetRegistrationTimeout(timeout)
	r.SetLogger(logger)

	e := echo.New()
	e.GET("/conversations/_stream/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.StreamWS(api.WrapContext(c), conn)
	})
	return e, logPath
}

// A registration can fail three ways, and the log must tell them apart. Two of
// them mean a peer is still there and misbehaving — worth reading. The third,
// a peer that simply hung up, is what every health probe and port scanner does.
//
// None of them may drag a stack trace along: the failure is always at the same
// line, so the trace says nothing the message does not, and it buries the two
// cases that matter under whatever a scanner generates.
func TestStreamWS_RegistrationFailuresAreDistinctAndStackFree(t *testing.T) {
	cases := []struct {
		name string
		// act drives the client side after the socket is up.
		act  func(t *testing.T, ctx context.Context, conn *websocket.Conn)
		want string
	}{
		{
			name: "peer hangs up before registering",
			act: func(_ *testing.T, _ context.Context, conn *websocket.Conn) {
				_ = conn.CloseNow() // what `curl -m 3` does
			},
			want: "peer closed before registering",
		},
		{
			name: "peer stays silent",
			act: func(_ *testing.T, ctx context.Context, conn *websocket.Conn) {
				<-ctx.Done() // say nothing until the server gives up
			},
			want: "no registration frame within 300ms",
		},
		{
			name: "first frame is not a registration",
			act: func(t *testing.T, ctx context.Context, conn *websocket.Conn) {
				if err := wsjson.Write(ctx, conn, map[string]string{"kind": "something-else"}); err != nil {
					t.Fatalf("write: %v", err)
				}
			},
			// The JSON encoder escapes the quotes around the kind, so match the
			// unescaped part.
			want: "first frame kind",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, logPath := newStreamEchoLogging(t, 300*time.Millisecond)
			srv := httptest.NewServer(e)
			t.Cleanup(srv.Close)

			dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(dialCtx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", nil)
			if err != nil {
				t.Fatalf("Dial: %v", err)
			}
			defer conn.CloseNow()

			actCtx, actCancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
			defer actCancel()
			tc.act(t, actCtx, conn)

			out := waitForLogLine(t, logPath, tc.want)

			// Still a warning: a channel that cannot register is worth seeing.
			if !strings.Contains(out, `"level":"warn"`) {
				t.Errorf("want a warning, got: %s", out)
			}
			for _, frame := range []string{"handler.go:", "goroutine", "/pkg/mod/", "runtime.goexit"} {
				if strings.Contains(out, frame) {
					t.Fatalf("log carries a stack trace (found %q):\n%s", frame, out)
				}
			}
		})
	}
}

// waitForLogLine polls until a line containing want appears, then returns the
// whole file: a stack trace lands on the lines after the message, so the entire
// buffer is what must be inspected for one.
func waitForLogLine(t *testing.T, path, want string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(b), want) {
			// Let any trailing frames flush, so their absence is a fact rather
			// than a race the test happened to win.
			time.Sleep(100 * time.Millisecond)
			b, _ = os.ReadFile(path)
			return string(b)
		}
		time.Sleep(20 * time.Millisecond)
	}
	b, _ := os.ReadFile(path)
	t.Fatalf("no log line containing %q within 3s; log was:\n%s", want, b)
	return ""
}
