package state

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestCombineDeployConfig(t *testing.T) {
	intent := Intent{
		L1ChainID:          1,
		L1ContractsLocator: artifacts.EmbeddedLocator,
	}
	chainState := ChainState{
		ID: common.HexToHash("0x123"),
	}
	chainIntent := ChainIntent{
		Eip1559Denominator:         1,
		Eip1559Elasticity:          2,
		BaseFeeVaultRecipient:      common.HexToAddress("0x123"),
		L1FeeVaultRecipient:        common.HexToAddress("0x456"),
		SequencerFeeVaultRecipient: common.HexToAddress("0x789"),
		Roles: ChainRoles{
			SystemConfigOwner: common.HexToAddress("0x123"),
			L1ProxyAdminOwner: common.HexToAddress("0x456"),
			L2ProxyAdminOwner: common.HexToAddress("0x789"),
			UnsafeBlockSigner: common.HexToAddress("0xabc"),
			Batcher:           common.HexToAddress("0xdef"),
		},
	}
	state := State{
		SuperchainDeployment: &addresses.SuperchainContracts{ProtocolVersionsProxy: common.HexToAddress("0x123")},
	}

	// apply hard fork overrides
	chainIntent.DeployOverrides = map[string]any{
		"l2GenesisFjordTimeOffset":    "0x1",
		"l2GenesisGraniteTimeOffset":  "0x2",
		"l2GenesisHoloceneTimeOffset": "0x3",
		"l2GenesisIsthmusTimeOffset":  "0x4",
		"l2GenesisJovianTimeOffset":   "0x5",
		"l2GenesisInteropTimeOffset":  "0x6",
	}

	out, err := CombineDeployConfig(&intent, &chainIntent, &state, &chainState)
	require.NoError(t, err)
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisFjordTimeOffset, hexutil.Uint64(1))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisGraniteTimeOffset, hexutil.Uint64(2))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisHoloceneTimeOffset, hexutil.Uint64(3))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisIsthmusTimeOffset, hexutil.Uint64(4))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisJovianTimeOffset, hexutil.Uint64(5))
	require.Equal(t, *out.L2InitializationConfig.UpgradeScheduleDeployConfig.L2GenesisInteropTimeOffset, hexutil.Uint64(6))
}
