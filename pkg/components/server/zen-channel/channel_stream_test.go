package zenchannel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/xhanio/errors"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
	"github.com/xhanio/zen/pkg/types/entity"
)

func TestSubscriber_SendsRegistrationFrameFirst(t *testing.T) {
	got := make(chan entity.ChannelRegistration, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		var reg entity.ChannelRegistration
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := wsjson.Read(ctx, conn, &reg); err != nil {
			return
		}
		got <- reg
		<-r.Context().Done()
	}))
	defer srv.Close()

	sub := &Subscriber{
		BaseURL: srv.URL,
		Registration: entity.ChannelRegistration{
			Kind:       entity.ChannelRegistrationKind,
			InstanceID: "01INSTANCE",
			SessionID:  "sess-uuid",
			Cwd:        "/repo",
			StartedAt:  time.Now(),
		},
		ClientInfo: func() (string, string) { return "claude-code", "2.1.205" },
		Push:       func(context.Context, ChannelNotification) error { return nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() { _ = sub.Run(ctx) }()

	select {
	case reg := <-got:
		if reg.Kind != entity.ChannelRegistrationKind {
			t.Fatalf("kind = %q", reg.Kind)
		}
		if reg.InstanceID != "01INSTANCE" || reg.SessionID != "sess-uuid" {
			t.Fatalf("registration = %+v", reg)
		}
		if reg.ClientName != "claude-code" || reg.ClientVer != "2.1.205" {
			t.Fatalf("client info not filled at dial time: %+v", reg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no registration frame received")
	}
}

// TestRunChannel_DoesNotDialBeforeHandshake pins bug 3: RunChannel used to
// start the stdio server and the WS subscriber concurrently, so the subscriber
// could dial the backend — and push a notification to stdout — before Claude
// Code had sent `initialize`.
func TestRunChannel_DoesNotDialBeforeHandshake(t *testing.T) {
	var mu sync.Mutex
	var dialed bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_stream/ws") {
			mu.Lock()
			dialed = true
			mu.Unlock()
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		<-r.Context().Done()
	}))
	defer srv.Close()

	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = RunChannel(ctx, ChannelOptions{
			BackendURL: srv.URL,
			In:         serverSide,
			Out:        serverSide,
			InstanceID: "01INSTANCE",
		})
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		serverSide.Close()
		<-done
	})

	// Give the subscriber goroutine ample opportunity to dial if it is going to.
	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	early := dialed
	mu.Unlock()
	if early {
		t.Fatal("subscriber dialed the backend before the MCP handshake")
	}

	// Complete the handshake; only now may it dial.
	enc := json.NewEncoder(clientSide)
	scan := bufio.NewScanner(clientSide)
	scan.Buffer(make([]byte, 1<<20), 1<<20)
	_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize"})
	if !scan.Scan() {
		t.Fatal("no response to initialize")
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := dialed
		mu.Unlock()
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("subscriber never dialed after the handshake completed")
}

