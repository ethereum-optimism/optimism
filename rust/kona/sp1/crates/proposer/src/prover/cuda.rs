//! Local SP1 proving on NVIDIA GPUs, through one `sp1-gpu-server` child per CUDA device.
//!
//! Each request runs in a spawned task that is never aborted. The SDK sends a request and reads
//! its response on one socket per device; a future dropped between the two leaves the response
//! for the next request on that device.

use std::{
    collections::HashMap, ffi::OsString, panic::AssertUnwindSafe, path::PathBuf, sync::Arc,
    time::Duration,
};

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow, ensure};
use async_trait::async_trait;
use futures::FutureExt;
use kona_sp1_host_utils::metrics::MetricsGauge;
use parking_lot::Mutex;
use sp1_core_executor::{ExecutionError, ExecutionReport, SP1ContextBuilder, SP1CoreOpts};
use sp1_primitives::SP1_CRATE_VERSION;
use sp1_sdk::{
    CudaProver, Elf, HashableKey, LightProver, ProveRequest, Prover, ProvingKey, SP1ProofMode,
    SP1ProofWithPublicValues, SP1Stdin, SP1VerifyingKey, cuda::builder::CudaProverBuilder,
};
use tokio::{
    sync::{Semaphore, SemaphorePermit, watch},
    time::Instant,
};

use super::{ProofId, ProofKeys, ProofTerminalState, ProofWaitError};
use crate::{
    config::{LocalCudaConfig, ProofProviderConfig},
    metrics::ProposerGauge,
    proving::stdin_encoded_len,
};

/// The image that carries `sp1-gpu-server`, the CUDA runtime, and the PLONK circuits.
const CUDA_IMAGE: &str = "op-zk-proposer-cuda";

type CudaClientError = <CudaProver as Prover>::Error;

/// Local proving on CUDA devices, one proof per device at a time. Clones share one state.
#[derive(Clone)]
pub struct LocalCudaProofProvider {
    core: Arc<Core<CudaProver>>,
    /// Verifies proofs. A `CudaProver` clone would keep its device client alive.
    verifier: LightProver,
}

impl std::fmt::Debug for LocalCudaProofProvider {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("LocalCudaProofProvider")
            .field("devices", &self.core.device_ids)
            .finish_non_exhaustive()
    }
}

impl LocalCudaProofProvider {
    /// Checks the GPU server binary and starts one server per configured device.
    pub async fn new(
        config: &LocalCudaConfig,
        provider_config: ProofProviderConfig,
    ) -> Result<Self> {
        check_gpu_server(std::env::var_os("HOME")).await?;
        ensure!(
            cfg!(target_arch = "x86_64"),
            "local CUDA proving needs an x86-64 host; run the {CUDA_IMAGE} image"
        );
        let opts =
            SP1CoreOpts { memory_limit: config.memory_limit_bytes.get(), ..Default::default() };
        let verifier = LightProver::new_with_opts(opts).await;
        let mut devices = Vec::with_capacity(config.devices.len());
        for &id in &config.devices {
            // `build` panics when the server does not start.
            let server = tokio::spawn(CudaProverBuilder::new().with_device_id(id).build())
                .await
                .map_err(|err| {
                    anyhow!(
                        "sp1-gpu-server for CUDA device {id} did not start: {err}; check the \
                         device and the {CUDA_IMAGE} image"
                    )
                })?;
            tracing::info!(device = id, "Started sp1-gpu-server");
            devices.push(Device { id, server, keys: HashMap::new() });
        }
        let core = Core::new(Box::new(verifier.clone()), devices, provider_config);
        Ok(Self { core, verifier })
    }

    /// Resolves with the first failure that leaves a GPU server unusable. The caller must return
    /// from `main`: dropping the runtime kills every server, and `std::process::exit` does not.
    pub async fn fatal(&self) -> anyhow::Error {
        self.core.fatal().await
    }

    pub(super) fn request_range_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> ProofId {
        self.core.request(Program::Range, PrestateKeys::from(keys), stdin)
    }

    pub(super) fn request_log_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> ProofId {
        self.core.request(Program::LogCommitment, PrestateKeys::from(keys), stdin)
    }

    pub(super) fn request_aggregation_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> ProofId {
        self.core.request(Program::Aggregation, PrestateKeys::from(keys), stdin)
    }

    pub(super) async fn wait_for_proof(
        &self,
        proof_id: ProofId,
    ) -> Result<SP1ProofWithPublicValues, ProofWaitError> {
        self.core.wait(proof_id).await
    }

    pub(super) fn verify(
        &self,
        proof: &SP1ProofWithPublicValues,
        verifying_key: &SP1VerifyingKey,
    ) -> Result<()> {
        Prover::verify(&self.verifier, proof, verifying_key, None).map_err(anyhow::Error::new)
    }
}

/// Checks that `$HOME/.sp1/bin/sp1-gpu-server` exists and reports the linked SP1 version.
/// Otherwise the SDK downloads a server and stops running servers with `systemctl`.
async fn check_gpu_server(home: Option<OsString>) -> Result<()> {
    let home = home.with_context(|| format!("$HOME is not set; run the {CUDA_IMAGE} image"))?;
    let path = PathBuf::from(home).join(".sp1/bin/sp1-gpu-server");
    ensure!(path.is_file(), "{} does not exist; run the {CUDA_IMAGE} image", path.display());
    let output =
        tokio::process::Command::new(&path).arg("--version").output().await.with_context(|| {
            format!("run {} --version; check the {CUDA_IMAGE} image", path.display())
        })?;
    let version = String::from_utf8_lossy(&output.stdout);
    ensure!(
        version.trim() == SP1_CRATE_VERSION,
        "{} reports version {:?}, but op-zk-proposer links SP1 {SP1_CRATE_VERSION}; run the \
         {CUDA_IMAGE} image of this build",
        path.display(),
        version.trim()
    );
    Ok(())
}

