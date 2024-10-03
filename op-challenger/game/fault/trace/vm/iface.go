package vm

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type StateConverter interface {
	// ConvertStateToProof reads the state snapshot at the specified path and converts it to ProofData.
	// Returns the proof data, the VM step the state is from and whether or not the VM had exited.
	ConvertStateToProof(ctx context.Context, statePath string) (*utils.ProofData, uint64, bool, error)
}
