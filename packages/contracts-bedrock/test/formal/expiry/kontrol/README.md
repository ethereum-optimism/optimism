# Kontrol proofs: per-message interop expiry

These are KEVM proofs, run with Kontrol 1.0.255 (the version pinned in `mise.toml`), of the contract-level
properties of per-message interop expiry. Each proof executes the real compiled contracts from `src/`, symbolically,
with every input the statement quantifies over left symbolic.

The proofs live in two folders, because the contracts under test need different compilers:

- `solc0825/`: the L2ToL2CrossDomainMessenger (solc 0.8.25).
- `solc0815/`: the UndeliveredMessageExporter, L1CrossDomainMessenger and SuperchainETHBridge (solc 0.8.15).
- `expiry-lemmas.md` has the extra KEVM lemmas the proofs need, each with its soundness argument. They cover the
  jump destinations of init code with symbolic constructor arguments (`new SafeSend{value}(from)` with a symbolic
  `from`) and the definedness of KEVM's jump-destination helper. `run-kontrol-expiry.sh` deletes Kontrol's cached
  copy of this file before each build, because Kontrol otherwise keeps compiling its first copy.
- `run-kontrol-expiry.sh` builds and runs everything.

## Running

Add this foundry profile (it is not in `foundry.toml`):

```toml
[profile.kexpiry]
src = "test/formal/expiry/kontrol"
out = "test/formal/expiry/kontrol/out"
test = "test/formal/expiry/kontrol"
script = "test/formal/expiry/kontrol"
```

Then, from `packages/contracts-bedrock`, with Kontrol 1.0.255 on PATH (for example inside
`runtimeverificationinc/kontrol:ubuntu-jammy-1.0.255`):

```sh
KONTROL_WORKERS=8 test/formal/expiry/kontrol/run-kontrol-expiry.sh            # every proof
test/formal/expiry/kontrol/run-kontrol-expiry.sh L2ToL2CrossDomainMessengerExpiryKontrol.prove_expireMessage_spec
```

Every `prove_*` must pass, except the `*_WITNESS` proofs, which must fail (see "Non-vacuity" below). The script ends
with `check-results.py`, which exits non-zero unless that holds for every proof and witness. On the full run recorded
under **Results** it exits non-zero: three proofs and one witness did not close.

The runner passes `--assume-defined` and `--no-stack-checks` to `kontrol prove`. The first lets KEVM assume its
partial functions are defined on the explored paths; the second skips EVM stack-depth checks (a stack overflow
reverts, so it can only add reverting paths). Both are assumptions of every result here, next to the gas and keccak
assumptions below.

## Non-vacuity

A proof whose assumptions (`vm.assume`, symbolic-storage constraints, mock answers, harness exclusions) cannot all
hold together, or cannot hold together with a successful call, passes vacuously. So every headline proof is paired,
in `witnesses.tsv`, with a `*_WITNESS` that asserts an outcome never happens and must FAIL with a counterexample
leaf. The witnesses have different roles, and not all share their proof's exact domain:
- **same setup, success reachable**: most iff and effect proofs (`expireMessage_spec`, the exporter and
  `relayUndeliveredMessage` specs, the relay symbolic-target proof, the bridge any-selector cases);
- **positive controls** for universal-rejection proofs: the witness changes the rejected value (a codeless target
  instead of 0x..07 or 0x..16, the exporter instead of 0x..23 as sender) to show the surrounding setup admits a
  success. It does not show success inside the rejected domain, which is empty by the proof;
- **rejection reachable**: `refundETH_alreadyRefundedReachable_WITNESS` and the unknown-selector bridge witness
  show the reverting outcome is reached;
- **instrumentation control**: `exporter_whitelistIsLive_WITNESS` removes 0x..07 from the whitelist to show it cuts
  off a call;
- the refund witness uses a tighter amount bound (`<= 2^127`) than `preimageBinding` (`<= 2^128`), which is enough
  for existence.
`check-results.py` checks that each witness has a failing leaf and no stuck node, not which assertion the leaf
reached.
`check-results.py` reads `kontrol list` (latest version of each proof) and fails the run if any of these hold:

- a `prove_*` in the `.k.sol` files is missing from `witnesses.tsv`;
- a proof did not pass, or has stuck nodes;
- a witness did not fail with a failing node, or has stuck nodes.

