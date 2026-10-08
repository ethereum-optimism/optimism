# Cross-layer mutation campaign: per-message interop expiry

This directory answers one question about the verification layers next to it: **which layer catches which
plausible implementation mistake, and does any mistake slip past every layer?**

Each mutant is one small edit to the real contracts of the exporter design. The edits cover the expiry
boundary, the send timestamp, the relay's bookkeeping, `expireMessage`'s authorization, the exporter's hash and
payload, the three checks of `L1CrossDomainMessenger.relayUndeliveredMessage`, `refundETH`, and the target rules.
Every layer that is cheap enough was run on every mutant. The other layers are mapped by reasoning from their
exact statements.

## Files

| File | What it holds |
|---|---|
| `mutants.tsv` | The catalogue: id, file, which existing mutant it reuses (if any), the mistake, and the `sed` edit. |
| `run.sh` | Reproduces the matrix for the executed layers (unit tests, invariants, Halmos, hevm-harness equivalence). |
| `.gitignore` | Ignores `results/` (one log per mutant and layer, and `matrix.tsv`). |

## Base

- **Contracts under test:** `src/` at `c7c51d79e2`, the style-guide tip of the exporter design (`EXPIRY_PERIOD =
  8 days`, the exporter at `Predeploys.UNDELIVERED_MESSAGE_EXPORTER`).
- **Checkout:** the formal branch at `e0ffb33a31`, whose `src/` equals `c7c51d79e2`'s. The Halmos directory was
  taken at its later state with `halmos.toml` and the phase-2 harness fixes (the `ReachL1CDMHalmos` INTEROP flag
  fix in particular: before it, every L1CDM mutant would have looked dead to that check for the wrong reason).
- **Later tip.** PR #23259 later moved to `448d31ad19`. Its only `src/` change is the messenger's version string,
  which no mutant touches and no layer here depends on; its test change (two relay fuzz tests skip forge's own
  addresses) fixes a flake that did not occur at the fixed seed used here (the baseline passed). The `sed` edits
  apply unchanged to it. Some layers were later re-targeted to that tip (the `expireMessage` EVM-Lean proof and the
  hevm bytecode). Their statements are the same, so the mapping below holds for either pin.
- **Layers that changed after the campaign.** Kontrol was committed after the campaign ran (`786c48383f`); it is
  mapped by its statements, like EVM-Lean (see "Layers"). The committed Halmos suite renamed the bridge's phase-2
  check (`check_reach_bridge_step`); the bridge's phase 2 decided no verdict here (it was
  stopped for K43 and K50).

## The question in one table

Legend: **K** caught; **·** survived; **n/a** the layer does not execute or state anything about that code;
**K(setUp)** the suite's setUp rejects the mutated code because it asserts a documented assumption;
**pin** only the bytecode pin breaks (the theorem statement would still be true of the mutated code);
**stmt** the theorem statement becomes false for the mutated code. Lean/Quint, EVM-Lean and Kontrol are mapped
by reasoning from their statements, not run (see "Layers").

**K(witness)** only the suite's deterministic witness tests fail (a path they assert no longer works), no
invariant does; **excl.** the change is outside the hevm harness's compared surface on purpose (`expireMessage`,
`EXPIRY_PERIOD`, unsafe targets), and only its bytecode pin (`gen-bytecodes.sh --check`) notices it. **pin†** the `expireMessage`
EVM-Lean statement uses `P_contract`, which `evm-lean/scripts/regen.sh` rewrites from the bytecode, so the
regenerated statement holds with the mutated period (read with the literal 691,200 kept, it would be "stmt").
Every mutant also changes `snapshots/semver-lock.json`'s init-code hash, which the repo's snapshot check flags as
a change, not as a semantic catch.

| ID | Mistake | Reuses | Protocol effect | Unit | Inv | Halmos | hevm eq. | Lean / Quint | EVM-Lean | Kontrol |
|---|---|---|---|---|---|---|---|---|---|---|
| K01 | expireMessage accepts t == sentAt + P (strict > becomes >=) | halmos M1, invariants M1 | margin | K | K | K | excl. | Lean `safe_variants` (`≥` at P = 8 is safe); `cex_nonStrict`, Quint `expireGeNoMargin` need P = W | stmt | K |
| K02 | P_contract = W_protocol = 7 days (margin dropped) | new | margin | K | · | K | excl. | `safe_variants` P = W; `safeNoMargin` | pin† | · |
| K03 | P_contract = 6 days, below the 7-day protocol window | new | double spend | K | K(setUp) | K | excl. | `cex_periodBelowWindow`; `periodBelowWindow` | pin† | K |
| K04 | sentAt + P computed unchecked (wraps for sentAt >= 2^256 - P) | new | none reachable | · | · | K | excl. | — | stmt | · |
| K05 | sendMessage does not record sentMessageTimestamps | new | liveness | K | K | K | K | — | pin | · |
| K06 | sendMessage records timestamp 1 instead of block.timestamp | halmos M13 | double spend | · | K | K | K | mechanism of `cex_resendNoRestart` / `resendNoRestart` | pin | · |
| K07 | relayMessage does not set successfulMessages | halmos M24 | double spend | K | K | K | K | named assumption only | pin | · |
| K08 | relayMessage has no replay check | invariants M3 | double delivery | K | K | K | K | — | pin | · |
| K09 | expireMessage drops its msg.sender == L2CrossDomainMessenger check | new | none reachable | K | · | K | excl. | — | stmt | K |
| K10 | expireMessage drops its xDomainMessageSender == otherMessenger check | new; halmos M7 is a different edit, see README | double spend | K | K | K | excl. | — | stmt | K |
| K11 | expireMessage checks msg.sender against the wrong predeploy (L2ToL1MessagePasser) | new | liveness | K | K(witness) | K | excl. | — | stmt | K |
| K12 | relayMessage skips the unsafe-target check | halmos M2, invariants M5 | defense in depth | K | K | K | excl. | `safety_without_targetRule`; `safeNoTargetRule` | pin | K |
| K13 | unsafe-target rule drops the L2ToL1MessagePasser (0x..16) | halmos M3c | defense in depth | K | K | K | excl. | passer rule not modeled (lemma) | pin | K |
| K14 | unsafe-target rule drops the L2CrossDomainMessenger (0x..07) | new | defense in depth | K | K | K | excl. | `targetRule = false`; `safeNoTargetRule` | pin | K |
| K15 | relayMessage has no destination == block.chainid check | invariants M4 | double delivery | K | K | K | K | — | pin | · |
| K16 | exporter hashes with destination = _source (wrong chain id) | halmos M4 | liveness | K | K | K | n/a | — | stmt | K |
| K17 | exporter has no !successfulMessages[H] check | new; halmos M5 is the same behaviour | double spend | K | K | K | n/a | — | stmt | K |
| K18 | exporter sends undeliveredAt = 0 | halmos M6 | liveness | K | K | K | n/a | — | stmt | K |
| K19 | exporter sends undeliveredAt = block.timestamp + 2 days | new | double spend | K | K | K | n/a | as `cex_periodBelowWindow` (effective P = 6 d) | stmt | K |
| K38 | exporter hash swaps destination and source (H of a message from this chain) | new | liveness | K | K | K | n/a | — | stmt | K |
| K20 | exporter sends to the wrong L1 messenger (_sourceMessenger + 1) | halmos M15 | API shift (route = argument + 1) | K | K(witness) | K | n/a | — | stmt | K |
| K21 | relayUndeliveredMessage drops check (a), the reverse binding portal.systemConfig.l1CrossDomainMessenger == caller | new | double spend | K | n/a | K | n/a | `cex_noRealMessengerCheck`; `noRealMessengerCheck` | stmt | K |
| K22 | relayUndeliveredMessage drops check (b), the lockbox authorization | halmos M8 | double spend | K | n/a | K | n/a | `cex_noLockboxCheck`(`_fakePortal`); `noLockboxCheck` | stmt | K |
| K23 | relayUndeliveredMessage drops check (c), xDomainMessageSender == exporter | new | double spend | K | n/a | K | n/a | `cex_noSenderCheck`; `noSenderCheck` | stmt | K |
| K24 | check (c) trusts the L2ToL2CrossDomainMessenger instead of the exporter (earlier design) | halmos M34 | double spend | K | n/a | K | n/a | `cex_messengerTrusted`; `messengerTrustedPrestaged` | stmt | K |
| K25 | checks (a) and (c) evaluated in the opposite order | new | equivalent | · | n/a | · | n/a | same guard | pin | · |
| K26 | forwarded expireMessage deposit targets the wrong L2 predeploy (CrossL2Inbox) | new | liveness | K | n/a | K | n/a | — | stmt | K |
| K27 | forwarded expireMessage deposit asks for a minimum gas limit of 10,000 instead of 100,000 | new; halmos M9 is 100_001 | envelope only (the deposit still carries the relay overheads) | K | n/a | K | n/a | — (gas not modeled) | stmt | K |
| K28 | relayUndeliveredMessage has no INTEROP feature gate | halmos M33 | none (gate) | K | n/a | K | n/a | safe for every `interop` | stmt | K |
| K29 | forwards the L1 block.timestamp instead of _undeliveredAt | new | double spend | K | n/a | K | n/a | — (fact time chosen by whoever relays on L1; subsumes `cex_periodBelowWindow`) | stmt | K |
| K30 | L1CrossDomainMessenger._isUnsafeTarget no longer blocks address(this) | halmos M20 | double spend | K | n/a | K | n/a | `cex_noUnsafeTargetCheck` (Lean only) | pin | · |
| K31 | refundETH has no refunded[H] check | halmos M10, invariants M2 | double refund | K | K | K | n/a | — | stmt | K |
| K32 | refundETH does not set refunded[H] | halmos M12 | double refund | K | K | K | n/a | — | stmt | K |
| K33 | refundETH has no expiredMessages[H] check | invariants M7 | double spend | K | K | K | n/a | — | stmt | K |
| K34 | refundETH pays _to instead of _from | halmos M11 | misdirected refund | K | K | K | n/a | — (`isBridge`) | stmt | · |
| K35 | refundETH mints _amount + 1 (wrong amount) | new | over-mint | K | K | K | n/a | — (amounts) | stmt | K (edge) |
| K36 | refundETH hash preimage uses amount 0 instead of _amount | new | unbounded mint | K | K(witness) | K | n/a | — (`isBridge`) | stmt | K |
| K37 | refundETH hash preimage uses nonce + 1 | halmos M18 | API shift | K | K(witness) | K | n/a | — | stmt | K |
| K39 | expireMessage drops its sentAt != 0 check | new; review R1 | unbounded mint | K | K | K | excl. | — (`expire` guard fixes sentAt ≠ 0) | stmt | K |
| K40 | expireMessage compares the source's block.timestamp instead of _undeliveredAt | new; review R1 | double spend | K | K | K | excl. | — (fact time ignored) | stmt | K |
| K41 | check (a) reads the caller's own systemConfig() instead of its portal's | new; review R1 | double spend | K‡ | n/a | K‡ | n/a | `cex_noRealMessengerCheck`; `noRealMessengerCheck` | stmt | K‡ |
| K42 | check (b) reversed: the caller portal's lockbox must authorize this chain's portal | new; review R1 | double spend | K‡ | n/a | K‡ | n/a | `cex_noLockboxCheck_fakePortal`; `noLockboxCheck` | stmt | K‡ |
| K43 | refundETH emits RefundETH only for a nonzero amount | new; review R2 | event only | · | · | · (phase 2 stopped; raw ERROR) | n/a | — | pin | · |
| K44 | relayMessage forwards only half the remaining gas to the target | new; review R2 | liveness (gas) | · | · | · | excl. | — (gas) | pin | · |
| K45 | relayMessage does not call CrossL2Inbox.validateMessage | new; review R2 | forged delivery, unbounded mint | · | · | · | K | — (`relay` needs an initiating event) | pin | · |
| K46 | relayMessage marks successfulMessages after the target call instead of before | new; review R2 | local only (the window rule absorbs it) | · | · | K | · | — | pin | · |
| K47 | exporter hash ignores the nonce | new; review R2 | liveness | K | K | K | n/a | — | stmt | K |
| K48 | the forwarded expireMessage carries a different hash | new; review R2 | liveness | K | n/a | K | n/a | — | stmt | K |
| K49 | refundETH hash preimage uses _from as the sender | halmos M19 (review R2 | unbounded mint | K | K(witness) | K | n/a | — (`isBridge`) | stmt | K |
| K50 | a new state variable declared before refunded shifts its storage slot (upgrade layout) | new; review R3 | double refund after a later upgrade | · | · | · (phase 2 stopped; raw ERROR) | n/a | — (no storage) | stmt | · |
| K51 | expireMessage emits MessageExpired twice | new; review R3 | event only | · | K(witness) | · | excl. | — | pin | · |

K01–K38 are the original catalogue; K39–K51 were added after the first review round (see "Review log").
**K‡** caught only because the test stand-in for the other chain's messenger or portal lacks the getter the
mutant newly calls, so every call reverts, including honest ones; the forged word the mutant would accept is never
tried (see "Surviving every executed layer"). **· (phase 2 stopped; raw ERROR)**: phase 1 survived, and the bridge's phase-2 check was stopped by hand (its baseline does not
finish in 90 minutes on the shared host, and it observes neither events nor storage layout, which is what K43
and K50 change).

