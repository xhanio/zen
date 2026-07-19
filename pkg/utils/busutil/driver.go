package busutil

import (
	"github.com/xhanio/framingo/pkg/services/pubsub/driver"
	"github.com/xhanio/framingo/pkg/utils/log"
)

// NewDriver builds Zen's in-memory bus with DropSubscriber rather than the
// default DropMessage.
//
// Both of Zen's consumers — the browser and the channel — reconnect and resume
// from their own cursor, so closing a wedged subscriber's channel costs it a
// round trip. A silently dropped message costs it the message, with nothing
// anywhere to say so. Message loss is unrecoverable and invisible; connection
// loss is recoverable and obvious.
//
// extra exists so a test can shrink the queue cap; production passes none.
func NewDriver(logger log.Logger, extra ...driver.Option) driver.Driver {
	opts := append([]driver.Option{driver.WithOnFull(driver.DropSubscriber)}, extra...)
	return driver.NewMemory(logger, opts...)
}
