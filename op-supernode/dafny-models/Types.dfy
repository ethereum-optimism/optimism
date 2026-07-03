// Shared type definitions for the Dafny interop model.

module Types {

  // Simplified representation of eth.ChainID
  type ChainID = nat

  // Simplified representation of eth.BlockID
  datatype BlockID = BlockID(number: nat, hash: nat)

  // Simplified representation of a block's metadata as returned by FetchReceipts.
  // Corresponds to eth.BlockInfo in Go; receipts are abstracted away.
  datatype BlockInfo = BlockInfo(id: BlockID, parentHash: nat, timestamp: nat)

  // Corresponds to interop.VerifiedResult
  datatype VerifiedResult = VerifiedResult(
    timestamp: nat,
    l1Inclusion: BlockID,
    l2Heads: map<ChainID, BlockID>
  )

  // Corresponds to interop.Decision
  datatype Decision = Wait | Advance | Invalidate | Rewind

  // Replaces Go nullable pointers and (value, bool) pairs.
  datatype Option<T(==)> = None | Some(value: T)

  // Corresponds to interop.Result
  datatype Result = Result(
    timestamp: nat,
    l1Inclusion: BlockID,
    l2Heads: map<ChainID, BlockID>,
    invalidHeads: map<ChainID, BlockID>
  )

  // Corresponds to interop.RewindPlan
  datatype RewindPlan = RewindPlan(
    rewindAtOrAfter: nat,
    resetAllChainsTo: Option<nat>,
    targetHeads: map<ChainID, BlockID>
  )

  // Corresponds to interop.PendingTransition.
  // The WAL operational details are abstracted away; modeled as a plain
  // optional value held in a field.
  datatype PendingTransition = PendingTransition(
    decision: Decision,
    result: Option<Result>,
    rewind: Option<RewindPlan>
  )

  // Sealed block entry stored in a LogsDB.
  // Corresponds to the block seal used by LogsDB.FindSealedBlock.
  datatype BlockSeal = BlockSeal(id: BlockID, timestamp: nat)

  // Consistent snapshot of the current round's state, captured upfront so that
  // the decision function operates on immutable data.
  // Corresponds to interop.RoundObservation; the test-only Paused field is omitted.
  datatype RoundObservation = RoundObservation(
    lastVerifiedTS: Option<nat>,
    lastVerified: Option<VerifiedResult>,
    nextTimestamp: nat,
    chainsReady: bool,
    blocksAtTS: map<ChainID, BlockID>,
    l1Heads: map<ChainID, BlockID>,
    l1Consistent: bool,
    l2sConsistent: bool
  )

  // Parallel query results from CheckChainsReady.
  // Corresponds to chainsReadyResult in interop.go.
  datatype ChainsReadyResult = ChainsReadyResult(
    blocks: map<ChainID, BlockID>,
    l1Heads: map<ChainID, BlockID>
  )

  // Outcome of a round.
  // Corresponds to interop.StepOutput, modeled as an algebraic type so that
  // Advance and Invalidate carry the verification result as a constructor
  // argument, avoiding an Option<Result> field and associated preconditions.
  datatype StepOutput =
    | WaitOutput
    | AdvanceOutput(result: Result)
    | InvalidateOutput(result: Result)
    | RewindOutput

  // Corresponds to types.ExecutingMessage in supervisor/types/types.go.
  // Models an executing message found in an L2 block's receipts.
  datatype ExecutingMessage = ExecutingMessage(
    chainID: ChainID,  // source chain the initiating message was emitted on
    blockNum: nat,     // block number of the initiating message
    logIdx: nat,       // log index within the initiating block
    timestamp: nat,    // timestamp of the initiating block
    checksum: nat      // simplified from MessageChecksum (hash of payload + identifier)
  )

  // Corresponds to types.ContainsQuery in supervisor/types/types.go.
  // Derived from an ExecutingMessage to query for the initiating message in a logsDB.
  datatype ContainsQuery = ContainsQuery(
    blockNum: nat,
    logIdx: nat,
    timestamp: nat,
    checksum: nat
  )

  datatype Log = Log(data: nat, checksum: nat)

  // The log data for a block: full list plus executing messages keyed by log index.
  // Corresponds to types.Receipts returned by ChainContainer.FetchReceipts.
  datatype BlockLogs = BlockLogs(fullLogs: seq<Log>, execMsgs: map<nat, ExecutingMessage>)

  // ----- System configuration constants -----------------------------------

