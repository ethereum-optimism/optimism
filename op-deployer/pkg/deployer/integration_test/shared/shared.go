package shared

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/opcmregistry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum"
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
		L1ChainID:  bigs.Uint64Strict(l1ChainID),
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

// lastUsedOPCMVersionSelector is the selector for SystemConfig.lastUsedOPCMVersion()
// keccak256("lastUsedOPCMVersion()")[:4] = 0x9fabcc84
var lastUsedOPCMVersionSelector = []byte{0x9f, 0xab, 0xcc, 0x84}

// versionSelector is the selector for ISemver.version()
// keccak256("version()")[:4] = 0x54fd4d50
var versionSelector = []byte{0x54, 0xfd, 0x4d, 0x50}

// ContractCaller abstracts contract calls for both script.Host and ethclient.Client
type ContractCaller interface {
	Call(to common.Address, data []byte) ([]byte, error)
}

// HostCaller adapts script.Host to ContractCaller
type HostCaller struct {
	Host *script.Host
}

func (h *HostCaller) Call(to common.Address, data []byte) ([]byte, error) {
	result, _, err := h.Host.Call(
		common.Address{19: 0x01}, // dummy caller
		to,
		data,
		1_000_000,
		uint256.NewInt(0),
	)
	return result, err
}

// RPCCaller adapts ethclient.Client to ContractCaller
type RPCCaller struct {
	Ctx    context.Context
	Client *ethclient.Client
}

func (r *RPCCaller) Call(to common.Address, data []byte) ([]byte, error) {
	msg := ethereum.CallMsg{To: &to, Data: data}
	return r.Client.CallContract(r.Ctx, msg, nil)
}

// isEVMRevert checks if an error is an EVM execution revert
func isEVMRevert(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "execution reverted") || strings.Contains(errStr, "revert")
}

// decodeABIString decodes an ABI-encoded string from call response data.
func decodeABIString(data []byte) (string, error) {
	if len(data) < 64 {
		return "", fmt.Errorf("invalid response length: %d", len(data))
	}

	length := new(big.Int).SetBytes(data[32:64]).Int64()
	stringStart := int64(64)
	stringEnd := stringStart + length

	if stringEnd > int64(len(data)) {
		return "", fmt.Errorf("malformed response")
	}

	return string(data[stringStart:stringEnd]), nil
}

// getOPCMVersion queries the OPCM contract's version() function.
func getOPCMVersion(caller ContractCaller, opcmAddr common.Address) (string, error) {
	data, err := caller.Call(opcmAddr, versionSelector)
	if err != nil {
		return "", fmt.Errorf("failed to call version(): %w", err)
	}
	return decodeABIString(data)
}

// getLastUsedOPCMVersion queries SystemConfig.lastUsedOPCMVersion() and returns the version string.
// Returns (version, true, nil) on success, ("", false, nil) if call reverted (pre-6.x.x chain).
func getLastUsedOPCMVersion(caller ContractCaller, systemConfigProxy common.Address) (string, bool, error) {
	data, err := caller.Call(systemConfigProxy, lastUsedOPCMVersionSelector)
	if err != nil {
		if isEVMRevert(err) {
			return "", false, nil
		}
		return "", false, err
	}
	version, err := decodeABIString(data)
	if err != nil {
		return "", true, nil // call succeeded but response malformed
	}
	return version, true, nil
}

// runSingleOPCMUpgradeResolved executes a single OPCM upgrade on the given host.
func runSingleOPCMUpgradeResolved(t *testing.T, host *script.Host, prank, systemConfigProxy common.Address, opcm opcmregistry.ResolvedOPCM) bool {
	t.Helper()

	upgradeConfig := buildOPCMUpgradeConfig(t, prank, opcm.Address, systemConfigProxy, opcm.OPCMVersion)
	if upgradeConfig == nil {
		return false
	}

	upgradeConfigBytes, err := json.Marshal(upgradeConfig)
	require.NoError(t, err)

	err = embedded.DefaultUpgrader.Upgrade(host, upgradeConfigBytes)
	if err != nil {
		t.Logf("OPCM %s (v%s) upgrade failed: %v", opcm.Address.Hex(), opcm.OPCMVersion.Raw, err)
		return false
	}
	t.Logf("Successfully executed OPCM %s (v%s) upgrade", opcm.Address.Hex(), opcm.OPCMVersion.Raw)
	return true
}

