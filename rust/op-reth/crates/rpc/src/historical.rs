//! Client support for optimism historical RPC requests.

use crate::sequencer::Error;
use alloy_eips::BlockId;
use alloy_json_rpc::{RpcRecv, RpcSend};
use alloy_primitives::{B256, BlockNumber};
use alloy_rpc_client::RpcClient;
use jsonrpsee::BatchResponseBuilder;
use jsonrpsee_core::{
    middleware::{Batch, BatchEntry, Notification, RpcServiceT},
    server::MethodResponse,
};
use jsonrpsee_types::{Params, Request};
use reth_storage_api::{BlockReaderIdExt, TransactionsProvider};
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
    S: RpcServiceT<
            MethodResponse = MethodResponse,
            BatchResponse = MethodResponse,
            NotificationResponse = MethodResponse,
        > + Send
        + Sync
        + Clone
        + 'static,
    P: BlockReaderIdExt + TransactionsProvider + Send + Sync + Clone + 'static,
{
    type MethodResponse = S::MethodResponse;
    type NotificationResponse = S::NotificationResponse;
    type BatchResponse = S::BatchResponse;

    fn call<'a>(&self, req: Request<'a>) -> impl Future<Output = Self::MethodResponse> + Send + 'a {
        let inner_service = self.inner.clone();
        let historical = self.historical.clone();

        Box::pin(async move {
            // Check if request should be forwarded to historical endpoint
            if let Some(response) = historical.maybe_forward_request(&req).await {
                return response;
            }

            // Handle the request with the inner service
            inner_service.call(req).await
        })
    }

    fn batch<'a>(
        &self,
        mut req: Batch<'a>,
    ) -> impl Future<Output = Self::BatchResponse> + Send + 'a {
        let this = self.clone();
        let historical = self.historical.clone();

        async move {
            let mut needs_forwarding = false;
            for entry in req.iter_mut() {
                if let Ok(BatchEntry::Call(call)) = entry &&
                    historical.should_forward_request(call)
                {
                    needs_forwarding = true;
                    break;
                }
            }

            if !needs_forwarding {
                // no call needs to be forwarded and we can simply perform this batch request
                return this.inner.batch(req).await;
            }

            // the entire response is checked above so we can assume that these don't exceed
            let mut batch_rp = BatchResponseBuilder::new_with_limit(usize::MAX);
            let mut got_notification = false;

            for batch_entry in req {
                match batch_entry {
                    Ok(BatchEntry::Call(req)) => {
                        let rp = this.call(req).await;
                        if let Err(err) = batch_rp.append(rp) {
                            return err;
                        }
                    }
                    Ok(BatchEntry::Notification(n)) => {
                        got_notification = true;
                        this.notification(n).await;
                    }
                    Err(err) => {
                        let (err, id) = err.into_parts();
                        let rp = MethodResponse::error(id, err);
                        if let Err(err) = batch_rp.append(rp) {
                            return err;
                        }
                    }
                }
            }

            // If the batch is empty and we got a notification, we return an empty response.
            if batch_rp.is_empty() && got_notification {
                MethodResponse::notification()
            }
            // An empty batch is regarded as an invalid request here.
            else {
                MethodResponse::from_batch(batch_rp.finish())
            }
        }
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

