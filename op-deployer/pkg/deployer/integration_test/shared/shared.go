package shared

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum/go-ethereum/accounts/abi"
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

// pastUpgradeExtraInstructionTOML represents an extra instruction in the TOML file
type pastUpgradeExtraInstructionTOML struct {
	Key     string `toml:"key"`
	Data    string `toml:"data,omitempty"`     // string data (will be converted to bytes)
	DataHex string `toml:"data_hex,omitempty"` // hex-encoded data
}

// pastDisputeGameConfigTOML represents a dispute game config in the TOML file (V2 only)
type pastDisputeGameConfigTOML struct {
	GameType   string `toml:"game_type"`            // "CANNON", "PERMISSIONED_CANNON", "CANNON_KONA", etc.
	Enabled    bool   `toml:"enabled"`              // whether this game type is enabled
	Prestate   string `toml:"prestate"`             // prestate hash for this game type
	InitBond   string `toml:"init_bond,omitempty"`  // initial bond in wei (optional)
	Proposer   string `toml:"proposer,omitempty"`   // proposer address (PERMISSIONED only)
	Challenger string `toml:"challenger,omitempty"` // challenger address (PERMISSIONED only)
}

// pastUpgradeTOML represents a single upgrade entry in the TOML file
type pastUpgradeTOML struct {
	Name          string            `toml:"name"`
	OpcmVersion   int               `toml:"opcm_version"`
	OpcmAddresses map[string]string `toml:"opcm_addresses"` // keys are chain IDs as strings
	// V1-only fields
	CannonPrestate     string `toml:"cannon_prestate,omitempty"`
	CannonKonaPrestate string `toml:"cannon_kona_prestate,omitempty"`
	// V2-only fields
	DisputeGameConfigs []pastDisputeGameConfigTOML       `toml:"dispute_game_configs,omitempty"`
	ExtraInstructions  []pastUpgradeExtraInstructionTOML `toml:"extra_instructions,omitempty"`
}

// pastUpgradesTOML represents the structure of the past_upgrades.toml file
type pastUpgradesTOML struct {
	Upgrades []pastUpgradeTOML `toml:"upgrades"`
}

// PastUpgradeExtraInstruction represents an extra instruction for V2 upgrades
type PastUpgradeExtraInstruction struct {
	Key  string
	Data []byte
}

// PastDisputeGameConfig represents a parsed dispute game config for V2 upgrades
type PastDisputeGameConfig struct {
	GameType   string         // "CANNON", "PERMISSIONED_CANNON", "CANNON_KONA", etc.
	Enabled    bool           // whether this game type is enabled
	Prestate   common.Hash    // prestate hash for this game type
	InitBond   *big.Int       // initial bond in wei (nil means use factory default)
	Proposer   common.Address // proposer address (only for PERMISSIONED game types)
	Challenger common.Address // challenger address (only for PERMISSIONED game types)
}

// PastUpgradeConfig defines configuration for a past upgrade that needs to be executed
// when forking a network at a block before that upgrade was executed.
type PastUpgradeConfig struct {
	// Name is a human-readable name for the upgrade (for logging)
	Name string
	// OpcmVersion is 1 or 2, determining which OPCM interface to use
	OpcmVersion int
	// OpcmAddresses maps chain IDs to the OPCM address used for this upgrade
	OpcmAddresses map[uint64]common.Address
	// V1-only fields
	// CannonPrestate is the prestate for cannon (V1 only)
	CannonPrestate common.Hash
	// CannonKonaPrestate is the prestate for cannon-kona (V1 only)
	CannonKonaPrestate common.Hash
	// V2-only fields
	// DisputeGameConfigs are the dispute game configurations (V2 only)
	DisputeGameConfigs []PastDisputeGameConfig
	// ExtraInstructions are additional config instructions (V2 only)
	ExtraInstructions []PastUpgradeExtraInstruction
}

var (
	pastUpgradesOnce sync.Once
	pastUpgrades     []PastUpgradeConfig
	pastUpgradesErr  error
)

// getRepoRoot finds the repository root by looking for the go.mod file
func getRepoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}

	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// loadPastUpgrades loads the past upgrades from the TOML file in contracts-bedrock
