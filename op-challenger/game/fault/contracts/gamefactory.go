package contracts

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	methodGameCount   = "gameCount"
	methodGameAtIndex = "gameAtIndex"
	methodGameImpls   = "gameImpls"
	methodInitBonds   = "initBonds"
	methodCreateGame  = "create"
	methodGames       = "games"

	eventDisputeGameCreated = "DisputeGameCreated"
)

var (
	ErrEventNotFound = errors.New("event not found")
)

type DisputeGameFactoryContract struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
	abi         *abi.ABI
}

func NewDisputeGameFactoryContract(m metrics.ContractMetricer, addr common.Address, caller *batching.MultiCaller) *DisputeGameFactoryContract {
	factoryAbi := snapshots.LoadDisputeGameFactoryABI()
	return &DisputeGameFactoryContract{
		metrics:     m,
		multiCaller: caller,
		contract:    batching.NewBoundContract(factoryAbi, addr),
		abi:         factoryAbi,
	}
}

func (f *DisputeGameFactoryContract) GetGameFromParameters(ctx context.Context, traceType uint32, outputRoot common.Hash, l2BlockNum uint64) (common.Address, error) {
	defer f.metrics.StartContractRequest("GetGameFromParameters")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodGames, traceType, outputRoot, common.BigToHash(big.NewInt(int64(l2BlockNum))).Bytes()))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to fetch game from parameters: %w", err)
	}
	return result.GetAddress(0), nil
}

func (f *DisputeGameFactoryContract) GetGameCount(ctx context.Context, blockHash common.Hash) (uint64, error) {
	defer f.metrics.StartContractRequest("GetGameCount")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.ByHash(blockHash), f.contract.Call(methodGameCount))
	if err != nil {
		return 0, fmt.Errorf("failed to load game count: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *DisputeGameFactoryContract) GetGame(ctx context.Context, idx uint64, blockHash common.Hash) (types.GameMetadata, error) {
	defer f.metrics.StartContractRequest("GetGame")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.ByHash(blockHash), f.contract.Call(methodGameAtIndex, new(big.Int).SetUint64(idx)))
	if err != nil {
		return types.GameMetadata{}, fmt.Errorf("failed to load game %v: %w", idx, err)
	}
	return f.decodeGame(idx, result), nil
}

func (f *DisputeGameFactoryContract) GetGameImpl(ctx context.Context, gameType faultTypes.GameType) (common.Address, error) {
	defer f.metrics.StartContractRequest("GetGameImpl")()
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodGameImpls, gameType))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to load game impl for type %v: %w", gameType, err)
	}
	return result.GetAddress(0), nil
}

func (f *DisputeGameFactoryContract) GetGamesAtOrAfter(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]types.GameMetadata, error) {
	defer f.metrics.StartContractRequest("GetGamesAtOrAfter")()
	count, err := f.GetGameCount(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	batchSize := uint64(f.multiCaller.BatchSize())
	rangeEnd := count

	var games []types.GameMetadata
	for {
		if rangeEnd == uint64(0) {
			// rangeEnd is exclusive so if its 0 we've reached the end.
			return games, nil
		}
		rangeStart := uint64(0)
		if rangeEnd > batchSize {
			rangeStart = rangeEnd - batchSize
		}
		calls := make([]batching.Call, 0, rangeEnd-rangeStart)
		for i := rangeEnd - 1; ; i-- {
			calls = append(calls, f.contract.Call(methodGameAtIndex, new(big.Int).SetUint64(i)))
			// Break once we've added the last call to avoid underflow when rangeStart == 0
			if i == rangeStart {
				break
			}
		}

		results, err := f.multiCaller.Call(ctx, rpcblock.ByHash(blockHash), calls...)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch games: %w", err)
		}

		for i, result := range results {
			idx := rangeEnd - uint64(i) - 1
			game := f.decodeGame(idx, result)
			if game.Timestamp < earliestTimestamp {
				return games, nil
			}
			games = append(games, game)
		}
		rangeEnd = rangeStart
	}
}

func (f *DisputeGameFactoryContract) GetAllGames(ctx context.Context, blockHash common.Hash) ([]types.GameMetadata, error) {
	defer f.metrics.StartContractRequest("GetAllGames")()
	count, err := f.GetGameCount(ctx, blockHash)
	if err != nil {
		return nil, err
	}

	calls := make([]batching.Call, count)
	for i := uint64(0); i < count; i++ {
		calls[i] = f.contract.Call(methodGameAtIndex, new(big.Int).SetUint64(i))
	}

	results, err := f.multiCaller.Call(ctx, rpcblock.ByHash(blockHash), calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch games: %w", err)
	}

	var games []types.GameMetadata
	for i, result := range results {
		games = append(games, f.decodeGame(uint64(i), result))
	}
	return games, nil
}

func (f *DisputeGameFactoryContract) CreateTx(ctx context.Context, traceType uint32, outputRoot common.Hash, l2BlockNum uint64) (txmgr.TxCandidate, error) {
	result, err := f.multiCaller.SingleCall(ctx, rpcblock.Latest, f.contract.Call(methodInitBonds, traceType))
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to fetch init bond: %w", err)
	}
	initBond := result.GetBigInt(0)
	call := f.contract.Call(methodCreateGame, traceType, outputRoot, common.BigToHash(big.NewInt(int64(l2BlockNum))).Bytes())
	candidate, err := call.ToTxCandidate()
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	candidate.Value = initBond
	return candidate, err
}

func (f *DisputeGameFactoryContract) DecodeDisputeGameCreatedLog(rcpt *ethTypes.Receipt) (common.Address, uint32, common.Hash, error) {
	for _, log := range rcpt.Logs {
		if log.Address != f.contract.Addr() {
			// Not from this contract
			continue
		}
		name, result, err := f.contract.DecodeEvent(log)
		if err != nil {
			// Not a valid event
			continue
		}
		if name != eventDisputeGameCreated {
			// Not the event we're looking for
			continue
		}

		return result.GetAddress(0), result.GetUint32(1), result.GetHash(2), nil
	}
	return common.Address{}, 0, common.Hash{}, fmt.Errorf("%w: %v", ErrEventNotFound, eventDisputeGameCreated)
}

func (f *DisputeGameFactoryContract) decodeGame(idx uint64, result *batching.CallResult) types.GameMetadata {
	gameType := result.GetUint32(0)
	timestamp := result.GetUint64(1)
	proxy := result.GetAddress(2)
	return types.GameMetadata{
		Index:     idx,
		GameType:  gameType,
		Timestamp: timestamp,
		Proxy:     proxy,
	}
}
