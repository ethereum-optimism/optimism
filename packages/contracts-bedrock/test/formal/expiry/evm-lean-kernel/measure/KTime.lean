import ExpiryEvm
import Kernel.Flat
open Lean Meta Elab Command Term

/-- `#ktime p` : elaborate the Prop `p`, synthesize its `Decidable` instance, and kernel-check
`Eq.refl true : decide p = true` synchronously; print ok/FAIL and wall ms. -/
elab "#ktime " t:term : command => do
  let env ← getEnv
  let ty ← liftTermElabM do
    let p ← elabTerm t (some (mkSort 0))
    synthesizeSyntheticMVarsNoPostponing
    let p ← instantiateMVars p
    let d ← mkDecide p
    let d ← instantiateMVars d
    mkEq d (mkConst ``Bool.true)
  let pf := mkApp2 (mkConst ``Eq.refl [1]) (mkConst ``Bool) (mkConst ``Bool.true)
  let decl := Declaration.thmDecl { name := `ktime_tmp, levelParams := [], type := ty, value := pf }
  let t0 ← IO.monoNanosNow
  let r := Lean.Environment.addDeclCore env 0 decl none
  let t1 ← IO.monoNanosNow
  match r with
  | .ok _ => logInfo m!"ok {(t1 - t0) / 1000000} ms : {t}"
  | .error e => logInfo m!"FAIL {(t1 - t0) / 1000000} ms : {t}\n{e.toMessageData {}}"