func loadPastUpgrades() ([]PastUpgradeConfig, error) {
	pastUpgradesOnce.Do(func() {
		repoRoot, err := getRepoRoot()
		if err != nil {
			pastUpgradesErr = err
			return
		}

		tomlPath := filepath.Join(repoRoot, "packages", "contracts-bedrock", "past_upgrades.toml")
		var tomlConfig pastUpgradesTOML
		if _, err := toml.DecodeFile(tomlPath, &tomlConfig); err != nil {
			pastUpgradesErr = err
			return
		}

		for _, upgrade := range tomlConfig.Upgrades {
			opcmAddresses := make(map[uint64]common.Address)
			for chainIDStr, addrStr := range upgrade.OpcmAddresses {
				var chainID uint64
				if _, err := fmt.Sscanf(chainIDStr, "%d", &chainID); err != nil {
					continue // skip invalid chain IDs
				}
				opcmAddresses[chainID] = common.HexToAddress(addrStr)
			}

			config := PastUpgradeConfig{
				Name:          upgrade.Name,
				OpcmVersion:   upgrade.OpcmVersion,
				OpcmAddresses: opcmAddresses,
			}

			if upgrade.OpcmVersion == 1 {
				// V1: Parse prestates directly
				config.CannonPrestate = common.HexToHash(upgrade.CannonPrestate)
				config.CannonKonaPrestate = common.HexToHash(upgrade.CannonKonaPrestate)
			} else if upgrade.OpcmVersion == 2 {
				// V2: Parse dispute game configs
				for _, dgc := range upgrade.DisputeGameConfigs {
					var initBond *big.Int
					if dgc.InitBond != "" {
						initBond = new(big.Int)
						initBond.SetString(dgc.InitBond, 10)
					}
					config.DisputeGameConfigs = append(config.DisputeGameConfigs, PastDisputeGameConfig{
						GameType:   dgc.GameType,
						Enabled:    dgc.Enabled,
						Prestate:   common.HexToHash(dgc.Prestate),
						InitBond:   initBond,
						Proposer:   common.HexToAddress(dgc.Proposer),
						Challenger: common.HexToAddress(dgc.Challenger),
					})
				}

				// Parse extra instructions
				for _, instr := range upgrade.ExtraInstructions {
					var data []byte
					if instr.DataHex != "" {
						data = common.FromHex(instr.DataHex)
					} else {
						data = []byte(instr.Data)
					}
					config.ExtraInstructions = append(config.ExtraInstructions, PastUpgradeExtraInstruction{
						Key:  instr.Key,
						Data: data,
					})
				}
			}

			pastUpgrades = append(pastUpgrades, config)
		}
	})

	return pastUpgrades, pastUpgradesErr
}

// GetPastUpgrades returns the list of past upgrades loaded from the YAML file.
// This is the list of past upgrades that need to be executed in order.
// Add new upgrades to packages/contracts-bedrock/past_upgrades.yaml
func GetPastUpgrades(t *testing.T) []PastUpgradeConfig {
	upgrades, err := loadPastUpgrades()
	require.NoError(t, err, "Failed to load past upgrades from packages/contracts-bedrock/past_upgrades.yaml")
	return upgrades
}

// RunPastUpgrades executes all past OPCM upgrades that have not yet been executed on the forked network.
// This is necessary when forking a network at a block before certain upgrades were executed.
func RunPastUpgrades(t *testing.T, host *script.Host, chainID uint64, prank common.Address, systemConfigProxy common.Address) {
	t.Helper()

	upgrades := GetPastUpgrades(t)
	for _, upgrade := range upgrades {
		opcmAddr, ok := upgrade.OpcmAddresses[chainID]
		if !ok {
			t.Logf("No %s OPCM address configured for chain %d, skipping", upgrade.Name, chainID)
			continue
		}

		var upgradeConfig embedded.UpgradeOPChainInput
		switch upgrade.OpcmVersion {
		case 1:
			upgradeConfig = embedded.UpgradeOPChainInput{
				Prank: prank,
				Opcm:  opcmAddr,
				ChainConfigs: []embedded.OPChainConfig{
					{
						SystemConfigProxy:  systemConfigProxy,
						CannonPrestate:     upgrade.CannonPrestate,
						CannonKonaPrestate: upgrade.CannonKonaPrestate,
					},
				},
			}
		case 2:
			upgradeConfig = buildV2UpgradeConfig(t, prank, opcmAddr, systemConfigProxy, upgrade)
		default:
			t.Logf("Unknown OPCM version %d for %s upgrade, skipping", upgrade.OpcmVersion, upgrade.Name)
			continue
		}

		upgradeConfigBytes, err := json.Marshal(upgradeConfig)
		require.NoError(t, err, "Failed to marshal %s upgrade config", upgrade.Name)

		err = embedded.DefaultUpgrader.Upgrade(host, upgradeConfigBytes)
		if err != nil {
			// It's acceptable for this to fail if the upgrade was already executed
			t.Logf("%s upgrade may have already been executed or failed: %v", upgrade.Name, err)
		} else {
			t.Logf("Successfully executed %s upgrade using OPCM v%d at %s", upgrade.Name, upgrade.OpcmVersion, opcmAddr.Hex())
		}
	}
}

