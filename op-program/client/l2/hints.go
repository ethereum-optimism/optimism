package l2

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

type L2BlockHint common.Hash

var _ preimage.Hint = L2BlockHint{}

func (l L2BlockHint) Hint() string {
	return "l2-block " + (common.Hash)(l).String()
}

type L2StateNodeHint common.Hash

var _ preimage.Hint = L2StateNodeHint{}

func (l L2StateNodeHint) Hint() string {
	return "l2-state-node " + (common.Hash)(l).String()
}

type L2CodeHint common.Hash

var _ preimage.Hint = L2CodeHint{}

func (l L2CodeHint) Hint() string {
	return "l2-code " + (common.Hash)(l).String()
}
