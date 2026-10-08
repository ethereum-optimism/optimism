import ExporterEvm

/-! `lake build ExporterEvm.Axioms` checks the axiom footprint of the headline theorems and their
non-vacuity partners, and fails the build if a check fails. (Same checker as
`../evm-lean-bridge/BridgeEvm/Axioms.lean`.)

`#assert_headline T` checks:
1. `T` depends only on `propext`, `Classical.choice`, `Quot.sound`. Every closed bytecode fact
   (instruction decodes, pc arithmetic, the JUMPDEST table, JUMPDEST membership) is checked by the
   Lean kernel (`decide +kernel`, see `ExporterEvm/KernelDecide.lean`); no `native_decide` axiom.
2. The partner `ExporterEvm.NonVacuous.nonvacuous_<n>` exists, where `<n>` is `T`'s name without
   the `ExporterEvm.` prefix and with `.` replaced by `_` (`NonVacuous.lean`), and its proof term
   applies `T`.
3. Every axiom of the partner beyond the three standard ones comes from a theorem whose name starts
   with `native_` (compiled evaluation of `Ξ` on the concrete bytecode); the traversal does not
   enter those theorems and lists them; each may use only the standard axioms plus the axioms
   `native_decide` adds for it (`<name>._native.native_decide.ax_*` in Lean 4.29).
So a headline theorem without a partner, a partner that does not apply its theorem, or compiled
evaluation outside a `native_*` lemma breaks the build.

The other `Concrete.*` executable tests run `Ξ` by compiled evaluation (`native_decide`); they are
not dependencies of the headline theorems, and their footprint is printed. -/

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
  let short := (n.toString.replace "ExporterEvm." "").replace "." "_"
  let wn := Name.mkStr (Name.mkStr (Name.mkStr .anonymous "ExporterEvm") "NonVacuous")
    ("nonvacuous_" ++ short)
  let env ← getEnv
  let some wi := env.find? wn
    | throwError m!"headline theorem {n} has no non-vacuity partner {wn}"
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
  logInfo m!"{n}: standard axioms only. Partner {wn} applies it; kernel-checked except the \
    compiled-evaluation lemmas {nat.toList}"

#assert_headline ExporterEvm.export_trace
#assert_headline ExporterEvm.export_outcome
#assert_headline ExporterEvm.export_success
#assert_headline ExporterEvm.export_revert
#assert_headline ExporterEvm.export_no_other_error
#assert_headline ExporterEvm.exportPreimage_size

-- The kernel-checked ties of the statement's definitions to Solidity's ABI (via cast), and the
-- kernel-checked parts of the partners.
#assert_std_axioms ExporterEvm.Concrete.exportHash_matches_cast
#assert_std_axioms ExporterEvm.Concrete.sendMessageCd_matches_cast
#assert_std_axioms ExporterEvm.NonVacuous.w_sel
#assert_std_axioms ExporterEvm.NonVacuous.w_args

#print axioms ExporterEvm.Concrete.success_reachable
#print axioms ExporterEvm.Concrete.relayed_reverts
