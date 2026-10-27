package dsl

import (
	"bytes"
	"math/big"
	"slices"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	undeliveredMessageExportedTopic = crypto.Keccak256Hash(
		[]byte("UndeliveredMessageExported(bytes32,uint256,address,uint256)"))
	expireMessageSelector = crypto.Keccak256([]byte("expireMessage(bytes32,uint256)"))[:4]
)

// ErrorSelector returns the 4-byte selector of a custom error signature, such as
// "CrossL2Inbox_NoExecutingDeposits()".
func ErrorSelector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:4]
}

// L2ToL2CrossDomainMessenger returns a binding of this chain's L2ToL2CrossDomainMessenger.
func (el *L2ELNode) L2ToL2CrossDomainMessenger() bindings.L2ToL2CrossDomainMessenger {
	return bindings.NewBindings[bindings.L2ToL2CrossDomainMessenger](bindings.WithClient(el.EthClient()),
		bindings.WithTo(predeploys.L2toL2CrossDomainMessengerAddr), bindings.WithTest(el.t))
}

// L1CrossDomainMessengerAddr returns this chain's L1CrossDomainMessenger, read from its
// SystemConfig.
func (n *L2Network) L1CrossDomainMessengerAddr() common.Address {
	systemConfig := bindings.NewSystemConfig(bindings.WithClient(n.PrimaryL1EL().EthClient()),
		bindings.WithTo(n.inner.Deployment().SystemConfigProxyAddr()), bindings.WithTest(n.t))
	messenger, err := contractio.Read(systemConfig.L1CrossDomainMessenger(), n.ctx)
	n.require.NoError(err, "failed to read the L1CrossDomainMessenger of chain %s", n.ChainID())
	return messenger
}

// ExportUndeliveredMessage forces a call to `exporter`, the UndeliveredMessageExporter on this
// chain, in as a deposit: the exporter tells `sourceMessenger`, the source chain's
// L1CrossDomainMessenger, through a withdrawal, that `m` has not been relayed on this chain.
// `l1GasLimit` is the gas the exporter reserves for that call on L1. It requires the export to
// succeed and to emit UndeliveredMessageExported for the message, and returns the receipt and this
// chain's timestamp at the export.
func (d *DepositEOA) ExportUndeliveredMessage(exporter common.Address, sourceMessenger common.Address,
	m SentMessage, l1GasLimit uint32, opts ...func(*DepositTxOpts)) (*types.Receipt, uint64) {
	call := bindings.NewBindings[bindings.UndeliveredMessageExporter](bindings.WithTest(d.l2.t)).
		ExportUndeliveredMessage(sourceMessenger, m.Source, m.Nonce, m.Sender, m.Target, m.Message, l1GasLimit)
	data, err := call.EncodeInput()
	d.l2.require.NoError(err, "failed to encode exportUndeliveredMessage")

	d.l2.log.Info("Exporting undelivered message through a deposit",
		"messageHash", m.Hash, "exporter", exporter, "sourceMessenger", sourceMessenger)
	rcpt := d.DepositTx(exporter, data, opts...)
	d.l2.require.Truef(slices.ContainsFunc(rcpt.Logs, func(l *types.Log) bool {
		return l.Address == exporter && len(l.Topics) > 1 &&
			l.Topics[0] == undeliveredMessageExportedTopic && l.Topics[1] == m.Hash
	}), "exporter %s emitted no UndeliveredMessageExported for message %s in tx %s", exporter, m.Hash, rcpt.TxHash)
	return rcpt, d.l2EL.BlockRefByHash(rcpt.BlockHash).Time
}

// ExpireMessageCall is a call to L2ToL2CrossDomainMessenger.expireMessage, read from a trace.
type ExpireMessageCall struct {
	MessageHash   common.Hash
	UndeliveredAt uint64
	// Output is the call's return data, or its revert data if it reverted.
	Output []byte
}

// ExpireMessageCall traces a transaction on this chain and returns the first call it made to this
// chain's L2ToL2CrossDomainMessenger.expireMessage. It requires there to be one.
func (el *L2ELNode) ExpireMessageCall(txHash common.Hash) ExpireMessageCall {
	frame, found := el.TraceCalls(txHash).Find(func(f CallFrame) bool {
		return f.To == predeploys.L2toL2CrossDomainMessengerAddr && bytes.HasPrefix(f.Input, expireMessageSelector)
	})
	el.require.Truef(found, "tx %s made no expireMessage call to the L2ToL2CrossDomainMessenger", txHash)
	args, err := abi.Arguments{{Type: abiType("bytes32")}, {Type: abiType("uint256")}}.Unpack(frame.Input[4:])
	el.require.NoErrorf(err, "failed to decode the expireMessage call in tx %s", txHash)
	call := ExpireMessageCall{
		MessageHash:   common.Hash(args[0].([32]byte)),
		UndeliveredAt: bigs.Uint64Strict(args[1].(*big.Int)),
		Output:        frame.Output,
	}
	el.log.Info("Found expireMessage call", "tx", txHash, "messageHash", call.MessageHash,
		"undeliveredAt", call.UndeliveredAt, "output", common.Bytes2Hex(call.Output))
	return call
}
