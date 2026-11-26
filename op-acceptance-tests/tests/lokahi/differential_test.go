package lokahi

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

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
// denied block.
//
// Once both batchers stop, both sides must settle on the same local-safe and safe heads, both
// verifiers must answer supernode_syncStatus alike apart from finality, and both must decide the
// same finalized block per chain. The rollup nodes' finalized heads may differ, as described at
// [checkFinalizedHeads], but only by op-supernode's side trailing, and it must catch up once new
// L1 data arrives.
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

	// Without new batches local-safe stops on both sides.
	sys.L2BatcherA.Stop()
	sys.L2BatcherB.Stop()
	chains := []differentialChain{
		{name: "chain A", id: sys.L2A.ChainID(), lokahiCL: sys.L2ACL, supernodeCL: sys.SupernodeCLA, lokahiEL: sys.L2ELA, supernodeEL: sys.SupernodeELA},
		{name: "chain B", id: sys.L2B.ChainID(), lokahiCL: sys.L2BCL, supernodeCL: sys.SupernodeCLB, lokahiEL: sys.L2ELB, supernodeEL: sys.SupernodeELB},
	}
	for _, chain := range chains {
		awaitSameSafeHeads(t, chain)
	}
	awaitSameSyncStatus(t, sys)
	lokahiRPC, err := rpc.DialContext(ctx, sys.Lokahi.UserRPC())
	t.Require().NoError(err)
	t.Cleanup(lokahiRPC.Close)
	lokahiFinalized := make([]eth.BlockID, len(chains))
	for i, chain := range chains {
		awaitSameFinalityDecision(t, sys, lokahiRPC, chain)
		lokahiFinalized[i] = checkFinalizedHeads(t, chain)
	}
	checkSyncStatusFinality(t, sys, chains)

	// With new L1 data op-supernode's side finalizes past what lokahi's side finalized before.
	sys.L2BatcherA.Start()
	sys.L2BatcherB.Start()
	for i, chain := range chains {
		awaitSupernodeSideFinalizes(t, chain, lokahiFinalized[i])
	}
}