The one proof without assumptions (`prove_expiryPeriod_atLeastProtocolWindow`, a constant comparison) is listed with
`-`. `prove_exporter_whitelistIsLive_WITNESS` also shows that Kontrol's call whitelist really cuts off a call outside
it.

## What is proved

The names match the properties in the Quint model. Message bytes are symbolic with a fixed length of 600 bytes
(`@custom:kontrol-bytes-length-equals`). The other inputs are symbolic within these domains:
- chain IDs, hashes, destinations, sources, nonces of L2 messages and refund amounts: full-width words (refund
  amounts up to the liquidity, see below);
- timestamps generated in the harness (`freshUInt(8)`): 64-bit; the L1 messenger's nonce: 240-bit; minimum gas
  limits: 32-bit; addresses: 160-bit, from `freshAddress`, which in the pinned Kontrol excludes the test contract and
  the cheat-code address;
- callers: fresh in `sendMessage`, the exporter any-selector proof and the bridge proofs; the test contract in the
  relay and exporter-spec proofs; the relay identifier's origin is 0x..23 and, except in the rejection proofs,
  its block number, log index and timestamp are fixed to (1, 0, 1);
- storage: the L2 messenger, the bridge and the mocks' answer slots are `symbolicStorage`; the L1 messenger is a
  fresh deployment with only its portal, SystemConfig, other-messenger and nonce slots written;
- calls carry zero value; relay targets are codeless accounts (no target code runs, so re-entry from a target is
  not explored here, see the self-target case below); ETHLiquidity holds `2^128` wei.

### UnsafeTargetRule (`solc0825/L2ToL2CrossDomainMessengerExpiry.k.sol`)

- `prove_sendMessage_unsafeTargetRule`: `sendMessage` succeeds if and only if `destination != chainid` and
  `target` is none of 0x..23, 0x..07 and 0x..16. This holds for every destination, target, message, caller,
  timestamp and chain ID. Assumption: `msgNonce < 2^240 - 1`, since the nonce space cannot be exhausted.
- `prove_sendMessage_rejectsPasser`: the 0x..16 case of the rule above, stated on its own.
- `prove_relayMessage_rejectsL2CrossDomainMessenger` and `prove_relayMessage_rejectsPasser`: `relayMessage`
  reverts with `L2ToL2CrossDomainMessenger_MessageTargetUnsafe` for every payload whose destination is this chain
  and whose target is 0x..07 (or 0x..16). The identifier, source, nonce, sender, message and `successfulMessages`
  are all symbolic.

### OnlyExportReachesL1

These proofs use recording mocks at 0x..07 (L2CrossDomainMessenger) and 0x..16 (L2ToL1MessagePasser). Each mock
records the caller of every non-static call it receives, whatever the calldata. The record is ordinary storage, so
it shows calls whose effects **survive**: a call made in a frame that later reverts is rolled back with it. "Makes
no call" below therefore means "no call survives in a successful or reverted outcome"; a call followed by a revert
is not observed (by code inspection, the current messenger and exporter make no such call).

- `prove_sendMessage_neverCallsL2CDMOrPasser`: `sendMessage` makes no call to either address.
- `prove_relayMessage_neverMakesMessengerCallL2CDMOrPasser`: `relayMessage`, for any target and message, never
  makes 0x..23 a caller of 0x..07 or 0x..16.
- `prove_relayMessage_selfTarget_neverCallsL2CDMOrPasser`: the same holds for target == 0x..23 with any message,
  i.e. a relayed call into the messenger itself.
