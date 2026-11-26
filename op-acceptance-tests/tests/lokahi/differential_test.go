package lokahi

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestLokahiMatchesSupernode compares lokahi with op-supernode on the same two chains and the
// same L1. lokahi follows the chains' sequencing kona-nodes; op-supernode derives the same chains
// from the same batches into execution layers of its own.
//
// After a valid and an invalid executing message, both verifiers must replace the same block with
// the same replacement block, and answer superroot_atTimestamp identically at every timestamp
// from activation to past the replacement: the super root, each chain's output root, the required
// L1 blocks, and each chain's optimistic output, which at the denied block's timestamp is the
// denied block. Once both batchers stop, both sides must settle on the same safe and finalized
// heads.
func TestLokahiMatchesSupernode(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiDifferential(t)
	ctx := t.Ctx()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	rng := rand.New(rand.NewSource(4321))
	validInit := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	validExec := bob.SendExecMessage(validInit)
	t.Logger().Info("Valid executing message included", "block", validExec.BlockID())

	invalidInit := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(invalidInit)
	denied := execMsg.BlockID()
	deniedTime := sys.L2B.TimestampForBlockNum(denied.Number)
	t.Logger().Info("Invalid executing message included", "block", denied, "timestamp", deniedTime)

	lokahiReplacement := awaitReplaced(t, sys.L2ELB, denied, 3*time.Minute)
	supernodeReplacement := awaitReplaced(t, sys.SupernodeELB, denied, 3*time.Minute)
	t.Logger().Info("Denied block replaced", "denied", denied,
		"lokahi_side", lokahiReplacement, "supernode_side", supernodeReplacement)
	t.Require().Equal(supernodeReplacement, lokahiReplacement,
		"lokahi's chain and op-supernode's chain must replace %s with the same block", denied)
	assertReplacement(t, sys.L2ELB, denied, lokahiReplacement, execMsg.TxHash())
	assertReplacement(t, sys.SupernodeELB, denied, supernodeReplacement, execMsg.TxHash())

	blockTime := sys.L2B.Escape().RollupConfig().BlockTime
	verifiedTo := deniedTime + 10*blockTime
	sys.LokahiRoots.AwaitValidatedTimestamp(verifiedTo)
	sys.Supernode.AwaitValidatedTimestamp(verifiedTo)

	compareSuperRoots(t, sys, sys.InteropActivationTime, verifiedTo)

	// At the denied block's timestamp both answer the replacement as chain B's output, and the
	// denied block as its optimistic output.
	replacementOutput, err := sys.L2BCL.Escape().RollupAPI().OutputAtBlock(ctx, lokahiReplacement.Number)
	t.Require().NoError(err)
	t.Require().Equal(lokahiReplacement.Hash, replacementOutput.BlockRef.Hash)
	chainB := sys.L2B.ChainID()
	for _, verifier := range []*dsl.Supernode{sys.LokahiRoots, sys.Supernode} {
		resp := verifier.SuperRootAtTimestamp(deniedTime)
		t.Require().NotNil(resp.Data, "%s must have verified %d", verifier, deniedTime)
		super, ok := resp.Data.Super.(*eth.SuperV1)
		t.Require().True(ok, "%s answers a %T super root", verifier, resp.Data.Super)
		var found bool
		for _, chain := range super.Chains {
			if chain.ChainID == chainB {
				found = true
				t.Require().Equal(replacementOutput.OutputRoot, chain.Output,
					"%s's super root at %d must hold the replacement's output for chain %s", verifier, deniedTime, chainB)
			}
		}
		t.Require().True(found, "%s's super root at %d must hold chain %s", verifier, deniedTime, chainB)
		optimistic, ok := resp.OptimisticAtTimestamp[chainB]
		t.Require().True(ok, "%s must answer chain %s's optimistic output at %d", verifier, chainB, deniedTime)
		t.Require().NotNil(optimistic.Output, "%s's optimistic output of chain %s at %d", verifier, chainB, deniedTime)
		t.Require().Equal(denied.Hash, optimistic.Output.BlockHash,
			"%s's optimistic output of chain %s at %d must be the denied block", verifier, chainB, deniedTime)
	}

	// Without new batches local-safe stops on both sides; then both must settle on the same safe
	// and finalized heads, which are the last derived blocks once L1 finality covers them.
	sys.L2BatcherA.Stop()
	sys.L2BatcherB.Stop()
	awaitSameHeads(t, "chain A", sys.L2ACL, sys.SupernodeCLA, sys.L2ELA, sys.SupernodeELA)
	awaitSameHeads(t, "chain B", sys.L2BCL, sys.SupernodeCLB, sys.L2ELB, sys.SupernodeELB)
}

