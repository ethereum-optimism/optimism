package opcm

import (
	"context"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

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

	genesis := NewGenesisState(common.Address{0x5e}, false)
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

	genesis := NewGenesisState(common.Address{0x5e}, false)
	env := &ForgeEnv{Client: client, Context: ctx, Genesis: genesis}
	_, err := broadcastScript(env, genesisTestCaller("GenesisPrank"), 0)
	require.NoError(t, err)

	require.NotContains(t, genesis.Allocs.Accounts, common.Address{0x56, 0x78})
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
`
