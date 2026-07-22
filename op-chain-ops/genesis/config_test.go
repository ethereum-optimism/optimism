package genesis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestConfigDataMarshalUnmarshal(t *testing.T) {
	b, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)

	dec := json.NewDecoder(bytes.NewReader(b))
	decoded := new(DeployConfig)
	require.NoError(t, dec.Decode(decoded))
	require.NoError(t, decoded.Check(testlog.Logger(t, log.LevelDebug)))

	encoded, err := json.MarshalIndent(decoded, "", "  ")
	require.NoError(t, err)
	require.JSONEq(t, string(b), string(encoded))
}

func TestUnmarshalL1StartingBlockTag(t *testing.T) {
	decoded := new(DeployConfig)
	require.NoError(t, json.Unmarshal([]byte(`{"l1StartingBlockTag": "earliest"}`), decoded))
	require.EqualValues(t, rpc.EarliestBlockNumber, *decoded.L1StartingBlockTag.BlockNumber)
	h := "0x86c7263d87140ca7cd9bf1bc9e95a435a7a0efc0ae2afaf64920c5b59a6393d4"
	require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(`{"l1StartingBlockTag": "%s"}`, h)), decoded))
	require.EqualValues(t, common.HexToHash(h), *decoded.L1StartingBlockTag.BlockHash)
}

func TestRegolithTimeZero(t *testing.T) {
	regolithOffset := hexutil.Uint64(0)
	config := &DeployConfig{
		L2InitializationConfig: L2InitializationConfig{
			UpgradeScheduleDeployConfig: UpgradeScheduleDeployConfig{
				L2GenesisRegolithTimeOffset: &regolithOffset,
			},
		},
	}
	require.Equal(t, uint64(0), *config.RegolithTime(1234))
}

func TestRegolithTimeAsOffset(t *testing.T) {
	regolithOffset := hexutil.Uint64(1500)
	config := &DeployConfig{
		L2InitializationConfig: L2InitializationConfig{
			UpgradeScheduleDeployConfig: UpgradeScheduleDeployConfig{
				L2GenesisRegolithTimeOffset: &regolithOffset,
			},
		},
	}
	require.Equal(t, uint64(1500+5000), *config.RegolithTime(5000))
}

func TestL2GenesisTime(t *testing.T) {
	t.Run("defaults to the L1 start time when unset", func(t *testing.T) {
		config := &DeployConfig{}
		got, err := config.L2GenesisTime(5000)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), got)
	})

	t.Run("returns the explicit timestamp when set", func(t *testing.T) {
		override := hexutil.Uint64(9000)
		config := &DeployConfig{L2InitializationConfig: L2InitializationConfig{
			L2GenesisBlockDeployConfig: L2GenesisBlockDeployConfig{L2GenesisBlockTimestamp: &override},
		}}
		got, err := config.L2GenesisTime(5000)
		require.NoError(t, err)
		require.Equal(t, uint64(9000), got)
	})

	t.Run("equal to the L1 start time is accepted", func(t *testing.T) {
		override := hexutil.Uint64(5000)
		config := &DeployConfig{L2InitializationConfig: L2InitializationConfig{
			L2GenesisBlockDeployConfig: L2GenesisBlockDeployConfig{L2GenesisBlockTimestamp: &override},
		}}
		got, err := config.L2GenesisTime(5000)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), got)
	})

	t.Run("below the L1 start time is rejected", func(t *testing.T) {
		override := hexutil.Uint64(4999)
		config := &DeployConfig{L2InitializationConfig: L2InitializationConfig{
			L2GenesisBlockDeployConfig: L2GenesisBlockDeployConfig{L2GenesisBlockTimestamp: &override},
		}}
		_, err := config.L2GenesisTime(5000)
		require.ErrorIs(t, err, ErrInvalidDeployConfig)
	})
}

func TestNewL2Genesis_TimestampOverride(t *testing.T) {
	regolithOffset := hexutil.Uint64(1500)
	makeConfig := func(timestamp *hexutil.Uint64) *DeployConfig {
		return &DeployConfig{L2InitializationConfig: L2InitializationConfig{
			L2CoreDeployConfig:          L2CoreDeployConfig{L2ChainID: 1234},
			L2GenesisBlockDeployConfig:  L2GenesisBlockDeployConfig{L2GenesisBlockTimestamp: timestamp},
			UpgradeScheduleDeployConfig: UpgradeScheduleDeployConfig{L2GenesisRegolithTimeOffset: &regolithOffset},
		}}
	}
	l1Start := &eth.BlockRef{Time: 5000}

	t.Run("unset uses the L1 start time", func(t *testing.T) {
		genesis, err := NewL2Genesis(makeConfig(nil), l1Start)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), genesis.Timestamp)
		require.Equal(t, uint64(5000+1500), *genesis.Config.RegolithTime)
	})

	t.Run("override sets the genesis timestamp", func(t *testing.T) {
		override := hexutil.Uint64(9000)
		genesis, err := NewL2Genesis(makeConfig(&override), l1Start)
		require.NoError(t, err)
		require.Equal(t, uint64(9000), genesis.Timestamp)
		require.Equal(t, uint64(9000+1500), *genesis.Config.RegolithTime)
	})

	t.Run("override below the L1 start time errors", func(t *testing.T) {
		override := hexutil.Uint64(4999)
		_, err := NewL2Genesis(makeConfig(&override), l1Start)
		require.ErrorIs(t, err, ErrInvalidDeployConfig)
	})
}

