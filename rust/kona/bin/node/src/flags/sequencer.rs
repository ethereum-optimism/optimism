//! Sequencer CLI Flags
//!
//! These are based on sequencer flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/flags.go#L233-L265

use super::signer::{SignerArgs, SignerKeyError};
use clap::{ArgAction, Parser};
use kona_node_service::SequencerConfig;
use std::time::Duration;
use url::Url;

/// Sequencer CLI Flags
///
/// `--sequencer.enabled` makes the node able to sequence, and then requires a block signer. The
/// other flags here configure that sequencer, and are ignored without it, so that setting
/// `--sequencer.enabled=false` turns a sequencer into a follower without unsetting them.
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
#[command(next_help_heading = "Sequencer")]
pub struct SequencerArgs {
    /// Run the sequencer. It starts stopped: start it with `admin_startSequencer`. Requires a
    /// block signer. Disabled by default.
    #[arg(
        id = "sequencer_enabled",
        long = "sequencer.enabled",
        env = "KONA_NODE_SEQUENCER_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true"
    )]
    pub enabled: bool,

    /// The sequencer's block signer.
    #[command(flatten)]
    pub signer: SignerArgs,

    /// Number of L1 blocks to keep distance from the L1 head as a sequencer for picking an L1
    /// origin.
    #[arg(long = "sequencer.l1-confs", default_value = "4", env = "KONA_NODE_SEQUENCER_L1_CONFS")]
    pub l1_confs: u64,

    /// Conductor service rpc endpoint. Providing this value will enable the conductor service.
    #[arg(long = "sequencer.conductor.url", env = "KONA_NODE_SEQUENCER_CONDUCTOR_URL")]
    pub conductor_url: Option<Url>,

    /// How long a conductor request may take before it fails and the sequencer tries again later,
    /// such as `500ms` or `2s`. op-node's `--conductor.rpc-timeout` defaults to the same second.
    #[arg(
        long = "sequencer.conductor.timeout",
        default_value = "1s",
        env = "KONA_NODE_SEQUENCER_CONDUCTOR_TIMEOUT",
        value_parser = humantime::parse_duration
    )]
    pub conductor_timeout: Duration,
}

impl Default for SequencerArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

/// An error in the sequencer's flags.
#[derive(Debug, thiserror::Error)]
pub enum SequencerFlagsError {
    /// The node was run with `--sequencer.enabled` but without a block signer.
    #[error(
        "--sequencer.enabled requires a block signer: --sequencer.signer.key-path, or \
         --sequencer.signer.url with --sequencer.signer.address"
    )]
    NoBlockSigner,
    /// The block signer's key could not be read.
    #[error(transparent)]
    SignerKey(#[from] SignerKeyError),
}

