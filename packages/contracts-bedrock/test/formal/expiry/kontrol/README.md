# Kontrol proofs: per-message interop expiry

These are KEVM proofs, run with Kontrol 1.0.255 (the version pinned in `mise.toml`), of the contract-level
properties of per-message interop expiry. Each proof executes the real compiled contracts from `src/`, symbolically,
with every input the statement quantifies over left symbolic.

The proofs live in two folders, because the contracts under test need different compilers:

- `solc0825/`: the L2ToL2CrossDomainMessenger (solc 0.8.25).
- `solc0815/`: the UndeliveredMessageExporter, L1CrossDomainMessenger and SuperchainETHBridge (solc 0.8.15).
- `expiry-lemmas.md` has the extra KEVM lemmas the proofs need, each with its soundness argument. They cover the
  jump destinations of init code with symbolic constructor arguments (`new SafeSend{value}(from)` with a symbolic
  `from`) and the definedness of KEVM's jump-destination helper.
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
with `check-results.py`, which exits non-zero unless that holds for every proof and witness.

## Non-vacuity

A proof whose assumptions (`vm.assume`, symbolic-storage constraints, mock answers, harness exclusions) cannot all
hold together, or cannot hold together with a successful call, passes vacuously. So every headline proof is paired,
in `witnesses.tsv`, with a `*_WITNESS` that has the same setup and assumptions and asserts that the success path
never happens. The witness must FAIL with a genuine counterexample: a failing leaf, not a stuck or pending node.
`check-results.py` reads `kontrol list` (latest version of each proof) and fails the run if any of these hold:

- a `prove_*` in the `.k.sol` files is missing from `witnesses.tsv`;
- a proof did not pass, or has stuck nodes;
- a witness did not fail with a failing node, or has stuck nodes.

The one proof without assumptions (`prove_expiryPeriod_atLeastProtocolWindow`, a constant comparison) is listed with
`-`. `prove_exporter_whitelistIsLive_WITNESS` also shows that Kontrol's call whitelist really cuts off a call outside
it.

## What is proved

The names match the properties in the Quint model. Message bytes are symbolic with a fixed length of 600 bytes
(`@custom:kontrol-bytes-length-equals`). Every other input is a full-width symbolic word. That covers chain IDs,
timestamps, nonces, addresses, callers, and the storage of the contract under test (`symbolicStorage`).

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
records the caller of every non-static call it receives, whatever the calldata.

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
  - "Only" combines two checks. Kontrol's call whitelist (`allowCallsToAddress`) admits CALLs to 0x..07 alone,
    plus the test's own call and the cheat-code address. A CALL anywhere else is cut off with
    `KONTROL_WHITELISTCALL`, which reverts the export, so no successful export made another CALL. The recorder at
    0x..07 then shows exactly one call, with exactly that calldata. `prove_exporter_whitelistIsLive_WITNESS` shows
    that the whitelist really fires.

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
- `prove_relayUndeliveredMessage_rejectsMessengerAsSender`: with the gate on and (a) and (b) satisfied, a word
  from 0x..23 is rejected.
- `prove_relayUndeliveredMessage_symbolicPortalChain`: the caller's portal and that portal's SystemConfig are
  fully symbolic addresses, and success implies the checks.

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
  with the post-condition of `prove_refundETH_preimageBinding` (a success sets `refunded[H]`), this also gives single
  use, at a fraction of the cost of the two-call proof.

### Phase 2: whole-contract reachability (any selector)

- `prove_exporter_anySelector_onlyExportPayload`: for ANY selector and caller (argument region laid out as an
  export call), every CALL the exporter makes goes to 0x..07, at most one is made, and if one is made the selector
  is `exportUndeliveredMessage` and the calldata is exactly the export payload for the decoded arguments.
- `prove_bridge_anySelector_refundedOnlyByRefundETH`: for ANY selector, caller (including 0x..23), five argument
  words and ANY hash H0: if `refunded[H0]` goes from false to true then the selector is `refundETH`, H0 is the
  refundETH preimage hash of the arguments, and `expiredMessages[H0]` holds.
- Not attempted in Kontrol (see the Halmos suite for these): `expiredMessages[H]` is set only by `expireMessage`
  for any calldata to the messenger (any-selector dispatch into `relayMessage` with symbolic dynamic offsets), and
  "the L1CrossDomainMessenger is an L1->L2 sender only via relayUndeliveredMessage" for any calldata.

## Results