/// One device's `sp1-gpu-server`.
#[async_trait]
trait DeviceServer: Send + Sync + 'static {
    type Key: Send + Sync + 'static;

    async fn setup(&self, elf: Elf) -> Result<Self::Key, CudaClientError>;

    fn vk_hash(key: &Self::Key) -> [u32; 8];

    async fn prove(
        &self,
        key: &Self::Key,
        stdin: SP1Stdin,
        mode: SP1ProofMode,
    ) -> Result<SP1ProofWithPublicValues, CudaClientError>;
}

#[async_trait]
impl DeviceServer for CudaProver {
    type Key = <Self as Prover>::ProvingKey;

    async fn setup(&self, elf: Elf) -> Result<Self::Key, CudaClientError> {
        Prover::setup(self, elf).await
    }

    fn vk_hash(key: &Self::Key) -> [u32; 8] {
        key.verifying_key().hash_u32()
    }

    async fn prove(
        &self,
        key: &Self::Key,
        stdin: SP1Stdin,
        mode: SP1ProofMode,
    ) -> Result<SP1ProofWithPublicValues, CudaClientError> {
        Prover::prove(self, key, stdin).mode(mode).await
    }
}

/// Runs a request's program before it reaches a device; the only cycle and memory guard, since
/// the server enforces neither.
#[async_trait]
trait PreExecutor: Send + Sync + 'static {
    async fn execute(&self, elf: Elf, stdin: SP1Stdin, cycle_limit: u64)
    -> Result<ExecutionReport>;
}

#[async_trait]
impl PreExecutor for LightProver {
    async fn execute(
        &self,
        elf: Elf,
        stdin: SP1Stdin,
        cycle_limit: u64,
    ) -> Result<ExecutionReport> {
        // Without gas the executor never checks the cycle limit and reports exit code 0.
        let context = SP1ContextBuilder::new()
            .max_cycles(cycle_limit)
            .calculate_gas(true)
            .set_deferred_proof_verification(true)
            .build();
        let (_, _, report) = self.inner().execute(&elf, stdin, context).await?;
        Ok(report)
    }
}

#[derive(Clone, Copy, Debug)]
enum Program {
    Range,
    LogCommitment,
    Aggregation,
}

impl Program {
    const fn name(self) -> &'static str {
        match self {
            Self::Range => "super-range",
            Self::LogCommitment => "log-commitment",
            Self::Aggregation => "super-aggregation",
        }
    }

    const fn mode(self) -> SP1ProofMode {
        match self {
            Self::Range | Self::LogCommitment => SP1ProofMode::Compressed,
            Self::Aggregation => SP1ProofMode::Plonk,
        }
    }

    const fn cycle_limit(self, config: &ProofProviderConfig) -> u64 {
        match self {
            Self::Range => config.range_cycle_limit,
            Self::LogCommitment => config.log_cycle_limit,
            Self::Aggregation => config.agg_cycle_limit,
        }
    }
}

/// One value per program.
#[derive(Clone, Debug, PartialEq, Eq)]
struct ByProgram<T> {
    range: T,
    log_commitment: T,
    aggregation: T,
}

impl<T> ByProgram<T> {
    const fn get(&self, program: Program) -> &T {
        match program {
            Program::Range => &self.range,
            Program::LogCommitment => &self.log_commitment,
            Program::Aggregation => &self.aggregation,
        }
    }
}

/// The ELF and verifying-key hash of one program.
#[derive(Clone)]
struct ProgramKey {
    elf: Elf,
    vk_hash: [u32; 8],
}

/// The programs of one prestate.
#[derive(Clone)]
struct PrestateKeys {
    /// The aggregation verifying-key hash, the on-chain prestate.
    prestate: B256,
    programs: ByProgram<ProgramKey>,
}

impl From<&ProofKeys> for PrestateKeys {
    fn from(keys: &ProofKeys) -> Self {
        let program = |pk: &sp1_sdk::SP1ProvingKey| ProgramKey {
            elf: pk.elf().clone(),
            vk_hash: pk.verifying_key().hash_u32(),
        };
        Self {
            prestate: B256::from(keys.agg_vk.bytes32_raw()),
            programs: ByProgram {
                range: program(&keys.range_pk),
                log_commitment: program(&keys.log_pk),
                aggregation: program(&keys.agg_pk),
            },
        }
    }
}

/// Why a local request produced no proof.
#[derive(Debug)]
enum Failure {
    /// The input breaks a limit or the guest fails; resubmitting it cannot succeed.
    Unexecutable(anyhow::Error),
    /// The request can be submitted again.
    ProverFailed(anyhow::Error),
    /// A GPU server connection failed, or a task panicked while it held a device; the process
    /// must exit.
    Fatal(anyhow::Error),
}

/// Pre-execution attempts while the executor's RSS monitor kills the child. One kill acts on a
/// host-dependent sample, so SP1's own worker retries it too.
const MEMORY_MONITOR_ATTEMPTS: usize = 3;

