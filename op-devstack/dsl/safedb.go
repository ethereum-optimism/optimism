package dsl

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

type safeHeadDBProvider interface {
	safeHeadAtL1Block(l1BlockNum uint64) *eth.SafeHeadResponse
}

func checkSafeHeadConsistent(t testreq.TestingT, maxL1BlockNum uint64, checkNode, sourceOfTruth safeHeadDBProvider) {
	require := testreq.New(t)
	l1BlockNum := maxL1BlockNum
	matchedSomething := false
	for {

		actual := checkNode.safeHeadAtL1Block(l1BlockNum)
		if actual == nil {
			// No further safe head data available
			// Stop iterating as long as we found _some_ data
			require.Truef(matchedSomething, "no safe head data available at L1 block %v", l1BlockNum)
			return
		}

		expected := sourceOfTruth.safeHeadAtL1Block(l1BlockNum)
		require.Equalf(expected, actual, "Mismatched safe head data at l1 block %v", l1BlockNum)
		if actual.L1Block.Number == 0 {
			return // Reached L1 and L2 genesis.
		}
		l1BlockNum = actual.L1Block.Number - 1
		matchedSomething = true
	}
}