  // Ghost constants representing the single model instance's configuration.
  // Named in SCREAMING_SNAKE_CASE to avoid shadowing by class fields.
  // Linked to the Interop class fields by the Valid() predicate in Interop.dfy:
  //   activationTimestamp == ACTIVATION_TIMESTAMP
  //   chains.Keys == CHAIN_IDS
  //   messageExpiryWindow == MESSAGE_EXPIRY_WINDOW
  const ACTIVATION_TIMESTAMP: nat
  const CHAIN_IDS: set<ChainID>
  // Maximum age (in seconds) of an initiating message that can be referenced by
  // an executing message. Corresponds to defaultMessageExpiryWindow in algo.go.
  const MESSAGE_EXPIRY_WINDOW: nat

  // ----- Structural validity predicates -----------------------------------

  // Structural validity of the rewind plan, independent of the verifiedDB state.
  // None case: rewindAtOrAfter is at or below the activation timestamp, so
  // every DB entry (all >= activationTimestamp) will be removed.
  // Some case: the target timestamp precedes the rewind point and the target
  // heads cover exactly the current set of chains.
  ghost predicate ValidRewindPlan(plan: RewindPlan)
  {
    match plan.resetAllChainsTo {
      case None =>
        plan.rewindAtOrAfter <= ACTIVATION_TIMESTAMP &&
        |plan.targetHeads| == 0
      case Some(ts) =>
        ACTIVATION_TIMESTAMP < plan.rewindAtOrAfter &&
        ts == plan.rewindAtOrAfter - 1 &&
        plan.targetHeads.Keys == CHAIN_IDS
    }
  }

  // Invariant on a stored pending transition: the decision is not Wait, the
  // optional fields are present when expected, and the result l2Heads cover
  // exactly the current set of chains.
  ghost predicate ValidPendingTransition(pending: PendingTransition)
  {
    pending.decision != Wait &&
    (pending.decision == Rewind <==> pending.rewind.Some?) &&
    (pending.decision == Rewind ==> ValidRewindPlan(pending.rewind.value)) &&
    (pending.decision == Rewind ==> pending.result.None?) &&
    (pending.decision == Advance ==> pending.result.Some?) &&
    (pending.decision == Advance ==> |pending.result.value.invalidHeads| == 0) &&
    (pending.decision == Invalidate ==> pending.result.Some?) &&
    (pending.decision == Invalidate ==> 0 < |pending.result.value.invalidHeads|) &&
    (pending.result.Some? ==> pending.result.value.l2Heads.Keys == CHAIN_IDS)
  }

  // Validity of a step output: the decision is not Wait, the observation
  // provides a last-verified timestamp when rewinding, and the l2Heads cover
  // exactly the current set of chains when advancing or invalidating.
  // Does not depend on either DB.
  ghost predicate ValidStepOutput(output: StepOutput, obs: RoundObservation)
  {
    match output {
      case WaitOutput => true
      case RewindOutput => obs.lastVerifiedTS.Some?
      case AdvanceOutput(result) =>
        result.timestamp == obs.nextTimestamp &&
        result.l2Heads == obs.blocksAtTS &&
        result.l2Heads.Keys == CHAIN_IDS &&
        |result.invalidHeads| == 0
      case InvalidateOutput(result) =>
        result.timestamp == obs.nextTimestamp &&
        result.l2Heads == obs.blocksAtTS &&
        result.l2Heads.Keys == CHAIN_IDS &&
        0 < |result.invalidHeads|
    }
  }

  // Structural validity of the round observation, independent of either DB:
  // the last verified timestamp is present when l1 is inconsistent (i.e., when
  // a rewind output will be produced), and the observed block map covers exactly
  // the current chain set when chains are ready.
  ghost predicate ValidRoundObservation(obs: RoundObservation)
  {
    (!obs.l1Consistent ==> obs.lastVerifiedTS.Some?) &&
    (obs.chainsReady ==> obs.blocksAtTS.Keys == CHAIN_IDS) &&
    (0 < |obs.blocksAtTS| ==> obs.chainsReady)
  }

  // Converts a set of ChainIDs to a sequence containing exactly those elements.
  // Order is non-deterministic; callers that iterate should not depend on it.
  method Enumerate(s: set<ChainID>) returns (result: seq<ChainID>)
    ensures forall x :: x in result <==> x in s
    ensures |result| == |s|
    ensures forall p, q :: 0 <= p < q < |result| ==> result[p] != result[q]
  {
    result := [];
    var remaining := s;
    while remaining != {}
      invariant remaining <= s
      invariant forall x :: x in result <==> (x in s && x !in remaining)
      invariant |result| + |remaining| == |s|
      invariant forall p, q :: 0 <= p < q < |result| ==> result[p] != result[q]
      decreases |remaining|
    {
      var x :| x in remaining;
      result := result + [x];
      remaining := remaining - {x};
    }
  }

}
