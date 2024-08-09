package script

import (
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

func bytes4(sig string) [4]byte {
	return [4]byte(crypto.Keccak256([]byte(sig))[:4])
}

func padU64(v uint64) []byte {
	var out [32]byte
	binary.BigEndian.PutUint64(out[24:], v)
	return out[:]
}

var (
	getNonce = bytes4("getNonce(address)")
)

type CheatCodesPrecompile struct {
	h *Host
}

var _ vm.PrecompiledContract = (*CheatCodesPrecompile)(nil)

func (c *CheatCodesPrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *CheatCodesPrecompile) Run(input []byte) ([]byte, error) {
	c.h.log.Info("cheatcode", "input", hexutil.Bytes(input))
	if len(input) < 4 {
		c.h.log.Error("Invalid cheatcode call", "input", hexutil.Bytes(input))
		return nil, fmt.Errorf("invalid cheatcode call: %x", input)
	}
	sig := [4]byte(input[:4])
	switch sig {
	case getNonce:
		return padU64(c.h.state.GetNonce(common.Address(input[4:]))), nil
	}
	return []byte{}, nil
}

// TODO: define backend in Go, by attaching methods to struct
// Then use reflection to turn every struct method into a bytes4 matcher with automated ABI decoding / encoding

// TODO construct abi Method definition for each func
// abi.Method{}
// abi.Arguments for args / return params
// compute the sig
// abi.ABI{}.Unpack()
