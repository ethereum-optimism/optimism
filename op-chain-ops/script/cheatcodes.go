package script

import (
	"github.com/ethereum/go-ethereum/common"
)

type CheatCodesPrecompile struct {
	h *Host
}

func (c *CheatCodesPrecompile) GetNonce(addr common.Address) uint64 {
	return c.h.state.GetNonce(addr)
}
