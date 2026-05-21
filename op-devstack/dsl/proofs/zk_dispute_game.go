package proofs

import (
	"encoding/binary"
	"math/big"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

// zkArgsLength is the byte length of packed ZK game args as produced by
// OPContractsManagerUtils._encodeGameArgs for ZK_DISPUTE_GAME.
//
// Layout (abi.encodePacked, 172 bytes):
//
//	[0-31]    absolutePrestate (bytes32)
//	[32-51]   verifier (address)
//	[52-59]   maxChallengeDuration (uint64)
//	[60-67]   maxProveDuration (uint64)
//	[68-99]   challengerBond (uint256)
//	[100-119] anchorStateRegistry (address)
//	[120-139] weth (address)
//	[140-171] l2ChainId (uint256)
const zkArgsLength = 172

// ZKGameArgs contains the parsed arguments for a ZK dispute game template.
type ZKGameArgs struct {
	AbsolutePrestate     common.Hash
	Verifier             common.Address
	MaxChallengeDuration uint64
	MaxProveDuration     uint64
	ChallengerBond       *big.Int
	AnchorStateRegistry  common.Address
	WETH                 common.Address
	L2ChainID            *big.Int
}

// ZKDisputeGame holds the impl address and parsed args for a deployed ZK dispute game.
type ZKDisputeGame struct {
	Address common.Address
	Args    ZKGameArgs
}

// ZKGameImpl returns the ZK dispute game implementation address and its parsed
// constructor args from the DisputeGameFactory.
func (f *DisputeGameFactory) ZKGameImpl() *ZKDisputeGame {
	impl := f.GameImpl(gameTypes.ZKDisputeGameType)
	raw := f.GameArgs(gameTypes.ZKDisputeGameType)
	f.require.Len(raw, zkArgsLength, "ZK game args must be exactly %d bytes", zkArgsLength)

	var prestate common.Hash
	copy(prestate[:], raw[0:32])

	var verifier common.Address
	copy(verifier[:], raw[32:52])

	var asr common.Address
	copy(asr[:], raw[100:120])

	var weth common.Address
	copy(weth[:], raw[120:140])

	return &ZKDisputeGame{
		Address: impl.Address,
		Args: ZKGameArgs{
			AbsolutePrestate:     prestate,
			Verifier:             verifier,
			MaxChallengeDuration: binary.BigEndian.Uint64(raw[52:60]),
			MaxProveDuration:     binary.BigEndian.Uint64(raw[60:68]),
			ChallengerBond:       new(big.Int).SetBytes(raw[68:100]),
			AnchorStateRegistry:  asr,
			WETH:                 weth,
			L2ChainID:            new(big.Int).SetBytes(raw[140:172]),
		},
	}
}
