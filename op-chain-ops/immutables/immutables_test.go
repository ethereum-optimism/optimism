package immutables_test

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBuildOptimism(t *testing.T) {
	cfg := immutables.PredeploysImmutableConfig{
		L2ToL1MessagePasser: struct{}{},
		DeployerWhitelist:   struct{}{},
		WETH9:               struct{}{},
		L2CrossDomainMessenger: struct{ OtherMessenger common.Address }{
			OtherMessenger: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		L2StandardBridge: struct {
			OtherBridge common.Address
			Messenger   common.Address
		}{
			OtherBridge: common.HexToAddress("0x1234567890123456789012345678901234567890"),
			Messenger:   predeploys.L2CrossDomainMessengerAddr,
		},
		SequencerFeeVault: struct {
			Recipient           common.Address
			MinWithdrawalAmount *big.Int
			WithdrawalNetwork   uint8
		}{
			Recipient:           common.HexToAddress("0x1234567890123456789012345678901234567890"),
			MinWithdrawalAmount: big.NewInt(100),
			WithdrawalNetwork:   0,
		},
		L1BlockNumber:       struct{}{},
		GasPriceOracle:      struct{}{},
		L1Block:             struct{}{},
		GovernanceToken:     struct{}{},
		LegacyMessagePasser: struct{}{},
		L2ERC721Bridge: struct {
			OtherBridge common.Address
			Messenger   common.Address
		}{
			OtherBridge: common.HexToAddress("0x1234567890123456789012345678901234567890"),
			Messenger:   predeploys.L2CrossDomainMessengerAddr,
		},
		OptimismMintableERC721Factory: struct {
			Bridge        common.Address
			RemoteChainId *big.Int
		}{
			Bridge:        predeploys.L2StandardBridgeAddr,
			RemoteChainId: big.NewInt(1),
		},
		OptimismMintableERC20Factory: struct {
			Bridge common.Address
		}{
			Bridge: predeploys.L2StandardBridgeAddr,
		},
		ProxyAdmin: struct{}{},
		BaseFeeVault: struct {
			Recipient           common.Address
			MinWithdrawalAmount *big.Int
			WithdrawalNetwork   uint8
		}{
			Recipient:           common.HexToAddress("0x1234567890123456789012345678901234567890"),
			MinWithdrawalAmount: big.NewInt(200),
			WithdrawalNetwork:   0,
		},
		L1FeeVault: struct {
			Recipient           common.Address
			MinWithdrawalAmount *big.Int
			WithdrawalNetwork   uint8
		}{
			Recipient:           common.HexToAddress("0x1234567890123456789012345678901234567890"),
			MinWithdrawalAmount: big.NewInt(200),
			WithdrawalNetwork:   1,
		},
		SchemaRegistry: struct{}{},
		EAS: struct{ Name string }{
			Name: "EAS",
		},
		Create2Deployer:              struct{}{},
		MultiCall3:                   struct{}{},
		Safe_v130:                    struct{}{},
		SafeL2_v130:                  struct{}{},
		MultiSendCallOnly_v130:       struct{}{},
		SafeSingletonFactory:         struct{}{},
		DeterministicDeploymentProxy: struct{}{},
		MultiSend_v130:               struct{}{},
		Permit2:                      struct{}{},
		SenderCreator:                struct{}{},
		EntryPoint:                   struct{}{},
	}

	require.NoError(t, cfg.Check())
	results, err := immutables.Deploy(&cfg)
	require.NoError(t, err)
	require.NotNil(t, results)

	// Build a mapping of all of the predeploys
	all := map[string]bool{}
	// Build a mapping of the predeploys with immutable config
	withConfig := map[string]bool{}

	require.NoError(t, cfg.ForEach(func(name string, predeployConfig any) error {
		all[name] = true

		// If a predeploy has no config, it needs to have no immutable references in the solc output.
		if reflect.ValueOf(predeployConfig).IsZero() {
			ref, _ := bindings.HasImmutableReferences(name)
			require.Zero(t, ref, "found immutable reference for %s", name)
			return nil
		}
		withConfig[name] = true
		return nil
	}))

	// Ensure that the PredeploysImmutableConfig is kept up to date
	for name := range predeploys.Predeploys {
		require.Truef(t, all[name], "predeploy %s not in set of predeploys", name)

		ref, err := bindings.HasImmutableReferences(name)
		// If there is predeploy config, there should be an immutable reference
		if withConfig[name] {
			require.NoErrorf(t, err, "error getting immutable reference for %s", name)
			require.NotZerof(t, ref, "no immutable reference for %s", name)
		} else {
			require.Zero(t, ref, "found immutable reference for %s", name)
		}
	}

	// Only the exact contracts that we care about are being modified
	require.Equal(t, len(results), len(withConfig))

	for name, bytecode := range results {
		// There is bytecode there
		require.Greater(t, len(bytecode), 0)
		// It is in the set of contracts that we care about
		require.Truef(t, withConfig[name], "contract %s not in set of contracts", name)
		// The immutable reference is present
		ref, err := bindings.HasImmutableReferences(name)
		require.NoErrorf(t, err, "cannot get immutable reference for %s", name)
		require.NotZerof(t, ref, "contract %s has no immutable reference", name)
	}
}