/// Execution errors are deterministic for the program and input, as SPN treats them, except an
/// external kill and `Other` (a failed or abnormally ended child, such as a full `/dev/shm`).
fn classify_execution_error(err: anyhow::Error) -> Failure {
    match err.downcast_ref::<ExecutionError>() {
        None | Some(ExecutionError::ChildKilled() | ExecutionError::Other(_)) => {
            Failure::ProverFailed(err)
        }
        Some(_) => Failure::Unexecutable(err),
    }
}

fn is_memory_monitor_kill(err: &anyhow::Error) -> bool {
    matches!(err.downcast_ref::<ExecutionError>(), Some(ExecutionError::KilledByMemoryMonitor(_)))
}

/// The SDK frames each server request with a `u32` length; a longer request corrupts the socket.
fn fits_server_frame(stdin_bytes: u64) -> bool {
    /// Room for the request's other fields.
    const REQUEST_OVERHEAD: u64 = 1024;
    stdin_bytes.saturating_add(REQUEST_OVERHEAD) <= u64::from(u32::MAX)
}

/// `CudaClientError` hides its variant; only its text names it. The server answers a failed
/// request with an internal or prover error. Any other error is a broken connection.
fn classify_client_error(err: CudaClientError) -> Failure {
    let text = err.to_string();
    let err = anyhow::Error::new(err);
    if text.contains("The server returned an internal error") ||
        text.contains("The server returned a prover error")
    {
        Failure::ProverFailed(err)
    } else {
        Failure::Fatal(err)
    }
}

/// A device's server and the proving keys it holds, by prestate. Keys are never evicted.
struct Device<S: DeviceServer> {
    id: u32,
    server: S,
    keys: HashMap<B256, ByProgram<S::Key>>,
}

impl<S: DeviceServer> Device<S> {
    async fn ensure_keys(&mut self, keys: &PrestateKeys) -> Result<(), Failure> {
        if self.keys.contains_key(&keys.prestate) {
            return Ok(());
        }
        let set = ByProgram {
            range: self.setup(Program::Range, keys).await?,
            log_commitment: self.setup(Program::LogCommitment, keys).await?,
            aggregation: self.setup(Program::Aggregation, keys).await?,
        };
        self.keys.insert(keys.prestate, set);
        Ok(())
    }

    async fn setup(&self, program: Program, keys: &PrestateKeys) -> Result<S::Key, Failure> {
        let expected = keys.programs.get(program);
        let key = self.server.setup(expected.elf.clone()).await.map_err(classify_client_error)?;
        let actual = S::vk_hash(&key);
        if actual != expected.vk_hash {
            return Err(Failure::ProverFailed(anyhow!(
                "CUDA device {} set up a {} verifying key {actual:?} that differs from the \
                 prestate's {:?}",
                self.id,
                program.name(),
                expected.vk_hash
            )));
        }
        Ok(key)
    }
}

/// Free devices; a lease takes one out and returns it on drop.
struct DevicePool<S: DeviceServer> {
    free: Mutex<Vec<Device<S>>>,
    permits: Semaphore,
}

impl<S: DeviceServer> DevicePool<S> {
    async fn lease(&self) -> DeviceLease<'_, S> {
        let permit = self.permits.acquire().await.expect("the device semaphore is never closed");
        let device = self.free.lock().pop().expect("a permit guarantees a free device");
        DeviceLease { pool: self, device: Some(device), permit: Some(permit) }
    }
}

struct DeviceLease<'a, S: DeviceServer> {
    pool: &'a DevicePool<S>,
    device: Option<Device<S>>,
    permit: Option<SemaphorePermit<'a>>,
}

impl<S: DeviceServer> DeviceLease<'_, S> {
    /// Keeps the device out of the pool for good, since its socket may hold a stale response.
    fn retire(mut self) {
        self.device = None;
        if let Some(permit) = self.permit.take() {
            permit.forget();
        }
    }
}

impl<S: DeviceServer> std::ops::Deref for DeviceLease<'_, S> {
    type Target = Device<S>;

    fn deref(&self) -> &Device<S> {
        self.device.as_ref().expect("present until drop")
    }
}

impl<S: DeviceServer> std::ops::DerefMut for DeviceLease<'_, S> {
    fn deref_mut(&mut self) -> &mut Device<S> {
        self.device.as_mut().expect("present until drop")
    }
}

impl<S: DeviceServer> Drop for DeviceLease<'_, S> {
    fn drop(&mut self) {
        if let Some(device) = self.device.take() {
            self.pool.free.lock().push(device);
        }
    }
}

#[derive(Clone)]
struct Finished {
    outcome: Result<Arc<SP1ProofWithPublicValues>, ProofTerminalState>,
    at: Instant,
}

struct Core<S: DeviceServer> {
    executor: Box<dyn PreExecutor>,
    device_ids: Vec<u32>,
    devices: DevicePool<S>,
    /// One permit per device, so pre-execution keeps pace with proving without holding a device.
    pre_execution: Semaphore,
    config: ProofProviderConfig,
    /// Outcomes by request; `None` while the request runs.
    outcomes: Mutex<HashMap<ProofId, watch::Receiver<Option<Finished>>>>,
    fatal: watch::Sender<Option<String>>,
}

impl<S: DeviceServer> Core<S> {
    fn new(
        executor: Box<dyn PreExecutor>,
        devices: Vec<Device<S>>,
        config: ProofProviderConfig,
    ) -> Arc<Self> {
        let device_ids = devices.iter().map(|device| device.id).collect();
        Arc::new(Self {
            executor,
            pre_execution: Semaphore::new(devices.len()),
            devices: DevicePool {
                permits: Semaphore::new(devices.len()),
                free: Mutex::new(devices),
            },
            device_ids,
            config,
            outcomes: Mutex::new(HashMap::new()),
            fatal: watch::Sender::new(None),
        })
    }

