//! Metrics for the Gossip stack.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the gauge that tracks gossip events.
    pub const GOSSIP_EVENT: &str = "kona_node_gossip_events";

    /// Identifier for the gauge that tracks libp2p gossipsub events.
    pub const GOSSIPSUB_EVENT: &str = "kona_node_gossipsub_events";

    /// Identifier for the gauge that tracks libp2p gossipsub connections.
    pub const GOSSIPSUB_CONNECTION: &str = "kona_node_gossipsub_connection";

    /// Identifier for the gauge that tracks unsafe blocks published.
    pub const UNSAFE_BLOCK_PUBLISHED: &str = "kona_node_unsafe_block_published";

    /// Identifier for the gauge that tracks the number of connected peers.
    pub const GOSSIP_PEER_COUNT: &str = "kona_node_swarm_peer_count";

    /// Identifier for the gauge that tracks the number of dialed peers.
    pub const DIAL_PEER: &str = "kona_node_dial_peer";

    /// Identifier for the gauge that tracks the number of errors when dialing peers.
    pub const DIAL_PEER_ERROR: &str = "kona_node_dial_peer_error";

    /// Identifier for the gauge that tracks RPC calls.
    pub const RPC_CALLS: &str = "kona_node_rpc_calls";

    /// Identifier for a gauge that tracks the number of banned peers.
    pub const BANNED_PEERS: &str = "kona_node_banned_peers";

    /// Identifier for a histogram that tracks peer scores.
    pub const PEER_SCORES: &str = "kona_node_peer_scores";

    /// Identifier for the gauge that tracks the duration of peer connections in seconds.
    pub const GOSSIP_PEER_CONNECTION_DURATION_SECONDS: &str =
        "kona_node_gossip_peer_connection_duration_seconds";

    /// Identifier for the counter that tracks total block validation attempts.
    pub const BLOCK_VALIDATION_TOTAL: &str = "kona_node_block_validation_total";

    /// Identifier for the counter that tracks successful block validations.
    pub const BLOCK_VALIDATION_SUCCESS: &str = "kona_node_block_validation_success";

    /// Identifier for the counter that tracks failed block validations by reason.
    pub const BLOCK_VALIDATION_FAILED: &str = "kona_node_block_validation_failed";

    /// Identifier for the histogram that tracks block validation duration in seconds.
    pub const BLOCK_VALIDATION_DURATION_SECONDS: &str =
        "kona_node_block_validation_duration_seconds";

    /// Identifier for the counter that tracks block version distribution.
    pub const BLOCK_VERSION: &str = "kona_node_block_version";

    /// Identifier for the counter that tracks messages the block handler rejected before block
    /// validation, by reason.
    ///
    /// The reasons are mutually exclusive per message: the handler's pre-decode snappy bound
    /// (`oversized_snappy`), envelope decode failures (`decode_error`), and unexpected topics
    /// (`unknown_topic`). Messages that decode but fail block validation are counted by
    /// [`Self::BLOCK_VALIDATION_FAILED`] instead; malformed frames caught earlier, in the
    /// gossipsub `message_id` function, are counted by [`Self::MESSAGE_ID_INVALID_SNAPPY`].
    pub const INVALID_MESSAGE: &str = "kona_node_gossip_invalid_message";

    /// Identifier for the counter that tracks inbound frames the gossipsub `message_id` function
    /// could not decompress within the gossip size bound.
    ///
    /// Recorded per receipt — before gossipsub de-duplication and before the block handler runs —
    /// so it counts on a larger denominator than [`Self::INVALID_MESSAGE`] and overlaps that
    /// counter's `oversized_snappy`/`decode_error` reasons for the same message. It is not a
    /// rejection on its own: the message-id function only assigns the message id.
    pub const MESSAGE_ID_INVALID_SNAPPY: &str = "kona_node_gossip_message_id_invalid_snappy";

    /// Initializes metrics for the Gossip stack.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in [`kona_gossip`][crate].
    #[cfg(feature = "metrics")]
    pub fn describe() {
        metrics::describe_gauge!(Self::RPC_CALLS, "Calls made to the Gossip RPC module");
        metrics::describe_gauge!(
            Self::GOSSIPSUB_EVENT,
            "Events received by the libp2p gossipsub Swarm"
        );
        metrics::describe_gauge!(Self::DIAL_PEER, "Number of peers dialed by the libp2p Swarm");
        metrics::describe_gauge!(
            Self::UNSAFE_BLOCK_PUBLISHED,
            "Number of OpNetworkPayloadEnvelope gossipped out through the libp2p Swarm"
        );
        metrics::describe_gauge!(
            Self::GOSSIP_PEER_COUNT,
            "Number of peers connected to the libp2p gossip Swarm"
        );
        metrics::describe_gauge!(
            Self::GOSSIPSUB_CONNECTION,
            "Connections made to the libp2p Swarm"
        );
        metrics::describe_gauge!(
            Self::BANNED_PEERS,
            "Number of peers banned by kona's gossip stack"
        );
        metrics::describe_histogram!(
            Self::PEER_SCORES,
            "Observations of peer scores in the gossipsub mesh"
        );
        metrics::describe_histogram!(
            Self::GOSSIP_PEER_CONNECTION_DURATION_SECONDS,
            "Duration of peer connections in seconds"
        );
        metrics::describe_counter!(
            Self::BLOCK_VALIDATION_TOTAL,
            "Total number of block validation attempts"
        );
        metrics::describe_counter!(
            Self::BLOCK_VALIDATION_SUCCESS,
            "Number of successful block validations"
        );
        metrics::describe_counter!(
            Self::BLOCK_VALIDATION_FAILED,
            "Number of failed block validations by reason"
        );
        metrics::describe_histogram!(
            Self::BLOCK_VALIDATION_DURATION_SECONDS,
            "Duration of block validation in seconds"
        );
        metrics::describe_counter!(Self::BLOCK_VERSION, "Distribution of block versions");
        metrics::describe_counter!(
            Self::INVALID_MESSAGE,
            "Number of inbound gossip messages the block handler rejected before block validation, by reason"
        );
        metrics::describe_counter!(
            Self::MESSAGE_ID_INVALID_SNAPPY,
            "Number of inbound gossip frames the message-id function could not decompress within the size bound"
        );
    }

    /// Initializes metrics to `0` so they can be queried immediately by consumers of prometheus
    /// metrics.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        // RPC Calls
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_self", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_peerCount", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_peers", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_peerStats", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_discoveryTable", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_blockPeer", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_listBlockedPeers", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_blockAddr", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_unblockAddr", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_listBlockedAddrs", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_blockSubnet", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_unblockSubnet", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_listBlockedSubnets", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_protectPeer", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_unprotectPeer", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_connectPeer", 0);
        kona_macros::set!(gauge, Self::RPC_CALLS, "method", "opp2p_disconnectPeer", 0);

        // Gossip Events
        kona_macros::set!(gauge, Self::GOSSIP_EVENT, "type", "message", 0);
        kona_macros::set!(gauge, Self::GOSSIP_EVENT, "type", "subscribed", 0);
        kona_macros::set!(gauge, Self::GOSSIP_EVENT, "type", "unsubscribed", 0);
        kona_macros::set!(gauge, Self::GOSSIP_EVENT, "type", "slow_peer", 0);
        kona_macros::set!(gauge, Self::GOSSIP_EVENT, "type", "not_supported", 0);

        // Peer dials
        kona_macros::set!(gauge, Self::DIAL_PEER, 0);
        kona_macros::set!(gauge, Self::DIAL_PEER_ERROR, 0);

        // Unsafe Blocks
        kona_macros::set!(gauge, Self::UNSAFE_BLOCK_PUBLISHED, 0);

        // Peer Counts
        kona_macros::set!(gauge, Self::GOSSIP_PEER_COUNT, 0);

        // Connection
        kona_macros::set!(gauge, Self::GOSSIPSUB_CONNECTION, "type", "connected", 0);
        kona_macros::set!(gauge, Self::GOSSIPSUB_CONNECTION, "type", "outgoing_error", 0);
        kona_macros::set!(gauge, Self::GOSSIPSUB_CONNECTION, "type", "incoming_error", 0);
        kona_macros::set!(gauge, Self::GOSSIPSUB_CONNECTION, "type", "closed", 0);

        // Gossipsub Events
        kona_macros::set!(gauge, Self::GOSSIPSUB_EVENT, "type", "subscribed", 0);
        kona_macros::set!(gauge, Self::GOSSIPSUB_EVENT, "type", "unsubscribed", 0);
        kona_macros::set!(gauge, Self::GOSSIPSUB_EVENT, "type", "gossipsub_not_supported", 0);
        kona_macros::set!(gauge, Self::GOSSIPSUB_EVENT, "type", "slow_peer", 0);
        kona_macros::set!(gauge, Self::GOSSIPSUB_EVENT, "type", "message_received", 0);

        // Banned Peers
        kona_macros::set!(gauge, Self::BANNED_PEERS, 0);

        // Block validation metrics
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_TOTAL, 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_SUCCESS, 0);

        // Block validation failures by reason
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "timestamp_future", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "timestamp_past", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "invalid_hash", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "invalid_signature", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "invalid_signer", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "too_many_blocks", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "block_seen", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "invalid_block", 0);
        kona_macros::set!(
            counter,
            Self::BLOCK_VALIDATION_FAILED,
            "reason",
            "parent_beacon_root",
            0
        );
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "blob_gas_used", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "excess_blob_gas", 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", "withdrawals_root", 0);

        // Block versions
        kona_macros::set!(counter, Self::BLOCK_VERSION, "version", "v1", 0);
        kona_macros::set!(counter, Self::BLOCK_VERSION, "version", "v2", 0);
        kona_macros::set!(counter, Self::BLOCK_VERSION, "version", "v3", 0);
        kona_macros::set!(counter, Self::BLOCK_VERSION, "version", "v4", 0);

        // Messages rejected by the block handler before validation, by reason
        kona_macros::set!(counter, Self::INVALID_MESSAGE, "reason", "oversized_snappy", 0);
        kona_macros::set!(counter, Self::INVALID_MESSAGE, "reason", "decode_error", 0);
        kona_macros::set!(counter, Self::INVALID_MESSAGE, "reason", "unknown_topic", 0);

        // Malformed frames caught in the message-id function (per receipt, before dedup)
        kona_macros::set!(counter, Self::MESSAGE_ID_INVALID_SNAPPY, 0);
    }
}
