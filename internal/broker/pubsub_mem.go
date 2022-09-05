package broker

import (
	"github.com/google/uuid"
	"sync"
)

type memoryPubsub struct {
	mut       sync.RWMutex
	listeners map[uint64]map[uuid.UUID]Listener
}

func (m *memoryPubsub) Subscribe(tailnet uint64, listener Listener) (cancel func(), err error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	var listeners map[uuid.UUID]Listener
	var ok bool
	if listeners, ok = m.listeners[tailnet]; !ok {
		listeners = map[uuid.UUID]Listener{}
		m.listeners[tailnet] = listeners
	}
	var id uuid.UUID
	for {
		id = uuid.New()
		if _, ok = listeners[id]; !ok {
			break
		}
	}

	listeners[id] = listener
	return func() {
		m.mut.Lock()
		defer m.mut.Unlock()
		listeners := m.listeners[tailnet]
		delete(listeners, id)
	}, nil
}

func (m *memoryPubsub) Publish(tailnet uint64, message *Signal) error {
	m.mut.RLock()
	defer m.mut.RUnlock()
	listeners, ok := m.listeners[tailnet]
	if !ok {
		return nil
	}
	for _, listener := range listeners {
		listener <- message
	}
	return nil
}

func (*memoryPubsub) Close() error {
	return nil
}

func NewPubsubInMemory() Pubsub {
	return &memoryPubsub{
		listeners: make(map[uint64]map[uuid.UUID]Listener),
	}
}
