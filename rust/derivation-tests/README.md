# derivation-tests

Cross-implementation derivation pipeline test framework for the OP Stack.

Constructs deterministic L1 and L2 chains in-process, serves them over JSON-RPC and Beacon API, and verifies derivation results across multiple implementations (op-program, kona-host).

## Running tests

All commands are run from the `rust/` directory.

Unit tests (no external binaries required):

```sh
just test-derivation
```

Integration tests automatically build op-program and kona-host before running:

```sh
# Run all integration tests (op-program + kona-host)
just test-derivation-integration

# Run only op-program integration tests
just test-derivation-op-program

# Run only kona-host integration tests
just test-derivation-kona-host
```

Integration tests accept exit code 1 (claim mismatch) because the Rust framework doesn't replicate EIP-4788/EIP-2935 system contract state changes during block execution. The important thing is that derivation completes without crashing.

## Authoring tests

Tests use a builder API centered on `DerivationTest`:

```rust
use derivation_tests::harness::DerivationTest;

let mut test = DerivationTest::new();

// Build L1 and L2 blocks
let l1_ref = test.l1.emit_empty_block();
test.l2.set_epoch(&test.l1.block(l1_ref));
let l2_ref = test.l2.build_empty_block();

// Encode as a batch and submit on L1
let batch = test.singular_batch_calldata(&[l2_ref], &l1_ref.into());
test.l1.emit_block_with_batches(vec![batch]);

// Verify the output root
let root = test.expected_super_root();
assert_ne!(root, alloy_primitives::B256::ZERO);
```

### Key concepts

**DeterministicConfig** -- every field is pinned (chain IDs, keys, timestamps, hardforks). The same config always produces identical chains. All OP Stack hardforks are active from genesis (time 0).

**L1ChainBuilder** -- produces L1 blocks containing batch submissions (calldata or blobs) and system config update events.

**L2ChainBuilder** -- executes L2 blocks via revm with real state transitions, including deposit transactions derived from L1.

**Batch encoding** -- supports singular batches (`singular_batch_calldata`) and span batches (`blob_span_batch`), with channel compression (zlib/brotli) and frame splitting.

**TestServers** -- starts L1 RPC, L2 RPC, and Beacon API servers on ephemeral ports. Required for op-program and kona-host integration tests.

### Adding a new scenario

1. Create a function that builds a `DerivationTest` with the desired chain structure.
2. Assert the super root against a golden hash constant (see below).
3. Optionally add an `#[ignore]` integration test that runs the scenario through op-program or kona-host.

See `tests/simple.rs` for examples covering empty blocks, singular batches, blob batches, user transfers, multi-block epochs, and system config updates.

## Golden super root values

Tests pin expected super root hashes as hardcoded constants in `tests/simple.rs`. This catches silent regressions in batch encoding, block execution, state root computation, or output root calculation -- a consistently wrong root would pass a pure determinism check.

The golden values are framework-internal. Cross-implementation validation happens via the `#[ignore]` integration tests that run op-program and kona-host against the same chains.

### Updating golden values

When an intentional change modifies the derivation output (e.g. a new hardfork, changed genesis config, or updated deposit tx encoding):

1. Run: `just test-derivation -- --nocapture 2>&1 | grep "super root"`
2. Each test prints its computed super root to stderr before asserting.
3. Copy the new values into the `EXPECTED_*` constants in `tests/simple.rs`.
4. Re-run `just test-derivation` to confirm all tests pass.

## Architecture

```
src/
├── config.rs        Deterministic test configuration (chain IDs, keys, hardforks)
├── harness/         Test runner, assertions, op-program/kona-host integration
├── l1/              L1 chain construction (blocks, batch submissions, blobs)
├── l2/              L2 chain construction with EVM execution (revm)
├── batch/           Batch encoding (singular, span), channel compression, framing
├── roots.rs         Output root and super root computation
├── server/          JSON-RPC (L1, L2) and Beacon API servers
└── state/           In-memory state DB, trie root computation, Merkle proofs
```
