package syncstatus

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

var _ activity.RPCActivity = (*SyncStatus)(nil)

// SyncStatus is an RPCActivity that provides a unified view of sync progress
// across all managed chains and verification activities.
type SyncStatus struct {
	chains    map[eth.ChainID]cc.ChainContainer
	verifiers []activity.VerificationActivity
}

// New creates a new SyncStatus activity.
func New(chains map[eth.ChainID]cc.ChainContainer, verifiers []activity.VerificationActivity) *SyncStatus {
	return &SyncStatus{
		chains:    chains,
		verifiers: verifiers,
	}
}

func (s *SyncStatus) Reset(eth.ChainID, uint64) {}

func (s *SyncStatus) RPCNamespace() string { return "supernode" }
func (s *SyncStatus) RPCService() any      { return &syncStatusAPI{s: s} }

// SyncStatusResponse is the JSON-RPC response for supernode_syncStatus.
type SyncStatusResponse struct {
	Chains    map[eth.ChainID]*eth.SyncStatus `json:"chains"`
	Verifiers []VerifierStatus                `json:"verifiers"`
}

// VerifierStatus describes the current state of a single verification activity.
type VerifierStatus struct {
	Name                    string      `json:"name"`
	CurrentL1               eth.BlockID `json:"currentL1"`
	LatestVerifiedTimestamp uint64      `json:"latestVerifiedTimestamp"`
	Initialized             bool        `json:"initialized"`
}

type syncStatusAPI struct{ s *SyncStatus }

// SyncStatus returns the sync status of all chains and verifiers.
func (api *syncStatusAPI) SyncStatus(ctx context.Context) (*SyncStatusResponse, error) {
	return api.s.syncStatus(ctx)
}

func (s *SyncStatus) syncStatus(ctx context.Context) (*SyncStatusResponse, error) {
	chains := make(map[eth.ChainID]*eth.SyncStatus, len(s.chains))
	for chainID, chain := range s.chains {
		status, err := chain.SyncStatus(ctx)
		if err != nil {
			return nil, err
		}
		chains[chainID] = status
	}

	verifiers := make([]VerifierStatus, 0, len(s.verifiers))
	for _, v := range s.verifiers {
		ts, initialized := v.LatestVerifiedTimestamp()
		verifiers = append(verifiers, VerifierStatus{
			Name:                    v.Name(),
			CurrentL1:               v.CurrentL1(),
			LatestVerifiedTimestamp: ts,
			Initialized:             initialized,
		})
	}

	return &SyncStatusResponse{
		Chains:    chains,
		Verifiers: verifiers,
	}, nil
}
