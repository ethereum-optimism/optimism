package opcm

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// testChainID is the chain ID of the genesis states of the tests.
const testChainID = 900

// TestGenesisStateSetBalanceZero checks that clearing a balance drops only empty accounts.
func TestGenesisStateSetBalanceZero(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		account *types.Account
		keep    bool
	}{
		{name: "absent"},
		{name: "balance only", account: &types.Account{Balance: big.NewInt(42)}},
		{name: "nonce", account: &types.Account{Balance: big.NewInt(42), Nonce: 1}, keep: true},
		{name: "code", account: &types.Account{Balance: big.NewInt(42), Code: []byte{0x00}}, keep: true},
		{name: "storage", account: &types.Account{
			Balance: big.NewInt(42),
			Storage: map[common.Hash]common.Hash{{0x01}: {0x02}},
		}, keep: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			addr := common.Address{0xaa}
			genesis := NewGenesisState(common.Address{0x5e}, testChainID, false)
			if tt.account != nil {
				genesis.Allocs.Accounts[addr] = *tt.account
			}
			genesis.SetBalance(addr, new(big.Int))
			account, ok := genesis.Allocs.Accounts[addr]
			require.Equal(t, tt.keep, ok)
			if tt.keep {
				expected := *tt.account
				expected.Balance = new(big.Int)
				require.Equal(t, expected, account)
			}
		})
	}
}

// TestGenesisStateReadOnlyBundle runs a genesis script on forge in a read-only bundle: the state
// files must stay out of the bundle, and forge must still be allowed to write them.
func TestGenesisStateReadOnlyBundle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := newGenesisTestClient(ctx, t, "")
	project := client.Wd

	before := listTree(t, project)
	t.Cleanup(func() { setTreeWritable(t, project, true) })
	setTreeWritable(t, project, false)

	genesis := NewGenesisState(common.Address{0x5e}, testChainID, false)
	env := &ForgeEnv{Client: client, Context: ctx, Genesis: genesis}
	_, err := broadcastScript(env, genesisTestCaller("Genesis"), 0)
	require.NoError(t, err)

	require.Equal(t, big.NewInt(5), genesis.Allocs.Accounts[common.Address{0x12, 0x34}].Balance)
	require.Equal(t, before, listTree(t, project), "the run wrote into the bundle")
}

// TestGenesisStateWithoutIsolation runs a genesis script in a bundle that turns on forge's call
// isolation, as the contracts bundle does for its tests. A pranked top-level call must not leave
// the pranked address in the state, as the genesis of earlier releases did not.
func TestGenesisStateWithoutIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := newGenesisTestClient(ctx, t, "isolate = true\n")

	genesis := NewGenesisState(common.Address{0x5e}, testChainID, false)
	env := &ForgeEnv{Client: client, Context: ctx, Genesis: genesis}
	_, err := broadcastScript(env, genesisTestCaller("GenesisPrank"), 0)
	require.NoError(t, err)

	require.NotContains(t, genesis.Allocs.Accounts, common.Address{0x56, 0x78})
}

// TestGenesisStateCreate2Deployer runs a genesis script that deploys a contract through forge's
// CREATE2 deployer. The deployer is a contract, so its nonce starts at 1, and the deployment bumps
// it to 2.
func TestGenesisStateCreate2Deployer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := newGenesisTestClient(ctx, t, "")

	genesis := NewGenesisState(common.Address{0x5e}, testChainID, false)
	env := &ForgeEnv{Client: client, Context: ctx, Genesis: genesis}
	_, err := broadcastScript(env, genesisTestCaller("GenesisCreate2"), 0)
	require.NoError(t, err)

	create2Deployer, ok := genesis.Allocs.Accounts[predeploys.DeterministicDeploymentProxyAddr]
	require.True(t, ok, "the CREATE2 deployer must be in the state")
	require.NotEmpty(t, create2Deployer.Code)
	require.Equal(t, uint64(2), create2Deployer.Nonce)
}

