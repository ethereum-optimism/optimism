// Dafny model for the Interop activity in interop.go.

include "VerifiedDB.dfy"
include "ChainContainer.dfy"
include "LogsDB.dfy"

module Interop {
  import opened Types
  import opened Utils
  import opened VerifiedDB
  import opened ChainContainer
  import opened LogsDB

  class Interop {

    var currentL1: BlockID
    const chains: map<ChainID, ChainContainer>
    const verifiedDB: VerifiedDB
    const activationTimestamp: nat
    const logsDBs: map<ChainID, LogsDB>

    constructor(chains: map<ChainID, ChainContainer>)
      requires chains.Keys == CHAIN_IDS
      ensures Valid()
      ensures this.chains == chains
    {
      var chainIDs := Enumerate(CHAIN_IDS);
      var logsDBs: map<ChainID, LogsDB> := map[];

      for i := 0 to |chainIDs|
        invariant forall k :: k in logsDBs <==> k in chainIDs[0..i]
        invariant forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==>
          logsDBs[k1] != logsDBs[k2]
        invariant forall chainID :: chainID in logsDBs ==>
          logsDBs[chainID].LatestSealedBlock() == None
      {
        var db := new LogsDB();
        logsDBs := logsDBs[chainIDs[i] := db];
      }

      this.verifiedDB := new VerifiedDB();
      this.activationTimestamp := ACTIVATION_TIMESTAMP;
      this.currentL1 := BlockID(0, 0);
      this.chains := chains;
      this.logsDBs := logsDBs;
    }

    // ========================================================================
    // Functions
    // ========================================================================

    // The block in the logsDB for chainID corresponding to the given timestamp.
    // The verifiedDB is used to get the block number corresponding to the timestamp,
    // which is then used to query the logsDB.
    ghost function SealedBlockForVerifiedAtTimestamp(chainID: ChainID, ts: nat) : Option<BlockSeal>
      reads verifiedDB, logsDBs[chainID]
      requires chainID in logsDBs.Keys
      requires verifiedDB.Has(ts)
      requires chainID in verifiedDB.Get(ts).l2Heads
    {
      var verifiedHeads := verifiedDB.Get(ts).l2Heads;
      logsDBs[chainID].FindSealedBlock(verifiedHeads[chainID].number)
    }

    // The timestamp that will be verified in the next round: the successor of the
    // last committed timestamp, or activationTimestamp if the verifiedDB is empty.
    ghost function NextTimestamp(): nat
      reads verifiedDB
      requires verifiedDB.Valid()
    {
      match verifiedDB.LastTimestamp() {
        case None => activationTimestamp
        case Some(ts) => ts + 1
      }
    }

    // ========================================================================
    // Predicates
    // ========================================================================

    // Main invariants of the Interop class that should be maintained throughout the
    // execution. Does not include consistency between the logsDBs and the verifiedDB,
    // which can be temporarily violated (see PendingTransitionIsConsistent for those).
    ghost predicate Valid()
      reads verifiedDB
    {
      activationTimestamp == ACTIVATION_TIMESTAMP &&
      chains.Keys == CHAIN_IDS &&

      /* LogsDBs invariants */
      // There is one logsDB for each chain.
      logsDBs.Keys == CHAIN_IDS &&
      // All logsDBs are distinct.
      (forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]) &&

      /* VerifiedDB invariants */
      verifiedDB.Valid() &&
      // If any timestamp has been verified, the activation timestamp must be in the DB.
      (verifiedDB.lastTimestamp.Some? ==> activationTimestamp in verifiedDB.db) &&
      // All committed timestamps are at or above the activation timestamp.
      (forall ts :: ts in verifiedDB.db ==> activationTimestamp <= ts) &&
      // Every committed verified result covers exactly the current set of chains.
      (forall ts :: ts in verifiedDB.db ==> verifiedDB.db[ts].l2Heads.Keys == CHAIN_IDS) &&
      // The pending transition is valid, if it exists.
      (verifiedDB.pendingTransition.Some? ==> ValidPendingTransition(verifiedDB.GetPendingTransition().value))
    }

    // The logsDB for chainID is in sync with the verifiedDB up to the given timestamp:
    // every L2 block up to this timestamp in the verifiedDB is also present in the logsDB
    // for the corresponding chain.
    ghost predicate DBsInSyncUpTo(chainID: ChainID, upperTS: nat)
      reads verifiedDB, logsDBs[chainID]
      requires chainID in logsDBs.Keys
    {
      forall t :: activationTimestamp <= t <= upperTS ==>
        verifiedDB.Has(t) &&
        chainID in verifiedDB.Get(t).l2Heads &&
        SealedBlockForVerifiedAtTimestamp(chainID, t).Some? &&
        SealedBlockForVerifiedAtTimestamp(chainID, t).value.id == verifiedDB.Get(t).l2Heads[chainID]
    }

    // The logsDB for chainID is in sync with the verifiedDB: the latest sealed
    // block in the logsDB matches the l2Head for chainID in the latest verified result,
    // and all previous blocks also agree.
    ghost predicate DBsInSync(chainID: ChainID)
      reads verifiedDB, logsDBs[chainID]
      requires verifiedDB.Valid()
      requires chainID in logsDBs.Keys
    {
      match verifiedDB.LastTimestamp() {
        case None =>
          logsDBs[chainID].LatestSealedBlock() == None
        case Some(ts) =>
          var l2Heads := verifiedDB.Get(ts).l2Heads;
          chainID in l2Heads &&
          logsDBs[chainID].LatestSealedBlock() == Some(l2Heads[chainID]) &&
          DBsInSyncUpTo(chainID, ts)
      }
    }

    // Lifted versions of DBsInSyncUpTo / DBsInSync that quantify over all chains.
    // Replace verbose forall patterns in method specifications.
    ghost predicate AllDBsInSyncUpTo(upper: nat)
      reads verifiedDB, logsDBs.Values
    {
      forall k :: k in logsDBs.Keys ==> DBsInSyncUpTo(k, upper)
    }

    ghost predicate AllDBsInSync()
      reads verifiedDB, logsDBs.Values
      requires verifiedDB.Valid()
    {
      forall k :: k in logsDBs.Keys ==> DBsInSync(k)
    }

    // Consistency of the rewind plan with the current verifiedDB state.
    // The target timestamp must be present in the DB, must be the maximum key
    // below rewindAtOrAfter (so it becomes the new last timestamp after the rewind),
    // and the target heads must match the verified result at that timestamp.
    ghost predicate PlanConsistentWithVerified(plan: RewindPlan)
      reads verifiedDB
    {
      plan.resetAllChainsTo.Some? ==>
        var ts := plan.resetAllChainsTo.value;
        verifiedDB.Has(ts) &&
        (forall t :: 0 <= t < plan.rewindAtOrAfter && verifiedDB.Has(t) ==> t <= ts) &&
        plan.targetHeads == verifiedDB.Get(ts).l2Heads
    }

