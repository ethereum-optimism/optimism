//! Sequencer CLI Flags
//!
//! These are based on sequencer flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/flags.go#L233-L265

use clap::{ArgAction, Parser};
use kona_node_service::SequencerConfig;
use std::time::Duration;
use url::Url;

/// Sequencer CLI Flags
///
/// `--sequencer.enabled` makes the node able to sequence. The other flags here configure that
/// sequencer, and are ignored without it, so that setting `--sequencer.enabled=false` turns a
/// sequencer into a follower without unsetting them. A sequencer that gossips its blocks also
/// needs a block signer: see [`SignerArgs`](super::SignerArgs).
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
#[command(next_help_heading = "Sequencer")]
pub struct SequencerArgs {
    /// Run the sequencer. It starts stopped: start it with `admin_startSequencer`. Requires
    /// `--rpc.enabled`, and with `--p2p.enabled` a block signer. Disabled by default.
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

impl SequencerArgs {
    /// Creates the [`SequencerConfig`] of a node run with `--sequencer.enabled`, or `None` for a
    /// node that cannot sequence.
    pub fn config(self) -> Option<SequencerConfig> {
        self.enabled.then_some(SequencerConfig {
            conductor_rpc_url: self.conductor_url,
            conductor_rpc_timeout: self.conductor_timeout,
            l1_conf_delay: self.l1_confs,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::{CommandFactory, error::ErrorKind};
    use std::collections::BTreeSet;

    /// Each sequencer flag other than `--sequencer.enabled`, with a valid value.
    const SEQUENCER_FLAGS: [(&str, &str); 3] = [
        ("sequencer.l1-confs", "2"),
        ("sequencer.conductor.url", "http://127.0.0.1:4"),
        ("sequencer.conductor.timeout", "500ms"),
    ];

    fn parse(args: &[&str]) -> Result<SequencerArgs, clap::Error> {
        SequencerArgs::try_parse_from(std::iter::once("node").chain(args.iter().copied()))
    }

    /// The sequencer's flags apply only with `--sequencer.enabled`: without it, the node cannot
    /// sequence.
    #[test]
    fn sequencer_flags_are_ignored_without_the_sequencer() {
        let command = SequencerArgs::command();
        let flags: BTreeSet<_> = command
            .get_arguments()
            .filter_map(|arg| arg.get_long())
            .filter(|flag| *flag != "sequencer.enabled")
            .collect();
        let listed: BTreeSet<_> = SEQUENCER_FLAGS.iter().map(|(flag, _)| *flag).collect();
        assert_eq!(flags, listed, "list each new sequencer flag in SEQUENCER_FLAGS");

        let args: Vec<_> = SEQUENCER_FLAGS
            .iter()
            .map(|(flag, value)| [format!("--{flag}"), value.to_string()])
            .collect();
        for args in args.iter().map(std::slice::from_ref).chain([&args[..]]) {
            let args: Vec<_> = args.iter().flatten().map(String::as_str).collect();
            let sequencer = parse(&args).unwrap_or_else(|error| panic!("{args:?}: {error}"));
            assert!(sequencer.config().is_none(), "{args:?}");
        }
    }

    #[test]
    fn config_is_none_without_the_sequencer() {
        assert!(parse(&[]).unwrap().config().is_none());
    }

    #[test]
    fn config_carries_the_sequencer_flags() {
        let config = parse(&["--sequencer.enabled"]).unwrap().config().unwrap();
        assert_eq!((config.conductor_rpc_url, config.l1_conf_delay), (None, 4));
        assert_eq!(config.conductor_rpc_timeout, Duration::from_secs(1));

        let config = parse(&[
            "--sequencer.enabled",
            "--sequencer.l1-confs",
            "2",
            "--sequencer.conductor.url",
            "http://127.0.0.1:4",
            "--sequencer.conductor.timeout",
            "250ms",
        ])
        .unwrap()
        .config()
        .unwrap();
        assert_eq!(config.conductor_rpc_timeout, Duration::from_millis(250));
        assert_eq!(config.conductor_rpc_url, Some("http://127.0.0.1:4".parse().unwrap()));
        assert_eq!(config.l1_conf_delay, 2);
    }

    #[test]
    fn the_switch_takes_true_or_false() {
        let config = |enabled| parse(&[enabled]).map(|args| args.config().is_some());
        assert!(config("--sequencer.enabled=true").unwrap());
        assert!(!config("--sequencer.enabled=false").unwrap());
        assert_eq!(config("--sequencer.enabled=yes").unwrap_err().kind(), ErrorKind::InvalidValue);
    }

    #[test]
    fn the_environment_variable_takes_true_or_false() {
        // SAFETY: nextest runs each test in its own process.
        unsafe { std::env::set_var("KONA_NODE_SEQUENCER_ENABLED", "true") };
        let enabled = parse(&[]);
        // SAFETY: as above.
        unsafe { std::env::set_var("KONA_NODE_SEQUENCER_ENABLED", "false") };
        let disabled = parse(&[]);
        // SAFETY: as above.
        unsafe { std::env::remove_var("KONA_NODE_SEQUENCER_ENABLED") };

        assert!(enabled.unwrap().config().is_some());
        assert!(disabled.unwrap().config().is_none());
    }
}
