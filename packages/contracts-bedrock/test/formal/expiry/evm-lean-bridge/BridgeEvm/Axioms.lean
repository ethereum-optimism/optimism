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

#assert_std_axioms BridgeEvm.refundETH_trace
#assert_std_axioms BridgeEvm.refundETH_outcome
#assert_std_axioms BridgeEvm.refundETH_success
#assert_std_axioms BridgeEvm.refundETH_no_other_error
#assert_std_axioms BridgeEvm.createStep_success
#assert_std_axioms BridgeEvm.storedMap_post
#assert_std_axioms BridgeEvm.refundETH_bridgeStorage
#assert_std_axioms BridgeEvm.RD.create
#assert_std_axioms BridgeEvm.refundPreimage_size

#print axioms BridgeEvm.Concrete.refundHash_matches_cast
#print axioms BridgeEvm.Concrete.success_reachable
#print axioms BridgeEvm.Concrete.wrong_preimage_reverts
#print axioms BridgeEvm.Concrete.already_refunded_reverts
