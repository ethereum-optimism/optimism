//! Succinct Prover Network request submission, polling, and verification.

use std::{future::Future, sync::Arc, time::Duration};

use anyhow::{Result, bail};
use async_trait::async_trait;
use kona_sp1_host_utils::metrics::MetricsGauge;
use sp1_sdk::{
    NetworkProver, ProveRequest, Prover, SP1ProofMode, SP1ProofWithPublicValues, SP1ProvingKey,
    SP1Stdin, SP1VerifyingKey,
    network::{
        NetworkMode,
        proto::{
            GetProofRequestStatusResponse,
            types::{ExecutionStatus, FulfillmentStatus, ProofRequest},
        },
    },
};
use tokio::time::sleep;

use super::{ProofId, ProofKeys, ProofTerminalState, ProofWaitError};
use crate::{config::ProofProviderConfig, metrics::ProposerGauge};

#[cfg(test)]
use alloy_primitives::B256;
#[cfg(test)]
use anyhow::Context;

/// Polling interval (in seconds) for checking proof status.
/// Matches the SP1 SDK's internal polling interval.
const PROOF_STATUS_POLL_INTERVAL: u64 = 2;
fn current_timestamp() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .expect("System time before Unix epoch")
        .as_secs()
}
#[async_trait]
trait NetworkProverApi: Send + Sync {
    async fn request_range_proof(
        &self,
        proving_key: &SP1ProvingKey,
        stdin: SP1Stdin,
        config: &ProofProviderConfig,
    ) -> Result<ProofId>;

    async fn request_aggregation_proof(
        &self,
        proving_key: &SP1ProvingKey,
        stdin: SP1Stdin,
        config: &ProofProviderConfig,
    ) -> Result<ProofId>;

    async fn get_proof_status(
        &self,
        proof_id: ProofId,
    ) -> Result<(GetProofRequestStatusResponse, Option<SP1ProofWithPublicValues>)>;

    async fn get_proof_request(&self, proof_id: ProofId) -> Result<Option<ProofRequest>>;

    async fn cancel_request(&self, proof_id: ProofId) -> Result<()>;

    fn verify(
        &self,
        proof: &SP1ProofWithPublicValues,
        verifying_key: &SP1VerifyingKey,
    ) -> Result<()>;
}

struct Sp1NetworkProverApi {
    prover: Arc<NetworkProver>,
}

#[async_trait]
impl NetworkProverApi for Sp1NetworkProverApi {
    async fn request_range_proof(
        &self,
        proving_key: &SP1ProvingKey,
        stdin: SP1Stdin,
        config: &ProofProviderConfig,
    ) -> Result<ProofId> {
        self.prover
            .prove(proving_key, stdin)
            .compressed()
            .skip_simulation(true)
            .strategy(config.range_proof_strategy)
            .timeout(Duration::from_secs(config.timeout))
            .min_auction_period(config.min_auction_period)
            .max_price_per_pgu(config.max_price_per_pgu)
            .cycle_limit(config.range_cycle_limit)
            .gas_limit(config.range_gas_limit)
            .request()
            .await
    }

    async fn request_aggregation_proof(
        &self,
        proving_key: &SP1ProvingKey,
        stdin: SP1Stdin,
        config: &ProofProviderConfig,
    ) -> Result<ProofId> {
        self.prover
            .prove(proving_key, stdin)
            .mode(SP1ProofMode::Plonk)
            .strategy(config.agg_proof_strategy)
            .timeout(Duration::from_secs(config.timeout))
            .min_auction_period(config.min_auction_period)
            .max_price_per_pgu(config.max_price_per_pgu)
            .cycle_limit(config.agg_cycle_limit)
            .gas_limit(config.agg_gas_limit)
            .request()
            .await
    }

    async fn get_proof_status(
        &self,
        proof_id: ProofId,
    ) -> Result<(GetProofRequestStatusResponse, Option<SP1ProofWithPublicValues>)> {
        self.prover.get_proof_status(proof_id).await
    }

    async fn get_proof_request(&self, proof_id: ProofId) -> Result<Option<ProofRequest>> {
        self.prover.get_proof_request(proof_id).await
    }

