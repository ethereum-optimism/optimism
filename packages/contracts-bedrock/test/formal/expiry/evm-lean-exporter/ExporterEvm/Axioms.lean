import ExporterEvm

/-! `lake build ExporterEvm.Axioms` checks the axiom footprint of the headline theorems.

Expected: exactly `propext`, `Classical.choice`, `Quot.sound`. Every closed bytecode fact
(instruction decodes, pc arithmetic, the JUMPDEST table, JUMPDEST membership) is checked by the
Lean kernel (`decide +kernel`, see `ExporterEvm/KernelDecide.lean`, copied from `../evm-lean`); no
`native_decide` / `Lean.ofReduceBool` axiom remains. `#assert_std_axioms` makes the build fail if
one comes back.

The `Concrete.*` executable witnesses and the `native_run_*` lemmas run `Ξ` by compiled
evaluation (`native_decide`); they are tests and non-vacuity witnesses, not dependencies of the
headline theorems. -/

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
/-- Fail unless `ExporterEvm.NonVacuous.nonvacuous_<n>` exists, where `<n>` is the theorem's name
without the `ExporterEvm.` prefix and with `.` replaced by `_` (see `NonVacuous.lean`), **and** its
proof term uses the theorem `n` itself (so a partner that does not instantiate the theorem, e.g.
one proving `True`, is rejected). Whether the instantiation discharges every hypothesis with a
concrete witness is a property of the partner's statement, reviewed by hand (`NonVacuous.lean`). -/
elab "#assert_nonvacuous " id:ident : command => do
  let n ← liftCoreM <| realizeGlobalConstNoOverloadWithInfo id
  let short := (n.toString.replace "ExporterEvm." "").replace "." "_"
  let partner := Name.mkStr (Name.mkStr (Name.mkStr .anonymous "ExporterEvm") "NonVacuous")
    ("nonvacuous_" ++ short)
  let env ← getEnv
  let some info := env.find? partner
    | throwError m!"{n} has no non-vacuity partner {partner}"
  let some val := info.value?
    | throwError m!"{partner} has no proof term"
  unless val.getUsedConstants.contains n do
    throwError m!"{partner} does not use {n}"
  logInfo m!"{n}: non-vacuity partner {partner} present and uses it"

/-- Every headline theorem: standard axioms only, and a `nonvacuous_` partner. -/
macro "#assert_headline " id:ident : command =>
  `(#assert_std_axioms $id
    #assert_nonvacuous $id)

#assert_headline ExporterEvm.export_trace
#assert_headline ExporterEvm.export_outcome
#assert_headline ExporterEvm.export_success
#assert_headline ExporterEvm.export_revert
#assert_headline ExporterEvm.export_no_other_error
#assert_headline ExporterEvm.exportPreimage_size

-- The kernel-checked ties of the statement's definitions to Solidity's ABI (via cast):
#assert_std_axioms ExporterEvm.Concrete.exportHash_matches_cast
#assert_std_axioms ExporterEvm.Concrete.sendMessageCd_matches_cast

-- The partners themselves: kernel-checked except the isolated `Ξ`-evaluation lemmas.
#print axioms ExporterEvm.NonVacuous.nonvacuous_export_success
#print axioms ExporterEvm.NonVacuous.nonvacuous_export_revert
#print axioms ExporterEvm.NonVacuous.nonvacuous_export_no_other_error
#print axioms ExporterEvm.NonVacuous.nonvacuous_exportPreimage_size

#print axioms ExporterEvm.Concrete.success_reachable
#print axioms ExporterEvm.Concrete.relayed_reverts