// The headline test for bug 1. Two channels with distinct SessionIDs subscribe
// to the same fan-out stream. One event, addressed to the first, must reach
// exactly one of them. On the pre-M3 code both push, and this fails.
func TestSubscriber_OnlyTargetedSessionPushes(t *testing.T) {
	ev := entity.ConversationEvent{
		ConversationID:  "01CONV",
		MessageID:       "01MSG",
		Role:            "user",
		Content:         "hello",
		TargetSessionID: "sess-A",
		CreatedAt:       time.Now(),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/conversations/01CONV") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"01CONV","title":"t"}`))
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		var reg entity.ChannelRegistration
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := wsjson.Read(ctx, conn, &reg); err != nil {
			return
		}
		// Broadcast the same event to every subscriber, exactly as the bus does.
		_ = wsjson.Write(ctx, conn, ev)
		<-r.Context().Done()
	}))
	defer srv.Close()

	run := func(sessionID string, pushed chan<- string) *Subscriber {
		return &Subscriber{
			BaseURL: srv.URL,
			Backend: zenbackend.New(srv.URL + "/api/v1"),
			Registration: entity.ChannelRegistration{
				Kind:       entity.ChannelRegistrationKind,
				InstanceID: "i-" + sessionID,
				SessionID:  sessionID,
			},
			Push: func(_ context.Context, n ChannelNotification) error {
				pushed <- sessionID
				return nil
			},
		}
	}

	pushed := make(chan string, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() { _ = run("sess-A", pushed).Run(ctx) }()
	go func() { _ = run("sess-B", pushed).Run(ctx) }()

	select {
	case who := <-pushed:
		if who != "sess-A" {
			t.Fatalf("event targeted at sess-A was pushed by %s", who)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the targeted session never pushed")
	}

	// The untargeted session must stay silent. Give it every chance to speak.
	select {
	case who := <-pushed:
		t.Fatalf("%s pushed an event it was not addressed for — bug 1 is not fixed", who)
	case <-time.After(500 * time.Millisecond):
	}
}

// An event addressed to nobody reaches nobody. Empty must never mean broadcast.
func TestSubscriber_NullTargetPushesToNobody(t *testing.T) {
	ev := entity.ConversationEvent{
		ConversationID: "01CONV", MessageID: "01MSG", Role: "user",
		Content: "hello", CreatedAt: time.Now(),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve the conversation lookup too, so a push would succeed if the
		// filter let the event through. Without this the test could pass
		// because conversation.Get failed, not because the event was dropped.
		if strings.HasSuffix(r.URL.Path, "/conversations/01CONV") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"01CONV","title":"t"}`))
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		var reg entity.ChannelRegistration
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := wsjson.Read(ctx, conn, &reg); err != nil {
			return
		}
		_ = wsjson.Write(ctx, conn, ev)
		<-r.Context().Done()
	}))
	defer srv.Close()

	pushed := make(chan struct{}, 1)
	sub := &Subscriber{
		BaseURL: srv.URL,
		Backend: zenbackend.New(srv.URL + "/api/v1"),
		Registration: entity.ChannelRegistration{
			Kind: entity.ChannelRegistrationKind, InstanceID: "i1", SessionID: "sess-A",
		},
		Push: func(context.Context, ChannelNotification) error { pushed <- struct{}{}; return nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { _ = sub.Run(ctx) }()

	select {
	case <-pushed:
		t.Fatal("an event with no target was pushed; empty must not mean broadcast")
	case <-time.After(700 * time.Millisecond):
	}
}

// TestSubscriber_StopsWhenDisplaced pins the flapping bug found in manual
// verification: a displaced channel used to reconnect and reclaim its
// SessionID, so two live processes sharing one session evicted each other in a
// loop (6 evictions in 11s). A displaced channel must stop instead.
func TestSubscriber_StopsWhenDisplaced(t *testing.T) {
	var mu sync.Mutex
	var dials int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		dials++
		mu.Unlock()

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		var reg entity.ChannelRegistration
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := wsjson.Read(ctx, conn, &reg); err != nil {
			return
		}
		// Displace it, exactly as StreamWS does when a newer channel registers.
		_ = conn.Close(websocket.StatusNormalClosure, entity.ChannelDisplacedReason)
	}))
	defer srv.Close()

	sub := &Subscriber{
		BaseURL:   srv.URL,
		DialDelay: 20 * time.Millisecond, // would redial fast if it were going to
		Registration: entity.ChannelRegistration{
			Kind:       entity.ChannelRegistrationKind,
			InstanceID: "01OLD",
			SessionID:  "shared-session",
		},
		Push: func(context.Context, ChannelNotification) error { return nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- sub.Run(ctx) }()

	select {
	case err := <-done:
		if !errors.Is(err, ErrDisplaced) {
			t.Fatalf("Run() = %v, want ErrDisplaced", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after being displaced; it is still reconnecting")
	}

	// Give any stray reconnect a chance to show up.
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	n := dials
	mu.Unlock()
	if n != 1 {
		t.Fatalf("dialed %d times, want exactly 1 — a displaced channel must not reclaim its session", n)
	}
}

func TestChannelIdentity_FallsBackWhenSessionIDAbsent(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")

	id := resolveSessionID("01INSTANCE", nil)
	if id != "01INSTANCE" {
		t.Fatalf("resolveSessionID() = %q, want the instance id as fallback", id)
	}
}

func TestChannelIdentity_UsesSessionIDWhenPresent(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SESSION_ID", "sess-uuid")

	id := resolveSessionID("01INSTANCE", nil)
	if id != "sess-uuid" {
		t.Fatalf("resolveSessionID() = %q, want the env value", id)
	}
}

func TestSubscriber_EmitsNotificationOnUserEvent(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/conversations/01CONV", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"01CONV","title":"t","anchor_kind":"card","anchor_id":"01CARD","created_at":"2026-06-27T00:00:00Z","last_message_at":"2026-06-27T00:00:00Z"}`))
	})
	mux.HandleFunc("/api/v1/conversations/_stream/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		ev := entity.ConversationEvent{
			ConversationID: "01CONV", MessageID: "01MSG", Role: "user",
			Content: "hi via channel",
			// Address the event at this subscriber's session. Without a target
			// the channel now drops it, and an empty SessionID on the
			// Registration would make this pass only because "" == "".
			TargetSessionID: "sess-A",
			CreatedAt:       time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC),
		}
		b, _ := json.Marshal(ev)
		_ = conn.Write(r.Context(), websocket.MessageText, b)
		time.Sleep(50 * time.Millisecond)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var mu sync.Mutex
	var got []ChannelNotification
	sub := &Subscriber{
		BaseURL: srv.URL,
		Backend: zenbackend.New(srv.URL + "/api/v1"),
		Registration: entity.ChannelRegistration{
			Kind: entity.ChannelRegistrationKind, InstanceID: "i1", SessionID: "sess-A",
		},
		Push: func(_ context.Context, note ChannelNotification) error {
			mu.Lock()
			defer mu.Unlock()
			got = append(got, note)
			return nil
		},
		PingPeriod: 10 * time.Second,
		DialDelay:  10 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { _ = sub.Run(ctx); close(done) }()

	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) < 1 {
		t.Fatal("no notification received")
	}
	if !strings.Contains(got[0].Content, "hi via channel") {
		t.Fatalf("content: %s", got[0].Content)
	}
	if got[0].Meta["anchor_kind"] != "card" || got[0].Meta["anchor_id"] != "01CARD" {
		t.Fatalf("meta should carry anchor: %v", got[0].Meta)
	}
	cancel()
	<-done
}

