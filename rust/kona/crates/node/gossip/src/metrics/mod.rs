//! Metrics for the Gossip stack.

use libp2p::gossipsub::TopicHash;

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the counter that tracks libp2p gossipsub events, by `type`.
    ///
    /// `message`, `subscribed` and `unsubscribed` also carry the `topic` the event is for. op-node
    /// counts them in `op_node_default_p2p_gossip_events_total`.
    pub const GOSSIP_EVENT: &str = "kona_node_gossip_events";

    /// Identifier for the counter that tracks libp2p gossipsub connections.
    pub const GOSSIPSUB_CONNECTION: &str = "kona_node_gossipsub_connection";

    /// Identifier for the counter that tracks unsafe blocks published.
    pub const UNSAFE_BLOCK_PUBLISHED: &str = "kona_node_unsafe_block_published";

    /// Identifier for the gauge that tracks the number of connected peers.
    pub const GOSSIP_PEER_COUNT: &str = "kona_node_swarm_peer_count";

    /// Identifier for the counter that tracks the number of dialed peers.
    pub const DIAL_PEER: &str = "kona_node_dial_peer";

    /// Identifier for the counter that tracks the number of errors when dialing peers.
    pub const DIAL_PEER_ERROR: &str = "kona_node_dial_peer_error";

    /// Identifier for the counter that tracks how many times a peer was banned for its score.
    ///
    /// Unlabelled: the banned peer and its score are in the ban's log line.
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

    /// Identifier for the counter of inbound connections the gate refused, by reason:
    /// `blocked_peer`, `banned_peer`, `blocked_address` or `blocked_subnet`. op-node counts its
    /// gater's decisions in `op_node_default_p2p_accepts`, by whether it allowed them.
    pub const INBOUND_CONNECTIONS_REFUSED: &str = "kona_node_gossip_inbound_connections_refused";

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
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in [`kona_gossip`][crate].
    pub fn describe() {
        metrics::describe_counter!(
            Self::GOSSIP_EVENT,
            "Events received by the libp2p gossipsub Swarm, by type"
        );
        metrics::describe_counter!(Self::DIAL_PEER, "Number of peers dialed by the libp2p Swarm");
        metrics::describe_counter!(Self::DIAL_PEER_ERROR, "Number of errors when dialing peers");
        metrics::describe_counter!(
            Self::UNSAFE_BLOCK_PUBLISHED,
            "Number of execution payload envelopes gossiped out through the libp2p swarm"
        );
        metrics::describe_gauge!(
            Self::GOSSIP_PEER_COUNT,
            "Number of peers connected to the libp2p gossip Swarm"
        );
        metrics::describe_counter!(
            Self::GOSSIPSUB_CONNECTION,
            "Connections made to the libp2p Swarm"
        );
        metrics::describe_counter!(
            Self::BANNED_PEERS,
            "Number of times kona's gossip stack banned a peer for its score"
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
            Self::INBOUND_CONNECTIONS_REFUSED,
            "Number of inbound connections the gate refused, by reason"
        );
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
    pub fn zero() {
        // Gossip events without a topic. Those with one are zeroed by `zero_topics`, once the
        // topics are known.
        metrics::counter!(Self::GOSSIP_EVENT, "type" => "slow_peer").absolute(0);
        metrics::counter!(Self::GOSSIP_EVENT, "type" => "not_supported").absolute(0);

        // Peer dials
        metrics::counter!(Self::DIAL_PEER).absolute(0);

        // Dial errors, by the `type` they are recorded with
        for kind in [
            // Recorded by the driver
            "invalid_enr",
            "invalid_multiaddr",
            "already_connected",
            "connection_error",
            // Recorded by the connection gater, see `DialError::reason`
            "already_dialing",
            "threshold_reached",
            "blocked_peer",
            "banned_peer",
            "invalid_ip_address",
            "blocked_address",
            "blocked_subnet",
        ] {
            metrics::counter!(Self::DIAL_PEER_ERROR, "type" => kind).absolute(0);
        }

        // Unsafe Blocks
        metrics::counter!(Self::UNSAFE_BLOCK_PUBLISHED).absolute(0);

        // Peer Counts
        metrics::gauge!(Self::GOSSIP_PEER_COUNT).set(0);

        // Connection
        metrics::counter!(Self::GOSSIPSUB_CONNECTION, "type" => "connected").absolute(0);
        metrics::counter!(Self::GOSSIPSUB_CONNECTION, "type" => "outgoing_error").absolute(0);
        metrics::counter!(Self::GOSSIPSUB_CONNECTION, "type" => "incoming_error").absolute(0);
        metrics::counter!(Self::GOSSIPSUB_CONNECTION, "type" => "closed").absolute(0);

        // Peer bans
        metrics::counter!(Self::BANNED_PEERS).absolute(0);

        // Block validation metrics
        metrics::counter!(Self::BLOCK_VALIDATION_TOTAL).absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_SUCCESS).absolute(0);

        // Block validation failures by reason
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "timestamp_future")
            .absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "timestamp_past").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "invalid_hash").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "invalid_signature")
            .absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "invalid_signer").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "too_many_blocks").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "block_seen").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "invalid_block").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "parent_beacon_root")
            .absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "blob_gas_used").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "excess_blob_gas").absolute(0);
        metrics::counter!(Self::BLOCK_VALIDATION_FAILED, "reason" => "withdrawals_root")
            .absolute(0);

        // Block versions
        metrics::counter!(Self::BLOCK_VERSION, "version" => "v1").absolute(0);
        metrics::counter!(Self::BLOCK_VERSION, "version" => "v2").absolute(0);
        metrics::counter!(Self::BLOCK_VERSION, "version" => "v3").absolute(0);
        metrics::counter!(Self::BLOCK_VERSION, "version" => "v4").absolute(0);

        // Messages rejected by the block handler before validation, by reason
        metrics::counter!(Self::INVALID_MESSAGE, "reason" => "invalid_snappy_length").absolute(0);
        metrics::counter!(Self::INVALID_MESSAGE, "reason" => "decode_error").absolute(0);
        metrics::counter!(Self::INVALID_MESSAGE, "reason" => "unknown_topic").absolute(0);

        // Inbound connections the gate refused, by reason
        for reason in ["blocked_peer", "banned_peer", "blocked_address", "blocked_subnet"] {
            metrics::counter!(Self::INBOUND_CONNECTIONS_REFUSED, "reason" => reason).absolute(0);
        }

        // Malformed frames caught in the message-id function (per receipt, before dedup)
        metrics::counter!(Self::MESSAGE_ID_INVALID_SNAPPY).absolute(0);
    }

    /// Initializes the gossip events recorded with a `topic` label to `0`, for each of `topics`.
    ///
    /// The subscription filter only lets through the topics the node subscribes to, so these are
    /// all the label sets that get recorded.
    pub fn zero_topics(topics: &[TopicHash]) {
        for topic in topics {
            for kind in ["message", "subscribed", "unsubscribed"] {
                metrics::counter!(Self::GOSSIP_EVENT, "type" => kind, "topic" => topic.to_string())
                    .absolute(0);
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use metrics_util::debugging::{DebugValue, DebuggingRecorder};

    /// Runs `zero` and `zero_topics` for `topic`, and returns the series of `name` they create,
    /// each as its sorted `key=value` labels, after checking it is a counter at 0.
    fn zeroed_series(name: &str, topic: &TopicHash) -> Vec<String> {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            Metrics::zero();
            Metrics::zero_topics(std::slice::from_ref(topic));
        });
        let mut series: Vec<_> = snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .filter(|(key, ..)| key.key().name() == name)
            .map(|(key, _, _, value)| {
                assert!(matches!(value, DebugValue::Counter(0)), "{key:?}: {value:?}");
                let mut labels: Vec<_> =
                    key.key().labels().map(|l| format!("{}={}", l.key(), l.value())).collect();
                labels.sort();
                labels.join(",")
            })
            .collect();
        series.sort();
        series
    }

    /// The gossip events are zeroed with the labels the driver records them with: a topic for
    /// messages and (un)subscriptions, none for the others.
    #[test]
    fn test_gossip_events_zeroed_with_recorded_labels() {
        let topic = TopicHash::from_raw("/optimism/10/0/blocks");
        assert_eq!(
            zeroed_series(Metrics::GOSSIP_EVENT, &topic),
            [
                "topic=/optimism/10/0/blocks,type=message",
                "topic=/optimism/10/0/blocks,type=subscribed",
                "topic=/optimism/10/0/blocks,type=unsubscribed",
                "type=not_supported",
                "type=slow_peer",
            ]
        );
    }

    /// Dial errors are only zeroed with a `type`, as they are always recorded with one.
    #[test]
    fn test_dial_errors_zeroed_with_a_type() {
        let topic = TopicHash::from_raw("/optimism/10/0/blocks");
        let series = zeroed_series(Metrics::DIAL_PEER_ERROR, &topic);
        assert!(!series.is_empty());
        assert!(series.iter().all(|labels| labels.starts_with("type=")), "{series:?}");
    }
}
