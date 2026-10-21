-- Hand-written models of the external functions that `KonaSp1.Code.Funs` calls, started from
-- the Aeneas template. Unmodelled functions stay axioms; `#print axioms` on a theorem lists
-- every one it depends on.
module
public import Aeneas
public import KonaSp1.Code.Types
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
open KonaSp1

/-- [core::convert::num::ptr_try_from_impls::{impl core::convert::TryFrom<u64, core::num::error::TryFromIntError> for usize}::try_from]:
    Source: '/rustc/library/core/src/convert/num.rs', lines 341:12-341:68
    Name pattern: [core::convert::num::ptr_try_from_impls::{core::convert::TryFrom<usize, u64, core::num::error::TryFromIntError>}::try_from]
    Visibility: public -/
@[rust_fun
  "core::convert::num::ptr_try_from_impls::{core::convert::TryFrom<usize, u64, core::num::error::TryFromIntError>}::try_from"]
def Usize.Insts.CoreConvertTryFromU64TryFromIntError.try_from
  (x : Std.U64) : Result (core.result.Result Std.Usize core.num.error.TryFromIntError) :=
  core.num.tryFromUScalar .Usize x

/-- [core::iter::traits::iterator::Iterator::flat_map]:
    Source: '/rustc/library/core/src/iter/traits/iterator.rs', lines 1547:4-1551:34
    Name pattern: [core::iter::traits::iterator::Iterator::flat_map]
    Visibility: public -/
@[trait_default, rust_fun "core::iter::traits::iterator::Iterator::flat_map"]
def core.iter.traits.iterator.Iterator.flat_map.default
  {Self : Type} {U : Type} {F : Type} {Clause0_Item : Type} {Clause1_Item :
  Type} {Clause1_IntoIter : Type} (IteratorInst :
  core.iter.traits.iterator.Iterator Self Clause0_Item)
  (collectIntoIteratorInst : core.iter.traits.collect.IntoIterator U
  Clause1_Item Clause1_IntoIter) (opsfunctionFnMutFTupleClause0_ItemUInst :
  core.ops.function.FnMut F Clause0_Item U) (iter : Self) (f : F) :
  Result (core.iter.adapters.flatten.FlatMap Self U F Clause1_Item Clause1_IntoIter) :=
  ok { iter := some { iter, f }, frontiter := none, backiter := none }

/-- [core::iter::traits::iterator::Iterator::map]:
    Source: '/rustc/library/core/src/iter/traits/iterator.rs', lines 845:4-848:34
    Name pattern: [core::iter::traits::iterator::Iterator::map]
    Visibility: public -/
@[trait_default, rust_fun "core::iter::traits::iterator::Iterator::map"]
def core.iter.traits.iterator.Iterator.map.default
  {Self : Type} {B : Type} {F : Type} {Clause0_Item : Type} (IteratorInst :
  core.iter.traits.iterator.Iterator Self Clause0_Item)
  (opsfunctionFnMutFTupleClause0_ItemBInst : core.ops.function.FnMut F
  Clause0_Item B) (iter : Self) (f : F) : Result (core.iter.adapters.map.Map Self F) :=
  ok { iter, f }

/-- [core::iter::adapters::flatten::{impl core::iter::traits::iterator::Iterator<Clause1_Item> for core::iter::adapters::flatten::FlatMap<I, U, F, Clause1_Item, Clause1_IntoIter>}::next]:
    Source: '/rustc/library/core/src/iter/adapters/flatten.rs', lines 63:4-63:41
    Name pattern: [core::iter::adapters::flatten::{core::iter::traits::iterator::Iterator<core::iter::adapters::flatten::FlatMap<@I, @U, @F, @Clause1_Item, @Clause1_IntoIter>, @Clause1_Item>}::next]
    Visibility: public -/
@[rust_fun
  "core::iter::adapters::flatten::{core::iter::traits::iterator::Iterator<core::iter::adapters::flatten::FlatMap<@I, @U, @F, @Clause1_Item, @Clause1_IntoIter>, @Clause1_Item>}::next"]