"Protocol effect" is what the mistake would do in the deployed system (argued per row in `mutants.tsv` terms
below and in "Equivalent and protocol-equivalent mutants"): *double spend* = the same message delivered on its
destination and refunded on its source; *margin* = consumes the 1-day margin between P_contract and W_protocol,
safe on its own; *defense in depth* = weakens the messenger's target rule, which the safety proof does not use;
*liveness* = refunds stop working, nothing is paid twice; *none reachable* = on every reachable state the
successful executions and their effects are the same (revert data and gas may differ); *API shift* = the same
payouts, reached through different arguments.

### Kills per layer (51 mutants)

| Layer | Caught | Survived | Not applicable or excluded |
|---|---|---|---|
| Unit tests | 42 (K41, K42 only through a fixture artifact) | K04, K06, K25, K43, K44, K45, K46, K50, K51 | — |
| Invariants | 30 (of them: K03 by a setUp assertion; K11, K20, K36, K37, K49, K51 by witness tests only) | K02, K04, K09, K43, K44, K45, K46, K50 | 13 L1CrossDomainMessenger mutants (the harness does not execute L1) |
| Halmos | 45 (K41, K42 only through a fixture artifact) | K25, K44, K45, K51 | K43, K50: phase 1 survived, phase 2 stopped (raw verdict ERROR) |
| hevm harness | 6 (K05–K08, K15, K45) | K46 | 14 messenger mutants outside the compared surface by design; 30 not the messenger |
| Kontrol (by statement) | 35 (K35 only at the liquidity edge; K41, K42 only through the same mock gap) | 16 | — |
| EVM-Lean (by statement) | 34 stmt | 17 pin (2 of them pin†) | — |

### Which checks caught each mutant

<details><summary>Failing tests and checks per mutant (unit: test contract and function; invariants: the first seed
that failed, invariants first, then witnesses)</summary>

- **K01**
  - unit: L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_withinWindow_reverts, L2ToL2_ExpireMessage_Test.test_expireMessage_atWindowEnd_reverts
  - inv(seed 1): TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expire_boundary, check_expire_iff, check_expire_iff_unbounded
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K02**
  - unit: L2ToL2_Uncategorized_Test.test_expiryPeriod_exceedsProtocolWindowByADay_succeeds
  - inv: -
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expiryPeriodIsCapPlusMargin
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K03**
  - unit: L2ToL2_Uncategorized_Test.test_expiryPeriod_exceedsProtocolWindowByADay_succeeds
  - inv(seed 1): Safety.setUp | witnesses: setUp, setUp, setUp
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_contractWindowCoversProtocolCap, check_expiryPeriodIsCapPlusMargin
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K04**
  - unit: -
  - inv: -
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expire_iff_unbounded
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K05**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Bridge_Integration_Test.test_refundETH_expireGasLimit_succeeds, Bridge_Integration_Test.test_refundETH_expireReplay_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_wrongPreimage_reverts, Bridge_RefundETH_Test.test_refundETH_destinationIsThisChain_reverts, Bridge_RefundETH_Test.test_refundETH_messageNotFromBridge_reverts, Bridge_RefundETH_Test.test_refundETH_senderRejectsETH_succeeds, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_succeeds, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_withinWindow_reverts, L2ToL2_ExpireMessage_Test.test_expireMessage_atWindowEnd_reverts, L2ToL2_SendMessage_Test.testFuzz_sendMessage_succeeds
  - inv(seed 1): Safety.invariant_sentTimestamps, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayAtWindowBoundary_succeeds, test_witness_relayBlocksExport_succeeds, test_witness_relayedExportCall_succeeds, test_witness_sentTimestampCorruptionDetected_succeeds, test_witness_targetRuleRejects_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_upgradedExporterForgesFact_succeeds, test_witness_wrongChainRelayRejected_succeeds, test_witness_wrongExportRejected_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_send_effects_and_frame
  - hevm: S-map: CAUGHT check_sendMessage_len0, check_sendMessage_len100, check_sendMessage_len128, check_sendMessage_len37, check_sendMessage_len4; S-all: SURVIVED
- **K06**
  - unit: -
  - inv(seed 1): Safety.invariant_ethConservation, Safety.invariant_expiredImpliesNeverRelayable, Safety.invariant_noDoubleSpend, Safety.invariant_sentTimestamps, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayAtWindowBoundary_succeeds, test_witness_relayBlocksExport_succeeds, test_witness_relayedExportCall_succeeds, test_witness_sentTimestampCorruptionDetected_succeeds, test_witness_targetRuleRejects_succeeds, test_witness_wrongChainRelayRejected_succeeds, test_witness_wrongExportRejected_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_send_effects_and_frame
  - hevm: S-map: CAUGHT check_sendMessage_len0, check_sendMessage_len100, check_sendMessage_len128, check_sendMessage_len37, check_sendMessage_len4; S-all: SURVIVED