// differentialChain is one chain as both sides run it.
type differentialChain struct {
	name                  string
	id                    eth.ChainID
	lokahiCL, supernodeCL *dsl.L2CLNode
	lokahiEL, supernodeEL *dsl.L2ELNode
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

// awaitSameSafeHeads waits until both sides of chain report the same local-safe and safe heads, on
// the rollup nodes and the execution layers.
func awaitSameSafeHeads(t devtest.T, chain differentialChain) {
	var lokahi, supernode sideHeads
	awaitHeads(t, func(ctx context.Context) bool {
		l, ok := readSideHeads(ctx, chain.lokahiCL, chain.lokahiEL)
		if !ok {
			return false
		}
		s, ok := readSideHeads(ctx, chain.supernodeCL, chain.supernodeEL)
		if !ok {
			return false
		}
		lokahi, supernode = l, s
		return l.localSafe == s.localSafe && l.safe == s.safe && l.elSafe == s.elSafe && l.safe == l.elSafe
	}, func() string {
		return fmt.Sprintf("%s: both sides must reach the same local-safe and safe heads; lokahi side %s, op-supernode side %s",
			chain.name, lokahi, supernode)
	})
	t.Logger().Info("Both sides reached the same safe heads", "chain", chain.name, "lokahi_side", lokahi, "supernode_side", supernode)
}

// awaitSameSyncStatus waits until both verifiers answer supernode_syncStatus alike in everything
// but finality and current L1: the chains, the lowest safe and local-safe timestamps, and each
// chain's local-safe and safe heads.
func awaitSameSyncStatus(t devtest.T, sys *presets.TwoL2LokahiDifferential) {
	var diffs []string
	awaitHeads(t, func(ctx context.Context) bool {
		lokahi, err := sys.LokahiRoots.QueryAPI().SyncStatus(ctx)
		if err != nil {
			return false
		}
		supernode, err := sys.Supernode.QueryAPI().SyncStatus(ctx)
		if err != nil {
			return false
		}
		diffs = syncStatusDiffs(lokahi, supernode)
		return len(diffs) == 0
	}, func() string {
		return fmt.Sprintf("both verifiers must answer supernode_syncStatus alike: %v", diffs)
	})
	t.Logger().Info("Both verifiers answer supernode_syncStatus alike")
}

func syncStatusDiffs(lokahi, supernode eth.SuperNodeSyncStatusResponse) []string {
	var diffs []string
	if fmt.Sprint(lokahi.ChainIDs) != fmt.Sprint(supernode.ChainIDs) {
		diffs = append(diffs, fmt.Sprintf("chain IDs: lokahi %v, op-supernode %v", lokahi.ChainIDs, supernode.ChainIDs))
	}
	if lokahi.SafeTimestamp != supernode.SafeTimestamp {
		diffs = append(diffs, fmt.Sprintf("safe timestamp: lokahi %d, op-supernode %d", lokahi.SafeTimestamp, supernode.SafeTimestamp))
	}
	if lokahi.LocalSafeTimestamp != supernode.LocalSafeTimestamp {
		diffs = append(diffs, fmt.Sprintf("local-safe timestamp: lokahi %d, op-supernode %d", lokahi.LocalSafeTimestamp, supernode.LocalSafeTimestamp))
	}
	for chainID, s := range supernode.Chains {
		l, ok := lokahi.Chains[chainID]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("chain %s: only op-supernode reports it", chainID))
			continue
		}
		if l.LocalSafeL2.ID() != s.LocalSafeL2.ID() {
			diffs = append(diffs, fmt.Sprintf("chain %s local-safe: lokahi %s, op-supernode %s", chainID, l.LocalSafeL2.ID(), s.LocalSafeL2.ID()))
		}
		if l.SafeL2.ID() != s.SafeL2.ID() {
			diffs = append(diffs, fmt.Sprintf("chain %s safe: lokahi %s, op-supernode %s", chainID, l.SafeL2.ID(), s.SafeL2.ID()))
		}
	}
	return diffs
}

// awaitSameFinalityDecision waits until lokahi's finalized head of chain and the block
// op-supernode's interop activity finalizes for it at its virtual node's finalized L1 are the
// chain's safe head, so both verifiers decide the same finalized block.
func awaitSameFinalityDecision(t devtest.T, sys *presets.TwoL2LokahiDifferential, lokahiRPC *rpc.Client, chain differentialChain) {
	var lokahi, supernode, safe eth.BlockID
	awaitHeads(t, func(ctx context.Context) bool {
		status, err := chainStatus(ctx, lokahiRPC, eth.EvilChainIDToUInt64(chain.id))
		if err != nil || status.Finalized == nil {
			return false
		}
		vnStatus, err := chain.supernodeCL.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			return false
		}
		lokahi = status.Finalized.Block
		supernode, _ = sys.Supernode.VerifiedBlockAtL1(chain.id, vnStatus.FinalizedL1)
		safe = vnStatus.SafeL2.ID()
		return lokahi == supernode && supernode == safe
	}, func() string {
		return fmt.Sprintf("%s: both verifiers must finalize the safe head %s; lokahi %s, op-supernode %s",
			chain.name, safe, lokahi, supernode)
	})
	t.Logger().Info("Both verifiers finalize the same block", "chain", chain.name, "block", lokahi)
}

