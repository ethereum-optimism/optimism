//! Sequencer configuration.

use url::Url;

/// Sequencer configuration.
#[derive(Default, Debug, Clone, PartialEq, Eq)]
pub struct SequencerConfig {
    /// Whether or not the sequencer is in recovery mode.
    pub sequencer_recovery_mode: bool,
    /// The [`Url`] for the conductor RPC endpoint. If [`Some`], enables the conductor service.
    pub conductor_rpc_url: Option<Url>,
    /// The confirmation delay for the sequencer.
    pub l1_conf_delay: u64,
}
