package genesis

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

// CheckL2AllocsOpts configures CheckL2GenesisAllocs.
type CheckL2AllocsOpts struct {
	// FundDevAccounts allows the DevAccounts to hold balances.
	FundDevAccounts bool
	// AllowedEOAs lists accounts allowed to hold a balance or nonce but never code,
	// e.g. interopgen prefunds or the proxy admin owner (nonce-bumped by older artifacts).
	AllowedEOAs []common.Address
}

var (
	// l2CodeNamespace is the namespace for L2 predeploy implementations.
	l2CodeNamespace = common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000")
	// createXAddr is the CreateX preinstall (Preinstalls.sol).
	createXAddr = common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed")
)

// namespaceByteLimit bounds byte 18: each namespace spans 2048 addresses (0x0000-0x07ff).
const namespaceByteLimit = 0x08

// CheckL2GenesisAllocs validates structural safety invariants of a freshly generated L2 genesis
// state dump: EIP-1967 proxy slot coherence, and every account classified into a known category.
// It is configuration-agnostic: it rejects unexpected accounts, not misconfigured storage values.
func CheckL2GenesisAllocs(allocs *foundry.ForgeAllocs, opts CheckL2AllocsOpts) error {
	var errs []error
	errs = append(errs, checkProxySlots(allocs)...)
	errs = append(errs, checkAccountCategories(allocs, opts)...)
	return errors.Join(errs...)
}

// checkProxySlots verifies EIP-1967 slot coherence: every non-zero admin slot points at the
// ProxyAdmin predeploy, and every non-zero implementation slot sits on a predeploy-namespace
// account and points at its code-namespace counterpart, which must have code.
func checkProxySlots(allocs *foundry.ForgeAllocs) []error {
	expectedAdmin := common.BytesToHash(predeploys.ProxyAdminAddr.Bytes())
	var errs []error
	adminSlots, implSlots := 0, 0

	for _, addr := range sortedAllocAddresses(allocs.Accounts) {
		account := allocs.Accounts[addr]

		if adminVal, ok := account.Storage[AdminSlot]; ok && adminVal != (common.Hash{}) {
			adminSlots++
			if adminVal != expectedAdmin {
				errs = append(errs, fmt.Errorf("account %s admin slot is %s, expected %s", addr, adminVal, expectedAdmin))
			}
		}

		implVal, ok := account.Storage[ImplementationSlot]
		if !ok || implVal == (common.Hash{}) {
			continue
		}
		implSlots++
		if !inNamespace(addr, l2PredeployNamespace) {
			errs = append(errs, fmt.Errorf("non-predeploy account %s has implementation slot value %s", addr, implVal))
			continue
		}
		impl := codeNamespaceCounterpart(addr)
		if common.BytesToAddress(implVal.Bytes()) != impl {
			errs = append(errs, fmt.Errorf("predeploy %s implementation slot points at %s, expected %s",
				addr, common.BytesToAddress(implVal.Bytes()), impl))
			continue
		}
		if implAccount, ok := allocs.Accounts[impl]; !ok || len(implAccount.Code) == 0 {
			errs = append(errs, fmt.Errorf("predeploy %s implementation %s has no code", addr, impl))
		}
	}

	if adminSlots == 0 {
		errs = append(errs, errors.New("no admin slots found in allocs"))
	}
	if implSlots == 0 {
		errs = append(errs, errors.New("no implementation slots found in allocs"))
	}
	return errs
}

