//! SP1 proof providers for challenged games.
//!
//! The mock pipeline computes outputs natively and submits placeholder bytes to deployments
//! whose verifier accepts arbitrary proofs. Network proving resolves [`ProofKeys`] per game
//! because games may use different prestates.

use std::{sync::Arc, time::Duration};

use alloy_primitives::B256;
use anyhow::{Context, Result, bail};
use kona_sp1_host_utils::metrics::MetricsGauge;
use sp1_sdk::{
    CpuProver, Elf, NetworkProver, ProveRequest, Prover, ProvingKey, SP1ProofMode,
    SP1ProofWithPublicValues, SP1ProvingKey, SP1Stdin, SP1VerifyingKey,
    network::{
        NetworkMode,
        proto::types::{ExecutionStatus, FulfillmentStatus},
    },
};
use tokio::time::sleep;

use crate::{
    config::{PrestatePrograms, ProofProviderConfig},
    metrics::ProposerGauge,
};

/// Polling interval (in seconds) for checking proof status.
/// Matches the SP1 SDK's internal polling interval.
pub const PROOF_STATUS_POLL_INTERVAL: u64 = 2;

/// Placeholder on-chain proof bytes submitted by the mock provider. Only a
/// deployment whose game verifier accepts arbitrary bytes (devstack's
/// `ZKMockVerifier`) can resolve games proven with these.
pub const MOCK_PROOF_BYTES: &[u8] = b"kona-sp1-mock-super-aggregation-proof";

/// Identifier returned by the SP1 prover network for a proof request.
pub type ProofId = B256;

fn current_timestamp() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .expect("System time before Unix epoch")
        .as_secs()
}

/// Container for the proving and verifying keys of one prestate's programs.
#[derive(Clone)]
pub struct ProofKeys {
    /// Proving key for the super-range program.
    pub range_pk: Arc<SP1ProvingKey>,
    /// Verifying key for the super-range program.
    pub range_vk: Arc<SP1VerifyingKey>,
    /// Proving key for the super-aggregation program.
    pub agg_pk: Arc<SP1ProvingKey>,
    /// Verifying key for the super-aggregation program. Its hash is the
    /// on-chain `absolutePrestate()`.
    pub agg_vk: Arc<SP1VerifyingKey>,
}

impl std::fmt::Debug for ProofKeys {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("ProofKeys").finish_non_exhaustive()
    }
}

/// Runs the SP1 proving-key setup for both of a prestate's programs. Setup
/// is CPU-bound and takes tens of seconds per ELF; the SDK dispatches the
/// heavy work internally, but callers must still not hold any lock across
/// this await.
pub async fn setup_proof_keys(programs: &PrestatePrograms) -> Result<ProofKeys> {
    let prover = CpuProver::new().await;
    let range_pk = prover
        .setup(Elf::Dynamic(programs.range_elf.clone().into()))
        .await
        .context("range ELF setup failed")?;
    let range_vk = range_pk.verifying_key().clone();
    let agg_pk = prover
        .setup(Elf::Dynamic(programs.aggregation_elf.clone().into()))
        .await
        .context("aggregation ELF setup failed")?;
    let agg_vk = agg_pk.verifying_key().clone();
    Ok(ProofKeys {
        range_pk: Arc::new(range_pk),
        range_vk: Arc::new(range_vk),
        agg_pk: Arc::new(agg_pk),
        agg_vk: Arc::new(agg_vk),
    })
}

/// Proof provider abstraction for generating range and aggregation proofs.
#[derive(Clone)]
pub enum ProofProvider {
    /// Real SP1 proving via the Succinct Prover Network.
    Network(NetworkProofProvider),
    /// Native-core pipeline with placeholder on-chain bytes; produces no SP1
    /// proofs at all.
    Mock(MockProofProvider),
}

impl std::fmt::Debug for ProofProvider {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Network(provider) => f.debug_tuple("Network").field(provider).finish(),
            Self::Mock(provider) => f.debug_tuple("Mock").field(provider).finish(),
        }
    }
}

impl ProofProvider {
    /// Generates a compressed super-range (or consolidation) proof.
    ///
    /// The mock provider never generates child proofs: the mock pipeline
    /// consumes natively computed outputs instead, so reaching this arm is a
    /// wiring bug.
    pub async fn generate_range_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> Result<SP1ProofWithPublicValues> {
        match self {
            Self::Network(provider) => provider.generate_range_proof(keys, stdin).await,
            Self::Mock(_) => bail!("mock provider produces no SP1 range proofs"),
        }
    }

