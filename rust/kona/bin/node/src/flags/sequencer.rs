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

    /// Reserved for op-node compatibility and not currently supported by kona-node.
    /// Values above 0 are rejected during startup.
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
    pub fn config(&self) -> anyhow::Result<SequencerConfig> {
        anyhow::ensure!(
            self.max_safe_lag == 0,
            "--sequencer.max-safe-lag is not implemented in kona-node. Only 0 (disabled) is supported; no safe-lag limit will be enforced."
        );

        Ok(SequencerConfig {
            sequencer_stopped: self.stopped,
            sequencer_recovery_mode: self.recover,
            conductor_rpc_url: self.conductor_rpc.clone(),
            conductor_rpc_timeout: self.conductor_rpc_timeout,
            l1_conf_delay: self.l1_confs,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_rejects_unsupported_max_safe_lag() {
        let args = SequencerArgs { max_safe_lag: 1, ..Default::default() };

        let err = args.config().unwrap_err();

        assert_eq!(
            err.to_string(),
            "--sequencer.max-safe-lag is not implemented in kona-node. Only 0 (disabled) is supported; no safe-lag limit will be enforced."
        );
    }

    #[test]
    fn config_propagates_conductor_rpc_timeout() {
        let timeout = Duration::from_secs(12);
        let args = SequencerArgs { conductor_rpc_timeout: timeout, ..Default::default() };

        let config = args.config().unwrap();

        assert_eq!(config.conductor_rpc_timeout, timeout);
    }
}
