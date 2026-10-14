//! Sequencer configuration.

use kona_sources::BlockSigner;
use std::time::Duration;
use url::Url;

/// Sequencer configuration.
///
/// A node given one can sequence. With P2P, it signs the blocks it builds for gossip with the
/// [`signer`](Self::signer); without P2P, it neither signs nor gossips them.
#[derive(Debug, Clone)]
pub struct SequencerConfig {
    /// Signs the sequencer's blocks for gossip. Unused by a node without P2P.
    pub signer: BlockSigner,
    /// The [`Url`] for the conductor RPC endpoint. If [`Some`], enables the conductor service.
    pub conductor_rpc_url: Option<Url>,
    /// How long a conductor request may take before it fails.
    pub conductor_rpc_timeout: Duration,
    /// The confirmation delay for the sequencer.
    pub l1_conf_delay: u64,
}
