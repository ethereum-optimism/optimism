//! Sequencer CLI Flags
//!
//! These are based on sequencer flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/flags.go#L233-L265

use super::signer::{BLOCK_SIGNER, SignerArgs, SignerKeyError};
use clap::{ArgAction, Parser};
use kona_node_service::SequencerConfig;
use std::time::Duration;
use url::Url;

/// The id of `--sequencer.enabled`, which every other sequencer flag requires.
pub(super) const SEQUENCER_ENABLED: &str = "sequencer_enabled";

/// Sequencer CLI Flags
///
/// `--sequencer.enabled` makes the node able to sequence, and requires a block signer. Every other
/// flag here configures that sequencer, so it requires `--sequencer.enabled`.
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
#[command(next_help_heading = "Sequencer")]
pub struct SequencerArgs {
    /// Run the sequencer. It starts stopped: start it with `admin_startSequencer`. Requires a
    /// block signer.
    ///
    /// The environment variable must be `true` or unset.
    #[arg(
        id = SEQUENCER_ENABLED,
        long = "sequencer.enabled",
        env = "KONA_NODE_SEQUENCER_ENABLED",
        action = ArgAction::Set,
        num_args = 0,
        default_missing_value = "true",
        value_parser = kona_cli::parse_enabled,
        requires = BLOCK_SIGNER
    )]
    pub enabled: Option<bool>,

    /// The sequencer's block signer.
    #[command(flatten)]
    pub signer: SignerArgs,

    /// Number of L1 blocks to keep distance from the L1 head as a sequencer for picking an L1
    /// origin.
    #[arg(
        long = "sequencer.l1-confs",
        default_value = "4",
        env = "KONA_NODE_SEQUENCER_L1_CONFS",
        requires = SEQUENCER_ENABLED
    )]
    pub l1_confs: u64,

    /// Conductor service rpc endpoint. Providing this value will enable the conductor service.
    #[arg(
        long = "sequencer.conductor-url",
        env = "KONA_NODE_SEQUENCER_CONDUCTOR_URL",
        requires = SEQUENCER_ENABLED
    )]
    pub conductor_url: Option<Url>,

    /// How long a conductor request may take, in milliseconds, before it fails and the sequencer
    /// tries again later. op-node's `--conductor.rpc-timeout` defaults to the same second.
    #[arg(
        long = "sequencer.conductor-timeout-ms",
        default_value = "1000",
        env = "KONA_NODE_SEQUENCER_CONDUCTOR_TIMEOUT_MS",
        requires = SEQUENCER_ENABLED
    )]
    pub conductor_timeout_ms: u64,
}