impl<P> HistoricalRpcInner<P>
where
    P: BlockReaderIdExt + TransactionsProvider + Send + Sync + Clone,
{
    /// Checks if a request should be forwarded to the historical endpoint (synchronous check).
    fn should_forward_request(&self, req: &Request<'_>) -> bool {
        match req.method_name() {
            "debug_traceTransaction" |
            "eth_getTransactionByHash" |
            "eth_getTransactionReceipt" |
            "eth_getRawTransactionByHash" => self.should_forward_transaction(req),
            method => self.should_forward_block_request(method, req),
        }
    }

    /// Checks if a request should be forwarded to the historical endpoint and returns
    /// the response if it was forwarded.
    async fn maybe_forward_request(&self, req: &Request<'_>) -> Option<MethodResponse> {
        if self.should_forward_request(req) {
            return self.forward_to_historical(req).await;
        }
        None
    }

    /// Determines if a transaction request should be forwarded
    fn should_forward_transaction(&self, req: &Request<'_>) -> bool {
        parse_transaction_hash_from_params(&req.params())
            .ok()
            .map(|tx_hash| {
                // Check if we can find the transaction locally and get its metadata
                match self.provider.transaction_by_hash_with_meta(tx_hash) {
                    Ok(Some((_, meta))) => {
                        // Transaction found - check if it's pre-bedrock based on block number
                        let is_pre_bedrock = meta.block_number < self.bedrock_block;
                        if is_pre_bedrock {
                            debug!(
                                target: "rpc::historical",
                                ?tx_hash,
                                block_num = meta.block_number,
                                bedrock = self.bedrock_block,
                                "transaction found in pre-bedrock block, forwarding to historical endpoint"
                            );
                        }
                        is_pre_bedrock
                    }
                    _ => {
                        // Transaction not found locally, optimistically forward to historical endpoint
                        debug!(
                            target: "rpc::historical",
                            ?tx_hash,
                            "transaction not found locally, forwarding to historical endpoint"
                        );
                        true
                    }
                }
            })
            .unwrap_or(false)
    }

    /// Determines if a block-based request should be forwarded
    fn should_forward_block_request(&self, method: &str, req: &Request<'_>) -> bool {
        let maybe_block_id = extract_block_id_for_method(method, &req.params());

        maybe_block_id.map(|block_id| self.is_pre_bedrock(block_id)).unwrap_or(false)
    }

    /// Checks if a block ID refers to a pre-bedrock block
    fn is_pre_bedrock(&self, block_id: BlockId) -> bool {
        match self.provider.block_number_for_id(block_id) {
            Ok(Some(num)) => {
                debug!(
                    target: "rpc::historical",
                    ?block_id,
                    block_num=num,
                    bedrock=self.bedrock_block,
                    "found block number"
                );
                num < self.bedrock_block
            }
            Ok(None) if block_id.is_hash() => {
                debug!(
                    target: "rpc::historical",
                    ?block_id,
                    "block hash not found locally, assuming pre-bedrock"
                );
                true
            }
            _ => {
                debug!(
                    target: "rpc::historical",
                    ?block_id,
                    "could not determine block number; not forwarding"
                );
                false
            }
        }
    }

    /// Forwards a request to the historical endpoint
    async fn forward_to_historical(&self, req: &Request<'_>) -> Option<MethodResponse> {
        debug!(
            target: "rpc::historical",
            method = %req.method_name(),
            params=?req.params(),
            "forwarding request to historical endpoint"
        );

        let params = req.params();
        let params_str = params.as_str().unwrap_or("[]");

        let params = serde_json::from_str::<serde_json::Value>(params_str).ok()?;

        let raw =
            self.client.request::<_, serde_json::Value>(req.method_name(), params).await.ok()?;

        let payload = jsonrpsee_types::ResponsePayload::success(raw).into();
        Some(MethodResponse::response(req.id.clone(), payload, usize::MAX))
    }
}

/// Error type for parameter parsing
#[derive(Debug)]
enum ParseError {
    InvalidFormat,
    MissingParameter,
}

/// Extracts the block ID from request parameters based on the method name
fn extract_block_id_for_method(method: &str, params: &Params<'_>) -> Option<BlockId> {
    match method {
        "eth_getBlockByNumber" |
        "eth_getBlockByHash" |
        "debug_traceBlockByNumber" |
        "debug_traceBlockByHash" => parse_block_id_from_params(params, 0),
        "eth_getBalance" |
        "eth_getCode" |
        "eth_getTransactionCount" |
        "eth_call" |
        "eth_estimateGas" |
        "eth_createAccessList" |
        "debug_traceCall" => parse_block_id_from_params(params, 1),
        "eth_getStorageAt" | "eth_getProof" => parse_block_id_from_params(params, 2),
        _ => None,
    }
}

/// Parses a `BlockId` from the given parameters at the specified position.
fn parse_block_id_from_params(params: &Params<'_>, position: usize) -> Option<BlockId> {
    let values: Vec<serde_json::Value> = params.parse().ok()?;
    let val = values.into_iter().nth(position)?;
    serde_json::from_value::<BlockId>(val).ok()
}

/// Parses a transaction hash from the first parameter.
fn parse_transaction_hash_from_params(params: &Params<'_>) -> Result<B256, ParseError> {
    let values: Vec<serde_json::Value> = params.parse().map_err(|_| ParseError::InvalidFormat)?;
    let val = values.into_iter().next().ok_or(ParseError::MissingParameter)?;
    serde_json::from_value::<B256>(val).map_err(|_| ParseError::InvalidFormat)
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

    /// Tests that transaction hashes can be parsed from params.
    #[test]
    fn parses_transaction_hash_from_params() {
        let hash = "0xdbdfa0f88b2cf815fdc1621bd20c2bd2b0eed4f0c56c9be2602957b5a60ec702";
        let params_str = format!(r#"["{hash}"]"#);
        let params = Params::new(Some(&params_str));
        let result = parse_transaction_hash_from_params(&params);
        assert!(result.is_ok());
        let parsed_hash = result.unwrap();
        assert_eq!(format!("{parsed_hash:?}"), hash);
    }

    /// Tests that invalid transaction hash returns error.
    #[test]
    fn returns_error_for_invalid_tx_hash() {
        let params = Params::new(Some(r#"["not_a_hash"]"#));
        let result = parse_transaction_hash_from_params(&params);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ParseError::InvalidFormat));
    }

    /// Tests that missing parameter returns appropriate error.
    #[test]
    fn returns_error_for_missing_parameter() {
        let params = Params::new(Some(r#"[]"#));
        let result = parse_transaction_hash_from_params(&params);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ParseError::MissingParameter));
    }
}
