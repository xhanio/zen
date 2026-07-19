package model

import (
	"github.com/xhanio/framingo/pkg/types/common"

	"github.com/xhanio/zen/pkg/types/entity"
)

// Delivery relays channel acks to the SPA. It stores nothing: delivery state is
// a UI hint, not a queue, and a backend restart is allowed to forget it.
type Delivery interface {
	common.Service

	// Ack records that a channel pushed this message into its session, and
	// relays it to every watcher. An empty id is ignored. Ack never blocks.
	Ack(messageID string)

	// Watch returns a stream of delivery events and a stop func. The channel is
	// buffered; a watcher that stalls loses events rather than stalling the
	// backend. A lost ack degrades the UI to "still waiting", never to a false
	// "delivered" — that is the only safe direction for this failure.
	Watch() (<-chan entity.DeliveryEvent, func())
}
