package params

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// FromSuperchainConfig builds an OP-Stack ChainConfig from a superchain registry
// chain config. It replaces op-geth's params.LoadOPStackChainConfig.
func FromSuperchainConfig(chConfig *superchain.ChainConfig) *ChainConfig {
	hardforks := chConfig.Hardforks
	out := &ChainConfig{
		ChainID:      new(big.Int).SetUint64(chConfig.ChainID),
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
		InteropTime:  hardforks.InteropTime,
	}

	if chConfig.Optimism != nil {
		out.Optimism = &OptimismConfig{
			EIP1559Elasticity:  chConfig.Optimism.EIP1559Elasticity,
			EIP1559Denominator: chConfig.Optimism.EIP1559Denominator,
		}
		if chConfig.Optimism.EIP1559DenominatorCanyon != nil {
			out.Optimism.EIP1559DenominatorCanyon = ptr.New(*chConfig.Optimism.EIP1559DenominatorCanyon)
		}
	}

	// OP Mainnet treats its Bedrock-migration block as genesis.
	if chConfig.ChainID == OPMainnetChainID {
		out.BedrockBlock = big.NewInt(OPMainnetGenesisBlockNum)
	}

	return out
}

// GethChainConfig returns the equivalent go-ethereum params.ChainConfig. It exists
// for code that drives a go-ethereum EVM, genesis, or block processor and therefore
// needs the concrete go-ethereum type. The Ethereum fork schedule is derived from the
// OP fork schedule: Shanghai activates with Canyon, Cancun with Ecotone, Prague with
// Isthmus, Osaka with Karst.
func (c *ChainConfig) GethChainConfig() *gethparams.ChainConfig {
	out := &gethparams.ChainConfig{
		ChainID:                 c.ChainID,
		HomesteadBlock:          common.Big0,
		DAOForkBlock:            nil,
		DAOForkSupport:          false,
		EIP150Block:             common.Big0,
		EIP155Block:             common.Big0,
		EIP158Block:             common.Big0,
		ByzantiumBlock:          common.Big0,
		ConstantinopleBlock:     common.Big0,
		PetersburgBlock:         common.Big0,
		IstanbulBlock:           common.Big0,
		MuirGlacierBlock:        common.Big0,
		BerlinBlock:             common.Big0,
		LondonBlock:             common.Big0,
		ArrowGlacierBlock:       common.Big0,
		GrayGlacierBlock:        common.Big0,
		MergeNetsplitBlock:      common.Big0,
		ShanghaiTime:            c.CanyonTime,
		CancunTime:              c.EcotoneTime,
		PragueTime:              c.IsthmusTime,
		OsakaTime:               c.KarstTime,
		TerminalTotalDifficulty: common.Big0,
		BedrockBlock:            c.BedrockBlock,
		RegolithTime:            c.RegolithTime,
		CanyonTime:              c.CanyonTime,
		EcotoneTime:             c.EcotoneTime,
		FjordTime:               c.FjordTime,
		GraniteTime:             c.GraniteTime,
		HoloceneTime:            c.HoloceneTime,
		IsthmusTime:             c.IsthmusTime,
		JovianTime:              c.JovianTime,
		KarstTime:               c.KarstTime,
		InteropTime:             c.InteropTime,
	}

	if c.Optimism != nil {
		out.Optimism = &gethparams.OptimismConfig{
			EIP1559Elasticity:  c.Optimism.EIP1559Elasticity,
			EIP1559Denominator: c.Optimism.EIP1559Denominator,
		}
		if c.Optimism.EIP1559DenominatorCanyon != nil {
			out.Optimism.EIP1559DenominatorCanyon = ptr.New(*c.Optimism.EIP1559DenominatorCanyon)
		}
	}

	// special overrides for OP-Stack chains with pre-Regolith upgrade history
	if c.ChainID != nil && c.ChainID.Cmp(big.NewInt(OPMainnetChainID)) == 0 {
		out.BerlinBlock = big.NewInt(3950000)
		out.LondonBlock = big.NewInt(OPMainnetGenesisBlockNum)
		out.ArrowGlacierBlock = big.NewInt(OPMainnetGenesisBlockNum)
		out.GrayGlacierBlock = big.NewInt(OPMainnetGenesisBlockNum)
		out.MergeNetsplitBlock = big.NewInt(OPMainnetGenesisBlockNum)
	}

	return out
}

// LoadChainConfigFromChainID loads the OP-Stack ChainConfig for the given L2 chain
// ID from the embedded superchain registry.
//
// This by-chain-ID convenience lives here, rather than in op-core/superchain,
// so that op-core/superchain stays a pure registry leaf: op-core/params imports
// op-core/superchain (one-way), which keeps the two packages free of an import cycle.
func LoadChainConfigFromChainID(chainID uint64) (*ChainConfig, error) {
	chain, err := superchain.GetChain(chainID)
	if err != nil {
		return nil, fmt.Errorf("unable to get chain %d from superchain registry: %w", chainID, err)
	}
	chConfig, err := chain.Config()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve chain %d config: %w", chainID, err)
	}
	return FromSuperchainConfig(chConfig), nil
}