    /// Generates the on-chain aggregation proof.
    ///
    /// See [`Self::generate_range_proof`] for the mock arm.
    pub async fn generate_agg_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> Result<SP1ProofWithPublicValues> {
        match self {
            Self::Network(provider) => provider.generate_agg_proof(keys, stdin).await,
            Self::Mock(_) => bail!("mock provider produces no SP1 aggregation proofs"),
        }
    }

    /// Returns whether this provider uses the mock pipeline.
    pub const fn is_mock(&self) -> bool {
        matches!(self, Self::Mock(_))
    }
}

/// Network-based proof provider using the SP1 prover network.
#[derive(Clone)]
pub struct NetworkProofProvider {
    prover: Arc<NetworkProver>,
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
    /// Creates a network provider with the given requester and request settings.
    pub const fn new(
        prover: Arc<NetworkProver>,
        config: ProofProviderConfig,
        network_mode: NetworkMode,
    ) -> Self {
        Self { prover, config, network_mode }
    }

    /// Generates a super-range proof via the network (request, then poll to
    /// completion).
    pub async fn generate_range_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> Result<SP1ProofWithPublicValues> {
        tracing::info!("Generating range proof via network");
        let proof_id = self.request_range_proof(keys, stdin).await?;
        self.wait_for_proof(proof_id).await
    }

    /// Generates an aggregation proof via the network (request, then poll to
    /// completion).
    pub async fn generate_agg_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> Result<SP1ProofWithPublicValues> {
        tracing::info!("Generating aggregation proof via network");
        let proof_id = self.request_agg_proof(keys, stdin).await?;
        self.wait_for_proof(proof_id).await
    }

    async fn request_range_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> Result<ProofId> {
        let proof_id = self
            .prover
            .prove(&keys.range_pk, stdin)
            .compressed()
            .skip_simulation(true)
            .strategy(self.config.range_proof_strategy)
            .timeout(Duration::from_secs(self.config.timeout))
            .min_auction_period(self.config.min_auction_period)
            .max_price_per_pgu(self.config.max_price_per_pgu)
            .cycle_limit(self.config.range_cycle_limit)
            .gas_limit(self.config.range_gas_limit)
            .request()
            .await?;

        tracing::info!(proof_id = %proof_id, "Range proof request submitted");
        Ok(proof_id)
    }

    async fn request_agg_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> Result<ProofId> {
        let proof_id = self
            .prover
            .prove(&keys.agg_pk, stdin)
            .mode(SP1ProofMode::Plonk)
            .strategy(self.config.agg_proof_strategy)
            .timeout(Duration::from_secs(self.config.timeout))
            .min_auction_period(self.config.min_auction_period)
            .max_price_per_pgu(self.config.max_price_per_pgu)
            .cycle_limit(self.config.agg_cycle_limit)
            .gas_limit(self.config.agg_gas_limit)
            .request()
            .await?;

        tracing::info!(proof_id = %proof_id, "Aggregation proof request submitted");
        Ok(proof_id)
    }

