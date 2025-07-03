//! Client support for optimism historical RPC requests.

use crate::sequencer::Error;
use alloy_eips::BlockId;
use alloy_json_rpc::{RpcRecv, RpcSend};
use alloy_primitives::BlockNumber;
use alloy_rpc_client::RpcClient;
use jsonrpsee_core::{
    middleware::{Batch, Notification, RpcServiceT},
    server::MethodResponse,
};
use jsonrpsee_types::{Params, Request};
use reth_storage_api::BlockReaderIdExt;
use std::{future::Future, sync::Arc};
use tracing::{debug, warn};

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
    pub fn new(endpoint: &str) -> Result<Self, Error> {
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

/// A layer that provides historical RPC forwarding functionality for a given service.
#[derive(Debug, Clone)]
pub struct HistoricalRpc<P> {
    inner: Arc<HistoricalRpcInner<P>>,
}

impl<P> HistoricalRpc<P> {
    /// Constructs a new historical RPC layer with the given provider, client and bedrock block
    /// number.
    pub fn new(provider: P, client: HistoricalRpcClient, bedrock_block: BlockNumber) -> Self {
        let inner = Arc::new(HistoricalRpcInner { provider, client, bedrock_block });

        Self { inner }
    }
}

impl<S, P> tower::Layer<S> for HistoricalRpc<P> {
    type Service = HistoricalRpcService<S, P>;

    fn layer(&self, inner: S) -> Self::Service {
        HistoricalRpcService::new(inner, self.inner.clone())
    }
}

/// A service that intercepts RPC calls and forwards pre-bedrock historical requests
/// to a dedicated endpoint.
///
/// This checks if the request is for a pre-bedrock block and forwards it via the configured
/// historical RPC client.
#[derive(Debug, Clone)]
pub struct HistoricalRpcService<S, P> {
    /// The inner service that handles regular RPC requests
    inner: S,
    /// The context required to forward historical requests.
    historical: Arc<HistoricalRpcInner<P>>,
}

impl<S, P> HistoricalRpcService<S, P> {
    /// Constructs a new historical RPC service with the given inner service, historical client,
    /// provider, and bedrock block number.
    const fn new(inner: S, historical: Arc<HistoricalRpcInner<P>>) -> Self {
        Self { inner, historical }
    }
}

impl<S, P> RpcServiceT for HistoricalRpcService<S, P>
where
    S: RpcServiceT<MethodResponse = MethodResponse> + Send + Sync + Clone + 'static,

    P: BlockReaderIdExt + Send + Sync + Clone + 'static,
{
    type MethodResponse = S::MethodResponse;
    type NotificationResponse = S::NotificationResponse;
    type BatchResponse = S::BatchResponse;

    fn call<'a>(&self, req: Request<'a>) -> impl Future<Output = Self::MethodResponse> + Send + 'a {
        let inner_service = self.inner.clone();
        let historical = self.historical.clone();

        Box::pin(async move {
            let maybe_block_id = match req.method_name() {
                "eth_getBlockByNumber" |
                "eth_getBlockByHash" |
                "debug_traceBlockByNumber" |
                "debug_traceBlockByHash" => parse_block_id_from_params(&req.params(), 0),
                "eth_getBalance" |
                "eth_getCode" |
                "eth_getTransactionCount" |
                "eth_call" |
                "eth_estimateGas" |
                "eth_createAccessList" |
                "debug_traceCall" => parse_block_id_from_params(&req.params(), 1),
                "eth_getStorageAt" | "eth_getProof" => parse_block_id_from_params(&req.params(), 2),
                "debug_traceTransaction" => {
                    // debug_traceTransaction takes a transaction hash as its first parameter,
                    // not a BlockId. We assume the op-reth instance is configured with minimal
                    // bootstrap without the bodies so we can't check if this tx is pre bedrock
                    None
                }
                _ => None,
            };

            // if we've extracted a block ID, check if it's pre-Bedrock
            if let Some(block_id) = maybe_block_id {
                let is_pre_bedrock = if let Ok(Some(num)) =
                    historical.provider.block_number_for_id(block_id)
                {
                    num < historical.bedrock_block
                } else {
                    // If we can't convert the hash to a number, assume it's post-Bedrock
                    debug!(target: "rpc::historical", ?block_id, "hash unknown; not forwarding");
                    false
                };

                // if the block is pre-Bedrock, forward the request to the historical client
                if is_pre_bedrock {
                    debug!(target: "rpc::historical", method = %req.method_name(), ?block_id, params=?req.params(), "forwarding pre-Bedrock request");

                    let params = req.params();
                    let params = params.as_str().unwrap_or("[]");
                    if let Ok(params) = serde_json::from_str::<serde_json::Value>(params) {
                        if let Ok(raw) = historical
                            .client
                            .request::<_, serde_json::Value>(req.method_name(), params)
                            .await
                        {
                            let payload = jsonrpsee_types::ResponsePayload::success(raw).into();
                            return MethodResponse::response(req.id, payload, usize::MAX);
                        }
                    }
                }
            }

            // handle the request with the inner service
            inner_service.call(req).await
        })
    }

    fn batch<'a>(&self, req: Batch<'a>) -> impl Future<Output = Self::BatchResponse> + Send + 'a {
        self.inner.batch(req)
    }

    fn notification<'a>(
        &self,
        n: Notification<'a>,
    ) -> impl Future<Output = Self::NotificationResponse> + Send + 'a {
        self.inner.notification(n)
    }
}