    async fn cancel_request(&self, proof_id: ProofId) -> Result<()> {
        self.prover.cancel_request(proof_id).await
    }

    fn verify(
        &self,
        proof: &SP1ProofWithPublicValues,
        verifying_key: &SP1VerifyingKey,
    ) -> Result<()> {
        Prover::verify(self.prover.as_ref(), proof, verifying_key, None).map_err(anyhow::Error::new)
    }
}
/// Network-based proof provider using the SP1 prover network.
#[derive(Clone)]
pub struct NetworkProofProvider {
    api: Arc<dyn NetworkProverApi>,
    config: ProofProviderConfig,
    network_mode: NetworkMode,
}

impl std::fmt::Debug for NetworkProofProvider {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("NetworkProofProvider")
            .field("config", &self.config)
            .field("network_mode", &self.network_mode)
            .finish_non_exhaustive()
    }
}

impl NetworkProofProvider {
    /// Creates a network provider with the given request settings.
    pub fn new(
        prover: Arc<NetworkProver>,
        config: ProofProviderConfig,
        network_mode: NetworkMode,
    ) -> Self {
        Self::with_api(Arc::new(Sp1NetworkProverApi { prover }), config, network_mode)
    }

    fn with_api(
        api: Arc<dyn NetworkProverApi>,
        config: ProofProviderConfig,
        network_mode: NetworkMode,
    ) -> Self {
        Self { api, config, network_mode }
    }

    pub(super) async fn request_range_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> Result<ProofId> {
        tracing::info!("Requesting range proof via network");
        let proof_id =
            self.api.request_range_proof(keys.range_pk.as_ref(), stdin, &self.config).await?;
        tracing::info!(proof_id = %proof_id, "Range proof request submitted");
        Ok(proof_id)
    }

    pub(super) async fn request_aggregation_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> Result<ProofId> {
        tracing::info!("Requesting aggregation proof via network");
        let proof_id =
            self.api.request_aggregation_proof(keys.agg_pk.as_ref(), stdin, &self.config).await?;
        tracing::info!(proof_id = %proof_id, "Aggregation proof request submitted");
        Ok(proof_id)
    }

    pub(super) fn verify(
        &self,
        proof: &SP1ProofWithPublicValues,
        verifying_key: &SP1VerifyingKey,
    ) -> Result<()> {
        self.api.verify(proof, verifying_key)
    }

    /// Waits for an existing proof request to finish.
    pub(super) async fn wait_for_proof(
        &self,
        proof_id: ProofId,
    ) -> Result<SP1ProofWithPublicValues, ProofWaitError> {
        let started_at = std::time::Instant::now();
        loop {
            self.ensure_within_proving_timeout(proof_id, started_at)?;
            if let Some(proof) = self.poll_once(proof_id).await? {
                return Ok(proof);
            }
            sleep(Duration::from_secs(PROOF_STATUS_POLL_INTERVAL)).await;
        }
    }

    fn ensure_within_proving_timeout(
        &self,
        proof_id: ProofId,
        started_at: std::time::Instant,
    ) -> Result<(), ProofWaitError> {
        let timeout = Duration::from_secs(self.config.timeout);
        let ProvingTimeout::Exceeded { elapsed_secs } =
            check_timeout(started_at.elapsed(), timeout)
        else {
            return Ok(());
        };

        tracing::warn!(
            proof_id = %proof_id,
            elapsed_secs,
            timeout_secs = self.config.timeout,
            "proving timeout exceeded"
        );
        ProposerGauge::ProvingTimeoutError.increment(1.0);
        let range_split_count = crate::env_var("RANGE_SPLIT_COUNT");
        Err(ProofWaitError::Uncertain(anyhow::anyhow!(
            "proving timeout: proof_id={proof_id}, elapsed={elapsed_secs}s, timeout={}s \
             (consider increasing {range_split_count} to shrink per-proof workloads)",
            self.config.timeout
        )))
    }

