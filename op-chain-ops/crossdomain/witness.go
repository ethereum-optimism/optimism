package crossdomain

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SentMessage represents an entry in the JSON file that is created by
// the `migration-data` package. Each entry represents a call to the
// `LegacyMessagePasser`. The `who` should always be the
// `L2CrossDomainMessenger` and the `msg` should be an abi encoded
// `relayMessage(address,address,bytes,uint256)`
type SentMessage struct {
	Who common.Address `json:"who"`
	Msg hexutil.Bytes  `json:"msg"`
}

// NewSentMessageFromJSON will read a JSON file from disk given a path to the JSON
// file. The JSON file this function reads from disk is an output from the
// `migration-data` package.
func NewSentMessageFromJSON(path string) ([]*SentMessage, error) {
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

// decodeWitnessCalldata abi decodes the calldata encoded in the input witness
// file. It errors if the 4 byte selector is not specifically for `passMessageToL1`.
// It also errors if the abi decoding fails.
func decodeWitnessCalldata(msg []byte) ([]byte, error) {
	abi, err := bindings.LegacyMessagePasserMetaData.GetAbi()
	if err != nil {
		panic("should always be able to get message passer abi")
	}

	if size := len(msg); size < 4 {
		return nil, fmt.Errorf("message too short: %d", size)
	}

	method, err := abi.MethodById(msg[:4])
	if err != nil {
		return nil, err
	}

	if method.Sig != "passMessageToL1(bytes)" {
		return nil, fmt.Errorf("unknown method: %s", method.Name)
	}

	out, err := method.Inputs.Unpack(msg[4:])
	if err != nil {
		return nil, err
	}

	cast, ok := out[0].([]byte)
	if !ok {
		panic("should always be able to cast type []byte")
	}
	return cast, nil
}

// ReadWitnessData will read messages and addresses from a raw l2geth state
// dump file.
func ReadWitnessData(path string) ([]*SentMessage, OVMETHAddresses, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open witness data file: %w", err)
	}
	defer f.Close()

	scan := bufio.NewScanner(f)
	var witnesses []*SentMessage
	addresses := make(map[common.Address]bool)
	for scan.Scan() {
		line := scan.Text()
		splits := strings.Split(line, "|")
		if len(splits) < 2 {
			return nil, nil, fmt.Errorf("invalid line: %s", line)
		}

		switch splits[0] {
		case "MSG":
			if len(splits) != 3 {
				return nil, nil, fmt.Errorf("invalid line: %s", line)
			}

			msg := splits[2]
			// Make sure that the witness data has a 0x prefix
			if !strings.HasPrefix(msg, "0x") {
				msg = "0x" + msg
			}

			msgB := hexutil.MustDecode(msg)

			// Skip any errors
			calldata, err := decodeWitnessCalldata(msgB)
			if err != nil {
				log.Warn("cannot decode witness calldata", "err", err)
				continue
			}

			witnesses = append(witnesses, &SentMessage{
				Who: common.HexToAddress(splits[1]),
				Msg: calldata,
			})
		case "ETH":
			addresses[common.HexToAddress(splits[1])] = true
		default:
			return nil, nil, fmt.Errorf("invalid line: %s", line)
		}
	}

	return witnesses, addresses, nil
}

// ToLegacyWithdrawal will convert a SentMessageJSON to a LegacyWithdrawal
// struct. This is useful because the LegacyWithdrawal struct has helper
// functions on it that can compute the withdrawal hash and the storage slot.
func (s *SentMessage) ToLegacyWithdrawal() (*LegacyWithdrawal, error) {
	data := make([]byte, len(s.Who)+len(s.Msg))
	copy(data, s.Msg)
	copy(data[len(s.Msg):], s.Who[:])

	var w LegacyWithdrawal
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

func (m *MigrationData) ToWithdrawals() (DangerousUnfilteredWithdrawals, []InvalidMessage, error) {
	messages := make(DangerousUnfilteredWithdrawals, 0)
	invalidMessages := make([]InvalidMessage, 0)
	for _, msg := range m.OvmMessages {
		wd, err := msg.ToLegacyWithdrawal()
		if err != nil {
			return nil, nil, fmt.Errorf("error serializing OVM message: %w", err)
		}
		messages = append(messages, wd)
	}
	for _, msg := range m.EvmMessages {
		wd, err := msg.ToLegacyWithdrawal()
		if err != nil {
			log.Warn("Discovered mal-formed withdrawal", "who", msg.Who, "data", msg.Msg)
			invalidMessages = append(invalidMessages, InvalidMessage(*msg))
			continue
		}
		messages = append(messages, wd)
	}
	return messages, invalidMessages, nil
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
