package contracts

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

var ErrUnsupportedGameType = errors.New("unsupported game type")

type GenericGameMetadata struct {
	L1Head        common.Hash
	L2SequenceNum uint64
	ProposedRoot  common.Hash
	Status        gameTypes.GameStatus
}

type DisputeGameContract interface {
	GetL1Head(ctx context.Context) (common.Hash, error)
	GetStatus(ctx context.Context) (gameTypes.GameStatus, error)
	GetGameRange(ctx context.Context) (prestateBlock uint64, poststateBlock uint64, retErr error)
	GetMetadata(ctx context.Context, block rpcblock.Block) (GenericGameMetadata, error)

	GetResolvedAt(ctx context.Context, block rpcblock.Block) (time.Time, error)
	CallResolve(ctx context.Context) (gameTypes.GameStatus, error)
	ResolveTx() (txmgr.TxCandidate, error)
}

func NewDisputeGameContractForGame(ctx context.Context, metrics metrics.ContractMetricer, caller *batching.MultiCaller, game gameTypes.GameMetadata) (DisputeGameContract, error) {
	return NewDisputeGameContract(ctx, metrics, caller, types.GameType(game.GameType), game.Proxy)
}

func NewDisputeGameContract(ctx context.Context, metrics metrics.ContractMetricer, caller *batching.MultiCaller, gameType types.GameType, addr common.Address) (DisputeGameContract, error) {
	switch gameType {
	case types.SuperCannonGameType, types.SuperCannonKonaGameType, types.SuperPermissionedGameType, types.SuperAsteriscKonaGameType:
		return NewSuperFaultDisputeGameContract(ctx, metrics, addr, caller)

	case types.CannonGameType,
		types.PermissionedGameType,
		types.CannonKonaGameType,
		types.AsteriscGameType,
		types.AlphabetGameType,
		types.FastGameType,
		types.AsteriscKonaGameType:
		return NewPreInteropFaultDisputeGameContract(ctx, metrics, addr, caller)
	case types.OptimisticZKGameType:
		return NewOptimisticZKDisputeGameContract(metrics, addr, caller)
	default:
		return nil, ErrUnsupportedGameType
	}
}
