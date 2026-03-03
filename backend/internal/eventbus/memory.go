package eventbus

import (
	"context"
	"sync"
)

const defaultBufferSize = 256

// memoryBus is an in-memory implementation of EventBus using Go channels.
type memoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan EventEnvelope
}

// New creates a new in-memory EventBus.
func New() EventBus {
	return &memoryBus{
		subscribers: make(map[string][]chan EventEnvelope),
	}
}

func (b *memoryBus) Publish(_ context.Context, envelope EventEnvelope) error {
	b.mu.RLock()
	subs := b.subscribers[envelope.Type]
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- envelope:
		default:
			// Buffer full — drop event for this subscriber rather than blocking.
		}
	}
	return nil
}

func (b *memoryBus) Subscribe(eventType string) <-chan EventEnvelope {
	ch := make(chan EventEnvelope, defaultBufferSize)
	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	b.mu.Unlock()
	return ch
}
