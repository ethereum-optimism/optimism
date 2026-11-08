//! Configuration for the [`SequencerActor`].
//!
//! [`SequencerActor`]: super::SequencerActor

use super::conductor::DEFAULT_CONDUCTOR_RPC_TIMEOUT;
use std::time::Duration;
use url::Url;

/// Configuration for the [`SequencerActor`].
///
/// [`SequencerActor`]: super::SequencerActor
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SequencerConfig {
    /// Whether to start stopped even without a conductor. A conductor always starts stopped.
    pub sequencer_stopped: bool,
    /// Whether or not the sequencer is in recovery mode.
    pub sequencer_recovery_mode: bool,
    /// The [`Url`] for the conductor RPC endpoint. If [`Some`], enables the conductor service.
    pub conductor_rpc_url: Option<Url>,
    /// Maximum duration of a conductor RPC.
    pub conductor_rpc_timeout: Duration,
    /// The confirmation delay for the sequencer.
    pub l1_conf_delay: u64,
}

impl SequencerConfig {
    /// Conductor-managed nodes must let the conductor reconcile its Raft head before starting.
    pub(crate) const fn initially_active(&self) -> bool {
        !self.sequencer_stopped && self.conductor_rpc_url.is_none()
    }
}

impl Default for SequencerConfig {
    fn default() -> Self {
        Self {
            sequencer_stopped: false,
            sequencer_recovery_mode: false,
            conductor_rpc_url: None,
            conductor_rpc_timeout: DEFAULT_CONDUCTOR_RPC_TIMEOUT,
            l1_conf_delay: 0,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn conductor_managed_sequencer_starts_stopped() {
        let mut config = SequencerConfig::default();
        assert!(config.initially_active());
        config.conductor_rpc_url = Some(Url::parse("http://localhost:8545").unwrap());
        assert!(!config.initially_active());
        config.conductor_rpc_url = None;
        config.sequencer_stopped = true;
        assert!(!config.initially_active());
    }
}