    async fn poll_once(
        &self,
        proof_id: ProofId,
    ) -> Result<Option<SP1ProofWithPublicValues>, ProofWaitError> {
        let (status, proof) = match self.fetch_proof_status(proof_id).await {
            Ok(result) => result,
            Err(err) => {
                tracing::warn!(proof_id = %proof_id, error = %err, "get_proof_status failed, retrying");
                return Ok(None);
            }
        };

        if let Some(proof) = Self::classify_status(proof_id, &status, proof)? {
            return Ok(Some(proof));
        }

        let request_details = if self.network_mode == NetworkMode::Mainnet {
            match self.fetch_request_details(proof_id).await {
                Ok(details) => details,
                Err(err) => {
                    tracing::warn!(proof_id = %proof_id, error = %err, "get_proof_request failed, retrying");
                    return Ok(None);
                }
            }
        } else {
            None
        };

        if let Some(details) = &request_details {
            if details.fulfillment_status == FulfillmentStatus::Fulfilled as i32 {
                return Ok(None);
            }
            if details.is_canceled {
                return Err(Self::terminal_error(
                    proof_id,
                    ProofTerminalState::Cancelled,
                    details.fulfillment_status,
                    details.execution_status,
                ));
            }
        }

        let current_time = current_timestamp();
        if let Some(details) = &request_details &&
            self.handle_auction_timeout(proof_id, details, current_time).await?
        {
            return Ok(None);
        }
        self.ensure_before_deadline(proof_id, &status, current_time)?;
        Ok(None)
    }

    async fn fetch_proof_status(
        &self,
        proof_id: ProofId,
    ) -> Result<(GetProofRequestStatusResponse, Option<SP1ProofWithPublicValues>)> {
        self.network_call_with_timeout(
            self.api.get_proof_status(proof_id),
            "get_proof_status",
            proof_id,
        )
        .await
    }

    async fn fetch_request_details(&self, proof_id: ProofId) -> Result<Option<ProofRequest>> {
        self.network_call_with_timeout(
            self.api.get_proof_request(proof_id),
            "get_proof_request",
            proof_id,
        )
        .await
    }

    fn classify_status(
        proof_id: ProofId,
        status: &GetProofRequestStatusResponse,
        proof: Option<SP1ProofWithPublicValues>,
    ) -> Result<Option<SP1ProofWithPublicValues>, ProofWaitError> {
        match check_status(status.fulfillment_status(), status.execution_status()) {
            ProofStatus::Ready => {
                tracing::info!(proof_id = %proof_id, "Proof fulfilled");
                let proof = proof.ok_or_else(|| {
                    ProofWaitError::Uncertain(anyhow::anyhow!(
                        "proof status is fulfilled but proof is missing: proof_id={proof_id}"
                    ))
                })?;
                Ok(Some(proof))
            }
            ProofStatus::Failed(state) => Err(Self::terminal_error(
                proof_id,
                state,
                status.fulfillment_status(),
                status.execution_status(),
            )),
            ProofStatus::Pending => {
                tracing::debug!(proof_id = %proof_id, "Proof pending/assigned, continuing");
                Ok(None)
            }
        }
    }

    async fn handle_auction_timeout(
        &self,
        proof_id: ProofId,
        details: &ProofRequest,
        current_time: u64,
    ) -> Result<bool, ProofWaitError> {
        let timeout_secs = self.config.auction_timeout;
        let AuctionTimeout::Exceeded { elapsed_secs } = check_auction(
            true,
            details.fulfillment_status,
            details.created_at,
            timeout_secs,
            current_time,
        ) else {
            return Ok(false);
        };

        tracing::warn!(
            proof_id = %proof_id,
            elapsed_secs,
            timeout_secs,
            "auction timeout exceeded, cancelling request"
        );
        self.network_call_with_timeout(
            self.api.cancel_request(proof_id),
            "cancel_request",
            proof_id,
        )
        .await
        .map_err(|err| {
            ProofWaitError::Uncertain(
                err.context(format!("failed to cancel proof request: proof_id={proof_id}")),
            )
        })?;
        ProposerGauge::AuctionTimeoutError.increment(1.0);

        let confirmed =
            self.fetch_request_details(proof_id).await.map_err(ProofWaitError::Uncertain)?;
        if let Some(confirmed) = &confirmed &&
            confirmed.fulfillment_status == FulfillmentStatus::Fulfilled as i32
        {
            return Ok(true);
        }
        if let Some(confirmed) = &confirmed &&
            confirmed.is_canceled
        {
            return Err(Self::terminal_error(
                proof_id,
                ProofTerminalState::Cancelled,
                confirmed.fulfillment_status,
                confirmed.execution_status,
            ));
        }
        Err(ProofWaitError::Uncertain(anyhow::anyhow!(
            "proof request cancellation was not confirmed: proof_id={proof_id}"
        )))
    }

