//! This is our custom implementation of validator struct

use crate::{
    InvalidCrossTx,
    interop_filter::{
        ExecutingDescriptor, InteropTxValidatorError,
        metrics::{EndpointMetrics, InteropMetrics},
        parse_access_list_items_to_inbox_entries,
    },
    maintain::FAILSAFE_HEARTBEAT_INTERVAL,
};
use alloy_consensus::Transaction;
use alloy_eips::eip2930::AccessList;
use alloy_primitives::{B256, TxHash};
use alloy_rpc_client::ReqwestClient;
use futures_util::{
    FutureExt, Stream,
    future::BoxFuture,
    stream::{self, FuturesUnordered, StreamExt},
};
use op_alloy_consensus::interop::SafetyLevel;
use reth_transaction_pool::PoolTransaction;
use std::{
    borrow::Cow,
    future::IntoFuture,
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    time::{Duration, Instant},
};
use tracing::{error, info, trace};

/// The default request timeout to use
pub const DEFAULT_REQUEST_TIMEOUT: Duration = Duration::from_secs(2);

/// Client for interop filter RPC requests.
#[derive(Debug, Clone)]
pub struct InteropFilterClient {
    /// Stores type's data.
    inner: Arc<InteropFilterClientInner>,
}

impl InteropFilterClient {
    /// Returns a new [`InteropFilterClientBuilder`] for the given interop filter endpoints.
    pub const fn builder(
        interop_endpoints: Vec<String>,
        chain_id: u64,
    ) -> InteropFilterClientBuilder {
        InteropFilterClientBuilder::new(interop_endpoints, chain_id)
    }

    /// Returns the configured request timeout.
    pub fn timeout(&self) -> Duration {
        self.inner.timeout
    }

    /// Returns configured minimum safety level. See [`InteropFilterClient`].
    pub fn safety(&self) -> SafetyLevel {
        self.inner.safety
    }

    /// Executes an `interop_checkAccessList` against every configured endpoint concurrently and
    /// combines the results by quorum agreement.
    ///
    /// A "response" counts only when an endpoint returns a definitive verdict (valid, or a known
    /// validation rejection). Transport errors and timeouts are non-responses. The aggregator
    /// decides as soon as `min_responses` definitive verdicts arrive and does not await slower
    /// endpoints; dropping the in-flight futures cancels their requests.
    ///
    /// Returns `Ok(())` iff `min_responses` definitive verdicts were collected and all agreed
    /// valid. All-invalid yields the real rejection reason; a mix yields
    /// [`Disagreement`](InteropTxValidatorError::Disagreement); too few definitive verdicts
    /// yields [`QuorumNotReached`](InteropTxValidatorError::QuorumNotReached) (fail closed).
    pub async fn check_access_list(
        &self,
        inbox_entries: &[B256],
        executing_descriptor: ExecutingDescriptor,
    ) -> Result<(), InteropTxValidatorError> {
        let mut futs: FuturesUnordered<_> = self
            .inner
            .endpoints
            .iter()
            .map(|endpoint| {
                CheckAccessListRequest {
                    client: endpoint.client.clone(),
                    inbox_entries: Cow::Borrowed(inbox_entries),
                    executing_descriptor,
                    timeout: self.inner.timeout,
                    safety: self.inner.safety,
                    metrics: self.inner.metrics.clone(),
                    endpoint_metrics: endpoint.metrics.clone(),
                }
                .into_future()
            })
            .collect();

        let (mut valid, mut invalid) = (0usize, 0usize);
        let mut first_invalid: Option<InteropTxValidatorError> = None;
        while let Some(result) = futs.next().await {
            match result {
                Ok(()) => valid += 1,
                // Failsafe on any endpoint is a hard rejection: short-circuit immediately, even
                // if the quorum was already met by valid responses. Dropping `futs` cancels the
                // remaining in-flight requests.
                Err(e) if e.is_failsafe() => return Err(self.reject_failsafe()),
                Err(e) if e.is_definitive_invalid() => {
                    invalid += 1;
                    first_invalid.get_or_insert(e);
                }
                // Non-response (timeout / transport): does not count toward quorum.
                Err(_) => continue,
            }
            // Quorum reached: stop without awaiting not-yet-responded endpoints. But a failsafe
            // that has *already* arrived must still reject, so non-blocking-drain the responses
            // that are ready right now first. We never await endpoints that haven't responded
            // (fast-accept preserved); dropping `futs` afterwards cancels them.
            if valid + invalid >= self.inner.min_responses {
                while let Some(ready) = futs.next().now_or_never().flatten() {
                    match ready {
                        Err(e) if e.is_failsafe() => return Err(self.reject_failsafe()),
                        Ok(()) => valid += 1,
                        Err(e) if e.is_definitive_invalid() => {
                            invalid += 1;
                            first_invalid.get_or_insert(e);
                        }
                        Err(_) => {}
                    }
                }
                break;
            }
        }
        // Dropping `futs` here cancels any in-flight slow requests.
        self.inner.metrics.record_quorum_outcome(valid, invalid, self.inner.min_responses);

        if valid + invalid < self.inner.min_responses {
            self.log_degraded_quorum(valid + invalid);
            return Err(InteropTxValidatorError::QuorumNotReached {
                received: valid + invalid,
                required: self.inner.min_responses,
            });
        }
        match (valid, invalid) {
            (_, 0) => Ok(()),
            // All collected verdicts agree invalid: surface the real rejection reason.
            (0, _) => Err(first_invalid.expect("invalid > 0 implies a recorded invalid verdict")),
            (valid, invalid) => {
                error!(
                    target: "txpool::interop",
                    valid,
                    invalid,
                    "interop endpoints disagreed; rejecting"
                );
                Err(InteropTxValidatorError::Disagreement { valid, invalid })
            }
        }
    }

