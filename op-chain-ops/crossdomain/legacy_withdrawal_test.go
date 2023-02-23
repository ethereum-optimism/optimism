package crossdomain_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// callFrame represents the response returned from geth's
// `debug_traceTransaction` callTracer
type callFrame struct {
	Type    string      `json:"type"`
	From    string      `json:"from"`
	To      string      `json:"to,omitempty"`
	Value   string      `json:"value,omitempty"`
	Gas     string      `json:"gas"`
	GasUsed string      `json:"gasUsed"`
	Input   string      `json:"input"`
	Output  string      `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
	Calls   []callFrame `json:"calls,omitempty"`
}

// stateDiff represents the response returned from geth's
// `debug_traceTransaction` preStateTracer
type stateDiff map[common.Address]stateDiffAccount

// stateDiffAccount represents a single account in the preStateTracer
type stateDiffAccount struct {
	Balance hexutil.Big   `json:"balance"`
	Code    hexutil.Bytes `json:"code"`
	Nonce   uint64        `json:"nonce"`
	Storage map[common.Hash]common.Hash
}

var (
	// traces represents a prepopulated map of call traces
	traces map[string]*callFrame
	// receipts represents a prepopulated map of receipts
	receipts map[string]*types.Receipt
	// stateDiffs represents a prepopulated map of state diffs
	stateDiffs map[string]stateDiff
	// passMessageABI is a JSON representation of the legacy L2ToL1MessagePasser ABI
	passMessageABI = "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"passMessageToL1\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
	// passMessage represents an initialized L2ToL1MessagePasser ABI
	passMessage abi.ABI
	// base represents
	base = "testdata"
	// callTracesPath represents
	callTracesPath = filepath.Join(base, "call-traces")
	// receiptsPath represents
	receiptsPath = filepath.Join(base, "receipts")
	// stateDiffsPath represents
	stateDiffsPath = filepath.Join(base, "state-diffs")
)

func init() {
	traces = make(map[string]*callFrame)
	receipts = make(map[string]*types.Receipt)
	stateDiffs = make(map[string]stateDiff)

	// Read all of the receipt test vectors into memory
	if err := readReceipts(); err != nil {
		panic(err)
	}
	// Read all of the transaction trace vectors into memory
	if err := readTraces(); err != nil {
		panic(err)
	}
	// Read all of the state diff vectors into memory
	if err := readStateDiffs(); err != nil {
		panic(err)
	}
	// Initialze the message passer ABI
	var err error
	passMessage, err = abi.JSON(strings.NewReader(passMessageABI))
	if err != nil {
		panic(err)
	}
}

// TestWithdrawalLegacyStorageSlot will test that the computation
// of the legacy storage slot is correct. It is done so using real
// test vectors generated from mainnet.
func TestWithdrawalLegacyStorageSlot(t *testing.T) {
	for hash, trace := range traces {
		t.Run(hash, func(t *testing.T) {
			// Given a callTrace, find the call that corresponds
			// to L2ToL1MessagePasser.passMessageToL1
			call := findPassMessage(trace)
			require.NotNil(t, call)

			receipt, ok := receipts[hash]
			require.True(t, ok)

			// Given a receipt, parse the cross domain message
			// from its logs
			msg, err := findCrossDomainMessage(receipt)
			require.Nil(t, err)
			// Ensure that it is a version 0 cross domain message
			require.Equal(t, uint64(0), msg.Version())

			// Encode the cross domain message
			encoded, err := msg.Encode()
			require.Nil(t, err)

			// ABI encode the serialized cross domain message
			packed, err := passMessage.Pack("passMessageToL1", encoded)
			require.Nil(t, err)

			// Decode the calldata where the L2CrossDomainMessenger is calling
			// L2ToL1MessagePasser.passMessageToL1 from the callTrace
			calldata := hexutil.MustDecode(call.Input)

			// If these values are the same, we know for a fact that the
			// cross domain message was correctly parsed from the logs.
			require.Equal(t, calldata, packed)

			// Cast the cross domain message to a withdrawal. Note that
			// this only works for legacy style messages
			withdrawal := toWithdrawal(t, common.HexToAddress(call.From), msg)

			// Compute the legacy storage slot for the withdrawal
			slot, err := withdrawal.StorageSlot()
			require.Nil(t, err)

			// Get the state diff that corresponds to this transaction
			diff, ok := stateDiffs[hash]
			require.True(t, ok)

			// Get the account out of the state diff that corresponds
			// to the L2ToL1MessagePasser
			messagePasser, ok := diff[predeploys.LegacyMessagePasserAddr]
			require.True(t, ok)

			// The computed storage slot must be in the state diff. Note
			// that the built-in preStateTracer includes the storage slots
			// that were altered by the transaction but the values are
			// the values before any modifications to state by the transaction
			_, ok = messagePasser.Storage[slot]
			require.True(t, ok)
		})
	}
}

func FuzzEncodeDecodeLegacyWithdrawal(f *testing.F) {
	f.Fuzz(func(t *testing.T, _msgSender, _target, _sender, _nonce, data []byte) {
		msgSender := common.BytesToAddress(_msgSender)
		target := common.BytesToAddress(_target)
		sender := common.BytesToAddress(_sender)
		nonce := new(big.Int).SetBytes(_nonce)

		withdrawal := crossdomain.NewLegacyWithdrawal(msgSender, target, sender, data, nonce)

		encoded, err := withdrawal.Encode()
		require.Nil(t, err)

		var w crossdomain.LegacyWithdrawal
		err = w.Decode(encoded)
		require.Nil(t, err)

		require.Equal(t, withdrawal.XDomainNonce.Uint64(), w.XDomainNonce.Uint64())
		require.Equal(t, withdrawal.XDomainSender, w.XDomainSender)
		require.Equal(t, withdrawal.XDomainTarget, w.XDomainTarget)
		require.Equal(t, withdrawal.XDomainData, w.XDomainData)
	})
}

// findPassMessage pulls the call from the L2CrossDomainMessenger to the
// L2ToL1MessagePasser out of the call trace. This call is used to assert
// against the calldata
func findPassMessage(trace *callFrame) *callFrame {
	isCall := trace.Type == "CALL"
	isTarget := trace.To == predeploys.LegacyMessagePasser
	isFrom := trace.From == predeploys.L2CrossDomainMessenger
	if isCall && isTarget && isFrom {
		return trace
	}
	for _, subcall := range trace.Calls {
		if call := findPassMessage(&subcall); call != nil {
			return call
		}
	}
	return nil
}

// findCrossDomainMessage will parse a CrossDomainMessage from a receipt
func findCrossDomainMessage(receipt *types.Receipt) (*crossdomain.CrossDomainMessage, error) {
	backend := backends.NewSimulatedBackend(nil, 15000000)
	l2xdm, err := bindings.NewL2CrossDomainMessenger(predeploys.L2CrossDomainMessengerAddr, backend)
	if err != nil {
		return nil, err
	}
	abi, _ := bindings.L2CrossDomainMessengerMetaData.GetAbi()
	var msg crossdomain.CrossDomainMessage

	seen := false

	// Assume there is only 1 deposit per transaction
	for _, log := range receipt.Logs {
		event, _ := abi.EventByID(log.Topics[0])
		// Not the event we are looking for
		if event == nil {
			continue
		}
		// Parse the legacy event
		if event.Name == "SentMessage" {
			e, _ := l2xdm.ParseSentMessage(*log)
			msg.Target = e.Target
			msg.Sender = e.Sender
			msg.Data = e.Message
			msg.Nonce = e.MessageNonce
			msg.GasLimit = e.GasLimit

			// Set seen to true to ensure that this event
			// was observed
			seen = true
		}
		// Parse the new extension event
		if event.Name == "SentMessageExtension1" {
			e, _ := l2xdm.ParseSentMessageExtension1(*log)
			msg.Value = e.Value
		}
	}
	if seen {
		return &msg, nil
	} else {
		return nil, fmt.Errorf("cannot find receipt for %s", receipt.TxHash)
	}
}

// readTraces will read all traces into memory
func readTraces() error {
	entries, err := os.ReadDir(callTracesPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		trace, err := readTrace(name)
		if err != nil {
			return err
		}
		traces[name] = trace
	}
	return nil
}

// readReceipts will read all receipts into memory
func readReceipts() error {
	entries, err := os.ReadDir(receiptsPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		trace, err := readReceipt(name)
		if err != nil {
			return err
		}
		receipts[name] = trace
	}
	return nil
}

// readStateDiffs will read all state diffs into memory
func readStateDiffs() error {
	entries, err := os.ReadDir(stateDiffsPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		diff, err := readStateDiff(name)
		if err != nil {
			return err
		}
		stateDiffs[name] = diff
	}
	return nil
}

// readTrace will read the transaction trace by hash from disk
func readTrace(hash string) (*callFrame, error) {
	vector := filepath.Join(callTracesPath, hash)
	file, err := os.ReadFile(vector)
	if err != nil {
		return nil, err
	}
	var trace callFrame
	if err := json.Unmarshal(file, &trace); err != nil {
		return nil, err
	}
	return &trace, nil
}

// readReceipt will read the receipt by hash from disk
func readReceipt(hash string) (*types.Receipt, error) {
	vector := filepath.Join(receiptsPath, hash)
	file, err := os.ReadFile(vector)
	if err != nil {
		return nil, err
	}
	var receipt types.Receipt
	if err := json.Unmarshal(file, &receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

// readStateDiff will read the state diff by hash from disk
func readStateDiff(hash string) (stateDiff, error) {
	vector := filepath.Join(stateDiffsPath, hash)
	file, err := os.ReadFile(vector)
	if err != nil {
		return nil, err
	}
	var diff stateDiff
	if err := json.Unmarshal(file, &diff); err != nil {
		return nil, err
	}
	return diff, nil
}

// ToWithdrawal will turn a CrossDomainMessage into a Withdrawal.
// This only works for version 0 CrossDomainMessages as not all of
// the data is present for version 1 CrossDomainMessages to be turned
// into Withdrawals.
func toWithdrawal(t *testing.T, msgSender common.Address, c *crossdomain.CrossDomainMessage) *crossdomain.LegacyWithdrawal {
	version := c.Version()
	switch version {
	case 0:
		if c.Value != nil && c.Value.Cmp(common.Big0) != 0 {
			t.Fatalf("version 0 messages must have 0 value")
		}
		w := &crossdomain.LegacyWithdrawal{
			MessageSender: msgSender,
			XDomainTarget: c.Target,
			XDomainSender: c.Sender,
			XDomainData:   c.Data,
			XDomainNonce:  c.Nonce,
		}
		return w
	case 1:
		t.Fatalf("cannot convert version 1 messages to withdrawals")
	default:
		t.Fatalf("unknown message version: %d", version)
	}
	return nil
}
