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
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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

// PastDisputeGameConfig represents a dispute game config for V2 upgrades.
type PastDisputeGameConfig struct {
	GameType   string
	Enabled    bool
	Prestate   common.Hash
	InitBond   *big.Int
	Proposer   common.Address // PERMISSIONED only
	Challenger common.Address // PERMISSIONED only
}

// PastUpgradeConfig defines configuration for a past upgrade.
type PastUpgradeConfig struct {
	Name               string
	OpcmVersion        int // 1 or 2
	OpcmAddresses      map[uint64]common.Address
	CannonPrestate     common.Hash                   // V1 only
	CannonKonaPrestate common.Hash                   // V1 only
	DisputeGameConfigs []PastDisputeGameConfig       // V2 only
	ExtraInstructions  []PastUpgradeExtraInstruction // V2 only
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

// GetPastUpgrades returns the list of past upgrades loaded from past_upgrades.toml.
func GetPastUpgrades(t *testing.T) []PastUpgradeConfig {
	upgrades, err := loadPastUpgrades()
	require.NoError(t, err, "Failed to load past upgrades")
	return upgrades
}

// RunPastUpgrades executes all past OPCM upgrades in-memory only (no broadcast).
func RunPastUpgrades(t *testing.T, host *script.Host, chainID uint64, prank common.Address, systemConfigProxy common.Address) {
	t.Helper()
	for _, upgrade := range GetPastUpgrades(t) {
		runSingleUpgrade(t, host, chainID, prank, systemConfigProxy, upgrade)
	}
}

// runSingleUpgrade executes a single upgrade on the given host.
func runSingleUpgrade(t *testing.T, host *script.Host, chainID uint64, prank, systemConfigProxy common.Address, upgrade PastUpgradeConfig) bool {
	t.Helper()

	opcmAddr, ok := upgrade.OpcmAddresses[chainID]
	if !ok {
		return false
	}

	upgradeConfig := buildUpgradeConfig(t, prank, opcmAddr, systemConfigProxy, upgrade)
	if upgradeConfig == nil {
		return false
	}

	upgradeConfigBytes, err := json.Marshal(upgradeConfig)
	require.NoError(t, err)

	err = embedded.DefaultUpgrader.Upgrade(host, upgradeConfigBytes)
	if err != nil {
		t.Logf("%s upgrade failed (may already be applied): %v", upgrade.Name, err)
		return false
	}
	t.Logf("Successfully executed %s upgrade", upgrade.Name)
	return true
}

// buildUpgradeConfig builds the upgrade config for the given upgrade.
func buildUpgradeConfig(t *testing.T, prank, opcmAddr, systemConfigProxy common.Address, upgrade PastUpgradeConfig) *embedded.UpgradeOPChainInput {
	t.Helper()

	switch upgrade.OpcmVersion {
	case 1:
		return &embedded.UpgradeOPChainInput{
			Prank: prank,
			Opcm:  opcmAddr,
			ChainConfigs: []embedded.OPChainConfig{{
				SystemConfigProxy:  systemConfigProxy,
				CannonPrestate:     upgrade.CannonPrestate,
				CannonKonaPrestate: upgrade.CannonKonaPrestate,
			}},
		}
	case 2:
		cfg := buildV2UpgradeConfig(t, prank, opcmAddr, systemConfigProxy, upgrade)
		return &cfg
	default:
		return nil
	}
}

func buildV2UpgradeConfig(t *testing.T, prank, opcmAddr, systemConfigProxy common.Address, upgrade PastUpgradeConfig) embedded.UpgradeOPChainInput {
	t.Helper()

	var disputeGameConfigs []embedded.DisputeGameConfig
	for _, dgc := range upgrade.DisputeGameConfigs {
		var gameArgs []byte
		var err error

		if dgc.GameType == "PERMISSIONED_CANNON" || dgc.GameType == "SUPER_PERMISSIONED_CANNON" {
			gameArgs, err = abi.Arguments{
				{Type: deployer.Bytes32Type},
				{Type: deployer.AddressType},
				{Type: deployer.AddressType},
			}.Pack(dgc.Prestate, dgc.Proposer, dgc.Challenger)
		} else {
			gameArgs, err = abi.Arguments{{Type: deployer.Bytes32Type}}.Pack(dgc.Prestate)
		}
		require.NoError(t, err)

		disputeGameConfigs = append(disputeGameConfigs, embedded.DisputeGameConfig{
			Enabled:  dgc.Enabled,
			InitBond: dgc.InitBond,
			GameType: stringToGameType(dgc.GameType),
			GameArgs: gameArgs,
		})
	}

	sort.Slice(disputeGameConfigs, func(i, j int) bool {
		return disputeGameConfigs[i].GameType < disputeGameConfigs[j].GameType
	})

	var extraInstructions []embedded.ExtraInstruction
	for _, instr := range upgrade.ExtraInstructions {
		extraInstructions = append(extraInstructions, embedded.ExtraInstruction{Key: instr.Key, Data: instr.Data})
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

// RunPastUpgradesWithRPC runs past upgrades and broadcasts them using Anvil impersonation.
func RunPastUpgradesWithRPC(t *testing.T, l1RPCUrl string, afactsFS foundry.StatDirFs, lgr log.Logger, chainID uint64, prank common.Address, systemConfigProxy common.Address) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	rpcClient, err := rpc.Dial(l1RPCUrl)
	require.NoError(t, err)

	ethClient := ethclient.NewClient(rpcClient)

	err = rpcClient.Call(nil, "anvil_impersonateAccount", prank)
	require.NoError(t, err)
	defer func() { _ = rpcClient.Call(nil, "anvil_stopImpersonatingAccount", prank) }()

	// Fund prank address for gas
	err = rpcClient.Call(nil, "anvil_setBalance", prank, "0x56bc75e2d63100000") // 100 ETH
	require.NoError(t, err)

	networkChainID, err := ethClient.ChainID(ctx)
	require.NoError(t, err)

	// Process each upgrade: deploy DummyCaller with correct OPCM, run upgrade, broadcast
	for _, upgrade := range GetPastUpgrades(t) {
		opcmAddr, ok := upgrade.OpcmAddresses[chainID]
		if !ok {
			continue
		}

		// Deploy DummyCallerGeneric with this upgrade's OPCM address
		deployDummyCaller(t, rpcClient, afactsFS, prank, opcmAddr)

		// Create fresh broadcaster and host for this upgrade
		bcaster := NewImpersonationBroadcaster(lgr, ethClient, rpcClient, prank, networkChainID)
		host, err := env.DefaultForkedScriptHost(ctx, bcaster, lgr, prank, afactsFS, rpcClient)
		require.NoError(t, err)

		// Run the upgrade
		if !runSingleUpgrade(t, host, chainID, prank, systemConfigProxy, upgrade) {
			continue
		}

		// Broadcast this upgrade's transactions
		if _, err = bcaster.Broadcast(ctx); err != nil {
			t.Logf("Warning: %s broadcast failed: %v", upgrade.Name, err)
		} else {
			t.Logf("Successfully broadcast %s upgrade", upgrade.Name)
		}
	}
}

// deployDummyCaller deploys DummyCallerGeneric at the prank address with the given OPCM address.
func deployDummyCaller(t *testing.T, rpcClient *rpc.Client, afactsFS foundry.StatDirFs, prank, opcmAddr common.Address) {
	t.Helper()

	artifacts := &foundry.ArtifactsFS{FS: afactsFS}
	artifact, err := artifacts.ReadArtifact("UpgradeOPChain.s.sol", "DummyCallerGeneric")
	require.NoError(t, err, "failed to read DummyCallerGeneric artifact")

	err = rpcClient.Call(nil, "anvil_setCode", prank, hexutil.Encode(artifact.DeployedBytecode.Object))
	require.NoError(t, err, "failed to deploy DummyCallerGeneric")

	err = rpcClient.Call(nil, "anvil_setStorageAt", prank, common.Hash{}, common.BytesToHash(opcmAddr.Bytes()))
	require.NoError(t, err, "failed to set OPCM address in storage")
}

// ImpersonationBroadcaster broadcasts transactions using Anvil impersonation.
type ImpersonationBroadcaster struct {
	lgr       log.Logger
	client    *ethclient.Client
	rpcClient *rpc.Client
	from      common.Address
	chainID   *big.Int
	bcasts    []script.Broadcast
	mtx       sync.Mutex
}

func NewImpersonationBroadcaster(lgr log.Logger, client *ethclient.Client, rpcClient *rpc.Client, from common.Address, chainID *big.Int) *ImpersonationBroadcaster {
	return &ImpersonationBroadcaster{
		lgr:       lgr,
		client:    client,
		rpcClient: rpcClient,
		from:      from,
		chainID:   chainID,
	}
}

func (b *ImpersonationBroadcaster) Hook(bcast script.Broadcast) {
	b.mtx.Lock()
	b.bcasts = append(b.bcasts, bcast)
	b.mtx.Unlock()
}

func (b *ImpersonationBroadcaster) Broadcast(ctx context.Context) ([]broadcaster.BroadcastResult, error) {
	b.mtx.Lock()
	bcasts := b.bcasts
	b.bcasts = nil
	b.mtx.Unlock()

	if len(bcasts) == 0 {
		return nil, nil
	}

	results := make([]broadcaster.BroadcastResult, len(bcasts))
	for i, bcast := range bcasts {
		result := broadcaster.BroadcastResult{Broadcast: bcast}

		var to *common.Address
		if bcast.Type == script.BroadcastCall {
			to = &bcast.To
		}

		nonce, err := b.client.PendingNonceAt(ctx, b.from)
		if err != nil {
			result.Err = fmt.Errorf("failed to get nonce: %w", err)
			results[i] = result
			continue
		}

		gasPrice, err := b.client.SuggestGasPrice(ctx)
		if err != nil {
			result.Err = fmt.Errorf("failed to get gas price: %w", err)
			results[i] = result
			continue
		}

		value := ((*uint256.Int)(bcast.Value)).ToBig()

		// Estimate gas
		msg := ethereum.CallMsg{
			From:     b.from,
			To:       to,
			GasPrice: gasPrice,
			Value:    value,
			Data:     bcast.Input,
		}
		gasLimit, err := b.client.EstimateGas(ctx, msg)
		if err != nil {
			result.Err = fmt.Errorf("failed to estimate gas: %w", err)
			results[i] = result
			continue
		}

		gasLimit = gasLimit * 120 / 100 // buffer

		var txHash common.Hash
		err = b.rpcClient.CallContext(ctx, &txHash, "eth_sendTransaction", map[string]interface{}{
			"from":     b.from,
			"to":       to,
			"gas":      fmt.Sprintf("0x%x", gasLimit),
			"gasPrice": fmt.Sprintf("0x%x", gasPrice),
			"value":    fmt.Sprintf("0x%x", value),
			"data":     hexutil.Encode(bcast.Input),
			"nonce":    fmt.Sprintf("0x%x", nonce),
		})
		if err != nil {
			result.Err = fmt.Errorf("failed to send transaction: %w", err)
			results[i] = result
			continue
		}

		result.TxHash = txHash
		b.lgr.Info("transaction sent via impersonation", "hash", txHash.Hex(), "from", b.from.Hex(), "nonce", nonce)

		receipt, err := b.waitForReceipt(ctx, txHash)
		if err != nil {
			result.Err = fmt.Errorf("failed to wait for receipt: %w", err)
			results[i] = result
			continue
		}

		result.Receipt = receipt
		if receipt.Status == 0 {
			result.Err = fmt.Errorf("transaction failed: %s", txHash.Hex())
			b.lgr.Error("transaction failed on chain", "hash", txHash.Hex())
		} else {
			b.lgr.Info("transaction confirmed", "hash", txHash.Hex(), "gasUsed", receipt.GasUsed)
		}

		results[i] = result
	}

	var errCount int
	for _, r := range results {
		if r.Err != nil {
			errCount++
		}
	}
	if errCount > 0 {
		return results, fmt.Errorf("%d transactions failed", errCount)
	}
	return results, nil
}

func (b *ImpersonationBroadcaster) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	for {
		receipt, err := b.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		if err != ethereum.NotFound {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
