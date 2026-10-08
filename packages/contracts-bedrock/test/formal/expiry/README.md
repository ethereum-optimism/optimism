# Formal verification: per-message interop expiry

This directory is a review aid for the interop message-expiry change (PR #23259):
- the L2ToL2CrossDomainMessenger records send timestamps and marks undeliverable messages expired;
- the source chain's L1CrossDomainMessenger relays "not relayed" facts;
- `SuperchainETHBridge.refundETH` pays back expired sends.

Several independent tools each check part of the safety argument. Every subdirectory has its own
README with exact statements, assumptions, bounds, run commands and a review log. Every layer was
reviewed by a fresh-context Claude reviewer and by Codex gpt-6-astra and gpt-6.1-sol. Each finding
was triaged against the code and either fixed or recorded with a reason.

## Read this first: which design is verified

The **exporter design** landed on `karl/message-expiry-refunds` at `5992028e08`:
1. `UndeliveredMessageExporter` at `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`. It is 0x...0030 (it was
   0x..2E in earlier drafts); the models refer to it only by name. Its genesis proxy
   has no implementation before the upgrade.
2. `L1CrossDomainMessenger.relayUndeliveredMessage` trusts only that sender, behind an INTEROP feature
   gate.
3. P_contract = 8 days.
4. Go and kona reject W_protocol > 7 days.
5. The messenger keeps its unsafe-target rule (the L2CrossDomainMessenger and, since `3b8d14c4ef`,
   the L2ToL1MessagePasser) as defense in depth.

The protocol models (Lean, Quint) prove this design safe. They also show that the **earlier design**
(`37b44c48c7`, which trusted 0x..23 as the L2 sender) double-spends if any chain that is, or becomes,
part of the lockbox once ran a messenger without the target rule. Before the upgrade, an attacker can
relay a message to the L2CrossDomainMessenger that pre-stages a forged `relayUndeliveredMessage`
withdrawal. See `cex_messengerTrusted` in Lean and `messengerTrustedPrestaged` in Quint.

The bytecode-level layers (Halmos, Kontrol, EVM-Lean, hevm) were first written against
`37b44c48c7` and are being re-targeted to the tip. Each layer's README names the commit and bytecode
it checked.

## Properties and what checks them

| Property | Lean (unbounded) | Quint / Apalache (bounded) | Halmos (symbolic, bytecode) | EVM-Lean (bytecode refinement) | Others |
|---|---|---|---|---|---|
| **NoDoubleSpend**: never both relayed on the destination and refunded on the source | proved for any `SafeConfig` (exporter design) | checked to depth 15 in 5 safe instances; 10 mitigation-off instances double-spend | (local properties only) | (`expireMessage` only) | Foundry invariants (pending) |
| **ExpiredImpliesNeverRelayable** | proved, for all extensions of the execution | checked | — | — | — |
| **NoForgedFact / OnlyDestinationCanExport** | proved | checked | export hash binding | — | — |
| **RefundImpliesExpired, AtMostOneRefund** | proved (standard chains) | checked | refund iff + single use | pending | — |
| **Exporter silent before upgrade** (derived activation) | proved, no axioms | modeled via `upgrade` | — | — | — |
| **Target rule** (defense in depth: an upgraded 0x..23 never initiates a withdrawal) | lemma; safety is proved without it | `MessengerSilentAfterUpgrade` | for all targets, including re-entrant relay targets; the L2ToL1MessagePasser case is pending in the code | — | Kontrol (pending) |
| **`expireMessage` auth + strict boundary + effects** | abstract guard | abstract guard | iff, full frame, unbounded overflow | proved on bytecode (sound; complete modulo gas) | Kontrol (pending) |
| **`relayUndeliveredMessage` three checks + exact deposit** | abstract guard | abstract guard incl. fake callers | iff, deposit envelope, topologies | in progress | Kontrol (pending) |
| **`refundETH` preimage binding, single use, balances** | `isBridge` abstraction | abstract | iff, balances, composed with real `sendETH` | in progress | — |
| **P_contract ≥ W_protocol required** | `cex_periodBelowWindow` (W=7, P=6) | `periodBelowWindow` | contract constant ≥ 7 days | constant parameterized | Go/Rust window differential (pending) |
| **Messenger behaviour unchanged for allowed targets** | — | — | — | — | hevm equivalence (pending) |

"Pending" means a worker is still running, or the code it targets has not landed. Each subdirectory
README is authoritative for its exact statement.

## Named assumptions confirmed by an independent model

An independent modeler (code-only, with its own explicit-state model and no access to these models)
found no violation in 2.7M–5.8M states, and a double spend under each mutation. It also pointed out
assumptions that should be named explicitly:
1. **Upgrade invariant.**
   - No implementation of a source messenger may re-emit `SentMessage` for a hash that already has a
     send timestamp. That includes a rollback to an implementation with `resendMessage`. A re-emitted
     event restarts the relay window; this is `resendNoRestart` / `cex_resendNoRestart`.
   - Every relay path on the destination sets `successfulMessages`.
   - The exporter's hashing mirrors the messenger's (`hevm/` and Halmos check both against the same
     formula).
   - Rollouts, rollbacks and downgrades are modeled in `rollout/`.
2. **Zero margin is safe.** P = W is still safe with the strict `>`, so the 1-day margin is defense
   in depth. See Lean `safe_variants` (P = W = 7), Quint `safeNoMargin`, and Dafny
   `ExpiredAtExportNeverValid` with `W <= P`.
3. **kona enforces the 7-day cap only at deserialization** (`depset.rs` `deserialize_override_window`).
   `get_message_expiry_window` does not clamp, and a `DependencySet` built in memory (tests,
   `arbitrary`) bypasses the cap. This is a trust boundary: every production depset must go through
   serde. Today the registry and the oracle fallback in `boot.rs` both do.

## The off-chain rule: existing Dafny model

The protocol rule that an executing message is valid only while
`initTimestamp <= execTimestamp <= initTimestamp + messageExpiryWindow` already has a formal model in
the repo:
- `op-supernode/dafny-models/Interop.dfy`: `ValidExecutingMessage` (~:516-529) and the imperative
  check returning `ErrMessageExpired` (~:1747-1750);
- `Types.dfy`: `MESSAGE_EXPIRY_WINDOW`.

The Lean and Quint relay rule (`t ≤ e + W`) matches its boundary exactly. Ours omits `init ≤ exec`,
which only makes the models more permissive. `dafny/` proves the off-chain half of the expiry
argument against that model's own definitions (in progress).

## Assumptions, all layers

Each layer states which of these it assumes and which it checks:
- **Timestamps:** block timestamps are monotone per chain, and the source records the initiating
  block's timestamp.
- **Finality:** finalized withdrawals reflect canonical history.
- **Hashing:** keccak is collision-resistant. This is idealized as injectivity over the relevant
  preimages.
- **Governance:**
  - only chains that ran the standard predeploys are authorized in a lockbox;
  - chain IDs are unique among lockbox members;
  - authorized portals' SystemConfigs name their real L1CrossDomainMessenger;
  - no implementation was set at the exporter address before the upgrade;
  - each cluster chain's L2 governance can upgrade its own exporter, which would let it forge facts for
    any destination. This is the same trust as the shared ETHLockbox, whose portals must share the
    proxy admin owner.
- **Protocol window:** W ≤ P, and the protocol window rule is enforced on a destination before its
  exporter goes live.
- **Addresses:** no EOA or aliased L1 address equals the exporter's or the messenger's address.

## Layout

| Directory | Tool | Status |
|---|---|---|
| `lean/` | Lean 4 protocol model and proof | done (v2.1), reviewed twice |
| `quint/` | Quint model; Apalache and simulation | v2.1; Apalache at depth 15 running |
| `halmos/` | Halmos symbolic checks on bytecode | v2, reviewed twice; v3 in progress |
| `evm-lean/` | EquiVM/EVMLean bytecode proof of `expireMessage` | done; review running |
| `evm-lean-l1cdm/`, `evm-lean-bridge/` | the same, for `relayUndeliveredMessage` and `refundETH` | in progress |
| `kontrol/` | Kontrol (KEVM) proofs | in progress |
| `invariants/` | Foundry stateful invariant harness | in progress |
| `hevm/`, `window-differential/` | hevm equivalence; Go/Rust window rule differential | in progress |
| `dafny/` | lemmas over the existing supernode Dafny model | in progress |
| `rollout/` | rollout orderings and misconfigurations (Quint) | in progress |
