// Off-chain half of the message-expiry safety argument, proved against the existing
// op-supernode Dafny model (op-supernode/dafny-models), using that model's own definitions:
//   - Interop.ValidExecutingMessage   (the declarative validity predicate)
//   - Interop.VerifyExecutingMessage  (the imperative check; returns false on ErrMessageExpired)
//   - Interop.messageExpiryWindow / Types.MESSAGE_EXPIRY_WINDOW (the window W)
// The theorems' subjects (ValidExecutingMessage, VerifyExecutingMessage's result) are the model's
// own, referenced, not restated. Two auxiliary statements hand-copy guard expressions from
// VerifyExecutingMessage's body and are labelled RESTATEMENT below. This file adds no axioms,
// no {:axiom}, no assume, and no bodiless lemmas.
//
// The on-chain half (L2ToL2CrossDomainMessenger.expireMessage accepts a fact only if
// undeliveredAt > sentAt + EXPIRY_PERIOD) is proved elsewhere (Lean / Kontrol / Halmos). This file
// proves: once the destination's block timestamp at export, tExport, is past
// initTimestamp + P with W <= P, no executing timestamp exec >= tExport makes the message valid.

include "../../../../../../op-supernode/dafny-models/Interop.dfy"

module ExpiryBridge {
  import opened Types
  import I = Interop
  import CC = ChainContainer

  // L2ToL2CrossDomainMessenger.EXPIRY_PERIOD = 8 days (the contract's expiry period P).
  const EXPIRY_PERIOD: nat := 691200
  // The cap that op-core (static_depset.go hydrate) and kona (depset.rs
  // deserialize_override_window) put on the protocol window: MessageExpiryTimeSecondsInterop /
  // MESSAGE_EXPIRY_WINDOW = 7 days. Types.MESSAGE_EXPIRY_WINDOW itself is abstract in the model.
  const PROTOCOL_WINDOW_CAP: nat := 604800

  // ---------------------------------------------------------------------------
  // 1. Main theorem, declarative form, over the instance's own window field.
  //    Only the two preconditions of ValidExecutingMessage itself are required; no Valid().
  // ---------------------------------------------------------------------------
  lemma ExpiredAtExportNeverValid(
      i: I.Interop, P: nat, tExport: nat, exec: nat, execChain: ChainID, msg: ExecutingMessage)
    requires execChain in CHAIN_IDS
    requires i.chains.Keys == CHAIN_IDS
    requires i.messageExpiryWindow <= P
    requires tExport > msg.timestamp + P
    requires exec >= tExport
    ensures !i.ValidExecutingMessage(exec, execChain, msg)
    // RESTATEMENT (hand-copied, not referenced): this is the text of the guard of
    // VerifyExecutingMessage's ErrMessageExpired branch (Interop.dfy:1748). Dafny cannot refer to
    // a method's internal guard, so this conjunct is only as faithful as the copy.
    ensures msg.timestamp + i.messageExpiryWindow < exec
  {
  }

  // ---------------------------------------------------------------------------
  // 2. Same theorem stated over the global constant, for any instance satisfying the model's
  //    class invariant Valid() (which pins messageExpiryWindow == MESSAGE_EXPIRY_WINDOW).
  // ---------------------------------------------------------------------------
  lemma ExpiredAtExportNeverValidGlobal(
      i: I.Interop, P: nat, tExport: nat, exec: nat, execChain: ChainID, msg: ExecutingMessage)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires MESSAGE_EXPIRY_WINDOW <= P
    requires tExport > msg.timestamp + P
    requires exec >= tExport
    ensures !i.ValidExecutingMessage(exec, execChain, msg)
    ensures msg.timestamp + MESSAGE_EXPIRY_WINDOW < exec
  {
    ExpiredAtExportNeverValid(i, P, tExport, exec, execChain, msg);
  }

  // ---------------------------------------------------------------------------
  // 3. Imperative form: the model's VerifyExecutingMessage returns false. The conclusion comes
  //    from that method's own (verified) postcondition `valid ==> ValidExecutingMessage(...)`.
  //    The model's result type has no error codes; that the failing branch is ErrMessageExpired
  //    (and not an earlier one) is shown by ExpiryIsTheFailingGuard below.
  // ---------------------------------------------------------------------------
  method ExpiredAtExportRejected(
      i: I.Interop, view: I.FrontierView, P: nat, tExport: nat, exec: nat,
      execChain: ChainID, msg: ExecutingMessage)
    returns (ok: bool)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires MESSAGE_EXPIRY_WINDOW <= P
    requires tExport > msg.timestamp + P
    requires exec >= tExport
    ensures !ok
  {
    ok := i.VerifyExecutingMessage(execChain, exec, msg, view);
    ExpiredAtExportNeverValidGlobal(i, P, tExport, exec, execChain, msg);
  }

