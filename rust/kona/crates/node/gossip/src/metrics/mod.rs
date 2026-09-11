//! Metrics for the Gossip stack.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the gauge that tracks gossip events.
    pub const GOSSIP_EVENT: &str = "kona_node_gossip_events";

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
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in [`kona_gossip`][crate].
    #[cfg(feature = "metrics")]
    pub fn describe() {
        metrics::describe_gauge!(Self::RPC_CALLS, "Calls made to the Gossip RPC module");
        metrics::describe_gauge!(Self::GOSSIP_EVENT, "Events received from the libp2p Swarm");
        metrics::describe_gauge!(
            Self::DIAL_PEER_ERROR,
            "Number of failed peer dials by the libp2p Swarm, by reason"
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
    pub fn zero() {
        // The emits live in `kona-rpc`: `opp2p_*` in `p2p.rs`, `admin_postUnsafePayload` in
        // `admin.rs`.
        for method in [
            "opp2p_self",
            "opp2p_peerCount",
            "opp2p_peers",
            "opp2p_peerStats",
            "opp2p_discoveryTable",
            "opp2p_blockPeer",
            "opp2p_unblockPeer",
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
            "admin_postUnsafePayload",
        ] {
            kona_macros::set!(gauge, Self::RPC_CALLS, "method", method, 0);
        }

        // The other three types also carry a `topic`, and for `subscribed`/`unsubscribed` that
        // topic comes from remote peers, so their value set is not knowable here.
        kona_macros::set!(gauge, Self::GOSSIP_EVENT, "type", "slow_peer", 0);
        kona_macros::set!(gauge, Self::GOSSIP_EVENT, "type", "not_supported", 0);

        // Peer dials
        kona_macros::set!(gauge, Self::DIAL_PEER, 0);

        // The reasons emitted by `driver.rs` and `gater.rs`.
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
            kona_macros::set!(gauge, Self::DIAL_PEER_ERROR, "type", reason, 0);
        }

        // Unsafe Blocks
        kona_macros::set!(gauge, Self::UNSAFE_BLOCK_PUBLISHED, 0);

        // Peer Counts
        kona_macros::set!(gauge, Self::GOSSIP_PEER_COUNT, 0);

        // Connection
        for event in ["connected", "outgoing_error", "incoming_error", "closed"] {
            kona_macros::set!(gauge, Self::GOSSIPSUB_CONNECTION, "type", event, 0);
        }

        // `BANNED_PEERS` is absent: its emit carries `peer_id`, so there is no finite series set.

        // Block validation metrics
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_TOTAL, 0);
        kona_macros::set!(counter, Self::BLOCK_VALIDATION_SUCCESS, 0);

        // The reasons come from the `match` in `BlockValidator::block_valid`.
        for reason in [
            "timestamp_future",
            "timestamp_past",
            "invalid_hash",
            "invalid_signature",
            "invalid_signer",
            "too_many_blocks",
            "block_seen",
            "invalid_block",
            "blob_gas_used",
            "excess_blob_gas",
            "withdrawals_root",
        ] {
            kona_macros::set!(counter, Self::BLOCK_VALIDATION_FAILED, "reason", reason, 0);
        }

        // Block versions
        for version in ["v1", "v2", "v3", "v4"] {
            kona_macros::set!(counter, Self::BLOCK_VERSION, "version", version, 0);
        }

        // Messages rejected by the block handler before validation, by reason
        for reason in ["invalid_snappy_length", "decode_error", "unknown_topic"] {
            kona_macros::set!(counter, Self::INVALID_MESSAGE, "reason", reason, 0);
        }

        // Malformed frames caught in the message-id function (per receipt, before dedup)
        kona_macros::set!(counter, Self::MESSAGE_ID_INVALID_SNAPPY, 0);
    }
}

#[cfg(all(test, feature = "metrics"))]
mod tests {
    use super::Metrics;
    use metrics_util::debugging::DebuggingRecorder;
    use std::collections::BTreeSet;

    /// The `label` values that `zero()` pre-creates for `metric`, or `None` for a series with no
    /// such label.
    fn zeroed(metric: &str, label: &str) -> BTreeSet<Option<String>> {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, Metrics::zero);

        snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .map(|(ckey, ..)| ckey)
            .filter(|ckey| ckey.key().name() == metric)
            .map(|ckey| {
                ckey.key().labels().find(|l| l.key() == label).map(|l| l.value().to_string())
            })
            .collect()
    }

    fn expect(metric: &str, label: &str, values: &[&str]) {
        let expected: BTreeSet<_> = values.iter().map(|v| Some(v.to_string())).collect();
        assert_eq!(zeroed(metric, label), expected, "{metric} pre-created the wrong {label} set");
    }

    #[test]
    fn dial_peer_errors_are_pre_created_per_reason() {
        // The nine reasons emitted by `driver.rs` and `gater.rs`.
        expect(
            Metrics::DIAL_PEER_ERROR,
            "type",
            &[
                "invalid_enr",
                "invalid_multiaddr",
                "already_connected",
                "already_dialing",
                "connection_error",
                "threshold_reached",
                "blocked_peer",
                "blocked_address",
                "blocked_subnet",
            ],
        );
    }

    #[test]
    fn gossip_events_pre_create_only_the_topic_free_types() {
        expect(Metrics::GOSSIP_EVENT, "type", &["slow_peer", "not_supported"]);
    }

    #[test]
    fn block_validation_failures_omit_the_unreachable_reason() {
        // `BlockInvalidError` has no variant that yields `parent_beacon_root`.
        assert!(
            !zeroed(Metrics::BLOCK_VALIDATION_FAILED, "reason")
                .contains(&Some("parent_beacon_root".to_string()))
        );
    }

    #[test]
    fn banned_peers_is_not_pre_created() {
        assert!(zeroed(Metrics::BANNED_PEERS, "peer_id").is_empty());
    }

    #[test]
    fn no_series_is_pre_created_without_an_emit() {
        assert!(zeroed("kona_node_gossipsub_events", "type").is_empty());
    }
}
