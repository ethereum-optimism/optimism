package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

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
		LagoonTime:              c.LagoonTime,
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
