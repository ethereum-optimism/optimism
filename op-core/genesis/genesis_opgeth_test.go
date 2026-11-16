package genesis

import (
	"maps"
	"testing"

	"github.com/ethereum/go-ethereum/core"

	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	"github.com/ethereum-optimism/optimism/op-core/params/opgethtest"
)

// This file holds the op-geth side of the differential tests. It only compiles against
// op-geth and goes away with it.

// opGethGenesis returns g as the op-geth core.Genesis that op-chain-ops builds for it, with
// op-geth's chain config for the fork schedule s.
func opGethGenesis(t *testing.T, g *Genesis, s forkstest.Schedule) *core.Genesis {
	t.Helper()
	return &core.Genesis{
		Config:        opgethtest.ChainConfig(t, 901, s),
		Nonce:         g.Nonce,
		Timestamp:     g.Timestamp,
		ExtraData:     g.ExtraData,
		GasLimit:      g.GasLimit,
		Difficulty:    g.Difficulty,
		Mixhash:       g.Mixhash,
		Coinbase:      g.Coinbase,
		Alloc:         maps.Clone(g.Alloc),
		Number:        g.Number,
		GasUsed:       g.GasUsed,
		ParentHash:    g.ParentHash,
		BaseFee:       g.BaseFee,
		ExcessBlobGas: g.ExcessBlobGas,
		BlobGasUsed:   g.BlobGasUsed,
		SlotNumber:    g.SlotNumber,
	}
}
