import L1cdmEvm
import L1cdmEvm.NonVacuity

/-! `lake build L1cdmEvm.Axioms` (or `lake env lean L1cdmEvm/Axioms.lean`) checks the axiom
footprint of the headline theorems and their non-vacuity witnesses, and fails the build if a check
fails.

`#assert_headline T` checks:
1. `T` depends only on `propext`, `Classical.choice`, `Quot.sound`: every closed bytecode fact
   (instruction decodes, pc arithmetic, the JUMPDEST table and membership, memory-descriptor
   equalities) is checked by the Lean kernel (`decide +kernel`, see `KernelDecide.lean`); no
   `native_decide` axiom.
2. A theorem `nonvacuous_T` exists in `T`'s namespace (`NonVacuity.lean`) and applies `T` in its
   proof term.
3. Every axiom of `nonvacuous_T` beyond the three standard ones comes from a theorem whose name
   starts with `native_` (compiled evaluation of `Ξ` on the concrete world); the traversal does not
   enter those theorems and lists them; each may use only the standard axioms plus the axioms
   `native_decide` adds for it (`<name>._native.native_decide.ax_*` in Lean 4.29).
So a headline theorem without a witness, a witness that does not apply its theorem, or compiled
evaluation outside a `native_*` lemma breaks the build.

`#assert_std_axioms` checks the kernel-checked parts of the witnesses. The `Concrete.*`
executable tests run `Ξ` by compiled evaluation (`native_decide`); their footprint is printed. -/

open Lean Elab Command

/-- `native_*` theorems: the only place compiled evaluation is allowed in a witness. -/
def isNativeLemma : Name → Bool
  | .str _ s => s.startsWith "native_"
  | _ => false

/-- Constants a declaration refers to (type, value, constructors), as `Lean.collectAxioms` walks
them. -/
def constDeps (ci : ConstantInfo) : Array Name :=
  let v := match ci with
    | .defnInfo d => d.value.getUsedConstants
    | .thmInfo t => t.value.getUsedConstants
    | .opaqueInfo o => o.value.getUsedConstants
    | .inductInfo i => i.ctors.toArray
    | _ => #[]
  ci.type.getUsedConstants ++ v

/-- Axioms reachable from the worklist, not entering `native_*` theorems (other than the root);
returns (axioms, native theorems met). -/
partial def axiomsOutsideNative (env : Environment) (root : Name) :
    List Name → NameSet → Array Name → Array Name → Array Name × Array Name
  | [], _, axs, nat => (axs, nat)
  | c :: rest, seen, axs, nat =>
    if seen.contains c then axiomsOutsideNative env root rest seen axs nat
    else
      let seen := seen.insert c
      if c != root && isNativeLemma c then axiomsOutsideNative env root rest seen axs (nat.push c)
      else match env.find? c with
        | some (.axiomInfo _) => axiomsOutsideNative env root rest seen (axs.push c) nat
        | some ci => axiomsOutsideNative env root ((constDeps ci).toList ++ rest) seen axs nat
        | none => axiomsOutsideNative env root rest seen axs nat

def stdAxioms : Array Name := #[``propext, ``Classical.choice, ``Quot.sound]

/-- Fail unless the constant depends on no axiom beyond `propext`, `Classical.choice`,
`Quot.sound`. -/
elab "#assert_std_axioms " id:ident : command => do
  let n ← liftCoreM <| realizeGlobalConstNoOverloadWithInfo id
  let axs ← liftCoreM <| Lean.collectAxioms n
  let bad := axs.filter (fun a => !stdAxioms.contains a)
  unless bad.isEmpty do
    throwError m!"{n} depends on non-standard axioms ({bad.size}): {bad.toList.take 5} …"
  logInfo m!"{n}: standard axioms only ({axs.toList})"

/-- See the module docstring. -/
elab "#assert_headline " id:ident : command => do
  let n ← liftCoreM <| realizeGlobalConstNoOverloadWithInfo id
  let axs ← liftCoreM <| Lean.collectAxioms n
  let bad := axs.filter (fun a => !stdAxioms.contains a)
  unless bad.isEmpty do
    throwError m!"{n} depends on non-standard axioms ({bad.size}): {bad.toList.take 5} …"
  let wn ← match n with
    | .str p s => pure (Name.str p ("nonvacuous_" ++ s))
    | _ => throwError m!"unexpected name {n}"
  let env ← getEnv
  let some wi := env.find? wn
    | throwError m!"headline theorem {n} has no non-vacuity witness {wn}"
  let .thmInfo wt := wi
    | throwError m!"{wn} is not a theorem"
  unless wt.value.getUsedConstants.contains n do
    throwError m!"{wn} does not apply {n} (its proof term does not mention it)"
  let (waxs, nat) := axiomsOutsideNative env wn [wn] {} #[] #[]
  let wbad := waxs.filter (fun a => !stdAxioms.contains a)
  unless wbad.isEmpty do
    throwError m!"{wn} uses non-standard axioms outside `native_*` lemmas: {wbad.toList}"
  for c in nat do
    let some (.thmInfo _) := env.find? c
      | throwError m!"{c} (named native_) is not a theorem"
    let caxs ← liftCoreM <| Lean.collectAxioms c
    -- `native_decide` (Lean 4.29) adds one axiom `c._native.native_decide.ax_*` per use.
    let cbad := caxs.filter (fun a =>
      !(stdAxioms.contains a || a == ``Lean.ofReduceBool || a == ``Lean.trustCompiler ||
        a.toString.startsWith (c.toString ++ "._native.native_decide.")))
    unless cbad.isEmpty do
      throwError m!"{c} depends on unexpected axioms: {cbad.toList}"
  logInfo m!"{n}: standard axioms only. Witness {wn} applies it; kernel-checked except the \
    compiled-evaluation lemmas {nat.toList}"

#assert_headline L1cdmEvm.relay_success
#assert_headline L1cdmEvm.relay_outcome
#assert_headline L1cdmEvm.relay_no_other_error
#assert_headline L1cdmEvm.send_success
#assert_headline L1cdmEvm.send_outcome
#assert_headline L1cdmEvm.relay_deposit

-- Kernel-checked parts of the witnesses.
#assert_std_axioms L1cdmEvm.mock_returnsWord
#assert_std_axioms L1cdmEvm.concrete_summaries
#assert_std_axioms L1cdmEvm.concrete_self
#assert_std_axioms L1cdmEvm.NV.witness_conds
#assert_std_axioms L1cdmEvm.NV.env_sel
#assert_std_axioms L1cdmEvm.NV.env_cds

#print axioms L1cdmEvm.Concrete.success_reachable
#print axioms L1cdmEvm.Concrete.badSender_reverts
#print axioms L1cdmEvm.Concrete.paused_reverts
