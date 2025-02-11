use jsonrpsee::core::{async_trait, RpcResult};
use jsonrpsee::http_client::HttpClient;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee::server::Server;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::Mutex;

const DEFAULT_DEBUG_API_PORT: &str = "5555";

#[derive(Serialize, Deserialize, Debug)]
pub struct SetDryRunRequest1 {}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct SetDryRunResponse1 {
    pub dry_run_state: bool,
}

#[rpc(server, client, namespace = "debug")]
trait DebugApi {
    #[method(name = "setDryRun")]
    async fn set_dry_run(&self, request: SetDryRunRequest1) -> RpcResult<SetDryRunResponse1>;
}

pub struct DebugServer {
    dry_run: Arc<Mutex<bool>>,
}

impl DebugServer {
    pub fn new(dry_run: Arc<Mutex<bool>>) -> Self {
        Self { dry_run }
    }

    pub async fn run(self) -> eyre::Result<()> {
        let server = Server::builder()
            .build(format!("127.0.0.1:{}", DEFAULT_DEBUG_API_PORT))
            .await?;

        let handle = server.start(self.into_rpc());

        tracing::info!("Debug server started on port {}", DEFAULT_DEBUG_API_PORT);

        // In this example we don't care about doing shutdown so let's it run forever.
        // You may use the `ServerHandle` to shut it down or manage it yourself.
        tokio::spawn(handle.stopped());

        Ok(())
    }
}

#[async_trait]
impl DebugApiServer for DebugServer {
    async fn set_dry_run(&self, _request: SetDryRunRequest1) -> RpcResult<SetDryRunResponse1> {
        let mut dry_run = self.dry_run.lock().await;
        *dry_run = !*dry_run;

        Ok(SetDryRunResponse1 {
            dry_run_state: *dry_run,
        })
    }
}

pub struct DebugClient {
    client: HttpClient,
}

impl DebugClient {
    fn new(url: &str) -> eyre::Result<Self> {
        let client = HttpClient::builder().build(url)?;

        Ok(Self { client })
    }

    pub async fn toggle_dry_run(&self) -> eyre::Result<SetDryRunResponse1> {
        let request = SetDryRunRequest1 {};
        let result = DebugApiClient::set_dry_run(&self.client, request).await?;
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
        let dry_run = Arc::new(Mutex::new(false));

        let server = DebugServer::new(dry_run.clone());
        let _ = server.run().await.unwrap();

        let client = DebugClient::default();
        let result = client.toggle_dry_run().await.unwrap();

        assert_eq!(result.dry_run_state, true);
        assert_eq!(result.dry_run_state, *dry_run.lock().await);

        let result = client.toggle_dry_run().await.unwrap();
        assert_eq!(result.dry_run_state, false);
        assert_eq!(result.dry_run_state, *dry_run.lock().await);
    }
}
