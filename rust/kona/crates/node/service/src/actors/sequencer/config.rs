//! Configuration for the [`SequencerActor`].
//!
//! [`SequencerActor`]: super::SequencerActor

use std::time::Duration;
use url::Url;

/// Configuration for the [`SequencerActor`].
///
/// [`SequencerActor`]: super::SequencerActor
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SequencerConfig {
    /// Whether or not the sequencer is enabled at startup.
    pub sequencer_stopped: bool,
    /// Whether or not the sequencer is in recovery mode.
    pub sequencer_recovery_mode: bool,
    /// The [`Url`] for the conductor RPC endpoint. If [`Some`], enables the conductor service.
    pub conductor_rpc_url: Option<Url>,
    /// Deadline for each conductor RPC, including its response body.
    pub conductor_rpc_timeout: Duration,
    /// The confirmation delay for the sequencer.
    pub l1_conf_delay: u64,
}

impl Default for SequencerConfig {
    fn default() -> Self {
        Self {
            sequencer_stopped: false,
            sequencer_recovery_mode: false,
            conductor_rpc_url: None,
            conductor_rpc_timeout: Duration::from_secs(1),
            l1_conf_delay: 0,
        }
    }
}
