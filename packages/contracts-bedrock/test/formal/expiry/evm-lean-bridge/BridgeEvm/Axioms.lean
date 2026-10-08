import BridgeEvm

/-! `lake env lean BridgeEvm/Axioms.lean` (also built by `lake build BridgeEvm.Axioms`) checks
the axiom footprint of the headline theorems.

Expected: exactly `propext`, `Classical.choice`, `Quot.sound`. Every closed bytecode fact
(instruction decodes, pc arithmetic, the JUMPDEST table, JUMPDEST membership) is checked by the
Lean kernel (`decide +kernel`, see `BridgeEvm/KernelDecide.lean`, copied from `../evm-lean`); no
`native_decide` / `Lean.ofReduceBool` axiom remains. `#assert_std_axioms` makes the build fail if
one comes back.

The `Concrete.*` executable witnesses run `Ξ` by compiled evaluation (`native_decide`); they are
tests, not dependencies of the headline theorems. -/

open Lean Elab Command in
/-- Fail unless the constant depends on no axiom beyond `propext`, `Classical.choice`,
`Quot.sound`. -/
elab "#assert_std_axioms " id:ident : command => do
  let n ← liftCoreM <| realizeGlobalConstNoOverloadWithInfo id
  let axs ← liftCoreM <| Lean.collectAxioms n
  let bad := axs.filter (fun a => !(#[``propext, ``Classical.choice, ``Quot.sound].contains a))
  unless bad.isEmpty do
    throwError m!"{n} depends on non-standard axioms ({bad.size}): {bad.toList.take 5} …"
  logInfo m!"{n}: standard axioms only ({axs.toList})"

open Lean Elab Command in
/-- Fail unless `BridgeEvm.NonVacuous.nonvacuous_<n>` exists, where `<n>` is the theorem's name
without the `BridgeEvm.` prefix and with `.` replaced by `_` (see `NonVacuous.lean`). -/
elab "#assert_nonvacuous " id:ident : command => do
  let n ← liftCoreM <| realizeGlobalConstNoOverloadWithInfo id
  let short := (n.toString.replace "BridgeEvm." "").replace "." "_"
  let partner := Name.mkStr (Name.mkStr (Name.mkStr .anonymous "BridgeEvm") "NonVacuous")
    ("nonvacuous_" ++ short)
  unless (← getEnv).contains partner do
    throwError m!"{n} has no non-vacuity partner {partner}"
  logInfo m!"{n}: non-vacuity partner {partner} present"

/-- Every headline theorem: standard axioms only, and a `nonvacuous_` partner. -/
macro "#assert_headline " id:ident : command =>
  `(#assert_std_axioms $id
    #assert_nonvacuous $id)

#assert_headline BridgeEvm.refundETH_trace
#assert_headline BridgeEvm.refundETH_outcome
#assert_headline BridgeEvm.refundETH_success
#assert_headline BridgeEvm.refundETH_no_other_error
#assert_headline BridgeEvm.createStep_success
#assert_headline BridgeEvm.storedMap_post
#assert_headline BridgeEvm.refundETH_store
#assert_headline BridgeEvm.RD.create
#assert_headline BridgeEvm.refundPreimage_size

-- The partners themselves: kernel-checked except the isolated `Ξ`-evaluation lemmas.
#print axioms BridgeEvm.NonVacuous.nonvacuous_refundETH_success
#print axioms BridgeEvm.NonVacuous.nonvacuous_RD_create

#print axioms BridgeEvm.Concrete.refundHash_matches_cast
#print axioms BridgeEvm.Concrete.success_reachable
#print axioms BridgeEvm.Concrete.wrong_preimage_reverts
#print axioms BridgeEvm.Concrete.already_refunded_reverts
