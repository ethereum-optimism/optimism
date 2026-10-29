# Compliance mocks

Devnet-only stand-ins for the Compliance Module and its Policy, so the bridge screener can be
built against the production interfaces before the real contracts exist.

**Not for production.** Unaudited, and they hold real value on whatever chain they are deployed
to.

## What is here

| Contract | Implements | Role |
| --- | --- | --- |
| `MockComplianceModule.sol` | `IComplianceModule`, which includes `IBridgeHook` | Takes custody of held items, owns the verdict registry, initiates completion |
| `MockPolicy.sol` | `IPolicy` | Decides what is held. Operator-driven, so the devnet is scriptable |

Deployed by `scripts/deploy/DeployComplianceMocks.s.sol`, which also points a live Portal's
`bridgeHook` at the module.

## Deploying

Broadcast from the L1 ProxyAdmin owner, which must also be the `owner` input: every call the
script makes is either `onlyOwner` on the ProxyAdmin or an owner-gated setter on a mock.

```
forge script scripts/deploy/DeployComplianceMocks.s.sol \
  --rpc-url $L1_RPC --broadcast --private-key $L1PAO_KEY \
  --sig 'run((address,address,address,address))' \
  "($PORTAL,$PROXY_ADMIN,$L1PAO,$SCREENER)"
```

The script refuses to run twice. Re-pointing an existing hook means removing it first, which is
a deliberate decision rather than something a redeploy should do silently.

### How the hook gets wired

The Portal already stores a bridge hook and `initialize` already sets one from a non-zero
argument, but `initialize` is a `reinitializer` whose version the live proxy has consumed, and
that version is `immutable` in the implementation. So the script clears the proxy's
`_initialized` byte and re-runs `initialize`, the same sequence OPCM uses in
`OPContractsManagerUtils.upgrade`:

1. `proxyAdmin.upgrade(portal, storageSetter)`
2. `storageSetter.setBytes32(slot0, current & ~0xff)`, masking only that byte so `_initializing`
   survives
3. `proxyAdmin.upgradeAndCall(portal, originalImpl, initialize(..., module))`

Step 3 upgrades to the implementation the Portal **already had**. Every value `initialize` is
given is read off the Portal first, so the call changes nothing but the hook.

## What the screener sees

### Today

The real Portal does not call the hook yet. Its `screenDeposit` / `screenWithdrawal` call sites
arrive with ethereum-optimism/optimism#23736 and the withdrawal path behind it, and
`completeDepositTransaction` / `completeWithdrawalTransaction` are empty stubs until the same
change lands. So right now:

- The module and policy are deployed, wired, and reachable at stable addresses.
- The verdict write path works end to end.
- No `Held` is emitted, because nothing holds anything.
- A completion emits `Completed` and changes nothing on the Portal.

Setting `bridgeHook` has exactly one effect on the chain today: the module joins the Portal's
unsafe-target set, so a withdrawal cannot target it.

### Once the Portal paths land

| Event | Emitted by | When |
| --- | --- | --- |
| `Held(bytes32 indexed id, Item item)` | module | An item is deferred into custody |
| `VerdictRecorded(bytes32 indexed id, uint64 expiresAt)` | module | The screener clears an item |
| `VerdictRevoked(bytes32 indexed id)` | module | A clearance is withdrawn |
| `Completed(bytes32 indexed id)` | module | A held item is released on its committed terms |
| `ItemObserved(bytes32 indexed id, Item item, uint8 kind, bool pass)` | policy | Every screening call |

`Held` carries the **full item**, not just its ID. The module stores only two timestamps, so
that log is the only on-chain record of the preimage a completion needs. Note the `Item` has a
dynamic `bytes data` field, so it ABI-encodes with an offset table rather than as a packed
struct.

`ItemObserved` is mock-only and exists to close a real gap: a deposit that **proceeds** leaves
no on-chain record of its item ID, because `TransactionDeposited` does not carry the deposit
counter. Without this feed, recovering the ID of a proceeding deposit means replaying every
deposit since genesis. `kind` is `0` for a deposit arriving, `1` for a withdrawal arriving, `2`
for a held item completing.

A **rejected** item emits nothing at all. Rejection reverts the whole transaction, which
discards the logs with it. That is the one outcome with no event, by construction.

## Driving the devnet

### Choosing what happens

The policy answers from storage. Three per-direction defaults, plus a per-item override that
beats the default:

```solidity
enum Outcome  { Proceed, Hold, Reject }
enum Override { Unset, Proceed, Hold, Reject }   // Unset falls through to the default

setDefaultDeposit(Outcome)      // deposits arriving
setDefaultWithdrawal(Outcome)   // withdrawals arriving
setDefaultRelease(Outcome)      // held items completing
setOverride(bytes32 id, Override)
```

`Proceed` returns true, `Hold` returns false, `Reject` reverts. All owner-gated.

Use the override for reproducible fixtures. Holding one deposit while another proceeds by
flipping a default between two user transactions races everything else in flight; pinning the
ID does not. Overrides can be set **before** the item exists, so a scenario can be scripted up
front.

### Writing verdicts

From the screener address:

```solidity
policy.clear(bytes32 id, uint64 expiresAt)   // -> module.recordVerdict, emits VerdictRecorded
policy.revoke(bytes32 id)                    // -> module.revokeVerdict, emits VerdictRevoked
policy.clearAndCompleteDeposit(Item, uint64 expiresAt)
policy.clearAndCompleteWithdrawal(Item, uint64 expiresAt)
```

Expiry is chosen entirely off chain, per the decision that max clearance age is not computed on
chain. An expired clearance counts as none, so the item is held like any other uncleared one
rather than paid.

The two `clearAndComplete*` helpers take an expiry that `IVisionPolicyV1` does not, because the
real policy derives it from its own configuration and this one cannot.

Completion itself is permissionless on the module, so a cleared user never depends on the
operator to send the transaction.

### Item IDs

`keccak256(abi.encode(item))` over `BridgeHookTypes.Item`, computed identically by the Portal
and the module. `src/libraries/BridgeHookHashing.sol` is the reference. An off-chain
implementation must match it byte for byte, or verdicts will not bind to the items they name.

## Known divergences

- `recordVerdict(bytes32, uint64)` and `expiresAt` follow `flux/compliance-specs`. specs#977
  landed upstream keeping `clearedAt`, so this is unsettled.
- `l1StandardBridge.holdERC20` / `restoreERC20` are still empty stubs, so once deposit holds
  land, a held ERC-20 keeps its tokens in the bridge and the module's `heldTokens` stays zero.
  The module only credits tokens it actually received.
- The module's `owner` cannot be rotated after deployment.
