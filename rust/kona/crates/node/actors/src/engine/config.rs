use crate::NodeMode;
use alloy_rpc_types_engine::JwtSecret;
use url::Url;

/// How the node reaches the L2 execution layer's Engine API, and the mode the engine runs in.
#[derive(Debug, Clone)]
pub struct EngineConfig {
    /// The engine rpc url.
    pub l2_url: Url,
    /// The engine jwt secret.
    pub l2_jwt_secret: JwtSecret,

    /// The mode of operation for the node.
    /// When the node is in sequencer mode, the engine actor will receive requests to build blocks
    /// from the sequencer actor.
    pub mode: NodeMode,
}
