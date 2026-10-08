import ExpiryEvm

/-! `lake env lean ExpiryEvm/Axioms.lean` prints the axiom footprint of the headline theorems.

Expected for the soundness theorems and the bridge: exactly `propext`, `Classical.choice`,
`Quot.sound`. Every closed bytecode fact (instruction decodes, pc arithmetic, the JUMPDEST table,
JUMPDEST membership) is checked by the Lean kernel (`decide +kernel`, see
`ExpiryEvm/KernelDecide.lean`); no `native_decide` / `Lean.ofReduceBool` axiom remains.
`#assert_std_axioms` below makes `lake build` fail if one comes back.

The `Concrete.*` executable witnesses still run `Ξ` by compiled evaluation (`native_decide`);
they are tests, not dependencies of the headline theorems. -/

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

#assert_std_axioms ExpiryEvm.expireMessage_outcome
#assert_std_axioms ExpiryEvm.expireMessage_success
#assert_std_axioms ExpiryEvm.expireMessage_revert_cause
#assert_std_axioms ExpiryEvm.expireMessage_no_other_error
#assert_std_axioms ExpiryEvm.Abstract.refines_expire

#print axioms ExpiryEvm.Concrete.success_reachable
#print axioms ExpiryEvm.Concrete.boundary_reverts