- **K07**
  - unit: Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_relayed_reverts, L2ToL2_RelayMessage_Test.testFuzz_relayMessage_alreadyRelayed_reverts, L2ToL2_RelayMessage_Test.testFuzz_relayMessage_metadataStore_succeeds
  - inv(seed 1): Safety.invariant_atMostOneRelay, Safety.invariant_ethConservation, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_relayAtWindowBoundary_succeeds, test_witness_relayBlocksExport_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_upgradedExporterForgesFact_succeeds, test_witness_wrongChainRelayRejected_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_OnlyExportReachesL1_relay_reentrant, check_relay_delivery_value_context_failure, check_relay_effects_and_frame
  - hevm: S-map: CAUGHT check_relayMessage_badEncoding_len33, check_relayMessage_badEncoding_offset20, check_relayMessage_badEncoding_offset60, check_relayMessage_len0, check_relayMessage_len100, check_relayMessage_len37, check_relayMessage_len4; S-all: SURVIVED
- **K08**
  - unit: L2ToL2_RelayMessage_Test.testFuzz_relayMessage_alreadyRelayed_reverts
  - inv(seed 1): Safety.invariant_atMostOneRelay, Safety.invariant_ethConservation, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_relayBlocksExport_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_relay_effects_and_frame
  - hevm: S-map: CAUGHT check_relayMessage_badEncoding_len33, check_relayMessage_badEncoding_offset20, check_relayMessage_badEncoding_offset60, check_relayMessage_len0, check_relayMessage_len100, check_relayMessage_len37, check_relayMessage_len4; S-all: CAUGHT check_allSlots_relayMessage_len0, check_allSlots_relayMessage_len100, check_allSlots_relayMessage_len37, check_allSlots_relayMessage_len4
- **K09**
  - unit: L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_notOtherMessenger_reverts
  - inv: -
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expire_iff, check_expire_iff_unbounded
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K10**
  - unit: L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_notOtherMessenger_reverts
  - inv(seed 1): Safety.invariant_ethConservation, Safety.invariant_expiredImpliesNeverRelayable, Safety.invariant_noDoubleSpend, Safety.invariant_noForgedFact, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_relayBlocksExport_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expire_iff, check_expire_iff_unbounded
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K11**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Bridge_Integration_Test.test_refundETH_expireGasLimit_succeeds, Bridge_Integration_Test.test_refundETH_expireReplay_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_wrongPreimage_reverts, Bridge_RefundETH_Test.test_refundETH_destinationIsThisChain_reverts, Bridge_RefundETH_Test.test_refundETH_messageNotFromBridge_reverts, Bridge_RefundETH_Test.test_refundETH_senderRejectsETH_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_fromItself_reverts, Exporter_ExportUndeliveredMessage_Test.test_exportUndeliveredMessage_otherChain_reverts, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_notOtherMessenger_reverts, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_succeeds, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_unknownMessage_reverts, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_withinWindow_reverts, L2ToL2_ExpireMessage_Test.test_expireMessage_atWindowEnd_reverts
  - inv(seed 1): - | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_upgradedExporterForgesFact_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expire_boundary, check_expire_iff, check_expire_iff_unbounded
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K12**
  - unit: L2ToL2_RelayMessage_Test.testFuzz_relayMessage_unsafeTarget_reverts
  - inv(seed 1): Safety.invariant_unsafeTargetRule, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_targetRuleRejects_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_OnlyExportReachesL1_relay_l2cdm, check_OnlyExportReachesL1_relay_passer, check_UnsafeTargetRule_relay, check_UnsafeTargetRule_relay_l2cdm, check_UnsafeTargetRule_relay_passer
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K13**
  - unit: L2ToL2_RelayMessage_Test.testFuzz_relayMessage_unsafeTarget_reverts, L2ToL2_SendMessage_Test.testFuzz_sendMessage_unsafeTarget_reverts
  - inv(seed 1): Safety.invariant_unsafeTargetRule, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_targetRuleRejects_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_OnlyExportReachesL1_relay_passer, check_UnsafeTargetRule_relay, check_UnsafeTargetRule_relay_passer, check_UnsafeTargetRule_send, check_UnsafeTargetRule_send_passer
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K14**
  - unit: L2ToL2_RelayMessage_Test.testFuzz_relayMessage_unsafeTarget_reverts, L2ToL2_SendMessage_Test.testFuzz_sendMessage_unsafeTarget_reverts
  - inv(seed 1): Safety.invariant_unsafeTargetRule, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_targetRuleRejects_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_OnlyExportReachesL1_relay_l2cdm, check_UnsafeTargetRule_relay, check_UnsafeTargetRule_relay_l2cdm, check_UnsafeTargetRule_send
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K15**
  - unit: L2ToL2_RelayMessage_Test.testFuzz_relayMessage_destinationNotRelayChain_reverts
  - inv(seed 1): Safety.invariant_destinationBinding, Safety.invariant_ethConservation, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_wrongChainRelayRejected_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_UnsafeTargetRule_relay
  - hevm: S-map: CAUGHT check_relayMessage_badEncoding_len33, check_relayMessage_badEncoding_offset20, check_relayMessage_badEncoding_offset60, check_relayMessage_len0, check_relayMessage_len100, check_relayMessage_len37, check_relayMessage_len4; S-all: CAUGHT check_allSlots_relayMessage_len0, check_allSlots_relayMessage_len100, check_allSlots_relayMessage_len37, check_allSlots_relayMessage_len4
- **K16**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_anyCaller_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_relayed_reverts, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_succeeds, Exporter_ExportUndeliveredMessage_Test.test_exportUndeliveredMessage_otherChain_reverts
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_onlyExportReachesL1, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayBlocksExport_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_wrongExportRejected_succeeds
  - halmos: ExporterExpiryHalmos: CAUGHT check_export_binding, check_exporter_anyCalldata_onlyExportPayload
- **K17**
  - unit: Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_relayed_reverts
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_ethConservation, Safety.invariant_expiredImpliesNeverRelayable, Safety.invariant_noDoubleSpend, Safety.invariant_onlyExportReachesL1, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_relayBlocksExport_succeeds
  - halmos: ExporterExpiryHalmos: CAUGHT check_export_binding
- **K18**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_anyCaller_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_fromItself_reverts, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_succeeds, Exporter_ExportUndeliveredMessage_Test.test_exportUndeliveredMessage_otherChain_reverts
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_onlyExportReachesL1, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_wrongExportRejected_succeeds
  - halmos: ExporterExpiryHalmos: CAUGHT check_export_binding, check_exporter_anyCalldata_onlyExportPayload
- **K19**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_anyCaller_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_fromItself_reverts, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_succeeds, Exporter_ExportUndeliveredMessage_Test.test_exportUndeliveredMessage_otherChain_reverts
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_onlyExportReachesL1, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_wrongExportRejected_succeeds
  - halmos: ExporterExpiryHalmos: CAUGHT check_export_binding, check_exporter_anyCalldata_onlyExportPayload
- **K20**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_anyCaller_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_fromItself_reverts, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_succeeds, Exporter_ExportUndeliveredMessage_Test.test_exportUndeliveredMessage_otherChain_reverts
  - inv(seed 1): - | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_wrongExportRejected_succeeds
  - halmos: ExporterExpiryHalmos: CAUGHT check_export_binding, check_exporter_anyCalldata_onlyExportPayload
- **K21**
  - unit: L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_borrowedPortal_reverts
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit, check_relayUndelivered_rejectsCallerClaimingPortalA
- **K22**
  - unit: L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_noLockbox_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_otherCluster_reverts
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K23**
  - unit: L1CDM_RelayUndeliveredMessage_Test.testFuzz_relayUndeliveredMessage_wrongL2Sender_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_l2ToL2CrossDomainMessengerSender_reverts
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit, check_relayUndelivered_rejectsL2ToL2AsSender, check_relayUndelivered_rejectsSelfCallOutsideRelay
- **K24**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_l2ToL2CrossDomainMessengerSender_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_succeeds
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit, check_relayUndelivered_rejectsL2ToL2AsSender
- **K25**
  - unit: -
  - inv: -
  - halmos: L1CDMExpiryHalmos: SURVIVED; ReachL1CDMHalmos: SURVIVED