// compareSuperRoots checks that lokahi and op-supernode answer superroot_atTimestamp identically at
// every timestamp in [from, to]. It reports every differing timestamp before it fails.
func compareSuperRoots(t devtest.T, sys *presets.TwoL2LokahiDifferential, from, to uint64) {
	var mismatches []string
	for ts := from; ts <= to; ts++ {
		lokahi := sys.LokahiRoots.SuperRootAtTimestamp(ts)
		supernode := sys.Supernode.SuperRootAtTimestamp(ts)
		for _, diff := range superRootDiffs(lokahi, supernode) {
			mismatches = append(mismatches, fmt.Sprintf("timestamp %d: %s", ts, diff))
		}
	}
	for _, m := range mismatches {
		t.Logger().Error("lokahi and op-supernode differ", "diff", m)
	}
	t.Require().Emptyf(mismatches, "lokahi and op-supernode answer %d of the timestamps %d to %d differently; first: %s",
		len(mismatches), from, to, firstOr(mismatches))
	t.Logger().Info("lokahi and op-supernode answer every timestamp identically", "from", from, "to", to)
}

// superRootDiffs lists how lokahi's answer differs from op-supernode's.
func superRootDiffs(lokahi, supernode eth.SuperRootAtTimestampResponse) []string {
	var diffs []string
	if (lokahi.Data == nil) != (supernode.Data == nil) {
		return []string{fmt.Sprintf("data present: lokahi %t, op-supernode %t", lokahi.Data != nil, supernode.Data != nil)}
	}
	if lokahi.Data != nil {
		if lokahi.Data.SuperRoot != supernode.Data.SuperRoot {
			diffs = append(diffs, fmt.Sprintf("super root: lokahi %s, op-supernode %s", lokahi.Data.SuperRoot, supernode.Data.SuperRoot))
		}
		if l, s := lokahi.Data.Super.Marshal(), supernode.Data.Super.Marshal(); string(l) != string(s) {
			diffs = append(diffs, fmt.Sprintf("super: lokahi %x, op-supernode %x", l, s))
		}
		if lokahi.Data.VerifiedRequiredL1 != supernode.Data.VerifiedRequiredL1 {
			diffs = append(diffs, fmt.Sprintf("verified required L1: lokahi %s, op-supernode %s",
				lokahi.Data.VerifiedRequiredL1, supernode.Data.VerifiedRequiredL1))
		}
	}
	if fmt.Sprint(lokahi.ChainIDs) != fmt.Sprint(supernode.ChainIDs) {
		diffs = append(diffs, fmt.Sprintf("chain IDs: lokahi %v, op-supernode %v", lokahi.ChainIDs, supernode.ChainIDs))
	}
	for chainID, s := range supernode.OptimisticAtTimestamp {
		l, ok := lokahi.OptimisticAtTimestamp[chainID]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("chain %s: optimistic output only from op-supernode", chainID))
			continue
		}
		if l.OutputRoot != s.OutputRoot {
			diffs = append(diffs, fmt.Sprintf("chain %s: optimistic output root: lokahi %s, op-supernode %s", chainID, l.OutputRoot, s.OutputRoot))
		}
		if l.RequiredL1 != s.RequiredL1 {
			diffs = append(diffs, fmt.Sprintf("chain %s: optimistic required L1: lokahi %s, op-supernode %s", chainID, l.RequiredL1, s.RequiredL1))
		}
		if (l.Output == nil) != (s.Output == nil) || (l.Output != nil && l.Output.BlockHash != s.Output.BlockHash) {
			diffs = append(diffs, fmt.Sprintf("chain %s: optimistic block: lokahi %v, op-supernode %v", chainID, blockHash(l), blockHash(s)))
		}
	}
	for chainID := range lokahi.OptimisticAtTimestamp {
		if _, ok := supernode.OptimisticAtTimestamp[chainID]; !ok {
			diffs = append(diffs, fmt.Sprintf("chain %s: optimistic output only from lokahi", chainID))
		}
	}
	return diffs
}

