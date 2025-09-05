package eth

import (
	"sync"
)

// FCUState represents the Fork Choice Update state with Latest, Safe, and Finalized block numbers
type FCUState struct {
	Latest    uint64 `json:"latest"`
	Safe      uint64 `json:"safe"`
	Finalized uint64 `json:"finalized"`
}

type SyncTesterSession struct {
	sync.Mutex

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