    /// Extracts commitments from access list entries pointing to the cross-L2 inbox and validates
    /// them against the interop filter.
    ///
    /// If commitment present pre-interop tx rejected.
    ///
    /// Returns:
    /// None - if tx is not cross chain,
    /// Some(Ok(()) - if tx is valid cross chain,
    /// Some(Err(e)) - if tx is not valid or interop is not active
    pub async fn is_valid_cross_tx(
        &self,
        access_list: Option<&AccessList>,
        hash: &TxHash,
        timestamp: u64,
        timeout: Option<u64>,
        is_interop_active: bool,
    ) -> Option<Result<(), InvalidCrossTx>> {
        // We don't need to check for deposit transaction in here, because they won't come from
        // txpool
        let access_list = access_list?;
        let inbox_entries = parse_access_list_items_to_inbox_entries(access_list.iter())
            .copied()
            .collect::<Vec<_>>();
        if inbox_entries.is_empty() {
            return None;
        }

        // Interop check
        if !is_interop_active {
            // No cross chain tx allowed before interop
            return Some(Err(InvalidCrossTx::CrossChainTxPreInterop));
        }

        // Fast-path: reject immediately if failsafe is active (no RPC round-trip)
        if self.is_failsafe_enabled() {
            return Some(Err(InvalidCrossTx::FailsafeEnabled));
        }

        if let Err(err) = self
            .check_access_list(
                inbox_entries.as_slice(),
                ExecutingDescriptor::new(self.inner.chain_id, timestamp, timeout),
            )
            .await
        {
            // A failsafe reported by any endpoint maps to the same rejection the fast-path uses.
            if err.is_failsafe() {
                trace!(target: "txpool", hash=%hash, "Cross chain transaction rejected: endpoint failsafe active");
                return Some(Err(InvalidCrossTx::FailsafeEnabled));
            }
            self.inner.metrics.increment_metrics_for_error(&err);
            trace!(target: "txpool", hash=%hash, err=%err, "Cross chain transaction invalid");
            return Some(Err(InvalidCrossTx::ValidationError(err)));
        }
        Some(Ok(()))
    }

    /// Records a hard failsafe rejection and flips the cached gate (and gauge) on immediately, so a
    /// failsafe detected on a check stops admission and is visible on the dashboard right away
    /// rather than only after the next ~1s failsafe poll. The poll re-confirms or clears it.
    fn reject_failsafe(&self) -> InteropTxValidatorError {
        self.inner.metrics.record_failsafe_reject();
        self.apply_failsafe_state(true);
        InteropTxValidatorError::FailsafeEnabled
    }

    /// Logs a rate-limited line when a check fails closed because too few endpoints reached the
    /// quorum. A degraded quorum silently rejects every interop tx, so without this the only signal
    /// is the `quorum_reject_not_reached` counter; the log makes the halted state visible. Logged
    /// at most once per [`FAILSAFE_HEARTBEAT_INTERVAL`] (and on the first occurrence) to avoid
    /// per-tx spam, and at `info` because the rejection itself is expected, by-design
    /// fail-closed behavior.
    fn log_degraded_quorum(&self, received: usize) {
        let now = self.inner.created_at.elapsed().as_millis() as u64;
        let interval = FAILSAFE_HEARTBEAT_INTERVAL.as_millis() as u64;
        let last = self.inner.last_degraded_log_ms.load(Ordering::Relaxed);
        let due = last == 0 || now.saturating_sub(last) >= interval;
        if due &&
            self.inner
                .last_degraded_log_ms
                .compare_exchange(last, now.max(1), Ordering::Relaxed, Ordering::Relaxed)
                .is_ok()
        {
            info!(
                target: "txpool::interop",
                received,
                required = self.inner.min_responses,
                endpoints = self.inner.endpoints.len(),
                "interop failing closed: too few endpoints returned a definitive verdict to reach \
                 quorum; all interop transactions are rejected until enough endpoints respond"
            );
        }
    }

