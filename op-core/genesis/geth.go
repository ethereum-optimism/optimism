package genesis

import (
	"github.com/ethereum/go-ethereum/core"
)

// GethGenesis returns the equivalent go-ethereum core.Genesis, for computing the genesis
// block (ToBlock) or committing the genesis state with go-ethereum. Its Config is
// g.Config.GethChainConfig(). The result shares memory with g.
func (g *Genesis) GethGenesis() *core.Genesis {
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