    // Consistency of the rewind plan with the logsDB for a given chain.
    // None case: always consistent — Clear() has no preconditions.
    // Some case: the target block is already sealed in the logsDB, satisfying
    // the precondition of LogsDB.Rewind.
    ghost predicate PlanConsistentWithLogs(plan: RewindPlan, chainID: ChainID)
      reads logsDBs[chainID]
      requires chainID in logsDBs.Keys
      requires plan.resetAllChainsTo.Some? ==> chainID in plan.targetHeads
    {
      plan.resetAllChainsTo.Some? ==>
        var sealedBlock := logsDBs[chainID].FindSealedBlock(plan.targetHeads[chainID].number);
        sealedBlock.Some? &&
        sealedBlock.value.id == plan.targetHeads[chainID]
    }

    // The verifiedDB is in the expected state after being rewound according to the
    // given rewind plan, which must be the same plan stored in the pending transition.
    ghost predicate RewoundVerifiedDB(plan: RewindPlan)
      reads verifiedDB
      requires PlanConsistentWithVerified(plan)
    {
      verifiedDB.pendingTransition.Some? &&
      verifiedDB.pendingTransition.value.decision == Rewind &&
      verifiedDB.pendingTransition.value.rewind == Some(plan) &&
      match plan.resetAllChainsTo {
        case None =>
          |verifiedDB.db| == 0
        case Some(ts) =>
          verifiedDB.LastTimestamp() == Some(ts) &&
          plan.targetHeads == verifiedDB.Get(ts).l2Heads
      }
    }

    // The logsDB for the given chainID is in the expected state after being rewound
    // according to the given rewind plan.
    ghost predicate RewoundLogsDB(plan: RewindPlan, chainID: ChainID)
      reads logsDBs[chainID]
      requires chainID in logsDBs.Keys
      requires plan.resetAllChainsTo.Some? ==> chainID in plan.targetHeads
      requires PlanConsistentWithLogs(plan, chainID)
    {
      match plan.resetAllChainsTo {
        case None =>
          logsDBs[chainID].LatestSealedBlock() == None
        case Some(_) =>
          logsDBs[chainID].LatestSealedBlock() == Some(plan.targetHeads[chainID])
      }
    }

    // The given pending transition is consistent with the current state of the verifiedDB,
    // so that it can be successfully applied by ApplyPendingTransition.
    ghost predicate TransitionConsistentWithVerified(pending: PendingTransition)
      reads verifiedDB
      requires verifiedDB.Valid()
      requires ValidPendingTransition(pending)
    {
      match pending.decision {
        case Rewind =>
          PlanConsistentWithVerified(pending.rewind.value)
        case Invalidate =>
          true
        case Advance =>
          var newTimestamp := pending.result.value.timestamp;
          var newL2Heads := pending.result.value.l2Heads;
          AdvancesVerifiedDB(newTimestamp, newL2Heads)
      }
    }

    // The given pending transition is consistent with the current state of the logsDBs,
    // so that it can be successfully applied by ApplyPendingTransition.
    ghost predicate TransitionConsistentWithLogs(pending: PendingTransition)
      reads logsDBs.Values
      requires ValidPendingTransition(pending)
    {
      match pending.decision {
        case Rewind =>
          forall k :: k in logsDBs.Keys && pending.rewind.value.resetAllChainsTo.Some? ==>
            k in pending.rewind.value.targetHeads &&
            PlanConsistentWithLogs(pending.rewind.value, k)
        case Invalidate =>
          true
        case Advance =>
          var newTimestamp := pending.result.value.timestamp;
          var newL2Heads := pending.result.value.l2Heads;
          newL2Heads.Keys == logsDBs.Keys &&
          AdvancesAllLogsDBs(newTimestamp, newL2Heads)
      }
    }

    // All preconditions needed to call ApplyPendingTransition in either the
    // crash-recovery path (pending transition already stored) or the fresh path
    // (no pending transition, AllDBsInSync required for ProgressInterop).
    // ValidPendingTransition is not included because it is already implied by Valid().
    ghost predicate PendingTransitionIsConsistent()
      reads verifiedDB, logsDBs.Values
      requires Valid()
    {
      match verifiedDB.GetPendingTransition() {
        case None =>
          AllDBsInSync()
        case Some(p) =>
          TransitionConsistentWithVerified(p) &&
          TransitionConsistentWithLogs(p) &&
          match p.decision {
            case Rewind =>
              p.rewind.value.resetAllChainsTo.Some? ==>
                AllDBsInSyncUpTo(p.rewind.value.resetAllChainsTo.value)
            case Invalidate =>
              AllDBsInSync()
            case Advance =>
              verifiedDB.LastTimestamp().Some? ==>
                AllDBsInSyncUpTo(verifiedDB.LastTimestamp().value)
          }
      }
    }

    // Consistency of the step output with the current verifiedDB state:
    // the previous DB entry exists when rewinding past the activation timestamp,
    // the timestamp matches the next sequential slot when advancing or invalidating,
    // and the new l2Heads advance monotonically by at most one block when advancing.
    ghost predicate OutputConsistentWithVerified(output: StepOutput, obs: RoundObservation)
      reads verifiedDB
      requires verifiedDB.Valid()
      requires ValidStepOutput(output, obs)
    {
      match output {
        case WaitOutput =>
          true
        case RewindOutput =>
          activationTimestamp < obs.lastVerifiedTS.value ==>
            obs.lastVerifiedTS.value - 1 in verifiedDB.db
        case AdvanceOutput(result) =>
          AdvancesVerifiedDB(result.timestamp, result.l2Heads)
        case InvalidateOutput(result) =>
          result.timestamp == NextTimestamp()
      }
    }

    // Consistency of the step output with the current logsDB state:
    // the new l2Heads are compatible with each chain's current tip when advancing,
    // and the rewind target block is already sealed in each logsDB when rewinding.
    ghost predicate OutputConsistentWithLogs(output: StepOutput, obs: RoundObservation)
      reads verifiedDB, logsDBs.Values
      requires Valid()
      requires ValidStepOutput(output, obs)
      requires OutputConsistentWithVerified(output, obs)
    {
      match output {
        case WaitOutput => true
        case RewindOutput =>
          forall k :: k in logsDBs.Keys && activationTimestamp < obs.lastVerifiedTS.value ==>
            var sealedBlock := SealedBlockForVerifiedAtTimestamp(k, obs.lastVerifiedTS.value - 1);
            sealedBlock.Some? &&
            sealedBlock.value.id == verifiedDB.Get(obs.lastVerifiedTS.value - 1).l2Heads[k]
        case AdvanceOutput(result) =>
          AdvancesAllLogsDBs(result.timestamp, result.l2Heads)
        case InvalidateOutput(_) => true
      }
    }