    const fn timeout(&self) -> Duration {
        Duration::from_secs(self.config.timeout)
    }

    fn request(self: &Arc<Self>, program: Program, keys: PrestateKeys, stdin: SP1Stdin) -> ProofId {
        self.remove_expired();
        let proof_id = ProofId::random();
        let (sender, receiver) = watch::channel(None);
        self.outcomes.lock().insert(proof_id, receiver);
        let core = Arc::clone(self);
        tokio::spawn(async move {
            let result = AssertUnwindSafe(core.prove(program, &keys, stdin))
                .catch_unwind()
                .await
                .unwrap_or_else(|_| {
                    Err(Failure::ProverFailed(anyhow!("a local proving task panicked")))
                });
            let outcome = core.finish(proof_id, program, result);
            sender.send_replace(Some(Finished { outcome, at: Instant::now() }));
        });
        tracing::info!(%proof_id, program = program.name(), "Local proof request started");
        proof_id
    }

    /// Removes finished outcomes that no one waited for within the proof timeout.
    fn remove_expired(&self) {
        let timeout = self.timeout();
        self.outcomes.lock().retain(|_, receiver| {
            receiver.borrow().as_ref().is_none_or(|finished| finished.at.elapsed() <= timeout)
        });
    }

    async fn prove(
        &self,
        program: Program,
        keys: &PrestateKeys,
        stdin: SP1Stdin,
    ) -> Result<SP1ProofWithPublicValues, Failure> {
        let stdin_bytes = stdin_encoded_len(&stdin).map_err(Failure::ProverFailed)?;
        if !fits_server_frame(stdin_bytes) {
            return Err(Failure::ProverFailed(anyhow!(
                "the {stdin_bytes}-byte stdin exceeds the GPU server's request size limit"
            )));
        }
        let elf = keys.programs.get(program).elf.clone();
        let cycle_limit = program.cycle_limit(&self.config);
        {
            let _permit = self
                .pre_execution
                .acquire()
                .await
                .expect("the pre-execution semaphore is never closed");
            let mut attempt = 1;
            let report = loop {
                match self.executor.execute(elf.clone(), stdin.clone(), cycle_limit).await {
                    Err(err)
                        if is_memory_monitor_kill(&err) && attempt < MEMORY_MONITOR_ATTEMPTS =>
                    {
                        tracing::warn!(attempt, error = %err, "Pre-execution killed by the memory monitor; running it again");
                        attempt += 1;
                    }
                    result => break result.map_err(classify_execution_error)?,
                }
            };
            if report.exit_code != 0 {
                return Err(Failure::Unexecutable(anyhow!(
                    "the guest exited with code {}",
                    report.exit_code
                )));
            }
        }
        let mut device = self.devices.lease().await;
        let id = device.id;
        let result = AssertUnwindSafe(Self::prove_on(&mut device, program, keys, stdin))
            .catch_unwind()
            .await;
        result.unwrap_or_else(|_| {
            device.retire();
            Err(Failure::Fatal(anyhow!("a local proving task panicked on CUDA device {id}")))
        })
    }

    async fn prove_on(
        device: &mut Device<S>,
        program: Program,
        keys: &PrestateKeys,
        stdin: SP1Stdin,
    ) -> Result<SP1ProofWithPublicValues, Failure> {
        device.ensure_keys(keys).await?;
        let key = device.keys[&keys.prestate].get(program);
        tracing::info!(device = device.id, program = program.name(), "Proving on CUDA device");
        device.server.prove(key, stdin, program.mode()).await.map_err(classify_client_error)
    }

    fn finish(
        &self,
        proof_id: ProofId,
        program: Program,
        result: Result<SP1ProofWithPublicValues, Failure>,
    ) -> Result<Arc<SP1ProofWithPublicValues>, ProofTerminalState> {
        let program = program.name();
        match result {
            Ok(proof) => {
                tracing::info!(%proof_id, program, "Local proof finished");
                Ok(Arc::new(proof))
            }
            Err(Failure::Unexecutable(err)) => {
                tracing::warn!(%proof_id, program, error = format!("{err:#}"), "Local proof request is unexecutable");
                Err(ProofTerminalState::Unexecutable)
            }
            Err(Failure::ProverFailed(err)) => {
                tracing::error!(%proof_id, program, error = format!("{err:#}"), "Local prover failed");
                Err(ProofTerminalState::ProverFailed)
            }
            Err(Failure::Fatal(err)) => {
                let message = format!("local CUDA prover cannot continue: {err:#}");
                tracing::error!(%proof_id, program, error = %message, "Local prover failed");
                self.fatal.send_if_modified(|fatal| {
                    let first = fatal.is_none();
                    if first {
                        *fatal = Some(message);
                    }
                    first
                });
                Err(ProofTerminalState::ProverFailed)
            }
        }
    }