- `prove_exporter_onlyCallsL2CDMWithFixedPayload` (`solc0815/UndeliveredMessageExporterExpiry.k.sol`): this holds
  for every source messenger, source, nonce, sender, target, message, min gas limit, chain ID, timestamp and set of
  relayed hashes:
  - The export succeeds if and only if `!successfulMessages[H]`, and it then returns `H`, where
    `H = keccak256(abi.encode(chainid, source, nonce, sender, target, message))`.
  - Its only non-static call is a single call to 0x..07 with calldata exactly
    `sendMessage(sourceMessenger, relayUndeliveredMessage(H, block.timestamp), minGasLimit)`.
  - "Only" combines two checks. Kontrol's call whitelist (`allowCallsToAddress`) admits CALLs to 0x..07, the
    exporter itself and the cheat-code address (the whitelist is global, not per caller). A CALL anywhere else is
    cut off with `KONTROL_WHITELISTCALL`, which reverts the export; since this proof asserts the iff on `ok`, no
    successful export made such a CALL. The recorder at 0x..07 then shows exactly one surviving call, with exactly
    that calldata. A call from the exporter to itself is admitted and not counted. DELEGATECALL, CALLCODE and
    CREATE are not covered by the whitelist (the exporter's code has none). The recorder at 0x..07 never reverts, so
    the iff is about an accepting L2CrossDomainMessenger. `prove_exporter_whitelistIsLive_WITNESS` shows that the
    whitelist really fires.

### relayUndeliveredMessage (`solc0815/L1CrossDomainMessengerExpiry.k.sol`)

- `prove_relayUndeliveredMessage_spec`: a call from any caller contract succeeds if and only if all four checks
  hold:
  - (g) A's `SystemConfig.isFeatureEnabled(INTEROP)` is true.
  - (a) `caller.portal().systemConfig().l1CrossDomainMessenger() == caller`.
  - (b) A's lockbox `authorizedPortals(caller.portal())` is true.
  - (c) `caller.xDomainMessageSender() == Predeploys.UNDELIVERED_MESSAGE_EXPORTER`.

  The four answers are fully symbolic. On success, A's portal receives exactly one deposit. It comes from A's
  L1CDM, has value 0, is not a creation, goes to 0x..07, has gas `baseGas(msg, 100_000)`, and carries data
  `relayMessage(versionedNonce, sender = A's L1CDM, target = 0x..23, 0, 100_000, expireMessage(H, t))`. On
  failure, no deposit is made.

  Forged answers covered: every stand-in answers the getters listed in `ExpiryMocks0815.sol` (each pointer and
  leaf getter of the caller, the portals, the SystemConfigs and the lockboxes), with a symbolic answer for each
  leaf getter. That includes getters the real code does not read: A's own `SystemConfig.l1CrossDomainMessenger()`,
  the caller portal's lockbox `authorizedPortals()`, the caller SystemConfig's feature flags, and the caller's own
  `systemConfig()`, which returns a separate stand-in (`callerOwnSystemConfig`) with its own symbolic answers. So a
  contract that reads one of those wrong getters is caught by the iff, not by a missing function reverting. Run on
  scratch copies of the contract with the `../mutation/mutants.tsv` edits, the iff fails with a genuine
  counterexample for both campaign mutants:
  - K41: check (a) reads the caller's own `systemConfig()` instead of its portal's;
  - K42: check (b) reversed, i.e. `callerPortal.ethLockbox().authorizedPortals(portal)`.

  Until this round's review, `caller.systemConfig()` returned the same stand-in as `callerPortal.systemConfig()`,
  so K41 was indistinguishable from the real check; the separate stand-in fixes that (see the review log).

  The pointer getters (`caller.portal()`, `caller.systemConfig()`, `portal.systemConfig()`, `portal.ethLockbox()`)
  return fixed stand-ins; A's portal's `systemConfig()` returns A's SystemConfig, as consistency requires. Fully
  symbolic pointers are covered only by `symbolicPortalChain`, which does not close. A's portal always accepts the
  deposit, so the iff's success direction is about an accepting portal (the real one can revert, e.g. on its
  resource metering).
- `prove_relayUndeliveredMessage_rejectsMessengerAsSender`: with the gate on and (a) and (b) satisfied, a word
  from 0x..23 is rejected.
- `prove_relayUndeliveredMessage_symbolicPortalChain`: the caller's portal and that portal's SystemConfig are
  fully symbolic addresses, and success implies the checks. As written its final assertion
  (`p == address(callerPortal) && s == address(callerSystemConfig)`) is too strong: the symbolic portal may alias
  A's portal or the caller, and on such a path the real function can pass every check, so the assertion fails
  without any authentication failure. It needs an assertion about the getters actually traversed before it can
  close; it did not close, and nothing relies on it.

### expireMessage

- `prove_expireMessage_spec`: this holds for every hash, `undeliveredAt`, caller, `xDomainMessageSender`,
  `otherMessenger` and storage, with `sentAt = sentMessageTimestamps[H] < 2^64` (any realistic block timestamp).
  `expireMessage` succeeds if and only if all of these hold:
  - `msg.sender == 0x..07`
  - `xDomainMessageSender == otherMessenger`
  - `sentAt != 0`
  - `undeliveredAt > sentAt + EXPIRY_PERIOD`

  Afterwards, `expiredMessages[H] == old || success`, and `sentMessageTimestamps[H]` is unchanged. The contract's
  `EXPIRY_PERIOD` is read from the contract, never hardcoded.
