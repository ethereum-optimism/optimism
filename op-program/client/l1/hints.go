package l1

import (
	"encoding/binary"

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

// NewBlobHint constructs a 40 byte blob hint with timestamp.
func NewBlobHint(blobHash common.Hash, timeStamp uint64) BlobHint {
	metaData := make([]byte, 8)
	binary.BigEndian.PutUint64(metaData[:], timeStamp)
	return BlobHint(append(blobHash[:], metaData[:]...))
}

// NewLegacyBlobHint is deprecated, do not use. Constructs a 48 byte blob hint with timestamp and index.
func NewLegacyBlobHint(blobHash common.Hash, index uint64, timeStamp uint64) BlobHint {
	metaData := make([]byte, 16)
	binary.BigEndian.PutUint64(metaData[0:8], index)
	binary.BigEndian.PutUint64(metaData[8:16], timeStamp)
	return BlobHint(append(blobHash[:], metaData[:]...))
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
