package dsl

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// DepositEOA wraps an L2 EOA so that initiating messages are sent via L1 deposit
// transactions rather than direct L2 transactions. The resulting InitMessage is
// compatible with SendExecMessage on the receiving chain.
type DepositEOA struct {
	l2    *EOA       // the L2 identity (for chain ID, block refs, etc.)
	l1    *EOA       // the L1 identity (for sending the deposit tx)
	l2EL  *L2ELNode  // L2 EL node to wait for derivation and fetch receipts
	l2Net *L2Network // L2 network for deposit contract address
}

// ViaDepositTx returns a DepositEOA that sends initiating messages through L1
// deposit transactions. The caller must fund both the L1 and L2 identities.
func (u *EOA) ViaDepositTx(l1 *EOA, l2EL *L2ELNode, l2Net *L2Network) *DepositEOA {
	return &DepositEOA{l2: u, l1: l1, l2EL: l2EL, l2Net: l2Net}
}

// SendInitMessage sends an initiating message via an L1 deposit transaction.
// It deposits a call to the EventLogger through OptimismPortal2, waits for L2
// derivation, and returns an InitMessage with InteropOutput populated from the
// L2 receipt — fully compatible with SendExecMessage.
func (d *DepositEOA) SendInitMessage(trigger *txintent.InitTrigger) *InitMessage {
	t := d.l2.t
	ctx := d.l2.ctx

	calldata, err := trigger.EncodeInput()
	t.Require().NoError(err, "failed to encode InitTrigger calldata")

	// Create OptimismPortal2 binding on L1.
	portalAddr := d.l2Net.DepositContractAddr()
	l1Client := d.l1.el.stackEL().EthClient()
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(l1Client),
		bindings.WithTo(portalAddr),
		bindings.WithTest(t),
	)

	// Read minimum gas and deposit the call.
	minGas, err := contractio.Read(portal.MinimumGasLimit(uint64(len(calldata))), ctx)
	t.Require().NoError(err, "failed to read MinimumGasLimit")
	depositCall := portal.DepositTransaction(trigger.Emitter, eth.ZeroWei, max(100_000, minGas), false, calldata)
	l1Receipt, err := contractio.Write(depositCall, ctx, d.l1.Plan())
	t.Require().NoError(err, "L1 deposit tx failed")
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, l1Receipt.Status, "L1 deposit tx reverted")

	// Derive the L2 deposit tx hash from the L1 receipt.
	var l2DepositTx *ethtypes.DepositTx
	for _, log := range l1Receipt.Logs {
		if l2DepositTx, err = derive.UnmarshalDepositLogEvent(log); err == nil {
			break
		}
	}
	t.Require().NotNil(l2DepositTx, "no TransactionDeposited event in L1 receipt")

	// Wait for L2 to derive past the deposit's L1 block, then fetch the receipt.
	d.l2EL.WaitL1OriginReached(eth.Unsafe, l1Receipt.BlockNumber.Uint64(), 120)
	l2Receipt := d.l2EL.WaitForReceipt(ethtypes.NewTx(l2DepositTx).Hash())
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, l2Receipt.Status, "deposit tx failed on L2")
	t.Require().NotZero(len(l2Receipt.Logs), "deposit tx emitted no logs on L2")

	// Build the InteropOutput from the L2 receipt so SendExecMessage works.
	l2BlockRef := d.l2EL.BlockRefByNumber(l2Receipt.BlockNumber.Uint64())
	var result txintent.InteropOutput
	err = result.FromReceipt(ctx, l2Receipt, l2BlockRef.BlockRef(), d.l2.ChainID())
	t.Require().NoError(err, "failed to build InteropOutput from L2 deposit receipt")

	// Construct an InitMessage with a pre-resolved Result lazy.
	tx := &txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]{}
	tx.Result.Set(&result)
	return &InitMessage{Tx: tx, Receipt: l2Receipt}
}

// SendRandomInitMessage creates a random initiating message and sends it via L1 deposit.
func (d *DepositEOA) SendRandomInitMessage(rng *rand.Rand, eventLoggerAddress common.Address, topicCount, dataLen int) *InitMessage {
	if topicCount > 4 {
		topicCount = 4
	}
	if topicCount < 1 {
		topicCount = 1
	}
	if dataLen < 1 {
		dataLen = 1
	}

	topics := make([][32]byte, topicCount)
	for i := range topics {
		copy(topics[i][:], testutils.RandomData(rng, 32))
	}

	trigger := &txintent.InitTrigger{
		Emitter:    eventLoggerAddress,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, dataLen),
	}
	return d.SendInitMessage(trigger)
}