impl Default for SequencerArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl SequencerArgs {
    /// Creates the [`SequencerConfig`] of a node run with `--sequencer.enabled`, or `None` for a
    /// node that cannot sequence.
    pub fn config(self) -> Result<Option<SequencerConfig>, SignerKeyError> {
        if self.enabled.is_none() {
            return Ok(None);
        }
        // `--sequencer.enabled` requires a block signer.
        let Some(signer) = self.signer.config()? else {
            return Ok(None);
        };
        Ok(Some(SequencerConfig {
            signer,
            conductor_rpc_url: self.conductor_url,
            conductor_rpc_timeout: Duration::from_millis(self.conductor_timeout_ms),
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
    const LOCAL: [&str; 3] = ["--sequencer.enabled", "--sequencer.signer.key.path", "/key"];
    const REMOTE: [&str; 5] = [
        "--sequencer.enabled",
        "--sequencer.signer.url",
        "http://127.0.0.1:8080",
        "--sequencer.signer.address",
        ADDRESS,
    ];

    /// Each sequencer flag other than `--sequencer.enabled`, with a valid value, whether it
    /// requires `--sequencer.enabled` itself rather than through another flag, and the flags that
    /// complete it.
    const SEQUENCER_FLAGS: [(&str, &str, bool, &[&str]); 10] = [
        ("sequencer.signer.key.path", "/key", true, &["--sequencer.enabled"]),
        ("sequencer.signer.url", "http://127.0.0.1:8080", true, &REMOTE),
        ("sequencer.signer.address", ADDRESS, false, &REMOTE),
        ("sequencer.signer.header", "a=b", false, &REMOTE),
        ("sequencer.signer.tls.ca", "/ca.pem", false, &REMOTE),
        (
            "sequencer.signer.tls.cert",
            "/cert.pem",
            false,
            &[
                "--sequencer.enabled",
                "--sequencer.signer.url",
                "http://127.0.0.1:8080",
                "--sequencer.signer.address",
                ADDRESS,
                "--sequencer.signer.tls.key",
                "/key.pem",
            ],
        ),
        (
            "sequencer.signer.tls.key",
            "/key.pem",
            false,
            &[
                "--sequencer.enabled",
                "--sequencer.signer.url",
                "http://127.0.0.1:8080",
                "--sequencer.signer.address",
                ADDRESS,
                "--sequencer.signer.tls.cert",
                "/cert.pem",
            ],
        ),
        ("sequencer.l1-confs", "2", true, &LOCAL),
        ("sequencer.conductor-url", "http://127.0.0.1:4", true, &LOCAL),
        ("sequencer.conductor-timeout-ms", "500", true, &LOCAL),
    ];

    fn parse(args: &[&str]) -> Result<SequencerArgs, clap::Error> {
        SequencerArgs::try_parse_from(std::iter::once("node").chain(args.iter().copied()))
    }

    #[test]
    fn every_sequencer_flag_requires_the_sequencer() {
        let command = SequencerArgs::command();
        let flags: BTreeSet<_> = command
            .get_arguments()
            .filter_map(|arg| arg.get_long())
            .filter(|flag| *flag != "sequencer.enabled")
            .collect();
        let listed: BTreeSet<_> = SEQUENCER_FLAGS.iter().map(|(flag, ..)| *flag).collect();
        assert_eq!(flags, listed, "list each new sequencer flag in SEQUENCER_FLAGS");

        for (flag, value, direct, completion) in SEQUENCER_FLAGS {
            let flag = format!("--{flag}");
            let error = parse(&[&flag, value]).unwrap_err();
            assert_eq!(error.kind(), ErrorKind::MissingRequiredArgument, "{flag}");
            if direct {
                assert!(error.to_string().contains("--sequencer.enabled"), "{flag}: {error}");
            }

            let mut args = completion.to_vec();
            if !args.contains(&flag.as_str()) {
                args.extend([flag.as_str(), value]);
            }
            parse(&args).unwrap_or_else(|error| panic!("{flag}: {error}"));
        }
    }

    #[test]
    fn the_sequencer_requires_a_block_signer() {
        let error = parse(&["--sequencer.enabled"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::MissingRequiredArgument);
    }

    #[test]
    fn config_is_none_without_the_sequencer() {
        assert!(parse(&[]).unwrap().config().unwrap().is_none());
    }

    #[test]
    fn config_carries_the_sequencer_flags() {
        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();

        let config = parse(&["--sequencer.enabled", "--sequencer.signer.key.path", key_path])
            .unwrap()
            .config()
            .unwrap()
            .unwrap();
        assert_eq!((config.conductor_rpc_url, config.l1_conf_delay), (None, 4));

        let config = parse(&[
            "--sequencer.enabled",
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

    #[test]
    fn the_flag_takes_no_value() {
        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();
        for enabled in ["--sequencer.enabled=true", "--sequencer.enabled=false"] {
            let error = parse(&[enabled, "--sequencer.signer.key.path", key_path]).unwrap_err();
            assert_eq!(error.kind(), ErrorKind::TooManyValues, "{enabled}");
        }
    }

    #[test]
    fn the_environment_variable_must_be_true_or_unset() {
        let key = key_file(KEY);
        let key_path = key.path().to_str().unwrap();

        // SAFETY: nextest runs each test in its own process.
        unsafe { std::env::set_var("KONA_NODE_SEQUENCER_ENABLED", "true") };
        let enabled = parse(&["--sequencer.signer.key.path", key_path]);
        // SAFETY: as above.
        unsafe { std::env::set_var("KONA_NODE_SEQUENCER_ENABLED", "false") };
        let disabled = parse(&["--sequencer.signer.key.path", key_path]);
        // SAFETY: as above.
        unsafe { std::env::remove_var("KONA_NODE_SEQUENCER_ENABLED") };

        assert!(enabled.unwrap().config().unwrap().is_some());
        assert_eq!(disabled.unwrap_err().kind(), ErrorKind::ValueValidation);
    }
}
