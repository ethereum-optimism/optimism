package genesis

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/params/opgethtest"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestNewL2GenesisMatchesOpGeth checks NewL2Genesis against op-geth for a range of fork
// schedules: the genesis.json is op-geth's encoding of op-geth's chain config for the
// deploy config, with osakaTime following Karst, and the genesis block is op-geth's. It is
// a pre-cutover check and goes away with op-geth.
func TestNewL2GenesisMatchesOpGeth(t *testing.T) {
	schedules := map[string]forkstest.Schedule{
		"staggered":         forkstest.Staggered(),
		"all after genesis": forkstest.StaticSchedule(2),
	}
	for _, fork := range forkstest.TimestampELForks {
		schedules["through "+string(fork)] = forkstest.AtGenesisThrough(fork)
	}
	for name, schedule := range schedules {
		t.Run(name, func(t *testing.T) {
			config := testDeployConfig(t)
			for _, fork := range forkstest.TimestampELForks {
				config.SetForkTimeOffset(fork, schedule(fork))
			}

			g, err := NewL2Genesis(config, &eth.BlockRef{Time: 1_700_000_000})
			require.NoError(t, err)
			// Non-empty storage, so that the Isthmus withdrawals root is not the empty root.
			g.Alloc[predeploys.L2ToL1MessagePasserAddr] = types.Account{
				Storage: map[common.Hash]common.Hash{{0x01}: {0x02}},
				Balance: common.Big0,
			}
			want := g.GethGenesis()
			want.Config = opgethtest.ChainConfig(t, config.L2ChainID, func(fork forks.Name) *uint64 {
				return offsetToUpgradeTime((*hexutil.Uint64)(config.ForkTimeOffset(fork)), g.Timestamp)
			})
			want.Config.Optimism = &params.OptimismConfig{
				EIP1559Denominator:       config.EIP1559Denominator,
				EIP1559Elasticity:        config.EIP1559Elasticity,
				EIP1559DenominatorCanyon: &config.EIP1559DenominatorCanyon,
			}

			wantJSON, err := json.Marshal(want)
			require.NoError(t, err)
			gotJSON, err := json.Marshal(g)
			require.NoError(t, err)
			require.JSONEq(t, string(wantJSON), string(gotJSON))
			require.Equal(t, want.ToBlock().Header(), g.ToBlock().Header())
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

func testDeployConfig(t *testing.T) *DeployConfig {
	data, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)
	config := new(DeployConfig)
	require.NoError(t, json.Unmarshal(data, config))
	return config
}
