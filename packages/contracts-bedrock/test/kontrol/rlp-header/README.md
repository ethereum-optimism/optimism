# RLP header observations

Experimental component, not yet proved. This is not a withdrawal, trie-inclusion,
or arbitrary-memory theorem.

The proposed theorem quantifies two independent 256-bit words and a full-width
declared span. The actual private `RLPReader._decodeLength` helper, exposed by a
generated internal wrapper, must return exactly the independently specified
first-item header tuple, or reject exactly when that grammar is invalid. The
grammar follows the [Ethereum RLP definition](https://ethereum.org/developers/docs/data-structures-and-encoding/rlp/).
It allows trailing bytes and does not recursively validate payloads.

The 64-byte observation buffer supplies all 33 bytes physically read by the
helper's explicit memory loads. Only the first nine affect its value decisions.
The declared span is independent: it is neither capped at 64 nor asserted to
describe an allocated array of that size. No well-formed-header assumption is
made. Invalid cases are caught and checked; a test-level revert is not success.
An acceptance witness includes a declared payload beyond the buffer.

The generator checks that removing the wrapper and reversing the library rename
recovers the production source exactly. This establishes source provenance,
not equality with an optimized production call site's bytecode. Compiler
correctness is a trust dependency.

This isolated profile disables optimization. The optimized compiler's checked
addition uses bitwise complement, which left infeasible overflow branches in
the first two runs. Unoptimized compilation uses subtraction for that check.
This changes proof bytecode, not production source or the quantified domain.

An additional, currently unproved correspondence obligation must relate real
memory spans to these observations: unread-payload independence, relocation,
allocation/nonaliasing, address arithmetic and compiled access footprint.
The direct harness uses its allocated pointer, not an arbitrary EVM address.
Gas is erased; finite-gas termination and memory expansion costs are outside the
claim. This obligation is not imported as an axiom into a withdrawal theorem.

`mise x -- just build-rlp-header` generates and compiles only. Proofs run in CI via
`mise x -- just test-rlp-header`, with fresh Kontrol output, stack checks, Cancun,
no deployment snapshot, no `--assume-defined`, and no custom K lemmas. The first
feasibility run has a 15-minute prover limit. A timeout is an incomplete proof.

The branch temporarily routes its manual Kontrol job to this isolated component
to avoid repeating the already verified eligibility suite. Restore full-suite
integration before treating this branch as a merge candidate.
