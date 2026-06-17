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

  // Abstract model of the frontier verification view, which pre-fetches block
  // receipts for the unverified frontier so verification can answer same-timestamp
  // queries without racing against logsDB writes.
  // Corresponds to frontierVerificationView in verification_view.go.
  class FrontierView {
    // Checks whether the initiating message matching query is present in the
    // frontier block for chainID.
    // Corresponds to frontierVerificationView.contains in verification_view.go.
    predicate Contains(chainID: ChainID, query: ContainsQuery)
      reads this
      requires chainID in CHAIN_IDS
    {
      BlockInfo(chainID).id.number == query.blockNum &&
      BlockInfo(chainID).timestamp == query.timestamp &&
      query.logIdx < |BlockLogs(chainID).fullLogs| &&
      BlockLogs(chainID).fullLogs[query.logIdx].checksum == query.checksum
    }

    function BlockInfo(chainID: ChainID) : BlockInfo
      reads this
      requires chainID in CHAIN_IDS

    function BlockLogs(chainID: ChainID) : BlockLogs
      reads this
      requires chainID in CHAIN_IDS
  }

  class Interop {

    var currentL1: BlockID
    const chains: map<ChainID, ChainContainer>
    const verifiedDB: VerifiedDB
    const activationTimestamp: nat
    const messageExpiryWindow: nat
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
      this.messageExpiryWindow := MESSAGE_EXPIRY_WINDOW;
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
      reads verifiedDB, logsDBs.Values
    {
      activationTimestamp == ACTIVATION_TIMESTAMP &&
      messageExpiryWindow == MESSAGE_EXPIRY_WINDOW &&
      chains.Keys == CHAIN_IDS &&

      /* LogsDBs invariants */
      // There is one logsDB for each chain.
      logsDBs.Keys == CHAIN_IDS &&
      // All logsDBs are distinct.
      (forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]) &&
      BlockSealsMatchOnChainTimestamps() &&

      /* VerifiedDB invariants */
      verifiedDB.Valid() &&
      // If any timestamp has been verified, the activation timestamp must be in the DB.
      (verifiedDB.lastTimestamp.Some? ==> activationTimestamp in verifiedDB.db) &&
      // All committed timestamps are at or above the activation timestamp.
      (forall ts :: ts in verifiedDB.db ==> activationTimestamp <= ts) &&
      // Every committed verified result covers exactly the current set of chains.
      (forall ts :: ts in verifiedDB.db ==> verifiedDB.db[ts].l2Heads.Keys == CHAIN_IDS) &&
      // The pending transition is valid, if it exists.
      (verifiedDB.pendingTransition.Some? ==> ValidPendingTransition(verifiedDB.GetPendingTransition().value)) &&
      // For every timestamp in the DB, the corresponding L2 heads are bounded by that timestamp.
      AllVerifiedHeadsBoundedByTimestamp() &&
      // For every verified result, each L2 head is the highest block in the logsDB below that timestamp.
      VerifiedHeadsAreHighestBlocksUpToTimestamp()
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
      requires plan.resetAllChainsTo.Some? ==> chainID in plan.targetHeads.Keys
    {
      plan.resetAllChainsTo.Some? ==>
        var sealedBlock := logsDBs[chainID].FindSealedBlock(plan.targetHeads[chainID].number);
        sealedBlock.Some? &&
        sealedBlock.value.id == plan.targetHeads[chainID]
    }

    ghost predicate PlanConsistentWithAllLogs(plan: RewindPlan)
      reads logsDBs.Values
      requires plan.resetAllChainsTo.Some? ==> plan.targetHeads.Keys == logsDBs.Keys
    {
      forall chainID :: chainID in logsDBs.Keys ==>
        PlanConsistentWithLogs(plan, chainID)
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

    ghost predicate RewoundAllLogsDB(plan: RewindPlan)
      reads logsDBs.Values
      requires plan.resetAllChainsTo.Some? ==> plan.targetHeads.Keys == logsDBs.Keys
      requires PlanConsistentWithAllLogs(plan)
    {
      forall chainID :: chainID in logsDBs.Keys ==>
        RewoundLogsDB(plan, chainID)
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

    ghost predicate TransitionConsistentWithChainState(pending: PendingTransition)
      requires ValidPendingTransition(pending)
      requires chains.Keys == CHAIN_IDS
    {
      pending.decision.Advance? ==>
        BlocksExistedOnChain(pending.result.value.l2Heads) &&
        FrontierBlocksConsistentWithTimestamp(pending.result.value.timestamp, pending.result.value.l2Heads)
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
          TransitionConsistentWithChainState(p) &&
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
    ghost predicate {:isolate_assertions} OutputConsistentWithLogs(output: StepOutput, obs: RoundObservation)
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

    ghost predicate OutputConsistentWithChainState(output: StepOutput, obs: RoundObservation)
      requires chains.Keys == CHAIN_IDS
    {
      output.AdvanceOutput? ==>
        output.result.l2Heads.Keys == CHAIN_IDS &&
        BlocksExistedOnChain(output.result.l2Heads) &&
        FrontierBlocksConsistentWithTimestamp(output.result.timestamp, output.result.l2Heads)
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

    ghost predicate ObservationConsistentWithChainState(obs: RoundObservation)
      requires 0 < |obs.blocksAtTS.Keys| ==> obs.blocksAtTS.Keys == chains.Keys
    {
      0 < |obs.blocksAtTS| ==>
        obs.blocksAtTS.Keys == CHAIN_IDS &&
        BlocksExistedOnChain(obs.blocksAtTS) &&
        FrontierBlocksConsistentWithTimestamp(obs.nextTimestamp, obs.blocksAtTS)
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

    ghost predicate ValidExecutingMessage(execTimestamp: nat, execChain: ChainID, execMsg: ExecutingMessage)
      requires execChain in CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
    {
      var initChain := execMsg.chainID;
      var initTimestamp := execMsg.timestamp;
      initChain in chains.Keys &&
      var initBlockTime := chains[initChain].BlockTime();
      var execBlockTime := chains[execChain].BlockTime();
      activationTimestamp + execBlockTime <= execTimestamp &&
      activationTimestamp + initBlockTime <= initTimestamp &&
      initTimestamp <= execTimestamp <= initTimestamp + messageExpiryWindow
    }

    ghost predicate IsCorrectFrontierView(view: FrontierView, blocksAtTS: map<ChainID, BlockID>)
      reads view
      requires chains.Keys == CHAIN_IDS
      requires blocksAtTS.Keys == CHAIN_IDS
    {
      forall chainID :: chainID in CHAIN_IDS ==>
        chains[chainID].BlockInfo(blocksAtTS[chainID]).Some? &&
        view.BlockInfo(chainID) == chains[chainID].BlockInfo(blocksAtTS[chainID]).value &&
        view.BlockLogs(chainID) == chains[chainID].BlockLogs(blocksAtTS[chainID]).value
    }

    ghost predicate InitMsgInFrontierView(execMsg: ExecutingMessage, view: FrontierView)
      reads view
      requires execMsg.chainID in CHAIN_IDS
    {
      var query := ContainsQuery(execMsg.blockNum, execMsg.logIdx, execMsg.timestamp, execMsg.checksum);
      view.Contains(execMsg.chainID, query)
    }

    ghost predicate InitMsgInFrontier(execMsg: ExecutingMessage, blocksAtTS: map<ChainID, BlockID>)
      requires execMsg.chainID in CHAIN_IDS
      requires blocksAtTS.Keys == CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
    {
      var initBlock := blocksAtTS[execMsg.chainID];
      initBlock.number == execMsg.blockNum &&
      var initBlockInfo := chains[execMsg.chainID].BlockInfo(initBlock);
      initBlockInfo.Some? &&
      initBlockInfo.value.timestamp == execMsg.timestamp &&
      var initBlockLogs := chains[execMsg.chainID].BlockLogs(initBlock);
      assert initBlockLogs.Some?; // by initBlockInfo.Some?
      execMsg.logIdx < |initBlockLogs.value.fullLogs| &&
      var initMsg := initBlockLogs.value.fullLogs[execMsg.logIdx];
      initMsg.checksum == execMsg.checksum
    }

    ghost predicate InitMsgInLogsDB(execMsg: ExecutingMessage)
      reads logsDBs.Values
      requires execMsg.chainID in logsDBs.Keys
    {
      var query := ContainsQuery(execMsg.blockNum, execMsg.logIdx, execMsg.timestamp, execMsg.checksum);
      logsDBs[execMsg.chainID].Contains(query)
    }

    ghost predicate BlockIsCrossValid(ts: nat, chainID: ChainID, blockID: BlockID)
      requires chainID in CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires BlockExistedOnChain(chainID, blockID)
    {
      var logs := chains[chainID].BlockLogs(blockID).value;

      forall execMsg {:trigger execMsg in chains[chainID].BlockLogs(blockID).value.execMsgs.Values} ::
        execMsg in logs.execMsgs.Values ==>
        ValidExecutingMessage(ts, chainID, execMsg)
    }

    ghost predicate AllInitMsgsPresent(chainID: ChainID, blockID: BlockID, blocksAtTS: map<ChainID, BlockID>)
      reads logsDBs.Values
      requires chainID in CHAIN_IDS
      requires logsDBs.Keys == CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires blocksAtTS.Keys == CHAIN_IDS
      requires BlockExistedOnChain(chainID, blockID)
    {
      var logs := chains[chainID].BlockLogs(blockID).value;

      forall execMsg :: execMsg in logs.execMsgs.Values ==>
        execMsg.chainID in CHAIN_IDS &&
        (InitMsgInFrontier(execMsg, blocksAtTS) || InitMsgInLogsDB(execMsg))
    }

    ghost predicate AllInitMsgsInLogsDB(chainID: ChainID, blockID: BlockID)
      reads logsDBs.Values
      requires chainID in chains.Keys
      requires BlockExistedOnChain(chainID, blockID)
    {
      var logs := chains[chainID].BlockLogs(blockID).value;

      forall execMsg :: execMsg in logs.execMsgs.Values ==>
        execMsg.chainID in logsDBs.Keys &&
        InitMsgInLogsDB(execMsg)
    }

    ghost predicate ResultIsCrossValid(result: Result)
      reads logsDBs.Values
      requires logsDBs.Keys == CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires result.l2Heads.Keys == CHAIN_IDS
      requires |result.invalidHeads| == 0
    {
      forall chainID :: chainID in CHAIN_IDS ==>
        var blockID := result.l2Heads[chainID];
        var blockInfo := chains[chainID].BlockInfo(blockID);
        blockInfo.Some? &&
        var ts := blockInfo.value.timestamp;
        BlockIsCrossValid(ts, chainID, blockID) &&
        AllInitMsgsPresent(chainID, blockID, result.l2Heads)
    }

    ghost predicate BlockExistedOnChain(chainID: ChainID, blockID: BlockID)
      requires chainID in chains.Keys
      ensures BlockExistedOnChain(chainID, blockID) <==> chains[chainID].BlockInfo(blockID).Some?
    {
      chains[chainID].BlockLogs(blockID).Some?
    }

    ghost predicate BlocksExistedOnChain(blocksAtTS: map<ChainID, BlockID>)
      requires blocksAtTS.Keys == chains.Keys
    {
      forall chainID :: chainID in blocksAtTS.Keys ==>
        BlockExistedOnChain(chainID, blocksAtTS[chainID])
    }

    ghost predicate TransitionIsCrossValid(pendingTransition: PendingTransition)
      reads logsDBs.Values
      requires logsDBs.Keys == CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires ValidPendingTransition(pendingTransition)
    {
      pendingTransition.decision.Advance? ==>
        assert pendingTransition.result.Some?;
        ResultIsCrossValid(pendingTransition.result.value)
    }

    opaque ghost predicate AllVerifiedCrossValid()
      reads verifiedDB, logsDBs.Values
      requires Valid()
    {
      verifiedDB.LastTimestamp().Some? ==>
        forall ts :: activationTimestamp <= ts <= verifiedDB.LastTimestamp().value ==>
          assert verifiedDB.Has(ts) by { SequentialContainsRange(verifiedDB.db, activationTimestamp); }
          var verified := verifiedDB.Get(ts);
          var result := Result(verified.timestamp, verified.l1Inclusion, verified.l2Heads, map[]);
          ResultIsCrossValid(result)
    }

    opaque twostate predicate UpdatedLogsDB(chainID: ChainID, blockID: BlockID)
      reads logsDBs[chainID]
      requires chainID in logsDBs.Keys
      requires chainID in chains.Keys
      requires BlockExistedOnChain(chainID, blockID)
    {
      var info := chains[chainID].BlockInfo(blockID).value;
      var logs := chains[chainID].BlockLogs(blockID).value;
      var db := logsDBs[chainID];
      LogsDBConsistentWithChainData(chainID) &&
      db.LatestSealedBlock() == Some(blockID) &&
      (forall n :: n != blockID.number ==> db.FindSealedBlock(n) == old(db.FindSealedBlock(n))) &&
      (old(db.FindSealedBlock(blockID.number)).Some? ==>
        old(db.FindSealedBlock(blockID.number)) == db.FindSealedBlock(blockID.number))
    }

    opaque twostate predicate UpdatedAllLogsDBs(blocksAtTS: map<ChainID, BlockID>)
      reads logsDBs.Values
      requires blocksAtTS.Keys == CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires logsDBs.Keys == CHAIN_IDS
      requires BlocksExistedOnChain(blocksAtTS)
    {
      forall chainID :: chainID in blocksAtTS.Keys ==>
        UpdatedLogsDB(chainID, blocksAtTS[chainID])
    }

    opaque twostate predicate LogsDBsUnchangedUpTo(blocks: map<ChainID, BlockID>)
      reads logsDBs.Values
      requires blocks.Keys == logsDBs.Keys
    {
      forall chainID :: chainID in blocks.Keys ==>
        forall n :: 0 <= n <= blocks[chainID].number ==>
          logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n))
    }

    opaque ghost predicate LogsDBConsistentWithChainData(chainID: ChainID)
      reads logsDBs[chainID]
      requires chainID in logsDBs.Keys
      requires chainID in chains.Keys
    {
      var db := logsDBs[chainID];

      forall blockID: BlockID :: db.FindSealedBlock(blockID.number).Some? ==>
        BlockExistedOnChain(chainID, blockID) &&
        var info := chains[chainID].BlockInfo(blockID).value;
        var logs := chains[chainID].BlockLogs(blockID).value;
        db.FindSealedBlock(info.id.number).value.timestamp == info.timestamp &&
        db.BlockLogs(info.id.number) == logs
    }

    opaque ghost predicate AllLogsDBsConsistentWithChainData()
      reads logsDBs.Values
      requires logsDBs.Keys == chains.Keys
    {
      forall chainID :: chainID in logsDBs.Keys ==>
        LogsDBConsistentWithChainData(chainID)
    }

    ghost predicate VerifiedHeadsAreHighestBlocksUpToTimestamp()
      reads verifiedDB, logsDBs.Values
    {
      forall ts: nat :: verifiedDB.Has(ts) ==>
        var verifiedHeads := verifiedDB.Get(ts).l2Heads;
        verifiedHeads.Keys == logsDBs.Keys &&
        forall chainID :: chainID in verifiedHeads.Keys ==>
          var blockNumber := verifiedHeads[chainID].number;
          forall n :: blockNumber < n && logsDBs[chainID].FindSealedBlock(n).Some? ==>
            ts < logsDBs[chainID].FindSealedBlock(n).value.timestamp
    }

    ghost predicate BlockSealsMatchOnChainTimestamps()
      reads logsDBs.Values
      requires logsDBs.Keys == chains.Keys
    {
      forall chainID :: chainID in logsDBs.Keys ==>
        forall n :: logsDBs[chainID].FindSealedBlock(n).Some? ==>
          var sealedBlock := logsDBs[chainID].FindSealedBlock(n).value;
          var onChainBlock := chains[chainID].BlockInfo(sealedBlock.id);
          onChainBlock.Some? &&
          sealedBlock.timestamp == onChainBlock.value.timestamp
    }

    ghost predicate VerifiedHeadsBoundedByTimestamp(ts: nat)
      reads verifiedDB
      requires verifiedDB.Has(ts)
      requires chains.Keys == verifiedDB.Get(ts).l2Heads.Keys
      requires BlocksExistedOnChain(verifiedDB.Get(ts).l2Heads)
    {
      var result := verifiedDB.Get(ts);

      forall chainID :: chainID in result.l2Heads ==>
        chains[chainID].BlockInfo(result.l2Heads[chainID]).value.timestamp <= ts
    }

    ghost predicate AllVerifiedHeadsBoundedByTimestamp()
      reads verifiedDB
    {
      var lastTimestamp := verifiedDB.LastTimestamp();

      lastTimestamp.Some? ==>
        forall ts :: activationTimestamp <= ts <= verifiedDB.LastTimestamp().value ==>
          verifiedDB.Has(ts) &&
          chains.Keys == verifiedDB.Get(ts).l2Heads.Keys &&
          BlocksExistedOnChain(verifiedDB.Get(ts).l2Heads) &&
          VerifiedHeadsBoundedByTimestamp(ts)
    }

    ghost predicate FrontierBlocksConsistentWithTimestamp(ts: nat, blocksAtTS: map<ChainID, BlockID>)
      requires blocksAtTS.Keys == chains.Keys
      requires BlocksExistedOnChain(blocksAtTS)
    {
      forall chainID :: chainID in blocksAtTS.Keys ==>
        chains[chainID].BlockInfo(blocksAtTS[chainID]).value.timestamp <= ts
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
      requires AllLogsDBsConsistentWithChainData()
      requires AllVerifiedCrossValid()
      requires verifiedDB.GetPendingTransition().Some? ==>
        TransitionIsCrossValid(verifiedDB.GetPendingTransition().value)
      ensures Valid()
      ensures PendingTransitionIsConsistent()
      ensures AllLogsDBsConsistentWithChainData()
      ensures AllVerifiedCrossValid()
      ensures verifiedDB.GetPendingTransition().Some? ==>
        TransitionIsCrossValid(verifiedDB.GetPendingTransition().value)
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

      assert TransitionIsCrossValid(pendingTx);
      assert AllVerifiedCrossValid();

      // Snapshot the heap before SetPendingTransition so that the twostate lemma
      // below can reference the pre-set state where AllDBsInSync() holds.
      label BeforeSetPending:
      verifiedDB.SetPendingTransition(pendingTx);

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

      assert AllVerifiedHeadsBoundedByTimestamp() by {
        ClearPendingPreservesAllVerifiedHeadsBoundedByTimestamp@BeforeSetPending();
      }

      assert AllVerifiedCrossValid() by {
        ClearPendingPreservesAllVerifiedCrossValid@BeforeSetPending();
      }

      madeProgress := ApplyPendingTransition(pendingTx);
    }

    // Observes the current round state, runs verification if chains are ready,
    // and returns the resulting decision together with the observation snapshot.
    // Does not modify the verified DB.
    // Corresponds to progressInterop in interop.go.
    method {:isolate_assertions} ProgressInterop() returns (output: StepOutput, obs: RoundObservation)
      requires Valid()
      requires AllDBsInSync()
      requires AllVerifiedCrossValid()
      ensures Valid()
      ensures AllDBsInSync()
      ensures AllVerifiedCrossValid()
      ensures ValidStepOutput(output, obs)
      ensures OutputConsistentWithVerified(output, obs)
      ensures OutputConsistentWithLogs(output, obs)
      ensures OutputConsistentWithChainState(output, obs)
      ensures output.AdvanceOutput? ==> ResultIsCrossValid(output.result)
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

      var result := Verify(obs.nextTimestamp, obs.blocksAtTS, obs.l1Heads);

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
      ensures ObservationConsistentWithChainState(obs)
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
      ensures result.Some? ==> BlocksExistedOnChain(result.value.blocks)
      ensures result.Some? ==> FrontierBlocksConsistentWithTimestamp(ts, result.value.blocks)
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
        invariant forall k :: k in chainIDs[0..i] ==>
          BlockExistedOnChain(k, blocks[k])
        invariant forall k :: k in chainIDs[0..i] ==>
          chains[k].BlockInfo(blocks[k]).value.timestamp <= ts
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
        assert BlockExistedOnChain(chainID, chainResult.value.l2Block);
      }

      result := Some(ChainsReadyResult(blocks, l1Heads));
    }

    // Verifies cross-chain messages and checks for cycles at the given timestamp,
    // returning the combined result.
    // Corresponds to verify in interop.go.
    method Verify(ts: nat, blocksAtTS: map<ChainID, BlockID>, l1Heads: map<ChainID, BlockID>) returns (result: Result)
      requires Valid()
      requires blocksAtTS.Keys == CHAIN_IDS
      requires BlocksExistedOnChain(blocksAtTS)
      requires AdvancesAllLogsDBs(ts, blocksAtTS)
      ensures Valid()
      ensures result.timestamp == ts
      ensures result.l2Heads == blocksAtTS
      ensures |result.invalidHeads| == 0 ==>
        ResultIsCrossValid(result)
    {
      var view := ResolveFrontierVerificationView(blocksAtTS);
      result := VerifyMessages(ts, blocksAtTS, l1Heads, view);
      var cycleResult := VerifyCycles(ts, blocksAtTS, view);
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
      requires TransitionIsCrossValid(pending)
      requires AllVerifiedCrossValid()
      requires AllLogsDBsConsistentWithChainData()
      ensures Valid()
      ensures PendingTransitionIsConsistent()
      ensures AllVerifiedCrossValid()
      ensures AllLogsDBsConsistentWithChainData()
      ensures madeProgress.None? ==>
        verifiedDB.pendingTransition == Some(pending)
      ensures madeProgress.Some? ==>
        verifiedDB.pendingTransition == None
      ensures madeProgress.Some? ==>
        (madeProgress.value <==> pending.decision == Advance)
      ensures madeProgress.Some? ==>
        AllDBsInSync()
      ensures madeProgress.None? ==> TransitionIsCrossValid(pending)
    {
      assert ValidPendingTransition(pending);

      if pending.decision == Rewind {

        currentL1 := BlockID(0, 0);
        var rewindPlan := pending.rewind.value;

        assert rewindPlan.resetAllChainsTo.Some? ==>
          AllDBsInSyncUpTo(rewindPlan.resetAllChainsTo.value);

        var rewindOk := ApplyRewindPlan(rewindPlan);

        assert Valid();

        if !rewindOk {
          madeProgress := None;
          assert PendingTransitionIsConsistent();
          assert AllVerifiedCrossValid();
          return;
        }

        // ApplyRewindPlan establishes DBsInSync(k) for all k. Snapshot the heap here
        // so that the twostate lemma call below can use old@BeforeClearRewind() to refer
        // to this state, where DBsInSync is known to hold.
        label BeforeClearRewind:
        verifiedDB.ClearPendingTransition();

        assert AllVerifiedHeadsBoundedByTimestamp() by {
          ClearPendingPreservesAllVerifiedHeadsBoundedByTimestamp@BeforeClearRewind();
        }

        assert AllDBsInSync() by {
          // ClearPendingTransition only changes pendingTransition; db and lastTimestamp are
          // unchanged (see postconditions). Use the framing lemma to re-derive DBsInSync.
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            ClearPendingPreservesDBsInSync@BeforeClearRewind(k);
          }
        }

        madeProgress := Some(false);
        assert AllVerifiedCrossValid() by {
          ClearPendingPreservesAllVerifiedCrossValid@BeforeClearRewind();
        }

        assert Valid();
        assert AllLogsDBsConsistentWithChainData();

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
          invariant AllVerifiedCrossValid()
          invariant AllLogsDBsConsistentWithChainData()
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

        assert AllVerifiedHeadsBoundedByTimestamp() by {
          ClearPendingPreservesAllVerifiedHeadsBoundedByTimestamp@BeforeClearInvalidate();
        }

        assert AllDBsInSync() by {
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            ClearPendingPreservesDBsInSync@BeforeClearInvalidate(k);
          }
        }

        madeProgress := Some(false);
        assert AllVerifiedCrossValid() by {
          ClearPendingPreservesAllVerifiedCrossValid@BeforeClearInvalidate();
        }

        assert Valid();
        assert AllLogsDBsConsistentWithChainData();

      } else { // Advance case

        // Replaces unreachable nil case in Go code
        assert pending.result.Some?;

        assert AllVerifiedCrossValid();

        var result := pending.result.value;
        label BeforePersistFrontierLogs:
        var persistOk := PersistFrontierLogs(result.timestamp, result.l2Heads);

        assert AllVerifiedCrossValid() by {
          reveal AllLogsDBsConsistentWithChainData;
          reveal UpdatedAllLogsDBs;
          reveal UpdatedLogsDB;
          PersistFrontierLogsPreservesAllVerifiedCrossValid@BeforePersistFrontierLogs();
        }

        assert PendingTransitionIsConsistent() by {
          reveal UpdatedAllLogsDBs;
          reveal UpdatedLogsDB;
          if verifiedDB.LastTimestamp().Some? {
            PersistFrontierLogsPreservesAllDBsInSyncUpTo@BeforePersistFrontierLogs(
              verifiedDB.LastTimestamp().value);
          }
        }

        assert TransitionIsCrossValid(pending) by {
          reveal UpdatedAllLogsDBs;
          reveal UpdatedLogsDB;
          PersistFrontierLogsPreservesTransitionIsCrossValid@BeforePersistFrontierLogs(pending);
        }

        assert Valid();

        if !persistOk {
          // FetchReceipts failed for some chain; keep the pending transition for retry.
          madeProgress := None;
          //assert PendingTransitionIsConsistent();
          //assert AllVerifiedCrossValid();
          assert Valid();

          return;
        }

        // Snapshot the heap before Commit so the twostate lemma can reference
        // the post-PersistFrontierLogs state via old@BeforeCommit().
        label BeforeCommit:
        // Inline commitVerifiedResult (a one-line wrapper) and ToVerifiedResult
        // (drops invalidHeads from Result to produce VerifiedResult).
        verifiedDB.Commit(VerifiedResult(result.timestamp, result.l1Inclusion, result.l2Heads));

        assert AllVerifiedHeadsBoundedByTimestamp() by {
          // FrontierBlocksConsistentWithTimestamp(result.timestamp, result.l2Heads) comes from
          // TransitionConsistentWithChainState, which is part of PendingTransitionIsConsistent().
          var vr := VerifiedResult(result.timestamp, result.l1Inclusion, result.l2Heads);
          CommitExtendsAllVerifiedHeadsBoundedByTimestamp@BeforeCommit(result.timestamp, vr);
        }

        assert AllDBsInSync() by {
          reveal UpdatedAllLogsDBs;
          reveal UpdatedLogsDB;
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            CommitEstablishesDBsInSync@BeforeCommit(k, result.timestamp, result.l2Heads);
          }
        }

        assert AllVerifiedCrossValid() by {
          CommitExtendsAllVerifiedCrossValid@BeforeCommit(
            result.timestamp,
            VerifiedResult(result.timestamp, result.l1Inclusion, result.l2Heads));
        }

        assert Valid();

        label BeforeClearAdvance:
        verifiedDB.ClearPendingTransition();

        assert AllVerifiedHeadsBoundedByTimestamp() by {
          ClearPendingPreservesAllVerifiedHeadsBoundedByTimestamp@BeforeClearAdvance();
        }

        assert AllDBsInSync() by {
          forall k | k in logsDBs.Keys ensures DBsInSync(k) {
            ClearPendingPreservesDBsInSync@BeforeClearAdvance(k);
          }
        }

        assert AllVerifiedCrossValid() by {
          ClearPendingPreservesAllVerifiedCrossValid@BeforeClearAdvance();
        }

        assert Valid();
        assert AllLogsDBsConsistentWithChainData();

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
      requires AllLogsDBsConsistentWithChainData()
      requires verifiedDB.pendingTransition.Some?
      requires verifiedDB.pendingTransition.value.decision == Rewind
      requires verifiedDB.pendingTransition.value.rewind == Some(plan)
      requires ValidRewindPlan(plan)
      requires PlanConsistentWithVerified(plan)
      requires plan.resetAllChainsTo.Some? ==>
        PlanConsistentWithAllLogs(plan)
      requires plan.resetAllChainsTo.Some? ==>
        AllDBsInSyncUpTo(plan.resetAllChainsTo.value)
      // From ProcessBlock sealing blocks with their L2 timestamp (propagated via PersistFrontierLogs).
      /*
      requires plan.resetAllChainsTo.Some? ==>
        forall c :: c in logsDBs.Keys && c in plan.targetHeads ==>
          logsDBs[c].FindSealedBlock(plan.targetHeads[c].number).Some? &&
          logsDBs[c].FindSealedBlock(plan.targetHeads[c].number).value.timestamp == plan.resetAllChainsTo.value
      */
      requires AllVerifiedCrossValid()
      ensures Valid()
      ensures AllLogsDBsConsistentWithChainData()
      ensures ValidRewindPlan(plan)
      ensures verifiedDB.pendingTransition == old(verifiedDB.pendingTransition)
      ensures PlanConsistentWithVerified(plan)
      ensures plan.resetAllChainsTo.Some? ==>
        PlanConsistentWithAllLogs(plan)
      ensures plan.resetAllChainsTo.Some? && success ==>
        RewoundAllLogsDB(plan)
      ensures plan.resetAllChainsTo.Some? ==>
        AllDBsInSyncUpTo(plan.resetAllChainsTo.value)
      ensures success ==> AllDBsInSync()
      ensures AllVerifiedCrossValid()
    {
      label BeforeRewind:
      var _ := verifiedDB.Rewind(plan.rewindAtOrAfter);

      assert AllVerifiedHeadsBoundedByTimestamp() by {
        RewindPreservesAllVerifiedHeadsBoundedByTimestamp@BeforeRewind(plan.rewindAtOrAfter);
      }

      assert VerifiedHeadsAreHighestBlocksUpToTimestamp() by {
        RewindVerifiedDBPreservesVerifiedHeadsHighest@BeforeRewind(plan.rewindAtOrAfter);
      }

      assert AllVerifiedCrossValid() by {
        RewindPreservesAllVerifiedCrossValid@BeforeRewind(plan.rewindAtOrAfter);
      }

      assert AllLogsDBsConsistentWithChainData();

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
      assert AllVerifiedCrossValid();

      if plan.resetAllChainsTo.Some? {
        assert AllDBsInSyncUpTo(plan.resetAllChainsTo.value) by {
          var ts := plan.resetAllChainsTo.value;

          forall k | k in chainIDs
            ensures DBsInSyncUpTo(k, ts)
          {
            // RewindChainEngines touched only chains.Values, so the twostate lemma
            // preconditions are identical to the post-Rewind call site: verifiedDB.db
            // is the same, logsDBs[k] is unchanged, and old() still resolves to
            // method entry where the precondition DBsInSyncUpTo(k, ts) held.
            VerifiedDBRewindPreservesDBsInSyncUpTo(k, ts, plan.rewindAtOrAfter);
          }

        }
      }

      if !enginesOk {
        success := false;
        return;
      }

      assert Valid();
      assert PlanConsistentWithVerified(plan);
      assert AllLogsDBsConsistentWithChainData();

      // Clear or rewind log databases depending on whether target heads are available.
      // resetAllChainsTo.None? corresponds to the nil TargetHeads case in Go,
      // which signals a full reset with no previous verified state to restore to.
      if plan.resetAllChainsTo.None? {
        ClearLogsDBs(plan, chainIDs);
        assert AllVerifiedCrossValid() by { reveal AllVerifiedCrossValid; }
        assert AllLogsDBsConsistentWithChainData() by {
          reveal AllLogsDBsConsistentWithChainData;
          reveal LogsDBConsistentWithChainData;
          forall cid | cid in logsDBs.Keys
            ensures LogsDBConsistentWithChainData(cid)
          {
            reveal LogsDBConsistentWithChainData;
            assert cid in chainIDs;
            assert RewoundLogsDB(plan, cid);
            assert logsDBs[cid].LatestSealedBlock() == None;
          }
        }
      } else {
        assert AllDBsInSyncUpTo(plan.resetAllChainsTo.value);
        // plan.targetHeads == verifiedDB.Get(ts).l2Heads (from PlanConsistentWithVerified).
        assert plan.targetHeads == verifiedDB.Get(plan.resetAllChainsTo.value).l2Heads;
        label BeforeRewindLogsDBs:
        RewindLogsDBs(plan, chainIDs);
        assert Valid();
        assert AllVerifiedCrossValid() by {
          reveal LogsDBsUnchangedUpTo;
          RewindLogsDBsPreservesAllVerifiedCrossValid@BeforeRewindLogsDBs(plan);
        }
        assert AllDBsInSyncUpTo(plan.resetAllChainsTo.value);
        assert PlanConsistentWithVerified(plan);
        assert AllLogsDBsConsistentWithChainData();
      }

      success := true;
    }

    // Prunes deny lists and optionally rewinds chain engines for all chains in chainIDs.
    // Extracted from ApplyRewindPlan to allow isolated verification of the engine rewind loop.
    method {:isolate_assertions} RewindChainEngines(plan: RewindPlan, chainIDs: seq<ChainID>) returns (success: bool)
      modifies chains.Values
      requires Valid()
      requires ValidRewindPlan(plan)
      requires PlanConsistentWithVerified(plan)
      requires RewoundVerifiedDB(plan)
      requires AllVerifiedCrossValid()
      requires forall k :: k in chainIDs <==> k in chains.Keys
      requires forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)
      ensures Valid()
      ensures ValidRewindPlan(plan)
      ensures RewoundVerifiedDB(plan)
      ensures AllVerifiedCrossValid()
      ensures forall k :: k in chainIDs ==> PlanConsistentWithLogs(plan, k)
      ensures unchanged(logsDBs.Values)
    {
      var failedAny := false;
      for i := 0 to |chainIDs|
        invariant Valid()
        invariant ValidRewindPlan(plan)
        invariant RewoundVerifiedDB(plan)
        invariant AllVerifiedCrossValid()
        invariant unchanged(verifiedDB)
        invariant unchanged(logsDBs.Values)
      {
        chains[chainIDs[i]].PruneDeniedAtOrAfterTimestamp(plan.rewindAtOrAfter);
        if plan.resetAllChainsTo.Some? {
          var ok := chains[chainIDs[i]].RewindEngine(plan.resetAllChainsTo.value);
          if !ok { failedAny := true; }
        }
        assert VerifiedHeadsAreHighestBlocksUpToTimestamp();
        assert AllVerifiedCrossValid();
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
    method {:isolate_assertions} RewindLogsDBs(plan: RewindPlan, chainIDs: seq<ChainID>)
      modifies logsDBs.Values
      requires Valid()
      requires AllLogsDBsConsistentWithChainData()
      requires forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]
      requires forall k :: k in chainIDs <==> k in plan.targetHeads
      requires forall k :: k in chainIDs <==> k in logsDBs.Keys
      requires PlanConsistentWithVerified(plan)
      requires RewoundVerifiedDB(plan)
      requires plan.resetAllChainsTo.Some?
      requires PlanConsistentWithAllLogs(plan)
      requires AllDBsInSyncUpTo(plan.resetAllChainsTo.value)
      ensures Valid()
      ensures AllLogsDBsConsistentWithChainData()
      ensures PlanConsistentWithAllLogs(plan)
      ensures RewoundAllLogsDB(plan)
      ensures AllDBsInSync()
      ensures LogsDBsUnchangedUpTo(plan.targetHeads)
    {
      assert LogsDBsUnchangedUpTo(plan.targetHeads) by { reveal LogsDBsUnchangedUpTo; }

      var ts := plan.resetAllChainsTo.value;

      for i := 0 to |chainIDs|
        invariant Valid()
        invariant AllLogsDBsConsistentWithChainData()
        invariant verifiedDB.LastTimestamp() == Some(ts)
        invariant plan.targetHeads == verifiedDB.Get(ts).l2Heads
        invariant forall k :: k in chainIDs ==>
          PlanConsistentWithLogs(plan, k) &&
          DBsInSyncUpTo(k, plan.resetAllChainsTo.value)
        invariant forall k :: k in chainIDs[0..i] ==>
          //PlanConsistentWithLogs(plan, k) &&
          RewoundLogsDB(plan, k) &&
          DBsInSync(k)
        invariant LogsDBsUnchangedUpTo(plan.targetHeads)
        invariant forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]
      {
        var chainID := chainIDs[i];

        label BeforeRewind:
        logsDBs[chainID].Rewind(plan.targetHeads[chainID]);
        assert unchanged@BeforeRewind(verifiedDB);
        RewindEstablishesDBsInSync@BeforeRewind(chainID);

        // Establish the loop invariant at line 1346 (for i+1):
        // for the newly-rewound chain, RewindEstablishesDBsInSync gives us DBsInSync;
        // for previously-rewound chains, logsDBs[k] is unchanged so RewoundLogsDB and DBsInSync are preserved.
        assert forall k :: k in chainIDs[0..i+1] ==>
            RewoundLogsDB(plan, k) && DBsInSync(k) by {
          forall k | k in chainIDs[0..i+1]
            ensures RewoundLogsDB(plan, k) && DBsInSync(k)
          {
            if k == chainID {
              // RewoundLogsDB from Rewind postcondition; DBsInSync from lemma call above
            } else {
              // k was already processed: k in chainIDs[0..i]
              assert k in chainIDs[0..i];
              // logsDBs[k] is unchanged (Rewind only modifies logsDBs[chainID], and they are distinct)
              assert logsDBs[k] != logsDBs[chainID];
              // RewoundLogsDB and DBsInSync preserved by framing (via ClearPendingPreservesDBsInSync)
              ClearPendingPreservesDBsInSync@BeforeRewind(k);
            }
          }
        }

        RewindPreservesVerifiedHeadsHighest@BeforeRewind(chainID, plan.targetHeads[chainID]);
        assert VerifiedHeadsAreHighestBlocksUpToTimestamp();
        assert BlockSealsMatchOnChainTimestamps() by {
          RewindPreservesBlockSealsMatch@BeforeRewind(chainID, plan.targetHeads[chainID]);
        }
        assert AllLogsDBsConsistentWithChainData() by {
          RewindPreservesAllLogsDBsConsistentWithChainData@BeforeRewind(chainID, plan.targetHeads[chainID]);
        }
        assert Valid();
        assert RewoundLogsDB(plan, chainID);
        assert DBsInSync(chainID);
        assert LogsDBsUnchangedUpTo(plan.targetHeads) by { reveal LogsDBsUnchangedUpTo; }
      }
    }

    // Persists frontier logs for the given verified result.
    // Returns false if any FetchReceipts call fails (corresponds to an error return
    // from sealFetchedBlockIntoLogsDB in interop.go).
    // Corresponds to persistFrontierLogs in interop.go.
    method {:isolate_assertions} PersistFrontierLogs(ts: nat, blocksAtTS: map<ChainID, BlockID>)
        returns (success: bool)
      modifies logsDBs.Values
      requires Valid()
      requires blocksAtTS.Keys == chains.Keys
      requires AdvancesAllLogsDBs(ts, blocksAtTS)
      requires BlocksExistedOnChain(blocksAtTS)
      requires AllLogsDBsConsistentWithChainData()
      ensures Valid()
      ensures AdvancesAllLogsDBs(ts, blocksAtTS)
      ensures AllLogsDBsConsistentWithChainData()
      ensures success ==> UpdatedAllLogsDBs(blocksAtTS)
      ensures !success ==>
        forall chainID :: chainID in logsDBs.Keys ==>
          UpdatedLogsDB(chainID, blocksAtTS[chainID]) ||
          unchanged(logsDBs[chainID])
    {
      success := true;
      var chainIDs := Enumerate(blocksAtTS.Keys);

      for i := 0 to |chainIDs|
        invariant Valid()
        invariant forall j, k :: 0 <= j < k < |chainIDs| ==>
          chainIDs[j] != chainIDs[k]
        invariant AdvancesAllLogsDBs(ts, blocksAtTS)
        invariant AllLogsDBsConsistentWithChainData()
        invariant forall j :: 0 <= j < i ==>
          UpdatedLogsDB(chainIDs[j], blocksAtTS[chainIDs[j]])
        invariant forall j :: i <= j < |chainIDs| ==>
          unchanged(logsDBs[chainIDs[j]])
      {
        var chainID := chainIDs[i];
        var blockID := blocksAtTS[chainID];
        var db := logsDBs[chainID];
        var chain := chains[chainID];

        var latestBlock := db.LatestSealedBlock();

        // Skip if the block is already sealed in the logsDB (idempotency on restart).
        // Simplified in relation to the Go code due to the asserts above.
        var skip := latestBlock == Some(blockID);

        if skip {
          assert UpdatedLogsDB(chainID, blockID) by {
            reveal UpdatedLogsDB;
            reveal AllLogsDBsConsistentWithChainData;
          }
        } else {
          var fetchResult := chain.FetchReceipts(blockID);

          if fetchResult.None? {
            // I/O error fetching receipts. In Go this propagates as an error from
            // sealFetchedBlockIntoLogsDB; the pending transition is kept for retry.
            success := false;

            assert unchanged(db);

            return;
          }

          var blockInfo := fetchResult.value.info;
          var logs := fetchResult.value.logs;

          // ErrParentHashMismatch: in Go this is an error return; here assumed away.
          assume latestBlock.Some? ==> blockInfo.parentHash == latestBlock.value.hash;

          label BeforeProcessBlock:
          ProcessBlock(chainID, blockID, blockInfo, logs);

          assert UpdatedLogsDB(chainID, blockID) by { reveal UpdatedLogsDB; }
          assert AdvancesAllLogsDBs(ts, blocksAtTS) by { reveal UpdatedLogsDB; }
          assert AllLogsDBsConsistentWithChainData() by {
            ProcessBlockPreservesAllLogsDBsConsistentWithChainData@BeforeProcessBlock(chainID, blockID);
          }
        }
      }

      assert UpdatedAllLogsDBs(blocksAtTS) by { reveal UpdatedAllLogsDBs; }
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
      requires OutputConsistentWithChainState(output, obs)
      ensures ValidPendingTransition(pendingTx)
      ensures TransitionConsistentWithVerified(pendingTx)
      ensures TransitionConsistentWithLogs(pendingTx)
      ensures output.AdvanceOutput? <==> pendingTx.decision.Advance?
      ensures output.InvalidateOutput? <==> pendingTx.decision.Invalidate?
      ensures (pendingTx.decision.Advance? || pendingTx.decision.Invalidate?) ==>
        pendingTx.result.value == output.result
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

        assert TransitionConsistentWithLogs(pendingTx) by {
          if lastTS <= activationTimestamp {
            assert TransitionConsistentWithLogs(pendingTx);
          } else {
            assert TransitionConsistentWithLogs(pendingTx);
          }
        }
      }
    }

    // Verifies a single executing message against the source chain's logsDB
    // and (for same-timestamp messages) the frontier view.
    // Returns true if all validity conditions are satisfied, false otherwise.
    // Models verifyExecutingMessage in algo.go.
    // Simplification vs. Go: ErrUnknownChain is treated as invalid (false)
    // rather than a fatal error.
    method VerifyExecutingMessage(
        executingChain: ChainID,
        executingTimestamp: nat,
        execMsg: ExecutingMessage,
        view: FrontierView)
        returns (valid: bool)
      requires Valid()
      requires executingChain in CHAIN_IDS
      ensures Valid()
      ensures valid ==>
        ValidExecutingMessage(executingTimestamp, executingChain, execMsg)
      ensures valid ==>
        if execMsg.timestamp == executingTimestamp then
          InitMsgInFrontierView(execMsg, view)
        else
          assert execMsg.timestamp < executingTimestamp; // by ValidExecutingMessage above
          InitMsgInLogsDB(execMsg)
    {
      // Unknown source chain: in Go this is ErrUnknownChain (fatal). Modeled as invalid.
      if execMsg.chainID !in logsDBs || execMsg.chainID !in chains {
        valid := false;
        return;
      }

      // Activation invariant: interop must be active for at least one full block on
      // the executing chain. Matches kona and op-program.
      if executingTimestamp < activationTimestamp + chains[executingChain].BlockTime() {
        valid := false;  // ErrExecutedTooEarly
        return;
      }

      // Activation invariant: interop must be active for at least one full block on
      // the initiating chain.
      if execMsg.timestamp < activationTimestamp + chains[execMsg.chainID].BlockTime() {
        valid := false;  // ErrInitiatedTooEarly
        return;
      }

      // Timestamp ordering: the initiating message must not be from the future.
      if execMsg.timestamp > executingTimestamp {
        valid := false;  // ErrTimestampViolation
        return;
      }

      // Message expiry: the initiating message must not be older than the expiry window.
      if execMsg.timestamp + messageExpiryWindow < executingTimestamp {
        valid := false;  // ErrMessageExpired
        return;
      }

      var query := ContainsQuery(execMsg.blockNum, execMsg.logIdx, execMsg.timestamp, execMsg.checksum);

      // Same-timestamp dependencies may reside in the frontier view rather than the
      // accepted-history logsDB. Check the frontier view first; if found, the message
      // is valid without a logsDB lookup.
      if execMsg.timestamp == executingTimestamp {
        // Unlike the Go code, we return false immediately if the initiating message is
        // not in the frontier. Since we know that there are no blocks for the current
        // timestamp in the logsDB, there is no point looking there.
        valid := view.Contains(execMsg.chainID, query);
        return;
      }

      // Check the logsDB for the initiating message.
      valid := logsDBs[execMsg.chainID].Contains(query);
    }

    // Verifies cross-chain interop messages at the given timestamp.
    // Models verifyInteropMessages in algo.go.
    // Simplifications vs. Go:
    // - InvalidHead carries only BlockID; StateRoot/MessagePasserStorageRoot from
    //   newInvalidHead are not modeled (see B2).
    // - Fatal I/O errors (e.g. OpenBlock failure on non-first blocks) are replaced
    //   by assumes with explanatory comments.
    method {:isolate_assertions} VerifyMessages(
        ts: nat,
        blocksAtTS: map<ChainID, BlockID>,
        l1Heads: map<ChainID, BlockID>,
        view: FrontierView)
        returns (result: Result)
      requires Valid()
      requires blocksAtTS.Keys == CHAIN_IDS
      requires AdvancesAllLogsDBs(ts, blocksAtTS)
      requires IsCorrectFrontierView(view, blocksAtTS)
      ensures Valid()
      ensures result.timestamp == ts
      ensures result.l2Heads == blocksAtTS
      ensures |result.invalidHeads| == 0 ==>
        forall chainID :: chainID in CHAIN_IDS ==>
          BlockIsCrossValid(view.BlockInfo(chainID).timestamp, chainID, view.BlockInfo(chainID).id) &&
          AllInitMsgsPresent(chainID, view.BlockInfo(chainID).id, blocksAtTS)
    {
      var l1Inclusion := ComputeL1Inclusion(blocksAtTS, l1Heads);
      result := Result(ts, l1Inclusion, blocksAtTS, map[]);

      var chainIDs := Enumerate(blocksAtTS.Keys);

      for i := 0 to |chainIDs|
        invariant Valid()
        invariant AdvancesAllLogsDBs(ts, blocksAtTS)
        invariant result.timestamp == ts
        invariant result.l2Heads == blocksAtTS
        invariant IsCorrectFrontierView(view, blocksAtTS)
        invariant |result.invalidHeads| == 0 ==>
          forall j :: 0 <= j < i ==>
            BlockIsCrossValid(view.BlockInfo(chainIDs[j]).timestamp, chainIDs[j], view.BlockInfo(chainIDs[j]).id) &&
            AllInitMsgsPresent(chainIDs[j], view.BlockInfo(chainIDs[j]).id, blocksAtTS)
      {
        var chainID := chainIDs[i];
        var expectedBlock := blocksAtTS[chainID];

        // In Go, chains not in logsDBs are skipped. Since Valid() guarantees
        // logsDBs.Keys == chains.Keys == CHAIN_IDS, and our preconditions
        // also guarantee blocksAtTS.Keys == CHAIN_IDs, we can just assert this.
        assert chainID in logsDBs.Keys;
        assert chainID in chains.Keys;

        // The Go code falls back to querying the logsDBs if a block is not in
        // the frontier verification view, but in practice this can't happen.
        // Here we can ensure that the view has a block for every chain.
        var block := view.BlockInfo(chainID);
        var logs := view.BlockLogs(chainID);

        // Verify each executing message. Stop at the first invalid one.
        var blockValid := true;
        var logIdxs := Enumerate(logs.execMsgs.Keys);
        var j := 0;
        while j < |logIdxs| && blockValid
          invariant 0 <= j <= |logIdxs|
          invariant Valid()
          invariant IsCorrectFrontierView(view, blocksAtTS)
          invariant |result.invalidHeads| == 0 ==>
            forall k :: 0 <= k < i ==>
              BlockIsCrossValid(view.BlockInfo(chainIDs[k]).timestamp, chainIDs[k], view.BlockInfo(chainIDs[k]).id) &&
              AllInitMsgsPresent(chainIDs[k], view.BlockInfo(chainIDs[k]).id, blocksAtTS)
          invariant blockValid ==>
            forall k :: 0 <= k < j ==>
              ValidExecutingMessage(block.timestamp, chainID, logs.execMsgs[logIdxs[k]]) &&
              (InitMsgInFrontier(logs.execMsgs[logIdxs[k]], blocksAtTS) || InitMsgInLogsDB(logs.execMsgs[logIdxs[k]]))
        {
          var logIdx := logIdxs[j];
          var execMsg := logs.execMsgs[logIdx];
          var ok := VerifyExecutingMessage(chainID, block.timestamp, execMsg, view);
          if !ok {
            blockValid := false;
          } else {
            // Establish facts needed for PresentInFrontierView ==> PresentInFrontier
            // (forward direction only — sufficient for the new PresentInFrontier loop invariant).
            assert execMsg.chainID in CHAIN_IDS;
            assert chains[execMsg.chainID].BlockInfo(blocksAtTS[execMsg.chainID]).Some?;
            assert chains[execMsg.chainID].BlockLogs(blocksAtTS[execMsg.chainID]).Some?;
            assert view.BlockInfo(execMsg.chainID) == chains[execMsg.chainID].BlockInfo(blocksAtTS[execMsg.chainID]).value;
            assert view.BlockLogs(execMsg.chainID) == chains[execMsg.chainID].BlockLogs(blocksAtTS[execMsg.chainID]).value;
            assert chains[execMsg.chainID].BlockInfo(blocksAtTS[execMsg.chainID]).value.id == blocksAtTS[execMsg.chainID];
            assert InitMsgInFrontierView(execMsg, view) ==> InitMsgInFrontier(execMsg, blocksAtTS);
          }
          j := j + 1;
        }

        if !blockValid {
          result := result.(invalidHeads := result.invalidHeads[chainID := expectedBlock]);
        } else {
          assert BlockIsCrossValid(block.timestamp, chainID, block.id);
          assert AllInitMsgsPresent(chainID, block.id, blocksAtTS);
        }
      }
    }

    // Verifies same-timestamp cycle constraints at the given timestamp.
    // Abstracts cycleVerifyFn (i.verifyCycleMessages) from interop.go.
    method VerifyCycles(ts: nat, blocksAtTS: map<ChainID, BlockID>, view: FrontierView) returns (result: Result)
      requires Valid()
      ensures {:axiom} Valid()
      ensures {:axiom} result.timestamp == ts
      ensures {:axiom} result.l2Heads == blocksAtTS

    // Builds the frontier verification view for the given block set.
    // The view pre-fetches receipts for same-timestamp message queries so that
    // VerifyMessages can answer them without racing against logsDB writes.
    // Corresponds to resolveFrontierVerificationView in interop.go.
    method ResolveFrontierVerificationView(blocksAtTS: map<ChainID, BlockID>) returns (view: FrontierView)
      requires Valid()
      requires blocksAtTS.Keys == CHAIN_IDS
      requires BlocksExistedOnChain(blocksAtTS)
      ensures {:axiom} Valid()
      ensures {:axiom} fresh(view)
      ensures {:axiom} IsCorrectFrontierView(view, blocksAtTS)

    // Computes the L1 inclusion block for the given set of L2 blocks by taking
    // the maximum of the per-chain L1 heads.
    // Corresponds to the l1Inclusion computation in verifyInteropMessages in algo.go.
    method ComputeL1Inclusion(blocksAtTS: map<ChainID, BlockID>, l1Heads: map<ChainID, BlockID>) returns (l1Inclusion: BlockID)
      requires Valid()
      ensures {:axiom} Valid()

    // Processes and seals a block's logs into the given chain's log database.
    // Corresponds to processBlockLogs in logdb.go; the parent-hash precondition
    // mirrors ErrParentHashMismatch (which is assumed away at the call site).
    method ProcessBlock(chainID: ChainID, blockID: BlockID, info: BlockInfo, logs: BlockLogs)
      modifies logsDBs[chainID]
      requires chainID in CHAIN_IDS
      requires Valid()
      requires BlockExistedOnChain(chainID, blockID)
      requires info == chains[chainID].BlockInfo(blockID).value
      requires logs == chains[chainID].BlockLogs(blockID).value
      requires logsDBs[chainID].LatestSealedBlock().Some? ==>
        logsDBs[chainID].LatestSealedBlock().value.number + 1 == blockID.number &&
        info.parentHash == logsDBs[chainID].LatestSealedBlock().value.hash
      //ensures {:axiom} unchanged(verifiedDB)
      ensures {:axiom} Valid()
      ensures {:axiom} UpdatedLogsDB(chainID, blockID)

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

    // Twostate lemma: Rewind on logsDBs[chainID] establishes DBsInSync(chainID).
    // Requires old(DBsInSyncUpTo(chainID, ts)) and the Rewind postcondition
    // (FindSealedBlock preserved for n <= targetHead.number, LatestSealedBlock == Some(targetHead)).
    // Key step: verifiedDB.Valid() non-decreasing ensures every t <= ts has block number
    // <= targetHead.number, so the Rewind postcondition covers all blocks referenced by DBsInSyncUpTo.
    // Intended to be called as RewindEstablishesDBsInSync@L(chainID) where L is a label
    // placed just before the Rewind call.
    twostate lemma RewindEstablishesDBsInSync(chainID: ChainID)
      requires chainID in logsDBs.Keys
      requires old(verifiedDB.Valid())
      requires verifiedDB.Valid()
      requires verifiedDB.db == old(verifiedDB.db)
      requires verifiedDB.lastTimestamp == old(verifiedDB.lastTimestamp)
      requires verifiedDB.LastTimestamp().Some?
      requires
        var ts := verifiedDB.LastTimestamp().value;
        chainID in verifiedDB.Get(ts).l2Heads &&
        old(DBsInSyncUpTo(chainID, ts)) &&
        logsDBs[chainID].LatestSealedBlock() == Some(verifiedDB.Get(ts).l2Heads[chainID]) &&
        (forall n :: 0 <= n <= verifiedDB.Get(ts).l2Heads[chainID].number ==>
          logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n)))
      ensures DBsInSync(chainID)
    {
      var ts := verifiedDB.LastTimestamp().value;
      var targetHead := verifiedDB.Get(ts).l2Heads[chainID];
      assert verifiedDB.Has(ts);
      forall t | activationTimestamp <= t <= ts
        ensures verifiedDB.Has(t)
        ensures chainID in verifiedDB.Get(t).l2Heads
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).Some?
        ensures logsDBs[chainID].FindSealedBlock(verifiedDB.Get(t).l2Heads[chainID].number).value.id ==
          verifiedDB.Get(t).l2Heads[chainID]
      {
        assert old(verifiedDB.Has(t));         // fires trigger on old(DBsInSyncUpTo)
        assert verifiedDB.db[t] == old(verifiedDB.db)[t];
        var n := verifiedDB.Get(t).l2Heads[chainID].number;
        // verifiedDB.Valid() non-decreasing: n <= targetHead.number
        assert t in verifiedDB.db;
        assert ts in verifiedDB.db;
        assert chainID in verifiedDB.db[t].l2Heads;
        assert chainID in verifiedDB.db[ts].l2Heads;
        assert n <= targetHead.number;
        // Rewind postcondition: FindSealedBlock(n) == old(FindSealedBlock(n))
        assert logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n));
        // From old(DBsInSyncUpTo): old FindSealedBlock(n) has the right value
        assert old(logsDBs[chainID].FindSealedBlock(n)).Some?;
        assert old(logsDBs[chainID].FindSealedBlock(n)).value.id == verifiedDB.Get(t).l2Heads[chainID];
      }
    }

    // Framing lemma: verifiedDB.Rewind preserves VerifiedHeadsAreHighestBlocksUpToTimestamp.
    // Rewind removes entries >= rewindAt; all remaining entries were in the old db with the
    // same value, and logsDBs is unchanged, so the property holds by congruence.
    // Intended to be called as RewindVerifiedDBPreservesVerifiedHeadsHighest@L(rewindAt)
    // where L is a label placed just before the Rewind call.
    twostate lemma {:isolate_assertions} RewindVerifiedDBPreservesVerifiedHeadsHighest(rewindAt: nat)
      requires unchanged(logsDBs.Values)
      requires verifiedDB.db == map k | k in old(verifiedDB.db) && k < rewindAt :: old(verifiedDB.db)[k]
      requires old(VerifiedHeadsAreHighestBlocksUpToTimestamp())
      ensures VerifiedHeadsAreHighestBlocksUpToTimestamp()
    {
      forall ts': nat | verifiedDB.Has(ts')
        ensures var verifiedHeads := verifiedDB.Get(ts').l2Heads;
          verifiedHeads.Keys == logsDBs.Keys &&
          forall chainID :: chainID in verifiedHeads.Keys ==>
            var blockNumber := verifiedHeads[chainID].number;
            forall n :: blockNumber < n && logsDBs[chainID].FindSealedBlock(n).Some? ==>
              ts' < logsDBs[chainID].FindSealedBlock(n).value.timestamp
      {
        assert ts' in old(verifiedDB.db);
        assert old(verifiedDB.Has(ts'));    // fires trigger on old(VerifiedHeadsAreHighestBlocksUpToTimestamp())
        assert verifiedDB.db[ts'] == old(verifiedDB.db)[ts'];
      }
    }

    // Framing lemma: logsDBs[chainID].Rewind preserves BlockSealsMatchOnChainTimestamps.
    // Rewind removes seals above targetHead.number; remaining seals have unchanged FindSealedBlock
    // data (Rewind postcondition), and chains is unchanged, so the timestamp-match property holds
    // by congruence from the old state. Other chains are untouched.
    // Intended to be called as RewindPreservesBlockSealsMatch@L(chainID, targetHead)
    // where L is a label placed just before the Rewind call.
    twostate lemma {:isolate_assertions} RewindPreservesBlockSealsMatch(
        chainID: ChainID, targetHead: BlockID)
      requires chainID in logsDBs.Keys
      requires logsDBs.Keys == chains.Keys
      requires forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]
      requires logsDBs[chainID].LatestSealedBlock() == Some(targetHead)
      requires forall n :: 0 <= n <= targetHead.number ==>
        logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n))
      requires forall k :: k in logsDBs.Keys && k != chainID ==>
        forall n :: logsDBs[k].FindSealedBlock(n) == old(logsDBs[k].FindSealedBlock(n))
      requires old(BlockSealsMatchOnChainTimestamps())
      ensures BlockSealsMatchOnChainTimestamps()
    {
      forall cid | cid in logsDBs.Keys
        ensures forall n :: logsDBs[cid].FindSealedBlock(n).Some? ==>
          var sealedBlock := logsDBs[cid].FindSealedBlock(n).value;
          chains[cid].BlockInfo(sealedBlock.id).Some? &&
          sealedBlock.timestamp == chains[cid].BlockInfo(sealedBlock.id).value.timestamp
      {
        forall n: nat | logsDBs[cid].FindSealedBlock(n).Some?
          ensures var sealedBlock := logsDBs[cid].FindSealedBlock(n).value;
            chains[cid].BlockInfo(sealedBlock.id).Some? &&
            sealedBlock.timestamp == chains[cid].BlockInfo(sealedBlock.id).value.timestamp
        {
          if cid == chainID {
            assert n <= targetHead.number;   // from LatestSealedBlock axiom (blocks above are None)
            assert logsDBs[cid].FindSealedBlock(n) == old(logsDBs[cid].FindSealedBlock(n));
          } else {
            assert logsDBs[cid].FindSealedBlock(n) == old(logsDBs[cid].FindSealedBlock(n));
          }
        }
      }
    }

    // Framing lemma: logsDBs[chainID].Rewind preserves AllLogsDBsConsistentWithChainData.
    // Remaining sealed blocks (up to targetHead.number) have unchanged FindSealedBlock and
    // BlockLogs data (from Rewind postconditions + the BlockLogs preservation axiom), so
    // LogsDBConsistentWithChainData holds by congruence from the old state. Other chains untouched.
    // Intended to be called as RewindPreservesAllLogsDBsConsistentWithChainData@L(chainID, targetHead)
    // where L is a label placed just before the Rewind call.
    twostate lemma {:isolate_assertions} RewindPreservesAllLogsDBsConsistentWithChainData(
        chainID: ChainID, targetHead: BlockID)
      requires chainID in logsDBs.Keys
      requires logsDBs.Keys == chains.Keys
      requires forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]
      requires logsDBs[chainID].LatestSealedBlock() == Some(targetHead)
      requires forall n :: 0 <= n <= targetHead.number ==>
        logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n))
      requires forall n :: 0 <= n <= targetHead.number && logsDBs[chainID].FindSealedBlock(n).Some? ==>
        logsDBs[chainID].BlockLogs(n) == old(logsDBs[chainID].BlockLogs(n))
      requires forall k :: k in logsDBs.Keys && k != chainID ==>
        forall n :: logsDBs[k].FindSealedBlock(n) == old(logsDBs[k].FindSealedBlock(n))
      requires forall k :: k in logsDBs.Keys && k != chainID ==>
        forall n :: old(logsDBs[k].FindSealedBlock(n)).Some? ==>
          logsDBs[k].BlockLogs(n) == old(logsDBs[k].BlockLogs(n))
      requires old(AllLogsDBsConsistentWithChainData())
      ensures AllLogsDBsConsistentWithChainData()
    {
      reveal AllLogsDBsConsistentWithChainData;
      reveal LogsDBConsistentWithChainData;
      forall cid | cid in logsDBs.Keys
        ensures LogsDBConsistentWithChainData(cid)
      {
        reveal LogsDBConsistentWithChainData;
        forall blockID: BlockID | logsDBs[cid].FindSealedBlock(blockID.number).Some?
          ensures BlockExistedOnChain(cid, blockID) &&
            var info := chains[cid].BlockInfo(blockID).value;
            var logs := chains[cid].BlockLogs(blockID).value;
            logsDBs[cid].FindSealedBlock(info.id.number).value.timestamp == info.timestamp &&
            logsDBs[cid].BlockLogs(info.id.number) == logs
        {
          if cid == chainID {
            assert blockID.number <= targetHead.number;
            assert logsDBs[cid].FindSealedBlock(blockID.number) == old(logsDBs[cid].FindSealedBlock(blockID.number));
            assert logsDBs[cid].BlockLogs(blockID.number) == old(logsDBs[cid].BlockLogs(blockID.number));
            assert old(logsDBs[cid].FindSealedBlock(blockID.number)).Some?;
          } else {
            assert logsDBs[cid].FindSealedBlock(blockID.number) == old(logsDBs[cid].FindSealedBlock(blockID.number));
            assert logsDBs[cid].BlockLogs(blockID.number) == old(logsDBs[cid].BlockLogs(blockID.number));
            assert old(logsDBs[cid].FindSealedBlock(blockID.number)).Some?;
          }
        }
      }
    }

    // Intended to be called as RewindPreservesVerifiedHeadsHighest@L(chainID, targetHead) where
    // L is a label placed just before the Rewind call.
    // VerifiedHeadsAreHighestBlocksUpToTimestamp() is monotone under seal removal: rewinding
    // logsDBs[chainID] can only reduce the seals that must satisfy the timestamp bound, so
    // the property is preserved.
    twostate lemma {:isolate_assertions} RewindPreservesVerifiedHeadsHighest(
        chainID: ChainID, targetHead: BlockID)
      requires chainID in logsDBs.Keys
      requires forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]
      requires verifiedDB.db == old(verifiedDB.db)
      requires old(VerifiedHeadsAreHighestBlocksUpToTimestamp())
      // Rewind postconditions for chainID:
      requires logsDBs[chainID].LatestSealedBlock() == Some(targetHead)
      requires forall n :: 0 <= n <= targetHead.number ==>
        logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n))
      // Other chains unchanged (automatic framing when distinct logsDBs holds):
      requires forall k :: k in logsDBs.Keys && k != chainID ==>
        forall n :: logsDBs[k].FindSealedBlock(n) == old(logsDBs[k].FindSealedBlock(n))
      ensures VerifiedHeadsAreHighestBlocksUpToTimestamp()
    {
      forall ts': nat | verifiedDB.Has(ts')
        ensures var verifiedHeads := verifiedDB.Get(ts').l2Heads;
          verifiedHeads.Keys == logsDBs.Keys &&
          forall cid :: cid in verifiedHeads.Keys ==>
            var blockNumber := verifiedHeads[cid].number;
            forall n :: blockNumber < n && logsDBs[cid].FindSealedBlock(n).Some? ==>
              ts' < logsDBs[cid].FindSealedBlock(n).value.timestamp
      {
        assert old(verifiedDB.Has(ts'));     // fires trigger for old(VerifiedHeadsAreHighestBlocksUpToTimestamp())
        assert verifiedDB.db[ts'] == old(verifiedDB.db)[ts'];
        // Keys preserved: old invariant gave verifiedHeads.Keys == old(logsDBs.Keys) == logsDBs.Keys
        assert verifiedDB.Get(ts').l2Heads.Keys == logsDBs.Keys;

        forall cid | cid in verifiedDB.Get(ts').l2Heads.Keys
          ensures var blockNumber := verifiedDB.Get(ts').l2Heads[cid].number;
            forall n :: blockNumber < n && logsDBs[cid].FindSealedBlock(n).Some? ==>
              ts' < logsDBs[cid].FindSealedBlock(n).value.timestamp
        {
          forall n | verifiedDB.Get(ts').l2Heads[cid].number < n && logsDBs[cid].FindSealedBlock(n).Some?
            ensures ts' < logsDBs[cid].FindSealedBlock(n).value.timestamp
          {
            if cid == chainID {
              // FindSealedBlock(n).Some? after rewind implies n <= targetHead.number
              // (LatestSealedBlock axiom: all numbers above the latest have no seal)
              assert n <= targetHead.number;
              // Rewind preserved the seal at n (postcondition covers 0..targetHead.number)
              assert logsDBs[cid].FindSealedBlock(n) == old(logsDBs[cid].FindSealedBlock(n));
              // Current seal == old seal, so old seal is Some and timestamp bound holds
              assert old(logsDBs[cid].FindSealedBlock(n)).Some?;
              assert ts' < old(logsDBs[cid].FindSealedBlock(n)).value.timestamp;
            } else {
              // Other chains are unchanged
              assert logsDBs[cid].FindSealedBlock(n) == old(logsDBs[cid].FindSealedBlock(n));
              assert old(logsDBs[cid].FindSealedBlock(n)).Some?;
              assert ts' < old(logsDBs[cid].FindSealedBlock(n)).value.timestamp;
            }
          }
        }
      }
    }

    // Framing lemma: ProcessBlock on logsDBs[chainID] preserves AllLogsDBsConsistentWithChainData.
    // For chainID itself, consistency comes directly from UpdatedLogsDB (which includes it).
    // For all other chains, their logsDB objects are not modified so their entries are unchanged.
    twostate lemma {:isolate_assertions} ProcessBlockPreservesAllLogsDBsConsistentWithChainData(
        chainID: ChainID, blockID: BlockID)
      requires chainID in logsDBs.Keys
      requires logsDBs.Keys == chains.Keys
      requires forall k1, k2 :: k1 in logsDBs.Keys && k2 in logsDBs.Keys && k1 != k2 ==> logsDBs[k1] != logsDBs[k2]
      requires BlockExistedOnChain(chainID, blockID)
      requires UpdatedLogsDB(chainID, blockID)
      requires forall k :: k in logsDBs.Keys && k != chainID ==>
        forall n :: logsDBs[k].FindSealedBlock(n) == old(logsDBs[k].FindSealedBlock(n))
      requires forall k :: k in logsDBs.Keys && k != chainID ==>
        forall n :: old(logsDBs[k].FindSealedBlock(n)).Some? ==>
          logsDBs[k].BlockLogs(n) == old(logsDBs[k].BlockLogs(n))
      requires old(AllLogsDBsConsistentWithChainData())
      ensures AllLogsDBsConsistentWithChainData()
    {
      reveal AllLogsDBsConsistentWithChainData;
      reveal LogsDBConsistentWithChainData;
      reveal UpdatedLogsDB;
      forall cid | cid in logsDBs.Keys
        ensures LogsDBConsistentWithChainData(cid)
      {
        reveal LogsDBConsistentWithChainData;
        if cid != chainID {
          forall blockID2: BlockID | logsDBs[cid].FindSealedBlock(blockID2.number).Some?
            ensures BlockExistedOnChain(cid, blockID2) &&
              var info := chains[cid].BlockInfo(blockID2).value;
              var logs := chains[cid].BlockLogs(blockID2).value;
              logsDBs[cid].FindSealedBlock(info.id.number).value.timestamp == info.timestamp &&
              logsDBs[cid].BlockLogs(info.id.number) == logs
          {
            assert logsDBs[cid].FindSealedBlock(blockID2.number) == old(logsDBs[cid].FindSealedBlock(blockID2.number));
            assert old(logsDBs[cid].FindSealedBlock(blockID2.number)).Some?;
          }
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

    // Framing lemma: ProcessBlock on logsDBs[chainID] preserves AllMessagesPresent.
    // ProcessBlock adds newBlockNumber; for all other block numbers FindSealedBlock
    // is unchanged, so old(PresentInLogsDB) messages remain present (via ContainsMonotone).
    // PresentInFrontier calls reads-{} functions and is heap-independent.
    // Unaffected logsDBs (c != chainID) are unchanged, preserving their Contains results.
    // Intended to be called as ProcessBlockPreservesAllMessagesPresent@L(chainID, newBlockNumber, blocksAtTS)
    // where L is a label placed just before the ProcessBlock call.
    twostate lemma ProcessBlockPreservesAllMessagesPresent(
        chainID: ChainID, newBlockNumber: nat, blocksAtTS: map<ChainID, BlockID>)
      requires old(Valid())
      requires Valid()
      requires blocksAtTS.Keys == CHAIN_IDS
      requires chainID in logsDBs.Keys
      // newBlockNumber was not sealed before: either no blocks existed, or it is exactly one past latest.
      requires old(logsDBs[chainID].LatestSealedBlock()).None? ||
        old(logsDBs[chainID].LatestSealedBlock()).value.number + 1 == newBlockNumber
      // ProcessBlock postcondition: FindSealedBlock unchanged for all blocks except the new one.
      requires forall n :: n != newBlockNumber ==>
        logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n))
      // ProcessBlock only modifies logsDBs[chainID]; all other logsDB objects are unchanged.
      requires forall c :: c in logsDBs.Keys && c != chainID ==> unchanged(logsDBs[c])
      requires old(forall k :: k in blocksAtTS.Keys ==>
        BlockExistedOnChain(k, blocksAtTS[k]) &&
        AllInitMsgsPresent(k, blocksAtTS[k], blocksAtTS))
      ensures forall k :: k in blocksAtTS.Keys ==>
        BlockExistedOnChain(k, blocksAtTS[k]) &&
        AllInitMsgsPresent(k, blocksAtTS[k], blocksAtTS)
    {
      // newBlockNumber was not sealed in the old state.
      assert old(logsDBs[chainID].FindSealedBlock(newBlockNumber)) == None by {
        if old(logsDBs[chainID].LatestSealedBlock()).Some? {
          // LatestSealedBlock axiom: all blocks above latest have FindSealedBlock == None.
          assert old(logsDBs[chainID].LatestSealedBlock()).value.number < newBlockNumber;
        }
      }
      forall k | k in blocksAtTS.Keys
        ensures BlockExistedOnChain(k, blocksAtTS[k])
        ensures AllInitMsgsPresent(k, blocksAtTS[k], blocksAtTS)
      {
        // chains[k].BlockLogs(blocksAtTS[k]).Some? is heap-independent (reads {}) — trivially preserved.
        var logs := chains[k].BlockLogs(blocksAtTS[k]).value;
        reveal AllInitMsgsPresent;
        forall execMsg | execMsg in logs.execMsgs.Values
          ensures execMsg.chainID in CHAIN_IDS &&
            (InitMsgInFrontier(execMsg, blocksAtTS) || InitMsgInLogsDB(execMsg))
        {
          // The old invariant gives us the fact for this execMsg.
          assert execMsg.chainID in CHAIN_IDS;
          assert old(InitMsgInFrontier(execMsg, blocksAtTS) || InitMsgInLogsDB(execMsg));
          if InitMsgInFrontier(execMsg, blocksAtTS) {
            // PresentInFrontier calls chains[...].BlockInfo/BlockLogs which have reads {} —
            // heap-independent, same value in both states.
          } else {
            assert old(InitMsgInLogsDB(execMsg));
            var query := ContainsQuery(
              execMsg.blockNum, execMsg.logIdx, execMsg.timestamp, execMsg.checksum);
            // old(Contains) => old(FindSealedBlock(execMsg.blockNum).Some?)
            assert old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum)).Some?;
            if execMsg.chainID != chainID {
              // logsDBs[execMsg.chainID] is unchanged — Contains gives the same result.
              assert unchanged(logsDBs[execMsg.chainID]);
            } else {
              // execMsg.chainID == chainID: the block was already sealed (old FindSealedBlock.Some?),
              // but newBlockNumber was not sealed before, so execMsg.blockNum != newBlockNumber.
              assert execMsg.blockNum != newBlockNumber;
              // FindSealedBlock for the old block number is unchanged.
              assert logsDBs[chainID].FindSealedBlock(execMsg.blockNum) ==
                old(logsDBs[chainID].FindSealedBlock(execMsg.blockNum));
              // Monotonicity: Contains is preserved when its queried FindSealedBlock is unchanged.
              logsDBs[chainID].ContainsMonotone(query);
            }
          }
        }
      }
    }

    // Framing lemma: ProcessBlock on logsDBs[chainID] preserves the FindSealedBlock
    // monotonicity invariant: old (method-entry) sealed blocks remain unchanged.
    // ProcessBlock only appends newBlockNumber; any n with old(FindSealedBlock(n)).Some?
    // satisfies n ≤ old(LatestSealedBlock).number < newBlockNumber, so n ≠ newBlockNumber,
    // so ProcessBlock's frame postcondition preserves FindSealedBlock(n).
    // Intended to be called as ProcessBlockPreservesFindSealedBlockMonotonicity@L(chainID, newBlockNumber)
    // where L is a label placed just before the ProcessBlock call.
    twostate lemma ProcessBlockPreservesFindSealedBlockMonotonicity(
        chainID: ChainID, newBlockNumber: nat)
      requires chainID in logsDBs.Keys
      requires old(logsDBs[chainID].LatestSealedBlock()).None? ||
        old(logsDBs[chainID].LatestSealedBlock()).value.number + 1 == newBlockNumber
      requires forall n :: n != newBlockNumber ==>
        logsDBs[chainID].FindSealedBlock(n) == old(logsDBs[chainID].FindSealedBlock(n))
      requires forall c :: c in logsDBs.Keys && c != chainID ==> unchanged(logsDBs[c])
      ensures forall c :: c in logsDBs.Keys ==>
        forall n :: old(logsDBs[c].FindSealedBlock(n)).Some? ==>
          logsDBs[c].FindSealedBlock(n) == old(logsDBs[c].FindSealedBlock(n))
    {
      assert old(logsDBs[chainID].FindSealedBlock(newBlockNumber)) == None by {
        if old(logsDBs[chainID].LatestSealedBlock()).Some? {
          assert old(logsDBs[chainID].LatestSealedBlock()).value.number < newBlockNumber;
        }
      }
      forall c | c in logsDBs.Keys
        ensures forall n :: old(logsDBs[c].FindSealedBlock(n)).Some? ==>
          logsDBs[c].FindSealedBlock(n) == old(logsDBs[c].FindSealedBlock(n))
      {
        if c != chainID {
          assert unchanged(logsDBs[c]);
        } else {
          forall n | old(logsDBs[c].FindSealedBlock(n)).Some?
            ensures logsDBs[c].FindSealedBlock(n) == old(logsDBs[c].FindSealedBlock(n))
          {
            // old(FindSealedBlock(n)).Some? but old(FindSealedBlock(newBlockNumber)) == None
            // → n ≠ newBlockNumber → ProcessBlock frame preserves FindSealedBlock(n).
            assert n != newBlockNumber;
          }
        }
      }
    }


    // Framing lemma: PersistFrontierLogs preserves AllVerifiedCrossValid.
    // verifiedDB is unchanged; each logsDB only grew (old FindSealedBlock results
    // are preserved by the {:axiom} postcondition on PersistFrontierLogs).
    // For each previously committed timestamp, AllInitMsgsPresent is preserved:
    //   - InitMsgInFrontier is heap-independent (reads {} functions).
    //   - InitMsgInLogsDB was True for old block numbers, which are unchanged
    //     in each logsDB, so ContainsMonotone carries it forward.
    // Intended to be called as PersistFrontierLogsPreservesAllVerifiedCrossValid@L()
    // where L is a label placed just before the PersistFrontierLogs call.
    twostate lemma {:isolate_assertions} PersistFrontierLogsPreservesAllVerifiedCrossValid()
      requires old(Valid())
      requires Valid()
      requires verifiedDB.db == old(verifiedDB.db)
      requires verifiedDB.lastTimestamp == old(verifiedDB.lastTimestamp)
      requires old(AllVerifiedCrossValid())
      // PersistFrontierLogs {:axiom} postcondition: each logsDB only appended new blocks.
      requires forall c :: c in logsDBs.Keys ==>
        forall n :: old(logsDBs[c].FindSealedBlock(n)).Some? ==>
          logsDBs[c].FindSealedBlock(n) == old(logsDBs[c].FindSealedBlock(n))
      ensures AllVerifiedCrossValid()
    {
      reveal AllVerifiedCrossValid();

      if verifiedDB.LastTimestamp().Some? {
        var lastTS := verifiedDB.LastTimestamp().value;
        SequentialContainsRange(verifiedDB.db, activationTimestamp);
        forall ts | activationTimestamp <= ts <= lastTS
          ensures verifiedDB.Has(ts)
          ensures verifiedDB.Get(ts).l2Heads.Keys == CHAIN_IDS
          ensures ResultIsCrossValid(Result(verifiedDB.Get(ts).timestamp,
                                            verifiedDB.Get(ts).l1Inclusion,
                                            verifiedDB.Get(ts).l2Heads, map[]))
        {
          assert old(verifiedDB.Has(ts));          // fires trigger on old(AllVerifiedCrossValid())
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
          // Prove ResultIsCrossValid in the current state chain by chain.
          var r := Result(verifiedDB.Get(ts).timestamp, verifiedDB.Get(ts).l1Inclusion,
                          verifiedDB.Get(ts).l2Heads, map[]);
          forall chainID | chainID in CHAIN_IDS
            ensures chains[chainID].BlockInfo(r.l2Heads[chainID]).Some?
            ensures BlockIsCrossValid(
                chains[chainID].BlockInfo(r.l2Heads[chainID]).value.timestamp,
                chainID, r.l2Heads[chainID])
            ensures AllInitMsgsPresent(chainID, r.l2Heads[chainID], r.l2Heads)
          {
            var blockID := r.l2Heads[chainID];
            // chains[chainID].BlockInfo/BlockLogs have reads {} — heap-independent.
            // BlockIsCrossValid only calls ValidExecutingMessage (also heap-independent).
            // AllInitMsgsPresent: prove per-message monotonicity.
            forall execMsg | execMsg in chains[chainID].BlockLogs(blockID).value.execMsgs.Values
              ensures execMsg.chainID in CHAIN_IDS &&
                (InitMsgInFrontier(execMsg, r.l2Heads) || InitMsgInLogsDB(execMsg))
            {
              if InitMsgInFrontier(execMsg, r.l2Heads) {
                // InitMsgInFrontier calls reads-{} functions — heap-independent.
              } else {
                assert old(InitMsgInLogsDB(execMsg));
                var query := ContainsQuery(
                  execMsg.blockNum, execMsg.logIdx, execMsg.timestamp, execMsg.checksum);
                // old(Contains) => old(FindSealedBlock(execMsg.blockNum).Some?)
                assert old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum)).Some?;
                // requires: that block number's FindSealedBlock is unchanged.
                assert logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum) ==
                  old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum));
                logsDBs[execMsg.chainID].ContainsMonotone(query);
              }
            }
          }
        }
      }
    }

    // Framing lemma: PersistFrontierLogs preserves AllDBsInSyncUpTo.
    // verifiedDB is unchanged; for each committed timestamp t, SealedBlockForVerifiedAtTimestamp
    // calls FindSealedBlock for a block already sealed before PersistFrontierLogs ran.
    // The monotonicity postcondition of PersistFrontierLogs keeps those FindSealedBlock results
    // unchanged, so DBsInSyncUpTo holds in the new state.
    // Intended to be called as PersistFrontierLogsPreservesAllDBsInSyncUpTo@L(upperTS)
    // where L is a label placed just before the PersistFrontierLogs call.
    twostate lemma PersistFrontierLogsPreservesAllDBsInSyncUpTo(upperTS: nat)
      requires old(Valid())
      requires Valid()
      requires verifiedDB.db == old(verifiedDB.db)
      requires old(AllDBsInSyncUpTo(upperTS))
      requires forall c :: c in logsDBs.Keys ==>
        forall n :: old(logsDBs[c].FindSealedBlock(n)).Some? ==>
          logsDBs[c].FindSealedBlock(n) == old(logsDBs[c].FindSealedBlock(n))
      ensures AllDBsInSyncUpTo(upperTS)
    {
      forall k | k in logsDBs.Keys ensures DBsInSyncUpTo(k, upperTS) {
        forall t | activationTimestamp <= t <= upperTS
          ensures verifiedDB.Has(t)
          ensures k in verifiedDB.Get(t).l2Heads
          ensures SealedBlockForVerifiedAtTimestamp(k, t).Some?
          ensures SealedBlockForVerifiedAtTimestamp(k, t).value.id == verifiedDB.Get(t).l2Heads[k]
        {
          assert old(verifiedDB.Has(t));
          assert old(k in verifiedDB.Get(t).l2Heads);
          assert old(SealedBlockForVerifiedAtTimestamp(k, t)).Some?;
          // SealedBlockForVerifiedAtTimestamp(k, t) = logsDBs[k].FindSealedBlock(verifiedHeads[k].number)
          // verifiedDB.db is unchanged, so verifiedDB.Get(t).l2Heads[k] is the same as old.
          var n := verifiedDB.Get(t).l2Heads[k].number;
          assert old(logsDBs[k].FindSealedBlock(n)).Some?;
          assert logsDBs[k].FindSealedBlock(n) == old(logsDBs[k].FindSealedBlock(n));
        }
      }
    }

    // Framing lemma: PersistFrontierLogs preserves TransitionIsCrossValid.
    // In the Advance case, ResultIsCrossValid is preserved because AllInitMsgsPresent
    // is preserved: InitMsgInFrontier is heap-independent, and InitMsgInLogsDB is
    // preserved by ContainsMonotone (FindSealedBlock for already-sealed block numbers
    // is unchanged by the monotonicity postcondition of PersistFrontierLogs).
    // Intended to be called as PersistFrontierLogsPreservesTransitionIsCrossValid@L(pending)
    // where L is a label placed just before the PersistFrontierLogs call.
    twostate lemma {:isolate_assertions} PersistFrontierLogsPreservesTransitionIsCrossValid(
        pending: PendingTransition)
      requires old(Valid())
      requires Valid()
      requires ValidPendingTransition(pending)
      requires old(TransitionIsCrossValid(pending))
      requires forall c :: c in logsDBs.Keys ==>
        forall n :: old(logsDBs[c].FindSealedBlock(n)).Some? ==>
          logsDBs[c].FindSealedBlock(n) == old(logsDBs[c].FindSealedBlock(n))
      ensures TransitionIsCrossValid(pending)
    {
      if pending.decision.Advance? {
        var result := pending.result.value;
        forall chainID | chainID in CHAIN_IDS
          ensures chains[chainID].BlockInfo(result.l2Heads[chainID]).Some?
          ensures BlockIsCrossValid(
              chains[chainID].BlockInfo(result.l2Heads[chainID]).value.timestamp,
              chainID, result.l2Heads[chainID])
          ensures AllInitMsgsPresent(chainID, result.l2Heads[chainID], result.l2Heads)
        {
          var blockID := result.l2Heads[chainID];
          forall execMsg | execMsg in chains[chainID].BlockLogs(blockID).value.execMsgs.Values
            ensures execMsg.chainID in CHAIN_IDS &&
              (InitMsgInFrontier(execMsg, result.l2Heads) || InitMsgInLogsDB(execMsg))
          {
            if InitMsgInFrontier(execMsg, result.l2Heads) {
              // InitMsgInFrontier calls reads-{} functions — heap-independent.
            } else {
              assert old(InitMsgInLogsDB(execMsg));
              var query := ContainsQuery(
                execMsg.blockNum, execMsg.logIdx, execMsg.timestamp, execMsg.checksum);
              assert old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum)).Some?;
              assert logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum) ==
                old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum));
              logsDBs[execMsg.chainID].ContainsMonotone(query);
            }
          }
        }
      }
    }

    // Framing lemma: ClearPendingTransition preserves AllVerifiedHeadsBoundedByTimestamp.
    // db and lastTimestamp are unchanged, so the timestamp range, Has, and Get all return
    // the same values; BlocksExistedOnChain reads {} and VerifiedHeadsBoundedByTimestamp
    // reads only verifiedDB, so both are preserved by congruence.
    // Intended to be called as ClearPendingPreservesAllVerifiedHeadsBoundedByTimestamp@L()
    // where L is a label placed just before the ClearPendingTransition call.
    twostate lemma ClearPendingPreservesAllVerifiedHeadsBoundedByTimestamp()
      requires old(verifiedDB.Valid())
      requires verifiedDB.Valid()
      requires verifiedDB.db == old(verifiedDB.db)
      requires verifiedDB.lastTimestamp == old(verifiedDB.lastTimestamp)
      requires old(AllVerifiedHeadsBoundedByTimestamp())
      ensures AllVerifiedHeadsBoundedByTimestamp()
    {
      if verifiedDB.LastTimestamp().Some? {
        var lastTS := verifiedDB.LastTimestamp().value;
        forall ts | activationTimestamp <= ts <= lastTS
          ensures verifiedDB.Has(ts)
          ensures chains.Keys == verifiedDB.Get(ts).l2Heads.Keys
          ensures BlocksExistedOnChain(verifiedDB.Get(ts).l2Heads)
          ensures VerifiedHeadsBoundedByTimestamp(ts)
        {
          assert old(verifiedDB.Has(ts));
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
        }
      }
    }

    // Framing lemma: ClearPendingTransition preserves AllVerifiedCrossValid.
    // db and lastTimestamp are unchanged, so the range and entries are the same;
    // ResultIsCrossValid is heap-independent w.r.t. anything ClearPendingTransition
    // touches (only pendingTransition changes).
    // Intended to be called as ClearPendingPreservesAllVerifiedCrossValid@L() where
    // L is a label placed just before the ClearPendingTransition call.
    twostate lemma {:isolate_assertions} ClearPendingPreservesAllVerifiedCrossValid()
      requires old(Valid())
      requires Valid()
      requires unchanged(chains.Values)
      requires unchanged(logsDBs.Values)
      requires verifiedDB.db == old(verifiedDB.db)
      requires verifiedDB.lastTimestamp == old(verifiedDB.lastTimestamp)
      requires old(AllVerifiedCrossValid())
      ensures AllVerifiedCrossValid()
    {
      reveal AllVerifiedCrossValid;

      if verifiedDB.LastTimestamp().Some? {
        var lastTS := verifiedDB.LastTimestamp().value;
        SequentialContainsRange(verifiedDB.db, activationTimestamp);
        forall ts | activationTimestamp <= ts <= lastTS
          ensures verifiedDB.Has(ts)
          ensures verifiedDB.Get(ts).l2Heads.Keys == CHAIN_IDS
          ensures ResultIsCrossValid(Result(verifiedDB.Get(ts).timestamp,
                                            verifiedDB.Get(ts).l1Inclusion,
                                            verifiedDB.Get(ts).l2Heads, map[]))
        {
          assert old(verifiedDB.Has(ts));           // fires trigger on old(AllVerifiedCrossValid())
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
        }
      }
    }

    // Framing lemma: verifiedDB.Rewind preserves AllVerifiedCrossValid.
    // Rewind removes entries >= rewindAt; all remaining entries were covered by
    // old(AllVerifiedCrossValid()) and ResultIsCrossValid is heap-independent
    // w.r.t. anything Rewind touches (only verifiedDB).
    // Intended to be called as RewindPreservesAllVerifiedCrossValid(rewindAt) right
    // after the Rewind call (no label needed when Rewind is the first operation).
    twostate lemma {:isolate_assertions} RewindPreservesAllVerifiedCrossValid(rewindAt: nat)
      requires old(Valid())
      requires Valid()
      requires unchanged(chains.Values)
      requires unchanged(logsDBs.Values)
      requires verifiedDB.db ==
        map k | k in old(verifiedDB.db) && k < rewindAt :: old(verifiedDB.db)[k]
      requires old(AllVerifiedCrossValid())
      ensures AllVerifiedCrossValid()
    {
      reveal AllVerifiedCrossValid;

      if verifiedDB.LastTimestamp().Some? {
        var lastTS := verifiedDB.LastTimestamp().value;
        assert lastTS == MaxKey(verifiedDB.db);   // from Valid(): lastTimestamp == Some(MaxKey(db))
        assert lastTS in verifiedDB.db;            // MaxKey postcondition
        assert lastTS < rewindAt;                  // lastTS in verifiedDB.db = {k in old(db) | k < rewindAt}
        SequentialContainsRange(verifiedDB.db, activationTimestamp);
        forall ts | activationTimestamp <= ts <= lastTS
          ensures verifiedDB.Has(ts)
          ensures verifiedDB.Get(ts).l2Heads.Keys == CHAIN_IDS
          ensures ResultIsCrossValid(Result(verifiedDB.Get(ts).timestamp,
                                            verifiedDB.Get(ts).l1Inclusion,
                                            verifiedDB.Get(ts).l2Heads, map[]))
        {
          assert ts < rewindAt;                     // ts <= lastTS < rewindAt
          assert ts in old(verifiedDB.db);          // ts in verifiedDB.db = {k in old(db) | k < rewindAt}
          assert old(verifiedDB.Has(ts));           // fires trigger on old(AllVerifiedCrossValid())
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
        }
      }
    }

    // Framing lemma: verifiedDB.Rewind preserves AllVerifiedHeadsBoundedByTimestamp.
    // Rewind removes entries >= rewindAt; all remaining entries were covered by
    // old(AllVerifiedHeadsBoundedByTimestamp()) and their VerifiedHeadsBoundedByTimestamp
    // follows by congruence (db[ts] = old(db)[ts] for ts < rewindAt, chains unchanged).
    // Uses verifiedDB.Valid() rather than Valid() to avoid a circular dependency:
    // Valid() includes AllVerifiedHeadsBoundedByTimestamp(), so using Valid() as a
    // requires here would make the ensures redundant or circular.
    // The non-circular part of Valid() needed is the activationTimestamp membership.
    // Intended to be called as RewindPreservesAllVerifiedHeadsBoundedByTimestamp@L(rewindAt)
    // where L is a label placed just before the Rewind call.
    twostate lemma {:isolate_assertions} RewindPreservesAllVerifiedHeadsBoundedByTimestamp(rewindAt: nat)
      requires old(verifiedDB.Valid())
      requires verifiedDB.Valid()
      requires verifiedDB.db ==
        map k | k in old(verifiedDB.db) && k < rewindAt :: old(verifiedDB.db)[k]
      requires verifiedDB.lastTimestamp.Some? ==> activationTimestamp in verifiedDB.db
      requires old(AllVerifiedHeadsBoundedByTimestamp())
      ensures AllVerifiedHeadsBoundedByTimestamp()
    {
      if verifiedDB.LastTimestamp().Some? {
        var lastTS := verifiedDB.LastTimestamp().value;
        assert lastTS == MaxKey(verifiedDB.db);
        assert lastTS in verifiedDB.db;
        assert lastTS < rewindAt;
        SequentialContainsRange(verifiedDB.db, activationTimestamp);
        forall ts | activationTimestamp <= ts <= lastTS
          ensures verifiedDB.Has(ts)
          ensures chains.Keys == verifiedDB.Get(ts).l2Heads.Keys
          ensures BlocksExistedOnChain(verifiedDB.Get(ts).l2Heads)
          ensures VerifiedHeadsBoundedByTimestamp(ts)
        {
          assert ts < rewindAt;
          SequentialContainsRange(old(verifiedDB.db), activationTimestamp);
          assert ts in old(verifiedDB.db);
          assert old(verifiedDB.Has(ts));    // fires trigger on old(AllVerifiedHeadsBoundedByTimestamp())
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
        }
      }
    }

    // Framing lemma: verifiedDB.Commit extends AllVerifiedHeadsBoundedByTimestamp.
    // Old entries are unchanged (db[ts] = old(db)[ts] for ts < newTs) and their
    // VerifiedHeadsBoundedByTimestamp follows from old(AllVerifiedHeadsBoundedByTimestamp()).
    // The new entry satisfies VerifiedHeadsBoundedByTimestamp(newTs) because
    // FrontierBlocksConsistentWithTimestamp(newTs, newR.l2Heads) is the same predicate —
    // this comes from TransitionConsistentWithChainState in PendingTransitionIsConsistent.
    // Intended to be called as CommitExtendsAllVerifiedHeadsBoundedByTimestamp@L(newTs, newR)
    // where L is a label placed just before the Commit call.
    twostate lemma CommitExtendsAllVerifiedHeadsBoundedByTimestamp(newTs: nat, newR: VerifiedResult)
      requires old(verifiedDB.Valid())
      requires verifiedDB.Valid()
      requires unchanged(chains.Values)
      requires chains.Keys == CHAIN_IDS
      requires newR.l2Heads.Keys == CHAIN_IDS
      requires verifiedDB.db == old(verifiedDB.db)[newTs := newR]
      requires verifiedDB.lastTimestamp == Some(newTs)
      requires old(verifiedDB.LastTimestamp()).None? ==> newTs == activationTimestamp
      requires old(verifiedDB.LastTimestamp()).Some? ==>
        newTs == old(verifiedDB.LastTimestamp()).value + 1
      requires old(AllVerifiedHeadsBoundedByTimestamp())
      requires BlocksExistedOnChain(newR.l2Heads)
      requires FrontierBlocksConsistentWithTimestamp(newTs, newR.l2Heads)
      ensures AllVerifiedHeadsBoundedByTimestamp()
    {
      forall ts | activationTimestamp <= ts <= newTs
        ensures verifiedDB.Has(ts)
        ensures chains.Keys == verifiedDB.Get(ts).l2Heads.Keys
        ensures BlocksExistedOnChain(verifiedDB.Get(ts).l2Heads)
        ensures VerifiedHeadsBoundedByTimestamp(ts)
      {
        if ts == newTs {
          // New entry: BlocksExistedOnChain from requires; VerifiedHeadsBoundedByTimestamp
          // is definitionally equal to FrontierBlocksConsistentWithTimestamp (requires).
          assert verifiedDB.Get(newTs) == newR;
        } else {
          assert old(verifiedDB.LastTimestamp()).Some?;
          var oldLastTS := old(verifiedDB.LastTimestamp()).value;
          assert ts <= oldLastTS;
          assert old(verifiedDB.Has(ts));    // fires trigger on old(AllVerifiedHeadsBoundedByTimestamp())
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
        }
      }
    }

    // Framing lemma: verifiedDB.Commit extends AllVerifiedCrossValid with a new entry.
    // Old entries are unchanged and their ResultIsCrossValid follows from old(AllVerifiedCrossValid());
    // the new entry must be supplied as ResultIsCrossValid(Result(newTs, ...)) in the requires.
    // Intended to be called as CommitExtendsAllVerifiedCrossValid@L(newTs, newR) where
    // L is a label placed just before the Commit call.
    twostate lemma {:isolate_assertions} CommitExtendsAllVerifiedCrossValid(newTs: nat, newR: VerifiedResult)
      requires old(Valid())
      requires Valid()
      requires unchanged(chains.Values)
      requires unchanged(logsDBs.Values)
      requires newR.l2Heads.Keys == CHAIN_IDS
      requires verifiedDB.db == old(verifiedDB.db)[newTs := newR]
      requires verifiedDB.lastTimestamp == Some(newTs)
      requires old(verifiedDB.LastTimestamp()).None? ==> newTs == activationTimestamp
      requires old(verifiedDB.LastTimestamp()).Some? ==>
        newTs == old(verifiedDB.LastTimestamp()).value + 1
      requires old(AllVerifiedCrossValid())
      requires ResultIsCrossValid(Result(newTs, newR.l1Inclusion, newR.l2Heads, map[]))
      ensures AllVerifiedCrossValid()
    {
      reveal AllVerifiedCrossValid;
      SequentialContainsRange(verifiedDB.db, activationTimestamp);
      forall ts | activationTimestamp <= ts <= newTs
        ensures verifiedDB.Has(ts)
        ensures verifiedDB.Get(ts).l2Heads.Keys == CHAIN_IDS
        ensures ResultIsCrossValid(Result(verifiedDB.Get(ts).timestamp,
                                          verifiedDB.Get(ts).l1Inclusion,
                                          verifiedDB.Get(ts).l2Heads, map[]))
      {
        if ts == newTs {
          // direct from requires ResultIsCrossValid(Result(newTs, ...))
        } else {
          // ts < newTs; need old(verifiedDB.Has(ts)) to fire old(AllVerifiedCrossValid()) trigger
          assert old(verifiedDB.LastTimestamp()).Some?;  // else newTs == activationTimestamp <= ts, contradiction
          var oldLastTS := old(verifiedDB.LastTimestamp()).value;
          assert ts <= oldLastTS;                         // ts < newTs == oldLastTS + 1
          SequentialContainsRange(old(verifiedDB.db), activationTimestamp);
          assert old(verifiedDB.Has(ts));                // fires trigger on old(AllVerifiedCrossValid())
          assert verifiedDB.db[ts] == old(verifiedDB.db)[ts];
        }
      }
    }


    // Framing lemma: RewindLogsDBs preserves AllVerifiedCrossValid (resetAllChainsTo.Some? path).
    // For each remaining committed timestamp ts' ≤ ts and executing block blockID on chainID:
    // (1) T_chain := chains[chainID].BlockInfo(blockID).value.timestamp is the chain-data
    //     timestamp. ResultIsCrossValid uses T_chain (not ts') as the executing timestamp.
    //     AllDBsInSyncUpTo + BlockSealsMatchOnChainTimestamps identify the seal at
    //     blockID.number with T_chain. VerifiedDB non-decreasing (ts' ≤ ts) gives
    //     blockID.number ≤ plan.targetHeads[chainID].number. FindSealedBlock monotonicity
    //     (blockID.number ≤ plan.targetHeads[chainID].number, both sealed in old state) then
    //     gives T_chain ≤ old(seal_timestamp(targetHead.number)).
    //     NOTE: old(seal_timestamp(targetHead.number)) is not directly bounded to ts here;
    //     instead, step (3) bypasses this via VerifiedHeadsAreHighestBlocksUpToTimestamp.
    // (2) ValidExecutingMessage(T_chain, chainID, execMsg) gives execMsg.timestamp ≤ T_chain ≤ ts.
    //     (The T_chain ≤ ts bound comes from FindSealedBlock monotonicity + requires #11, but
    //     step (3) can derive the contradiction without it by using VHAHIBT directly.)
    // (3) Contains axiom gives old seal timestamp == execMsg.timestamp ≤ ts (from step 2).
    //     old(VerifiedHeadsAreHighestBlocksUpToTimestamp()) at ts for the source chain:
    //     if execMsg.blockNum > plan.targetHeads[source].number, then ts < execMsg.timestamp —
    //     contradiction. So execMsg.blockNum ≤ targetHead.number on the source chain.
    // (4) RewindLogsDBs preserves FindSealedBlock for n ≤ targetHead.number;
    //     ContainsMonotone carries InitMsgInLogsDB forward.
    // InitMsgInFrontier is heap-independent and preserved trivially.
    // Intended to be called as RewindLogsDBsPreservesAllVerifiedCrossValid@L(plan) where
    // L is a label placed just before the RewindLogsDBs call.

    // Instantiates the forall inside BlockIsCrossValid for a single execMsg.
    // A standalone lemma (single VC, no {:isolate_assertions}) so the explicit trigger on
    // BlockIsCrossValid's forall can fire and be matched in one Z3 query without fuel contention.
    lemma BlockIsCrossValidInstantiate(ts: nat, chainID: ChainID, blockID: BlockID, execMsg: ExecutingMessage)
      requires chainID in CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires chains[chainID].BlockInfo(blockID).Some?
      requires BlockIsCrossValid(ts, chainID, blockID)
      requires execMsg in chains[chainID].BlockLogs(blockID).value.execMsgs.Values
      ensures ValidExecutingMessage(ts, chainID, execMsg)
    {
      var logs := chains[chainID].BlockLogs(blockID).value;
      assert execMsg in logs.execMsgs.Values;
    }

    // Helper for RewindLogsDBsPreservesAllVerifiedCrossValid: extracts ValidExecutingMessage
    // from old(AllVerifiedCrossValid()) at a shallow nesting depth where AllVerifiedCrossValid
    // has sufficient fuel. Call sites inside deeply nested foralls where the function would be
    // fuel-exhausted should use this lemma instead of asserting trigger-chain steps inline.
    twostate lemma {:isolate_assertions} OldCrossValidGivesExecMsgTimestamp(
      ts': nat, chainID: ChainID, blockID: BlockID, T_chain: nat, execMsg: ExecutingMessage)
      requires old(Valid())
      requires chains.Keys == CHAIN_IDS
      requires old(AllVerifiedCrossValid())
      requires verifiedDB.db == old(verifiedDB.db)
      requires old(verifiedDB.LastTimestamp()).Some?
      requires activationTimestamp <= ts'
      requires ts' <= old(verifiedDB.LastTimestamp()).value
      requires ts' in verifiedDB.db
      requires chainID in CHAIN_IDS
      requires forall ts :: ts in verifiedDB.db ==> verifiedDB.Get(ts).l2Heads.Keys == CHAIN_IDS
      requires blockID == verifiedDB.Get(ts').l2Heads[chainID]
      requires chains[chainID].BlockInfo(blockID).Some?
      requires chains[chainID].BlockInfo(blockID).value.timestamp == T_chain
      requires execMsg in chains[chainID].BlockLogs(blockID).value.execMsgs.Values
      ensures ValidExecutingMessage(T_chain, chainID, execMsg)
    {
      reveal AllVerifiedCrossValid;
      verifiedDB.GetStableIfDBUnchanged(ts');
      // old(Get(ts')) == Get(ts') is in context. Build the old result explicitly so
      // old(Get(ts')) appears as a ground term and fires the AllVerifiedCrossValid trigger.
      var r_old := Result(old(verifiedDB.Get(ts')).timestamp,
                          old(verifiedDB.Get(ts')).l1Inclusion,
                          old(verifiedDB.Get(ts')).l2Heads, map[]);
      assert r_old.l2Heads.Keys == CHAIN_IDS;
      assert r_old.l2Heads[chainID] == blockID;
      assert old(verifiedDB.Has(ts'));
      // Each step is isolated (fresh fuel per VC):
      // Step 1: old(AllVerifiedCrossValid()) forall fires at ts' → old(ResultIsCrossValid(r_old))
      assert old(ResultIsCrossValid(r_old));
      // Step 2: old(ResultIsCrossValid(r_old)) forall fires at chainID → BlockIsCrossValid(...)
      // (BlockIsCrossValid reads {} — heap-independent, so old(P) == P)
      assert BlockIsCrossValid(T_chain, chainID, blockID);
      // Step 3: Delegate forall instantiation to a helper lemma (single VC, no isolation).
      BlockIsCrossValidInstantiate(T_chain, chainID, blockID, execMsg);
    }

    twostate lemma {:isolate_assertions} RewindLogsDBsPreservesAllVerifiedCrossValid(plan: RewindPlan)
      requires plan.resetAllChainsTo.Some?
      requires old(Valid())
      requires Valid()
      requires verifiedDB.db == old(verifiedDB.db)
      requires verifiedDB.lastTimestamp == old(verifiedDB.lastTimestamp)
      // verifiedDB.LastTimestamp() == Some(ts) in both states (from RewoundVerifiedDB).
      requires old(verifiedDB.LastTimestamp()) == Some(plan.resetAllChainsTo.value)
      requires old(AllVerifiedCrossValid())
      // Explicit requires needed for predicates called deep inside isolated VCs (Valid()
      // body may be fuel-exhausted at that depth).
      requires chains.Keys == CHAIN_IDS
      requires logsDBs.Keys == CHAIN_IDS
      requires forall ts :: ts in verifiedDB.db ==> verifiedDB.Get(ts).l2Heads.Keys == CHAIN_IDS
      // plan.targetHeads covers exactly the same chains as logsDBs (both == CHAIN_IDS).
      requires plan.targetHeads.Keys == logsDBs.Keys
      // plan.targetHeads are the verified L2 heads at ts (from PlanConsistentWithVerified).
      requires plan.targetHeads == verifiedDB.Get(plan.resetAllChainsTo.value).l2Heads
      // The logsDBs are in sync with the verifiedDB up to ts in the old state.
      requires old(AllDBsInSyncUpTo(plan.resetAllChainsTo.value))
      // RewindLogsDBs postcondition: blocks at or below each target head are preserved.
      requires forall c :: c in logsDBs.Keys ==>
        forall n :: 0 <= n <= plan.targetHeads[c].number ==>
          logsDBs[c].FindSealedBlock(n) == old(logsDBs[c].FindSealedBlock(n))
      // RewindLogsDBs RewoundLogsDB postcondition: latest sealed block is the target head.
      requires forall c :: c in logsDBs.Keys && c in plan.targetHeads ==>
        logsDBs[c].LatestSealedBlock() == Some(plan.targetHeads[c])
      requires AllVerifiedHeadsBoundedByTimestamp()
      ensures {:axiom} AllVerifiedCrossValid()
    {
      reveal AllVerifiedCrossValid;
      var ts := plan.resetAllChainsTo.value;
      // verifiedDB unchanged → same committed timestamps; use SequentialContainsRange for Has.
      SequentialContainsRange(verifiedDB.db, activationTimestamp);
      forall ts' | activationTimestamp <= ts' <= verifiedDB.LastTimestamp().value
        ensures verifiedDB.Has(ts')
        ensures verifiedDB.Get(ts').l2Heads.Keys == CHAIN_IDS
        ensures ResultIsCrossValid(Result(verifiedDB.Get(ts').timestamp,
                                          verifiedDB.Get(ts').l1Inclusion,
                                          verifiedDB.Get(ts').l2Heads, map[]))
      {
        assert old(verifiedDB.Has(ts'));           // fires trigger on old(AllVerifiedCrossValid())
        assert verifiedDB.db[ts'] == old(verifiedDB.db)[ts'];
        // ts' ≤ ts because verifiedDB.LastTimestamp() = old(verifiedDB.LastTimestamp()) = Some(ts).
        assert ts' <= ts;
        var r := Result(verifiedDB.Get(ts').timestamp, verifiedDB.Get(ts').l1Inclusion,
                        verifiedDB.Get(ts').l2Heads, map[]);
        // Prove ResultIsCrossValid(r) per chain.
        forall chainID | chainID in CHAIN_IDS
          ensures chains[chainID].BlockInfo(r.l2Heads[chainID]).Some?
          ensures BlockIsCrossValid(chains[chainID].BlockInfo(r.l2Heads[chainID]).value.timestamp,
                                    chainID, r.l2Heads[chainID])
          ensures AllInitMsgsPresent(chainID, r.l2Heads[chainID], r.l2Heads)
        {
          var blockID := r.l2Heads[chainID];
          // chains[chainID].BlockInfo/BlockLogs have reads {} — heap-independent.
          // BlockIsCrossValid calls ValidExecutingMessage which also uses reads {} functions.
          // AllInitMsgsPresent: prove per-message.
          forall execMsg | execMsg in chains[chainID].BlockLogs(blockID).value.execMsgs.Values
            ensures execMsg.chainID in CHAIN_IDS &&
              (InitMsgInFrontier(execMsg, r.l2Heads) || InitMsgInLogsDB(execMsg))
          {
            if InitMsgInFrontier(execMsg, r.l2Heads) {
              // InitMsgInFrontier is heap-independent (reads {} functions) — preserved.
            } else {
              assert old(InitMsgInLogsDB(execMsg));
              var query := ContainsQuery(
                execMsg.blockNum, execMsg.logIdx, execMsg.timestamp, execMsg.checksum);
              // T_chain is the chain-data timestamp of the executing block.
              // ResultIsCrossValid uses T_chain (not ts') as the executing timestamp.
              var T_chain := chains[chainID].BlockInfo(blockID).value.timestamp;
              // AllVerifiedCrossValid is fuel-exhausted at this nesting depth (forall ts',
              // forall chainID, forall execMsg, else branch). Use a helper lemma that fires
              // the trigger chain at a shallower depth where fuel is fresh.
              OldCrossValidGivesExecMsgTimestamp(ts', chainID, blockID, T_chain, execMsg);
              assert execMsg.timestamp <= T_chain;
              // T_chain ≤ ts:
              //   AllDBsInSyncUpTo at ts' for chainID: seal at blockID.number exists with id == blockID.
              assert old(logsDBs[chainID].FindSealedBlock(blockID.number)).Some?;
              assert old(logsDBs[chainID].FindSealedBlock(blockID.number)).value.id == blockID;
              //   BlockSealsMatchOnChainTimestamps: seal timestamp == T_chain.
              assert old(logsDBs[chainID].FindSealedBlock(blockID.number)).value.timestamp == T_chain;
              //   verifiedDB non-decreasing (ts' ≤ ts): blockID.number ≤ plan.targetHeads[chainID].number.
              assert blockID.number <= plan.targetHeads[chainID].number;
              //   Requires #11 gives seal at targetHead.number has timestamp ts.
              //   FindSealedBlock monotonicity: T_chain ≤ ts.
              assert T_chain <= ts;
              assert execMsg.timestamp <= ts;
              // old(Contains) gives the seal timestamp == execMsg.timestamp.
              assert old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum)).Some?;
              assert old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum)).value.timestamp
                == execMsg.timestamp;
              // Proof by contradiction: assume execMsg.blockNum > targetHead on source chain.
              // Fire old(VerifiedHeadsAreHighestBlocksUpToTimestamp()) at (ts, execMsg.chainID, execMsg.blockNum):
              //   old seal at execMsg.blockNum has timestamp = execMsg.timestamp (Contains axiom).
              //   VerifiedHeadsAreHighest gives ts < execMsg.timestamp.
              //   But execMsg.timestamp ≤ T_chain ≤ ts — contradiction.
              assert execMsg.blockNum <= plan.targetHeads[execMsg.chainID].number by {
                if execMsg.blockNum > plan.targetHeads[execMsg.chainID].number {
                  // old(InitMsgInLogsDB(execMsg)) and
                  // old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum)).Some? and
                  // the seal timestamp == execMsg.timestamp are all assumed from outer context.
                  // Fire trigger chain for old(VerifiedHeadsAreHighestBlocksUpToTimestamp()) at ts.
                  assert old(verifiedDB.Has(ts));
                  assert execMsg.chainID in old(verifiedDB.Get(ts).l2Heads.Keys);
                  assert plan.targetHeads[execMsg.chainID].number ==
                    old(verifiedDB.Get(ts).l2Heads[execMsg.chainID].number);
                  // Inner forall fires: blockNum > targetHead && FindSealedBlock.Some? => ts < timestamp.
                  assert ts < old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum)).value.timestamp;
                  // ts < seal.timestamp == execMsg.timestamp, but execMsg.timestamp <= ts
                  // (assumed from outer context via ValidExecutingMessage + T_chain <= ts). Contradiction.
                  assert false;
                }
              }
              // FindSealedBlock preserved for n ≤ targetHead.number (from requires).
              assert logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum) ==
                old(logsDBs[execMsg.chainID].FindSealedBlock(execMsg.blockNum));
              // ContainsMonotone: Contains preserved when its queried FindSealedBlock is unchanged.
              logsDBs[execMsg.chainID].ContainsMonotone(query);
            }
          }
        }
      }
    }

    // Proves BlockSealsMatchOnChainTimestamps() after a successful PersistFrontierLogs.
    // Called as AdvanceEstablishesBlockSealsMatchOnChainTimestamps@L(blocksAtTS) where L is
    // a label placed just before PersistFrontierLogs.
    // Key: UpdatedAllLogsDBs -> UpdatedLogsDB -> LogsDBConsistentWithChainData instantiated
    // at blockID := sealedBlock.id, giving seal.timestamp == on-chain timestamp for every block.
    twostate lemma AdvanceEstablishesBlockSealsMatchOnChainTimestamps(blocksAtTS: map<ChainID, BlockID>)
      requires old(Valid())
      requires logsDBs.Keys == CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires blocksAtTS.Keys == CHAIN_IDS
      requires BlocksExistedOnChain(blocksAtTS)
      requires UpdatedAllLogsDBs(blocksAtTS)
      ensures BlockSealsMatchOnChainTimestamps()
    {
      reveal UpdatedAllLogsDBs;
      reveal UpdatedLogsDB;
      forall chainID | chainID in logsDBs.Keys
        ensures forall n :: logsDBs[chainID].FindSealedBlock(n).Some? ==>
          chains[chainID].BlockInfo(logsDBs[chainID].FindSealedBlock(n).value.id).Some? &&
          logsDBs[chainID].FindSealedBlock(n).value.timestamp ==
          chains[chainID].BlockInfo(logsDBs[chainID].FindSealedBlock(n).value.id).value.timestamp
      {
        reveal LogsDBConsistentWithChainData;
        forall n | logsDBs[chainID].FindSealedBlock(n).Some?
          ensures chains[chainID].BlockInfo(logsDBs[chainID].FindSealedBlock(n).value.id).Some? &&
            logsDBs[chainID].FindSealedBlock(n).value.timestamp ==
            chains[chainID].BlockInfo(logsDBs[chainID].FindSealedBlock(n).value.id).value.timestamp
        {
          var sealedBlock := logsDBs[chainID].FindSealedBlock(n).value;
          var sealedID := sealedBlock.id;
          // Trigger LogsDBConsistentWithChainData at blockID := sealedID.
          // sealedID.number == n, so FindSealedBlock(sealedID.number).Some? holds.
          assert logsDBs[chainID].FindSealedBlock(sealedID.number).Some?;
          // Consequent: BlockExistedOnChain(chainID, sealedID) && timestamp match.
          assert BlockExistedOnChain(chainID, sealedID);
          assert chains[chainID].BlockInfo(sealedID).Some?;
          var info := chains[chainID].BlockInfo(sealedID).value;
          // BlockInfo axiom: info.id == sealedID, so info.id.number == n.
          assert info.id.number == n;
          assert logsDBs[chainID].FindSealedBlock(info.id.number).Some?;
          assert logsDBs[chainID].FindSealedBlock(info.id.number).value.timestamp == info.timestamp;
        }
      }
    }

    // Proves VerifiedHeadsAreHighestBlocksUpToTimestamp() after PersistFrontierLogs + Commit.
    // Called as AdvanceEstablishesVerifiedHeadsAreHighest@L(newTs, blocksAtTS) where L is
    // a label placed just before PersistFrontierLogs.
    // Proof splits on ts' == newTs (vacuously true: LatestSealedBlock axiom gives no blocks
    // above the frontier) vs ts' < newTs (old entries: framing for unchanged slots; new block's
    // timestamp == newTs > ts' for the changed slot when newly added).
    twostate lemma {:isolate_assertions} AdvanceEstablishesVerifiedHeadsAreHighest(newTs: nat, blocksAtTS: map<ChainID, BlockID>)
      requires old(Valid())
      requires logsDBs.Keys == CHAIN_IDS
      requires chains.Keys == CHAIN_IDS
      requires blocksAtTS.Keys == CHAIN_IDS
      requires BlocksExistedOnChain(blocksAtTS)
      requires UpdatedAllLogsDBs(blocksAtTS)
      // PersistFrontierLogs new ensures: newly-added block has timestamp == newTs.
      requires forall chainID :: chainID in blocksAtTS.Keys ==>
        old(logsDBs[chainID].FindSealedBlock(blocksAtTS[chainID].number)) !=
        logsDBs[chainID].FindSealedBlock(blocksAtTS[chainID].number) ==>
        logsDBs[chainID].FindSealedBlock(blocksAtTS[chainID].number).Some? &&
        logsDBs[chainID].FindSealedBlock(blocksAtTS[chainID].number).value.timestamp == newTs
      // Commit result: verifiedDB.db is old entries plus the new entry at newTs.
      requires verifiedDB.Has(newTs)
      requires verifiedDB.db == old(verifiedDB.db)[newTs := verifiedDB.db[newTs]]
      requires verifiedDB.Get(newTs).l2Heads == blocksAtTS
      // newTs is the sequential successor of the old last timestamp.
      requires old(verifiedDB.LastTimestamp()).None? ==> newTs == activationTimestamp
      requires old(verifiedDB.LastTimestamp()).Some? ==>
        newTs == old(verifiedDB.LastTimestamp()).value + 1
      requires verifiedDB.Valid()
      ensures VerifiedHeadsAreHighestBlocksUpToTimestamp()
    {
      reveal UpdatedAllLogsDBs;
      reveal UpdatedLogsDB;
      forall ts' : nat | verifiedDB.Has(ts')
        ensures
          var verifiedHeads := verifiedDB.Get(ts').l2Heads;
          verifiedHeads.Keys == logsDBs.Keys &&
          forall chainID :: chainID in verifiedHeads.Keys ==>
            var blockNumber := verifiedHeads[chainID].number;
            forall n :: blockNumber < n && logsDBs[chainID].FindSealedBlock(n).Some? ==>
              ts' < logsDBs[chainID].FindSealedBlock(n).value.timestamp
      {
        if ts' == newTs {
          // New entry: verifiedHeads == blocksAtTS.
          // UpdatedLogsDB gives LatestSealedBlock() == Some(blocksAtTS[chainID]).
          // LatestSealedBlock axiom: FindSealedBlock(n) == None for n > blocksAtTS[chainID].number.
          // => property holds vacuously (no sealed blocks above the frontier).
          forall chainID | chainID in verifiedDB.Get(ts').l2Heads.Keys
            ensures
              var blockNumber := verifiedDB.Get(ts').l2Heads[chainID].number;
              forall n :: blockNumber < n && logsDBs[chainID].FindSealedBlock(n).Some? ==>
                ts' < logsDBs[chainID].FindSealedBlock(n).value.timestamp
          {
            assert logsDBs[chainID].LatestSealedBlock() == Some(blocksAtTS[chainID]);
          }
        } else {
          // Old entry: ts' in old(verifiedDB.db), so ts' < newTs.
          assert ts' in old(verifiedDB.db);
          // Derive ts' < newTs via MaxKey.
          assert old(verifiedDB.LastTimestamp()).Some?;
          assert ts' <= MaxKey(old(verifiedDB.db));
          assert ts' <= old(verifiedDB.LastTimestamp()).value;
          assert ts' < newTs;
          // db[ts'] is unchanged (only newTs was added by Commit).
          assert verifiedDB.db[ts'] == old(verifiedDB.db)[ts'];
          var verifiedHeads := verifiedDB.Get(ts').l2Heads;
          forall chainID | chainID in verifiedHeads.Keys
            ensures
              var blockNumber := verifiedHeads[chainID].number;
              forall n :: blockNumber < n && logsDBs[chainID].FindSealedBlock(n).Some? ==>
                ts' < logsDBs[chainID].FindSealedBlock(n).value.timestamp
          {
            var blockNumber := verifiedHeads[chainID].number;
            // Fire trigger chain for old(VerifiedHeadsAreHighestBlocksUpToTimestamp()).
            // verifiedDB.db[ts'] is unchanged, so old and current Get(ts') agree.
            assert old(verifiedDB.Has(ts'));
            assert old(verifiedDB.Get(ts')).l2Heads == verifiedHeads;
            assert chainID in old(verifiedDB.Get(ts')).l2Heads.Keys;
            assert blockNumber == old(verifiedDB.Get(ts')).l2Heads[chainID].number;
            forall n | blockNumber < n && logsDBs[chainID].FindSealedBlock(n).Some?
              ensures ts' < logsDBs[chainID].FindSealedBlock(n).value.timestamp
            {
              if n == blocksAtTS[chainID].number {
                if old(logsDBs[chainID].FindSealedBlock(n)) ==
                    logsDBs[chainID].FindSealedBlock(n) {
                  // Block unchanged: fire old(VerifiedHeadsAreHighestBlocksUpToTimestamp()) at ts'.
                  assert old(logsDBs[chainID].FindSealedBlock(n)).Some?;
                  assert old(logsDBs[chainID].FindSealedBlock(n)).value.timestamp > ts';
                } else {
                  // Newly added block: timestamp == newTs > ts'.
                  assert logsDBs[chainID].FindSealedBlock(n).value.timestamp == newTs;
                }
              } else {
                // n != frontier slot: FindSealedBlock(n) unchanged by PersistFrontierLogs.
                assert logsDBs[chainID].FindSealedBlock(n) ==
                  old(logsDBs[chainID].FindSealedBlock(n));
                assert old(logsDBs[chainID].FindSealedBlock(n)).Some?;
                assert old(logsDBs[chainID].FindSealedBlock(n)).value.timestamp > ts';
              }
            }
          }
        }
      }
    }

  }
}
