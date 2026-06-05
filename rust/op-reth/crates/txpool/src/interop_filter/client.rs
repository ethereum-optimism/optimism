//! This is our custom implementation of validator struct

use crate::{
    InvalidCrossTx,
    interop_filter::{
        ExecutingDescriptor, InteropTxValidatorError, metrics::InteropMetrics,
        parse_access_list_items_to_inbox_entries,
    },
};
use alloy_consensus::Transaction;
use alloy_eips::eip2930::AccessList;
use alloy_primitives::{B256, TxHash};
use alloy_rpc_client::ReqwestClient;
use futures_util::{
    Stream,
    future::BoxFuture,
    stream::{self, StreamExt},
};
use op_alloy_consensus::interop::SafetyLevel;
use reth_transaction_pool::PoolTransaction;
use std::{
    borrow::Cow,
    future::IntoFuture,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    time::{Duration, Instant},
};
use tracing::trace;

/// The default request timeout to use
pub const DEFAULT_REQUEST_TIMEOUT: Duration = Duration::from_secs(2);

/// Client for interop filter RPC requests.
#[derive(Debug, Clone)]
pub struct InteropFilterClient {
    /// Stores type's data.
    inner: Arc<InteropFilterClientInner>,
}

impl InteropFilterClient {
    /// Returns a new [`InteropFilterClientBuilder`].
    pub fn builder(
        interop_endpoint: impl Into<String>,
        chain_id: u64,
    ) -> InteropFilterClientBuilder {
        InteropFilterClientBuilder::new(interop_endpoint, chain_id)
    }

    /// Returns the configured request timeout.
    pub fn timeout(&self) -> Duration {
        self.inner.timeout
    }

    /// Returns configured minimum safety level. See [`InteropFilterClient`].
    pub fn safety(&self) -> SafetyLevel {
        self.inner.safety
    }

    /// Executes an `interop_checkAccessList` with the configured safety level.
    pub fn check_access_list<'a>(
        &self,
        inbox_entries: &'a [B256],
        executing_descriptor: ExecutingDescriptor,
    ) -> CheckAccessListRequest<'a> {
        CheckAccessListRequest {
            client: self.inner.client.clone(),
            inbox_entries: Cow::Borrowed(inbox_entries),
            executing_descriptor,
            timeout: self.inner.timeout,
            safety: self.inner.safety,
            metrics: self.inner.metrics.clone(),
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
            self.inner.metrics.increment_metrics_for_error(&err);
            trace!(target: "txpool", hash=%hash, err=%err, "Cross chain transaction invalid");
            return Some(Err(InvalidCrossTx::ValidationError(err)));
        }
        Some(Ok(()))
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

    /// Queries the interop filter for failsafe state and caches the result.
    /// Calls `admin_getFailsafeEnabled` RPC.
    pub async fn query_failsafe(&self) -> Result<bool, InteropTxValidatorError> {
        let result = tokio::time::timeout(
            self.inner.timeout,
            self.inner.client.request::<_, bool>("admin_getFailsafeEnabled", ()),
        )
        .await
        .map_err(|_| InteropTxValidatorError::Timeout(self.inner.timeout.as_secs()))?
        .map_err(InteropTxValidatorError::from_json_rpc)?;

        self.apply_failsafe_state(result);
        Ok(result)
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

/// Holds interop RPC data. Inner type of [`InteropFilterClient`].
#[derive(Debug)]
pub(crate) struct InteropFilterClientInner {
    client: ReqwestClient,
    /// The chain ID of the executing chain
    chain_id: u64,
    /// The default
    safety: SafetyLevel,
    /// The default request timeout
    timeout: Duration,
    /// Metrics for tracking interop RPC operations.
    metrics: InteropMetrics,
    /// Cached failsafe state, polled by the background failsafe task.
    failsafe_enabled: AtomicBool,
}

/// Builds [`InteropFilterClient`].
#[derive(Debug)]
pub struct InteropFilterClientBuilder {
    /// Interop filter RPC endpoint.
    endpoint: String,
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
    /// Creates a new builder.
    pub fn new(interop_endpoint: impl Into<String>, chain_id: u64) -> Self {
        Self {
            endpoint: interop_endpoint.into(),
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

    /// Creates a new interop RPC client.
    pub async fn build(self) -> InteropFilterClient {
        let Self { endpoint, chain_id, timeout, safety } = self;

        let client = ReqwestClient::builder()
            .connect(endpoint.as_str())
            .await
            .expect("building interop filter client");

        InteropFilterClient {
            inner: Arc::new(InteropFilterClientInner {
                client,
                chain_id,
                safety,
                timeout,
                metrics: InteropMetrics::default(),
                failsafe_enabled: AtomicBool::new(false),
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
        let Self { client, inbox_entries, executing_descriptor, timeout, safety, metrics } = self;
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
            metrics.record_interop_query(start.elapsed());

            result
                .map_err(|_| InteropTxValidatorError::Timeout(timeout.as_secs()))?
                .map_err(InteropTxValidatorError::from_json_rpc)
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::interop_filter::CROSS_L2_INBOX_ADDRESS;
    use alloy_eips::eip2930::AccessListItem;

    async fn test_client() -> InteropFilterClient {
        InteropFilterClient::builder("http://localhost:8545", 1).build().await
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
}
