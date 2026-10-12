use alloy_rpc_types_engine::JwtSecret;
use anyhow::{Result, anyhow, bail};
use clap::ArgGroup;
use kona_node_service::DerivationDelegateConfig;
use std::path::PathBuf;
use url::Url;

const DEFAULT_L2_TRUST_RPC: bool = true;
const DEFAULT_L1_TRUST_RPC: bool = true;

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
///
/// The engine API's JWT secret comes from exactly one of `--l2-engine-jwt-secret` and
/// `--l2-engine-jwt-encoded`.
#[derive(Clone, Debug, clap::Args)]
#[command(group(
    ArgGroup::new("l2_engine_jwt")
        .required(true)
        .args(["l2_engine_jwt_secret", "l2_engine_jwt_encoded"])
))]
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
    /// If false, block hash verification is performed for all retrieved blocks.
    #[arg(
        long,
        visible_alias = "l2.trust-rpc",
        env = "KONA_NODE_L2_TRUST_RPC",
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
            l2_trust_rpc: DEFAULT_L2_TRUST_RPC,
        }
    }
}

impl L2ClientArgs {
    /// Returns the engine API's JWT secret, read from the `--l2-engine-jwt-secret` file or taken
    /// from `--l2-engine-jwt-encoded`.
    pub fn jwt_secret(&self) -> Result<JwtSecret> {
        match (&self.l2_engine_jwt_secret, self.l2_engine_jwt_encoded) {
            (Some(path), None) => {
                let hex = std::fs::read_to_string(path)
                    .map_err(|e| anyhow!("Failed to read the JWT secret file {path:?}: {e}"))?;
                JwtSecret::from_hex(hex)
                    .map_err(|e| anyhow!("Failed to parse the JWT secret in {path:?}: {e}"))
            }
            (None, Some(secret)) => Ok(secret),
            _ => bail!(
                "Exactly one of --l2-engine-jwt-secret and --l2-engine-jwt-encoded must be set"
            ),
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
    use clap::{Parser, error::ErrorKind};
    use std::io::Write;

    const JWT_HEX: &str = "0xabababababababababababababababababababababababababababababababab";

    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        l2: L2ClientArgs,
    }

    fn parse(args: &[&str]) -> Result<L2ClientArgs, clap::Error> {
        TestCli::try_parse_from(
            ["test", "--l2-engine-rpc", "http://localhost:8551"].iter().chain(args),
        )
        .map(|cli| cli.l2)
    }

    #[test]
    fn jwt_secret_from_encoded_flag() {
        let args = parse(&["--l2-engine-jwt-encoded", JWT_HEX]).unwrap();
        assert_eq!(args.jwt_secret().unwrap(), JwtSecret::from_hex(JWT_HEX).unwrap());
    }

    #[test]
    fn jwt_secret_from_file() {
        let mut file = tempfile::NamedTempFile::new().unwrap();
        writeln!(file, "{JWT_HEX}").unwrap();

        let args = parse(&["--l2-engine-jwt-secret", file.path().to_str().unwrap()]).unwrap();
        assert_eq!(args.jwt_secret().unwrap(), JwtSecret::from_hex(JWT_HEX).unwrap());
    }

    #[test]
    fn jwt_secret_errors_on_a_missing_file() {
        let args = parse(&["--l2-engine-jwt-secret", "/nonexistent/jwt.hex"]).unwrap();
        assert!(args.jwt_secret().is_err());
    }

    #[test]
    fn jwt_secret_source_required() {
        assert_eq!(parse(&[]).unwrap_err().kind(), ErrorKind::MissingRequiredArgument);
    }

    #[test]
    fn jwt_secret_sources_conflict() {
        let err = parse(&["--l2-engine-jwt-encoded", JWT_HEX, "--l2-engine-jwt-secret", "jwt.hex"])
            .unwrap_err();
        assert_eq!(err.kind(), ErrorKind::ArgumentConflict);
    }
}
