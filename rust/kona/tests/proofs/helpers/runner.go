package helpers

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func WithPreInteropDefaults(t helpers.Testing, l2ClaimBlockNum uint64, l2 *helpers.L2Verifier, l2Eng *helpers.L2Engine) FixtureInputParam {
	return func(f *FixtureInputs) {
		// Fetch the pre and post output roots for the fault proof.
		l2PreBlockNum := l2ClaimBlockNum - 1
		if l2ClaimBlockNum == 0 {
			// If we are at genesis, we assert that we don't move the chain at all.
			l2PreBlockNum = 0
		}
		rollupClient := l2.RollupClient()
		preRoot, err := rollupClient.OutputAtBlock(t.Ctx(), l2PreBlockNum)
		require.NoError(t, err)
		claimRoot, err := rollupClient.OutputAtBlock(t.Ctx(), l2ClaimBlockNum)
		require.NoError(t, err)

		f.L2BlockNumber = l2ClaimBlockNum
		f.L2Claim = common.Hash(claimRoot.OutputRoot)
		f.L2Head = preRoot.BlockRef.Hash
		f.L2OutputRoot = common.Hash(preRoot.OutputRoot)
		f.L2ChainID = eth.ChainIDFromBig(l2.RollupCfg.L2ChainID)

		f.L2Sources = []*FaultProofProgramL2Source{
			{
				Node:        l2,
				Engine:      l2Eng,
				ChainConfig: l2Eng.L2Chain().Config(),
			},
		}
	}
}

// RunFaultProofProgram runs the fault proof program for the transition to the given L2 block number from the preceding one.
func RunFaultProofProgram(t helpers.Testing, logger log.Logger, l1 *helpers.L1Miner, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	l1Head := l1.L1Chain().CurrentBlock()

	fixtureInputs := &FixtureInputs{
		L1Head: l1Head.Hash(),
	}
	for _, apply := range fixtureInputParams {
		apply(fixtureInputs)
	}
	require.Greater(t, len(fixtureInputs.L2Sources), 0, "Must specify at least one L2 source")

	// Run the fault proof program from the state transition from L2 block l2ClaimBlockNum - 1 -> l2ClaimBlockNum.
	workDir := t.TempDir()
	fakeBeacon := fakebeacon.NewBeacon(
		logger,
		l1.BlobStore(),
		l1.L1Chain().Genesis().Time(),
		12,
	)
	require.NoError(t, fakeBeacon.Start("127.0.0.1:0"))
	defer fakeBeacon.Close()

	rollupCfgs := make([]*rollup.Config, 0, len(fixtureInputs.L2Sources))
	l1chainConfig := l1.L1Chain().Config()
	l2Endpoints := make([]string, 0, len(fixtureInputs.L2Sources))
	var closeProxies []func()
	defer func() {
		for _, closeFn := range closeProxies {
			closeFn()
		}
	}()
	for _, source := range fixtureInputs.L2Sources {
		rollupCfgs = append(rollupCfgs, source.Node.RollupCfg)
		endpoint := source.Engine.HTTPEndpoint()
		if fixtureInputs.L2RPCTracker != nil {
			proxyURL, closeFn := fixtureInputs.L2RPCTracker.StartProxy(endpoint)
			closeProxies = append(closeProxies, closeFn)
			endpoint = proxyURL
		}
		l2Endpoints = append(l2Endpoints, endpoint)
	}

	err := RunKonaNative(t, workDir, rollupCfgs, l1chainConfig, l1.HTTPEndpoint(), fakeBeacon.BeaconAddr(), l2Endpoints, *fixtureInputs)
	checkResult(t, err)
}
