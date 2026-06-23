package sdm

import (
	"testing"

	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// buildSDMPremiumStandardSystem mirrors buildSDMRethSystem (Interop/SDM at genesis, stock op-reth
// verifier) but boots op-reth-premium as the sequencer EL in its NON-subblocks mode
// (--subblocks.enable=false). In that mode premium drives the standard OpPayloadBuilder against the
// premium SDM EVM config, so it produces ordinary blocks with a trailing 0x7D — no flashblocks. The
// stock op-reth verifier derives them by the consensus rule alone (it never opts in).
//
// Skips when the premium binary is absent (it lives in a separate repo); see premiumBinaryAvailable.
func buildSDMPremiumStandardSystem(t devtest.T) *sdmRethSystem {
	// --subblocks.enable=false selects premium's standard SDM payload path. This is a premium-only
	// flag, so it must be a per-node option: a stock op-reth verifier would reject it.
	return buildSDMPremiumSystem(t, []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs("--subblocks.enable=false")})
}

// buildSDMPremiumSystem boots op-reth-premium as the SDM sequencer EL (Interop/SDM at genesis, opted
// in) alongside a stock op-reth verifier. seqExtraOpts are applied only to the premium sequencer.
// With no extra opts, premium runs its DEFAULT subblocks producer (--subblocks.enable=true) — the
// MODE A path; with --subblocks.enable=false it runs the standard payload builder — the MODE B path.
// Either way the canonical block carries one trailing 0x7D, which the stock verifier derives.
//
// Skips when the premium binary is absent (it lives in a separate repo); see premiumBinaryAvailable.
func buildSDMPremiumSystem(t devtest.T, seqExtraOpts []sysgo.OpRethOption) *sdmRethSystem {
	sysgo.SkipOnOpGeth(t, "SDM acceptance tests require op-reth post-exec support")
	if !premiumBinaryAvailable() {
		t.Skip("op-reth-premium binary not provided; set RUST_BINARY_PATH_OP_RETH_PREMIUM (or RUST_SRC_DIR_OP_RETH_PREMIUM + RUST_JIT_BUILD=1)")
	}

	clKind := sysgo.ResolveMixedL2CLKind()
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth-premium",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpRethPremium,
				CLKind:      clKind,
				IsSequencer: true,
				OpRethOpts:  seqExtraOpts,
			},
			{
				ELKey:       "verifier-op-reth",
				CLKey:       "verifier",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      clKind,
				IsSequencer: false,
			},
		},
		InteropAtGenesis: true,
	})
	return finishSDMRethSystem(t, runtime, true)
}

// TestSDMPremiumStandardSequencerProducesAndDerives is the MODE B (non-subblocks) acceptance gate:
// op-reth-premium runs as the SDM sequencer with --subblocks.enable=false, produces a standard block
// carrying exactly one trailing 0x7D with real warming refunds, and a stock op-reth verifier derives
// that block (and blocks after it) from L1. It mirrors TestSDMPostExecBlockDerivesAndChainProgresses
// but swaps the sequencer EL for op-reth-premium, proving the premium binary produces SDM blocks the
// public stack can verify without premium code.
//
// Run it by providing the premium binary, e.g.:
//
//	RUST_BINARY_PATH_OP_RETH_PREMIUM=/path/to/op-reth-premium \
//	  go test ./op-acceptance-tests/tests/sdm/ -run TestSDMPremiumStandardSequencerProducesAndDerives
func TestSDMPremiumStandardSequencerProducesAndDerives(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := buildSDMPremiumStandardSystem(t)
	assertPremiumSDMProducesAndDerives(t, sys, "non-subblocks")
}

// TestSDMPremiumSubblocksSequencerProducesAndDerives is the MODE A (subblocks) acceptance gate:
// op-reth-premium runs as the SDM sequencer in its DEFAULT subblocks mode (--subblocks.enable=true),
// so the canonical block's trailing 0x7D is appended by the subblocks producer's seal path (not the
// standard OpPayloadBuilder). It asserts the same consensus-visible invariant as MODE B — exactly
// one trailing 0x7D with real refunds, derived by a stock op-reth verifier — over the subblocks
// production path. (The per-subblock streamed post_exec_tx is asserted by premium's own producer
// unit tests; this gate covers the canonical block the public stack derives.)
//
//	RUST_BINARY_PATH_OP_RETH_PREMIUM=/path/to/op-reth-premium \
//	  go test ./op-acceptance-tests/tests/sdm/ -run TestSDMPremiumSubblocksSequencerProducesAndDerives
func TestSDMPremiumSubblocksSequencerProducesAndDerives(gt *testing.T) {
	t := devtest.SerialT(gt)
	// No extra opts → premium's default --subblocks.enable=true (the subblocks producer).
	sys := buildSDMPremiumSystem(t, nil)
	assertPremiumSDMProducesAndDerives(t, sys, "subblocks")
}