    fn ensure_before_deadline(
        &self,
        proof_id: ProofId,
        status: &GetProofRequestStatusResponse,
        current_time: u64,
    ) -> Result<(), ProofWaitError> {
        let Deadline::Exceeded { deadline } = check_deadline(status.deadline(), current_time)
        else {
            return Ok(());
        };

        tracing::warn!(
            proof_id = %proof_id,
            deadline,
            current_time,
            "proof request deadline exceeded"
        );
        ProposerGauge::DeadlineExceededError.increment(1.0);
        Err(Self::terminal_error(
            proof_id,
            ProofTerminalState::Expired,
            FulfillmentStatus::Expired as i32,
            ExecutionStatus::Unexecuted as i32,
        ))
    }

    const fn terminal_error(
        proof_id: ProofId,
        state: ProofTerminalState,
        fulfillment_status: i32,
        execution_status: i32,
    ) -> ProofWaitError {
        ProofWaitError::Terminal { proof_id, state, fulfillment_status, execution_status }
    }

    async fn network_call_with_timeout<F, T>(
        &self,
        future: F,
        operation: &str,
        proof_id: ProofId,
    ) -> Result<T>
    where
        F: Future<Output = Result<T, anyhow::Error>>,
    {
        let timeout_secs = self.config.network_calls_timeout;
        match tokio::time::timeout(Duration::from_secs(timeout_secs), future).await {
            Ok(Ok(result)) => Ok(result),
            Ok(Err(err)) => {
                tracing::warn!(proof_id = %proof_id, operation, error = %err, "Network error");
                Err(err)
            }
            Err(_) => {
                tracing::warn!(proof_id = %proof_id, operation, timeout_secs, "Network call timed out");
                ProposerGauge::NetworkCallTimeout.increment(1.0);
                bail!(
                    "Network timeout after {}s for {} (proof_id={})",
                    timeout_secs,
                    operation,
                    proof_id
                )
            }
        }
    }
}
/// Client-side proof request timeout status.
#[derive(Debug, PartialEq, Eq)]
enum ProvingTimeout {
    /// Polling may continue.
    Ok,
    /// The configured timeout elapsed.
    Exceeded {
        /// Seconds elapsed since the request started.
        elapsed_secs: u64,
    },
}

/// Equality remains within the timeout.
fn check_timeout(elapsed: Duration, timeout: Duration) -> ProvingTimeout {
    if elapsed > timeout {
        ProvingTimeout::Exceeded { elapsed_secs: elapsed.as_secs() }
    } else {
        ProvingTimeout::Ok
    }
}

/// Prover-network auction timeout status.
#[derive(Debug, PartialEq, Eq)]
enum AuctionTimeout {
    /// The auction remains within its timeout.
    Ok,
    /// Auction timeout does not apply to this request.
    Skip,
    /// The auction timeout elapsed.
    Exceeded {
        /// Seconds elapsed since the request was created.
        elapsed_secs: u64,
    },
}

/// Applies only to mainnet requests in `Requested` state; equality remains
/// within the timeout.
const fn check_auction(
    is_mainnet: bool,
    fulfillment_status: i32,
    created_at: u64,
    auction_timeout: u64,
    current_time: u64,
) -> AuctionTimeout {
    if !is_mainnet {
        return AuctionTimeout::Skip;
    }

    if fulfillment_status != FulfillmentStatus::Requested as i32 {
        return AuctionTimeout::Skip;
    }

    let deadline = created_at + auction_timeout;
    if current_time > deadline {
        AuctionTimeout::Exceeded { elapsed_secs: current_time - created_at }
    } else {
        AuctionTimeout::Ok
    }
}

