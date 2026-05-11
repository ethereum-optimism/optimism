package karst

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

// callVersionString calls version() on a contract and returns the decoded string.
func callVersionString(t devtest.T, client apis.EthClient, addr common.Address) string {
	selector := crypto.Keccak256([]byte("version()"))[:4]
	raw, err := client.Call(t.Ctx(), ethereum.CallMsg{To: &addr, Data: selector}, rpc.LatestBlockNumber)
	t.Require().NoError(err, "version() call failed on %s", addr)
	stringType, _ := abi.NewType("string", "", nil)
	decoded, err := abi.Arguments{{Type: stringType}}.Unpack(raw)
	t.Require().NoError(err)
	return decoded[0].(string)
}

// loadInitcodeFromArtifact loads deployment bytecode from a forge artifact JSON file.
func loadInitcodeFromArtifact(t devtest.T, artifactPath string) []byte {
	data, err := os.ReadFile(artifactPath)
	t.Require().NoError(err, "failed to read artifact: %s", artifactPath)
	var artifact struct {
		Bytecode struct {
			Object string `json:"object"`
		} `json:"bytecode"`
	}
	t.Require().NoError(json.Unmarshal(data, &artifact))
	t.Require().NotEmpty(artifact.Bytecode.Object)
	return common.FromHex(artifact.Bytecode.Object)
}

// patchVersionInInitcode patches the version string inside initcode.
//
// Solidity stores a string constant via two instructions:
//   - PUSH1 <len>  — the string length, a few bytes before PUSH32
//   - PUSH32 <str left-aligned in 32 bytes>  — the string data
//
// We locate both by searching for the PUSH32 pattern (0x7f + currentVersion
// left-aligned in 32 zero-padded bytes) and scanning backwards for the
// length PUSH1. This makes the patch independent of the current version value.
func patchVersionInInitcode(t devtest.T, initcode []byte, currentVersion, newVersion string) []byte {
	t.Require().LessOrEqual(len([]byte(newVersion)), 32, "new version must fit in a PUSH32 (max 32 bytes)")

	currentVersionBytes := []byte(currentVersion)
	newVersionBytes := []byte(newVersion)

	currentPush32Arg := make([]byte, 32)
	copy(currentPush32Arg, currentVersionBytes)

	push32Pattern := append([]byte{0x7f}, currentPush32Arg...)
	idx := bytes.Index(initcode, push32Pattern)
	t.Require().GreaterOrEqual(idx, 0, "PUSH32 version pattern not found in initcode (current version: %q)", currentVersion)

	result := make([]byte, len(initcode))
	copy(result, initcode)

	newPush32Arg := make([]byte, 32)
	copy(newPush32Arg, newVersionBytes)
	copy(result[idx+1:idx+33], newPush32Arg)

	searchStart := idx - 30
	if searchStart < 0 {
		searchStart = 0
	}
	found := false
	for i := idx - 1; i >= searchStart; i-- {
		if result[i] == 0x60 && int(result[i+1]) == len(currentVersionBytes) {
			result[i+1] = byte(len(newVersionBytes))
			found = true
			break
		}
	}
	t.Require().True(found, "PUSH1 length byte not found near version PUSH32 (current version: %q)", currentVersion)

	return result
}

// buildNUTDepositTx creates a serialized deposit transaction suitable for use as a custom NUT.
func buildNUTDepositTx(t devtest.T, intent string, from common.Address, to *common.Address, data []byte, gasLimit uint64) hexutil.Bytes {
	source := derive.UpgradeDepositSource{Intent: intent}
	depTx := &types.DepositTx{
		SourceHash:          source.SourceHash(),
		From:                from,
		To:                  to,
		Mint:                big.NewInt(0),
		Value:               big.NewInt(0),
		Gas:                 gasLimit,
		IsSystemTransaction: false,
		Data:                data,
	}
	encoded, err := types.NewTx(depTx).MarshalBinary()
	t.Require().NoError(err, "failed to marshal NUT deposit tx")
	return encoded
}

