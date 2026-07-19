package model

import (
	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/entity"
)

// Registration is the handle StreamWS holds for as long as its channel is
// registered. Evicted closes when a newer channel claims the same SessionID,
// telling the older connection to shut its socket.
type Registration interface {
	Channel() *entity.Channel
	Evicted() <-chan struct{}
}

// Presence is the in-memory registry of live channels, keyed by SessionID.
// A session has at most one live channel.
type Presence interface {
	common.Service

	// Register adds ch. If a channel with the same SessionID is already
	// registered, it is displaced: its Registration's Evicted channel closes,
	// and the caller owning that connection is expected to close its socket.
	Register(ch *entity.Channel) Registration

	// Unregister removes the registration returned by Register, and only that
	// one. Passing a registration that has already been displaced is a no-op,
	// so a losing connection's deferred Unregister cannot evict its
	// replacement — not even when the replacement shares its InstanceID,
	// which a reconnecting channel process always does.
	Unregister(reg Registration)

	// List returns a snapshot of live channels, newest ConnectedAt first.
	List() []*entity.Channel

	// Has reports whether a channel is registered for this SessionID. The empty
	// string is never live, so a null routing target matches nobody.
	Has(sessionID string) bool

	// Get returns the live channel registered for sessionID, or false if none.
	// The empty string is never live.
	Get(sessionID string) (*entity.Channel, bool)

	// Watch returns a channel that receives a fresh snapshot on every change,
	// and a stop func that unsubscribes and closes it. The snapshot channel has
	// depth 1 and coalesces: a slow watcher sees the latest registry, never a
	// stale one, and never blocks the registry. Snapshots are the whole truth,
	// so coalescing loses nothing.
	Watch() (<-chan []*entity.Channel, func())
}
