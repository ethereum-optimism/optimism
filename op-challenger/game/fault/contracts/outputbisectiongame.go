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

var (
	methodGenesisBlockNumber = "genesisBlockNumber"
	methodGenesisOutputRoot  = "genesisOutputRoot"
	methodSplitDepth         = "splitDepth"
	methodL2BlockNumber      = "l2BlockNumber"
)

type OutputBisectionGameContract struct {
	disputeGameContract
}

func NewOutputBisectionGameContract(addr common.Address, caller *batching.MultiCaller) (*OutputBisectionGameContract, error) {
	contractAbi, err := bindings.OutputBisectionGameMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load output bisection game ABI: %w", err)
	}

	return &OutputBisectionGameContract{
		disputeGameContract: disputeGameContract{
			multiCaller: caller,
			contract:    batching.NewBoundContract(contractAbi, addr),
			version:     1,
		},
	}, nil
}

// GetBlockRange returns the block numbers of the absolute pre-state block (typically genesis or the bedrock activation block)
// and the post-state block (that the proposed output root is for).
func (c *OutputBisectionGameContract) GetBlockRange(ctx context.Context) (prestateBlock uint64, poststateBlock uint64, retErr error) {
	results, err := c.multiCaller.Call(ctx, batching.BlockLatest,
		c.contract.Call(methodGenesisBlockNumber),
		c.contract.Call(methodL2BlockNumber))
	if err != nil {
		retErr = fmt.Errorf("failed to retrieve game block range: %w", err)
		return
	}
	if len(results) != 2 {
		retErr = fmt.Errorf("expected 2 results but got %v", len(results))
		return
	}
	prestateBlock = results[0].GetBigInt(0).Uint64()
	poststateBlock = results[1].GetBigInt(0).Uint64()
	return
}

func (c *OutputBisectionGameContract) GetGenesisOutputRoot(ctx context.Context) (common.Hash, error) {
	genesisOutputRoot, err := c.multiCaller.SingleCall(ctx, batching.BlockLatest, c.contract.Call(methodGenesisOutputRoot))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to retrieve genesis output root: %w", err)
	}
	return genesisOutputRoot.GetHash(0), nil
}

func (c *OutputBisectionGameContract) GetSplitDepth(ctx context.Context) (uint64, error) {
	splitDepth, err := c.multiCaller.SingleCall(ctx, batching.BlockLatest, c.contract.Call(methodSplitDepth))
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve split depth: %w", err)
	}
	return splitDepth.GetBigInt(0).Uint64(), nil
}

func (f *OutputBisectionGameContract) UpdateOracleTx(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	if data.IsLocal {
		return f.addLocalDataTx(claimIdx, data)
	}
	return f.addGlobalDataTx(ctx, data)
}

func (f *OutputBisectionGameContract) addLocalDataTx(claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	call := f.contract.Call(
		methodAddLocalData,
		data.GetIdent(),
		new(big.Int).SetUint64(claimIdx),
		new(big.Int).SetUint64(uint64(data.OracleOffset)),
	)
	return call.ToTxCandidate()
}

func (f *OutputBisectionGameContract) addGlobalDataTx(ctx context.Context, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
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