impl SequencerArgs {
    /// Creates the [`SequencerConfig`] of a node run with `--sequencer.enabled`, or `None` for a
    /// node that cannot sequence.
    pub fn config(self) -> Result<Option<SequencerConfig>, SequencerFlagsError> {
        if !self.enabled {
            return Ok(None);
        }
        let signer = self.signer.config()?.ok_or(SequencerFlagsError::NoBlockSigner)?;
        Ok(Some(SequencerConfig {
            signer,
            conductor_rpc_url: self.conductor_url,
            conductor_rpc_timeout: self.conductor_timeout,
            l1_conf_delay: self.l1_confs,
        }))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::flags::signer::test_utils::{KEY, key_file};
    use clap::{CommandFactory, error::ErrorKind};
    use std::collections::BTreeSet;

    const ADDRESS: &str = "0x0000000000000000000000000000000000000001";
    const LOCAL: [&str; 3] = ["--sequencer.enabled", "--sequencer.signer.key-path", "/key"];
    const REMOTE: [&str; 5] = [
        "--sequencer.enabled",
        "--sequencer.signer.url",
        "http://127.0.0.1:8080",
        "--sequencer.signer.address",
        ADDRESS,
    ];

    /// Each sequencer flag other than `--sequencer.enabled`, with a valid value, and the flags that
    /// complete it.
    const SEQUENCER_FLAGS: [(&str, &str, &[&str]); 10] = [
        ("sequencer.signer.key-path", "/key", &["--sequencer.enabled"]),
        ("sequencer.signer.url", "http://127.0.0.1:8080", &REMOTE),
        ("sequencer.signer.address", ADDRESS, &REMOTE),
        ("sequencer.signer.header", "a=b", &REMOTE),
        ("sequencer.signer.tls.ca-path", "/ca.pem", &REMOTE),
        (
            "sequencer.signer.tls.cert-path",
            "/cert.pem",
            &[
                "--sequencer.enabled",
                "--sequencer.signer.url",
                "http://127.0.0.1:8080",
                "--sequencer.signer.address",
                ADDRESS,
                "--sequencer.signer.tls.key-path",
                "/key.pem",
            ],
        ),
        (
            "sequencer.signer.tls.key-path",
            "/key.pem",
            &[
                "--sequencer.enabled",
                "--sequencer.signer.url",
                "http://127.0.0.1:8080",
                "--sequencer.signer.address",
                ADDRESS,
                "--sequencer.signer.tls.cert-path",
                "/cert.pem",
            ],
        ),
        ("sequencer.l1-confs", "2", &LOCAL),
        ("sequencer.conductor.url", "http://127.0.0.1:4", &LOCAL),
        ("sequencer.conductor.timeout", "500ms", &LOCAL),
    ];

    fn parse(args: &[&str]) -> Result<SequencerArgs, clap::Error> {
        SequencerArgs::try_parse_from(std::iter::once("node").chain(args.iter().copied()))
    }

    /// The sequencer's flags apply only with `--sequencer.enabled`: without it, each parses and the
    /// node cannot sequence.
    #[test]
    fn sequencer_flags_are_ignored_without_the_sequencer() {
        let command = SequencerArgs::command();
        let flags: BTreeSet<_> = command
            .get_arguments()
            .filter_map(|arg| arg.get_long())
            .filter(|flag| *flag != "sequencer.enabled")
            .collect();
        let listed: BTreeSet<_> = SEQUENCER_FLAGS.iter().map(|(flag, ..)| *flag).collect();
        assert_eq!(flags, listed, "list each new sequencer flag in SEQUENCER_FLAGS");

        for (flag, value, completion) in SEQUENCER_FLAGS {
            let flag = format!("--{flag}");
            let mut args: Vec<_> =
                completion.iter().copied().filter(|arg| *arg != "--sequencer.enabled").collect();
            if !args.contains(&flag.as_str()) {
                args.extend([flag.as_str(), value]);
            }
            let sequencer = parse(&args).unwrap_or_else(|error| panic!("{flag}: {error}"));
            assert!(sequencer.config().unwrap().is_none(), "{flag}");
        }
    }

    #[test]
    fn the_sequencer_requires_a_block_signer() {
        let error = parse(&["--sequencer.enabled"]).unwrap().config().unwrap_err();
        assert!(matches!(error, SequencerFlagsError::NoBlockSigner));
    }

    #[test]
    fn config_is_none_without_the_sequencer() {
        assert!(parse(&[]).unwrap().config().unwrap().is_none());
    }

    #[test]
    fn config_carries_the_sequencer_flags() {
        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();

        let config = parse(&["--sequencer.enabled", "--sequencer.signer.key-path", key_path])
            .unwrap()
            .config()
            .unwrap()
            .unwrap();
        assert_eq!((config.conductor_rpc_url, config.l1_conf_delay), (None, 4));
        assert_eq!(config.conductor_rpc_timeout, Duration::from_secs(1));

        let config = parse(&[
            "--sequencer.enabled",
            "--sequencer.signer.key-path",
            key_path,
            "--sequencer.l1-confs",
            "2",
            "--sequencer.conductor.url",
            "http://127.0.0.1:4",
            "--sequencer.conductor.timeout",
            "250ms",
        ])
        .unwrap()
        .config()
        .unwrap()
        .unwrap();
        assert_eq!(config.conductor_rpc_timeout, Duration::from_millis(250));
        assert_eq!(config.conductor_rpc_url, Some("http://127.0.0.1:4".parse().unwrap()));
        assert_eq!(config.l1_conf_delay, 2);
    }

    #[test]
    fn the_switch_takes_true_or_false() {
        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();
        let config = |enabled| {
            parse(&[enabled, "--sequencer.signer.key-path", key_path])
                .map(|args| args.config().unwrap().is_some())
        };
        assert!(config("--sequencer.enabled=true").unwrap());
        assert!(!config("--sequencer.enabled=false").unwrap());
        assert_eq!(config("--sequencer.enabled=yes").unwrap_err().kind(), ErrorKind::InvalidValue);
    }

    #[test]
    fn the_environment_variable_takes_true_or_false() {
        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();

        // SAFETY: nextest runs each test in its own process.
        unsafe { std::env::set_var("KONA_NODE_SEQUENCER_ENABLED", "true") };
        let enabled = parse(&["--sequencer.signer.key-path", key_path]);
        // SAFETY: as above.
        unsafe { std::env::set_var("KONA_NODE_SEQUENCER_ENABLED", "false") };
        let disabled = parse(&["--sequencer.signer.key-path", key_path]);
        // SAFETY: as above.
        unsafe { std::env::remove_var("KONA_NODE_SEQUENCER_ENABLED") };

        assert!(enabled.unwrap().config().unwrap().is_some());
        assert!(disabled.unwrap().config().unwrap().is_none());
    }
}
