package node

import (
	"fmt"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutil"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/core/types"
)

type PayloadID [8]byte

func (b PayloadID) String() string {
	return hexutility.Encode(b[:])
}

func (b PayloadID) MarshalText() ([]byte, error) {
	return hexutility.Bytes(b[:]).MarshalText()
}

func (b *PayloadID) UnmarshalText(input []byte) error {
	err := hexutility.UnmarshalFixedText("PayloadID", input, b[:])
	if err != nil {
		return fmt.Errorf("invalid payload id %q: %w", input, err)
	}
	return nil
}

type ExecutePayloadStatus string

type Block struct {
	ParentHash   common.Hash      `json:"parentHash"`
	UncleHash    common.Hash      `json:"sha3Uncles"`
	Coinbase     common.Address   `json:"miner"`
	Root         common.Hash      `json:"stateRoot"`
	TxHash       common.Hash      `json:"transactionsRoot"`
	ReceiptHash  common.Hash      `json:"receiptsRoot"`
	Bloom        hexutility.Bytes `json:"logsBloom"`
	Difficulty   hexutil.Big      `json:"difficulty"`
	Number       hexutil.Uint64   `json:"number"`
	GasLimit     hexutil.Uint64   `json:"gasLimit"`
	GasUsed      hexutil.Uint64   `json:"gasUsed"`
	Time         hexutil.Uint64   `json:"timestamp"`
	Extra        hexutility.Bytes `json:"extraData"`
	MixDigest    common.Hash      `json:"mixHash"`
	Nonce        types.BlockNonce `json:"nonce"`
	Hash         common.Hash      `json:"hash"`
	Transactions []*common.Hash   `json:"transactions"`
}

type ForkchoiceUpdatedResult struct {
	PayloadStatus PayloadStatusV1 `json:"payloadStatus"`
	PayloadID     *PayloadID      `json:"payloadId"`
}

type PayloadStatusV1 struct {
	Status          ExecutePayloadStatus `json:"status"`
	LatestValidHash *common.Hash         `json:"latestValidHash,omitempty"`
	ValidationError *string              `json:"validationError,omitempty"`
}

type TraceTransaction struct {
	From    common.Address      `json:"from"`
	Value   hexutil.Big         `json:"value"`
	Gas     hexutil.Uint64      `json:"gas"`
	GasUsed hexutil.Uint64      `json:"gasUsed"`
	Input   hexutility.Bytes    `json:"input"`
	Output  hexutility.Bytes    `json:"output"`
	To      common.Address      `json:"to,omitempty"`
	Calls   []*TraceTransaction `json:"calls,omitempty"`
}
