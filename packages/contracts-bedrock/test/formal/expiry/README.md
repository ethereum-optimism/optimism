# Formal verification: per-message interop expiry

This directory is a review aid for the interop message-expiry change (PR #23259):
- the L2ToL2CrossDomainMessenger records send timestamps and marks undeliverable messages expired;
- the source chain's L1CrossDomainMessenger relays "not relayed" facts;
- `SuperchainETHBridge.refundETH` pays back expired sends.

Several independent tools each check part of the safety argument. Every subdirectory has its own
README with exact statements, assumptions, bounds and run commands. Review coverage differs by
layer and by version; see **Layers, pins and reviews** at the end.

## What is shown, and under which conditions

Safety here means: no message is both relayed on its destination and refunded on its source. It
is **conditional**:
- **Lean** (`lean/`) proves it for messages from standard source chains, in every execution
  reachable from the modeled genesis, for any number of chains, messages and lockbox joins and any
  length of time. The hypotheses are `SafeConfig`, `HashInjective` (an idealized injective hash),
  `ChainIdUnique`, `Init` and `GovInit`; see **Assumptions**.
- **Quint** (`quint/`) checks the same properties on a finite model (four chains, three messages,
  times up to day 20) for every execution of up to 10 steps, or 9 for `safeResendRestarts`.
- **The rollout model** (`rollout/`) checks deployment orderings, rollbacks and misconfigurations on
  a finite model (three chains, three messages, times up to 16 days) for every execution of up to
  11 events. It holds only under the activation conditions AC1–AC5, which the code does not all
  enforce: see **Governance and deployment obligations**.
- **The bytecode and test layers** check, each within its stated domain, the contract-level steps
  the protocol models take as given.

Nothing here is a composed, end-to-end proof from deployed bytecode to the protocol theorem; the
links between layers are stated correspondences, listed per layer.

## Read this first: which design is verified

The **exporter design** landed on the PR #23259 branch at `5992028e08`:
1. `UndeliveredMessageExporter` at `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`. It is 0x...0030 (it was
   0x..2E in earlier drafts); the models refer to it only by name. Its genesis proxy
   has no implementation before the upgrade.
2. `L1CrossDomainMessenger.relayUndeliveredMessage` trusts only that sender, behind an INTEROP feature
   gate.
3. P_contract = 8 days.
4. Go and kona reject W_protocol > 7 days.
5. The messenger keeps its unsafe-target rule (the L2CrossDomainMessenger and, since `3b8d14c4ef`,
   the L2ToL1MessagePasser) as defense in depth.

The contracts tip is `448d31ad19`; since `c7c51d79e2` only the messenger's `version` string changed
("2.0.0"). Every layer names the commit it checked (table at the end).

The protocol models also show that the **earlier design** (`37b44c48c7`, which trusted 0x..23 as
the L2 sender) double-spends if any chain that is, or becomes, part of the lockbox once ran a
messenger without the target rule. Before the upgrade, anyone can relay a message to the
L2CrossDomainMessenger that pre-stages a forged `relayUndeliveredMessage` withdrawal. See
`cex_messengerTrusted` in Lean and `messengerTrustedPrestaged` in Quint.

## Properties and what checks them

| Property | Lean (unbounded) | Quint / Apalache (bounded) | Halmos (symbolic, bytecode) | EVM-Lean (bytecode) | Others |
|---|---|---|---|---|---|
| **NoDoubleSpend**: never both relayed on the destination and refunded on the source | proved for standard sources, every reachable execution, under the hypotheses above | holds for every execution of up to 10 steps (`safe`, `safeNoTargetRule`, `safeNoMargin`, `safeShorterWindow`) or 9 steps (`safeResendRestarts`); each unsafe instance double-spends within 6–11 steps | (local properties only) | (`expireMessage` only) | Foundry invariants (randomized); `rollout/` (orderings, 11 events) |
| **ExpiredImpliesNeverRelayable** | proved for standard sources, for all extensions of the execution | checked to the same depths | — | — | — |
| **Expiration provenance** (`NoForgedFact`, `OnlyDestinationCanExport`) | `onlyDestinationCanExport`: a message expired on a standard source has a matching export on its destination; `noForgedFact`: every modeled deposit came from a standard chain's exporter | `NoForgedFact`: every accepted expiration was exported by the message's destination | exporter hash binding | `evm-lean-exporter/`: exact preimage, chain ID 1 only | Kontrol |
| **RefundImpliesExpired, AtMostOneRefund** | proved (standard chains) | checked | refund iff and single use | `refundETH` (chain ID 1): `refunded[H]` stored only after reading `expiredMessages(H)` true and `refunded[H]` false | Kontrol (single use from two closed proofs); Foundry invariants |
| **Exporter silent before upgrade** (derived activation) | proved within the model, no axioms; historical inertness of the address is assumed | modeled via `upgrade` | — | — | — |
| **Target rule** (defense in depth: an upgraded 0x..23 never initiates a withdrawal) | lemma; safety is proved without it | `MessengerSilentAfterUpgrade` | for the covered target classes (codeless accounts, listed predeploys, a probe, one re-entry by export or send); 0x..23 as a target excluded; L2ToL1MessagePasser included | — | Kontrol (self-target proof not closed); Foundry invariants |
| **`expireMessage` auth + strict boundary + effects** | abstract guard | abstract guard | iff, full frame, unbounded overflow | soundness on bytecode, under two getter summaries | Kontrol |
| **`relayUndeliveredMessage` checks + exact deposit** | abstract guard | abstract guard incl. fake callers | iff, deposit envelope, listed topologies | `evm-lean-l1cdm/`: under getter summaries; unproxied code only | Kontrol (fixed pointer stand-ins) |
| **`refundETH` preimage binding, single use, balances** | `isBridge` abstraction | abstract | iff, balances, composed with real `sendETH` | soundness at the store point (chain ID 1); payout checked concretely | Kontrol |
| **P_contract ≥ W_protocol required** | `cex_periodBelowWindow` (W=7, P=6) | `periodBelowWindow` | contract constant ≥ 7 days | constant read from bytecode | Dafny lemmas; Go/Rust window differential (overrides above 7 days rejected) |
| **Messenger behaviour unchanged for allowed targets** | — | — | — | — | equivalence with develop `c2fe2a991b` on the shared surface (see below) |

The equivalence row (`hevm/`) is narrower than "unchanged": shared getters are equivalent at
bytecode level; `sendMessage` and `relayMessage` agree on outcome, return data, outside calls and
storage for message lengths 0/4/37/100/128 (send) and 0/4/37/100 (relay) with one fixed relay
caller. Storage is compared as one symbolic key per mapping (S-map) and as non-hash-derived raw
slots (S-all); events are covered by a concrete fuzz, and gas is not compared. The hevm engine
itself covers only `sendMessage` with an empty message; the rest is Halmos.

Each subdirectory README is authoritative for its exact statement, bounds and assumptions.

## Governance and deployment obligations

`rollout/` names the activation conditions under which no modeled ordering violates safety, and
shows that dropping any one of them gives a double spend. Several are process, not code:

| Condition | Requires | Enforced by code | Left to process |
|---|---|---|---|
| **AC1** (window) | from the moment a destination's exporter first has an implementation, every relay on it is judged with W ≤ P, forever after | op-core, kona and op-interop-filter reject overrides above 7 days; kona's getter also falls back to 7 days (since `d36e37862b`) | every node, filter and **absolute prestate** that judges the destination's relays must include the cap before its exporter goes live (BC3) |
| **AC2** (no resend-capable messenger) | no resend-capable messenger implementation (1.3.x, with `resendMessage`) is ever live on a chain with expiry | NUT path: the L2ContractsManager's semver guard | deployment assumption: 1.3.x is never shipped to a production chain, so no rollback target with `resendMessage` exists. Requires the Lagoon re-snapshot (BC1) |
| **AC3** (exporter provenance) | on every current or future lockbox member, the exporter address only ever ran the standard implementation or nothing | genesis proxy without implementation; the L2ContractsManager sets only the standard implementation | the member's L2 ProxyAdmin owner can set anything; `ETHLockbox.authorizePortal` does not check a joiner's history (BC4) |
| **AC4** (chain IDs) | no chain that can relay messages sent from a lockbox member shares the L2 chain ID of a member or of another such chain | the OPCM migrator rejects duplicates among the chains it migrates | lockbox joins and every chain in members' dependency sets (BC4) |
| **AC5** (L1 implementation) | no L1CrossDomainMessenger that trusts 0x..23 is ever installed, even briefly | the tip's check (c) | never deploy an L1CrossDomainMessenger built from a PR-branch commit before `1086b6de3e` |

**Deployment assumption (AC2): no resend-capable messenger implementation is ever live on a chain
with expiry.** The 1.3.x messenger, which has `resendMessage`, is never shipped to a production
chain, so there is no resend-capable implementation to roll back to. The rollout model shows why the
assumption is needed (BC2): `ProxyAdmin.upgrade` has no version check, and 1.3.1's `resendMessage`
checks only `sentMessages`, which 2.0.0 still writes, so a 1.3.1 implementation live after a refund
would let anyone resend the message and relay it within a new window (`govMessengerDowngrade`, 11
events). The code is unchanged; the assumption carries this.

**Launch blocker (BC1): the locked Lagoon bundle ships the 1.3.1 messenger.** `fork_lock.toml` pins
`lagoon` to `fa9974a2`, whose bundle deploys messenger 1.3.1, bridge 1.0.1 and no exporter. Shipped
as locked, it would put a resend-capable messenger on production chains and break AC2; it would also
disable expiry, permanently for messages sent while 1.3.1 is live (they have no timestamp). The
bundle must be re-snapshotted (`just nut-snapshot-for lagoon`) before Lagoon ships. This is the
concrete step that makes AC2 hold.

`rollout/` also lists BC3 (the cap holds only for software that includes it; kona-host's fallback
dependency set is local key 8, which `SuperFaultDisputeGame.addLocalData` cannot supply, so a
cluster outside the prestate's embedded registry has no working on-chain proof) and BC4 (the
lockbox join path skips the migrator's checks).

## Named assumptions

These were pointed out by an independent explicit-state model, written from the code by a modeler
without access to these models. That model and its results are **not in this tree** and cannot be
reproduced from it; it is cited only as the source of these assumptions, each of which is checked
or stated in the layers named.
1. **Upgrade invariant.**
   - No implementation of a source messenger may re-emit `SentMessage` for a hash that already has a
     send timestamp. In practice no resend-capable implementation (1.3.x) is ever live on a chain with
     expiry (AC2; BC1 is the step that ensures it).
     A re-emitted event restarts the relay window; this is `resendNoRestart` / `cex_resendNoRestart`.
   - Every relay path on the destination sets `successfulMessages`.
   - Upgrades and rollbacks preserve storage layout, nonces, send timestamps and the successful,
     expired and refunded markers, with their meanings. The models keep these maps across upgrades;
     a bridge whose layout change makes refunded hashes read as unrefunded refunds twice (mutant
     K50 in `mutation/`), and no executed layer catches that.
   - The exporter's hashing mirrors the messenger's. Halmos checks both against the same formula
     (`check_export_binding`, `check_send_effects_and_frame`); Kontrol and `evm-lean-exporter/`
     (chain ID 1) prove the exporter's preimage.
   - Rollouts, rollbacks and downgrades are modeled in `rollout/`.
2. **Zero margin is safe.** P = W is still safe with the strict `>`, so the 1-day margin is defense
   in depth. See Lean `safe_variants` (P = W = 7), Quint `safeNoMargin`, rollout
   `rolloutWindowAtP`, and Dafny `ExpiredAtExportNeverValid` with `W <= P`.
3. **The window cap in kona.** `depset.rs` rejects an override above 7 days when it deserializes a
   dependency set, and since `d36e37862b` `get_message_expiry_window` falls back to 7 days for a
   larger value set in memory. The effective window therefore never exceeds the cap, however the
   dependency set was built. The remaining gap is BC3 (above).

## The off-chain rule: existing Dafny model

The protocol rule that an executing message is valid only while
`initTimestamp <= execTimestamp <= initTimestamp + messageExpiryWindow` already has a formal model in
the repo:
- `op-supernode/dafny-models/Interop.dfy`: `ValidExecutingMessage` (~:516-529) and the imperative
  check returning `ErrMessageExpired` (~:1747-1750);
- `Types.dfy`: `MESSAGE_EXPIRY_WINDOW`.

The Lean and Quint relay rule (`t ≤ e + W`) matches its boundary exactly. Ours omits `init ≤ exec`,
which only makes the models more permissive. `dafny/` proves numerical expiry rejection against that
model's own definitions. The correspondence between the model's events and windows and the
contracts' message and `sentAt` (including day-to-second scaling) is argued, not proved; timestamp
binding uses the model's existing LogsDB/BlockInfo axioms.

## Assumptions, all layers

Each layer states which of these it assumes and which it checks:
- **Timestamps:** block timestamps are monotone per chain, and the source records the initiating
  block's timestamp.
- **Finality:** finalized withdrawals reflect canonical history.
- **Hashing:** Lean assumes a globally injective abstract hash over its whole preimage domain. That
  is an idealization of keccak's collision resistance, not a property keccak has. Kontrol's built-in
  lemmas additionally assume a keccak of symbolic bytes never equals a concrete value and is not
  within 32 of 0 or 2^256. Halmos and hevm use their own symbolic hash models.
- **Governance** (see also AC1–AC5):
  - only chains that ran the standard predeploys are authorized in a lockbox;
  - every current or future lockbox member's exporter address stays empty or runs the standard
    exporter throughout its history (Lean `exporterGovernance`, rollout AC3). Each member's L2
    governance (its L2 ProxyAdmin owner) can upgrade its own exporter and so forge facts for any
    destination; it is trusted not to. This is comparable to the trust in the shared ETHLockbox: a
    member's L2 governance can already make arbitrary withdrawals from it by changing its own L2
    state (the lockbox's own check compares the portals' L1 ProxyAdmin owners, a separate role);
  - no chain, inside the lockbox or not, shares a chain ID with a standard chain (Lean
    `ChainIdUnique`, rollout AC4). Uniqueness among members alone is not enough: a non-member with
    a member's chain ID can deliver that member's messages (`rollout/`, `cexNonMember`);
  - authorized portals' SystemConfigs name their real L1CrossDomainMessenger;
  - no implementation was set at the exporter address before the upgrade;
  - no resend-capable messenger implementation (1.3.x) is ever live on a chain with expiry (AC2;
    requires the Lagoon re-snapshot, BC1), and no L1CrossDomainMessenger that trusts 0x..23 (AC5).
- **Protocol window:** W ≤ P from the moment a destination's exporter goes live, in every node,
  filter and absolute prestate that judges its relays (AC1), and W never later rises above P.
  Lean and Quint fix W from genesis.
- **Addresses:** no EOA or aliased L1 address equals the exporter's or the messenger's address.
- **History:** pre-Bedrock (legacy) withdrawals carry no `relayUndeliveredMessage` from the
  exporter; the models' genesis is Bedrock-era. Message nonces are fresh.
- **Standard code:** failed L1 relays and failed L2 deposits replay only the original message
  (versioned hash), and the cross-domain messengers' `xDomainMessageSender` and the L1
  messenger's self-target rule behave as in the standard code. Lean lists these as obligations
  discharged outside it.
- **Model scope:** Quint and rollout have a single lockbox; Lean and Quint do not model leaving a
  lockbox or later changes to the windows.

### Harness domains of the bytecode and test layers

| Layer | Domain and abstractions |
|---|---|
| `halmos/` | one call or a short fixed sequence from a symbolic state; patched SELFDESTRUCT; finite byte-length sets and canonical payloads; listed relay targets and L1 topologies; refund balances and amounts ≤ 2^198, recipients outside the harness accounts |
| `kontrol/` | messages exactly 600 bytes; `sentAt < 2^64`, nonce below exhaustion; symbolic mock answers with fixed pointer stand-ins on L1; address exclusions; gas not modeled |
| `evm-lean/` | `expireMessage` bytecode under two `ReturnsAddress` getter summaries; per-key slot distinctness; deposit history supplied externally |
| `evm-lean-l1cdm/` | stable getter summaries (return data < 2^32, independent of gas and balances); composition only for unproxied code |
| `evm-lean-bridge/`, `evm-lean-exporter/` | EVMLean's fixed chain ID 1; bridge store theorem needs non-empty executing code; exporter calldata < 2^63; callees not proved |
| `hevm/` | concrete chain ID, one funded relay caller, `msg.value < 2^120`, structured mocks, the lengths above |
| `invariants/` | randomized: one source, three chains, one shared monotone clock; abstract L1 hop with membership and INTEROP assumed; CI budget per the suite's inline config |
| `dafny/` | registered chains, model validity; timestamp binding uses LogsDB/BlockInfo axioms and a correct-frontier hypothesis |

## Open obligations

Everything a layer says did not close, was not run, or is bounded:
- **Kontrol** (`kontrol/`): these proofs did not close: `prove_relayMessage_selfTarget_neverCallsL2CDMOrPasser`
  (argued per function instead), `prove_relayUndeliveredMessage_symbolicPortalChain` and its
  witness (fully symbolic pointers; `prove_relayUndeliveredMessage_spec` closes with fixed pointer
  stand-ins), and `prove_refundETH_singleUse` (single use follows from
  `prove_refundETH_preimageBinding` and `prove_refundETH_alreadyRefundedReverts`, for an immediately
  repeated call). `symbolicPortalChain`'s assertion is also too strong under pointer aliasing and
  needs rewriting. Two of the suite's KEVM lemmas lack side conditions their soundness argument uses;
  adding them needs a re-run of the bridge proofs. The exporter any-selector proof and the bridge
  non-refund cases are narrower than first stated (surviving calls to 0x..07 only; false → true
  changes of `refunded` only). Any-calldata exclusivity for the messenger and the
  L1CrossDomainMessenger is left to Halmos.
- **Halmos** (`halmos/`): reachability is two steps from the deployed state (one step from fully
  symbolic storage for the bridge); three steps exceeded 40 minutes. Arbitrary target code,
  deeper re-entry and other L1 pointer-alias topologies are not explored.
- **hevm** (`hevm/`): the hevm engine handles only an empty-message `sendMessage`; relay exceeded
  16 GB. Events and gas are not compared symbolically.
- **EVM-Lean** (`evm-lean*/`): soundness only, no completeness or liveness theorem; the bridge and
  exporter proofs are for chain ID 1; `evm-lean-l1cdm` does not compose through the deployed proxy
  and does not prove its callees; the bridge proves the store point, not final-state single use;
  concrete execution witnesses use `native_decide` (`evm-lean-kernel/`).
- **Quint** (`quint/`): `safeResendRestarts` is checked to 9 steps, shallower than the 11-step
  counterexample of its mitigation-off instance `resendNoRestart`; deeper safe-instance runs were
  stopped.
- **Rollout** (`rollout/`): bounded at 11 events (`SafetyFull` at 10), not a completeness
  threshold; no unbounded counterpart. A late L1 INTEROP enable on a source is argued, not
  checked.
- **Mutation campaign** (`mutation/`): K44 (gas forwarding) and K50 (bridge storage layout)
  survive every executed layer; the Lean, Quint and EVM-Lean columns are statement mappings, not
  per-mutant reruns.
- **Foundry invariants** (`invariants/`): Medusa was not run; expected-to-fail suites are skipped
  in CI (their deterministic witnesses run).
- **Dafny** (`dafny/`): the step from the executing message to the contract's `sentAt` (event and
  window correspondence) is outside the model; accepting executions for timestamp binding are not
  exhibited.

## Layers, pins and reviews

R1 is a fresh-context reviewer; R2 and R3 are independent model-based reviewers.

| Directory | Tool | Checked at | Externally reviewed (R1, R2, R3) | Changed since, no external review |
|---|---|---|---|---|
| `lean/` | Lean 4 protocol model and proof | design of `5992028e08` (no bytecode) | v1, v2; v2.2–v2.3 (this round; see its log) | — |
| `quint/` | Quint model; Apalache and simulation | design of `5992028e08` | v1, v2; v2.1 (this round; see its log) | — |
| `rollout/` | Quint rollout model | code cited at `e1b3903ab8` | v1 | the `SafetyFull` depth-10 result |
| `dafny/` | lemmas over the existing supernode Dafny model | constants of `5992028e08` | v1 | v2 (review fixes), v3 (non-vacuity witnesses) |
| `evm-lean/` | EquiVM/EVMLean proof of `expireMessage` bytecode | `448d31ad19` | rounds 1 (`37b44c48c7`) and 2 (`5992028e08`) | round 3 (non-vacuity), retargets |
| `evm-lean-l1cdm/` | the same, for `relayUndeliveredMessage` | `c7c51d79e2` | round 2 (at `52ff613e14`) | retarget to `c7c51d79e2` |
| `evm-lean-bridge/` | the same, for `refundETH` | `c7c51d79e2` | round 1 (before `c7c51d79e2`) | non-vacuity rounds, retarget |
| `evm-lean-exporter/` | the same, for `exportUndeliveredMessage` | `c7c51d79e2` | round 1 (at `c7c51d79e2`) | — |
| `evm-lean-kernel/` | kernel-checking measurements | — | no proofs to review | — |
| `halmos/` | Halmos symbolic checks on bytecode, with an enforced mutant suite | `c7c51d79e2` | rounds 1–3 (exporter design, before `c7c51d79e2`) | round 4 (mutation-campaign fixes), retarget |
| `kontrol/` | Kontrol (KEVM) proofs | `c7c51d79e2`; `relayUndeliveredMessage` proofs re-run on the reviewed harness | round 1 (this round; see its log) | — |
| `invariants/` | Foundry stateful invariant harness; runs in contracts-bedrock CI | `c7c51d79e2` | round 1 | round 2 (non-vacuity), retarget |
| `hevm/` | hevm and Halmos equivalence with develop | `448d31ad19` | v1 (fixes in v2) | v4 (S-all scope), retarget |
| `mutation/` | cross-layer mutation campaign | `c7c51d79e2` (re-runs at `557e7691e9`) | rounds 1, 2 | gap closures after round 2 |
| `window-differential/` | Go/Rust window differential; Go/Rust tests run in CI | `96a08ba3e4` | v1 | v2 (a Go and a Rust code review were applied) |

A retarget re-runs the same statements on the newer contracts (renamed errors, the exporter's new
address, the version string); each layer's README records what changed.
