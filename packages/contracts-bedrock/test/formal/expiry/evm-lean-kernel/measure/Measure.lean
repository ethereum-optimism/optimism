import ExpiryEvm
import Kernel.Flat
import Kernel.KD
open Lean Meta Elab Command

/-! Kernel-check the `native_decide` facts of the headline theorems, one by one, synchronously
(`Environment.addDeclCore`, no compiled code). For each axiom `ax : decide p = true` we submit
`theorem ax.kernel : decide p = true := Eq.refl true`, optionally after replacing
`l2tol2Runtime` by `l2tol2RuntimeFlat` (variant `flat`). -/

def roots : List Name := [``ExpiryEvm.expireMessage_outcome, ``ExpiryEvm.expireMessage_success,
    ``ExpiryEvm.expireMessage_revert_cause, ``ExpiryEvm.expireMessage_no_other_error,
    ``ExpiryEvm.Abstract.refines_expire]

def nativeAxioms : CommandElabM (Array Name) := do
  let mut all : NameSet := {}
  for r in roots do
    for a in (← liftCoreM <| Lean.collectAxioms r) do all := all.insert a
  return (all.toList.filter (fun n => (n.toString.splitOn "native_decide").length > 1)).toArray.qsort
    (fun a b => a.toString < b.toString)

def classify (ty : Expr) : String :=
  if ty.containsConst (· == ``Ethereum.EVM.decode) then "decode"
  else if ty.containsConst (· == ``Ethereum.EVM.D_J) then "jumpdest_table"
  else if ty.containsConst (· == ``Array.contains) then "jumpdest_member"
  else "pc_arith"

elab "#kernel_measure " v:ident lo:num hi:num : command => do
  let natives ← nativeAxioms
  let env ← getEnv
  let mut out := ""
  for i in [lo.getNat:min hi.getNat natives.size] do
    let n := natives[i]!
    let some ci := env.find? n | continue
    let flat (e : Expr) := e.replace (fun e => if e.isConstOf ``ExpiryEvm.l2tol2Runtime then
        some (mkConst ``ExpiryEvm.l2tol2RuntimeFlat) else none)
    let toL (c : Expr) := mkApp2 (mkConst ``Array.toList [0]) (mkConst ``UInt8) (mkApp (mkConst ``ByteArray.data) c)
    let viaList (e : Expr) := e.replace (fun e =>
      if e.isAppOfArity ``Ethereum.EVM.decode 2 then
        let c := e.getArg! 0; let pc := e.getArg! 1
        some (mkApp (mkConst ``KD.decodeList)
          (mkApp3 (mkConst ``List.drop [0]) (mkConst ``UInt8) (mkApp (mkConst ``Ethereum.UInt256.toNat) pc) (toL c)))
      else if e.isAppOfArity ``Ethereum.EVM.D_J 2 then
        let c := e.getArg! 0; let i := e.getArg! 1
        let len := mkApp2 (mkConst ``List.length [0]) (mkConst ``UInt8) (toL c)
        some (mkApp4 (mkConst ``KD.jumpdestScan)
          (mkApp2 (mkConst ``Nat.add) len (mkNatLit 1))
          (mkApp3 (mkConst ``List.drop [0]) (mkConst ``UInt8) i (toL c)) i
          (mkApp (mkConst ``Array.empty [0]) (mkConst ``Ethereum.UInt256)))
      else none)
    let ty := match v.getId with
      | `flat => flat ci.type
      | `list => viaList (flat ci.type)
      | `listchunk => viaList ci.type
      | _ => ci.type
    let pf := mkApp2 (mkConst ``Eq.refl [1]) (mkConst ``Bool) (mkConst ``Bool.true)
    let decl := Declaration.thmDecl { name := n ++ `kernel, levelParams := [], type := ty, value := pf }
    let t0 ← IO.monoNanosNow
    let res := match Lean.Environment.addDeclCore env 0 decl none with
      | .ok _ => "ok" | .error _ => "FAIL"
    let t1 ← IO.monoNanosNow
    out := out ++ s!"{i}\t{classify ci.type}\t{n}\t{res}\t{(t1 - t0) / 1000000}\n"
    IO.FS.writeFile s!"Kernel/m_{v.getId}_{lo.getNat}_{hi.getNat}.tsv" out
