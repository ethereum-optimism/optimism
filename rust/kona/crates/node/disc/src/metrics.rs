//! Metrics for the discovery service.

/// Container for discovery metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Label key carrying the L2 chain ID, present on every metric emitted by the discovery
    /// service.
    pub const CHAIN_ID_LABEL: &str = kona_macros::CHAIN_ID_LABEL;

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
    pub fn init(chain_id: u64) {
        Self::describe();
        Self::zero(chain_id);
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
    pub fn zero(chain_id: u64) {
        let chain_id = kona_macros::chain_id_label(chain_id);

        // Discovery Event
        for event in ["discovered", "session_established", "unverifiable_enr"] {
            kona_macros::set!(
                gauge,
                Self::DISCOVERY_EVENT,
                0,
                "type" => event,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }

        // Peer Counts
        kona_macros::set!(
            gauge,
            Self::DISCOVERY_PEER_COUNT,
            0,
            Self::CHAIN_ID_LABEL => chain_id.clone()
        );
        // The emit site in `driver.rs` tags this gauge with a constant `find_node` label, so the
        // pre-created series has to carry it too or it is never the one incremented.
        kona_macros::set!(
            gauge,
            Self::FIND_NODE_REQUEST,
            0,
            "find_node" => "find_node",
            Self::CHAIN_ID_LABEL => chain_id
        );
    }
}
