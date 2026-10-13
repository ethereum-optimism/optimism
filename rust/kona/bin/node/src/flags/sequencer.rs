//! Sequencer CLI Flags
//!
//! These are based on sequencer flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/flags.go#L233-L265

use super::signer::{BLOCK_SIGNER, SignerArgs, SignerKeyError};
use clap::Parser;
use kona_node_service::SequencerConfig;
use url::Url;

/// Sequencer CLI Flags
///
/// A node given a block signer can sequence. Every other flag here configures that sequencer, so
/// it requires a block signer.
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
#[command(next_help_heading = "Sequencer")]
pub struct SequencerArgs {
    /// The sequencer's block signer.
    #[command(flatten)]
    pub signer: SignerArgs,

    /// Number of L1 blocks to keep distance from the L1 head as a sequencer for picking an L1
    /// origin.
    #[arg(
        long = "sequencer.l1-confs",
        default_value = "4",
        env = "KONA_NODE_SEQUENCER_L1_CONFS",
        requires = BLOCK_SIGNER
    )]
    pub l1_confs: u64,

    /// Conductor service rpc endpoint. Providing this value will enable the conductor service.
    #[arg(
        long = "sequencer.conductor-url",
        env = "KONA_NODE_SEQUENCER_CONDUCTOR_URL",
        requires = BLOCK_SIGNER
    )]
    pub conductor_url: Option<Url>,
}

impl Default for SequencerArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl SequencerArgs {
    /// Creates the [`SequencerConfig`] of a node given a block signer, or `None` if the node has
    /// none and so cannot sequence.
    pub fn config(self) -> Result<Option<SequencerConfig>, SignerKeyError> {
        let Some(signer) = self.signer.config()? else {
            return Ok(None);
        };
        Ok(Some(SequencerConfig {
            signer,
            conductor_rpc_url: self.conductor_url,
            l1_conf_delay: self.l1_confs,
        }))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::flags::signer::test_utils::{KEY, key_file};
    use clap::{Args, CommandFactory};
    use std::collections::BTreeSet;

    /// Each flag that configures the sequencer, other than the signer's, with a valid value.
    const SEQUENCER_FLAGS: [(&str, &str); 2] =
        [("sequencer.l1-confs", "2"), ("sequencer.conductor-url", "http://127.0.0.1:4")];

    fn parse(args: &[&str]) -> Result<SequencerArgs, clap::Error> {
        SequencerArgs::try_parse_from(std::iter::once("node").chain(args.iter().copied()))
    }

    #[test]
    fn every_sequencer_flag_requires_a_block_signer() {
        let signer = SignerArgs::augment_args(clap::Command::new("signer"));
        let signer_flags: BTreeSet<_> =
            signer.get_arguments().filter_map(|arg| arg.get_long()).collect();
        let command = SequencerArgs::command();
        let sequencer_flags: BTreeSet<_> = command
            .get_arguments()
            .filter_map(|arg| arg.get_long())
            .filter(|flag| !signer_flags.contains(flag))
            .collect();
        let listed: BTreeSet<_> = SEQUENCER_FLAGS.iter().map(|(flag, _)| *flag).collect();
        assert_eq!(sequencer_flags, listed, "list each new sequencer flag in SEQUENCER_FLAGS");

        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();
        for (flag, value) in SEQUENCER_FLAGS {
            let flag = format!("--{flag}");
            let error = parse(&[&flag, value]).unwrap_err();
            assert_eq!(error.kind(), clap::error::ErrorKind::MissingRequiredArgument, "{flag}");

            let args = parse(&[&flag, value, "--sequencer.signer.key.path", key_path]).unwrap();
            assert!(args.config().unwrap().is_some(), "{flag}");
        }
    }

    #[test]
    fn config_is_none_without_a_block_signer() {
        assert!(parse(&[]).unwrap().config().unwrap().is_none());
    }

    #[test]
    fn config_carries_the_sequencer_flags() {
        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();

        let config = parse(&["--sequencer.signer.key.path", key_path]).unwrap().config().unwrap();
        let config = config.unwrap();
        assert_eq!((config.conductor_rpc_url, config.l1_conf_delay), (None, 4));

        let config = parse(&[
            "--sequencer.signer.key.path",
            key_path,
            "--sequencer.l1-confs",
            "2",
            "--sequencer.conductor-url",
            "http://127.0.0.1:4",
        ])
        .unwrap()
        .config()
        .unwrap()
        .unwrap();
        assert_eq!(config.conductor_rpc_url, Some("http://127.0.0.1:4".parse().unwrap()));
        assert_eq!(config.l1_conf_delay, 2);
    }
}