#[derive(Debug)]
struct HistoricalRpcInner<P> {
    /// Provider used to determine if a block is pre-bedrock
    provider: P,
    /// Client used to forward historical requests
    client: HistoricalRpcClient,
    /// Bedrock transition block number
    bedrock_block: BlockNumber,
}

/// Parses a `BlockId` from the given parameters at the specified position.
fn parse_block_id_from_params(params: &Params<'_>, position: usize) -> Option<BlockId> {
    let values: Vec<serde_json::Value> = params.parse().ok()?;
    let val = values.into_iter().nth(position)?;
    serde_json::from_value::<BlockId>(val).ok()
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::{BlockId, BlockNumberOrTag};
    use jsonrpsee::types::Params;
    use jsonrpsee_core::middleware::layer::Either;
    use reth_node_builder::rpc::RethRpcMiddleware;
    use reth_storage_api::noop::NoopProvider;
    use tower::layer::util::Identity;

    #[test]
    fn check_historical_rpc() {
        fn assert_historical_rpc<T: RethRpcMiddleware>() {}
        assert_historical_rpc::<HistoricalRpc<NoopProvider>>();
        assert_historical_rpc::<Either<HistoricalRpc<NoopProvider>, Identity>>();
    }

    /// Tests that various valid id types can be parsed from the first parameter.
    #[test]
    fn parses_block_id_from_first_param() {
        // Test with a block number
        let params_num = Params::new(Some(r#"["0x64"]"#)); // 100
        assert_eq!(
            parse_block_id_from_params(&params_num, 0).unwrap(),
            BlockId::Number(BlockNumberOrTag::Number(100))
        );

        // Test with the "earliest" tag
        let params_tag = Params::new(Some(r#"["earliest"]"#));
        assert_eq!(
            parse_block_id_from_params(&params_tag, 0).unwrap(),
            BlockId::Number(BlockNumberOrTag::Earliest)
        );
    }

    /// Tests that the function correctly parses from a position other than 0.
    #[test]
    fn parses_block_id_from_second_param() {
        let params =
            Params::new(Some(r#"["0x0000000000000000000000000000000000000000", "latest"]"#));
        let result = parse_block_id_from_params(&params, 1).unwrap();
        assert_eq!(result, BlockId::Number(BlockNumberOrTag::Latest));
    }

    /// Tests that the function returns nothing if the parameter is missing or empty.
    #[test]
    fn defaults_to_latest_when_param_is_missing() {
        let params = Params::new(Some(r#"["0x0000000000000000000000000000000000000000"]"#));
        let result = parse_block_id_from_params(&params, 1);
        assert!(result.is_none());
    }

    /// Tests that the function doesn't parse anything if the parameter is not a valid block id.
    #[test]
    fn returns_error_for_invalid_input() {
        let params = Params::new(Some(r#"[true]"#));
        let result = parse_block_id_from_params(&params, 0);
        assert!(result.is_none());
    }
}
