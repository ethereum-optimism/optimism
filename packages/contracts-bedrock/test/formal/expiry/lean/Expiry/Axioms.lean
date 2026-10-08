import Lean
import Expiry.Safety
import Expiry.Counterexamples
import Expiry.NonVacuity

/-!
Axiom footprint and non-vacuity check, run by `lake build`.

`#assert_headline T` fails the build unless
1. `T` depends on no axiom beyond `propext`, `Classical.choice`, `Quot.sound`;
2. a theorem `nonvacuous_T` (same namespace) exists, applies `T` in its proof term, and is itself
   kernel-checked with the same three axioms only (`NonVacuity.lean`).
So adding a headline theorem to the list below without a witness breaks the build.
-/

open Expiry Expiry.Examples

open Lean Elab Command in
/-- See the module docstring. -/
elab "#assert_headline " id:ident : command => do
  let n ← liftCoreM <| realizeGlobalConstNoOverloadWithInfo id
  let std := #[``propext, ``Classical.choice, ``Quot.sound]
  let axs ← liftCoreM <| Lean.collectAxioms n
  let bad := axs.filter (fun a => !std.contains a)
  unless bad.isEmpty do
    throwError m!"{n} depends on non-standard axioms: {bad.toList}"
  let wn ← match n with
    | .str p s => pure (Name.str p ("nonvacuous_" ++ s))
    | _ => throwError m!"unexpected name {n}"
  let some wi := (← getEnv).find? wn
    | throwError m!"headline theorem {n} has no non-vacuity witness {wn}"
  let .thmInfo wt := wi
    | throwError m!"{wn} is not a theorem"
  unless wt.value.getUsedConstants.contains n do
    throwError m!"{wn} does not apply {n} (its proof term does not mention it)"
  let waxs ← liftCoreM <| Lean.collectAxioms wn
  let wbad := waxs.filter (fun a => !std.contains a)
  unless wbad.isEmpty do
    throwError m!"{wn} depends on non-standard axioms: {wbad.toList}"
  logInfo m!"{n}: standard axioms only; witness {wn} applies it (standard axioms only)"

-- Headline theorems (Safety.lean): each needs `nonvacuous_<name>` in NonVacuity.lean.
#assert_headline exporterSilentBeforeUpgrade
#assert_headline joinNeedsNoHistoryCheck
#assert_headline noDoubleSpend
#assert_headline refundImpliesExpired
#assert_headline atMostOneRefund
#assert_headline noForgedFact
#assert_headline expiredImpliesNeverRelayable
#assert_headline expired_no_relay_step
#assert_headline onlyDestinationCanExport
#assert_headline safety
#assert_headline safety_without_targetRule
#assert_headline messengerSilentAfterUpgrade

-- Witnesses and counterexamples (Counterexamples.lean).
#print axioms refund_reachable
#print axioms relay_reachable_at_edge
#print axioms lateJoin_multiSource_reachable
#print axioms forgery_reachable_but_harmless
#print axioms safe_variants
#print axioms legacyResend_reachable
#print axioms cex_messengerTrusted
#print axioms cex_nonstandardJoin
#print axioms cex_periodBelowWindow
#print axioms cex_nonStrict
#print axioms cex_resendNoRestart
#print axioms cex_noRealMessengerCheck
#print axioms cex_noLockboxCheck
#print axioms cex_noLockboxCheck_fakePortal
#print axioms cex_sysConfigInconsistent
#print axioms cex_noUnsafeTargetCheck
#print axioms cex_noSenderCheck
#print axioms cex_hashCollision
#print axioms cex_duplicateChainId
#print axioms cex_exporterGovernanceUpgrade
#print axioms messengerSpeaks_without_targetRule
