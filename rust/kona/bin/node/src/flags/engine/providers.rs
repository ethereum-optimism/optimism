use alloy_rpc_types_engine::JwtSecret;
use kona_node_service::DerivationDelegateConfig;
use std::path::PathBuf;
use url::Url;

const DEFAULT_L2_ENGINE_TIMEOUT: u64 = 30_000;

const DEFAULT_L2_TRUST_RPC: bool = true;
const DEFAULT_L1_TRUST_RPC: bool = false;

/// L1 client arguments.
#[derive(Clone, Debug, clap::Args)]
pub struct L1ClientArgs {
    /// URL of the L1 execution client RPC API.
    #[arg(long, visible_alias = "l1", env = "KONA_NODE_L1_ETH_RPC")]
    pub l1_eth_rpc: Url,
    /// Whether to trust the L1 RPC.
    /// If false, block hash verification is performed for all retrieved blocks.
    #[arg(
        long,
        visible_alias = "l1.trust-rpc",
        env = "KONA_NODE_L1_TRUST_RPC",
        action = clap::ArgAction::Set,
        num_args = 0..=1,
        default_missing_value = "true",
        default_value_t = DEFAULT_L1_TRUST_RPC
    )]
    pub l1_trust_rpc: bool,
    /// URL of the L1 beacon API.
    #[arg(long, visible_alias = "l1.beacon", env = "KONA_NODE_L1_BEACON")]
    pub l1_beacon: Url,
    /// Duration in seconds of an L1 slot.
    ///
    /// This is an optional argument that can be used to use a fixed slot duration for l1 blocks
    /// and bypass the initial beacon spec fetch. This is useful for testing purposes when the
    /// l1-beacon spec endpoint is not available (with anvil for example).
    #[arg(
        long,
        visible_alias = "l1.slot-duration-override",
        env = "KONA_NODE_L1_SLOT_DURATION_OVERRIDE"
    )]
    pub l1_slot_duration_override: Option<u64>,
}

impl Default for L1ClientArgs {
    fn default() -> Self {
        Self {
            l1_eth_rpc: Url::parse("http://localhost:8545").unwrap(),
            l1_trust_rpc: DEFAULT_L1_TRUST_RPC,
            l1_beacon: Url::parse("http://localhost:5052").unwrap(),
            l1_slot_duration_override: None,
        }
    }
}

/// L2 client arguments.
#[derive(Clone, Debug, clap::Args)]
pub struct L2ClientArgs {
    /// URI of the engine API endpoint of an L2 execution client.
    #[arg(long, visible_alias = "l2", env = "KONA_NODE_L2_ENGINE_RPC")]
    pub l2_engine_rpc: Url,
    /// JWT secret for the auth-rpc endpoint of the execution client.
    /// This MUST be a valid path to a file containing the hex-encoded JWT secret.
    #[arg(long, visible_alias = "l2.jwt-secret", env = "KONA_NODE_L2_ENGINE_AUTH")]
    pub l2_engine_jwt_secret: Option<PathBuf>,
    /// Hex encoded JWT secret to use for the authenticated engine-API RPC server.
    /// This MUST be a valid hex-encoded JWT secret of 64 digits.
    #[arg(long, visible_alias = "l2.jwt-secret-encoded", env = "KONA_NODE_L2_ENGINE_AUTH_ENCODED")]
    pub l2_engine_jwt_encoded: Option<JwtSecret>,
    /// Timeout for http calls in milliseconds.
    #[arg(
        long,
        visible_alias = "l2.timeout",
        env = "KONA_NODE_L2_ENGINE_TIMEOUT",
        default_value_t = DEFAULT_L2_ENGINE_TIMEOUT
    )]
    pub l2_engine_timeout: u64,
    /// If false, block hash verification is performed for all retrieved blocks.
    #[arg(
        long,
        visible_alias = "l2.trust-rpc",
        env = "KONA_NODE_L2_TRUST_RPC",
        action = clap::ArgAction::Set,
        num_args = 0..=1,
        default_missing_value = "true",
        default_value_t = DEFAULT_L2_TRUST_RPC
    )]
    pub l2_trust_rpc: bool,
}

impl Default for L2ClientArgs {
    fn default() -> Self {
        Self {
            l2_engine_rpc: Url::parse("http://localhost:8551").unwrap(),
            l2_engine_jwt_secret: None,
            l2_engine_jwt_encoded: None,
            l2_engine_timeout: DEFAULT_L2_ENGINE_TIMEOUT,
            l2_trust_rpc: DEFAULT_L2_TRUST_RPC,
        }
    }
}

/// L2 derivation delegate connection arguments.
#[derive(Clone, Debug, Default, clap::Args)]
pub struct DerivationDelegateArgs {
    /// The source must be an OP Stack L2 CL RPC exposing `optimism_syncStatus`.
    #[arg(long, visible_alias = "l2.follow.source", env = "KONA_NODE_L2_FOLLOW_SOURCE")]
    pub l2_follow_source: Option<Url>,
}

