//! Client support for optimism historical RPC requests.

use crate::sequencer::Error;
use alloy_eips::{BlockId, BlockNumberOrTag};
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
use std::{borrow::Cow, future::Future, sync::Arc};
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
    ///
    /// `max_blocks_per_filter` is the node's `--rpc.max-blocks-per-filter` setting, applied to
    /// `eth_getLogs` filters before they are forwarded so that forwarding doesn't bypass it.
    pub fn new(
        provider: P,
        client: HistoricalRpcClient,
        bedrock_block: BlockNumber,
        max_blocks_per_filter: Option<u64>,
    ) -> Self {
        let inner =
            Arc::new(HistoricalRpcInner { provider, client, bedrock_block, max_blocks_per_filter });

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
            // A log filter straddling the bedrock transition is the only request served by both
            // backends, so it is the only one that needs the inner service here.
            if req.method_name() == "eth_getLogs" {
                return match historical.route_log_filter(&req) {
                    LogFilterRoute::Local => inner_service.call(req).await,
                    LogFilterRoute::Historical => {
                        match historical.forward_to_historical(&req).await {
                            Some(response) => response,
                            None => inner_service.call(req).await,
                        }
                    }
                    LogFilterRoute::Split => {
                        historical.serve_split_log_filter(&inner_service, req).await
                    }
                };
            }

            // Check if request should be forwarded to historical endpoint
            if historical.should_forward_request(&req) &&
                let Some(response) = historical.forward_to_historical(&req).await
            {
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

/// Which backend serves an `eth_getLogs` request.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum LogFilterRoute {
    /// The local node alone.
    Local,
    /// The historical endpoint alone: no block the filter can match is post-bedrock.
    Historical,
    /// Both, because the filter straddles the bedrock transition.
    Split,
}

#[derive(Debug)]
struct HistoricalRpcInner<P> {
    /// Provider used to determine if a block is pre-bedrock
    provider: P,
    /// Client used to forward historical requests
    client: HistoricalRpcClient,
    /// Bedrock transition block number
    bedrock_block: BlockNumber,
    /// The node's `--rpc.max-blocks-per-filter` limit, if any
    max_blocks_per_filter: Option<u64>,
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
            "eth_getLogs" => self.route_log_filter(req) != LogFilterRoute::Local,
            method => self.should_forward_block_request(method, req),
        }
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

    /// Decides which backend serves an `eth_getLogs` request.
    ///
    /// A `blockHash` filter targets a single block and is decided like every other hash-addressed
    /// method. A range filter goes to the historical endpoint when every block it can match is
    /// pre-bedrock, and is split across both backends when it starts below the transition and
    /// runs into it.
    fn route_log_filter(&self, req: &Request<'_>) -> LogFilterRoute {
        let (from_bound, to_bound) = match parse_log_filter_block_option(&req.params()) {
            Some(FilterBlockOption::AtBlockHash(hash)) => {
                return if self.is_pre_bedrock(BlockId::Hash(hash.into())) {
                    LogFilterRoute::Historical
                } else {
                    LogFilterRoute::Local
                };
            }
            Some(FilterBlockOption::Range { from_block, to_block }) => (from_block, to_block),
            None => return LogFilterRoute::Local,
        };

        let from_block = from_bound.and_then(|bound| self.resolve_filter_bound(bound));
        let to_block = to_bound.and_then(|bound| self.resolve_filter_bound(bound));

        let Some(to_block) = to_block.filter(|to_block| *to_block < self.bedrock_block) else {
            let Some(from_block) = from_block.filter(|from| *from < self.bedrock_block) else {
                return LogFilterRoute::Local;
            };

            // The pre-bedrock half ends at the last block before the transition.
            if self.fits_span_limit(from_block, self.bedrock_block.saturating_sub(1)) {
                return LogFilterRoute::Split;
            }

            // Too wide to split, so the whole range is left to local handling, which rejects it:
            // the range it was asked for is wider still.
            warn!(
                target: "rpc::historical",
                from_block,
                to_block = %to_bound.unwrap_or(BlockNumberOrTag::Latest),
                bedrock = self.bedrock_block,
                "log filter spans the bedrock transition and its pre-bedrock half exceeds the per-filter block limit; not splitting it"
            );
            return LogFilterRoute::Local;
        };

        // A range open below has no span to meter: the historical endpoint resolves the missing
        // bound to its own tip, the last pre-bedrock block.
        let Some(from_block) = from_block else { return LogFilterRoute::Historical };

        // Forwarding skips the local `eth_getLogs` handler, so honor its span limit here as well
        // rather than letting a caller aim an unmetered scan at the historical endpoint. A wider
        // range is left to local handling, which rejects it.
        if self.fits_span_limit(from_block, to_block) {
            LogFilterRoute::Historical
        } else {
            LogFilterRoute::Local
        }
    }

    /// Returns whether a block range sent to the historical endpoint stays within the node's
    /// `--rpc.max-blocks-per-filter` limit.
    fn fits_span_limit(&self, from_block: BlockNumber, to_block: BlockNumber) -> bool {
        self.max_blocks_per_filter.is_none_or(|max| to_block.saturating_sub(from_block) <= max)
    }

    /// Resolves an `eth_getLogs` range bound to the block number it names, or `None` when it
    /// names the chain tip — which, on a chain that went through the transition, is post-bedrock.
    ///
    /// An absent bound is the tip too, for both reth and the historical endpoint.
    fn resolve_filter_bound(&self, bound: BlockNumberOrTag) -> Option<BlockNumber> {
        match bound {
            BlockNumberOrTag::Number(_) | BlockNumberOrTag::Earliest => {
                self.provider.block_number_for_id(BlockId::Number(bound)).ok().flatten()
            }
            _ => None,
        }
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
            // Local handling answers a pre-bedrock log filter with an empty array, which a caller
            // cannot tell apart from "no matching logs". Fail the request instead, as the receipt
            // stitching path does.
            Err(err) if req.method_name() == "eth_getLogs" => {
                warn!(
                    target: "rpc::historical",
                    %err,
                    "failed to fetch pre-bedrock logs from historical endpoint"
                );
                return Some(pre_bedrock_logs_error(req.id.clone()));
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

    /// Serves an `eth_getLogs` filter that straddles the bedrock transition from both backends:
    /// the historical endpoint answers for the blocks below the transition and the local node for
    /// the rest. Their logs are concatenated, which keeps the response ordered by block number the
    /// way a single-backend response is.
    ///
    /// A failure on either side fails the whole request. Returning the surviving half alone
    /// would be a well-formed answer to a narrower query than the caller asked for, which is the
    /// silent gap this split exists to close.
    async fn serve_split_log_filter<S>(&self, inner: &S, req: Request<'_>) -> MethodResponse
    where
        S: RpcServiceT<MethodResponse = MethodResponse>,
    {
        let Some((pre_bedrock_filter, post_bedrock_filter)) = self.split_log_filter(&req) else {
            // `route_log_filter` parsed the same params, so this is unreachable.
            return inner.call(req).await;
        };

        debug!(
            target: "rpc::historical",
            %pre_bedrock_filter,
            %post_bedrock_filter,
            "splitting log filter at the bedrock transition"
        );

        let pre_bedrock_logs = match self
            .client
            .request::<_, Vec<serde_json::Value>>("eth_getLogs", (pre_bedrock_filter,))
            .await
        {
            Ok(logs) => logs,
            Err(err) => {
                warn!(
                    target: "rpc::historical",
                    %err,
                    "failed to fetch pre-bedrock logs from historical endpoint"
                );
                return pre_bedrock_logs_error(req.id.clone());
            }
        };

        let mut post_bedrock_req = req;
        let id = post_bedrock_req.id.clone();
        if let Err(err) = set_log_filter(&mut post_bedrock_req, &post_bedrock_filter) {
            warn!(target: "rpc::historical", %err, "failed to build the post-bedrock half of a split log filter");
            return pre_bedrock_logs_error(id);
        }

        let post_bedrock_response = inner.call(post_bedrock_req).await;
        if !post_bedrock_response.is_success() {
            return post_bedrock_response;
        }

        let Some(logs) = concat_logs(pre_bedrock_logs, &post_bedrock_response) else {
            warn!(
                target: "rpc::historical",
                "local node answered the post-bedrock half of a split log filter with a non-array result"
            );
            return pre_bedrock_logs_error(id);
        };

        let payload = jsonrpsee_types::ResponsePayload::success(logs).into();
        MethodResponse::response(id, payload, usize::MAX)
    }

    /// Splits an `eth_getLogs` filter at the bedrock transition into the filter for the historical
    /// endpoint and the filter for the local node, carrying the filter's other fields over
    /// untouched.
    fn split_log_filter(
        &self,
        req: &Request<'_>,
    ) -> Option<(serde_json::Value, serde_json::Value)> {
        let values: Vec<serde_json::Value> = req.params().parse().ok()?;
        let filter = values.into_iter().next().filter(serde_json::Value::is_object)?;

        let mut pre_bedrock = filter.clone();
        pre_bedrock["toBlock"] =
            serde_json::json!(BlockNumberOrTag::Number(self.bedrock_block.saturating_sub(1)));

        let mut post_bedrock = filter;
        post_bedrock["fromBlock"] = serde_json::json!(BlockNumberOrTag::Number(self.bedrock_block));

        Some((pre_bedrock, post_bedrock))
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

/// The JSON-RPC error returned when pre-bedrock logs could not be fetched.
///
/// The message is deliberately generic: an error's `Display` may embed raw responses from the
/// internal historical endpoint.
fn pre_bedrock_logs_error(id: jsonrpsee_types::Id<'_>) -> MethodResponse {
    MethodResponse::error(
        id,
        jsonrpsee_types::ErrorObject::owned(
            jsonrpsee_types::error::INTERNAL_ERROR_CODE,
            "failed to fetch pre-bedrock logs from historical endpoint",
            None::<()>,
        ),
    )
}

/// Replaces the filter of an `eth_getLogs` request, leaving its id and extensions in place.
fn set_log_filter(
    req: &mut Request<'_>,
    filter: &serde_json::Value,
) -> Result<(), serde_json::Error> {
    req.params = Some(Cow::Owned(serde_json::value::to_raw_value(&[filter])?));
    Ok(())
}

/// Concatenates pre-bedrock logs with the logs of a successful local `eth_getLogs` response.
///
/// Returns `None` if the local response doesn't carry an array of logs.
fn concat_logs(
    mut logs: Vec<serde_json::Value>,
    local: &MethodResponse,
) -> Option<Vec<serde_json::Value>> {
    let mut response = serde_json::from_str::<serde_json::Value>(local.as_json().get()).ok()?;
    let serde_json::Value::Array(local_logs) = response.get_mut("result")?.take() else {
        return None;
    };

    logs.extend(local_logs);
    Some(logs)
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
        _ => None,
    }
}

/// Parses the blocks an `eth_getLogs` filter is scoped to from the first parameter.
fn parse_log_filter_block_option(params: &Params<'_>) -> Option<FilterBlockOption> {
    let values: Vec<serde_json::Value> = params.parse().ok()?;
    let filter = serde_json::from_value::<Filter>(values.into_iter().next()?).ok()?;

    Some(filter.block_option)
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
    use rstest::rstest;
    use serde_json::json;
    use std::sync::Mutex;
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
            max_blocks_per_filter: None,
        }
    }

    /// A historical layer running with a `--rpc.max-blocks-per-filter` limit.
    fn limited_historical(max_blocks_per_filter: u64) -> HistoricalRpcInner<NoopProvider> {
        HistoricalRpcInner {
            max_blocks_per_filter: Some(max_blocks_per_filter),
            ..mocked_historical(Asserter::new())
        }
    }

    fn logs_request(filter: &str) -> Request<'static> {
        owned_request("eth_getLogs", &format!("[{filter}]"))
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
            max_blocks_per_filter: None,
        };
        (historical, requests)
    }

    fn recorded_requests(log: &RequestLog) -> Vec<(String, serde_json::Value)> {
        log.lock().unwrap().clone()
    }

    /// An [`RpcServiceT`] standing in for the local node: it answers every call with the same
    /// result or error, and records what it was asked for.
    #[derive(Clone)]
    struct LocalNode {
        result: Result<serde_json::Value, String>,
        requests: RequestLog,
    }

    impl LocalNode {
        fn answering(result: serde_json::Value) -> Self {
            Self { result: Ok(result), requests: RequestLog::default() }
        }

        fn failing(message: &str) -> Self {
            Self { result: Err(message.to_string()), requests: RequestLog::default() }
        }
    }

    impl RpcServiceT for LocalNode {
        type MethodResponse = MethodResponse;
        type NotificationResponse = MethodResponse;
        type BatchResponse = MethodResponse;

        fn call<'a>(&self, req: Request<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
            let params =
                serde_json::from_str(req.params().as_str().unwrap_or("[]")).unwrap_or_default();
            self.requests.lock().unwrap().push((req.method_name().to_string(), params));

            let result = self.result.clone();
            async move {
                match result {
                    Ok(result) => MethodResponse::response(
                        req.id.clone(),
                        jsonrpsee_types::ResponsePayload::success(result).into(),
                        usize::MAX,
                    ),
                    Err(message) => MethodResponse::error(
                        req.id.clone(),
                        jsonrpsee_types::ErrorObject::owned(
                            jsonrpsee_types::error::INTERNAL_ERROR_CODE,
                            message,
                            None::<()>,
                        ),
                    ),
                }
            }
        }

        // `async fn` would tie the returned future to the borrow of `self`, which the trait's
        // `+ 'a` bound doesn't allow.
        #[allow(clippy::manual_async_fn)]
        fn batch<'a>(&self, _req: Batch<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
            async { unimplemented!("batches don't reach the split path") }
        }

        #[allow(clippy::manual_async_fn)]
        fn notification<'a>(
            &self,
            _n: Notification<'a>,
        ) -> impl Future<Output = MethodResponse> + Send + 'a {
            async { unimplemented!("notifications don't reach the split path") }
        }
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

    /// A WARN-level event captured by [`WarnRecorder`]: its target and its field names.
    type WarnEvent = (String, Vec<String>);

    /// A [`tracing::Subscriber`] that records WARN-level events.
    struct WarnRecorder {
        warns: Arc<Mutex<Vec<WarnEvent>>>,
    }

    /// Collects the field names of a `tracing` event.
    #[derive(Default)]
    struct FieldNames(Vec<String>);

    impl tracing::field::Visit for FieldNames {
        fn record_debug(&mut self, field: &tracing::field::Field, _value: &dyn std::fmt::Debug) {
            self.0.push(field.name().to_string());
        }
    }

    impl tracing::Subscriber for WarnRecorder {
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
                let mut fields = FieldNames::default();
                event.record(&mut fields);
                self.warns.lock().unwrap().push((event.metadata().target().to_string(), fields.0));
            }
        }

        fn enter(&self, _span: &tracing::span::Id) {}

        fn exit(&self, _span: &tracing::span::Id) {}
    }

    /// Runs `f` and returns the WARN events it emitted.
    fn warn_events_while<T>(f: impl FnOnce() -> T) -> Vec<WarnEvent> {
        let warns = Arc::new(Mutex::new(Vec::new()));
        tracing::subscriber::with_default(WarnRecorder { warns: warns.clone() }, f);
        warns.lock().unwrap().clone()
    }

    /// Runs `fut` on a current-thread runtime and returns the number of WARN events emitted.
    fn warns_during<F: Future<Output = ()>>(fut: F) -> usize {
        warn_events_while(|| {
            tokio::runtime::Builder::new_current_thread().build().unwrap().block_on(fut)
        })
        .len()
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

    /// Tests which backend serves each `eth_getLogs` filter, using the OP Mainnet blocks from the
    /// bug report: bedrock is 105235063, so `0x645c276` (105235062) is pre-bedrock and `0x645d1d8`
    /// (105239000) is post-bedrock.
    #[rstest]
    #[case::pre_bedrock_block(
        r#"{"fromBlock":"0x645c276","toBlock":"0x645c276"}"#,
        LogFilterRoute::Historical
    )]
    #[case::genesis_to_pre_bedrock(
        r#"{"fromBlock":"0x0","toBlock":"0x645c276"}"#,
        LogFilterRoute::Historical
    )]
    #[case::upper_bound_only(r#"{"toBlock":"0x645c276"}"#, LogFilterRoute::Historical)]
    #[case::to_earliest(r#"{"toBlock":"earliest"}"#, LogFilterRoute::Historical)]
    #[case::with_address(
        r#"{"toBlock":"0x645c276","address":"0x5e61a079a178f0e5784107a4963baae0c5a680c6"}"#,
        LogFilterRoute::Historical
    )]
    // A block hash the provider doesn't know is assumed pre-bedrock, as for every other
    // hash-addressed method.
    #[case::unknown_block_hash(
        r#"{"blockHash":"0xdbdfa0f88b2cf815fdc1621bd20c2bd2b0eed4f0c56c9be2602957b5a60ec702"}"#,
        LogFilterRoute::Historical
    )]
    #[case::post_bedrock_block(
        r#"{"fromBlock":"0x645d1d8","toBlock":"0x645d1d8"}"#,
        LogFilterRoute::Local
    )]
    #[case::from_latest(r#"{"fromBlock":"latest","toBlock":"0x645d1d8"}"#, LogFilterRoute::Local)]
    #[case::no_bounds(r#"{}"#, LogFilterRoute::Local)]
    // Straddling filters: an explicit post-bedrock upper bound, and every way of naming the
    // chain tip, which is where the everyday "all logs for this contract" query lands.
    #[case::crossing_bedrock(
        r#"{"fromBlock":"0x645c276","toBlock":"0x645d1d8"}"#,
        LogFilterRoute::Split
    )]
    #[case::genesis_to_latest(r#"{"fromBlock":"0x0","toBlock":"latest"}"#, LogFilterRoute::Split)]
    #[case::earliest_to_finalized(
        r#"{"fromBlock":"earliest","toBlock":"finalized"}"#,
        LogFilterRoute::Split
    )]
    #[case::open_ended(r#"{"fromBlock":"0x0"}"#, LogFilterRoute::Split)]
    fn routes_log_filters_by_bedrock_transition(
        #[case] filter: &str,
        #[case] route: LogFilterRoute,
    ) {
        let historical = mocked_historical(Asserter::new());
        assert_eq!(historical.route_log_filter(&logs_request(filter)), route);
    }

    /// Tests that the batch path takes over for every filter the historical endpoint has a part
    /// in, so a split filter inside a batch isn't handed to the inner service whole.
    #[rstest]
    #[case::historical(r#"{"toBlock":"0x645c276"}"#, true)]
    #[case::split(r#"{"fromBlock":"0x645c276","toBlock":"0x645d1d8"}"#, true)]
    #[case::local(r#"{"fromBlock":"0x645d1d8","toBlock":"0x645d1d8"}"#, false)]
    fn should_forward_covers_split_log_filters(#[case] filter: &str, #[case] forwarded: bool) {
        let historical = mocked_historical(Asserter::new());
        assert_eq!(historical.should_forward_request(&logs_request(filter)), forwarded);
    }

    /// Tests that a request without any filter parameter is left to local handling.
    #[test]
    fn does_not_forward_log_request_without_params() {
        let historical = mocked_historical(Asserter::new());
        assert!(!historical.should_forward_request(&owned_request("eth_getLogs", "[]")));
    }

    /// Tests that forwarding doesn't bypass `--rpc.max-blocks-per-filter`: a pre-bedrock range
    /// wider than the limit stays local, where reth rejects it.
    #[test]
    fn does_not_forward_log_filters_wider_than_the_span_limit() {
        let historical = limited_historical(100_000);
        let range_ending_pre_bedrock = |span: u64| {
            let to = 105_235_062u64;
            logs_request(&format!(r#"{{"fromBlock":"{:#x}","toBlock":"{to:#x}"}}"#, to - span))
        };

        assert_eq!(
            historical.route_log_filter(&range_ending_pre_bedrock(100_000)),
            LogFilterRoute::Historical
        );
        assert_eq!(
            historical.route_log_filter(&range_ending_pre_bedrock(100_001)),
            LogFilterRoute::Local
        );

        // A range open below has no span to meter: the historical endpoint resolves the missing
        // bound to its own tip, so it scans a single block.
        assert_eq!(
            historical.route_log_filter(&logs_request(r#"{"toBlock":"0x645c276"}"#)),
            LogFilterRoute::Historical
        );
    }

    /// Tests that the span limit applies to the pre-bedrock half of a split as well, so a split
    /// can't be used to aim an unmetered scan at the historical endpoint.
    #[test]
    fn does_not_split_log_filters_whose_pre_bedrock_half_exceeds_the_span_limit() {
        let historical = limited_historical(100_000);
        // The last pre-bedrock block is 105235062, so the pre-bedrock half spans `span` blocks.
        let range_crossing_bedrock = |span: u64| {
            logs_request(&format!(
                r#"{{"fromBlock":"{:#x}","toBlock":"latest"}}"#,
                105_235_062 - span
            ))
        };

        assert_eq!(
            historical.route_log_filter(&range_crossing_bedrock(100_000)),
            LogFilterRoute::Split
        );
        assert_eq!(
            historical.route_log_filter(&range_crossing_bedrock(100_001)),
            LogFilterRoute::Local
        );
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

    /// Tests that a filter crossing bedrock that is too wide to split warns about the
    /// pre-bedrock logs local handling can't reach. The upper bound may be an explicit
    /// post-bedrock block or anything tracking the chain tip, which is where the everyday "all
    /// logs for this contract" query lands.
    #[rstest]
    #[case::to_post_bedrock_block(r#"{"fromBlock":"0x63d2fc6","toBlock":"0x645d1d8"}"#)]
    #[case::to_latest(r#"{"fromBlock":"0x0","toBlock":"latest"}"#)]
    #[case::to_finalized(r#"{"fromBlock":"earliest","toBlock":"finalized"}"#)]
    #[case::open_ended(r#"{"fromBlock":"0x0"}"#)]
    fn unsplittable_crossing_bedrock_log_filter_warns(#[case] filter: &str) {
        let warns = warn_events_while(|| {
            let historical = limited_historical(100_000);
            assert_eq!(historical.route_log_filter(&logs_request(filter)), LogFilterRoute::Local);
        });

        let [(target, fields)] = warns.as_slice() else {
            panic!("expected exactly one warning, got {warns:?}");
        };
        assert_eq!(target, "rpc::historical");
        for field in ["from_block", "to_block", "bedrock"] {
            assert!(fields.contains(&field.to_string()), "missing {field} in {fields:?}");
        }
    }

    /// Tests that a filter the layer can serve in full stays quiet: one wholly above bedrock,
    /// one whose lower bound tracks the chain tip and so is above it too, and one straddling the
    /// transition that is narrow enough to split.
    #[rstest]
    #[case::post_bedrock_range(r#"{"fromBlock":"0x645d1d8","toBlock":"0x645d1d9"}"#)]
    #[case::from_latest(r#"{"fromBlock":"latest","toBlock":"0x645d1d8"}"#)]
    #[case::no_bounds(r#"{}"#)]
    #[case::splittable(r#"{"fromBlock":"0x645c276","toBlock":"0x645d1d8"}"#)]
    fn servable_log_filter_does_not_warn(#[case] filter: &str) {
        let warns = warn_events_while(|| {
            let historical = limited_historical(100_000);
            historical.route_log_filter(&logs_request(filter));
        });

        assert!(warns.is_empty(), "{warns:?}");
    }

    /// Tests that a failed `eth_getLogs` forward returns a JSON-RPC error rather than falling
    /// through to local handling, which would answer a pre-bedrock filter with an empty array
    /// that looks like "no matching logs".
    #[tokio::test]
    async fn log_forward_failure_returns_error() {
        let asserter = Asserter::new();
        asserter.push_failure_msg("historical endpoint unreachable");

        let historical = mocked_historical(asserter);
        let req = logs_request(r#"{"toBlock":"0x645c276"}"#);
        let resp = historical.forward_to_historical(&req).await.unwrap();

        assert!(!resp.is_success());
        assert_eq!(resp.as_error_code(), Some(jsonrpsee_types::error::INTERNAL_ERROR_CODE));
        assert_eq!(
            error_message_of(&resp),
            "failed to fetch pre-bedrock logs from historical endpoint"
        );
    }

    /// Tests that a filter straddling the transition is answered from both backends: each is
    /// asked only for its own half, with the filter's other fields carried over, and the logs
    /// are concatenated pre-bedrock first so the response stays ordered by block number.
    #[tokio::test]
    async fn splits_log_filter_across_both_backends() {
        let pre_bedrock_log = json!({"blockNumber": "0x645c276", "data": "0x01"});
        let post_bedrock_log = json!({"blockNumber": "0x645d1d8", "data": "0x02"});
        let asserter = Asserter::new();
        asserter.push_success(&json!([pre_bedrock_log]));

        let (historical, forwarded) = recorded_historical(asserter);
        let local = LocalNode::answering(json!([post_bedrock_log]));
        let address = "0x5e61a079a178f0e5784107a4963baae0c5a680c6";
        let req = logs_request(&format!(
            r#"{{"fromBlock":"0x645c276","toBlock":"0x645d1d8","address":"{address}"}}"#
        ));

        let resp = historical.serve_split_log_filter(&local, req).await;

        assert!(resp.is_success());
        assert_eq!(result_of(&resp), json!([pre_bedrock_log, post_bedrock_log]));
        assert_eq!(
            recorded_requests(&forwarded),
            vec![(
                "eth_getLogs".to_string(),
                // bedrock is 105235063, so the pre-bedrock half ends at 0x645c276.
                json!([{"fromBlock": "0x645c276", "toBlock": "0x645c276", "address": address}])
            )]
        );
        assert_eq!(
            recorded_requests(&local.requests),
            vec![(
                "eth_getLogs".to_string(),
                json!([{"fromBlock": "0x645c277", "toBlock": "0x645d1d8", "address": address}])
            )]
        );
    }

    /// Tests that an open-ended filter splits into a bounded pre-bedrock query and a local one
    /// that keeps following the chain tip.
    #[tokio::test]
    async fn splits_open_ended_log_filter() {
        let asserter = Asserter::new();
        asserter.push_success(&json!([]));

        let (historical, forwarded) = recorded_historical(asserter);
        let local = LocalNode::answering(json!([]));
        let resp =
            historical.serve_split_log_filter(&local, logs_request(r#"{"fromBlock":"0x0"}"#)).await;

        assert!(resp.is_success());
        assert_eq!(result_of(&resp), json!([]));
        assert_eq!(
            recorded_requests(&forwarded),
            vec![(
                "eth_getLogs".to_string(),
                json!([{"fromBlock": "0x0", "toBlock": "0x645c276"}])
            )]
        );
        assert_eq!(
            recorded_requests(&local.requests),
            vec![("eth_getLogs".to_string(), json!([{"fromBlock": "0x645c277"}]))]
        );
    }

    /// Tests that a failing historical half fails the whole split without querying the local
    /// node: the post-bedrock logs on their own look like a complete answer.
    #[tokio::test]
    async fn split_fails_when_the_historical_half_fails() {
        let asserter = Asserter::new();
        asserter.push_failure_msg("historical endpoint unreachable");

        let historical = mocked_historical(asserter);
        let local = LocalNode::answering(json!([{"blockNumber": "0x645d1d8"}]));
        let req = logs_request(r#"{"fromBlock":"0x645c276","toBlock":"latest"}"#);

        let resp = historical.serve_split_log_filter(&local, req).await;

        assert!(!resp.is_success());
        assert_eq!(resp.as_error_code(), Some(jsonrpsee_types::error::INTERNAL_ERROR_CODE));
        assert_eq!(
            error_message_of(&resp),
            "failed to fetch pre-bedrock logs from historical endpoint"
        );
        assert!(recorded_requests(&local.requests).is_empty());
    }

    /// Tests that a failing local half fails the whole split, passing the local node's own error
    /// through — it is the one that knows why the post-bedrock query failed.
    #[tokio::test]
    async fn split_fails_when_the_local_half_fails() {
        let asserter = Asserter::new();
        asserter.push_success(&json!([{"blockNumber": "0x645c276"}]));

        let historical = mocked_historical(asserter);
        let local = LocalNode::failing("query exceeds max block range");
        let req = logs_request(r#"{"fromBlock":"0x645c276","toBlock":"latest"}"#);

        let resp = historical.serve_split_log_filter(&local, req).await;

        assert!(!resp.is_success());
        assert_eq!(error_message_of(&resp), "query exceeds max block range");
    }
}
