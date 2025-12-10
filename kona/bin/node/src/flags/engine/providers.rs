use alloy_rpc_types_engine::JwtSecret;
use std::path::PathBuf;
use url::Url;

const DEFAULT_BUILDER_TIMEOUT: u64 = 30;
const DEFAULT_L2_ENGINE_TIMEOUT: u64 = 30_000;

const DEFAULT_L2_TRUST_RPC: bool = true;
const DEFAULT_L1_TRUST_RPC: bool = true;

/// Rollup-boost builder client arguments.
#[derive(Clone, Debug, clap::Args)]
pub struct BuilderClientArgs {
    /// URL of the builder RPC API.
    #[arg(
        long,
        visible_alias = "builder",
        env = "KONA_NODE_BUILDER_RPC",
        default_value = "http://localhost:8552"
    )]
    pub l2_builder_rpc: Url,
    /// Hex encoded JWT secret to use for the authenticated builder RPC server.
    #[arg(long, visible_alias = "builder.auth", env = "KONA_NODE_BUILDER_AUTH")]
    pub builder_jwt_secret: Option<JwtSecret>,
    /// Path to a JWT secret to use for the authenticated builder RPC server.
    #[arg(long, visible_alias = "builder.jwt-path", env = "KONA_NODE_BUILDER_JWT_PATH")]
    pub builder_jwt_path: Option<PathBuf>,
    /// Timeout for http calls in milliseconds.
    #[arg(
        long,
        visible_alias = "builder.timeout",
        env = "KONA_NODE_BUILDER_TIMEOUT",
        default_value_t = DEFAULT_BUILDER_TIMEOUT
    )]
    pub builder_timeout: u64,
}

impl Default for BuilderClientArgs {
    fn default() -> Self {
        Self {
            l2_builder_rpc: Url::parse("http://localhost:8552").unwrap(),
            builder_jwt_secret: None,
            builder_jwt_path: None,
            builder_timeout: DEFAULT_BUILDER_TIMEOUT,
        }
    }
}

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
    /// This MUST be a valid path to a file containing the hex-encoded JWT secret.
    #[arg(long, visible_alias = "l2.jwt-path", env = "KONA_NODE_L2_ENGINE_JWT_PATH")]
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