    /// Awaits a request's outcome for up to the proof timeout. On timeout the request keeps
    /// running and a later wait resumes it.
    async fn wait(&self, proof_id: ProofId) -> Result<SP1ProofWithPublicValues, ProofWaitError> {
        let terminal = |state| ProofWaitError::Terminal {
            proof_id,
            state,
            fulfillment_status: 0,
            execution_status: 0,
        };
        let receiver = self.outcomes.lock().get(&proof_id).cloned();
        let Some(mut receiver) = receiver else {
            tracing::warn!(%proof_id, "Unknown local proof request");
            return Err(terminal(ProofTerminalState::ProverFailed));
        };
        let outcome = match tokio::time::timeout(self.timeout(), receiver.wait_for(Option::is_some))
            .await
        {
            Ok(Ok(finished)) => finished.as_ref().expect("waited for an outcome").outcome.clone(),
            // The runtime dropped the task.
            Ok(Err(_)) => Err(ProofTerminalState::ProverFailed),
            Err(_) => {
                tracing::warn!(%proof_id, timeout_secs = self.config.timeout, "proving timeout exceeded");
                ProposerGauge::ProvingTimeoutError.increment(1.0);
                return Err(ProofWaitError::Uncertain(anyhow!(
                    "local proof {proof_id} still running after {}s; a later wait resumes it \
                     (consider increasing {})",
                    self.config.timeout,
                    crate::env_var("SP1_TIMEOUT_SECONDS")
                )));
            }
        };
        self.outcomes.lock().remove(&proof_id);
        outcome.map(|proof| (*proof).clone()).map_err(terminal)
    }

    async fn fatal(&self) -> anyhow::Error {
        let mut receiver = self.fatal.subscribe();
        let message = receiver.wait_for(Option::is_some).await.expect("self owns the sender");
        anyhow!("{}", message.as_deref().expect("waited for a message"))
    }
}

#[cfg(test)]
mod tests {
    use std::{
        io,
        os::unix::fs::PermissionsExt,
        sync::atomic::{AtomicBool, AtomicUsize, Ordering},
    };

    use sp1_sdk::{SP1Proof, SP1PublicValues, network::FulfillmentStrategy};

    use super::*;

    /// Running and peak counts of one scripted operation, with a gate that holds it.
    struct Probe {
        calls: AtomicUsize,
        active: AtomicUsize,
        peak: AtomicUsize,
        gate: Semaphore,
    }

    impl Probe {
        fn open() -> Self {
            Self::gated(Semaphore::MAX_PERMITS)
        }

        fn gated(permits: usize) -> Self {
            Self {
                calls: AtomicUsize::new(0),
                active: AtomicUsize::new(0),
                peak: AtomicUsize::new(0),
                gate: Semaphore::new(permits),
            }
        }

        /// Counts the call as active for one second of test time and until the gate opens.
        async fn run(&self) {
            self.calls.fetch_add(1, Ordering::SeqCst);
            let active = self.active.fetch_add(1, Ordering::SeqCst) + 1;
            self.peak.fetch_max(active, Ordering::SeqCst);
            tokio::time::sleep(Duration::from_secs(1)).await;
            self.gate.acquire().await.unwrap().forget();
            self.active.fetch_sub(1, Ordering::SeqCst);
        }

        fn calls(&self) -> usize {
            self.calls.load(Ordering::SeqCst)
        }

        fn peak(&self) -> usize {
            self.peak.load(Ordering::SeqCst)
        }
    }

    /// The result of the call with the given zero-based index.
    type ExecuteResult = fn(usize) -> Result<ExecutionReport>;

    struct ScriptedExecutor {
        probe: Probe,
        result: ExecuteResult,
    }

    #[async_trait]
    impl PreExecutor for Arc<ScriptedExecutor> {
        async fn execute(
            &self,
            _elf: Elf,
            _stdin: SP1Stdin,
            _cycle_limit: u64,
        ) -> Result<ExecutionReport> {
            let call = self.probe.calls();
            self.probe.run().await;
            (self.result)(call)
        }
    }

    /// Answers for every device of one test.
    struct ServerScript {
        prove: Probe,
        /// The device and verifying-key hash of each setup.
        setups: Mutex<Vec<(u32, [u32; 8])>>,
        wrong_vk: AtomicBool,
        prove_error: Option<fn() -> CudaClientError>,
    }

    struct ScriptedServer {
        id: u32,
        script: Arc<ServerScript>,
    }

    #[async_trait]
    impl DeviceServer for ScriptedServer {
        type Key = [u32; 8];

        async fn setup(&self, elf: Elf) -> Result<[u32; 8], CudaClientError> {
            let hash =
                if self.script.wrong_vk.load(Ordering::SeqCst) { [0; 8] } else { vk_hash(&elf) };
            self.script.setups.lock().push((self.id, hash));
            Ok(hash)
        }

        fn vk_hash(key: &[u32; 8]) -> [u32; 8] {
            *key
        }

        async fn prove(
            &self,
            _key: &[u32; 8],
            _stdin: SP1Stdin,
            _mode: SP1ProofMode,
        ) -> Result<SP1ProofWithPublicValues, CudaClientError> {
            self.script.prove.run().await;
            self.script.prove_error.map_or_else(|| Ok(test_proof()), |error| Err(error()))
        }
    }

    fn vk_hash(elf: &[u8]) -> [u32; 8] {
        [u32::from(elf[0]); 8]
    }

    fn prestate_keys() -> PrestateKeys {
        let program =
            |elf: &'static [u8]| ProgramKey { elf: Elf::Static(elf), vk_hash: vk_hash(elf) };
        PrestateKeys {
            prestate: B256::repeat_byte(0xaa),
            programs: ByProgram {
                range: program(b"r"),
                log_commitment: program(b"l"),
                aggregation: program(b"a"),
            },
        }
    }