    // Consistency of the round observation with the current verifiedDB state:
    // the last-verified timestamp and next-timestamp fields mirror the DB,
    // the previous DB entry exists when rewinding past the activation timestamp,
    // and the observed block heads advance monotonically by at most one block.
    ghost predicate ObservationConsistentWithVerified(obs: RoundObservation)
      reads verifiedDB
      requires verifiedDB.Valid()
      requires ValidRoundObservation(obs)
    {
      obs.lastVerifiedTS == verifiedDB.lastTimestamp &&
      obs.nextTimestamp == NextTimestamp() &&
      (!obs.l1Consistent && activationTimestamp < obs.lastVerifiedTS.value ==>
        obs.lastVerifiedTS.value - 1 in verifiedDB.db) &&
      (obs.chainsReady && obs.l2sConsistent && obs.l1Consistent ==>
        AdvancesVerifiedDB(obs.nextTimestamp, obs.blocksAtTS))
    }

    // Consistency of the round observation with the current logsDB state:
    // the rewind target block is already sealed in each logsDB when l1 is
    // inconsistent, and the observed block heads are compatible with each
    // chain's current tip when chains are ready.
    ghost predicate ObservationConsistentWithLogs(obs: RoundObservation)
      reads verifiedDB, logsDBs.Values
      requires Valid()
      requires ValidRoundObservation(obs)
      requires ObservationConsistentWithVerified(obs)
    {
      (!obs.l1Consistent && obs.lastVerifiedTS.value > activationTimestamp ==>
        forall k :: k in logsDBs.Keys ==>
          var sealedBlock := SealedBlockForVerifiedAtTimestamp(k, obs.lastVerifiedTS.value - 1);
          sealedBlock.Some? &&
          sealedBlock.value.id == verifiedDB.Get(obs.lastVerifiedTS.value - 1).l2Heads[k]) &&
      (obs.chainsReady && obs.l2sConsistent && obs.l1Consistent ==>
        AdvancesAllLogsDBs(obs.nextTimestamp, obs.blocksAtTS))
    }

    // The given timestamp and frontier blocks are a valid sequence to the last entry
    // currently in the verifiedDB.
    ghost predicate AdvancesVerifiedDB(ts: nat, blocksAtTS: map<ChainID, BlockID>)
      reads verifiedDB
      requires verifiedDB.Valid()
    {
      match verifiedDB.LastTimestamp() {
        case None =>
          ts == activationTimestamp
        case Some(lastTS) =>
          ts == lastTS + 1 &&
          var lastVerifiedResult := verifiedDB.Get(lastTS);
          blocksAtTS.Keys == lastVerifiedResult.l2Heads.Keys &&
          forall chainID :: chainID in blocksAtTS.Keys ==>
            var lastBlock := lastVerifiedResult.l2Heads[chainID];
            lastBlock.number <= blocksAtTS[chainID].number <= lastBlock.number + 1
      }
    }

    // The given timestamp and block are a valid sequence to the last entry currently
    // in the logsDB for the given chain.
    ghost predicate AdvancesLogsDB(ts: nat, chainID: ChainID, newBlock: BlockID)
      reads logsDBs[chainID]
      requires chainID in logsDBs
    {
      match logsDBs[chainID].LatestSealedBlock() {
        case None =>
          ts == activationTimestamp
        case Some(latestBlock) =>
          latestBlock.number <= newBlock.number <= latestBlock.number + 1 &&
          (latestBlock.number == newBlock.number ==> latestBlock == newBlock)
      }
    }

    // The given timestamp and frontier blocks are a valid sequence to the current
    // last entries of each logsDB.
    ghost predicate AdvancesAllLogsDBs(ts: nat, blocksAtTS: map<ChainID, BlockID>)
      reads logsDBs.Values
      requires blocksAtTS.Keys == logsDBs.Keys
    {
      forall chainID :: chainID in logsDBs.Keys ==>
        AdvancesLogsDB(ts, chainID, blocksAtTS[chainID])
    }

    // ========================================================================
    // Methods
    // ========================================================================

    // Attempts to advance the verified timestamp and persist the result.
    // Returns None if an I/O operation failed (pending transition preserved for retry),
    // Some(true) if the verified timestamp was advanced, Some(false) otherwise.
    // Corresponds to progressAndRecord in interop.go.
    method {:isolate_assertions} ProgressAndRecord() returns (madeProgress: Option<bool>)
      modifies this, verifiedDB, chains.Values, logsDBs.Values
      requires Valid()
      requires PendingTransitionIsConsistent()
      ensures Valid()
      ensures PendingTransitionIsConsistent()
    {
      // Crash-recovery path: resume an interrupted transition if one was
      // persisted from a previous run.
      var pending := verifiedDB.GetPendingTransition();

      if pending.Some? {
        madeProgress := ApplyPendingTransition(pending.value);
        return;
      }

      // Observe the current round state and decide on the next action.
      var output, obs := ProgressInterop();

      if output.WaitOutput? {
        // Chains not yet ready; refresh L1 state while idle.
        RefreshCurrentL1OnWait();
        madeProgress := Some(false);
        return;
      }

      // Persist the transition as a WAL entry before applying it, so that a
      // crash between the two steps is recoverable on the next startup.
      var pendingTx := BuildPendingTransition(output, obs);

      // Snapshot the heap before SetPendingTransition so that the twostate lemma
      // below can reference the pre-set state where AllDBsInSync() holds.
      label BeforeSetPending:
      verifiedDB.SetPendingTransition(pendingTx);

      madeProgress := ApplyPendingTransition(pendingTx) by {
        // Prove ApplyPendingTransition's preconditions

        // SetPendingTransition only changes pendingTransition; db and lastTimestamp
        // are unchanged. Re-derive DBsInSync for all chains via the framing lemma.
        forall k | k in logsDBs.Keys ensures DBsInSync(k) {
          ClearPendingPreservesDBsInSync@BeforeSetPending(k);
        }

        // For the Rewind case: AllDBsInSync() with LastTimestamp() == Some(ts) implies
        // AllDBsInSyncUpTo(ts' ) for any ts' <= ts. Apply the helper lemma if needed.
        if pendingTx.decision == Rewind && pendingTx.rewind.value.resetAllChainsTo.Some? {
          var ts := pendingTx.rewind.value.resetAllChainsTo.value;
          // PlanConsistentWithVerified guarantees verifiedDB.Has(ts), hence ts <= LastTimestamp().value.
          AllDBsInSyncImpliesAllDBsInSyncUpTo(ts);
        }
      }
    }

    // Observes the current round state, runs verification if chains are ready,
    // and returns the resulting decision together with the observation snapshot.
    // Does not modify the verified DB.
    // Corresponds to progressInterop in interop.go.
    method {:isolate_assertions} ProgressInterop() returns (output: StepOutput, obs: RoundObservation)
      requires Valid()
      requires AllDBsInSync()
      ensures Valid()
      ensures AllDBsInSync()
      ensures ValidStepOutput(output, obs)
      ensures OutputConsistentWithVerified(output, obs)
      ensures OutputConsistentWithLogs(output, obs)
    {
      obs := ObserveRound();

      if !obs.chainsReady {
        output := WaitOutput;
        return;
      }
      if !obs.l2sConsistent {
        output := WaitOutput;
        return;
      }
      if !obs.l1Consistent {
        output := RewindOutput;
        return;
      }

      var result := Verify(obs.nextTimestamp, obs.blocksAtTS);
      if result.l2Heads == map[] {
        output := WaitOutput;
      } else if result.invalidHeads != map[] {
        output := InvalidateOutput(result);
      } else {
        output := AdvanceOutput(result);
      }
    }

