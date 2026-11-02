//! The kona rollup node, run inside op-reth with `--rollup.consensus.enabled`.
//!
//! The rollup node takes kona-node's `node` flags and hardfork overrides, each renamed under
//! `--rollup.consensus.` and read from the same `KONA_NODE_*` environment variables. It drives
//! op-reth through the Engine API, at the `--rollup.consensus.l2-engine-rpc` it is given.

use alloy_chains::Chain;
use clap::{
    Arg, ArgAction, ArgMatches, Args, Command, CommandFactory, FromArgMatches, builder::Resettable,
    parser::ValueSource,
};
use kona_node::{
    cli::ignore_empty_list_envs,
    commands::NodeCommand,
    flags::{GlobalArgs, OverrideArgs, init_node_metrics},
};
use std::ffi::OsString;
use tracing::info;

/// The flag that runs the rollup node.
const ENABLED: &str = "rollup.consensus.enabled";

/// The prefix of the rollup node's flags, and of the ids of their copies in op-reth's command.
const PREFIX: &str = "rollup.consensus.";

/// The rollup node's command-line arguments.
#[derive(Debug, Clone, Default)]
pub(crate) struct RollupConsensusArgs {
    /// kona's arguments, when `--rollup.consensus.enabled` is set.
    kona: Option<KonaArgs>,
}

/// kona-node's arguments.
#[derive(Debug, Clone)]
struct KonaArgs {
    node: NodeCommand,
    overrides: OverrideArgs,
}

impl RollupConsensusArgs {
    /// Returns whether the rollup node runs.
    pub(crate) const fn enabled(&self) -> bool {
        self.kona.is_some()
    }

    /// Runs the rollup node for the L2 `chain`, or never resolves when it is disabled.
    ///
    /// kona's metrics are recorded by op-reth's metrics recorder, so they are reported when
    /// `metrics` is set.
    pub(crate) async fn run(self, chain: Chain, metrics: bool) -> eyre::Result<()> {
        let Some(KonaArgs { node, overrides }) = self.kona else {
            return std::future::pending().await;
        };

        let mut global =
            GlobalArgs { l2_chain_id: chain, override_args: overrides, ..Default::default() };
        global.metrics.enabled = metrics;
        if metrics {
            init_node_metrics();
        }
        node.init_cli_metrics(&global.metrics).map_err(|err| eyre::eyre!(err))?;

        info!(target: "reth::cli", "Starting the rollup node");
        node.run(&global).await.map_err(|err| eyre::eyre!(err))
    }
}

/// kona-node's `node` command and hardfork overrides, with each flag renamed under [`PREFIX`].
///
/// The arguments keep kona's ids, so its matches parse into [`NodeCommand`] and [`OverrideArgs`].
fn kona_command() -> Command {
    let cmd = OverrideArgs::augment_args(NodeCommand::command());
    ignore_empty_list_envs(cmd).mut_args(|arg| {
        let long = arg.get_long().map(|long| format!("{PREFIX}{long}"));
        let visible = arg.get_visible_aliases().unwrap_or_default();
        let aliases: Vec<_> = arg
            .get_all_aliases()
            .unwrap_or_default()
            .into_iter()
            .map(|alias| (format!("{PREFIX}{alias}"), visible.contains(&alias)))
            .collect();
        let arg = arg.alias(None).short(None).short_alias(None);
        let arg = match long {
            Some(long) => arg.long(long),
            None => arg,
        };
        aliases.into_iter().fold(arg, |arg, (alias, visible)| {
            if visible { arg.visible_alias(alias) } else { arg.alias(alias) }
        })
    })
}

/// The id, in op-reth's command, of the copy of kona's argument `arg`.
fn copy_id(arg: &Arg) -> String {
    format!("{PREFIX}{}", arg.get_id())
}