- `prove_expiryPeriod_atLeastProtocolWindow`: `EXPIRY_PERIOD >= 7 days`. That is the assumption
  P_contract >= W_protocol, with W_protocol capped at 7 days by op-core and kona.

### refundETH (`solc0815/SuperchainETHBridgeExpiry.k.sol`)

- `prove_refundETH_preimageBinding`: this holds for every destination, nonce, from, to, amount (up to the
  liquidity), chain ID, `refunded[]` and `expiredMessages[]`. `refundETH` succeeds if and only if
  `expiredMessages[H] && !refunded[H]`, where H is written out in the test, independently of `Hashing`:
  `H = keccak256(abi.encode(destination, chainid, nonce, bridge, bridge, relayETH(from, to, amount)))`.
  Afterwards, `refunded[H] == old || success`.
- `prove_refundETH_singleUse`: after any successful refund, the same call reverts with
  `SuperchainETHBridge_AlreadyRefunded` (two calls in one proof).
- `prove_refundETH_alreadyRefundedReverts`: from any state with `refunded[H]` set, the refund for H reverts (with
  `SuperchainETHBridge_AlreadyRefunded`, or `SuperchainETHBridge_MessageNotExpired` if H is not expired). Together
  with the post-condition of `prove_refundETH_preimageBinding` (a success sets `refunded[H]`), this gives single use
  for an immediately repeated call: the repeat reverts, with `AlreadyRefunded` when `expiredMessages[H]` is still
  set. Across intervening transactions it also needs `refunded[H]` never to be cleared and, for the exact error,
  `expiredMessages[H]` never to be cleared; neither is proved here (the bridge any-selector cases below exclude
  only false → true changes).

### Phase 2: whole-contract reachability (any selector)

- `prove_exporter_anySelector_onlyExportPayload`: for ANY selector and caller (argument region laid out as an
  export call), at most one call to 0x..07 survives, and if one does, the selector is `exportUndeliveredMessage`
  and the calldata is exactly the export payload for the decoded arguments. The whitelist is on and this proof does
  not assert `ok`: a CALL outside the whitelist aborts the exporter under Kontrol's semantics and the proof then
  passes trivially on that path. So, unlike the headline exporter proof, it does not show that no selector makes a
  CALL elsewhere; Halmos's `check_exporter_anyCalldata_onlyExportPayload` and code inspection cover that.
- `prove_bridge_anySelector_*` (five proofs, one per case): for ANY selector, any caller (including 0x..23) and
  five argument words, `refunded` changes only through `refundETH`. The selector space is split into `refundETH`,
  `sendETH`, `relayETH`, the two getters, and "none of the five" (no fallback, so it reverts).
  - For the four non-refund cases: for ANY hash H0, if `refunded[H0]` goes from false to true then the selector is
    `refundETH`. That is all these cases assert: clearing an entry or rewriting the same value would not be
    caught (the current code has no such write). `w2` and `w3` are address-typed, so in the `relayETH` case the
    amount is below 2^160, and the recipient exclusions also apply to words a case does not use.
  - For the `refundETH` case: Kontrol's storage whitelist allows writes only to the slot of `refunded[H]`, H being
    the preimage hash of the arguments. The proof asserts `ok == (expiredMessages[H] && !refunded[H])`. A write to
    any other slot would be cut off and revert the refund, which falsifies that assertion. So a successful refund
    writes no slot but `refunded[H]`, and a failed one writes nothing.
  - `prove_bridge_anySelector_refundETHSucceeds_WITNESS` shows a refund still succeeds under the whitelist, i.e.
    the whitelisted slot is the one refundETH writes.
- Not attempted in Kontrol (see the Halmos suite for these): `expiredMessages[H]` is set only by `expireMessage`
  for any calldata to the messenger (any-selector dispatch into `relayMessage` with symbolic dynamic offsets), and
  "the L1CrossDomainMessenger is an L1->L2 sender only via relayUndeliveredMessage" for any calldata.

## Results