// checkFinalizedHeads waits until lokahi's side finalizes its safe head, and checks op-supernode's
// side against it. It returns lokahi's side's finalized head.
//
// op-node finalizes the last L2 block derived from a finalized L1 block, and skips it while that
// block is not yet safe, so op-supernode's side can trail kona's, which finalizes the verified
// head once L1 finality covers it, until new L1 data arrives.
func checkFinalizedHeads(t devtest.T, chain differentialChain) eth.BlockID {
	var lokahi sideHeads
	awaitHeads(t, func(ctx context.Context) bool {
		l, ok := readSideHeads(ctx, chain.lokahiCL, chain.lokahiEL)
		if !ok {
			return false
		}
		lokahi = l
		return l.finalized == l.safe && l.elFinalized == l.finalized
	}, func() string {
		return fmt.Sprintf("%s: lokahi's side must finalize its safe head; %s", chain.name, lokahi)
	})
	supernode, ok := readSideHeads(t.Ctx(), chain.supernodeCL, chain.supernodeEL)
	t.Require().True(ok, "%s: read op-supernode's side", chain.name)
	t.Logger().Info("Finalized heads", "chain", chain.name, "lokahi_side", lokahi, "supernode_side", supernode)
	t.Require().Equal(supernode.finalized, supernode.elFinalized, "%s: op-supernode's side's rollup node and execution layer", chain.name)
	t.Require().LessOrEqualf(supernode.finalized.Number, lokahi.finalized.Number,
		"%s: op-supernode's side must not finalize past lokahi's side; lokahi side %s, op-supernode side %s", chain.name, lokahi, supernode)
	requireCanonical(t, chain.name+": lokahi's side's finalized head on op-supernode's side", chain.supernodeEL, lokahi.finalized)
	requireCanonical(t, chain.name+": op-supernode's side's finalized head on lokahi's side", chain.lokahiEL, supernode.finalized)
	return lokahi.finalized
}

// checkSyncStatusFinality checks the finality supernode_syncStatus reports, which both verifiers
// take from the rollup nodes: op-supernode's may trail lokahi's, as at [checkFinalizedHeads], and
// each chain's finalized block is canonical on both sides.
func checkSyncStatusFinality(t devtest.T, sys *presets.TwoL2LokahiDifferential, chains []differentialChain) {
	ctx := t.Ctx()
	lokahi, err := sys.LokahiRoots.QueryAPI().SyncStatus(ctx)
	t.Require().NoError(err)
	supernode, err := sys.Supernode.QueryAPI().SyncStatus(ctx)
	t.Require().NoError(err)
	t.Logger().Info("supernode_syncStatus finality", "lokahi", lokahi.FinalizedTimestamp, "supernode", supernode.FinalizedTimestamp)
	t.Require().LessOrEqual(supernode.FinalizedTimestamp, lokahi.FinalizedTimestamp,
		"op-supernode's finalized timestamp must not pass lokahi's")
	for _, chain := range chains {
		l, s := lokahi.Chains[chain.id].FinalizedL2.ID(), supernode.Chains[chain.id].FinalizedL2.ID()
		t.Require().LessOrEqualf(s.Number, l.Number, "%s: op-supernode's finalized %s must not pass lokahi's %s", chain.name, s, l)
		requireCanonical(t, chain.name+": lokahi's syncStatus finalized on op-supernode's side", chain.supernodeEL, l)
		requireCanonical(t, chain.name+": op-supernode's syncStatus finalized on lokahi's side", chain.lokahiEL, s)
	}
}

// awaitSupernodeSideFinalizes waits until op-supernode's side of chain finalizes target, on the
// rollup node and the execution layer.
func awaitSupernodeSideFinalizes(t devtest.T, chain differentialChain, target eth.BlockID) {
	var supernode sideHeads
	awaitHeads(t, func(ctx context.Context) bool {
		s, ok := readSideHeads(ctx, chain.supernodeCL, chain.supernodeEL)
		if !ok {
			return false
		}
		supernode = s
		return s.finalized.Number >= target.Number && s.elFinalized.Number >= target.Number
	}, func() string {
		return fmt.Sprintf("%s: op-supernode's side must finalize %s once new L1 data arrives; %s", chain.name, target, supernode)
	})
	requireCanonical(t, chain.name+": lokahi's side's earlier finalized head on op-supernode's side", chain.supernodeEL, target)
	t.Logger().Info("op-supernode's side finalized past lokahi's earlier finalized head", "chain", chain.name,
		"target", target, "supernode_side", supernode)
}

// requireCanonical checks that block is el's block at its height.
func requireCanonical(t devtest.T, what string, el *dsl.L2ELNode, block eth.BlockID) {
	t.Require().Equal(block, el.BlockRefByNumber(block.Number).ID(), what)
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
