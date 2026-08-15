//! Local sequencing configuration.

use url::Url;

/// Configuration for the local producer owned by the unsafe-chain service.
#[derive(Default, Debug, Clone, PartialEq, Eq)]
pub struct SequencerConfig {
    /// Whether local sequencing starts in the stopped state.
    pub sequencer_stopped: bool,
    /// Whether local blocks initially exclude transaction-pool transactions.
    pub sequencer_recovery_mode: bool,
    /// Optional HA conductor endpoint.
    pub conductor_rpc_url: Option<Url>,
    /// L1 confirmation delay used when selecting origins.
    pub l1_conf_delay: u64,
}
