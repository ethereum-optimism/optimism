# Private ETH profile

The private chain uses ordinary ETH deposits and stock messenger send, resend and relay semantics.
A forced transaction can create an initiating message in private state. Only sequencer-published
replay transactions put that message into the public projection's interop history.

## Publication and sequencer outages

The batcher can publish private events from both sequenced and forced transactions. Deposits
execute with ordinary OP semantics on both the private chain and its public projection. There is
no projection-specific EVM deposit hook or receipt-log suppression. Minting, nonce changes, gas,
transfers and execution outside the protected protocol contracts follow the execution client's
normal rules; reverting a contract call does not undo deposit minting or the sender nonce.

The public-projection genesis enables `L1Block.isFeatureEnabled["PRIVATE_PROJECTION"]` and installs
the matching guarded contracts. With that flag enabled, `EventReplayer.replayEvent`, the replay
messenger's `replaySentMessage`, and the inbox's validation, export and import paths require the
immediate caller to match the current `L1Block.batcherHash`. Forwarders cannot borrow `tx.origin`'s
authority. Batcher rotation follows ordinary L1 attributes updates. The feature is disabled on
ordinary private/public chains, preserving their permissionless inbox behavior.

This protects projection protocol calls from unauthorized deposits; it does not make all deposits
inert or prohibit arbitrary application logs. The batcher remains a trusted publisher, and the
guard authorizes its identity rather than detecting transaction type. Direct `sendMessage`,
`resendMessage` and `relayMessage` calls on the projection messenger remain unsupported.

The supernode's claim scanner accepts range attestations only from sequenced transactions and
ignores deposit claim calldata regardless of receipt status.

Only the current batcher, read from the standard L1 attributes, can post projection range claims.
This prevents a forced call from advancing the claim cursor and blocking recovery. Batcher rotation
uses the existing L1 configuration; no additional key or genesis setting is needed.

The projection is identified from its installed genesis messenger, claim registry and event
replayer implementations. Both `--rollup.private` and a materialized projection genesis select the
same behavior. No new op-deployer override or messenger-policy development flag is needed.

When the private sequencer is offline, normal sequencing-window expiry lets public derivation
produce blocks without sequencer transactions. Those blocks still execute system and user deposits,
but unauthorized callers cannot publish replayed messages or invoke the guarded inbox paths.
The production sequencing window remains unchanged.

After private execution and publication have caught up, a user can force a new L1 deposit to
create a new initiating message. Its private execution receives a new message nonce; the operator
publishes the event and the recipient uses its new public identifier. An expired fallback position
is not retroactively populated with that event.

Separately, anyone with the original message parameters can call the existing
`resendMessage`. It checks the private messenger's stored hash and emits the same message again.
The new event has a new block position and timestamp; consumers use that newly published identifier.
The message hash and nonce are unchanged, so destination-side replay protection still applies.
Publication requires the sequencer to return; forced creation alone does not guarantee outbound
message delivery. Executing messages remain subject to the existing inbox/access-list validation.

## ETH backing and bridge permissions

`SuperchainETHBridge` has independent `allowedSendChain` and `allowedRelayChain` mappings, initially
empty. The L2 ProxyAdmin owner calls
`setChainPermissions(uint256 chainId, bool allowSend, bool allowRelay)`. Sending checks the
destination before burning; relaying checks the authenticated messenger source before releasing
liquidity and retains the remote bridge identity check.

L1's existing `ETH_LOCKBOX` feature controls lockbox use separately. An enabled flag does not prove
that a peer shares the same backing. Route approval records that governance decision. For A → B,
A must allow sending to B and B must allow relaying from A. Stop sends and drain pending transfers
before revoking relay permission.

Keep the private profile's native routes empty: the renderer rejects native bridge messages.
Allowlisting a peer does not add private native ETH bridge support. Ordinary ETH deposit funding
and generic application messaging remain available.

## NetChef integration

Use stock op-deployer `0.8.0-rc.2` with the matching custom contract-artifact bundle and interop
active at genesis. NetChef generates the genesis and rollup artifacts normally. Private ELs, the
projection EL, batcher and supernode use the same private-chain genesis source. The projection EL
retains `--rollup.private`; the supernode derives its projection internally. All component images
must match the contract bundle and projection genesis. The contract guards replace the experimental
deposit no-op and receipt-log-suppression rules. Existing prototype databases require a coordinated
migration or a fresh devnet; this is not a rolling, backward-compatible client update.

There is no manual genesis transformation or upload step in this deployment path. A changed
genesis requires fresh chain databases; redownloading genesis cannot migrate an initialized DB.
Verify the actual L1 portal backing before funding a live chain.

The standalone `op-private-interop/cmd/genesis` remains an offline tool for preparing a supported
external ETH source with the pinned bridge implementation. It cannot change an L1 portal's asset
mode or backing. Its materialized projection is an inspection artifact when using `--rollup.private`;
do not project the genesis twice.

## Validation scope

The acceptance suite covers deposit funding, sequencer publication of forced sends, authenticated
resend, bidirectional application messaging, and public progress through a private-node outage.
The recovery test restarts the private nodes and batcher, forces a fresh deposit, checks the new
message nonce and waits for the recipient's cross-safe frontier. Contract tests cover feature-off
behavior, caller authorization, forwarding, batcher rotation and ordinary access-list validation.
The online deposit test checks ordinary projection funding and a reverting direct messenger call.
The outage fixture shortens the sequencing window to ten L1 blocks; this does not change the
devnet's production setting.

`RUST_JIT_BUILD=1 mise x -- go run ./op-up --private-interop --smoke` runs chain-ops `interopsmoke`
in-process with the private message-position resolver. Native ETH bridging is skipped. Standalone
remote private-pair smoke still requires a resolver. Local tests and genesis-target NetChef
simulation do not establish that a live Sepolia deployment is healthy.

V1 remains operator-attested, with the existing proof-bytes extension reserved for later
verification. Private-state proofs and private withdrawal settlement are outside this patch.