impl DerivationDelegateArgs {
    /// Builds the derivation delegate configuration if an L2 CL URL was provided.
    pub fn config(self) -> Option<DerivationDelegateConfig> {
        self.l2_follow_source.map(|url| DerivationDelegateConfig { l2_cl_url: url })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::{CommandFactory, FromArgMatches, Parser};
    use rstest::rstest;

    const CLIENT_ARGS: &[&str] = &[
        "kona-node",
        "--l1=http://localhost:8545",
        "--l1.beacon=http://localhost:5052",
        "--l2=http://localhost:8551",
    ];

    #[derive(Debug, Parser)]
    struct ClientArgs {
        #[command(flatten)]
        l1: L1ClientArgs,
        #[command(flatten)]
        l2: L2ClientArgs,
    }

    #[rstest]
    #[case::defaults(&[], false, true)]
    #[case::l1_false(&["--l1-trust-rpc=false"], false, true)]
    #[case::l1_false_space(&["--l1-trust-rpc", "false"], false, true)]
    #[case::l1_true(&["--l1-trust-rpc=true"], true, true)]
    #[case::l1_true_space(&["--l1-trust-rpc", "true"], true, true)]
    #[case::l1_bare(&["--l1-trust-rpc"], true, true)]
    #[case::l1_alias_false(&["--l1.trust-rpc=false"], false, true)]
    #[case::l1_alias_false_space(&["--l1.trust-rpc", "false"], false, true)]
    #[case::l1_alias_true(&["--l1.trust-rpc=true"], true, true)]
    #[case::l1_alias_true_space(&["--l1.trust-rpc", "true"], true, true)]
    #[case::l1_alias_bare(&["--l1.trust-rpc"], true, true)]
    #[case::l2_false(&["--l2-trust-rpc=false"], false, false)]
    #[case::l2_false_space(&["--l2-trust-rpc", "false"], false, false)]
    #[case::l2_true(&["--l2-trust-rpc=true"], false, true)]
    #[case::l2_true_space(&["--l2-trust-rpc", "true"], false, true)]
    #[case::l2_bare(&["--l2-trust-rpc"], false, true)]
    #[case::l2_alias_false(&["--l2.trust-rpc=false"], false, false)]
    #[case::l2_alias_false_space(&["--l2.trust-rpc", "false"], false, false)]
    #[case::l2_alias_true(&["--l2.trust-rpc=true"], false, true)]
    #[case::l2_alias_true_space(&["--l2.trust-rpc", "true"], false, true)]
    #[case::l2_alias_bare(&["--l2.trust-rpc"], false, true)]
    #[case::both_false(&["--l1.trust-rpc=false", "--l2.trust-rpc=false"], false, false)]
    #[case::both_false_space(&["--l1.trust-rpc", "false", "--l2.trust-rpc", "false"], false, false)]
    #[case::both_bare(&["--l1.trust-rpc", "--l2.trust-rpc"], true, true)]
    fn test_parse_trust_rpc_args(#[case] flags: &[&str], #[case] l1: bool, #[case] l2: bool) {
        // Keep these CLI/default tests independent of the operator's environment.
        let command = ClientArgs::command()
            .mut_arg("l1_trust_rpc", |arg| arg.env(None::<&str>))
            .mut_arg("l2_trust_rpc", |arg| arg.env(None::<&str>));
        let args = [CLIENT_ARGS, flags].concat();
        let matches = command.try_get_matches_from(args).unwrap();
        let parsed = ClientArgs::from_arg_matches(&matches).unwrap();
        assert_eq!(parsed.l1.l1_trust_rpc, l1);
        assert_eq!(parsed.l2.l2_trust_rpc, l2);
    }

    #[test]
    fn test_default_rpc_trust() {
        assert!(!L1ClientArgs::default().l1_trust_rpc);
        assert!(L2ClientArgs::default().l2_trust_rpc);
    }

    #[test]
    fn test_trust_rpc_environment() {
        // Re-run this test in subprocesses so environment changes cannot race other tests.
        if let Ok(expected) = std::env::var("KONA_TEST_TRUST_RPC_EXPECTED") {
            let flags = std::env::var("KONA_TEST_TRUST_RPC_FLAGS").unwrap();
            let parsed =
                ClientArgs::parse_from(CLIENT_ARGS.iter().copied().chain(flags.split_whitespace()));
            let expected = expected.parse::<bool>().unwrap();
            assert_eq!(parsed.l1.l1_trust_rpc, expected);
            assert_eq!(parsed.l2.l2_trust_rpc, expected);
            return;
        }

        for env_value in ["true", "false"] {
            for cli_value in ["env", "true", "false", "bare"] {
                let flags = match cli_value {
                    "env" => String::new(),
                    "bare" => "--l1.trust-rpc --l2.trust-rpc".to_string(),
                    value => format!("--l1.trust-rpc={value} --l2.trust-rpc={value}"),
                };
                let expected = match cli_value {
                    "env" => env_value,
                    "bare" => "true",
                    value => value,
                };
                let output = std::process::Command::new(std::env::current_exe().unwrap())
                    .args(["--exact", std::thread::current().name().unwrap(), "--nocapture"])
                    .env("KONA_NODE_L1_TRUST_RPC", env_value)
                    .env("KONA_NODE_L2_TRUST_RPC", env_value)
                    .env("KONA_TEST_TRUST_RPC_EXPECTED", expected)
                    .env("KONA_TEST_TRUST_RPC_FLAGS", flags)
                    .output()
                    .unwrap();
                assert!(output.status.success(), "{output:?}");
                assert!(String::from_utf8_lossy(&output.stdout).contains("1 passed"), "{output:?}");
            }
        }
    }

    #[rstest]
    #[case::l1("--l1.trust-rpc=invalid")]
    #[case::l2("--l2.trust-rpc=invalid")]
    fn test_reject_invalid_trust_rpc(#[case] flag: &str) {
        let err = ClientArgs::try_parse_from(["kona-node", flag]).unwrap_err();
        assert_eq!(err.kind(), clap::error::ErrorKind::InvalidValue);
    }
}
