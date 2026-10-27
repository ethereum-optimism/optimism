//! Sequencer CLI Flags
//!
//! These are based on sequencer flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/flags.go#L233-L265

use clap::Parser;
use kona_node_service::SequencerConfig;
use std::{num::ParseIntError, time::Duration};
use url::Url;

/// Sequencer CLI Flags
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
pub struct SequencerArgs {
    /// Initialize the sequencer in a stopped state. The sequencer can be started using the
    /// `admin_startSequencer` RPC. Conductor-managed nodes always start stopped.
    #[arg(
        long = "sequencer.stopped",
        default_value = "false",
        env = "KONA_NODE_SEQUENCER_STOPPED"
    )]
    pub stopped: bool,

    /// Pause new block builds when the unsafe head is this many blocks ahead of the safe head.
    /// Disabled if 0. Building resumes when the gap drops below the limit.
    #[arg(
        long = "sequencer.max-safe-lag",
        default_value = "0",
        env = "KONA_NODE_SEQUENCER_MAX_SAFE_LAG"
    )]
    pub max_safe_lag: u64,

    /// Number of L1 blocks to keep distance from the L1 head as a sequencer for picking an L1
    /// origin.
    #[arg(long = "sequencer.l1-confs", default_value = "4", env = "KONA_NODE_SEQUENCER_L1_CONFS")]
    pub l1_confs: u64,

    /// Forces the sequencer to strictly prepare the next L1 origin and create empty L2 blocks
    #[arg(
        long = "sequencer.recover",
        default_value = "false",
        env = "KONA_NODE_SEQUENCER_RECOVER"
    )]
    pub recover: bool,

    /// Conductor service rpc endpoint. Providing this value will enable the conductor service.
    #[arg(long = "conductor.rpc", env = "KONA_NODE_CONDUCTOR_RPC")]
    pub conductor_rpc: Option<Url>,

    /// Conductor service rpc timeout.
    #[arg(
        long = "conductor.rpc.timeout",
        default_value = "1",
        env = "KONA_NODE_CONDUCTOR_RPC_TIMEOUT",
        value_parser = |arg: &str| -> Result<Duration, ParseIntError> {Ok(Duration::from_secs(arg.parse()?))}
    )]
    pub conductor_rpc_timeout: Duration,
}

impl Default for SequencerArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl SequencerArgs {
    /// Creates a [`SequencerConfig`] from the [`SequencerArgs`].
    pub fn config(&self) -> SequencerConfig {
        SequencerConfig {
            sequencer_stopped: self.stopped || self.conductor_rpc.is_some(),
            sequencer_recovery_mode: self.recover,
            max_safe_lag: self.max_safe_lag,
            conductor_rpc_url: self.conductor_rpc.clone(),
            conductor_rpc_timeout: self.conductor_rpc_timeout,
            l1_conf_delay: self.l1_confs,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    #[rstest]
    fn conductor_managed_config_starts_stopped(
        #[values(true, false)] stopped: bool,
        #[values(true, false)] managed: bool,
    ) {
        let args = SequencerArgs {
            stopped,
            conductor_rpc: managed.then(|| "http://127.0.0.1:8547".parse().unwrap()),
            ..Default::default()
        };
        assert_eq!(args.config().sequencer_stopped, stopped || managed);
    }

    #[test]
    fn max_safe_lag_disabled() {
        assert_eq!(SequencerArgs::default().max_safe_lag, 0);
        let args = SequencerArgs::parse_from(["kona-node", "--sequencer.max-safe-lag=0"]);
        assert_eq!(args.max_safe_lag, 0);
    }

    #[rstest]
    #[case("1")]
    #[case("100")]
    #[case("18446744073709551615")]
    fn configured_max_safe_lag_reaches_sequencer_config(#[case] value: &str) {
        let args = SequencerArgs::parse_from(["kona-node", "--sequencer.max-safe-lag", value]);
        assert_eq!(args.config().max_safe_lag, value.parse::<u64>().unwrap());
    }

    #[test]
    fn max_safe_lag_from_environment() {
        const ENV: &str = "KONA_NODE_SEQUENCER_MAX_SAFE_LAG";
        // Parse in a child process so changing the environment cannot race other tests.
        if std::env::var_os("KONA_TEST_SAFE_LAG_CHILD").is_some() {
            let args = SequencerArgs::parse_from(["kona-node"]);
            assert_eq!(
                args.config().max_safe_lag,
                std::env::var(ENV).unwrap().parse::<u64>().unwrap()
            );
            let args = SequencerArgs::parse_from(["kona-node", "--sequencer.max-safe-lag=0"]);
            assert_eq!(args.config().max_safe_lag, 0, "explicit zero overrides the environment");
            return;
        }

        for value in ["0", "1", "18446744073709551615"] {
            let output = std::process::Command::new(std::env::current_exe().unwrap())
                .args([
                    "--exact",
                    "flags::sequencer::tests::max_safe_lag_from_environment",
                    "--nocapture",
                ])
                .env("KONA_TEST_SAFE_LAG_CHILD", "1")
                .env(ENV, value)
                .output()
                .unwrap();
            assert!(output.status.success(), "{output:?}");
        }
    }

    #[test]
    fn configured_conductor_timeout_reaches_sequencer_config() {
        let args = SequencerArgs::parse_from(["kona-node", "--conductor.rpc.timeout", "7"]);
        assert_eq!(args.config().conductor_rpc_timeout, Duration::from_secs(7));
        assert_eq!(
            SequencerArgs::default().config().conductor_rpc_timeout,
            SequencerConfig::default().conductor_rpc_timeout
        );
    }
}