- **K26**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_succeeds
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K27**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_succeeds
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K28**
  - unit: L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_interopDisabled_reverts
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K29**
  - unit: L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_succeeds
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K30**
  - unit: L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_ownChain_reverts, L1CDM_Uncategorized_Test.test_relayMessage_toSelf_reverts
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_L1_relayMessage_rejectsSelfAndPortalTargets
- **K31**
  - unit: Bridge_RefundETH_Test.testFuzz_refundETH_succeeds
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_atMostOneRefund, Safety.invariant_ethConservation, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_refundAfterExpiry_succeeds
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_iff_effects_singleUse
- **K32**
  - unit: Bridge_RefundETH_Test.testFuzz_refundETH_succeeds
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_atMostOneRefund, Safety.invariant_ethConservation, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_legacyForgedFact_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_upgradedExporterForgesFact_succeeds
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_iff_effects_singleUse
- **K33**
  - unit: Bridge_RefundETH_Test.testFuzz_refundETH_wrongPreimage_reverts, Bridge_RefundETH_Test.test_refundETH_destinationIsThisChain_reverts, Bridge_RefundETH_Test.test_refundETH_messageNotFromBridge_reverts, Bridge_RefundETH_Test.test_refundETH_notExpired_reverts
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_ethConservation, Safety.invariant_noDoubleSpend, Safety.invariant_refundImpliesExpired, TightWindow.invariant_allSafetyProperties | witnesses: -
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_iff_effects_singleUse, check_sendETH_then_refund
- **K34**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Bridge_Integration_Test.test_refundETH_expireReplay_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_succeeds, Bridge_RefundETH_Test.test_refundETH_senderRejectsETH_succeeds
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_atMostOneRefund, TightWindow.invariant_allSafetyProperties | witnesses: -
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_iff_effects_singleUse, check_sendETH_then_refund
- **K35**
  - unit: Bridge_RefundETH_Test.testFuzz_refundETH_succeeds
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_ethConservation, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_iff_effects_singleUse, check_sendETH_then_refund
- **K36**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Bridge_Integration_Test.test_refundETH_expireReplay_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_succeeds, Bridge_RefundETH_Test.test_refundETH_senderRejectsETH_succeeds
  - inv(seed 1): - | witnesses: test_witness_legacyForgedFact_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_upgradedExporterForgesFact_succeeds
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_frame, check_refund_iff_effects_singleUse, check_sendETH_then_refund
- **K37**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Bridge_Integration_Test.test_refundETH_expireReplay_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_wrongPreimage_reverts, Bridge_RefundETH_Test.test_refundETH_senderRejectsETH_succeeds
  - inv(seed 1): - | witnesses: test_witness_legacyForgedFact_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_upgradedExporterForgesFact_succeeds
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_frame, check_refund_iff_effects_singleUse, check_sendETH_then_refund
- **K38**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_anyCaller_succeeds, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_relayed_reverts, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_succeeds, Exporter_ExportUndeliveredMessage_Test.test_exportUndeliveredMessage_otherChain_reverts
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_onlyExportReachesL1, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayBlocksExport_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_wrongExportRejected_succeeds
  - halmos: ExporterExpiryHalmos: CAUGHT check_export_binding, check_exporter_anyCalldata_onlyExportPayload
- **K39**
  - unit: Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_fromItself_reverts, Exporter_ExportUndeliveredMessage_Test.test_exportUndeliveredMessage_otherChain_reverts, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_unknownMessage_reverts
  - inv(seed 1): Safety.invariant_noForgedFact, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_wrongExportRejected_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expire_iff, check_expire_iff_unbounded
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K40**
  - unit: Bridge_RefundETH_Test.testFuzz_refundETH_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_wrongPreimage_reverts, Bridge_RefundETH_Test.test_refundETH_destinationIsThisChain_reverts, Bridge_RefundETH_Test.test_refundETH_messageNotFromBridge_reverts, Bridge_RefundETH_Test.test_refundETH_senderRejectsETH_succeeds, L2ToL2_ExpireMessage_Test.testFuzz_expireMessage_succeeds, L2ToL2_ExpireMessage_Test.test_expireMessage_atWindowEnd_reverts
  - inv(seed 1): Safety.invariant_ethConservation, Safety.invariant_expiredImpliesNeverRelayable, Safety.invariant_noDoubleSpend, TightWindow.invariant_allSafetyProperties | witnesses: test_witness_earlyFactRejected_succeeds, test_witness_upgradedExporterForgesFact_succeeds
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_expire_boundary, check_expire_iff, check_expire_iff_unbounded
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K41**
  - unit: L1CDM_RelayUndeliveredMessage_Test.testFuzz_relayUndeliveredMessage_wrongL2Sender_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_borrowedPortal_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_l2ToL2CrossDomainMessengerSender_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_otherCluster_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_succeeds
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K42**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, L1CDM_RelayUndeliveredMessage_Test.testFuzz_relayUndeliveredMessage_wrongL2Sender_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_l2ToL2CrossDomainMessengerSender_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_otherCluster_reverts, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_succeeds
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K43**
  - unit: -
  - inv: -
  - halmos: RefundExpiryHalmos: SURVIVED; ReachBridgeHalmos: stopped by hand
- **K44**
  - unit: -
  - inv: -
  - halmos: L2ToL2ExpiryHalmos: SURVIVED; ReachL2ToL2Halmos: SURVIVED
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K45**
  - unit: -
  - inv: -
  - halmos: L2ToL2ExpiryHalmos: SURVIVED; ReachL2ToL2Halmos: SURVIVED
  - hevm: S-map: CAUGHT check_relayMessage_badEncoding_hugeLen, check_relayMessage_badEncoding_hugeOffset, check_relayMessage_badEncoding_len33, check_relayMessage_badEncoding_len65, check_relayMessage_badEncoding_offset20, check_relayMessage_badEncoding_offset60, check_relayMessage_len0, check_relayMessage_len100, check_relayMessage_len37, check_relayMessage_len4, check_relayMessage_rawPayload_len0, check_relayMessage_rawPayload_len100, check_relayMessage_rawPayload_len128; S-all: CAUGHT check_allSlots_relayMessage_len0, check_allSlots_relayMessage_len100, check_allSlots_relayMessage_len37, check_allSlots_relayMessage_len4
- **K46**
  - unit: -
  - inv: -
  - halmos: L2ToL2ExpiryHalmos: CAUGHT check_OnlyExportReachesL1_relay_reentrant
  - hevm: S-map: SURVIVED; S-all: SURVIVED
- **K47**
  - unit: Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_fromItself_reverts, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_relayed_reverts, Exporter_ExportUndeliveredMessage_Test.testFuzz_exportUndeliveredMessage_succeeds
  - inv(seed 1): NoUnsafeTargetRule.invariant_safetyWithoutTargetRule, Safety.invariant_onlyExportReachesL1, TightWindow.invariant_allSafetyProperties | witnesses: -
  - halmos: ExporterExpiryHalmos: CAUGHT check_export_binding, check_exporter_anyCalldata_onlyExportPayload
- **K48**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, L1CDM_RelayUndeliveredMessage_Test.test_relayUndeliveredMessage_succeeds
  - inv: -
  - halmos: L1CDMExpiryHalmos: CAUGHT check_relayUndelivered_iff_and_deposit
- **K49**
  - unit: Bridge_Integration_Test.test_refundETH_endToEnd_succeeds, Bridge_Integration_Test.test_refundETH_expireReplay_succeeds, Bridge_RefundETH_Test.testFuzz_refundETH_succeeds, Bridge_RefundETH_Test.test_refundETH_messageNotFromBridge_reverts, Bridge_RefundETH_Test.test_refundETH_senderRejectsETH_succeeds
  - inv(seed 1): - | witnesses: test_witness_legacyForgedFact_succeeds, test_witness_refundAfterExpiry_succeeds, test_witness_relayedExportCall_succeeds, test_witness_unsafeWindowDoubleSpend_succeeds, test_witness_upgradedExporterForgesFact_succeeds
  - halmos: RefundExpiryHalmos: CAUGHT check_refund_frame, check_refund_iff_effects_singleUse, check_sendETH_then_refund
- **K50**
  - unit: -
  - inv: -
  - halmos: RefundExpiryHalmos: SURVIVED; ReachBridgeHalmos: stopped by hand
- **K51**
  - unit: -
  - inv(seed 1): - | witnesses: test_witness_refundAfterExpiry_succeeds
  - halmos: L2ToL2ExpiryHalmos: SURVIVED; ReachL2ToL2Halmos: SURVIVED
  - hevm: S-map: SURVIVED; S-all: SURVIVED

</details>

## Surviving every executed layer

**Four mutants have no recorded catch in any executed layer: K25, K43, K44 and K50.** For K43 and K50, Halmos's
phase 2 was stopped, so "survived" there means phase 1 and every other layer. **Two more, K41 and K42, are caught
only for the wrong reason, so no executed layer checks what they break.** K25 is equivalent; the other five are
gaps. K41 and K42 are the important ones: each is a double spend.

