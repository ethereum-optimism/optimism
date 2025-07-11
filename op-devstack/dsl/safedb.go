package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

type safeHeadDBProvider interface {
	safeHeadAtL1Block(l1BlockNum uint64) *eth.SafeHeadResponse
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
