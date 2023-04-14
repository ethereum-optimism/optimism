package l2

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

type BlockHeaderHint common.Hash

var _ preimage.Hint = BlockHeaderHint{}

func (l BlockHeaderHint) Hint() string {
	return "l2-block-header " + (common.Hash)(l).String()
}

type TransactionsHint common.Hash

var _ preimage.Hint = TransactionsHint{}

func (l TransactionsHint) Hint() string {
	return "l2-transactions " + (common.Hash)(l).String()
}

type CodeHint common.Hash

var _ preimage.Hint = CodeHint{}

func (l CodeHint) Hint() string {
	return "l2-code " + (common.Hash)(l).String()
}

type StateNodeHint common.Hash

var _ preimage.Hint = StateNodeHint{}

func (l StateNodeHint) Hint() string {
	return "l2-state-node " + (common.Hash)(l).String()
}
