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

	SessionID string `json:"session_id"`

	// Non canonical view of the chain
	Validated uint64 `json:"validated"`
	// Canonical view of the chain
	CurrentState FCUState `json:"current_state"`
	// payloads
	Payloads map[PayloadID]*ExecutionPayloadEnvelope `json:"-"`

	ELSyncTarget uint64 `json:"el_sync_target"`
	ELSyncActive bool   `json:"el_sync_active"`

	InitialState        FCUState `json:"initial_state"`
	InitialELSyncActive bool     `json:"initial_el_sync_active"`
}

func (s *SyncTesterSession) UpdateFCULatest(latest uint64) {
	s.CurrentState.Latest = latest
}

func (s *SyncTesterSession) UpdateFCUSafe(safe uint64) {
	s.CurrentState.Safe = safe
}

func (s *SyncTesterSession) UpdateFCUFinalized(finalized uint64) {
	s.CurrentState.Finalized = finalized
}

func (s *SyncTesterSession) FinishELSync(target uint64) {
	s.ELSyncActive = false
	s.Validated = target
}

func (s *SyncTesterSession) IsELSyncFinished() bool {
	return !s.ELSyncActive
}

func (s *SyncTesterSession) ResetSession() {
	s.CurrentState = s.InitialState
	s.Validated = s.InitialState.Latest
	s.Payloads = make(map[PayloadID]*ExecutionPayloadEnvelope)
	s.ELSyncActive = s.InitialELSyncActive
}

func NewSyncTesterSession(sessionID string, latest, safe, finalized, elSyncTarget uint64, elSyncActive bool) *SyncTesterSession {
	return &SyncTesterSession{
		SessionID: sessionID,
		Validated: latest,
		CurrentState: FCUState{
			Latest:    latest,
			Safe:      safe,
			Finalized: finalized,
		},
		Payloads:     make(map[PayloadID]*ExecutionPayloadEnvelope),
		ELSyncTarget: elSyncTarget,
		ELSyncActive: elSyncActive,
		InitialState: FCUState{
			Latest:    latest,
			Safe:      safe,
			Finalized: finalized,
		},
		InitialELSyncActive: elSyncActive,
	}
}
