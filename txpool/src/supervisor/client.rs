//! This is our custom implementation of validator struct

use crate::supervisor::{ExecutingDescriptor, InteropTxValidatorError};
use alloy_primitives::B256;
use alloy_rpc_client::ReqwestClient;
use futures_util::future::BoxFuture;
use op_alloy_consensus::interop::SafetyLevel;
use std::{borrow::Cow, future::IntoFuture, time::Duration};

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
}

impl SupervisorClient {
    /// Creates a new supervisor validator.
    pub async fn new(supervisor_endpoint: impl Into<String>, safety: SafetyLevel) -> Self {
        let client = ReqwestClient::builder()
            .connect(supervisor_endpoint.into().as_str())
            .await
            .expect("building supervisor client");
        Self { client, safety, timeout: DEFAULT_REQUEST_TIMOUT }
    }

    /// Configures a custom timeout
    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Returns safely level
    pub fn safety(&self) -> SafetyLevel {
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
        }
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
}

impl CheckAccessListRequest<'_> {
    /// Configures the timeout to use for the request if any.
    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// Configures the [`SafetyLevel`] for this request
    pub fn with_safety(mut self, safety: SafetyLevel) -> Self {
        self.safety = safety;
        self
    }
}

impl<'a> IntoFuture for CheckAccessListRequest<'a> {
    type Output = Result<(), InteropTxValidatorError>;
    type IntoFuture = BoxFuture<'a, Self::Output>;

    fn into_future(self) -> Self::IntoFuture {
        let Self { client, inbox_entries, executing_descriptor, timeout, safety } = self;
        Box::pin(async move {
            tokio::time::timeout(
                timeout,
                client.request(
                    "supervisor_checkAccessList",
                    (inbox_entries, safety, executing_descriptor),
                ),
            )
            .await
            .map_err(|_| InteropTxValidatorError::ValidationTimeout(timeout.as_secs()))?
            .map_err(InteropTxValidatorError::client)
        })
    }
}
