//! Specifies peer monitoring configuration.

use std::time::Duration;

/// Specifies how to monitor peer scores and ban poorly performing peers.
#[derive(Debug, Clone)]
pub struct PeerMonitoring {
    /// The threshold under which a peer should be banned.
    pub ban_threshold: f64,
    /// The duration of a peer's ban.
    pub ban_duration: Duration,
}
