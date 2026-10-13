//! Sequencer CLI Flags
//!
//! These are based on sequencer flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/flags.go#L233-L265

use clap::Parser;
use kona_node_service::SequencerConfig;
use kona_sources::BlockSigner;
use url::Url;

/// Sequencer CLI Flags
///
/// They configure a node that can sequence, which is a node given a block signer.
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
pub struct SequencerArgs {
    /// Number of L1 blocks to keep distance from the L1 head as a sequencer for picking an L1
    /// origin. Ignored without a block signer.
    #[arg(long = "sequencer.l1-confs", default_value = "4", env = "KONA_NODE_SEQUENCER_L1_CONFS")]
    pub l1_confs: u64,

    /// Conductor service rpc endpoint. Providing this value will enable the conductor service.
    /// Requires a block signer.
    #[arg(long = "conductor.rpc", env = "KONA_NODE_CONDUCTOR_RPC")]
    pub conductor_rpc: Option<Url>,
}

impl Default for SequencerArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

/// `--conductor.rpc` was set without a block signer: only a node that can sequence uses the
/// conductor.
#[derive(Debug, thiserror::Error)]
#[error(
    "--conductor.rpc requires a block signer: set --p2p.sequencer.key, --p2p.sequencer.key.path, \
     or --p2p.signer.endpoint with --p2p.signer.address"
)]
pub struct ConductorWithoutSignerError;

impl SequencerArgs {
    /// Creates the [`SequencerConfig`] of a node with the given block `signer`, or `None` if the
    /// node has no signer and so cannot sequence.
    pub fn config(
        &self,
        signer: Option<BlockSigner>,
    ) -> Result<Option<SequencerConfig>, ConductorWithoutSignerError> {
        match (signer, &self.conductor_rpc) {
            (None, Some(_)) => Err(ConductorWithoutSignerError),
            (None, None) => Ok(None),
            (Some(signer), conductor_rpc_url) => Ok(Some(SequencerConfig {
                signer,
                conductor_rpc_url: conductor_rpc_url.clone(),
                l1_conf_delay: self.l1_confs,
            })),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_signer_local::PrivateKeySigner;

    #[test]
    fn config_is_some_exactly_when_there_is_a_signer() {
        let conductor: Url = "http://127.0.0.1:4".parse().unwrap();
        let signer = || Some(BlockSigner::from(PrivateKeySigner::random()));
        let args = |conductor_rpc| SequencerArgs { l1_confs: 2, conductor_rpc };

        assert!(args(None).config(None).unwrap().is_none());
        assert!(args(Some(conductor.clone())).config(None).is_err());

        let config = args(None).config(signer()).unwrap().unwrap();
        assert_eq!((config.conductor_rpc_url, config.l1_conf_delay), (None, 2));

        let config = args(Some(conductor.clone())).config(signer()).unwrap().unwrap();
        assert_eq!(config.conductor_rpc_url, Some(conductor));
    }
}
