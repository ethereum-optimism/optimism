//! Configuration for the [`SequencerActor`].
//!
//! [`SequencerActor`]: super::SequencerActor

use super::conductor::DEFAULT_CONDUCTOR_RPC_TIMEOUT;
use std::time::Duration;
use url::Url;

/// Configuration for the [`SequencerActor`].
///
/// The sequencer boots stopped unless [`SequencerConfig::sequencer_start`] is set and no
/// conductor is configured. A stopped sequencer is started with `admin_startSequencer`, which
/// op-conductor, an operator or a test harness calls once the node is up.
///
/// [`SequencerActor`]: super::SequencerActor
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SequencerConfig {
    /// Whether to boot the sequencer active, for a single sequencer with no standby. Ignored when
    /// a conductor is configured: op-conductor starts the sequencer itself.
    pub sequencer_start: bool,
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
    /// Whether the sequencer boots active. A conductor-managed sequencer always boots stopped, so
    /// the conductor can reconcile its Raft head before starting it.
    pub(crate) const fn initially_active(&self) -> bool {
        self.sequencer_start && self.conductor_rpc_url.is_none()
    }
}

impl Default for SequencerConfig {
    fn default() -> Self {
        Self {
            sequencer_start: false,
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
    fn sequencer_boots_active_only_when_started_without_conductor() {
        let mut config = SequencerConfig::default();
        assert!(!config.initially_active());
        config.sequencer_start = true;
        assert!(config.initially_active());
        config.conductor_rpc_url = Some(Url::parse("http://localhost:8547").unwrap());
        assert!(!config.initially_active());
    }
}
