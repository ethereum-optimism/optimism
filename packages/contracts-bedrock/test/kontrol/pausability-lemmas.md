Kontrol Lemmas
==============

Lemmas are K rewrite rules that enhance the reasoning power of Kontrol. For more information on lemmas, consult [this section](https://docs.runtimeverification.com/kontrol/guides/advancing-proofs) of the Kontrol documentation.

This file contains the necessary lemmas to run the proofs included in the [proofs](./proofs) folder. Similarly to other files such as [`cheatcodes.md`](https://github.com/runtimeverification/kontrol/blob/master/src/kontrol/kdist/cheatcodes.md) in Kontrol, an idiomatic way of programming in K is with [literate programming](https://en.wikipedia.org/wiki/Literate_programming), allowing to better document the code.

## Imports

For writing the lemmas we use the [`foundry.md`](https://github.com/runtimeverification/kontrol/blob/master/src/kontrol/kdist/foundry.md) file. This file contains and imports all necessary definitions to write the lemmas.

```k
requires "foundry.md"

module PAUSABILITY-LEMMAS
    imports BOOL
    imports FOUNDRY
    imports INT-SYMBOLIC
```

## Arithmetic

Lemmas on arithmetic reasoning.

```k
    // Cancellativity #1
    rule A +Int ( (B -Int A) +Int C ) => B +Int C [simplification]

    // Cancellativity #2
    rule (A -Int B) -Int (C -Int B) => A -Int C [simplification]

    // Cancellativity #3
    rule A -Int (A +Int B) => 0 -Int B [simplification]

    // Various inequalities
    rule X  <Int A &Int B => true requires X <Int 0 andBool 0 <=Int A andBool 0 <=Int B [concrete(X), simplification]
    rule X  <Int A +Int B => true requires X <Int 0 andBool 0 <=Int A andBool 0 <=Int B [concrete(X), simplification]
    rule X <=Int A +Int B => true requires X <Int 0 andBool 0 <=Int A andBool 0 <=Int B [concrete(X), simplification]

    // Upper bound on (pow256 - 32) &Int lengthBytes(X)
    rule notMaxUInt5 &Int Y <=Int Y => true
      requires 0 <=Int Y
      [simplification]

    // Bounds on notMaxUInt5 &Int ( X +Int 31 )
    rule X <=Int   notMaxUInt5 &Int ( X +Int 31 )          => true requires 0 <=Int X                   [simplification]
    rule X <=Int   notMaxUInt5 &Int ( Y +Int 31 )          => true requires X <=Int 0 andBool 0 <=Int Y [simplification, concrete(X)]
    rule X <=Int ( notMaxUInt5 &Int ( X +Int 31 ) ) +Int Y => true requires 0 <=Int X andBool 0 <=Int Y [simplification, concrete(Y)]

    rule notMaxUInt5 &Int X +Int 31 <Int Y => true requires 0 <=Int X andBool X +Int 32 <=Int Y [simplification, concrete(Y)]

    rule notMaxUInt5 &Int X +Int 31 <Int X +Int 32 => true requires 0 <=Int X [simplification]

    //
    // #buf
    //

    // Invertibility of #buf and #asWord
    // TODO: remove once the KEVM PR is merged
    //rule #buf ( WIDTH , #asWord ( BA:Bytes ) ) => BA
    //  requires lengthBytes(BA) ==K WIDTH
    //  [simplification]
```

## `#asWord`

Lemmas about [`#asWord`](https://github.com/runtimeverification/evm-semantics/blob/master/kevm-pyk/src/kevm_pyk/kproj/evm-semantics/evm-types.md#bytes-helper-functions). `#asWord` will interpret a stack of bytes as a single word (with MSB first).

```k
    // Move to function parameters
    rule { #asWord ( X ) #Equals #asWord ( Y ) } => #Top
      requires X ==K Y
      [simplification]

    // #asWord ignores leading zeros
    rule #asWord ( BA1 +Bytes BA2 ) => #asWord ( BA2 )
      requires #asInteger(BA1) ==Int 0
      [simplification, concrete(BA1)]

    // Equality and #range
    rule #asWord ( #range ( #buf ( 32 , _X:Int ) , S:Int , W:Int ) ) ==Int Y:Int => false
        requires S +Int W <=Int 32
         andBool (2 ^Int (8 *Int W)) <=Int Y
        [concrete(S, W, Y), simplification]

    // #asWord is equality
    // TODO: remove once the KEVM PR is merged
    //rule #asWord ( #range ( #buf (SIZE, X), START, WIDTH) ) => X
    //  requires 0 <=Int SIZE andBool 0 <=Int X andBool 0 <=Int START andBool 0 <=Int WIDTH
    //   andBool SIZE ==Int START +Int WIDTH
    //   andBool X <Int 2 ^Int (8 *Int WIDTH)
    //  [simplification, concrete(SIZE, START, WIDTH)]
```

## `#asInteger`

Lemmas about [`#asInteger`](https://github.com/runtimeverification/evm-semantics/blob/master/kevm-pyk/src/kevm_pyk/kproj/evm-semantics/evm-types.md#bytes-helper-functions). `#asInteger` will interperet a stack of bytes as a single arbitrary-precision integer (with MSB first).

```k
    // Conversion from bytes always yields a non-negative integer
    rule 0 <=Int #asInteger ( _ ) => true [simplification]
```

## `#padRightToWidth`

Lemmas about [`#padRightToWidth`](https://github.com/runtimeverification/evm-semantics/blob/master/kevm-pyk/src/kevm_pyk/kproj/evm-semantics/evm-types.md#bytes-helper-functions). `#padToWidth(N, WS)` and `#padRightToWidth` make sure that a Bytes is the correct size.

```k
    rule #padRightToWidth (W, X) => X +Bytes #buf(W -Int lengthBytes(X), 0)
      [concrete(W), simplification]
```

## `#range(M, START, WIDTH)`

Lemmas about [`#range(M, START, WIDTH)`](https://github.com/runtimeverification/evm-semantics/blob/master/kevm-pyk/src/kevm_pyk/kproj/evm-semantics/evm-types.md#bytes-helper-functions). `#range(M, START, WIDTH)` access the range of `M` beginning with `START` of width `WIDTH`.

```k
    // Parameter equality
    rule { #range (A, B, C) #Equals #range (A, B, D) } => #Top
      requires C ==Int D
      [simplification]
```

## Bytes indexing and update

```k
    rule B:Bytes [ X:Int ] => #asWord ( #range (B, X, 1) )
      requires X <=Int lengthBytes(B)
      [simplification(40)]

    // Empty update has no effect
    rule B:Bytes [ START:Int := b"" ] => B
      requires 0 <=Int START andBool START <=Int lengthBytes(B)
      [simplification]

    // Update of tail
    rule ( B1:Bytes +Bytes B2:Bytes ) [ S:Int := B ] => B1 +Bytes ( B2 [ S -Int lengthBytes(B1) := B ] )
      requires lengthBytes(B1) <=Int S
      [simplification]

    // Consecutive quasi-contiguous byte-array update
    rule B [ S1 := B1 ] [ S2 := B2 ] => B [ S1 := #range(B1, 0, S2 -Int S1) +Bytes B2 ]
      requires 0 <=Int S1 andBool S1 <=Int S2 andBool S2 <=Int S1 +Int lengthBytes(B1)
      [simplification]

    // Parameter equality: byte-array update
    rule { B1:Bytes [ S1:Int := B2:Bytes ] #Equals B3:Bytes [ S2:Int := B4:Bytes ] } => #Top
      requires B1 ==K B3 andBool S1 ==Int S2 andBool B2 ==K B4
      [simplification]
```

Summaries
---------

Summary functions are rewrite rules that encapsulate the effects of executing a function. Thus, instead of executing the function itself, Kontrol will just apply the summary rule.

## `copy_memory_to_memory` summary

The following rule is a summarization of the `copy_memory_to_memory` function. This function is automatically generated by the Solidity compiler. In it's Yul form, it is as follows:

```solidity
function copy_memory_to_memory(src, dst, length) {
  let i := 0
  for { } lt(i, length) { i := add(i, 32) }
  {
    mstore(add(dst, i), mload(add(src, i)))
  }
  if gt(i, length)
  {
    // clear end
    mstore(add(dst, length), 0)
  }
}
```

It is used to copy `length` bytes of memory from index `src` to index `dest`, doing so in steps of 32 bytes, and right-padding with zeros to a multiple of 32.

We need to enforce some limit on the length of bytearrays and indices into bytearrays in order to avoid chop-reasoning.

```k
    syntax Int ::= "maxBytesLength" [alias]
    rule maxBytesLength => 9223372036854775808
```


This rule cannot be used without the `[symbolic]` tag because it uses "existentials", which is not correct, it uses variables that are learnt from the requires and not from the structure.

```k
    rule [copy-memory-to-memory-summary]:
      <k> #execute ... </k>
      <useGas> false </useGas>
      <schedule> SHANGHAI </schedule>
      <jumpDests> JUMPDESTS </jumpDests>
      // The program and program counter are symbolic, focusing on the part we will be executing (CP)
      <program> PROGRAM </program>
      <pc> PCOUNT => PCOUNT +Int 53 </pc>
      // The word stack has the appropriate form, as per the compiled code
      <wordStack> LENGTH : _ : SRC : DEST : WS </wordStack>
      // The program copies LENGTH bytes of memory from SRC +Int 32 to DEST +Int OFFSET,
      // padded with 32 zeros in case LENGTH is not divisible by 32
      <localMem>
        LM => LM [ DEST +Int 32 := #range ( LM, SRC +Int 32, LENGTH ) +Bytes
                                   #buf ( ( ( notMaxUInt5 &Int ( LENGTH +Int maxUInt5 ) ) -Int LENGTH ) , 0 ) +Bytes
                                   #buf ( ( ( ( 32 -Int ( ( notMaxUInt5 &Int ( LENGTH +Int maxUInt5 ) ) -Int LENGTH ) ) ) modInt 32 ), 0 ) ]
      </localMem>
      requires
       // The current program we are executing differs from the original one only in the hardcoded jump addresses,
       // which are now relative to PCOUNT, and the hardcoded offset, which is now symbolic.
               #range(PROGRAM, PCOUNT, 53) ==K b"`\x00[\x81\x81\x10\x15b\x00\x81`W` \x81\x85\x01\x81\x01Q\x86\x83\x01\x82\x01R\x01b\x00\x81BV[\x81\x81\x11\x15b\x00\x81sW`\x00` \x83\x87\x01\x01R[P"
                                               [ 08 := #buf(3, PCOUNT +Int 32) ]
                                               [ 28 := #buf(3, PCOUNT +Int  2) ]
                                               [ 38 := #buf(3, PCOUNT +Int 51) ]

       // Various well-formedness constraints. In particular, the maxBytesLength-related ones are present to
       // remove various chops that would otherwise creep into the execution, and are reasonable since byte
       // arrays in actual programs would never reach that size.
       andBool 0 <=Int PCOUNT
       andBool 0 <=Int LENGTH andBool LENGTH <Int maxBytesLength
       andBool 0 <=Int SRC    andBool SRC    <Int maxBytesLength
       andBool 0 <=Int DEST   andBool DEST   <Int maxBytesLength
       andBool #sizeWordStack(WS) <=Int 1015

       andBool SRC +Int LENGTH <=Int DEST // No overlap between source and destination
       andBool DEST <=Int lengthBytes(LM) // Destination starts within current memory
       // All JUMPDESTs in the program are valid
       andBool (PCOUNT +Int 2) in JUMPDESTS andBool (PCOUNT +Int 32) in JUMPDESTS andBool (PCOUNT +Int 51) in JUMPDESTS
       andBool PCOUNT +Int 51 <Int 2 ^Int 16  // and fit into two bytes
      [priority(30), concrete(JUMPDESTS, PROGRAM, PCOUNT), preserves-definedness]

endmodule
```
