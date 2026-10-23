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
    /// `admin_startSequencer` RPC.
    #[arg(
        long = "sequencer.stopped",
        default_value = "false",
        env = "KONA_NODE_SEQUENCER_STOPPED"
    )]
    pub stopped: bool,

    /// Maximum number of L2 blocks between safe and unsafe heads. Safe-lag enforcement is not
    /// implemented; only 0 (disabled) is supported.
    #[arg(
        long = "sequencer.max-safe-lag",
        default_value = "0",
        env = "KONA_NODE_SEQUENCER_MAX_SAFE_LAG",
        value_parser = Self::parse_max_safe_lag
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
    fn parse_max_safe_lag(value: &str) -> anyhow::Result<u64> {
        let lag = value.parse()?;
        anyhow::ensure!(
            lag == 0,
            "safe-lag enforcement is not implemented; remove --sequencer.max-safe-lag / \
             KONA_NODE_SEQUENCER_MAX_SAFE_LAG or set it to 0"
        );
        Ok(lag)
    }

    /// Creates a [`SequencerConfig`] from the [`SequencerArgs`].
    pub fn config(&self) -> SequencerConfig {
        SequencerConfig {
            sequencer_stopped: self.stopped,
            sequencer_recovery_mode: self.recover,
            conductor_rpc_url: self.conductor_rpc.clone(),
            conductor_rpc_timeout: self.conductor_rpc_timeout,
            l1_conf_delay: self.l1_confs,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::error::ErrorKind;
    use rstest::rstest;

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
    fn rejects_nonzero_max_safe_lag(#[case] value: &str) {
        let error = SequencerArgs::try_parse_from(["kona-node", "--sequencer.max-safe-lag", value])
            .unwrap_err();
        assert_eq!(error.kind(), ErrorKind::ValueValidation);
        assert!(error.to_string().contains("safe-lag enforcement is not implemented"));
        assert!(error.to_string().contains("or set it to 0"));
    }

    #[test]
    fn max_safe_lag_from_environment() {
        const ENV: &str = "KONA_NODE_SEQUENCER_MAX_SAFE_LAG";
        // Parse in a child process so changing the environment cannot race other tests.
        if std::env::var_os("KONA_TEST_SAFE_LAG_CHILD").is_some() {
            let result = SequencerArgs::try_parse_from(["kona-node"]);
            if std::env::var(ENV).unwrap() == "0" {
                assert_eq!(result.unwrap().max_safe_lag, 0);
            } else {
                let error = result.unwrap_err();
                assert_eq!(error.kind(), ErrorKind::ValueValidation);
                assert!(error.to_string().contains("safe-lag enforcement is not implemented"));
            }
            // An explicit zero must override an unsupported environment setting.
            assert!(
                SequencerArgs::try_parse_from(["kona-node", "--sequencer.max-safe-lag=0"]).is_ok()
            );
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
