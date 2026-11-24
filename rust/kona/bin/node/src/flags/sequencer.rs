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
///
/// The sequencer boots stopped unless `--sequencer.start` is set. A stopped sequencer is started
/// with the `admin_startSequencer` RPC.
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
pub struct SequencerArgs {
    /// Boot the sequencer active, at its unsafe head once that is known. Only for a single
    /// sequencer with no standby: a stop made through `admin_stopSequencer` doesn't survive a
    /// restart, so a stopped node boots active again. Without this flag the sequencer boots
    /// stopped and is started with `admin_startSequencer`. Can't be combined with
    /// `--conductor.rpc`: op-conductor starts the sequencer itself.
    #[arg(long = "sequencer.start", default_value = "false", env = "KONA_NODE_SEQUENCER_START")]
    pub start: bool,

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
    ///
    /// Fails if `--sequencer.start` is combined with `--conductor.rpc`. This is checked here
    /// rather than with clap's `conflicts_with`, which would also reject
    /// `KONA_NODE_SEQUENCER_START=false`.
    pub fn config(&self) -> anyhow::Result<SequencerConfig> {
        anyhow::ensure!(
            !(self.start && self.conductor_rpc.is_some()),
            "--sequencer.start can't be combined with --conductor.rpc: op-conductor starts the \
             sequencer itself"
        );
        Ok(SequencerConfig {
            sequencer_start: self.start,
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
    fn configured_conductor_timeout_reaches_sequencer_config() {
        let args = SequencerArgs::parse_from(["kona-node", "--conductor.rpc.timeout", "7"]);
        assert_eq!(args.config().unwrap().conductor_rpc_timeout, Duration::from_secs(7));
        assert_eq!(
            SequencerArgs::default().config().unwrap().conductor_rpc_timeout,
            SequencerConfig::default().conductor_rpc_timeout
        );
    }

    #[test]
    fn sequencer_boots_stopped_by_default() {
        assert!(!SequencerArgs::default().config().unwrap().sequencer_start);
        let args = SequencerArgs::parse_from(["kona-node", "--sequencer.start"]);
        assert!(args.config().unwrap().sequencer_start);
    }

    #[test]
    fn sequencer_start_conflicts_with_conductor() {
        let mut args = SequencerArgs::parse_from([
            "kona-node",
            "--sequencer.start",
            "--conductor.rpc",
            "http://localhost:8547",
        ]);
        assert!(args.config().is_err());
        // An explicit `KONA_NODE_SEQUENCER_START=false` is allowed with a conductor.
        args.start = false;
        assert!(!args.config().unwrap().sequencer_start);
    }
}
