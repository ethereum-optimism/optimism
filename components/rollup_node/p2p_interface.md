# P2P Interface

This peer-to-peer (P2P) network service in the rollup node is optional,
but greatly improves the latency between the view of sequencers and the rest of the network,
as well as enabling faster historical sync to be bootstrapped, without relying on a single endpoint.

In summary, the P2P stack looks like:
- Discovery to find peers: [Discv5][discv5]
- Connections, peering, transport security, multi-plexing, gossip: [LibP2P][libp2p]

This document only specifies the composition and configuration of these network libraries.
These components have their own standards, implementations in Go/Rust/Java/Nim/JS/more,
and adopted several other blockchains, most notably the [L1 consensus layer (Eth2)][eth2-p2p].

## P2P configuration

### Identification

Nodes have a **separate** network- and consensus-identity.
The network identity is a `secp256k1` key, used for both discovery and active LibP2P connections.

Common representations of network identity:
- `PeerID`: a LibP2P specific ID derived from the pubkey (through protobuf encoding, typing and hashing)
- `NodeID`: a Discv5 specific ID derived from the pubkey (through hashing, used in the DHT)
- `Multi-address`: an unsigned address, containing: IP, TCP port, PeerID
- `ENR`: a signed record used for discovery, containing: IP, TCP port, UDP port, signature (pubkey can be derived) and L2 network identification. Generally encoded in base64.

### Discv5

#### Structure

The Ethereum Node Record (ENR) for an Optimism rollup node must contain the following values, identified by unique keys:

- An IPv4 address (`ip` field) and/or IPv6 address (`ip6` field).
- A TCP port (`tcp` field) representing the local libp2p listening port.
- A UDP port (`udp` field) representing the local discv5 listening port.
- An Optimism (`optimism` field) L2 network identifier

The `optimism` value is encoded as the concatenation of:
- chain ID (`varint`)
- fork ID (`varint`)

Note that DiscV5 is a shared DHT (Distributed Hash Table): the L1 consensus and execution nodes, as well as testnet nodes,
and even external IOT nodes, all communicate records in this large common DHT.
This makes it more difficult to censor the discovery of node records.

The discovery process in Optimism is a pipeline of node records:
1. Fill the table with `FINDNODES` if necessary (Performed by Discv5 library)
2. Pull additional records with searches to random Node IDs if necessary (e.g. iterate [`RandomNodes()`][discv5-random-nodes] in Go implementation)
3. Pull records from the DiscV5 module when looking for peers
4. Check if the record contains the `optimism` entry, verify it matches the chain ID and current or future fork number
5. If not already connected, and not recently disconnected or put on deny-list, attempt to dial.

### LibP2P

#### Transport

TCP transport. Additional transports are supported by LibP2P, but not required.

#### Dialing

Nodes should be publicly dialable, not rely on relay extensions, and able to dial both IPv4 and IPv6.

#### NAT

The listening endpoint must be publicly facing, but may be configured behind a NAT.
LibP2P has optional automation for this.

#### Peergating and pruning

TODO: max-peers, grace periods, pruning

#### Transport security

[Libp2p-noise][libp2p-noise], `XX` handshake, with the the `secp256k1` P2P identity, as popularized in Eth2.

#### Protocol negotiation

[Multistream-select 1.0][multistream-select] (`/multistream/1.0.0`) is an interactive protocol
used to negotiate sub-protocols supported in LibP2P peers. Multistream-select 2.0 may be used in the future.

#### Identify

LibP2P offers a minimal identification module to share client version and programming language.
This is optional and can be disabled for enhanced privacy.
It also includes the same protocol information, which can speed up initial connections.

#### Multiplexing

For async communication over different channels over the same connection, multiplexing is used.
[mplex][mplex] (`/mplex/6.7.0`) is required, and [yamux][yamux] (`/yamux/1.0.0`) is recommended but optional 

#### GossipSub

[GossipSub 1.1](gossipsub) (`/meshsub/1.1.0`, i.e. with peer-scoring extension) is a pubsub protocol for mesh-networks,
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
* If `message.data` has a valid snappy decompression, set `message-id` to the first 20 bytes of the `SHA256` hash of
  the concatenation of `MESSAGE_DOMAIN_VALID_SNAPPY` with the snappy decompressed message data,
  i.e. `SHA256(MESSAGE_DOMAIN_VALID_SNAPPY + snappy_decompress(message.data))[:20]`.
