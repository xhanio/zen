package presence

import (
	"sort"
	"sync"

	"github.com/xhanio/zen/pkg/types/entity"
	"github.com/xhanio/zen/pkg/types/model"
)

var _ model.Registration = (*registration)(nil)

type registration struct {
	ch      *entity.Channel
	evicted chan struct{}
	once    sync.Once
}

func (r *registration) Channel() *entity.Channel { return r.ch }
func (r *registration) Evicted() <-chan struct{} { return r.evicted }
func (r *registration) evict()                   { r.once.Do(func() { close(r.evicted) }) }

type watcher struct {
	ch   chan []*entity.Channel
	once sync.Once
}

func (m *manager) Register(ch *entity.Channel) model.Registration {
	reg := &registration{ch: ch, evicted: make(chan struct{})}

	m.mu.Lock()
	if old, ok := m.bySession[ch.SessionID]; ok {
		m.log.Warnf("session %s reconnected; evicting stale channel %s", ch.SessionID, old.ch.InstanceID)
		old.evict()
	}
	m.bySession[ch.SessionID] = reg
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	m.log.Infof("channel %s registered for session %s (cwd %s)", ch.InstanceID, ch.SessionID, ch.Cwd)
	m.broadcast(snapshot)
	return reg
}

// Unregister removes the registration a caller received from Register, and only
// that one. Identity is the registration itself, not its InstanceID: a channel
// process keeps one InstanceID across reconnects, so a stale connection and the
// connection that replaced it can carry the same InstanceID. Matching on the ID
// would let a losing connection's deferred Unregister evict its own successor.
func (m *manager) Unregister(reg model.Registration) {
	own, ok := reg.(*registration)
	if !ok || own == nil {
		return
	}

	m.mu.Lock()
	current, exists := m.bySession[own.ch.SessionID]
	if !exists || current != own {
		// Already displaced by a newer connection. Nothing of ours to remove.
		m.mu.Unlock()
		own.evict()
		return
	}
	delete(m.bySession, own.ch.SessionID)
	own.evict()
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	m.log.Infof("channel %s unregistered", own.ch.InstanceID)
	m.broadcast(snapshot)
}

func (m *manager) Has(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.bySession[sessionID]
	return ok
}

func (m *manager) Get(sessionID string) (*entity.Channel, bool) {
	if sessionID == "" {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	reg, ok := m.bySession[sessionID]
	if !ok {
		return nil, false
	}
	return reg.ch, true
}

func (m *manager) List() []*entity.Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshotLocked()
}

func (m *manager) Watch() (<-chan []*entity.Channel, func()) {
	w := &watcher{ch: make(chan []*entity.Channel, 1)}

	m.mu.Lock()
	m.watchers[w] = struct{}{}
	m.mu.Unlock()

	stop := func() {
		m.mu.Lock()
		_, live := m.watchers[w]
		delete(m.watchers, w)
		m.mu.Unlock()
		if live {
			w.once.Do(func() { close(w.ch) })
		}
	}
	return w.ch, stop
}

// snapshotLocked copies the registry, newest connection first. Callers hold
// m.mu (read or write).
func (m *manager) snapshotLocked() []*entity.Channel {
	out := make([]*entity.Channel, 0, len(m.bySession))
	for _, reg := range m.bySession {
		out = append(out, reg.ch)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ConnectedAt.After(out[j].ConnectedAt)
	})
	return out
}

// broadcast delivers the snapshot to every watcher without ever blocking the
// registry. A watcher's channel has depth 1; if a snapshot is already queued we
// drain it and enqueue the newer one. Coalescing is safe because each snapshot
// is the complete registry, so the newest one subsumes every earlier one —
// unlike a message stream, where dropping loses information.
//
// Holding m.mu.RLock here also makes the send safe against stop(): stop takes
// the write lock to remove the watcher before it closes the channel, so a
// watcher we can still see under RLock cannot be concurrently closed.
func (m *manager) broadcast(snapshot []*entity.Channel) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for w := range m.watchers {
		select {
		case w.ch <- snapshot:
		default:
			select {
			case <-w.ch:
			default:
			}
			select {
			case w.ch <- snapshot:
			default:
			}
		}
	}
}