| ID | Mistake | Why it survives | What would kill it |
|---|---|---|---|
| K25 | checks (a) and (c) of `relayUndeliveredMessage` in the opposite order | **Equivalent** (argued below): every check is a view call and every failure reverts. It also survived `ReachL1CDMHalmos.check_reach_l1cdm_sequence2` (2319 s; 2463 s for the whole Halmos layer). Kontrol's `prove_relayUndeliveredMessage_spec` and the EVM-Lean `relay_success` statement would still hold. | Nothing should. |
| K50 | a new state variable declared before `SuperchainETHBridge.refunded` (storage layout shift) | **Gap: upgrade history.** Every executed layer deploys fresh code and reads `refunded` through its getter. After a later upgrade with this layout, every hash refunded before the upgrade would read as unrefunded and could be refunded again. | Already caught outside this campaign's layers: the committed storage-layout snapshot (`snapshots/storageLayout/SuperchainETHBridge.json` has `refunded` at slot 0; with K50 it is slot 1, so regenerating the snapshots with `just snapshots-check` followed by `git diff --exit-code snapshots/` fails), and the `refundETH` EVM-Lean statement, which fixes the `refunded` slot (by reasoning). |
| K44 | `relayMessage` forwards only half the remaining gas to the target | **Gap: gas forwarding.** No layer asserts how much gas the relay forwards: the unit tests' targets are cheap, Halmos and the models do not check gas, and the hevm harness excludes gas on purpose. A relay to an expensive target can fail where it would have succeeded; the message then stays unrelayed and can still expire, so nothing is paid twice. | A unit test that relays to a target which needs most of the transaction's gas (for example, one that burns a fixed large amount) and asserts success with a gas limit just above the requirement. |
| K41 | check (a) reads `caller.systemConfig()` instead of `caller.portal().systemConfig()` | **Gap: the stand-ins.** The mutant asks the calling contract for its SystemConfig. The Halmos `MockCallerMessenger`, Kontrol's caller mock and the unit tests' `vm.mockCall` fixtures do not answer `systemConfig()`, so the call reverts for honest and forged callers alike. The Halmos counterexample is the honest direction (should succeed, reverts); `check_relayUndelivered_rejectsCallerClaimingPortalA` passes; the unit attack tests fail only on revert data. A real L1CrossDomainMessenger does have `systemConfig()`, and a fake caller can answer anything, so the mutant accepts a forged word. | In Halmos and Kontrol, give the caller mock a symbolic, attacker-chosen `systemConfig()` (and the portal mock a symbolic `ethLockbox()`); in the unit tests `borrowedPortal` and `otherCluster`, mock those getters with attacker-favourable answers and expect the custom error. Lean's `cex_noRealMessengerCheck` is the attack. |
| K42 | check (b) asks the caller's portal's lockbox about this chain's portal | **Gap: the stand-ins**, as for K41: the portal mocks do not answer `ethLockbox()`. A fake portal can return a lockbox that authorizes anything. | As for K41. Lean's `cex_noLockboxCheck_fakePortal` is the attack. |
| K43 | `refundETH` emits `RefundETH` only for a nonzero amount | **Gap: events, minor.** The successful-refund fuzz test bounds the amount from 1, and no other layer asserts the `RefundETH` event. | Run `testFuzz_refundETH_succeeds` with `_amount = 0` allowed (or add a zero-amount case) and `vm.expectEmit` on `RefundETH`. |

**Every other non-equivalent mutant is caught by at least one executed layer.** The ones with a single catching layer:
- **K45** (`relayMessage` skips `CrossL2Inbox.validateMessage`: any payload can be relayed without an initiating
  message, a forged delivery). **Only the hevm harness** catches it, by comparing the calls made with develop's.
  The unit tests mock the inbox with `vm.mockCall` and never `vm.expectCall` it; the Halmos inbox mock and the
  invariant handler's mock accept everything. Checks that would kill it: a unit test where `vm.mockCallRevert` on
  the inbox makes `relayMessage` revert, plus `vm.expectCall` on the inbox in the success test; in Halmos, an inbox
  mock that records its argument and an assertion that a successful relay validated exactly `(id,
  keccak256(payload))`. This is not new code in this PR (develop's messenger has the same call), so it is a gap of
  the test suites, not of the expiry change.
- **K46** (`relayMessage` marks the message relayed only after the target call). **Only Halmos** catches it
  (`check_OnlyExportReachesL1_relay_reentrant`: a target that calls the exporter during its own relay gets a "not
  relayed" export for the message being relayed). This breaks the local property only: that export carries the
  relay's own time, which is at most `sentAt + W <= sentAt + P`, so `expireMessage` rejects it, and a replay of the
  failed withdrawal carries the same time. A cheap second layer: let the hevm harness's target mock read
  `successfulMessages(h)` back during the call (develop answers true, K46 false).
- **K51** (duplicate `MessageExpired`) is caught only by the invariant witness `test_witness_refundAfterExpiry_succeeds`,
  which counts that event.
- **K04** (Halmos only, `check_expire_iff_unbounded`) changes no reachable state (argued below). `ReachL2ToL2Halmos`
  does not catch it; see finding 14 for why its symbolic-storage step should have.

Narrowest coverage by contract:
- **L1CrossDomainMessenger** (K21–K30, K41, K42, K48): only the unit tests and Halmos run its code; the invariant
  harness abstracts the L1 hop. Both catch every one except K25, and K41 and K42 only through the mock gap above. Lean has a counterexample for each dropped or
  weakened check (K21–K24, K30, K41, K42; Quint for the same except K30, whose rule is fixed there), and the
  EVM-Lean `relay_success` statement would fail for every one except K25 and K30 (`relayMessage` is not covered).
- **K36** (refund preimage with amount 0, an unbounded mint once a zero-value `sendETH` expires): the random
  invariant campaign did not find the mint (zero-amount sends are rare in its handler); it failed only through the
  witness tests, because ordinary refunds stop working. The unit tests and Halmos catch it directly.

This answers the question for **this catalogue and the observations the layers make**. It does not cover arbitrary
implementation changes outside the catalogue.

## Findings and surprises

1. **Unit tests miss K06 (send records timestamp 1).** `testFuzz_sendMessage_succeeds` asserts
   `sentMessageTimestamps(h) == block.timestamp`, but it runs at forge's default `block.timestamp == 1`, so it cannot
   tell `= 1` from `= block.timestamp`. K06 is a double spend (every message expires immediately). The invariant
   suite (`invariant_sentTimestamps`, `invariant_noDoubleSpend`), Halmos (`check_send_effects_and_frame`) and the
   hevm harness (S-map) all catch it. The check that would kill it in the unit tests: warp first, for example
   `vm.warp(bound(_ts, 2, type(uint64).max))` with a fuzzed `_ts`, or a fixed non-1 time, before `sendMessage`.
2. **The hevm harness's Halmos S-all checks do not see a changed mapping entry.** K07 (relay no longer sets
   `successfulMessages[H]`) leaves the outcome and the calls unchanged and differs from develop only in that
   mapping entry. The S-map checks (solidity layout) catch it; every `check_allSlots_*` (generic layout) passes with
   complete exploration. A probe on the same build gave the same result for `check_allSlots_relayMessage_len37`
   (passes under the generic layout; errors under the solidity layout, which cannot load a symbolic slot).
   - Mechanism: Halmos 0.3.3's generic storage decodes a keccak-derived slot into its preimage and keeps one SMT
     array per preimage width, so the 256-bit symbolic slot `_s` of `_checkAllSlots` never aliases a mapping entry.
     The stray-`sstore(5, 1)` mutants of `../hevm/run.sh` are caught because slot 5 is not hash-derived.
   - So Halmos S-all covers the un-hashed slot domain (including symbolic indices into it), not mapping entries. The texts that say "every raw slot" or "every storage
     slot" (`../hevm/README.md`, the header of `../hevm/L2ToL2Equivalence.t.sol`, and the top-level README's
     equivalence row), and "a PASS there is at least as strong", overstate it. The hevm-engine proof
     `prove_sendMessage_len0` uses a different storage model and is not shown to be affected.
   - K05 and K06 surviving S-all is by design (`_checkAllSlots` pins `sentMessageTimestamps[h]`), not evidence of
     the blindness; K07 is.
   - Checks that would kill K07 under S-all: also compare `vm.load` at `keccak256(abi.encode(k, b))` for a symbolic
     key `k` and each mapping base slot `b`; or add an expected-FAIL witness that runs S-all against a mutant that
     drops one mapping write. This is a statement-fidelity issue of that layer, not a contract issue. S-map
     compares the mapping entries it names (the relayed hash and the symbolic keys of the new-only mappings), which
     is what caught K07; it is not a whole-storage comparison either.
3. **`../halmos/mutants.sh` M7 is not the mutant its label says.** Its `sed` replaces the call to
   `xDomainMessageSender()` with `address(0)`, which leaves the condition `address(0) != otherMessenger()`. On a
   deployed chain (`otherMessenger()` nonzero) that makes `expireMessage` always revert, a liveness mutant. In the
   Halmos domain, where `otherMessenger()` may be zero, it also accepts calls the original rejects. Either way it
   does not "skip the sender check". K10 here is the real removal (both comparison operands become `address(0)`),
   and `check_expire_iff` kills it too. Suggested fix: relabel M7, or use K10's edit.
4. **The invariant suite never executes the L1CrossDomainMessenger.** By design (README "Facts and the L1 hop"), so
   K21–K30 survive it by construction. It is listed as n/a, not as a gap.
