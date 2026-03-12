# DSL Style Guide

This guide defines the principles and conventions for the derivation-tests DSL. Follow it when writing tests, adding new DSL methods, or evolving existing ones. The goal is tests that read as scenario descriptions, not protocol implementation details.

## Core Principle: Tests Describe Timelines

A test should read as "what happened" — not "how to make it happen." The reader should understand the scenario without knowing how L1 blocks are encoded, how epochs work, or how batch submission is wired internally.

```rust
// Good: reads as a scenario description
let mut test = DerivationTest::new();
test.advance_l1(2);
test.derive_empty_l2_block();
test.submit_batch_with(BatchConfig::singular_calldata());
let root = test.finalize();
```

```rust
// Bad: protocol mechanics leak into the test
let mut test = DerivationTest::new();
test.l1.emit_empty_block();
test.l1.emit_empty_block();
let l1_block = test.l1.block_at(1).unwrap().clone();
test.l2.set_epoch(&l1_block);
let block_ref = test.l2.build_empty_block().unwrap();
let batch = test.singular_batch_calldata(&[block_ref], &l1_block);
test.l1.emit_block_with_batches(vec![batch]);
test.l1.emit_empty_block();
let root = test.expected_super_root();
```

## Two-Layer Architecture

The DSL has two layers. Tests choose which layer to use based on what they're testing.

**DSL layer** (`test.advance_l1()`, `test.derive_l2_block()`, `test.submit_batch()`, etc.):
Scenario-oriented methods on `DerivationTest`. Handles epoch management, block tracking, batch encoding, and chain sealing automatically. Use this for happy-path tests and for the setup portion of adversarial tests.

**Low-level layer** (`test.l1`, `test.l2`, `test.singular_batch_calldata()`, etc.):
Direct access to `L1ChainBuilder` and `L2ChainBuilder`. No automatic epoch tracking, no pending block management, no defaults. Use this when the test needs to construct something the DSL intentionally doesn't support — invalid batches, malformed frames, wrong batcher addresses, manual epoch control.

### Mixing Layers

Tests can freely mix both layers. The pattern for adversarial tests is: **DSL for setup, low-level for the adversarial part**.

```rust
fn test_wrong_batcher_address() {
    let mut test = DerivationTest::new();

    // DSL: set up a normal chain
    test.advance_l1(2);
    test.derive_empty_l2_block();

    // Low-level: inject adversarial data
    let fake_batch = Bytes::from(vec![0x00, 0xDE, 0xAD, 0xBE, 0xEF]);
    test.l1.emit_block_with_raw_txs(vec![fake_batch]);

    // Assert the framework handles it
    let root = test.expected_super_root();
    assert_ne!(root, B256::ZERO);
}
```

When dropping to the low-level layer mid-test, add a brief comment explaining why (e.g., "drop to low-level for adversarial part"). This makes the boundary visible.

## Naming Conventions

### L1/L2 Disambiguation

Method names must be unambiguous about which chain they operate on. If a method could plausibly apply to either chain, include `l1` or `l2` in the name.

| Method | Why it works |
|--------|-------------|
| `advance_l1(count)` | Explicitly L1 |
| `derive_empty_l2_block()` | Explicitly L2 |
| `derive_l2_block()` | Explicitly L2 |
| `submit_batch()` | Unambiguous — batches are always submitted to L1 |
| `advance_epoch()` | Unambiguous — epochs are an L2-relative concept |
| `finalize()` | Unambiguous — seals the entire test |

When adding a new method, ask: "If I read this name with no context, could I confuse which chain it targets?" If yes, add the chain qualifier.

### Verb Conventions

- **`advance_*`** — Move time or state forward. No new data, just progression.
- **`derive_*`** — Build L2 blocks (the "derivation" in derivation-tests).
- **`submit_*`** — Encode and post data to L1.
- **`finalize`** — Seal the test scenario and compute the expected result.

New methods should follow these verb families. Avoid introducing synonyms (`create_l2_block`, `build_l2_block`, `add_l2_block`) when an existing verb fits.

## Defaults and Configuration

### Sensible Defaults