    /// Returns the cached failsafe state.
    pub fn is_failsafe_enabled(&self) -> bool {
        self.inner.failsafe_enabled.load(Ordering::Acquire)
    }

    /// Applies a freshly polled failsafe state to the cached flag (read live by the admission
    /// fast-path and the block-event handler) and the gauge. Split out of [`Self::query_failsafe`]
    /// so the live, restart-free state transition can be exercised in tests without an RPC.
    pub(crate) fn apply_failsafe_state(&self, enabled: bool) {
        self.inner.failsafe_enabled.store(enabled, Ordering::Release);
        self.inner.metrics.set_failsafe_enabled(enabled);
    }

    /// Queries every configured interop filter for failsafe state and caches the OR across the
    /// endpoints. Calls `admin_getFailsafeEnabled` on each endpoint concurrently and short-circuits
    /// on the first `true` (the OR is decisive, so a single slow endpoint must not delay turning
    /// the gate on).
    ///
    /// Clearing the gate (to `false`) requires *every* endpoint to have replied `false`. If some
    /// endpoints did not answer and none reported failsafe, the result is "don't know": the cached
    /// state is left unchanged rather than cleared, because a silent endpoint might itself be the
    /// one in failsafe. Turning the gate *on* never requires unanimity. This matters once
    /// `min_responses` is lowered for availability — with the unanimity default a silent endpoint
    /// also fails the check closed, so the partial-information case never decided admission.
    ///
    /// If no endpoint replies, the cache is left unchanged and an error is returned (matching the
    /// previous single-endpoint behavior).
    pub async fn query_failsafe(&self) -> Result<bool, InteropTxValidatorError> {
        let endpoint_count = self.inner.endpoints.len();
        let mut futs: FuturesUnordered<_> = self
            .inner
            .endpoints
            .iter()
            .map(|endpoint| {
                tokio::time::timeout(
                    self.inner.timeout,
                    endpoint.client.request::<_, bool>("admin_getFailsafeEnabled", ()),
                )
            })
            .collect();

        let mut replied = 0usize;
        let mut enabled = false;
        while let Some(res) = futs.next().await {
            if let Ok(Ok(v)) = res {
                replied += 1;
                // Any endpoint reporting failsafe is decisive: turn the gate on immediately.
                if v {
                    enabled = true;
                    break;
                }
            }
        }

        if replied == 0 {
            // No endpoint answered: leave the cache unchanged (the single-endpoint behavior).
            return Err(InteropTxValidatorError::Timeout(self.inner.timeout.as_secs()));
        }
        if !enabled && replied < endpoint_count {
            // Some endpoints did not answer and none reported failsafe. We cannot confirm failsafe
            // is off — a silent endpoint might itself be in failsafe — so leave the cached gate
            // unchanged rather than clearing it on partial information.
            return Ok(self.is_failsafe_enabled());
        }
        // Decisive: an endpoint reported failsafe, or every endpoint replied and all said off.
        self.apply_failsafe_state(enabled);
        Ok(enabled)
    }

    /// Creates a stream that revalidates interop transactions against the interop filter.
    /// Returns
    /// An implementation of `Stream` that is `Send`-able and tied to the lifetime `'a` of `self`.
    /// Each item yielded by the stream is a tuple `(TItem, Option<Result<(), InvalidCrossTx>>)`.
    ///   - The first element is the original `TItem` that was revalidated.
    ///   - The second element is the `Option<Result<(), InvalidCrossTx>>` describes the outcome
    ///     - `None`: Transaction was not identified as a cross-chain candidate by initial checks.
    ///     - `Some(Ok(()))`: Interop filter confirmed the transaction is valid.
    ///     - `Some(Err(InvalidCrossTx))`: Interop filter indicated the transaction is invalid.
    pub fn revalidate_interop_txs_stream<'a, TItem, InputIter>(
        &'a self,
        txs_to_revalidate: InputIter,
        current_timestamp: u64,
        revalidation_window: u64,
        max_concurrent_queries: usize,
    ) -> impl Stream<Item = (TItem, Option<Result<(), InvalidCrossTx>>)> + Send + 'a
    where
        InputIter: IntoIterator<Item = TItem> + Send + 'a,
        InputIter::IntoIter: Send + 'a,
        TItem: PoolTransaction + Transaction + Send,
    {
        stream::iter(txs_to_revalidate.into_iter().map(move |tx_item| {
            let client_for_async_task = self.clone();

            async move {
                let validation_result = client_for_async_task
                    .is_valid_cross_tx(
                        tx_item.access_list(),
                        tx_item.hash(),
                        current_timestamp,
                        Some(revalidation_window),
                        true,
                    )
                    .await;

                // return the original transaction paired with its validation result.
                (tx_item, validation_result)
            }
        }))
        .buffered(max_concurrent_queries)
    }
}

