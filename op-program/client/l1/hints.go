package l1

import (
	"github.com/ethereum/go-ethereum/common"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

const (
	HintL1BlockHeader  = "l1-block-header"
	HintL1Transactions = "l1-transactions"
	HintL1Receipts     = "l1-receipts"
	HintL2Output       = "l1-l2-output"
)

type BlockHeaderHint common.Hash

var _ preimage.Hint = BlockHeaderHint{}

func (l BlockHeaderHint) Hint() string {
	return HintL1BlockHeader + " " + (common.Hash)(l).String()
}

type TransactionsHint common.Hash

var _ preimage.Hint = TransactionsHint{}

func (l TransactionsHint) Hint() string {
	return HintL1Transactions + " " + (common.Hash)(l).String()
}

type ReceiptsHint common.Hash

var _ preimage.Hint = ReceiptsHint{}

func (l ReceiptsHint) Hint() string {
	return HintL1Receipts + " " + (common.Hash)(l).String()
}

type L2OutputHint common.Hash

var _ preimage.Hint = L2OutputHint{}

func (l L2OutputHint) Hint() string {
	return HintL2Output + " " + (common.Hash)(l).String()
}
