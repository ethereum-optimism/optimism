// Independent input representation under development; correspondence remains unproved.
// No production execution summary or proved claim.
requires "serialization.md"

module WITHDRAWAL-CALLDATA
    imports SERIALIZATION

    syntax Bytes ::= #wPaddedBytes(Bytes) [function]
                   | #wDynamicBytes(Bytes) [function]
                   | #wTuple(Int, Int, Int, Int, Int, Bytes) [function]
                   | #wProofArray(List) [function]
                   | #wProofHeads(List, Int) [function]
                   | #wProofTails(List) [function]
                   | #wProveCalldata(Int, Int, Int, Int, Int, Bytes,
                                     Int, Int, Int, Int, Int, List) [function]
                   | #wProveEncoded(Bytes, Int, Bytes, List) [function]

    rule #wPaddedBytes(DATA)
      => DATA +Bytes #buf(32 *Int ((lengthBytes(DATA) +Int 31) /Int 32) -Int lengthBytes(DATA), 0)
    rule #wDynamicBytes(DATA) => #wordBytes(lengthBytes(DATA)) +Bytes #wPaddedBytes(DATA)

    // Tuple offsets start at the tuple, not at the outer call's argument head.
    rule #wTuple(NONCE, SENDER, TARGET, VALUE, GAS, DATA)
      => #wordBytes(NONCE) +Bytes #wordBytes(SENDER) +Bytes #wordBytes(TARGET)
       +Bytes #wordBytes(VALUE) +Bytes #wordBytes(GAS) +Bytes #wordBytes(192)
       +Bytes #wDynamicBytes(DATA)

    // Dynamic-array element offsets are relative to the first offset word,
    // immediately after the array's length word.
    rule #wProofArray(PROOF)
      => #wordBytes(size(PROOF)) +Bytes #wProofHeads(PROOF, 32 *Int size(PROOF))
       +Bytes #wProofTails(PROOF)
    rule #wProofHeads(.List, _) => .Bytes
    rule #wProofHeads(ListItem(NODE:Bytes) REST, OFFSET)
      => #wordBytes(OFFSET)
       +Bytes #wProofHeads(REST, OFFSET +Int 32 +Int 32 *Int ((lengthBytes(NODE) +Int 31) /Int 32))
    rule #wProofTails(.List) => .Bytes
    rule #wProofTails(ListItem(NODE:Bytes) REST)
      => #wDynamicBytes(NODE) +Bytes #wProofTails(REST)

    rule #wProveCalldata(NONCE, SENDER, TARGET, VALUE, GAS, DATA,
                        INDEX, VERSION, STATE, STORAGE, BLOCK, PROOF)
      => #wProveEncoded(#wTuple(NONCE, SENDER, TARGET, VALUE, GAS, DATA), INDEX,
                       #wordBytes(VERSION) +Bytes #wordBytes(STATE)
                       +Bytes #wordBytes(STORAGE) +Bytes #wordBytes(BLOCK), PROOF)
    // Seven head words: dynamic tuple, index, four output words, dynamic proof.
    // Selector verified against the compiled OptimismPortal2 artifact.
    rule #wProveEncoded(TX, INDEX, OUTPUT, PROOF)
      => b"\x48\x70\x49\x6f" +Bytes #wordBytes(224) +Bytes #wordBytes(INDEX)
       +Bytes OUTPUT +Bytes #wordBytes(224 +Int lengthBytes(TX))
       +Bytes TX +Bytes #wProofArray(PROOF)

    // Representation domains only; do not add RLP, path, membership or game assumptions.
    // A future claim must separately retain all scalar ABI domains and data's uint64 length.
    syntax Bool ::= #wWitnessDomain(List) [function]
                  | #wWitnessElementsDomain(List) [function]
    rule #wWitnessDomain(PROOF)
      => size(PROOF) <Int 2 ^Int 64 andBool #wWitnessElementsDomain(PROOF)
    rule #wWitnessElementsDomain(.List) => true
    rule #wWitnessElementsDomain(ListItem(NODE:Bytes) REST)
      => lengthBytes(NODE) <Int 2 ^Int 64 andBool #wWitnessElementsDomain(REST)
    rule #wWitnessElementsDomain(_) => false [owise]
endmodule