// checkAccountCategories verifies that every account falls into a known category.
func checkAccountCategories(allocs *foundry.ForgeAllocs, opts CheckL2AllocsOpts) []error {
	devAccounts := make(map[common.Address]struct{})
	if opts.FundDevAccounts {
		for _, addr := range DevAccounts {
			devAccounts[addr] = struct{}{}
		}
	}
	allowedEOAs := make(map[common.Address]struct{})
	for _, addr := range opts.AllowedEOAs {
		allowedEOAs[addr] = struct{}{}
	}

	var errs []error
	for _, addr := range sortedAllocAddresses(allocs.Accounts) {
		if err := classifyAccount(addr, allocs.Accounts[addr], devAccounts, allowedEOAs); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func classifyAccount(
	addr common.Address,
	account types.Account,
	devAccounts map[common.Address]struct{},
	allowedEOAs map[common.Address]struct{},
) error {
	switch {
	case isPrecompile(addr):
		if balanceOf(account).Cmp(big.NewInt(1)) != 0 || len(account.Code) != 0 ||
			len(account.Storage) != 0 || account.Nonce != 0 {
			return fmt.Errorf("precompile %s must hold exactly 1 wei and nothing else: %s", addr, describeAccount(account))
		}
	case inNamespace(addr, l2PredeployNamespace):
		// Membership only: slot coherence is covered by checkProxySlots.
	case inNamespace(addr, l2CodeNamespace):
		if len(account.Code) == 0 {
			return fmt.Errorf("code-namespace account %s has no code: %s", addr, describeAccount(account))
		}
	case isNamedPreinstall(addr):
		if len(account.Code) == 0 {
			return fmt.Errorf("preinstall %s has no code: %s", addr, describeAccount(account))
		}
	case addr == predeploys.EIP4788ContractDeployer, addr == predeploys.EIP2935ContractDeployer:
		if account.Nonce != 1 || len(account.Code) != 0 {
			return fmt.Errorf("preinstall sender %s must be an EOA with nonce 1: %s", addr, describeAccount(account))
		}
	case isMember(addr, devAccounts), isMember(addr, allowedEOAs):
		if len(account.Code) != 0 {
			return fmt.Errorf("allowed EOA %s must not have code: %s", addr, describeAccount(account))
		}
	default:
		return fmt.Errorf("stray account %s: %s", addr, describeAccount(account))
	}
	return nil
}

// isNamedPreinstall returns true for preinstalls outside the predeploy namespace: ProxyDisabled
// registry entries, CreateX, and the EIP-4788/EIP-2935 system contracts. In-namespace
// ProxyDisabled entries (WETH, GovernanceToken) are classified by namespace membership first.
func isNamedPreinstall(addr common.Address) bool {
	if p, ok := predeploys.PredeploysByAddress[addr]; ok && p.ProxyDisabled {
		return true
	}
	return addr == createXAddr || addr == predeploys.EIP4788ContractAddr || addr == predeploys.EIP2935ContractAddr
}

// isPrecompile returns true for addresses below 0x100, all of which receive 1 wei at genesis.
func isPrecompile(addr common.Address) bool {
	for _, b := range addr[:19] {
		if b != 0 {
			return false
		}
	}
	return true
}

// inNamespace returns true if addr is one of the 2048 addresses starting at prefix.
func inNamespace(addr common.Address, prefix common.Address) bool {
	return bytes.Equal(addr[:18], prefix[:18]) && addr[18] < namespaceByteLimit
}

// codeNamespaceCounterpart maps a predeploy address to its code-namespace implementation.
func codeNamespaceCounterpart(addr common.Address) common.Address {
	out := l2CodeNamespace
	out[18] = addr[18]
	out[19] = addr[19]
	return out
}

func isMember(addr common.Address, set map[common.Address]struct{}) bool {
	_, ok := set[addr]
	return ok
}

func balanceOf(account types.Account) *big.Int {
	if account.Balance == nil {
		return new(big.Int)
	}
	return account.Balance
}

func describeAccount(account types.Account) string {
	return fmt.Sprintf("code length %d, nonce %d, balance %s, %d storage slots",
		len(account.Code), account.Nonce, balanceOf(account), len(account.Storage))
}

func sortedAllocAddresses(accounts types.GenesisAlloc) []common.Address {
	addrs := make([]common.Address, 0, len(accounts))
	for addr := range accounts {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
	})
	return addrs
}
