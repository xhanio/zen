package conversation_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
)

// assertSilent fails if conn delivers anything within the window. A read that
// ends in DeadlineExceeded is the pass: nothing arrived. Any other error means
// the socket broke, which proves nothing about routing.
func assertSilent(t *testing.T, ctx context.Context, conn *websocket.Conn, window time.Duration) {
	t.Helper()
	readCtx, cancel := context.WithTimeout(ctx, window)
	defer cancel()
	var leaked entity.ConversationEvent
	err := wsjson.Read(readCtx, conn, &leaked)
	if err == nil {
		t.Fatalf("received an event addressed to %q: %+v", leaked.TargetSessionID, leaked)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("read failed for the wrong reason (proves nothing): %v", err)
	}
}

// A user message names exactly one session. The backend must deliver it to that
// session's channel and to no other.
//
// Broadcasting to every channel and letting each one drop what is not its own
// makes correctness a property of the client: a channel built before the filter
// existed, or one that simply chooses not to filter, answers a message meant for
// someone else. It also hands one session's text to every other session's
// process, which no amount of client-side discipline can take back.
//
// The channel keeps its own filter. This is the layer that must not need it.
func TestStreamWS_DeliversOnlyToTheAddressedSession(t *testing.T) {
	e, svc, pres, _ := newEchoRESTWithPresence(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wsURL := "ws" + srv.URL[4:] + "/conversations/_stream/ws"

	addressed := dialStream(t, ctx, wsURL, "i1", "sess-A")
	defer addressed.CloseNow()
	bystander := dialStream(t, ctx, wsURL, "i2", "sess-B")
	defer bystander.CloseNow()
	waitFor(t, func() bool { return len(pres.List()) == 2 })

	c, _ := svc.Create(context.Background(), "x", nil, nil)
	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "hi", nil,
		model.WithTargetSession("sess-A")); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	// The addressed channel receives it, target intact.
	var got entity.ConversationEvent
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	err := wsjson.Read(readCtx, addressed, &got)
	readCancel()
	if err != nil {
		t.Fatalf("addressed channel received nothing: %v", err)
	}
	if got.TargetSessionID != "sess-A" || got.Content != "hi" {
		t.Fatalf("addressed channel got %+v, want the sess-A event", got)
	}

	// The bystander never sees it — not even to drop it.
	assertSilent(t, ctx, bystander, time.Second)
}

// An empty target is addressed to nobody: it is the "no session was connected
// when I posted" path, and it must reach no channel at all. Strict equality
// gives this for free, which is why the filter is written as equality rather
// than as "skip when the target is set and differs".
func TestStreamWS_UntargetedMessageReachesNoChannel(t *testing.T) {
	e, svc, pres, _ := newEchoRESTWithPresence(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wsURL := "ws" + srv.URL[4:] + "/conversations/_stream/ws"

	conn := dialStream(t, ctx, wsURL, "i1", "sess-A")
	defer conn.CloseNow()
	waitFor(t, func() bool { return len(pres.List()) == 1 })

	c, _ := svc.Create(context.Background(), "x", nil, nil)
	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "to nobody", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	assertSilent(t, ctx, conn, time.Second)
}