// buildCustomNUTs constructs the three deposit transactions for the post-Karst upgrade:
//  1. ConditionalDeployer.deploy(salt, patchedSchemaRegistryInitcode)
//  2. ConditionalDeployer.deploy(salt2, minimalTestL2CMInitcode)
//  3. L2ProxyAdmin.upgradePredeploys(testL2CMAddr)
func buildCustomNUTs(
	t devtest.T,
	implSalt [32]byte, patchedInitcode []byte,
	l2CMSalt [32]byte, l2CMInitcode []byte,
	testL2CMAddr common.Address,
) ([]hexutil.Bytes, uint64) {
	depositor := common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001")
	cdAddr := predeploys.ConditionalDeployerAddr
	paAddr := predeploys.ProxyAdminAddr

	cdABI, err := abi.JSON(bytes.NewReader([]byte(
		`[{"inputs":[{"name":"_salt","type":"bytes32"},{"name":"_code","type":"bytes"}],"name":"deploy","outputs":[{"name":"implementation_","type":"address"}],"stateMutability":"nonpayable","type":"function"}]`,
	)))
	t.Require().NoError(err)

	paABI, err := abi.JSON(bytes.NewReader([]byte(
		`[{"inputs":[{"name":"_l2ContractsManager","type":"address"}],"name":"upgradePredeploys","outputs":[],"stateMutability":"nonpayable","type":"function"}]`,
	)))
	t.Require().NoError(err)

	deployImplData, err := cdABI.Pack("deploy", implSalt, patchedInitcode)
	t.Require().NoError(err)

	deployL2CMData, err := cdABI.Pack("deploy", l2CMSalt, l2CMInitcode)
	t.Require().NoError(err)

	upgradeData, err := paABI.Pack("upgradePredeploys", testL2CMAddr)
	t.Require().NoError(err)

	const implGas = uint64(3_000_000)
	const l2CMGas = uint64(500_000)
	const upgradeGas = uint64(300_000)

	tx1 := buildNUTDepositTx(t, "Custom NUT 0: Deploy Patched SchemaRegistry", depositor, &cdAddr, deployImplData, implGas)
	tx2 := buildNUTDepositTx(t, "Custom NUT 1: Deploy MinimalTestL2CM", depositor, &cdAddr, deployL2CMData, l2CMGas)
	tx3 := buildNUTDepositTx(t, "Custom NUT 2: Upgrade Predeploys", depositor, &paAddr, upgradeData, upgradeGas)

	return []hexutil.Bytes{tx1, tx2, tx3}, implGas + l2CMGas + upgradeGas
}

