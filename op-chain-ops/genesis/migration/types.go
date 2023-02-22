package migration

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SentMessageJSON represents an entry in the JSON file that is created by
// the `migration-data` package. Each entry represents a call to the
// `LegacyMessagePasser`. The `who` should always be the
// `L2CrossDomainMessenger` and the `msg` should be an abi encoded
// `relayMessage(address,address,bytes,uint256)`
type SentMessage struct {
	Who common.Address `json:"who"`
	Msg hexutil.Bytes  `json:"msg"`
}

// NewSentMessageJSON will read a JSON file from disk given a path to the JSON
// file. The JSON file this function reads from disk is an output from the
// `migration-data` package.
func NewSentMessage(path string) ([]*SentMessage, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot find sent message json at %s: %w", path, err)
	}

	var j []*SentMessage
	if err := json.Unmarshal(file, &j); err != nil {
		return nil, err
	}

	return j, nil
}

// ToLegacyWithdrawal will convert a SentMessageJSON to a LegacyWithdrawal
// struct. This is useful because the LegacyWithdrawal struct has helper
// functions on it that can compute the withdrawal hash and the storage slot.
func (s *SentMessage) ToLegacyWithdrawal() (*crossdomain.LegacyWithdrawal, error) {
	data := make([]byte, len(s.Who)+len(s.Msg))
	copy(data, s.Msg)
	copy(data[len(s.Msg):], s.Who[:])

	var w crossdomain.LegacyWithdrawal
	if err := w.Decode(data); err != nil {
		return nil, err
	}
	return &w, nil
}

// OVMETHAddresses represents a list of addresses that interacted with
// the ERC20 representation of ether in the pre-bedrock system.
type OVMETHAddresses map[common.Address]bool

// NewAddresses will read an addresses.json file from the filesystem.
func NewAddresses(path string) (OVMETHAddresses, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot find addresses json at %s: %w", path, err)
	}

	var addresses []common.Address
	if err := json.Unmarshal(file, &addresses); err != nil {
		return nil, err
	}

	ovmeth := make(OVMETHAddresses)
	for _, addr := range addresses {
		ovmeth[addr] = true
	}

	return ovmeth, nil
}

// Allowance represents the allowances that were set in the
// legacy ERC20 representation of ether
type Allowance struct {
	From common.Address `json:"fr"`
	To   common.Address `json:"to"`
}

// NewAllowances will read the ovm-allowances.json from the file system.
func NewAllowances(path string) ([]*Allowance, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot find allowances json at %s: %w", path, err)
	}

	var allowances []*Allowance
	if err := json.Unmarshal(file, &allowances); err != nil {
		return nil, err
	}

	return allowances, nil
}

// MigrationData represents all of the data required to do a migration
type MigrationData struct {
	// OvmAddresses represents the set of addresses that interacted with the
	// LegacyERC20ETH contract before the evm equivalence upgrade
	OvmAddresses OVMETHAddresses
	// EvmAddresses represents the set of addresses that interacted with the
	// LegacyERC20ETH contract after the evm equivalence upgrade
	EvmAddresses OVMETHAddresses
	// OvmAllowances represents the set of allowances in the LegacyERC20ETH from
	// before the evm equivalence upgrade
	OvmAllowances []*Allowance
	// OvmMessages represents the set of withdrawals through the
	// L2CrossDomainMessenger from before the evm equivalence upgrade
	OvmMessages []*SentMessage
	// OvmMessages represents the set of withdrawals through the
	// L2CrossDomainMessenger from after the evm equivalence upgrade
	EvmMessages []*SentMessage
}

func (m *MigrationData) ToWithdrawals() (crossdomain.DangerousUnfilteredWithdrawals, error) {
	messages := make(crossdomain.DangerousUnfilteredWithdrawals, 0)
	for _, msg := range m.OvmMessages {
		wd, err := msg.ToLegacyWithdrawal()
		if err != nil {
			return nil, err
		}
		messages = append(messages, wd)
		if err != nil {
			return nil, err
		}
	}
	for _, msg := range m.EvmMessages {
		wd, err := msg.ToLegacyWithdrawal()
		if err != nil {
			return nil, err
		}
		messages = append(messages, wd)
	}
	return messages, nil
}

func (m *MigrationData) Addresses() []common.Address {
	addresses := make([]common.Address, 0)
	for addr := range m.EvmAddresses {
		addresses = append(addresses, addr)
	}
	for addr := range m.OvmAddresses {
		addresses = append(addresses, addr)
	}
	return addresses
}
