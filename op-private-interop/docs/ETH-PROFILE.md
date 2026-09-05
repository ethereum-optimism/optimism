# Private ETH profile

The private chain uses ordinary ETH deposits and stock messenger send, resend and relay semantics.
A forced transaction can create an initiating message in private state. Only sequencer-published
replay transactions put that message into the public projection's interop history.

## Publication and sequencer outages

The batcher can publish private events from both sequenced and forced transactions. The projection
EL executes its own L1 deposits, but clears their receipt logs before computing receipt roots and
bloom filters. This prevents deposits from independently injecting public messages, including a
direct call to the projection's replay messenger. Deposit status, gas accounting, balances and
state changes retain their normal execution semantics. Private and ordinary public chains keep
their deposit logs.

Only the current batcher, read from the standard L1 attributes, can post projection range claims.
This prevents a forced call from advancing the claim cursor and blocking recovery. Batcher rotation
uses the existing L1 configuration; no additional key or genesis setting is needed.

The projection is identified from its installed genesis messenger, claim registry and event
replayer implementations. Both `--rollup.private` and a materialized projection genesis select the
same behavior. No new op-deployer override or messenger-policy development flag is needed.

When the private sequencer is offline, normal sequencing-window expiry lets public derivation
produce blocks without sequencer transactions or interop events. Those blocks still include
required system/deposit transactions. The production sequencing window remains unchanged.

The batcher uses the projection EL's latest derived block to skip expired fallback blocks through
its stock pruning path. This endpoint must follow L1 derivation only, with sequencing and unsafe
P2P input disabled. The EL `safe` label can lag fallback while cross-chain safety catches up,
causing batches to repeatedly miss their inclusion window. This is a publication cursor only; it does not mark private execution safe. Wait for
new range claims to be published before resending, since a restarting sequencer can still be
catching up through blocks whose publication windows have expired.

Once publication resumes, anyone with the original message parameters can call the existing
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
must match the contract bundle and projection behavior.

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
The recovery test restarts the private nodes and batcher, resends an omitted forced message, and
waits for the recipient's cross-safe frontier. The outage fixture shortens the sequencing window
to ten L1 blocks; this does not change the devnet's production setting.

`RUST_JIT_BUILD=1 mise x -- go run ./op-up --private-interop --smoke` runs chain-ops `interopsmoke`
in-process with the private message-position resolver. Native ETH bridging is skipped. Standalone
remote private-pair smoke still requires a resolver. Local tests and genesis-target NetChef
simulation do not establish that a live Sepolia deployment is healthy.

V1 remains operator-attested, with the existing proof-bytes extension reserved for later
verification. Private-state proofs and private withdrawal settlement are outside this patch.
