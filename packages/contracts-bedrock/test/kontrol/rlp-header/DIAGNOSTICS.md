# Byte and word diagnostics

This research branch selects four small tool diagnostics instead of the header
proofs. The parser branch and its incomplete proofs remain preserved separately.
No diagnostic proves RLP correctness, withdrawal authorization, or a generic K
byte lemma. No result is imported as an assumption into another proof.

Two assertions compare equivalent source expressions for the upper bound of a
Solidity memory byte. The compiler may normalize them to the same expression;
saved graphs must be inspected before inferring anything about comparison
orientation. Two further assertions compare Solidity memory indexing with the
production header decoder's assembly expressions: BYTE over MLOAD at offset
zero, and a high-byte mask over MLOAD at offset one. Each quantifies independent unrestricted 256-bit head and
tail words, within a fixed allocated 64-byte observation. There are no input
assumptions, external calls, deployment snapshots or custom K lemmas.

The isolated profile remains unoptimized Solc 0.8.15. CI uses the pinned Kontrol
definition, stack checks, Cancun execution and gas erasure. A five-minute global
prover budget bounds this diagnostic. Failed or pending graphs remain failures
or incomplete obligations, including when a projected SMT model is infeasible
under concrete byte semantics. Passing only a byte bound does not establish the
independent byte-extraction equalities or the parser proofs.

Each RPC server records a separate solver transcript and simplifier log under
`kout-proofs/diagnostics`, included in the existing compressed CI artifacts.
Logging changes diagnostic overhead; these timings are not parser benchmarks.
