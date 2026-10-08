package dsl

import (
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ETHSend is a SuperchainETHBridge.sendETH.
type ETHSend struct {
	commonImpl
	// Message is the message that relays the ETH on the destination.
	Message SentMessage
	// Receipt is the receipt of the sendETH transaction.
	Receipt *types.Receipt
	// BlockTime is the timestamp of the block the send is in.
	BlockTime uint64

	tx *txintent.IntentTx[*rawCall, *txintent.InteropOutput]
}

// SendETH sends `amount` of the sender's ETH to `recipient` on `destination` through the
// SuperchainETHBridge. It requires the send to succeed and returns it; its message is not relayed.
func SendETH(sender *EOA, recipient common.Address, destination eth.ChainID, amount eth.ETH) *ETHSend {
	bridge := bindings.NewBindings[bindings.SuperchainETHBridge](bindings.WithTo(predeploys.SuperchainETHBridgeAddr))
	call := bridge.SendETH(recipient, destination)
	data, err := call.EncodeInput()
	sender.require.NoError(err, "failed to encode sendETH")

	sender.log.Info("Sending ETH through the SuperchainETHBridge",
		"from", sender.Address(), "to", recipient, "destination", destination, "amount", amount)
	tx := txintent.NewIntent[*rawCall, *txintent.InteropOutput](sender.Plan(), txplan.WithValue(amount))
	tx.Content.Set(&rawCall{to: predeploys.SuperchainETHBridgeAddr, data: data})
	rcpt, err := tx.PlannedTx.Included.Eval(sender.ctx)
	sender.require.NoError(err, "sendETH was not included")
	sender.require.Equal(types.ReceiptStatusSuccessful, rcpt.Status, "sendETH failed")
	block, err := tx.PlannedTx.IncludedBlock.Eval(sender.ctx)
	sender.require.NoError(err, "sendETH block not found")
	msg, err := SentMessageFromReceipt(rcpt, sender.ChainID())
	sender.require.NoError(err, "sendETH emitted no SentMessage")

	return &ETHSend{commonImpl: sender.commonImpl, Message: msg, Receipt: rcpt, BlockTime: block.Time, tx: tx}
}

// Relay waits for `validator` to validate the send's block, relays its message on the
// destination as `relayer`, requires the relay to succeed, and returns its receipt.
func (s *ETHSend) Relay(relayer *EOA, validator SuperRootSource) *types.Receipt {
	s.log.Info("Waiting for the ETH send to be validated", "timestamp", s.BlockTime)
	validator.AwaitValidatedTimestamp(s.BlockTime)

	s.log.Info("Relaying the ETH send", "messageHash", s.Message.Hash, "relayer", relayer.Address())
	relayTx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](relayer.Plan())
	relayTx.Content.DependOn(&s.tx.Result)
	relayTx.Content.Fn(txintent.RelayIndexed(
		predeploys.L2toL2CrossDomainMessengerAddr, &s.tx.Result, &s.tx.PlannedTx.Included, s.Message.LogIndex))
	rcpt, err := relayTx.PlannedTx.Included.Eval(s.ctx)
	s.require.NoError(err, "relay of message %s was not included", s.Message.Hash)
	s.require.Equal(types.ReceiptStatusSuccessful, rcpt.Status, "relay of message %s failed", s.Message.Hash)
	return rcpt
}

// rawCall is a txintent call over pre-encoded calldata.
type rawCall struct {
	to   common.Address
	data []byte
}

func (c *rawCall) To() (*common.Address, error)          { return &c.to, nil }
func (c *rawCall) EncodeInput() ([]byte, error)          { return c.data, nil }
func (c *rawCall) AccessList() (types.AccessList, error) { return nil, nil }
