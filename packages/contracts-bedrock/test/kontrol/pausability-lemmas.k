requires "evm.md"
requires "foundry.md"

module PAUSABILITY-LEMMAS
    imports BOOL
    imports FOUNDRY
    // imports INFINITE-GAS
    imports INT-SYMBOLIC
    // imports MAP-SYMBOLIC
    // imports SET-SYMBOLIC

    syntax StepSort ::= Int
                      | Bool
                      | Bytes
                      | Set
 // ------------------------

    syntax KItem ::= runLemma ( StepSort )
                   | doneLemma( StepSort )
 // --------------------------------------
    rule <k> runLemma(T) => doneLemma(T) ... </k>

    // We need to enforce some limit on the length of bytearrays
    // and indices into bytearrays in order to avoid chop-reasoning
    syntax Int ::= "maxBytesLength" [alias]
    rule maxBytesLength => 9223372036854775808

    //
    // Arithmetic
    //

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

    //
    // lengthBytes
    //

    // Size of paddings

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
    rule #buf ( WIDTH , #asWord ( BA:Bytes ) ) => BA
      requires lengthBytes(BA) ==K WIDTH
      [simplification]

    //
    // #asWord
    //

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
    rule #asWord ( #range ( #buf (SIZE, X), START, WIDTH) ) => X
      requires 0 <=Int SIZE andBool 0 <=Int X andBool 0 <=Int START andBool 0 <=Int WIDTH
       andBool SIZE ==Int START +Int WIDTH
       andBool X <Int 2 ^Int (8 *Int WIDTH)
      [simplification, concrete(SIZE, START, WIDTH)]

    //
    // #asInteger
    //

    // Conversion from bytes always yields a non-negative integer
    rule 0 <=Int #asInteger ( _ ) => true [simplification]

    //
    // #padRightToWidth
    //

    rule #padRightToWidth (W, X) => X +Bytes #buf(W -Int lengthBytes(X), 0)
      [concrete(W), simplification]

    //
    // #range(M, START, WIDTH)
    //

    // Parameter equality
    rule { #range (A, B, C) #Equals #range (A, B, D) } => #Top
      requires C ==Int D
      [simplification]

    //
    // Bytes indexing and update
    //

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

    //
    // SUMMARIES
    //

    // This rule cannot be used without the [symbolic] tag because it uses
    // "existentials", which is not correct, it uses variables that are learnt
    // from the requires and not from the structure

    // copy-memory-to-memory
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