5. **The invariant suite catches the boundary mutant K01 only in its `TightWindow` configuration** (W = P). With
   the real 1-day margin, K01 is safe, which is what Lean `safe_variants` proves (Quint has no instance with `>=`
   and P = 8 days; its `expireGeNoMargin` is the P = W combination). K02 (P = W) is caught by no invariant, as
   Lean `safe_variants` and Quint `safeNoMargin` predict; the unit test
   `test_expiryPeriod_exceedsProtocolWindowByADay_succeeds` and Halmos `check_expiryPeriodIsCapPlusMargin` pin the
   margin.
6. **K03 (P = 6 days)** is caught by the invariant suite's setUp, which asserts the documented assumption
   `P_contract >= W_protocol`. The suite's own `UnsafeWindow` configuration (skipped in CI) is where that double
   spend is exhibited.
7. **Liveness mutants are caught as readily as safety mutants.** K05, K11, K16, K18, K26, K38, K47 and K48 never
   pay twice, but each breaks a unit test, a Halmos iff or binding check, and (except the L1 ones) an invariant
   witness or `OnlyExportReachesL1`'s "says what an export would say" clause. So do K20 (an API shift) and K27 (a
   changed deposit envelope).
8. **Timing mistakes on the export path.**
   - K19 (the exporter claims a time 2 days late) acts like P_contract = 6 days, so it maps onto
     `cex_periodBelowWindow`. With a claim only 1 day late it would be absorbed by the margin, like K01 and K02.
   - K29 (the L1 relay forwards L1 `block.timestamp`) is worse than a smaller period: the forwarded time is chosen by
     whoever executes the L1 relay. An export made right after the send (while the message is still unrelayed)
     can be finalized late, or made to fail inside `L1CrossDomainMessenger.relayMessage` (low `_minGasLimit`) and
     replayed at any later L1 time. Meanwhile the message is relayed on its destination inside the window. Once
     L1 time passes `sentAt + 8 days`, the word expires it: effective P is 0. No model has a parameter for this; it
     subsumes `cex_periodBelowWindow`.
   - K40 (`expireMessage` compares the source's own clock instead of `_undeliveredAt`) has the same effect: any
     early export expires the message once the source's clock passes `sentAt + P`.
9. **K38 shows two defenses stacking.** If the exporter swapped destination and source in the hash, a chain could
   only describe messages it sent itself. The source has no timestamp for such a hash on any other chain, and the
   route back to the chain's own L1CrossDomainMessenger is blocked by `L1CrossDomainMessenger._isUnsafeTarget`.
10. **The protocol models cover the L1 checks one for one, but not the L2-side bookkeeping.** K07, K08, K10, K15,
    K17, K31–K36, K39, K40, K45–K47 and K49 have no model counterpart, because the models fix those guards structurally (`relay` requires
    `¬relayed`, `expire` requires a deposit from the source's L1CrossDomainMessenger, `refund` requires `expired ∧
    ¬refunded`, `isBridge` abstracts the refund preimage). The local layers (unit tests, Halmos, the invariants,
    EVM-Lean and Kontrol statements for `refundETH`, the exporter and `expireMessage`) are what pin those down. The
    unit tests, invariants and Halmos were run and catch them, except K45 (only the hevm harness); the EVM-Lean and
    Kontrol columns are by reasoning.
11. **Upgrade history, and events other than the one witness that counts them, are outside the executed layers.**
    K50 (storage-layout shift in the bridge) and K43 (a missing zero-amount `RefundETH` event) survive all of them; see "Surviving every executed layer". K51 (a duplicate
    `MessageExpired`) is caught only because one invariant-suite witness counts that event. The repo's storage-layout
    snapshot check is what guards K50's class.
12. **The relay's inbox call and its mark-before-call order each rest on one layer.** K45 (no
    `CrossL2Inbox.validateMessage`: forged delivery, and through a forged `relayETH` an unbounded mint) is caught
    only by the hevm harness, and K46 (mark after the call; local only) only by Halmos's re-entrant export check.
    The checks that would add a second layer are proposed under "Surviving every executed layer".
13. **Relay gas forwarding is asserted by no layer.** K44 (half the gas forwarded to the relay target) survives all of them, as R2
    predicted; K27's changed minimum gas limit is caught only through the exact deposit envelope.
14. **`ReachL2ToL2Halmos` drops overflow paths.** Its symbolic-storage step can start from `sentAt` near `2^256`,
    yet it misses K04: its (E) clause `assert(pre.ts != 0 && _s.t > pre.ts + period)` is checked arithmetic, which
    panics with code 0x11 in the wrapped case, and Halmos 0.3.3 counts only `Panic(0x01)` as a failure, so the path
    is dropped silently. Fix: bound it as `check_expire_iff_unbounded` does (`pre.ts <= type(uint256).max - period
    && ...`), or compare without overflow.
15. **K41 and K42 expose a stand-in gap in three layers at once** (Halmos, Kontrol, unit tests); see "Surviving
    every executed layer". The layers state the right property, but their models of "another chain's messenger and
    portal" lack two getters that a mistaken implementation might call.

## Equivalent and protocol-equivalent mutants

Three kinds of "no behaviour change" appear, and each is argued from the code, not asserted.

**K25: checks (a) and (c) in the opposite order. Equivalent: the same successful calls, the same effects.**
0. Scope: this assumes getters whose answer does not depend on call order or gas, which is what the real
   `OptimismPortal2`, `SystemConfig`, `ETHLockbox` and `L1CrossDomainMessenger` getters are. The EVM-Lean
   `relay_success` statement assumes the same (`ReturnsWord`: a stable returned word across successful calls).
1. The three checks are one `if (x || y || z) revert`, after the INTEROP gate and before the only state-changing
   statement (`this.sendMessage`).
2. Every call in them is a `view` function on an interface or contract type: `caller.portal()`,
   `callerPortal.systemConfig()`, `.l1CrossDomainMessenger()`, `portal.ethLockbox()`, `.authorizedPortals()` and
   `caller.xDomainMessageSender()`. Solidity therefore emits `STATICCALL`, and no callee can change any state,
   whatever code it runs.
3. With short-circuit evaluation, the original reverts iff some check fails or some evaluated getter reverts. The
   reordered code reverts iff the same holds for its own evaluation order. The only difference is a revert in a
   getter that the original never evaluated because an earlier check had already failed. Then both versions revert,
   with different revert data.
4. So the set of successful calls, their effects (one self-call that deposits `expireMessage(H, t)`) and the final
   state are identical. Only the revert data on failing calls and the gas differ. Gas includes gas-boundary
   outcomes: under the 63/64 rule a call with barely enough gas can succeed in one order and run out of gas in the
   other. For a relayed withdrawal that only moves the attempt into `failedMessages`, from where it can be replayed. No property in this repo, and no
   protocol property, depends on which of two reasons a rejected `relayUndeliveredMessage` gives.

**K09: `expireMessage` without its `msg.sender == L2CrossDomainMessenger` check. Equivalent on reachable
states, not equivalent as code.** The remaining check reads `L2CrossDomainMessenger.xDomainMessageSender()` and
compares it with `otherMessenger()`, the source chain's L1CrossDomainMessenger.
1. Outside an L2CrossDomainMessenger relay, `xDomainMessageSender()` reverts ("xDomainMessageSender is not set"),
   so `expireMessage` reverts for every caller.
2. Inside a relay, it returns the message's L1 sender. A message whose L1 sender is A's L1CrossDomainMessenger is
   produced only by `relayUndeliveredMessage`'s self-call `this.sendMessage(0x..23, expireMessage(H, t),
   100_000)`:
   - `CrossDomainMessenger.sendMessage` records `msg.sender` as the sender, and the L1CDM is its own caller only
     there;
   - `L1CrossDomainMessenger._isUnsafeTarget` stops anyone from relaying an L2 message to the L1CDM itself.
3. Such a relay runs exactly one call: 0x..23's `expireMessage`. That function calls only the L2CrossDomainMessenger's
   getters. A replay of a failed relay re-runs the same stored message. So no other code ever runs while
   `xDomainMessageSender() == otherMessenger()`, and nobody else can call `expireMessage` during such a frame.
4. Hence the set of successful `expireMessage` executions is unchanged on every reachable state. The invariant
   campaign agrees: three seeds, no failure, although `forgeExpiry` calls `expireMessage` from arbitrary L2
   callers.
5. The argument **depends on K30's rule**, and on the L2CrossDomainMessenger being initialized: an uninitialized one
   has `xDomainMsgSender == 0`, so `xDomainMessageSender()` and `otherMessenger()` both return 0 and the check
   passes for any caller. Predeploys are initialized at genesis or in the upgrade. With both K09 and K30, K30's double spend gets a second route. This is
   why the unit tests and Halmos are right to kill K09: they check the stated local spec (`check_expire_iff` takes
   the getter results as symbolic values, which a real L2CrossDomainMessenger never returns outside a relay).

**K04: `sentAt + EXPIRY_PERIOD` computed in an `unchecked` block. Equivalent on reachable states.**
1. `sentMessageTimestamps[H]` is written only by `sendMessage`, with `block.timestamp`.
2. A block timestamp is far below `2^256 - 8 days`, so the sum never wraps.
3. Only `check_expire_iff_unbounded`, which runs from fully symbolic storage, sees the wrapped case. The phase-2
   check `ReachL2ToL2Halmos` did not: its sequence check starts from the deployed state, and its symbolic-storage
   step loses the wrapped case to an overflow panic in its own assertion (finding 14). The EVM-Lean statement of
   `expireMessage` has "no overflow" as an explicit success condition, so it would become false. Kontrol's
   `prove_expireMessage_spec` assumes `sentAt < 2^64`, so it would still hold.

**Protocol-equivalent: local behaviour changes, but the expiry safety properties still hold.** The protocol
models prove these safe, and the matrix shows that only the local layers (unit tests, Halmos, the EVM-Lean
statements) flag them:
- **K01** (`>=` at `sentAt + P`, which admits the equality boundary) and **K02** (`P = W = 7 days`, which drops
  the whole margin) each weaken the 1-day margin. Each one alone is
  safe: Lean `safe_variants` covers both, and Quint `safeNoMargin` covers K02. Only the two together, `>=` with `P = W`, double-spend:
  Lean `cex_nonStrict` and Quint `expireGeNoMargin`. The invariant suite catches K01 only in its `TightWindow`
  configuration (W = P), which is exactly that combination.
- **K28** (no INTEROP gate). Lean proves safety for every value of `Config.interop`, so the gate is not part of
  the safety argument; it keeps chains that do not run interop out of the flow. Checks (a), (b) and (c) still bind
  the word to an exporter of a lockbox member.
- **K12, K13, K14** (target rule weakened). The rule is defense in depth: Lean `safety_without_targetRule`, Quint
  `safeNoTargetRule`, and the invariant configuration `NoUnsafeTargetRule`. They are caught because the local
  property UnsafeTargetRule is checked directly (unit tests, Halmos, the invariant suite's `invariant_unsafeTargetRule`).
- **K46** (mark relayed after the target call): a re-entrant export during the relay carries the relay's own time,
  at most `sentAt + W <= sentAt + P`, so `expireMessage` rejects it; only the local OnlyExportReachesL1 property
  breaks.
- **K37** (refund preimage uses `nonce + 1`): `refundETH(d, n, ...)` refunds the message with nonce `n + 1` and the
  same `(from, to, amount)`, and pays its own `from` and `amount`. Every expired bridge message can still be
  refunded once, through the shifted nonce, and nothing is paid twice. Only the API is wrong, and a message with
  nonce 0 can never be refunded.

## Layers and how each was run

All executed layers ran on a 32-core Linux host, each heavy command under a 16 GB memory cap with swap off, at
most two campaigns at a time, with forge 1.8.3 (the repo pin) and Halmos 0.3.3 with
`../halmos/halmos-selfdestruct.patch`. `run.sh` runs a baseline (the unmutated code, id `K00`) through every
selected layer first and stops unless all of it passes.

| Layer | What `run.sh` runs per mutant | Budget | Typical time per mutant |
|---|---|---|---|
| Unit tests (PR #23259) | `FOUNDRY_PROFILE=liteci forge test --fuzz-seed 1` on `test/L1/L1CrossDomainMessenger.t.sol`, `test/L2/L2ToL2CrossDomainMessenger.t.sol`, `test/L2/UndeliveredMessageExporter.t.sol` and `test/L2/SuperchainETHBridge.t.sol`, all four for every mutant (93 tests) | liteci: 128 fuzz runs | 20–45 s, including the build |
| Invariants (`../invariants`) | `FOUNDRY_PROFILE=liteci forge test --match-path 'test/formal/expiry/invariants/*'`, seed 1; if nothing fails, seeds 2 and 3 | the inline-pinned liteci budget, 32 runs x 256 depth per configuration | 90–110 s per seed; 250–340 s for a survivor |
| Halmos (`../halmos`) | the expected-PASS checks (from `expected.tsv`) of the touched contract's phase-1 contract (`L2ToL2ExpiryHalmos`, `ExporterExpiryHalmos`, `L1CDMExpiryHalmos`, `RefundExpiryHalmos`); if none fails, also its phase-2 reachability contract | `halmos.toml` (bytes lengths 0,1,32,33,100,132,260; 60 s per assertion query); 2400 s or 5400 s per process | 10–40 s (L1CDM: 140–200 s); phase 2: 76 s (exporter) to about 40 min (L1CDM) |
| hevm harness (`../hevm`) | the mutated messenger is compiled as the harness's NEW code (as `../hevm/run.sh` does for its own mutants), and every `check_sendMessage_*`, `check_relayMessage_*` (S-map, solidity layout) and `check_allSlots_*` (S-all, generic layout) runs under Halmos. The mutant is compiled with the harness's `foundry.toml` (optimizer off), while develop's side is the optimized repo build, as in `../hevm/run.sh`; the unmutated control (K00) passes under the same setup | no solver timeout, as in `../hevm/run.sh` | 110–200 s |
| Lean (`../lean`), Quint (`../quint`) | not run: each mutant is mapped to the model parameter, instance or counterexample theorem it corresponds to | — | — |
| EVM-Lean (`../evm-lean*`) | not rebuilt: the proofs are pinned to the c7c51d79e2 bytecode, so every non-equivalent mutant breaks the pin. The column says whether the **statement** would become false of the mutated code (stmt) or would still hold and only the pin breaks (pin) | — | — |
| Kontrol (`../kontrol`) | not run: it was committed after the campaign, and each mutant needs a full `kontrol build --rekompile` of the proof project before its proofs (several of which take 5–35 min). The column maps each mutant to the proof statements in `../kontrol/README.md`: K = some proved statement would become false | — | — |

How `run.sh` avoids stale or misattributed results:
- **In place.** `DeployUtils.getDeployedCode` reads `forge-artifacts/` whatever `FOUNDRY_OUT` says, so the forge
  layers run in place. Before every mutant, `src/` is restored (`git checkout -- src`), `forge clean` removes
  `forge-artifacts/` and `cache/`, and `cache/invariant` is removed (forge replays a persisted failing sequence
  first, which would carry a kill from one mutant to the next). One checkout per concurrent campaign.
- **Canary.** After the unit-test build, the touched contract's liteci runtime bytecode must differ from the
  baseline's; otherwise the row says so. No row tripped it.
- **Halmos rebuilds** with `forge build --force` (test contracts embed creation code, and an incremental build can
  miss them; `../halmos/README.md` records 16 false survivors from that).
- **A kill** is a failing test or invariant (forge), or a check that was expected to PASS producing a counterexample
  (Halmos). The baseline of every layer passes on the same checkout, so a failure is attributable to the mutant.
  setUp failures are reported as such.
- **Phase-2 baselines.** `ReachL2ToL2Halmos` (1118 s) and `ReachL1CDMHalmos` (2342 s) were established on the same
  checkout and Halmos directory by separate runs, and `ReachExporterHalmos` (98 s, with `ExporterExpiryHalmos`) by
  a first campaign run, so the final campaigns used `SKIP_BASE_REACH=1`. `ReachBridgeHalmos` did not finish within
  5400 s on the shared host, so it has no baseline here; for K43 and K50, which survived phase 1, it was stopped by
  hand (it observes neither events nor storage layout).
- **Phase 2 ran only where phase 1 did not already kill**: K25, K44, K45 and K51 (all survived it); stopped for K43
  and K50. K04 and K09 were also run through
  phase 2 (`REACH=always`): `ReachL2ToL2Halmos` misses K04 (1157 s; see finding 14: its sequence check starts
  from the deployed state, and its symbolic-storage step drops the overflow panic, so this is a harness gap, not
  evidence about reachable states) and kills K09 with both `check_reach_sequence2` and `check_reach_step_symbolicStorage`,
  because its (E) clause names `msg.sender == 0x..07` and its L2CrossDomainMessenger mock answers the getters
  with fresh symbolic values in every step (957 s).

The campaigns themselves were run as follows (two checkouts in parallel, each `ONLY` set split further over time):

```sh
ONLY='^K(0[1-9]|1[0-5])$' SKIP_BASE_REACH=1 run.sh                      # messenger mutants
ONLY='^K(1[6-9]|2[0-9]|3[0-8])$' SKIP_BASE_REACH=1 HALMOS_TIMEOUT=5400 run.sh   # exporter, L1CDM, bridge
ONLY='^K0[49]$' LAYERS=halmos REACH=always SKIP_BASE_REACH=1 run.sh     # phase 2 for K04 and K09
```

The phase-2 baselines they skipped were established by separate runs on the same checkout and Halmos directory
(`ReachL2ToL2Halmos` 1118 s, `ReachL1CDMHalmos` 2342 s, `ReachExporterHalmos` 98 s, all PASS). A first run of the second campaign was stopped
by hand during its own `ReachL1CDMHalmos` baseline (exit 143 in that log), because 2400 s was too short on the
shared host.

Reproduce (from `packages/contracts-bedrock`, on a clean checkout, bash >= 4.4). Without `SKIP_BASE_REACH=1`, the
baseline runs every selected phase-2 contract first; on a loaded host give it `HALMOS_TIMEOUT=5400` or more:

```sh
FORGE=<forge 1.8.3> HALMOS=<patched halmos> \
  WRAP="systemd-run --user --scope -q -p MemoryMax=16G -p MemorySwapMax=0" \
  test/formal/expiry/mutation/run.sh                 # all mutants, all layers
ONLY='^K0[1-5]$' LAYERS=unit,inv test/formal/expiry/mutation/run.sh   # a subset
```

`run.sh` changed during the campaign: the forge-output parser was rewritten (failing invariants print their name on
a later line), forge runs that end abnormally are now ERROR, and an unvalidated Halmos counterexample is now
INCONCLUSIVE rather than a kill. No verdict depends on these changes: every failing-check list in this README was
re-parsed from the raw logs with the final parser, every killed forge log ends with its test-suite summary, and
every Halmos kill had a validated counterexample.

Results land in `results/matrix.tsv` (id, layer, verdict, seconds, failing checks) with one log per mutant and
layer.

## Review log

Reviewers: R1 (fresh-context reviewer), R2 and R3 (independent model-based reviewers). All three were asked for a
correctness and coverage audit: catalogue fidelity, verdict fidelity against the raw logs, the equivalence
arguments, the model and EVM-Lean mapping, `run.sh`, and missing mutants.

### Round 1

Verdict of all three: the matrix matches the logs (each spot-checked or re-parsed them; all 38 edits apply once, on
the intended line; every Halmos kill has a validated counterexample; no kill comes from a compile or unrelated
failure). The findings were about the runner, explanations and coverage.

| # | Reviewer(s) | Finding | Disposition |
|---|---|---|---|
| 1 | R1, R2, R3 | `run.sh` ignored forge's exit status; a killed or truncated forge run would read as SURVIVED (no saved row affected) | **Fixed.** Exit status other than 0/1, or no final "Ran N test suites" line, is ERROR. |
| 2 | R2, R3 | The hevm layer could read a stale JSON from an earlier run; unvalidated Halmos models counted as kills; an empty check list or a misspelled layer passed | **Fixed.** The JSON is deleted first; only validated counterexamples kill (others are INCONCLUSIVE); empty check lists are ERROR; unknown layers abort. |
| 3 | R2, R3 | Artifact isolation relied on the environment: inherited `FOUNDRY_OUT`/`FOUNDRY_CACHE_PATH`, unchecked `forge clean`, an advisory canary, and a clean-tree check that ignored staged changes | **Fixed.** The forge layers unset the `FOUNDRY_*` path overrides; a failed clean is ERROR; a tripped canary makes the unit verdict ERROR; `src/` is compared with and restored from `HEAD`. |
| 4 | R1 | EVM-Lean "stmt" for K02/K03 | **Changed to "pin".** `evm-lean/scripts/regen.sh` rewrites `P_contract` from the bytecode's `PUSH3`, so the regenerated statement holds with 7 or 6 days. R2 and R3 read the statement with the literal 691,200 and called it "stmt"; both readings are noted in the legend. The point that matters: EVM-Lean does not guard the period's value; the unit test and Halmos do. |
| 5 | R1, R2, R3 | K29's "effective P ≈ 1 day" is wrong: the L1 relay time is chosen by whoever relays (late finalization, or a failed relay replayed later) | **Fixed.** Effective P is 0; mapped as "fact time chosen by the relayer", which subsumes `cex_periodBelowWindow`. |
| 6 | R1, R2, R3 | Finding 2 (hevm S-all) is right, but the boundary was imprecise; the sibling hevm texts overstate S-all | **Fixed.** Mechanism stated (Halmos generic storage keys hash-derived slots by preimage width); S-all covers the un-hashed slot domain; K05/K06 are excluded by design; the overstating texts are listed; the hevm-engine proof is not claimed to be affected. |
| 7 | R1, R2, R3 | Finding 3 (halmos M7): "always reverts" needs `otherMessenger() != 0`; in the Halmos domain M7 also accepts | **Fixed.** |
| 8 | R2, R3 | K20 is an API shift (the caller can pass `M - 1`), not a liveness loss; K27 changes a minimum gas limit, not a cap | **Fixed** in the matrix and the catalogue. |
| 9 | R2, R3 | "None reachable" was too strong for K09 (a direct call gets different revert data); K25's argument needs order- and gas-independent getters | **Fixed.** Definitions narrowed to successful executions and effects; the getter assumption and gas-boundary outcomes stated. |
| 10 | R1 | K09 also assumes an initialized L2CrossDomainMessenger | **Added.** |
| 11 | R1, R2, R3 | K01 has no exact Quint instance (`safeNoMargin` is K02); K30 has no Quint instance; K04's wrap bound is `>=` | **Fixed.** |
| 12 | R1, R2, R3 | Phase-2 claims not backed by the shared artifacts (K04/K09 still running; the separate baselines not shared); the reproduction command omitted `SKIP_BASE_REACH=1` | **Fixed.** K04/K09 phase-2 results filled in; the exact campaign invocations and the baseline runs are listed; the reproduction note says when phase-2 baselines are needed. |
| 13 | R1 | Kontrol is committed at the branch head | **Mapped** by statement, like EVM-Lean (column added). Running it per mutant needs a full `kontrol build` per mutant. |
| 14 | R1, R2, R3 | Missing mutants | **Added K39–K51** (see their rows): R1's sentAt check, source-clock expiry, check (a) via the caller's own SystemConfig, check (b) reversed; R2's zero-amount event, half-gas relay, no `validateMessage`, mark-after-call, nonce-free export hash, wrong forwarded hash, refund sender = `from`; R3's storage-layout shift and duplicate `MessageExpired`. |
| 15 | R2 | The hevm layer compiles the mutant with the optimizer off, while develop is the optimized build | **Stated** in the layers table (the unmutated control passes under the same setup). |


### Round 2

All three re-checked the round-1 dispositions, the new rows K39–K51 against their logs, and the survivor analysis.
They confirmed every new `sed`, every new verdict against the raw results, the per-layer counts, and the survivor
set. Findings:

| # | Reviewer(s) | Finding | Disposition |
|---|---|---|---|
| 1 | R1 (HIGH) | K41 and K42 are "caught" only because the Halmos, Kontrol and unit-test stand-ins for the other chain's messenger and portal lack `systemConfig()` / `ethLockbox()`: every call reverts, including honest ones, and the forged word is never tried. | **Accepted.** Marked K‡, moved next to the survivors with the checks that would test them (symbolic attacker-chosen getters in the mocks; attacker-favourable mocks in the unit attack tests). Not fixed here: those files belong to the other layers. |
| 2 | R1, R2, R3 | K46 is not a double spend: a re-entrant export carries the relay's own time, at most `sentAt + W < sentAt + P`. | **Fixed:** "local only (the window rule absorbs it)"; a second-layer check proposed for the hevm harness. |
| 3 | R1, R3 | The reason `ReachL2ToL2Halmos` misses K04 was wrong: its symbolic-storage step loses the wrapped case to a checked-arithmetic panic (0x11) in its own assertion, which Halmos 0.3.3 does not count as a failure. | **Fixed** (finding 14, with the bound that would close it). |
| 4 | R1 | Round-1 items 6 and 7 were not fixed in the sibling files (`../hevm/README.md`, `../hevm/L2ToL2Equivalence.t.sol`, the top-level README, `../halmos/README.md`). | **Listed for their owners.** This campaign writes only its own directory; findings 2 and 3 name each text. |
| 5 | R1, R2, R3 | Stale statements: K45 in finding 10's "caught by" list; the bridge phase-2 and "phase 2 ran only for K25" notes; K51 missing from the single-layer list; K20/K27 still called liveness; K49's invariant catch is witness-only; "no layer models gas" and "no other layer asserts events" too broad; truncated hevm check lists; "four survivors" should say K43/K50's phase 2 was stopped. | **Fixed.** |
| 6 | R1, R2, R3 | `run.sh`: overrides still inherited by the hevm build, its Halmos run and the final clean; `forge clean` and `git checkout` statuses ignored; empty hevm check lists pass; a tripped canary still lets the invariant layer run; stale `../hevm/mutants/`; untracked files under `src/` not detected; "Ran 1 test suite" not matched; no bash version guard. | **Fixed.** A failed restore stops the run. The final `run.sh` was re-run on K08 and K45 (unit and hevm layers) and reproduced their verdicts. |
| 7 | R2 | The K50 snapshot check needs a diff after regeneration. | **Fixed** (`just snapshots-check` then `git diff --exit-code snapshots/`). |
| 8 | R1 | K45's effect is an unbounded mint as well (a forged `relayETH`). | **Fixed.** |
