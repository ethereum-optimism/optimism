package superchain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// OpChainConfig builds an OP-Stack chain config from the registry chain config.
func (c *ChainConfig) OpChainConfig() *opparams.ChainConfig {
	hardforks := c.Hardforks
	out := &opparams.ChainConfig{
		ChainID:      new(big.Int).SetUint64(c.ChainID),
		BedrockBlock: common.Big0,
		RegolithTime: ptr.New(uint64(0)),
		CanyonTime:   hardforks.CanyonTime,
		EcotoneTime:  hardforks.EcotoneTime,
		FjordTime:    hardforks.FjordTime,
		GraniteTime:  hardforks.GraniteTime,
		HoloceneTime: hardforks.HoloceneTime,
		IsthmusTime:  hardforks.IsthmusTime,
		JovianTime:   hardforks.JovianTime,
		KarstTime:    hardforks.KarstTime,
		LagoonTime:   hardforks.LagoonTime,
	}

	if c.Optimism != nil {
		out.Optimism = &opparams.OptimismConfig{
			EIP1559Elasticity:  c.Optimism.EIP1559Elasticity,
			EIP1559Denominator: c.Optimism.EIP1559Denominator,
		}
		if c.Optimism.EIP1559DenominatorCanyon != nil {
			out.Optimism.EIP1559DenominatorCanyon = ptr.New(*c.Optimism.EIP1559DenominatorCanyon)
		}
	}

	// OP Mainnet treats its Bedrock-migration block as genesis.
	if c.ChainID == opparams.OPMainnetChainID {
		out.BedrockBlock = big.NewInt(opparams.OPMainnetGenesisBlockNum)
	}

	return out
}
