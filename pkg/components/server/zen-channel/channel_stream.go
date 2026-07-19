package zenchannel

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/utils/log"

	zenbackend "github.com/xhanio/zen/pkg/components/client/zen-backend"
	"github.com/xhanio/zen/pkg/types/entity"
)

// Subscriber tails the backend fan-out WS (/api/v1/conversations/_stream/ws),
// transforms each role=user ConversationEvent into a ChannelNotification, and
// hands it to Push. Mirrors streamLoop() from the deleted Bun server.
//
// Reconnect semantics: DialDelay backs off exponentially up to 30s on
// consecutive errors; a healthy disconnect resets it to DialDelay.
type Subscriber struct {
	// BaseURL is the backend HTTP root (e.g. "http://127.0.0.1:38000"); the
	// /api/v1 prefix is appended.
	BaseURL string

	Backend zenbackend.Client
	Push    func(ctx context.Context, note ChannelNotification) error
	Log     log.Logger

	// Registration is written as the first frame on every (re)connect. The
	// backend closes any /_stream/ws socket that does not register.
	Registration entity.ChannelRegistration

	// ClientInfo, when set, is called at dial time to fill the registration's
	// client name and version from the completed MCP handshake.
	ClientInfo func() (string, string)

	DialDelay  time.Duration // default 1s, doubles up to 30s
	PingPeriod time.Duration // default 25s
}

// ErrDisplaced is returned when the backend closed this channel's stream
// because a newer channel claimed the same SessionID. It is terminal: retrying
// would reclaim the session and start an eviction loop between the two live
// processes. See entity.ChannelDisplacedReason.
var ErrDisplaced = errors.Newf("channel displaced by a newer instance for the same session")

// displaced reports whether err is the backend's displacement close frame.
func displaced(err error) bool {
	var ce websocket.CloseError
	return errors.As(err, &ce) && ce.Reason == entity.ChannelDisplacedReason
}

// Run blocks until ctx is cancelled, reconnecting on every stream error.
//
// The one error it does not retry is displacement: the backend has handed this
// session to a newer channel, and dialing back would take it away again.
func (s *Subscriber) Run(ctx context.Context) error {
	if s.Log == nil {
		s.Log = log.New(log.NoStdout())
	}
	if s.DialDelay == 0 {
		s.DialDelay = 1 * time.Second
	}
	if s.PingPeriod == 0 {
		s.PingPeriod = 25 * time.Second
	}

	const maxBackoff = 30 * time.Second
	backoff := s.DialDelay
	for {
		connected, err := s.streamOnce(ctx)
		if err != nil {
			if displaced(err) {
				s.Log.Warnf("displaced: session %s was claimed by a newer channel; stopping", s.Registration.SessionID)
				return ErrDisplaced
			}
			s.Log.Warnf("stream error: %v", err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// A session that actually connected resets the ladder — before the wait,
		// so a healthy connection that drops (a redeploy, a restart) reconnects
		// at DialDelay rather than waiting out a backoff grown by an earlier
		// outage. The ladder only climbs while the server is unreachable.
		if connected {
			backoff = s.DialDelay
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if !connected {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// streamOnce dials, registers, and pumps events until the connection ends. The
// bool reports whether a connection was actually established (dial + register
// succeeded) before the error — it drives the caller's backoff reset, so a
// healthy connection that drops is distinguished from a server that was never
// reachable.
func (s *Subscriber) streamOnce(ctx context.Context) (bool, error) {
	wsURL := wsURLFor(s.BaseURL)
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return false, err
	}
	defer conn.Close(websocket.StatusGoingAway, "subscriber stop")

	// Register before anything else: the backend closes a socket that stays
	// silent. Written inside streamOnce so every reconnect re-registers, which
	// is what the backend expects after it evicts a stale socket.
	reg := s.Registration
	if s.ClientInfo != nil {
		reg.ClientName, reg.ClientVer = s.ClientInfo()
	}
	regCtx, regCancel := context.WithTimeout(ctx, 5*time.Second)
	err = wsjson.Write(regCtx, conn, reg)
	regCancel()
	if err != nil {
		return false, err
	}
	// Dialed and registered: a real connection. From here a failure is a drop,
	// not an unreachable server, so the caller should reconnect promptly.
	s.Log.Debugf("registered instance %s for session %s", reg.InstanceID, reg.SessionID)

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Client-side ping pump — symmetric with the backend's server-side ping
	// added in v0.3.1. Both ends detect half-open within a ping period.
	go func() {
		ticker := time.NewTicker(s.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pctx, pcancel := context.WithTimeout(streamCtx, 10*time.Second)
				err := conn.Ping(pctx)
				pcancel()
				if err != nil {
					cancel()
					return
				}
			case <-streamCtx.Done():
				return
			}
		}
	}()

	for {
		typ, data, err := conn.Read(streamCtx)
		if err != nil {
			return true, err
		}
		if typ != websocket.MessageText {
			continue
		}
		s.Log.Debugf("ws recv %s", string(data))
		var ev entity.ConversationEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			s.Log.Warnf("bad frame: %v", err)
			continue
		}
		if ev.Role != "user" {
			continue
		}
		// Routing lives at the edge: the bus keeps its dumb broadcast and each
		// channel keeps only what is addressed to it. Strict equality means an
		// empty target matches no live session, so "addressed to nobody" needs
		// no special case — and can never be mistaken for "addressed to all".
		if ev.TargetSessionID != s.Registration.SessionID {
			s.Log.Debugf("dropping event for session %q (I am %q)", ev.TargetSessionID, s.Registration.SessionID)
			continue
		}
		conv, err := s.Backend.Conversation().Get(streamCtx, ev.ConversationID)
		if err != nil {
			s.Log.Warnf("conversation.get(%s) failed: %v", ev.ConversationID, err)
			continue
		}
		note := FormatNotification(&ev, conv)
		s.Log.Debugf("ws dispatch conversation=%s meta=%v", ev.ConversationID, note.Meta)
		if err := s.Push(streamCtx, note); err != nil {
			// No ack. The message surfaces as not delivered rather than
			// vanishing, which is the entire reason acks exist.
			s.Log.Errorf("push failed: %v", err)
			continue
		}
		// The push succeeded: Claude Code has the notification. This says
		// nothing about whether the model will act on it — that is unobservable
		// by design. Writing here while the ping pump calls Ping is safe:
		// coder/websocket conn.go:30 says all methods but Reader/Read may be
		// called concurrently.
		ackCtx, ackCancel := context.WithTimeout(streamCtx, 5*time.Second)
		err = wsjson.Write(ackCtx, conn, entity.ChannelAck{
			Kind:      entity.ChannelAckKind,
			MessageID: ev.MessageID,
		})
		ackCancel()
		if err != nil {
			s.Log.Warnf("ack for %s failed: %v", ev.MessageID, err)
		}
	}
}

func wsURLFor(baseURL string) string {
	u := strings.TrimRight(baseURL, "/")
	switch {
	case strings.HasPrefix(u, "https://"):
		u = "wss://" + strings.TrimPrefix(u, "https://")
	case strings.HasPrefix(u, "http://"):
		u = "ws://" + strings.TrimPrefix(u, "http://")
	}
	return u + "/api/v1/conversations/_stream/ws"
}
