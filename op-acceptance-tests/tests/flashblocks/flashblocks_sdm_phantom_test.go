package flashblocks

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// phantomMaxGasPerTxn caps op-rbuilder's per-tx gas used. A candidate over the cap is executed
// (warming the slots it touches) then declined — the "executed but not committed" path that leaks
// SDM warming when it isn't rolled back. Sized between the victim's gas and the poisoner's.
const phantomMaxGasPerTxn = 200_000

// The declined "poisoner" call writes a superset of the committed "victim" call's slots, so the
// victim re-touches slots that — in the canonical block — were only warmed by a tx that never
// entered the block.
const (
	poisonerSlots = 30 // run(30) ≈ 0.68M gas  > cap  => executed then declined
	victimSlots   = 4  // run(4)  ≈ 0.11M gas  < cap  => committed
)

// TestFlashblocksSDMPhantomWarmingDivergence makes the op-rbuilder phantom-warming bug observable
// at the acceptance level (ethereum-optimism/optimism#21354).
//
// SDM block-warming refunds are recorded during EVM execution (before the commit decision) and
// aren't journaled. When op-rbuilder executes a candidate then declines it (here: over
// `--builder.max_gas_per_txn`), the declined candidate's warming is (pre-fix) left behind, and a
// later committed tx touching the same slots claims a "phantom" refund for a tx that never entered
// the block. Commit-only paths (block import, `debug_replaySDMBlock` derivation) never see that
// warmth. This test drives a persistently-declined poisoner alongside committed victims and, per
// victim block, compares the producer-baked SDM payload against the commit-only derivation; they
// must be identical. Pre-fix they diverge and the failure prints the phantom entries.
func TestFlashblocksSDMPhantomWarmingDivergence(gt *testing.T) {
	t := devtest.SerialT(gt)
	sysgo.SkipOnKonaNode(t, "flashblocks acceptance preset requires user RPC")
	sysgo.SkipOnOpGeth(t, "SDM flashblocks require op-reth post-exec support")

	// Cap per-tx gas so the poisoner is declined post-execution (the warming-leak path). SDM rides
	// Interop; activating it at genesis turns SDM on across op-reth, op-rbuilder, and op-node.
	sys := presets.NewSingleChainWithFlashblocks(t,
		presets.WithInteropAtGenesis(),
		presets.WithOPRBuilderOption(sysgo.OPRBuilderNodeWithExtraArgs(
			fmt.Sprintf("--builder.max_gas_per_txn=%d", phantomMaxGasPerTxn),
		)),
	)

	driveViaTestSequencer(t, sys, 2)

	// Opt SDM PostExec production in on both the sequencer EL (op-reth fallback) and op-rbuilder;
	// the flag is process-local and starts disabled on boot. See flashblocks_sdm_test.go.
	setFlashblocksSDMEnabled(t, sys.L2EL.Escape().L2EthClient().RPC(), true)
	setFlashblocksSDMEnabled(t, sys.L2OPRBuilder.Escape().L2EthClient().RPC(), true)

	rpc := sys.L2EL.Escape().L2EthClient().RPC()

	// Deploy the StateBloat contract; run(n) writes slots [0, n).
	deployer := sys.FunderL2.NewFundedEOA(eth.OneEther)
	stateBloat := flashblocksDeployContract(t, deployer, sdm.StateBloatBin)

	// The poisoner: one persistent, top-priority call that always exceeds the gas cap, so op-rbuilder
	// re-declines it every block — warming the poisoner's slots before any victim commits. A declined
	// candidate isn't marked invalid, so it lingers in the mempool and keeps poisoning blocks. It's
	// never included, so we don't wait for it.
	poisoner := sys.FunderL2.NewFundedEOA(eth.OneEther)
	flashblocksSubmitTxWithoutWait(t, poisoner, poisoner.PendingNonce(),
		txplan.WithTo(&stateBloat),
		txplan.WithData(sdm.EncodeRun(poisonerSlots)),
		txplan.WithGasLimit(1_500_000),
		// Highest priority so op-rbuilder processes (and declines) it first in each block.
		txplan.WithGasTipCap(big.NewInt(5_000_000_000)),
		txplan.WithGasFeeCap(big.NewInt(500_000_000_000)),
	)

	// The victims: small committed calls that re-touch the poisoner's slots, sent one at a time
	// (waiting for each receipt) so each lands in its own block behind the declined poisoner.
	const victimCount = 6
	victim := sys.FunderL2.NewFundedEOA(eth.OneEther)
	victimBlocks := make([]uint64, 0, victimCount)
	seen := make(map[uint64]bool)
	for i := range victimCount {
		ptx := txplan.NewPlannedTx(
			victim.Plan(),
			txplan.WithNonce(victim.PendingNonce()),
			txplan.WithTo(&stateBloat),
			txplan.WithData(sdm.EncodeRun(victimSlots)),
			txplan.WithGasLimit(400_000),
		)
		receipt, err := ptx.Included.Eval(t.Ctx())
		t.Require().NoError(err, "victim %d must be included", i)
		bn := bigs.Uint64Strict(receipt.BlockNumber)
		if !seen[bn] {
			seen[bn] = true
			victimBlocks = append(victimBlocks, bn)
		}
	}
	slices.Sort(victimBlocks)
	t.Require().NotEmpty(victimBlocks, "expected at least one victim block")

	// Compare producer-baked vs commit-only-derived SDM payload for each victim block.
	var divergentBlocks []uint64
	var viz strings.Builder
	for _, bn := range victimBlocks {
		replay, err := sdm.ReplayBlockWithSDM(t.Ctx(), rpc, bn, true)
		t.Require().NoError(err, "debug_replaySDMBlock(%d)", bn)
		if !replay.PostExecTxPresent {
			// No SDM post-exec tx in this block (no committed warm re-touch landed here); skip.
			continue
		}

		producer := entryMap(replay.EmbeddedPayload)
		derived := entryMap(&replay.SynthesizedPayload)
		if payloadsEqual(producer, derived) {
			continue
		}
		divergentBlocks = append(divergentBlocks, bn)
		writeBlockViz(&viz, bn, replay, producer, derived)
	}

	t.Logger().Info("SDM phantom-warming scan complete",
		"victim_blocks", len(victimBlocks),
		"divergent_blocks", len(divergentBlocks))

	t.Require().Empty(divergentBlocks,
		"producer-baked SDM payload diverges from commit-only derivation in %d block(s) — "+
			"op-rbuilder credited phantom warming refunds from a declined (uncommitted) candidate "+
			"(ethereum-optimism/optimism#21354):\n%s", len(divergentBlocks), viz.String())
}