  // The contract's period instantiated: P = EXPIRY_PERIOD (8 days) and the 7-day cap on W.
  method ExpiredAtExportRejectedWithCap(
      i: I.Interop, view: I.FrontierView, tExport: nat, exec: nat,
      execChain: ChainID, msg: ExecutingMessage)
    returns (ok: bool)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires MESSAGE_EXPIRY_WINDOW <= PROTOCOL_WINDOW_CAP
    requires tExport > msg.timestamp + EXPIRY_PERIOD
    requires exec >= tExport
    ensures !ok
    ensures !i.ValidExecutingMessage(exec, execChain, msg)
  {
    ok := ExpiredAtExportRejected(i, view, EXPIRY_PERIOD, tExport, exec, execChain, msg);
    ExpiredAtExportNeverValidGlobal(i, EXPIRY_PERIOD, tExport, exec, execChain, msg);
  }

  // Which branch fails (RESTATEMENT). When the source chain is registered, both activation checks
  // pass and init <= exec, the ensures below say every guard before the expiry check is false and
  // the expiry guard is true. The guard expressions are HAND-COPIED from VerifyExecutingMessage's
  // body (Interop.dfy:1722, 1729, 1736, 1742, 1748); Dafny cannot reference a method's internal
  // guards and the model returns a bool with no error code, so the link to "returns from the
  // ErrMessageExpired branch" rests on the copy being faithful (checked by reading). The verified,
  // non-restated result is ExpiredAtExportRejected: the method returns false.
  lemma ExpiryIsTheFailingGuard(
      i: I.Interop, P: nat, tExport: nat, exec: nat, execChain: ChainID, msg: ExecutingMessage)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires msg.chainID in CHAIN_IDS
    requires i.activationTimestamp + i.chains[execChain].BlockTime() <= exec
    requires i.activationTimestamp + i.chains[msg.chainID].BlockTime() <= msg.timestamp
    requires MESSAGE_EXPIRY_WINDOW <= P
    requires tExport > msg.timestamp + P
    requires exec >= tExport
    ensures msg.chainID in i.logsDBs && msg.chainID in i.chains              // not ErrUnknownChain
    ensures !(exec < i.activationTimestamp + i.chains[execChain].BlockTime())        // not ErrExecutedTooEarly
    ensures !(msg.timestamp < i.activationTimestamp + i.chains[msg.chainID].BlockTime()) // not ErrInitiatedTooEarly
    ensures !(msg.timestamp > exec)                                           // not ErrTimestampViolation
    ensures msg.timestamp + i.messageExpiryWindow < exec                      // ErrMessageExpired fires
  {
  }

  // Timestamp binding: the expiry rule is only meaningful if an executing message cannot claim a
  // later initiating timestamp than the real one (else it could stretch its window). When the
  // model's VerifyExecutingMessage accepts, msg.timestamp equals the timestamp the model's chain
  // data (ChainContainer.BlockInfo) gives the initiating block:
  //  - logsDB path (msg.timestamp < exec): via the LogsDB.Contains {:axiom} ensures (timestamp
  //    equality; mirrors raftwallogdb/db.go Contains `rec.Timestamp != query.Timestamp` ->
  //    ErrConflict), LogsDB.FindSealedBlock's {:axiom} (id.number == number), and the model's
  //    proved invariant AllLogsDBsConsistentWithChainData (part of Valid()).
  //  - frontier path (msg.timestamp == exec): via FrontierView.Contains's body and the HYPOTHESIS
  //    IsCorrectFrontierView(view, blocksAtTS). In the model that property is supplied by
  //    ResolveFrontierVerificationView's {:axiom} ensures (Interop.dfy:1889).
  // Outside the model: ExecutingMessage.checksum is an abstract nat, so the step "this executing
  // message references H's SentMessage log, hence its timestamp is the contract's sentAt" is an
  // argument about Go/kona's checksum, not something this lemma proves.
  method TimestampBoundToInitBlock(
      i: I.Interop, view: I.FrontierView, blocksAtTS: map<ChainID, BlockID>,
      exec: nat, execChain: ChainID, msg: ExecutingMessage)
    returns (ok: bool)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires blocksAtTS.Keys == CHAIN_IDS
    requires i.IsCorrectFrontierView(view, blocksAtTS)
    ensures ok ==> msg.chainID in CHAIN_IDS
    ensures ok && msg.timestamp < exec ==>
      i.logsDBs[msg.chainID].FindSealedBlock(msg.blockNum).Some? &&
      var sealed := i.logsDBs[msg.chainID].FindSealedBlock(msg.blockNum).value;
      sealed.timestamp == msg.timestamp &&
      i.chains[msg.chainID].BlockInfo(sealed.id).Some? &&
      i.chains[msg.chainID].BlockInfo(sealed.id).value.id.number == msg.blockNum &&
      i.chains[msg.chainID].BlockInfo(sealed.id).value.timestamp == msg.timestamp
    ensures ok && msg.timestamp == exec ==>
      i.chains[msg.chainID].BlockInfo(blocksAtTS[msg.chainID]).Some? &&
      i.chains[msg.chainID].BlockInfo(blocksAtTS[msg.chainID]).value.id.number == msg.blockNum &&
      i.chains[msg.chainID].BlockInfo(blocksAtTS[msg.chainID]).value.timestamp == msg.timestamp
  {
    ok := i.VerifyExecutingMessage(execChain, exec, msg, view);
    if ok && msg.timestamp < exec {
      var c := msg.chainID;
      var sealed := i.logsDBs[c].FindSealedBlock(msg.blockNum).value;
      assert i.LogsDBConsistentWithChainData(c) by {
        reveal i.AllLogsDBsConsistentWithChainData();
      }
      reveal i.LogsDBConsistentWithChainData();
      assert i.logsDBs[c].FindSealedBlock(sealed.id.number).Some?;
      assert i.BlockExistedOnChain(c, sealed.id);
    }
  }

