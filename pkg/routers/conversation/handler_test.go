package conversation_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	frmessagebus "github.com/xhanio/framingo/pkg/services/messagebus"
	frpubsub "github.com/xhanio/framingo/pkg/services/pubsub"
	fapi "github.com/xhanio/framingo/pkg/types/api"
	fmodel "github.com/xhanio/framingo/pkg/types/model"
	frlog "github.com/xhanio/framingo/pkg/utils/log"

	conversationrouter "github.com/xhanio/zen/pkg/routers/conversation"
	conversationsvc "github.com/xhanio/zen/pkg/services/conversation"
	deliverysvc "github.com/xhanio/zen/pkg/services/delivery"
	presencesvc "github.com/xhanio/zen/pkg/services/presence"
	"github.com/xhanio/zen/pkg/services/repository"
	"github.com/xhanio/zen/pkg/services/repository/testutil"
	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/busutil"
)

type validatorWrap struct{ v *validator.Validate }

func (w *validatorWrap) Validate(i any) error { return w.v.Struct(i) }

func newEchoWithConversationRouter(t *testing.T) *echo.Echo {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))
	svc := conversationsvc.New(repo)
	r := conversationrouter.NewForTest(svc)
	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.POST("/conversations", api.WrapHandler(r.CreateConversation))
	e.GET("/conversations", api.WrapHandler(r.ListConversations))
	e.GET("/conversations/:id", api.WrapHandler(r.GetConversation))
	e.PUT("/conversations/:id", api.WrapHandler(r.UpdateConversationTitle))
	e.DELETE("/conversations/:id", api.WrapHandler(r.DeleteConversation))
	e.GET("/conversations/:id/messages", api.WrapHandler(r.ListMessages))
	e.POST("/conversations/:id/messages", api.WrapHandler(r.AppendMessage))
	return e
}

