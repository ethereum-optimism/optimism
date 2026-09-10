# SPEC — the ETH net-flow solvency cap

**Status: implemented. Ratified by Karl 2026-08-26, amended the same day (home-chain pinning, see
"The single counter" below).**

> References below to `PLAN.md` and `g-decisions.md` are to the Silhouette project's
> design record, which is not in this repository; `README.md` says where it and the shelf branches
> are. Everything normative is restated here.

Scope: `SuperchainETHBridge.sol` (the uniform cap, on every chain) and the new
`SuperchainETHBridgePinned.sol` (the home-chain pin, in the private chain's genesis only).
`ETHLiquidity` is untouched.

Two mechanisms, one property:

| | Where | What it does |
|---|---|---|
| Net-flow cap | `SuperchainETHBridge`, every chain | No counterparty can mint more ETH here than we sent it. Safety floor. |
| Home-chain pin | `SuperchainETHBridgePinned`, P's genesis only | P exchanges ETH with exactly one chain, both directions. Makes the home chain's counter the single global figure. |

## Why

Under V1-ATTESTED (PLAN.md), the private chain P is operator-attested: the L1 batch transaction's own
signature is the whole proof of P's state. A dishonest or compromised attester can therefore claim any
P state it likes, including P blocks that export fabricated `SentMessage` events. Every one of those
fabricated messages is, to the public chain A, a perfectly valid initiating message — it is in P's
real (attested) logs, at a real index, under a real block hash, and stock interop validation accepts
it. Executing one on A calls `SuperchainETHBridge.relayETH`, which calls `ETHLiquidity.mint`, which
hands over real ETH out of the lockbox. Without a cap, one lie mints up to `uint128.max` wei on A.

The cap turns "unbounded" into "exactly what we chose to risk". It is not a fix for attestation; it is
the blast-radius bound that makes attestation an acceptable v1 trust delta.

## The invariant

For every counterparty chain `S`, at every point in the chain's history:

```
  Σ relayETH amounts credited on this chain from source S
      ≤  Σ sendETH amounts sent from this chain to destination S
```

Equivalently, the contract maintains `netSent[S] = sent(S) − relayed(S) ≥ 0` as an explicit storage
figure, and `relayETH` refuses any relay that would drive it negative.

This is the **lockbox invariant, per counterparty**: a counterparty chain can only ever pull ETH back
out of this chain's lockbox, never conjure new ETH into it. At genesis every `netSent[S]` is zero, so
**a chain nobody has sent ETH to can extract nothing** — no configuration, no allow-list, no owner.
The "onboard to P only through one public chain" property that v1 wants falls out of this rather than
being enforced anywhere: P's cap on chain A *is* the sum of the deposits A made to P.

## Mechanism (the whole change)

`sendETH(_to, _chainId)`:

```solidity
netSent[_chainId] += msg.value;
```

`relayETH(_from, _to, _amount)`, after the two existing authorization checks:

```solidity
if (netSent[source] < _amount) revert InsufficientNetFlow();
netSent[source] -= _amount;
```

`source` is not caller-supplied. It is the second return value of
`IL2ToL2CrossDomainMessenger.crossDomainMessageContext()`, which is `onlyEntered` and reads the
message context the messenger wrote to **transient storage** in `relayMessage` before dispatching the
call (`src/L2/L2ToL2CrossDomainMessenger.sol`, `crossDomainMessageContext` at :115, context stored at
:263). It is the same value the pre-existing `InvalidCrossDomainSender` check relies on for the sender
half, so the cap inherits exactly the trust the bridge already places in the messenger — no new trust
assumption, no new argument to spoof. `relayETH` already reverts for any caller that is not the
messenger.

Storage: one `mapping(uint256 => uint256) public netSent` at slot 0. The predeploy had no storage
before (its storage-layout snapshot was `[]`). This is in-convention for interop L2 predeploys —
`L2ToL2CrossDomainMessenger` likewise declares its mappings from slot 0 with no spacers — and is
upgrade-safe here because the contract is interop-gated and has never shipped with storage. Semver
1.0.1 → 1.1.0.

No new events. `SendETH` and `RelayETH` already carry `(amount, chain)`, so `netSent` is derivable
from existing logs and a `NetFlowUpdated` event would be redundant bytes and a bigger diff. Recorded
as a deliberate call, not an oversight.

## What this protects

- **Extraction.** The maximum ETH a lying P can remove from chain A is `netSent_A[P]`, i.e. exactly
  the ETH A voluntarily sent to P. A depset participant is bounded by its own credit line.
- **Chains that never received anything.** Cap 0 at genesis, forever, absent a real `sendETH`. A
  newly added, wholly malicious depset member can extract nothing.
- **Independence.** Caps are per chain ID. Sending to P grants no allowance to any other chain, and an
  exhausted or overdrawn counterparty cannot touch another's figure.
- **Double-spend of the credit.** The counter decrements once per relay and cannot go negative, so a
  successfully relayed credit cannot be spent twice even if message-level replay protection were
  somehow bypassed. (Message replay protection itself is unchanged: the messenger's
  `successfulMessages` map. The cap is a second, independent line.)

## What this does NOT protect

Stated plainly, because a cap is easy to over-read:

1. **It does not protect value already sent to P.** ETH that A sent to P is at P's mercy; that was
   already true, and the cap's whole content is that the loss stops there. A malicious P can steal all
   of it — it just cannot steal more than it.
2. **It is ETH-level only.** `SuperchainTokenBridge` / `SuperchainERC20` mint token balances on the
   same message machinery with no equivalent accounting, and a lying P can mint arbitrary amounts of
   any `SuperchainERC20` on A. The ERC20 analog (per-token, per-chain net-flow caps in
   `SuperchainTokenBridge`) is **noted as future work, not implemented here**. Any application-level
   cross-chain contract that mints on message receipt has the same exposure and is out of scope.
3. **It does not make P's state trustworthy.** Nothing here verifies anything about P. Real
   verification is the ZK stack on the shelf (`proofType: groth16`), per V1-ATTESTED.
4. **It does not protect P's own users** from A, or from P's operator.
5. **It is per-chain, not per-user.** Within a chain's cap, the attacker chooses the recipient. The
   cap bounds total loss, not who is harmed.
6. **Unrelayed sends still count.** If A sends 10 ETH to P and the message is never executed on P,
   A's `netSent[P]` stays 10 and P can still extract 10. This is correct against the invariant as
   stated (the 10 was voluntarily sent and already burned on A) but it is worth being explicit: the
   cap tracks *sent*, not *received*.

## The single counter — home-chain pinning on P (amendment, Karl 2026-08-26)

The uniform cap alone gives a *fragmented* view: P would have a separate credit line on every public
chain that ever sent it ETH, and no single number would tell you how much ETH P holds. The amendment
adds a second, complementary mechanism on the side we own.

`SuperchainETHBridgePinned` (P's genesis only) pins **both directions** to one configured home chain:

```solidity
sendETH(_to, _chainId):   if (_chainId == 0 || _chainId != homeChainId) revert NotHomeChain();
relayETH(_from,_to,_amt): if (source == 0 || source != homeChainId)     revert NotHomeChain();
```

It inherits the cap unchanged — the pin narrows, it never loosens. Because P can only send to, and
only accept from, the home chain, the home chain is by construction the sole mint/burn venue for P's
ETH, and therefore **`netSent_home[P]` on the home chain is the one global running figure for all ETH
P holds**. That is the consistent view. On P's own side, exactly one counter (`netSent[home]`) can
ever be nonzero; every other counter is provably dead, because the only writer of `netSent` is
`sendETH` and `sendETH` accepts nothing but the home chain.

Configuration: `homeChainId` is plain storage at **slot 1** with **no setter and no owner**. The only
thing that can write it is the chain's genesis allocation. It is storage rather than an `immutable`
because predeploy implementations are installed as deployed bytecode, which carries no
constructor-set immutables. An unconfigured contract (`homeChainId == 0`) is **inert** — every send
and every relay reverts — so a genesis mistake fails closed instead of open. Both properties are
asserted as tests, including the slot number the alloc depends on.

Shape: a subclass of `SuperchainETHBridge` overriding `sendETH`/`relayETH`/`version`, exactly
following this repo's existing predeploy-variant precedent (`L2ToL1MessagePasserCGT` over
`L2ToL1MessagePasser`). The cap logic is inherited, not copied, so the security-critical accounting
cannot drift between the two contracts. Cost of that choice: the base's `version` constant became a
`virtual` function and `sendETH`/`relayETH` became `public virtual` — four ABI-compatible lines in the
shared contract, no behavior change. Version string: `1.1.0+home-pinned`.

## Misroutes and stranding (honest accounting)

The pin removes the worse half of the problem and cannot remove the other half.

- **Outbound from P: impossible, not merely refused.** `sendETH` on P reverts *before* `ETHLiquidity.burn`,
  so a P user who names the wrong destination keeps their ETH. There is no P-side misroute case.
- **Inbound to P from a non-home chain B: strands.** B's bridge is the stock uniform-cap contract; it
  has no idea P is pinned. A `sendETH(to, P)` on B **burns at the source** and emits a valid message,
  and P's `relayETH` then refuses it (`NotHomeChain`) because B is not P's home. The ETH is burned on
  B and never minted on P. P cannot prevent this — the burn happens on a chain P does not control —
  and, note, neither could the uniform cap alone: the same funds would be equally stuck.
  **Those funds are stranded until message expiry + refund lands.** The designated escape hatch is
  the interop-expiry-refunds workstream (branch `karl/interop-expiry-refunds` in the optimism
  monorepo: refunds for undelivered interop messages). It is **referenced here, not implemented
  here** — out of scope for this lane. Until it lands, the operational answer is the front-end/routing
  one: only the home chain should ever be offered as a source for ETH into P.
- **Triangular routing on public chains generally.** The uniform cap makes public-chain ETH
  path-dependent: A→B→C→A fails on the last leg (A never sent to C), and ETH must return the way it
  came. For a general Superchain with hub-and-spoke ETH flow this is a real behavioral change to the
  ETH bridge, not a pure safety addition, and it is the main thing an upstream reviewer will push
  back on. Alternatives considered and not taken: cap only a configured set of untrusted chains
  (reintroduces configuration and an owner — rejected for v1); net across the whole depset rather
  than per counterparty (weaker invariant, one malicious member can spend the others' credit).
  **v1 accepts it**, since the Silhouette topology is bilateral and confining P's exit to the chain
  that funded it is the point.

## Genesis integration (v1 local harness)

The modified predeploy reaches a chain through the ordinary predeploy path — nothing bespoke:

- `scripts/L2Genesis.s.sol:631` `setSuperchainETHBridge()` calls
  `_setImplementationCode(Predeploys.SUPERCHAIN_ETH_BRIDGE)` behind
  `Predeploys.assertGates(..., DevFeatures.OPTIMISM_PORTAL_INTEROP, ...)`, so the bridge is written
  into the genesis allocs whenever interop is enabled — which the Silhouette harness requires anyway.
  The implementation bytecode written is whatever `forge build` produced from this worktree's
  `src/L2/SuperchainETHBridge.sol`, so the cap is present from block 0 with `netSent` empty (= all
  caps 0, the security property, asserted as a test).
- op-deployer consumes those allocs when generating the chain's genesis (`op-deployer` L2 genesis
  task, which runs `L2Genesis.s.sol` against the contracts-bedrock artifacts). For the local harness:
  build contracts in **this** worktree and point the deployer's artifacts locator at the resulting
  `forge-artifacts` (a `file://` locator), rather than at a tagged release, or the stock uncapped
  bridge is what lands in genesis. This is the single integration trap: a released-artifact locator
  silently ships the version without the cap.
- **Public chains (A): no storage pre-seeded.** Empty `netSent` is the intended initial state, and any
  pre-seeded value would be an un-audited credit line handed to a counterparty.
- **P: the pinned variant plus one storage word.** P needs a different implementation behind the same
  predeploy proxy `0x42...24`, and exactly one alloc storage entry. Two routes:
  1. *Harness route (v1, no upstream diff).* Generate P's allocs normally, then post-process the JSON:
     two edits, on **two different accounts**, because this predeploy is proxied:
     - **Code** goes on the *implementation* account:
       `Predeploys.predeployToCodeNamespace(0x42..24)` =
       `0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30024` (`src/libraries/Predeploys.sol:229`) — replace its
       code with `SuperchainETHBridgePinned`'s deployed bytecode.
     - **Storage** goes on the *proxy* account `0x4200000000000000000000000000000000000024`, since the
       proxy delegatecalls and therefore owns the state: set key `0x00..01` (i.e. `homeChainId`) to A's
       chain ID. Putting it on the implementation account instead is the easy mistake and yields a
       silently inert bridge.

     Nothing else changes; `netSent` stays empty. The slot number is pinned by a test (the tests etch
     code and storage on the same account, so they fix the slot, not the account split); the namespace
     derivation is pinned by the library function above.
  2. *In-tree route (if this goes upstream).* Add a `VariantKind` to `src/libraries/Predeploys.sol`
     alongside `DEFAULT`/`CGT` and thread the selector through `resolveVariant` and `L2Genesis.s.sol`.
     Deliberately **not** done here: the existing selector is keyed solely on custom-gas-token, so a
     third kind touches shared upstream library code for a Silhouette-specific chain. Recorded as the
     clean shape.
  A verification step belongs in the harness bring-up: read `homeChainId()` off P and assert it equals
  A's chain ID before any ETH moves. Zero means the swap silently didn't happen, and the bridge is
  inert (fails closed) rather than unpinned.
- `L2ContractsManager` (the upgrade path for already-running chains) references implementations by
  name and address, not by version, so nothing there pins 1.0.1 and needs bumping. For chains already
  running the old bridge, an upgrade would start every counterparty at cap 0 — correct, but it would
  strand in-flight ETH already sent under the old rules. Not a v1 concern (P restarts on a fresh
  genesis per DR-5) and flagged for anyone who takes this upstream.

## Tests

`packages/contracts-bedrock/test/L2/SuperchainETHBridge.t.sol`, contract
`SuperchainETHBridge_NetFlowCap_Test` plus assertions added to the two pre-existing success tests.
Cases: genesis cap is zero for any chain ID; relay with no net flow reverts; overdraft by 1 wei
reverts; a reverted relay consumes none of the cap; exact-amount relay empties the cap and a second
relay reverts (no double decrement); partial relays drain to exactly the sent total; caps independent
across chain IDs; A→P→A→P round-trip accounting; zero-amount relay allowed at a zero cap; zero-value
send credits nothing; repeated sends accumulate; and a 16-step fuzzed interleaving of sends and relays
asserting `relayed ≤ sent` and `netSent == sent − relayed` after every step.

Credit is seeded in tests only through a real `sendETH` (helper `_sendToChain`), never by writing
storage directly — so no test can grant an allowance the contract itself would not.

`test/L2/SuperchainETHBridgePinned.t.sol` covers the pin: the genesis slot and the getter agree;
`netSent` still occupies slot 0; version is the variant string; unconfigured is inert for both
directions even with a credited cap; `sendETH` to any non-home chain reverts **with the sender's
balance intact and no cap credited** (no burn); chain ID 0 refused; relay from the home chain
succeeds and draws down the cap; relay from any non-home source is refused *by the pin* even when its
cap has been force-credited via `vm.store` (proving the pin is an independent check, not a restatement
of the cap); the cap still binds on home-chain relays; a non-messenger caller still gets
`Unauthorized`; and the single-counter property — one counter carries all traffic, no other counter can
become nonzero, and a 12-step fuzzed interleaving keeps `netSent[home] == sent − relayed` at every
step. Non-home caps are force-written with `vm.store` **only** in tests whose point is that the pin
refuses them anyway.
