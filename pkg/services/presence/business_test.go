package presence

import (
	"testing"
	"time"

	"github.com/xhanio/zen/pkg/types/entity"
)

func chn(instance, session string) *entity.Channel {
	return &entity.Channel{
		InstanceID:  instance,
		SessionID:   session,
		Cwd:         "/repo",
		StartedAt:   time.Now(),
		ConnectedAt: time.Now(),
	}
}

func TestGet_ReturnsRegisteredChannel(t *testing.T) {
	m := New()
	m.Register(chn("i1", "sess-A"))
	got, ok := m.Get("sess-A")
	if !ok {
		t.Fatal("Get(sess-A) = !ok, want ok")
	}
	if got.Cwd != "/repo" {
		t.Fatalf("cwd = %q, want /repo", got.Cwd)
	}
	if _, ok := m.Get("unknown"); ok {
		t.Fatal("Get(unknown) = ok, want !ok")
	}
	if _, ok := m.Get(""); ok {
		t.Fatal(`Get("") = ok, want !ok`)
	}
}

func TestRegister_ListsChannel(t *testing.T) {
	p := New()
	p.Register(chn("i1", "s1"))

	got := p.List()
	if len(got) != 1 {
		t.Fatalf("List() len = %d, want 1", len(got))
	}
	if got[0].InstanceID != "i1" || got[0].SessionID != "s1" {
		t.Fatalf("List()[0] = %+v", got[0])
	}
}

func TestRegister_SameSessionEvictsOlder(t *testing.T) {
	p := New()
	first := p.Register(chn("i1", "s1"))
	second := p.Register(chn("i2", "s1"))

	select {
	case <-first.Evicted():
	case <-time.After(time.Second):
		t.Fatal("first registration was not evicted")
	}

	select {
	case <-second.Evicted():
		t.Fatal("second registration must not be evicted")
	default:
	}

	got := p.List()
	if len(got) != 1 || got[0].InstanceID != "i2" {
		t.Fatalf("List() = %+v, want only i2", got)
	}
}

func TestRegister_DifferentSessionsCoexist(t *testing.T) {
	p := New()
	p.Register(chn("i1", "s1"))
	p.Register(chn("i2", "s2"))

	if got := p.List(); len(got) != 2 {
		t.Fatalf("List() len = %d, want 2", len(got))
	}
}

// A displaced connection runs its deferred Unregister after losing the
// SessionID. That must not remove the channel that replaced it.
func TestUnregister_DisplacedInstanceIsNoOp(t *testing.T) {
	p := New()
	first := p.Register(chn("i1", "s1"))
	p.Register(chn("i2", "s1"))

	p.Unregister(first)

	got := p.List()
	if len(got) != 1 || got[0].InstanceID != "i2" {
		t.Fatalf("List() = %+v, want only i2", got)
	}
}

func TestUnregister_RemovesLiveChannel(t *testing.T) {
	p := New()
	reg := p.Register(chn("i1", "s1"))
	p.Unregister(reg)

	if got := p.List(); len(got) != 0 {
		t.Fatalf("List() len = %d, want 0", len(got))
	}
}

func TestWatch_ReceivesSnapshotOnChange(t *testing.T) {
	p := New()
	snaps, stop := p.Watch()
	defer stop()

	p.Register(chn("i1", "s1"))

	select {
	case snap := <-snaps:
		if len(snap) != 1 || snap[0].InstanceID != "i1" {
			t.Fatalf("snapshot = %+v", snap)
		}
	case <-time.After(time.Second):
		t.Fatal("no snapshot delivered")
	}
}

// The registry must never block on a watcher that is not reading, and the
// watcher must end up with the newest snapshot rather than an old one.
func TestWatch_CoalescesAndNeverBlocks(t *testing.T) {
	p := New()
	snaps, stop := p.Watch()
	defer stop()

	p.Register(chn("i1", "s1"))
	p.Register(chn("i2", "s2"))
	p.Register(chn("i3", "s3"))

	select {
	case snap := <-snaps:
		if len(snap) != 3 {
			t.Fatalf("coalesced snapshot len = %d, want 3 (the latest)", len(snap))
		}
	case <-time.After(time.Second):
		t.Fatal("no snapshot delivered")
	}
}

func TestWatch_StopClosesChannel(t *testing.T) {
	p := New()
	snaps, stop := p.Watch()
	stop()

	if _, ok := <-snaps; ok {
		t.Fatal("channel should be closed after stop()")
	}
}

// stop() twice must not panic on a double close.
func TestWatch_StopIsIdempotent(t *testing.T) {
	p := New()
	_, stop := p.Watch()
	stop()
	stop()
}

func TestHas_ReportsLiveSessions(t *testing.T) {
	p := New()
	p.Register(chn("i1", "s1"))

	if !p.Has("s1") {
		t.Fatal("Has(s1) = false, want true")
	}
	if p.Has("s-gone") {
		t.Fatal("Has(s-gone) = true, want false")
	}
}

func TestHas_FalseAfterUnregister(t *testing.T) {
	p := New()
	reg := p.Register(chn("i1", "s1"))
	p.Unregister(reg)

	if p.Has("s1") {
		t.Fatal("Has(s1) = true after unregister")
	}
}

// The empty string is never a live session, so a null target can never match.
func TestHas_EmptyStringIsNeverLive(t *testing.T) {
	p := New()
	p.Register(chn("i1", "s1"))

	if p.Has("") {
		t.Fatal(`Has("") = true; a null target must match nobody`)
	}
}

// A channel process keeps one InstanceID across reconnects, so the stale
// connection and the one that replaced it carry the same InstanceID. Matching
// on that ID would let the stale connection's deferred Unregister evict its own
// successor and take the session offline while a channel is holding it.
func TestUnregister_SameInstanceReconnectIsNoOpForTheStaleConnection(t *testing.T) {
	p := New()
	stale := p.Register(chn("i1", "s1"))
	p.Register(chn("i1", "s1")) // same process redials

	p.Unregister(stale)

	if !p.Has("s1") {
		t.Fatal("the stale connection's Unregister removed its replacement")
	}
	got := p.List()
	if len(got) != 1 || got[0].InstanceID != "i1" {
		t.Fatalf("List() = %+v, want the live i1", got)
	}
}
