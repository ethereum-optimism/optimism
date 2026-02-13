# Flashblocks P2P Extension

*This document is an extension to the original Flashblocks specification, modifying the flashblock propagation mechanism to use a peer-to-peer (P2P) network instead of WebSockets. It highlights the new P2P protocol and the changes in Rollup-Boost and builder interactions, aimed at simplifying distribution and improving fault tolerance in High Availability (HA) sequencer setups.*

**Table of Contents**

* [Abstract](#abstract)
* [Motivation](#motivation)
* [Specification](#specification)

  * [Terminology](#terminology)
  * [Data Structures](#data-structures)

    * [**`Authorization`**](#authorization)
    * [**`Authorized Message`**](#authorized-message)
    * [**`StartPublish`**](#startpublish)
    * [**`StopPublish`**](#stoppublish)
  * [Flashblocks P2P Protocol](#flashblocks-p2p-protocol)

    * [Protocol Overview](#protocol-overview)
    * [Message Types](#message-types)
    * [Authorization and Security](#authorization-and-security)
    * [Multi-Builder Coordination](#multi-builder-coordination)
  * [Rollup-Boost and Builder Communication](#rollup-boost-and-builder-communication)

# Abstract

This document introduces an enhancement to Flashblocks where the propagation of partial blocks (“flashblocks”) is done over an Ethereum P2P subprotocol instead of a WebSocket broadcast. By integrating flashblock distribution into the peer-to-peer network, we eliminate the need for a dedicated WebSocket proxy and enable more robust, decentralized propagation of flashblock data. Crucially, this P2P approach uses cryptographic authorization to ensure that only an **authorized** block builder (and its designated successors in an HA setup) can publish flashblocks, improving fault tolerance during sequencer failovers. The end result is a simpler and more resilient system for delivering rapid preconfirmation data to users, without altering the core OP Stack protocol.

# Motivation

The original Flashblocks design relied on a centralized broadcast (via Rollup-Boost and a WebSocket proxy) to propagate flashblocks to RPC providers. While effective, that design introduced operational complexity and potential single points of failure:

* **Operational Complexity:** Sequencer operators had to manage a WebSocket broadcasting service (e.g. Rollup-Boost’s WebSocket proxy) to fan-out flashblocks to providers. In multi-sequencer (HA) configurations, handing off this connection or migrating subscribers was cumbersome.
* **Failover Challenges:** In a High Availability sequencer setup, if the active sequencer failed the act of switching to a new sequencer/rollup-boost/builder combo would mean that already published flashblocks would not make it in the new block produced by the new builder. This breaks the promise that flashblocks makes to its consumers.
* **Scalability and Decentralization:** Relying on a single hub (the sequencer’s Rollup-Boost) to redistribute flashblocks could become a bottleneck. A P2P approach can naturally scale out to many peers and align with Ethereum’s existing propagation model for blocks and transactions.

**P2P Propagation** addresses these issues by leveraging a gossip network for flashblocks. In this model, any number of RPC provider nodes (or other interested parties) can connect to the flashblock P2P network to receive preconfirmation updates. Failover is handled gracefully through the RLPx protocol: if a new sequencer takes over, its builder is already aware of previously published flashblocks, and so it can build on top of what has already been promised to the network.

# Specification

## Terminology

We inherit all terminology from the original Flashblocks spec (Sequencer, Block Builder, Rollup-Boost, etc.), with a few new terms introduced:

* **Authorizer** – The entity that vouches for a block builder’s legitimacy to produce flashblocks. In practice, this is rollup-boost who signs an authorization for a given builder each block cycle.
* **Builder Public Key** – A cryptographic public key identifying a builder on the flashblocks P2P network. This is distinct from an Ethereum address; it’s used for signing/validating flashblock messages.
* **Flashblocks P2P Network** – The peer-to-peer overlay network (using Ethereum’s devp2p protocols) through which flashblock messages are gossiped. Participants include all builders and one or more subscribing nodes (e.g. RPC providers, possibly other sequencer nodes in standby).
* **Publisher** – The current active builder that is publishing flashblocks for the ongoing L2 block. In an HA setup, the role of publisher can transfer to a new builder if the sequencer fails over.

## Data Structures

The fundamental flashblock data structures (`FlashblocksPayloadV1`, `ExecutionPayloadFlashblockResultV1`, `ExecutionPayloadStaticV1`, and the various Metadata containers) remain unchanged. Flashblocks are still represented as a sequence of incremental payloads culminating in a full block.

To support P2P propagation and authorization, we introduce several new structures:

### **`Authorization`**

Represents a sequencer’s cryptographic authorization for a specific builder to produce a block with a given payload context. This is essentially a signed token from the sequencer (authorizer) that the builder includes with its flashblocks.

```rust
pub struct Authorization {
    pub payload_id: PayloadId,
    pub timestamp: u64,
    pub builder_vk: VerifyingKey,
    pub authorizer_sig: Signature,
}
```

* `payload_id`: The unique ID for this block’s payload (as provided by `engine_forkchoiceUpdated` in the OP Stack Engine API). All flashblocks for the block share this ID.
* `timestamp`: The timestamp associated with this payload
* `builder_vk`: The verifying key identifying the builder authorized to publish this block’s flashblocks. Peers will use this to verify the builder’s signatures on messages.
* `authorizer_sig`: A signature produced by the sequencer (authorizer) over the concatenation of `payload_id`, `timestamp`, and `builder_vk`. This proves that the sequencer has approved the given builder (and key) to act for this block. Only one authorizer key (controlled by the rollup-boost operator) is recognized by the network, and all peers are configured with its public key for verification.

### **`Authorized Message`**

Container for any flashblocks P2P message that requires authorization. It bundles a payload (one of the message types defined below) with the authorization and a builder’s signature.

```rust
pub struct Authorized {
    pub msg: AuthorizedMsg,
    pub authorization: Authorization,
    pub actor_sig: Signature,
}
```

```rust
pub enum AuthorizedMsg {
    FlashblocksPayloadV1(FlashblocksPayloadV1) = 0x00,
    StartPublish(StartPublish) = 0x01,
    StopPublish(StopPublish) = 0x02,
}
```

* `authorization`: The Authorization object, as described above.
* `msg`: The message content. This is a tagged union that can be one of:

  * A **Flashblock Payload** – Contains a `FlashblocksPayloadV1` (partial block delta), see below.
  * A **StartPublish** signal – Indicates the builder is starting to publish a new block (detailed in [StartPublish](#startpublish)).
  * A **StopPublish** signal – Indicates the builder is stopping publication (detailed in [StopPublish](#stoppublish)).

* `actor_sig`: The builder’s signature over the combination of the `msg` and the `authorization`. This attests that the message indeed comes from the holder of the `builder_sk` in the Authorization, and that it hasn’t been tampered with in transit.

Every P2P message in the Flashblocks protocol is sent as an `AuthorizedMessage`. This double-signature scheme (authorizer + builder) provides two layers of security:

1. Only a builder with a valid Authorization (signed by the sequencer) can get its messages accepted by peers.
2. Only the genuine builder (holding the private key corresponding to `builder_sk`) can produce a valid `builder_signature` on the message content.

### **`StartPublish`**

A small message indicating the intention to begin publishing flashblocks for a new L2 block.

```rust
pub struct StartPublish;
```

The `StartPublish` message is always sent wrapped in an `AuthorizedMessage` (with the appropriate authorization and signatures). It serves as an announcement to the network that *“Builder X is about to start publishing”*

### **`StopPublish`**

An authorized message indicating that the builder will no longer publish any flashblocks

```rust
pub struct StopPublish;
```

**Note:** A builder will typically send a `StopPublish` when it receives a `ForkChoiceUpdated` without an accompanying `Authorization` from rollup-boost or upon handing off flashblock production to a new builder.

## Flashblocks P2P Protocol

### Protocol Overview

Flashblocks P2P communication is implemented as a custom Ethereum subprotocol. Specifically, it defines a new devp2p capability:

* **Protocol Name:** `flblk` (flashblocks)
* **Version:** `1`

Nodes that support flashblocks will advertise this capability when establishing devp2p connections. Once connected, they can exchange flashblock messages as defined in this spec.

All flashblock messages are encoded in a compact binary format (analogous to Ethereum block gossip). Each message begins with a one-byte type discriminator, followed by the serialized content. The primary message type is an `AuthorizedMessage` (discriminator `0x00`), which, as described, contains a nested payload type.

**Key design features of the P2P protocol:**

* **Multipeer Gossip:** A builder’s flashblock is forwarded to all connected peers, who in turn may forward it to their peers, etc., ensuring the payload reaches all participants without needing a single central broadcaster. The protocol includes basic duplicate suppression so that flashblocks aren’t endlessly propagated in loops.
* **Real-time Coordination:** Using `StartPublish` and `StopPublish` signals, multiple potential publishers (builders) can coordinate access to the network. This prevents conflicts where two builders might try to publish simultaneously, and allows a smooth handoff in failover scenarios (detailed below).

### Message Types

Within the `AuthorizedMsg` union, we define the following variants and their semantics:

* **Flashblock Payload Message:** Carries a `FlashblocksPayloadV1` (as defined in the original spec) for a specific partial block. This includes the incremental transactions, updated state root, receipts root, logs bloom, etc., up through that flashblock. Peers receiving this message will apply the included state updates to their preconfirmation cache. Each Flashblock message has an `index` (the flashblock sequence number) and may include the `base` section if it’s the first flashblock (index 0) for that block.
* **StartPublish Message:** Announces the start of a new publishers flashblock sequence. Peers use this to note which builder is now active for a given L2 block number, possibly resetting any previous state or halting their own publishing.
* **StopPublish Message:** Indicates the end of the flashblock sequence for the current publisher. After this message, no further flashblocks from that publisher should arrive. Inactive or waiting publishers use this as a cue that they may now take over for subsequent flashblocks.

All these are encapsulated in `AuthorizedMsg` with the requisite signatures.

### Authorization and Security

The P2P protocol introduces a trust model wherein peers accept flashblocks only from an **authorized builder**. The security measures include:

* **Authorizer Signature Verification:** Upon receiving any `AuthorizedMessage`, a peer will first verify the `authorizer_sig` in the `Authorization` against the known authorizer public key. This confirms that rollup-boost has indeed permitted the stated builder to produce the block with the given `payload_id` and timestamp. If this signature is missing or invalid, the message is discarded as untrusted.
* **Builder Signature Verification:** Next, the peer verifies the `builder_signature` on the message content using the `builder_vk` provided in the Authorization. This ensures the message was genuinely produced by the authorized builder and not altered. If this check fails, the message is rejected.
* **Payload Consistency Checks:** Peers also check that the fields in the message are self-consistent and match expectations:

  * The `payload_id` in the Authorization must match the `FlashblocksPayloadV1.payload_id` (for flashblock messages). Each builder’s flashblock messages carry the same payload\_id that was authorized, ensuring they all belong to the same block-building session.
  * **Freshness:** The `timestamp` in Authorization helps guard against replay of old messages. If a flashblock or StartPublish arrives with a significantly older timestamp (or for an already completed block), peers will ignore it and decrement the sender's reputation.

These measures ensure that **only** the rollup-boost sanctioned builder’s data is propagated and that it’s cryptographically sound. Unauthorized parties cannot inject false flashblocks or tamper with content without detection. This design also allows dynamic builder changes: as long as the sequencer signs a new Authorization, the peers will accept the new builder’s messages even if they have never seen that builder before, because trust is transitive from the authorizers’s key.

### Multi-Builder Coordination

A major benefit of the P2P approach is the ability to coordinate multiple builders in an HA (High Availability) setting. The `StartPublish` and `StopPublish` messages, in conjunction with a small amount of logic in Rollup-Boost and the network, handle the arbitration:

* **Single Publisher Rule:** The network expects at most one builder to be actively publishing flashblocks for a given L2 block number at any time. If two different builders both attempt to publish for the same block, the conflict must be resolved to maintain a consistent preconfirmation state.
* **Announcing Intent – `StartPublish`:** When Rollup-Boost (sequencer) initiates a new block with an external builder, it immediately broadcasts a `StartPublish` message (as an AuthorizedMessage) from that builder. This tells all peers: “Builder X is about to start publishing” If any other builder was thinking of building block N (perhaps there was a recent failover), it will see this and **stand down**.
* **Graceful Yield – reacting to `StartPublish`:** If a builder is currently publishing and receives a `StartPublish` from a *different* builder for the same or next block, it means a failover or override is happening. The expected behavior is that the current publisher will cease publishing (and issue a `StopPublish`). The protocol is designed such that the honest builder who is not supposed to publish will yield to the authorized one. The reference implementation will automatically send a `StopPublish` if it is publishing and learns that another builder has taken over authority for the block. The new builder will wait until it receives the `StopPublish` before continuing.
* **Completion – `StopPublish`:** When a builder receives the next FCU _without_ an accompanying `Authorization`, it will send out a `StopPublish`. This removes the builder from the “active publisher” role in the eyes of the network. If there was another builder in waiting (perhaps one that had attempted to start earlier but was told to wait), that waiting builder will now see that the coast is clear. 
* **Timeouts and Fallback:** There is an implicit timeout in the coordination. If a builder is in a waiting state after announcing `StartPublish` but for some reason the previous publisher fails to produce a `StopPublish` (for example, if it crashed mid-block), other participants will not wait indefinitely. In our design, if a new block number is reached and the previous publisher hasn’t stopped we assume the previous builder is incapacitated and proceed with the new publisher. 

This coordination ensures that in an HA setup with multiple sequencer instances and multiple builders, **preconfirmation data remains consistent**: only one set of flashblocks is ever in flight for a given block. If a sequencer failover occurs, the worst-case scenario (which occurs only during a very rare race condition) is a single block publication gap or discontinuity at a block boundary. In the far more likely case, there will be exactly no flashblock disruption. The next publisher will simply start where the last publisher left off, even if that is mid block.

## Rollup-Boost and Builder Communication

In the P2P-enhanced design, Rollup-Boost’s interaction with the external block builder is slightly adjusted:

* **Authorization Delivery:** When the sequencer (op-node) triggers a new block proposal via `engine_forkchoiceUpdated` (with payload attributes), Rollup-Boost creates an `Authorization` for the chosen builder. This requires that Rollup-Boost knows the builder’s public key in advance. In practice, the builder can be configured or registered with Rollup-Boost, providing its long-term public key. Rollup-Boost uses its **authorizer private key** (associated with the L2 chain or sequencer) to sign the authorization (covering payload\_id, timestamp, builder’s key).
* **Forkchoice Updated Forwarding:** Rollup-Boost forwards the fork choice update to the builder as usual (so the builder can start building the block). In this modified protocol, the fork choice update (or a parallel communication) includes the newly created `Authorization`. For example, a custom field or side-channel could convey the authorizer’s signature to the builder. **(Implementation-wise, this might be an extension of the Engine API or an internal call – the key point is the builder receives the Authorization token before it begins sending flashblocks.)**
* **StartPublish Broadcast:** If the builder was not previously publishing, then immediately after receiving the authorization it will emit a `StartPublish` message over the P2P network. This tells all listening nodes that the authorized builder will begin flashblock publication.
* **Streaming Flashblocks:** The builder executes transactions and produces flashblocks incrementally just as described in the original spec’s Flashblock Construction Process. However, instead of returning these payloads to Rollup-Boost, the builder now signs each flashblock with its key and directly broadcasts an Authorized Flashblock message to the P2P network. 
* **No Inline Validation by Sequencer:** In the original design, Rollup-Boost would validate each flashblock against the local execution engine before propagating it. In the P2P model, this is not done synchronously for each flashblock (it would negate some latency benefits). Instead, trust is managed via the Authorization. The sequencer trusts its chosen builder to only send valid blocks (and will ultimately verify the final block when `engine_getPayload` is called). Peers trust the flashblocks because they trust the Rollup-Boost’s signature. 

In summary, Rollup-Boost’s role shifts from being a **middleman for data** to being a **controller and coordinator**. It authorizes the builder and informs the network about which builder is active, but it doesn’t need to ferry every flashblock through itself. This streamlines the path from builder to RPC providers.