Run on c7c51d79e2 (logic unchanged at 448d31ad19, where only the messenger's version string differs), Kontrol
1.0.255, 10 workers, one 28 GB memory-capped container. Wall times are per proof, on a shared, loaded host. The
`relayUndeliveredMessage` proofs and their witnesses were re-run in this round's review on the current harness
(separate stand-in for `caller.systemConfig()`), with `KONTROL_FRESH=1`, 4 workers and a 16 GB container: both
proofs passed and both witnesses failed with one failing leaf and no stuck node; `check-results.py` accepts those
four results.

| Proof | Result | Time |
|---|---|---|
| `prove_sendMessage_unsafeTargetRule` | passed | 8m |
| `prove_sendMessage_rejectsPasser` | passed | 4m |
| `prove_sendMessage_neverCallsL2CDMOrPasser` | passed | 20m |
| `prove_relayMessage_rejectsL2CrossDomainMessenger` | passed | 1m |
| `prove_relayMessage_rejectsPasser` | passed | 1m |
| `prove_relayMessage_neverMakesMessengerCallL2CDMOrPasser` | passed | 9m |
| `prove_relayMessage_selfTarget_neverCallsL2CDMOrPasser` | NOT CLOSED (see below) | - |
| `prove_expireMessage_spec` | passed | 10m |
| `prove_expiryPeriod_atLeastProtocolWindow` | passed | <1m |
| `prove_exporter_onlyCallsL2CDMWithFixedPayload` | passed | 3m |
| `prove_exporter_anySelector_onlyExportPayload` | passed | 22m |
| `prove_relayUndeliveredMessage_spec` | passed (rerun this round with a separate `caller.systemConfig()` stand-in) | 3m |
| `prove_relayUndeliveredMessage_rejectsMessengerAsSender` | passed (rerun this round) | 1m |
| `prove_relayUndeliveredMessage_symbolicPortalChain` | NOT CLOSED: path explosion, stopped after ~4h at >2,000 nodes | - |
| `prove_refundETH_preimageBinding` | passed | 35m |
| `prove_refundETH_alreadyRefundedReverts` | passed | 6m |
| `prove_refundETH_singleUse` | NOT CLOSED: stopped after ~4h; single use follows from the two proofs around it | - |
| `prove_bridge_anySelector_refundETH` | passed (storage whitelist) | 24m |
| `prove_bridge_anySelector_sendETH` | passed | 21m |
| `prove_bridge_anySelector_relayETH` | passed | 53m |
| `prove_bridge_anySelector_getters` | passed | 14m |
| `prove_bridge_anySelector_otherSelectors` | passed | 16m |

Every `*_WITNESS` failed with a genuine counterexample (a failing leaf, no stuck nodes). The one exception is
`prove_relayUndeliveredMessage_symbolicPortalChainCanSucceed_WITNESS`, which explodes like its proof and was stopped.

The bridge any-selector property was first a single proof with a fresh H0 for every selector. It failed twice, the
second time after 2h38m with 93 nodes still pending. Both counterexamples are spurious. The selector is `refundETH`
(0xe17a776b), the path condition contains `H0 != H` (H being the refund hash of the arguments), and the failing read
is `refunded[H0]` looked up in storage just written at key `refunded[H]`. Kontrol cannot resolve that lookup: it
needs `keccak(H0 . 0) != keccak(H . 0)` from `H0 != H`, and its injectivity rule leaves the preimage comparison
unevaluated. `w1 = 2^160` in the model is the refund nonce, a full `uint256` both in the harness hash and in the
contract, so the harness hash agrees with the contract's. The fix is the split described under Phase 2, which uses
the storage whitelist for the `refundETH` case.

`prove_relayMessage_selfTarget_neverCallsL2CDMOrPasser` does not close. The relayed self-call dispatches a fully
symbolic 600-byte message into the messenger's own ABI decoder, where the dynamic `bytes` offsets are symbolic. One
symbolic step ran for about 3 hours, and its backend grew past 12 GB before it was killed. Its witness
(`prove_relayMessage_selfTargetCanSucceed_WITNESS`) does fail, so a relayed self-call can succeed. What is left is
the per-function argument: `sendMessage` makes no call (proved above), `relayMessage` reverts on re-entry
(`nonReentrant`), and `expireMessage` reverts at its `msg.sender == 0x..07` check before any call, since there
`msg.sender` is 0x..23. This is a manual argument by code inspection, for every message length (the `sendMessage`
proof it cites fixes 600 bytes, while the nested message can have any length). The Halmos suite excludes target
0x..23 for the same reason as the other relay proofs: every source chain's `sendMessage` rejects it.

## Model, mocks and assumptions

Each answer mock answers with whatever is in its storage, and the proofs make that storage symbolic, so it stands
for any contract that gives those answers and does not revert. The recorders start at zero, the pointer getters
return fixed stand-ins, and A's portal always accepts a deposit. A constant answer is without loss of generality
where the code under test reads each getter once per call, which holds for the real L1 messenger, exporter and
expiry paths. It is not a universal fact: the two-call `refundETH_singleUse` reads `expiredMessages(H)` twice (its
constancy relies on the mapping being unchanged), and the harness itself calls some getters before the contract
does. Because the mocks never revert, iff results are statements about this accepting environment: they show the
contract's own checks, not that every real callee would let the call succeed.

- **CrossL2Inbox (0x..22)**: `validateMessage` accepts everything. This is an over-approximation, so
  "relayMessage reverts" results hold a fortiori for the real inbox.
- **Canonical payload encoding**: relay payloads use the canonical SentMessage encoding. After decoding,
  `relayMessage` depends only on the decoded tuple, so this loses no decoded behaviour.
- **L2CrossDomainMessenger (0x..07)**:
  - For `expireMessage`, the mock's `xDomainMessageSender()` and `otherMessenger()` return symbolic addresses. This
    over-approximates the real getter, which reverts outside a relay.
  - For OnlyExportReachesL1, the mock records every non-static call. A STATICCALL into it would revert. Static calls
    cannot initiate withdrawals, and the messenger only static-calls 0x..07 from `expireMessage`, after its
    `msg.sender == 0x..07` check.
- **Exporter's view of 0x..23**: `successfulMessages` is a symbolic mapping, i.e. any set of relayed hashes.
- **L1 side**: the caller's portal, its SystemConfig, A's portal, A's lockbox and A's SystemConfig are mocks. A's
  L1CDM is the real implementation, with `portal`, `systemConfig`, `otherMessenger` and `msgNonce` written straight
  into its storage slots.
- **Harness exclusions** (addresses a symbolic value may not take):
  - A relayMessage target: the messenger itself, the test contract, the cheat-code address, the console address,
    and the precompile range 0x0..0x1ff. In `symbolicPortalChain` the caller's portal and SystemConfig exclude the
    cheat-code address and the precompile range (and, through `freshAddress`, the test contract), but not the
    console address. Precompiles never call other contracts, and KEVM's BLS12 hooks crash on symbolic input.
  - The refund recipient `from` (and relayETH's recipient in the any-selector proof) is not an account of the
    harness: the test contract, the cheat-code address, the three etched predeploys, the wiped implementation
    addresses, and the two SafeSends created during the call. KEVM cannot alias a symbolic SELFDESTRUCT
    beneficiary with an existing account. Payment is by SELFDESTRUCT, which runs no code, so who the recipient is
    changes neither success nor `refunded`.
- **Governance assumption** (for the exporter design): each cluster chain's L2 governance (its L2 ProxyAdmin owner) can
  upgrade its own UndeliveredMessageExporter. A malicious upgrade could then forge "undelivered" facts for any message
  to that chain. This is comparable to the trust in the shared ETHLockbox: a member's L2 governance can already make
  arbitrary withdrawals from it by changing its own L2 state (the lockbox's own check compares the portals' L1
  ProxyAdmin owners, a separate role). These proofs assume the deployed exporter is the code in `src/`.
- **keccak**: Kontrol's built-in `KECCAK-LEMMAS` assume keccak is injective (collision resistance, the assumption
  stated for this work), that a keccak of symbolic bytes never equals a concrete value (mapping slots never collide
  with fixed slots), and that keccak results are not within 32 of 0 or 2^256. The proofs rely on these.
- **This suite's lemmas** (`expiry-lemmas.md`): the third rule (the definedness of KEVM's jump-destination helper)
  is sound as argued. The first two (the prefix lookup and the length of `#computeValidJumpDests`) are justified by
  the helper's invariant `LEN == lengthBytes(PGM) == lengthBytes(RESULT)` with `I >= 0`, which KEVM's own calls
  maintain, but the rules do not require it as a side condition. As stated they also match instances outside that
  invariant, where they would turn an undefined term into a defined one. Adding the side conditions and re-running
  the proofs that use the rules (the bridge proofs, through `new SafeSend{value}(from)`) is an open item.