    fn test_proof() -> SP1ProofWithPublicValues {
        SP1ProofWithPublicValues {
            proof: SP1Proof::Core(Vec::new()),
            public_values: SP1PublicValues::new(),
            sp1_version: String::new(),
            tee_proof: None,
        }
    }

    fn config() -> ProofProviderConfig {
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
            log_proof_strategy: FulfillmentStrategy::Auction,
            log_cycle_limit: 1,
            log_gas_limit: 1,
            max_price_per_pgu: None,
            min_auction_period: 1,
        }
    }

    struct Harness {
        core: Arc<Core<ScriptedServer>>,
        executor: Arc<ScriptedExecutor>,
        script: Arc<ServerScript>,
    }

    impl Harness {
        fn new(devices: u32) -> Self {
            Self::with(
                devices,
                Probe::open(),
                Probe::open(),
                |_| Ok(ExecutionReport::default()),
                None,
            )
        }

        fn with(
            devices: u32,
            execute: Probe,
            prove: Probe,
            result: ExecuteResult,
            prove_error: Option<fn() -> CudaClientError>,
        ) -> Self {
            let executor = Arc::new(ScriptedExecutor { probe: execute, result });
            let script = Arc::new(ServerScript {
                prove,
                setups: Mutex::new(Vec::new()),
                wrong_vk: AtomicBool::new(false),
                prove_error,
            });
            let devices = (0..devices)
                .map(|id| Device {
                    id,
                    server: ScriptedServer { id, script: Arc::clone(&script) },
                    keys: HashMap::new(),
                })
                .collect();
            let core = Core::new(Box::new(Arc::clone(&executor)), devices, config());
            Self { core, executor, script }
        }

        fn request(&self) -> ProofId {
            self.core.request(Program::Range, prestate_keys(), SP1Stdin::new())
        }

        async fn finished(&self, proof_id: ProofId) {
            let mut receiver = self.core.outcomes.lock()[&proof_id].clone();
            receiver.wait_for(Option::is_some).await.unwrap();
        }

        /// The terminal state of a request whose pre-execution returns `failure`.
        async fn terminal_state(failure: ExecuteResult) -> ProofTerminalState {
            let harness = Self::with(1, Probe::open(), Probe::open(), failure, None);
            let proof_id = harness.request();
            let err = harness.core.wait(proof_id).await.unwrap_err();
            assert_eq!(harness.script.prove.calls(), 0);
            let ProofWaitError::Terminal { state, .. } = err else { panic!("{err:?}") };
            state
        }
    }

    fn execution_error(err: ExecutionError) -> anyhow::Error {
        anyhow::Error::new(err).context("execute")
    }

    #[tokio::test(start_paused = true)]
    async fn request_then_wait_returns_proof() {
        let harness = Harness::new(1);
        let proof_id = harness.request();

        harness.core.wait(proof_id).await.unwrap();

        assert!(harness.core.outcomes.lock().is_empty());
        assert_eq!(harness.script.prove.calls(), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn wait_again_after_timeout_resumes_same_task() {
        let harness = Harness::with(
            1,
            Probe::open(),
            Probe::gated(0),
            |_| Ok(ExecutionReport::default()),
            None,
        );
        let proof_id = harness.request();

        let err = harness.core.wait(proof_id).await.unwrap_err();
        assert!(matches!(err, ProofWaitError::Uncertain(_)), "{err:?}");

        harness.script.prove.gate.add_permits(1);
        harness.core.wait(proof_id).await.unwrap();
        assert_eq!(harness.script.prove.calls(), 1);
    }

    #[tokio::test]
    async fn unknown_proof_id_is_retryable_terminal() {
        let harness = Harness::new(1);

        let err = harness.core.wait(B256::repeat_byte(1)).await.unwrap_err();

        assert!(
            matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::ProverFailed, .. }),
            "{err:?}"
        );
    }

    #[tokio::test(start_paused = true)]
    async fn cycle_limit_breach_is_unexecutable() {
        let state = Harness::terminal_state(|_| {
            Err(execution_error(ExecutionError::ExceededCycleLimit(1)))
        })
        .await;
        assert_eq!(state, ProofTerminalState::Unexecutable);
    }

    #[tokio::test(start_paused = true)]
    async fn memory_limit_breach_is_unexecutable() {
        let state =
            Harness::terminal_state(|_| Err(execution_error(ExecutionError::TooMuchMemory())))
                .await;
        assert_eq!(state, ProofTerminalState::Unexecutable);
    }

    #[tokio::test(start_paused = true)]
    async fn memory_monitor_kill_is_unexecutable() {
        let state = Harness::terminal_state(|_| {
            Err(execution_error(ExecutionError::KilledByMemoryMonitor(1)))
        })
        .await;
        assert_eq!(state, ProofTerminalState::Unexecutable);
    }

    #[tokio::test(start_paused = true)]
    async fn memory_monitor_kill_runs_again_before_unexecutable() {
        let kill = |_| Err(execution_error(ExecutionError::KilledByMemoryMonitor(1)));
        let harness = Harness::with(1, Probe::open(), Probe::open(), kill, None);
        let proof_id = harness.request();
        harness.core.wait(proof_id).await.unwrap_err();
        assert_eq!(harness.executor.probe.calls(), MEMORY_MONITOR_ATTEMPTS);

        let first_kill = |call| match call {
            0 => Err(execution_error(ExecutionError::KilledByMemoryMonitor(1))),
            _ => Ok(ExecutionReport::default()),
        };
        let harness = Harness::with(1, Probe::open(), Probe::open(), first_kill, None);
        let proof_id = harness.request();
        harness.core.wait(proof_id).await.unwrap();
        assert_eq!(harness.executor.probe.calls(), 2);
    }

    #[tokio::test(start_paused = true)]
    async fn deterministic_guest_fault_is_unexecutable() {
        let state = Harness::terminal_state(|_| {
            Err(execution_error(ExecutionError::UnsupportedSyscall(0)))
        })
        .await;
        assert_eq!(state, ProofTerminalState::Unexecutable);
    }

    #[tokio::test(start_paused = true)]
    async fn other_execution_error_is_prover_failed() {
        let state = Harness::terminal_state(|_| {
            Err(execution_error(ExecutionError::Other("child terminated abnormally".into())))
        })
        .await;
        assert_eq!(state, ProofTerminalState::ProverFailed);
        let state = Harness::terminal_state(|_| Err(anyhow!("deferred proof failed"))).await;
        assert_eq!(state, ProofTerminalState::ProverFailed);
    }

    #[test]
    fn server_frame_holds_u32_length() {
        assert!(fits_server_frame(u64::from(u32::MAX) - 1024));
        assert!(!fits_server_frame(u64::from(u32::MAX) - 1023));
        assert!(!fits_server_frame(u64::MAX));
    }

    #[tokio::test(start_paused = true)]
    async fn prove_panic_resolves_fatal() {
        let harness = Harness::with(
            1,
            Probe::open(),
            Probe::open(),
            |_| Ok(ExecutionReport::default()),
            Some(|| panic!("scripted server panic")),
        );
        let proof_id = harness.request();

        let err = harness.core.wait(proof_id).await.unwrap_err();

        assert!(
            matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::ProverFailed, .. }),
            "{err:?}"
        );
        let fatal = harness.core.fatal().await.to_string();
        assert!(fatal.contains("panicked"), "{fatal}");
        assert!(harness.core.devices.free.lock().is_empty());
    }

    #[tokio::test(start_paused = true)]
    async fn pre_execution_panic_is_prover_failed() {
        let harness = Harness::with(
            1,
            Probe::open(),
            Probe::open(),
            |_| panic!("scripted executor panic"),
            None,
        );
        let proof_id = harness.request();

        let err = harness.core.wait(proof_id).await.unwrap_err();

        assert!(
            matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::ProverFailed, .. }),
            "{err:?}"
        );
        assert!(harness.core.fatal.borrow().is_none());
        assert_eq!(harness.core.devices.free.lock().len(), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn nonzero_exit_code_is_unexecutable() {
        let state = Harness::terminal_state(|_| {
            let mut report = ExecutionReport::default();
            report.exit_code = 1;
            Ok(report)
        })
        .await;
        assert_eq!(state, ProofTerminalState::Unexecutable);
    }

    #[tokio::test(start_paused = true)]
    async fn child_killed_is_prover_failed() {
        let state =
            Harness::terminal_state(|_| Err(execution_error(ExecutionError::ChildKilled()))).await;
        assert_eq!(state, ProofTerminalState::ProverFailed);
    }

    #[tokio::test(start_paused = true)]
    async fn server_prover_error_is_prover_failed() {
        let harness = Harness::with(
            1,
            Probe::open(),
            Probe::open(),
            |_| Ok(ExecutionReport::default()),
            Some(|| CudaClientError::ServerError("out of memory".into())),
        );
        let proof_id = harness.request();

        let err = harness.core.wait(proof_id).await.unwrap_err();

        assert!(
            matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::ProverFailed, .. }),
            "{err:?}"
        );
        assert!(harness.core.fatal.borrow().is_none());
    }

    #[tokio::test(start_paused = true)]
    async fn server_io_failure_resolves_fatal() {
        let harness = Harness::with(
            1,
            Probe::open(),
            Probe::open(),
            |_| Ok(ExecutionReport::default()),
            Some(|| CudaClientError::Read(io::ErrorKind::BrokenPipe.into())),
        );
        let proof_id = harness.request();

        let err = harness.core.wait(proof_id).await.unwrap_err();

        assert!(
            matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::ProverFailed, .. }),
            "{err:?}"
        );
        let fatal = harness.core.fatal().await.to_string();
        assert!(fatal.contains("Failed to read the response"), "{fatal}");
    }

    #[tokio::test(start_paused = true)]
    async fn pre_execution_holds_no_device_lease() {
        let harness = Harness::with(
            1,
            Probe::open(),
            Probe::gated(0),
            |_| Ok(ExecutionReport::default()),
            None,
        );
        let first = harness.request();
        while harness.script.prove.calls() == 0 {
            tokio::time::sleep(Duration::from_millis(100)).await;
        }

        let second = harness.request();
        let pre_executed = async {
            while harness.executor.probe.calls() < 2 ||
                harness.executor.probe.active.load(Ordering::SeqCst) > 0
            {
                tokio::time::sleep(Duration::from_millis(100)).await;
            }
        };
        tokio::time::timeout(Duration::from_secs(30), pre_executed)
            .await
            .expect("the second request pre-executes while the first proves");
        assert_eq!(harness.script.prove.calls(), 1);
        harness.script.prove.gate.add_permits(2);
        harness.core.wait(first).await.unwrap();
        harness.core.wait(second).await.unwrap();
    }

    #[tokio::test(start_paused = true)]
    async fn pre_execution_bounded_by_device_count() {
        let harness = Harness::new(2);
        let proof_ids: Vec<_> = (0..5).map(|_| harness.request()).collect();

        for proof_id in proof_ids {
            harness.core.wait(proof_id).await.unwrap();
        }

        assert_eq!(harness.executor.probe.calls(), 5);
        assert_eq!(harness.executor.probe.peak(), 2);
    }

    #[tokio::test(start_paused = true)]
    async fn requests_never_exceed_device_count() {
        let harness = Harness::new(2);
        let proof_ids: Vec<_> = (0..5).map(|_| harness.request()).collect();

        for proof_id in proof_ids {
            harness.core.wait(proof_id).await.unwrap();
        }

        assert_eq!(harness.script.prove.calls(), 5);
        assert_eq!(harness.script.prove.peak(), 2);
    }

    #[tokio::test]
    async fn vk_mismatch_fails_with_program_name() {
        let harness = Harness::new(1);
        harness.script.wrong_vk.store(true, Ordering::SeqCst);
        let mut device = harness.core.devices.lease().await;

        let Err(Failure::ProverFailed(err)) = device.ensure_keys(&prestate_keys()).await else {
            panic!("expected a prover failure");
        };

        assert!(err.to_string().contains("super-range"), "{err}");
        assert!(device.keys.is_empty());
    }

    #[tokio::test(start_paused = true)]
    async fn keys_set_up_once_per_device_and_prestate() {
        let harness = Harness::new(1);

        for _ in 0..2 {
            let proof_id = harness.request();
            harness.core.wait(proof_id).await.unwrap();
        }

        let keys = prestate_keys();
        let programs = &keys.programs;
        let expected = [
            (0, programs.range.vk_hash),
            (0, programs.log_commitment.vk_hash),
            (0, programs.aggregation.vk_hash),
        ];
        assert_eq!(*harness.script.setups.lock(), expected);
    }

    #[tokio::test(start_paused = true)]
    async fn dropped_waiter_does_not_cancel_proving() {
        let harness = Harness::with(
            1,
            Probe::open(),
            Probe::gated(0),
            |_| Ok(ExecutionReport::default()),
            None,
        );
        let proof_id = harness.request();
        while harness.script.prove.calls() == 0 {
            tokio::time::sleep(Duration::from_millis(100)).await;
        }

        let wait = harness.core.wait(proof_id);
        tokio::time::timeout(Duration::from_secs(1), wait).await.unwrap_err();

        harness.script.prove.gate.add_permits(1);
        harness.core.wait(proof_id).await.unwrap();
        assert_eq!(harness.script.prove.calls(), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn unwaited_finished_entry_expires() {
        let harness = Harness::new(1);
        let proof_id = harness.request();
        harness.finished(proof_id).await;

        tokio::time::advance(harness.core.timeout()).await;
        let kept = harness.request();
        assert!(harness.core.outcomes.lock().contains_key(&proof_id));

        tokio::time::advance(Duration::from_secs(1)).await;
        harness.request();

        let outcomes = harness.core.outcomes.lock();
        assert!(!outcomes.contains_key(&proof_id));
        assert!(outcomes.contains_key(&kept));
    }

    #[tokio::test]
    async fn startup_rejects_missing_or_mismatched_gpu_server() {
        let err = check_gpu_server(None).await.unwrap_err().to_string();
        assert!(err.contains(CUDA_IMAGE), "{err}");

        let home = tempfile::tempdir().unwrap();
        let err = check_gpu_server(Some(home.path().into())).await.unwrap_err().to_string();
        assert!(err.contains(CUDA_IMAGE), "{err}");

        let bin = home.path().join(".sp1/bin");
        std::fs::create_dir_all(&bin).unwrap();
        let server = bin.join("sp1-gpu-server");
        for (version, ok) in [("6.9.0", false), (SP1_CRATE_VERSION, true)] {
            std::fs::write(&server, format!("#!/bin/sh\necho {version}\n")).unwrap();
            std::fs::set_permissions(&server, std::fs::Permissions::from_mode(0o755)).unwrap();
            let result = check_gpu_server(Some(home.path().into())).await;
            match result {
                Ok(()) => assert!(ok, "{version} accepted"),
                Err(err) => {
                    assert!(!ok, "{version}: {err}");
                    assert!(err.to_string().contains(CUDA_IMAGE), "{err}");
                }
            }
        }
    }

    #[test]
    fn classify_client_error_texts() {
        let bincode_error = || Box::new(bincode1::ErrorKind::Custom("bad".into()));
        let io_error = || io::Error::from(io::ErrorKind::BrokenPipe);
        let cases: [(CudaClientError, bool); 10] = [
            (CudaClientError::Connect(io_error()), false),
            (CudaClientError::Serialize(bincode_error()), false),
            (CudaClientError::Deserialize(bincode_error()), false),
            (CudaClientError::Write(io_error()), false),
            (CudaClientError::Read(io_error()), false),
            (CudaClientError::ServerError("cuda".into()), true),
            (CudaClientError::UnexpectedResponse("Ok"), false),
            (CudaClientError::ProverError("prover".into()), true),
            // `Download` takes a reqwest 0.12 error and has the same text.
            (CudaClientError::DownloadIO(io_error()), false),
            (CudaClientError::Unexpected("other".into()), false),
        ];
        for (err, retryable) in cases {
            let text = err.to_string();
            match classify_client_error(err) {
                Failure::ProverFailed(_) => assert!(retryable, "{text}"),
                Failure::Fatal(_) => assert!(!retryable, "{text}"),
                Failure::Unexecutable(_) => panic!("{text}"),
            }
        }
    }
}