/// Server-side proof deadline status.
#[derive(Debug, PartialEq, Eq)]
enum Deadline {
    /// The deadline remains in the future.
    Ok,
    /// The deadline has been reached.
    Exceeded {
        /// Server-side deadline in Unix seconds.
        deadline: u64,
    },
}

/// Treats equality as expired.
const fn check_deadline(deadline: u64, current_time: u64) -> Deadline {
    if current_time >= deadline { Deadline::Exceeded { deadline } } else { Deadline::Ok }
}

/// Combined fulfillment and execution status for a proof request.
#[derive(Debug, PartialEq, Eq)]
enum ProofStatus {
    /// A fulfilled proof is available.
    Ready,
    /// The request reached a terminal state without a proof.
    Failed(ProofTerminalState),
    /// Polling should continue.
    Pending,
}

/// Fulfillment takes precedence over terminal execution status.
fn check_status(fulfillment_status: i32, execution_status: i32) -> ProofStatus {
    let fulfillment_status = FulfillmentStatus::try_from(fulfillment_status).ok();
    let execution_status = ExecutionStatus::try_from(execution_status).ok();

    match fulfillment_status {
        Some(FulfillmentStatus::Fulfilled) => return ProofStatus::Ready,
        Some(FulfillmentStatus::Unfulfillable) => {
            return ProofStatus::Failed(ProofTerminalState::Unfulfillable);
        }
        Some(FulfillmentStatus::Reverted) => {
            return ProofStatus::Failed(ProofTerminalState::Reverted);
        }
        Some(FulfillmentStatus::Expired) => {
            return ProofStatus::Failed(ProofTerminalState::Expired);
        }
        _ => {}
    }
    match execution_status {
        Some(ExecutionStatus::Unexecutable) => {
            ProofStatus::Failed(ProofTerminalState::Unexecutable)
        }
        Some(ExecutionStatus::ValidationFailed) => {
            ProofStatus::Failed(ProofTerminalState::ValidationFailed)
        }
        _ => ProofStatus::Pending,
    }
}
#[cfg(test)]
mod tests {
    use std::{
        collections::VecDeque,
        sync::atomic::{AtomicUsize, Ordering},
        time::Duration,
    };

    use parking_lot::Mutex;

    use sp1_sdk::{
        SP1Proof, SP1PublicValues,
        network::{FulfillmentStrategy, proto::auction_types},
    };

    use super::*;

    struct ScriptedNetworkApi {
        statuses:
            Mutex<VecDeque<(GetProofRequestStatusResponse, Option<SP1ProofWithPublicValues>)>>,
        details: Mutex<Option<ProofRequest>>,
        status_ids: Mutex<Vec<ProofId>>,
        request_calls: AtomicUsize,
        status_calls: AtomicUsize,
    }

    impl ScriptedNetworkApi {
        fn new(
            statuses: Vec<(GetProofRequestStatusResponse, Option<SP1ProofWithPublicValues>)>,
            details: Option<ProofRequest>,
        ) -> Self {
            Self {
                statuses: Mutex::new(statuses.into()),
                details: Mutex::new(details),
                status_ids: Mutex::new(Vec::new()),
                request_calls: AtomicUsize::new(0),
                status_calls: AtomicUsize::new(0),
            }
        }
    }

    #[async_trait]
    impl NetworkProverApi for ScriptedNetworkApi {
        async fn request_range_proof(
            &self,
            _proving_key: &SP1ProvingKey,
            _stdin: SP1Stdin,
            _config: &ProofProviderConfig,
        ) -> Result<ProofId> {
            self.request_calls.fetch_add(1, Ordering::SeqCst);
            bail!("unexpected range proof request")
        }

        async fn request_aggregation_proof(
            &self,
            _proving_key: &SP1ProvingKey,
            _stdin: SP1Stdin,
            _config: &ProofProviderConfig,
        ) -> Result<ProofId> {
            self.request_calls.fetch_add(1, Ordering::SeqCst);
            bail!("unexpected aggregation proof request")
        }

        async fn get_proof_status(
            &self,
            proof_id: ProofId,
        ) -> Result<(GetProofRequestStatusResponse, Option<SP1ProofWithPublicValues>)> {
            self.status_calls.fetch_add(1, Ordering::SeqCst);
            self.status_ids.lock().push(proof_id);
            self.statuses.lock().pop_front().context("no scripted proof status")
        }

