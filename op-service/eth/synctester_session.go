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

// ELSyncPolicy defines the policy for determining the synchronization status
// of the execution layer (EL) during EL Sync, as triggered exclusively by
// ForkchoiceUpdated (FCU) calls.
//
// In the EL Sync process, the consensus layer (CL) notifies the EL of the
// current head via FCU. The EL then evaluates its internal sync state and
// reports whether it is still syncing or fully in sync. An ELSyncPolicy
// implementation encapsulates this decision logic.
//
// The purpose of this interface is to provide a configurable or mockable
// strategy for how the EL responds to FCU-triggered sync checks—useful in
// testing, simulation, or devnet environments where the real EL behavior
// needs to be emulated.
type ELSyncPolicy interface {
	ELSyncStatus(num uint64) ExecutePayloadStatus
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

	ELSyncPolicy ELSyncPolicy `json:"-"`
	ELSyncActive bool         `json:"el_sync_active"`

	InitialState FCUState `json:"initial_state"`
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

func (s *SyncTesterSession) ResetSession() {
	s.CurrentState = s.InitialState
	s.Validated = s.InitialState.Latest
	s.Payloads = make(map[PayloadID]*ExecutionPayloadEnvelope)
}

func (s *SyncTesterSession) IsELSyncActive() bool {
	return s.ELSyncActive
}

func NewSyncTesterSession(sessionID string, latest, safe, finalized uint64, elSyncActive bool, elSyncState ELSyncPolicy) *SyncTesterSession {
	return &SyncTesterSession{
		SessionID: sessionID,
		Validated: latest,
		CurrentState: FCUState{
			Latest:    latest,
			Safe:      safe,
			Finalized: finalized,
		},
		Payloads:     make(map[PayloadID]*ExecutionPayloadEnvelope),
		ELSyncActive: elSyncActive,
		ELSyncPolicy: elSyncState,
		InitialState: FCUState{
			Latest:    latest,
			Safe:      safe,
			Finalized: finalized,
		},
	}
}
