// Non-vacuity witnesses: one `Nonvacuous_<Name>` method per lemma/method of ExpiryBridge.dfy.
// run.sh checks that every lemma/method declared in ExpiryBridge.dfy has a witness here that calls
// it, and that this file verifies with 0 errors.
//
// Each witness builds a concrete instance with the model's own constructor (new I.Interop(...),
// whose postcondition gives Valid()), picks a concrete executing message, and calls the lemma: Dafny
// then has to prove every `requires` of the lemma JOINTLY at the call site, and the witness asserts
// the lemma's conclusion on that instance. Where the conclusion is negative ("not valid", "rejected"),
// the witness also shows that the same message IS valid at the boundary, so the conclusion is not
// true merely because nothing is ever valid.
//
// Residual hypotheses of the witnesses (Types.dfy and ChainContainer.dfy leave these abstract, so
// they can only enter as `requires`; each is a statement about an abstract constant or function of
// the model and is satisfiable by choosing that constant/function):
//   - c in CHAIN_IDS                     (CHAIN_IDS is non-empty)
//   - cc: a ChainContainer object         (the class has no constructor outside its module)
//   - MESSAGE_EXPIRY_WINDOW <= PROTOCOL_WINDOW_CAP   (only where the lemma itself assumes the cap)
//   - MESSAGE_EXPIRY_WINDOW > 0          (only ShortPeriodCounterexample: P < W needs W >= 1)
//   - cc.BlockInfo(blk).Some?            (only TimestampBoundToInitBlock: some block exists)
// ExpectFail.dfy's NonVacuityContextConsistent checks that their conjunction does not let Dafny
// prove `false` (a smoke test against an inconsistent context, not a consistency proof).

include "ExpiryBridge.dfy"

module ExpiryBridgeNonVacuity {
  import opened Types
  import I = Interop
  import CC = ChainContainer
  import B = ExpiryBridge

  // An initiating timestamp late enough for both activation checks (one ChainContainer serves
  // every chain, so both block times are cc.BlockTime()).
  function InitTime(i: I.Interop, cc: CC.ChainContainer): nat
    reads i
  {
    i.activationTimestamp + cc.BlockTime() + cc.BlockTime()
  }

  method Nonvacuous_ExpiredAtExportNeverValid(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    B.BoundaryStillValid(i, c, msg);
    assert i.ValidExecutingMessage(msg.timestamp + i.messageExpiryWindow, c, msg);
    var P := i.messageExpiryWindow;          // W <= P with P = W
    var tExport := msg.timestamp + P + 1;
    B.ExpiredAtExportNeverValid(i, P, tExport, tExport, c, msg);
    assert !i.ValidExecutingMessage(tExport, c, msg);
  }

  method Nonvacuous_ExpiredAtExportNeverValidGlobal(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    B.BoundaryStillValid(i, c, msg);
    var tExport := msg.timestamp + MESSAGE_EXPIRY_WINDOW + 1;
    B.ExpiredAtExportNeverValidGlobal(i, MESSAGE_EXPIRY_WINDOW, tExport, tExport + 5, c, msg);
    assert !i.ValidExecutingMessage(tExport + 5, c, msg);
  }

