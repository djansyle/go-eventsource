package esd

import "sync"

// EventHandler handles the event and apply the change to the state.
type EventHandler func(Event) error

// ApplyHandler holds the information of what `Event` does the `Handler` can process
type ApplyHandler struct {
	EventType             string
	AggregateType         uint64
	AggregateClassVersion uint32
	Handler               EventHandler
}

// CanHandle base method for determining whether the `Handler` can process the event
func (ah *ApplyHandler) CanHandle(event Event) bool {
	return ah.EventType == event.Type && ah.AggregateType == event.AggregateType && ah.AggregateClassVersion == event.AggregateClassVersion
}

// Aggregate is the base struct for all aggregates
type Aggregate struct {
	sync.RWMutex
	ID      string
	Type    uint64
	Version uint64

	applyHanders []*ApplyHandler

	rabbitEventStore EventStore
}

// AggregateHandler is the interface that needs to be satisfied by all aggregates
type AggregateHandler interface {
	AddApplyHandler(*ApplyHandler)
	Apply(Event) error

	Fold(AggregateHandler) error
	RetrieveEvents() ([]Event, error)

	TakeSnapshot(AggregateHandler) error
	GetSnapshot() (*Snapshot, error)
	LoadFromSnapshot(AggregateHandler) error
	ApplySnapshot(*Snapshot) error

	SerializeState() (string, error)
}

// AddApplyHandler adds the handler to the apply handler list
func (a *Aggregate) AddApplyHandler(handler *ApplyHandler) {
	a.Lock()
	defer a.Unlock()

	a.applyHanders = append(a.applyHanders, handler)
}

// Fold applies the series of event to the aggregate
func (a *Aggregate) Fold(aggregate AggregateHandler) error {
	events, err := a.RetrieveEvents()
	if err != nil {
		return err
	}

	a.Lock()
	defer a.Unlock()
	for _, event := range events {
		err = aggregate.Apply(event)

		if err != nil {
			return err
		}

		a.Version = event.AggregateVersion
	}

	return nil
}

// Apply loops through the handlers and try to process the event
func (a *Aggregate) Apply(event Event) error {
	for _, handler := range a.applyHanders {
		if !handler.CanHandle(event) {
			continue
		}

		err := handler.Handler(event)
		if err != nil {
			return err
		}
	}

	return nil
}

// RetrieveEvents retrieves the event of the aggregate prior to the current version
func (a *Aggregate) RetrieveEvents() ([]Event, error) {
	a.RLock()
	defer a.RUnlock()

	events, err := a.rabbitEventStore.RetrieveEvents(
		&RetrieveEventsOption{
			AggregateID:           a.ID,
			AggregateType:         a.Type,
			SinceAggregateVersion: a.Version,
		})

	if err != nil {
		return nil, err
	}

	return events, nil
}

// ApplyAndSave tries to save the event and apply when there were no error saving the event
func (a *Aggregate) ApplyAndSave(aggregate AggregateHandler, event Event) error {
	a.Lock()
	defer a.Unlock()

	savedEvent, err := a.rabbitEventStore.CreateEvent(event)
	if err != nil {
		return err
	}

	err = aggregate.Apply(savedEvent)
	if err != nil {
		return err
	}

	a.Version = savedEvent.AggregateVersion
	return nil
}

// GetSnapshot retrieves the latest snapshot from the eventstore of the aggregate
func (a *Aggregate) GetSnapshot() (*Snapshot, error) {
	snapshot, err := a.rabbitEventStore.RetrieveSnapshot(&RetrieveSnapshotOption{
		AggregateID:   a.ID,
		AggregateType: a.Type,
	})

	if err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// TakeSnapshot creates a new snapshot of the current state of the aggregate and send to eventstore
func (a *Aggregate) TakeSnapshot(aggregate AggregateHandler) error {
	a.RLock()
	defer a.RUnlock()

	serialized, err := aggregate.SerializeState()
	if err != nil {
		return err
	}

	_, err = a.rabbitEventStore.CreateSnapshot(&CreateSnapshotOption{
		AggregateVersion: a.Version,
		AggregateID:      a.ID,
		AggregateType:    a.Type,
		State:            serialized,
	})

	return err
}

// NewApplyHandler creates a new instance of the ApplyHandler for the current aggregate type
func (a *Aggregate) NewApplyHandler(eventType string, aggregateClassVersion uint32, handler EventHandler) *ApplyHandler {
	return &ApplyHandler{
		EventType:             eventType,
		AggregateType:         a.Type,
		AggregateClassVersion: aggregateClassVersion,
		Handler:               handler,
	}
}

// LoadFromSnapshot retrieves the latest snapshot of the aggregate and applies it to the `aggregate`
func (a *Aggregate) LoadFromSnapshot(aggregate AggregateHandler) error {
	a.Lock()
	defer a.Unlock()

	snapshot, err := a.GetSnapshot()
	if err != nil {
		return err
	}

	aggregate.ApplySnapshot(snapshot)
	a.Version = snapshot.AggregateVersion
	return nil
}
