import ExpiryEvm
open Lean Meta Elab Command

/-! For every `native_decide` axiom of the headline theorems, compute the constants reachable
from its statement through definition bodies (and instance/recursor references), and report
which of them carry `@[implemented_by]` or `@[extern]` (i.e. where compiled evaluation does not
run the Lean definition). -/

def roots : List Name := [``ExpiryEvm.expireMessage_outcome, ``ExpiryEvm.expireMessage_success,
    ``ExpiryEvm.expireMessage_revert_cause, ``ExpiryEvm.expireMessage_no_other_error,
    ``ExpiryEvm.Abstract.refines_expire]

partial def reach (env : Environment) (todo : List Name) (seen : NameSet) : NameSet :=
  match todo with
  | [] => seen
  | n :: rest =>
    if seen.contains n then reach env rest seen else
    let seen := seen.insert n
    let cs : List Name := match env.find? n with
      | some (.defnInfo d) => d.type.getUsedConstants.toList ++ d.value.getUsedConstants.toList
      | some (.opaqueInfo d) => d.type.getUsedConstants.toList ++ d.value.getUsedConstants.toList
      | some (.inductInfo d) => d.ctors
      | some (.ctorInfo d) => [d.induct]
      | some (.recInfo d) => d.type.getUsedConstants.toList
      | some (.thmInfo _) => []   -- proofs are irrelevant for evaluation
      | some ci => ci.type.getUsedConstants.toList
      | none => []
    reach env (cs ++ rest) seen

elab "#native_reach" : command => do
  let env ← getEnv
  let mut all : NameSet := {}
  for r in roots do
    for a in (← liftCoreM <| Lean.collectAxioms r) do all := all.insert a
  let natives := all.toList.filter (fun n => (n.toString.splitOn "native_decide").length > 1)
  let mut start : List Name := []
  for n in natives do
    if let some ci := env.find? n then start := ci.type.getUsedConstants.toList ++ start
  let r := reach env start {}
  let mut impl : Array String := #[]
  let mut ext : Array String := #[]
  for n in r.toList do
    if let some t := Compiler.implementedByAttr.getParam? env n then impl := impl.push s!"{n} -> {t}"
    if isExtern env n then ext := ext.push n.toString
  logInfo m!"reachable constants: {r.size}\nimplemented_by ({impl.size}): {impl.qsort (· < ·)}\nextern ({ext.size}): {ext.qsort (· < ·)}"
  let ethy := r.toList.filter (fun n => n.toString.startsWith "Ethereum" || n.toString.startsWith "ffi")
  logInfo m!"Ethereum/ffi constants reached ({ethy.length}): {(ethy.map toString).toArray.qsort (· < ·)}"
  for bad in [`totallySafePerformIO, `unsafePerformIO, `ffi.sha256, `ffi.BLAKE2Compress,
              `Ethereum.EVM.Ξ, `Ethereum.EVM.Θ, `Ethereum.EVM.X] do
    logInfo m!"{bad} reached: {r.contains bad}"

#native_reach
