package eth

import (
	"sync"
	"sync/atomic"
)

// FCUState represents the Fork Choice Update state with Latest, Safe, and Finalized block numbers
type FCUState struct {
	Latest    uint64 `json:"latest"`
	Safe      uint64 `json:"safe"`
	Finalized uint64 `json:"finalized"`
}

type SyncTesterSession struct {
	mu     sync.RWMutex
	closed atomic.Bool

	SessionID string `json:"sessionID"`

	// Non canonical view of the chain
	Validated uint64 `json:"validated"`
	// Canonical view of the chain
	CurrentState FCUState `json:"currentState"`
	// payloads
	Payloads map[PayloadID]*ExecutionPayloadEnvelope `json:"-"`

	InitialState FCUState `json:"initialState"`
}

func (s *SyncTesterSession) UpdateFCUState(latest, safe, finalized uint64) {
	s.CurrentState.Latest = latest
	s.CurrentState.Safe = safe
	s.CurrentState.Finalized = finalized
}

func (s *SyncTesterSession) Close() {
	s.closed.Store(true)
}

func (s *SyncTesterSession) IsClosed() bool {
	return s.closed.Load()
}

func (s *SyncTesterSession) Lock() {
	s.mu.Lock()
}

func (s *SyncTesterSession) RLock() {
	s.mu.RLock()
}

func (s *SyncTesterSession) Unlock() {
	s.mu.Unlock()
}

func (s *SyncTesterSession) RUnlock() {
	s.mu.RUnlock()
}

func NewSyncTesterSession(sessionID string, latest, safe, finalized uint64) *SyncTesterSession {
	return &SyncTesterSession{
		SessionID: sessionID,
		Validated: latest,
		CurrentState: FCUState{
			Latest:    latest,
			Safe:      safe,
			Finalized: finalized,
		},
		Payloads: make(map[PayloadID]*ExecutionPayloadEnvelope),
		InitialState: FCUState{
			Latest:    latest,
			Safe:      safe,
			Finalized: finalized,
		},
	}
}