/// Returns the command-line arguments that set kona's flags, as kona's command reads them.
///
/// Only flags set on the command line are returned: kona's command reads its environment and
/// defaults itself. They are grouped by flag rather than kept in the order given, which matters
/// only to flags that override one another, and kona has none.
fn command_line_args(kona: &Command, matches: &ArgMatches) -> Vec<OsString> {
    let mut args = Vec::new();
    for arg in kona.get_arguments() {
        let id = copy_id(arg);
        if matches.value_source(&id) != Some(ValueSource::CommandLine) {
            continue;
        }
        let long = format!("--{}", arg.get_long().expect("kona's flags are all long"));
        for occurrence in matches.get_raw_occurrences(&id).into_iter().flatten() {
            if !arg.get_action().takes_values() {
                args.push(long.clone().into());
                continue;
            }
            let values: Vec<_> = occurrence.collect();
            let values = match arg.get_value_delimiter() {
                Some(delimiter) => vec![values.join(delimiter.to_string().as_ref())],
                None => values.into_iter().map(Into::into).collect(),
            };
            for value in values {
                let mut flag = OsString::from(format!("{long}="));
                flag.push(value);
                args.push(flag);
            }
        }
    }
    args
}

impl Args for RollupConsensusArgs {
    fn augment_args(cmd: Command) -> Command {
        let heading = "Rollup Consensus";
        // Copies of kona's flags, so that op-reth accepts them and lists them in its help. kona's
        // own command validates them, so the copies require only that the rollup node runs.
        let kona = kona_command();
        let copies = kona.get_arguments().map(|arg| {
            arg.clone()
                .id(copy_id(arg))
                .required(false)
                .requires(Resettable::Reset)
                .requires(ENABLED)
                .conflicts_with(Resettable::Reset)
                .help_heading(heading)
        });
        cmd.arg(
            Arg::new(ENABLED)
                .long(ENABLED)
                .action(ArgAction::SetTrue)
                .help_heading(heading)
                .help("Run the kona rollup node in this process, configured by the other --rollup.consensus.* flags"),
        )
        .args(copies)
    }

    fn augment_args_for_update(cmd: Command) -> Command {
        Self::augment_args(cmd)
    }
}

impl FromArgMatches for RollupConsensusArgs {
    fn from_arg_matches(matches: &ArgMatches) -> Result<Self, clap::Error> {
        if !matches.get_flag(ENABLED) {
            return Ok(Self::default());
        }

        let mut kona = kona_command();
        let args = command_line_args(&kona, matches);
        let matches = kona.try_get_matches_from_mut(
            std::iter::once(OsString::from("op-reth node")).chain(args),
        )?;
        let node = NodeCommand::from_arg_matches(&matches).map_err(|err| err.format(&mut kona))?;
        let overrides =
            OverrideArgs::from_arg_matches(&matches).map_err(|err| err.format(&mut kona))?;
        Ok(Self { kona: Some(KonaArgs { node, overrides }) })
    }

