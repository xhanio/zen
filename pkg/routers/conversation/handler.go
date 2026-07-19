package conversation

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/xhanio/errors"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
	"github.com/xhanio/zen/pkg/utils/ulidutil"
)

// StreamPingInterval is the period of the server-side WS ping pump that keeps
// the per-conversation and fan-out streams from sitting half-open behind NAT /
// proxies. Exported so tests can shrink it.
var StreamPingInterval = 30 * time.Second

// defaultRegistrationTimeout bounds how long a /_stream/ws connection may stay
// silent before it is closed. A half-open socket must never appear as a live
// session. It is a router field rather than a package var so tests can shrink
// it without racing the handler goroutine that reads it.
const defaultRegistrationTimeout = 5 * time.Second

func (r *router) CreateConversation(c api.Context) error {
	var req api.CreateConversationRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	conv, err := r.svc.Create(c, req.Title, req.AnchorKind, req.AnchorID)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusCreated, conv)
}

func (r *router) ListConversations(c api.Context) error {
	var anchorKind, anchorID *string
	if k := c.QueryParam("anchor_kind"); k != "" {
		anchorKind = &k
	}
	if id := c.QueryParam("anchor_id"); id != "" {
		anchorID = &id
	}
	pending := c.QueryParam("pending") == "true"
	limit := 0
	if l := c.QueryParam("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil {
			return errors.BadRequest.Newf("limit must be an integer")
		}
		limit = n
	}
	cs, counts, err := r.svc.List(c, anchorKind, anchorID, pending, limit)
	if err != nil {
		return errors.Wrap(err)
	}
	resp := api.ConversationListResponse{Conversations: cs}
	if pending {
		resp.UnansweredCounts = counts
	}
	return c.JSON(http.StatusOK, resp)
}

func (r *router) GetConversation(c api.Context) error {
	id := c.Param("id")
	conv, err := r.svc.Get(c, id)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, conv)
}

func (r *router) UpdateConversationTitle(c api.Context) error {
	id := c.Param("id")
	var req api.UpdateConversationTitleRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	conv, err := r.svc.UpdateTitle(c, id, req.Title)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, conv)
}

func (r *router) DeleteConversation(c api.Context) error {
	id := c.Param("id")
	if err := r.svc.Delete(c, id); err != nil {
		return errors.Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (r *router) ListMessages(c api.Context) error {
	id := c.Param("id")
	limit := 0
	if l := c.QueryParam("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil {
			return errors.BadRequest.Newf("limit must be an integer")
		}
		limit = n
	}
	// `after` is the consumer's cursor. It refetches the span it missed rather
	// than the whole thread, which is what makes gap recovery cheap. An empty
	// cursor is exactly the old behaviour: everything.
	after := c.QueryParam("after")

	msgs, err := r.svc.GetMessagesAfter(c, id, after, limit)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusOK, api.MessageListResponse{Messages: msgs})
}

func (r *router) AppendMessage(c api.Context) error {
	id := c.Param("id")
	var req api.AppendMessageRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	// Validating against the live registry costs one map lookup and turns a
	// silent dead-end into a 409 the picker can act on. A race remains between
	// this check and the publish — the session can die in between — and that is
	// not worth locking against: the message simply shows as not delivered.
	var opts []model.AppendOption
	switch req.Role {
	case "user":
		// A user turn is attributed to the session it is addressed to; the cwd
		// is read from the live registry (the same lookup the 409 needs).
		if req.TargetSessionID != nil && *req.TargetSessionID != "" {
			if r.presence == nil {
				return errors.Conflict.Newf("session %s is not connected", *req.TargetSessionID)
			}
			ch, ok := r.presence.Get(*req.TargetSessionID)
			if !ok {
				return errors.Conflict.Newf("session %s is not connected", *req.TargetSessionID)
			}
			opts = append(opts,
				model.WithTargetSession(ch.SessionID),
				model.WithSession(ch.SessionID, ch.Cwd))
		}
	default:
		// assistant / system: the channel attributes the message to its own
		// session and supplies its own cwd. No target — it addresses nobody.
		if req.SessionID != nil && *req.SessionID != "" {
			cwd := ""
			if req.SessionCwd != nil {
				cwd = *req.SessionCwd
			}
			opts = append(opts, model.WithSession(*req.SessionID, cwd))
		}
	}

	msg, err := r.svc.AppendMessage(c, id, req.Role, req.Content, req.SelectionText, opts...)
	if err != nil {
		return errors.Wrap(err)
	}
	return c.JSON(http.StatusCreated, msg)
}

