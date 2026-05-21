package gameargs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

const (
	PermissionlessArgsLength = 124
	PermissionedArgsLength   = 164
	ZKArgsLength             = 172
)

var (
	ErrInvalidGameArgs = errors.New("invalid game args")
)

type GameArgs struct {
	AbsolutePrestate    common.Hash
	Vm                  common.Address
	AnchorStateRegistry common.Address
	Weth                common.Address
	L2ChainID           eth.ChainID
	Proposer            common.Address
	Challenger          common.Address
}

func (g GameArgs) PackPermissionless() []byte {
	chainID := g.L2ChainID.Bytes32()
	return slices.Concat(
		g.AbsolutePrestate[:],
		g.Vm[:],
		g.AnchorStateRegistry[:],
		g.Weth[:],
		chainID[:],
	)
}

func (g GameArgs) PackPermissioned() []byte {
	return slices.Concat(
		g.PackPermissionless(),
		g.Proposer[:],
		g.Challenger[:],
	)
}

// ZKGameArgs holds the arguments for a ZK dispute game, matching the packed layout
// defined in LibGameArgs.sol (ZK_ARGS_LENGTH = 172 bytes).
type ZKGameArgs struct {
	AbsolutePrestate     common.Hash
	Verifier             common.Address
	MaxChallengeDuration uint64
	MaxProveDuration     uint64
	ChallengerBond       *big.Int
	AnchorStateRegistry  common.Address
	Weth                 common.Address
	L2ChainID            *big.Int
}

// Pack encodes the ZK game args using abi.encodePacked layout (172 bytes).
// Layout: absolutePrestate(32) + verifier(20) + maxChallengeDuration(8) +
// maxProveDuration(8) + challengerBond(32) + anchorStateRegistry(20) +
// weth(20) + l2ChainId(32)
func (z ZKGameArgs) Pack() []byte {
	dur1 := make([]byte, 8)
	binary.BigEndian.PutUint64(dur1, z.MaxChallengeDuration)
	dur2 := make([]byte, 8)
	binary.BigEndian.PutUint64(dur2, z.MaxProveDuration)
	bond := make([]byte, 32)
	z.ChallengerBond.FillBytes(bond)
	chainID := make([]byte, 32)
	z.L2ChainID.FillBytes(chainID)
	return slices.Concat(
		z.AbsolutePrestate[:],
		z.Verifier[:],
		dur1,
		dur2,
		bond,
		z.AnchorStateRegistry[:],
		z.Weth[:],
		chainID,
	)
}

func Parse(args []byte) (GameArgs, error) {
	if len(args) != PermissionlessArgsLength && len(args) != PermissionedArgsLength {
		return GameArgs{}, fmt.Errorf("%w: invalid length (%v)", ErrInvalidGameArgs, len(args))
	}
	var output GameArgs
	output.AbsolutePrestate = common.BytesToHash(args[0:32])
	output.Vm = common.BytesToAddress(args[32:52])
	output.AnchorStateRegistry = common.BytesToAddress(args[52:72])
	output.Weth = common.BytesToAddress(args[72:92])
	var chainID [32]byte
	copy(chainID[:], args[92:124])
	output.L2ChainID = eth.ChainIDFromBytes32(chainID)

	if len(args) == PermissionedArgsLength {
		output.Proposer = common.BytesToAddress(args[124:144])
		output.Challenger = common.BytesToAddress(args[144:164])
	}
	return output, nil
}
