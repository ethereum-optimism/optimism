package opcm

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// offlineScriptChainID is the chain ID of the Go script host's default context, so that offline forge
// runs produce the same state. It matters for the L1 dev genesis, whose Permit2 preinstall has the
// chain ID built in; L2Genesis sets the L2 chain ID itself.
const offlineScriptChainID = 1337

// GenesisState is the L1 state of a genesis deployment. Each script runs offline in its own forge
// process: it loads this state, and the state it leaves replaces it.
type GenesisState struct {
	// Allocs holds the L1 accounts, the sender included.
	Allocs *foundry.ForgeAllocs
	// Sender sends the transactions of the scripts.
	Sender common.Address
	// NoCodeSizeLimit admits contracts over the EIP-170 limit, such as the lite profile's, and lets
	// the scripts run a build of such a profile.
	NoCodeSizeLimit bool
}

// NewGenesisState returns an empty state for a genesis deployment.
func NewGenesisState(sender common.Address, noCodeSizeLimit bool) *GenesisState {
	return &GenesisState{
		Allocs: &foundry.ForgeAllocs{Accounts: types.GenesisAlloc{
			// No code, so forge keeps its own CREATE2 deployer here. Nonce 0, not a contract's 1, keeps
			// the L1 dev genesis of earlier releases.
			predeploys.DeterministicDeploymentProxyAddr: {Balance: new(big.Int)},
		}},
		Sender:          sender,
		NoCodeSizeLimit: noCodeSizeLimit,
	}
}

// SetBalance sets the balance of an account.
func (g *GenesisState) SetBalance(addr common.Address, balance *big.Int) {
	acc := g.account(addr)
	acc.Balance = new(big.Int).Set(balance)
	g.Allocs.Accounts[addr] = acc
}

// Wipe removes the code, nonce and balance of an account.
func (g *GenesisState) Wipe(addr common.Address) {
	acc := g.account(addr)
	acc.Code = nil
	acc.Nonce = 0
	acc.Balance = new(big.Int)
	g.Allocs.Accounts[addr] = acc
	g.normalize()
}

// Create puts code at the sender's next CREATE address, as a deployment from the sender does. The
// code must need no init code: no constructor, immutables or initialized storage.
func (g *GenesisState) Create(code []byte) common.Address {
	sender := g.account(g.Sender)
	addr := crypto.CreateAddress(g.Sender, sender.Nonce)
	sender.Nonce++
	g.Allocs.Accounts[g.Sender] = sender
	g.Allocs.Accounts[addr] = types.Account{Code: code, Nonce: 1, Balance: new(big.Int)}
	return addr
}

func (g *GenesisState) account(addr common.Address) types.Account {
	acc, ok := g.Allocs.Accounts[addr]
	if !ok {
		return types.Account{Balance: new(big.Int)}
	}
	return acc
}

// run runs one script on the state through fn, and replaces this state with the one it leaves.
func (g *GenesisState) run(client *forge.Client, fn func(client *forge.Client, flags []string) error) error {
	// The state files stay out of the bundle, which may be read-only.
	dir, err := os.MkdirTemp("", "op-deployer-genesis-")
	if err != nil {
		return fmt.Errorf("failed to create the genesis state directory: %w", err)
	}
	defer os.RemoveAll(dir)

	stateIn := filepath.Join(dir, "in.json")
	stateOut := filepath.Join(dir, "out.json")
	senderOut := filepath.Join(dir, "sender.json")
	// Offline, forge starts the sender at nonce 1, so the state always holds the sender, to reset
	// its nonce.
	in := g.Allocs.Copy()
	in.Accounts[g.Sender] = g.account(g.Sender)
	if err := jsonutil.WriteJSON(in, ioutil.ToBasicFile(stateIn, 0o644)); err != nil {
		return fmt.Errorf("failed to write the genesis state: %w", err)
	}
	// forge isolates the top-level calls of a script unless the config turns it off, and the
	// contracts bundle turns it on for its tests. An isolated call runs as a transaction, so a call
	// that the script pranks bumps the nonce of the pranked address, such as the L2 depositor, and
	// leaves it in the state. The Go script host does not isolate calls.
	env := []string{
		"OP_DEPLOYER_STATE_IN=" + stateIn,
		"OP_DEPLOYER_STATE_OUT=" + stateOut,
		"OP_DEPLOYER_SENDER_OUT=" + senderOut,
		"FOUNDRY_ISOLATE=false",
	}
	flags := []string{
		"--sender", g.Sender.Hex(),
		"--chain", strconv.Itoa(offlineScriptChainID),
		"--block-number", "0",
		"--block-timestamp", "0",
	}
	if g.NoCodeSizeLimit {
		flags = append(flags, "--disable-code-size-limit")
	}
	runClient := client.WithEnv(env...)
	// An offline run deploys to no chain, so without a code size limit it may run a build of another
	// profile, such as a lite build.
	runClient.AnyCacheProfile = g.NoCodeSizeLimit
	// This replaces the project's fs_permissions, so client.FsPermissions must grant what the script
	// reads.
	runClient.FsPermissions = append(slices.Clone(client.FsPermissions), forge.FsPermission{Access: "read-write", Path: dir})
	if err := fn(runClient, flags); err != nil {
		return err
	}

	allocs, err := foundry.LoadForgeAllocs(stateOut)
	if err != nil {
		return fmt.Errorf("failed to read the genesis state that the script left: %w", err)
	}
	// forge leaves the sender out of the state dump, so the script writes it separately.
	sender, err := jsonutil.LoadJSON[struct {
		Nonce   string `json:"nonce"`
		Balance string `json:"balance"`
	}](senderOut)
	if err != nil {
		return fmt.Errorf("failed to read the sender that the script left: %w", err)
	}
	nonce, err := strconv.ParseUint(sender.Nonce, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid sender nonce %q: %w", sender.Nonce, err)
	}
	balance, ok := new(big.Int).SetString(sender.Balance, 10)
	if !ok {
		return fmt.Errorf("invalid sender balance %q", sender.Balance)
	}
	allocs.Accounts[g.Sender] = types.Account{Nonce: nonce, Balance: balance}
	g.Allocs = allocs
	g.normalize()
	return nil
}

// normalize drops zero storage slots and empty accounts, as the EVM does.
func (g *GenesisState) normalize() {
	for addr, acc := range g.Allocs.Accounts {
		for slot, value := range acc.Storage {
			if value == (common.Hash{}) {
				delete(acc.Storage, slot)
			}
		}
		if len(acc.Code) == 0 {
			acc.Code = nil
		}
		if len(acc.Storage) == 0 {
			acc.Storage = nil
		}
		if acc.Nonce == 0 && acc.Balance.Sign() == 0 && acc.Code == nil && acc.Storage == nil {
			delete(g.Allocs.Accounts, addr)
			continue
		}
		g.Allocs.Accounts[addr] = acc
	}
}
