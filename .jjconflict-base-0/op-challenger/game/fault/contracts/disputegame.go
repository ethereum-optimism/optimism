package contracts

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
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
	Addr() common.Address
	GetL1Head(ctx context.Context) (common.Hash, error)
	GetStatus(ctx context.Context) (gameTypes.GameStatus, error)
	GetGameRange(ctx context.Context) (prestateBlock uint64, poststateBlock uint64, retErr error)
	GetMetadata(ctx context.Context, block rpcblock.Block) (GenericGameMetadata, error)

	GetResolvedAt(ctx context.Context, block rpcblock.Block) (time.Time, error)
	CallResolve(ctx context.Context) (gameTypes.GameStatus, error)
	ResolveTx() (txmgr.TxCandidate, error)
}

func NewDisputeGameContractForGame(ctx context.Context, metrics metrics.ContractMetricer, caller *batching.MultiCaller, game gameTypes.GameMetadata) (DisputeGameContract, error) {
	return NewDisputeGameContract(ctx, metrics, caller, gameTypes.GameType(game.GameType), game.Proxy)
}

func NewDisputeGameContract(ctx context.Context, metrics metrics.ContractMetricer, caller *batching.MultiCaller, gameType gameTypes.GameType, addr common.Address) (DisputeGameContract, error) {
	switch gameType {
	case gameTypes.SuperCannonGameType, gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType:
		return NewSuperFaultDisputeGameContract(ctx, metrics, addr, caller)

	case gameTypes.CannonGameType,
		gameTypes.PermissionedGameType,
		gameTypes.CannonKonaGameType,
		gameTypes.AlphabetGameType,
		gameTypes.FastGameType:
		return NewPreInteropFaultDisputeGameContract(ctx, metrics, addr, caller)
	case gameTypes.OptimisticZKGameType:
		return NewOptimisticZKDisputeGameContract(metrics, addr, caller)
	default:
		return nil, ErrUnsupportedGameType
	}
}
