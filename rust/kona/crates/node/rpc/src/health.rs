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
pub struct HealthzRpc {
    version: &'static str,
}

impl HealthzRpc {
    /// Uses the running binary's release version.
    pub const fn new(version: &'static str) -> Self {
        Self { version }
    }
}

#[async_trait]
impl HealthzApiServer for HealthzRpc {
    async fn healthz(&self) -> RpcResult<HealthzResponse> {
        Ok(HealthzResponse { version: self.version.to_string() })
    }
}