/// A single configured interop filter endpoint and its per-endpoint metrics.
#[derive(Debug)]
pub(crate) struct Endpoint {
    /// RPC client for this endpoint.
    client: ReqwestClient,
    /// Metrics labeled with this endpoint's index.
    metrics: EndpointMetrics,
}

/// Holds interop RPC data. Inner type of [`InteropFilterClient`].
#[derive(Debug)]
pub(crate) struct InteropFilterClientInner {
    /// Interop filter endpoints queried concurrently for each check.
    endpoints: Vec<Endpoint>,
    /// Minimum number of definitive verdicts required to decide a check.
    min_responses: usize,
    /// The chain ID of the executing chain
    chain_id: u64,
    /// The default
    safety: SafetyLevel,
    /// The default request timeout
    timeout: Duration,
    /// Metrics for tracking interop RPC operations.
    metrics: InteropMetrics,
    /// Cached failsafe state (OR across endpoints), polled by the background failsafe task.
    failsafe_enabled: AtomicBool,
    /// Reference instant for rate-limiting the degraded-quorum log.
    created_at: Instant,
    /// Millis (since [`created_at`](Self::created_at)) of the last degraded-quorum log; `0` means
    /// never. Rate-limits [`log_degraded_quorum`](InteropFilterClient::log_degraded_quorum).
    last_degraded_log_ms: AtomicU64,
}

/// Builds [`InteropFilterClient`].
#[derive(Debug)]
pub struct InteropFilterClientBuilder {
    /// Interop filter RPC endpoints, queried concurrently for each check.
    endpoints: Vec<String>,
    /// Minimum number of definitive verdicts required to decide a check. When unset, defaults to
    /// the number of endpoints (unanimity).
    min_responses: Option<usize>,
    /// The chain ID of the executing chain.
    chain_id: u64,
    /// Timeout for requests.
    ///
    /// NOTE: this timeout is only effective if it's shorter than the timeout configured for the
    /// underlying [`ReqwestClient`].
    timeout: Duration,
    /// Minimum [`SafetyLevel`] of cross-chain transactions accepted by this client.
    safety: SafetyLevel,
}

impl InteropFilterClientBuilder {
    /// Creates a new builder for the given interop filter endpoints.
    pub const fn new(interop_endpoints: Vec<String>, chain_id: u64) -> Self {
        Self {
            endpoints: interop_endpoints,
            min_responses: None,
            chain_id,
            timeout: DEFAULT_REQUEST_TIMEOUT,
            safety: SafetyLevel::CrossUnsafe,
        }
    }

    /// Configures a custom timeout
    pub const fn timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Sets minimum safety level to accept for cross chain transactions.
    pub const fn minimum_safety(mut self, min_safety: SafetyLevel) -> Self {
        self.safety = min_safety;
        self
    }

    /// Sets the minimum number of definitive verdicts required to decide a check. Defaults to the
    /// number of endpoints (unanimity) when not set.
    pub const fn min_responses(mut self, min_responses: usize) -> Self {
        self.min_responses = Some(min_responses);
        self
    }

    /// Creates a new interop RPC client.
    ///
    /// Panics if no endpoints are configured, or if `min_responses` is 0 or exceeds the number of
    /// endpoints. This is a startup-boundary validation, matching the existing
    /// `.expect("building interop filter client")` failure mode.
    pub async fn build(self) -> InteropFilterClient {
        let Self { endpoints, min_responses, chain_id, timeout, safety } = self;

        assert!(!endpoints.is_empty(), "interop filter client requires at least one endpoint");
        let min_responses = min_responses.unwrap_or(endpoints.len());
        assert!(
            min_responses >= 1 && min_responses <= endpoints.len(),
            "interop filter min_responses ({min_responses}) must be between 1 and the number of \
             endpoints ({})",
            endpoints.len()
        );

        let mut clients = Vec::with_capacity(endpoints.len());
        for (idx, endpoint) in endpoints.iter().enumerate() {
            let client = ReqwestClient::builder()
                .connect(endpoint.as_str())
                .await
                .expect("building interop filter client");
            // Label by index, not the raw URL: interop-http URLs can carry basic-auth credentials.
            let metrics = EndpointMetrics::for_endpoint(idx);
            clients.push(Endpoint { client, metrics });
        }

        InteropFilterClient {
            inner: Arc::new(InteropFilterClientInner {
                endpoints: clients,
                min_responses,
                chain_id,
                safety,
                timeout,
                metrics: InteropMetrics::default(),
                failsafe_enabled: AtomicBool::new(false),
                created_at: Instant::now(),
                last_degraded_log_ms: AtomicU64::new(0),
            }),
        }
    }
}