    // Captures a consistent snapshot of the current round state by reading the
    // verified DB and querying all chains for their status at the next timestamp.
    // Corresponds to observeRound in interop.go.
    method {:isolate_assertions} ObserveRound() returns (obs: RoundObservation)
      requires Valid()
      requires AllDBsInSync()
      ensures Valid()
      ensures AllDBsInSync()
      ensures ValidRoundObservation(obs)
      ensures ObservationConsistentWithVerified(obs)
      ensures ObservationConsistentWithLogs(obs)
    {
      // Read the last verified timestamp and its result from the DB.
      var lastTS := verifiedDB.LastTimestamp();

      if lastTS.Some? {
        var lastResult := verifiedDB.Get(lastTS.value);
        obs := RoundObservation(
          lastVerifiedTS := Some(lastTS.value),
          lastVerified   := Some(lastResult),
          nextTimestamp  := lastTS.value + 1,
          chainsReady    := false,
          blocksAtTS     := map[],
          l1Heads        := map[],
          l1Consistent   := true,
          l2sConsistent  := true
        );
      } else {
        obs := RoundObservation(
          lastVerifiedTS := None,
          lastVerified   := None,
          nextTimestamp  := activationTimestamp,
          chainsReady    := false,
          blocksAtTS     := map[],
          l1Heads        := map[],
          l1Consistent   := true,
          l2sConsistent  := true
        );
      }

      // Check whether all chains have data at the next timestamp.
      var ready := CheckChainsReady(obs.nextTimestamp);

      if ready.None? {
        return;
      }

      obs := obs.(
        chainsReady := true,
        blocksAtTS  := ready.value.blocks,
        l1Heads     := ready.value.l1Heads
      );

      // Check that all L1 heads are on the same canonical fork.
      var l1Consistent, l2sConsistent := CheckL1Consistent(obs.l1Heads, obs.lastVerified);
      obs := obs.(l1Consistent := l1Consistent, l2sConsistent := l2sConsistent);

      assert ObservationConsistentWithVerified(obs) by {
        // Help Dafny prove the following ObservationConsistentWithVerified conjunct:
        // obs.lastVerifiedTS.value - 1 is in the DB when lastVerifiedTS.value > activationTimestamp.
        // Follows from activationTimestamp in verifiedDB.db (Interop.Valid) + Sequential.
        if obs.lastVerifiedTS.Some? && obs.lastVerifiedTS.value > activationTimestamp {
          SequentialContainsRange(verifiedDB.db, activationTimestamp);
        }
      }
    }

    // Checks whether all chains have a block at the given timestamp.
    // Returns None if any chain is not yet ready, Some with the per-chain blocks
    // and L1 inclusion heads otherwise.
    // Corresponds to checkChainsReady in interop.go; the parallel fan-out is
    // abstracted away as a sequential loop, and the ethereum.NotFound error is
    // replaced by an Option return.
    method {:isolate_assertions} CheckChainsReady(ts: nat) returns (result: Option<ChainsReadyResult>)
      requires Valid()
      requires AllDBsInSync()
      requires forall k :: k in logsDBs && logsDBs[k].LatestSealedBlock().None? ==>
        ts == activationTimestamp
      ensures Valid()
      ensures result.Some? ==> result.value.blocks.Keys == chains.Keys
      ensures result.Some? ==> AdvancesAllLogsDBs(ts, result.value.blocks)
    {
      var blocks: map<ChainID, BlockID> := map[];
      var l1Heads: map<ChainID, BlockID> := map[];
      var chainIDs := Enumerate(chains.Keys);

      for i := 0 to |chainIDs|
        invariant Valid()
        invariant forall k :: k in blocks.Keys <==> k in chainIDs[0..i]
        invariant forall k :: k in logsDBs && logsDBs[k].LatestSealedBlock().None? ==>
          ts == activationTimestamp
        invariant forall k :: k in chainIDs[0..i] ==>
          AdvancesLogsDB(ts, k, blocks[k])
      {
        var chainID := chainIDs[i];
        var chainResult := chains[chainID].OptimisticAt(ts);
        if chainResult.None? {
          result := None;
          return;
        }
        blocks  := blocks[chainID  := chainResult.value.l2Block];
        l1Heads := l1Heads[chainID := chainResult.value.l1Head];

        if logsDBs[chainID].LatestSealedBlock().Some? {
          var latestBlock := logsDBs[chainID].LatestSealedBlock().value;
          var newBlock := chainResult.value.l2Block;
          assume latestBlock.number <= newBlock.number <= latestBlock.number + 1;
          assume latestBlock.number == newBlock.number ==> latestBlock == newBlock;
        }

        assert AdvancesLogsDB(ts, chainID, chainResult.value.l2Block);
      }

      result := Some(ChainsReadyResult(blocks, l1Heads));
    }

    // Verifies cross-chain messages and checks for cycles at the given timestamp,
    // returning the combined result.
    // Corresponds to verify in interop.go. The frontierView setup
    // (resolveFrontierVerificationView) is abstracted away.
    method Verify(ts: nat, blocksAtTS: map<ChainID, BlockID>) returns (result: Result)
      requires Valid()
      ensures Valid()
      ensures result.timestamp == ts
      ensures result.l2Heads == blocksAtTS
    {
      result := VerifyMessages(ts, blocksAtTS);
      var cycleResult := VerifyCycles(ts, blocksAtTS);
      result := result.(invalidHeads := result.invalidHeads + cycleResult.invalidHeads);
    }

