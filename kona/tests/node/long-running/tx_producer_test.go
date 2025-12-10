package node

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	node_utils "github.com/op-rs/kona/node/utils"
)

// Define a global atomic counter for the number of transactions produced.
var (
	txProduced = atomic.Int64{}
)

type TxProducer struct {
	t        devtest.T
	out      *node_utils.MixedOpKonaPreset
	accounts []*dsl.EOA
	// Unique identifier for the producer/receiver pair
	idx         int
	pending_txs chan<- *txplan.PlannedTx
}

type TxReceiver struct {
	t   devtest.T
	out *node_utils.MixedOpKonaPreset
	// Unique identifier for the producer/receiver pair
	idx int
	txs <-chan *txplan.PlannedTx
}

func (tp *TxProducer) NewFunder() *dsl.Funder {
	return dsl.NewFunder(tp.out.Wallet, tp.out.Faucet, tp.out.L2ELSequencerNodes()[0])
}

func (tp *TxProducer) NewAccounts(count int, fundAmount eth.ETH) []*dsl.EOA {
	new_accounts := tp.NewFunder().NewFundedEOAs(count, fundAmount)
	tp.accounts = append(tp.accounts, new_accounts...)

	return new_accounts
}

func (tp *TxProducer) NewAccount(fundAmount eth.ETH) *dsl.EOA {
	new_account := tp.NewFunder().NewFundedEOA(fundAmount)
	tp.accounts = append(tp.accounts, new_account)

	return new_account
}

func NewTxProducer(t devtest.T, out *node_utils.MixedOpKonaPreset, txs chan<- *txplan.PlannedTx, idx int) *TxProducer {
	return &TxProducer{
		out:         out,
		t:           t,
		accounts:    []*dsl.EOA{},
		pending_txs: txs,
		idx:         idx,
	}
}

func (tp *TxProducer) Start(wg *sync.WaitGroup) {
	// Initialize the accounts
	tp.NewAccounts(*initNumAccounts, eth.Ether(uint64(*fundAmount)))
	tp.t.Logf("%d accounts initialized", *initNumAccounts)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			var toAccount *dsl.EOA
			if rand.Intn(100) < *percentageNewAccounts {
				toAccount = tp.NewAccount(eth.Ether(uint64(*fundAmount)))
			} else {
				toAccount = tp.accounts[rand.Intn(len(tp.accounts))]
			}

			fromAccount := tp.accounts[rand.Intn(len(tp.accounts))]

			if fromAccount.GetBalance().Lt(eth.HalfEther) {
				tp.NewFunder().FundAtLeast(fromAccount, eth.HalfEther)
			}

			amount := fromAccount.GetBalance().Mul(uint64(rand.Intn(100))).Div(100)

			tp.t.Logf("producer %d: producing transaction from %s to %s with amount %s", tp.idx, fromAccount.Address(), toAccount.Address(), amount)

			new_planned_txs := fromAccount.Transact(fromAccount.PlanTransfer(toAccount.Address(), amount))

			tp.t.Logf("producer %d: transaction produced with hash: %s", tp.idx, new_planned_txs.Signed.Value().Hash())

			tp.pending_txs <- new_planned_txs
		}
	}()
}

func NewTxReceiver(t devtest.T, out *node_utils.MixedOpKonaPreset, txs <-chan *txplan.PlannedTx, idx int) *TxReceiver {
	return &TxReceiver{
		t:   t,
		txs: txs,
		idx: idx,
		out: out,
	}
}

func (tr *TxReceiver) processTx(tx *txplan.PlannedTx) {
	inclusionBlock, err := tx.IncludedBlock.Eval(tr.t.Ctx())
	if err != nil {
		tr.t.Errorf("producer %d: transaction (hash %s) receipt not found. error: %s", tr.idx, tx.Signed.Value().Hash(), err)
		return
	}

	_, err = tx.Success.Eval(tr.t.Ctx())
	if err != nil {
		tr.t.Errorf("producer %d: transaction (hash %s) failed. error: %s", tr.idx, tx.Signed.Value().Hash(), err)
	}

	// Ensure the block containing the transaction has propagated to the rest of the network.
	for _, node := range tr.out.L2ELNodes() {
		block := node.WaitForBlockNumber(inclusionBlock.Number)
		blockID := block.Hash()

		// It's possible that the block has already been included, and `WaitForBlockNumber` returns a block
		// at a taller height.
		if block.NumberU64() > inclusionBlock.Number {
			blockID = node.BlockRefByNumber(inclusionBlock.Number).Hash
		}

		// Ensure that the block ID matches the expected inclusion block hash.
		if blockID != inclusionBlock.Hash {
			tr.t.Errorf("producer %d: transaction (hash %s) not included in block %d with hash %s.", tr.idx, tx.Signed.Value().Hash(), inclusionBlock.Number, inclusionBlock.Hash)
		}
	}

	txProduced.Add(1)
	tr.t.Logf("producer %d: transaction (hash %s) included in block %d with hash %s. %d transactions produced.", tr.idx, tx.Signed.Value().Hash(), inclusionBlock.Number, inclusionBlock.Hash, txProduced.Load())
}

func (tr *TxReceiver) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-tr.t.Ctx().Done():
				tr.t.Logf("receiver context done")
				return
			case tx := <-tr.txs:
				tr.processTx(tx)
			}
		}
	}()

}

// Produces transactions in a loop. Ensures that...
// - transactions get included
// - transactions get gossiped
func TestTxProducer(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := node_utils.NewMixedOpKona(t)

	var wg sync.WaitGroup

	for i := 0; i < *num_threads; i++ {
		txs := make(chan *txplan.PlannedTx)
		txProducer := NewTxProducer(t, out, txs, i)
		txReceiver := NewTxReceiver(t, out, txs, i)

		txProducer.Start(&wg)
		txReceiver.Start(&wg)
	}

	wg.Wait()

	t.Logf("producer and receiver threads finished")
}
