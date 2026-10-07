import ExpiryEvm

/-! `lake env lean ExpiryEvm/Axioms.lean` prints the axiom footprint of the headline theorems.
Expected: `propext`, `Classical.choice`, `Quot.sound`, and `…._native.native_decide.ax_*`
(closed facts about the concrete bytecode — instruction decodes at concrete pcs, the JUMPDEST
table, a few closed word computations — checked by compiled evaluation; see README). -/

#print axioms ExpiryEvm.expireMessage_outcome
#print axioms ExpiryEvm.expireMessage_success
#print axioms ExpiryEvm.expireMessage_revert_cause
#print axioms ExpiryEvm.expireMessage_no_other_error
#print axioms ExpiryEvm.Abstract.refines_expire
#print axioms ExpiryEvm.Concrete.success_reachable
#print axioms ExpiryEvm.Concrete.boundary_reverts
