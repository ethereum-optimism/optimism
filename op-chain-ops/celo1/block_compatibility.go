package celo1

import (
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// This file takes care of supporting older block header formats from before
// the gingerbread fork.

type beforeGingerbreadHeader struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	Coinbase    common.Address `json:"miner"            gencodec:"required"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       types.Bloom    `json:"logsBloom"        gencodec:"required"`
	Number      *big.Int       `json:"number"           gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        uint64         `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`

	// Used to cache deserialized istanbul extra data
	extraLock  sync.Mutex
	extraError error
}

type afterGingerbreadHeader Header

func (h *Header) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	var raw rlp.RawValue
	err := s.Decode(&raw)
	if err != nil {
		return err
	}
	headerSize := len(raw) - int(size)
	numElems, err := rlp.CountValues(raw[headerSize:])
	if err != nil {
		return err
	}
	if numElems == 10 {
		// Before gingerbread
		decodedHeader := beforeGingerbreadHeader{}
		err = rlp.DecodeBytes(raw, &decodedHeader)

		h.ParentHash = decodedHeader.ParentHash
		h.Coinbase = decodedHeader.Coinbase
		h.Root = decodedHeader.Root
		h.TxHash = decodedHeader.TxHash
		h.ReceiptHash = decodedHeader.ReceiptHash
		h.Bloom = decodedHeader.Bloom
		h.Number = decodedHeader.Number
		h.GasUsed = decodedHeader.GasUsed
		h.Time = decodedHeader.Time
		h.Extra = decodedHeader.Extra
	} else {
		// After gingerbread
		decodedHeader := afterGingerbreadHeader{}
		err = rlp.DecodeBytes(raw, &decodedHeader)

		h.ParentHash = decodedHeader.ParentHash
		h.UncleHash = decodedHeader.UncleHash
		h.Coinbase = decodedHeader.Coinbase
		h.Root = decodedHeader.Root
		h.TxHash = decodedHeader.TxHash
		h.ReceiptHash = decodedHeader.ReceiptHash
		h.Bloom = decodedHeader.Bloom
		h.Difficulty = decodedHeader.Difficulty
		h.Number = decodedHeader.Number
		h.GasLimit = decodedHeader.GasLimit
		h.GasUsed = decodedHeader.GasUsed
		h.Time = decodedHeader.Time
		h.Extra = decodedHeader.Extra
		h.MixDigest = decodedHeader.MixDigest
		h.Nonce = decodedHeader.Nonce
		h.BaseFee = decodedHeader.BaseFee
	}

	return err
}

func (h *Header) EncodeRLP(w io.Writer) error {
	if (h.UncleHash == common.Hash{}) {
		// Before gingerbread hardfork Celo did not include all of
		// Ethereum's header fields. In that case we must omit the new
		// fields from the header when encoding as RLP to maintain the same encoding and hashes.
		// `UncleHash` is a safe way to check, since it is the zero hash before
		// gingerbread and non-zero after.
		rlpFields := []interface{}{
			h.ParentHash,
			h.Coinbase,
			h.Root,
			h.TxHash,
			h.ReceiptHash,
			h.Bloom,
			h.Number,
			h.GasUsed,
			h.Time,
			h.Extra,
		}
		return rlp.Encode(w, rlpFields)
	} else {
		rlpFields := []interface{}{
			h.ParentHash,
			h.UncleHash,
			h.Coinbase,
			h.Root,
			h.TxHash,
			h.ReceiptHash,
			h.Bloom,
			h.Difficulty,
			h.Number,
			h.GasLimit,
			h.GasUsed,
			h.Time,
			h.Extra,
			h.MixDigest,
			h.Nonce,
			h.BaseFee,
		}
		return rlp.Encode(w, rlpFields)
	}
}
