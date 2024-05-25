# Rollup-node P2P interface

The [rollup node](./rollup-node.md) has an optional peer-to-peer (P2P) network service to improve the latency between
the view of sequencers and the rest of the network by bypassing the L1 in the happy case,
without relying on a single centralized endpoint.

This also enables faster historical sync to be bootstrapped by providing block headers to sync towards,
and only having to compare the L2 chain inputs to the L1 data as compared to processing everything one block at a time.

The rollup node will *always* prioritize L1 and reorganize to match the canonical chain.
The L2 data retrieved via the P2P interface is strictly a speculative extension, also known as the "unsafe" chain,
to improve the happy case performance.

This also means that P2P behavior is a soft-rule: nodes keep each other in check with scoring and eventual banning
of malicious peers by identity or IP. Any behavior on the P2P layer does not affect the rollup security, at worst nodes
rely on higher-latency data from L1 to serve.

In summary, the P2P stack looks like:

- Discovery to find peers: [Discv5][discv5]
- Connections, peering, transport security, multiplexing, gossip: [LibP2P][libp2p]
- Application-layer publishing and validation of gossiped messages like L2 blocks.

This document only specifies the composition and configuration of these network libraries.
These components have their own standards, implementations in Go/Rust/Java/Nim/JS/more,
and are adopted by several other blockchains, most notably the [L1 consensus layer (Eth2)][eth2-p2p].

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [P2P configuration](#p2p-configuration)
  - [Identification](#identification)
  - [Discv5](#discv5)
    - [Structure](#structure)
  - [LibP2P](#libp2p)
    - [Transport](#transport)
    - [Dialing](#dialing)
    - [NAT](#nat)
    - [Peer management](#peer-management)
    - [Transport security](#transport-security)
    - [Protocol negotiation](#protocol-negotiation)
    - [Identify](#identify)
    - [Ping](#ping)
    - [Multiplexing](#multiplexing)
    - [GossipSub](#gossipsub)
      - [Content-based message identification](#content-based-message-identification)
      - [Message compression and limits](#message-compression-and-limits)
      - [Message ID computation](#message-id-computation)
    - [Heartbeat and parameters](#heartbeat-and-parameters)
    - [Topic configuration](#topic-configuration)
    - [Topic validation](#topic-validation)
- [Gossip Topics](#gossip-topics)
  - [`blocksv1`](#blocksv1)
  - [`blocksv2`](#blocksv2)
    - [Block encoding](#block-encoding)
    - [Block signatures](#block-signatures)
    - [Block validation](#block-validation)
      - [Block processing](#block-processing)
      - [Block topic scoring parameters](#block-topic-scoring-parameters)
- [Req-Resp](#req-resp)
  - [`payload_by_number`](#payload_by_number)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## P2P configuration

### Identification

Nodes have a **separate** network- and consensus-identity.
The network identity is a `secp256k1` key, used for both discovery and active LibP2P connections.

Common representations of network identity:

- `PeerID`: a LibP2P specific ID derived from the pubkey (through protobuf encoding, typing and hashing)
- `NodeID`: a Discv5 specific ID derived from the pubkey (through hashing, used in the DHT)
- `Multi-address`: an unsigned address, containing: IP, TCP port, PeerID
- `ENR`: a signed record used for discovery, containing: IP, TCP port, UDP port, signature (pubkey can be derived)
  and L2 network identification. Generally encoded in base64.

### Discv5

#### Structure

The Ethereum Node Record (ENR) for an Optimism rollup node must contain the following values, identified by unique keys:

- An IPv4 address (`ip` field) and/or IPv6 address (`ip6` field).
- A TCP port (`tcp` field) representing the local libp2p listening port.
- A UDP port (`udp` field) representing the local discv5 listening port.
- An OpStack (`opstack` field) L2 network identifier

The `opstack` value is encoded as a single RLP `bytes` value, the concatenation of:

- chain ID (`unsigned varint`)
- fork ID (`unsigned varint`)

Note that DiscV5 is a shared DHT (Distributed Hash Table): the L1 consensus and execution nodes,
as well as testnet nodes, and even external IOT nodes, all communicate records in this large common DHT.
This makes it more difficult to censor the discovery of node records.

The discovery process in Optimism is a pipeline of node records:

1. Fill the table with `FINDNODES` if necessary (Performed by Discv5 library)
2. Pull additional records with searches to random Node IDs if necessary
   (e.g. iterate [`RandomNodes()`][discv5-random-nodes] in Go implementation)
3. Pull records from the DiscV5 module when looking for peers
4. Check if the record contains the `opstack` entry, verify it matches the chain ID and current or future fork number
5. If not already connected, and not recently disconnected or put on deny-list, attempt to dial.

### LibP2P

#### Transport

TCP transport. Additional transports are supported by LibP2P, but not required.

#### Dialing

Nodes should be publicly dialable, not rely on relay extensions, and able to dial both IPv4 and IPv6.

#### NAT

The listening endpoint must be publicly facing, but may be configured behind a NAT.
LibP2P will use PMP / UPNP based techniques to track the external IP of the node.
It is recommended to disable the above if the external IP is static and configured manually.

#### Peer management

The default is to maintain a peer count with a tide-system based on active peer count:

- At "low tide" the node starts to actively search for additional peer connections.
- At "high tide" the node starts to prune active connections,
  except those that are marked as trusted or have a grace period.

Peers will have a grace period for a configurable amount of time after joining.
In an emergency, when memory runs low, the node should start pruning more aggressively.

Peer records can be persisted to disk to quickly reconnect with known peers after restarting the rollup node.

The discovery process feeds the peerstore with peer records to connect to, tagged with a time-to-live (TTL).
The current P2P processes do not require selective topic-specific peer connections,
other than filtering for the basic network participation requirement.

Peers may be banned if their performance score is too low, or if an objectively malicious action was detected.

Banned peers will be persisted to the same data-store as the peerstore records.

TODO: the connection gater does currently not gate by IP address on the dial Accept-callback.

#### Transport security

[Libp2p-noise][libp2p-noise], `XX` handshake, with the `secp256k1` P2P identity, as popularized in Eth2.
The TLS option is available as well, but `noise` should be prioritized in negotiation.

#### Protocol negotiation

[Multistream-select 1.0][multistream-select] (`/multistream/1.0.0`) is an interactive protocol
used to negotiate sub-protocols supported in LibP2P peers. Multistream-select 2.0 may be used in the future.

#### Identify

LibP2P offers a minimal identification module to share client version and programming language.
This is optional and can be disabled for enhanced privacy.
It also includes the same protocol negotiation information, which can speed up initial connections.

#### Ping

LibP2P includes a simple ping protocol to track latency between connections.
This should be enabled to help provide insight into the network health.

#### Multiplexing

For async communication over different channels over the same connection, multiplexing is used.
[mplex][mplex] (`/mplex/6.7.0`) is required, and [yamux][yamux] (`/yamux/1.0.0`) is recommended but optional

#### GossipSub

[GossipSub 1.1][gossipsub] (`/meshsub/1.1.0`, i.e. with peer-scoring extension) is a pubsub protocol for mesh-networks,
deployed on L1 consensus (Eth2) and other protocols such as Filecoin, offering lots of customization options.

##### Content-based message identification

Messages are deduplicated, and filtered through application-layer signature verification.
Thus origin-stamping is disabled and published messages must only contain application data,
enforced through a [`StrictNoSign` Signature Policy][signature-policy]

This provides greater privacy, and allows sequencers (consensus identity) to maintain
multiple network identities for redundancy.

##### Message compression and limits

The application contents are compressed with [snappy][snappy] single-block-compression
(as opposed to frame-compression), and constrained to 10 MiB.

##### Message ID computation

[Same as L1][l1-message-id], with recognition of compression:

- If `message.data` has a valid snappy decompression, set `message-id` to the first 20 bytes of the `SHA256` hash of
  the concatenation of `MESSAGE_DOMAIN_VALID_SNAPPY` with the snappy decompressed message data,
  i.e. `SHA256(MESSAGE_DOMAIN_VALID_SNAPPY + snappy_decompress(message.data))[:20]`.
- Otherwise, set `message-id` to the first 20 bytes of the `SHA256` hash of
  the concatenation of `MESSAGE_DOMAIN_INVALID_SNAPPY` with the raw message data,
  i.e. `SHA256(MESSAGE_DOMAIN_INVALID_SNAPPY + message.data)[:20]`.

#### Heartbeat and parameters

GossipSub [parameters][gossip-parameters]:

- `D` (topic stable mesh target count): 8
- `D_low` (topic stable mesh low watermark): 6
- `D_high` (topic stable mesh high watermark): 12
- `D_lazy` (gossip target): 6
- `heartbeat_interval` (interval of heartbeat, in seconds): 0.5
- `fanout_ttl` (ttl for fanout maps for topics we are not subscribed to but have published to, in seconds): 24
- `mcache_len` (number of windows to retain full messages in cache for `IWANT` responses): 12
- `mcache_gossip` (number of windows to gossip about): 3
- `seen_ttl` (number of heartbeat intervals to retain message IDs): 130 (= 65 seconds)

Notable differences from L1 consensus (Eth2):

- `seen_ttl` does not need to cover a full L1 epoch (6.4 minutes), but rather just a small window covering latest blocks
- `fanout_ttl`: adjusted to lower than `seen_ttl`
- `mcache_len`: a larger number of heartbeats can be retained since the gossip is much less noisy.
- `heartbeat_interval`: faster interval to reduce latency, bandwidth should still be reasonable since
  there are far fewer messages to gossip about each interval than on L1 which uses an interval of 0.7 seconds.

#### Topic configuration

Topics have string identifiers and are communicated with messages and subscriptions.
`/optimism/chain_id/hardfork_version/Name`

- `chain_id`: replace with decimal representation of chain ID
- `hardfork_version`: replace with decimal representation of hardfork, starting at `0`
- `Name`: topic application-name

Note that the topic encoding depends on the topic, unlike L1,
since there are less topics, and all are snappy-compressed.

#### Topic validation

To ensure only valid messages are relayed, and malicious peers get scored based on application behavior,
an [extended validator][extended-validator] checks the message before it is relayed or processed.
The extended validator emits one of the following validation signals:

- `ACCEPT` valid, relayed to other peers and passed to local topic subscriber
- `IGNORE` scored like inactivity, message is dropped and not processed
- `REJECT` score penalties, message is dropped

## Gossip Topics

There are two topics for distributing blocks to other nodes faster than proxying through L1 would. These are:

### `blocksv1`

Pre-Canyon/Shanghai blocks are broadcast on `/optimism/<chainId>/0/blocks`.

### `blocksv2`

Post-Canyon/Shanghai blocks are broadcast on `/optimism/<chainId>/1/blocks`.

#### Block encoding

A block is structured as the concatenation of:

- `signature`: A `secp256k1` signature, always 65 bytes, `r (uint256), s (uint256), y_parity (uint8)`
- `payload`: A SSZ-encoded `ExecutionPayload`, always the remaining bytes.

The topic uses Snappy block-compression (i.e. no snappy frames):
the above needs to be compressed after encoding, and decompressed before decoding.

#### Block signatures

The `signature` is a `secp256k1` signature, and signs over a message:
`keccak256(domain ++ chain_id ++ payload_hash)`, where:

- `domain` is 32 bytes, reserved for message types and versioning info. All zero for this signature.
- `chain_id` is a big-endian encoded `uint256`.
- `payload_hash` is `keccak256(payload)`, where `payload` is the SSZ-encoded `ExecutionPayload`

The `secp256k1` signature must have `y_parity = 1 or 0`, the `chain_id` is already signed over.

#### Block validation

An [extended-validator] checks the incoming messages as follows, in order of operation:

- `[REJECT]` if the compression is not valid
- `[REJECT]` if the block encoding is not valid
- `[REJECT]` if the `payload.timestamp` is older than 60 seconds in the past
  (graceful boundary for worst-case propagation and clock skew)
- `[REJECT]` if the `payload.timestamp` is more than 5 seconds into the future
- `[REJECT]` if the `block_hash` in the `payload` is not valid
- `[REJECT]` if the block is on the V1 topic and has withdrawals
- `[REJECT]` if the block is on the V2 topic and does not have withdrawals
- `[REJECT]` if the block is on the V2 topic and has a non-zero amount of withdrawals
- `[REJECT]` if more than 5 different blocks have been seen with the same block height
- `[IGNORE]` if the block has already been seen
- `[REJECT]` if the signature by the sequencer is not valid
- Mark the block as seen for the given block height

The block is signed by the corresponding sequencer, to filter malicious messages.
The sequencer model is singular but may change to multiple sequencers in the future.
A default sequencer pubkey is distributed with rollup nodes and should be configurable.

Note that blocks that a block may still be propagated even if the L1 already confirmed a different block.
The local L1 view of the node may be wrong, and the time and signature validation will prevent spam.
Hence, calling into the execution engine with a block lookup every propagation step is not worth the added delay.

##### Block processing

A node may apply the block to their local engine ahead of L1 availability, if it ensures that:

- The application of the block is reversible, in case of a conflict with delayed L1 information
- The subsequent forkchoice-update ensures this block is recognized as "unsafe"
  (see [`engine_forkchoiceUpdatedV2`](./exec-engine.md#engine_forkchoiceupdatedv2))

##### Block topic scoring parameters

TODO: GossipSub per-topic scoring to fine-tune incentives for ideal propagation delay and bandwidth usage.

## Req-Resp

The op-node implements a similar request-response encoding for its sync protocols as the L1 ethereum Beacon-Chain.
See [L1 P2P-interface req-resp specification][eth2-p2p-reqresp] and [Altair P2P update][eth2-p2p-altair-reqresp].

However, the protocol is simplified, to avoid several issues seen in L1:

- Error strings in responses, if there is any alternative response,
  should not need to be compressed or have an artificial global length limit.
- Payload lengths should be fixed-length: byte-by-byte uvarint reading from the underlying stream is undesired.
- `<context-bytes>` are relaxed to encode a `uint32`, rather than a beacon-chain `ForkDigest`.
- Payload-encoding may change per hardfork, so is not part of the protocol-ID.
- Usage of response-chunks is specific to the req-resp method: most basic req-resp does not need chunked responses.
- Compression is encouraged to be part of the payload-encoding, specific to the req-resp method, where necessary:
  pings and such do not need streaming frame compression etc.

And the protocol ID format follows the same scheme as L1,
except the trailing encoding schema part, which is now message-specific:

```text
/ProtocolPrefix/MessageName/SchemaVersion/
```

The req-resp protocols served by the op-node all have `/ProtocolPrefix` set to `/opstack/req`.

Individual methods may include the chain ID as part of the `/MessageName` segment,
so it's immediately clear which chain the method applies to, if the communication is chain-specific.
Other methods may include chain-information in the request and/or response data,
such as the `ForkDigest` `<context-bytes>` in L1 beacon chain req-resp protocols.

Each segment starts with a `/`, and may contain multiple `/`, and the final protocol ID is suffixed with a `/`.

### `payload_by_number`

This is an optional chain syncing method, to request/serve execution payloads by number.
This serves as a method to fill gaps upon missed gossip, and sync short to medium ranges of unsafe L2 blocks.

Protocol ID: `/opstack/req/payload_by_number/<chain-id>/0/`

- `/MessageName` is `/block_by_number/<chain-id>` where `<chain-id>` is set to the op-node L2 chain ID.
- `/SchemaVersion` is `/0`

Request format: `<num>`: a little-endian `uint64` - the block number to request.

Response format: `<response> = <res><version><payload>`

- `<res>` is a byte code describing the result.
  - `0` on success, `<version><payload>` should follow.
  - `1` if valid request, but unavailable payload.
  - `2` if invalid request
  - `3+` if other error
  - The `>= 128` range is reserved for future use.
- `<version>` is a little-endian `uint32`, identifying the type of `ExecutionPayload` (fork-specific)
- `<payload>` is an encoded block, read till stream EOF.

The input of `<response>` should be limited, as well as any generated decompressed output,
to avoid unexpected resource usage or zip-bomb type attacks.
A 10 MB limit is recommended, to ensure all blocks may be synced.
Implementations may opt for a different limit, since this sync method is optional.

`<version>` list:

- `0`: SSZ-encoded `ExecutionPayload`, with Snappy framing compression,
  matching the `ExecutionPayload` SSZ definition of the L1 Merge, L2 Bedrock and L2 Regolith versions.
- Other versions may be listed here with future network upgrades, such as the L1 Shanghai upgrade.

The request is by block-number, enabling parallel fetching of a chain across many peers.

A `res = 0` response should be verified to:

- Have a block-number matching the requested block number.
- Have a consistent `blockhash` w.r.t. the other block contents.
- Build towards a known canonical block.
  - This can be verified by checking if the parent-hash of a previous trusted canonical block matches
    that of the verified hash of the retrieved block.
  - For unsafe blocks this may be relaxed to verification against the parent-hash of any previously trusted block:
    - The gossip validation process limits the amount of blocks that may be trusted to sync towards.
    - The unsafe blocks should be queued for processing, the latest received L2 unsafe blocks should always
      override any previous chain, until the final L2 chain can be reproduced from L1 data.

A `res > 0` response code should not be accepted. The result code is helpful for debugging,
but the client should regard any error like any other unanswered request, as the responding peer cannot be trusted.

----

[libp2p]: https://libp2p.io/
[discv5]: https://github.com/ethereum/devp2p/blob/master/discv5/discv5.md
[discv5-random-nodes]: https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.12/p2p/discover#UDPv5.RandomNodes
[eth2-p2p]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/p2p-interface.md
[eth2-p2p-reqresp]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/p2p-interface.md#the-reqresp-domain
[eth2-p2p-altair-reqresp]: https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/p2p-interface.md#the-reqresp-domain
[libp2p-noise]: https://github.com/libp2p/specs/tree/master/noise
[multistream-select]: https://github.com/multiformats/multistream-select/
[mplex]: https://github.com/libp2p/specs/tree/master/mplex
[yamux]: https://github.com/hashicorp/yamux/blob/master/spec.md
[gossipsub]: https://github.com/libp2p/specs/blob/master/pubsub/gossipsub/gossipsub-v1.1.md
[signature-policy]: https://github.com/libp2p/specs/blob/master/pubsub/README.md#signature-policy-options
[snappy]: https://github.com/google/snappy
[l1-message-id]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/p2p-interface.md#topics-and-messages
[gossip-parameters]: https://github.com/libp2p/specs/blob/master/pubsub/gossipsub/gossipsub-v1.0.md#parameters
[extended-validator]: https://github.com/libp2p/specs/blob/master/pubsub/gossipsub/gossipsub-v1.1.md#extended-validators
