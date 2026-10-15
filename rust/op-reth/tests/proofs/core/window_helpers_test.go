package core

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/rust/op-reth/tests/proofs/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// l2BlockTimeSeconds keeps the runs short: the chain has to grow past the proofs window for
	// pruning to run, so halving the block time halves that wait.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so how far reth's datadir trails its head does
	// not depend on reth's defaults.
	memoryBlockBufferTarget = 5
	// referenceEthProofWindow lets the sequencer answer eth_getProof for every block of the run
	// from reth's own historical state; reth's default window of zero rejects them all.
	referenceEthProofWindow = 1_000_000
	// windowMargin is how far past a full proofs window the chain is driven before the window is
	// inspected, so its earliest block is one pruning left behind.
	windowMargin = 10

	// blockAttempts bounds, in 2s polls, the wait for a node to reach a block.
	blockAttempts = 30
	// pruneTimeout bounds how long the store may take to prune once the chain is a window past
	// the workload's setup.
	pruneTimeout = 90 * time.Second
	// storeCatchUpTimeout bounds the wait for the follower's store to flush the chain head once
	// sequencing stopped.
	storeCatchUpTimeout = 60 * time.Second
	// syncAttempts bounds, in 2s polls, the wait for the follower to catch up with the sequencer.
	syncAttempts = 60
)

// persistenceThresholds are the reth persistence thresholds the cells run under: one that keeps
// reth's persisted tip inside a 20-block proofs window and one that keeps it behind the window.
var persistenceThresholds = []uint64{7, 50}

var (
	recorderNumberSlot    = common.HexToHash("0x0")
	recorderTimestampSlot = common.HexToHash("0x1")
)

// skipUnlessOpRethValidator skips a test that needs the validator to serve proofs history.
func skipUnlessOpRethValidator(t devtest.T) {
	if kind := utils.ValidatorELKind(t); kind != sysgo.MixedL2ELOpReth {
		t.Skipf("proofs history is an op-reth feature; OP_DEVSTACK_PROOF_VALIDATOR_EL selects %s", kind)
	}
}

// newProofsFollowerSystem starts a sequencer without proofs history, which serves as the
// independent eth_getProof reference, and a validator running proofs history as the follower under
// test with the given op-reth options.
func newProofsFollowerSystem(t devtest.T, sequencerArgs []string, followerOpts ...sysgo.OpRethOption) *utils.MixedOpProofPreset {
	skipUnlessOpRethValidator(t)
	return utils.NewMixedOpProofPreset(t, func(cfg *utils.MixedOpProofPresetConfig) {
		cfg.DeployerOptions = []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)}
		cfg.SequencerOpRethOpts = []sysgo.OpRethOption{
			sysgo.OpRethWithoutProofsHistory(),
			sysgo.OpRethWithEthProofWindow(referenceEthProofWindow),
			sysgo.OpRethWithExtraArgs(sequencerArgs...),
		}
		cfg.ValidatorOpRethOpts = followerOpts
	})
}

// engineArgs pin reth's persistence schedule. Both nodes get them so the sequencer persists on
// the same schedule as the follower.
func engineArgs(persistenceThreshold uint64) []string {
	return []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", persistenceThreshold),
	}
}

// workload changes state in nearly every block: alice and bob transfer back and forth and bob
// writes to a contract, so every block of the window carries balance and storage changes.
type workload struct {
	alice *dsl.EOA
	bob   *dsl.EOA
	// carol is funded before the workload starts and never transacts, so a transaction of hers
	// with nonce zero executes against the state of any block after her funding.
	carol    *dsl.EOA
	recorder common.Address
	// setupBlock is the first block in which every account of the workload exists and the
	// recorder has been written to.
	setupBlock uint64
	t          devtest.T
}

func newWorkload(t devtest.T, funder *dsl.FunderEOA) *workload {
	w := &workload{
		alice: funder.NewFundedEOA(eth.OneEther),
		bob:   funder.NewFundedEOA(eth.OneEther),
		carol: funder.NewFundedEOA(eth.OneEther),
		t:     t,
	}
	w.recorder = w.alice.DeployBlockRecorder()
	w.setupBlock = includedBlock(t, w.bob.Transact(w.bob.Plan(), txplan.WithTo(&w.recorder)))
	t.Logger().Info("Workload set up", "setup_block", w.setupBlock, "recorder", w.recorder)
	return w
}

// accounts are the accounts whose proofs the checks compare with the reference.
func (w *workload) accounts() []dsl.ProofAccount {
	return []dsl.ProofAccount{
		{Address: w.alice.Address()},
		{Address: w.bob.Address()},
		{Address: w.recorder, StorageKeys: []common.Hash{recorderNumberSlot, recorderTimestampSlot}},
	}
}

