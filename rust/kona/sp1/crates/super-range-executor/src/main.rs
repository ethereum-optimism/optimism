//! CLI entrypoint for the kona-sp1 super-range executor.

use std::{path::PathBuf, process::ExitCode};

use alloy_primitives::B256;
use clap::Parser;
use kona_sp1_client_utils::super_root::TimestampSpan;
use kona_sp1_super_range_executor::{
    ClaimSource, EXIT_INFRA, EXIT_INVALID, EXIT_VALID, Endpoints, RunConfig, Verdict, run,
};

/// Runs the kona-sp1 super-range program against live supernode data.
#[derive(Parser, Debug)]
#[command(
    name = "kona-sp1-super-range-executor",
    about = "Collect a super-range witness from the interop preimage server and execute the \
             kona-sp1 super-range ELF."
)]
struct Cli {
    /// Replay the collected witnesses through the shared native cores instead of executing the
    /// SP1 guest ELF. Like the guest, the replay panics on a preimage missing from the witness.
    #[arg(long)]
    native_core: bool,
    /// Corrupt the claimed optimistic output root the guest sees, so it rejects the claim.
    /// Witness collection still runs on the real claim; only the guest's view is tampered. Used
    /// to exercise the invalid-claim (soundness) path.
    #[arg(long)]
    corrupt_claimed_root: bool,
    /// Print the PGU, instruction count, stdin size, and cycle-tracker totals of each SP1 guest
    /// execution to stdout. Cycle-tracker totals need a build with `--features sp1-sdk/profiling`.
    #[arg(long, conflicts_with = "native_core")]
    report: bool,
    /// Supernode JSON-RPC endpoint.
    #[arg(long, env = "SUPER_NODE_ADDRESS", required_unless_present = "claim")]
    supernode_address: Option<String>,
    /// Trusted L1 head hash.
    #[arg(long, env, required_unless_present = "claim")]
    l1_head: Option<B256>,
    /// First superchain timestamp to prove. Defaults to `--end-timestamp`.
    #[arg(long)]
    start_timestamp: Option<u64>,
    /// Last superchain timestamp to prove.
    #[arg(long, env, required_unless_present = "claim")]
    end_timestamp: Option<u64>,
    /// Save the claim fetched from the supernode to this file once the run validates it, for a
    /// later `--claim` run against `--data-dir`.
    #[arg(long, requires = "data_dir")]
    save_claim: Option<PathBuf>,
    /// Prove a claim saved by `--save-claim` instead of fetching one from a supernode.
    #[arg(
        long,
        conflicts_with_all = ["supernode_address", "l1_head", "start_timestamp", "end_timestamp", "save_claim"],
    )]
    claim: Option<PathBuf>,
    /// L1 execution JSON-RPC endpoint. Omit it, with the beacon and L2 endpoints, to serve
    /// preimages only from `--data-dir`.
    #[arg(
        long,
        env,
        required_unless_present = "claim",
        requires_all = ["l1_beacon_address", "l2_node_addresses"],
    )]
    l1_node_address: Option<String>,
    /// L1 beacon API endpoint.
    #[arg(long, env, requires = "l1_node_address")]
    l1_beacon_address: Option<String>,
    /// L2 execution JSON-RPC endpoints, comma-separated.
    #[arg(long, value_delimiter = ',', env, requires = "l1_node_address")]
    l2_node_addresses: Vec<String>,
    /// Directory of the host's preimage store, kept after the run. Without endpoints, the host
    /// serves preimages only from it.
    #[arg(long, required_unless_present = "l1_node_address")]
    data_dir: Option<PathBuf>,
    /// Rollup config JSON paths, comma-separated.
    #[arg(long, alias = "rollup-cfgs", value_delimiter = ',', env)]
    rollup_config_paths: Option<Vec<PathBuf>>,
    /// L1 config JSON path.
    #[arg(long, alias = "l1-cfg", env)]
    l1_config_path: Option<PathBuf>,
    /// Dependency-set JSON path.
    #[arg(long, alias = "depset-cfg", env)]
    dependency_set_path: Option<PathBuf>,
}

fn main() -> ExitCode {
    init_tracing();
    let cli = Cli::parse();

    let runtime = match tokio::runtime::Builder::new_multi_thread().enable_all().build() {
        Ok(runtime) => runtime,
        Err(err) => return infra(format!("failed to build tokio runtime: {err}")),
    };

    let config = match cli.into_config() {
        Ok(config) => config,
        Err(err) => return infra(format!("invalid arguments: {err:?}")),
    };

    match runtime.block_on(run(config)) {
        Ok(Verdict::Valid) => ExitCode::from(EXIT_VALID),
        Ok(Verdict::Invalid) => ExitCode::from(EXIT_INVALID),
        Err(err) => infra(format!("failed to evaluate super-range claim: {err:?}")),
    }
}

impl Cli {
    fn into_config(self) -> anyhow::Result<RunConfig> {
        let claim = match self.claim {
            Some(path) => ClaimSource::File(path),
            None => {
                let end =
                    self.end_timestamp.expect("clap requires --end-timestamp without --claim");
                ClaimSource::Supernode {
                    address: self.supernode_address.expect("clap requires --supernode-address"),
                    l1_head: self.l1_head.expect("clap requires --l1-head"),
                    span: TimestampSpan::new(self.start_timestamp.unwrap_or(end), end)?,
                    save_to: self.save_claim,
                }
            }
        };
        let endpoints = self.l1_node_address.map(|l1_node_address| Endpoints {
            l1_node_address,
            l1_beacon_address: self.l1_beacon_address.expect("clap requires --l1-beacon-address"),
            l2_node_addresses: self.l2_node_addresses,
        });
        Ok(RunConfig {
            native_core: self.native_core,
            corrupt_claimed_root: self.corrupt_claimed_root,
            report: self.report,
            claim,
            endpoints,
            data_dir: self.data_dir,
            rollup_config_paths: self.rollup_config_paths,
            l1_config_path: self.l1_config_path,
            dependency_set_path: self.dependency_set_path,
        })
    }
}