def
  core.iter.adapters.flatten.FlatMap.Insts.CoreIterTraitsIteratorIterator.next
  {I : Type} {U : Type} {F : Type} {Clause0_Item : Type} {Clause1_Item : Type}
  {Clause1_IntoIter : Type} (traitsiteratorIteratorInst :
  core.iter.traits.iterator.Iterator I Clause0_Item)
  (traitscollectIntoIteratorInst : core.iter.traits.collect.IntoIterator U
  Clause1_Item Clause1_IntoIter) (opsfunctionFnMutFTupleClause0_ItemUInst :
  core.ops.function.FnMut F Clause0_Item U)
  (s : core.iter.adapters.flatten.FlatMap I U F Clause1_Item Clause1_IntoIter) :
  Result ((Option Clause1_Item) × (core.iter.adapters.flatten.FlatMap I U F
    Clause1_Item Clause1_IntoIter)) :=
  -- `FlattenCompat::next` with the unspecialized `Fuse::next` and `Map::next` inlined. The
  -- `FusedIterator` specialization of `Fuse` does not clear `iter`; it yields the same items.
  let innerInst := traitscollectIntoIteratorInst.iteratorInst
  let back (s : core.iter.adapters.flatten.FlatMap I U F Clause1_Item Clause1_IntoIter) :
      Result (ControlFlow (core.iter.adapters.flatten.FlatMap I U F Clause1_Item Clause1_IntoIter)
        ((Option Clause1_Item) ×
          (core.iter.adapters.flatten.FlatMap I U F Clause1_Item Clause1_IntoIter))) :=
    match s.backiter with
    | none => ok (done (none, s))
    | some inner => do
      let (o, inner1) ← innerInst.next inner
      match o with
      | some x => ok (done (some x, { s with backiter := some inner1 }))
      | none => ok (done (none, { s with backiter := none }))
  loop (fun s =>
    match s.frontiter with
    | some inner => do
      let (o, inner1) ← innerInst.next inner
      match o with
      | some x => ok (done (some x, { s with frontiter := some inner1 }))
      | none => ok (cont { s with frontiter := none })
    | none =>
      match s.iter with
      | none => back s
      | some m => do
        let (o, it) ← traitsiteratorIteratorInst.next m.iter
        match o with
        | none => back { s with iter := none }
        | some x =>
          let (u, f) ← opsfunctionFnMutFTupleClause0_ItemUInst.call_mut m.f x
          let inner ← traitscollectIntoIteratorInst.into_iter u
          ok (cont { s with iter := some { iter := it, f }, frontiter := some inner })) s

/-- [core::iter::adapters::map::{impl core::iter::traits::iterator::Iterator<B> for core::iter::adapters::map::Map<I, F>}::next]:
    Source: '/rustc/library/core/src/iter/adapters/map.rs', lines 106:4-106:35
    Name pattern: [core::iter::adapters::map::{core::iter::traits::iterator::Iterator<core::iter::adapters::map::Map<@I, @F>, @B>}::next]
    Visibility: public -/
@[rust_fun
  "core::iter::adapters::map::{core::iter::traits::iterator::Iterator<core::iter::adapters::map::Map<@I, @F>, @B>}::next"]
def core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next
  {B : Type} {I : Type} {F : Type} {Clause0_Item : Type}
  (traitsiteratorIteratorInst : core.iter.traits.iterator.Iterator I
  Clause0_Item) (opsfunctionFnMutFTupleClause0_ItemBInst :
  core.ops.function.FnMut F Clause0_Item B) (m : core.iter.adapters.map.Map I F) :
  Result ((Option B) × (core.iter.adapters.map.Map I F)) := do
  let (o, iter) ← traitsiteratorIteratorInst.next m.iter
  match o with
  | none => ok (none, { m with iter })
  | some x =>
    let (y, f) ← opsfunctionFnMutFTupleClause0_ItemBInst.call_mut m.f x
    ok (some y, { iter, f })

/-- [core::option::{core::option::Option<T>}::unwrap_or_default]:
    Source: '/rustc/library/core/src/option.rs', lines 1093:4-1095:27
    Name pattern: [core::option::{core::option::Option<@T>}::unwrap_or_default]
    Visibility: public -/
@[rust_fun "core::option::{core::option::Option<@T>}::unwrap_or_default"]
axiom core.option.Option.unwrap_or_default
  {T : Type} (defaultDefaultInst : core.default.Default T) :
  Option T → Result T

