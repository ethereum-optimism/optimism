//! Additional configuration for the OP builder

use alloy_rpc_types_engine::PayloadId;
use reth_optimism_txpool::interop::InteropFailsafe;
use std::{
    collections::HashMap,
    sync::{
        Arc, Mutex, MutexGuard,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    time::{Duration, Instant},
};
use tokio::sync::Notify;

/// Settings for the OP builder.
#[derive(Debug, Clone, Default)]
pub struct OpBuilderConfig {
    /// Data availability configuration for the OP builder.
    pub da_config: OpDAConfig,
    /// Gas limit configuration for the OP builder.
    pub gas_limit_config: OpGasLimitConfig,
    /// Local SDM `PostExec` production operator opt-in. Shared with the admin RPC.
    pub operator_sdm_opt_in: OperatorSdmOptIn,
    /// Interop failsafe gate. Set by the interop filter client; read by the builder to exclude
    /// interop txs from blocks while it is enabled.
    pub interop_failsafe: InteropFailsafe,
    /// Maximum cumulative uncompressed (EIP-2718 encoded) block size in bytes.
    ///
    /// `None` disables the limit (the historical behavior). When set, the payload builder stops
    /// pulling mempool transactions once including the next one would push the block's total
    /// EIP-2718 encoded transaction size past this value. This bounds the size of the
    /// `engine_getPayload` response so it does not exceed the limits assumed by consensus-layer
    /// clients (e.g. the common 10 MiB JSON payload cap). op-geth enforces an equivalent but
    /// non-configurable cap via `params.MaxBlockSize`.
    pub max_uncompressed_block_size: Option<u64>,
    /// Decides when a payload is worth sealing before the consensus layer's deadline. Shared
    /// with the engine API, which serves the readiness it records.
    pub multi_block_policy: MultiBlockPolicy,
}

impl OpBuilderConfig {
    /// Creates a new OP builder configuration with the given data availability configuration.
    pub fn new(da_config: OpDAConfig, gas_limit_config: OpGasLimitConfig) -> Self {
        Self {
            da_config,
            gas_limit_config,
            operator_sdm_opt_in: OperatorSdmOptIn::default(),
            interop_failsafe: InteropFailsafe::default(),
            max_uncompressed_block_size: None,
            multi_block_policy: MultiBlockPolicy::default(),
        }
    }

    /// Returns the Data Availability configuration for the OP builder, if it has configured
    /// constraints.
    pub fn constrained_da_config(&self) -> Option<&OpDAConfig> {
        if self.da_config.is_empty() { None } else { Some(&self.da_config) }
    }
}

/// How long a payload job's build state is retained when nothing ever resolves it — a build that
/// the consensus layer abandons leaves an entry behind, so entries also expire by age.
const JOB_RETENTION: Duration = Duration::from_secs(60);

/// Decides when a payload under construction is worth sealing, so the consensus layer can seal
/// early and fit more than one block into a block time.
///
/// The builder evaluates the policy on every build and records the verdict here; the engine
/// API's `engine_awaitPayloadReadyV1` serves it. Both thresholds are optional and unset by
/// default: an unconfigured policy never reports ready, leaving the consensus layer to seal at
/// its own deadline as before.
#[derive(Debug, Clone, Default)]
pub struct MultiBlockPolicy {
    /// Ready once the built payload contains at least this many user (mempool) transactions.
    min_txs: Option<u64>,
    /// Ready once a payload holding user transactions has been building for at least this long.
    min_build_time: Option<Duration>,
    jobs: Arc<Jobs>,
}

/// Per-payload-job build state, shared between the payload builder and the engine API.
#[derive(Debug, Default)]
struct Jobs {
    state: Mutex<HashMap<PayloadId, JobState>>,
    /// Woken whenever a build is recorded, so waiters re-evaluate their verdict and their timer.
    ready: Notify,
}

#[derive(Debug, Clone, Copy)]
struct JobState {
    /// When the job started building the payload it currently holds.
    started: Instant,
    /// Whether the job is ready irrespective of how long it has been building.
    ready: bool,
    /// Whether the most recent build produced any user transaction.
    has_user_txs: bool,
}

impl JobState {
    const fn started_at(now: Instant) -> Self {
        Self { started: now, ready: false, has_user_txs: false }
    }
}

/// A job's verdict, together with the instant at which build time alone would make it ready.
struct Readiness {
    ready: bool,
    time_ready_at: Option<Instant>,
}

impl MultiBlockPolicy {
    /// The longest [`Self::wait_ready`] waits, whatever the caller asks for. It bounds the damage
    /// a bogus request can do to an engine-API connection; a consensus layer asks for at most the
    /// time left in its slot.
    pub const MAX_WAIT: Duration = Duration::from_secs(12);

    /// Creates a policy from its thresholds. `None` disables a threshold; a policy with no
    /// threshold at all is never satisfied.
    pub fn new(min_txs: Option<u64>, min_build_time: Option<Duration>) -> Self {
        Self { min_txs, min_build_time, jobs: Default::default() }
    }

    /// Returns whether any threshold is configured.
    pub const fn is_configured(&self) -> bool {
        self.min_txs.is_some() || self.min_build_time.is_some()
    }

    /// Records that a build for the given payload job has started.
    ///
    /// `first_build` marks a build for a job that holds no payload yet: it starts the job's
    /// clock and clears any earlier verdict, so a payload id a later job reuses never inherits
    /// the readiness of the job before it. A continuation build keeps the clock running.
    pub fn begin_build(&self, id: PayloadId, first_build: bool) {
        if !self.is_configured() {
            return;
        }
        let now = Instant::now();
        let mut jobs = self.lock();
        jobs.retain(|_, job| now.duration_since(job.started) < JOB_RETENTION);
        if first_build {
            jobs.insert(id, JobState::started_at(now));
        } else {
            jobs.entry(id).or_insert_with(|| JobState::started_at(now));
        }
    }

    /// Records a finished build carrying `user_txs` user transactions and returns whether the
    /// payload now satisfies the policy.
    pub fn record_built_payload(&self, id: PayloadId, user_txs: u64) -> bool {
        if !self.is_configured() {
            return false;
        }
        self.begin_build(id, false);
        let mut jobs = self.lock();
        let Some(job) = jobs.get_mut(&id) else { return false };
        job.has_user_txs = user_txs > 0;
        job.ready |= self.min_txs.is_some_and(|min| user_txs >= min);
        let job = *job;
        drop(jobs);
        // A build that is not ready can still have brought the build-time threshold into play,
        // so waiters are woken either way to recompute when it comes due.
        self.jobs.ready.notify_waiters();
        self.evaluate(&job).ready
    }

    /// Marks the job ready outright, for a payload that is final however long it builds.
    pub fn mark_ready(&self, id: PayloadId) {
        if !self.is_configured() {
            return;
        }
        let now = Instant::now();
        let mut jobs = self.lock();
        jobs.entry(id).or_insert_with(|| JobState::started_at(now)).ready = true;
        drop(jobs);
        self.jobs.ready.notify_waiters();
    }

    /// Returns whether the given payload job satisfies the policy.
    pub fn is_ready(&self, id: PayloadId) -> bool {
        self.readiness(id).ready
    }

    /// Drops the given job's build state, which the engine API does once the job resolves or
    /// turns out to be unknown to the payload builder.
    pub fn forget(&self, id: PayloadId) {
        if self.is_configured() {
            self.lock().remove(&id);
        }
    }

    /// Waits for the given payload job to satisfy the policy, for at most `max_wait` (itself
    /// capped at [`Self::MAX_WAIT`]). Returns whether it is ready.
    pub async fn wait_ready(&self, id: PayloadId, max_wait: Duration) -> bool {
        let deadline = Instant::now() + max_wait.min(Self::MAX_WAIT);
        loop {
            // Register for the wake-up before checking readiness, so a verdict recorded in
            // between is not missed.
            let mut notified = core::pin::pin!(self.jobs.ready.notified());
            notified.as_mut().enable();

            let Readiness { ready, time_ready_at } = self.readiness(id);
            if ready {
                return true;
            }
            let now = Instant::now();
            if now >= deadline {
                return false;
            }
            // The build-time threshold comes due on its own, without a builder update.
            let wake = time_ready_at.map_or(deadline, |at| at.min(deadline));
            let _ = tokio::time::timeout(wake.saturating_duration_since(now), notified).await;
        }
    }

    fn readiness(&self, id: PayloadId) -> Readiness {
        if !self.is_configured() {
            return Readiness { ready: false, time_ready_at: None };
        }
        let job = self.lock().get(&id).copied();
        job.map_or(Readiness { ready: false, time_ready_at: None }, |job| self.evaluate(&job))
    }

    fn evaluate(&self, job: &JobState) -> Readiness {
        // Build time alone must never seal an empty payload: an idle sequencer would otherwise
        // fill every block time with a group of empty siblings.
        let time_ready_at =
            self.min_build_time.filter(|_| job.has_user_txs).map(|min| job.started + min);
        Readiness {
            ready: job.ready || time_ready_at.is_some_and(|at| Instant::now() >= at),
            time_ready_at,
        }
    }

    fn lock(&self) -> MutexGuard<'_, HashMap<PayloadId, JobState>> {
        // A panic while holding the lock loses at most one job's readiness, which degrades to
        // the consensus layer sealing at its deadline.
        self.jobs.state.lock().unwrap_or_else(|err| err.into_inner())
    }
}