  // ---------------------------------------------------------------------------
  // 4. Boundary / non-vacuity: at exec == init + W the message is still valid (when the
  //    activation checks pass), and at init + W + 1 it is not. So the theorem above is not
  //    vacuously true of ValidExecutingMessage, and the boundary is inclusive (<=), as in Go
  //    (`exec - init > W` rejects), kona (`exec - init <= W` accepts), Lean and Quint (`t <= e + W`).
  // ---------------------------------------------------------------------------
  lemma BoundaryStillValid(i: I.Interop, execChain: ChainID, msg: ExecutingMessage)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires msg.chainID in CHAIN_IDS
    requires i.activationTimestamp + i.chains[msg.chainID].BlockTime() <= msg.timestamp
    requires i.activationTimestamp + i.chains[execChain].BlockTime() <= msg.timestamp + MESSAGE_EXPIRY_WINDOW
    ensures i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW, execChain, msg)
    ensures !i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW + 1, execChain, msg)
  {
  }

  // The hypotheses of BoundaryStillValid are satisfiable for every Valid instance and every pair
  // of registered chains: an initiating timestamp late enough for both activation checks exists.
  lemma BoundaryHypothesesSatisfiable(i: I.Interop, execChain: ChainID, initChain: ChainID)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires initChain in CHAIN_IDS
    ensures exists msg: ExecutingMessage ::
      msg.chainID == initChain &&
      i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW, execChain, msg)
  {
    var t := i.activationTimestamp + i.chains[initChain].BlockTime() + i.chains[execChain].BlockTime();
    var msg := ExecutingMessage(initChain, 0, 0, t, 0);
    BoundaryStillValid(i, execChain, msg);
  }

  // End-to-end reachability witness: the model's own constructor yields a Valid() instance (so the
  // Valid() hypotheses above are not vacuous), a message valid at the boundary exists on it, and
  // for that same message the imperative check rejects every exec >= tExport once tExport is past
  // init + EXPIRY_PERIOD. Hypotheses: CHAIN_IDS (abstract in Types.dfy) is non-empty, some
  // ChainContainer object cc exists (the model's ChainContainer has no constructor, so it cannot be
  // allocated from outside its module), and the cap. cc serves every chain; its BlockTime() is free.
  method EndToEndWitness(cc: CC.ChainContainer, execChain: ChainID, initChain: ChainID, view: I.FrontierView)
    requires execChain in CHAIN_IDS
    requires initChain in CHAIN_IDS
    requires MESSAGE_EXPIRY_WINDOW <= PROTOCOL_WINDOW_CAP
  {
    var chains := map k | k in CHAIN_IDS :: cc;
    var i := new I.Interop(chains);
    assert i.Valid();
    var t := i.activationTimestamp + i.chains[initChain].BlockTime() + i.chains[execChain].BlockTime();
    var msg := ExecutingMessage(initChain, 0, 0, t, 0);
    BoundaryStillValid(i, execChain, msg);
    assert i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW, execChain, msg);
    var tExport := msg.timestamp + EXPIRY_PERIOD + 1;
    var ok := ExpiredAtExportRejectedWithCap(i, view, tExport, tExport, execChain, msg);
    assert !ok;
  }

  // ---------------------------------------------------------------------------
  // 5. Counterexample to the timing implication when P < W: there is an export time
  //    tExport > init + P (so the contract would accept the "undelivered" fact) and an
  //    exec >= tExport at which the message still satisfies ValidExecutingMessage. This is temporal
  //    eligibility only: it does not establish initiating-log presence, imperative acceptance,
  //    export, expiry or refund. The full double-spend execution is Lean's cex_periodBelowWindow.
  // ---------------------------------------------------------------------------
  lemma ShortPeriodCounterexample(i: I.Interop, P: nat, execChain: ChainID, msg: ExecutingMessage)
      returns (tExport: nat, exec: nat)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires msg.chainID in CHAIN_IDS
    requires i.activationTimestamp + i.chains[msg.chainID].BlockTime() <= msg.timestamp
    requires i.activationTimestamp + i.chains[execChain].BlockTime() <= msg.timestamp + MESSAGE_EXPIRY_WINDOW
    requires P < MESSAGE_EXPIRY_WINDOW
    ensures tExport > msg.timestamp + P   // expireMessage would accept the fact (t > sentAt + P)
    ensures exec >= tExport               // the relay happens at or after the export
    ensures i.ValidExecutingMessage(exec, execChain, msg)
  {
    tExport := msg.timestamp + P + 1;
    exec := msg.timestamp + MESSAGE_EXPIRY_WINDOW;
    BoundaryStillValid(i, execChain, msg);
  }

  // ---------------------------------------------------------------------------
  // 6. The 7-day cap. MESSAGE_EXPIRY_WINDOW is an abstract const in Types.dfy (declared with no
  //    value), so the model holds for every W, and the cap can only enter as a hypothesis. With
  //    the cap, the theorem's side condition W <= P holds for P = EXPIRY_PERIOD, with one day of
  //    margin.
  // ---------------------------------------------------------------------------
  lemma CapImpliesWindowWithinPeriod()
    requires MESSAGE_EXPIRY_WINDOW <= PROTOCOL_WINDOW_CAP
    ensures MESSAGE_EXPIRY_WINDOW <= EXPIRY_PERIOD
    ensures MESSAGE_EXPIRY_WINDOW + 86400 <= EXPIRY_PERIOD
  {
  }

  // ---------------------------------------------------------------------------
  // 7. Relation to the Lean/Quint relay rule `t <= e + W`. Every message valid in the Dafny model
  //    satisfies the numerical window check t <= e + W with e = initTimestamp (they omit
  //    init <= exec and the activation checks). Full relay-set inclusion additionally needs the
  //    event correspondence (Lean `events z h e` / Quint `e in evts` = this message's initiating
  //    log) and window identification (Quint counts in days); neither is encoded here. The
  //    converse fails on the omitted checks (shown by the witness below).
  // ---------------------------------------------------------------------------
  lemma DafnyValidImpliesLeanWithinWindow(i: I.Interop, exec: nat, execChain: ChainID, msg: ExecutingMessage)
    requires execChain in CHAIN_IDS
    requires i.chains.Keys == CHAIN_IDS
    requires i.ValidExecutingMessage(exec, execChain, msg)
    ensures exec <= msg.timestamp + i.messageExpiryWindow
  {
  }

  // With activation passing, Dafny validity is exactly `init <= exec <= init + W`.
  lemma ValidIffWindow(i: I.Interop, exec: nat, execChain: ChainID, msg: ExecutingMessage)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires msg.chainID in CHAIN_IDS
    requires i.activationTimestamp + i.chains[execChain].BlockTime() <= exec
    requires i.activationTimestamp + i.chains[msg.chainID].BlockTime() <= msg.timestamp
    ensures i.ValidExecutingMessage(exec, execChain, msg) ==>
      (msg.timestamp <= exec && exec <= msg.timestamp + MESSAGE_EXPIRY_WINDOW)
    ensures (msg.timestamp <= exec && exec <= msg.timestamp + MESSAGE_EXPIRY_WINDOW) ==>
      i.ValidExecutingMessage(exec, execChain, msg)
  {
  }

  // The Lean/Quint check is strictly weaker: exec < init passes `t <= e + W` but is invalid here.
  lemma LeanAcceptsFutureInitDafnyRejects(i: I.Interop, execChain: ChainID, msg: ExecutingMessage)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires msg.timestamp > 0
    ensures (msg.timestamp - 1) <= msg.timestamp + MESSAGE_EXPIRY_WINDOW
    ensures !i.ValidExecutingMessage(msg.timestamp - 1, execChain, msg)
  {
  }
}
