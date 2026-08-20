//! Metrics for the Gossip stack.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Label key carrying the L2 chain ID, present on every metric emitted by the gossip stack.
    pub const CHAIN_ID_LABEL: &str = kona_macros::CHAIN_ID_LABEL;

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
    /// (`invalid_snappy_length`, covering both unreadable and over-`MAX_GOSSIP_SIZE` headers),
    /// envelope decode failures (`decode_error`), and unexpected topics (`unknown_topic`).
    /// Messages that decode but fail block validation are counted by
    /// [`Self::BLOCK_VALIDATION_FAILED`] instead; malformed frames caught earlier, in the
    /// gossipsub `message_id` function, are counted by [`Self::MESSAGE_ID_INVALID_SNAPPY`].
    pub const INVALID_MESSAGE: &str = "kona_node_gossip_invalid_message";

    /// Identifier for the counter that tracks inbound frames the gossipsub `message_id` function
    /// could not decompress within the gossip size bound.
    ///
    /// Recorded per receipt — before gossipsub de-duplication and before the block handler runs —
    /// so it counts on a larger denominator than [`Self::INVALID_MESSAGE`] and overlaps that
    /// counter's `invalid_snappy_length`/`decode_error` reasons for the same message. It is not a
    /// rejection on its own: the message-id function only assigns the message id.
    pub const MESSAGE_ID_INVALID_SNAPPY: &str = "kona_node_gossip_message_id_invalid_snappy";

    /// Initializes metrics for the Gossip stack.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init(chain_id: u64) {
        Self::describe();
        Self::zero(chain_id);
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
            "Number of execution payload envelopes gossiped out through the libp2p swarm"
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
    pub fn zero(chain_id: u64) {
        let chain_id = kona_macros::chain_id_label(chain_id);

        // RPC Calls
        for method in [
            "opp2p_self",
            "opp2p_peerCount",
            "opp2p_peers",
            "opp2p_peerStats",
            "opp2p_discoveryTable",
            "opp2p_blockPeer",
            "opp2p_listBlockedPeers",
            "opp2p_blockAddr",
            "opp2p_unblockAddr",
            "opp2p_listBlockedAddrs",
            "opp2p_blockSubnet",
            "opp2p_unblockSubnet",
            "opp2p_listBlockedSubnets",
            "opp2p_protectPeer",
            "opp2p_unprotectPeer",
            "opp2p_connectPeer",
            "opp2p_disconnectPeer",
        ] {
            kona_macros::set!(
                gauge,
                Self::RPC_CALLS,
                0,
                "method" => method,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Gossip Events
        for event in ["message", "subscribed", "unsubscribed", "slow_peer", "not_supported"] {
            kona_macros::set!(
                gauge,
                Self::GOSSIP_EVENT,
                0,
                "type" => event,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Unlabelled gauges: peer dials, unsafe blocks, peer counts.
        //
        // `BANNED_PEERS` is deliberately absent: its emit carries a `peer_id`, so there is no
        // finite set of series to pre-create.
        for metric in [Self::DIAL_PEER, Self::UNSAFE_BLOCK_PUBLISHED, Self::GOSSIP_PEER_COUNT] {
            kona_macros::set!(gauge, metric, 0, Self::CHAIN_ID_LABEL => chain_id.clone());
        }

        // Dial failures, by the reason recorded at each emit site in `driver.rs` and `gater.rs`.
        for reason in [
            "invalid_enr",
            "invalid_multiaddr",
            "already_connected",
            "already_dialing",
            "connection_error",
            "threshold_reached",
            "blocked_peer",
            "blocked_address",
            "blocked_subnet",
        ] {
            kona_macros::set!(
                gauge,
                Self::DIAL_PEER_ERROR,
                0,
                "type" => reason,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Connection
        for kind in ["connected", "outgoing_error", "incoming_error", "closed"] {
            kona_macros::set!(
                gauge,
                Self::GOSSIPSUB_CONNECTION,
                0,
                "type" => kind,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Gossipsub Events
        for event in [
            "subscribed",
            "unsubscribed",
            "gossipsub_not_supported",
            "slow_peer",
            "message_received",
        ] {
            kona_macros::set!(
                gauge,
                Self::GOSSIPSUB_EVENT,
                0,
                "type" => event,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Block validation metrics
        for metric in [Self::BLOCK_VALIDATION_TOTAL, Self::BLOCK_VALIDATION_SUCCESS] {
            kona_macros::set!(counter, metric, 0, Self::CHAIN_ID_LABEL => chain_id.clone());
        }

        // Block validation failures by reason
        for reason in [
            "timestamp_future",
            "timestamp_past",
            "invalid_hash",
            "invalid_signature",
            "invalid_signer",
            "too_many_blocks",
            "block_seen",
            "invalid_block",
            "parent_beacon_root",
            "blob_gas_used",
            "excess_blob_gas",
            "withdrawals_root",
        ] {
            kona_macros::set!(
                counter,
                Self::BLOCK_VALIDATION_FAILED,
                0,
                "reason" => reason,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Block versions
        for version in ["v1", "v2", "v3", "v4"] {
            kona_macros::set!(
                counter,
                Self::BLOCK_VERSION,
                0,
                "version" => version,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Messages rejected by the block handler before validation, by reason
        for reason in ["invalid_snappy_length", "decode_error", "unknown_topic"] {
            kona_macros::set!(
                counter,
                Self::INVALID_MESSAGE,
                0,
                "reason" => reason,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Malformed frames caught in the message-id function (per receipt, before dedup)
        kona_macros::set!(
            counter,
            Self::MESSAGE_ID_INVALID_SNAPPY,
            0,
            Self::CHAIN_ID_LABEL => chain_id
        );
    }
}
