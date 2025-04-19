use jsonrpsee::core::{RpcResult, async_trait};
use jsonrpsee::http_client::HttpClient;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee::server::Server;
use parking_lot::Mutex;
use serde::{Deserialize, Serialize};
use std::sync::Arc;

use crate::server::ExecutionMode;

#[derive(Serialize, Deserialize, Debug)]
pub struct SetExecutionModeRequest {
    pub execution_mode: ExecutionMode,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct SetExecutionModeResponse {
    pub execution_mode: ExecutionMode,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct GetExecutionModeResponse {
    pub execution_mode: ExecutionMode,
}

#[rpc(server, client, namespace = "debug")]
trait DebugApi {
    #[method(name = "setExecutionMode")]
    async fn set_execution_mode(
        &self,
        request: SetExecutionModeRequest,
    ) -> RpcResult<SetExecutionModeResponse>;

    #[method(name = "getExecutionMode")]
    async fn get_execution_mode(&self) -> RpcResult<GetExecutionModeResponse>;
}

pub struct DebugServer {
    execution_mode: Arc<Mutex<ExecutionMode>>,
}

impl DebugServer {
    pub fn new(execution_mode: Arc<Mutex<ExecutionMode>>) -> Self {
        Self { execution_mode }
    }

    pub async fn run(self, debug_addr: &str) -> eyre::Result<()> {
        let server = Server::builder().build(debug_addr).await?;

        let handle = server.start(self.into_rpc());

        tracing::info!("Debug server listening on addr {}", debug_addr);

        // In this example we don't care about doing shutdown so let's it run forever.
        // You may use the `ServerHandle` to shut it down or manage it yourself.
        tokio::spawn(handle.stopped());

        Ok(())
    }

    pub fn execution_mode(&self) -> ExecutionMode {
        *self.execution_mode.lock()
    }

    pub fn set_execution_mode(&self, mode: ExecutionMode) {
        *self.execution_mode.lock() = mode;
    }
}

#[async_trait]
impl DebugApiServer for DebugServer {
    async fn set_execution_mode(
        &self,
        request: SetExecutionModeRequest,
    ) -> RpcResult<SetExecutionModeResponse> {
        self.set_execution_mode(request.execution_mode);

        tracing::info!("Set execution mode to {:?}", request.execution_mode);

        Ok(SetExecutionModeResponse {
            execution_mode: request.execution_mode,
        })
    }

    async fn get_execution_mode(&self) -> RpcResult<GetExecutionModeResponse> {
        Ok(GetExecutionModeResponse {
            execution_mode: self.execution_mode(),
        })
    }
}

pub struct DebugClient {
    client: HttpClient,
}

impl DebugClient {
    pub fn new(url: &str) -> eyre::Result<Self> {
        let client = HttpClient::builder().build(url)?;

        Ok(Self { client })
    }

    pub async fn set_execution_mode(
        &self,
        execution_mode: ExecutionMode,
    ) -> eyre::Result<SetExecutionModeResponse> {
        let request = SetExecutionModeRequest { execution_mode };
        let result = DebugApiClient::set_execution_mode(&self.client, request).await?;
        Ok(result)
    }

    pub async fn get_execution_mode(&self) -> eyre::Result<GetExecutionModeResponse> {
        let result = DebugApiClient::get_execution_mode(&self.client).await?;
        Ok(result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const DEFAULT_ADDR: &str = "127.0.0.1:5555";

    #[tokio::test]
    async fn test_debug_client() {
        // spawn the server and try to modify it with the client
        let execution_mode = Arc::new(Mutex::new(ExecutionMode::Enabled));

        let server = DebugServer::new(execution_mode.clone());
        server.run(DEFAULT_ADDR).await.unwrap();

        let client = DebugClient::new(format!("http://{}", DEFAULT_ADDR).as_str()).unwrap();

        // Test setting execution mode to Disabled
        let result = client
            .set_execution_mode(ExecutionMode::Disabled)
            .await
            .unwrap();
        assert_eq!(result.execution_mode, ExecutionMode::Disabled);

        // Verify with get_execution_mode
        let status = client.get_execution_mode().await.unwrap();
        assert_eq!(status.execution_mode, ExecutionMode::Disabled);

        // Test setting execution mode back to Enabled
        let result = client
            .set_execution_mode(ExecutionMode::Enabled)
            .await
            .unwrap();
        assert_eq!(result.execution_mode, ExecutionMode::Enabled);

        // Verify again with get_execution_mode
        let status = client.get_execution_mode().await.unwrap();
        assert_eq!(status.execution_mode, ExecutionMode::Enabled);

        // Test setting fallback execution mode
        let result = client
            .set_execution_mode(ExecutionMode::Fallback)
            .await
            .unwrap();

        assert_eq!(result.execution_mode, ExecutionMode::Fallback);

        // Verify again with get_execution_mode
        let status = client.get_execution_mode().await.unwrap();
        assert_eq!(status.execution_mode, ExecutionMode::Fallback);
    }
}
