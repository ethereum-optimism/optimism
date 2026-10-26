package opcm

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
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

// genesisStateDir holds the state files of the script runs, relative to the forge project. The
// project's foundry.toml must give forge read-write access to it.
const genesisStateDir = ".op-deployer"

// genesisChainID is the chain ID that the scripts of a genesis deployment see. It is not the L1
// chain ID, to keep the L1 dev genesis that earlier op-deployer releases made: the Permit2 preinstall
// depends on it.
const genesisChainID = 1337

// GenesisState is the L1 state of a genesis deployment. op-deployer runs each deploy script offline,
// in its own forge process. The script loads this state, and the state that it leaves replaces this
// state.
type GenesisState struct {
	// Allocs holds the L1 accounts, the sender included.
	Allocs *foundry.ForgeAllocs
	// Sender sends the transactions of the scripts.
	Sender common.Address
	// NoCodeSizeLimit lets the scripts deploy contracts larger than the EIP-170 limit, such as the
	// unoptimized contracts of the lite profile.
	NoCodeSizeLimit bool
}

// NewGenesisState returns an empty state for a genesis deployment.
func NewGenesisState(sender common.Address, noCodeSizeLimit bool) *GenesisState {
	return &GenesisState{
		Allocs: &foundry.ForgeAllocs{Accounts: types.GenesisAlloc{
			// forge keeps its own deployer code at this address, because the account has no code. The
			// nonce starts at zero, not at the one of a deployed contract, to keep the L1 dev genesis
			// that earlier op-deployer releases made.
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

// Create puts a contract with the given runtime code at the next CREATE address of the sender, as a
// deployment from the sender does. The contract must have no constructor logic and no immutables.
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

// run runs one forge script on the state. fn runs the script with the given client and flags.
// When the script writes, the state that it leaves replaces this state.
func (g *GenesisState) run(client *forge.Client, writes bool, fn func(client *forge.Client, flags []string) error) error {
	parent := filepath.Join(client.Wd, genesisStateDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("failed to create the genesis state directory: %w", err)
	}
	dir, err := os.MkdirTemp(parent, "run-")
	if err != nil {
		return fmt.Errorf("failed to create a genesis state directory: %w", err)
	}
	defer os.RemoveAll(dir)

	stateIn := filepath.Join(dir, "in.json")
	stateOut := filepath.Join(dir, "out.json")
	senderOut := filepath.Join(dir, "sender.json")
	// forge deploys the script contract from the sender, so the state always sets the sender, to
	// undo the nonce that the script deployment used.
	in := g.Allocs.Copy()
	in.Accounts[g.Sender] = g.account(g.Sender)
	if err := jsonutil.WriteJSON(in, ioutil.ToBasicFile(stateIn, 0o644)); err != nil {
		return fmt.Errorf("failed to write the genesis state: %w", err)
	}
	env := []string{"OP_DEPLOYER_STATE_IN=" + stateIn}
	if writes {
		env = append(env, "OP_DEPLOYER_STATE_OUT="+stateOut, "OP_DEPLOYER_SENDER_OUT="+senderOut)
	}
	flags := []string{
		"--sender", g.Sender.Hex(),
		"--chain", strconv.Itoa(genesisChainID),
		"--block-number", "0",
		"--block-timestamp", "0",
	}
	if g.NoCodeSizeLimit {
		flags = append(flags, "--disable-code-size-limit")
	}
	if err := fn(client.WithEnv(env...), flags); err != nil {
		return err
	}
	if !writes {
		return nil
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

// normalize removes what the EVM state does not keep: zero storage slots and empty accounts.
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