func TestRollupConfig_TimestampOverride(t *testing.T) {
	regolithOffset := hexutil.Uint64(1500)
	makeConfig := func(timestamp *hexutil.Uint64) *DeployConfig {
		return &DeployConfig{
			L2InitializationConfig: L2InitializationConfig{
				L2GenesisBlockDeployConfig:  L2GenesisBlockDeployConfig{L2GenesisBlockTimestamp: timestamp},
				UpgradeScheduleDeployConfig: UpgradeScheduleDeployConfig{L2GenesisRegolithTimeOffset: &regolithOffset},
			},
			L1DependenciesConfig: L1DependenciesConfig{
				SystemConfigProxy:   common.HexToAddress("0x01"),
				OptimismPortalProxy: common.HexToAddress("0x02"),
			},
		}
	}
	l1Start := &eth.BlockRef{Hash: common.HexToHash("0xaa"), Number: 100, Time: 5000}

	t.Run("unset uses the L1 start time", func(t *testing.T) {
		cfg, err := makeConfig(nil).RollupConfig(l1Start, common.HexToHash("0xbb"), 0)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), cfg.Genesis.L2Time)
		require.Equal(t, uint64(5000+1500), *cfg.RegolithTime)
	})

	t.Run("override sets L2Time", func(t *testing.T) {
		override := hexutil.Uint64(9000)
		cfg, err := makeConfig(&override).RollupConfig(l1Start, common.HexToHash("0xbb"), 0)
		require.NoError(t, err)
		require.Equal(t, uint64(9000), cfg.Genesis.L2Time)
		require.Equal(t, uint64(9000+1500), *cfg.RegolithTime)
	})

	t.Run("override below the L1 start time errors", func(t *testing.T) {
		override := hexutil.Uint64(4999)
		_, err := makeConfig(&override).RollupConfig(l1Start, common.HexToHash("0xbb"), 0)
		require.ErrorIs(t, err, ErrInvalidDeployConfig)
	})
}

func TestCanyonTimeZero(t *testing.T) {
	canyonOffset := hexutil.Uint64(0)
	config := &DeployConfig{
		L2InitializationConfig: L2InitializationConfig{
			UpgradeScheduleDeployConfig: UpgradeScheduleDeployConfig{
				L2GenesisCanyonTimeOffset: &canyonOffset,
			},
		},
	}
	require.Equal(t, uint64(0), *config.CanyonTime(1234))
}

func TestCanyonTimeOffset(t *testing.T) {
	canyonOffset := hexutil.Uint64(1500)
	config := &DeployConfig{
		L2InitializationConfig: L2InitializationConfig{
			UpgradeScheduleDeployConfig: UpgradeScheduleDeployConfig{
				L2GenesisCanyonTimeOffset: &canyonOffset,
			},
		},
	}
	require.Equal(t, uint64(1234+1500), *config.CanyonTime(1234))
}

func TestForksCantActivateAtSamePostGenesisBlock(t *testing.T) {
	postGenesisOffset := uint64(1500)
	config := &UpgradeScheduleDeployConfig{}
	for _, fork := range config.forks() {
		config.SetForkTimeOffset(forks.Name(fork.Name), &postGenesisOffset)
	}
	err := config.Check(testlog.Logger(t, log.LevelDebug))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "Forks in general cannot activate at the same post-Genesis block"))
}

// TestCopy will copy a DeployConfig and ensure that the copy is equal to the original.
func TestCopy(t *testing.T) {
	b, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)

	decoded := new(DeployConfig)
	require.NoError(t, json.NewDecoder(bytes.NewReader(b)).Decode(decoded))

	cpy := decoded.Copy()
	require.EqualValues(t, decoded, cpy)

	offset := hexutil.Uint64(100)
	cpy.L2GenesisRegolithTimeOffset = &offset
	require.NotEqual(t, decoded, cpy)
}

