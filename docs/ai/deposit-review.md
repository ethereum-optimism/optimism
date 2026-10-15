# Reviewing deposit derivation and execution

This area guide covers deposit transactions from the L1 deposit contract to the L2 receipt.
It pairs with the [`deposit-reviewer`](../../.claude/agents/deposit-reviewer.md) agent.

Read [spec-driven-review.md](spec-driven-review.md) in full before this guide.
Read [execution-layer.md](execution-layer.md) for the execution-layer model.

This guide owns deposit scope, code navigation, and domain checks.
The shared guide owns the review process, evidence contract, output, and mapping validation.
Neither guide defines protocol behavior.

## Scope

Review these behaviors:

- `TransactionDeposited` log decoding and the derived deposit fields.
- Source-hash computation for every deposit domain.
- Deposit transaction encoding and decoding, including the `0x7E` typed envelope.
- L1 attributes deposit construction and its calldata for each fork.
- Network upgrade transactions that derivation generates as deposits.
- The unconditional mint and its position relative to execution rollback. The mint is the only path that creates L2 ETH with no L2 source.
- Deposit execution rules that differ from a type-2 transaction.
- Nonce handling, including the increment on failure.
- Gas limits, intrinsic gas, the block gas pool, and fork gas caps.
- Fee exemptions, including base fee, priority fee, L1 cost, operator fee, and refunds.
- Deposit receipt fields and their fork-gated presence.
- `OptimismPortal` deposit creation, address aliasing, and guaranteed gas.

Do not review these areas unless changed behavior crosses their boundary:

- Batch decoding and validation.
- Channel and frame handling.
- Non-deposit transaction execution.
- Withdrawal proving and finalization.
- Dispute game logic.

## Specification sources

