package driver

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/internal/testlog"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
)

type testID string

func (id testID) ID() eth.BlockID {
	parts := strings.Split(string(id), ":")
	if len(parts) != 2 {
		panic("bad id")
	}
	if len(parts[0]) > 32 {
		panic("test ID hash too long")
	}
	var h common.Hash
	copy(h[:], parts[0])
	v, err := strconv.ParseUint(parts[1], 0, 64)
	if err != nil {
		panic(err)
	}
	return eth.BlockID{
		Hash:   h,
		Number: v,
	}
}

type outputHandlerFn func(ctx context.Context, l2Head eth.BlockID, l2Finalized eth.BlockID, l1Window []eth.BlockID) (eth.BlockID, error)

func (fn outputHandlerFn) step(ctx context.Context, l2Head eth.BlockID, l2Finalized eth.BlockID, l1Window []eth.BlockID) (eth.BlockID, error) {
	return fn(ctx, l2Head, l2Finalized, l1Window)
}

type outputArgs struct {
	l2Head      eth.BlockID
	l2Finalized eth.BlockID
	l1Window    []eth.BlockID
}

type outputReturnArgs struct {
	l2Head eth.BlockID
	err    error
}

type stateTestCaseStep struct {
	expectedL1Head testID
	expectedL2Head testID
	expectedWindow []testID

	l1action func(t *testing.T, s *state, src *fakeChainSource, l1Heads chan eth.L1Node)
	l2action func(t *testing.T, expectedWindow []testID, s *state, src *fakeChainSource, outputIn chan outputArgs, outputReturn chan outputReturnArgs)
	reorg    bool
}

func advanceL1(t *testing.T, s *state, src *fakeChainSource, l1Heads chan eth.L1Node) {
	l1Heads <- src.advanceL1()
}

func stutterL1(t *testing.T, s *state, src *fakeChainSource, l1Heads chan eth.L1Node) {
	l1Heads <- src.l1Head()
}

func stutterAdvance(t *testing.T, s *state, src *fakeChainSource, l1Heads chan eth.L1Node) {
	l1Heads <- src.l1Head()
	l1Heads <- src.l1Head()
	l1Heads <- src.l1Head()
	l1Heads <- src.advanceL1()
	l1Heads <- src.l1Head()
	l1Heads <- src.l1Head()
	l1Heads <- src.l1Head()
}

func stutterL2(t *testing.T, expectedWindow []testID, s *state, src *fakeChainSource, outputIn chan outputArgs, outputReturn chan outputReturnArgs) {
	select {
	case <-outputIn:
		t.Error("Got a step when no step should have occurred (l1 only advance)")
	default:
	}
}

func advanceL2(t *testing.T, expectedWindow []testID, s *state, src *fakeChainSource, outputIn chan outputArgs, outputReturn chan outputReturnArgs) {
	args := <-outputIn
	assert.Equal(t, int(s.Config.SeqWindowSize), len(args.l1Window), "Invalid L1 window size")
	assert.Equal(t, len(expectedWindow), len(args.l1Window), "L1 Window size does not match expectedWindow")
	for i := range expectedWindow {
		assert.Equal(t, expectedWindow[i].ID(), args.l1Window[i], "Window elements must match")
	}
	outputReturn <- outputReturnArgs{l2Head: src.setL2Head(int(args.l2Head.Number) + 1).Self, err: nil}
}

func reorg__L2(t *testing.T, expectedWindow []testID, s *state, src *fakeChainSource, outputIn chan outputArgs, outputReturn chan outputReturnArgs) {
	args := <-outputIn
	assert.Equal(t, int(s.Config.SeqWindowSize), len(args.l1Window), "Invalid L1 window size")
	assert.Equal(t, len(expectedWindow), len(args.l1Window), "L1 Window size does not match expectedWindow")
	for i := range expectedWindow {
		assert.Equal(t, expectedWindow[i].ID(), args.l1Window[i], "Window elements must match")
	}
	src.reorgL2()
	outputReturn <- outputReturnArgs{l2Head: src.setL2Head(int(args.l2Head.Number) + 1).Self, err: nil}
}

type stateTestCase struct {
	name      string
	l1Chains  []string
	l2Chains  []string
	steps     []stateTestCaseStep
	seqWindow int
	genesis   rollup.Genesis
}

func (tc *stateTestCase) Run(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	chainSource := NewFakeChainSource(tc.l1Chains, tc.l2Chains, log)
	l1headsCh := make(chan eth.L1Node, 10)
	// Unbuffered channels to force a sync point between
	outputIn := make(chan outputArgs)
	outputReturn := make(chan outputReturnArgs)
	outputHandler := func(ctx context.Context, l2Head eth.BlockID, l2Finalized eth.BlockID, l1Window []eth.BlockID) (eth.BlockID, error) {
		outputIn <- outputArgs{l2Head: l2Head, l2Finalized: l2Finalized, l1Window: l1Window}
		r := <-outputReturn
		return r.l2Head, r.err
	}
	config := rollup.Config{SeqWindowSize: uint64(tc.seqWindow), Genesis: tc.genesis}
	state := NewState(log, config, &inputImpl{chainSource: chainSource, genesis: &tc.genesis}, outputHandlerFn(outputHandler))
	defer func() {
		assert.NoError(t, state.Close(), "Error closing state")
	}()

	err := state.Start(context.Background(), l1headsCh)
	assert.NoError(t, err, "Error starting the state object")

	for _, step := range tc.steps {
		if step.reorg {
			chainSource.reorgL1()
		}
		step.l1action(t, state, chainSource, l1headsCh)
		<-time.After(5 * time.Millisecond)
		step.l2action(t, step.expectedWindow, state, chainSource, outputIn, outputReturn)
		<-time.After(5 * time.Millisecond)

		assert.Equal(t, step.expectedL1Head.ID(), state.l1Head, "l1 head")
		assert.Equal(t, step.expectedL2Head.ID(), state.l2Head, "l2 head")
	}
}

