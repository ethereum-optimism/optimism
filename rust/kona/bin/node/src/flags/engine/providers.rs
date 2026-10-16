use alloy_rpc_types_engine::JwtSecret;
use anyhow::{Result, anyhow};
use std::path::PathBuf;
use url::Url;

/// L1 client arguments.
#[derive(Clone, Debug, clap::Args)]
#[command(next_help_heading = "L1")]
pub struct L1ClientArgs {
    /// URL of the L1 execution client's RPC API.
    #[arg(long = "l1.eth-url", env = "KONA_NODE_L1_ETH_URL", value_name = "URL")]
    pub eth_url: Url,
    /// URL of the L1 beacon API.
    #[arg(long = "l1.beacon-url", env = "KONA_NODE_L1_BEACON_URL", value_name = "URL")]
    pub beacon_url: Url,
}

/// L2 client arguments.
#[derive(Clone, Debug, clap::Args)]
#[command(next_help_heading = "L2 Engine")]
pub struct L2ClientArgs {
    /// URL of the L2 execution client's engine API.
    #[arg(long = "l2.engine.url", env = "KONA_NODE_L2_ENGINE_URL", value_name = "URL")]
    pub engine_url: Url,
    /// Path to a file containing the hex-encoded JWT secret for the execution client's
    /// authenticated engine API. The secret is only accepted from a file, so that it stays out of
    /// the process's command line.
    #[arg(
        long = "l2.engine.jwt-secret-path",
        env = "KONA_NODE_L2_ENGINE_JWT_SECRET_PATH",
        value_name = "PATH"
    )]
    pub engine_jwt_secret: PathBuf,
}

impl L2ClientArgs {
    /// Returns the engine API's JWT secret, read from the `--l2.engine.jwt-secret-path` file.
    pub fn jwt_secret(&self) -> Result<JwtSecret> {
        let path = &self.engine_jwt_secret;
        let hex = std::fs::read_to_string(path)
            .map_err(|e| anyhow!("Failed to read the JWT secret file {path:?}: {e}"))?;
        JwtSecret::from_hex(hex)
            .map_err(|e| anyhow!("Failed to parse the JWT secret in {path:?}: {e}"))
    }
}

/// L2 derivation delegate connection arguments.
#[derive(Clone, Debug, Default, clap::Args)]
#[command(next_help_heading = "Derivation Delegate")]
pub struct DerivationDelegateArgs {
    /// URL of an OP Stack L2 consensus-layer node to follow, instead of deriving the chain from
    /// L1. Its RPC must serve `optimism_syncStatus`.
    #[arg(
        long = "derivation-trusted-l2-cl-url",
        env = "KONA_NODE_DERIVATION_TRUSTED_L2_CL_URL",
        value_name = "URL"
    )]
    pub cl_url: Option<Url>,
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
            ["test", "--l2.engine.url", "http://localhost:8551"].iter().chain(args),
        )
        .map(|cli| cli.l2)
    }

    #[test]
    fn jwt_secret_from_file() {
        let mut file = tempfile::NamedTempFile::new().unwrap();
        writeln!(file, "{JWT_HEX}").unwrap();

        let args = parse(&["--l2.engine.jwt-secret-path", file.path().to_str().unwrap()]).unwrap();
        assert_eq!(args.jwt_secret().unwrap(), JwtSecret::from_hex(JWT_HEX).unwrap());
    }

    #[test]
    fn jwt_secret_errors_on_a_missing_file() {
        let args = parse(&["--l2.engine.jwt-secret-path", "/nonexistent/jwt.hex"]).unwrap();
        assert!(args.jwt_secret().is_err());
    }

    #[test]
    fn jwt_secret_file_required() {
        assert_eq!(parse(&[]).unwrap_err().kind(), ErrorKind::MissingRequiredArgument);
    }

    #[test]
    fn jwt_secret_not_accepted_on_the_command_line() {
        let err =
            parse(&["--l2.engine.jwt-secret-path", "jwt.hex", "--l2-engine-jwt-encoded", JWT_HEX])
                .unwrap_err();
        assert_eq!(err.kind(), ErrorKind::UnknownArgument);
    }
}
