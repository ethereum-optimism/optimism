//! Client support for optimism historical RPC requests.

use crate::sequencer::Error;
use alloy_eips::BlockId;
use alloy_json_rpc::{RpcRecv, RpcSend};
use alloy_primitives::{B256, BlockNumber};
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_eth::{Filter, FilterBlockOption};
use alloy_transport::TransportErrorKind;
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

    /// Constructs a historical RPC client around an existing [`RpcClient`], for tests.
    #[cfg(test)]
    fn with_client(client: RpcClient) -> Self {
        Self {
            inner: Arc::new(HistoricalRpcClientInner {
                historical_endpoint: "http://localhost:0".to_string(),
                client,
            }),
        }
    }

    /// Returns a reference to the underlying RPC client
    fn client(&self) -> &RpcClient {
        &self.inner.client
    }

    /// Forwards a JSON-RPC request to the historical endpoint.
    ///
    /// Errors are returned to the caller unlogged: expected errors (e.g. the method-not-found
    /// probe answered by l2geth) must not produce log noise, so callers log at the level
    /// appropriate to how they handle the failure.
    pub async fn request<Params: RpcSend, Resp: RpcRecv>(
        &self,
        method: &str,
        params: Params,
    ) -> Result<Resp, Error> {
        let resp = self.client().request::<Params, Resp>(method.to_string(), params).await?;

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

        let raw = match self.client.request::<_, serde_json::Value>(req.method_name(), params).await
        {
            Ok(raw) => raw,
            // l2geth doesn't serve `eth_getBlockReceipts`; stitch the response from per-tx
            // receipts instead. Any other error keeps the existing fall-through-to-local
            // behavior for all methods.
            Err(err)
                if req.method_name() == "eth_getBlockReceipts" && is_method_not_found(&err) =>
            {
                debug!(
                    target: "rpc::historical",
                    "historical endpoint lacks eth_getBlockReceipts, stitching from per-transaction receipts"
                );
                return Some(self.stitch_block_receipts(req).await);
            }
            Err(err) => {
                warn!(
                    target: "rpc::historical",
                    method = %req.method_name(),
                    %err,
                    "historical endpoint request failed; falling back to local handling"
                );
                return None;
            }
        };

        let payload = jsonrpsee_types::ResponsePayload::success(raw).into();
        Some(MethodResponse::response(req.id.clone(), payload, usize::MAX))
    }

    /// Serves `eth_getBlockReceipts` against a historical endpoint that lacks the method by
    /// fetching the block's transaction hashes and issuing one `eth_getTransactionReceipt` per
    /// hash. Receipts are passed through verbatim, matching what the forwarded
    /// `eth_getTransactionReceipt` returns for the same transactions.
    ///
    /// Failures produce a JSON-RPC error response instead of falling through to local handling,
    /// which cannot serve pre-bedrock receipts.
    async fn stitch_block_receipts(&self, req: &Request<'_>) -> MethodResponse {
        match self.fetch_stitched_block_receipts(req).await {
            Ok(raw) => {
                let payload = jsonrpsee_types::ResponsePayload::success(raw).into();
                MethodResponse::response(req.id.clone(), payload, usize::MAX)
            }
            Err(err) => {
                warn!(
                    target: "rpc::historical",
                    %err,
                    "failed to stitch eth_getBlockReceipts response from historical endpoint"
                );
                // Keep the client-facing message generic: the error Display may embed raw
                // responses from the internal historical endpoint.
                MethodResponse::error(
                    req.id.clone(),
                    jsonrpsee_types::ErrorObject::owned(
                        jsonrpsee_types::error::INTERNAL_ERROR_CODE,
                        "failed to fetch pre-bedrock receipts from historical endpoint",
                        None::<()>,
                    ),
                )
            }
        }
    }

    /// Assembles an `eth_getBlockReceipts` response from per-transaction receipt requests.
    ///
    /// Returns JSON `null` if the historical endpoint doesn't know the block, and `[]` for a
    /// block without transactions.
    async fn fetch_stitched_block_receipts(
        &self,
        req: &Request<'_>,
    ) -> Result<serde_json::Value, Error> {
        let block_id = parse_block_id_from_params(&req.params(), 0).ok_or_else(|| {
            Error::TransportError(TransportErrorKind::custom_str("invalid block id parameter"))
        })?;

        // `full = false` returns transaction hashes only
        let (method, block_param) = match block_id {
            BlockId::Hash(hash) => ("eth_getBlockByHash", serde_json::json!(hash.block_hash)),
            BlockId::Number(number) => ("eth_getBlockByNumber", serde_json::json!(number)),
        };
        let block =
            self.client.request::<_, serde_json::Value>(method, (block_param, false)).await?;

        if block.is_null() {
            return Ok(serde_json::Value::Null);
        }

        let tx_hashes = block
            .get("transactions")
            .and_then(serde_json::Value::as_array)
            .cloned()
            .unwrap_or_default();

        // pre-bedrock OP mainnet blocks contain at most one transaction
        let mut receipts = Vec::with_capacity(tx_hashes.len());
        for tx_hash in tx_hashes {
            let receipt = self
                .client
                .request::<_, serde_json::Value>("eth_getTransactionReceipt", (tx_hash,))
                .await?;
            if receipt.is_null() {
                return Err(Error::TransportError(TransportErrorKind::custom_str(
                    "historical endpoint returned no receipt for a block transaction",
                )));
            }
            receipts.push(receipt);
        }

        Ok(serde_json::Value::Array(receipts))
    }
}

