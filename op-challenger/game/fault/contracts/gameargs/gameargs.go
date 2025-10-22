package gameargs

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

const (
	PermissionlessArgsLength = 124
	PermissionedArgsLength   = 164
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
