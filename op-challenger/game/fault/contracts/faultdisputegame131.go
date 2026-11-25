package contracts

import (
	"context"
	_ "embed"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

//go:embed abis/FaultDisputeGame-1.3.1.json
var faultDisputeGameAbi131 []byte

type FaultDisputeGameContract131 struct {
	FaultDisputeGameContractLatest
}

func isLegacyGameClosed(ctx context.Context, game DisputeGameContract) (bool, error) {
	// Legacy games have no separate close cycle.
	status, err := game.GetStatus(ctx)
	if err != nil {
		return false, err
	}
	return status != gameTypes.GameStatusInProgress, nil
}

func (f *FaultDisputeGameContract131) IsClosed(ctx context.Context) (bool, error) {
	return isLegacyGameClosed(ctx, f)
}

func (f *FaultDisputeGameContract131) GetBondDistributionMode(ctx context.Context, block rpcblock.Block) (types.BondDistributionMode, error) {
	return types.LegacyDistributionMode, nil
}

func (f *FaultDisputeGameContract131) CloseGameTx(ctx context.Context) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, ErrCloseGameNotSupported
}