// buildOPCMUpgradeConfig builds the upgrade config for the given OPCM.
func buildOPCMUpgradeConfig(t *testing.T, prank, opcmAddr, systemConfigProxy common.Address, version opcmregistry.Semver) *embedded.UpgradeOPChainInput {
	t.Helper()

	if version.IsV1OPCM() {
		// V1 OPCM (6.x.x) - uses ChainConfigs with prestates
		return &embedded.UpgradeOPChainInput{
			Prank: prank,
			Opcm:  opcmAddr,
			ChainConfigs: []embedded.OPChainConfig{{
				SystemConfigProxy:  systemConfigProxy,
				CannonPrestate:     opcmregistry.DummyCannonPrestate,
				CannonKonaPrestate: opcmregistry.DummyCannonKonaPrestate,
			}},
		}
	}

	// V2 OPCM (7.x.x+) - uses UpgradeInputV2 with dispute game configs
	cfg := buildV2OPCMUpgradeConfig(t, prank, opcmAddr, systemConfigProxy)
	return &cfg
}

// buildV2OPCMUpgradeConfig builds a V2 upgrade config with dummy dispute game configs
func buildV2OPCMUpgradeConfig(t *testing.T, prank, opcmAddr, systemConfigProxy common.Address) embedded.UpgradeOPChainInput {
	t.Helper()

	// Build dispute game configs with dummy prestates
	// CANNON and PERMISSIONED_CANNON are the standard game types
	disputeGameConfigs := []embedded.DisputeGameConfig{
		{
			Enabled:  true,
			InitBond: big.NewInt(0),
			GameType: embedded.GameTypeCannon,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{
				AbsolutePrestate: opcmregistry.DummyCannonPrestate,
			},
		},
		{
			Enabled:  true,
			InitBond: big.NewInt(0),
			GameType: embedded.GameTypePermissionedCannon,
			PermissionedDisputeGameConfig: &embedded.PermissionedDisputeGameConfig{
				AbsolutePrestate: opcmregistry.DummyCannonPrestate,
				Proposer:         common.Address{},
				Challenger:       common.Address{},
			},
		},
		{
			Enabled:  true,
			InitBond: big.NewInt(0),
			GameType: embedded.GameTypeCannonKona,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{
				AbsolutePrestate: opcmregistry.DummyCannonKonaPrestate,
			},
		},
	}

	// Sort by game type (required by OPCM)
	sort.Slice(disputeGameConfigs, func(i, j int) bool {
		return disputeGameConfigs[i].GameType < disputeGameConfigs[j].GameType
	})

	return embedded.UpgradeOPChainInput{
		Prank: prank,
		Opcm:  opcmAddr,
		UpgradeInputV2: &embedded.UpgradeInputV2{
			SystemConfig:       systemConfigProxy,
			DisputeGameConfigs: disputeGameConfigs,
			ExtraInstructions:  nil,
		},
	}
}

// DeployDummyCaller deploys DummyCaller at the prank address with the given OPCM address.
func DeployDummyCaller(t *testing.T, rpcClient *rpc.Client, afactsFS foundry.StatDirFs, prank, opcmAddr common.Address) {
	t.Helper()

	artifacts := &foundry.ArtifactsFS{FS: afactsFS}
	artifact, err := artifacts.ReadArtifact("DummyCaller.sol", "DummyCaller.0.8.15")
	require.NoError(t, err, "failed to read DummyCaller artifact")

	err = rpcClient.Call(nil, "anvil_setCode", prank, hexutil.Encode(artifact.DeployedBytecode.Object))
	require.NoError(t, err, "failed to deploy DummyCaller")

	err = rpcClient.Call(nil, "anvil_setStorageAt", prank, common.Hash{}, common.BytesToHash(opcmAddr.Bytes()))
	require.NoError(t, err, "failed to set OPCM address in storage")
}

