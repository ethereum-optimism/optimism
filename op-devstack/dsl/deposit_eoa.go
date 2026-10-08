package dsl

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// DepositEOA wraps an L2 EOA so that transactions are sent via L1 deposit
// transactions rather than direct L2 transactions.
type DepositEOA struct {
	l2    *EOA       // the L2 identity (for chain ID, block refs, etc.)
	l1    *EOA       // the L1 identity (for sending the deposit tx)
	l2EL  *L2ELNode  // L2 EL node to wait for derivation and fetch receipts
	l2Net *L2Network // L2 network for deposit contract address
}

// ViaDepositTx returns a DepositEOA that sends transactions through L1
// deposit transactions. The caller must fund both the L1 and L2 identities.
func (u *EOA) ViaDepositTx(l1 *EOA, l2EL *L2ELNode, l2Net *L2Network) *DepositEOA {
	return &DepositEOA{l2: u, l1: l1, l2EL: l2EL, l2Net: l2Net}
}

// DepositTxOpts configures a deposit transaction.
type DepositTxOpts struct {
	// GasLimit is the deposit's L2 gas limit. Zero uses max(100k, the portal's minimum).
	GasLimit uint64
}

// DepositTx sends a transaction to the given address with the given calldata
// via OptimismPortal2 on L1. It waits for L2 derivation and returns the L2 receipt.
func (d *DepositEOA) DepositTx(to common.Address, calldata []byte, opts ...func(*DepositTxOpts)) *ethtypes.Receipt {
	l2Receipt := d.sendDeposit(to, calldata, opts...)
	d.l2.t.Require().Equal(ethtypes.ReceiptStatusSuccessful, l2Receipt.Status, "deposit tx failed on L2")
	return l2Receipt
}

// DepositTxExpectRevert sends a transaction to the given address with the given calldata via
// OptimismPortal2 on L1 and requires that it reverts on L2 with exactly the given error
// signature (e.g. "CrossL2Inbox_NoExecutingDeposits()"). It waits for L2 derivation and
// returns the L2 receipt.
func (d *DepositEOA) DepositTxExpectRevert(to common.Address, calldata []byte, errorSignature string, opts ...func(*DepositTxOpts)) *ethtypes.Receipt {
	t := d.l2.t

	l2Receipt := d.sendDeposit(to, calldata, opts...)
	t.Require().Equal(ethtypes.ReceiptStatusFailed, l2Receipt.Status, "deposit tx unexpectedly succeeded on L2")

	trace := d.l2EL.TraceCalls(l2Receipt.TxHash)
	expected := crypto.Keccak256([]byte(errorSignature))[:4]
	t.Require().Equal(expected, []byte(trace.Output), "deposit tx reverted for an unexpected reason")
	return l2Receipt
}

// sendDeposit sends the deposit transaction on L1, waits for L2 derivation, and returns the
// L2 receipt without asserting the L2 execution status.
func (d *DepositEOA) sendDeposit(to common.Address, calldata []byte, opts ...func(*DepositTxOpts)) *ethtypes.Receipt {
	t := d.l2.t
	ctx := d.l2.ctx
	var o DepositTxOpts
	for _, opt := range opts {
		opt(&o)
	}

	portalAddr := d.l2Net.DepositContractAddr()
	l1Client := d.l1.el.stackEL().EthClient()
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(l1Client),
		bindings.WithTo(portalAddr),
		bindings.WithTest(t),
	)

	gasLimit := o.GasLimit
	if gasLimit == 0 {
		minGas, err := contractio.Read(portal.MinimumGasLimit(uint64(len(calldata))), ctx)
		t.Require().NoError(err, "failed to read MinimumGasLimit")
		gasLimit = max(100_000, minGas)
	}
	d.l2.log.Info("Sending deposit", "to", to, "gasLimit", gasLimit)
	depositCall := portal.DepositTransaction(to, eth.ZeroWei, gasLimit, false, calldata)
	l1Receipt, err := contractio.Write(depositCall, ctx, d.l1.Plan())
	t.Require().NoError(err, "L1 deposit tx failed")
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, l1Receipt.Status, "L1 deposit tx reverted")

	return d.l2EL.WaitForDeposit(portalAddr, l1Receipt)
}

// SendInitMessage sends an initiating message via an L1 deposit transaction.
// Returns an InitMessage with InteropOutput populated from the L2 receipt,
// fully compatible with SendExecMessage on the receiving chain.
func (d *DepositEOA) SendInitMessage(trigger *txintent.InitTrigger) *InitMessage {
	t := d.l2.t

	calldata, err := trigger.EncodeInput()
	t.Require().NoError(err, "failed to encode InitTrigger calldata")

	l2Receipt := d.DepositTx(trigger.Emitter, calldata)
	t.Require().NotZero(len(l2Receipt.Logs), "deposit tx emitted no logs on L2")

	l2BlockRef := d.l2EL.BlockRefByNumber(bigs.Uint64Strict(l2Receipt.BlockNumber))
	var result txintent.InteropOutput
	err = result.FromReceipt(d.l2.ctx, l2Receipt, l2BlockRef.BlockRef(), d.l2.ChainID())
	t.Require().NoError(err, "failed to build InteropOutput from L2 deposit receipt")

	tx := &txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]{}
	tx.Result.Set(&result)
	return &InitMessage{Tx: tx, Receipt: l2Receipt}
}

// SendRandomInitMessage creates a random initiating message and sends it via L1 deposit.
func (d *DepositEOA) SendRandomInitMessage(rng *rand.Rand, eventLoggerAddress common.Address) *InitMessage {
	topics := make([][32]byte, 2)
	for i := range topics {
		copy(topics[i][:], testutils.RandomData(rng, 32))
	}
	trigger := &txintent.InitTrigger{
		Emitter:    eventLoggerAddress,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, 10),
	}
	return d.SendInitMessage(trigger)
}