func postJSON(t *testing.T, e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestCreateConversation_HTTP(t *testing.T) {
	e := newEchoWithConversationRouter(t)
	rec := postJSON(t, e, http.MethodPost, "/conversations", `{"title":"my chat"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST got %d: %s", rec.Code, rec.Body.String())
	}
	var c entity.Conversation
	if err := json.NewDecoder(rec.Body).Decode(&c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.ID == "" || c.Title != "my chat" {
		t.Fatalf("bad response: %+v", c)
	}
}

func TestCreateConversation_HalfAnchorRejected_HTTP(t *testing.T) {
	e := newEchoWithConversationRouter(t)
	rec := postJSON(t, e, http.MethodPost, "/conversations", `{"title":"x","anchor_kind":"card"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAppendMessage_HTTP_AutoSetsTitle(t *testing.T) {
	e := newEchoWithConversationRouter(t)
	rec := postJSON(t, e, http.MethodPost, "/conversations", `{"title":""}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create conv: %d %s", rec.Code, rec.Body.String())
	}
	var c entity.Conversation
	_ = json.NewDecoder(rec.Body).Decode(&c)

	rec = postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"user","content":"What is X?","selection_text":"X is..."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("append: %d %s", rec.Code, rec.Body.String())
	}
	var m entity.Message
	_ = json.NewDecoder(rec.Body).Decode(&m)
	if m.Role != "user" || m.Content != "What is X?" {
		t.Fatalf("bad msg: %+v", m)
	}

	req := httptest.NewRequest(http.MethodGet, "/conversations/"+c.ID, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var got entity.Conversation
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.Title != "What is X?" {
		t.Fatalf("title not auto-set: %q", got.Title)
	}
}

func TestListConversations_Pending_HTTP(t *testing.T) {
	e := newEchoWithConversationRouter(t)

	rec := postJSON(t, e, http.MethodPost, "/conversations", `{"title":"a"}`)
	var a entity.Conversation
	_ = json.NewDecoder(rec.Body).Decode(&a)
	_ = postJSON(t, e, http.MethodPost, "/conversations/"+a.ID+"/messages",
		`{"role":"user","content":"q1"}`)

	rec = postJSON(t, e, http.MethodPost, "/conversations", `{"title":"b"}`)
	var b entity.Conversation
	_ = json.NewDecoder(rec.Body).Decode(&b)
	_ = postJSON(t, e, http.MethodPost, "/conversations/"+b.ID+"/messages",
		`{"role":"user","content":"q1"}`)
	_ = postJSON(t, e, http.MethodPost, "/conversations/"+b.ID+"/messages",
		`{"role":"assistant","content":"a1"}`)

	req := httptest.NewRequest(http.MethodGet, "/conversations?pending=true", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("List pending: %d %s", rec.Code, rec.Body.String())
	}
	type listResp struct {
		Conversations    []*entity.Conversation `json:"conversations"`
		UnansweredCounts []int                  `json:"unanswered_counts"`
	}
	var got listResp
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if len(got.Conversations) != 1 || got.Conversations[0].ID != a.ID {
		t.Fatalf("expected only conversation a pending, got %+v", got.Conversations)
	}
	if len(got.UnansweredCounts) != 1 || got.UnansweredCounts[0] != 1 {
		t.Fatalf("expected unanswered=[1], got %v", got.UnansweredCounts)
	}
}

func TestDeleteConversation_HTTP(t *testing.T) {
	e := newEchoWithConversationRouter(t)
	rec := postJSON(t, e, http.MethodPost, "/conversations", `{"title":"x"}`)
	var c entity.Conversation
	_ = json.NewDecoder(rec.Body).Decode(&c)

	req := httptest.NewRequest(http.MethodDelete, "/conversations/"+c.ID, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/conversations/"+c.ID, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}

// newEchoWithBus is like newEchoWithConversationRouter but wires pubsub +
// messagebus so the WS path and long-poll can be exercised end-to-end.
func newEchoWithBus(t *testing.T) (*echo.Echo, conversationsvc.Manager, fmodel.MessageBus) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))

	bus := frmessagebus.New(frpubsub.New(busutil.NewDriver(frlog.Default)))
	if err := bus.Start(context.Background()); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(func() { _ = bus.Stop(true) })

	svc := conversationsvc.New(repo, conversationsvc.WithMessageBus(bus))
	r := conversationrouter.NewForTestWithBus(svc, bus)

	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.POST("/conversations", api.WrapHandler(r.CreateConversation))
	e.GET("/conversations/:id", api.WrapHandler(r.GetConversation))
	e.POST("/conversations/:id/messages", api.WrapHandler(r.AppendMessage))
	e.GET("/conversations/:id/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.WSConversation(api.WrapContext(c), conn)
	})
	return e, svc, bus
}

// newEchoWithPresence mounts only /_stream/ws, against a router that knows
// about presence. regTimeout of 0 leaves the production default; the tests that
// assert on an unregistered socket pass a short one so they do not wait 5s.
func newEchoWithPresence(t *testing.T, regTimeout time.Duration) (*echo.Echo, presencesvc.Manager) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))

	bus := frmessagebus.New(frpubsub.New(busutil.NewDriver(frlog.Default)))
	if err := bus.Start(context.Background()); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(func() { _ = bus.Stop(true) })

	svc := conversationsvc.New(repo, conversationsvc.WithMessageBus(bus))
	pres := presencesvc.New()
	r := conversationrouter.NewForTestWithPresence(svc, bus, pres)
	if regTimeout > 0 {
		r.SetRegistrationTimeout(regTimeout)
	}

	e := echo.New()
	e.GET("/conversations/_stream/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.StreamWS(api.WrapContext(c), conn)
	})
	return e, pres
}

// newEchoRESTWithPresence mounts the REST conversation routes against a router
// that knows about presence, so target validation can be exercised. It also
// mounts the fan-out stream, so one helper can drive REST and observe what each
// channel receives — which TestStreamWS_DeliversOnlyToTheAddressedSession needs.
func newEchoRESTWithPresence(t *testing.T) (*echo.Echo, conversationsvc.Manager, presencesvc.Manager, deliverysvc.Manager) {
	t.Helper()
	repo := repository.New(testutil.NewDB(t))

	bus := frmessagebus.New(frpubsub.New(busutil.NewDriver(frlog.Default)))
	if err := bus.Start(context.Background()); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(func() { _ = bus.Stop(true) })

	svc := conversationsvc.New(repo, conversationsvc.WithMessageBus(bus))
	pres := presencesvc.New()
	del := deliverysvc.New()
	r := conversationrouter.NewForTestWithDelivery(svc, bus, pres, del)

	e := echo.New()
	e.Validator = &validatorWrap{v: validator.New()}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}
		ae := fapi.WrapError(err, c)
		_ = c.JSON(ae.Status, ae)
	}
	e.POST("/conversations", api.WrapHandler(r.CreateConversation))
	e.POST("/conversations/:id/messages", api.WrapHandler(r.AppendMessage))
	e.POST("/conversations/:id/messages/:mid/dispatch", api.WrapHandler(r.DispatchMessage))
	e.GET("/conversations/_stream/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.StreamWS(api.WrapContext(c), conn)
	})
	return e, svc, pres, del
}

func liveChannel(sessionID string) *entity.Channel {
	return &entity.Channel{
		InstanceID: "i-" + sessionID, SessionID: sessionID, Cwd: "/repo",
		StartedAt: time.Now(), ConnectedAt: time.Now(),
	}
}

func TestAppendMessage_AcceptsLiveTarget(t *testing.T) {
	e, svc, pres, _ := newEchoRESTWithPresence(t)
	pres.Register(liveChannel("sess-A"))
	c, _ := svc.Create(context.Background(), "x", nil, nil)

	rec := postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"user","content":"hi","target_session_id":"sess-A"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
}

// A message addressed to a session that is gone must not vanish silently.
func TestAppendMessage_409OnDeadTarget(t *testing.T) {
	e, svc, _, _ := newEchoRESTWithPresence(t)
	c, _ := svc.Create(context.Background(), "x", nil, nil)

	rec := postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"user","content":"hi","target_session_id":"sess-gone"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

// A null target skips validation entirely and posts undelivered. That is the
// no-session-connected path, not an error.
func TestAppendMessage_NullTargetPostsUndelivered(t *testing.T) {
	e, svc, _, _ := newEchoRESTWithPresence(t)
	c, _ := svc.Create(context.Background(), "x", nil, nil)

	rec := postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"user","content":"hi"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body.String())
	}
}

func TestAppendMessage_UserSnapshotsSessionCwd(t *testing.T) {
	e, svc, pres, _ := newEchoRESTWithPresence(t)
	pres.Register(liveChannel("sess-A")) // liveChannel sets Cwd "/repo"
	c, _ := svc.Create(context.Background(), "x", nil, nil)

	rec := postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"user","content":"hi","target_session_id":"sess-A"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
	var msg entity.Message
	_ = json.Unmarshal(rec.Body.Bytes(), &msg)
	if msg.SessionID == nil || *msg.SessionID != "sess-A" {
		t.Fatalf("session_id = %v", msg.SessionID)
	}
	if msg.SessionCwd == nil || *msg.SessionCwd != "/repo" {
		t.Fatalf("session_cwd = %v", msg.SessionCwd)
	}
}

func TestAppendMessage_AssistantAttributesOwnSession(t *testing.T) {
	e, svc, _, _ := newEchoRESTWithPresence(t)
	c, _ := svc.Create(context.Background(), "x", nil, nil)

	rec := postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"assistant","content":"hi","session_id":"sess-A","session_cwd":"/repo"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
	var msg entity.Message
	_ = json.Unmarshal(rec.Body.Bytes(), &msg)
	if msg.SessionID == nil || *msg.SessionID != "sess-A" {
		t.Fatalf("session_id = %v", msg.SessionID)
	}
	if msg.SessionCwd == nil || *msg.SessionCwd != "/repo" {
		t.Fatalf("session_cwd = %v", msg.SessionCwd)
	}
}

func TestDispatch_FillsSessionSetOnce(t *testing.T) {
	e, svc, pres, _ := newEchoRESTWithPresence(t)
	pres.Register(liveChannel("sess-A"))
	pres.Register(liveChannel("sess-B"))
	c, _ := svc.Create(context.Background(), "x", nil, nil)

	// Posted with no session connected → undelivered, session null.
	rec := postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"user","content":"hi"}`)
	var msg entity.Message
	_ = json.Unmarshal(rec.Body.Bytes(), &msg)
	if msg.SessionID != nil {
		t.Fatalf("fresh post should have null session, got %v", msg.SessionID)
	}

	// Dispatch to A fills the session.
	rec = postJSON(t, e, http.MethodPost,
		"/conversations/"+c.ID+"/messages/"+msg.ID+"/dispatch",
		`{"target_session_id":"sess-A"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("dispatch A: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := svc.GetMessages(context.Background(), c.ID, 0)
	if got[0].SessionID == nil || *got[0].SessionID != "sess-A" {
		t.Fatalf("after dispatch A: %v", got[0].SessionID)
	}

	// Dispatch to B is a delivery retry; the stored session stays A.
	rec = postJSON(t, e, http.MethodPost,
		"/conversations/"+c.ID+"/messages/"+msg.ID+"/dispatch",
		`{"target_session_id":"sess-B"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("dispatch B: %d %s", rec.Code, rec.Body.String())
	}
	got, _ = svc.GetMessages(context.Background(), c.ID, 0)
	if *got[0].SessionID != "sess-A" {
		t.Fatalf("session re-pointed to %q, want sess-A", *got[0].SessionID)
	}
}

// The assistant reply comes from the channel and addresses nobody.
func TestAppendMessage_AssistantNeedsNoTarget(t *testing.T) {
	e, svc, _, _ := newEchoRESTWithPresence(t)
	c, _ := svc.Create(context.Background(), "x", nil, nil)

	rec := postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
		`{"role":"assistant","content":"answer"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
}

func registerFrame(instance, session string) entity.ChannelRegistration {
	return entity.ChannelRegistration{
		Kind:       entity.ChannelRegistrationKind,
		InstanceID: instance,
		SessionID:  session,
		Cwd:        "/repo",
		StartedAt:  time.Now(),
		ClientName: "claude-code",
		ClientVer:  "2.1.205",
	}
}

// dialStream opens a /_stream/ws connection and completes the registration
// handshake, which the server now requires before it will subscribe the socket.
func dialStream(t *testing.T, ctx context.Context, wsURL, instance, session string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Dial %s: %v", wsURL, err)
	}
	if err := wsjson.Write(ctx, conn, registerFrame(instance, session)); err != nil {
		t.Fatalf("register %s: %v", instance, err)
	}
	return conn
}

// waitFor polls cond for up to a second. The registration happens on the
// server's goroutine, so the test cannot assert immediately after the write.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within 1s")
}

func TestStreamWS_RegistersOnFrame(t *testing.T) {
	e, pres := newEchoWithPresence(t, 0)
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
		t.Fatalf("write registration: %v", err)
	}

	waitFor(t, func() bool {
		got := pres.List()
		return len(got) == 1 && got[0].SessionID == "s1"
	})
}

func TestStreamWS_UnregistersOnClose(t *testing.T) {
	e, pres := newEchoWithPresence(t, 0)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if err := wsjson.Write(ctx, conn, registerFrame("i1", "s1")); err != nil {
		t.Fatalf("write registration: %v", err)
	}
	waitFor(t, func() bool { return len(pres.List()) == 1 })

	conn.CloseNow()
	waitFor(t, func() bool { return len(pres.List()) == 0 })
}

func TestStreamWS_ClosesConnectionWithoutRegistration(t *testing.T) {
	e, pres := newEchoWithPresence(t, 200*time.Millisecond)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()

	// Never send a registration frame. The server must hang up, and must not
	// register anything. Assert on the close status rather than merely on
	// err != nil: a read that fails because the test's own context expired
	// would otherwise look like a pass.
	_, _, err = conn.Read(ctx)
	if got := websocket.CloseStatus(err); got != websocket.StatusPolicyViolation {
		t.Fatalf("close status = %v (err %v), want StatusPolicyViolation", got, err)
	}
	if got := pres.List(); len(got) != 0 {
		t.Fatalf("List() = %+v, want empty", got)
	}
}

func TestStreamWS_FirstFrameMustBeRegistration(t *testing.T) {
	e, pres := newEchoWithPresence(t, 200*time.Millisecond)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()

	// A well-formed JSON frame that is not a registration.
	if err := wsjson.Write(ctx, conn, map[string]string{"kind": "something-else"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, _, err = conn.Read(ctx)
	if got := websocket.CloseStatus(err); got != websocket.StatusPolicyViolation {
		t.Fatalf("close status = %v (err %v), want StatusPolicyViolation", got, err)
	}
	if got := pres.List(); len(got) != 0 {
		t.Fatalf("List() = %+v, want empty", got)
	}
}

func TestStreamWS_ReconnectEvictsStaleSocket(t *testing.T) {
	e, pres := newEchoWithPresence(t, 0)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + srv.URL[4:] + "/conversations/_stream/ws"

	old, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Dial old: %v", err)
	}
	defer old.CloseNow()
	if err := wsjson.Write(ctx, old, registerFrame("i1", "s1")); err != nil {
		t.Fatalf("register old: %v", err)
	}
	waitFor(t, func() bool { return len(pres.List()) == 1 })

	// Same SessionID, new InstanceID: the old socket must be hung up.
	fresh, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Dial fresh: %v", err)
	}
	defer fresh.CloseNow()
	if err := wsjson.Write(ctx, fresh, registerFrame("i2", "s1")); err != nil {
		t.Fatalf("register fresh: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	if _, _, err := old.Read(readCtx); err == nil {
		t.Fatal("expected the displaced socket to be closed")
	}

	got := pres.List()
	if len(got) != 1 || got[0].InstanceID != "i2" {
		t.Fatalf("List() = %+v, want only i2", got)
	}
}

func TestWSConversation_DeliversAppendedMessage(t *testing.T) {
	e, svc, _ := newEchoWithBus(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	c, err := svc.Create(context.Background(), "x", nil, nil)
	if err != nil {
		t.Fatalf("Create conv: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + srv.URL[4:] + "/conversations/" + c.ID + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Dial WS: %v", err)
	}
	defer conn.CloseNow()

	time.Sleep(50 * time.Millisecond)
	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "hello over ws", nil); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	var got entity.ConversationEvent
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	if err := wsjson.Read(readCtx, conn, &got); err != nil {
		t.Fatalf("wsjson.Read: %v", err)
	}
	if got.ConversationID != c.ID || got.Role != "user" || got.Content != "hello over ws" {
		t.Fatalf("bad WS event: %+v", got)
	}
}

func TestWSConversation_FiltersOtherConversations(t *testing.T) {
	e, svc, _ := newEchoWithBus(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	a, _ := svc.Create(context.Background(), "a", nil, nil)
	b, _ := svc.Create(context.Background(), "b", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + srv.URL[4:] + "/conversations/" + a.ID + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Dial WS: %v", err)
	}
	defer conn.CloseNow()

	time.Sleep(50 * time.Millisecond)
	if _, err := svc.AppendMessage(context.Background(), b.ID, "user", "wrong room", nil); err != nil {
		t.Fatalf("AppendMessage b: %v", err)
	}
	if _, err := svc.AppendMessage(context.Background(), a.ID, "user", "right room", nil); err != nil {
		t.Fatalf("AppendMessage a: %v", err)
	}

	var got entity.ConversationEvent
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	if err := wsjson.Read(readCtx, conn, &got); err != nil {
		t.Fatalf("wsjson.Read: %v", err)
	}
	if got.ConversationID != a.ID || got.Content != "right room" {
		t.Fatalf("expected A event, got %+v", got)
	}
}

func TestStreamWS_DeliversUserMessage(t *testing.T) {
	e, svc, bus := newEchoWithBus(t)
	r := conversationrouter.NewForTestWithPresence(svc, bus, presencesvc.New())
	e.GET("/_stream/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.StreamWS(api.WrapContext(c), conn)
	})

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	c, err := svc.Create(context.Background(), "x", nil, nil)
	if err != nil {
		t.Fatalf("Create conv: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + srv.URL[4:] + "/_stream/ws"
	conn := dialStream(t, ctx, wsURL, "i1", "s1")
	defer conn.CloseNow()

	time.Sleep(50 * time.Millisecond)
	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "hi from stream", nil,
		model.WithTargetSession("s1")); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	var got entity.ConversationEvent
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	if err := wsjson.Read(readCtx, conn, &got); err != nil {
		t.Fatalf("wsjson.Read: %v", err)
	}
	if got.ConversationID != c.ID || got.Role != "user" || got.Content != "hi from stream" {
		t.Fatalf("bad stream event: %+v", got)
	}
}

func TestStreamWS_FiltersAssistantRole(t *testing.T) {
	e, svc, bus := newEchoWithBus(t)
	r := conversationrouter.NewForTestWithPresence(svc, bus, presencesvc.New())
	e.GET("/_stream/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.StreamWS(api.WrapContext(c), conn)
	})

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	c, _ := svc.Create(context.Background(), "x", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialStream(t, ctx, "ws"+srv.URL[4:]+"/_stream/ws", "i1", "s1")
	defer conn.CloseNow()

	time.Sleep(50 * time.Millisecond)
	if _, err := svc.AppendMessage(context.Background(), c.ID, "assistant", "answer", nil); err != nil {
		t.Fatalf("AppendMessage assistant: %v", err)
	}
	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "question", nil,
		model.WithTargetSession("s1")); err != nil {
		t.Fatalf("AppendMessage user: %v", err)
	}

	var got entity.ConversationEvent
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	if err := wsjson.Read(readCtx, conn, &got); err != nil {
		t.Fatalf("wsjson.Read: %v", err)
	}
	if got.Role != "user" || got.Content != "question" {
		t.Fatalf("expected user event, got role=%s content=%q", got.Role, got.Content)
	}
}

func TestStreamWS_PingKeepsConnectionAliveAcrossIdleGap(t *testing.T) {
	old := conversationrouter.StreamPingInterval
	conversationrouter.StreamPingInterval = 50 * time.Millisecond
	t.Cleanup(func() { conversationrouter.StreamPingInterval = old })

	e, svc, bus := newEchoWithBus(t)
	r := conversationrouter.NewForTestWithPresence(svc, bus, presencesvc.New())
	e.GET("/_stream/ws", func(c echo.Context) error {
		conn, err := websocket.Accept(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		return r.StreamWS(api.WrapContext(c), conn)
	})

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	c, _ := svc.Create(context.Background(), "x", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialStream(t, ctx, "ws"+srv.URL[4:]+"/_stream/ws", "i1", "s1")
	defer conn.CloseNow()

	// Idle gap longer than 2 ping intervals — without server-side pings the
	// connection would be half-open from coder/websocket's view (no read
	// loop firing), but with pings the conn stays healthy.
	time.Sleep(150 * time.Millisecond)

	if _, err := svc.AppendMessage(context.Background(), c.ID, "user", "after-idle", nil,
		model.WithTargetSession("s1")); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	var got entity.ConversationEvent
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	if err := wsjson.Read(readCtx, conn, &got); err != nil {
		t.Fatalf("wsjson.Read after idle gap: %v", err)
	}
	if got.ConversationID != c.ID || got.Role != "user" || got.Content != "after-idle" {
		t.Fatalf("bad event after idle: %+v", got)
	}
}

func TestDispatchMessage_HTTP_AcceptsLiveTarget(t *testing.T) {
	e, svc, pres, _ := newEchoRESTWithPresence(t)
	pres.Register(liveChannel("sess-A"))
	c, _ := svc.Create(context.Background(), "x", nil, nil)
	msg, _ := svc.AppendMessage(context.Background(), c.ID, "user", "queued", nil)

	rec := postJSON(t, e, http.MethodPost,
		"/conversations/"+c.ID+"/messages/"+msg.ID+"/dispatch",
		`{"target_session_id":"sess-A"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDispatchMessage_HTTP_409OnDeadTarget(t *testing.T) {
	e, svc, _, _ := newEchoRESTWithPresence(t)
	c, _ := svc.Create(context.Background(), "x", nil, nil)
	msg, _ := svc.AppendMessage(context.Background(), c.ID, "user", "queued", nil)

	rec := postJSON(t, e, http.MethodPost,
		"/conversations/"+c.ID+"/messages/"+msg.ID+"/dispatch",
		`{"target_session_id":"sess-gone"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestDispatchMessage_HTTP_RequiresTarget(t *testing.T) {
	e, svc, _, _ := newEchoRESTWithPresence(t)
	c, _ := svc.Create(context.Background(), "x", nil, nil)
	msg, _ := svc.AppendMessage(context.Background(), c.ID, "user", "queued", nil)

	rec := postJSON(t, e, http.MethodPost,
		"/conversations/"+c.ID+"/messages/"+msg.ID+"/dispatch", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

// An ack on the stream socket reaches the delivery service.
func TestStreamWS_AckReachesDelivery(t *testing.T) {
	e, _, _, del := newEchoRESTWithPresence(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	events, stop := del.Watch()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialStream(t, ctx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", "i1", "sess-A")
	defer conn.CloseNow()

	if err := wsjson.Write(ctx, conn, entity.ChannelAck{
		Kind: entity.ChannelAckKind, MessageID: "01MSG",
	}); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	select {
	case ev := <-events:
		if ev.MessageID != "01MSG" || ev.State != entity.DeliveryStateDelivered {
			t.Fatalf("event = %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ack never reached the delivery service")
	}
}

// A frame that is not an ack must not kill the socket, and must not ack.
func TestStreamWS_UnknownInboundFrameIsIgnored(t *testing.T) {
	e, _, pres, del := newEchoRESTWithPresence(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	events, stop := del.Watch()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialStream(t, ctx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", "i1", "sess-A")
	defer conn.CloseNow()
	waitFor(t, func() bool { return len(pres.List()) == 1 })

	if err := wsjson.Write(ctx, conn, map[string]string{"kind": "something-else"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	select {
	case ev := <-events:
		t.Fatalf("a non-ack frame produced a delivery event: %+v", ev)
	case <-time.After(300 * time.Millisecond):
	}
	if len(pres.List()) != 1 {
		t.Fatal("the socket was torn down by an unknown frame")
	}
}

// An ack carrying no message id is not a delivery.
func TestStreamWS_AckWithoutMessageIDIsIgnored(t *testing.T) {
	e, _, _, del := newEchoRESTWithPresence(t)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	events, stop := del.Watch()
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialStream(t, ctx, "ws"+srv.URL[4:]+"/conversations/_stream/ws", "i1", "sess-A")
	defer conn.CloseNow()

	if err := wsjson.Write(ctx, conn, entity.ChannelAck{Kind: entity.ChannelAckKind}); err != nil {
		t.Fatalf("write: %v", err)
	}

	select {
	case ev := <-events:
		t.Fatalf("empty ack produced a delivery event: %+v", ev)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestListMessages_AfterCursorReturnsOnlyNewer(t *testing.T) {
	e := newEchoWithConversationRouter(t)
	rec := postJSON(t, e, http.MethodPost, "/conversations", `{"title":"x"}`)
	var c entity.Conversation
	_ = json.NewDecoder(rec.Body).Decode(&c)

	ids := make([]string, 3)
	for i := range ids {
		rec = postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages",
			`{"role":"user","content":"m"}`)
		var m entity.Message
		_ = json.NewDecoder(rec.Body).Decode(&m)
		ids[i] = m.ID
	}

	req := httptest.NewRequest(http.MethodGet, "/conversations/"+c.ID+"/messages?after="+ids[0], nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Messages []*entity.Message `json:"messages"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Messages) != 2 || resp.Messages[0].ID != ids[1] {
		t.Fatalf("got %d messages, want the two after the cursor", len(resp.Messages))
	}
}

// No cursor keeps the old behaviour: everything.
func TestListMessages_WithoutAfterReturnsEverything(t *testing.T) {
	e := newEchoWithConversationRouter(t)
	rec := postJSON(t, e, http.MethodPost, "/conversations", `{"title":"x"}`)
	var c entity.Conversation
	_ = json.NewDecoder(rec.Body).Decode(&c)
	_ = postJSON(t, e, http.MethodPost, "/conversations/"+c.ID+"/messages", `{"role":"user","content":"m"}`)

	req := httptest.NewRequest(http.MethodGet, "/conversations/"+c.ID+"/messages", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var resp struct {
		Messages []*entity.Message `json:"messages"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Messages) != 1 {
		t.Fatalf("got %d, want 1", len(resp.Messages))
	}
}