func TestSubscriber_SkipsNonUserEvents(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/conversations/_stream/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		ev := entity.ConversationEvent{
			ConversationID: "01CONV", MessageID: "01MSG", Role: "assistant",
			Content: "wrong role", CreatedAt: time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC),
		}
		b, _ := json.Marshal(ev)
		_ = conn.Write(r.Context(), websocket.MessageText, b)
		time.Sleep(50 * time.Millisecond)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var n int
	var mu sync.Mutex
	sub := &Subscriber{
		BaseURL: srv.URL,
		Backend: zenbackend.New(srv.URL + "/api/v1"),
		Push: func(_ context.Context, _ ChannelNotification) error {
			mu.Lock()
			n++
			mu.Unlock()
			return nil
		},
		DialDelay: 10 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	_ = sub.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	if n != 0 {
		t.Fatalf("assistant events must not push; got %d", n)
	}
}

// The channel acks a message it actually pushed into its Claude Code session.
func TestSubscriber_AcksAfterSuccessfulPush(t *testing.T) {
	ev := entity.ConversationEvent{
		ConversationID: "01CONV", MessageID: "01MSG", Role: "user",
		Content: "hi", TargetSessionID: "sess-A", CreatedAt: time.Now(),
	}
	acks := make(chan entity.ChannelAck, 2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/conversations/01CONV") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"01CONV","title":"t"}`))
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		var reg entity.ChannelRegistration
		if err := wsjson.Read(ctx, conn, &reg); err != nil {
			return
		}
		_ = wsjson.Write(ctx, conn, ev)

		for {
			var ack entity.ChannelAck
			if err := wsjson.Read(ctx, conn, &ack); err != nil {
				return
			}
			acks <- ack
		}
	}))
	defer srv.Close()

	sub := &Subscriber{
		BaseURL: srv.URL,
		Backend: zenbackend.New(srv.URL + "/api/v1"),
		Registration: entity.ChannelRegistration{
			Kind: entity.ChannelRegistrationKind, InstanceID: "i1", SessionID: "sess-A",
		},
		Push: func(context.Context, ChannelNotification) error { return nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() { _ = sub.Run(ctx) }()

	select {
	case ack := <-acks:
		if ack.Kind != entity.ChannelAckKind || ack.MessageID != "01MSG" {
			t.Fatalf("ack = %+v", ack)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no ack after a successful push")
	}
}

// A push that fails must not ack. The message then surfaces as not delivered
// rather than vanishing — that is the entire reason acks exist.
func TestSubscriber_DoesNotAckWhenPushFails(t *testing.T) {
	ev := entity.ConversationEvent{
		ConversationID: "01CONV", MessageID: "01MSG", Role: "user",
		Content: "hi", TargetSessionID: "sess-A", CreatedAt: time.Now(),
	}
	acks := make(chan entity.ChannelAck, 2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/conversations/01CONV") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"01CONV","title":"t"}`))
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		var reg entity.ChannelRegistration
		if err := wsjson.Read(ctx, conn, &reg); err != nil {
			return
		}
		_ = wsjson.Write(ctx, conn, ev)

		for {
			var ack entity.ChannelAck
			if err := wsjson.Read(ctx, conn, &ack); err != nil {
				return
			}
			acks <- ack
		}
	}))
	defer srv.Close()

	sub := &Subscriber{
		BaseURL: srv.URL,
		Backend: zenbackend.New(srv.URL + "/api/v1"),
		Registration: entity.ChannelRegistration{
			Kind: entity.ChannelRegistrationKind, InstanceID: "i1", SessionID: "sess-A",
		},
		Push: func(context.Context, ChannelNotification) error {
			return fmt.Errorf("claude code is not listening")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { _ = sub.Run(ctx) }()

	select {
	case ack := <-acks:
		t.Fatalf("acked a push that failed: %+v", ack)
	case <-time.After(700 * time.Millisecond):
	}
}

// A backend that drops a healthy connection — a redeploy, a restart — must be
// reconnected to promptly. The dial backoff exists for an unreachable server;
// it must reset once a connection actually establishes, or a channel that has
// weathered a few drops ends up pinned at the 30s ceiling and takes half a
// minute to notice the server is back.
func TestSubscriber_ReconnectsPromptlyAfterHealthyDrops(t *testing.T) {
	var mu sync.Mutex
	var dials int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		dials++
		mu.Unlock()
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		var reg entity.ChannelRegistration
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		if err := wsjson.Read(ctx, conn, &reg); err != nil {
			cancel()
			return
		}
		cancel()
		// Registered — a genuine, healthy connection — then drop it, the way a
		// server going down mid-stream does. NOT displacement, so Run retries.
		_ = conn.Close(websocket.StatusGoingAway, "server restarting")
	}))
	defer srv.Close()

	sub := &Subscriber{
		BaseURL:   srv.URL,
		DialDelay: 20 * time.Millisecond,
		Registration: entity.ChannelRegistration{
			Kind: entity.ChannelRegistrationKind, InstanceID: "01INST", SessionID: "s1",
		},
		Push: func(context.Context, ChannelNotification) error { return nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	go func() { _ = sub.Run(ctx) }()
	<-ctx.Done()

	mu.Lock()
	got := dials
	mu.Unlock()
	// Steady 20ms cadence would give ~18 in 400ms. Exponential (no reset) gives
	// ~5 (20+40+80+160+…). 8 is comfortably above the buggy ceiling.
	if got < 8 {
		t.Fatalf("only %d reconnects in 400ms — backoff is not resetting after a healthy drop", got)
	}
}
