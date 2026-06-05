package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

type safeHeadDBProvider interface {
	safeHeadAtL1Block(l1BlockNum uint64) *eth.SafeHeadResponse
}

// safeHeadDBFloor walks the safe head db backwards from maxL1BlockNum and returns the lowest
// L2 safe-head block number that still has a recorded entry, i.e. the oldest point the db has
// data for. The returned value is aligned to the db's per-L1-block granularity, which makes it
// a stable baseline: unlike the live safe head (which advances continuously, possibly mid-L1-block),
// the floor only changes when the db is reset/truncated. Capture it before an action and re-check
// afterwards to assert the db was not truncated. Fails the test if the db has no data at all.
func safeHeadDBFloor(t devtest.T, maxL1BlockNum uint64, node safeHeadDBProvider) uint64 {
	require := testreq.New(t)
	l1BlockNum := maxL1BlockNum
	var floor *uint64
	for {
		actual := node.safeHeadAtL1Block(l1BlockNum)
		if actual == nil {
			require.NotNil(floor, "no safe head data available at or below L1 block %v", maxL1BlockNum)
			return *floor
		}
		floor = &actual.SafeHead.Number
		if actual.L1Block.Number == 0 {
			return *floor // Reached L1 and L2 genesis.
		}
		l1BlockNum = actual.L1Block.Number - 1
	}
}

func checkSafeHeadConsistent(t devtest.T, maxL1BlockNum uint64, checkNode, sourceOfTruth safeHeadDBProvider, minRequiredL2Block *uint64) {
	require := testreq.New(t)
	l1BlockNum := maxL1BlockNum
	var minL2BlockRecorded *uint64
	for {
		actual := checkNode.safeHeadAtL1Block(l1BlockNum)
		if actual == nil {
			// No further safe head data available
			// Stop iterating as long as we found _some_ data
			require.NotNil(minL2BlockRecorded, "no safe head data available at L1 block %v", l1BlockNum)
			if minRequiredL2Block != nil {
				// Ensure we had data back at least as far as minRequiredL2Block
				require.LessOrEqual(*minL2BlockRecorded, *minRequiredL2Block, "safe head db did not go back far enough")
			}
			return
		}

		expected := sourceOfTruth.safeHeadAtL1Block(l1BlockNum)
		require.Equalf(expected, actual, "Mismatched safe head data at l1 block %v", l1BlockNum)
		if actual.L1Block.Number == 0 {
			return // Reached L1 and L2 genesis.
		}
		l1BlockNum = actual.L1Block.Number - 1
		minL2BlockRecorded = &actual.SafeHead.Number
	}
}
