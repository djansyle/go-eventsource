package eventsource

import (
	"encoding/json"
	"testing"

	mapstructure "github.com/mitchellh/mapstructure"
)

type accountMockEventStore struct{}

func (a *accountMockEventStore) RetrieveEvents(opt *RetrieveEventsOption) ([]Event, error) {
	if opt.AggregateID == "uniqueid" {
		return []Event{
			Event{AggregateID: "uniqueid", AggregateClassVersion: 1, AggregateType: 1, AggregateVersion: 1, Body: map[string]interface{}{"firstName": "John", "lastName": "Joe"}, Type: "AccountCreated"},
			Event{AggregateID: "uniqueid", AggregateClassVersion: 1, AggregateType: 1, AggregateVersion: 1, Body: map[string]interface{}{"lastName": "Doe"}, Type: "AccountUpdated"},
		}, nil
	}

	return []Event{}, nil
}

func (a *accountMockEventStore) CreateEvent(event Event) (Event, error) {
	return Event{}, nil
}

func (a *accountMockEventStore) CreateSnapshot(opt *CreateSnapshotOption) (bool, error) {
	return true, nil
}

func (a *accountMockEventStore) RetrieveSnapshot(opt *RetrieveSnapshotOption) (Snapshot, error) {
	return Snapshot{
		AggregateID:      "uniqueid",
		AggregateType:    1,
		AggregateVersion: 20,
		State:            []byte("{\"firstName\": \"Bob\", \"lastName\": \"Bar\"}"),
	}, nil
}

type accountState struct {
	FirstName string
	LastName  string
}

type accountAggregate struct {
	Aggregate

	State *accountState
}

type AccountCreatedEvent struct {
	FirstName string `mapstructure:"firstName"`
	LastName  string `mapstructure:"lastName"`
}

type AccountUpdatedEvent struct {
	FirstName *string `mapstructure:"firstName"`
	LastName  *string `mapstructure:"lastName"`
}

func newAccountAggregate() *accountAggregate {
	aggregate := &accountAggregate{
		Aggregate: Aggregate{aggregateID: "uniqueid", aggregateType: 1, aggregateVersion: 0, rabbitEventStore: &accountMockEventStore{}},
		State:     &accountState{}}

	aggregate.AddApplyHandler(aggregate.NewApplyHandler("AccountCreated", 1, aggregate.accountCreatedHandler))
	aggregate.AddApplyHandler(aggregate.NewApplyHandler("AccountUpdated", 1, aggregate.accountUpdatedHandler))

	return aggregate
}

func (a *accountAggregate) accountCreatedHandler(event Event) error {
	var body AccountCreatedEvent
	err := mapstructure.Decode(event.Body, &body)
	if err != nil {
		return err
	}

	a.State.FirstName = body.FirstName
	a.State.LastName = body.LastName
	return nil
}

func (a *accountAggregate) accountUpdatedHandler(event Event) error {
	var body AccountUpdatedEvent
	err := mapstructure.Decode(event.Body, &body)
	if err != nil {
		return err
	}

	if body.FirstName != nil {
		a.State.FirstName = *body.FirstName
	}

	if body.LastName != nil {
		a.State.LastName = *body.LastName
	}
	return nil

}

func (a *accountAggregate) SerializeState() (string, error) {
	str, err := json.Marshal(a.State)

	if err != nil {
		return "", err
	}

	return string(str), nil
}

func (a *accountAggregate) ApplySnapshot(snapshot *Snapshot) error {
	err := json.Unmarshal(snapshot.State, a.State)
	return err
}

// TestAggregate tests the implementation of the Aggregate
func TestAggregate(t *testing.T) {
	aggregate := newAccountAggregate()
	aggregate.Fold(aggregate)

	if aggregate.State.FirstName != "John" {
		t.Fatalf("expecting value to be %q but got %q", "john", aggregate.State.FirstName)
	}

	if aggregate.State.LastName != "Doe" {
		t.Fatalf("expecting value to be %q but got %q", "Doe", aggregate.State.LastName)
	}

	aggregate.LoadFromSnapshot(aggregate)
	if aggregate.State.FirstName != "Bob" {
		t.Fatalf("expecting value to be %q but got %q", "Bob", aggregate.State.FirstName)
	}

	if aggregate.State.LastName != "Bar" {
		t.Fatalf("expecting value to be %q but got %q", "Bar", aggregate.State.LastName)
	}

	if aggregate.Version() != 20 {
		t.Fatalf("expecting value to be %d but got %d", 20, aggregate.Version())
	}
}