func blockHash(out eth.OutputWithRequiredL1) any {
	if out.Output == nil {
		return nil
	}
	return out.Output.BlockHash
}

func firstOr(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return items[0]
}

// awaitSameHeads waits until lokahi's side and op-supernode's side of a chain report the same
// local-safe and safe heads, on the rollup nodes and the execution layers, and then until both
// finalize up to that safe head.
func awaitSameHeads(t devtest.T, name string, lokahiCL, supernodeCL *dsl.L2CLNode, lokahiEL, supernodeEL *dsl.L2ELNode) {
	// lokahi and supernode hold the last heads both sides answered, for the failure message.
	var lokahi, supernode sideHeads
	read := func(ctx context.Context) bool {
		l, ok := readSideHeads(ctx, lokahiCL, lokahiEL)
		if !ok {
			return false
		}
		s, ok := readSideHeads(ctx, supernodeCL, supernodeEL)
		if !ok {
			return false
		}
		lokahi, supernode = l, s
		return true
	}
	awaitHeads(t, func(ctx context.Context) bool {
		return read(ctx) && lokahi.localSafe == supernode.localSafe && lokahi.safe == supernode.safe &&
			lokahi.elSafe == supernode.elSafe && lokahi.safe == lokahi.elSafe
	}, func() string {
		return fmt.Sprintf("%s: both sides must reach the same local-safe and safe heads; lokahi side %s, op-supernode side %s",
			name, lokahi, supernode)
	})
	t.Logger().Info("Both sides reached the same safe heads", "chain", name, "lokahi_side", lokahi, "supernode_side", supernode)
	awaitHeads(t, func(ctx context.Context) bool {
		return read(ctx) && lokahi.finalized == lokahi.safe && supernode.finalized == supernode.safe &&
			lokahi.elFinalized == lokahi.finalized && supernode.elFinalized == supernode.finalized
	}, func() string {
		return fmt.Sprintf("%s: both sides must finalize their safe head; lokahi side %s, op-supernode side %s",
			name, lokahi, supernode)
	})
	t.Logger().Info("Both sides finalized the same heads", "chain", name, "lokahi_side", lokahi, "supernode_side", supernode)
}

// sideHeads are one side's heads of a chain: its rollup node's and its execution layer's.
type sideHeads struct {
	localSafe, safe, finalized eth.BlockID
	elSafe, elFinalized        eth.BlockID
}

func (h sideHeads) String() string {
	return fmt.Sprintf("{local-safe %s safe %s finalized %s, EL safe %s finalized %s}",
		h.localSafe, h.safe, h.finalized, h.elSafe, h.elFinalized)
}

func readSideHeads(ctx context.Context, cl *dsl.L2CLNode, el *dsl.L2ELNode) (sideHeads, bool) {
	status, err := cl.Escape().RollupAPI().SyncStatus(ctx)
	if err != nil {
		return sideHeads{}, false
	}
	elSafe, err := el.Escape().EthClient().BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return sideHeads{}, false
	}
	elFinalized, err := el.Escape().EthClient().BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return sideHeads{}, false
	}
	return sideHeads{
		localSafe:   status.LocalSafeL2.ID(),
		safe:        status.SafeL2.ID(),
		finalized:   status.FinalizedL2.ID(),
		elSafe:      elSafe.ID(),
		elFinalized: elFinalized.ID(),
	}, true
}

// awaitHeads polls until done holds, for at most three minutes, and fails with describe otherwise.
func awaitHeads(t devtest.T, done func(ctx context.Context) bool, describe func() string) {
	ctx, cancel := context.WithTimeout(t.Ctx(), 3*time.Minute)
	defer cancel()
	err := wait.For(ctx, 2*time.Second, func() (bool, error) {
		return done(ctx), nil
	})
	t.Require().NoError(err, describe())
}
