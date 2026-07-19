package busutil

import (
	"context"
	"testing"
	"time"

	"github.com/xhanio/framingo/pkg/services/pubsub/driver"
	"github.com/xhanio/framingo/pkg/utils/log"
)

// A subscriber that stops draining must have its channel closed, not have its
// messages silently discarded. The two consumers that read this bus — the
// browser and the channel — both reconnect and resume from a cursor, so a
// closed channel loses nothing while a dropped message loses everything and
// says nothing.
//
// The queue cap is shrunk so the test does not have to publish 50k messages;
// the policy under test is the OnFull choice, not the cap.
func TestPubsubDriver_EvictsSubscriberThatStopsDraining(t *testing.T) {
	d := NewDriver(log.Default, driver.WithQueueCap(1), driver.WithChannelBuffer(1))

	ch, err := d.Subscribe("wedged", "zen")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Never read from ch. Publish until the pending queue overflows and the
	// driver gives up on this subscriber.
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		if err := d.Publish(ctx, "publisher", "zen", "test", i); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	// Drain whatever was buffered; the channel must then be closed rather than
	// merely empty. A DropMessage driver leaves it open forever.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // closed: the subscriber was evicted
			}
		case <-deadline:
			t.Fatal("subscriber was never evicted: the bus is silently dropping messages")
		}
	}
}
