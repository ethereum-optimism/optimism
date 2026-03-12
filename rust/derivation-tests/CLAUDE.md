# derivation-tests

Cross-implementation derivation pipeline test framework for the OP Stack. Builds deterministic L1/L2 chains in-process and verifies derivation across op-program and kona-host.

## DSL Style Guide

**Read [DSL_STYLE_GUIDE.md](DSL_STYLE_GUIDE.md) before writing or modifying tests.** It defines the principles, naming conventions, and patterns for the scenario-oriented DSL.

Key rules:
- Tests describe timelines, not protocol mechanics
- Use DSL methods for happy-path and setup; drop to low-level (`test.l1`, `test.l2`) only for adversarial/edge-case parts
- Method names must be unambiguous about L1 vs L2 (e.g., `derive_l2_block`, not `derive_block`)
- New methods follow existing verb families: `advance_*`, `derive_*`, `submit_*`
- Config structs with `Default` for optional parameters, not boolean flags

## Running Tests

From the `rust/` directory:

```sh
# Unit tests
just test-derivation

# Integration tests (builds op-program and kona-host automatically)
just test-derivation-integration
```

## Golden Values

Tests in `tests/simple.rs` pin expected super root hashes. When an intentional change modifies derivation output:

1. Run: `just test-derivation -- --no-capture 2>&1 | grep "super root"`
2. Copy new values into `EXPECTED_*` constants
3. Re-run to confirm

## Architecture

- `src/harness/dsl.rs` — DSL types (`BatchConfig`, `BlockBuilder`)
- `src/harness/test.rs` — `DerivationTest` with DSL methods
- `src/harness/runner.rs` — op-program and kona-host execution
- `src/l1/`, `src/l2/` — Chain builders (low-level layer)
- `src/batch/` — Batch encoding (singular, span), channel compression
- `src/server/` — JSON-RPC and Beacon API test servers
- `src/state/` — In-memory state DB, trie roots, Merkle proofs
