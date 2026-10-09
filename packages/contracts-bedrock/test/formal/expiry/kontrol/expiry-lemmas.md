Expiry Kontrol lemmas
=====================

Lemmas for the per-message interop expiry proofs in this folder. Each lemma is a `simplification` rule that must
be a theorem of KEVM's semantics; the argument for each is given next to it.

```k
requires "foundry.md"

module EXPIRY-LEMMAS
    imports BOOL
    imports FOUNDRY
    imports INT-SYMBOLIC
```

## Jump destinations of init code with symbolic constructor arguments

`new SafeSend{ value: amount }(payable(from))` (SuperchainETHBridge.refundETH) runs init code
`creationCode +Bytes #buf(32, from)` with a symbolic `from`. KEVM computes the valid jump destinations with

```
#computeValidJumpDests(PGM, I, RESULT, LEN)
  => RESULT                                                       requires I >=Int LEN
  => #computeValidJumpDests(PGM, I +Int 1, RESULT[I <- 1], LEN)    requires I <Int LEN andBool PGM[I] ==Int 91
  => #computeValidJumpDests(PGM, I +Int #widthOpCode(PGM[I]), RESULT, LEN)   otherwise
```

which cannot step once `I` reaches the symbolic bytes, so `JUMP`/`JUMPI` inside the constructor get stuck even
though every jump target lies in the concrete code (KEVM's `jump` rule needs `DEST <Int lengthBytes(DESTS)` and
`DESTS[DEST] ==Int 1`).

Soundness: each step from position `I` only writes bits at indices `>= I` and never changes the length of `RESULT`
(`RESULT[I <- 1]` with `I < LEN = lengthBytes(RESULT)`). So, by induction on `LEN -Int I`, the final bit at any
`0 <= D < I` is `RESULT[D]`, and the final length is `lengthBytes(RESULT)`. Bits at positions `>= I` (which may
depend on the symbolic bytes) are left unevaluated, not assumed. `preserves-definedness` is justified because
`RESULT[D]` is defined for `0 <= D < I <= lengthBytes(RESULT)` and `lengthBytes` is total.

```k
    rule #computeValidJumpDests(_PGM, I, RESULT, _LEN) [ D ] => RESULT [ D ]
      requires 0 <=Int D andBool D <Int I andBool I <=Int lengthBytes(RESULT)
      [simplification, preserves-definedness]

    rule lengthBytes(#computeValidJumpDests(_PGM, _I, RESULT, _LEN)) => lengthBytes(RESULT)
      [simplification, preserves-definedness]
```

The helper `#computeValidJumpDests(PGM, I, RESULT, LEN)` is declared partial, so once the (total) outer function has
been unfolded into it and cannot be fully evaluated, the backend may split on its definedness and produce a branch
with `#Not(#Ceil(...))`, i.e. an execution where the jump-destination table does not exist. That branch is infeasible:
under the invariants the outer function establishes (`LEN == lengthBytes(PGM) == lengthBytes(RESULT)`,
`0 <= I`), every step is defined (`PGM[I]` is only read when `I < LEN`, `RESULT[I <- 1]` only written when
`I < LEN`) and the recursion terminates because `I` strictly increases (`#widthOpCode` is at least 1). The rule
below states that definedness, so the infeasible branch is pruned.

```k
    rule #Ceil(#computeValidJumpDests(PGM, I, RESULT, LEN)) => #Top
      requires 0 <=Int I andBool LEN ==Int lengthBytes(PGM) andBool lengthBytes(RESULT) ==Int LEN
      [simplification]

endmodule
```