/// A Request future that issues an `interop_checkAccessList` request.
#[derive(Debug, Clone)]
pub struct CheckAccessListRequest<'a> {
    client: ReqwestClient,
    inbox_entries: Cow<'a, [B256]>,
    executing_descriptor: ExecutingDescriptor,
    timeout: Duration,
    safety: SafetyLevel,
    metrics: InteropMetrics,
    endpoint_metrics: EndpointMetrics,
}

impl<'a> CheckAccessListRequest<'a> {
    /// Configures the timeout to use for the request if any.
    pub const fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Configures the [`SafetyLevel`] for this request
    pub const fn with_safety(mut self, safety: SafetyLevel) -> Self {
        self.safety = safety;
        self
    }
}

impl<'a> IntoFuture for CheckAccessListRequest<'a> {
    type Output = Result<(), InteropTxValidatorError>;
    type IntoFuture = BoxFuture<'a, Self::Output>;

    fn into_future(self) -> Self::IntoFuture {
        let Self {
            client,
            inbox_entries,
            executing_descriptor,
            timeout,
            safety,
            metrics,
            endpoint_metrics,
        } = self;
        Box::pin(async move {
            let start = Instant::now();

            let result = tokio::time::timeout(
                timeout,
                client.request(
                    "interop_checkAccessList",
                    (inbox_entries, safety, executing_descriptor),
                ),
            )
            .await;
            let elapsed = start.elapsed();
            metrics.record_interop_query(elapsed);
            endpoint_metrics.record_query(elapsed);

            let verdict = result
                .map_err(|_| InteropTxValidatorError::Timeout(timeout.as_secs()))
                .and_then(|r| r.map_err(InteropTxValidatorError::from_json_rpc));

            match &verdict {
                Ok(()) => endpoint_metrics.record_valid(),
                Err(e) if e.is_definitive_invalid() => endpoint_metrics.record_invalid(),
                Err(_) => endpoint_metrics.record_unavailable(),
            }
            verdict
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::interop_filter::CROSS_L2_INBOX_ADDRESS;
    use alloy_eips::eip2930::AccessListItem;

    async fn test_client() -> InteropFilterClient {
        InteropFilterClient::builder(vec!["http://localhost:8545".to_string()], 1).build().await
    }

    /// An access list that marks the tx as cross-chain (one entry targeting the cross-L2 inbox).
    fn interop_access_list() -> AccessList {
        AccessList(vec![AccessListItem {
            address: CROSS_L2_INBOX_ADDRESS,
            storage_keys: vec![B256::ZERO],
        }])
    }

    /// The failsafe gate is a live, runtime-mutable flag: enabling it rejects interop txs on the
    /// fast-path (no RPC), and clearing it flips the gate back in-process — so admission resumes
    /// without restarting reth. This is the property the `poll_failsafe` loop relies on.
    #[tokio::test]
    async fn failsafe_gate_is_live_and_resumes_without_restart() {
        let client = test_client().await;
        let access_list = interop_access_list();
        let hash = TxHash::ZERO;

        // Filter signals failsafe enabled -> admission fast-path rejects immediately, no RPC.
        client.apply_failsafe_state(true);
        assert!(client.is_failsafe_enabled());
        let outcome = client.is_valid_cross_tx(Some(&access_list), &hash, 0, None, true).await;
        assert!(
            matches!(outcome, Some(Err(InvalidCrossTx::FailsafeEnabled))),
            "failsafe-enabled interop tx should be rejected on the fast-path, got {outcome:?}"
        );

        // Filter clears failsafe -> the cached gate flips back in-process on the next poll, so
        // interop admission resumes without restarting the execution layer.
        client.apply_failsafe_state(false);
        assert!(!client.is_failsafe_enabled(), "failsafe gate must clear at runtime, no restart");
    }

    use jsonrpsee::types::ErrorObjectOwned;
    use jsonrpsee_server::{RpcModule, ServerBuilder, ServerHandle};
    use op_alloy_rpc_types::SuperchainDAError;
    use std::net::SocketAddr;

    /// Behavior of a mock interop filter endpoint for `interop_checkAccessList`.
    #[derive(Clone, Copy, Debug)]
    enum Verdict {
        /// Responds immediately with a valid result.
        Valid,
        /// Responds immediately with a definitive validation rejection (maps to `InvalidEntry`).
        Invalid,
        /// Responds immediately with a generic `-32602` rejection that is not a `SuperchainDAError`
        /// code (e.g. "failed to parse access entry"). Maps to `Rejected`, a definitive verdict.
        GenericRejection,
        /// Responds immediately with a non-definitive internal error (maps to `Other`).
        InternalError,
        /// Responds immediately with `FutureData` (-321401): the node is out of sync. Maps to the
        /// soft, non-response `DataUnavailable`.
        OutOfSyncFutureData,
        /// Responds immediately with `Uninitialized` (-320400): the node is out of sync. Maps to
        /// the soft, non-response `DataUnavailable`.
        OutOfSyncUninitialized,
        /// Responds immediately with the filter's failsafe rejection (maps to `FailsafeEnabled`).
        Failsafe,
        /// Sleeps far longer than the client timeout before responding valid (a slow endpoint).
        Slow,
    }

    /// Behavior of a mock endpoint for `admin_getFailsafeEnabled`.
    #[derive(Clone, Copy)]
    enum Failsafe {
        /// Responds immediately with the given boolean.
        Reply(bool),
        /// Responds immediately with an error.
        Error,
        /// Sleeps far longer than the client timeout before responding `false`.
        Slow,
    }

    struct MockEndpoint {
        url: String,
        _handle: ServerHandle,
    }

    impl MockEndpoint {
        async fn start(check: Verdict, failsafe: Failsafe) -> Self {
            let server = ServerBuilder::default()
                .build("127.0.0.1:0".parse::<SocketAddr>().unwrap())
                .await
                .unwrap();
            let addr = server.local_addr().unwrap();

            let mut module = RpcModule::new(());
            module
                .register_async_method(
                    "interop_checkAccessList",
                    move |_params, _ctx, _| async move {
                        match check {
                            Verdict::Valid => Ok(()),
                            Verdict::Invalid => Err(ErrorObjectOwned::owned(
                                SuperchainDAError::ConflictingData as i32,
                                "conflicting data",
                                None::<()>,
                            )),
                            // Generic -32602 rejection (e.g. malformed entry). Not a
                            // SuperchainDAError code, not -320602; must count as a definitive
                            // rejection and surface its message.
                            Verdict::GenericRejection => Err(ErrorObjectOwned::owned(
                                -32602,
                                "failed to parse access entry",
                                None::<()>,
                            )),
                            Verdict::InternalError => {
                                Err(ErrorObjectOwned::owned(-32603, "internal error", None::<()>))
                            }
                            // Out-of-sync soft failures: node lacks the data yet.
                            Verdict::OutOfSyncFutureData => {
                                Err(ErrorObjectOwned::owned(-321401, "future data", None::<()>))
                            }
                            Verdict::OutOfSyncUninitialized => {
                                Err(ErrorObjectOwned::owned(-320400, "uninitialized", None::<()>))
                            }
                            // The filter emits the dedicated failsafe code -320602. The message is
                            // intentionally unrelated to "failsafe" to prove detection is by code,
                            // not by message text.
                            Verdict::Failsafe => {
                                Err(ErrorObjectOwned::owned(-320602, "rejected", None::<()>))
                            }
                            Verdict::Slow => {
                                tokio::time::sleep(Duration::from_secs(60)).await;
                                Ok(())
                            }
                        }
                    },
                )
                .unwrap();
            module
                .register_async_method(
                    "admin_getFailsafeEnabled",
                    move |_params, _ctx, _| async move {
                        match failsafe {
                            Failsafe::Reply(v) => Ok(v),
                            Failsafe::Error => {
                                Err(ErrorObjectOwned::owned(-32603, "internal error", None::<()>))
                            }
                            Failsafe::Slow => {
                                tokio::time::sleep(Duration::from_secs(60)).await;
                                Ok(false)
                            }
                        }
                    },
                )
                .unwrap();

            let handle = server.start(module);
            Self { url: format!("http://{addr}"), _handle: handle }
        }
    }

    /// Builds a client over the given mock endpoints with a short request timeout.
    async fn client_for(
        endpoints: &[&MockEndpoint],
        min_responses: Option<usize>,
    ) -> InteropFilterClient {
        let urls = endpoints.iter().map(|e| e.url.clone()).collect();
        let mut builder =
            InteropFilterClient::builder(urls, 10).timeout(Duration::from_millis(300));
        if let Some(min) = min_responses {
            builder = builder.min_responses(min);
        }
        builder.build().await
    }

    fn descriptor() -> ExecutingDescriptor {
        ExecutingDescriptor::new(10, 0, None)
    }

    async fn check(client: &InteropFilterClient) -> Result<(), InteropTxValidatorError> {
        client.check_access_list(&[B256::ZERO], descriptor()).await
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn quorum_all_valid_accepts() {
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &b], None).await;
        assert!(check(&client).await.is_ok());
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn quorum_all_invalid_rejects_with_real_error() {
        let a = MockEndpoint::start(Verdict::Invalid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::Invalid, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &b], None).await;
        let err = check(&client).await.unwrap_err();
        assert!(
            matches!(
                err,
                InteropTxValidatorError::InvalidEntry(SuperchainDAError::ConflictingData)
            ),
            "expected the real rejection reason, got {err:?}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn generic_rejection_counts_and_surfaces_message() {
        // A generic -32602 rejection (not a SuperchainDAError code) must count as a definitive
        // verdict and reject with the filter's real message preserved (single endpoint, min 1).
        let a = MockEndpoint::start(Verdict::GenericRejection, Failsafe::Reply(false)).await;
        let client = client_for(&[&a], None).await;
        let err = check(&client).await.unwrap_err();
        assert!(
            matches!(&err, InteropTxValidatorError::Rejected { code: -32602, .. }),
            "expected a definitive Rejected verdict, got {err:?}"
        );
        assert!(
            err.to_string().contains("failed to parse access entry"),
            "rejection must preserve the filter's message, got: {err}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn out_of_sync_endpoint_is_soft() {
        // One endpoint is out of sync (FutureData / Uninitialized), the others are valid and meet
        // the quorum. The soft node must be ignored (treated as a non-response), not a rejection.
        for soft in [Verdict::OutOfSyncFutureData, Verdict::OutOfSyncUninitialized] {
            let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
            let b = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
            let out_of_sync = MockEndpoint::start(soft, Failsafe::Reply(false)).await;
            // min 2 of 3: the two valid endpoints meet the quorum; the soft node is ignored.
            let client = client_for(&[&a, &b, &out_of_sync], Some(2)).await;
            assert!(
                check(&client).await.is_ok(),
                "an out-of-sync soft failure ({soft:?}) must not cause rejection when quorum is met"
            );
        }
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn all_endpoints_out_of_sync_fail_closed() {
        // Every endpoint is out of sync: no definitive verdicts collected, so fail closed.
        let a = MockEndpoint::start(Verdict::OutOfSyncFutureData, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::OutOfSyncUninitialized, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &b], None).await;
        let err = check(&client).await.unwrap_err();
        assert!(
            matches!(err, InteropTxValidatorError::QuorumNotReached { received: 0, required: 2 }),
            "all endpoints out of sync must fail closed, got {err:?}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn disagreement_logs_and_rejects() {
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::Invalid, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &b], None).await;
        let err = check(&client).await.unwrap_err();
        assert!(
            matches!(err, InteropTxValidatorError::Disagreement { valid: 1, invalid: 1 }),
            "expected disagreement, got {err:?}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn failsafe_on_any_endpoint_hard_rejects() {
        // All three endpoints respond immediately, and `min_responses == 3` forces the aggregator
        // to collect every verdict before deciding. This makes the test deterministic (no
        // arrival-order race): the failsafe verdict is always observed, and any failsafe response
        // already received must hard-reject even though two valid verdicts are also in hand.
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let failsafe = MockEndpoint::start(Verdict::Failsafe, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &b, &failsafe], Some(3)).await;
        let err = check(&client).await.unwrap_err();
        assert!(
            matches!(err, InteropTxValidatorError::FailsafeEnabled),
            "failsafe on any endpoint must hard-reject, got {err:?}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn failsafe_on_check_flips_cached_gate() {
        // A failsafe detected on a check must flip the cached gate (and gauge) immediately, so
        // admission stops and the dashboard reflects it without waiting for the next failsafe poll.
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let failsafe = MockEndpoint::start(Verdict::Failsafe, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &failsafe], Some(2)).await;
        assert!(!client.is_failsafe_enabled());
        assert!(matches!(
            check(&client).await.unwrap_err(),
            InteropTxValidatorError::FailsafeEnabled
        ));
        assert!(
            client.is_failsafe_enabled(),
            "a failsafe detected on a check must flip the cached gate immediately"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn ignores_slow_endpoint_once_quorum_met() {
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let slow = MockEndpoint::start(Verdict::Slow, Failsafe::Reply(false)).await;
        // min 2 of 3; the slow endpoint must not delay the decision.
        let client = client_for(&[&a, &b, &slow], Some(2)).await;
        let start = Instant::now();
        assert!(check(&client).await.is_ok());
        assert!(start.elapsed() < Duration::from_secs(1), "decision waited on the slow endpoint");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn slow_non_failsafe_endpoint_does_not_delay_accept() {
        // Two valid endpoints meet a quorum of 2; a third endpoint is slow but NOT a failsafe.
        // The aggregator must fast-accept without awaiting the slow endpoint.
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let slow = MockEndpoint::start(Verdict::Slow, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &b, &slow], Some(2)).await;
        let start = Instant::now();
        assert!(check(&client).await.is_ok());
        assert!(
            start.elapsed() < Duration::from_secs(1),
            "a slow non-failsafe endpoint must not delay accept"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn timeouts_do_not_count_fail_closed() {
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let slow = MockEndpoint::start(Verdict::Slow, Failsafe::Reply(false)).await;
        // Require 2 definitive verdicts but only one endpoint responds; the timeout is not a
        // response, so quorum is never reached.
        let client = client_for(&[&a, &slow], Some(2)).await;
        let err = check(&client).await.unwrap_err();
        assert!(
            matches!(err, InteropTxValidatorError::QuorumNotReached { received: 1, required: 2 }),
            "expected fail-closed quorum-not-reached, got {err:?}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn transport_error_not_counted() {
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::InternalError, Failsafe::Reply(false)).await;
        // Internal errors are non-responses; with min 2 of 2, quorum is not reached.
        let client = client_for(&[&a, &b], None).await;
        let err = check(&client).await.unwrap_err();
        assert!(
            matches!(err, InteropTxValidatorError::QuorumNotReached { received: 1, required: 2 }),
            "expected quorum-not-reached, got {err:?}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn single_endpoint_behaves_as_before() {
        let valid = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let invalid = MockEndpoint::start(Verdict::Invalid, Failsafe::Reply(false)).await;

        let valid_client = client_for(&[&valid], None).await;
        assert!(check(&valid_client).await.is_ok());

        let invalid_client = client_for(&[&invalid], None).await;
        assert!(matches!(
            check(&invalid_client).await.unwrap_err(),
            InteropTxValidatorError::InvalidEntry(SuperchainDAError::ConflictingData)
        ));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn failsafe_or_across_endpoints() {
        let off = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let on = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(true)).await;
        let client = client_for(&[&off, &on], None).await;
        assert!(client.query_failsafe().await.unwrap());
        assert!(client.is_failsafe_enabled());
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn failsafe_short_circuits_on_first_true() {
        let on = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(true)).await;
        let slow = MockEndpoint::start(Verdict::Valid, Failsafe::Slow).await;
        let client = client_for(&[&on, &slow], None).await;
        let start = Instant::now();
        assert!(client.query_failsafe().await.unwrap());
        assert!(start.elapsed() < Duration::from_secs(1), "failsafe waited on the slow endpoint");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn failsafe_partial_replies_do_not_clear_gate() {
        // One endpoint is unreachable (slow/timing out) and the other replies "not in failsafe".
        // The silent endpoint might itself be the one in failsafe, so a partial poll must NOT clear
        // a previously-set gate to false. Only a unanimous "all replied false" may clear it. This
        // bug only surfaces once `min_responses` is lowered for availability (with the unanimity
        // default the slow endpoint also fails the check closed).
        let silent = MockEndpoint::start(Verdict::Valid, Failsafe::Slow).await;
        let healthy = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let client = client_for(&[&silent, &healthy], Some(1)).await;
        // Failsafe was previously detected and cached.
        client.apply_failsafe_state(true);
        let result = client.query_failsafe().await;
        assert!(
            client.is_failsafe_enabled(),
            "a partial poll (one endpoint silent) must not clear the failsafe gate, got {result:?}"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn failsafe_unanimous_false_clears_gate() {
        // Every endpoint replied "not in failsafe", so the gate is cleared (decisive).
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let b = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        let client = client_for(&[&a, &b], None).await;
        client.apply_failsafe_state(true);
        assert!(!client.query_failsafe().await.unwrap());
        assert!(!client.is_failsafe_enabled(), "a unanimous all-false poll must clear the gate");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn failsafe_all_error_keeps_cache() {
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Error).await;
        let b = MockEndpoint::start(Verdict::Valid, Failsafe::Error).await;
        let client = client_for(&[&a, &b], None).await;
        // Seed a known cached value, then confirm an all-error poll leaves it unchanged and errors.
        client.inner.failsafe_enabled.store(true, Ordering::Release);
        assert!(client.query_failsafe().await.is_err());
        assert!(client.is_failsafe_enabled(), "cache should be unchanged when no endpoint replies");
    }

    #[tokio::test(flavor = "multi_thread")]
    #[should_panic(expected = "min_responses")]
    async fn builder_rejects_bad_min_responses() {
        let a = MockEndpoint::start(Verdict::Valid, Failsafe::Reply(false)).await;
        // min_responses exceeds the single configured endpoint.
        let _ = client_for(&[&a], Some(2)).await;
    }

    #[tokio::test(flavor = "multi_thread")]
    #[should_panic(expected = "at least one endpoint")]
    async fn builder_rejects_no_endpoints() {
        let _ = InteropFilterClient::builder(Vec::new(), 10).build().await;
    }
}