        async fn get_proof_request(&self, _proof_id: ProofId) -> Result<Option<ProofRequest>> {
            Ok(self.details.lock().clone())
        }

        async fn cancel_request(&self, _proof_id: ProofId) -> Result<()> {
            if let Some(details) = self.details.lock().as_mut() {
                details.is_canceled = true;
            }
            Ok(())
        }

        fn verify(
            &self,
            _proof: &SP1ProofWithPublicValues,
            _verifying_key: &SP1VerifyingKey,
        ) -> Result<()> {
            Ok(())
        }
    }

    fn provider_with_api(
        api: Arc<dyn NetworkProverApi>,
        network_mode: NetworkMode,
    ) -> NetworkProofProvider {
        NetworkProofProvider::with_api(
            api,
            ProofProviderConfig {
                timeout: 60,
                network_calls_timeout: 5,
                auction_timeout: 10,
                range_proof_strategy: FulfillmentStrategy::Auction,
                agg_proof_strategy: FulfillmentStrategy::Auction,
                range_cycle_limit: 1,
                range_gas_limit: 1,
                agg_cycle_limit: 1,
                agg_gas_limit: 1,
                max_price_per_pgu: 1,
                min_auction_period: 1,
            },
            network_mode,
        )
    }

    fn proof_status(
        fulfillment_status: FulfillmentStatus,
        execution_status: ExecutionStatus,
    ) -> GetProofRequestStatusResponse {
        proof_status_with_deadline(fulfillment_status, execution_status, u64::MAX)
    }

    fn proof_status_with_deadline(
        fulfillment_status: FulfillmentStatus,
        execution_status: ExecutionStatus,
        deadline: u64,
    ) -> GetProofRequestStatusResponse {
        auction_types::GetProofRequestStatusResponse {
            fulfillment_status: fulfillment_status as i32,
            execution_status: execution_status as i32,
            deadline,
            ..Default::default()
        }
        .into()
    }

    fn test_proof() -> SP1ProofWithPublicValues {
        SP1ProofWithPublicValues {
            proof: SP1Proof::Core(Vec::new()),
            public_values: SP1PublicValues::new(),
            sp1_version: String::new(),
            tee_proof: None,
        }
    }

    #[test]
    fn timeout_transitions() {
        // (elapsed, limit, expected)
        let cases = [
            (30, 60, ProvingTimeout::Ok),
            (60, 60, ProvingTimeout::Ok),
            (61, 60, ProvingTimeout::Exceeded { elapsed_secs: 61 }),
        ];
        for (elapsed, limit, expected) in cases {
            let result = check_timeout(Duration::from_secs(elapsed), Duration::from_secs(limit));
            assert_eq!(result, expected, "elapsed={elapsed} limit={limit}");
        }
    }

    #[test]
    fn auction_transitions() {
        // (is_mainnet, fulfillment_status, current_time, expected) with
        // created_at=1000 and auction_timeout=60.
        let cases = [
            (false, FulfillmentStatus::Requested as i32, 2000, AuctionTimeout::Skip),
            (true, FulfillmentStatus::Assigned as i32, 2000, AuctionTimeout::Skip),
            (true, FulfillmentStatus::Fulfilled as i32, 2000, AuctionTimeout::Skip),
            (true, FulfillmentStatus::Requested as i32, 1050, AuctionTimeout::Ok),
            (true, FulfillmentStatus::Requested as i32, 1060, AuctionTimeout::Ok),
            (
                true,
                FulfillmentStatus::Requested as i32,
                1061,
                AuctionTimeout::Exceeded { elapsed_secs: 61 },
            ),
        ];
        for (is_mainnet, status, current_time, expected) in cases {
            let result = check_auction(is_mainnet, status, 1000, 60, current_time);
            assert_eq!(result, expected, "mainnet={is_mainnet} status={status} t={current_time}");
        }
    }

