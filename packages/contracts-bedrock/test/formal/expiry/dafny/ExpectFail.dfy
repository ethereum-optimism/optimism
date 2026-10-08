// EXPECTED TO FAIL. run.sh checks that Dafny exits 4 (verification errors) with exactly two
// errors, both "a postcondition could not be proved", whose related locations are exactly the two
// ensures lines marked EXPECT-FAIL below (one in each lemma), and nothing else.
// Each lemma is a too-strong variant of a theorem in ExpiryBridge.dfy; if either ever verified,
// the main theorem's hypotheses would be shown to be unnecessary or the model vacuous.

include "../../../../../../op-supernode/dafny-models/Interop.dfy"

module ExpiryBridgeExpectFail {
  import opened Types
  import I = Interop

  // FAILS: the main theorem without the side condition W <= P. With P < W, a message exported
  // after init + P can still be valid at init + W (ExpiryBridge.ShortPeriodCounterexample).
  lemma NoPeriodAssumption(
      i: I.Interop, P: nat, tExport: nat, exec: nat, execChain: ChainID, msg: ExecutingMessage)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    requires tExport > msg.timestamp + P
    requires exec >= tExport
    ensures !i.ValidExecutingMessage(exec, execChain, msg) // EXPECT-FAIL NoPeriodAssumption
  {
  }

  // FAILS: an exclusive boundary. At exec == init + W the message is still valid
  // (ExpiryBridge.BoundaryStillValid), so a rule that expired it there would disagree with the model.
  lemma ExclusiveBoundary(i: I.Interop, execChain: ChainID, msg: ExecutingMessage)
    requires i.Valid()
    requires execChain in CHAIN_IDS
    ensures !i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW, execChain, msg) // EXPECT-FAIL ExclusiveBoundary
  {
  }
}
