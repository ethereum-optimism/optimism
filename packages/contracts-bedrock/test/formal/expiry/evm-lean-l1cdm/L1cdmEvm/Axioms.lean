import L1cdmEvm

/-! `lake build L1cdmEvm.Axioms` (or `lake env lean L1cdmEvm/Axioms.lean`) checks the axiom
footprint of the headline theorems and fails the build unless it is exactly within `propext`,
`Classical.choice`, `Quot.sound`: every closed bytecode fact (instruction decodes, pc arithmetic,
the JUMPDEST table and membership, memory-descriptor equalities) is checked by the Lean kernel
(`decide +kernel`, see `KernelDecide.lean`); no `native_decide` / `Lean.ofReduceBool` axiom.

The `Concrete.*` executable witnesses run `Ξ` by compiled evaluation (`native_decide`); they are
tests, not dependencies of the headline theorems, and their footprint is printed. -/

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

#assert_std_axioms L1cdmEvm.relay_success
#assert_std_axioms L1cdmEvm.relay_outcome
#assert_std_axioms L1cdmEvm.relay_no_other_error
#assert_std_axioms L1cdmEvm.send_success
#assert_std_axioms L1cdmEvm.send_outcome
#assert_std_axioms L1cdmEvm.relay_deposit
#assert_std_axioms L1cdmEvm.mock_returnsWord
#assert_std_axioms L1cdmEvm.concrete_summaries
#assert_std_axioms L1cdmEvm.concrete_self

#print axioms L1cdmEvm.Concrete.success_reachable
#print axioms L1cdmEvm.Concrete.badSender_reverts
