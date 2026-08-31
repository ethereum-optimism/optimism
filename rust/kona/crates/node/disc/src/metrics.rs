//! Metrics for the discovery service.

/// Container for discovery metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for discv5 events.
    pub const DISCOVERY_EVENT: &str = "kona_node_discovery_events";

    /// Counter for the number of `FIND_NODE` requests.
    pub const FIND_NODE_REQUEST: &str = "kona_node_find_node_requests";

    /// Timer for the time taken to store ENRs in the bootstore.
    pub const ENR_STORE_TIME: &str = "kona_node_enr_store_time";

    /// Identifier for the gauge that tracks the number of peers in the discovery service.
    pub const DISCOVERY_PEER_COUNT: &str = "kona_node_discovery_peer_count";

    /// Initializes metrics for the discovery service.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in the discovery service.
    #[cfg(feature = "metrics")]
    pub fn describe() {
        metrics::describe_gauge!(Self::DISCOVERY_EVENT, "Events received by the discv5 service");
        metrics::describe_histogram!(
            Self::ENR_STORE_TIME,
            "Observations of elapsed time to store ENRs in the on-disk bootstore"
        );
        metrics::describe_gauge!(
            Self::DISCOVERY_PEER_COUNT,
            "Number of peers connected to the discv5 service"
        );
        metrics::describe_gauge!(
            Self::FIND_NODE_REQUEST,
            "Requests made to find a node through the discv5 peer discovery service"
        );
    }

    /// Initializes metrics to `0` so they can be queried immediately by consumers of prometheus
    /// metrics.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        // Discovery Event
        kona_macros::set!(gauge, Self::DISCOVERY_EVENT, "type", "discovered", 0);
        kona_macros::set!(gauge, Self::DISCOVERY_EVENT, "type", "session_established", 0);
        kona_macros::set!(gauge, Self::DISCOVERY_EVENT, "type", "unverifiable_enr", 0);

        // Peer Counts
        kona_macros::set!(gauge, Self::DISCOVERY_PEER_COUNT, 0);

        // The emit carries a constant `find_node` label, which the shipped dashboard selects on.
        kona_macros::set!(gauge, Self::FIND_NODE_REQUEST, "find_node", "find_node", 0);
    }
}

#[cfg(all(test, feature = "metrics"))]
mod tests {
    use super::Metrics;
    use metrics_util::debugging::DebuggingRecorder;

    #[test]
    fn find_node_requests_carry_the_label_the_emit_uses() {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, Metrics::zero);

        let labelled = snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .filter(|(ckey, ..)| ckey.key().name() == Metrics::FIND_NODE_REQUEST)
            .any(|(ckey, ..)| {
                ckey.key().labels().any(|l| l.key() == "find_node" && l.value() == "find_node")
            });

        assert!(labelled, "the pre-created find-node series must carry the emitted label");
    }
}
