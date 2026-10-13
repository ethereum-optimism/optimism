//! Sequencer configuration.

use kona_sources::BlockSigner;
use url::Url;

/// Sequencer configuration.
///
/// A node given one can sequence: it signs the blocks it builds for gossip with the
/// [`signer`](Self::signer).
#[derive(Debug, Clone)]
pub struct SequencerConfig {
    /// Signs the sequencer's blocks for gossip.
    pub signer: BlockSigner,
    /// The [`Url`] for the conductor RPC endpoint. If [`Some`], enables the conductor service.
    pub conductor_rpc_url: Option<Url>,
    /// The confirmation delay for the sequencer.
    pub l1_conf_delay: u64,
}
