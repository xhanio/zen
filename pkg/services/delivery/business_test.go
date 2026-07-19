package delivery

import (
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/types/entity"
)

func TestAck_ReachesAWatcher(t *testing.T) {
	d := New()
	events, stop := d.Watch()
	defer stop()

	d.Ack("01MSG")

	select {
	case ev := <-events:
		if ev.MessageID != "01MSG" || ev.State != entity.DeliveryStateDelivered {
			t.Fatalf("event = %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("no delivery event")
	}
}

func TestAck_ReachesEveryWatcher(t *testing.T) {
	d := New()
	a, stopA := d.Watch()
	defer stopA()
	b, stopB := d.Watch()
	defer stopB()

	d.Ack("01MSG")

	for i, ch := range []<-chan entity.DeliveryEvent{a, b} {
		select {
		case ev := <-ch:
			if ev.MessageID != "01MSG" {
				t.Fatalf("watcher %d got %+v", i, ev)
			}
		case <-time.After(time.Second):
			t.Fatalf("watcher %d got nothing", i)
		}
	}
}

// Discrete events, not snapshots: two acks must both arrive. If this service
// ever coalesces the way presence does, the second ack silently replaces the
// first and a message is stuck on "waiting" forever.
func TestAck_EventsAreNotCoalesced(t *testing.T) {
	d := New()
	events, stop := d.Watch()
	defer stop()

	d.Ack("01MSG-A")
	d.Ack("01MSG-B")

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case ev := <-events:
			got[ev.MessageID] = true
		case <-time.After(time.Second):
			t.Fatalf("only got %v", got)
		}
	}
	if !got["01MSG-A"] || !got["01MSG-B"] {
		t.Fatalf("lost an ack: %v", got)
	}
}

// A message id the backend never saw is still just an id. Acks are advisory:
// the service does not validate, it relays.
func TestAck_EmptyMessageIDIsIgnored(t *testing.T) {
	d := New()
	events, stop := d.Watch()
	defer stop()

	d.Ack("")

	select {
	case ev := <-events:
		t.Fatalf("empty ack was relayed: %+v", ev)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestAck_NeverBlocksOnAStalledWatcher(t *testing.T) {
	d := New()
	_, stop := d.Watch() // never read from
	defer stop()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 500; i++ {
			d.Ack("01MSG")
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Ack blocked on a watcher that is not reading")
	}
}

func TestWatch_StopClosesChannel(t *testing.T) {
	d := New()
	events, stop := d.Watch()
	stop()

	if _, ok := <-events; ok {
		t.Fatal("channel should be closed after stop()")
	}
}

func TestWatch_StopIsIdempotent(t *testing.T) {
	d := New()
	_, stop := d.Watch()
	stop()
	stop()
}

func TestAck_AfterStopDoesNotPanic(t *testing.T) {
	d := New()
	_, stop := d.Watch()
	stop()
	d.Ack("01MSG") // must not send on a closed channel
}
