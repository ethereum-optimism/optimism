package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"
)

// GethChainConfig returns the go-ethereum params.ChainConfig with c's Ethereum fork
// schedule, for code that needs the concrete go-ethereum type. The schedule is derived
// from the OP fork schedule: Shanghai activates with Canyon, Cancun with Ecotone, Prague
// with Isthmus, Osaka with Karst. The result carries no OP Stack fields, so go-ethereum
// treats it as an Ethereum chain config.
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
