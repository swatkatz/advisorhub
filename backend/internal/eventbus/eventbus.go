package eventbus

import (
	"context"
	"encoding/json"
	"time"
)

// EventSource indicates which subsystem produced the event.
type EventSource string

const (
	SourceReactive   EventSource = "REACTIVE"
	SourceTemporal   EventSource = "TEMPORAL"
	SourceAnalytical EventSource = "ANALYTICAL"
	SourceSystem     EventSource = "SYSTEM"
)

// EntityType is a typed constant for the kind of entity an event concerns.
type EntityType string

const (
	EntityTypeClient          EntityType = "Client"
	EntityTypeAccount         EntityType = "Account"
	EntityTypeTransfer        EntityType = "Transfer"
	EntityTypeRESPBeneficiary EntityType = "RESPBeneficiary"
)

// EventEnvelope is the canonical event wrapper used across all contexts.
type EventEnvelope struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	EntityID   string          `json:"entity_id"`
	EntityType EntityType      `json:"entity_type"`
	Payload    json.RawMessage `json:"payload"`
	Source     EventSource     `json:"source"`
	Timestamp  time.Time       `json:"timestamp"`
}

// EventBus defines the pub/sub interface for routing domain events.
type EventBus interface {
	// Publish sends an event to all subscribers of envelope.Type.
	// Non-blocking: uses buffered channels so publishers are not blocked by slow consumers.
	Publish(ctx context.Context, envelope EventEnvelope) error

	// Subscribe returns a read-only channel that receives all events matching eventType.
	// Multiple subscribers per event type are supported.
	Subscribe(eventType string) <-chan EventEnvelope
}
