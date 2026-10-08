import ExpiryEvm

/-! `lake build` (or `lake env lean ExpiryEvm/Axioms.lean`) checks the axiom footprint of the
headline theorems and their non-vacuity witnesses, and fails the build if a check fails.

`#assert_headline T` checks:
1. `T` depends only on `propext`, `Classical.choice`, `Quot.sound`. Every closed bytecode fact
   (instruction decodes, pc arithmetic, the JUMPDEST table, JUMPDEST membership) is checked by the
   Lean kernel (`decide +kernel`, see `ExpiryEvm/KernelDecide.lean`); no `native_decide` /
   `Lean.ofReduceBool` axiom.
2. A theorem `nonvacuous_T` exists in `T`'s namespace (`NonVacuity.lean`) and applies `T` in its
   proof term.
3. Every axiom of `nonvacuous_T` beyond the three standard ones comes from a theorem whose name
   starts with `native_` (the compiled evaluations of `Ξ`); the traversal does not enter those
   theorems and lists them. Each listed `native_*` theorem may use only the standard axioms plus
   the axioms `native_decide` adds for that theorem itself (`<name>._native.native_decide.ax_*`
   in Lean 4.29; `Lean.ofReduceBool` / `Lean.trustCompiler` in older versions).
So a headline theorem without a witness, a witness that does not apply its theorem, or compiled
evaluation outside a `native_*` lemma breaks the build.

`#assert_std_axioms` checks supporting lemmas (kernel-checked parts of the witnesses). The other
`Concrete.*` executable tests run `Ξ` by compiled evaluation (`native_decide`); their footprint is
printed. -/

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

#assert_headline ExpiryEvm.expireMessage_outcome
#assert_headline ExpiryEvm.expireMessage_success
#assert_headline ExpiryEvm.expireMessage_revert_cause
#assert_headline ExpiryEvm.expireMessage_no_other_error
#assert_headline ExpiryEvm.Abstract.refines_expire

-- Kernel-checked parts of the witnesses.
#assert_std_axioms ExpiryEvm.Concrete.mock_returnsAddress
#assert_std_axioms ExpiryEvm.NV.witness_conds
#assert_std_axioms ExpiryEvm.Concrete.env_sel
#assert_std_axioms ExpiryEvm.Concrete.env_cds

#print axioms ExpiryEvm.Concrete.native_xi_success
#print axioms ExpiryEvm.Concrete.success_reachable
#print axioms ExpiryEvm.Concrete.boundary_reverts
