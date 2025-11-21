package contracts

import (
	"context"
	_ "embed"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

//go:embed abis/FaultDisputeGame-1.3.1.json
var faultDisputeGameAbi131 []byte

type FaultDisputeGameContract131 struct {
	FaultDisputeGameContractLatest
}

func (f *FaultDisputeGameContract131) GetBondDistributionMode(ctx context.Context, block rpcblock.Block) (types.BondDistributionMode, error) {
	return types.LegacyDistributionMode, nil
}

func (f *FaultDisputeGameContract131) CloseGameTx(ctx context.Context) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, ErrCloseGameNotSupported
}
