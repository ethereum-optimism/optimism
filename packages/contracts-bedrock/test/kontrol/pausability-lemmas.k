requires "evm.md"
requires "foundry.md"

module PAUSABILITY-LEMMAS [symbolic]
    imports BOOL
    imports FOUNDRY
    imports INFINITE-GAS
    imports INT-SYMBOLIC
    imports MAP-SYMBOLIC
    imports SET-SYMBOLIC

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

    rule ((X up/Int Y) *Int Y) <Int Z => true
      requires X <Int ((Z -Int Y) +Int 1)
      [simplification]

    rule ((X up/Int 32) *Int 32) <Int X => false
      [simplification]

    rule A modInt B => A
      requires 0 <=Int A andBool A <Int B
      [simplification]

    //
    // These slow the execution down considerably
    //
    // rule   A <=Int B                =>   A  ==Int  B   requires B <=Int A [simplification, concrete(A)]
    // rule { A <=Int B #Equals true } => { A #Equals B } requires B <=Int A [simplification, concrete(A)]
    // rule { true #Equals A <=Int B } => { A #Equals B } requires B <=Int A [simplification, concrete(A)]

    rule (   A +Int B          ) -Int B => A        [simplification]
    rule ( ( A +Int B ) +Int C ) -Int B => A +Int C [simplification]

    rule ( ( A +Int B ) +Int ( C +Int D ) ) => ( A +Int C +Int ( B +Int D ) ) [simplification, concrete(B, D)]

    rule A +Int B <=Int C +Int D => A <=Int C +Int (D -Int B) requires D >=Int B [simplification(60), concrete(B, D)]
    rule A +Int B <=Int C +Int D => A +Int (B -Int D) <=Int C requires B  <Int D [simplification(60), concrete(B, D)]

    rule A ==Int B *Int C => A /Int C ==Int B
        requires A modInt C ==Int 0
        [simplification, concrete(A, C)]

    //
    // Bool
    //

    rule false ==Bool X => notBool X [simplification]
    rule X ==Bool false => notBool X [simplification]

    rule   notBool X ==Bool  notBool Y   =>   X ==Bool  Y   [simplification]
    rule { notBool X #Equals notBool Y } => { X #Equals Y } [simplification]

    rule bool2Word ( X ) => 1 requires X         [simplification]
    rule bool2Word ( X ) => 0 requires notBool X [simplification]

    rule   bool2Word ( X )  ==Int  bool2Word ( Y )   =>   X ==Bool  Y   [simplification]
    rule { bool2Word ( X ) #Equals bool2Word ( Y ) } => { X #Equals Y } [simplification]

    //
    // ML
    //

    rule { true  #Equals X     ==K   Y     } => { X #Equals Y } [simplification]
    rule { true  #Equals X:Int ==Int Y:Int } => { X #Equals Y } [simplification]
    rule { false #Equals X     ==K   Y     } => #Not ( { X #Equals Y } ) [simplification]
    rule { false #Equals X:Int ==Int Y:Int } => #Not ( { X #Equals Y } ) [simplification]

    rule { true  #Equals notBool X:Bool } => { false #Equals X } [simplification]
    rule { false #Equals notBool X:Bool } => { true  #Equals X } [simplification]

    rule { X     ==K   Y     #Equals true  } => { X #Equals Y } [simplification]
    rule { X:Int ==Int Y:Int #Equals true  } => { X #Equals Y } [simplification]
    rule { X     ==K   Y     #Equals false } => #Not ( { X #Equals Y } ) [simplification]
    rule { X:Int ==Int Y:Int #Equals false } => #Not ( { X #Equals Y } ) [simplification]

    rule { notBool X:Bool #Equals true  } => { false #Equals X } [simplification]
    rule { notBool X:Bool #Equals false } => { true  #Equals X } [simplification]

    rule #Not ( { X #Equals 0 } ) => { X #Equals 1 } requires #rangeBool ( X ) [simplification]
    rule #Not ( { X #Equals 1 } ) => { X #Equals 0 } requires #rangeBool ( X ) [simplification]

    //
    // &Int
    //

    // Commutativity
    rule   A &Int B  ==Int  B &Int A   => true [simplification, smt-lemma]
    rule { A &Int B #Equals B &Int A } => #Top [simplification]

    // Distributivity of &Int and |Int
    rule A &Int (B |Int C) => (A &Int B) |Int (A &Int C)
    [concrete(A, B), simplification]

    rule A &Int (B |Int C) => (A &Int B) |Int (A &Int C)
    [concrete(A, C), simplification]

    // &Int on non-negative integers remains non-negative
    rule 0 <=Int (X &Int Y) => true
      requires 0 <=Int X
       andBool 0 <=Int Y
      [simplification, smt-lemma]

    // Result of &Int cannot be greater than the operands
    rule (X &Int Y) <=Int Z => true
      requires 0 <=Int X
       andBool 0 <=Int Y
       andBool (X <=Int Z orBool Y <=Int Z)
      [simplification]

    // Anything negative is <Int than &Int
    rule A <Int X &Int Y => true
      requires 0 <=Int X andBool 0 <=Int Y
       andBool A <Int 0
      [simplification, concrete(A)]

    // Deconstruction of (maxUInt &Int ...)
    rule maxUInt8   &Int #asWord ( BA ) => #asWord ( #range(BA, 31,  1) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt16  &Int #asWord ( BA ) => #asWord ( #range(BA, 30,  2) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt24  &Int #asWord ( BA ) => #asWord ( #range(BA, 29,  3) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt32  &Int #asWord ( BA ) => #asWord ( #range(BA, 28,  4) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt40  &Int #asWord ( BA ) => #asWord ( #range(BA, 27,  5) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt48  &Int #asWord ( BA ) => #asWord ( #range(BA, 26,  6) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt56  &Int #asWord ( BA ) => #asWord ( #range(BA, 25,  7) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt64  &Int #asWord ( BA ) => #asWord ( #range(BA, 24,  8) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt72  &Int #asWord ( BA ) => #asWord ( #range(BA, 23,  9) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt80  &Int #asWord ( BA ) => #asWord ( #range(BA, 22, 10) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt88  &Int #asWord ( BA ) => #asWord ( #range(BA, 21, 11) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt96  &Int #asWord ( BA ) => #asWord ( #range(BA, 20, 12) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt104 &Int #asWord ( BA ) => #asWord ( #range(BA, 19, 13) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt112 &Int #asWord ( BA ) => #asWord ( #range(BA, 18, 14) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt120 &Int #asWord ( BA ) => #asWord ( #range(BA, 17, 15) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt128 &Int #asWord ( BA ) => #asWord ( #range(BA, 16, 16) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt136 &Int #asWord ( BA ) => #asWord ( #range(BA, 15, 17) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt144 &Int #asWord ( BA ) => #asWord ( #range(BA, 14, 18) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt152 &Int #asWord ( BA ) => #asWord ( #range(BA, 13, 19) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt160 &Int #asWord ( BA ) => #asWord ( #range(BA, 12, 20) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt168 &Int #asWord ( BA ) => #asWord ( #range(BA, 11, 21) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt176 &Int #asWord ( BA ) => #asWord ( #range(BA, 10, 22) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt184 &Int #asWord ( BA ) => #asWord ( #range(BA,  9, 23) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt192 &Int #asWord ( BA ) => #asWord ( #range(BA,  8, 24) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt200 &Int #asWord ( BA ) => #asWord ( #range(BA,  7, 25) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt208 &Int #asWord ( BA ) => #asWord ( #range(BA,  6, 26) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt216 &Int #asWord ( BA ) => #asWord ( #range(BA,  5, 27) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt224 &Int #asWord ( BA ) => #asWord ( #range(BA,  4, 28) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt232 &Int #asWord ( BA ) => #asWord ( #range(BA,  3, 29) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt240 &Int #asWord ( BA ) => #asWord ( #range(BA,  2, 30) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt248 &Int #asWord ( BA ) => #asWord ( #range(BA,  1, 31) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule maxUInt256 &Int #asWord ( BA ) => #asWord ( #range(BA,  0, 32) ) requires lengthBytes(BA) ==Int 32 [simplification]

    // Deconstruction of (notMaxUInt &Int ...)
    rule notMaxUInt8   &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 31) +Bytes #buf (  1, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt16  &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 30) +Bytes #buf (  2, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt32  &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 28) +Bytes #buf (  4, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt64  &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 24) +Bytes #buf (  8, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt96  &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 20) +Bytes #buf ( 12, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt128 &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 16) +Bytes #buf ( 16, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt160 &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 12) +Bytes #buf ( 20, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt192 &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 8)  +Bytes #buf ( 24, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt208 &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 6)  +Bytes #buf ( 26, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt224 &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 4)  +Bytes #buf ( 28, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt240 &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 2)  +Bytes #buf ( 30, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]
    rule notMaxUInt248 &Int #asWord ( BA ) => #asWord ( #range(BA, 0, 1)  +Bytes #buf ( 31, 0 ) ) requires lengthBytes(BA) ==Int 32 [simplification]

    // Irrelevance of lower bits
    rule notMaxUInt8   &Int (X |Int (maxUInt8   &Int _)) => notMaxUInt8   &Int X [simplification]
    rule notMaxUInt16  &Int (X |Int (maxUInt16  &Int _)) => notMaxUInt16  &Int X [simplification]
    rule notMaxUInt32  &Int (X |Int (maxUInt32  &Int _)) => notMaxUInt32  &Int X [simplification]
    rule notMaxUInt64  &Int (X |Int (maxUInt64  &Int _)) => notMaxUInt64  &Int X [simplification]
    rule notMaxUInt96  &Int (X |Int (maxUInt96  &Int _)) => notMaxUInt96  &Int X [simplification]
    rule notMaxUInt128 &Int (X |Int (maxUInt128 &Int _)) => notMaxUInt128 &Int X [simplification]
    rule notMaxUInt160 &Int (X |Int (maxUInt160 &Int _)) => notMaxUInt160 &Int X [simplification]
    rule notMaxUInt192 &Int (X |Int (maxUInt192 &Int _)) => notMaxUInt192 &Int X [simplification]
    rule notMaxUInt208 &Int (X |Int (maxUInt208 &Int _)) => notMaxUInt208 &Int X [simplification]
    rule notMaxUInt224 &Int (X |Int (maxUInt224 &Int _)) => notMaxUInt224 &Int X [simplification]
    rule notMaxUInt240 &Int (X |Int (maxUInt240 &Int _)) => notMaxUInt240 &Int X [simplification]
    rule notMaxUInt248 &Int (X |Int (maxUInt248 &Int _)) => notMaxUInt248 &Int X [simplification]

    // &Int yields zero for notMax and operand in appropriate range
    rule [bitwise-and-zero]:
      X &Int Y => 0
      requires #rangeUInt(256, X)
       andBool pow256 -Int X ==Int 2 ^Int log2Int(pow256 -Int X)
       andBool 0 <=Int Y andBool Y <Int 2 ^Int log2Int(pow256 -Int X)
       [concrete(X), simplification, comm]

    // Deconstruction of <<Int into #buf
    rule X <<Int Y => #asWord ( #buf ( 32 -Int (Y /Int 8) , X ) +Bytes #buf ( Y /Int 8 , 0 ) )
      requires 0 <=Int X andBool X <Int 2 ^Int (256 -Int Y)
       andBool 0 <=Int Y andBool Y <=Int 256 andBool Y modInt 8 ==Int 0
      [simplification, concrete(Y)]

    rule 0 <=Int (X &Int Y) +Int Z => true
      requires 0 <=Int X &Int Y
       andBool 0 <=Int Z
      [simplification]

    rule Z <Int X &Int Y => false
      requires #rangeUInt(256, X)
       andBool #rangeUInt(256, Y)
       andBool #rangeUInt(256, Z)
       andBool ((Y <Int Z) orBool (X <Int Z))
      [simplification]

    rule X &Int Y <Int Z => true
      requires #rangeUInt(256, X)
       andBool #rangeUInt(256, Y)
       andBool #rangeUInt(256, Z)
       andBool ((Y <Int Z) orBool (X <Int Z))
      [simplification]

    rule X &Int #asWord ( _Y +Bytes Z ) => X &Int #asWord ( Z )
      requires X <Int 2 ^Int (8 *Int lengthBytes(Z))
      [concrete(X, Z), simplification]

    rule X &Int #asWord ( _ +Bytes Z ) >>Int T => X &Int #asWord ( Z ) >>Int T
      requires X <Int 2 ^Int (8 *Int lengthBytes(Z) -Int T)
      [concrete(X, Z, T), simplification]

    // Moving from &Int to modInt
    rule 1 &Int X => X modInt 2 [simplification]

    //
    // |Int
    //

    // Commutativity
    rule   A |Int B  ==Int  B |Int A   => true [simplification, smt-lemma]
    rule { A |Int B #Equals B |Int A } => #Top [simplification]

    // Non-zeroedness of |Int
    rule X |Int _ ==Int 0 => false
      requires 0 <Int X
      [simplification]

    // |Int distributivity over #asWord and +Bytes, v1
    rule A |Int #asWord ( BA1 +Bytes BA2 ) =>
      #asWord ( BA1 +Bytes #buf ( lengthBytes(BA2), A |Int #asWord ( BA2 ) ) )
      requires A <Int 2 ^Int (8 *Int lengthBytes(BA2))
      [concrete(A), simplification]

    // |Int distributivity over #asWord and +Bytes, v2
    rule A |Int #asWord ( BA1 +Bytes BA2 ) =>
      #asWord (
        #buf ( lengthBytes(BA1), (A >>Int (8 *Int lengthBytes(BA2))) |Int #asWord ( BA1 ) )
        +Bytes
        #buf ( lengthBytes(BA2), (A modInt (2 ^Int (8 *Int lengthBytes(BA2)))) |Int #asWord ( BA2 ) )
      )
      requires 0 <=Int A
      [simplification(40), concrete(A, BA1)]

    // Prepend 4 bytes (used for function selectors)
    rule A |Int #asWord ( BUF ) => #asWord ( #range ( #buf ( 32 , A ) , 0 , 4 ) +Bytes BUF )
        requires notMaxUInt224 &Int A ==Int A
         andBool lengthBytes ( BUF ) ==Int 28
        [simplification, concrete(A)]

    // Irrelevance of lower bits for >>Int
    rule (X |Int Y) >>Int Z => X >>Int Z
      requires 0 <=Int Z
       andBool 0 <=Int Y andBool Y <Int 2 ^Int Z
       andBool X &Int (maxUInt256 -Int (2 ^Int Z -Int 1)) ==Int X
      [simplification, concrete(Z)]

    //
    // xorInt
    //

    // Commutativity
    rule   A xorInt B  ==Int  B xorInt A   => true [simplification, smt-lemma]
    rule { A xorInt B #Equals B xorInt A } => #Top [simplification]

    // Non-negativity of xorInt
    rule [bitwise-xor-geq-zero]:
      0 <=Int (A xorInt B) => true
      requires 0 <=Int A andBool 0 <=Int B
      [simplification]

    // xorInt
    rule [bitwise-xor-lt-pow256]:
      (A xorInt B) <Int pow256 => true
      requires 0 <=Int A andBool A <Int pow256
       andBool 0 <=Int B andBool B <Int pow256
      [simplification]

    // xor in terms of -Int
    rule X xorInt maxUInt256 => maxUInt256 -Int X
      requires #rangeUInt ( 256 , X )
      [simplification]

    //
    // Arithmetic
    //

    // Reflexivity of <=Int
    rule A <=Int A => true [simplification]

    // Cancellativity #1
    rule A +Int B -Int B +Int C => A +Int C [simplification]

    // Cancellativity #2
    rule A -Int B +Int C -Int D +Int B +Int E => A -Int D +Int C +Int E [simplification]

    // Cancellativity #3
    rule ( A +Int B ) +Int C <=Int ( D +Int B ) +Int E => A +Int C <=Int D +Int E [simplification]

    // Cancellativity #4
    rule A +Int B <=Int ( A +Int C ) +Int D => B <=Int C +Int D [simplification]

    // Cancellativity #5
    rule A +Int ( (B -Int A) +Int C ) => B +Int C [simplification]

    // Cancellativity #6
    rule (A -Int B) -Int (C -Int B) => A -Int C [simplification]

    // Distributivity of minInt against +Int
    rule minInt ( A:Int +Int B:Int, C:Int +Int B:Int ) => minInt ( A, C ) +Int B [simplification]

    // Definition of maxInt
    rule maxInt(I1:Int, I2:Int) => I2 requires I1 <=Int I2 [simplification]
    rule maxInt(I1:Int, I2:Int) => I1 requires I1  >Int I2 [simplification]

    // Maximum is not greater than if both operands are not greater than
    rule maxInt(A:Int, B:Int) <=Int X:Int => A <=Int X andBool B <=Int X [simplification]

    // Cutting impossible branches
    rule { A:Int #Equals B:Int *Int X:Int +Int C:Int } => #Bottom
      requires A  <Int C
       andBool 0 <=Int B
       andBool 0 <=Int X
       [simplification]

    // Custom distributivity simplification
    rule 32 *Int ( A:Int +Int B:Int ) => 32 *Int A +Int 32 *Int B
      [simplification]

    // Matching resolutions

    rule { 32 *Int A:Int +Int X:Int #Equals 32 *Int B:Int +Int Y:Int } =>
         { B #Equals A +Int 1 }
         requires X -Int Y ==Int 32
         [simplification]

    rule { A:Int *Int X:Int +Int B #Equals A *Int Y:Int +Int B } =>
         { X #Equals Y }
         requires A =/=Int 0
         [simplification]

    rule { B #Equals A *Int Y:Int +Int B } =>
         { 0 #Equals Y }
         requires A =/=Int 0
         [simplification]

    // Chop custom simplification
    rule chop ( lengthBytes ( X ) +Int 115792089237316195423570985008687907853269984665640564039457584007913129640009 ) =>
         lengthBytes ( X ) +Int 73
         [simplification]

    // Chop custom simplification
    rule 0 ==Int chop (A:Int +Int B:Int) => A ==Int pow256 -Int B
      requires 0 <Int B
       andBool #rangeUInt(256, A)
       [concrete(B), simplification]

    rule { 0 #Equals chop (A:Int +Int B:Int) } => { A #Equals pow256 -Int B }
      requires 0 <Int B
       andBool #rangeUInt(256, A)
       [concrete(B), simplification]

    // Extremely specialised simplification
    rule A ==Int B => true
      requires 32 *Int A -Int 9 <Int 32 *Int B +Int 23
       andBool 32 *Int B -Int 9 <Int 32 *Int A +Int 23
      [simplification, smt-lemma]

    // FIXME: This should be provable by SMT but isn't
    rule 0 ==Int X -Int #asWord ( #range ( _ , _ , Y ) ) *Int Z => false
      requires ( 2 ^Int (Y *Int 8) ) *Int Z <Int X
      [concrete(X, Y), simplification]

    rule { 0 #Equals X -Int #asWord ( #range ( _ , _ , Y ) ) *Int Z } => #Bottom
      requires ( 2 ^Int (Y *Int 8) ) *Int Z <Int X
      [concrete(X, Y), simplification]

    // Range bit-shift arithmetic
    rule 0 <=Int (A <<Int X) => true
      requires 0 <=Int A
       andBool 0 <=Int X
      [simplification]

    // Lukso-specific bit-shift arithmetic
    rule pow96 <=Int (A <<Int 96) => true
      requires 0 <=Int A
      [simplification]

    // Range bit-shift arithmetic
    rule (A <<Int X) <Int pow256 => true
      requires A <Int 2 ^Int (256 -Int X)
      [simplification]

    //
    // Specialised chop-range
    //

    rule A <=Int B +Int C => true
      requires A <=Int 0
       andBool 0 <=Int B
       andBool 0 <=Int C
       [simplification, concrete(A)]

    rule A +Int B <Int C => true
      requires A <Int C /Int 2
       andBool B <Int C /Int 2
       [simplification, concrete(C)]

    rule A <Int B +Int C => true
      requires A <=Int 0
       andBool 0 <=Int B andBool 0 <=Int C
       andBool (0 <Int B  orBool 0 <Int C)
       [simplification, concrete(A)]

    //
    // Bytes
    //

    rule A1 +Bytes B1 ==K A2 +Bytes B2 => false
      requires #range(A1, 0, minInt(lengthBytes(A1), lengthBytes(A2)))
                 =/=K
               #range(A2, 0, minInt(lengthBytes(A1), lengthBytes(A2)))
       andBool lengthBytes(A1 +Bytes B1) ==Int lengthBytes(A2 +Bytes B2)
      [concrete(A1, A2), simplification]

    rule A +Bytes (B +Bytes C) => (A +Bytes B) +Bytes C
      [concrete(A, B), simplification]

    rule (A +Bytes B) +Bytes C => A +Bytes (B +Bytes C)
      [concrete(B, C), simplification]

    rule A +Bytes (B +Bytes C) +Bytes D => (A +Bytes B) +Bytes C +Bytes D
      [concrete(A, B), simplification]

    rule (A +Bytes B) +Bytes (C +Bytes D) => A +Bytes (B +Bytes C) +Bytes D
      [concrete(B, C), simplification]

    //
    // lengthBytes
    //

    // Size of paddings
    rule lengthBytes ( padRightBytes ( BA:Bytes, WIDTH, _ ) ) =>
      lengthBytes(BA) +Int WIDTH
      requires 0 <=Int WIDTH
      [simplification]

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

    // Specialised simplification
    rule notMaxUInt5 &Int ( notMaxUInt5 &Int lengthBytes ( VV6__data_3c5818c8:Bytes ) +Int 31 ) +Int 63 => ( notMaxUInt5 &Int lengthBytes ( VV6__data_3c5818c8:Bytes ) +Int 31 ) +Int 32
      [simplification]

    //
    // #buf
    //

    // Invertibility of #buf and #asWord
    rule #buf ( WIDTH , #asWord ( BA:Bytes ) ) => BA
      requires lengthBytes(BA) ==K WIDTH
      [simplification]

    // Local injectivity for 8 bits
    rule { #buf ( 8 , A:Int ) #Equals #buf ( 8 , B:Int ) } => { A #Equals B }
    requires 0 <=Int A andBool A <Int pow64
     andBool 0 <=Int B andBool B <Int pow64
     [simplification]

    // Local injectivity for 32 bits
    rule { #buf ( 32 , A:Int ) #Equals #buf ( 32 , B:Int ) } => { A #Equals B }
    requires 0 <=Int A andBool A <Int pow256
     andBool 0 <=Int B andBool B <Int pow256
     [simplification]

    rule { #buf ( N, X ) #Equals #buf ( N, Y ) } => { X #Equals Y }
      requires 0 <=Int N
       andBool 0 <=Int X
       andBool 0 <=Int Y
       andBool X <Int 2 ^Int ( 8 *Int N )
       andBool Y <Int 2 ^Int ( 8 *Int N )
      [concrete(N), simplification]

    rule #buf ( N, X ) ==K #buf ( N, Y ) => X ==Int Y
      requires 0 <=Int N
       andBool 0 <=Int X
       andBool 0 <=Int Y
       andBool X <Int 2 ^Int ( 8 *Int N )
       andBool Y <Int 2 ^Int ( 8 *Int N )
      [concrete(N), simplification]

    //
    // #asWord
    //

    // Move to function parameters
    rule { #asWord ( X ) #Equals #asWord ( Y ) } => #Top
      requires X ==K Y
      [simplification]

    // Bounds: inherited from KEVM with smt-lemma
    rule 0 <=Int #asWord(_BA)             => true [simplification, smt-lemma]
    rule         #asWord(_BA) <Int pow256 => true [simplification, smt-lemma]

    // Invertibility of #buf with #asWord
    rule #buf ( 32 , #asWord ( X ) ) => X requires lengthBytes ( X ) ==Int 32 [simplification]
    rule #asWord ( #buf ( 32 , X ) ) => X requires #rangeUInt ( 256 , X )     [simplification]

    // #asWord ignores leading zeros
    rule #asWord ( BA1 +Bytes BA2 ) => #asWord ( BA2 )
      requires #asInteger(BA1) ==Int 0
      [simplification, concrete(BA1)]

    // <Int via ByteArray length
    rule #asWord ( BA ) <Int X => true
        requires 0 <Int X andBool lengthBytes( BA ) <=Int log2Int( X ) divInt 8
        [simplification, concrete(X)]

    // #asWord is not equal to the target if the leading bytes alone are greater than the target
    // or the trailing bytes are smaller than the difference between the target and leading bytes
    rule #asWord ( BA1 +Bytes BA2 ) ==Int X => false
      requires #asWord ( BA1 ) <<Int ( 8 *Int lengthBytes(BA2) ) >Int X
        orBool X -Int #asWord ( BA1 ) <<Int ( 8 *Int lengthBytes(BA2) ) >=Int 2 ^Int ( 8 *Int lengthBytes(BA2) )
      [simplification(40), concrete(BA1)]

    // #asWord is not smaller-or-equal to the target if the leading bytes alone are greater than the target
    rule #asWord ( BA1 +Bytes BA2 ) <=Int X => false
      requires #asWord ( BA1 ) <<Int ( 8 *Int lengthBytes(BA2) ) >Int X
      [simplification, concrete(BA1)]

    // #asWord is smaller-or-equal to the target if the trailing bytes are smaller than the difference between the target and leading bytes
    rule #asWord ( BA1 +Bytes BA2 ) <=Int X => true
      requires X -Int #asWord ( BA1 ) <<Int ( 8 *Int lengthBytes(BA2) ) >=Int 2 ^Int ( 8 *Int lengthBytes(BA2) ) -Int 1
      [simplification, concrete(BA1)]

    // and otherwise is not smaller-or-equal
    rule #asWord ( BA1 +Bytes BA2 ) <=Int X => false
         requires X <Int ( #asWord ( BA1 ) <<Int (8 *Int lengthBytes(BA2)) )
         [concrete(BA1), simplification]

    // Similar for greater-or-equal
    rule X <=Int #asWord ( BA1 +Bytes BA2 ) => true
      requires lengthBytes( BA1 +Bytes BA2 ) ==Int 32
       andBool X <=Int #asWord( BA1 ) <<Int (8 *Int lengthBytes(BA2))
      [concrete(BA1, X), simplification]

    // and smaller
    rule #asWord ( BA1 +Bytes BA2 ) <Int X => true
      requires lengthBytes( BA1 +Bytes BA2) ==Int 32
       andBool X -Int #asWord( BA1 ) <<Int ( 8 *Int lengthBytes(BA2) ) >Int 2 ^Int ( 8 *Int lengthBytes(BA2) ) -Int 1
      [concrete(BA1, X), simplification]

    // and greater
    rule X <Int #asWord ( BA1 +Bytes BA2 ) => true
      requires lengthBytes( BA1 +Bytes BA2 ) ==Int 32
       andBool X <Int #asWord( BA1 ) <<Int (8 *Int lengthBytes(BA2))
      [concrete(BA1, X), simplification]

    // Leveraging concreteness in #asWord equality
    rule #asWord ( A +Bytes B ) ==Int C:Int =>
      #asWord(B) ==Int C -Int ( #asWord(A) *Int (2 ^Int (8 *Int lengthBytes(B)) ) )
      [concrete(A, C), simplification, comm]

    rule { #asWord ( A +Bytes B ) #Equals C } =>
      { #asWord(B) #Equals C -Int ( #asWord(A) *Int (2 ^Int (8 *Int lengthBytes(B)) ) ) }
      [concrete(A, C), simplification, comm]

    rule #asWord ( A +Bytes B ) ==Int C:Int =>
      C -Int #asWord(B) ==Int #asWord(A) *Int (2 ^Int (8 *Int lengthBytes(B)))
      [concrete(B, C), simplification, comm]

    rule { #asWord ( A +Bytes B ) #Equals C:Int } =>
      { C -Int #asWord(B) #Equals #asWord(A) *Int (2 ^Int (8 *Int lengthBytes(B))) }
      [concrete(B, C), simplification, comm]

    // Equality and #range
    rule #asWord ( #range ( #buf ( 32 , _X:Int ) , S:Int , W:Int ) ) ==Int Y:Int => false
        requires S +Int W <=Int 32
         andBool (2 ^Int (8 *Int W)) <=Int Y
        [concrete(S, W, Y), simplification]

    // #asWord is equality
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

    // Consecutive quasi-contiguous byte-array update
    rule B [ S1 := B1 ] [ S2 := B2 ] => B [ S1 := #range(B1, 0, S2 -Int S1) +Bytes B2 ]
      requires 0 <=Int S1 andBool S1 <=Int S2 andBool S2 <=Int S1 +Int lengthBytes(B1)
      [simplification]

    // Parameter equality: byte-array update
    rule { B1:Bytes [ S1:Int := B2:Bytes ] #Equals B3:Bytes [ S2:Int := B4:Bytes ] } => #Top
      requires B1 ==K B3 andBool S1 ==Int S2 andBool B2 ==K B4
      [simplification]

    //
    // #memoryUsageUpdate
    //

    rule { #memoryUsageUpdate ( A , B , C ) #Equals #memoryUsageUpdate ( D , E , F ) } => #Top
      requires A ==Int D andBool B ==Int E andBool C ==Int F
      [simplification]

    //
    // keccak
    //

    // keccak does not equal a concrete value
    rule [keccak-eq-conc-false]: keccak(_A)  ==Int _B => false [symbolic(_A), concrete(_B), simplification, comm]
    rule [keccak-neq-conc-true]: keccak(_A) =/=Int _B => true  [symbolic(_A), concrete(_B), simplification, comm]

    rule [keccak-eq-conc-false-ml]: { keccak(_A) #Equals _B } => #Bottom [symbolic(_A), concrete(_B), simplification, comm]

    // corollary of `keccak-eq-conc-false`
    rule [keccak-eq-conc-false-extended]:
      ( ( keccak ( _X ) +Int A ) modInt pow256 ) ==Int _Y => false
      requires 0 <Int A andBool A <Int pow256
      [simplification, symbolic(_X), concrete(A, _Y)]

    // keccak is injective
    rule [keccak-inj]: keccak(A) ==Int keccak(B) => A ==K B [simplification]

    // keccak has no "fixpoint"
    rule [keccak-no-fix-eq-false]: #buf(32, keccak(X))  ==K X => false [simplification]
    rule [keccak-no-fix-neq-true]: #buf(32, keccak(X)) =/=K X => true  [simplification]

    // disequality of keccak under shifting
    rule ( ( keccak ( _X ) +Int A ) modInt pow256 ) ==Int keccak ( _Y ) => false
      requires 0 <Int A andBool A <Int pow256
      [simplification, symbolic(_X, _Y), concrete(A)]

    // chop of negative keccak
    rule chop (0 -Int keccak(BA)) => pow256 -Int keccak(BA)
        [simplification]

    // keccak cannot equal a number outside of its range
    rule { X #Equals keccak (_) } => #Bottom
      requires X <Int 0 orBool X >=Int pow256
      [concrete(X), simplification]

    // Custom chop simplification
    rule chop (A:Int +Int B:Int) =>  A +Int B
      requires #rangeUInt(64, A)
       andBool #rangeUInt(64, B)
       [simplification]

    // chop is idempotent through +Int
    rule chop ( chop ( X ) +Int Y ) => chop ( X +Int Y ) [simplification]

    //
    // NEW: SUMMARIES
    //

    rule [copy-memory-to-memory-provable-full]:
      <k> #execute ... </k>
      <useGas> false </useGas>
      <schedule> SHANGHAI </schedule>
      <jumpDests> JUMPDESTS </jumpDests>
      // The program and program counter are symbolic, focusing on the part we will be executing (CP)
      <program> PROGRAM </program>
      <pc> PCOUNT => PCOUNT +Int 47 </pc>
      // The word stack has the appropriate form, as per the compiled code
      <wordStack> SRC : DEST : LENGTH : WS </wordStack>
      // The program copies LENGTH bytes of memory from SRC +Int 32 to DEST +Int OFFSET,
      // padded with 32 zeros in case LENGTH is not divisible by 32
      <localMem>
        LM => LM [ DEST := #range ( LM, SRC, LENGTH ) +Bytes
                           #buf ( ( ( notMaxUInt5 &Int ( LENGTH +Int maxUInt5 ) ) -Int LENGTH ) , 0 ) +Bytes
                           #buf ( ( ( ( 32 -Int ( ( notMaxUInt5 &Int ( LENGTH +Int maxUInt5 ) ) -Int LENGTH ) ) ) modInt 32 ), 0 ) ]
      </localMem>
      requires
       // The current program we are executing differs from the original one only in the hardcoded jump addresses,
       // which are now relative to PCOUNT, and the hardcoded offset, which is now symbolic.
               #range(PROGRAM, PCOUNT, 47) ==K b"[`\x00[\x83\x81\x10\x15b\x00]'W\x81\x81\x01Q\x83\x82\x01R` \x01b\x00]\rV[\x83\x81\x11\x15b\x00]7W`\x00\x84\x84\x01R[P"
                                               [ 09 := #buf(3, PCOUNT +Int 29) ]
                                               [ 25 := #buf(3, PCOUNT +Int  3) ]
                                               [ 35 := #buf(3, PCOUNT +Int 45) ]

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
       andBool PCOUNT in JUMPDESTS andBool (PCOUNT +Int 3) in JUMPDESTS andBool (PCOUNT +Int 29) in JUMPDESTS andBool (PCOUNT +Int 45) in JUMPDESTS
       andBool PCOUNT +Int 47 <Int 2 ^Int 16  // and fit into two bytes
      [priority(30), concrete(JUMPDESTS, PROGRAM, PCOUNT), preserves-definedness]

endmodule

module PAUSABILITY-LEMMAS-SPEC
    imports PAUSABILITY-LEMMAS

    claim [program-01]:
      <k>
          runLemma (
            #range(PROGRAM, PCOUNT, 25) ==K b"`\x00\x92\x91\x90\x83[\x81\x81\x10\x15a\x05\xb6W\x82\x81\x01\x84\x01Q\x86\x82\x01a"
              andBool
            #range(PROGRAM, PCOUNT +Int 27, 20) ==K b"\x01R\x83\x01a\x05\x99V[\x81\x81\x11\x15a\x05\xc9W`\x00a"
              andBool
            #range(PROGRAM, PCOUNT +Int 49, 7) ==K b"\x83\x88\x01\x01R[P"
              andBool
            #asInteger ( #range( PROGRAM, PCOUNT +Int 25, 2 ) ) ==Int #asInteger ( #range( PROGRAM, PCOUNT +Int 47, 2 ) )
          ) =>
          doneLemma ( true )
      </k>
      <program> PROGRAM </program>
      <pc> PCOUNT </pc>
      requires
        PROGRAM ==K b"`\x80`@R4\x80\x15a\x00\x10W`\x00\x80\xfd[P`\x046\x10a\x00AW`\x005`\xe0\x1c\x80c\n\x92T\xe4\x14a\x00FW\x80c\x92\xf8<\x90\x14a\x00PW\x80c\xa1\x02\xc4%\x14a\x00cW[`\x00\x80\xfd[a\x00Na\x00vV[\x00[a\x00Na\x00^6`\x04a\x036V[a\x00\xc1V[a\x00Na\x00q6`\x04a\x036V[a\x012V[`@Qa\x00\x82\x90a\x029V[`@Q\x80\x91\x03\x90`\x00\xf0\x80\x15\x80\x15a\x00\x9eW=`\x00\x80>=`\x00\xfd[P`\x00\x80T`\x01`\x01`\xa0\x1b\x03\x19\x16`\x01`\x01`\xa0\x1b\x03\x92\x90\x92\x16\x91\x90\x91\x17\x90UV[`\x00T`@QcHpIo`\xe0\x1b\x81R`\x01`\x01`\xa0\x1b\x03\x90\x91\x16\x90cHpIo\x90a\x00\xf9\x90\x88\x90\x88\x90\x88\x90\x88\x90\x88\x90`\x04\x01a\x057V[`\x00`@Q\x80\x83\x03\x81`\x00\x87\x80;\x15\x80\x15a\x01\x13W`\x00\x80\xfd[PZ\xf1\x15\x80\x15a\x01'W=`\x00\x80>=`\x00\xfd[PPPPPPPPPV[`\x00\x80T`@\x80Qc\x84V\xcbY`\xe0\x1b\x81R\x90Q`\x01`\x01`\xa0\x1b\x03\x90\x92\x16\x92c\x84V\xcbY\x92`\x04\x80\x84\x01\x93\x82\x90\x03\x01\x81\x83\x87\x80;\x15\x80\x15a\x01sW`\x00\x80\xfd[PZ\xf1\x15\x80\x15a\x01\x87W=`\x00\x80>=`\x00\xfd[PPPP\x7f\x88\\\xb6\x92@\xa95\xd62\xd7\x9c1q\tp\x9e\xcf\xa9\x1a\x80bo\xf3\x98\x9dh\xf6\x7f[\x1d\xd1-`\x00\x1c`\x01`\x01`\xa0\x1b\x03\x16c\xf4\x84H\x14`@Q\x81c\xff\xff\xff\xff\x16`\xe0\x1b\x81R`\x04\x01`\x00`@Q\x80\x83\x03\x81`\x00\x87\x80;\x15\x80\x15a\x01\xe9W`\x00\x80\xfd[PZ\xf1\x15\x80\x15a\x01\xfdW=`\x00\x80>=`\x00\xfd[PP`\x00T`@QcHpIo`\xe0\x1b\x81R`\x01`\x01`\xa0\x1b\x03\x90\x91\x16\x92PcHpIo\x91Pa\x00\xf9\x90\x88\x90\x88\x90\x88\x90\x88\x90\x88\x90`\x04\x01a\x057V[a\x03E\x80a\x062\x839\x01\x90V[cNH{q`\xe0\x1b`\x00R`A`\x04R`$`\x00\xfd[`@Q`\xc0\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x02\x7fWa\x02\x7fa\x02FV[`@R\x90V[`@Q`\x1f\x82\x01`\x1f\x19\x16\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x02\xaeWa\x02\xaea\x02FV[`@R\x91\x90PV[\x805`\x01`\x01`\xa0\x1b\x03\x81\x16\x81\x14a\x02\xcdW`\x00\x80\xfd[\x91\x90PV[`\x00`\x80\x82\x84\x03\x12\x15a\x02\xe4W`\x00\x80\xfd[P\x91\x90PV[`\x00\x80\x83`\x1f\x84\x01\x12a\x02\xfcW`\x00\x80\xfd[P\x815g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x15a\x03\x14W`\x00\x80\xfd[` \x83\x01\x91P\x83` \x82`\x05\x1b\x85\x01\x01\x11\x15a\x03/W`\x00\x80\xfd[\x92P\x92\x90PV[`\x00\x80`\x00\x80`\x00`\xe0\x86\x88\x03\x12\x15a\x03NW`\x00\x80\xfd[\x855g\xff\xff\xff\xff\xff\xff\xff\xff\x80\x82\x11\x15a\x03fW`\x00\x80\xfd[\x90\x87\x01\x90`\xc0\x82\x8a\x03\x12\x15a\x03zW`\x00\x80\xfd[a\x03\x82a\x02\\V[\x825\x81R` a\x03\x93\x81\x85\x01a\x02\xb6V[\x81\x83\x01Ra\x03\xa3`@\x85\x01a\x02\xb6V[`@\x83\x01R``\x84\x015``\x83\x01R`\x80\x84\x015`\x80\x83\x01R`\xa0\x84\x015\x83\x81\x11\x15a\x03\xceW`\x00\x80\xfd[\x80\x85\x01\x94PP\x8a`\x1f\x85\x01\x12a\x03\xe3W`\x00\x80\xfd[\x835\x83\x81\x11\x15a\x03\xf5Wa\x03\xf5a\x02FV[a\x04\x07`\x1f\x82\x01`\x1f\x19\x16\x83\x01a\x02\x85V[\x81\x81R\x8c\x83\x83\x88\x01\x01\x11\x15a\x04\x1bW`\x00\x80\xfd[\x81\x83\x87\x01\x84\x83\x017`\x00\x91\x81\x01\x83\x01\x91\x90\x91R`\xa0\x83\x01R\x90\x97P\x88\x015\x95Pa\x04H\x89`@\x8a\x01a\x02\xd2V[\x94P`\xc0\x88\x015\x91P\x80\x82\x11\x15a\x04^W`\x00\x80\xfd[Pa\x04k\x88\x82\x89\x01a\x02\xeaV[\x96\x99\x95\x98P\x93\x96P\x92\x94\x93\x92PPPV[\x81\x83R\x81\x81` \x85\x017P`\x00\x82\x82\x01` \x90\x81\x01\x91\x90\x91R`\x1f\x90\x91\x01`\x1f\x19\x16\x90\x91\x01\x01\x90V[\x81\x83R`\x00` \x80\x85\x01\x80\x81\x96P\x85`\x05\x1b\x81\x01\x91P\x84`\x00[\x87\x81\x10\x15a\x05*W\x82\x84\x03\x89R\x815`\x1e\x19\x886\x03\x01\x81\x12a\x04\xe0W`\x00\x80\xfd[\x87\x01\x85\x81\x01\x905g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x15a\x04\xfcW`\x00\x80\xfd[\x806\x03\x82\x13\x15a\x05\x0bW`\x00\x80\xfd[a\x05\x16\x86\x82\x84a\x04|V[\x9a\x87\x01\x9a\x95PPP\x90\x84\x01\x90`\x01\x01a\x04\xbfV[P\x91\x97\x96PPPPPPPV[`\xe0\x80\x82R\x86Q\x90\x82\x01R` \x80\x87\x01Q`\x01`\x01`\xa0\x1b\x03\x90\x81\x16a\x01\x00\x84\x01R`@\x88\x01Q\x16a\x01 \x83\x01R``\x87\x01Qa\x01@\x83\x01R`\x80\x87\x01Qa\x01`\x83\x01R`\xa0\x87\x01Q`\xc0a\x01\x80\x84\x01R\x80Qa\x01\xa0\x84\x01\x81\x90R`\x00\x92\x91\x90\x83[\x81\x81\x10\x15a\x05\xb6W\x82\x81\x01\x84\x01Q\x86\x82\x01a\x01\xc0\x01R\x83\x01a\x05\x99V[\x81\x81\x11\x15a\x05\xc9W`\x00a\x01\xc0\x83\x88\x01\x01R[P\x82\x85\x01\x89\x90R`\x1f\x01`\x1f\x19\x16\x84\x01\x90Pa\x01\xc0a\x06\f`@\x86\x01\x89\x805\x82R` \x81\x015` \x83\x01R`@\x81\x015`@\x83\x01R``\x81\x015``\x83\x01RPPV[\x80\x85\x83\x03\x01`\xc0\x86\x01Ra\x06#\x81\x83\x01\x87\x89a\x04\xa5V[\x9a\x99PPPPPPPPPPV\xfe`\x80`@R4\x80\x15a\x00\x10W`\x00\x80\xfd[Pa\x03%\x80a\x00 `\x009`\x00\xf3\xfe`\x80`@R4\x80\x15a\x00\x10W`\x00\x80\xfd[P`\x046\x10a\x006W`\x005`\xe0\x1c\x80cHpIo\x14a\x00;W\x80c\x84V\xcbY\x14a\x00PW[`\x00\x80\xfd[a\x00Na\x00I6`\x04a\x01\xa9V[a\x00bV[\x00[a\x00N`\x00\x80T`\xff\x19\x16`\x01\x17\x90UV[`\x00T`\xff\x16\x15a\x00\xb2W`@QbF\x1b\xcd`\xe5\x1b\x81R` `\x04\x82\x01R`\x16`$\x82\x01Ru\x13\xdc\x1d\x1a[Z\\\xdbT\x1b\xdc\x9d\x18[\x0e\x88\x1c\x18]\\\xd9Y`R\x1b`D\x82\x01R`d\x01`@Q\x80\x91\x03\x90\xfd[PPPPPV[cNH{q`\xe0\x1b`\x00R`A`\x04R`$`\x00\xfd[`@Q`\xc0\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x00\xf2Wa\x00\xf2a\x00\xb9V[`@R\x90V[`@Q`\x1f\x82\x01`\x1f\x19\x16\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x01!Wa\x01!a\x00\xb9V[`@R\x91\x90PV[\x805`\x01`\x01`\xa0\x1b\x03\x81\x16\x81\x14a\x01@W`\x00\x80\xfd[\x91\x90PV[`\x00`\x80\x82\x84\x03\x12\x15a\x01WW`\x00\x80\xfd[P\x91\x90PV[`\x00\x80\x83`\x1f\x84\x01\x12a\x01oW`\x00\x80\xfd[P\x815g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x15a\x01\x87W`\x00\x80\xfd[` \x83\x01\x91P\x83` \x82`\x05\x1b\x85\x01\x01\x11\x15a\x01\xa2W`\x00\x80\xfd[\x92P\x92\x90PV[`\x00\x80`\x00\x80`\x00`\xe0\x86\x88\x03\x12\x15a\x01\xc1W`\x00\x80\xfd[\x855g\xff\xff\xff\xff\xff\xff\xff\xff\x80\x82\x11\x15a\x01\xd9W`\x00\x80\xfd[\x90\x87\x01\x90`\xc0\x82\x8a\x03\x12\x15a\x01\xedW`\x00\x80\xfd[a\x01\xf5a\x00\xcfV[\x825\x81R` a\x02\x06\x81\x85\x01a\x01)V[\x81\x83\x01Ra\x02\x16`@\x85\x01a\x01)V[`@\x83\x01R``\x84\x015``\x83\x01R`\x80\x84\x015`\x80\x83\x01R`\xa0\x84\x015\x83\x81\x11\x15a\x02AW`\x00\x80\xfd[\x80\x85\x01\x94PP\x8a`\x1f\x85\x01\x12a\x02VW`\x00\x80\xfd[\x835\x83\x81\x11\x15a\x02hWa\x02ha\x00\xb9V[a\x02z`\x1f\x82\x01`\x1f\x19\x16\x83\x01a\x00\xf8V[\x81\x81R\x8c\x83\x83\x88\x01\x01\x11\x15a\x02\x8eW`\x00\x80\xfd[\x81\x83\x87\x01\x84\x83\x017`\x00\x91\x81\x01\x83\x01\x91\x90\x91R`\xa0\x83\x01R\x90\x97P\x88\x015\x95Pa\x02\xbb\x89`@\x8a\x01a\x01EV[\x94P`\xc0\x88\x015\x91P\x80\x82\x11\x15a\x02\xd1W`\x00\x80\xfd[Pa\x02\xde\x88\x82\x89\x01a\x01]V[\x96\x99\x95\x98P\x93\x96P\x92\x94\x93\x92PPPV\xfe\xa2dipfsX\"\x12 \xdc'\xd1\x9b\x8dS\xe0\xa2O\x86k\x19\xa4\x9f\xd1\xa3\x12_\xfb\x0e\x17\xcbDk}\xd6\xc7\x9a\xc5\x9f\xfb\x8bdsolcC\x00\x08\x0f\x003\xa2dipfsX\"\x12 \xb0<\xc1\x9e\xa1U(<)%|.w\xe8\x8c\xa2\x85\xe8y\xc9T\xebI\xf9\x1b\xc2\x91\xd6,\xc58\xeedsolcC\x00\x08\x0f\x003"
      andBool
        PCOUNT ==Int 1427

    claim [bytes-indexing-via-range]:
      <k> runLemma ( PROGRAM [ PCOUNT ] ==K 96) => doneLemma ( true ) ... </k>
      requires #range(PROGRAM, PCOUNT, 25) ==K b"`\x00\x92\x91\x90\x83[\x81\x81\x10\x15a\x05\xb6W\x82\x81\x01\x84\x01Q\x86\x82\x01a"
       andBool #range(PROGRAM, PCOUNT +Int 27, 20) ==K b"\x01R\x83\x01a\x05\x99V[\x81\x81\x11\x15a\x05\xc9W`\x00a"
       andBool #range(PROGRAM, PCOUNT +Int 49, 7) ==K b"\x83\x88\x01\x01R[P"
       andBool OFFSET:Int ==Int #asInteger ( #range( PROGRAM, PCOUNT +Int 25, 2 ) )
       andBool OFFSET:Int ==Int #asInteger ( #range( PROGRAM, PCOUNT +Int 47, 2 ) )
       andBool PCOUNT <=Int lengthBytes(PROGRAM)

    claim [bytes-indexing-via-concat]:
      <k> runLemma ( PROGRAM [ PCOUNT ] ==Int 96 ) => doneLemma ( true ) ... </k>
      requires PROGRAM ==K PROG_PRE +Bytes b"`\x00\x92\x91\x90\x83[\x81\x81\x10\x15a\x05\xb6W\x82\x81\x01\x84\x01Q\x86\x82\x01a" +Bytes
                           OFFSET_BYTES +Bytes b"\x01R\x83\x01a\x05\x99V[\x81\x81\x11\x15a\x05\xc9W`\x00a" +Bytes OFFSET_BYTES +Bytes
                           b"\x83\x88\x01\x01R[P" +Bytes _PROG_POST
       andBool lengthBytes(PROG_PRE) ==Int PCOUNT
       andBool lengthBytes(OFFSET_BYTES) ==Int 2
       andBool PCOUNT <=Int lengthBytes(PROGRAM)

    claim [copy-memory-to-memory-provable]:
      <k> #execute ... </k>
      <useGas> false </useGas>
      <schedule> SHANGHAI </schedule>
      <jumpDests> JUMPDESTS </jumpDests>
      // The program and program counter are symbolic, focusing on the part we will be executing (CP)
      <program> PROG_PRE +Bytes CP +Bytes _PROG_POST </program>
      <pc> PCOUNT => PCOUNT +Int lengthBytes(CP) </pc>
      // The word stack has the appropriate form, as per the compiled code
      <wordStack> LENGTH : SRC : STEP : DEST : WS => LENGTH : SRC : STEP : 0 : DEST : WS </wordStack>
      // The program copies LENGTH bytes of memory from SRC +Int 32 to DEST +Int OFFSET,
      // padded with 32 zeros in case LENGTH is not divisible by 32
      <localMem>
        LM => LM [ DEST +Int OFFSET := #range ( LM, SRC +Int 32, LENGTH ) +Bytes
                                       #buf ( ( ( notMaxUInt5 &Int ( LENGTH +Int maxUInt5 ) ) -Int LENGTH ) , 0 ) +Bytes
                                       #buf ( ( ( ( 32 -Int ( ( notMaxUInt5 &Int ( LENGTH +Int maxUInt5 ) ) -Int LENGTH ) ) ) modInt 32 ), 0 ) ]
      </localMem>
      // OP is the program corresponding to the compiled EVM bytecode shown above, starting from program counter 1427
      requires OP ==K b"`\x00\x92\x91\x90\x83[\x81\x81\x10\x15a\x05\xb6W\x82\x81\x01\x84\x01Q\x86\x82\x01a\x01\xc0\x01R\x83\x01a\x05\x99V[\x81\x81\x11\x15a\x05\xc9W`\x00a\x01\xc0\x83\x88\x01\x01R[P"
       // The execution effectively starts from CP
       andBool lengthBytes(PROG_PRE) ==Int PCOUNT
       // OFFSET_BYTES and OFFSET represent a symbolic offset, generalizing the concrete hardcoded offset (448).
       andBool lengthBytes(OFFSET_BYTES) ==K 2
       andBool OFFSET ==Int #asWord ( OFFSET_BYTES )

       // The current program we are executing differs from the original one only in the hardcoded jump addresses,
       // which are now relative to PCOUNT, and the hardcoded offset, which is now symbolic.
       andBool CP ==K OP
                      [ 12 := #buf(2, PCOUNT +Int 35) ]
                      [ 25 := OFFSET_BYTES ]
                      [ 32 := #buf(2, PCOUNT +Int  6) ]
                      [ 41 := #buf(2, PCOUNT +Int 54) ]
                      [ 47 := OFFSET_BYTES ]

       // STEP always equals 32 as the memory is copied in chunks of 32 bytes
       // This equality is placed in the requires clause instead of in the LHS
       // of the config directly to enable unification of the LHS to pass trivially,
       // speeding up the execution
       andBool STEP ==Int 32

       // Various well-formedness constraints. In particular, the maxBytesLength-related ones are present to
       // remove various chops that would otherwise creep into the execution, and are reasonable since byte
       // arrays in actual programs would never reach that size.
       andBool 0 <=Int PCOUNT
       andBool 0 <=Int LENGTH andBool LENGTH <Int maxBytesLength
       andBool 0 <=Int SRC    andBool SRC    <Int maxBytesLength
       andBool 0 <=Int DEST   andBool DEST   <Int maxBytesLength
       andBool 0 <=Int OFFSET andBool OFFSET <Int maxBytesLength
       andBool #sizeWordStack(WS) <=Int 1014

       andBool SRC +Int 32 +Int LENGTH <=Int DEST +Int OFFSET // No overlap between source and destination
       andBool DEST +Int OFFSET <=Int lengthBytes(LM)         // Destination starts within current memory
       // All JUMPDESTs in the program are valid
       andBool (PCOUNT +Int 6) in JUMPDESTS andBool (PCOUNT +Int 35) in JUMPDESTS andBool (PCOUNT +Int 54) in JUMPDESTS
       andBool PCOUNT +Int 55 <Int 2 ^Int 16  // and fit into two bytes

       // The length needs to be fixed when there's no invariant present
       andBool LENGTH ==Int 42

    claim [side-condition-check]:
      <k> runLemma (
                    #range(PROGRAM, PCOUNT +Int 25, 2) ==K #range(PROGRAM, PCOUNT +Int 47, 2)
            andBool #range(PROGRAM, PCOUNT, 56) ==K
                      b"`\x00\x92\x91\x90\x83[\x81\x81\x10\x15a\x05\xb6W\x82\x81\x01\x84\x01Q\x86\x82\x01a\x01\xc0\x01R\x83\x01a\x05\x99V[\x81\x81\x11\x15a\x05\xc9W`\x00a\x01\xc0\x83\x88\x01\x01R[P"
                        [ 12 := #buf(2, PCOUNT +Int 35) ]
                        [ 25 := #range(PROGRAM, PCOUNT +Int 25, 2) ]
                        [ 32 := #buf(2, PCOUNT +Int  6) ]
                        [ 41 := #buf(2, PCOUNT +Int 54) ]
                        [ 47 := #range(PROGRAM, PCOUNT +Int 47, 2) ]

            andBool 0 <=Int PCOUNT
            andBool 0 <=Int LENGTH andBool LENGTH <Int maxBytesLength
            andBool 0 <=Int SRC    andBool SRC    <Int maxBytesLength
            andBool 0 <=Int DEST   andBool DEST   <Int maxBytesLength
            andBool 0 <=Int #asWord ( #range(PROGRAM, PCOUNT +Int 25, 2) ) andBool #asWord ( #range(PROGRAM, PCOUNT +Int 25, 2) ) <Int maxBytesLength
            andBool #sizeWordStack(WS) <=Int 1014

            andBool SRC +Int 32 +Int LENGTH <=Int DEST +Int #asWord ( #range(PROGRAM, PCOUNT +Int 25, 2) )
            andBool DEST +Int #asWord ( #range(PROGRAM, PCOUNT +Int 25, 2) ) <=Int lengthBytes(LM)

            andBool (PCOUNT +Int 6) in JUMPDESTS:Set andBool (PCOUNT +Int 35) in JUMPDESTS andBool (PCOUNT +Int 54) in JUMPDESTS
            andBool PCOUNT +Int 55 <Int 2 ^Int 16
          ) => doneLemma (
            true
          )
      </k>
      requires PROGRAM ==K b"`\x80`@R4\x80\x15a\x00\x10W`\x00\x80\xfd[P`\x046\x10a\x00AW`\x005`\xe0\x1c\x80c\n\x92T\xe4\x14a\x00FW\x80c\x92\xf8<\x90\x14a\x00PW\x80c\xa1\x02\xc4%\x14a\x00cW[`\x00\x80\xfd[a\x00Na\x00vV[\x00[a\x00Na\x00^6`\x04a\x036V[a\x00\xc1V[a\x00Na\x00q6`\x04a\x036V[a\x012V[`@Qa\x00\x82\x90a\x029V[`@Q\x80\x91\x03\x90`\x00\xf0\x80\x15\x80\x15a\x00\x9eW=`\x00\x80>=`\x00\xfd[P`\x00\x80T`\x01`\x01`\xa0\x1b\x03\x19\x16`\x01`\x01`\xa0\x1b\x03\x92\x90\x92\x16\x91\x90\x91\x17\x90UV[`\x00T`@QcHpIo`\xe0\x1b\x81R`\x01`\x01`\xa0\x1b\x03\x90\x91\x16\x90cHpIo\x90a\x00\xf9\x90\x88\x90\x88\x90\x88\x90\x88\x90\x88\x90`\x04\x01a\x057V[`\x00`@Q\x80\x83\x03\x81`\x00\x87\x80;\x15\x80\x15a\x01\x13W`\x00\x80\xfd[PZ\xf1\x15\x80\x15a\x01'W=`\x00\x80>=`\x00\xfd[PPPPPPPPPV[`\x00\x80T`@\x80Qc\x84V\xcbY`\xe0\x1b\x81R\x90Q`\x01`\x01`\xa0\x1b\x03\x90\x92\x16\x92c\x84V\xcbY\x92`\x04\x80\x84\x01\x93\x82\x90\x03\x01\x81\x83\x87\x80;\x15\x80\x15a\x01sW`\x00\x80\xfd[PZ\xf1\x15\x80\x15a\x01\x87W=`\x00\x80>=`\x00\xfd[PPPP\x7f\x88\\\xb6\x92@\xa95\xd62\xd7\x9c1q\tp\x9e\xcf\xa9\x1a\x80bo\xf3\x98\x9dh\xf6\x7f[\x1d\xd1-`\x00\x1c`\x01`\x01`\xa0\x1b\x03\x16c\xf4\x84H\x14`@Q\x81c\xff\xff\xff\xff\x16`\xe0\x1b\x81R`\x04\x01`\x00`@Q\x80\x83\x03\x81`\x00\x87\x80;\x15\x80\x15a\x01\xe9W`\x00\x80\xfd[PZ\xf1\x15\x80\x15a\x01\xfdW=`\x00\x80>=`\x00\xfd[PP`\x00T`@QcHpIo`\xe0\x1b\x81R`\x01`\x01`\xa0\x1b\x03\x90\x91\x16\x92PcHpIo\x91Pa\x00\xf9\x90\x88\x90\x88\x90\x88\x90\x88\x90\x88\x90`\x04\x01a\x057V[a\x03E\x80a\x062\x839\x01\x90V[cNH{q`\xe0\x1b`\x00R`A`\x04R`$`\x00\xfd[`@Q`\xc0\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x02\x7fWa\x02\x7fa\x02FV[`@R\x90V[`@Q`\x1f\x82\x01`\x1f\x19\x16\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x02\xaeWa\x02\xaea\x02FV[`@R\x91\x90PV[\x805`\x01`\x01`\xa0\x1b\x03\x81\x16\x81\x14a\x02\xcdW`\x00\x80\xfd[\x91\x90PV[`\x00`\x80\x82\x84\x03\x12\x15a\x02\xe4W`\x00\x80\xfd[P\x91\x90PV[`\x00\x80\x83`\x1f\x84\x01\x12a\x02\xfcW`\x00\x80\xfd[P\x815g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x15a\x03\x14W`\x00\x80\xfd[` \x83\x01\x91P\x83` \x82`\x05\x1b\x85\x01\x01\x11\x15a\x03/W`\x00\x80\xfd[\x92P\x92\x90PV[`\x00\x80`\x00\x80`\x00`\xe0\x86\x88\x03\x12\x15a\x03NW`\x00\x80\xfd[\x855g\xff\xff\xff\xff\xff\xff\xff\xff\x80\x82\x11\x15a\x03fW`\x00\x80\xfd[\x90\x87\x01\x90`\xc0\x82\x8a\x03\x12\x15a\x03zW`\x00\x80\xfd[a\x03\x82a\x02\\V[\x825\x81R` a\x03\x93\x81\x85\x01a\x02\xb6V[\x81\x83\x01Ra\x03\xa3`@\x85\x01a\x02\xb6V[`@\x83\x01R``\x84\x015``\x83\x01R`\x80\x84\x015`\x80\x83\x01R`\xa0\x84\x015\x83\x81\x11\x15a\x03\xceW`\x00\x80\xfd[\x80\x85\x01\x94PP\x8a`\x1f\x85\x01\x12a\x03\xe3W`\x00\x80\xfd[\x835\x83\x81\x11\x15a\x03\xf5Wa\x03\xf5a\x02FV[a\x04\x07`\x1f\x82\x01`\x1f\x19\x16\x83\x01a\x02\x85V[\x81\x81R\x8c\x83\x83\x88\x01\x01\x11\x15a\x04\x1bW`\x00\x80\xfd[\x81\x83\x87\x01\x84\x83\x017`\x00\x91\x81\x01\x83\x01\x91\x90\x91R`\xa0\x83\x01R\x90\x97P\x88\x015\x95Pa\x04H\x89`@\x8a\x01a\x02\xd2V[\x94P`\xc0\x88\x015\x91P\x80\x82\x11\x15a\x04^W`\x00\x80\xfd[Pa\x04k\x88\x82\x89\x01a\x02\xeaV[\x96\x99\x95\x98P\x93\x96P\x92\x94\x93\x92PPPV[\x81\x83R\x81\x81` \x85\x017P`\x00\x82\x82\x01` \x90\x81\x01\x91\x90\x91R`\x1f\x90\x91\x01`\x1f\x19\x16\x90\x91\x01\x01\x90V[\x81\x83R`\x00` \x80\x85\x01\x80\x81\x96P\x85`\x05\x1b\x81\x01\x91P\x84`\x00[\x87\x81\x10\x15a\x05*W\x82\x84\x03\x89R\x815`\x1e\x19\x886\x03\x01\x81\x12a\x04\xe0W`\x00\x80\xfd[\x87\x01\x85\x81\x01\x905g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x15a\x04\xfcW`\x00\x80\xfd[\x806\x03\x82\x13\x15a\x05\x0bW`\x00\x80\xfd[a\x05\x16\x86\x82\x84a\x04|V[\x9a\x87\x01\x9a\x95PPP\x90\x84\x01\x90`\x01\x01a\x04\xbfV[P\x91\x97\x96PPPPPPPV[`\xe0\x80\x82R\x86Q\x90\x82\x01R` \x80\x87\x01Q`\x01`\x01`\xa0\x1b\x03\x90\x81\x16a\x01\x00\x84\x01R`@\x88\x01Q\x16a\x01 \x83\x01R``\x87\x01Qa\x01@\x83\x01R`\x80\x87\x01Qa\x01`\x83\x01R`\xa0\x87\x01Q`\xc0a\x01\x80\x84\x01R\x80Qa\x01\xa0\x84\x01\x81\x90R`\x00\x92\x91\x90\x83[\x81\x81\x10\x15a\x05\xb6W\x82\x81\x01\x84\x01Q\x86\x82\x01a\x01\xc0\x01R\x83\x01a\x05\x99V[\x81\x81\x11\x15a\x05\xc9W`\x00a\x01\xc0\x83\x88\x01\x01R[P\x82\x85\x01\x89\x90R`\x1f\x01`\x1f\x19\x16\x84\x01\x90Pa\x01\xc0a\x06\f`@\x86\x01\x89\x805\x82R` \x81\x015` \x83\x01R`@\x81\x015`@\x83\x01R``\x81\x015``\x83\x01RPPV[\x80\x85\x83\x03\x01`\xc0\x86\x01Ra\x06#\x81\x83\x01\x87\x89a\x04\xa5V[\x9a\x99PPPPPPPPPPV\xfe`\x80`@R4\x80\x15a\x00\x10W`\x00\x80\xfd[Pa\x03%\x80a\x00 `\x009`\x00\xf3\xfe`\x80`@R4\x80\x15a\x00\x10W`\x00\x80\xfd[P`\x046\x10a\x006W`\x005`\xe0\x1c\x80cHpIo\x14a\x00;W\x80c\x84V\xcbY\x14a\x00PW[`\x00\x80\xfd[a\x00Na\x00I6`\x04a\x01\xa9V[a\x00bV[\x00[a\x00N`\x00\x80T`\xff\x19\x16`\x01\x17\x90UV[`\x00T`\xff\x16\x15a\x00\xb2W`@QbF\x1b\xcd`\xe5\x1b\x81R` `\x04\x82\x01R`\x16`$\x82\x01Ru\x13\xdc\x1d\x1a[Z\\\xdbT\x1b\xdc\x9d\x18[\x0e\x88\x1c\x18]\\\xd9Y`R\x1b`D\x82\x01R`d\x01`@Q\x80\x91\x03\x90\xfd[PPPPPV[cNH{q`\xe0\x1b`\x00R`A`\x04R`$`\x00\xfd[`@Q`\xc0\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x00\xf2Wa\x00\xf2a\x00\xb9V[`@R\x90V[`@Q`\x1f\x82\x01`\x1f\x19\x16\x81\x01g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x82\x82\x10\x17\x15a\x01!Wa\x01!a\x00\xb9V[`@R\x91\x90PV[\x805`\x01`\x01`\xa0\x1b\x03\x81\x16\x81\x14a\x01@W`\x00\x80\xfd[\x91\x90PV[`\x00`\x80\x82\x84\x03\x12\x15a\x01WW`\x00\x80\xfd[P\x91\x90PV[`\x00\x80\x83`\x1f\x84\x01\x12a\x01oW`\x00\x80\xfd[P\x815g\xff\xff\xff\xff\xff\xff\xff\xff\x81\x11\x15a\x01\x87W`\x00\x80\xfd[` \x83\x01\x91P\x83` \x82`\x05\x1b\x85\x01\x01\x11\x15a\x01\xa2W`\x00\x80\xfd[\x92P\x92\x90PV[`\x00\x80`\x00\x80`\x00`\xe0\x86\x88\x03\x12\x15a\x01\xc1W`\x00\x80\xfd[\x855g\xff\xff\xff\xff\xff\xff\xff\xff\x80\x82\x11\x15a\x01\xd9W`\x00\x80\xfd[\x90\x87\x01\x90`\xc0\x82\x8a\x03\x12\x15a\x01\xedW`\x00\x80\xfd[a\x01\xf5a\x00\xcfV[\x825\x81R` a\x02\x06\x81\x85\x01a\x01)V[\x81\x83\x01Ra\x02\x16`@\x85\x01a\x01)V[`@\x83\x01R``\x84\x015``\x83\x01R`\x80\x84\x015`\x80\x83\x01R`\xa0\x84\x015\x83\x81\x11\x15a\x02AW`\x00\x80\xfd[\x80\x85\x01\x94PP\x8a`\x1f\x85\x01\x12a\x02VW`\x00\x80\xfd[\x835\x83\x81\x11\x15a\x02hWa\x02ha\x00\xb9V[a\x02z`\x1f\x82\x01`\x1f\x19\x16\x83\x01a\x00\xf8V[\x81\x81R\x8c\x83\x83\x88\x01\x01\x11\x15a\x02\x8eW`\x00\x80\xfd[\x81\x83\x87\x01\x84\x83\x017`\x00\x91\x81\x01\x83\x01\x91\x90\x91R`\xa0\x83\x01R\x90\x97P\x88\x015\x95Pa\x02\xbb\x89`@\x8a\x01a\x01EV[\x94P`\xc0\x88\x015\x91P\x80\x82\x11\x15a\x02\xd1W`\x00\x80\xfd[Pa\x02\xde\x88\x82\x89\x01a\x01]V[\x96\x99\x95\x98P\x93\x96P\x92\x94\x93\x92PPPV\xfe\xa2dipfsX\"\x12 \xdc'\xd1\x9b\x8dS\xe0\xa2O\x86k\x19\xa4\x9f\xd1\xa3\x12_\xfb\x0e\x17\xcbDk}\xd6\xc7\x9a\xc5\x9f\xfb\x8bdsolcC\x00\x08\x0f\x003\xa2dipfsX\"\x12 \xb0<\xc1\x9e\xa1U(<)%|.w\xe8\x8c\xa2\x85\xe8y\xc9T\xebI\xf9\x1b\xc2\x91\xd6,\xc58\xeedsolcC\x00\x08\x0f\x003"
       andBool PCOUNT ==Int 1427
       andBool LENGTH ==Int lengthBytes ( BYTES_DATA:Bytes )
       andBool SRC ==Int 320
       andBool DEST ==Int ( ( notMaxUInt5 &Int ( lengthBytes ( BYTES_DATA:Bytes ) +Int maxUInt5 ) ) +Int 356 )
       andBool JUMPDESTS ==K ( SetItem ( 1961 ) ( SetItem ( 1967 ) ( SetItem ( 1943 ) ( SetItem ( 1985 ) ( SetItem ( 1938 ) ( SetItem ( 1915 ) ( SetItem ( 1907 ) ( SetItem ( 1803 ) ( SetItem ( 1825 ) ( SetItem ( 1866 ) ( SetItem ( 1860 ) ( SetItem ( 1716 ) ( SetItem ( 1796 ) ( SetItem ( 1672 ) ( SetItem ( 1677 ) ( SetItem ( 1696 ) ( SetItem ( 1691 ) ( SetItem ( 1698 ) ( SetItem ( 1634 ) ( SetItem ( 1602 ) ( SetItem ( 1571 ) ( SetItem ( 1548 ) ( SetItem ( 1433 ) ( SetItem ( 1462 ) ( SetItem ( 1481 ) ( SetItem ( 1302 ) ( SetItem ( 1335 ) ( SetItem ( 1322 ) ( SetItem ( 1291 ) ( SetItem ( 1276 ) ( SetItem ( 1248 ) ( SetItem ( 1215 ) ( SetItem ( 1131 ) ( SetItem ( 1118 ) ( SetItem ( 1189 ) ( SetItem ( 1148 ) ( SetItem ( 1096 ) ( SetItem ( 1031 ) ( SetItem ( 1013 ) ( SetItem ( 1051 ) ( SetItem ( 78 ) ( SetItem ( 70 ) ( SetItem ( 65 ) ( SetItem ( 16 ) ( SetItem ( 80 ) ( SetItem ( 99 ) ( SetItem ( 94 ) ( SetItem ( 2252 ) ( SetItem ( 2272 ) ( SetItem ( 2216 ) ( SetItem ( 2234 ) ( SetItem ( 2317 ) ( SetItem ( 2339 ) ( SetItem ( 2352 ) ( SetItem ( 2091 ) ( SetItem ( 2009 ) ( SetItem ( 2067 ) ( SetItem ( 2043 ) ( SetItem ( 2036 ) ( SetItem ( 2152 ) ( SetItem ( 2195 ) ( SetItem ( 2111 ) ( SetItem ( 2119 ) ( SetItem ( 2136 ) ( SetItem ( 371 ) ( SetItem ( 306 ) ( SetItem ( 391 ) ( SetItem ( 295 ) ( SetItem ( 275 ) ( SetItem ( 249 ) ( SetItem ( 113 ) ( SetItem ( 118 ) ( SetItem ( 130 ) ( SetItem ( 193 ) ( SetItem ( 158 ) ( SetItem ( 870 ) ( SetItem ( 815 ) ( SetItem ( 822 ) ( SetItem ( 846 ) ( SetItem ( 890 ) ( SetItem ( 898 ) ( SetItem ( 788 ) ( SetItem ( 740 ) ( SetItem ( 746 ) ( SetItem ( 764 ) ( SetItem ( 717 ) ( SetItem ( 722 ) ( SetItem ( 645 ) ( SetItem ( 639 ) ( SetItem ( 604 ) ( SetItem ( 694 ) ( SetItem ( 686 ) ( SetItem ( 582 ) ( SetItem ( 569 ) ( SetItem ( 509 ) ( SetItem ( 489 ) ( SetItem ( 915 ) ( SetItem ( 974 ) ( SetItem ( 931 ) SetItem ( 995 ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) ) )
       andBool WS:WordStack ==K ( 10 : ( ( lengthBytes ( BYTES_DATA:Bytes ) +Int 484 ) : ( 68 : ( L2OutputIndex:Int : ( 128 : ( 249 : ( _ : ( 491460923342184218035706888008750043977755113263 : ( 10 : ( ( lengthBytes ( BYTES_DATA:Bytes ) +Int 484 ) : ( 68 : ( L2OutputIndex:Int : ( 128 : ( 78 : ( _ : .WordStack ) ) ) ) ) ) ) ) ) ) ) ) ) ) )
       andBool LM ==K b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +Bytes #buf ( 32 , ( ( notMaxUInt5 &Int ( lengthBytes ( BYTES_DATA:Bytes ) +Int maxUInt5 ) ) +Int 352 ) ) +Bytes b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +Bytes #buf ( 32 , Nonce:Int ) +Bytes #buf ( 32 , Sender:Int ) +Bytes #buf ( 32 , Target:Int ) +Bytes #buf ( 32 , Value:Int ) +Bytes #buf ( 32 , GasLimit:Int ) +Bytes b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01@" +Bytes #buf ( 32 , lengthBytes ( BYTES_DATA:Bytes ) ) +Bytes BYTES_DATA:Bytes +Bytes #buf ( ( ( notMaxUInt5 &Int ( lengthBytes ( BYTES_DATA:Bytes ) +Int maxUInt5 ) ) -Int lengthBytes ( BYTES_DATA:Bytes ) ) , 0 ) +Bytes b"HpIo\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xe0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +Bytes #buf ( 32 , Nonce:Int ) +Bytes #buf ( 32 , Sender:Int ) +Bytes #buf ( 32 , Target:Int ) +Bytes #buf ( 32 , Value:Int ) +Bytes #buf ( 32 , GasLimit:Int ) +Bytes b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xc0" +Bytes #buf ( 32 , lengthBytes ( BYTES_DATA:Bytes ) )

       andBool lengthBytes ( BYTES_DATA:Bytes ) <=Int 1048576

endmodule