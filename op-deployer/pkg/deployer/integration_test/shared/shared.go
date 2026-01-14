package shared

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// AddrFor generates an address for a given role
func AddrFor(t *testing.T, dk *devkeys.MnemonicDevKeys, key devkeys.Key) common.Address {
	addr, err := dk.Address(key)
	require.NoError(t, err)
	return addr
}

func NewChainIntent(t *testing.T, dk *devkeys.MnemonicDevKeys, l1ChainID *big.Int, l2ChainID *uint256.Int, gasLimit uint64) *state.ChainIntent {
	return &state.ChainIntent{
		ID:                         l2ChainID.Bytes32(),
		BaseFeeVaultRecipient:      AddrFor(t, dk, devkeys.BaseFeeVaultRecipientRole.Key(l1ChainID)),
		L1FeeVaultRecipient:        AddrFor(t, dk, devkeys.L1FeeVaultRecipientRole.Key(l1ChainID)),
		SequencerFeeVaultRecipient: AddrFor(t, dk, devkeys.SequencerFeeVaultRecipientRole.Key(l1ChainID)),
		OperatorFeeVaultRecipient:  AddrFor(t, dk, devkeys.OperatorFeeVaultRecipientRole.Key(l1ChainID)),
		Eip1559DenominatorCanyon:   standard.Eip1559DenominatorCanyon,
		Eip1559Denominator:         standard.Eip1559Denominator,
		Eip1559Elasticity:          standard.Eip1559Elasticity,
		GasLimit:                   gasLimit,
		Roles: state.ChainRoles{
			L1ProxyAdminOwner: AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			L2ProxyAdminOwner: AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			SystemConfigOwner: AddrFor(t, dk, devkeys.SystemConfigOwner.Key(l1ChainID)),
			UnsafeBlockSigner: AddrFor(t, dk, devkeys.SequencerP2PRole.Key(l1ChainID)),
			Batcher:           AddrFor(t, dk, devkeys.BatcherRole.Key(l1ChainID)),
			Proposer:          AddrFor(t, dk, devkeys.ProposerRole.Key(l1ChainID)),
			Challenger:        AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainID)),
		},
		UseRevenueShare:    false,
		ChainFeesRecipient: common.Address{},
		// CustomGasToken defaults to disabled (all fields nil/empty)
		CustomGasToken: state.CustomGasToken{},
	}
}

func NewIntent(
	t *testing.T,
	l1ChainID *big.Int,
	dk *devkeys.MnemonicDevKeys,
	l2ChainID *uint256.Int,
	l1Loc *artifacts.Locator,
	l2Loc *artifacts.Locator,
	gasLimit uint64,
) (*state.Intent, *state.State) {
	intent := &state.Intent{
		ConfigType: state.IntentTypeCustom,
		L1ChainID:  l1ChainID.Uint64(),
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID)),
			ProtocolVersionsOwner:     AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainID)),
			SuperchainGuardian:        AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainID)),
			Challenger:                AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainID)),
		},
		FundDevAccounts:    false,
		L1ContractsLocator: l1Loc,
		L2ContractsLocator: l2Loc,
		Chains: []*state.ChainIntent{
			NewChainIntent(t, dk, l1ChainID, l2ChainID, gasLimit),
		},
	}
	st := &state.State{
		Version: 1,
	}
	return intent, st
}

// DefaultPrivkey returns the default private key for testing
func DefaultPrivkey(t *testing.T) (string, *ecdsa.PrivateKey, *devkeys.MnemonicDevKeys) {
	pkHex := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	pk, err := crypto.HexToECDSA(pkHex)
	require.NoError(t, err)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	return pkHex, pk, dk
}

// PastUpgradeConfig defines configuration for a past upgrade that needs to be executed
// when forking a network at a block before that upgrade was executed.
type PastUpgradeConfig struct {
	// Name is a human-readable name for the upgrade (for logging)
	Name string
	// OpcmAddresses maps chain IDs to the OPCM address used for this upgrade
	OpcmAddresses map[uint64]common.Address
	// BuildUpgradeConfig builds the upgrade configuration for the given parameters
	BuildUpgradeConfig func(prank, systemConfigProxy common.Address) embedded.UpgradeOPChainInput
}

// PastUpgrades is the list of past upgrades that need to be executed in order.
// Add new upgrades here when introducing new upgrades that need to be sequenced.
var PastUpgrades = []PastUpgradeConfig{
	{
		Name: "U18",
		OpcmAddresses: map[uint64]common.Address{
			1:        common.HexToAddress("0x50F47B43c24F40B92C873Fa0704D4207586D0C9f"), // Mainnet
			11155111: common.HexToAddress("0xf0a2e224519e876979ea6b2cd15ef5cc3d6703bd"), // Sepolia
		},
		BuildUpgradeConfig: func(prank, systemConfigProxy common.Address) embedded.UpgradeOPChainInput {
			return embedded.UpgradeOPChainInput{
				Prank: prank,
				// Opcm is set by the caller based on chainID
				ChainConfigs: []embedded.OPChainConfig{
					{
						SystemConfigProxy:  systemConfigProxy,
						CannonPrestate:     common.Hash{'C', 'A', 'N', 'N', 'O', 'N'},
						CannonKonaPrestate: common.Hash{'K', 'O', 'N', 'A'},
					},
				},
			}
		},
	},
}

// RunPastUpgrades executes all past OPCM upgrades that have not yet been executed on the forked network.
// This is necessary when forking a network at a block before certain upgrades were executed.
func RunPastUpgrades(t *testing.T, host *script.Host, chainID uint64, prank common.Address, systemConfigProxy common.Address) {
	t.Helper()

	for _, upgrade := range PastUpgrades {
		opcmAddr, ok := upgrade.OpcmAddresses[chainID]
		if !ok {
			t.Logf("No %s OPCM address configured for chain %d, skipping", upgrade.Name, chainID)
			continue
		}

		upgradeConfig := upgrade.BuildUpgradeConfig(prank, systemConfigProxy)
		upgradeConfig.Opcm = opcmAddr

		upgradeConfigBytes, err := json.Marshal(upgradeConfig)
		require.NoError(t, err, "Failed to marshal %s upgrade config", upgrade.Name)

		err = embedded.DefaultUpgrader.Upgrade(host, upgradeConfigBytes)
		if err != nil {
			// It's acceptable for this to fail if the upgrade was already executed
			t.Logf("%s upgrade may have already been executed or failed: %v", upgrade.Name, err)
		} else {
			t.Logf("Successfully executed %s upgrade using OPCM at %s", upgrade.Name, opcmAddr.Hex())
		}
	}
}

// RunPastUpgradesWithRPC is a convenience function that creates a script host and runs past upgrades.
// Use this when you don't already have a script host available.
func RunPastUpgradesWithRPC(t *testing.T, l1RPCUrl string, afactsFS foundry.StatDirFs, lgr log.Logger, chainID uint64, prank common.Address, systemConfigProxy common.Address) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(l1RPCUrl)
	require.NoError(t, err)

	host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		lgr,
		prank,
		afactsFS,
		rpcClient,
	)
	require.NoError(t, err)

	RunPastUpgrades(t, host, chainID, prank, systemConfigProxy)
}
