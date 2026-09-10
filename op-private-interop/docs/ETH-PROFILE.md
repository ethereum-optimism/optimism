# Private ETH profile

The private chain uses ordinary ETH deposits and stock messenger send, resend and relay semantics.
A forced transaction can create an initiating message in private state. Only sequencer-published
replay transactions put that message into the public projection's interop history.

## Publication and sequencer outages

The batcher can publish private events from both sequenced and forced transactions. On the
public projection, user deposits are successful zero-effect no-ops in every block, including
sequencing-window fallback blocks. They remain in the transaction list and have receipts, but
consume zero execution gas and do not mint ETH, transfer value, increment sender nonces, create
contracts, write storage or emit logs. The receipt's deposit nonce records the unchanged sender
nonce. This prevents a forced call from mutating projection protocol state as well as injecting
public messages. Ordinary private and public chains retain normal deposit execution.

This also preserves deterministic message positions: the public block's replay logs match the
private block's selected interop logs, in order, starting at log index zero. User deposits cannot
insert application logs ahead of that sequence, including through arbitrary targets, multicalls
that catch inner reverts, or contract creation. These public indices need not equal the private
block's unfiltered log indices. Receipt and transaction positions still include the no-op deposits.

The experimental `PRIVATE_PROJECTION` contract guards have been removed. Guarding protocol
entrypoints alone cannot stop a deposit from emitting logs through an unrelated contract. The
execution rule skips the whole user deposit before entering the EVM; it does not execute and then
strip receipt logs. Private execution and recovery continue to execute ordinary deposits.

L1-attributes and network-upgrade deposits still execute. The projection checks the L1-info source
hash and block position, and recognizes complete canonical upgrade transactions. It does not
classify system deposits by destination or the legacy `is_system_transaction` gas flag. Copying
system calldata into a portal deposit does not grant execution. New network upgrades must be
included in the canonical upgrade catalogue used by this classification.

The supernode's claim scanner ignores deposit transactions, even when their receipts say success:
a no-op receipt does not mean the registry executed or authorized the supplied claim calldata.

Only the current batcher, read from the standard L1 attributes, can post projection range claims.
This prevents a forced call from advancing the claim cursor and blocking recovery. Batcher rotation
uses the existing L1 configuration; no additional key or genesis setting is needed.

The projection is identified from its installed genesis messenger, claim registry and event
replayer implementations. Both `--rollup.private` and a materialized projection genesis select the
same behavior. No new op-deployer override or messenger-policy development flag is needed.

When the private sequencer is offline, normal sequencing-window expiry lets public derivation
produce blocks without sequencer transactions or interop events. Those blocks still include
required system/deposit transactions. The production sequencing window remains unchanged.

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
must match the contract bundle and projection behavior. The no-op rule changes projection
consensus from the earlier log-suppression and contract-guard prototypes; their databases require a
coordinated migration or a fresh devnet. It is not a rolling, backward-compatible client update.

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
message nonce and waits for the recipient's cross-safe frontier. Rust execution tests verify no-op
balances, nonces, storage, contract creation, logs and gas, with and without post-exec accounting.
`TestPrivateDepositLogsDoNotShiftPublishedMessages` forces a multicall containing two WETH logs
and two initiating messages. It checks the entire public log sequence without emitter filtering,
the actual indices of both replayed messages, and their successful cross-safe relays.
The outage fixture shortens the sequencing window to ten L1 blocks; this does not change the
devnet's production setting.

`RUST_JIT_BUILD=1 mise x -- go run ./op-up --private-interop --smoke` runs chain-ops `interopsmoke`
in-process with the private message-position resolver. Native ETH bridging is skipped. Standalone
remote private-pair smoke still requires a resolver. Local tests and genesis-target NetChef
simulation do not establish that a live Sepolia deployment is healthy.

V1 remains operator-attested, with the existing proof-bytes extension reserved for later
verification. Private-state proofs and private withdrawal settlement are outside this patch.
