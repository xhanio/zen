package presence_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/labstack/echo/v4"

	presencerouter "github.com/xhanio/zen/pkg/routers/presence"
	deliverysvc "github.com/xhanio/zen/pkg/services/delivery"
	presencesvc "github.com/xhanio/zen/pkg/services/presence"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

func newEcho(t *testing.T) (*echo.Echo, presencesvc.Manager, deliverysvc.Manager) {
	t.Helper()
	pres := presencesvc.New()
	del := deliverysvc.New()
	r := presencerouter.NewForTest(pres, del)

	e := echo.New()
	e.GET("/_sessions/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.SessionsWS(api.WrapContext(c), conn)
	})
	return e, pres, del
}

func chn(instance, session, cwd string) *entity.Channel {
	return &entity.Channel{
		InstanceID:  instance,
		SessionID:   session,
		Cwd:         cwd,
		StartedAt:   time.Now(),
		ConnectedAt: time.Now(),
	}
}

func dial(t *testing.T, ctx context.Context, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/_sessions/ws", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	return conn
}

func readFrame(t *testing.T, ctx context.Context, conn *websocket.Conn) api.SessionsFrame {
	t.Helper()
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var f api.SessionsFrame
	if err := wsjson.Read(readCtx, conn, &f); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	return f
}

// A client that connects after channels are already live must see them, not
// wait for the next change. Otherwise a page refresh shows "no sessions".
func TestSessionsWS_SendsSnapshotOnConnect(t *testing.T) {
	e, pres, _ := newEcho(t)
	pres.Register(chn("i1", "s1", "/repo"))

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dial(t, ctx, srv)
	defer conn.CloseNow()

	f := readFrame(t, ctx, conn)
	if f.Kind != api.SessionsFrameKind {
		t.Fatalf("kind = %q, want %q", f.Kind, api.SessionsFrameKind)
	}
	if len(f.Sessions) != 1 || f.Sessions[0].SessionID != "s1" {
		t.Fatalf("sessions = %+v", f.Sessions)
	}
}

func TestSessionsWS_PushesOnRegister(t *testing.T) {
	e, pres, _ := newEcho(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dial(t, ctx, srv)
	defer conn.CloseNow()

	if f := readFrame(t, ctx, conn); len(f.Sessions) != 0 {
		t.Fatalf("initial snapshot = %+v, want empty", f.Sessions)
	}

	pres.Register(chn("i1", "s1", "/repo"))

	f := readFrame(t, ctx, conn)
	if len(f.Sessions) != 1 || f.Sessions[0].InstanceID != "i1" {
		t.Fatalf("sessions = %+v", f.Sessions)
	}
}

func TestSessionsWS_PushesOnUnregister(t *testing.T) {
	e, pres, _ := newEcho(t)
	reg := pres.Register(chn("i1", "s1", "/repo"))

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dial(t, ctx, srv)
	defer conn.CloseNow()
	_ = readFrame(t, ctx, conn) // initial snapshot

	pres.Unregister(reg)

	f := readFrame(t, ctx, conn)
	if len(f.Sessions) != 0 {
		t.Fatalf("sessions = %+v, want empty after unregister", f.Sessions)
	}
}

// Two browser tabs each get their own watcher.
func TestSessionsWS_MultipleClientsEachSeeChanges(t *testing.T) {
	e, pres, _ := newEcho(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a := dial(t, ctx, srv)
	defer a.CloseNow()
	b := dial(t, ctx, srv)
	defer b.CloseNow()
	_ = readFrame(t, ctx, a)
	_ = readFrame(t, ctx, b)

	pres.Register(chn("i1", "s1", "/repo"))

	if f := readFrame(t, ctx, a); len(f.Sessions) != 1 {
		t.Fatalf("client a saw %+v", f.Sessions)
	}
	if f := readFrame(t, ctx, b); len(f.Sessions) != 1 {
		t.Fatalf("client b saw %+v", f.Sessions)
	}
}

// The store distinguishes frame types by `kind`; M4 adds a second one.
func TestSessionsFrame_CarriesKindDiscriminator(t *testing.T) {
	if api.SessionsFrameKind != "sessions" {
		t.Fatalf("SessionsFrameKind = %q", api.SessionsFrameKind)
	}
}

func TestSessionsWS_EmitsDeliveryFrame(t *testing.T) {
	e, _, del := newEcho(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dial(t, ctx, srv)
	defer conn.CloseNow()
	_ = readFrame(t, ctx, conn) // initial sessions snapshot

	del.Ack("01MSG")

	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	var f api.DeliveryFrame
	if err := wsjson.Read(readCtx, conn, &f); err != nil {
		t.Fatalf("read delivery frame: %v", err)
	}
	if f.Kind != api.DeliveryFrameKind {
		t.Fatalf("kind = %q, want %q", f.Kind, api.DeliveryFrameKind)
	}
	if f.MessageID != "01MSG" || f.State != entity.DeliveryStateDelivered {
		t.Fatalf("frame = %+v", f)
	}
}

// Sessions and delivery share one socket, distinguished only by kind.
func TestSessionsWS_SessionsAndDeliveryShareTheStream(t *testing.T) {
	e, pres, del := newEcho(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dial(t, ctx, srv)
	defer conn.CloseNow()
	_ = readFrame(t, ctx, conn) // initial snapshot

	pres.Register(chn("i1", "s1", "/repo"))
	del.Ack("01MSG")

	kinds := map[string]bool{}
	for i := 0; i < 2; i++ {
		readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
		var probe struct {
			Kind string `json:"kind"`
		}
		err := wsjson.Read(readCtx, conn, &probe)
		readCancel()
		if err != nil {
			t.Fatalf("read frame %d: %v", i, err)
		}
		kinds[probe.Kind] = true
	}
	if !kinds[api.SessionsFrameKind] || !kinds[api.DeliveryFrameKind] {
		t.Fatalf("saw kinds %v, want both sessions and delivery", kinds)
	}
}

// Two tabs each get their own delivery stream.
func TestSessionsWS_DeliveryReachesEveryClient(t *testing.T) {
	e, _, del := newEcho(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a := dial(t, ctx, srv)
	defer a.CloseNow()
	b := dial(t, ctx, srv)
	defer b.CloseNow()
	_ = readFrame(t, ctx, a)
	_ = readFrame(t, ctx, b)

	del.Ack("01MSG")

	for i, conn := range []*websocket.Conn{a, b} {
		readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
		var f api.DeliveryFrame
		err := wsjson.Read(readCtx, conn, &f)
		readCancel()
		if err != nil {
			t.Fatalf("client %d: %v", i, err)
		}
		if f.MessageID != "01MSG" {
			t.Fatalf("client %d frame = %+v", i, f)
		}
	}
}
