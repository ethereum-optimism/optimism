use jsonrpsee::core::{async_trait, RpcResult};
use jsonrpsee::http_client::HttpClient;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee::server::Server;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::Mutex;

use crate::server::ExecutionMode;

const DEFAULT_DEBUG_API_PORT: u16 = 5555;

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

    pub async fn run(self, port: Option<u16>) -> eyre::Result<()> {
        let port = port.unwrap_or(DEFAULT_DEBUG_API_PORT);

        let server = Server::builder()
            .build(format!("127.0.0.1:{}", port))
            .await?;

        let handle = server.start(self.into_rpc());

        tracing::info!("Debug server started on port {}", port);

        // In this example we don't care about doing shutdown so let's it run forever.
        // You may use the `ServerHandle` to shut it down or manage it yourself.
        tokio::spawn(handle.stopped());

        Ok(())
    }
}

#[async_trait]
impl DebugApiServer for DebugServer {
    async fn set_execution_mode(
        &self,
        request: SetExecutionModeRequest,
    ) -> RpcResult<SetExecutionModeResponse> {
        let mut execution_mode = self.execution_mode.lock().await;
        *execution_mode = request.execution_mode.clone();

        tracing::info!("Set execution mode to {:?}", request.execution_mode);

        Ok(SetExecutionModeResponse {
            execution_mode: request.execution_mode,
        })
    }

    async fn get_execution_mode(&self) -> RpcResult<GetExecutionModeResponse> {
        let execution_mode = self.execution_mode.lock().await;
        Ok(GetExecutionModeResponse {
            execution_mode: execution_mode.clone(),
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

impl Default for DebugClient {
    fn default() -> Self {
        Self::new(format!("http://localhost:{}", DEFAULT_DEBUG_API_PORT).as_str()).unwrap()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_debug_client() {
        // spawn the server and try to modify it with the client
        let execution_mode = Arc::new(Mutex::new(ExecutionMode::Enabled));

        let server = DebugServer::new(execution_mode.clone());
        let _ = server.run(None).await.unwrap();

        let client = DebugClient::default();

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
    }
}