DSL methods should do the right thing with zero configuration. The most common case should require the fewest arguments.

- `submit_batch()` uses the default `BatchConfig` (span batch via blobs) — the most common production format.
- `derive_empty_l2_block()` auto-sets the epoch to L1 genesis if no epoch has been set.
- `finalize()` emits the trailing L1 block that the derivation pipeline needs.

### Config Structs with `Default`

When a method needs optional configuration, use a config struct that implements `Default`. Provide `*_with(config)` as the configurable variant and a bare method as the default variant.

```rust
// Default: span batch via blobs
test.submit_batch();

// Configured: singular batch via calldata
test.submit_batch_with(BatchConfig::singular_calldata());
```

Add convenience constructors on the config struct for common combinations (`BatchConfig::singular_calldata()`, `BatchConfig::span_calldata()`). Make them `const fn` where possible.

Do not add boolean parameters or feature flags to existing methods. If a method needs a new option, add it to the config struct.

## Adding New DSL Methods

Before adding a new method, consider:

1. **Is this a scenario concern or a protocol concern?** If it's about what happened in the test (e.g., "a user deposited ETH"), it belongs in the DSL. If it's about how the protocol encodes something (e.g., "the channel uses brotli compression"), it belongs in the low-level layer or as a config option.

2. **Does it compose with existing methods?** New methods should work with the existing `derive → submit → finalize` flow. They should not require callers to manage internal state (pending blocks, epoch tracking) manually.

3. **Does it handle its own preconditions?** DSL methods should assert on bad state rather than silently producing wrong results. Use `assert!` or `expect()` with clear messages. Example: `submit_batch()` panics if there are no pending blocks rather than submitting an empty batch.

4. **Is the name unambiguous?** Apply the L1/L2 disambiguation rule. Use existing verb families.

### The BlockBuilder Pattern

For methods that need to collect multiple inputs before producing a result, use a builder that holds `&mut DerivationTest` and consumes itself on `.build()`:

```rust
test.derive_l2_block()
    .with_funded_transfer(recipient, amount)
    .with_tx(custom_op_tx)
    .build();
```

The builder pattern keeps `DerivationTest` borrowed for the minimum duration. After `.build()`, the borrow is released and the caller can call `submit_batch()` normally. New builder methods should follow the `with_*` prefix convention and take `mut self` → return `Self`.

## Test Smells

### Comment Explaining DSL Calls

If a DSL method needs a comment to explain what it does, the method is poorly named or too low-level. Raise the abstraction.

```rust
// Smell: comment explains what advance_l1 does
// Emit 2 empty L1 blocks to create epochs
test.advance_l1(2);

// Good: the method name speaks for itself
test.advance_l1(2);
```

Comments are appropriate for explaining **why** (e.g., "need 6 blocks to fill one epoch") or for marking the boundary between DSL and low-level code.

### Reimplementing DSL Logic in Tests

If a test manually tracks pending blocks, manages epochs, or encodes batches when the DSL could do it, the test is working at the wrong layer. Either use the DSL or extend it.

```rust
// Smell: manually tracking what the DSL tracks for you
let block_ref = test.l2.build_empty_block().unwrap();
let batch = test.singular_batch_calldata(&[block_ref], &epoch);
test.l1.emit_block_with_batches(vec![batch]);

// Good: let the DSL handle it
test.derive_empty_l2_block();
test.submit_batch_with(BatchConfig::singular_calldata());
```

### Unwrap Chains in Tests

Tests should not contain chains of `.unwrap()` on DSL operations. DSL methods use `expect()` internally with descriptive messages. If a test needs to handle errors (e.g., testing that something fails), drop to the low-level layer where the Result types are exposed.

## Extending vs. Replacing

When evolving the DSL:

- **Extend**: Add new methods alongside existing ones. The low-level API (`test.l1`, `test.l2`) is stable and public — never remove it.
- **Replace in tests**: When a new DSL method covers a pattern that tests were doing manually, rewrite those tests to use the DSL method. Don't leave two styles in the same test file.
- **Don't duplicate**: If two methods do similar things with slightly different config, use the config struct pattern instead of separate methods.