    // Applies a pending transition and returns whether the verified timestamp
    // was advanced.
    // Corresponds to applyPendingTransition in interop.go. commitVerifiedResult
    // and ToVerifiedResult are inlined.
    method {:isolate_assertions} ApplyPendingTransition(pending: PendingTransition) returns (madeProgress: Option<bool>)
      modifies this, verifiedDB, chains.Values, logsDBs.Values
      requires Valid()
      requires verifiedDB.GetPendingTransition() == Some(pending)
      requires PendingTransitionIsConsistent()
      ensures Valid()
      ensures AllDBsInSync()
      ensures PendingTransitionIsConsistent()
      ensures madeProgress.None? ==>
        verifiedDB.pendingTransition == Some(pending)
      ensures madeProgress.Some? ==>
        (madeProgress.value <==> pending.decision == Advance)
    {
      assert ValidPendingTransition(pending);

      if pending.decision == Rewind {

        currentL1 := BlockID(0, 0);
        var rewindPlan := pending.rewind.value;

        assert rewindPlan.resetAllChainsTo.Some? ==>
          AllDBsInSyncUpTo(rewindPlan.resetAllChainsTo.value);

        var rewindOk := ApplyRewindPlan(rewindPlan);

        if !rewindOk {
          madeProgress := None;
          assert PendingTransitionIsConsistent();
          return;
        }

        // ApplyRewindPlan establishes DBsInSync(k) for all k. Snapshot the heap here
        // so that the twostate lemma call below can use old@BeforeClearRewind() to refer
        // to this state, where DBsInSync is known to hold.
        label BeforeClearRewind:
        verifiedDB.ClearPendingTransition();

        assert AllDBsInSync() by {
          // ClearPendingTransition only changes pendingTransition; db and lastTimestamp are
          // unchanged (see postconditions). Use the framing lemma to re-derive DBsInSync.
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            ClearPendingPreservesDBsInSync@BeforeClearRewind(k);
          }
        }

        madeProgress := Some(false);

      } else if pending.decision == Invalidate {

        // Replaces unreachable nil case in Go code
        assert pending.result.Some?;

        // Corresponds to the invalidateBlock loop in applyPendingTransition in interop.go.
        // The sort over chain IDs is dropped (see modeling note 9).
        var failedAny := false;
        var invSeq := Enumerate(pending.result.value.invalidHeads.Keys);

        for i := 0 to |invSeq|
          invariant Valid()
          invariant verifiedDB.pendingTransition == Some(pending)
          invariant AllDBsInSync()
        {
          var chainID := invSeq[i];
          if chainID in chains {
            var ok := chains[chainID].InvalidateBlock(
              pending.result.value.invalidHeads[chainID],
              pending.result.value.timestamp);

            if !ok { failedAny := true; }
          }
        }

        if failedAny {
          madeProgress := None;
          return;
        }

        // Loop invariant established DBsInSync(k) for all k. Snapshot the heap here
        // so the twostate lemma below can use old@BeforeClearInvalidate() to refer
        // to this state.
        label BeforeClearInvalidate:
        verifiedDB.ClearPendingTransition();

        assert AllDBsInSync() by {
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            ClearPendingPreservesDBsInSync@BeforeClearInvalidate(k);
          }
        }