// TestL2CMUpgrade_Karst simulates a post-Karst upgrade on an already-Karst chain. It deploys a
// patched SchemaRegistry implementation via ConditionalDeployer, deploys a MinimalTestL2CM, and
// triggers L2ProxyAdmin.upgradePredeploys() via DEPOSITOR — exactly the production NUT flow —
// then verifies the upgrade is observable on L2 and covered by an L1 dispute game.
func TestL2CMUpgrade_Karst(gt *testing.T) {
	t := devtest.ParallelT(gt)

	wd, err := os.Getwd()
	t.Require().NoError(err)
	root, err := opservice.FindMonorepoRoot(wd)
	t.Require().NoError(err)

	// Load forge artifacts.
	schemaRegistryInitcode := loadInitcodeFromArtifact(t,
		filepath.Join(root, "packages", "contracts-bedrock", "forge-artifacts", "SchemaRegistry.sol", "SchemaRegistry.json"))
	minimalTestL2CMInitcode := loadInitcodeFromArtifact(t,
		filepath.Join(root, "packages", "contracts-bedrock", "forge-artifacts", "MinimalTestL2CM.sol", "MinimalTestL2CM.json"))

	// Pre-compute the patched SchemaRegistry implementation address.
	// The current version is patched to a recognisably high value so the upgrade is easily verified.
	// (patchVersionInInitcode will fail loudly if the version string is not found in the bytecode.)
	const newVersion = "999.999.999"

	var implSalt [32]byte
	copy(implSalt[:], []byte("karst-upgrade-test-impl-v1"))
	deterministicProxy := predeploys.DeterministicDeploymentProxyAddr

	var l2CMSalt [32]byte
	copy(l2CMSalt[:], []byte("karst-upgrade-test-l2cm-v1"))

	// Pre-compute L2CM address: it depends on the patched impl address (baked as an immutable),
	// but we need the impl address first. We will fill in the initcode after reading version().
	// Both addresses are deterministic from salts + initcodes, so they can be computed offline
	// once the version is known. We defer that to after system startup (see below).

	// Schedule the custom NUT activation 30 seconds after genesis.
	activation, nutOpt := sysgo.WithCustomNUTAtOffset(30)

	sys := presets.NewMinimal(t,
		presets.WithDeployerOptions(sysgo.WithKarstAtGenesis),
		presets.WithGameTypeAdded(gameTypes.CannonGameType),
		presets.WithRespectedGameTypeOverride(gameTypes.CannonGameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameTypes.CannonGameType)
		}),
		presets.WithGlobalL2CLOption(nutOpt),
	)

	// Read the current SchemaRegistry version from the running chain and build NUT transactions.
	currentVersion := callVersionString(t, sys.L2EL.EthClient(), predeploys.SchemaRegistryAddr)
	t.Logger().Info("Current SchemaRegistry version", "version", currentVersion)

	patchedInitcode := patchVersionInInitcode(t, schemaRegistryInitcode, currentVersion, newVersion)
	newImplAddr := crypto.CreateAddress2(deterministicProxy, implSalt, crypto.Keccak256(patchedInitcode))
	t.Logger().Info("Pre-computed patched SchemaRegistry implementation", "addr", newImplAddr)

	// Build MinimalTestL2CM initcode with constructor args (proxy, newImpl).
	addressType, _ := abi.NewType("address", "", nil)
	ctorArgs, err := abi.Arguments{{Type: addressType}, {Type: addressType}}.Pack(predeploys.SchemaRegistryAddr, newImplAddr)
	t.Require().NoError(err)
	l2CMInitcodeWithArgs := append(minimalTestL2CMInitcode, ctorArgs...)
	testL2CMAddr := crypto.CreateAddress2(deterministicProxy, l2CMSalt, crypto.Keccak256(l2CMInitcodeWithArgs))
	t.Logger().Info("Pre-computed MinimalTestL2CM", "addr", testL2CMAddr)

	// Register the NUT transactions before the activation block is sequenced.
	nutTxs, totalGas := buildCustomNUTs(t, implSalt, patchedInitcode, l2CMSalt, l2CMInitcodeWithArgs, testL2CMAddr)
	activation.SetTransactions(nutTxs, totalGas)
	t.Logger().Info("Registered custom NUT transactions", "activationTime", activation.Time, "txCount", len(nutTxs))

	// Wait for the activation block (poll until version() returns the new version).
	t.Require().Eventually(func() bool {
		ver := callVersionString(t, sys.L2EL.EthClient(), predeploys.SchemaRegistryAddr)
		return ver == newVersion
	}, 2*time.Minute, 2*time.Second, fmt.Sprintf("SchemaRegistry version should update to %s after custom NUT activation", newVersion))
	t.Logger().Info("Verified version() on L2", "version", newVersion)

	// Verify ERC-1967 implementation slot.
	implSlot, err := sys.L2EL.EthClient().GetStorageAt(
		t.Ctx(), predeploys.SchemaRegistryAddr, genesis.ImplementationSlot, "latest",
	)
	t.Require().NoError(err)
	t.Require().Equal(newImplAddr, common.BytesToAddress(implSlot[:]))

	// Retrieve the block number at which the upgrade landed.
	latestBlock, err := sys.L2EL.EthClient().InfoByLabel(t.Ctx(), eth.Unsafe)
	t.Require().NoError(err)
	upgradeBlockNumber := latestBlock.NumberU64()

	// Wait for an L1 dispute game covering the post-upgrade block.
	dgf := sys.DisputeGameFactory()
	initialGameCount := dgf.GameCount()
	t.Require().Eventually(func() bool {
		count := dgf.GameCount()
		for i := initialGameCount; i < count; i++ {
			if dgf.GameAtIndex(i).L2SequenceNumber() >= upgradeBlockNumber {
				return true
			}
		}
		return false
	}, 5*time.Minute, 5*time.Second, fmt.Sprintf("L1 must have a dispute game covering upgrade block %d", upgradeBlockNumber))
	t.Logger().Info("Verified L1 dispute game covers upgrade block", "block", upgradeBlockNumber)
}