func TestDriver(t *testing.T) {
	cases := []stateTestCase{
		{
			name:      "Simple extensions",
			l1Chains:  []string{"abcdefgh"},
			l2Chains:  []string{"ABCDEF"},
			seqWindow: 2,
			genesis:   fakeGenesis('a', 'A', 0),
			steps: []stateTestCaseStep{
				{l1action: stutterL1, l2action: stutterL2, expectedL1Head: "a:0", expectedL2Head: "A:0"},
				{l1action: advanceL1, l2action: stutterL2, expectedL1Head: "b:1", expectedL2Head: "A:0", expectedWindow: []testID{"a:0", "b:1"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "c:2", expectedL2Head: "B:1", expectedWindow: []testID{"b:1", "c:2"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "d:3", expectedL2Head: "C:2", expectedWindow: []testID{"c:2", "d:3"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "e:4", expectedL2Head: "D:3", expectedWindow: []testID{"d:3", "e:4"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "f:5", expectedL2Head: "E:4", expectedWindow: []testID{"e:4", "f:5"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "g:6", expectedL2Head: "F:5", expectedWindow: []testID{"f:5", "g:6"}},
			},
		},
		{
			name:      "Reorg",
			l1Chains:  []string{"abcdefg", "abcxyzw"},
			l2Chains:  []string{"ABCDEF", "ABCXYZ"},
			seqWindow: 2,
			genesis:   fakeGenesis('a', 'A', 0),
			steps: []stateTestCaseStep{
				{l1action: stutterL1, l2action: stutterL2, expectedL1Head: "a:0", expectedL2Head: "A:0"},
				{l1action: advanceL1, l2action: stutterL2, expectedL1Head: "b:1", expectedL2Head: "A:0", expectedWindow: []testID{"a:0", "b:1"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "c:2", expectedL2Head: "B:1", expectedWindow: []testID{"b:1", "c:2"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "d:3", expectedL2Head: "C:2", expectedWindow: []testID{"c:2", "d:3"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "e:4", expectedL2Head: "D:3", expectedWindow: []testID{"d:3", "e:4"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "f:5", expectedL2Head: "E:4", expectedWindow: []testID{"e:4", "f:5"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "g:6", expectedL2Head: "F:5", expectedWindow: []testID{"f:5", "g:6"}},
				{l1action: stutterL1, l2action: reorg__L2, expectedL1Head: "w:6", expectedL2Head: "X:3", expectedWindow: []testID{"x:3", "y:4"}, reorg: true},
				{l1action: stutterL1, l2action: advanceL2, expectedL1Head: "w:6", expectedL2Head: "Y:4", expectedWindow: []testID{"y:4", "z:5"}},
				{l1action: stutterL1, l2action: advanceL2, expectedL1Head: "w:6", expectedL2Head: "Z:5", expectedWindow: []testID{"z:5", "w:6"}},
				{l1action: stutterL1, l2action: stutterL2, expectedL1Head: "w:6", expectedL2Head: "Z:5", expectedWindow: []testID{"z:5", "w:6"}},
				{l1action: stutterL1, l2action: stutterL2, expectedL1Head: "w:6", expectedL2Head: "Z:5", expectedWindow: []testID{"z:5", "w:6"}},
			},
		},
		{
			name:      "Simple extensions with multi-step stutter",
			l1Chains:  []string{"abcdefgh"},
			l2Chains:  []string{"ABCDEF"},
			seqWindow: 2,
			genesis:   fakeGenesis('a', 'A', 0),
			steps: []stateTestCaseStep{
				{l1action: stutterL1, l2action: stutterL2, expectedL1Head: "a:0", expectedL2Head: "A:0"},
				{l1action: advanceL1, l2action: stutterL2, expectedL1Head: "b:1", expectedL2Head: "A:0", expectedWindow: []testID{"a:0", "b:1"}},
				{l1action: stutterAdvance, l2action: advanceL2, expectedL1Head: "c:2", expectedL2Head: "B:1", expectedWindow: []testID{"b:1", "c:2"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "d:3", expectedL2Head: "C:2", expectedWindow: []testID{"c:2", "d:3"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "e:4", expectedL2Head: "D:3", expectedWindow: []testID{"d:3", "e:4"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "f:5", expectedL2Head: "E:4", expectedWindow: []testID{"e:4", "f:5"}},
				{l1action: advanceL1, l2action: advanceL2, expectedL1Head: "g:6", expectedL2Head: "F:5", expectedWindow: []testID{"f:5", "g:6"}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.Run)
	}

}
