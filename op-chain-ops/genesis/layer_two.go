package genesis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
)

type L2AllocsMode string

const (
	L2AllocsDelta   L2AllocsMode = "delta"
	L2AllocsEcotone L2AllocsMode = "" // the default in solidity scripting / testing
)

type AllocsLoader func(mode L2AllocsMode) *ForgeAllocs

// BuildL2Genesis will build the L2 genesis block.
func BuildL2Genesis(config *DeployConfig, dump *ForgeAllocs, l1StartBlock *types.Block) (*core.Genesis, error) {
	genspec, err := NewL2Genesis(config, l1StartBlock)
	if err != nil {
		return nil, err
	}
	genspec.Alloc = dump.Accounts
	// ensure the dev accounts are not funded unintentionally
	if hasDevAccounts, err := HasAnyDevAccounts(dump.Accounts); err != nil {
		return nil, fmt.Errorf("failed to check dev accounts: %w", err)
	} else if hasDevAccounts != config.FundDevAccounts {
		return nil, fmt.Errorf("deploy config mismatch with allocs. Deploy config fundDevAccounts: %v, actual allocs: %v", config.FundDevAccounts, hasDevAccounts)
	}
	// sanity check the permit2 immutable, to verify we using the allocs for the right chain.
	if permit2 := genspec.Alloc[predeploys.Permit2Addr].Code; len(permit2) != 0 {
		if len(permit2) < 6945+32 {
			return nil, fmt.Errorf("permit2 code is too short")
		}
		chainID := [32]byte(permit2[6945 : 6945+32])
		expected := uint256.MustFromBig(genspec.Config.ChainID).Bytes32()
		if chainID != expected {
			return nil, fmt.Errorf("allocs were generated for chain ID %x, but expected chain %x (%d)", chainID, expected, genspec.Config.ChainID)
		}
	}
	return genspec, nil
}

var testMnemonic = "test test test test test test test test test test test junk"

func HasAnyDevAccounts(allocs core.GenesisAlloc) (bool, error) {
	wallet, err := hdwallet.NewFromMnemonic(testMnemonic)
	if err != nil {
		return false, fmt.Errorf("failed to create wallet: %w", err)
	}
	account := func(path string) accounts.Account {
		return accounts.Account{URL: accounts.URL{Path: path}}
	}
	for i := 0; i < 30; i++ {
		key, err := wallet.PrivateKey(account(fmt.Sprintf("m/44'/60'/0'/0/%d", i)))
		if err != nil {
			return false, err
		}
		addr := crypto.PubkeyToAddress(key.PublicKey)
		if _, ok := allocs[addr]; ok {
			return true, nil
		}
	}
	return false, nil
}

func LoadForgeAllocs(allocsPath string) (*ForgeAllocs, error) {
	path := filepath.Join(allocsPath)
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open forge allocs %q: %w", path, err)
	}
	defer f.Close()
	var out ForgeAllocs
	if err := json.NewDecoder(f).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to json-decode forge allocs %q: %w", path, err)
	}
	return &out, nil
}