// DispatchMessage re-publishes a stored message at a live session. It serves
// both the "nothing was connected when I posted" path and the "resend to
// another session" path, so neither duplicates a row.
func (r *router) DispatchMessage(c api.Context) error {
	id := c.Param("id")
	mid := c.Param("mid")

	var req api.DispatchRequest
	if err := c.Bind(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if err := c.Validate(&req); err != nil {
		return errors.BadRequest.Wrap(err)
	}
	if r.presence == nil {
		return errors.Conflict.Newf("session %s is not connected", req.TargetSessionID)
	}
	ch, ok := r.presence.Get(req.TargetSessionID)
	if !ok {
		return errors.Conflict.Newf("session %s is not connected", req.TargetSessionID)
	}
	if err := r.svc.DispatchMessage(c, id, mid, req.TargetSessionID, ch.Cwd); err != nil {
		return errors.Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (r *router) WSConversation(c api.Context, conn *websocket.Conn) error {
	id := c.Param("id")
	if _, err := r.svc.Get(c, id); err != nil {
		_ = conn.Close(websocket.StatusPolicyViolation, "conversation not found")
		return nil
	}
	if r.bus == nil {
		_ = conn.Close(websocket.StatusInternalError, "bus not configured")
		return nil
	}

	// The suffix identifies the connection, not the conversation: Unsubscribe
	// removes every subscriber sharing a name, and Publish skips self-delivery
	// by name too, so two sockets on one conversation must never collide.
	subName := "ws:" + id + ":" + ulidutil.New()
	messenger, err := r.bus.NewMessenger(subName)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "subscribe failed")
		return nil
	}
	defer messenger.Close()

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	go pingPump(ctx, cancel, conn)

	// Outbound pump: bus → ws, filtered to this conversation's events.
	go func() {
		for {
			select {
			case msg, ok := <-messenger.Ch():
				if !ok {
					// The bus dropped this subscriber for not draining fast
					// enough. Hang up: a socket held open over a dead
					// subscription is silently deaf, and the client has no
					// reason to reconnect. Closing costs it one round trip,
					// after which it refills from its cursor.
					dropSubscription(ctx, cancel, conn)
					return
				}
				ev, ok := payloadEvent(msg.Payload)
				if !ok || ev.ConversationID != id {
					continue
				}
				if err := wsjson.Write(ctx, conn, ev); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Inbound pump: ws → bus (best-effort echo; SPA writes happen via REST).
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return nil
		}
		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		_ = messenger.Send(ctx, "ws_inbound", raw)
	}
}

// dropSubscription hangs up a socket whose bus subscription was closed under
// it. ctx.Err() distinguishes the two ways a subscription channel closes: our
// own teardown (ctx already cancelled — the socket is going away regardless)
// versus the driver evicting a subscriber that stopped draining, which is the
// case the client must be told about.
func dropSubscription(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	if ctx.Err() != nil {
		return
	}
	_ = conn.Close(websocket.StatusInternalError, "subscription dropped")
	cancel()
}

// payloadEvent un-wraps the payload published by AppendMessage. Pubsub stores
// the payload as `any`, so a type assertion is needed before filtering.
func payloadEvent(payload any) (*entity.ConversationEvent, bool) {
	if ev, ok := payload.(*entity.ConversationEvent); ok {
		return ev, true
	}
	return nil, false
}

// pingPump issues a WebSocket control ping every StreamPingInterval. A failed
// ping cancels the surrounding ctx so the outbound and inbound pumps tear down.
func pingPump(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	ticker := time.NewTicker(StreamPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pingCtx)
			pingCancel()
			if err != nil {
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *router) StreamWS(c api.Context, conn *websocket.Conn) error {
	if r.bus == nil {
		_ = conn.Close(websocket.StatusInternalError, "bus not configured")
		return nil
	}
	if r.presence == nil {
		_ = conn.Close(websocket.StatusInternalError, "presence not configured")
		return nil
	}

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	// The first frame must be a registration, and it must arrive promptly.
	reg, err := readRegistration(ctx, conn, r.registrationTimeout())
	if err != nil {
		// errors.Message, not %v: a handled error is reported by its message.
		// %v asks xhanio/errors for the stack too, which is what an *unhandled*
		// error deserves — one whose origin is in doubt. This one's origin is
		// the line above.
		r.log.Warnf("stream ws: %s", errors.Message(err))
		// Only tell a peer that is still listening. Writing a close frame to a
		// socket that already hung up buys nothing and can only fail.
		if !errors.Is(err, errPeerHungUp) {
			_ = conn.Close(websocket.StatusPolicyViolation, "registration required")
		}
		return nil
	}

	ch := &entity.Channel{
		InstanceID:  reg.InstanceID,
		SessionID:   reg.SessionID,
		Cwd:         reg.Cwd,
		StartedAt:   reg.StartedAt,
		ClientName:  reg.ClientName,
		ClientVer:   reg.ClientVer,
		ConnectedAt: time.Now(),
	}
	registration := r.presence.Register(ch)
	defer r.presence.Unregister(registration)

	// A newer channel claiming this SessionID evicts us; hang up so the client
	// stops believing it is subscribed.
	go func() {
		select {
		case <-registration.Evicted():
			_ = conn.Close(websocket.StatusNormalClosure, entity.ChannelDisplacedReason)
			cancel()
		case <-ctx.Done():
		}
	}()

	// The InstanceID prefix keeps subscriber names greppable against the
	// channel's own logs, but it cannot stand alone: a channel process keeps one
	// InstanceID for its lifetime and redials on every transient error, so two
	// sockets can briefly share it. Unsubscribe removes every subscriber with a
	// given name, so a stale socket's deferred Close would unsubscribe its own
	// replacement. The suffix makes the name identify the connection.
	subName := "stream:" + ch.InstanceID + ":" + ulidutil.New()
	messenger, err := r.bus.NewMessenger(subName)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "subscribe failed")
		return nil
	}
	defer func() {
		// Cancel first: the outbound pump must exit on ctx.Done rather than see
		// its channel close and mistake our own teardown for an eviction.
		cancel()
		messenger.Close()
	}()

	go pingPump(ctx, cancel, conn)

	// Outbound pump: bus → ws, filtered to ConversationEvent + role=user, and
	// addressed to this channel's session.
	go func() {
		for {
			select {
			case msg, ok := <-messenger.Ch():
				if !ok {
					dropSubscription(ctx, cancel, conn)
					return
				}
				if msg.Kind != entity.ConversationEventKind {
					continue
				}
				ev, ok := payloadEvent(msg.Payload)
				if !ok || ev.Role != "user" {
					continue
				}
				// The bus fans every event out to every channel. Selection
				// happens here, not at the far end of the socket: a channel
				// built before it learned to filter — or one that declines to —
				// would otherwise answer a message meant for another session,
				// and would hold that session's text either way. The channel
				// keeps its own filter; this is the layer that must not need it.
				//
				// Strict equality, so an empty target reaches nobody rather
				// than everybody.
				if ev.TargetSessionID != ch.SessionID {
					continue
				}
				if err := wsjson.Write(ctx, conn, ev); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Inbound pump: the channel acks a message it pushed into its session.
	// An unparseable or unknown frame is ignored, never fatal: acks are
	// advisory, and a channel that speaks a newer dialect must not be hung up.
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil
		}
		var ack entity.ChannelAck
		if err := json.Unmarshal(data, &ack); err != nil {
			continue
		}
		if ack.Kind != entity.ChannelAckKind || ack.MessageID == "" {
			continue
		}
		if r.delivery != nil {
			r.delivery.Ack(ack.MessageID)
		}
	}
}

func (r *router) registrationTimeout() time.Duration {
	if r.regTimeout > 0 {
		return r.regTimeout
	}
	return defaultRegistrationTimeout
}

// errPeerHungUp marks the one registration failure the caller must treat
// differently: the socket died before a frame arrived. A health probe, a
// scanner, a channel killed mid-handshake. There is nobody left to tell and
// nothing to fix, so the caller must not try to send it a close frame.
//
// The other failures need no sentinel — the caller does the same thing for all
// of them, and their message says which one it was.
var errPeerHungUp = errors.Newf("peer closed before registering")

// peerIsGone reports whether err means the other end stopped talking, whether
// it hung up abruptly (EOF) or sent a close frame on the way out.
func peerIsGone(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, context.Canceled) ||
		websocket.CloseStatus(err) != -1
}

// readRegistration reads exactly one frame and requires it to be a well-formed
// ChannelRegistration. It bounds the wait so a silent socket cannot linger.
//
// The read runs on its own goroutine rather than under a context.WithTimeout:
// coder/websocket tears the connection down when a read context is cancelled,
// which would deny the caller the chance to send a close frame, and the peer
// would see a bare EOF instead of StatusPolicyViolation. On timeout the read
// goroutine stays parked until the caller closes the conn, which it always does.
func readRegistration(ctx context.Context, conn *websocket.Conn, timeout time.Duration) (*entity.ChannelRegistration, error) {
	type result struct {
		reg entity.ChannelRegistration
		err error
	}
	done := make(chan result, 1)
	go func() {
		var res result
		res.err = wsjson.Read(ctx, conn, &res.reg)
		done <- res
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var reg entity.ChannelRegistration
	select {
	case res := <-done:
		if res.err != nil {
			if peerIsGone(res.err) {
				return nil, errPeerHungUp
			}
			return nil, errors.Wrapf(res.err, "unreadable registration frame")
		}
		reg = res.reg
	case <-timer.C:
		return nil, errors.Newf("no registration frame within %s", timeout)
	case <-ctx.Done():
		return nil, errPeerHungUp
	}

	if reg.Kind != entity.ChannelRegistrationKind {
		return nil, errors.Newf("first frame kind %q, want %q", reg.Kind, entity.ChannelRegistrationKind)
	}
	if reg.InstanceID == "" || reg.SessionID == "" {
		return nil, errors.Newf("registration missing instance_id or session_id")
	}
	return &reg, nil
}
