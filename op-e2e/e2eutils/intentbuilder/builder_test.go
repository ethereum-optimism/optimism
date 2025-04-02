package intentbuilder

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"net/url"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	// Create a new builder
	builder := New()
	require.NotNil(t, builder)

	// Configure Superchain
	builder, superchainConfig := builder.WithSuperchain()
	require.NotNil(t, superchainConfig)
	superchainConfigProxyAddr := common.HexToAddress("0x9999")
	superchainConfig.WithSuperchainConfigProxy(superchainConfigProxyAddr)
	superchainConfig.WithProxyAdminOwner(common.HexToAddress("0xaaaa"))
	superchainConfig.WithGuardian(common.HexToAddress("0xbbbb"))
	superchainConfig.WithProtocolVersionsOwner(common.HexToAddress("0xcccc"))

	// Configure L1
	l1StartTimestamp := uint64(1000)
	builder, l1Config := builder.WithL1(eth.ChainIDFromUInt64(1))
	require.NotNil(t, l1Config)
	l1Config.WithStartTimestamp(l1StartTimestamp)

	// Configure L2
	builder, l2Config := builder.WithL2(eth.ChainIDFromUInt64(420))
	require.NotNil(t, l2Config)

	// Test direct L2Configurator methods
	require.Equal(t, eth.ChainIDFromUInt64(420), l2Config.ChainID())
	l2Config.WithBlockTime(2)
	l2Config.WithL1StartBlockHash(common.HexToHash("0x5678"))

	// Test ContractsConfigurator methods
	l2Config.WithL1ContractsLocator("http://l1.example.com")
	l2Config.WithL2ContractsLocator("http://l2.example.com")

	// Test L2VaultsConfigurator methods
	baseFeeRecipient := common.HexToAddress("0x1111")
	sequencerFeeRecipient := common.HexToAddress("0x2222")
	l1FeeRecipient := common.HexToAddress("0x3333")
	l2Config.WithBaseFeeVaultRecipient(baseFeeRecipient)
	l2Config.WithSequencerFeeVaultRecipient(sequencerFeeRecipient)
	l2Config.WithL1FeeVaultRecipient(l1FeeRecipient)

	// Test L2RolesConfigurator methods
	l1ProxyAdminOwner := common.HexToAddress("0x4444")
	l2ProxyAdminOwner := common.HexToAddress("0x5555")
	systemConfigOwner := common.HexToAddress("0x6666")
	unsafeBlockSigner := common.HexToAddress("0x7777")
	batcher := common.HexToAddress("0x8888")
	proposer := common.HexToAddress("0x9999")
	challenger := common.HexToAddress("0xaaaa")
	l2Config.WithL1ProxyAdminOwner(l1ProxyAdminOwner)
	l2Config.WithL2ProxyAdminOwner(l2ProxyAdminOwner)
	l2Config.WithSystemConfigOwner(systemConfigOwner)
	l2Config.WithUnsafeBlockSigner(unsafeBlockSigner)
	l2Config.WithBatcher(batcher)
	l2Config.WithProposer(proposer)
	l2Config.WithChallenger(challenger)

	// Test L2FeesConfigurator methods
	l2Config.WithEIP1559DenominatorCanyon(250)
	l2Config.WithEIP1559Denominator(50)
	l2Config.WithEIP1559Elasticity(10)
	l2Config.WithOperatorFeeScalar(100)
	l2Config.WithOperatorFeeConstant(200)

	// Test L2HardforkConfigurator methods
	isthmusOffset := uint64(8000)
	l2Config.WithForkAtGenesis(rollup.Holocene)
	l2Config.WithForkAtOffset(rollup.Isthmus, &isthmusOffset)

	// Build the intent
	intent, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, intent)

	// Create expected intent structure
	chainID := eth.ChainIDFromUInt64(420)
	expectedIntent := &state.Intent{
		ConfigType:            state.IntentTypeCustom,
		L1ChainID:             1,
		SuperchainConfigProxy: &superchainConfigProxyAddr,
		SuperchainRoles: &state.SuperchainRoles{
			ProxyAdminOwner:       common.HexToAddress("0xaaaa"),
			Guardian:              common.HexToAddress("0xbbbb"),
			ProtocolVersionsOwner: common.HexToAddress("0xcccc"),
		},
		L1StartTimestamp: &l1StartTimestamp,
		L1ContractsLocator: &artifacts.Locator{
			URL: &url.URL{
				Scheme: "http",
				Host:   "l1.example.com",
			},
		},
		L2ContractsLocator: &artifacts.Locator{
			URL: &url.URL{
				Scheme: "http",
				Host:   "l2.example.com",
			},
		},
		Chains: []*state.ChainIntent{
			{
				ID:                         common.BigToHash((&chainID).ToBig()),
				BaseFeeVaultRecipient:      baseFeeRecipient,
				SequencerFeeVaultRecipient: sequencerFeeRecipient,
				L1FeeVaultRecipient:        l1FeeRecipient,
				Roles: state.ChainRoles{
					L1ProxyAdminOwner: l1ProxyAdminOwner,
					L2ProxyAdminOwner: l2ProxyAdminOwner,
					SystemConfigOwner: systemConfigOwner,
					UnsafeBlockSigner: unsafeBlockSigner,
					Batcher:           batcher,
					Proposer:          proposer,
					Challenger:        challenger,
				},
				Eip1559DenominatorCanyon: 250,
				Eip1559Denominator:       50,
				Eip1559Elasticity:        10,
				OperatorFeeScalar:        100,
				OperatorFeeConstant:      200,
				DeployOverrides: map[string]any{
					"l2BlockTime":                 uint64(2),
					"l2GenesisRegolithTimeOffset": 0,
					"l2GenesisCanyonTimeOffset":   0,
					"l2GenesisDeltaTimeOffset":    0,
					"l2GenesisEcotoneTimeOffset":  0,
					"l2GenesisFjordTimeOffset":    0,
					"l2GenesisGraniteTimeOffset":  0,
					"l2GenesisHoloceneTimeOffset": 0,
					"l2GenesisIsthmusTimeOffset":  isthmusOffset,
				},
			},
		},
	}

	// Convert both intents to JSON for comparison
	actualJSON, err := json.Marshal(intent)
	require.NoError(t, err)
	expectedJSON, err := json.Marshal(expectedIntent)
	require.NoError(t, err)

	require.JSONEq(t, string(expectedJSON), string(actualJSON))
}
