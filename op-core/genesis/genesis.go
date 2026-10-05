// Package genesis holds the OP Stack L2 genesis specification and its JSON encoding.
//
// The JSON layout is that of a go-ethereum genesis file (core.Genesis): header fields and
// the alloc in go-ethereum's encodings, plus a "config" object. Upstream go-ethereum's
// chain config has no OP Stack fields, so "config" is the JSON encoding of the op-core
// params.ChainConfig instead, which carries the go-ethereum fork schedule it derives.
package genesis

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/params"
)

// Genesis specifies the genesis block of an OP Stack L2 chain: its chain configuration,
// header fields and initial state.
//
// The Ethereum fork schedule in the encoded "config" is not stored: it is derived from
// Config (see params.ChainConfig.MarshalJSON and UnmarshalJSON).
type Genesis struct {
	Config     *params.ChainConfig
	Nonce      uint64
	Timestamp  uint64
	ExtraData  []byte
	GasLimit   uint64
	Difficulty *big.Int
	Mixhash    common.Hash
	Coinbase   common.Address
	Alloc      types.GenesisAlloc

	Number        uint64
	GasUsed       uint64
	ParentHash    common.Hash
	BaseFee       *big.Int
	ExcessBlobGas *uint64
	BlobGasUsed   *uint64
	SlotNumber    *uint64
}

// genesisJSON is the wire form of Genesis. Field names, order and encodings follow
// go-ethereum's core.Genesis. Pointers tell absent fields apart when decoding, so that
// the required ones can be enforced.
type genesisJSON struct {
	Config        *params.ChainConfig                        `json:"config"`
	Nonce         *math.HexOrDecimal64                       `json:"nonce"`
	Timestamp     *math.HexOrDecimal64                       `json:"timestamp"`
	ExtraData     *hexutil.Bytes                             `json:"extraData"`
	GasLimit      *math.HexOrDecimal64                       `json:"gasLimit"`
	Difficulty    *math.HexOrDecimal256                      `json:"difficulty"`
	Mixhash       *common.Hash                               `json:"mixHash"`
	Coinbase      *common.Address                            `json:"coinbase"`
	Alloc         map[common.UnprefixedAddress]types.Account `json:"alloc"`
	Number        *math.HexOrDecimal64                       `json:"number"`
	GasUsed       *math.HexOrDecimal64                       `json:"gasUsed"`
	ParentHash    *common.Hash                               `json:"parentHash"`
	BaseFee       *math.HexOrDecimal256                      `json:"baseFeePerGas"`
	ExcessBlobGas *math.HexOrDecimal64                       `json:"excessBlobGas"`
	BlobGasUsed   *math.HexOrDecimal64                       `json:"blobGasUsed"`
	SlotNumber    *uint64                                    `json:"slotNumber"`
	// StateHash is an op-geth genesis key that substitutes a state root for the alloc.
	// Genesis has no such field, so a genesis that sets it is rejected rather than read
	// as one with an empty state.
	StateHash *common.Hash `json:"stateHash,omitempty"`
}

// MarshalJSON encodes g as a go-ethereum genesis file whose "config" carries the OP
// Stack chain configuration.
func (g Genesis) MarshalJSON() ([]byte, error) {
	extraData := hexutil.Bytes(g.ExtraData)
	enc := genesisJSON{
		Config:        g.Config,
		Nonce:         (*math.HexOrDecimal64)(&g.Nonce),
		Timestamp:     (*math.HexOrDecimal64)(&g.Timestamp),
		ExtraData:     &extraData,
		GasLimit:      (*math.HexOrDecimal64)(&g.GasLimit),
		Difficulty:    (*math.HexOrDecimal256)(g.Difficulty),
		Mixhash:       &g.Mixhash,
		Coinbase:      &g.Coinbase,
		Number:        (*math.HexOrDecimal64)(&g.Number),
		GasUsed:       (*math.HexOrDecimal64)(&g.GasUsed),
		ParentHash:    &g.ParentHash,
		BaseFee:       (*math.HexOrDecimal256)(g.BaseFee),
		ExcessBlobGas: (*math.HexOrDecimal64)(g.ExcessBlobGas),
		BlobGasUsed:   (*math.HexOrDecimal64)(g.BlobGasUsed),
		SlotNumber:    g.SlotNumber,
	}
	if g.Alloc != nil {
		enc.Alloc = make(map[common.UnprefixedAddress]types.Account, len(g.Alloc))
		for addr, account := range g.Alloc {
			enc.Alloc[common.UnprefixedAddress(addr)] = account
		}
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON decodes a go-ethereum genesis file whose "config" carries the OP Stack
// chain configuration.
func (g *Genesis) UnmarshalJSON(input []byte) error {
	var dec genesisJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.StateHash != nil {
		return errors.New("genesis with a stateHash is not supported")
	}
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Genesis")
	}
	if dec.Difficulty == nil {
		return errors.New("missing required field 'difficulty' for Genesis")
	}
	if dec.Alloc == nil {
		return errors.New("missing required field 'alloc' for Genesis")
	}

	out := Genesis{Config: dec.Config}
	if dec.Nonce != nil {
		out.Nonce = uint64(*dec.Nonce)
	}
	if dec.Timestamp != nil {
		out.Timestamp = uint64(*dec.Timestamp)
	}
	if dec.ExtraData != nil {
		out.ExtraData = *dec.ExtraData
	}
	out.GasLimit = uint64(*dec.GasLimit)
	out.Difficulty = (*big.Int)(dec.Difficulty)
	if dec.Mixhash != nil {
		out.Mixhash = *dec.Mixhash
	}
	if dec.Coinbase != nil {
		out.Coinbase = *dec.Coinbase
	}
	out.Alloc = make(types.GenesisAlloc, len(dec.Alloc))
	for addr, account := range dec.Alloc {
		out.Alloc[common.Address(addr)] = account
	}
	if dec.Number != nil {
		out.Number = uint64(*dec.Number)
	}
	if dec.GasUsed != nil {
		out.GasUsed = uint64(*dec.GasUsed)
	}
	if dec.ParentHash != nil {
		out.ParentHash = *dec.ParentHash
	}
	out.BaseFee = (*big.Int)(dec.BaseFee)
	out.ExcessBlobGas = (*uint64)(dec.ExcessBlobGas)
	out.BlobGasUsed = (*uint64)(dec.BlobGasUsed)
	out.SlotNumber = dec.SlotNumber
	*g = out
	return nil
}