// TestGenesisStateRunAnyCacheProfile checks that a genesis run without a code size limit may run a
// contracts build of another profile, such as a lite build, and that a run with the limit may not.
func TestGenesisStateRunAnyCacheProfile(t *testing.T) {
	for _, noCodeSizeLimit := range []bool{false, true} {
		genesis := NewGenesisState(common.Address{0x5e}, testChainID, noCodeSizeLimit)
		errStop := errors.New("stop")
		var anyCacheProfile bool
		err := genesis.run(&forge.Client{}, func(client *forge.Client, _ []string) error {
			anyCacheProfile = client.AnyCacheProfile
			return errStop
		})
		require.ErrorIs(t, err, errStop)
		require.Equal(t, noCodeSizeLimit, anyCacheProfile, "noCodeSizeLimit %v", noCodeSizeLimit)
	}
}

// newGenesisTestClient builds a forge project with the test scripts and config appended to its
// default profile.
func newGenesisTestClient(ctx context.Context, t *testing.T, config string) *forge.Client {
	t.Helper()
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "foundry.toml"), []byte("[profile.default]\nsolc_version = '0.8.15'\n"+config), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(project, "script"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "script", "Genesis.s.sol"), []byte(genesisTestScript), 0o644))
	client, err := forge.NewStandardClient(project)
	require.NoError(t, err)
	client.Stdout, client.Stderr = io.Discard, io.Discard
	require.NoError(t, client.Build(ctx))
	client.FsPermissions = []forge.FsPermission{{Access: "read", Path: "./out/"}}
	return client
}

// genesisTestCaller calls runWithBytes of a contract of the test scripts.
func genesisTestCaller(contract string) newScriptCaller[byte, forge.None] {
	return func(c *forge.Client) forge.ScriptCaller[byte, forge.None] {
		return func(ctx context.Context, input byte, opts ...string) (forge.None, []forge.Transaction, error) {
			_, txs, err := c.RunScript(ctx, "script/Genesis.s.sol:"+contract, "runWithBytes(bytes)", []byte{input}, opts...)
			return forge.None{}, txs, err
		}
	}
}

// listTree returns the paths under dir.
func listTree(t *testing.T, dir string) []string {
	t.Helper()
	var paths []string
	require.NoError(t, filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		paths = append(paths, path)
		return err
	}))
	return paths
}

// setTreeWritable adds or removes the write permission of every directory under dir. Root ignores
// it, so the test also compares the tree before and after the run.
func setTreeWritable(t *testing.T, dir string, writable bool) {
	t.Helper()
	require.NoError(t, filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		mode := os.FileMode(0o555)
		if writable {
			mode = 0o755
		}
		return os.Chmod(path, mode)
	}))
}

const genesisTestScript = `// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;
interface Vm {
    function envString(string calldata name) external view returns (string memory);
    function deal(address account, uint256 newBalance) external;
    function dumpState(string calldata path) external;
    function writeFile(string calldata path, string calldata data) external;
    function toString(uint256 value) external pure returns (string memory);
    function getNonce(address account) external view returns (uint64);
    function prank(address sender) external;
    function loadAllocs(string calldata path) external;
    function broadcast() external;
}
contract Target {
    function poke() external {}
}
contract GenesisBase {
    Vm constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));
    function dump() internal {
        vm.dumpState(vm.envString("OP_DEPLOYER_STATE_OUT"));
        vm.writeFile(vm.envString("OP_DEPLOYER_SENDER_OUT"), string.concat(
            '{"nonce":"', vm.toString(uint256(vm.getNonce(msg.sender))),
            '","balance":"', vm.toString(msg.sender.balance), '"}'
        ));
    }
}
contract Genesis is GenesisBase {
    function runWithBytes(bytes calldata) external returns (bytes memory) {
        vm.deal(address(0x1234000000000000000000000000000000000000), 5);
        dump();
        return "";
    }
}
contract GenesisPrank is GenesisBase {
    function runWithBytes(bytes calldata) external returns (bytes memory) {
        Target target = new Target();
        vm.prank(address(0x5678000000000000000000000000000000000000));
        target.poke();
        dump();
        return "";
    }
}
contract GenesisCreate2 is GenesisBase {
    function runWithBytes(bytes calldata) external returns (bytes memory) {
        vm.loadAllocs(vm.envString("OP_DEPLOYER_STATE_IN"));
        vm.broadcast();
        new Target{salt: bytes32(0)}();
        dump();
        return "";
    }
}
`
