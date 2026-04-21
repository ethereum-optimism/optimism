use async_trait::async_trait;
use jsonrpsee::core::RpcResult;

use crate::jsonrpsee::HealthzApiServer;

/// A healthcheck response for the RPC server.
#[derive(Debug, Clone, serde::Deserialize, serde::Serialize)]
pub struct HealthzResponse {
    /// The application version.
    pub version: String,
}

/// The healthz rpc server.
#[derive(Debug, Clone)]
pub struct HealthzRpc {}

#[async_trait]
impl HealthzApiServer for HealthzRpc {
    async fn healthz(&self) -> RpcResult<HealthzResponse> {
        Ok(HealthzResponse { version: env!("CARGO_PKG_VERSION").to_string() })
    }
}
