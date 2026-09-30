package dsl

import (
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

	// Open the block building with the included transactions
	require.NoError(t, ca.Open(ctx))

	// Seal, sign, and commit the block
	// Commit is what makes the block canonical in the EL
	require.NoError(t, ca.Seal(ctx))
	require.NoError(t, ca.Sign(ctx))
	require.NoError(t, ca.Commit(ctx))

	// Publish is optional - it broadcasts via P2P which may not be enabled in tests.
	// The block is already committed and canonical at this point.
	_ = ca.Publish(ctx) // ignore publish errors
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
