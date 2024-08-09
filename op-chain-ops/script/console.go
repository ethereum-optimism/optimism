package script

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ConsolePrecompile struct {
	h *Host
}

var _ vm.PrecompiledContract = (*ConsolePrecompile)(nil)

func (c *ConsolePrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *ConsolePrecompile) Run(input []byte) ([]byte, error) {
	c.h.log.Info("console", "input", hexutil.Bytes(input))
	// TODO: log the message
	return []byte{}, nil
}