    fn update_from_arg_matches(&mut self, matches: &ArgMatches) -> Result<(), clap::Error> {
        *self = Self::from_arg_matches(matches)?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::{Parser, error::ErrorKind};

    /// A command with only the rollup node's flags.
    #[derive(Debug, Parser)]
    struct TestCommand {
        #[command(flatten)]
        consensus: RollupConsensusArgs,
    }

    fn parse(args: &[&str]) -> Result<RollupConsensusArgs, clap::Error> {
        TestCommand::try_parse_from(std::iter::once("op-reth").chain(args.iter().copied()))
            .map(|cmd| cmd.consensus)
    }

    const REQUIRED: [&str; 7] = [
        "--rollup.consensus.enabled",
        "--rollup.consensus.l1-eth-rpc",
        "http://localhost:8545",
        "--rollup.consensus.l1-beacon",
        "http://localhost:5052",
        "--rollup.consensus.l2-engine-rpc",
        "http://localhost:8551",
    ];

    fn kona(args: &[&str]) -> KonaArgs {
        let args: Vec<_> = REQUIRED.iter().chain(args).copied().collect();
        parse(&args).unwrap().kona.unwrap()
    }

    /// Each of kona's flags is copied under the prefix, with its aliases and environment variable.
    #[test]
    fn test_copies_every_kona_flag() {
        let original = OverrideArgs::augment_args(NodeCommand::command());
        let command = TestCommand::command();
        for arg in original.get_arguments() {
            let long = arg.get_long().expect("kona's flags are all long");
            let copy = command
                .get_arguments()
                .find(|copy| copy.get_long() == Some(&format!("{PREFIX}{long}")))
                .unwrap_or_else(|| panic!("--{long} is not copied"));
            assert_eq!(copy.get_env(), arg.get_env(), "--{long}");
            let aliases: Vec<_> = arg
                .get_all_aliases()
                .unwrap_or_default()
                .into_iter()
                .map(|alias| format!("{PREFIX}{alias}"))
                .collect();
            assert_eq!(copy.get_all_aliases().unwrap_or_default(), aliases, "--{long}");
        }
    }

    #[test]
    fn test_disabled_by_default() {
        assert!(!parse(&[]).unwrap().enabled());
    }

    const ENODE_1: &str = "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305";
    const ENODE_2: &str = "enode://dd751a9ef8912be1bfa7a5e34e2c3785cc5253110bd929f385e07ba7ac19929fb0e0c5d93f77827291f4da02b2232240fbc47ea7ce04c46e333e452f8656b667@34.65.107.0:30305";

    #[test]
    fn test_reads_prefixed_flags() {
        let bootnodes = format!("{ENODE_1},{ENODE_2}");
        let KonaArgs { node, overrides } = kona(&[
            "--rollup.consensus.mode",
            "Sequencer",
            "--rollup.consensus.p2p.bootnodes",
            &bootnodes,
            "--rollup.consensus.verifier.l1-confs=4",
            "--rollup.consensus.l1-trust-rpc",
            "false",
            "--rollup.consensus.p2p.no-discovery",
            "--rollup.consensus.canyon-override",
            "7",
        ]);
        assert_eq!(node.node_mode.to_string(), "Sequencer");
        assert_eq!(node.l1_rpc_args.l1_eth_rpc.as_str(), "http://localhost:8545/");
        assert_eq!(node.l2_client_args.l2_engine_rpc.as_str(), "http://localhost:8551/");
        assert_eq!(node.p2p_flags.bootnodes.len(), 2);
        assert_eq!(node.verifier_l1_confs, 4);
        assert!(!node.l1_rpc_args.l1_trust_rpc);
        assert!(node.p2p_flags.no_discovery);
        assert_eq!(overrides.canyon_override, Some(7));
    }

    #[test]
    fn test_reads_repeated_flags() {
        let node = kona(&[
            "--rollup.consensus.p2p.bootnodes",
            ENODE_1,
            "--rollup.consensus.p2p.bootnodes",
            ENODE_2,
        ])
        .node;
        assert_eq!(node.p2p_flags.bootnodes.len(), 2);
    }

    #[test]
    fn test_reads_prefixed_aliases() {
        let node = kona(&["--rollup.consensus.rpc.port", "9545"]).node;
        assert_eq!(node.rpc_flags.listen_port, 9545);
    }

    #[test]
    fn test_requires_kona_required_flags() {
        let err = parse(&["--rollup.consensus.enabled"]).unwrap_err();
        assert_eq!(err.kind(), ErrorKind::MissingRequiredArgument);
        assert!(err.to_string().contains("--rollup.consensus.l1-eth-rpc"), "{err}");
    }

    #[test]
    fn test_checks_kona_flag_relations() {
        let args: Vec<_> = REQUIRED
            .iter()
            .copied()
            .chain(["--rollup.consensus.p2p.signer.header", "a=b"])
            .collect();
        let err = parse(&args).unwrap_err();
        assert!(err.to_string().contains("--rollup.consensus.p2p.signer.endpoint"), "{err}");
    }

    #[test]
    fn test_rejects_flags_when_disabled() {
        let err = parse(&["--rollup.consensus.mode", "Sequencer"]).unwrap_err();
        assert_eq!(err.kind(), ErrorKind::MissingRequiredArgument);
        assert!(err.to_string().contains("--rollup.consensus.enabled"), "{err}");
    }

    #[test]
    fn test_rejects_unprefixed_flags() {
        let mut args = REQUIRED.to_vec();
        args.extend(["--mode", "Sequencer"]);
        assert_eq!(parse(&args).unwrap_err().kind(), ErrorKind::UnknownArgument);
    }
}