    /// Waits for a proof to be fulfilled by polling the network.
    ///
    /// Timeout behavior:
    /// - **Proving timeout** (`config.timeout`): overall maximum wait from start.
    /// - **Network call timeout** (`config.network_calls_timeout`): per-call timeout; retries on
    ///   failure.
    /// - **Auction timeout** (`config.auction_timeout`): cancels if no prover picks up the request
    ///   (mainnet only).
    /// - **Server deadline** (`status.deadline()`): server-side proving deadline.
    async fn wait_for_proof(&self, proof_id: ProofId) -> Result<SP1ProofWithPublicValues> {
        let start_time = std::time::Instant::now();
        let proving_timeout = Duration::from_secs(self.config.timeout);
        let is_mainnet = self.network_mode == NetworkMode::Mainnet;

        loop {
            if let ProvingTimeout::Exceeded { elapsed_secs } =
                check_timeout(start_time.elapsed(), proving_timeout)
            {
                tracing::warn!(
                    proof_id = %proof_id,
                    elapsed_secs,
                    timeout_secs = self.config.timeout,
                    "proving timeout exceeded"
                );
                ProposerGauge::ProvingTimeoutError.increment(1.0);
                let range_split_count = crate::env_var("RANGE_SPLIT_COUNT");
                bail!(
                    "Proving timeout: proof_id={}, elapsed={}s, timeout={}s \
                    (consider increasing {range_split_count} to shrink per-proof workloads)",
                    proof_id,
                    elapsed_secs,
                    self.config.timeout
                );
            }

            let (status, proof) = match self
                .network_call_with_timeout(
                    self.prover.get_proof_status(proof_id),
                    "get_proof_status",
                    proof_id,
                )
                .await
            {
                Ok(result) => result,
                Err(err) => {
                    tracing::warn!(proof_id = %proof_id, error = %err, "get_proof_status failed, retrying...");
                    sleep(Duration::from_secs(PROOF_STATUS_POLL_INTERVAL)).await;
                    continue;
                }
            };

            // Take a fulfilled proof before any auction or deadline
            // bookkeeping: it is already paid for, and discarding it over a
            // stale server deadline or a failed follow-up network call
            // throws away the spend.
            match check_status(status.fulfillment_status(), status.execution_status()) {
                ProofStatus::Ready => {
                    tracing::info!(proof_id = %proof_id, "Proof fulfilled");
                    return proof.ok_or_else(|| {
                        anyhow::anyhow!("Proof status is fulfilled but proof is None")
                    });
                }
                ProofStatus::Failed => {
                    bail!(
                        "Proving failed: proof_id={}, fulfillment_status={}, execution_status={}",
                        proof_id,
                        status.fulfillment_status(),
                        status.execution_status()
                    );
                }
                ProofStatus::Pending => {
                    tracing::debug!(proof_id = %proof_id, "Proof pending/assigned, continuing...");
                }
            }

            // Request details feed only the auction check, which is
            // meaningful on mainnet alone: skip the call entirely elsewhere
            // so a failing details endpoint cannot starve the deadline
            // check below, and reserved-strategy polling costs one RPC per
            // iteration instead of two. Retry transient failures.
            let request_details = if is_mainnet {
                match self
                    .network_call_with_timeout(
                        self.prover.get_proof_request(proof_id),
                        "get_proof_request",
                        proof_id,
                    )
                    .await
                {
                    Ok(result) => result,
                    Err(err) => {
                        tracing::warn!(proof_id = %proof_id, error = %err, "get_proof_request failed, retrying...");
                        sleep(Duration::from_secs(PROOF_STATUS_POLL_INTERVAL)).await;
                        continue;
                    }
                }
            } else {
                None
            };

            let current_time = current_timestamp();

            if let Some(details) = &request_details {
                let timeout_secs = self.config.auction_timeout;
                if let AuctionTimeout::Exceeded { elapsed_secs } = check_auction(
                    is_mainnet,
                    details.fulfillment_status,
                    details.created_at,
                    timeout_secs,
                    current_time,
                ) {
                    tracing::warn!(
                        proof_id = %proof_id,
                        elapsed_secs,
                        timeout_secs,
                        "auction timeout exceeded, cancelling request"
                    );
                    if let Err(err) = self
                        .network_call_with_timeout(
                            self.prover.cancel_request(proof_id),
                            "cancel_request",
                            proof_id,
                        )
                        .await
                    {
                        tracing::error!(proof_id = %proof_id, error = %err, "Failed to cancel proof request");
                    }
                    ProposerGauge::AuctionTimeoutError.increment(1.0);
                    bail!(
                        "Auction timeout: proof_id={}, elapsed={}s, timeout={}s",
                        proof_id,
                        elapsed_secs,
                        timeout_secs
                    );
                }
            }

            if let Deadline::Exceeded { deadline } = check_deadline(status.deadline(), current_time)
            {
                tracing::warn!(
                    proof_id = %proof_id,
                    deadline,
                    current_time,
                    "Proof request deadline exceeded"
                );
                ProposerGauge::DeadlineExceededError.increment(1.0);
                bail!(
                    "Deadline exceeded: proof_id={}, deadline={}, current_time={}",
                    proof_id,
                    deadline,
                    current_time
                );
            }

            sleep(Duration::from_secs(PROOF_STATUS_POLL_INTERVAL)).await;
        }
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

/// Mock proof provider: the pipeline runs the guest logic natively (witness
/// collection already computes the outputs) and submits [`MOCK_PROOF_BYTES`]
/// on-chain. Holds no SP1 client, requires no ELFs and no credentials.
#[derive(Clone, Copy, Debug, Default)]
pub struct MockProofProvider;

/// Client-side proof request timeout status.
#[derive(Debug, PartialEq, Eq)]
pub enum ProvingTimeout {
    /// Polling may continue.
    Ok,
    /// The configured timeout elapsed.
    Exceeded {
        /// Seconds elapsed since the request started.
        elapsed_secs: u64,
    },
}

/// Equality remains within the timeout.
pub fn check_timeout(elapsed: Duration, timeout: Duration) -> ProvingTimeout {
    if elapsed > timeout {
        ProvingTimeout::Exceeded { elapsed_secs: elapsed.as_secs() }
    } else {
        ProvingTimeout::Ok
    }
}

/// Prover-network auction timeout status.
#[derive(Debug, PartialEq, Eq)]
pub enum AuctionTimeout {
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
pub const fn check_auction(
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
pub enum Deadline {
    /// The deadline remains in the future.
    Ok,
    /// The deadline has been reached.
    Exceeded {
        /// Server-side deadline in Unix seconds.
        deadline: u64,
    },
}

/// Treats equality as expired.
pub const fn check_deadline(deadline: u64, current_time: u64) -> Deadline {
    if current_time >= deadline { Deadline::Exceeded { deadline } } else { Deadline::Ok }
}

/// Combined fulfillment and execution status for a proof request.
#[derive(Debug, PartialEq, Eq)]
pub enum ProofStatus {
    /// A fulfilled proof is available.
    Ready,
    /// The request cannot produce a proof.
    Failed,
    /// Polling should continue.
    Pending,
}

/// Fulfillment takes precedence over terminal execution status.
pub fn check_status(fulfillment_status: i32, execution_status: i32) -> ProofStatus {
    let fulfillment_status = FulfillmentStatus::try_from(fulfillment_status).ok();
    let execution_status = ExecutionStatus::try_from(execution_status).ok();

    if matches!(fulfillment_status, Some(FulfillmentStatus::Fulfilled)) {
        return ProofStatus::Ready;
    }
    if matches!(
        fulfillment_status,
        Some(
            FulfillmentStatus::Unfulfillable |
                FulfillmentStatus::Reverted |
                FulfillmentStatus::Expired
        )
    ) || matches!(
        execution_status,
        Some(ExecutionStatus::Unexecutable | ExecutionStatus::ValidationFailed)
    ) {
        return ProofStatus::Failed;
    }
    ProofStatus::Pending
}

#[cfg(test)]
mod tests {
    use std::time::Duration;

    use super::*;

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
    fn status_transitions() {
        let fulfillment_statuses = [
            FulfillmentStatus::UnspecifiedFulfillmentStatus,
            FulfillmentStatus::Requested,
            FulfillmentStatus::Assigned,
            FulfillmentStatus::Fulfilled,
            FulfillmentStatus::Unfulfillable,
            FulfillmentStatus::Reverted,
            FulfillmentStatus::Expired,
        ];
        let execution_statuses = [
            ExecutionStatus::UnspecifiedExecutionStatus,
            ExecutionStatus::Unexecuted,
            ExecutionStatus::Executed,
            ExecutionStatus::Unexecutable,
            ExecutionStatus::ValidationFailed,
        ];

        let mut cases = Vec::new();
        for execution_status in execution_statuses {
            cases.push((
                FulfillmentStatus::Fulfilled as i32,
                execution_status as i32,
                ProofStatus::Ready,
            ));
        }
        cases.push((FulfillmentStatus::Fulfilled as i32, 999, ProofStatus::Ready));

        for fulfillment_status in [
            FulfillmentStatus::Unfulfillable,
            FulfillmentStatus::Reverted,
            FulfillmentStatus::Expired,
        ] {
            cases.push((
                fulfillment_status as i32,
                ExecutionStatus::Unexecuted as i32,
                ProofStatus::Failed,
            ));
            cases.push((fulfillment_status as i32, 999, ProofStatus::Failed));
        }

        for execution_status in [ExecutionStatus::Unexecutable, ExecutionStatus::ValidationFailed] {
            cases.push((
                FulfillmentStatus::Requested as i32,
                execution_status as i32,
                ProofStatus::Failed,
            ));
            cases.push((999, execution_status as i32, ProofStatus::Failed));
        }

        for fulfillment_status in &fulfillment_statuses[..3] {
            for execution_status in &execution_statuses[..3] {
                cases.push((
                    *fulfillment_status as i32,
                    *execution_status as i32,
                    ProofStatus::Pending,
                ));
            }
        }
        cases.push((FulfillmentStatus::Requested as i32, 999, ProofStatus::Pending));
        cases.push((999, ExecutionStatus::Unexecuted as i32, ProofStatus::Pending));
        cases.push((999, 998, ProofStatus::Pending));

        for (fulfillment_status, execution_status, expected) in cases {
            assert_eq!(
                check_status(fulfillment_status, execution_status),
                expected,
                "fulfillment_status={fulfillment_status} execution_status={execution_status}"
            );
        }
    }
}
