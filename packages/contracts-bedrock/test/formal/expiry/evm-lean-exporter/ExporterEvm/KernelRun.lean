import Reasoning.Reach
import ExporterEvm.KernelDecide

/-!
`kevm_run`: EquiVM's `evm_run` with every auto-supplied `decode` obligation discharged by
`evm_kdecide` (kernel) instead of `native_decide`. Same syntax and same steps otherwise
(copied from `Reasoning/Reach.lean` at EquiVM `b0e9d55a`).
-/

open Reasoning.Reach in
syntax "kevm_run " term:max " with " "[" evmStep,* "]" : term

open Lean in
macro_rules
  | `(kevm_run $base:term with [ $steps,* ]) => do
      let mut acc := base
      for s in steps.getElems do
        match s with
        | `(evmStep| raw $op:ident $args*) =>
            acc ← `($(acc).$op $args*)
        | `(evmStep| $op:ident $args*) =>
            match op.getId with
            | `jump    => acc ← `($(acc).jump (by evm_kdecide) $(args[0]!) (by evm_ov))
            | `jumpiT  => acc ← `($(acc).jumpiT (by evm_kdecide) $(args[0]!) $(args[1]!) (by evm_ov))
            | `jumpiNT => acc ← `($(acc).jumpiNT (by evm_kdecide) $(args[0]!) (by evm_ov))
            | _        => acc ← `($(acc).$op $args* (by evm_kdecide) (by evm_ov))
        | _ => Macro.throwUnsupported
      return acc
