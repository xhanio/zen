package entity

// ChannelAckKind is the Kind value on a ChannelAck frame.
const ChannelAckKind = "ack"

// ChannelAck is written by a channel back up the /_stream/ws socket after it
// has successfully pushed a notification into its Claude Code session.
//
// It is a transport fact: the channel knows whether Push succeeded. It says
// nothing about whether the model read the message or intends to answer — that
// is unobservable by design, and pretending otherwise is what a typing
// indicator would have done.
type ChannelAck struct {
	Kind      string `json:"kind"`
	MessageID string `json:"message_id"`
}

// DeliveryStateDelivered is the only state a DeliveryEvent carries today.
// "Posted" is known from the HTTP 201 and "replied" from the assistant message,
// so neither needs an event.
const DeliveryStateDelivered = "delivered"

// DeliveryEvent announces that one message reached one session.
//
// Unlike a presence snapshot, this is a discrete event: a newer one does not
// subsume an older one, so it must never be coalesced.
type DeliveryEvent struct {
	MessageID string `json:"message_id"`
	State     string `json:"state"`
}