// entryMap indexes an SDM payload's refund entries by tx index. A nil payload yields an empty map.
func entryMap(p *sdm.PostExecPayload) map[uint64]uint64 {
	out := make(map[uint64]uint64)
	if p == nil {
		return out
	}
	for _, e := range p.GasRefundEntries {
		out[e.Index] = e.GasRefund
	}
	return out
}

func payloadsEqual(a, b map[uint64]uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for idx, v := range a {
		if b[idx] != v {
			return false
		}
	}
	return true
}

// writeBlockViz renders a per-tx table contrasting the producer-baked refund against the
// commit-only-derived refund, flagging phantom entries (present in the producer payload, absent or
// smaller in derivation).
func writeBlockViz(w *strings.Builder, bn uint64, replay *sdm.ReplaySDMBlock, producer, derived map[uint64]uint64) {
	fmt.Fprintf(w, "block %d (hash %s):\n", bn, replay.BlockHash)
	fmt.Fprintf(w, "  embedded(producer) entries=%d  synthesized(derivation) entries=%d  mismatches=%d\n",
		len(producer), len(derived), len(replay.Mismatches))
	fmt.Fprintf(w, "  %-9s %-18s %-18s %s\n", "tx_index", "producer_refund", "derived_refund", "")
	idxs := unionKeys(producer, derived)
	for _, idx := range idxs {
		p, pOK := producer[idx]
		d, dOK := derived[idx]
		flag := ""
		if pOK && (!dOK || d < p) {
			flag = "<- phantom"
		}
		fmt.Fprintf(w, "  %-9d %-18d %-18d %s\n", idx, p, d, flag)
	}
}

func unionKeys(a, b map[uint64]uint64) []uint64 {
	set := make(map[uint64]struct{})
	for k := range a {
		set[k] = struct{}{}
	}
	for k := range b {
		set[k] = struct{}{}
	}
	out := make([]uint64, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
