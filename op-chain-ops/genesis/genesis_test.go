package genesis

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestNewL2GenesisMatchesOpGeth checks NewL2Genesis against the op-geth chain config it
// built before the move to op-core/genesis, for a staggered schedule and every fork as the
// latest at genesis. The genesis.json is op-geth's encoding, with osakaTime now following
// Karst; the genesis block is op-geth's; and the genesis.json op-chain-ops wrote before the
// move, without osakaTime, decodes to the same block. It is a pre-cutover check and goes
// away with op-geth.
func TestNewL2GenesisMatchesOpGeth(t *testing.T) {
	schedules := map[string]func(*DeployConfig){
		"staggered": func(d *DeployConfig) {
			d.ActivateForkAtGenesis(forks.Delta)
			for i, fork := range forks.From(forks.Ecotone) {
				offset := uint64(2 * (i + 1))
				d.SetForkTimeOffset(fork, &offset)
			}
		},
	}
	for _, fork := range forks.From(forks.Regolith) {
		schedules["through "+string(fork)] = func(d *DeployConfig) { d.ActivateForkAtGenesis(fork) }
	}
	for name, schedule := range schedules {
		t.Run(name, func(t *testing.T) {
			config := testDeployConfig(t)
			schedule(config)

			g, err := NewL2Genesis(config, &eth.BlockRef{Time: 1_700_000_000})
			require.NoError(t, err)
			// Non-empty storage, so that the Isthmus withdrawals root is not the empty root.
			g.Alloc[predeploys.L2ToL1MessagePasserAddr] = types.Account{
				Storage: map[common.Hash]common.Hash{{0x01}: {0x02}},
				Balance: common.Big0,
			}
			want := g.GethGenesis()
			want.Config = opGethL2ChainConfig(config, g.Timestamp)

			wantJSON, err := json.Marshal(want)
			require.NoError(t, err)
			gotJSON, err := json.Marshal(g)
			require.NoError(t, err)
			require.JSONEq(t, string(wantJSON), string(gotJSON))
			got := g.ToBlock()
			require.Equal(t, want.ToBlock().Header(), got.Header())

			legacyConfig := *want.Config
			legacyConfig.OsakaTime = nil
			legacy := *want
			legacy.Config = &legacyConfig
			legacyJSON, err := json.Marshal(&legacy)
			require.NoError(t, err)
			var decoded opgenesis.Genesis
			require.NoError(t, json.Unmarshal(legacyJSON, &decoded))
			require.Equal(t, got.Hash(), decoded.ToBlock().Hash())
		})
	}
}

func TestNewL2GenesisRefusesOPMainnetChainID(t *testing.T) {
	config := testDeployConfig(t)
	config.L2ChainID = opparams.OPMainnetChainID
	_, err := NewL2Genesis(config, &eth.BlockRef{Time: 1_700_000_000})
	require.ErrorIs(t, err, ErrInvalidDeployConfig)
	require.ErrorContains(t, err, "reserved for OP Mainnet")
}

// opGethL2ChainConfig is the op-geth chain config NewL2Genesis built before it moved to
// op-core/genesis, with Osaka added at Karst. The test deploy config sets every EIP-1559
// parameter, so NewL2Genesis's defaults for unset ones don't apply.
func opGethL2ChainConfig(config *DeployConfig, genesisTime uint64) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:                 new(big.Int).SetUint64(config.L2ChainID),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		BedrockBlock:            new(big.Int).SetUint64(uint64(config.L2GenesisBlockNumber)),
		RegolithTime:            config.RegolithTime(genesisTime),
		CanyonTime:              config.CanyonTime(genesisTime),
		ShanghaiTime:            config.CanyonTime(genesisTime),
		CancunTime:              config.EcotoneTime(genesisTime),
		EcotoneTime:             config.EcotoneTime(genesisTime),
		FjordTime:               config.FjordTime(genesisTime),
		GraniteTime:             config.GraniteTime(genesisTime),
		HoloceneTime:            config.HoloceneTime(genesisTime),
		IsthmusTime:             config.IsthmusTime(genesisTime),
		JovianTime:              config.JovianTime(genesisTime),
		KarstTime:               config.KarstTime(genesisTime),
		PragueTime:              config.IsthmusTime(genesisTime),
		OsakaTime:               config.KarstTime(genesisTime),
		LagoonTime:              config.LagoonTime(genesisTime),
		Optimism: &params.OptimismConfig{
			EIP1559Denominator:       config.EIP1559Denominator,
			EIP1559Elasticity:        config.EIP1559Elasticity,
			EIP1559DenominatorCanyon: &config.EIP1559DenominatorCanyon,
		},
	}
}

func testDeployConfig(t *testing.T) *DeployConfig {
	data, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)
	config := new(DeployConfig)
	require.NoError(t, json.Unmarshal(data, config))
	return config
}