    #[test]
    fn deadline_transitions() {
        let cases = [
            (2000, 1500, Deadline::Ok),
            (2000, 2000, Deadline::Exceeded { deadline: 2000 }),
            (2000, 2001, Deadline::Exceeded { deadline: 2000 }),
        ];
        for (deadline, current, expected) in cases {
            assert_eq!(check_deadline(deadline, current), expected, "d={deadline} t={current}");
        }
    }

    #[test]
    fn server_deadline_is_retryable_terminal() {
        let provider = provider_with_api(
            Arc::new(ScriptedNetworkApi::new(Vec::new(), None)),
            NetworkMode::Reserved,
        );
        let proof_id = B256::repeat_byte(0x44);
        let status = proof_status_with_deadline(
            FulfillmentStatus::Requested,
            ExecutionStatus::Unexecuted,
            100,
        );

        let err = provider.ensure_before_deadline(proof_id, &status, 101).unwrap_err();

        assert!(matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::Expired, .. }));
    }

    #[test]
    fn status_transitions() {
        let cases = [
            (
                FulfillmentStatus::Fulfilled as i32,
                ExecutionStatus::ValidationFailed as i32,
                ProofStatus::Ready,
            ),
            (
                FulfillmentStatus::Unfulfillable as i32,
                ExecutionStatus::Unexecuted as i32,
                ProofStatus::Failed(ProofTerminalState::Unfulfillable),
            ),
            (
                FulfillmentStatus::Reverted as i32,
                ExecutionStatus::Unexecuted as i32,
                ProofStatus::Failed(ProofTerminalState::Reverted),
            ),
            (
                FulfillmentStatus::Expired as i32,
                ExecutionStatus::Unexecuted as i32,
                ProofStatus::Failed(ProofTerminalState::Expired),
            ),
            (
                FulfillmentStatus::Requested as i32,
                ExecutionStatus::Unexecutable as i32,
                ProofStatus::Failed(ProofTerminalState::Unexecutable),
            ),
            (
                FulfillmentStatus::Requested as i32,
                ExecutionStatus::ValidationFailed as i32,
                ProofStatus::Failed(ProofTerminalState::ValidationFailed),
            ),
            (
                FulfillmentStatus::Requested as i32,
                ExecutionStatus::Executed as i32,
                ProofStatus::Pending,
            ),
            (999, 998, ProofStatus::Pending),
        ];

        for (fulfillment_status, execution_status, expected) in cases {
            assert_eq!(
                check_status(fulfillment_status, execution_status),
                expected,
                "fulfillment_status={fulfillment_status} execution_status={execution_status}"
            );
        }
    }

    #[tokio::test]
    async fn wait_for_proof_polls_existing_request_without_submitting() {
        let api = Arc::new(ScriptedNetworkApi::new(
            vec![(
                proof_status(FulfillmentStatus::Fulfilled, ExecutionStatus::Executed),
                Some(test_proof()),
            )],
            None,
        ));
        let provider = provider_with_api(api.clone(), NetworkMode::Reserved);

        let proof = provider.wait_for_proof(B256::repeat_byte(0x33)).await.unwrap();

        assert!(proof.public_values.as_slice().is_empty());
        assert_eq!(api.status_calls.load(Ordering::SeqCst), 1);
        assert_eq!(api.request_calls.load(Ordering::SeqCst), 0);
        assert_eq!(api.status_ids.lock().as_slice(), &[B256::repeat_byte(0x33)]);
    }

    #[tokio::test]
    async fn cancelled_request_details_are_terminal() {
        let details = ProofRequest {
            is_canceled: true,
            fulfillment_status: FulfillmentStatus::Requested as i32,
            execution_status: ExecutionStatus::Unexecuted as i32,
            ..Default::default()
        };
        let api = Arc::new(ScriptedNetworkApi::new(
            vec![(proof_status(FulfillmentStatus::Requested, ExecutionStatus::Unexecuted), None)],
            Some(details),
        ));
        let provider = provider_with_api(api, NetworkMode::Mainnet);

        let err = provider.wait_for_proof(B256::repeat_byte(0x44)).await.unwrap_err();
        assert!(matches!(
            err,
            ProofWaitError::Terminal { state: ProofTerminalState::Cancelled, .. }
        ));
    }
}
