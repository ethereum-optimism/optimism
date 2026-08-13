package derivation

import (
	"math/big"
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// TestValidatorRecoversFromInvalidSpanBatch verifies a validator recovers after its execution
// layer rejects derived attributes from a span batch. The acceptance suite runs this test with
// both op-node and kona-node through DEVSTACK_L2CL_KIND.
func TestValidatorRecoversFromInvalidSpanBatch(gt *testing.T) {
	t := devtest.ParallelT(gt)
	activationDelta := uint64(1)
	var engineFault *sysgo.EngineFaultProxy
	sys := presets.NewSingleChainMultiNodeWithoutP2PWithoutCheck(t,
		presets.WithDeployerOptions(
			sysgo.WithHardforkSequentialActivation(forks.Ecotone, forks.Ecotone, &activationDelta),
		),
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			cfg.Stopped = true
		}),
		presets.WithGlobalL2CLOption(sysgo.L2CLSequencerMaxSafeLag(6)),
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, target sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			if target.Name != "b" {
				return
			}
			sysgo.L2CLEngineRPCProxy(func(engineRPC string) string {
				engineFault = sysgo.StartEngineFaultProxy(p, target.String(), engineRPC)
				return engineFault.URL()
			}).Apply(p, target, cfg)
		})),
	)

	t.Require().NotNil(engineFault)
	sys.L2CL.Reached(safety.LocalUnsafe, 6, 30)
	originalHeadHash := sys.L2CL.StopSequencer()
	originalHead := sys.L2EL.BlockRefByHash(originalHeadHash)
	validPrefix := sys.L2EL.BlockRefByNumber(originalHead.Number - 1)
	// Feed every prefix block explicitly so the test does not depend on P2P peer-connection
	// timing. Leave the validator one block behind so deriving the invalid final block exercises
	// the force-build path.
	for blockNumber := uint64(1); blockNumber <= validPrefix.Number; blockNumber++ {
		sys.L2CLB.SignalTarget(sys.L2EL, blockNumber)
	}
	sys.L2CLB.ReachedRef(safety.LocalUnsafe, validPrefix.ID(), 30)

	// Diverge from the original unsafe chain at the penultimate block. The transaction only needs
	// to make the derived attributes differ; it is signed independently of the EL and need not be
	// accepted by the sequencer's tx pool.
	divergentKey, err := crypto.GenerateKey()
	t.Require().NoError(err)
	divergentAddr := crypto.PubkeyToAddress(divergentKey.PublicKey)
	divergentTx, err := types.SignNewTx(divergentKey, types.LatestSigner(sys.L2Chain.Escape().ChainConfig()), &types.DynamicFeeTx{
		ChainID:   sys.L2Chain.ChainID().ToBig(),
		Nonce:     0,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: big.NewInt(2_000_000_000),
		Gas:       21_000,
		To:        &divergentAddr,
	})
	t.Require().NoError(err)
	validBatch := sys.L2Chain.NewSpanBatch(1, originalHead.Number)
	invalidBatch := sys.L2Chain.NewSpanBatch(1, originalHead.Number,
		dsl.WithSpanBatchTransaction(originalHead.Number-1, divergentTx))

	engineFault.InvalidateNextForkchoiceWithAttributes()
	invalidBatch.Submit()
	t.Require().Eventually(func() bool {
		return engineFault.InvalidForkchoiceCount() == 1
	}, 30*time.Second, 100*time.Millisecond, "validator must process the injected INVALID response")
	// Reject the same final attributes again if the client retries them unchanged. The fixed client
	// resets instead; the old kona-node loops on this response and never reaches the valid batch.
	engineFault.ReinjectLastInvalidForkchoiceWithAttributes()

	// The rejected final block must restore the backed-up original unsafe branch. A subsequent
	// valid batch then proves derivation can continue and promote that branch to local-safe.
	validBatch.Submit()
	sys.L2CLB.ReachedRef(safety.LocalSafe, originalHead.ID(), 30)
}