/// Shareable operator opt-in flag for SDM `PostExec` production.
///
/// `false` on construction. The admin RPC writes; the payload builder reads. The protocol gate
/// (chain spec Interop activation) is checked separately; both must be true to actually produce.
#[derive(Debug, Clone, Default)]
pub struct OperatorSdmOptIn {
    inner: Arc<AtomicBool>,
}

impl OperatorSdmOptIn {
    /// Returns the current opt-in state.
    pub fn enabled(&self) -> bool {
        self.inner.load(Ordering::Acquire)
    }

    /// Sets the opt-in state.
    pub fn set(&self, enabled: bool) {
        self.inner.store(enabled, Ordering::Release);
    }
}

/// Contains the Data Availability configuration for the OP builder.
///
/// This type is shareable and can be used to update the DA configuration for the OP payload
/// builder.
#[derive(Debug, Clone, Default)]
pub struct OpDAConfig {
    inner: Arc<OpDAConfigInner>,
}

impl OpDAConfig {
    /// Creates a new Data Availability configuration with the given maximum sizes.
    pub fn new(max_da_tx_size: u64, max_da_block_size: u64) -> Self {
        let this = Self::default();
        this.set_max_da_size(max_da_tx_size, max_da_block_size);
        this
    }

    /// Returns whether the configuration is empty.
    pub fn is_empty(&self) -> bool {
        self.max_da_tx_size().is_none() && self.max_da_block_size().is_none()
    }