  method Nonvacuous_ExpiredAtExportRejected(cc: CC.ChainContainer, c: ChainID, view: I.FrontierView)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    B.BoundaryStillValid(i, c, msg);
    var tExport := msg.timestamp + MESSAGE_EXPIRY_WINDOW + 1;
    var ok := B.ExpiredAtExportRejected(i, view, MESSAGE_EXPIRY_WINDOW, tExport, tExport, c, msg);
    assert !ok;
  }

  method Nonvacuous_ExpiredAtExportRejectedWithCap(cc: CC.ChainContainer, c: ChainID, view: I.FrontierView)
    requires c in CHAIN_IDS
    requires MESSAGE_EXPIRY_WINDOW <= B.PROTOCOL_WINDOW_CAP
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    B.BoundaryStillValid(i, c, msg);
    var tExport := msg.timestamp + B.EXPIRY_PERIOD + 1;
    var ok := B.ExpiredAtExportRejectedWithCap(i, view, tExport, tExport, c, msg);
    assert !ok;
    assert !i.ValidExecutingMessage(tExport, c, msg);
  }

  method Nonvacuous_ExpiryIsTheFailingGuard(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    var tExport := msg.timestamp + MESSAGE_EXPIRY_WINDOW + 1;
    B.ExpiryIsTheFailingGuard(i, MESSAGE_EXPIRY_WINDOW, tExport, tExport, c, msg);
    assert msg.timestamp + i.messageExpiryWindow < tExport;
  }

  // The relevant execution for this lemma (VerifyExecutingMessage accepting, ok == true) is not
  // exhibited: acceptance depends on the abstract logs-DB / frontier-view contents. The witness shows
  // that the hypotheses (Valid instance, a correct frontier view) are jointly satisfiable; the
  // correct view comes from the model's ResolveFrontierVerificationView, whose
  // `ensures {:axiom} IsCorrectFrontierView(...)` is how the model itself supplies that property.
  method Nonvacuous_TimestampBoundToInitBlock(cc: CC.ChainContainer, c: ChainID, blk: BlockID)
    requires c in CHAIN_IDS
    requires cc.BlockInfo(blk).Some?
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var blocksAtTS := map k | k in CHAIN_IDS :: blk;
    assert i.BlocksExistedOnChain(blocksAtTS) by {
      forall k | k in blocksAtTS.Keys ensures i.BlockExistedOnChain(k, blocksAtTS[k]) {
        assert i.chains[k] == cc;
      }
    }
    var view := i.ResolveFrontierVerificationView(blocksAtTS);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    var ok := B.TimestampBoundToInitBlock(i, view, blocksAtTS, msg.timestamp, c, msg);
    assert ok ==> msg.chainID in CHAIN_IDS;
  }

  method Nonvacuous_BoundaryStillValid(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    B.BoundaryStillValid(i, c, msg);
    assert i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW, c, msg);
    assert !i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW + 1, c, msg);
  }

  method Nonvacuous_BoundaryHypothesesSatisfiable(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    B.BoundaryHypothesesSatisfiable(i, c, c);
    assert exists msg: ExecutingMessage ::
      msg.chainID == c && i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW, c, msg);
  }

  method Nonvacuous_EndToEndWitness(cc: CC.ChainContainer, c: ChainID, view: I.FrontierView)
    requires c in CHAIN_IDS
    requires MESSAGE_EXPIRY_WINDOW <= B.PROTOCOL_WINDOW_CAP
  {
    B.EndToEndWitness(cc, c, c, view);
  }

  method Nonvacuous_ShortPeriodCounterexample(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
    requires MESSAGE_EXPIRY_WINDOW > 0
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    var P := MESSAGE_EXPIRY_WINDOW - 1;
    var tExport, exec := B.ShortPeriodCounterexample(i, P, c, msg);
    assert tExport > msg.timestamp + P && exec >= tExport;
    assert i.ValidExecutingMessage(exec, c, msg);
  }

  method Nonvacuous_CapImpliesWindowWithinPeriod()
    requires MESSAGE_EXPIRY_WINDOW <= B.PROTOCOL_WINDOW_CAP
  {
    B.CapImpliesWindowWithinPeriod();
    assert MESSAGE_EXPIRY_WINDOW + 86400 <= B.EXPIRY_PERIOD;
  }

  method Nonvacuous_DafnyValidImpliesLeanWithinWindow(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    B.BoundaryStillValid(i, c, msg);
    var exec := msg.timestamp + MESSAGE_EXPIRY_WINDOW;
    B.DafnyValidImpliesLeanWithinWindow(i, exec, c, msg);
    assert exec <= msg.timestamp + i.messageExpiryWindow;
  }

  method Nonvacuous_ValidIffWindow(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc), 0);
    var exec := msg.timestamp + MESSAGE_EXPIRY_WINDOW;
    B.ValidIffWindow(i, exec, c, msg);
    assert i.ValidExecutingMessage(exec, c, msg);
  }

  method Nonvacuous_LeanAcceptsFutureInitDafnyRejects(cc: CC.ChainContainer, c: ChainID)
    requires c in CHAIN_IDS
  {
    var i := new I.Interop(map k | k in CHAIN_IDS :: cc);
    var msg := ExecutingMessage(c, 0, 0, InitTime(i, cc) + 1, 0);
    B.LeanAcceptsFutureInitDafnyRejects(i, c, msg);
    assert !i.ValidExecutingMessage(msg.timestamp - 1, c, msg);
  }
}