// assertPremiumSDMProducesAndDerives drives a repeated-slot warming workload through a premium SDM
// sequencer, asserts the produced block carries exactly one trailing 0x7D with non-empty refunds,
// and that a stock op-reth verifier derives that block (and a later sentinel block) from L1. Shared
// by the MODE A (subblocks) and MODE B (standard) gates — the only difference is the producer path
// inside premium; the consensus-visible result is identical.
func assertPremiumSDMProducesAndDerives(t devtest.T, sys *sdmRethSystem, mode string) {
	verifyOpReth(t, sys.L2EL)
	verifyOpReth(t, sys.L2ELVerifier)

	// Repeated-slot workload generates warming refunds, so the produced block must carry a 0x7D.
	block, included, targetBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().NotEmpty(included, "target block must include workload transactions")
	postExecTx, postExecPos := findPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "premium SDM sequencer must include a post-exec tx before batching")
	t.Require().Greater(len(postExecTx.Input), 0, "post-exec tx input must not be empty")
	// The 0x7D must be the final tx; since findPostExecTransaction returns the first match, a
	// trailing position proves there is exactly one and it is last (the consensus layout rule).
	t.Require().Equal(len(block.Transactions)-1, postExecPos, "post-exec tx must be the single trailing tx")

	payload, err := sdmpkg.DecodePayload(postExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode")
	t.Require().Equal(targetBlockNum, payload.BlockNumber, "post-exec payload must target the produced block")
	t.Require().NotEmpty(payload.GasRefundEntries, "post-exec payload must be non-empty for repeated-slot workload")
	targetRef := sys.L2EL.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(block.Hash, targetRef.Hash, "selected post-exec block hash must match canonical sequencer block")

	// Land a sentinel tx after the post-exec block so derivation must progress past it.
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	sentinel := txplan.NewPlannedTx(
		alice.Plan(),
		txplan.WithTo(addrPtr(common.HexToAddress("0x000000000000000000000000000000000000dEaD"))),
		txplan.WithValue(eth.OneHundredthEther),
	)
	sentinelReceipt, err := sentinel.Included.Eval(t.Ctx())
	t.Require().NoError(err, "sentinel tx after the post-exec block must be included")
	t.Require().Equal(types.ReceiptStatusSuccessful, sentinelReceipt.Status, "sentinel tx must succeed")
	sentinelBlockNum := bigs.Uint64Strict(sentinelReceipt.BlockNumber)
	t.Require().Greater(sentinelBlockNum, targetBlockNum,
		"sentinel tx must land after the post-exec block so derivation proves chain progress past it")
	sentinelRef := sys.L2EL.BlockRefByNumber(sentinelBlockNum)

	// Batch to L1; the stock verifier must derive the premium-produced post-exec block and beyond.
	sys.L2Batcher.Start()
	dsl.CheckAll(t,
		sys.L2CL.ReachedRefFn(safety.CrossSafe, sentinelRef.ID(), 120),
		sys.L2CLVerifier.ReachedRefFn(safety.CrossSafe, sentinelRef.ID(), 120),
		sys.L2EL.ReachedFn(eth.Safe, sentinelBlockNum, 120),
		sys.L2ELVerifier.ReachedFn(eth.Safe, sentinelBlockNum, 120),
	)

	verifierPostExecRef := sys.L2ELVerifier.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(targetRef.Hash, verifierPostExecRef.Hash,
		"stock verifier must derive the same post-exec block as the premium sequencer")
	verifierSentinelRef := sys.L2ELVerifier.BlockRefByNumber(sentinelBlockNum)
	t.Require().Equal(sentinelRef.Hash, verifierSentinelRef.Hash,
		"verifier must derive blocks after the post-exec block")

	assertPremiumPayloadMatchesPublicReplay(t, sys, targetBlockNum, targetRef.Hash, postExecPos, payload)

	var totalSDMRefund uint64
	for _, entry := range payload.GasRefundEntries {
		totalSDMRefund += entry.GasRefund
	}
	t.Logger().Info("premium SDM block produced and derived",
		"mode", mode,
		"post_exec_block", targetBlockNum,
		"post_exec_block_hash", targetRef.Hash,
		"post_exec_tx_index", postExecPos,
		"payload_entries", len(payload.GasRefundEntries),
		"total_sdm_refund", totalSDMRefund,
		"sentinel_block", sentinelBlockNum)
}

func assertPremiumPayloadMatchesPublicReplay(
	t devtest.T,
	sys *sdmRethSystem,
	blockNum uint64,
	blockHash common.Hash,
	postExecPos int,
	payload *sdmpkg.PostExecPayload,
) {
	replay := replayBlockWithSDM(t, sys.L2ELVerifier, blockNum)
	t.Require().Equal(blockNum, replay.BlockNum, "public replay must target the premium-produced block")
	t.Require().Equal(blockHash, replay.BlockHash, "public replay block hash must match premium block")
	t.Require().True(replay.PostExecTxPresent, "public replay must report the premium post-exec tx")
	t.Require().NotNil(replay.PostExecTxIndex, "public replay must report the post-exec tx index")
	t.Require().Equal(uint64(postExecPos), *replay.PostExecTxIndex,
		"public replay post-exec tx index must match the premium payload")
	t.Require().Empty(replay.Mismatches,
		"premium post-exec payload must match public SDM replay")
	t.Require().NotNil(replay.EmbeddedPayload, "public replay must decode the embedded premium payload")
	t.Require().Equal(payload, replay.EmbeddedPayload,
		"public replay embedded payload must match the premium sequencer payload")
	t.Require().Equal(*payload, replay.SynthesizedPayload,
		"premium post-exec payload must equal the public SDM replay-synthesized payload")
	t.Require().Equal(len(payload.GasRefundEntries), replay.Summary.PostExecPayloadEntryCount,
		"replay summary payload entry count must match the premium payload")
	t.Require().Equal(replay.Summary.ReplayRefundTotal, replay.Summary.PayloadRefundTotal,
		"public replay and premium payload refund totals must match")
}