// driveUntilBlock sends transactions one after another until one lands in a block at or past
// target, and returns that block.
func (w *workload) driveUntilBlock(target uint64) uint64 {
	w.t.Logger().Info("Driving the chain with the workload", "target", target)
	for {
		w.alice.Transfer(w.bob.Address(), eth.OneHundredthEther)
		w.bob.Transfer(w.alice.Address(), eth.OneHundredthEther)
		block := includedBlock(w.t, w.bob.Transact(w.bob.Plan(), txplan.WithTo(&w.recorder)))
		if block >= target {
			w.t.Logger().Info("Workload reached the target", "target", target, "block", block)
			return block
		}
	}
}

// signedTransferFromCarol signs a transfer that executes against the state of any block after
// carol's funding, since she never spends her first nonce on the chain.
func (w *workload) signedTransferFromCarol(t devtest.T) []byte {
	return utils.SignedTransfer(t, w.carol, w.alice.Address(), eth.OneWei, txplan.WithStaticNonce(0))
}

func includedBlock(t devtest.T, tx *txplan.PlannedTx) uint64 {
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "transaction must be included")
	return bigs.Uint64Strict(receipt.BlockNumber)
}

// stopChainAndSettleStore stops sequencing, waits for the follower to import the head and for its
// store to flush up to it, and returns the head and the window the store serves from then on.
// With no new blocks nothing extends or prunes the window, so every check that follows sees the
// same edges.
func stopChainAndSettleStore(t devtest.T, sys *utils.MixedOpProofPreset) (eth.L2BlockRef, dsl.ProofsSyncStatus) {
	head := sys.L2ELSequencer.BlockRefByHash(sys.L2CLSequencer.StopSequencer())
	follower := sys.L2ELValidator
	follower.Reached(eth.Unsafe, head.Number, blockAttempts)
	window := follower.WaitForProofsStoreBlock(head.Number, storeCatchUpTimeout)
	t.Require().Equalf(head.Number, *window.Latest, "the store cannot be ahead of the chain head %s; window %s", head, window)
	t.Logger().Info("Chain stopped and proofs store settled", "head", head, "window", window)
	return head, window
}

func blockNumberTag(number uint64) string {
	return hexutil.Uint64(number).String()
}

// normalizedProofAt requests eth_getProof for account at blockTag from node and returns the
// outcome, with a successful proof normalized so proofs from different nodes compare equal.
func normalizedProofAt(t devtest.T, node *dsl.L2ELNode, account dsl.ProofAccount, blockTag string) (*eth.AccountResult, error) {
	storageKeys := account.StorageKeys
	if storageKeys == nil {
		storageKeys = []common.Hash{}
	}
	proof, err := node.EthClient().GetProof(t.Ctx(), account.Address, storageKeys, blockTag)
	if err != nil {
		return nil, err
	}
	dsl.NormalizeProofResponse(proof)
	return proof, nil
}

// requireProofRefused requires a proof request to have failed with an error naming block, so a
// client can tell a block outside the window from a stale or wrong proof.
func requireProofRefused(t devtest.T, err error, block uint64, request string) {
	t.Require().Errorf(err, "eth_getProof %s must be refused", request)
	namesBlock := regexp.MustCompile(`\b` + strconv.FormatUint(block, 10) + `\b`)
	t.Require().Truef(namesBlock.MatchString(err.Error()),
		"eth_getProof %s must be refused with an error naming block %d, got: %v", request, block, err)
	t.Logger().Info("eth_getProof refused as expected", "request", request, "block", block, "err", err)
}

// requireProofVerifies requires a normalized proof to verify against the state root of the block
// at number on node.
func requireProofVerifies(t devtest.T, node *dsl.L2ELNode, proof *eth.AccountResult, number uint64) {
	header, err := node.EthClient().InfoByNumber(t.Ctx(), number)
	t.Require().NoErrorf(err, "must fetch header %d", number)
	t.Require().NoErrorf(dsl.VerifyAccountProof(proof, header.Root()), "proof for %s at block %d must verify against state root %s",
		proof.Address, number, header.Root())
}

// executionWitness requests debug_executionWitness for block from node and returns the raw
// outcome.
func executionWitness(t devtest.T, node *dsl.L2ELNode, block uint64) (eth.ExecutionWitness, error) {
	var witness eth.ExecutionWitness
	err := node.EthClient().RPC().CallContext(t.Ctx(), &witness, "debug_executionWitness", blockNumberTag(block))
	return witness, err
}

func requireWitnessNonEmpty(t devtest.T, witness eth.ExecutionWitness, block uint64) {
	t.Require().NotEmptyf(witness.State, "the witness of block %d must carry state", block)
	t.Require().NotEmptyf(witness.Keys, "the witness of block %d must carry keys", block)
}