    /// Returns the max allowed data availability size per transactions, if any.
    pub fn max_da_tx_size(&self) -> Option<u64> {
        let val = self.inner.max_da_tx_size.load(std::sync::atomic::Ordering::Relaxed);
        if val == 0 { None } else { Some(val) }
    }

    /// Returns the max allowed data availability size per block, if any.
    pub fn max_da_block_size(&self) -> Option<u64> {
        let val = self.inner.max_da_block_size.load(std::sync::atomic::Ordering::Relaxed);
        if val == 0 { None } else { Some(val) }
    }

    /// Sets the maximum data availability size currently allowed for inclusion. 0 means no maximum.
    pub fn set_max_da_size(&self, max_da_tx_size: u64, max_da_block_size: u64) {
        self.set_max_tx_size(max_da_tx_size);
        self.set_max_block_size(max_da_block_size);
    }

    /// Sets the maximum data availability size per transaction currently allowed for inclusion. 0
    /// means no maximum.
    pub fn set_max_tx_size(&self, max_da_tx_size: u64) {
        self.inner.max_da_tx_size.store(max_da_tx_size, std::sync::atomic::Ordering::Relaxed);
    }

    /// Sets the maximum data availability size per block currently allowed for inclusion. 0 means
    /// no maximum.
    pub fn set_max_block_size(&self, max_da_block_size: u64) {
        self.inner.max_da_block_size.store(max_da_block_size, std::sync::atomic::Ordering::Relaxed);
    }
}

#[derive(Debug, Default)]
struct OpDAConfigInner {
    /// Don't include any transactions with data availability size larger than this in any built
    /// block
    ///
    /// 0 means no limit.
    max_da_tx_size: AtomicU64,
    /// Maximum total data availability size for a block
    ///
    /// 0 means no limit.
    max_da_block_size: AtomicU64,
}

/// Contains the Gas Limit configuration for the OP builder.
///
/// This type is shareable and can be used to update the Gas Limit configuration for the OP payload
/// builder.
#[derive(Debug, Clone, Default)]
pub struct OpGasLimitConfig {
    /// Gas limit for a transaction
    ///
    /// 0 means use the default gas limit.
    gas_limit: Arc<AtomicU64>,
}

