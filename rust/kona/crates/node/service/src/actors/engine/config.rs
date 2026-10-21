use crate::NodeMode;
use alloy_rpc_types_engine::JwtSecret;
use std::time::Duration;
use url::Url;

/// Inputs for constructing the node's shared Engine API connection.
#[derive(Debug, Clone)]
pub struct EngineConfig {
    /// The engine RPC URL.
    pub l2_url: Url,
    /// The engine JWT secret.
    pub l2_jwt_secret: JwtSecret,
    /// Deadline for each complete L2 RPC response.
    pub timeout: Duration,
    /// The node's mode of operation.
    pub mode: NodeMode,
}