- **Out of scope here**:
  - Cross-chain timing: protocol window <= contract period, monotone timestamps, withdrawal finality. These are
    covered by the Quint/Lean models.
- **Gas** is not modelled (Kontrol's default). Out-of-gas paths are not explored.

## Review log

**Round 1** (this suite's first external review, of the version run on `c7c51d79e2`): R1 (fresh-context reviewer),
R2 and R3 (independent model-based reviewers), as a statement-fidelity audit of every proof and witness against the
contracts, the mocks, the lemmas and the result gate. None found a contract that breaks a stated property. All three
confirmed that the hash preimages (exporter, refund, L1 deposit data) and the storage slots match the contracts, that
`relayUndeliveredMessage`'s getters are read once per path, that the self-target residual argument is sound by code
inspection, and that single use follows from the two closed refund proofs for an immediately repeated call. Each
finding and what became of it:

| # | Reviewers | Finding | Disposition |
|---|---|---|---|
| 1 | R1, R2, R3 (high) | `caller.systemConfig()` and `callerPortal.systemConfig()` returned the same stand-in, so the campaign's K41 (check (a) reads the caller's own `systemConfig()`) was indistinguishable from the real check. The README described a different mutation | **Fixed.** Separate stand-in `callerOwnSystemConfig` with its own symbolic answers. Re-ran the `relayUndeliveredMessage` proofs and witnesses (pass / fail as expected) and the iff on scratch copies with the `../mutation/mutants.tsv` K41 and K42 edits: both fail with an assertion counterexample. README describes the campaign mutants |
| 2 | R1, R2, R3 (high/medium) | `symbolicPortalChain`'s final assertion is false when the symbolic portal aliases A's portal or the caller, so its failure to close is not only path explosion | **Documented.** The proof is marked as needing an assertion about the getters actually traversed; nothing relies on it |
| 3 | R2, R3 (high), R1 (low) | Lemma rules 1 and 2 lack the side conditions (`LEN == lengthBytes(PGM) == lengthBytes(RESULT)`, `I >= 0`) their soundness argument uses; rule 3 is sound | **Open, documented** under "Model, mocks and assumptions": adding the side conditions needs a re-run of the bridge proofs |
| 4 | R1, R2, R3 (high/medium) | The exporter any-selector proof discards `ok`, so a whitelisted-out CALL becomes a passing revert; recorders miss calls rolled back by a revert; the whitelist also admits the exporter and the cheat-code address | **Narrowed.** The README states what the recorders and whitelist observe, which statements are about surviving calls, and that other-address CALLs for arbitrary selectors are left to Halmos and code inspection |
| 5 | R1, R2, R3 (medium) | The bridge non-refund cases only check false → true; "never write `refunded`" was too strong; cross-transaction single use also needs no clearing | **Narrowed** in the bridge and single-use text |
| 6 | R1, R2, R3 (medium) | `check-results.py` accepted a witness with a missing or non-numeric failing count, a proof with pending or failing nodes, admitted proofs and unparseable headers | **Fixed.** It parses the counts as non-negative integers, requires proofs to be unadmitted with no pending, failing or stuck node and witnesses to have a failing leaf and no stuck node, rejects unparseable headers and tolerates whitespace in declarations. Checked on this round's `kontrol list` and on malformed inputs. It still does not check which assertion a witness reached or result freshness (stated) |
| 7 | R1, R2, R3 (medium) | "Every other input is a full-width symbolic word" and "each getter is called at most once per path" were overstated; mocks never revert | **Fixed.** Per-input domains (64-bit generated timestamps, 240-bit L1 nonce, 32-bit gas, fixed callers and identifier fields, concrete L1 storage, `2^128` liquidity, address-typed bridge words), the mock behaviour and the scope of the once-per-path argument are stated |
| 8 | R1, R2, R3 (medium/low) | Witnesses do not all share their proof's setup | **Fixed.** The README lists the witness roles |
| 9 | R1, R2 (medium) | `--assume-defined` and `--no-stack-checks` were not disclosed | **Fixed** under "Running" |
| 10 | R1 (medium) | The relay proofs use codeless targets only | **Stated**; re-entry from target code is the self-target argument and Halmos's re-entrant target |
| 11 | R1, R3 (low) | The console exclusion claimed for `symbolicPortalChain` is not imposed; the gate fails on the recorded full run; the self-target argument cites a 600-byte proof | **Fixed** in the text |
| 12 | R1 (low) | The relay rejection proofs fix the identifier origin to 0x..23; the exporter iff assumes an accepting 0x..07 | **Stated** |
