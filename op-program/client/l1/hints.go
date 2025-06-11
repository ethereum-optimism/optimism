package l1

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

const (
	HintL1BlockHeader  = "l1-block-header"
	HintL1Transactions = "l1-transactions"
	HintL1Receipts     = "l1-receipts"
	HintL1Blob         = "l1-blob"
	HintL1Precompile   = "l1-precompile"
	HintL1PrecompileV2 = "l1-precompile-v2"
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

type BlobHint []byte

var _ preimage.Hint = BlobHint{}

func (l BlobHint) Hint() string {
	return HintL1Blob + " " + hexutil.Encode(l)
}

type PrecompileHint []byte

var _ preimage.Hint = PrecompileHint{}

func (l PrecompileHint) Hint() string {
	return HintL1Precompile + " " + hexutil.Encode(l)
}

type PrecompileHintV2 []byte

var _ preimage.Hint = PrecompileHintV2{}

func (l PrecompileHintV2) Hint() string {
	return HintL1PrecompileV2 + " " + hexutil.Encode(l)
}
