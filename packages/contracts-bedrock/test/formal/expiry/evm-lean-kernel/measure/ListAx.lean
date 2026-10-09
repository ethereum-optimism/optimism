import ExpiryEvm
open Lean Meta Elab Command

elab "#dump_native_axioms" : command => do
  let env ← getEnv
  let roots := [``ExpiryEvm.expireMessage_outcome, ``ExpiryEvm.expireMessage_success,
    ``ExpiryEvm.expireMessage_revert_cause, ``ExpiryEvm.expireMessage_no_other_error,
    ``ExpiryEvm.Abstract.refines_expire]
  let mut all : NameSet := {}
  for r in roots do
    let axs ← Lean.collectAxioms r
    for a in axs do all := all.insert a
  let natives := all.toList.filter (fun n => (n.toString.splitOn "native_decide").length > 1)
  logInfo m!"total axioms {all.size}, native {natives.length}"
  let mut out := ""
  for n in natives do
    let some ci := env.find? n | continue
    let t ← liftTermElabM <| ppExpr ci.type
    out := out ++ s!"{n}\t{(toString t).replace "\n" " "}\n"
  IO.FS.writeFile "Kernel/native_axioms.tsv" out

#dump_native_axioms
