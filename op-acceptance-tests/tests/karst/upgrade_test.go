package karst

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

var conditionalDeployerAddr = common.HexToAddress("0x420000000000000000000000000000000000002C")

func loadSchemaRegistryInitcode(t devtest.T) []byte {
	wd, err := os.Getwd()
	t.Require().NoError(err)
	root, err := opservice.FindMonorepoRoot(wd)
	t.Require().NoError(err)

	artifactPath := filepath.Join(root, "packages", "contracts-bedrock", "forge-artifacts", "SchemaRegistry.sol", "SchemaRegistry.json")
	data, err := os.ReadFile(artifactPath)
	t.Require().NoError(err, "failed to read SchemaRegistry artifact")

	var artifact struct {
		Bytecode struct {
			Object string `json:"object"`
		} `json:"bytecode"`
	}
	t.Require().NoError(json.Unmarshal(data, &artifact))
	t.Require().NotEmpty(artifact.Bytecode.Object)
	return common.FromHex(artifact.Bytecode.Object)
}

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

	// Build the expected 32-byte PUSH32 argument for the current version.
	currentPush32Arg := make([]byte, 32)
	copy(currentPush32Arg, currentVersionBytes)

	push32Pattern := append([]byte{0x7f}, currentPush32Arg...)
	idx := bytes.Index(initcode, push32Pattern)
	t.Require().GreaterOrEqual(idx, 0, "PUSH32 version pattern not found in initcode (current version: %q)", currentVersion)

	result := make([]byte, len(initcode))
	copy(result, initcode)

	// Overwrite the 32-byte PUSH32 argument with the new version (left-aligned, zero-padded).
	newPush32Arg := make([]byte, 32)
	copy(newPush32Arg, newVersionBytes)
	copy(result[idx+1:idx+33], newPush32Arg)

	// Scan backwards from the PUSH32 to find the PUSH1 <len(currentVersionBytes)> instruction.
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

// TestL2CMUpgrade_Karst deploys a patched SchemaRegistry implementation via
// ConditionalDeployer, upgrades the proxy via L2ProxyAdmin, and verifies the
// change is observable both on L2 (version string, ERC-1967 slot) and on L1
// (dispute game covering the post-upgrade block).
func TestL2CMUpgrade_Karst(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t,
		presets.WithDeployerOptions(sysgo.WithKarstAtGenesis),
		presets.WithGameTypeAdded(gameTypes.CannonGameType),
		presets.WithRespectedGameTypeOverride(gameTypes.CannonGameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameTypes.CannonGameType)
		}),
	)

	devKeys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err)
	paOwnerKey := devkeys.L2ProxyAdminOwnerRole.Key(sys.L2Chain.ChainID().ToBig())
	privKey, err := devKeys.Secret(paOwnerKey)
	t.Require().NoError(err)
	l2PAO := dsl.NewEOA(dsl.NewKey(t, privKey), sys.L2EL)
	sys.FunderL2.Fund(l2PAO, eth.OneEther)

	currentVersion := callVersionString(t, sys.L2EL.EthClient(), predeploys.SchemaRegistryAddr)
	t.Logger().Info("Current SchemaRegistry version", "version", currentVersion)

	const newVersion = "999.999.999"
	initcode := loadSchemaRegistryInitcode(t)
	initcode = patchVersionInInitcode(t, initcode, currentVersion, newVersion)

	var salt [32]byte
	copy(salt[:], []byte("karst-upgrade-test-v1"))

	deterministicProxy := predeploys.DeterministicDeploymentProxyAddr
	newImplAddr := crypto.CreateAddress2(deterministicProxy, salt, crypto.Keccak256(initcode))

	cdABI, err := abi.JSON(bytes.NewReader([]byte(
		`[{"inputs":[{"name":"_salt","type":"bytes32"},{"name":"_code","type":"bytes"}],"name":"deploy","outputs":[{"name":"implementation_","type":"address"}],"stateMutability":"nonpayable","type":"function"}]`,
	)))
	t.Require().NoError(err)
	deployCalldata, err := cdABI.Pack("deploy", salt, initcode)
	t.Require().NoError(err)

	cdAddr := conditionalDeployerAddr
	l2PAO.Transact(l2PAO.Plan(), txplan.WithTo(&cdAddr), txplan.WithData(deployCalldata))
	t.Logger().Info("Deployed patched SchemaRegistry implementation", "impl", newImplAddr)

	paABI, err := abi.JSON(bytes.NewReader([]byte(
		`[{"inputs":[{"name":"_proxy","type":"address"},{"name":"_implementation","type":"address"}],"name":"upgrade","outputs":[],"stateMutability":"nonpayable","type":"function"}]`,
	)))
	t.Require().NoError(err)
	upgradeCalldata, err := paABI.Pack("upgrade", predeploys.SchemaRegistryAddr, newImplAddr)
	t.Require().NoError(err)

	proxyAdminAddr := predeploys.ProxyAdminAddr
	upgradeTx := l2PAO.Transact(l2PAO.Plan(), txplan.WithTo(&proxyAdminAddr), txplan.WithData(upgradeCalldata))
	upgradeReceipt, err := upgradeTx.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	upgradeBlockNumber := upgradeReceipt.BlockNumber.Uint64()
	t.Logger().Info("Upgraded SchemaRegistry proxy", "block", upgradeBlockNumber, "newImpl", newImplAddr)

	// Verify ERC-1967 slot and version() on L2.
	implSlot, err := sys.L2EL.EthClient().GetStorageAt(
		t.Ctx(), predeploys.SchemaRegistryAddr, genesis.ImplementationSlot, "latest",
	)
	t.Require().NoError(err)
	t.Require().Equal(newImplAddr, common.BytesToAddress(implSlot[:]))
	t.Require().Equal(newVersion, callVersionString(t, sys.L2EL.EthClient(), predeploys.SchemaRegistryAddr))
	t.Logger().Info("Verified version() on L2", "version", newVersion)

	// Wait for a dispute game on L1 that covers the post-upgrade block.
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