* Otherwise, set `message-id` to the first 20 bytes of the `SHA256` hash of
  the concatenation of `MESSAGE_DOMAIN_INVALID_SNAPPY` with the raw message data,
  i.e. `SHA256(MESSAGE_DOMAIN_INVALID_SNAPPY + message.data)[:20]`.

#### Heartbeat and parameters

GossipSub [parameters][gossip-parameters]:

- `D` (topic stable mesh target count): 8
- `D_low` (topic stable mesh low watermark): 6
- `D_high` (topic stable mesh high watermark): 12
- `D_lazy` (gossip target): 6
- `heartbeat_interval` (frequency of heartbeat, seconds): 1.0
- `fanout_ttl` (ttl for fanout maps for topics we are not subscribed to but have published to, seconds): 24
- `mcache_len` (number of windows to retain full messages in cache for `IWANT` responses): 6
- `mcache_gossip` (number of windows to gossip about): 3
- `seen_ttl` (number of heartbeat intervals to retain message IDs): 40

Notable differences from L1 consensus (Eth2):
- `seen_ttl` does not need to cover a full L1 epoch (6.4 minutes), but rather just a small window covering latest blocks
- `fanout_ttl` adjusted to lower than `seen_ttl`
- `heartbeat_interval` lowered for bandwidth saving, since there are less messages per second (no 600 attestations every second)


#### Topic configuration

Topics have string identifiers and are communicated with messages and subscriptions.
`/optimism/chain_id/hardfork_version/Name`

- `chain_id`: replace with decimal representation of chain ID
- `hardfork_version`: replace with decimal representation of hardfork
- `Name`: topic application-name

Note that the topic encoding depends on the topic, unlike L1, since there are less topics, and all are snappy-compressed.

## Gossip Topics

### `blocks`

The primary topic of the L2, to distribute blocks to other nodes faster than proxying through L1 would.

#### Block encoding

TODO: encode execution payload (SSZ or RLP), with sequencer identifier and signature.

TODO: signature type and verification (options: `secp256k1` like transactions, or `BLS12-381` with pubkeys on G1, like L1)

#### Block validation

To ensure malicious peers get scored based on application behavior the validation signals 
`ACCEPT` (valid), `IGNORE` (like inactivity) or `REJECT` (score penalties).

In order of operation:
- `[REJECT]` if the encoding or compression is not valid
- `[REJECT]` if the block timestamp is older than 20 seconds in the past (graceful boundary for worst-case propagation and clock skew)
- `[REJECT]` if the block timestamp is more than 5 seconds into the future (graceful boundary for clock skew)
- `[REJECT]` if the signature by the sequencer is not valid
- `[REJECT]` if more than 5 blocks have been seen with the same block height
- `[IGNORE]` if a conflicting block was already seen on L1. It may not be malicious due to racing between L1 confirmation and L2 propagation, but should be filtered out.

The block is signed by the corresponding sequencer, to filter malicious messages.
The sequencer model is singular but may change to multiple sequencers in the future.
A default sequencer pubkey is distributed with rollup nodes and should be configurable.

##### Block processing

A node may apply the block to their local engine ahead of L1 availability, if it ensures that:
- The application of the block is reversible, in case of a conflict with delayed L1 information
- The subsequent forkchoice-update ensures this block is recognized as "unsafe" (see [`engine_forkchoiceUpdatedV1`](./exec_engine.md#engine_forkchoiceupdatedv1))

##### Block topic scoring parameters

TODO: GossipSub per-topic scoring to fine-tune incentives for ideal propagation delay and bandwidth usage.

----

[consensus-layer]: ./consensus_layer.md
[libp2p]: https://libp2p.io/
[discv5]: https://github.com/ethereum/devp2p/blob/master/discv5/discv5.md
[discv5-random-nodes]: https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.12/p2p/discover#UDPv5.RandomNodes
[eth2-p2p]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/p2p-interface.md
[libp2p-noise]: https://github.com/libp2p/specs/tree/master/noise
[multistream-select]: https://github.com/multiformats/multistream-select/
[mplex]: https://github.com/libp2p/specs/tree/master/mplex
[yamux]: https://github.com/hashicorp/yamux/blob/master/spec.md
[gossipsub]: https://github.com/libp2p/specs/blob/master/pubsub/gossipsub/gossipsub-v1.1.md
[signature-policy]: https://github.com/libp2p/specs/blob/master/pubsub/README.md#signature-policy-options
[snappy]: https://github.com/google/snappy
[l1-message-id]: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/p2p-interface.md#topics-and-messages
[gossip-parameters]: https://github.com/libp2p/specs/blob/master/pubsub/gossipsub/gossipsub-v1.0.md#parameters

