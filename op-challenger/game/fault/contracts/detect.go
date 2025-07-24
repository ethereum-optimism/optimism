package contracts

import (
	"context"
	"fmt"

	faultTypes "github.com/HashKeyChain/verse/op-challenger/game/fault/types"
	"github.com/HashKeyChain/verse/op-service/sources/batching"
	"github.com/HashKeyChain/verse/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
)

var (
	methodGameType = "gameType"

	gameTypeABI = mustParseAbi([]byte(`[{
		"inputs": [],
		"name": "gameType",
		"outputs": [{"type": "uint32"}],
		"stateMutability": "view",
		"type": "function"
	}]`))
)

func DetectGameType(ctx context.Context, addr common.Address, caller *batching.MultiCaller) (faultTypes.GameType, error) {
	result, err := caller.SingleCall(ctx, rpcblock.Latest, batching.NewContractCall(gameTypeABI, addr, methodGameType))
	if err != nil {
		return faultTypes.UnknownGameType, fmt.Errorf("failed to detect game type: %w", err)
	}
	gameType := faultTypes.GameType(result.GetUint32(0))
	switch gameType {
	case faultTypes.CannonGameType,
		faultTypes.PermissionedGameType,
		faultTypes.AsteriscGameType,
		faultTypes.AlphabetGameType,
		faultTypes.FastGameType,
		faultTypes.AsteriscKonaGameType,
		faultTypes.SuperCannonGameType,
		faultTypes.SuperPermissionedGameType,
		faultTypes.SuperAsteriscKonaGameType:
		return gameType, nil
	default:
		return faultTypes.UnknownGameType, fmt.Errorf("unsupported game type: %d", gameType)
	}
}
