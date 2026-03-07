package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Emitter creates and dispatches events.
type Emitter struct {
	store      Store
	pipelineID string
	pr         int
	commit     string
	branch     string
}

// NewEmitter creates a new event emitter bound to a pipeline context.
func NewEmitter(store Store, pipelineID string, pr int, commit, branch string) *Emitter {
	return &Emitter{
		store:      store,
		pipelineID: pipelineID,
		pr:         pr,
		commit:     commit,
		branch:     branch,
	}
}

// Emit creates and stores an event.
func (e *Emitter) Emit(eventType model.EventType, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := model.Event{
		ID:         generateID(),
		Type:       eventType,
		Timestamp:  time.Now().UTC(),
		PipelineID: e.pipelineID,
		PR:         e.pr,
		Commit:     e.commit,
		Branch:     e.branch,
		Payload:    payloadBytes,
	}

	return e.store.Emit(event)
}

// Store is the interface for persisting events.
type Store interface {
	Emit(event model.Event) error
	Query(filter EventFilter) ([]model.Event, error)
}

// EventFilter constrains event queries.
type EventFilter struct {
	Types      []model.EventType
	After      time.Time
	Before     time.Time
	PR         *int
	PipelineID *string
	Language   *string
	Limit      int
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
