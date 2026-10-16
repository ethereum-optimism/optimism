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
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in the discovery service.
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
    pub fn zero() {
        // Discovery Event
        metrics::gauge!(Self::DISCOVERY_EVENT, "type" => "discovered").set(0);
        metrics::gauge!(Self::DISCOVERY_EVENT, "type" => "session_established").set(0);
        metrics::gauge!(Self::DISCOVERY_EVENT, "type" => "unverifiable_enr").set(0);

        // Peer Counts
        metrics::gauge!(Self::DISCOVERY_PEER_COUNT).set(0);
        metrics::gauge!(Self::FIND_NODE_REQUEST).set(0);
    }
}
