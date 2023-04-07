package l1

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

type L1BlockHint common.Hash

var _ preimage.Hint = L1BlockHint{}

func (l L1BlockHint) Hint() string {
	return "l1-block " + (common.Hash)(l).String()
}
