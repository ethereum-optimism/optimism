-- Hand-written models of the external types that `KonaSp1.Code.Types` uses, started from the
-- Aeneas template. Unmodelled types stay axioms.
module
public import Aeneas
@[expose] public section
open Aeneas Aeneas.Std Result ControlFlow Error
set_option linter.dupNamespace false
set_option linter.hashCommand false
set_option linter.unusedVariables false
set_option linter.style.whitespace false
set_option linter.style.setOption false
set_option linter.style.longLine false
set_option linter.internalConstructors false

/- You can set the `maxHeartbeats` value with the `-max-heartbeats` CLI option -/
set_option maxHeartbeats 1000000

/- You can set the `maxRecDepth` value with the `-max-recdepth` CLI option -/
set_option maxRecDepth 2048

/-- [core::iter::adapters::flatten::FlatMap]
    Source: '/rustc/library/core/src/iter/adapters/flatten.rs', lines 17:0-17:41
    Name pattern: [core::iter::adapters::flatten::FlatMap]
    Visibility: public -/
@[rust_type "core::iter::adapters::flatten::FlatMap"]
structure core.iter.adapters.flatten.FlatMap (I : Type) (U : Type) (F : Type)
  (Clause0_Item : Type) (Clause0_IntoIter : Type) where
  /-- `Fuse<Map<I, F>>`; `none` once fused. -/
  iter : Option (core.iter.adapters.map.Map I F)
  frontiter : Option Clause0_IntoIter
  backiter : Option Clause0_IntoIter

/-- [core::slice::iter::Windows]
    Source: '/rustc/library/core/src/slice/iter.rs', lines 1329:0-1329:29
    Name pattern: [core::slice::iter::Windows]
    Visibility: public -/
@[rust_type "core::slice::iter::Windows"]
axiom core.slice.iter.Windows (T : Type) : Type

/-- [ruint::Uint]
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/ruint-1.20.0/src/lib.rs', lines 158:0-158:54
    Name pattern: [ruint::Uint]
    Visibility: public -/
@[rust_type "ruint::Uint"]
structure ruint.Uint (BITS : Std.Usize) (LIMBS : Std.Usize) where
  val : Nat
  hBound : val < 2 ^ BITS.val
deriving DecidableEq