Run on c7c51d79e2 (logic unchanged at 448d31ad19, where only the messenger's version string differs), Kontrol
1.0.255, 10 workers, one 28 GB memory-capped container. Wall times are per proof, on a shared, loaded host.

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
| `prove_relayUndeliveredMessage_spec` | passed | 5m |
| `prove_relayUndeliveredMessage_rejectsMessengerAsSender` | passed | 1m |
| `prove_relayUndeliveredMessage_symbolicPortalChain` | NOT CLOSED: path explosion, stopped after ~4h at >2,000 nodes | - |
| `prove_refundETH_preimageBinding` | passed | 35m |
| `prove_refundETH_alreadyRefundedReverts` | passed | 6m |
| `prove_refundETH_singleUse` | NOT CLOSED: stopped after ~4h; single use follows from the two proofs around it | - |
| `prove_bridge_anySelector_refundedOnlyByRefundETH` | OPEN: rerunning with the mapping-key lemma (see below) | - |

Every `*_WITNESS` failed with a genuine counterexample (a failing leaf, no stuck nodes). There are two exceptions.
`prove_relayUndeliveredMessage_symbolicPortalChainCanSucceed_WITNESS` explodes like its proof and was stopped.
`prove_bridge_anySelectorRefunds_WITNESS` is being rerun together with its proof.

`prove_bridge_anySelector_refundedOnlyByRefundETH` failed on its first run with a spurious counterexample. In that
counterexample, `refunded[H0]` reads as set after a refund of a different hash H, because Kontrol could not resolve
the lookup of key H0 in storage just written at key H. The lemma "Storage slots of distinct mapping keys" in
`expiry-lemmas.md` is meant to close this gap; the rerun that would confirm it is in progress.

`prove_relayMessage_selfTarget_neverCallsL2CDMOrPasser` does not close. The relayed self-call dispatches a fully
symbolic 600-byte message into the messenger's own ABI decoder, where the dynamic `bytes` offsets are symbolic. One
symbolic step ran for about 3 hours, and its backend grew past 12 GB before it was killed. Its witness
(`prove_relayMessage_selfTargetCanSucceed_WITNESS`) does fail, so a relayed self-call can succeed. What is left is
the per-function argument: `sendMessage` makes no call (proved above), `relayMessage` reverts on re-entry
(`nonReentrant`), and `expireMessage` reverts at its `msg.sender == 0x..07` check before any call, since there
`msg.sender` is 0x..23. The Halmos suite excludes target 0x..23 for the same reason as the other relay proofs: every
source chain's `sendMessage` rejects it.

## Model, mocks and assumptions

Every mock answers with whatever is in its storage, and the proofs make that storage symbolic. So each mock stands
for any contract that gives those answers. Each getter is called at most once per path, so a constant answer is
without loss of generality.

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
  - A relayMessage target, and the caller's portal and SystemConfig in `symbolicPortalChain`: the test contract,
    the cheat-code address, the console address, and the precompile range 0x0..0x1ff. Precompiles never call
    other contracts, and KEVM's BLS12 hooks crash on symbolic input.
  - The refund recipient `from` (and relayETH's recipient in the any-selector proof) is not an account of the
    harness: the test contract, the cheat-code address, the three etched predeploys, the wiped implementation
    addresses, and the two SafeSends created during the call. KEVM cannot alias a symbolic SELFDESTRUCT
    beneficiary with an existing account. Payment is by SELFDESTRUCT, which runs no code, so who the recipient is
    changes neither success nor `refunded`.
- **Governance assumption** (for the exporter design): each cluster chain's L2 governance (its L2 ProxyAdmin owner)
  can upgrade its own UndeliveredMessageExporter. A malicious upgrade could then forge "undelivered" facts for any
  message to that chain. This is the same trust as the shared ETHLockbox, whose portals must share the proxy admin
  owner. These proofs assume the deployed exporter is the code in `src/`.
- **keccak**: Kontrol's built-in `KECCAK-LEMMAS` assume keccak is injective (collision resistance, the assumption
  stated for this work), that a keccak of symbolic bytes never equals a concrete value (mapping slots never collide
  with fixed slots), and that keccak results are not within 32 of 0 or 2^256. `expiry-lemmas.md` adds one instance
  of the same injectivity for mapping-slot preimages. The proofs rely on these.
- **Out of scope here**:
  - Cross-chain timing: protocol window <= contract period, monotone timestamps, withdrawal finality. These are
    covered by the Quint/Lean models.
- **Gas** is not modelled (Kontrol's default). Out-of-gas paths are not explored.