/// Returns true if the given error is a JSON-RPC "method not found" error response.
fn is_method_not_found(err: &Error) -> bool {
    match err {
        Error::TransportError(err) => err
            .as_error_resp()
            .is_some_and(|resp| resp.code == jsonrpsee_types::error::METHOD_NOT_FOUND_CODE as i64),
        _ => false,
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
        "eth_getBlockReceipts" |
        "eth_getHeaderByNumber" |
        "eth_getHeaderByHash" |
        "eth_getBlockTransactionCountByNumber" |
        "eth_getBlockTransactionCountByHash" |
        "eth_getUncleCountByBlockNumber" |
        "eth_getUncleCountByBlockHash" |
        "eth_getUncleByBlockNumberAndIndex" |
        "eth_getUncleByBlockHashAndIndex" |
        "eth_getTransactionByBlockNumberAndIndex" |
        "eth_getTransactionByBlockHashAndIndex" |
        "eth_getRawTransactionByBlockNumberAndIndex" |
        "eth_getRawTransactionByBlockHashAndIndex" |
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
        "eth_getLogs" => parse_block_id_from_log_filter(params),
        _ => None,
    }
}

/// Parses the block that decides forwarding for an `eth_getLogs` filter.
///
/// A `blockHash` filter targets that single block. A block range is represented by its upper
/// bound, because a range lies entirely before bedrock exactly when its highest block does.
/// Ranges reaching bedrock or beyond — including ones that cross it — yield a post-bedrock block
/// and are served locally as before; an open-ended range (no `toBlock`) reaches the chain tip and
/// is likewise not forwarded.
fn parse_block_id_from_log_filter(params: &Params<'_>) -> Option<BlockId> {
    let values: Vec<serde_json::Value> = params.parse().ok()?;
    let filter = serde_json::from_value::<Filter>(values.into_iter().next()?).ok()?;

    match filter.block_option {
        FilterBlockOption::AtBlockHash(hash) => Some(BlockId::Hash(hash.into())),
        FilterBlockOption::Range { to_block, .. } => to_block.map(BlockId::Number),
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
    use alloy_json_rpc::{ErrorPayload, RequestPacket, ResponsePacket};
    use alloy_transport::{
        TransportError, TransportFut,
        mock::{Asserter, MockTransport},
    };
    use jsonrpsee::types::Params;
    use jsonrpsee_core::middleware::layer::Either;
    use jsonrpsee_types::Id;
    use reth_node_builder::rpc::RethRpcMiddleware;
    use reth_storage_api::noop::NoopProvider;
    use serde_json::json;
    use std::sync::{
        Mutex,
        atomic::{AtomicUsize, Ordering},
    };
    use tower::layer::util::Identity;

    fn method_not_found_payload() -> ErrorPayload {
        ErrorPayload {
            code: jsonrpsee_types::error::METHOD_NOT_FOUND_CODE as i64,
            message: "the method eth_getBlockReceipts does not exist/is not available".into(),
            data: None,
        }
    }

    fn mocked_historical(asserter: Asserter) -> HistoricalRpcInner<NoopProvider> {
        HistoricalRpcInner {
            provider: NoopProvider::default(),
            client: HistoricalRpcClient::with_client(RpcClient::mocked(asserter)),
            bedrock_block: 105235063,
        }
    }

    /// Shared log of outbound `(method, params)` pairs captured by [`RecordingTransport`].
    type RequestLog = Arc<Mutex<Vec<(String, serde_json::Value)>>>;

    /// A [`MockTransport`] wrapper that records outbound request methods and params, so tests
    /// can assert what the stitching logic sends upstream (the FIFO mock alone checks neither).
    #[derive(Clone, Debug)]
    struct RecordingTransport {
        inner: MockTransport,
        requests: RequestLog,
    }

    impl tower::Service<RequestPacket> for RecordingTransport {
        type Response = ResponsePacket;
        type Error = TransportError;
        type Future = TransportFut<'static>;

        fn poll_ready(
            &mut self,
            cx: &mut std::task::Context<'_>,
        ) -> std::task::Poll<Result<(), Self::Error>> {
            tower::Service::poll_ready(&mut self.inner, cx)
        }

        fn call(&mut self, req: RequestPacket) -> Self::Future {
            if let RequestPacket::Single(single) = &req {
                let params = single
                    .params()
                    .map(|raw| serde_json::from_str(raw.get()).unwrap())
                    .unwrap_or(serde_json::Value::Null);
                self.requests.lock().unwrap().push((single.method().to_string(), params));
            }
            tower::Service::call(&mut self.inner, req)
        }
    }

    fn recorded_historical(asserter: Asserter) -> (HistoricalRpcInner<NoopProvider>, RequestLog) {
        let requests = RequestLog::default();
        let transport =
            RecordingTransport { inner: MockTransport::new(asserter), requests: requests.clone() };
        let historical = HistoricalRpcInner {
            provider: NoopProvider::default(),
            client: HistoricalRpcClient::with_client(RpcClient::new(transport, true)),
            bedrock_block: 105235063,
        };
        (historical, requests)
    }

    fn recorded_requests(log: &RequestLog) -> Vec<(String, serde_json::Value)> {
        log.lock().unwrap().clone()
    }

    fn owned_request(method: &str, params: &str) -> Request<'static> {
        Request::owned(
            method.to_string(),
            Some(serde_json::value::RawValue::from_string(params.to_string()).unwrap()),
            Id::Number(1),
        )
    }

    fn result_of(resp: &MethodResponse) -> serde_json::Value {
        let json: serde_json::Value = serde_json::from_str(resp.as_json().get()).unwrap();
        json.get("result").cloned().unwrap()
    }

    fn error_message_of(resp: &MethodResponse) -> String {
        let json: serde_json::Value = serde_json::from_str(resp.as_json().get()).unwrap();
        json["error"]["message"].as_str().unwrap().to_string()
    }

    /// A [`tracing::Subscriber`] that counts WARN-level events.
    struct WarnCounter {
        warns: Arc<AtomicUsize>,
    }

    impl tracing::Subscriber for WarnCounter {
        fn enabled(&self, metadata: &tracing::Metadata<'_>) -> bool {
            *metadata.level() <= tracing::Level::WARN
        }

        fn new_span(&self, _span: &tracing::span::Attributes<'_>) -> tracing::span::Id {
            tracing::span::Id::from_u64(1)
        }

        fn record(&self, _span: &tracing::span::Id, _values: &tracing::span::Record<'_>) {}

        fn record_follows_from(&self, _span: &tracing::span::Id, _follows: &tracing::span::Id) {}

        fn event(&self, event: &tracing::Event<'_>) {
            if *event.metadata().level() == tracing::Level::WARN {
                self.warns.fetch_add(1, Ordering::SeqCst);
            }
        }

        fn enter(&self, _span: &tracing::span::Id) {}

        fn exit(&self, _span: &tracing::span::Id) {}
    }

    /// Runs `fut` on a current-thread runtime and returns the number of WARN events emitted.
    fn warns_during<F: Future<Output = ()>>(fut: F) -> usize {
        let warns = Arc::new(AtomicUsize::new(0));
        tracing::subscriber::with_default(WarnCounter { warns: warns.clone() }, || {
            tokio::runtime::Builder::new_current_thread().build().unwrap().block_on(fut)
        });
        warns.load(Ordering::SeqCst)
    }

    #[test]
    fn check_historical_rpc() {
        fn assert_historical_rpc<T: RethRpcMiddleware>() {}
        assert_historical_rpc::<HistoricalRpc<NoopProvider>>();
        assert_historical_rpc::<Either<HistoricalRpc<NoopProvider>, Identity>>();
    }

    /// Tests that block-number-parameterized methods extract the block id from the first
    /// parameter.
    #[test]
    fn extracts_block_id_for_block_number_methods() {
        for (method, params_str) in [
            ("eth_getBlockReceipts", r#"["0x64"]"#),
            ("eth_getHeaderByNumber", r#"["0x64"]"#),
            ("eth_getBlockTransactionCountByNumber", r#"["0x64"]"#),
            ("eth_getUncleCountByBlockNumber", r#"["0x64"]"#),
            ("eth_getUncleByBlockNumberAndIndex", r#"["0x64", "0x0"]"#),
            ("eth_getTransactionByBlockNumberAndIndex", r#"["0x64", "0x0"]"#),
            ("eth_getRawTransactionByBlockNumberAndIndex", r#"["0x64", "0x0"]"#),
        ] {
            let params = Params::new(Some(params_str));
            assert_eq!(
                extract_block_id_for_method(method, &params).unwrap(),
                BlockId::Number(BlockNumberOrTag::Number(100)),
                "{method}"
            );
        }
    }

    /// Tests that block-hash-parameterized methods extract the block id from the first parameter.
    #[test]
    fn extracts_block_id_for_block_hash_methods() {
        let hash = "0xdbdfa0f88b2cf815fdc1621bd20c2bd2b0eed4f0c56c9be2602957b5a60ec702";
        for (method, params_str) in [
            ("eth_getBlockReceipts", format!(r#"["{hash}"]"#)),
            ("eth_getHeaderByHash", format!(r#"["{hash}"]"#)),
            ("eth_getBlockTransactionCountByHash", format!(r#"["{hash}"]"#)),
            ("eth_getUncleCountByBlockHash", format!(r#"["{hash}"]"#)),
            ("eth_getUncleByBlockHashAndIndex", format!(r#"["{hash}", "0x0"]"#)),
            ("eth_getTransactionByBlockHashAndIndex", format!(r#"["{hash}", "0x0"]"#)),
            ("eth_getRawTransactionByBlockHashAndIndex", format!(r#"["{hash}", "0x0"]"#)),
        ] {
            let params = Params::new(Some(&params_str));
            assert_eq!(
                extract_block_id_for_method(method, &params).unwrap(),
                BlockId::Hash(hash.parse::<B256>().unwrap().into()),
                "{method}"
            );
        }
    }

    /// Tests that an `eth_getLogs` range filter extracts its upper bound, the block that decides
    /// whether the whole range is pre-bedrock.
    #[test]
    fn extracts_block_id_for_log_filter_range() {
        for params_str in [
            r#"[{"fromBlock":"0x64","toBlock":"0x64"}]"#,
            r#"[{"fromBlock":"0x0","toBlock":"0x64"}]"#,
            r#"[{"toBlock":"0x64"}]"#,
        ] {
            let params = Params::new(Some(params_str));
            assert_eq!(
                extract_block_id_for_method("eth_getLogs", &params).unwrap(),
                BlockId::Number(BlockNumberOrTag::Number(100)),
                "{params_str}"
            );
        }
    }

    /// Tests that an `eth_getLogs` filter pinned to a block hash extracts that hash.
    #[test]
    fn extracts_block_id_for_log_filter_block_hash() {
        let hash = "0xdbdfa0f88b2cf815fdc1621bd20c2bd2b0eed4f0c56c9be2602957b5a60ec702";
        let params_str = format!(r#"[{{"blockHash":"{hash}"}}]"#);
        let params = Params::new(Some(&params_str));
        assert_eq!(
            extract_block_id_for_method("eth_getLogs", &params).unwrap(),
            BlockId::Hash(hash.parse::<B256>().unwrap().into())
        );
    }

    /// Tests that an `eth_getLogs` filter without an upper bound extracts no block id, leaving it
    /// to local handling.
    #[test]
    fn extracts_no_block_id_for_open_ended_log_filter() {
        for params_str in [r#"[{"fromBlock":"0x0"}]"#, r#"[{}]"#, r#"[]"#] {
            let params = Params::new(Some(params_str));
            assert!(extract_block_id_for_method("eth_getLogs", &params).is_none(), "{params_str}");
        }
    }

    /// Tests that `eth_getLogs` is forwarded when its filter is confined to pre-bedrock blocks,
    /// using the OP Mainnet blocks from the bug report (bedrock is 105235063): pre-bedrock
    /// 105235062 / `0x645c276` and post-bedrock 105239000 / `0x645d1d8`. A range crossing bedrock
    /// stays local.
    #[test]
    fn forwards_pre_bedrock_log_filters() {
        let historical = mocked_historical(Asserter::new());
        let logs_request = |from: &str, to: &str| {
            owned_request(
                "eth_getLogs",
                &format!(
                    r#"[{{"fromBlock":"{from}","toBlock":"{to}","address":"0x5e61a079a178f0e5784107a4963baae0c5a680c6"}}]"#
                ),
            )
        };

        assert!(historical.should_forward_request(&logs_request("0x645c276", "0x645c276")));
        assert!(historical.should_forward_request(&logs_request("0x0", "0x645c276")));
        assert!(!historical.should_forward_request(&logs_request("0x645d1d8", "0x645d1d8")));
        assert!(!historical.should_forward_request(&logs_request("0x645c276", "0x645d1d8")));
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

    /// Tests that `eth_getBlockReceipts` responses from endpoints that serve the method natively
    /// are passed through unchanged by a single upstream call.
    #[tokio::test]
    async fn forwards_block_receipts_natively() {
        let asserter = Asserter::new();
        let receipts = json!([{"transactionHash": "0xabc", "status": "0x1"}]);
        asserter.push_success(&receipts);

        let (historical, requests) = recorded_historical(asserter.clone());
        let req = owned_request("eth_getBlockReceipts", r#"["0x64"]"#);
        let resp = historical.forward_to_historical(&req).await.unwrap();

        assert!(resp.is_success());
        assert_eq!(result_of(&resp), receipts);
        assert_eq!(
            recorded_requests(&requests),
            vec![("eth_getBlockReceipts".to_string(), json!(["0x64"]))],
            "expected a single upstream call"
        );
    }

    /// Tests that a method-not-found error from the historical endpoint (l2geth) stitches the
    /// response from the block's per-transaction receipts, passed through verbatim, fetching the
    /// block by number with hashes only (`full = false`) and each receipt by transaction hash.
    #[tokio::test]
    async fn stitches_block_receipts_on_method_not_found() {
        let asserter = Asserter::new();
        let tx_hash = "0x9c50bb3ba00b689fcbca3fb0837ca1af1cbd9e6b16fea8f0e4f47442dd0f78ec";
        let receipt = json!({"transactionHash": tx_hash, "status": "0x1", "l1Fee": "0x143839f0f0"});
        asserter.push_failure(method_not_found_payload());
        asserter.push_success(&json!({"number": "0x64", "transactions": [tx_hash]}));
        asserter.push_success(&receipt);

        let (historical, requests) = recorded_historical(asserter.clone());
        let req = owned_request("eth_getBlockReceipts", r#"["0x64"]"#);
        let resp = historical.forward_to_historical(&req).await.unwrap();

        assert!(resp.is_success());
        assert_eq!(result_of(&resp), json!([receipt]));
        assert_eq!(
            recorded_requests(&requests),
            vec![
                ("eth_getBlockReceipts".to_string(), json!(["0x64"])),
                ("eth_getBlockByNumber".to_string(), json!(["0x64", false])),
                ("eth_getTransactionReceipt".to_string(), json!([tx_hash])),
            ]
        );
    }

    /// Tests that stitching works for hash block ids as well, fetching the block by hash.
    #[tokio::test]
    async fn stitches_block_receipts_for_hash_param() {
        let asserter = Asserter::new();
        let block_hash = "0xdbdfa0f88b2cf815fdc1621bd20c2bd2b0eed4f0c56c9be2602957b5a60ec702";
        let tx_hash = "0x9c50bb3ba00b689fcbca3fb0837ca1af1cbd9e6b16fea8f0e4f47442dd0f78ec";
        let receipt = json!({"transactionHash": tx_hash, "status": "0x1"});
        asserter.push_failure(method_not_found_payload());
        asserter.push_success(&json!({"hash": block_hash, "transactions": [tx_hash]}));
        asserter.push_success(&receipt);

        let (historical, requests) = recorded_historical(asserter);
        let req = owned_request("eth_getBlockReceipts", &format!(r#"["{block_hash}"]"#));
        let resp = historical.forward_to_historical(&req).await.unwrap();

        assert!(resp.is_success());
        assert_eq!(result_of(&resp), json!([receipt]));
        assert_eq!(
            recorded_requests(&requests),
            vec![
                ("eth_getBlockReceipts".to_string(), json!([block_hash])),
                ("eth_getBlockByHash".to_string(), json!([block_hash, false])),
                ("eth_getTransactionReceipt".to_string(), json!([tx_hash])),
            ]
        );
    }

    /// Tests that a block without transactions stitches to an empty array without issuing any
    /// receipt requests.
    #[tokio::test]
    async fn stitches_empty_receipts_for_empty_block() {
        let asserter = Asserter::new();
        asserter.push_failure(method_not_found_payload());
        asserter.push_success(&json!({"number": "0x0", "transactions": []}));

        let (historical, requests) = recorded_historical(asserter);
        let req = owned_request("eth_getBlockReceipts", r#"["0x0"]"#);
        let resp = historical.forward_to_historical(&req).await.unwrap();

        assert!(resp.is_success());
        assert_eq!(result_of(&resp), json!([]));
        assert_eq!(
            recorded_requests(&requests),
            vec![
                ("eth_getBlockReceipts".to_string(), json!(["0x0"])),
                ("eth_getBlockByNumber".to_string(), json!(["0x0", false])),
            ]
        );
    }

    /// Tests that a block unknown to the historical endpoint stitches to `null`, mirroring what
    /// an endpoint serving `eth_getBlockReceipts` natively returns.
    #[tokio::test]
    async fn stitches_null_for_unknown_block() {
        let asserter = Asserter::new();
        asserter.push_failure(method_not_found_payload());
        asserter.push_success(&serde_json::Value::Null);

        let (historical, requests) = recorded_historical(asserter);
        let req = owned_request("eth_getBlockReceipts", r#"["0x64"]"#);
        let resp = historical.forward_to_historical(&req).await.unwrap();

        assert!(resp.is_success());
        assert_eq!(result_of(&resp), serde_json::Value::Null);
        assert_eq!(
            recorded_requests(&requests),
            vec![
                ("eth_getBlockReceipts".to_string(), json!(["0x64"])),
                ("eth_getBlockByNumber".to_string(), json!(["0x64", false])),
            ]
        );
    }

    /// Tests that a failing receipt request during stitching produces a JSON-RPC error instead
    /// of falling through to local handling, with a generic message that doesn't echo internal
    /// endpoint errors to the caller.
    #[tokio::test]
    async fn stitch_receipt_failure_returns_error() {
        let asserter = Asserter::new();
        let tx_hash = "0x9c50bb3ba00b689fcbca3fb0837ca1af1cbd9e6b16fea8f0e4f47442dd0f78ec";
        asserter.push_failure(method_not_found_payload());
        asserter.push_success(&json!({"number": "0x64", "transactions": [tx_hash]}));
        asserter.push_failure_msg("receipt fetch failed");

        let historical = mocked_historical(asserter);
        let req = owned_request("eth_getBlockReceipts", r#"["0x64"]"#);
        let resp = historical.forward_to_historical(&req).await.unwrap();

        assert!(!resp.is_success());
        assert_eq!(resp.as_error_code(), Some(jsonrpsee_types::error::INTERNAL_ERROR_CODE));
        assert_eq!(
            error_message_of(&resp),
            "failed to fetch pre-bedrock receipts from historical endpoint"
        );
    }

    /// Tests that method-not-found errors for methods other than `eth_getBlockReceipts` keep the
    /// existing fall-through-to-local behavior.
    #[tokio::test]
    async fn method_not_found_falls_through_for_other_methods() {
        let asserter = Asserter::new();
        asserter.push_failure(method_not_found_payload());

        let historical = mocked_historical(asserter);
        let req = owned_request("eth_getHeaderByNumber", r#"["0x64"]"#);
        assert!(historical.forward_to_historical(&req).await.is_none());
    }

    /// Tests that the stitched happy path emits no warnings: the method-not-found probe answer
    /// from l2geth is expected, not a failure.
    #[test]
    fn stitched_happy_path_emits_no_warnings() {
        let warns = warns_during(async {
            let asserter = Asserter::new();
            let tx_hash = "0x9c50bb3ba00b689fcbca3fb0837ca1af1cbd9e6b16fea8f0e4f47442dd0f78ec";
            asserter.push_failure(method_not_found_payload());
            asserter.push_success(&json!({"number": "0x64", "transactions": [tx_hash]}));
            asserter.push_success(&json!({"transactionHash": tx_hash, "status": "0x1"}));

            let historical = mocked_historical(asserter);
            let req = owned_request("eth_getBlockReceipts", r#"["0x64"]"#);
            let resp = historical.forward_to_historical(&req).await.unwrap();
            assert!(resp.is_success());
        });
        assert_eq!(warns, 0, "expected no warnings on the stitched happy path");
    }

    /// Tests that a stitch failure is logged exactly once, not per layer.
    #[test]
    fn stitch_failure_warns_once() {
        let warns = warns_during(async {
            let asserter = Asserter::new();
            let tx_hash = "0x9c50bb3ba00b689fcbca3fb0837ca1af1cbd9e6b16fea8f0e4f47442dd0f78ec";
            asserter.push_failure(method_not_found_payload());
            asserter.push_success(&json!({"number": "0x64", "transactions": [tx_hash]}));
            asserter.push_failure_msg("receipt fetch failed");

            let historical = mocked_historical(asserter);
            let req = owned_request("eth_getBlockReceipts", r#"["0x64"]"#);
            let resp = historical.forward_to_historical(&req).await.unwrap();
            assert!(!resp.is_success());
        });
        assert_eq!(warns, 1, "expected exactly one warning for a stitch failure");
    }

    /// Tests that a real forwarding failure still warns once when falling back to local
    /// handling.
    #[test]
    fn forward_failure_fall_through_warns_once() {
        let warns = warns_during(async {
            let asserter = Asserter::new();
            asserter.push_failure_msg("boom");

            let historical = mocked_historical(asserter);
            let req = owned_request("eth_getHeaderByNumber", r#"["0x64"]"#);
            assert!(historical.forward_to_historical(&req).await.is_none());
        });
        assert_eq!(warns, 1, "expected exactly one warning for a forwarding failure");
    }
}
