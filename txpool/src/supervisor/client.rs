//! This is our custom implementation of validator struct

use crate::{
    supervisor::{
        metrics::SupervisorMetrics, parse_access_list_items_to_inbox_entries, ExecutingDescriptor,
        InteropTxValidatorError,
    },
    InvalidCrossTx,
};
use alloy_eips::eip2930::AccessList;
use alloy_primitives::{TxHash, B256};
use alloy_rpc_client::ReqwestClient;
use futures_util::future::BoxFuture;
use op_alloy_consensus::interop::SafetyLevel;
use std::{
    borrow::Cow,
    future::IntoFuture,
    time::{Duration, Instant},
};
use tracing::trace;

/// Supervisor hosted by op-labs
// TODO: This should be changes to actual supervisor url
pub const DEFAULT_SUPERVISOR_URL: &str = "http://localhost:1337/";

/// The default request timeout to use
const DEFAULT_REQUEST_TIMOUT: Duration = Duration::from_millis(100);

/// Implementation of the supervisor trait for the interop.
#[derive(Debug, Clone)]
pub struct SupervisorClient {
    client: ReqwestClient,
    /// The default
    safety: SafetyLevel,
    /// The default request timeout
    timeout: Duration,
    /// Metrics for tracking supervisor operations
    metrics: SupervisorMetrics,
}

impl SupervisorClient {
    /// Creates a new supervisor validator.
    pub async fn new(supervisor_endpoint: impl Into<String>, safety: SafetyLevel) -> Self {
        let client = ReqwestClient::builder()
            .connect(supervisor_endpoint.into().as_str())
            .await
            .expect("building supervisor client");
        Self {
            client,
            safety,
            timeout: DEFAULT_REQUEST_TIMOUT,
            metrics: SupervisorMetrics::default(),
        }
    }

    /// Configures a custom timeout
    pub const fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Returns safely level
    pub const fn safety(&self) -> SafetyLevel {
        self.safety
    }

    /// Executes a `supervisor_checkAccessList` with the configured safety level.
    pub fn check_access_list<'a>(
        &self,
        inbox_entries: &'a [B256],
        executing_descriptor: ExecutingDescriptor,
    ) -> CheckAccessListRequest<'a> {
        CheckAccessListRequest {
            client: self.client.clone(),
            inbox_entries: Cow::Borrowed(inbox_entries),
            executing_descriptor,
            timeout: self.timeout,
            safety: self.safety,
            metrics: self.metrics.clone(),
        }
    }

    /// Extracts commitment from access list entries, pointing to 0x420..022 and validates them
    /// against supervisor.
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
            return Some(Err(InvalidCrossTx::CrossChainTxPreInterop))
        }

        if let Err(err) = self
            .check_access_list(
                inbox_entries.as_slice(),
                ExecutingDescriptor::new(timestamp, timeout),
            )
            .await
        {
            trace!(target: "txpool", hash=%hash, err=%err, "Cross chain transaction invalid");
            return Some(Err(InvalidCrossTx::ValidationError(err)));
        }
        Some(Ok(()))
    }
}

/// A Request future that issues a `supervisor_checkAccessList` request.
#[derive(Debug, Clone)]
pub struct CheckAccessListRequest<'a> {
    client: ReqwestClient,
    inbox_entries: Cow<'a, [B256]>,
    executing_descriptor: ExecutingDescriptor,
    timeout: Duration,
    safety: SafetyLevel,
    metrics: SupervisorMetrics,
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
                    "supervisor_checkAccessList",
                    (inbox_entries, safety, executing_descriptor),
                ),
            )
            .await;
            metrics.record_supervisor_query(start.elapsed());

            result
                .map_err(|_| InteropTxValidatorError::Timeout(timeout.as_secs()))?
                .map_err(InteropTxValidatorError::other)
        })
    }
}
