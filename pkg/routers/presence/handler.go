package presence

import (
	"context"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/xhanio/zen/pkg/types/api"
	"github.com/xhanio/zen/pkg/types/entity"
)

// PingInterval keeps the stream from sitting half-open behind a proxy. It is a
// package var only because nothing reads it concurrently with a test writing
// it: it is captured once, before the pump starts.
var PingInterval = 30 * time.Second

// SessionsWS streams the live channel registry to one SPA client: a snapshot on
// connect, then a fresh snapshot on every change.
func (r *router) SessionsWS(c api.Context, conn *websocket.Conn) error {
	if r.presence == nil {
		_ = conn.Close(websocket.StatusInternalError, "presence not configured")
		return nil
	}

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	// Subscribe before sending the initial snapshot. The other order has a gap:
	// a channel registering between the snapshot and the subscription would be
	// invisible until the next unrelated change.
	snaps, stopSnaps := r.presence.Watch()
	defer stopSnaps()

	// A nil channel blocks forever in a select, so when delivery is not
	// configured that case is simply never chosen.
	var deliveries <-chan entity.DeliveryEvent
	if r.delivery != nil {
		d, stopDeliveries := r.delivery.Watch()
		defer stopDeliveries()
		deliveries = d
	}

	if err := writeSessions(ctx, conn, r.presence.List()); err != nil {
		return nil
	}

	go pingPump(ctx, cancel, conn)

	// Inbound pump: the SPA sends nothing today; this detects close.
	go func() {
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		select {
		case snapshot, ok := <-snaps:
			if !ok {
				return nil
			}
			if err := writeSessions(ctx, conn, snapshot); err != nil {
				return nil
			}
		case ev, ok := <-deliveries:
			if !ok {
				return nil
			}
			if err := writeDelivery(ctx, conn, ev); err != nil {
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func writeDelivery(ctx context.Context, conn *websocket.Conn, ev entity.DeliveryEvent) error {
	return wsjson.Write(ctx, conn, api.DeliveryFrame{
		Kind:      api.DeliveryFrameKind,
		MessageID: ev.MessageID,
		State:     ev.State,
	})
}

func writeSessions(ctx context.Context, conn *websocket.Conn, sessions []*entity.Channel) error {
	// Go marshals a nil slice as `null`; normalise so the store never has to
	// guard frame.sessions.length against null.
	if sessions == nil {
		sessions = []*entity.Channel{}
	}
	return wsjson.Write(ctx, conn, api.SessionsFrame{
		Kind:     api.SessionsFrameKind,
		Sessions: sessions,
	})
}

func pingPump(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			pctx, pcancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pctx)
			pcancel()
			if err != nil {
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
