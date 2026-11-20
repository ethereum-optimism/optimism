package genesis

import (
	"github.com/ethereum/go-ethereum/core"
)

// gethGenesis returns the equivalent go-ethereum core.Genesis. Its Config is
// g.Config.GethChainConfig(), which has no OP Stack rules: the result is fit for the
// genesis state root and header, not for running a node. The result shares memory
// with g.
//
// Compute the genesis block with ToBlock instead: go-ethereum's core.Genesis.ToBlock
// doesn't apply the Isthmus withdrawals root.
func (g *Genesis) gethGenesis() *core.Genesis {
	out := &core.Genesis{
		Nonce:         g.Nonce,
		Timestamp:     g.Timestamp,
		ExtraData:     g.ExtraData,
		GasLimit:      g.GasLimit,
		Difficulty:    g.Difficulty,
		Mixhash:       g.Mixhash,
		Coinbase:      g.Coinbase,
		Alloc:         g.Alloc,
		Number:        g.Number,
		GasUsed:       g.GasUsed,
		ParentHash:    g.ParentHash,
		BaseFee:       g.BaseFee,
		ExcessBlobGas: g.ExcessBlobGas,
		BlobGasUsed:   g.BlobGasUsed,
		SlotNumber:    g.SlotNumber,
	}
	if g.Config != nil {
		out.Config = g.Config.GethChainConfig()
	}
	return out
}
