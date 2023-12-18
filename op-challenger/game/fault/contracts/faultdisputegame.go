package contracts

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodProposals = "proposals"
)

type FaultDisputeGameContract struct {
	disputeGameContract
}

func NewFaultDisputeGameContract(addr common.Address, caller *batching.MultiCaller) (*FaultDisputeGameContract, error) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load fault dispute game ABI: %w", err)
	}

	return &FaultDisputeGameContract{
		disputeGameContract: disputeGameContract{
			multiCaller: caller,
			contract:    batching.NewBoundContract(fdgAbi, addr),
			version:     0,
		},
	}, nil
}

// GetProposals returns the agreed and disputed proposals
func (f *FaultDisputeGameContract) GetProposals(ctx context.Context) (Proposal, Proposal, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodProposals))
	if err != nil {
		return Proposal{}, Proposal{}, fmt.Errorf("failed to fetch proposals: %w", err)
	}

	var agreed, disputed contractProposal
	result.GetStruct(0, &agreed)
	result.GetStruct(1, &disputed)
	return asProposal(agreed), asProposal(disputed), nil
}

func (f *FaultDisputeGameContract) UpdateOracleTx(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	if data.IsLocal {
		return f.addLocalDataTx(data)
	}
	return f.addGlobalDataTx(ctx, data)
}

func (f *FaultDisputeGameContract) addLocalDataTx(data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	call := f.contract.Call(
		methodAddLocalData,
		data.GetIdent(),
		types.NoLocalContext,
		new(big.Int).SetUint64(uint64(data.OracleOffset)),
	)
	return call.ToTxCandidate()
}

func (f *FaultDisputeGameContract) addGlobalDataTx(ctx context.Context, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	vm, err := f.vm(ctx)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	oracle, err := vm.Oracle(ctx)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	return oracle.AddGlobalDataTx(data)
}
