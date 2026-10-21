// Package opgeth converts op-core types to and from op-geth's versions of the go-ethereum
// types, which carry OP Stack fields that upstream go-ethereum lacks. It is for code that
// runs op-geth in-process or reads those fields off op-geth types, and builds only while
// go-ethereum resolves to op-geth. The op-core packages produce and accept upstream
// go-ethereum types only.
package opgeth

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
)

// ChainConfig returns c as an op-geth chain config: c.GethChainConfig() with the OP
// Stack fork schedule and EIP-1559 parameters set. Without them op-geth treats the chain
// as an Ethereum chain: IsOptimism reports false and no OP Stack rule applies. The result
// shares memory with c.
func ChainConfig(c *opparams.ChainConfig) *params.ChainConfig {
	out := c.GethChainConfig()
	out.BedrockBlock = c.BedrockBlock
	out.RegolithTime = c.RegolithTime
	out.CanyonTime = c.CanyonTime
	out.EcotoneTime = c.EcotoneTime
	out.FjordTime = c.FjordTime
	out.GraniteTime = c.GraniteTime
	out.HoloceneTime = c.HoloceneTime
	out.IsthmusTime = c.IsthmusTime
	out.JovianTime = c.JovianTime
	out.KarstTime = c.KarstTime
	out.LagoonTime = c.LagoonTime
	if c.Optimism != nil {
		out.Optimism = &params.OptimismConfig{
			EIP1559Elasticity:        c.Optimism.EIP1559Elasticity,
			EIP1559Denominator:       c.Optimism.EIP1559Denominator,
			EIP1559DenominatorCanyon: c.Optimism.EIP1559DenominatorCanyon,
		}
	}
	return out
}

// Genesis returns g as an op-geth genesis, for starting op-geth on it: g.GethGenesis()
// with its Config from ChainConfig. The result shares memory with g.
func Genesis(g *opgenesis.Genesis) *core.Genesis {
	out := g.GethGenesis()
	if g.Config != nil {
		out.Config = ChainConfig(g.Config)
	}
	return out
}
