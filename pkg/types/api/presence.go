package api

import "github.com/xhanio/zen/pkg/types/entity"

// SessionsFrameKind is the Kind value on a SessionsFrame.
const SessionsFrameKind = "sessions"

// SessionsFrame is a snapshot of the live channel registry, pushed on connect
// and on every change.
//
// /_sessions/ws is read only by the SPA — no channel consumes it — so it may
// carry a Kind discriminator and grow new frame types (M4 adds "delivery")
// without disturbing the ConversationEvent wire format the channel reads.
//
// It is a snapshot rather than a delta on purpose: a dropped snapshot is
// harmless because the next one subsumes it.
type SessionsFrame struct {
	Kind     string            `json:"kind"`
	Sessions []*entity.Channel `json:"sessions"`
}

// DeliveryFrameKind is the Kind value on a DeliveryFrame.
const DeliveryFrameKind = "delivery"

// DeliveryFrame announces that one message reached one session. It shares
// /_sessions/ws with SessionsFrame, distinguished by Kind — which is why that
// stream carried a discriminator from the day it was built.
type DeliveryFrame struct {
	Kind      string `json:"kind"`
	MessageID string `json:"message_id"`
	State     string `json:"state"`
}
