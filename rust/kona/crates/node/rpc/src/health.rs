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
    /// The application version.
    version: String,
}

impl HealthzRpc {
    /// Constructs a healthz RPC server with the application version.
    pub const fn new(version: String) -> Self {
        Self { version }
    }
}

#[async_trait]
impl HealthzApiServer for HealthzRpc {
    async fn healthz(&self) -> RpcResult<HealthzResponse> {
        Ok(HealthzResponse { version: self.version.clone() })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn returns_configured_version() {
        for version in ["1.2.3-test", "0.0.0-dev"] {
            let rpc = HealthzRpc::new(version.to_owned());
            assert_eq!(rpc.healthz().await.unwrap().version, version);
        }
    }
}
