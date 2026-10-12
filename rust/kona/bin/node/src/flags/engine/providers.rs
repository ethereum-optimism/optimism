use alloy_rpc_types_engine::JwtSecret;
use anyhow::{Result, anyhow};
use kona_node_service::DerivationDelegateConfig;
use std::path::PathBuf;
use url::Url;

/// L1 client arguments.
#[derive(Clone, Debug, clap::Args)]
pub struct L1ClientArgs {
    /// URL of the L1 execution client RPC API.
    #[arg(long, visible_alias = "l1", env = "KONA_NODE_L1_ETH_RPC")]
    pub l1_eth_rpc: Url,
    /// URL of the L1 beacon API.
    #[arg(long, visible_alias = "l1.beacon", env = "KONA_NODE_L1_BEACON")]
    pub l1_beacon: Url,
}

/// L2 client arguments.
#[derive(Clone, Debug, clap::Args)]
pub struct L2ClientArgs {
    /// URI of the engine API endpoint of an L2 execution client.
    #[arg(long, visible_alias = "l2", env = "KONA_NODE_L2_ENGINE_RPC")]
    pub l2_engine_rpc: Url,
    /// Path to a file containing the hex-encoded JWT secret for the execution client's
    /// authenticated engine API. The secret is only accepted from a file, so that it stays out of
    /// the process's command line.
    #[arg(long, visible_alias = "l2.jwt-secret", env = "KONA_NODE_L2_ENGINE_AUTH")]
    pub l2_engine_jwt_secret: PathBuf,
}

impl L2ClientArgs {
    /// Returns the engine API's JWT secret, read from the `--l2-engine-jwt-secret` file.
    pub fn jwt_secret(&self) -> Result<JwtSecret> {
        let path = &self.l2_engine_jwt_secret;
        let hex = std::fs::read_to_string(path)
            .map_err(|e| anyhow!("Failed to read the JWT secret file {path:?}: {e}"))?;
        JwtSecret::from_hex(hex)
            .map_err(|e| anyhow!("Failed to parse the JWT secret in {path:?}: {e}"))
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
    fn jwt_secret_file_required() {
        assert_eq!(parse(&[]).unwrap_err().kind(), ErrorKind::MissingRequiredArgument);
    }

    #[test]
    fn jwt_secret_not_accepted_on_the_command_line() {
        let err = parse(&["--l2-engine-jwt-secret", "jwt.hex", "--l2-engine-jwt-encoded", JWT_HEX])
            .unwrap_err();
        assert_eq!(err.kind(), ErrorKind::UnknownArgument);
    }
}
