package esd

import "encoding/json"

// EventStore is the interface that needs to be satisfied for all the eventstore implementation
type EventStore interface {
	RetrieveEvents(*RetrieveEventsOption) ([]Event, error)
	CreateEvent(Event) (Event, error)
	CreateSnapshot(*CreateSnapshotOption) (bool, error)
	RetrieveSnapshot(*RetrieveSnapshotOption) (Snapshot, error)
}

// Event that is being store in the eventstore service
type Event struct {
	ID                    string                 `json:"id"`
	AggregateID           string                 `json:"aggregateId"`
	AggregateType         uint64                 `json:"aggregateType"`
	AggregateVersion      uint64                 `json:"aggregateVersion"`
	AggregateClassVersion uint32                 `json:"aggregateClassVersion"`
	Type                  string                 `json:"type"`
	Body                  map[string]interface{} `json:"body"`
	Timestamp             uint64                 `json:"timestamp"`
}

// Snapshot holds the state of the aggregate with it's type and version
type Snapshot struct {
	AggregateID      string          `json:"aggregateId"`
	AggregateType    uint64          `json:"aggregateType"`
	AggregateVersion uint64          `json:"aggregateVersion"`
	State            json.RawMessage `json:"state"`
}

// RetrieveEventsOption for matching events in the eventstore
type RetrieveEventsOption struct {
	AggregateID           string  // Match all events that matches the aggregate id
	AggregateType         uint64  // Match all events that is only on the aggregate types
	SinceAggregateVersion uint64  // Match all events after the provider aggregate version
	SinceID               *string // Match all events after the provider event
}

// CreateSnapshotOption for creating snapshot for the eventstore
type CreateSnapshotOption struct {
	AggregateVersion uint64
	AggregateID      string
	AggregateType    uint64
	State            interface{}
}

// RetrieveSnapshotOption to retrieve snapshot of the provided id and type
type RetrieveSnapshotOption struct {
	AggregateID   string
	AggregateType uint64
}