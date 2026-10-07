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

The contracts at this PR's base (`37b44c48c7`) implement the **earlier design**:
- `relayUndeliveredMessage` trusts the L2ToL2CrossDomainMessenger (0x..23) as the L2 sender;
- the expiry period is 7 days;
- the protocol window has no cap.

The protocol models (Lean, Quint) show that this design **double-spends** if any chain that is, or
becomes, part of the lockbox once ran a messenger without the target rule. Before its upgrade, an
attacker could relay a message to the L2CrossDomainMessenger that pre-stages a forged
`relayUndeliveredMessage` withdrawal for a future message hash. See `cex_messengerTrusted` in Lean
and `messengerTrustedPrestaged` in Quint.

The **exporter design** is pending on `karl/message-expiry-refunds`, and that is what the protocol
models prove safe:
1. an `UndeliveredMessageExporter` predeploy at 0x4200..002E, whose genesis proxy has no
   implementation before the upgrade;
2. L1 trusting only that sender;
3. an INTEROP gate on L1;
4. P_contract = 8 days;
5. W_protocol ≤ 7 days, enforced (rejected, not clamped) in op-core and kona.

The bytecode-level layers (Halmos, Kontrol, EVM-Lean, hevm) currently check the code at
`37b44c48c7`. They read the trusted sender and the expiry constant from the contracts, so they can
be re-targeted once the change lands.

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
  - no implementation was set at 0x..2E before the upgrade.
- **Protocol window:** W ≤ P, and the protocol window rule is enforced on a destination before its
  exporter goes live.
- **Addresses:** no EOA or aliased L1 address equals 0x..2E or 0x..23.

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
