package dsl

import (
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type TestSequencer struct {
	commonImpl

	inner stack.TestSequencer
}

func NewTestSequencer(inner stack.TestSequencer) *TestSequencer {
	return &TestSequencer{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (s *TestSequencer) String() string {
	return s.inner.Name()
}

func (s *TestSequencer) Escape() stack.TestSequencer {
	return s.inner
}

// SequenceBlock builds a block at deterministic timestamp (parent.Time + blockTime).
// This is useful for tests that need predictable block timestamps.
func (s *TestSequencer) SequenceBlock(t devtest.T, chainID eth.ChainID, parent common.Hash) {
	ca := s.Escape().ControlAPI(chainID)

	require.NoError(t, ca.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent}))
	require.NoError(t, ca.Next(t.Ctx()))
}

// BuildL1UntilSafeOriginPast sequences L1 blocks one at a time until l2EL's safe head has an L1
// origin above startL1, and returns that safe head so a test can reorg the L1 block it was derived
// from. The caller stops L1's own block production first, so this loop is the only L1 producer.
func (s *TestSequencer) BuildL1UntilSafeOriginPast(t devtest.T, l1ChainID eth.ChainID, l2EL *L2ELNode, startL1 uint64) eth.L2BlockRef {
	ca := s.Escape().ControlAPI(l1ChainID)
	var safe eth.L2BlockRef
	t.Require().Eventuallyf(func() bool {
		// Next tolerates a build job left behind by a timed-out attempt, where New would report
		// a conflict. Errors are retried rather than fatal: a transient engine stall must not
		// end the test.
		if err := ca.Next(t.Ctx()); err != nil {
			t.Logger().Warn("L1 block build failed, retrying", "err", err)
			return false
		}
		ref, err := l2EL.blockRefByLabel(eth.Safe)
		if err != nil {
			t.Logger().Warn("L2 safe head read failed, retrying", "err", err)
			return false
		}
		safe = ref
		t.Logger().Info("Built L1 block", "l2_safe", safe, "l2_safe_l1_origin", safe.L1Origin, "start_l1", startL1)
		return safe.Number > 0 && safe.L1Origin.Number > startL1
	}, 120*time.Second, 2*time.Second, "L2 safe head never reached an L1 origin above %d", startL1)
	return safe
}

// AdvanceToBlockParent sequences empty blocks until the next block lands at timestamp.
// It returns that next block's parent.
func (s *TestSequencer) AdvanceToBlockParent(t devtest.T, network *L2Network, timestamp uint64) eth.L2BlockRef {
	for {
		parent := network.PrimaryEL().BlockRefByLabel(eth.Unsafe)
		nextTimestamp := network.TimestampForBlockNum(parent.Number + 1)
		t.Require().LessOrEqual(nextTimestamp, timestamp,
			"chain %s advanced past target timestamp %d", network.ChainID(), timestamp)
		if nextTimestamp == timestamp {
			return parent
		}
		s.SequenceBlock(t, network.ChainID(), parent.Hash)
	}
}

// SequenceBlockWithTxs builds a block with timestamp parent.Time + blockTime with the supplied transactions (bypassing the mempool).
// This makes it ideal for same-timestamp interop testing, and avoids the chance that transactions are sequenced into later blocks.
func (s *TestSequencer) SequenceBlockWithTxs(t devtest.T, chainID eth.ChainID, parent common.Hash, rawTxs [][]byte) {
	ctx := t.Ctx()
	ca := s.Escape().ControlAPI(chainID)

	// Start a new block building job
	require.NoError(t, ca.New(ctx, seqtypes.BuildOpts{Parent: parent}))

	// Include each transaction BEFORE opening
	// IncludeTx adds to the job's attrs.Transactions which are used when Open() starts block building
	for _, rawTx := range rawTxs {
		require.NoError(t, ca.IncludeTx(ctx, hexutil.Bytes(rawTx)))
	}

	// Next opens, seals, signs, commits and publishes the job started above, then resets the
	// sequencer so the next block on this chain can start a new job.
	require.NoError(t, ca.Next(ctx))
}

// SequenceBlockWithPlannedTxs assigns consecutive nonces and includes the planned transactions in one block.
func (s *TestSequencer) SequenceBlockWithPlannedTxs(
	t devtest.T,
	parent common.Hash,
	sender *EOA,
	planned ...*txplan.PlannedTx,
) {
	s.log.Info("sequencing block with planned transactions",
		"chain_id", sender.ChainID(),
		"transaction_count", len(planned),
	)
	baseNonce := sender.PendingNonce()
	rawTxs := make([][]byte, len(planned))
	for i, plannedTx := range planned {
		txplan.WithStaticNonce(baseNonce + uint64(i))(plannedTx)
		signed, err := plannedTx.Signed.Eval(t.Ctx())
		t.Require().NoError(err, "failed to sign transaction %d", i)
		rawTxs[i], err = signed.MarshalBinary()
		t.Require().NoError(err, "failed to marshal transaction %d", i)
	}
	s.SequenceBlockWithTxs(t, sender.ChainID(), parent, rawTxs)
}
