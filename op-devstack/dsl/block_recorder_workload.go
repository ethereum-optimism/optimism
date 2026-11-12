package dsl

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

// BlockRecorderWorkload changes state in nearly every block: a writer and a counterparty transfer
// back and forth and the writer calls a block recorder contract, so every block it fills carries
// balance and storage changes that a rebuilt block would not repeat.
type BlockRecorderWorkload struct {
	commonImpl

	counterparty  *EOA
	inclusionPoll txplan.Option
	recorder      common.Address
	// setupBlock is the block that completed the funding and the contract deployment; a chain that
	// lacks it lacks the workload's accounts.
	setupBlock uint64
	writer     *EOA
}

// NewBlockRecorderWorkload funds two senders, deploys a block recorder and calls it once.
func NewBlockRecorderWorkload(t devtest.T, funder *FunderEOA) *BlockRecorderWorkload {
	w := &BlockRecorderWorkload{
		commonImpl:    commonFromT(t),
		counterparty:  funder.NewFundedEOA(eth.OneEther),
		inclusionPoll: txplan.Combine(),
		writer:        funder.NewFundedEOA(eth.OneEther),
	}
	w.recorder = w.writer.DeployBlockRecorder()
	w.setupBlock = IncludedBlockNumber(t, w.writer.Transact(w.writer.Plan(), txplan.WithTo(&w.recorder)))
	w.log.Info("Block recorder workload set up", "recorder", w.recorder, "setup_block", w.setupBlock)
	return w
}

// WithFreshSenders returns a workload that writes to the same contract from two newly funded
// accounts. A node that reorgs puts the transactions of the blocks it drops back into its pool, so
// new transactions from the original senders could race those for the same nonces.
func (w *BlockRecorderWorkload) WithFreshSenders(funder *FunderEOA) *BlockRecorderWorkload {
	fresh := *w
	fresh.counterparty = funder.NewFundedEOA(eth.OneEther)
	fresh.writer = funder.NewFundedEOA(eth.OneEther)
	return &fresh
}

// WithInclusionPoll returns a workload that polls el for its transactions' receipts every
// interval, up to maxAttempts times, so DriveUntilBlock returns within about interval of the block
// that held its last write rather than after the default exponential backoff.
func (w *BlockRecorderWorkload) WithInclusionPoll(el *L2ELNode, interval time.Duration, maxAttempts int) *BlockRecorderWorkload {
	polled := *w
	polled.inclusionPoll = txplan.WithRetryInclusion(el.EthClient(), maxAttempts, retry.Fixed(interval))
	return &polled
}

// Accounts are the senders and the recorder with the storage slots it writes.
func (w *BlockRecorderWorkload) Accounts() []ProofAccount {
	return append(w.SenderAccounts(),
		ProofAccount{Address: w.recorder, StorageKeys: []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}})
}

func (w *BlockRecorderWorkload) SenderAccounts() []ProofAccount {
	return []ProofAccount{{Address: w.counterparty.Address()}, {Address: w.writer.Address()}}
}

func (w *BlockRecorderWorkload) SetupBlock() uint64 {
	return w.setupBlock
}

// DriveUntilBlock sends transactions one after another until a contract write lands in a block at
// or past target, and returns that block.
func (w *BlockRecorderWorkload) DriveUntilBlock(target uint64) uint64 {
	w.log.Info("Driving the block recorder workload", "target", target)
	for {
		w.counterparty.Transact(w.counterparty.PlanTransfer(w.writer.Address(), eth.OneHundredthEther), w.inclusionPoll)
		w.writer.Transact(w.writer.PlanTransfer(w.counterparty.Address(), eth.OneHundredthEther), w.inclusionPoll)
		write := w.writer.Transact(w.writer.Plan(), txplan.WithTo(&w.recorder), w.inclusionPoll)
		block := IncludedBlockNumber(w.t, write)
		if block >= target {
			w.log.Info("Block recorder workload reached target", "target", target, "last_write_block", block)
			return block
		}
	}
}