// buildV2UpgradeConfig builds an UpgradeOPChainInput for OPCM v2 upgrades
func buildV2UpgradeConfig(t *testing.T, prank, opcmAddr, systemConfigProxy common.Address, upgrade PastUpgradeConfig) embedded.UpgradeOPChainInput {
	t.Helper()

	bytes32Type := deployer.Bytes32Type
	addressType := deployer.AddressType

	// Build dispute game configs from parsed TOML config
	var disputeGameConfigs []embedded.DisputeGameConfig
	for _, dgc := range upgrade.DisputeGameConfigs {
		var gameArgs []byte
		var err error

		// Encode gameArgs based on game type
		if dgc.GameType == "PERMISSIONED_CANNON" || dgc.GameType == "SUPER_PERMISSIONED_CANNON" {
			// Permissioned games: prestate + proposer + challenger
			gameArgs, err = abi.Arguments{
				abi.Argument{Type: bytes32Type},
				abi.Argument{Type: addressType},
				abi.Argument{Type: addressType},
			}.Pack(dgc.Prestate, dgc.Proposer, dgc.Challenger)
			require.NoError(t, err, "Failed to encode %s args for %s", dgc.GameType, upgrade.Name)
		} else {
			// Non-permissioned games: just the prestate hash
			gameArgs, err = abi.Arguments{abi.Argument{Type: bytes32Type}}.Pack(dgc.Prestate)
			require.NoError(t, err, "Failed to encode %s args for %s", dgc.GameType, upgrade.Name)
		}

		// Convert game type string to embedded.GameType
		gameType := stringToGameType(dgc.GameType)

		disputeGameConfigs = append(disputeGameConfigs, embedded.DisputeGameConfig{
			Enabled:  dgc.Enabled,
			InitBond: dgc.InitBond,
			GameType: gameType,
			GameArgs: gameArgs,
		})
	}

	// Sort by game type (ascending numerical order, required by OPCM)
	sort.Slice(disputeGameConfigs, func(i, j int) bool {
		return disputeGameConfigs[i].GameType < disputeGameConfigs[j].GameType
	})

	// Build extra instructions
	var extraInstructions []embedded.ExtraInstruction
	for _, instr := range upgrade.ExtraInstructions {
		extraInstructions = append(extraInstructions, embedded.ExtraInstruction{
			Key:  instr.Key,
			Data: instr.Data,
		})
	}

	return embedded.UpgradeOPChainInput{
		Prank: prank,
		Opcm:  opcmAddr,
		UpgradeInputV2: &embedded.UpgradeInputV2{
			SystemConfig:       systemConfigProxy,
			DisputeGameConfigs: disputeGameConfigs,
			ExtraInstructions:  extraInstructions,
		},
	}
}

// stringToGameType converts a game type string to embedded.GameType
func stringToGameType(gameType string) embedded.GameType {
	switch gameType {
	case "CANNON":
		return embedded.GameTypeCannon // 0
	case "PERMISSIONED_CANNON":
		return embedded.GameTypePermissionedCannon // 1
	case "SUPER_CANNON":
		return embedded.GameType(4)
	case "SUPER_PERMISSIONED_CANNON":
		return embedded.GameType(5)
	case "CANNON_KONA":
		return embedded.GameTypeCannonKona // 8
	case "SUPER_CANNON_KONA":
		return embedded.GameType(9)
	default:
		panic(fmt.Sprintf("unknown game type: %s", gameType))
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