Start from [Deposits](https://specs.optimism.io/protocol/deposits.html).
The base page states the wire format, source hashes, execution rules, receipt fields, and the deposit contract.

Fork documents amend these rules.
Search every applicable `specs/protocol/<fork>/` directory for touched concepts.
Do not limit the search to `deposits.md`.

At minimum, check these amendments:

| Fork | Deposit effect | Specification |
| --- | --- | --- |
| Regolith | Disables `isSystemTx`, meters real gas, adds `depositNonce` | [Regolith](https://specs.optimism.io/protocol/regolith/overview.html) |
| Canyon | Makes `depositNonce` and `depositReceiptVersion` always present | [Canyon](https://specs.optimism.io/protocol/canyon/overview.html) |
| Ecotone | Changes L1 attributes calldata | [Ecotone L1 attributes](https://specs.optimism.io/protocol/ecotone/l1-attributes.html) |
| Isthmus | Adds upgrade deposits, operator fee, and EIP-6110 request exclusion | [Isthmus derivation](https://specs.optimism.io/protocol/isthmus/derivation.html), [Isthmus execution engine](https://specs.optimism.io/protocol/isthmus/exec-engine.html) |
| Jovian | Adds upgrade deposits and the DA footprint gas rule | [Jovian derivation](https://specs.optimism.io/protocol/jovian/derivation.html), [Jovian execution engine](https://specs.optimism.io/protocol/jovian/exec-engine.html) |
| Karst | Enables EIP-7825 and exempts deposits from the cap | [Karst overview](https://specs.optimism.io/protocol/karst/overview.html), [Karst derivation](https://specs.optimism.io/protocol/karst/derivation.html) |
| Lagoon | Adds post-exec transactions beside deposits | [Lagoon post-exec](https://specs.optimism.io/protocol/lagoon/post-exec.html) |

Each fork that adds a network upgrade transaction adds deposit rules.
Read the full upgrade-transaction section of that fork, including every field value and the intent string.

Upgrade-deposit rules do not always live under `specs/protocol/<fork>/`.
Recent forks build their upgrade deposits from a shared bundle format instead of per-fork constants.
Read these two pages as well, for every fork from Karst onward:

- [L2 upgrades, execution](https://specs.optimism.io/protocol/l2-upgrades-1-execution.html) for the bundle format, the intent prefix rule, the sender rule, and the gas allocation.
- [Interop derivation](https://specs.optimism.io/interop/derivation.html) for the Lagoon activation deposits and the dependency-set conditional deposits.

[Guaranteed gas market](https://specs.optimism.io/protocol/guaranteed-gas-market.html) states the L1 gas purchase and the per-block gas bound.

One upgrade deposit carries a non-zero mint.
Treat the bundle data as a mint source, not as static configuration.

## Specification-to-code map

This mapping belongs in the monorepo because implementation names change independently from protocol behavior.
Each entry names a Go package, a Rust crate, or a contract, plus one or more anchor symbols in it.
Find each anchor with a repository grep, then follow affected callers and callees.
The anchors are entry points, not a complete list of relevant files.

Go packages are paths relative to the repository root.
Rust entries use crate names. Contract entries use contract names.

This repository owns the canonical Rust deposit types.
It does not vendor them. `op-alloy-consensus` holds the wire format and the source-hash rules, and Kona imports both.

| Behavior | Specification | Go | Rust | Contracts |
| --- | --- | --- | --- | --- |
| `0x7E` wire format, encode, decode, hash | [Deposited transaction type](https://specs.optimism.io/protocol/deposits.html#the-deposited-transaction-type) | `op-core/types`: `DepositTx`, `MarshalBinary`, `UnmarshalDepositTx`, `Hash` | `op-alloy-consensus`: `TxDeposit`, `OpTxType`; `op-revm`: `DepositTransactionParts`, `DEPOSIT_TRANSACTION_TYPE` | `OptimismPortal2`: `TransactionDeposited` |
| Typed-envelope dispatch, including every fallback | [Deposited transaction type](https://specs.optimism.io/protocol/deposits.html#the-deposited-transaction-type) | `op-core/types`: `UnmarshalDepositTx`, the JSON transaction decoder | `op-alloy-consensus`: `OpTxEnvelope`, `OpPooledTransaction`, `fallback_decode`; `reth-optimism-primitives`: the `Decodable2718` table for `OpTransactionSigned` | n/a |
| Source hash domains | [Source hash computation](https://specs.optimism.io/protocol/deposits.html#source-hash-computation) | `op-node/rollup/derive`: `UserDepositSource`, `L1InfoDepositSource`, `UpgradeDepositSource` | `op-alloy-consensus`: `DepositSourceDomain`, `UserDepositSource`, `L1InfoDepositSource`, `UpgradeDepositSource`. Kona holds no second implementation. | n/a |
| Deposit log decode and the derived fields | [User-deposited transactions](https://specs.optimism.io/protocol/deposits.html#user-deposited-transactions) | `op-node/rollup/derive`: `UnmarshalDepositLogEvent`, `unmarshalDepositVersion0`, `MarshalDepositLogEvent` | `kona-protocol`: `decode_deposit`, `DepositError` | `OptimismPortal2`: `depositTransaction`, the `opaqueData` packing |
| Per-block deposit list | [Kinds of deposited transactions](https://specs.optimism.io/protocol/deposits.html#kinds-of-deposited-transactions) | `op-node/rollup/derive`: `UserDeposits`, `DeriveDeposits` | `kona-derive`: `StatefulAttributesBuilder`, `derive_deposits` | n/a |
| Transaction order in the block | [Kinds of deposited transactions](https://specs.optimism.io/protocol/deposits.html#kinds-of-deposited-transactions) | `op-node/rollup/derive`: `PreparePayloadAttributes` | `kona-derive`: `prepare_payload_attributes` | n/a |
| L1 attributes deposit and calldata | [L1 attributes deposited transaction](https://specs.optimism.io/protocol/deposits.html#l1-attributes-deposited-transaction), [Ecotone L1 attributes](https://specs.optimism.io/protocol/ecotone/l1-attributes.html) | `op-node/rollup/derive`: `L1InfoDeposit`, `L1InfoDepositBytes`, `L1InfoDepositerAddress`, the per-fork marshal functions | `kona-protocol`: `L1BlockInfoTx`, `try_new_with_deposit_tx`, `encode_calldata`, `decode_calldata`, `L1_INFO_DEPOSITOR_ADDRESS` | `L1Block`: the `setL1BlockValues` functions, `DEPOSITOR_ACCOUNT` |
| Network upgrade deposits | The upgrade-transaction section of each fork specification, [L2 upgrades](https://specs.optimism.io/protocol/l2-upgrades-1-execution.html) | `op-node/rollup/derive`: `UpgradeTransactions`, `UpgradeGas`, `nutBundleForFork`, the Lagoon activation transactions | `kona-hardforks`: the per-fork upgrade types; `op-alloy-consensus`: the NUT bundle conversion | `NetworkUpgradeTxns` |
| Upgrade bundle data | [L2 upgrades](https://specs.optimism.io/protocol/l2-upgrades-1-execution.html) | `op-core/nuts/bundles`: the per-fork NUT bundle JSON, shared by both clients | the same JSON files | n/a |
| Validation and authorization | [Validation and authorization](https://specs.optimism.io/protocol/deposits.html#validation-and-authorization-of-deposited-transactions) | n/a | `op-revm`: `validate_env`, `validate_initial_tx_gas` | `OptimismPortal2`: `depositTransaction` |
| Mint, caller deduction, nonce | [Execution](https://specs.optimism.io/protocol/deposits.html#execution), [Nonce handling](https://specs.optimism.io/protocol/deposits.html#nonce-handling) | n/a | `op-revm`: `validate_against_state_and_deduct_caller` | `OptimismPortal2`, `ETHLockbox` |
| Gas result, refunds, system transactions | [Execution](https://specs.optimism.io/protocol/deposits.html#execution), [Regolith](https://specs.optimism.io/protocol/regolith/overview.html) | n/a | `op-revm`: `last_frame_result`, `refund`, `tx_gas`; `alloy-op-evm`: `validate_block_gas` | `ResourceMetering`: `metered`, `useGas` |
| Fee exemptions | [Execution](https://specs.optimism.io/protocol/deposits.html#execution) | n/a | `op-revm`: `reward_beneficiary`, `reimburse_caller` | n/a |
| Post-exec refund eligibility | [Lagoon post-exec](https://specs.optimism.io/protocol/lagoon/post-exec.html), [Lagoon SDM](https://specs.optimism.io/protocol/lagoon/sdm.html) | n/a | `alloy-op-evm`: `post_exec_settlement_deltas`, `sanitize_producer_refund`, `verifier_post_exec_refund_for_tx` | n/a |
| Failure semantics | [Execution](https://specs.optimism.io/protocol/deposits.html#execution) | n/a | `op-revm`: `execution_result`, `catch_error_failed_deposit` | n/a |
| Deposit receipt fields | [Deposit receipt](https://specs.optimism.io/protocol/deposits.html#deposit-receipt), [Canyon](https://specs.optimism.io/protocol/canyon/overview.html) | `op-core/types`: `depositReceiptRLP`, `MarshalBinary` | `op-alloy-consensus`: `OpDepositReceipt`; `alloy-op-evm`: the receipt build; `reth-optimism-evm`: `strip_deposit_nonce`; `reth-optimism-rpc`: the RPC receipt build | n/a |
| Receipts-root encoding | [Deposit receipt](https://specs.optimism.io/protocol/deposits.html#deposit-receipt), [Canyon](https://specs.optimism.io/protocol/canyon/overview.html) | `op-core/types`: `Receipts.EncodeIndex` | `reth-optimism-consensus`: `calculate_receipt_root_optimism`; `kona-executor`: `compute_receipts_root` | n/a |
| Address aliasing | [Address aliasing](https://specs.optimism.io/protocol/deposits.html#address-aliasing) | n/a | n/a | `AddressAliasHelper`: `applyL1ToL2Alias`, `undoL1ToL2Alias` |
| Guaranteed gas | [Guaranteed gas market](https://specs.optimism.io/protocol/guaranteed-gas-market.html) | n/a | n/a | `ResourceMetering` |
| Fault-proof execution environment | [Deposits](https://specs.optimism.io/protocol/deposits.html) | n/a | `kona-client`: the FPVM transaction environment; `kona-executor`: the block builder | n/a |

The fault-proof executor holds no separate deposit execution rules.
It uses `alloy-op-evm` and `op-revm`, so an execution finding usually affects both clients.

The receipts-root encoding is the exception.
Three implementations reproduce it: op-node in Go, `reth-optimism-consensus`, and `kona-executor`.
Each one applies the Regolith-to-Canyon nonce strip with its own fork predicate.
Compare all three when a change touches the deposit receipt.

Fork gates live in `alloy-op-hardforks` and in `op-node/rollup`.
`op-revm` maps each OP fork onto an L1 specification identifier.

Methods in `op-revm` carry `UPSTREAM-MIRROR` comments.
Each comment names the upstream `revm` function that the override copies.
Use those names to compare an override with its upstream source.

## Existing parity tests

Check whether a change needs a new case in one of these:

| Test | Anchor |
| --- | --- |
| Go and Rust upgrade-deposit bytes | `ops/scripts`: the interop deposits diff script and its two dump commands |
| Deposit gas parity against the proof | `rust/kona/tests/proofs`: `TestDepositGasParity` |
| `0x7E` encoding and hash against op-geth | `op-core/types`: `TestDepositTxMarshalBinaryDifferential`, `TestDepositHashParity` |
| Execution golden vectors | `revm-ee-tests`: the deposit and halted-deposit vectors |
| Karst cap exemption | `alloy-op-evm`: the deposit gas-cap test |
| Mint on a reverted deposit | `op-e2e/system/bridge`: `TestMintOnRevertedDeposit` |

## Inherited rules

The specification defines deposit execution by difference.
It states that a deposit is processed "exactly like a type-2 (EIP-1559) transaction, with the exception of" a listed set.

This form has produced real consensus bugs.
An upstream L1 change can be behavior-preserving on L1 and unsafe on the OP Stack, because an OP exception removes the L1 invariant that made the change safe.

Apply this rule when the specification inherits L1 behavior:

- Name the exact upstream rule set and its version.
- Enumerate the stated OP differences.
- For each difference, ask which L1 invariant it removes.
- Check whether the implementation still relies on that removed invariant.

Do not treat model knowledge of L1 behavior as the inherited rule set.
Read the upstream rule or its implementation at the reviewed revision.

## When to run this reviewer

Run this reviewer when a change can alter a behavior in the scope above.
A change in a mapped path or its dependencies is a strong signal.
Also run it for these changes:

- A new network upgrade transaction, or a change to an existing one.
- A change to the L1 attributes transaction or its calldata.
- A fork that changes gas caps, intrinsic gas, or the gas pool.
- A change to transaction type dispatch, RLP decoding, or the typed envelope.
- A change to receipt encoding, decoding, or hashing.
- A change to `OptimismPortal`, address aliasing, or resource metering.
- An upstream `revm`, `alloy`, or `reth` bump that touches transaction execution, gas, or receipts. See [reth-update-review.md](reth-update-review.md) for the drift method.
- A Kona client release or a prestate change that contains mapped changes.

## Deposit analysis

Trace a deposit from the L1 log through the L2 receipt.
Follow the shared guide to prove each implementation's behavior.
Add one output to the list it gives: the receipt fields that the path writes.

### Cross-component values

A deposit crosses derivation, execution, and the receipt.
A value that agrees inside one component can still disagree across the boundary.

Check that these values agree end to end:

- The derived `gas` and the gas the execution layer applies.
- The derived `mint` and the balance increase.
- The nonce used for contract address derivation and the `depositNonce` in the receipt.
- The gas the execution output reports and the gas the block pool subtracts.
- The transaction hash that each implementation computes for the same deposit.

## Domain checks

Apply these checks to any changed deposit behavior.

### Mint and value

- `mint` is `uint256` in the wire format. Check every narrower internal type.
- The mint applies before execution and survives a failed deposit.
- The rollback point is after the mint, not before it.
- A failed `value` transfer produces a failed receipt, not a dropped deposit.

### Nonce

- The `from` nonce increments once, including on failure.
- The receipt records the nonce from before execution.
- A rollback must not leave a second increment or lose the first.
- Contract creation uses the recorded nonce, not zero.

### Gas

- Deposits buy gas on L1 and never refund it.
- The deposit gas limit is exempt from the EIP-7825 transaction gas cap after Karst.
- Intrinsic gas still applies.
- The block gas pool still applies.
- Pre-Regolith gas accounting differs, and `isSystemTx` changes it again.
- Check the cap arithmetic order. `min(limit - intrinsic, cap)` and `min(limit, cap) - intrinsic` differ only when the cap binds.

### Fees

- No base fee, priority fee, L1 cost, or operator fee applies to a deposit.
- The fee recipient receives nothing from a deposit.
- Total base fee accounting does not change.
- A deposit is not eligible for a Lagoon post-exec refund.
- A post-exec payload entry that targets a deposit makes the block invalid.

### Receipt

- `depositNonce` and `depositReceiptVersion` presence is fork-gated, and the version must be `1`.
- The consensus receipt encoding and the receipt hash must match the fork rule.
- The deposit receipt RLP carries the status, `cumulativeGasUsed`, the bloom, the logs, `depositNonce` and `depositReceiptVersion`. The Go receipt struct also holds the L1 fee fields, which never reach it. A client difference in those fields misses the receipts root but is still a cross-client divergence.
- `cumulativeGasUsed` reflects metered gas after Regolith.

### Decoding and dispatch

- The `0x7E` prefix identifies a deposit. No other prefix may produce one.
- A prefixless or untyped body must not reach the deposit decoder through a fallback path.
- Field count, field order, and `to` as null against zero are distinct cases.
- Non-canonical RLP, truncation, and trailing bytes each need a defined outcome.
- Network, pooled, and consensus decode paths can differ. Check each one.

### Derivation

- Each source-hash domain uses the stated constant and the stated preimage.
- The log index is the index in the whole block, not in the transaction.
- The L1 attributes deposit is always first, and user deposits follow.
- User deposits appear only in the first L2 block of an epoch.
- Upgrade deposits appear only on the activation block, in the stated order.
- The intent string of each upgrade deposit must match the specification byte for byte.

### Contracts

- Address aliasing adds the constant with `uint160` overflow.
- A contract creation deposit with a non-zero `to` must revert.
- Guaranteed gas must bound the total deposit gas in one L1 block.

## Sibling paths in deposits

The shared guide requires a rule-list diff against sibling paths.
In this area the sibling pairs are:

- The L1 attributes deposit against a user deposit, in the same client.
- An upgrade deposit against a user deposit, in the same client.
- The derivation-side field construction against the execution-side field use.
- The consensus decode path against any network or pooled decode path.
- The same rule list in op-node against the matching list in Kona.
- The execution rule set in op-reth against the rule set in the fault-proof executor.

## Adversarial boundaries

A deposit is permissionless. Any L1 account can create one.
The attacker controls `to`, `value`, `mint`, `gas`, `data`, and the creation flag, within the portal limits.

Check these boundaries:

- Zero, maximum, and one-past-maximum `gas`, including the guaranteed gas bound.
- A gas limit above and below the EIP-7825 cap.
- `mint` and `value` at `uint256` maximum, and at every internal type boundary.
- Empty `data`, and `data` at the L1 size bound.
- Contract creation with empty init code.
- A deposit that targets a precompile.
- A deposit that reverts, runs out of gas, or fails the value transfer.
- A deposit in the first and in a later block of an epoch.
- A deposit in the fork activation block itself.

## Dismissal checks

Apply the shared dismissal checks first.
These deposit cases extend them:

- The portal already bounds the value, so the L2 path cannot reach it.
- The difference appears only before the fork that activates the rule.
- A later guard rejects the input before the suspected operation.