fn infra(message: String) -> ExitCode {
    tracing::error!("{message}");
    eprintln!("{message}");
    ExitCode::from(EXIT_INFRA)
}

fn init_tracing() {
    let _ = tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()),
        )
        .try_init();
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::{CommandFactory, FromArgMatches};
    use kona_sp1_client_utils::super_root::SuperRootError;
    use std::path::Path;

    const L1_HEAD: &str = "0x0101010101010101010101010101010101010101010101010101010101010101";

    /// Parses `args` ignoring the environment, which clap counts as explicit arguments.
    fn parse(args: &[&str]) -> Result<Cli, clap::Error> {
        let matches = Cli::command()
            .mut_args(|arg| arg.env(None))
            .try_get_matches_from(std::iter::once("executor").chain(args.iter().copied()))?;
        Cli::from_arg_matches(&matches)
    }

    const ENDPOINTS: [&str; 3] = [
        "--l1-node-address=http://l1",
        "--l1-beacon-address=http://beacon",
        "--l2-node-addresses=http://a,http://b",
    ];

    fn supernode_args(timestamps: &[&str]) -> Vec<String> {
        let mut args =
            vec!["--supernode-address=http://supernode".into(), format!("--l1-head={L1_HEAD}")];
        args.extend(timestamps.iter().map(|arg| arg.to_string()));
        args.extend(ENDPOINTS.iter().map(|arg| arg.to_string()));
        args
    }

    fn parse_owned(args: &[String]) -> Result<Cli, clap::Error> {
        parse(&args.iter().map(String::as_str).collect::<Vec<_>>())
    }

    #[test]
    fn supernode_claim_spans_the_start_through_the_end() {
        let config = parse_owned(&supernode_args(&[
            "--start-timestamp=101",
            "--end-timestamp=110",
            "--save-claim=claim.json",
            "--data-dir=store",
        ]))
        .unwrap()
        .into_config()
        .unwrap();

        let ClaimSource::Supernode { span, save_to, .. } = config.claim else {
            panic!("expected a supernode claim");
        };
        assert_eq!(span, TimestampSpan::new(101, 110).unwrap());
        assert_eq!(save_to, Some(PathBuf::from("claim.json")));
        assert_eq!(config.endpoints.unwrap().l2_node_addresses, ["http://a", "http://b"]);
    }

    #[test]
    fn supernode_claim_defaults_to_the_end_timestamp_alone() {
        let config =
            parse_owned(&supernode_args(&["--end-timestamp=110"])).unwrap().into_config().unwrap();

        let ClaimSource::Supernode { span, .. } = config.claim else {
            panic!("expected a supernode claim");
        };
        assert_eq!(span, TimestampSpan::new(110, 110).unwrap());
    }

    #[test]
    fn saved_claim_runs_offline_from_the_data_dir() {
        let config =
            parse(&["--claim=claim.json", "--data-dir=store"]).unwrap().into_config().unwrap();

        let ClaimSource::File(path) = &config.claim else {
            panic!("expected a saved claim");
        };
        assert_eq!(path, Path::new("claim.json"));
        assert!(config.endpoints.is_none());
        assert_eq!(config.data_dir, Some(PathBuf::from("store")));
    }

    #[test]
    fn rejects_invalid_argument_combinations() {
        use clap::error::ErrorKind::{ArgumentConflict, MissingRequiredArgument};
        let cases = [
            (vec!["--claim=claim.json", "--data-dir=store", "--end-timestamp=1"], ArgumentConflict),
            (vec!["--claim=claim.json"], MissingRequiredArgument),
            (
                vec!["--claim=claim.json", "--data-dir=store", "--l1-node-address=http://l1"],
                MissingRequiredArgument,
            ),
        ];
        for (args, kind) in cases {
            assert_eq!(parse(&args).err().map(|err| err.kind()), Some(kind), "{args:?}");
        }
        let supernode_without_l1 = [
            "--supernode-address=http://supernode".to_string(),
            format!("--l1-head={L1_HEAD}"),
            "--end-timestamp=1".into(),
            "--data-dir=store".into(),
        ];
        assert_eq!(
            parse_owned(&supernode_without_l1).err().map(|err| err.kind()),
            Some(MissingRequiredArgument),
        );
        let save_without_data_dir = supernode_args(&["--end-timestamp=1", "--save-claim=c.json"]);
        assert_eq!(
            parse_owned(&save_without_data_dir).err().map(|err| err.kind()),
            Some(MissingRequiredArgument),
        );
    }

    #[test]
    fn rejects_a_start_after_the_end() {
        let err = parse_owned(&supernode_args(&["--start-timestamp=111", "--end-timestamp=110"]))
            .unwrap()
            .into_config()
            .unwrap_err();

        assert_eq!(
            err.downcast_ref::<SuperRootError>(),
            Some(&SuperRootError::InvalidTimestampSpan { start: 111, end: 110 }),
        );
    }
}
