# Upstream assumptions the OP Stack breaks

The precondition question, and the places where the OP Stack breaks an assumption upstream
reth and revm rely on. Both the reth update procedure
([UPDATING-RETH](UPDATING-RETH.md), step 7) and its review
([`docs/ai/reth-update-review.md`](../docs/ai/reth-update-review.md)) use it.

## The question

Ask this on every consensus-adjacent upstream change, whether adapting to it or
reviewing it:

> **Which upstream precondition makes this change safe, and does the OP Stack hold it?**

Upstream can prove a refactor behaviour-preserving because L1 validation rejects the inputs
that would distinguish old from new. The OP Stack deliberately admits some of those inputs.
When it does, an upstream "no-op refactor" — with no upstream discussion, no failing test
and no diff in our tree — is an OP behaviour change.

The tell is upstream code that reasons about a *range* rather than a *branch*: a `min`/`max`
or clamp removed, an arm deleted as unreachable, a `saturating_*` relaxed to plain
arithmetic, or a comment of the form "X is always ≤ Y here".
Each one encodes an assumption. Name the assumption, then check it against the list below.

## Where OP breaks upstream preconditions

**1. Deposit transactions skip `validate_env` entirely.**
`OpHandler::validate_env` (`rust/op-revm/src/handler.rs`) returns `Ok(())` for
`DEPOSIT_TRANSACTION_TYPE` before reaching `validation::validate_env`, because deposits
are pre-verified on L1 (
[specs.optimism.io/protocol/deposits](https://specs.optimism.io/protocol/deposits.html)).
Read the upstream body to keep this current; as of `revm` 43 it means a deposit is not
checked for:

- `prevrandao` present (Merge+) and `excess_blob_gas` present (Cancun+) on the block
- chain id present and matching (EIP-155)
- the EIP-7825 transaction gas-limit cap (`tx.gas_limit() > cfg.tx_gas_limit_cap()`), which
  upstream also skips once EIP-8037 state gas is enabled
- gas price ≥ basefee, and priority fee ≤ max fee
- the transaction type being enabled by the spec (2930 / 1559 / 4844 / 7702)
- every EIP-4844 blob check (max fee, non-empty, versioned-hash prefix, count)
- a non-empty EIP-7702 authorization list
- `tx.gas_limit() > block.gas_limit()`
- the EIP-3860 initcode size limit
- `nonce == u64::MAX`

**Asymmetry worth knowing:** `OpHandler` does *not* override `validate_initial_tx_gas`, so
deposits still get the intrinsic-gas, EIP-7623 floor and EIP-8037 regular-gas checks. Skipping
`validate_env` is not the same as skipping validation — check which one a given upstream
assumption is enforced by. The cap does not bind a deposit's *execution* either:
`OpHandler::tx_gas` splits a deposit's gas limit with the cap lifted, so the whole limit is
regular gas and the EIP-8037 reservoir is empty.

**2. Deposits take a separate `validate_against_state_and_deduct_caller` branch.**
Same file. Relative to the upstream body, the deposit arm skips
`validate_account_nonce_and_code_with_components` (so no EIP-3607 caller-has-code
rejection and no nonce-match check) and skips `ensure_enough_balance` (so no
`LackOfFundForMaxFee`; the balance update saturates instead). It additionally credits
`tx.mint()`.

**3. A failed deposit is not an excluded transaction.** `OpHandler::catch_error` routes
deposit failures to `catch_error_failed_deposit`, which reverts to the first checkpoint,
still bumps the nonce, still persists the mint, and reports `OpHaltReason::FailedDeposit`.
Upstream logic that assumes "invalid transaction ⇒ no state change / not in the block"
does not hold.

**4. Pre-Regolith deposits skip the block-gas admission check.**
`validate_block_gas` in `rust/alloy-op-evm/src/block/mod.rs` returns early for them.

**5. The post-exec transaction type (`0x7D`) never reaches revm.**
`OpBlockExecutor::execute_transaction_without_commit` synthesises its result and returns
before `evm.transact`. Upstream invariants of the form "every transaction in a block was
executed" do not hold.

**6. A per-account storage root is consensus.**
Upstream precondition: provider storage-root reads serve only RPC, so upstream tolerates
divergence there that it would never accept in the state root. Since Isthmus the header's
`withdrawalsRoot` is the L2ToL1MessagePasser storage root, which op-reth computes through
`StorageRootProvider::storage_root` (`rust/op-reth/crates/consensus/src/validation/isthmus.rs`)
when building (`rust/op-reth/crates/evm/src/build.rs`) and validating
(`rust/op-reth/crates/node/src/engine.rs`) a block. Any upstream change to that method, to
the providers that implement it (including overlay code it shares with `storage_proof` and
`storage_multiproof`), or to how trie tables are persisted (partial persistence, state
masking, overlay construction) is consensus-affecting for OP even when every upstream
state-root test passes. Test it with persistence and state masking both engaged: a low
persistence threshold, a nonzero `num_state_masking_blocks`, and enough blocks to trigger
persistence. `with_persistence_threshold(0)` turns masking off, and under the default
threshold a short test never persists at all.

## How to check one

For preconditions 1–5:

1. From the upstream diff, name the assumption in one sentence
   ("this branch was unreachable because L1 rejects `gas_limit > cap` pre-execution").
2. Find where upstream *enforces* it — usually a `return Err(...)` in
   `revm-handler/src/validation.rs` or `pre_execution.rs`. Read that function at the new
   version rather than trusting the list above.
3. Trace whether an OP transaction can reach the changed code without passing that check:
   grep `rust/op-revm/src/handler.rs` for the enforcing function and for
   `DEPOSIT_TRANSACTION_TYPE`.
4. If it can, treat the change as consensus-affecting regardless of how upstream labelled
   it. It needs a test, red against a deliberately-broken variant and green against the
   real code ([UPDATING-RETH](UPDATING-RETH.md), Procedure step 7).
