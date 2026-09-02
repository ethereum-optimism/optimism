package helpers

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// WithSuperDefaults prepares the interop (super-root) inputs for the state transition into
// l2ClaimBlockNum.
//
// kona-host super proves one sub-transition at a time, and the derivation step of a timestamp
// advances the active chain by exactly one L2 block (see sub_transition's
// disputed_l2_block_number). So the agreed prestate here is the single-chain super root at the
// timestamp one second before the claimed block's timestamp — which resolves to block
// l2ClaimBlockNum-1, because TargetBlockNumber rounds down — and the honest claim is the
// commitment to the transition state after step 0 appended the claimed block.
//
// Only step 0 is proven. Steps 1..StepsPerTimestamp-2 are padding that the client short-circuits,
// and the consolidation step is covered against a real dependency set by the ConsolidateStep cases
// in op-acceptance-tests' superfaultproofs suite; re-proving it for every transition here would
// double the suite's kona-host invocations for a single-chain no-op.
func WithSuperDefaults(t helpers.Testing, l2ClaimBlockNum uint64, l2 *helpers.L2Verifier, l2Eng *helpers.L2Engine) FixtureInputParam {
	return func(f *FixtureInputs) {
		cfg := l2.RollupCfg
		require.Greater(t, l2ClaimBlockNum, cfg.Genesis.L2.Number,
			"cannot prove the transition into L2 genesis: there is no super root before the L2 genesis timestamp")

		claimTimestamp := cfg.TimestampForBlock(l2ClaimBlockNum)

		rollupClient := l2.RollupClient()
		preRoot, err := rollupClient.OutputAtBlock(t.Ctx(), l2ClaimBlockNum-1)
		require.NoError(t, err)
		claimRoot, err := rollupClient.OutputAtBlock(t.Ctx(), l2ClaimBlockNum)
		require.NoError(t, err)

		chainID := eth.ChainIDFromBig(cfg.L2ChainID)
		agreed := eth.NewSuperV1(claimTimestamp-1, eth.ChainIDAndOutput{
			ChainID: chainID,
			Output:  preRoot.OutputRoot,
		})
		optimistic := eth.OptimisticBlock{
			BlockHash:  claimRoot.BlockRef.Hash,
			OutputRoot: claimRoot.OutputRoot,
		}
		// With a one-chain dependency set there is no second chain to derive, so the honest
		// post-state of step 0 is the transition state holding just this chain's optimistic block.
		claimed := &eth.TransitionState{
			SuperRoot:       agreed.Marshal(),
			PendingProgress: []eth.OptimisticBlock{optimistic},
			Step:            1,
		}

		f.L2BlockNumber = l2ClaimBlockNum
		f.ClaimTimestamp = claimTimestamp
		f.AgreedPrestate = agreed.Marshal()
		f.L2Claim = claimed.Hash()
		f.L2ChainID = chainID

		f.L2Sources = []*FaultProofProgramL2Source{
			{
				Node:        l2,
				Engine:      l2Eng,
				ChainConfig: l2Eng.L2Chain().Config(),
			},
		}
	}
}

// WithSupernode points the kona-sp1 super-range executor at an HTTP endpoint serving
// `superroot_atTimestamp`. Ignored by the native fault-proof program, which is handed its agreed
// pre-state and claim directly.
func WithSupernode(address string) FixtureInputParam {
	return func(f *FixtureInputs) {
		f.SupernodeAddress = address
	}
}

// ProgramRunner runs a fault-proof program (or its SP1 equivalent) for one state transition against
// the prepared chain inputs, returning ErrClaimNotValid when the claim is rejected.
type ProgramRunner func(t helpers.Testing, workDir string, rollupCfgs []*rollup.Config, l1chainConfig *params.ChainConfig, l1Rpc string, l1BeaconRpc string, l2Rpcs []string, fixtureInputs FixtureInputs) error

// RunFaultProofProgram runs the native fault proof program (kona-host super, i.e. the interop
// client program) for the transition to the given L2 block number from the preceding one.
func RunFaultProofProgram(t helpers.Testing, logger log.Logger, l1 *helpers.L1Miner, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	runProgram(t, logger, l1, RunKonaSuperNative, false, checkResult, fixtureInputParams...)
}

// RunSP1SuperRangeProgram runs the kona-sp1 super-range guest in SP1 execute mode over the
// super-root transition to the given L2 block number from the preceding one.
func RunSP1SuperRangeProgram(t helpers.Testing, logger log.Logger, l1 *helpers.L1Miner, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	runProgram(t, logger, l1, RunSuperRangeExecutor, true, checkResult, fixtureInputParams...)
}

// runProgram prepares the chain inputs (beacon, configs, L2 endpoints) for a single state
// transition and dispatches them to the given program runner. allowSP1Options must be true only
// for runners that honor SP1-only fixture options (currently the SP1 range executor).
func runProgram(t helpers.Testing, logger log.Logger, l1 *helpers.L1Miner, run ProgramRunner, allowSP1Options bool, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	l1Head := l1.L1Chain().CurrentBlock()

	fixtureInputs := &FixtureInputs{
		L1Head: l1Head.Hash(),
	}
	for _, apply := range fixtureInputParams {
		apply(fixtureInputs)
	}
	require.Greater(t, len(fixtureInputs.L2Sources), 0, "Must specify at least one L2 source")
	// SP1-only fixture options are ignored by the native fault-proof program, so passing them to
	// RunFaultProofProgram is a mistake that would silently pass. Fail loudly.
	require.False(t, (fixtureInputs.CorruptClaim || fixtureInputs.SP1NativeCore) && !allowSP1Options,
		"SP1-only fixture options are only honored by RunSP1SuperRangeProgram; the native fault-proof program ignores them")

	// Run the program from the state transition from L2 block l2ClaimBlockNum - 1 -> l2ClaimBlockNum.
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

	err := run(t, workDir, rollupCfgs, l1chainConfig, l1.HTTPEndpoint(), fakeBeacon.BeaconAddr(), l2Endpoints, *fixtureInputs)
	checkResult(t, err)
}
