package conversation_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/labstack/echo/v4"
	frmessagebus "github.com/xhanio/framingo/pkg/services/messagebus"
	frpubsub "github.com/xhanio/framingo/pkg/services/pubsub"
	frdriver "github.com/xhanio/framingo/pkg/services/pubsub/driver"
	frlog "github.com/xhanio/framingo/pkg/utils/log"

	"github.com/xhanio/zen/pkg/utils/busutil"

	conversationrouter "github.com/xhanio/zen/pkg/routers/conversation"
	conversationsvc "github.com/xhanio/zen/pkg/services/conversation"
	presencesvc "github.com/xhanio/zen/pkg/services/presence"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/model"
)

// newEchoWithDriver is newEchoWithPresence plus a handle on the pubsub driver,
// so a test can drop a subscription the way the driver drops a laggard: by
// closing its channel out from under the handler.
func newEchoWithDriver(t *testing.T) (*echo.Echo, presencesvc.Manager, conversationsvc.Manager, frdriver.Driver) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))

	drv := busutil.NewDriver(frlog.Default)
	bus := frmessagebus.New(frpubsub.New(drv))
	if err := bus.Start(context.Background()); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(func() { _ = bus.Stop(true) })

	svc := conversationsvc.New(repo, conversationsvc.WithMessageBus(bus))
	pres := presencesvc.New()
	r := conversationrouter.NewForTestWithPresence(svc, bus, pres)

	e := echo.New()
	e.GET("/conversations/:id/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.WSConversation(api.WrapContext(c), conn)
	})
	e.GET("/conversations/_stream/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.StreamWS(api.WrapContext(c), conn)
	})
	return e, pres, svc, drv
}

// subscribersWithPrefix returns the current subscriber names carrying prefix.
func subscribersWithPrefix(drv frdriver.Driver, prefix string) []string {
	var out []string
	for _, name := range drv.GetSubscribers(frmessagebus.DefaultTopic) {
		if strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return out
}

// subscriberWithPrefix polls until exactly one subscriber whose name starts
// with prefix exists, then returns its name.
func subscriberWithPrefix(t *testing.T, drv frdriver.Driver, prefix string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := subscribersWithPrefix(drv, prefix); len(got) == 1 {
			return got[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no single subscriber with prefix %q appeared", prefix)
	return ""
}

// waitForSoleSubscriber polls until exactly one "stream:" subscriber remains and
// it is not `gone`. Waiting on the count alone races: a replacement subscribes
// only after its Register has already evicted its predecessor, so the count
// passes through 1 with neither socket subscribed.
func waitForSoleSubscriber(t *testing.T, drv frdriver.Driver, gone string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got := subscribersWithPrefix(drv, "stream:")
		if len(got) == 1 && got[0] != gone {
			return got[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("never settled on a single live subscriber (have %v, want one that is not %q)",
		subscribersWithPrefix(drv, "stream:"), gone)
	return ""
}

// assertClosedByServer fails unless the peer sent a close frame. A read
// deadline expiring means the socket is still open — which is the bug.
func assertClosedByServer(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, _, err := conn.Read(ctx)
		if err == nil {
			continue // a queued data frame; keep reading for the close
		}
		if websocket.CloseStatus(err) == -1 {
			t.Fatalf("socket was not closed by the server; read failed with %v "+
				"(the subscription is gone but the client still believes it is subscribed)", err)
		}
		return
	}
}

// The driver may drop a subscriber that stops draining: it closes the
// subscription channel. When that happens the handler must hang up, because a
// client holding an open socket with a dead subscription receives nothing and
// has no reason to reconnect. Under DropSubscriber a lost frame is recoverable;
// a silently deaf socket is not.
func TestWSConversation_DroppedSubscriptionHangsUpTheSocket(t *testing.T) {
	e, _, svc, drv := newEchoWithDriver(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	conv, err := svc.Create(context.Background(), "x", nil, nil)
	if err != nil {
		t.Fatalf("Create conv: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/conversations/"+conv.ID+"/ws", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()

	name := subscriberWithPrefix(t, drv, "ws:")
	if err := drv.Unsubscribe(name, frmessagebus.DefaultTopic); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	assertClosedByServer(t, conn)
}

func TestStreamWS_DroppedSubscriptionHangsUpTheSocket(t *testing.T) {
	e, pres, _, drv := newEchoWithDriver(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()
	if err := wsjson.Write(ctx, conn, registerFrame("i1", "s1")); err != nil {
		t.Fatalf("register: %v", err)
	}
	waitFor(t, func() bool { return len(pres.List()) == 1 })

	name := subscriberWithPrefix(t, drv, "stream:")
	if err := drv.Unsubscribe(name, frmessagebus.DefaultTopic); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	assertClosedByServer(t, conn)
}

// A channel process keeps one InstanceID for its lifetime and redials on every
// transient error, so the server can briefly hold two sockets for the same
// InstanceID. The stale one tears down second. Neither its deferred
// messenger.Close() (which unsubscribes BY NAME) nor its deferred
// presence.Unregister (which matches BY InstanceID) may take the live
// connection down with it.
func TestStreamWS_SameInstanceReconnect_StaleSocketDoesNotKillItsReplacement(t *testing.T) {
	e, pres, svc, drv := newEchoWithDriver(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	url := "ws" + srv.URL[4:] + "/conversations/_stream/ws"

	stale, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Dial stale: %v", err)
	}
	defer stale.CloseNow()
	if err := wsjson.Write(ctx, stale, registerFrame("i1", "s1")); err != nil {
		t.Fatalf("register stale: %v", err)
	}
	waitFor(t, func() bool { return len(pres.List()) == 1 })
	staleSub := subscriberWithPrefix(t, drv, "stream:")

	// The same process redials. Same InstanceID, same SessionID.
	live, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Dial live: %v", err)
	}
	defer live.CloseNow()
	if err := wsjson.Write(ctx, live, registerFrame("i1", "s1")); err != nil {
		t.Fatalf("register live: %v", err)
	}

	// The stale socket is hung up, and its handler runs its deferred cleanup.
	assertClosedByServer(t, stale)
	waitForSoleSubscriber(t, drv, staleSub)

	// The session must still be present: the live socket is holding it.
	if !pres.Has("s1") {
		t.Fatal("stale socket's deferred Unregister removed the live registration")
	}

	// And the live socket must still receive events addressed to that session.
	conv, err := svc.Create(context.Background(), "x", nil, nil)
	if err != nil {
		t.Fatalf("Create conv: %v", err)
	}
	if _, err := svc.AppendMessage(context.Background(), conv.ID, "user", "ping", nil,
		model.WithTargetSession("s1")); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(ctx, 3*time.Second)
	defer readCancel()
	var got map[string]any
	if err := wsjson.Read(readCtx, live, &got); err != nil {
		t.Fatalf("live socket received nothing (%v): the stale socket's "+
			"messenger.Close() unsubscribed it by name", err)
	}
	if got["content"] != "ping" {
		t.Fatalf("live socket got %+v, want the ping event", got)
	}
}