        madeProgress := Some(false);

      } else { // Advance case

        // Replaces unreachable nil case in Go code
        assert pending.result.Some?;

        var result := pending.result.value;
        PersistFrontierLogs(result.timestamp, result.l2Heads);

        // Snapshot the heap before Commit so the twostate lemma can reference
        // the post-PersistFrontierLogs state via old@BeforeCommit().
        label BeforeCommit:
        // Inline commitVerifiedResult (a one-line wrapper) and ToVerifiedResult
        // (drops invalidHeads from Result to produce VerifiedResult).
        verifiedDB.Commit(VerifiedResult(result.timestamp, result.l1Inclusion, result.l2Heads));

        assert AllDBsInSync() by {
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            CommitEstablishesDBsInSync@BeforeCommit(k, result.timestamp, result.l2Heads);
          }
        }

        label BeforeClearAdvance:
        verifiedDB.ClearPendingTransition();

        assert AllDBsInSync() by {
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            ClearPendingPreservesDBsInSync@BeforeClearAdvance(k);
          }
        }

        currentL1 := result.l1Inclusion;
        madeProgress := Some(true);
      }
    }

    // Rewinds the verifiedDB and applies chain-level engine/log resets.
    // Corresponds to applyRewindPlan in interop.go. The sort over chain IDs is
    // dropped.
    method {:isolate_assertions} ApplyRewindPlan(plan: RewindPlan) returns (success: bool)
      modifies verifiedDB, chains.Values, logsDBs.Values
      requires Valid()
      requires verifiedDB.pendingTransition.Some?
      requires verifiedDB.pendingTransition.value.decision == Rewind
      requires verifiedDB.pendingTransition.value.rewind == Some(plan)
      requires ValidRewindPlan(plan)
      requires PlanConsistentWithVerified(plan)
      requires plan.resetAllChainsTo.Some? ==>
        forall k :: k in logsDBs.Keys ==> PlanConsistentWithLogs(plan, k)
      requires plan.resetAllChainsTo.Some? ==>
        AllDBsInSyncUpTo(plan.resetAllChainsTo.value)
      ensures Valid()
      ensures ValidRewindPlan(plan)
      ensures verifiedDB.pendingTransition == old(verifiedDB.pendingTransition)
      ensures PlanConsistentWithVerified(plan)
      ensures plan.resetAllChainsTo.Some? ==>
        forall k :: k in logsDBs.Keys ==>
          PlanConsistentWithLogs(plan, k) &&
          RewoundLogsDB(plan, k)
      ensures AllDBsInSync()
    {
      var _ := verifiedDB.Rewind(plan.rewindAtOrAfter);

      assert unchanged(logsDBs.Values);
      assert Valid();
      assert PlanConsistentWithVerified(plan);
      assert RewoundVerifiedDB(plan);

      // Prune deny lists and optionally rewind chain engines.
      // These operations modify only chain containers, leaving verifiedDB and
      // logsDBs unchanged.
      var chainIDs := Enumerate(chains.Keys);

      var enginesOk := RewindChainEngines(plan, chainIDs);

      assert unchanged(logsDBs.Values);

      // Clear or rewind log databases depending on whether target heads are available.
      // resetAllChainsTo.None? corresponds to the nil TargetHeads case in Go,
      // which signals a full reset with no previous verified state to restore to.
      if plan.resetAllChainsTo.None? {
        ClearLogsDBs(plan, chainIDs);
      } else {
        // Establish DBsInSyncUpTo here, right before RewindLogsDBs needs it.
        // RewindChainEngines touched only chains.Values, so the twostate lemma
        // preconditions are identical to the post-Rewind call site: verifiedDB.db
        // is the same, logsDBs[k] is unchanged, and old() still resolves to
        // method entry where the precondition DBsInSyncUpTo(k, ts) held.
        var ts := plan.resetAllChainsTo.value;

        assert AllDBsInSyncUpTo(ts) by {
          forall k | k in chainIDs
            ensures DBsInSyncUpTo(k, ts)
          {
            VerifiedDBRewindPreservesDBsInSyncUpTo(k, ts, plan.rewindAtOrAfter);
          }
        }

        RewindLogsDBs(plan, chainIDs);
      }
      success := enginesOk;
    }

    // Prunes deny lists and optionally rewinds chain engines for all chains in chainIDs.
    // Extracted from ApplyRewindPlan to allow isolated verification of the engine rewind loop.
    method RewindChainEngines(plan: RewindPlan, chainIDs: seq<ChainID>) returns (success: bool)
      modifies chains.Values
      requires Valid()
      requires ValidRewindPlan(plan)
      requires PlanConsistentWithVerified(plan)
      requires RewoundVerifiedDB(plan)
      requires forall k :: k in chainIDs <==> k in chains.Keys
      requires forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)
      ensures Valid()
      ensures ValidRewindPlan(plan)
      ensures RewoundVerifiedDB(plan)
      ensures forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)
      ensures unchanged(logsDBs.Values)
    {
      var failedAny := false;
      for i := 0 to |chainIDs|
        invariant Valid()
        invariant ValidRewindPlan(plan)
        invariant RewoundVerifiedDB(plan)
      {
        chains[chainIDs[i]].PruneDeniedAtOrAfterTimestamp(plan.rewindAtOrAfter);
        if plan.resetAllChainsTo.Some? {
          var ok := chains[chainIDs[i]].RewindEngine(plan.resetAllChainsTo.value);
          if !ok { failedAny := true; }
        }
      }
      success := !failedAny;
    }

    // Clears all logsDBs in chainIDs.
    // Extracted from ApplyRewindPlan to allow isolated verification of the clear loop.
    method ClearLogsDBs(plan: RewindPlan, chainIDs: seq<ChainID>)
      modifies logsDBs.Values
      requires Valid()
      requires ValidRewindPlan(plan)
      requires PlanConsistentWithVerified(plan)
      requires RewoundVerifiedDB(plan)
      requires plan.resetAllChainsTo.None?
      requires forall k :: k in chainIDs <==> k in chains.Keys
      ensures Valid()
      ensures ValidRewindPlan(plan)
      ensures forall k :: k in chainIDs ==> RewoundLogsDB(plan, k)
    {
      for i := 0 to |chainIDs|
        invariant Valid()
        invariant |verifiedDB.db| == 0
        invariant forall k :: k in chainIDs[0..i] ==>
          logsDBs[k].LatestSealedBlock() == None
        invariant forall k :: k in chainIDs[0..i] ==>
          RewoundLogsDB(plan, k)
      {
        logsDBs[chainIDs[i]].Clear();
      }
    }

    // Rewinds each logsDB in chainIDs to the corresponding target head in plan.
    // Extracted from ApplyRewindPlan to allow isolated verification of the rewind loop.
    method RewindLogsDBs(plan: RewindPlan, chainIDs: seq<ChainID>)
      modifies logsDBs.Values
      requires verifiedDB.Valid()
      requires forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]
      requires forall k :: k in chainIDs <==> k in plan.targetHeads
      requires PlanConsistentWithVerified(plan)
      requires RewoundVerifiedDB(plan)
      requires plan.resetAllChainsTo.Some?
      requires forall k :: k in chainIDs <==> k in logsDBs.Keys
      requires forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)
      requires forall k :: k in chainIDs ==> DBsInSyncUpTo(k, plan.resetAllChainsTo.value)
      ensures forall k :: k in chainIDs ==>
        PlanConsistentWithLogs(plan, k) &&
        RewoundLogsDB(plan, k) &&
        DBsInSync(k)
    {
      var ts := plan.resetAllChainsTo.value;
      for i := 0 to |chainIDs|
        invariant verifiedDB.LastTimestamp() == Some(ts)
        invariant plan.targetHeads == verifiedDB.Get(ts).l2Heads
        invariant forall k :: k in chainIDs ==>
          PlanConsistentWithLogs(plan, k) &&
          DBsInSyncUpTo(k, plan.resetAllChainsTo.value)
        invariant forall k :: k in chainIDs[0..i] ==>
          PlanConsistentWithLogs(plan, k) &&
          RewoundLogsDB(plan, k) &&
          DBsInSync(k)
      {
        var chainID := chainIDs[i];
        logsDBs[chainID].Rewind(plan.targetHeads[chainID]);
      }
    }

    // Persists frontier logs for the given verified result.
    // Corresponds to persistFrontierLogs in interop.go.
    method {:isolate_assertions} PersistFrontierLogs(ts: nat, blocksAtTS: map<ChainID, BlockID>)
      modifies logsDBs.Values
      requires Valid()
      requires blocksAtTS.Keys == chains.Keys
      requires AdvancesAllLogsDBs(ts, blocksAtTS)
      requires verifiedDB.LastTimestamp().Some? ==>
        AllDBsInSyncUpTo(verifiedDB.LastTimestamp().value)
      ensures Valid()
      ensures forall k :: k in blocksAtTS.Keys ==>
        logsDBs[k].LatestSealedBlock() == Some(blocksAtTS[k])
      ensures verifiedDB.LastTimestamp().Some? ==>
        AllDBsInSyncUpTo(verifiedDB.LastTimestamp().value)
    {
      var chainIDs := Enumerate(blocksAtTS.Keys);

      for i := 0 to |chainIDs|
        invariant Valid()
        invariant forall j, k :: 0 <= j < k < |chainIDs| ==>
          chainIDs[j] != chainIDs[k]
        invariant forall j :: i <= j < |chainIDs| ==>
          AdvancesLogsDB(ts, chainIDs[j], blocksAtTS[chainIDs[j]])
        invariant forall j :: 0 <= j < i ==>
          logsDBs[chainIDs[j]].LatestSealedBlock() == Some(blocksAtTS[chainIDs[j]])
        invariant verifiedDB.LastTimestamp().Some? ==>
          AllDBsInSyncUpTo(verifiedDB.LastTimestamp().value)
      {
        var chainID := chainIDs[i];
        var blockID := blocksAtTS[chainID];
        var db := logsDBs[chainID];
        var chain := chains[chainID];

        var latestBlock := db.LatestSealedBlock();

        // Follows from AdvancesLogsDB(ts, chainID, blockID)
        if latestBlock.Some? {
          assert latestBlock.value.number <= blockID.number;
          assert latestBlock.value.number == blockID.number ==> latestBlock == Some(blockID);
        }

        // Skip if the block is already sealed in the logsDB (idempotency on restart).
        // Simplified in relation to the Go code due to the asserts above.
        var skip := latestBlock == Some(blockID);

        if !skip {
          var blockInfo := chain.FetchReceipts(blockID);
          // Preconditions replacing ErrStaleLogsDB and ErrParentHashMismatch:
          // in Go these would be error returns; here they are assumed away.
          assume latestBlock.Some? ==> blockInfo.parentHash == latestBlock.value.hash;

          if verifiedDB.LastTimestamp().Some? {
            var lastTS := verifiedDB.LastTimestamp().value;

            assert SealedBlockForVerifiedAtTimestamp(chainID, lastTS).Some? ==>
              db.LatestSealedBlock().Some?;

            assert AllDBsInSyncUpTo(lastTS) by {
              forall k | k in blocksAtTS.Keys && k != chainID
                ensures DBsInSyncUpTo(k, lastTS)
              {
                assert logsDBs[k] != db;
              }
            }
          }

          label BeforeProcessBlock:
          ProcessBlock(db, blockInfo);

          assert logsDBs[chainID].LatestSealedBlock() == Some(blocksAtTS[chainID]);

          if verifiedDB.LastTimestamp().Some? {
            var lastTS := verifiedDB.LastTimestamp().value;

            assert AllDBsInSyncUpTo(lastTS) by {
              ProcessBlockPreservesDBsInSyncUpTo@BeforeProcessBlock(chainID, lastTS, blockInfo.id.number);
            }
          }
        }
      }
    }

    // Constructs a pending transition from the given decision and observation.
    // Does not modify any state.
    // Corresponds to buildPendingTransition in interop.go; buildRewindPlan is
    // inlined here.
    method {:isolate_assertions} BuildPendingTransition(output: StepOutput, obs: RoundObservation)
        returns (pendingTx: PendingTransition)
      requires Valid()
      requires !output.WaitOutput?
      requires ValidStepOutput(output, obs)
      requires OutputConsistentWithVerified(output, obs)
      requires OutputConsistentWithLogs(output, obs)
      ensures ValidPendingTransition(pendingTx)
      ensures TransitionConsistentWithVerified(pendingTx)
      ensures TransitionConsistentWithLogs(pendingTx)
    {
      if output.AdvanceOutput? {
        pendingTx := PendingTransition(Advance, Some(output.result), None);
      } else if output.InvalidateOutput? {
        pendingTx := PendingTransition(Invalidate, Some(output.result), None);
      } else {
        // RewindOutput: build a rewind plan (inlined from buildRewindPlan).
        var lastTS := obs.lastVerifiedTS.value;
        var rewindPlan: RewindPlan;

        if lastTS <= activationTimestamp {
          // At or before the activation timestamp there is no previous verified
          // entry to restore, so only rewindAtOrAfter is set.
          rewindPlan := RewindPlan(lastTS, None, map[]);
        } else {
          var prevResult := verifiedDB.Get(lastTS - 1);
          rewindPlan := RewindPlan(lastTS, Some(lastTS - 1), prevResult.l2Heads);
        }

        pendingTx := PendingTransition(Rewind, None, Some(rewindPlan));
        assert TransitionConsistentWithLogs(pendingTx);
      }
    }

    // Checks L1 consistency from two perspectives.
    // l1Consistent: the last verified L1 inclusion block is on the same chain
    //   as the current frontier L1 heads (requires lastVerified to be present).
    // l2sConsistent: the frontier L1 heads from all L2 chains agree with each other.
    // Corresponds to the SameL1Chain check inside observeRound in interop.go.
    method CheckL1Consistent(
        l1Heads: map<ChainID, BlockID>,
        lastVerified: Option<VerifiedResult>)
        returns (l1Consistent: bool, l2sConsistent: bool)
      requires Valid()
      ensures {:axiom} Valid()
      ensures {:axiom} !l1Consistent ==> lastVerified.Some?

    // Verifies cross-chain interop messages at the given timestamp.
    // Abstracts verifyFn (i.verifyInteropMessages) from interop.go.
    method VerifyMessages(ts: nat, blocksAtTS: map<ChainID, BlockID>) returns (result: Result)
      requires Valid()
      ensures {:axiom} Valid()
      ensures {:axiom} result.timestamp == ts
      ensures {:axiom} result.l2Heads == blocksAtTS

    // Verifies same-timestamp cycle constraints at the given timestamp.
    // Abstracts cycleVerifyFn (i.verifyCycleMessages) from interop.go.
    method VerifyCycles(ts: nat, blocksAtTS: map<ChainID, BlockID>) returns (result: Result)
      requires Valid()
      ensures {:axiom} Valid()

    // Processes and seals a block's logs in the given chain's log database.
    // Abstracts processBlockLogs in logdb.go; the stale-data and parent-hash
    // error returns from persistFrontierLogs are replaced with preconditions.
    method ProcessBlock(db: LogsDB, info: BlockInfo)
      modifies db
      requires Valid()
      requires db.LatestSealedBlock().None? || (
        db.LatestSealedBlock().value.number + 1 == info.id.number &&
        info.parentHash == db.LatestSealedBlock().value.hash)
      ensures {:axiom} unchanged(verifiedDB)
      ensures {:axiom} Valid()
      ensures {:axiom} db.LatestSealedBlock() == Some(info.id)
      ensures {:axiom} forall n :: n != info.id.number ==> db.FindSealedBlock(n) == old(db.FindSealedBlock(n))

    // Updates currentL1 when no progress is made.
    // Corresponds to refreshCurrentL1OnWait in interop.go.
    method RefreshCurrentL1OnWait()
      modifies this
      requires Valid()
      ensures {:axiom} Valid()

    // ========================================================================
    // Lemmas
    // ========================================================================

    // AllDBsInSync() implies AllDBsInSyncUpTo(upper) for any upper <= LastTimestamp().
    // DBsInSync(k) (Some branch) includes DBsInSyncUpTo(k, ts), and since upper <= ts the
    // quantifier bound [activationTimestamp, upper] is a subset of [activationTimestamp, ts].
    lemma AllDBsInSyncImpliesAllDBsInSyncUpTo(upper: nat)
      requires verifiedDB.Valid()
      requires AllDBsInSync()
      requires verifiedDB.LastTimestamp().Some?
      requires upper <= verifiedDB.LastTimestamp().value
      ensures AllDBsInSyncUpTo(upper)
    {
      var ts := verifiedDB.LastTimestamp().value;
      forall k | k in logsDBs.Keys ensures DBsInSyncUpTo(k, upper) {
        assert DBsInSync(k);
        assert DBsInSyncUpTo(k, ts);
      }
    }

    // Framing lemma: verifiedDB.Rewind preserves DBsInSyncUpTo for any upper < rewindAtOrAfter.
    // The LogsDB for chainID is unchanged (not in Rewind's modifies set), so all
    // FindSealedBlock results are identical; the verifiedDB entries at t <= upper survive
    // the rewind unchanged; therefore the predicate body holds in the new state too.
    twostate lemma VerifiedDBRewindPreservesDBsInSyncUpTo(
        chainID: ChainID, upper: nat, rewindAtOrAfter: nat)
      requires chainID in logsDBs.Keys
      requires upper < rewindAtOrAfter
      requires old(DBsInSyncUpTo(chainID, upper))
      requires verifiedDB.db ==
        map k | k in old(verifiedDB.db) && k < rewindAtOrAfter :: old(verifiedDB.db)[k]
      requires unchanged(logsDBs[chainID])
      ensures DBsInSyncUpTo(chainID, upper)
    {
      forall t | activationTimestamp <= t <= upper
        ensures verifiedDB.Has(t)
        ensures chainID in verifiedDB.Get(t).l2Heads
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).Some?
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).value.id ==
          verifiedDB.Get(t).l2Heads[chainID]
      {
        assert old(verifiedDB.Has(t));                      // fires trigger on old(DBsInSyncUpTo) at t
        assert verifiedDB.db[t] == old(verifiedDB.db)[t];  // t survives the rewind (t < rewindAtOrAfter)
      }
    }

    // Framing lemma: when verifiedDB.db is unchanged and logsDBs[chainID] is unchanged,
    // DBsInSyncUpTo is preserved across any state change (e.g. ClearPendingTransition).
    twostate lemma ClearPendingPreservesDBsInSyncUpTo(chainID: ChainID, upper: nat)
      requires chainID in logsDBs.Keys
      requires old(DBsInSyncUpTo(chainID, upper))
      requires verifiedDB.db == old(verifiedDB.db)
      requires unchanged(logsDBs[chainID])
      ensures DBsInSyncUpTo(chainID, upper)
    {
      forall t | activationTimestamp <= t <= upper
        ensures verifiedDB.Has(t)
        ensures chainID in verifiedDB.Get(t).l2Heads
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).Some?
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).value.id ==
          verifiedDB.Get(t).l2Heads[chainID]
      {
        assert old(verifiedDB.Has(t));
        assert verifiedDB.db[t] == old(verifiedDB.db)[t];
      }
    }

    // Framing lemma: ClearPendingTransition preserves DBsInSync.
    // When only verifiedDB.pendingTransition changes (db and lastTimestamp unchanged)
    // and logsDBs[chainID] is unchanged, DBsInSync is preserved.
    // Intended to be called as ClearPendingPreservesDBsInSync@L(k) where L is a label
    // placed just before the ClearPendingTransition call so that old() resolves to the
    // pre-clear state where DBsInSync(chainID) is already established.
    twostate lemma ClearPendingPreservesDBsInSync(chainID: ChainID)
      requires chainID in logsDBs.Keys
      requires old(verifiedDB.Valid())
      requires verifiedDB.Valid()
      requires verifiedDB.db == old(verifiedDB.db)
      requires verifiedDB.lastTimestamp == old(verifiedDB.lastTimestamp)
      requires unchanged(logsDBs[chainID])
      requires old(DBsInSync(chainID))
      ensures DBsInSync(chainID)
    {
      match old(verifiedDB.LastTimestamp()) {
        case None => {}
        case Some(ts) => {
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
          ClearPendingPreservesDBsInSyncUpTo(chainID, ts);
        }
      }
    }

    // Framing lemma: Commit establishes DBsInSync at the new timestamp.
    // Intended to be called as CommitEstablishesDBsInSync@L(k, ts, l2Heads) where L is
    // placed just before the Commit call (i.e. just after PersistFrontierLogs), so that
    // old() resolves to the post-PersistFrontierLogs state where:
    //   - logsDBs[k].LatestSealedBlock() == Some(l2Heads[k])     (PersistFrontierLogs post)
    //   - DBsInSyncUpTo(k, old LastTimestamp) holds (if non-empty) (PersistFrontierLogs axiom)
    // Proof splits on t == newTs (new entry, proved from LatestSealedBlock axiom) vs
    // t < newTs (old entries, preserved via db[t] == old(db)[t] + unchanged logsDBs).
    twostate lemma CommitEstablishesDBsInSync(
        chainID: ChainID, newTs: nat, newL2Heads: map<ChainID, BlockID>)
      requires chainID in logsDBs.Keys
      requires chainID in newL2Heads
      requires old(logsDBs[chainID].LatestSealedBlock()) == Some(newL2Heads[chainID])
      requires old(verifiedDB.LastTimestamp()).None? ==> newTs == activationTimestamp
      requires old(verifiedDB.LastTimestamp()).Some? ==>
        newTs == old(verifiedDB.LastTimestamp()).value + 1
      // newTs - 1 == old(verifiedDB.LastTimestamp()).value (from the require above),
      // so this encodes old(DBsInSyncUpTo(chainID, old(LastTimestamp()).value)) without nesting old().
      requires old(verifiedDB.LastTimestamp()).Some? ==>
        old(DBsInSyncUpTo(chainID, newTs - 1))
      requires verifiedDB.LastTimestamp() == Some(newTs)
      requires verifiedDB.Has(newTs)
      requires verifiedDB.db == old(verifiedDB.db)[newTs := verifiedDB.db[newTs]]
      requires verifiedDB.Get(newTs).l2Heads == newL2Heads
      requires unchanged(logsDBs[chainID])
      requires verifiedDB.Valid()
      ensures DBsInSync(chainID)
    {
      forall t | activationTimestamp <= t <= newTs
        ensures verifiedDB.Has(t)
        ensures chainID in verifiedDB.Get(t).l2Heads
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).Some?
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).value.id ==
          verifiedDB.Get(t).l2Heads[chainID]
      {
        if t == newTs {
          // verifiedDB.Has(newTs): from newTs in old(db)[newTs := ...] = verifiedDB.db
          // chainID in newL2Heads: from requires
          // LatestSealedBlock() == Some(newL2Heads[chainID]): old() + unchanged(logsDBs)
          // => FindSealedBlock via LatestSealedBlock axiom
        } else {
          // t < newTs. If old(LastTimestamp()).None?, then newTs == activationTimestamp,
          // so activationTimestamp <= t < activationTimestamp is impossible.
          // Otherwise old(LastTimestamp()) = Some(oldTs), newTs == oldTs + 1, t <= oldTs.
          assert old(verifiedDB.Has(t));           // fires trigger on old(DBsInSyncUpTo)
          assert verifiedDB.db[t] == old(verifiedDB.db)[t];  // t != newTs
        }
      }
    }

    // Framing lemma: ProcessBlock on logsDBs[chainID] (adding block at newBlockNumber,
    // which must equal oldLatestBlock.number + 1) preserves DBsInSyncUpTo(chainID, upper).
    // All blocks referenced by DBsInSyncUpTo have numbers <= oldLatestBlock.number, so
    // the FindSealedBlock results they query are unchanged by ProcessBlock.
    twostate lemma ProcessBlockPreservesDBsInSyncUpTo(
        chainID: ChainID, upper: nat, newBlockNumber: nat)
      requires chainID in logsDBs.Keys
      requires old(verifiedDB.Valid())
      requires verifiedDB.db == old(verifiedDB.db)
      requires old(DBsInSyncUpTo(chainID, upper))
      requires forall n :: n != newBlockNumber ==>
        logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n))
      requires old(logsDBs[chainID].LatestSealedBlock()).Some?
      requires newBlockNumber == old(logsDBs[chainID].LatestSealedBlock()).value.number + 1
      ensures DBsInSyncUpTo(chainID, upper)
    {
      var latestNum := old(logsDBs[chainID].LatestSealedBlock()).value.number;

      forall t | activationTimestamp <= t <= upper
        ensures verifiedDB.Has(t)
        ensures chainID in verifiedDB.Get(t).l2Heads
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).Some?
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).value.id ==
          verifiedDB.Get(t).l2Heads[chainID]
      {
        assert old(verifiedDB.Has(t));             // fires trigger on old(DBsInSyncUpTo)
        assert verifiedDB.db[t] == old(verifiedDB.db)[t];
        var n := verifiedDB.Get(t).l2Heads[chainID].number;
        // old(FindSealedBlock(n)).Some? from DBsInSyncUpTo, so n <= latestNum by
        // the LatestSealedBlock axiom contrapositive
        assert old(logsDBs[chainID].FindSealedBlock(n)).Some?;
        assert n <= latestNum;
        assert n != newBlockNumber;
        assert logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n));
      }
    }

  }
}
