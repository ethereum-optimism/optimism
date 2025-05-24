//! Client support for optimism historical RPC requests.

use crate::sequencer::Error;
use alloy_json_rpc::{RpcRecv, RpcSend};
use alloy_rpc_client::RpcClient;
use std::sync::Arc;
use tracing::warn;

/// A client that can be used to forward RPC requests for historical data to an endpoint.
///
/// This is intended to be used for OP-Mainnet pre-bedrock data, allowing users to query historical
/// state.
#[derive(Debug, Clone)]
pub struct HistoricalRpcClient {
    inner: Arc<HistoricalRpcClientInner>,
}

impl HistoricalRpcClient {
    /// Constructs a new historical RPC client with the given endpoint URL.
    pub async fn new(endpoint: &str) -> Result<Self, Error> {
        let client = RpcClient::new_http(
            endpoint.parse::<reqwest::Url>().map_err(|err| Error::InvalidUrl(err.to_string()))?,
        );

        Ok(Self {
            inner: Arc::new(HistoricalRpcClientInner {
                historical_endpoint: endpoint.to_string(),
                client,
            }),
        })
    }

    /// Returns a reference to the underlying RPC client
    fn client(&self) -> &RpcClient {
        &self.inner.client
    }

    /// Forwards a JSON-RPC request to the historical endpoint
    pub async fn request<Params: RpcSend, Resp: RpcRecv>(
        &self,
        method: &str,
        params: Params,
    ) -> Result<Resp, Error> {
        let resp =
            self.client().request::<Params, Resp>(method.to_string(), params).await.inspect_err(
                |err| {
                    warn!(
                        target: "rpc::historical",
                        %err,
                        "HTTP request to historical endpoint failed"
                    );
                },
            )?;

        Ok(resp)
    }

    /// Returns the configured historical endpoint URL
    pub fn endpoint(&self) -> &str {
        &self.inner.historical_endpoint
    }
}

#[derive(Debug)]
struct HistoricalRpcClientInner {
    historical_endpoint: String,
    client: RpcClient,
}