// TestL1Deployments ensures that NewL1Deployments can read a JSON file
// from disk and deserialize all of the key/value pairs correctly.
func TestL1Deployments(t *testing.T) {
	deployments, err := NewL1Deployments("testdata/l1-deployments.json")
	require.NoError(t, err)

	require.NotEqual(t, deployments.AddressManager, common.Address{})
	require.NotEqual(t, deployments.DisputeGameFactory, common.Address{})
	require.NotEqual(t, deployments.DisputeGameFactoryProxy, common.Address{})
	require.NotEqual(t, deployments.L1CrossDomainMessenger, common.Address{})
	require.NotEqual(t, deployments.L1CrossDomainMessengerProxy, common.Address{})
	require.NotEqual(t, deployments.L1ERC721Bridge, common.Address{})
	require.NotEqual(t, deployments.L1ERC721BridgeProxy, common.Address{})
	require.NotEqual(t, deployments.L1StandardBridge, common.Address{})
	require.NotEqual(t, deployments.L1StandardBridgeProxy, common.Address{})
	require.NotEqual(t, deployments.L2OutputOracle, common.Address{})
	require.NotEqual(t, deployments.L2OutputOracleProxy, common.Address{})
	require.NotEqual(t, deployments.OptimismMintableERC20Factory, common.Address{})
	require.NotEqual(t, deployments.OptimismMintableERC20FactoryProxy, common.Address{})
	require.NotEqual(t, deployments.OptimismPortal, common.Address{})
	require.NotEqual(t, deployments.OptimismPortalProxy, common.Address{})
	require.NotEqual(t, deployments.ProxyAdmin, common.Address{})
	require.NotEqual(t, deployments.SystemConfig, common.Address{})
	require.NotEqual(t, deployments.SystemConfigProxy, common.Address{})

	require.Equal(t, "AddressManager", deployments.GetName(deployments.AddressManager))
	require.Equal(t, "OptimismPortalProxy", deployments.GetName(deployments.OptimismPortalProxy))
	// One that doesn't exist returns empty string
	require.Equal(t, "", deployments.GetName(common.Address{19: 0xff}))
}

// This test guarantees that getters and setters for all forks are present.
func TestUpgradeScheduleDeployConfig_ForkGettersAndSetters(t *testing.T) {
	var d UpgradeScheduleDeployConfig
	for i, fork := range forks.From(forks.Regolith) {
		require.Nil(t, d.ForkTimeOffset(fork))
		offset := uint64(i * 42)
		d.SetForkTimeOffset(fork, &offset)
		require.Equal(t, offset, *d.ForkTimeOffset(fork))
	}
}

func TestUpgradeScheduleDeployConfig_ActivateForkAtOffset(t *testing.T) {
	var d UpgradeScheduleDeployConfig
	ts := uint64(42)
	t.Run("invalid", func(t *testing.T) {
		require.Panics(t, func() { d.ActivateForkAtOffset(forks.Bedrock, ts) })
	})

	t.Run("regolith", func(t *testing.T) {
		d.ActivateForkAtOffset(forks.Regolith, ts)
		require.EqualValues(t, &ts, d.L2GenesisRegolithTimeOffset)
		for _, fork := range scheduleableForks[1:] {
			require.Nil(t, d.ForkTimeOffset(fork))
		}
	})

	t.Run("ecotone", func(t *testing.T) {
		d.ActivateForkAtOffset(forks.Ecotone, ts)
		require.EqualValues(t, &ts, d.L2GenesisEcotoneTimeOffset)
		for _, fork := range scheduleableForks[:3] {
			require.Zero(t, *d.ForkTimeOffset(fork))
		}
		for _, fork := range scheduleableForks[4:] {
			require.Nil(t, d.ForkTimeOffset(fork))
		}
	})
}

func TestUpgradeScheduleDeployConfig_SolidityForkNumber(t *testing.T) {
	// Iterate over all of them in case more are added
	for i, fork := range scheduleableForks[2:] {
		var d UpgradeScheduleDeployConfig
		d.ActivateForkAtOffset(fork, 0)
		require.EqualValues(t, i+1, d.SolidityForkNumber(uint64(42)))
	}

	// Also validate that each fork manually, for sanity
	tests := []struct {
		fork     rollup.ForkName
		expected int64
	}{
		{forks.Delta, 1},
		{forks.Ecotone, 2},
		{forks.Fjord, 3},
		{forks.Granite, 4},
		{forks.Holocene, 5},
		{forks.Isthmus, 6},
		{forks.Jovian, 7},
		{forks.Karst, 8},
		{forks.Lagoon, 9},
	}
	for _, tt := range tests {
		var d UpgradeScheduleDeployConfig
		d.ActivateForkAtGenesis(tt.fork)
		require.EqualValues(t, tt.expected, d.SolidityForkNumber(uint64(42)))
	}
}
