# Temporary native CALL diagnostic

This bundle contains eight source claims derived from the audited initializer
receipts in CircleCI pipeline 134089, revision
`130460a3390c5b1eda665fbc1b92a89b8a5575f2`. Its manifest records each source hash
and the compiled definition hash. CI also checks the fresh caller runtime before
using these snapshots. The first probe executes receipt 49 only; the other seven
are explicitly outstanding.

The claims preserve the saved conditions and declare canonical input domains.
They test a constructed CALL boundary with actual Portal and dependency code.
Caller gas/memory correspondence, raw ABI completeness, trie inclusion, factory
composition, record provenance and both finalizers remain required obligations.
A passing diagnostic is not completion of withdrawal security verification.

The compact archive transports generated source to CI without running a local
prover. It does not contain a proof result or a fabricated frontend cache. Native
source parsing and initialized-domain inspection remain required.
Byte literals use hexadecimal escapes so bytecode cannot be mistaken for source
comments by the outer reader; this changes no literal byte or claim condition.

Remove this temporary snapshot bundle and runner, restore the full CI suite, and
provide reproducible proof generation before PR readiness. This directory is
diagnostic scaffolding, not the proposed final proof interface.