/-- [core::option::{core::option::Option<T>}::and_then]:
    Source: '/rustc/library/core/src/option.rs', lines 1539:4-1541:61
    Name pattern: [core::option::{core::option::Option<@T>}::and_then]
    Visibility: public -/
@[rust_fun "core::option::{core::option::Option<@T>}::and_then"]
def core.option.Option.and_then
  {T : Type} {U : Type} {F : Type} (opsfunctionFnOnceFTupleTOptionInst :
  core.ops.function.FnOnce F T (Option U)) (o : Option T) (f : F) : Result (Option U) :=
  match o with
  | none => ok none
  | some x => opsfunctionFnOnceFTupleTOptionInst.call_once f x

/-- [core::option::{core::option::Option<&'_0 T>}::copied]:
    Source: '/rustc/library/core/src/option.rs', lines 2135:4-2137:16
    Name pattern: [core::option::{core::option::Option<&'0 @T>}::copied]
    Visibility: public -/
@[rust_fun "core::option::{core::option::Option<&'0 @T>}::copied"]
def core.option.OptionShared0T.copied
  {T : Type} (markerCopyInst : core.marker.Copy T) (o : Option T) : Result (Option T) :=
  ok o

/-- [core::option::{impl core::cmp::PartialEq<core::option::Option<T>> for core::option::Option<T>}::eq]:
    Source: '/rustc/library/core/src/option.rs', lines 2441:4-2441:38
    Name pattern: [core::option::{core::cmp::PartialEq<core::option::Option<@T>, core::option::Option<@T>>}::eq]
    Visibility: public -/
@[rust_fun
  "core::option::{core::cmp::PartialEq<core::option::Option<@T>, core::option::Option<@T>>}::eq"]
def core.option.Option.Insts.CoreCmpPartialEqOption.eq
  {T : Type} (cmpPartialEqInst : core.cmp.PartialEq T T) (a b : Option T) : Result Bool :=
  match a, b with
  | some x, some y => cmpPartialEqInst.eq x y
  | none, none => ok true
  | _, _ => ok false

/-- [core::result::{core::result::Result<T, E>}::ok]:
    Source: '/rustc/library/core/src/result.rs', lines 709:4-712:28
    Name pattern: [core::result::{core::result::Result<@T, @E>}::ok]
    Visibility: public -/
@[rust_fun "core::result::{core::result::Result<@T, @E>}::ok"]
def core.result.Result.ok
  {T : Type} {E : Type} (r : core.result.Result T E) : Result (Option T) :=
  match r with
  | .Ok x => Aeneas.Std.Result.ok (some x)
  | .Err _ => Aeneas.Std.Result.ok none

/-- [core::slice::iter::{impl core::iter::traits::iterator::Iterator<&'a [T]> for core::slice::iter::Windows<'a, T>}::next]:
    Source: '/rustc/library/core/src/slice/iter.rs', lines 1354:4-1354:41
    Name pattern: [core::slice::iter::{core::iter::traits::iterator::Iterator<core::slice::iter::Windows<'a, @T>, &'a [@T]>}::next]
    Visibility: public -/
@[rust_fun
  "core::slice::iter::{core::iter::traits::iterator::Iterator<core::slice::iter::Windows<'a, @T>, &'a [@T]>}::next"]
axiom
  core.slice.iter.Windows.Insts.CoreIterTraitsIteratorIteratorSharedASlice.next
  {T : Type} :
  core.slice.iter.Windows T → Result ((Option (Slice T)) ×
    (core.slice.iter.Windows T))

/-- [core::slice::{[T]}::first]:
    Source: '/rustc/library/core/src/slice/mod.rs', lines 157:4-157:43
    Name pattern: [core::slice::{[@T]}::first]
    Visibility: public -/
@[rust_fun "core::slice::{[@T]}::first"]
def core.slice.Slice.first {T : Type} (s : Slice T) : Result (Option T) :=
  ok s.val.head?

/-- [core::slice::{[T]}::last]:
    Source: '/rustc/library/core/src/slice/mod.rs', lines 283:4-283:42
    Name pattern: [core::slice::{[@T]}::last]
    Visibility: public -/
@[rust_fun "core::slice::{[@T]}::last"]
def core.slice.Slice.last {T : Type} (s : Slice T) : Result (Option T) :=
  ok s.val.getLast?

/-- [core::slice::{[T]}::windows]:
    Source: '/rustc/library/core/src/slice/mod.rs', lines 1116:4-1116:62
    Name pattern: [core::slice::{[@T]}::windows]
    Visibility: public -/
@[rust_fun "core::slice::{[@T]}::windows"]
axiom core.slice.Slice.windows
  {T : Type} : Slice T → Std.Usize → Result (core.slice.iter.Windows T)

/-- [alloc::vec::{alloc::vec::Vec<T>}::is_empty]:
    Source: '/rustc/library/alloc/src/vec/mod.rs', lines 3160:4-3160:40
    Name pattern: [alloc::vec::{alloc::vec::Vec<@T>}::is_empty]
    Visibility: public -/
@[rust_fun "alloc::vec::{alloc::vec::Vec<@T>}::is_empty"]
def alloc.vec.Vec.is_empty
  {T : Type} (A : Type) (v : alloc.vec.Vec T) : Result Bool :=
  ok v.val.isEmpty

/-- [alloc::vec::{impl core::iter::traits::collect::IntoIterator<&'a T, core::slice::iter::Iter<'a, T>> for &'a alloc::vec::Vec<T>}::into_iter]:
    Source: '/rustc/library/alloc/src/vec/mod.rs', lines 4088:4-4088:40
    Name pattern: [alloc::vec::{core::iter::traits::collect::IntoIterator<&'a alloc::vec::Vec<@T>, &'a @T, core::slice::iter::Iter<'a, @T>>}::into_iter]
    Visibility: public -/
@[rust_fun
  "alloc::vec::{core::iter::traits::collect::IntoIterator<&'a alloc::vec::Vec<@T>, &'a @T, core::slice::iter::Iter<'a, @T>>}::into_iter"]
def SharedAVec.Insts.CoreIterTraitsCollectIntoIteratorSharedATIter.into_iter
  {T : Type} (A : Type) (v : alloc.vec.Vec T) : Result (core.slice.iter.Iter T) :=
  ok ⟨ alloc.vec.Vec.deref v, 0 ⟩

/-- [alloc::vec::{impl core::default::Default for alloc::vec::Vec<T>}::default]:
    Source: '/rustc/library/alloc/src/vec/mod.rs', lines 4417:4-4417:26
    Name pattern: [alloc::vec::{core::default::Default<alloc::vec::Vec<@T>>}::default]
    Visibility: public -/
@[rust_fun
  "alloc::vec::{core::default::Default<alloc::vec::Vec<@T>>}::default"]
axiom alloc.vec.Vec.Insts.CoreDefaultDefault.default
  (T : Type) : Result (alloc.vec.Vec T)

/-- [alloy_primitives::bits::fixed::{impl core::clone::Clone for alloy_primitives::bits::fixed::FixedBytes<N>}::clone]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/alloy-primitives-1.7.3/src/bits/fixed.rs', lines 16:4-16:9
    Name pattern: [alloy_primitives::bits::fixed::{core::clone::Clone<alloy_primitives::bits::fixed::FixedBytes<@N>>}::clone]
    Visibility: public -/
@[rust_fun
  "alloy_primitives::bits::fixed::{core::clone::Clone<alloy_primitives::bits::fixed::FixedBytes<@N>>}::clone"]
def alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCloneClone.clone
  {N : Std.Usize} (x : alloy_primitives.bits.fixed.FixedBytes N) :
  Result (alloy_primitives.bits.fixed.FixedBytes N) :=
  ok x

/-- [alloy_primitives::bits::fixed::{impl core::cmp::PartialEq<alloy_primitives::bits::fixed::FixedBytes<N>> for alloy_primitives::bits::fixed::FixedBytes<N>}::eq]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/alloy-primitives-1.7.3/src/bits/fixed.rs', lines 16:17-16:26
    Name pattern: [alloy_primitives::bits::fixed::{core::cmp::PartialEq<alloy_primitives::bits::fixed::FixedBytes<@N>, alloy_primitives::bits::fixed::FixedBytes<@N>>}::eq]
    Visibility: public -/
@[rust_fun
  "alloy_primitives::bits::fixed::{core::cmp::PartialEq<alloy_primitives::bits::fixed::FixedBytes<@N>, alloy_primitives::bits::fixed::FixedBytes<@N>>}::eq"]
def
  alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq
  {N : Std.Usize} (a b : alloy_primitives.bits.fixed.FixedBytes N) : Result Bool :=
  ok (decide (a = b))

/-- [alloy_primitives::bits::address::{impl core::clone::Clone for alloy_primitives::bits::address::Address}::clone]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/alloy-primitives-1.7.3/src/bits/macros.rs', lines 42:12-42:17
    Name pattern: [alloy_primitives::bits::address::{core::clone::Clone<alloy_primitives::bits::address::Address>}::clone]
    Visibility: public -/
@[rust_fun
  "alloy_primitives::bits::address::{core::clone::Clone<alloy_primitives::bits::address::Address>}::clone"]
axiom alloy_primitives.bits.address.Address.Insts.CoreCloneClone.clone
  :
  alloy_primitives.bits.address.Address → Result
    alloy_primitives.bits.address.Address

/-- [alloy_primitives::bits::address::{impl core::cmp::PartialEq<alloy_primitives::bits::address::Address> for alloy_primitives::bits::address::Address}::eq]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/alloy-primitives-1.7.3/src/bits/macros.rs', lines 45:12-45:21
    Name pattern: [alloy_primitives::bits::address::{core::cmp::PartialEq<alloy_primitives::bits::address::Address, alloy_primitives::bits::address::Address>}::eq]
    Visibility: public -/
@[rust_fun
  "alloy_primitives::bits::address::{core::cmp::PartialEq<alloy_primitives::bits::address::Address, alloy_primitives::bits::address::Address>}::eq"]
axiom alloy_primitives.bits.address.Address.Insts.CoreCmpPartialEqAddress.eq
  :
  alloy_primitives.bits.address.Address →
    alloy_primitives.bits.address.Address → Result Bool

/-- [kona_interop::constants::SUPER_ROOT_VERSION]
    Source: 'kona/crates/protocol/interop/src/constants.rs', lines 7:0-7:32
    Name pattern: [kona_interop::constants::SUPER_ROOT_VERSION]
    Visibility: public -/
@[rust_const "kona_interop::constants::SUPER_ROOT_VERSION"]
axiom kona_interop.constants.SUPER_ROOT_VERSION : Result Std.U8

/-- [kona_interop::root::{impl core::clone::Clone for kona_interop::root::SuperRoot}::clone]:
    Source: 'kona/crates/protocol/interop/src/root.rs', lines 12:16-12:21
    Name pattern: [kona_interop::root::{core::clone::Clone<kona_interop::root::SuperRoot>}::clone]
    Visibility: public -/
@[rust_fun
  "kona_interop::root::{core::clone::Clone<kona_interop::root::SuperRoot>}::clone"]
axiom kona_interop.root.SuperRoot.Insts.CoreCloneClone.clone
  : kona_interop.root.SuperRoot → Result kona_interop.root.SuperRoot

/-- [kona_interop::root::{impl core::cmp::PartialEq<kona_interop::root::SuperRoot> for kona_interop::root::SuperRoot}::eq]:
    Source: 'kona/crates/protocol/interop/src/root.rs', lines 12:27-12:36
    Name pattern: [kona_interop::root::{core::cmp::PartialEq<kona_interop::root::SuperRoot, kona_interop::root::SuperRoot>}::eq]
    Visibility: public -/
@[rust_fun
  "kona_interop::root::{core::cmp::PartialEq<kona_interop::root::SuperRoot, kona_interop::root::SuperRoot>}::eq"]
axiom kona_interop.root.SuperRoot.Insts.CoreCmpPartialEqSuperRoot.eq
  : kona_interop.root.SuperRoot → kona_interop.root.SuperRoot → Result Bool

/-- [kona_interop::root::{kona_interop::root::SuperRoot}::new]:
    Source: 'kona/crates/protocol/interop/src/root.rs', lines 24:4-24:82
    Name pattern: [kona_interop::root::{kona_interop::root::SuperRoot}::new]
    Visibility: public -/
@[rust_fun "kona_interop::root::{kona_interop::root::SuperRoot}::new"]
axiom kona_interop.root.SuperRoot.new
  :
  Std.U64 → alloc.vec.Vec kona_interop.root.OutputRootWithChain → Result
    kona_interop.root.SuperRoot

/-- [ruint::cmp::{impl core::cmp::PartialOrd<ruint::Uint<BITS, LIMBS>> for ruint::Uint<BITS, LIMBS>}::ge]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/ruint-1.20.0/src/cmp.rs', lines 8:12-8:46
    Name pattern: [ruint::cmp::{core::cmp::PartialOrd<ruint::Uint<@BITS, @LIMBS>, ruint::Uint<@BITS, @LIMBS>>}::ge]
    Visibility: public -/
@[rust_fun
  "ruint::cmp::{core::cmp::PartialOrd<ruint::Uint<@BITS, @LIMBS>, ruint::Uint<@BITS, @LIMBS>>}::ge"]
axiom ruint.Uint.Insts.CoreCmpPartialOrdUint.ge
  {BITS : Std.Usize} {LIMBS : Std.Usize} :
  ruint.Uint BITS LIMBS → ruint.Uint BITS LIMBS → Result Bool

/-- [ruint::from::{ruint::Uint<BITS, LIMBS>}::from]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/ruint-1.20.0/src/from.rs', lines 160:4-162:29
    Name pattern: [ruint::from::{ruint::Uint<@BITS, @LIMBS>}::from]
    Visibility: public -/
@[rust_fun "ruint::from::{ruint::Uint<@BITS, @LIMBS>}::from"]
def ruint.from.Uint.from
  {T : Type} {BITS : Std.Usize} {LIMBS : Std.Usize} (UintTryFromUintTInst :
  ruint.from.UintTryFrom (ruint.Uint BITS LIMBS) T) (x : T) :
  Result (ruint.Uint BITS LIMBS) := do
  let r ← UintTryFromUintTInst.uint_try_from x
  match r with
  | .Ok v => ok v
  | .Err _ => fail .panic

/-- [ruint::from::{impl ruint::from::UintTryFrom<T> for ruint::Uint<BITS, LIMBS>}::uint_try_from]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/ruint-1.20.0/src/from.rs', lines 371:4-371:65
    Name pattern: [ruint::from::{ruint::from::UintTryFrom<ruint::Uint<@BITS, @LIMBS>, @T>}::uint_try_from]
    Visibility: public -/
@[rust_fun
  "ruint::from::{ruint::from::UintTryFrom<ruint::Uint<@BITS, @LIMBS>, @T>}::uint_try_from"]
def ruint.Uint.Insts.RuintFromUintTryFrom.uint_try_from
  {T : Type} {BITS : Std.Usize} {LIMBS : Std.Usize}
  (coreconvertTryFromUintTToUintErrorUintInst : core.convert.TryFrom
  (ruint.Uint BITS LIMBS) T (ruint.from.ToUintError (ruint.Uint BITS LIMBS))) (x : T) :
  Result (core.result.Result (ruint.Uint BITS LIMBS)
    (ruint.from.ToUintError (ruint.Uint BITS LIMBS))) :=
  coreconvertTryFromUintTToUintErrorUintInst.try_from x

/-- [ruint::from::{impl core::convert::TryFrom<u64, ruint::from::ToUintError<ruint::Uint<BITS, LIMBS>>> for ruint::Uint<BITS, LIMBS>}::try_from]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/ruint-1.20.0/src/from.rs', lines 428:4-428:56
    Name pattern: [ruint::from::{core::convert::TryFrom<ruint::Uint<@BITS, @LIMBS>, u64, ruint::from::ToUintError<ruint::Uint<@BITS, @LIMBS>>>}::try_from]
    Visibility: public -/
@[rust_fun
  "ruint::from::{core::convert::TryFrom<ruint::Uint<@BITS, @LIMBS>, u64, ruint::from::ToUintError<ruint::Uint<@BITS, @LIMBS>>>}::try_from"]
def ruint.Uint.Insts.CoreConvertTryFromU64ToUintErrorUint.try_from
  (BITS : Std.Usize) (LIMBS : Std.Usize) (x : Std.U64) :
  Result (core.result.Result (ruint.Uint BITS LIMBS)
    (ruint.from.ToUintError (ruint.Uint BITS LIMBS))) :=
  if h : x.val < 2 ^ BITS.val then ok (.Ok ⟨x.val, h⟩)
  else
    have hpos : 0 < 2 ^ BITS.val := Nat.two_pow_pos _
    ok (.Err (.ValueTooLarge BITS ⟨x.val % 2 ^ BITS.val, Nat.mod_lt _ hpos⟩))

/-- [ruint::{impl core::clone::Clone for ruint::Uint<BITS, LIMBS>}::clone]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/ruint-1.20.0/src/lib.rs', lines 152:9-152:14
    Name pattern: [ruint::{core::clone::Clone<ruint::Uint<@BITS, @LIMBS>>}::clone]
    Visibility: public -/
@[rust_fun "ruint::{core::clone::Clone<ruint::Uint<@BITS, @LIMBS>>}::clone"]
axiom ruint.Uint.Insts.CoreCloneClone.clone
  {BITS : Std.Usize} {LIMBS : Std.Usize} :
  ruint.Uint BITS LIMBS → Result (ruint.Uint BITS LIMBS)

/-- [ruint::{impl core::cmp::PartialEq<ruint::Uint<BITS, LIMBS>> for ruint::Uint<BITS, LIMBS>}::eq]:
    Source: '/cargo/registry/src/index.crates.io-1949cf8c6b5b557f/ruint-1.20.0/src/lib.rs', lines 155:11-155:20
    Name pattern: [ruint::{core::cmp::PartialEq<ruint::Uint<@BITS, @LIMBS>, ruint::Uint<@BITS, @LIMBS>>}::eq]
    Visibility: public -/
@[rust_fun
  "ruint::{core::cmp::PartialEq<ruint::Uint<@BITS, @LIMBS>, ruint::Uint<@BITS, @LIMBS>>}::eq"]
def ruint.Uint.Insts.CoreCmpPartialEqUint.eq
  {BITS : Std.Usize} {LIMBS : Std.Usize} (a b : ruint.Uint BITS LIMBS) : Result Bool :=
  ok (decide (a = b))

/-- [kona_sp1_client_utils::super_root::ensure_range_previous_super_root_coverage]:
    Source: 'kona/sp1/crates/client/src/super_root.rs', lines 934:0-974:1 -/
axiom super_root.ensure_range_previous_super_root_coverage
  :
  super_root.TimestampSpan → Slice (ruint.Uint 256#usize 4#usize) → Slice
    super_root.SuperRootProof → Result (core.result.Result Unit
    super_root.SuperRootError)

/-- [kona_sp1_client_utils::super_root::ensure_range_transition_coverage]:
    Source: 'kona/sp1/crates/client/src/super_root.rs', lines 890:0-932:1 -/
axiom super_root.ensure_range_transition_coverage
  :
  super_root.TimestampSpan → Slice (ruint.Uint 256#usize 4#usize) → Slice
    super_root.SuperRangeTransition → Result (core.result.Result Unit
    super_root.SuperRootError)

/-- [kona_sp1_client_utils::super_root::ensure_consolidation_input_coverage]:
    Source: 'kona/sp1/crates/client/src/super_root.rs', lines 1013:0-1054:1 -/
axiom super_root.ensure_consolidation_input_coverage
  :
  super_root.TimestampSpan → Slice
    super_root.SuperConsolidationTransitionInput → Result (core.result.Result
    Unit super_root.SuperRootError)

/-- [kona_sp1_client_utils::super_root::hash_super_root_proof]:
    Source: 'kona/sp1/crates/client/src/super_root.rs', lines 851:0-853:1
    Visibility: public -/
axiom super_root.hash_super_root_proof
  :
  super_root.SuperRootProof → Result (core.result.Result
    (alloy_primitives.bits.fixed.FixedBytes 32#usize)
    super_root.SuperRootError)

/-- [kona_sp1_client_utils::super_root::{kona_sp1_client_utils::super_root::SuperAggregationInputs}::zk_dispute_game_public_values]:
    Source: 'kona/sp1/crates/client/src/super_root.rs', lines 327:4-329:5
    Visibility: public -/
axiom super_root.SuperAggregationInputs.zk_dispute_game_public_values
  : super_root.SuperAggregationInputs → Result (alloc.vec.Vec Std.U8)

/-- [kona_sp1_client_utils::super_root::encode_super_root_proof]:
    Source: 'kona/sp1/crates/client/src/super_root.rs', lines 841:0-848:1
    Visibility: public -/
axiom super_root.encode_super_root_proof
  :
  super_root.SuperRootProof → Result (core.result.Result (alloc.vec.Vec
    Std.U8) super_root.SuperRootError)