// RunPastUpgrades executes all past OPCM upgrades in-memory only (no broadcast).
// It fetches OPCM addresses from the superchain-registry, queries their actual versions on-chain,
// and applies upgrades in version order, skipping any that have already been applied.
func RunPastUpgrades(t *testing.T, host *script.Host, chainID uint64, prank common.Address, systemConfigProxy common.Address) {
	t.Helper()

	caller := &HostCaller{Host: host}

	// Create version querier that uses the host to query on-chain
	queryVersion := func(addr common.Address) (string, error) {
		return getOPCMVersion(caller, addr)
	}

	// Get resolved OPCMs (fetches from registry, queries versions on-chain, filters >= 6.x.x)
	resolved, err := opcmregistry.GetResolvedOPCMs(chainID, queryVersion)
	if err != nil {
		t.Logf("Failed to get resolved OPCMs: %v", err)
		return
	}

	if len(resolved) == 0 {
		t.Logf("No OPCMs >= 6.x.x found for chain %d", chainID)
		return
	}

	// Query the current lastUsedOPCMVersion from SystemConfig
	lastVersion, hasLastVersion, err := getLastUsedOPCMVersion(caller, systemConfigProxy)
	if err != nil {
		t.Fatalf("Failed to query lastUsedOPCMVersion: %v", err)
	}
	if !hasLastVersion {
		t.Logf("SystemConfig.lastUsedOPCMVersion() reverted - chain is pre-6.x.x, will apply all upgrades from 6.x.x")
		lastVersion = ""
	} else {
		t.Logf("SystemConfig.lastUsedOPCMVersion() = %s", lastVersion)
	}

	// Filter to only include OPCMs with version > lastUsedOPCMVersion
	toApply, err := opcmregistry.FilterByLastUsedOPCMVersion(resolved, lastVersion)
	if err != nil {
		t.Logf("Warning: failed to filter by lastUsedOPCMVersion: %v", err)
		toApply = resolved
	}

	for _, opcm := range toApply {
		runSingleOPCMUpgradeResolved(t, host, prank, systemConfigProxy, opcm)
	}
}

// RunPastUpgradesWithRPC runs past upgrades and broadcasts them using Anvil impersonation.
// It fetches OPCM data from the superchain-registry and applies upgrades in version order,
// skipping any upgrades that have already been applied based on SystemConfig.lastUsedOPCMVersion().
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

	caller := &RPCCaller{Ctx: ctx, Client: ethClient}

	// Create version querier that uses RPC to query on-chain
	queryVersion := func(addr common.Address) (string, error) {
		return getOPCMVersion(caller, addr)
	}

	// Get resolved OPCMs (fetches from registry, queries versions on-chain, filters >= 6.x.x)
	resolved, err := opcmregistry.GetResolvedOPCMs(chainID, queryVersion)
	if err != nil {
		t.Logf("Failed to get resolved OPCMs: %v", err)
		return
	}

	if len(resolved) == 0 {
		t.Logf("No OPCMs >= 6.x.x found for chain %d", chainID)
		return
	}

	// Query the current lastUsedOPCMVersion from SystemConfig
	lastVersion, hasLastVersion, err := getLastUsedOPCMVersion(caller, systemConfigProxy)
	if err != nil {
		t.Fatalf("Failed to query lastUsedOPCMVersion: %v", err)
	}
	if !hasLastVersion {
		t.Logf("SystemConfig.lastUsedOPCMVersion() reverted - chain is pre-6.x.x, will apply all upgrades from 6.x.x")
		lastVersion = ""
	} else {
		t.Logf("SystemConfig.lastUsedOPCMVersion() = %s", lastVersion)
	}

	// Filter to only include OPCMs with version > lastUsedOPCMVersion
	toApply, err := opcmregistry.FilterByLastUsedOPCMVersion(resolved, lastVersion)
	if err != nil {
		t.Logf("Warning: failed to filter by lastUsedOPCMVersion: %v", err)
		toApply = resolved
	}

	// Process each OPCM upgrade: deploy DummyCaller with correct OPCM, run upgrade, broadcast
	for _, opcm := range toApply {
		// Deploy DummyCaller with this OPCM's address
		DeployDummyCaller(t, rpcClient, afactsFS, prank, opcm.Address)

		// Create fresh broadcaster and host for this upgrade
		bcaster := NewImpersonationBroadcaster(lgr, ethClient, rpcClient, prank, networkChainID)
		host, err := env.DefaultForkedScriptHost(ctx, bcaster, lgr, prank, afactsFS, rpcClient)
		require.NoError(t, err)

		// Run the upgrade
		if !runSingleOPCMUpgradeResolved(t, host, prank, systemConfigProxy, opcm) {
			continue
		}

		// Broadcast this upgrade's transactions
		if _, err = bcaster.Broadcast(ctx); err != nil {
			t.Logf("Warning: OPCM %s (v%s) broadcast failed: %v", opcm.Address.Hex(), opcm.OPCMVersion.Raw, err)
		} else {
			t.Logf("Successfully broadcast OPCM %s (v%s) upgrade", opcm.Address.Hex(), opcm.OPCMVersion.Raw)
		}
	}
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