impl OpGasLimitConfig {
    /// Creates a new Gas Limit configuration with the given maximum gas limit.
    pub fn new(max_gas_limit: u64) -> Self {
        let this = Self::default();
        this.set_gas_limit(max_gas_limit);
        this
    }
    /// Returns the gas limit for a transaction, if any.
    pub fn gas_limit(&self) -> Option<u64> {
        let val = self.gas_limit.load(std::sync::atomic::Ordering::Relaxed);
        if val == 0 { None } else { Some(val) }
    }
    /// Sets the gas limit for a transaction. 0 means use the default gas limit.
    pub fn set_gas_limit(&self, gas_limit: u64) {
        self.gas_limit.store(gas_limit, std::sync::atomic::Ordering::Relaxed);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_da() {
        let da = OpDAConfig::default();
        assert_eq!(da.max_da_tx_size(), None);
        assert_eq!(da.max_da_block_size(), None);
        da.set_max_da_size(100, 200);
        assert_eq!(da.max_da_tx_size(), Some(100));
        assert_eq!(da.max_da_block_size(), Some(200));
        da.set_max_da_size(0, 0);
        assert_eq!(da.max_da_tx_size(), None);
        assert_eq!(da.max_da_block_size(), None);
    }

    #[test]
    fn test_da_constrained() {
        let config = OpBuilderConfig::default();
        assert!(config.constrained_da_config().is_none());
    }

    #[tokio::test]
    async fn multi_block_policy_without_thresholds_is_never_ready() {
        let policy = MultiBlockPolicy::default();
        let id = PayloadId::new([1; 8]);

        assert!(!policy.is_configured());
        policy.begin_build(id, true);
        assert!(!policy.record_built_payload(id, 1_000));
        policy.mark_ready(id);
        assert!(!policy.is_ready(id));
        assert!(!policy.wait_ready(id, Duration::from_millis(10)).await);
    }

    #[tokio::test]
    async fn multi_block_policy_is_ready_at_min_txs() {
        let policy = MultiBlockPolicy::new(Some(2), None);
        let id = PayloadId::new([2; 8]);

        policy.begin_build(id, true);
        assert!(!policy.record_built_payload(id, 1));
        assert!(!policy.is_ready(id));

        assert!(policy.record_built_payload(id, 2));
        assert!(policy.is_ready(id));
        assert!(policy.wait_ready(id, Duration::ZERO).await);

        // A job the builder never built for is not ready; the engine API reports such a payload
        // id as unknown instead.
        assert!(!policy.is_ready(PayloadId::new([3; 8])));

        policy.forget(id);
        assert!(!policy.is_ready(id));
    }

    #[tokio::test]
    async fn multi_block_policy_is_ready_at_min_build_time() {
        let policy = MultiBlockPolicy::new(None, Some(Duration::from_millis(200)));
        let id = PayloadId::new([4; 8]);

        policy.begin_build(id, true);
        assert!(!policy.record_built_payload(id, 1), "the threshold cannot have elapsed yet");

        // The threshold comes due on its own, without a further build.
        assert!(policy.wait_ready(id, Duration::from_secs(30)).await);
        assert!(policy.is_ready(id));
    }

    /// An idle sequencer must produce one block per block time, not a group of empty siblings,
    /// so build time alone never makes an empty payload worth sealing.
    #[tokio::test]
    async fn multi_block_policy_build_time_never_seals_an_empty_payload() {
        let policy = MultiBlockPolicy::new(Some(4), Some(Duration::from_millis(1)));
        let id = PayloadId::new([5; 8]);

        policy.begin_build(id, true);
        tokio::time::sleep(Duration::from_millis(20)).await;
        assert!(!policy.record_built_payload(id, 0), "an empty payload is never worth sealing");
        assert!(!policy.wait_ready(id, Duration::from_millis(20)).await);

        // One user transaction is enough once the threshold has elapsed, even far below min-txs.
        assert!(policy.record_built_payload(id, 1));
    }

    /// A payload id is a hash of the attributes and the parent, so a consensus layer that
    /// abandons a build and re-issues the same attributes gets the same id back for a job that
    /// has just started.
    #[tokio::test]
    async fn multi_block_policy_first_build_clears_a_reused_payload_id() {
        let policy = MultiBlockPolicy::new(Some(1), None);
        let id = PayloadId::new([6; 8]);

        policy.begin_build(id, true);
        assert!(policy.record_built_payload(id, 1));

        policy.begin_build(id, true);
        assert!(!policy.is_ready(id), "a new job must not inherit the previous job's verdict");

        // A continuation build of the same job keeps what the job has established.
        assert!(policy.record_built_payload(id, 1));
        policy.begin_build(id, false);
        assert!(policy.is_ready(id));
    }

    /// A payload the builder froze cannot improve, so the consensus layer is told at once rather
    /// than waiting out its slot for a block that is already final.
    #[tokio::test]
    async fn multi_block_policy_mark_ready_is_immediate() {
        let policy = MultiBlockPolicy::new(Some(1_000), None);
        let id = PayloadId::new([7; 8]);

        policy.begin_build(id, true);
        assert!(!policy.record_built_payload(id, 1));

        policy.mark_ready(id);
        assert!(policy.is_ready(id));
        assert!(policy.wait_ready(id, Duration::ZERO).await);
    }

    #[test]
    fn test_gas_limit() {
        let gas_limit = OpGasLimitConfig::default();
        assert_eq!(gas_limit.gas_limit(), None);
        gas_limit.set_gas_limit(50000);
        assert_eq!(gas_limit.gas_limit(), Some(50000));
        gas_limit.set_gas_limit(0);
        assert_eq!(gas_limit.gas_limit(), None);
    }
}
