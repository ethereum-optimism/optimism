//! Sequencer configuration.

use std::time::Duration;
use url::Url;

/// Sequencer configuration.
///
/// A node given one can sequence.
#[derive(Debug, Clone)]
pub struct SequencerConfig {
    /// The [`Url`] for the conductor RPC endpoint. If [`Some`], enables the conductor service.
    pub conductor_rpc_url: Option<Url>,
    /// How long a conductor request may take before it fails.
    pub conductor_rpc_timeout: Duration,
    /// The confirmation delay for the sequencer.
    pub l1_conf_delay: u64,
}
