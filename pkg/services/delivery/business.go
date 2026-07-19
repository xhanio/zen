package delivery

import (
	"sync"

	"github.com/xhanio/zen/pkg/types/entity"
)

// watcherBuffer is deep enough that a healthy SPA socket never fills it. A
// stalled one loses acks, which is the safe failure: the thread stays on
// "waiting" rather than claiming a delivery that never happened.
const watcherBuffer = 64

type watcher struct {
	ch   chan entity.DeliveryEvent
	once sync.Once
}

func (m *manager) Ack(messageID string) {
	if messageID == "" {
		return
	}
	ev := entity.DeliveryEvent{MessageID: messageID, State: entity.DeliveryStateDelivered}

	// Holding the read lock here makes the send safe against stop(): stop takes
	// the write lock to remove the watcher before closing its channel, so a
	// watcher visible under RLock cannot be concurrently closed.
	m.mu.RLock()
	defer m.mu.RUnlock()
	for w := range m.watchers {
		select {
		case w.ch <- ev:
		default:
			// Never coalesce and never block. Delivery events are discrete: a
			// newer one does not subsume an older one, so replacing a queued
			// event would silently lose an ack. Dropping is loud on purpose.
			m.log.Warnf("delivery watcher is stalled; dropping ack for %s", messageID)
		}
	}
}

func (m *manager) Watch() (<-chan entity.DeliveryEvent, func()) {
	w := &watcher{ch: make(chan entity.DeliveryEvent, watcherBuffer)}

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
