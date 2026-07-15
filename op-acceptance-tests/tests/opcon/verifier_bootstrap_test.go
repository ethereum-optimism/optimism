package opcon

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// These tests pin op-con-node's Direct Sync BOOTSTRAP path: a verifier that
// joins LATE, after the sequencer's bounded signed replay ring has evicted the
// early chain, must still build the full unsafe chain from the sequencer's
// payload websocket alone. Bootstrap, live-follow, and recovery are one
// protocol: the consumer's sync engine opens a cursor subscription at its next
// block, is rejected below the ring's signed horizon (-32020), falls back to
// CATCH-UP range pulls (opql_getUnsafePayloads) through the sequencer's
// UNSIGNED EL-reconstructed cold tier, chains into the signed ring tail, and
// hands off to LIVE. The trust boundary is the producer's safe claim
// (opql_unsafePayloadsInfo.safeHead): unsigned payloads are admitted only at
// heights covered by L1 derivation; above it a sequencer signature is
// mandatory. Both tests only make sense with op-con-node on both ends (ring,
// cold tier, and --follow are opql-specific), so they skip on other CL kinds.

// opConBootstrapRingBlocks shrinks the sequencer's signed replay ring
// (--sequencer-payload-ring-blocks) so a short devnet chain already outgrows
// it: with ~10+ blocks produced and only the newest 4 retained signed, the
// late joiner's cursor (block 1) is guaranteed to fall below the signed
// horizon and the cold-tier bootstrap ladder is exercised.
const opConBootstrapRingBlocks = 4

// TestOpConVerifierBootstrapsBelowRingHorizon is the CL-only bootstrap
// flagship: a fresh op-con-node verifier joins a chain that has grown well
// past the sequencer's replay ring, with the batcher running from genesis so
// the producer's safe claim covers the evicted history by join time. The
// verifier's opening subscribe is rejected below the signed horizon; it must
// CATCH-UP through the unsigned cold tier (admitted because those heights are
// under the safe claim), chain into the signed ring tail, go LIVE, and keep up
// with the sequencer's tip — and the block that was the sequencer's tip at
// join time must be canonical (not reorged) on the verifier's EL, which
// transitively pins the entire bootstrapped history.
func TestOpConVerifierBootstrapsBelowRingHorizon(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node Direct Sync bootstrap (signed replay ring + cold tier + --follow)")
	sys := presets.NewSingleChainOpConBootstrapWithoutCheck(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOpConPayloadRingBlocks(opConBootstrapRingBlocks)),
	)

	// Grow the chain well past the ring depth AND let the batcher's L1
	// derivation advance the sequencer's safe claim over the evicted span
	// (blocks below unsafe-tip minus ring), so the cold tier's unsigned history
	// is admissible when the verifier joins.
	dsl.CheckAll(t,
		sys.L2CL.ReachedFn(types.LocalUnsafe, 10, 30),
		sys.L2CL.ReachedFn(types.LocalSafe, 6, 60),
	)

	// The sequencer's tip at join time: the verifier must recover everything up
	// to (and past) this block, starting from genesis.
	joinRef := sys.L2EL.BlockRefByLabel(eth.Unsafe)

	sys.AddWSFollowVerifier(t)

	// The verifier bootstraps the whole span and keeps up with live production:
	// past the join-time tip plus a few live blocks, within a generous budget.
	sys.L2CLB.Reached(types.LocalUnsafe, joinRef.Number+3, 45)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 30)

	// The join-time tip is canonical on the verifier's EL — the bootstrapped
	// history is the sequencer's chain, not a fork (parent-hash links pin every
	// ancestor below it).
	sys.L2ELB.VerifyNotReorged(joinRef)
}

// TestOpConVerifierUnsignedPacedBySafeClaim pins the bootstrap trust boundary,
// live: unsigned cold-tier payloads are only admitted up to the producer's
// claimed safe head. The batcher starts STOPPED, so the sequencer's safe head
// stays at genesis while its unsafe chain grows past the ring — ALL evicted
// history is unsigned-above-claim. A late-joining verifier must therefore
// HOLD at genesis (its sync engine re-polls the cursor rather than ingest
// unsigned blocks it cannot attribute). Once the batcher is started and
// batches land on L1, the producer's safe claim advances, the held history
// becomes admissible, and the verifier must resume and reach the tip.
//
// This is the property that makes cold-tier bootstrap safe: without the
// safe-claim gate, a compromised (or wrong-chain) websocket source could feed
// a fresh verifier an arbitrary unsigned history.
func TestOpConVerifierUnsignedPacedBySafeClaim(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node Direct Sync safe-claim pacing (unsigned cold tier held above the claim)")
	sys := presets.NewSingleChainOpConBootstrapWithoutCheck(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOpConPayloadRingBlocks(opConBootstrapRingBlocks)),
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			// No batches on L1: the sequencer's safe claim stays at genesis, so
			// every cold-tier payload the late joiner pulls is unsigned above it.
			cfg.Stopped = true
		}),
	)
	require := t.Require()

	// Outgrow the ring with derivation dark: unsafe advances, safe stays 0.
	sys.L2CL.Reached(types.LocalUnsafe, 10, 30)
	require.Zero(sys.L2CL.HeadBlockRef(types.LocalSafe).Number,
		"sequencer safe head must stay at genesis while the batcher is stopped")

	sys.AddWSFollowVerifier(t)

	// The verifier must HOLD: its cursor is below the signed horizon, and every
	// cold-tier payload is unsigned above the (genesis) safe claim, so nothing
	// may reach its EL for a sustained window (~14s, 7 blocks of production).
	sys.L2ELB.NotAdvancedUnsafe(7)
	require.Zero(sys.L2CLB.HeadBlockRef(types.LocalUnsafe).Number,
		"verifier must not ingest unsigned payloads above the producer's safe claim")

	// Open the L1 backstop: batches land, the sequencer's safe claim advances
	// over the held history.
	sys.L2Batcher.Start()
	sys.L2CL.Advanced(types.LocalSafe, 4, 60)

	// The verifier's held cursor resumes as the claim advances: it admits the
	// now-covered unsigned history, chains into the signed ring, and reaches
	// the sequencer's tip.
	tipRef := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	sys.L2CLB.Reached(types.LocalUnsafe, tipRef.Number, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 30)
}
