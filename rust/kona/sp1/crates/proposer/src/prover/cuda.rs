//! Local SP1 proving on NVIDIA GPUs, through one `sp1-gpu-server` child per CUDA device.
//!
//! The proposer starts each server and speaks its socket protocol (`sp1_cuda::api`) itself, so
//! the SP1 SDK's CUDA client, which downloads a missing server, never runs. Each request runs in
//! a spawned task that is never aborted, so it always reads its response (see [`Connection`]).

use std::{
    collections::HashMap, io, panic::AssertUnwindSafe, path::Path, process::Stdio, sync::Arc,
    time::Duration,
};

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow, bail, ensure};
use async_trait::async_trait;
use futures::FutureExt;
use kona_sp1_host_utils::metrics::MetricsGauge;
use parking_lot::Mutex;
use sp1_core_executor::{ExecutionError, ExecutionReport, SP1ContextBuilder, SP1CoreOpts};
use sp1_cuda::{
    api::{Request, Response},
    client::socket_path,
};
use sp1_primitives::SP1_CRATE_VERSION;
use sp1_prover_types::{SerializableRiscvMachine, network_base_types::ProofMode};
use sp1_sdk::{
    Elf, HashableKey, LightProver, Prover, ProvingKey, RiscvAir, SP1Proof,
    SP1ProofWithPublicValues, SP1Stdin, SP1VerifyingKey,
};
use thiserror::Error;
use tokio::{
    io::{AsyncReadExt, AsyncWriteExt},
    net::UnixStream,
    process::{Child, Command},
    sync::{OwnedSemaphorePermit, Semaphore, SemaphorePermit, watch},
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

/// How long a started server may take to open its socket; it initializes CUDA first.
const SERVER_START_TIMEOUT: Duration = Duration::from_secs(60);

/// Local proving on CUDA devices, one proof per device at a time. Clones share one state.
#[derive(Clone)]
pub struct LocalCudaProofProvider {
    core: Arc<Core<GpuServer>>,
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
    /// Checks the GPU server binary and starts one server per configured device. Each device may
    /// hold [`QUEUED_STDINS_PER_DEVICE`] stdins of up to `max_stdin_bytes`.
    pub async fn new(
        config: &LocalCudaConfig,
        provider_config: ProofProviderConfig,
        max_stdin_bytes: u64,
    ) -> Result<Self> {
        let path = config.server_path.as_path();
        check_gpu_server(path).await?;
        ensure!(
            cfg!(target_arch = "x86_64"),
            "local CUDA proving needs an x86-64 host; run the {CUDA_IMAGE} image"
        );
        let opts =
            SP1CoreOpts { memory_limit: config.memory_limit_bytes.get(), ..Default::default() };
        let verifier = LightProver::new_with_opts(opts).await;
        let memory_limit = config.memory_limit_bytes.get();
        let mut devices = Vec::with_capacity(config.devices.len());
        for &id in &config.devices {
            let server = GpuServer::start(path, id, memory_limit).await.with_context(|| {
                format!("start sp1-gpu-server for CUDA device {id}; check the device and the {CUDA_IMAGE} image")
            })?;
            tracing::info!(device = id, "Started sp1-gpu-server");
            devices.push(Device { id, server, keys: HashMap::new() });
        }
        let stdin_kib = max_stdin_bytes.div_ceil(1024);
        let queued_stdin_bytes = QUEUED_STDINS_PER_DEVICE * devices.len() as u64 * stdin_kib * 1024;
        let core =
            Core::new(Box::new(verifier.clone()), devices, provider_config, queued_stdin_bytes);
        Ok(Self { core, verifier })
    }

    /// Resolves with the first failure that leaves a GPU server unusable. The caller must return
    /// from `main`: dropping the runtime kills every server, and outside Linux
    /// `std::process::exit` does not.
    pub async fn fatal(&self) -> anyhow::Error {
        self.core.fatal().await
    }

    pub(super) async fn request_range_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> ProofId {
        self.core.request(Program::Range, PrestateKeys::from(keys), stdin).await
    }

    pub(super) async fn request_log_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> ProofId {
        self.core.request(Program::LogCommitment, PrestateKeys::from(keys), stdin).await
    }

    pub(super) async fn request_aggregation_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> ProofId {
        self.core.request(Program::Aggregation, PrestateKeys::from(keys), stdin).await
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

/// Checks that the server binary exists and reports the linked SP1 version, whose protocol types
/// the proposer speaks.
async fn check_gpu_server(path: &Path) -> Result<()> {
    ensure!(path.is_file(), "{} does not exist; run the {CUDA_IMAGE} image", path.display());
    let output = server_command(path).arg("--version").output().await.with_context(|| {
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

/// A command for the server binary, without the `OP_ZK_PROPOSER_*` variables that hold the
/// proposer's keys.
fn server_command(path: &Path) -> Command {
    let mut command = Command::new(path);
    for (name, _) in std::env::vars_os() {
        if name.to_string_lossy().starts_with(crate::ENV_VAR_PREFIX) {
            command.env_remove(name);
        }
    }
    command
}

/// One device's `sp1-gpu-server`.
#[async_trait]
trait DeviceServer: Send + Sync + 'static {
    type Key: Send + Sync + 'static;

    async fn setup(&self, elf: Elf) -> Result<Self::Key, ServerError>;

    fn vk_hash(key: &Self::Key) -> [u32; 8];

    async fn prove(
        &self,
        key: &Self::Key,
        stdin: SP1Stdin,
        mode: ProofMode,
    ) -> Result<SP1ProofWithPublicValues, ServerError>;
}

/// Why a server request failed.
#[derive(Debug, Error)]
enum ServerError {
    /// The server answered with an internal or prover error.
    #[error("the server rejected the request: {0}")]
    Rejected(String),
    /// The request does not fit the protocol's `u32` length prefix; nothing was sent.
    #[error("the {0}-byte request exceeds the server's frame size")]
    TooLarge(usize),
    #[error("encode the request: {0}")]
    Encode(bincode1::Error),
    #[error("decode the response: {0}")]
    Decode(bincode1::Error),
    #[error("server connection failed")]
    Io(#[from] io::Error),
    #[error("unexpected {0} response")]
    Unexpected(&'static str),
}

impl ServerError {
    /// A failure before the request was sent, or a rejection, leaves the socket in step. Any other
    /// failure is fatal: I/O may leave it out of step (see [`Connection`]), and an undecodable or
    /// unexpected response means the server speaks another protocol.
    fn into_failure(self) -> Failure {
        match self {
            Self::Rejected(_) | Self::TooLarge(_) | Self::Encode(_) => {
                Failure::ProverFailed(self.into())
            }
            Self::Decode(_) | Self::Io(_) | Self::Unexpected(_) => Failure::Fatal(self.into()),
        }
    }
}

const fn response_name(response: &Response) -> &'static str {
    match response {
        Response::Ok => "Ok",
        Response::Setup { .. } => "Setup",
        Response::Proof { .. } => "Proof",
        Response::ProverError(_) => "ProverError",
        Response::InternalError(_) => "InternalError",
        Response::ConnectionClosed => "ConnectionClosed",
    }
}

/// A server's socket, framed as `sp1_cuda::client` frames it: a little-endian `u32` length, then
/// the bincode 1 encoding. A request whose response is not read leaves the socket out of step:
/// the next request on it would read that response.
struct Connection {
    stream: tokio::sync::Mutex<UnixStream>,
}

impl Connection {
    /// Sends `request` and reads its response. The request and its encoding are dropped once
    /// sent, so a long proof does not hold copies of its stdin.
    async fn call(&self, request: Request) -> Result<Response, ServerError> {
        let bytes = bincode1::serialize(&request).map_err(ServerError::Encode)?;
        drop(request);
        let len = u32::try_from(bytes.len()).map_err(|_| ServerError::TooLarge(bytes.len()))?;
        let mut stream = self.stream.lock().await;
        stream.write_all(&len.to_le_bytes()).await?;
        stream.write_all(&bytes).await?;
        drop(bytes);
        let mut len = [0; 4];
        stream.read_exact(&mut len).await?;
        let mut response = vec![0; u32::from_le_bytes(len) as usize];
        stream.read_exact(&mut response).await?;
        match bincode1::deserialize(&response).map_err(ServerError::Decode)? {
            Response::InternalError(err) | Response::ProverError(err) => {
                Err(ServerError::Rejected(err))
            }
            response => Ok(response),
        }
    }
}

/// A server key: its ID on the server, and the hash of its verifying key.
struct ServerKey {
    id: [u8; 32],
    vk_hash: [u32; 8],
}

/// A started `sp1-gpu-server` child and its connection. Dropping it kills the child.
struct GpuServer {
    connection: Connection,
    _child: Child,
}

impl GpuServer {
    /// Starts the server for device `id` and connects to its socket. The server's executor gets
    /// the pre-execution memory limit, so both enforce the same one.
    async fn start(path: &Path, id: u32, memory_limit: u64) -> Result<Self> {
        let socket = socket_path(id);
        // A socket file left by another server could take the connection.
        match std::fs::remove_file(&socket) {
            Ok(()) => {}
            Err(err) if err.kind() == io::ErrorKind::NotFound => {}
            Err(err) => return Err(err).with_context(|| format!("remove {}", socket.display())),
        }
        let mut command = server_command(path);
        command
            .env("CUDA_VISIBLE_DEVICES", id.to_string())
            .env("MEMORY_LIMIT", memory_limit.to_string())
            .stdin(Stdio::null())
            .kill_on_drop(true);
        #[cfg(target_os = "linux")]
        {
            let proposer = std::process::id();
            // SAFETY: the hook calls only async-signal-safe functions and does not allocate.
            unsafe { command.pre_exec(move || kill_with_proposer(proposer)) };
        }
        let mut child = command.spawn().with_context(|| format!("run {}", path.display()))?;
        let stream = connect(&socket, &mut child).await?;
        Ok(Self {
            connection: Connection { stream: tokio::sync::Mutex::new(stream) },
            _child: child,
        })
    }
}

/// Makes the kernel kill the server when the proposer dies, since a signal or a crash ends the
/// proposer without running `kill_on_drop`. The kernel tracks the spawning thread, so spawn from
/// a thread that lives as long as the proposer.
#[cfg(target_os = "linux")]
fn kill_with_proposer(proposer: u32) -> io::Result<()> {
    // SAFETY: `prctl` and `getppid` are async-signal-safe.
    if unsafe { libc::prctl(libc::PR_SET_PDEATHSIG, libc::SIGKILL) } != 0 {
        return Err(io::Error::last_os_error());
    }
    // The proposer can die before `prctl` takes effect.
    if unsafe { libc::getppid() } as u32 != proposer {
        return Err(io::Error::from_raw_os_error(libc::ESRCH));
    }
    Ok(())
}

async fn connect(socket: &Path, child: &mut Child) -> Result<UnixStream> {
    let deadline = Instant::now() + SERVER_START_TIMEOUT;
    loop {
        let err = match UnixStream::connect(socket).await {
            Ok(stream) => return Ok(stream),
            Err(err) => err,
        };
        if let Some(status) = child.try_wait()? {
            bail!("the server exited before it opened {}: {status}", socket.display());
        }
        if Instant::now() >= deadline {
            bail!(
                "the server did not open {} within {}s: {err}",
                socket.display(),
                SERVER_START_TIMEOUT.as_secs()
            );
        }
        tokio::time::sleep(Duration::from_millis(100)).await;
    }
}

#[async_trait]
impl DeviceServer for GpuServer {
    type Key = ServerKey;

    async fn setup(&self, elf: Elf) -> Result<ServerKey, ServerError> {
        let request = Request::Setup {
            elf: elf.to_vec(),
            machine: SerializableRiscvMachine::from(RiscvAir::machine()),
        };
        match self.connection.call(request).await? {
            Response::Setup { id, vk } => Ok(ServerKey { id, vk_hash: vk.hash_u32() }),
            response => Err(ServerError::Unexpected(response_name(&response))),
        }
    }

    fn vk_hash(key: &ServerKey) -> [u32; 8] {
        key.vk_hash
    }

    async fn prove(
        &self,
        key: &ServerKey,
        stdin: SP1Stdin,
        mode: ProofMode,
    ) -> Result<SP1ProofWithPublicValues, ServerError> {
        // The SDK's CUDA client sends a zero nonce too.
        let request = Request::ProveWithMode { mode, key: key.id, stdin, proof_nonce: [0; 4] };
        match self.connection.call(request).await? {
            Response::Proof { proof } => {
                let proof = SP1ProofWithPublicValues::from(proof);
                if proof_matches_mode(&proof.proof, mode) {
                    Ok(proof)
                } else {
                    Err(ServerError::Unexpected("wrong proof kind"))
                }
            }
            response => Err(ServerError::Unexpected(response_name(&response))),
        }
    }
}

/// Whether a proof is of the requested kind. A wrong kind would pass verification and then fail
/// the aggregation that consumes it.
const fn proof_matches_mode(proof: &SP1Proof, mode: ProofMode) -> bool {
    matches!(
        (proof, mode),
        (SP1Proof::Core(_), ProofMode::Core) |
            (SP1Proof::Compressed(_), ProofMode::Compressed) |
            (SP1Proof::Plonk(_), ProofMode::Plonk) |
            (SP1Proof::Groth16(_), ProofMode::Groth16)
    )
}

/// Runs a request's program before it reaches a device; the only cycle guard, since the server
/// enforces only its memory limit.
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

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
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

    const fn mode(self) -> ProofMode {
        match self {
            Self::Range | Self::LogCommitment => ProofMode::Compressed,
            Self::Aggregation => ProofMode::Plonk,
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

/// Stdins that one device may hold: one in pre-execution and one that waits for or uses the
/// device. Unlike an SPN upload, a local request keeps its stdin until it finishes.
const QUEUED_STDINS_PER_DEVICE: u64 = 2;

/// Pre-execution attempts while the executor's RSS monitor kills the child. One kill acts on a
/// host-dependent sample, so SP1's own worker retries it too.
const MEMORY_MONITOR_ATTEMPTS: usize = 3;

/// Execution errors are deterministic for the program and input, except an external kill and
/// `Other`, which also covers a failed or abnormally ended child, such as a full `/dev/shm`. Both
/// are retried.
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

/// The encoded length of `stdin`, which must fit the server frame (see [`fits_server_frame`]).
fn frame_checked_len(stdin: &SP1Stdin) -> Result<u64> {
    let bytes = stdin_encoded_len(stdin)?;
    ensure!(
        fits_server_frame(bytes),
        "the {bytes}-byte stdin exceeds the GPU server's request size limit"
    );
    Ok(bytes)
}

/// Whether a stdin of `stdin_bytes` fits [`Connection::call`]'s `u32` frame; checked before
/// admission.
fn fits_server_frame(stdin_bytes: u64) -> bool {
    /// Room for the request's other fields.
    const REQUEST_OVERHEAD: u64 = 1024;
    stdin_bytes.saturating_add(REQUEST_OVERHEAD) <= u64::from(u32::MAX)
}

/// A device's server and the proving keys it holds, by prestate and program. Keys are never
/// evicted.
struct Device<S: DeviceServer> {
    id: u32,
    server: S,
    keys: HashMap<(B256, Program), S::Key>,
}

impl<S: DeviceServer> Device<S> {
    /// Sets up the key of `program` on first use.
    async fn ensure_key(&mut self, program: Program, keys: &PrestateKeys) -> Result<(), Failure> {
        let id = (keys.prestate, program);
        if !self.keys.contains_key(&id) {
            let key = self.setup(program, keys).await?;
            self.keys.insert(id, key);
        }
        Ok(())
    }

    async fn setup(&self, program: Program, keys: &PrestateKeys) -> Result<S::Key, Failure> {
        let expected = keys.programs.get(program);
        let key =
            self.server.setup(expected.elf.clone()).await.map_err(ServerError::into_failure)?;
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
    /// Keeps the device out of the pool for good; its socket may be out of step (see
    /// [`Connection`]).
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
    /// KiB of stdin held by unfinished requests; see [`Core::request`].
    queued_stdin: Arc<Semaphore>,
    queued_stdin_kib: u32,
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
        queued_stdin_bytes: u64,
    ) -> Arc<Self> {
        let queued_stdin_kib = u32::try_from(queued_stdin_bytes / 1024).unwrap_or(u32::MAX).max(1);
        let device_ids = devices.iter().map(|device| device.id).collect();
        Arc::new(Self {
            executor,
            pre_execution: Semaphore::new(devices.len()),
            queued_stdin: Arc::new(Semaphore::new(queued_stdin_kib as usize)),
            queued_stdin_kib,
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

    /// Starts a request once its stdin fits the queued-stdin budget, so a game that submits many
    /// requests waits here, as for an SPN upload. A stdin larger than the budget waits for an
    /// empty queue; one that cannot fit the server frame fails without waiting.
    async fn request(
        self: &Arc<Self>,
        program: Program,
        keys: PrestateKeys,
        stdin: SP1Stdin,
    ) -> ProofId {
        let admission = match frame_checked_len(&stdin) {
            Ok(bytes) => {
                let kib = u32::try_from(bytes.div_ceil(1024)).unwrap_or(u32::MAX);
                Ok(Arc::clone(&self.queued_stdin)
                    .acquire_many_owned(kib.clamp(1, self.queued_stdin_kib))
                    .await
                    .expect("the queued-stdin semaphore is never closed"))
            }
            Err(err) => Err(err),
        };
        self.remove_expired();
        let proof_id = ProofId::random();
        let (sender, receiver) = watch::channel(None);
        self.outcomes.lock().insert(proof_id, receiver);
        let admission = match admission {
            Ok(admission) => admission,
            Err(err) => {
                let outcome = self.finish(proof_id, program, Err(Failure::ProverFailed(err)));
                sender.send_replace(Some(Finished { outcome, at: Instant::now() }));
                return proof_id;
            }
        };
        let core = Arc::clone(self);
        tokio::spawn(async move {
            let _admission: OwnedSemaphorePermit = admission;
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
            .await
            .unwrap_or_else(|_| {
                Err(Failure::Fatal(anyhow!("a local proving task panicked on CUDA device {id}")))
            });
        if matches!(result, Err(Failure::Fatal(_))) {
            device.retire();
        }
        result
    }

    async fn prove_on(
        device: &mut Device<S>,
        program: Program,
        keys: &PrestateKeys,
        stdin: SP1Stdin,
    ) -> Result<SP1ProofWithPublicValues, Failure> {
        device.ensure_key(program, keys).await?;
        let key = &device.keys[&(keys.prestate, program)];
        tracing::info!(device = device.id, program = program.name(), "Proving on CUDA device");
        device.server.prove(key, stdin, program.mode()).await.map_err(ServerError::into_failure)
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
        os::unix::fs::PermissionsExt,
        sync::atomic::{AtomicBool, AtomicU8, AtomicUsize, Ordering},
    };

    use sp1_sdk::{SP1PublicValues, network::FulfillmentStrategy};
    use sp1_verifier::{PlonkBn254Proof, ProofFromNetwork};

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
        /// The first ELF byte of the program whose setup the server rejects; 0 for none.
        rejected_setup: AtomicU8,
        prove_error: Option<fn() -> ServerError>,
    }

    struct ScriptedServer {
        id: u32,
        script: Arc<ServerScript>,
    }

    #[async_trait]
    impl DeviceServer for ScriptedServer {
        type Key = [u32; 8];

        async fn setup(&self, elf: Elf) -> Result<[u32; 8], ServerError> {
            let hash =
                if self.script.wrong_vk.load(Ordering::SeqCst) { [0; 8] } else { vk_hash(&elf) };
            self.script.setups.lock().push((self.id, hash));
            if elf[0] == self.script.rejected_setup.load(Ordering::SeqCst) {
                return Err(ServerError::Rejected("setup".into()));
            }
            Ok(hash)
        }

        fn vk_hash(key: &[u32; 8]) -> [u32; 8] {
            *key
        }

        async fn prove(
            &self,
            _key: &[u32; 8],
            _stdin: SP1Stdin,
            _mode: ProofMode,
        ) -> Result<SP1ProofWithPublicValues, ServerError> {
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
            prove_error: Option<fn() -> ServerError>,
        ) -> Self {
            Self::build(devices, execute, prove, result, prove_error, u64::MAX)
        }

        fn build(
            devices: u32,
            execute: Probe,
            prove: Probe,
            result: ExecuteResult,
            prove_error: Option<fn() -> ServerError>,
            queued_stdin_bytes: u64,
        ) -> Self {
            let executor = Arc::new(ScriptedExecutor { probe: execute, result });
            let script = Arc::new(ServerScript {
                prove,
                setups: Mutex::new(Vec::new()),
                wrong_vk: AtomicBool::new(false),
                rejected_setup: AtomicU8::new(0),
                prove_error,
            });
            let devices = (0..devices)
                .map(|id| Device {
                    id,
                    server: ScriptedServer { id, script: Arc::clone(&script) },
                    keys: HashMap::new(),
                })
                .collect();
            let core =
                Core::new(Box::new(Arc::clone(&executor)), devices, config(), queued_stdin_bytes);
            Self { core, executor, script }
        }

        async fn request(&self) -> ProofId {
            self.core.request(Program::Range, prestate_keys(), SP1Stdin::new()).await
        }

        async fn finished(&self, proof_id: ProofId) {
            let mut receiver = self.core.outcomes.lock()[&proof_id].clone();
            receiver.wait_for(Option::is_some).await.unwrap();
        }

        /// The terminal state of a request whose pre-execution returns `failure`.
        async fn terminal_state(failure: ExecuteResult) -> ProofTerminalState {
            let harness = Self::with(1, Probe::open(), Probe::open(), failure, None);
            let proof_id = harness.request().await;
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
        let proof_id = harness.request().await;

        harness.core.wait(proof_id).await.unwrap();

        assert!(harness.core.outcomes.lock().is_empty());
        assert_eq!(harness.script.prove.calls(), 1);
    }

    /// Pins that a request waits while its stdin would take the queued bytes over the budget.
    #[tokio::test(start_paused = true)]
    async fn request_waits_for_the_queued_stdin_budget() {
        let harness = Harness::build(
            1,
            Probe::open(),
            Probe::gated(0),
            |_| Ok(ExecutionReport::default()),
            None,
            4096,
        );
        let request = || {
            let mut stdin = SP1Stdin::new();
            stdin.write_vec(vec![0; 1024]);
            harness.core.request(Program::Range, prestate_keys(), stdin)
        };
        let first = request().await;
        request().await;
        let third = request();
        tokio::pin!(third);

        assert!(tokio::time::timeout(Duration::from_secs(60), &mut third).await.is_err());

        harness.script.prove.gate.add_permits(1);
        harness.finished(first).await;
        tokio::time::timeout(Duration::from_secs(60), third).await.unwrap();
    }

    /// Pins that a stdin larger than the whole budget waits for an empty queue, then runs alone.
    #[tokio::test(start_paused = true)]
    async fn stdin_over_the_budget_waits_for_an_empty_queue() {
        let harness = Harness::build(
            1,
            Probe::open(),
            Probe::gated(0),
            |_| Ok(ExecutionReport::default()),
            None,
            1024,
        );
        let request = || {
            let mut stdin = SP1Stdin::new();
            stdin.write_vec(vec![0; 1024]);
            harness.core.request(Program::Range, prestate_keys(), stdin)
        };
        let first = request().await;
        let second = request();
        tokio::pin!(second);

        assert!(tokio::time::timeout(Duration::from_secs(60), &mut second).await.is_err());

        harness.script.prove.gate.add_permits(1);
        harness.finished(first).await;
        tokio::time::timeout(Duration::from_secs(60), second).await.unwrap();
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
        let proof_id = harness.request().await;

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
        let proof_id = harness.request().await;
        harness.core.wait(proof_id).await.unwrap_err();
        assert_eq!(harness.executor.probe.calls(), MEMORY_MONITOR_ATTEMPTS);

        let first_kill = |call| match call {
            0 => Err(execution_error(ExecutionError::KilledByMemoryMonitor(1))),
            _ => Ok(ExecutionReport::default()),
        };
        let harness = Harness::with(1, Probe::open(), Probe::open(), first_kill, None);
        let proof_id = harness.request().await;
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
        let proof_id = harness.request().await;

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
        let proof_id = harness.request().await;

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
            Some(|| ServerError::Rejected("out of memory".into())),
        );
        let proof_id = harness.request().await;

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
            Some(|| ServerError::Io(io::ErrorKind::BrokenPipe.into())),
        );
        let proof_id = harness.request().await;

        let err = harness.core.wait(proof_id).await.unwrap_err();

        assert!(
            matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::ProverFailed, .. }),
            "{err:?}"
        );
        let fatal = harness.core.fatal().await.to_string();
        assert!(fatal.contains("server connection failed"), "{fatal}");
        assert!(harness.core.devices.free.lock().is_empty());
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
        let first = harness.request().await;
        while harness.script.prove.calls() == 0 {
            tokio::time::sleep(Duration::from_millis(100)).await;
        }

        let second = harness.request().await;
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
        let mut proof_ids = Vec::new();
        for _ in 0..5 {
            proof_ids.push(harness.request().await);
        }

        for proof_id in proof_ids {
            harness.core.wait(proof_id).await.unwrap();
        }

        assert_eq!(harness.executor.probe.calls(), 5);
        assert_eq!(harness.executor.probe.peak(), 2);
    }

    #[tokio::test(start_paused = true)]
    async fn requests_never_exceed_device_count() {
        let harness = Harness::new(2);
        let mut proof_ids = Vec::new();
        for _ in 0..5 {
            proof_ids.push(harness.request().await);
        }

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

        let Err(Failure::ProverFailed(err)) =
            device.ensure_key(Program::Range, &prestate_keys()).await
        else {
            panic!("expected a prover failure");
        };

        assert!(err.to_string().contains("super-range"), "{err}");
        assert!(device.keys.is_empty());
    }

    /// Pins that a failing aggregation setup neither blocks range proofs nor repeats their setup.
    #[tokio::test(start_paused = true)]
    async fn failed_setup_of_one_program_leaves_the_others_usable() {
        let harness = Harness::new(1);
        harness.script.rejected_setup.store(b'a', Ordering::SeqCst);
        let keys = prestate_keys();

        let aggregation =
            harness.core.request(Program::Aggregation, keys.clone(), SP1Stdin::new()).await;
        let err = harness.core.wait(aggregation).await.unwrap_err();
        assert!(
            matches!(err, ProofWaitError::Terminal { state: ProofTerminalState::ProverFailed, .. }),
            "{err:?}"
        );
        for _ in 0..2 {
            let proof_id = harness.request().await;
            harness.core.wait(proof_id).await.unwrap();
        }

        let programs = &keys.programs;
        let expected = [(0, programs.aggregation.vk_hash), (0, programs.range.vk_hash)];
        assert_eq!(*harness.script.setups.lock(), expected);
    }

    #[tokio::test(start_paused = true)]
    async fn keys_set_up_once_per_device_and_prestate() {
        let harness = Harness::new(1);

        for _ in 0..2 {
            let proof_id = harness.request().await;
            harness.core.wait(proof_id).await.unwrap();
        }

        assert_eq!(*harness.script.setups.lock(), [(0, prestate_keys().programs.range.vk_hash)]);
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
        let proof_id = harness.request().await;
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
        let proof_id = harness.request().await;
        harness.finished(proof_id).await;

        tokio::time::advance(harness.core.timeout()).await;
        let kept = harness.request().await;
        assert!(harness.core.outcomes.lock().contains_key(&proof_id));

        tokio::time::advance(Duration::from_secs(1)).await;
        harness.request().await;

        let outcomes = harness.core.outcomes.lock();
        assert!(!outcomes.contains_key(&proof_id));
        assert!(outcomes.contains_key(&kept));
    }

    #[tokio::test]
    async fn startup_rejects_missing_or_mismatched_gpu_server() {
        let dir = tempfile::tempdir().unwrap();
        let server = dir.path().join("sp1-gpu-server");
        let err = check_gpu_server(&server).await.unwrap_err().to_string();
        assert!(err.contains(CUDA_IMAGE), "{err}");

        for (version, ok) in [("6.9.0", false), (SP1_CRATE_VERSION, true)] {
            std::fs::write(&server, format!("#!/bin/sh\necho {version}\n")).unwrap();
            std::fs::set_permissions(&server, std::fs::Permissions::from_mode(0o755)).unwrap();
            match check_gpu_server(&server).await {
                Ok(()) => assert!(ok, "{version} accepted"),
                Err(err) => {
                    assert!(!ok, "{version}: {err}");
                    assert!(err.to_string().contains(CUDA_IMAGE), "{err}");
                }
            }
        }
    }

    /// Pins that the `--version` run gets no `OP_ZK_PROPOSER_*` variable.
    #[tokio::test]
    #[serial_test::serial]
    async fn version_check_gets_no_proposer_env() {
        let dir = tempfile::tempdir().unwrap();
        let server = dir.path().join("sp1-gpu-server");
        let prefix = crate::ENV_VAR_PREFIX;
        std::fs::write(
            &server,
            format!("#!/bin/sh\nenv | grep -q '^{prefix}' || echo {SP1_CRATE_VERSION}\n"),
        )
        .unwrap();
        std::fs::set_permissions(&server, std::fs::Permissions::from_mode(0o755)).unwrap();
        let name = format!("{prefix}_TEST_SECRET");
        // SAFETY: The environment-mutating tests in this crate are serialized.
        unsafe { std::env::set_var(&name, "secret") };

        let result = check_gpu_server(&server).await;

        // SAFETY: as above.
        unsafe { std::env::remove_var(&name) };
        result.unwrap();
    }

    /// Pins the image's server and circuit downloads to the linked SP1 versions, which the
    /// startup check and the server require.
    #[test]
    fn image_pins_match_the_linked_sp1_versions() {
        let melange = Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../../../../melange/op-zk-proposer-cuda.yaml");
        let melange = std::fs::read_to_string(melange).unwrap();
        let circuits = sp1_sdk::SP1_CIRCUIT_VERSION.trim();
        for pin in [
            format!("/v{SP1_CRATE_VERSION}/sp1_gpu_server_v{SP1_CRATE_VERSION}_x86_64.tar.gz"),
            format!("/{circuits}-plonk.tar.gz"),
            format!("\"$dir/{circuits}\""),
        ] {
            assert!(melange.contains(&pin), "melange/op-zk-proposer-cuda.yaml lacks {pin}");
        }
    }

    /// Pins that each program's mode accepts only the proof kind of that mode.
    #[test]
    fn proof_kind_must_match_the_program_mode() {
        let core = SP1Proof::Core(Vec::new());
        let plonk = SP1Proof::Plonk(PlonkBn254Proof::default());

        assert!(proof_matches_mode(&plonk, Program::Aggregation.mode()));
        assert!(!proof_matches_mode(&core, Program::Aggregation.mode()));
        assert!(!proof_matches_mode(&core, Program::Range.mode()));
        assert!(!proof_matches_mode(&plonk, Program::LogCommitment.mode()));
    }

    #[test]
    fn server_error_classes() {
        let bincode_error = || Box::new(bincode1::ErrorKind::Custom("bad".into()));
        let cases = [
            (ServerError::Rejected("cuda".into()), true),
            (ServerError::TooLarge(1), true),
            (ServerError::Encode(bincode_error()), true),
            (ServerError::Decode(bincode_error()), false),
            (ServerError::Io(io::ErrorKind::BrokenPipe.into()), false),
            (ServerError::Unexpected("Ok"), false),
        ];
        for (err, retryable) in cases {
            let text = err.to_string();
            match err.into_failure() {
                Failure::ProverFailed(_) => assert!(retryable, "{text}"),
                Failure::Fatal(_) => assert!(!retryable, "{text}"),
                Failure::Unexecutable(_) => panic!("{text}"),
            }
        }
    }

    /// A fake server end that answers each request with the next of `responses`, and returns the
    /// requests it read.
    fn fake_server(
        mut stream: UnixStream,
        responses: Vec<Response>,
    ) -> tokio::task::JoinHandle<Vec<Request>> {
        tokio::spawn(async move {
            let mut requests = Vec::new();
            for response in responses {
                let mut len = [0; 4];
                stream.read_exact(&mut len).await.unwrap();
                let mut request = vec![0; u32::from_le_bytes(len) as usize];
                stream.read_exact(&mut request).await.unwrap();
                requests.push(bincode1::deserialize(&request).unwrap());
                let response = bincode1::serialize(&response).unwrap();
                stream.write_all(&(response.len() as u32).to_le_bytes()).await.unwrap();
                stream.write_all(&response).await.unwrap();
            }
            requests
        })
    }

    fn connection(stream: UnixStream) -> Connection {
        Connection { stream: tokio::sync::Mutex::new(stream) }
    }

    /// Pins the request and response framing of `sp1_cuda::client`.
    #[tokio::test]
    async fn prove_round_trips_the_wire_protocol() {
        let (client, server) = UnixStream::pair().unwrap();
        let proof = ProofFromNetwork {
            proof: SP1Proof::Core(Vec::new()),
            public_values: SP1PublicValues::from(&[7u8][..]),
            sp1_version: "v6.1.0".into(),
        };
        let requests = fake_server(server, vec![Response::Proof { proof }]);
        let gpu = connection(client);
        let mut stdin = SP1Stdin::new();
        stdin.write(&42u64);
        let key = ServerKey { id: [9; 32], vk_hash: [0; 8] };

        let request = Request::ProveWithMode {
            mode: ProofMode::Plonk,
            key: key.id,
            stdin,
            proof_nonce: [0; 4],
        };
        let Response::Proof { proof } = gpu.call(request).await.unwrap() else {
            panic!("expected a proof")
        };

        assert_eq!(SP1ProofWithPublicValues::from(proof).public_values.as_slice(), [7]);
        let requests = requests.await.unwrap();
        let [Request::ProveWithMode { mode, key, stdin, proof_nonce }] = requests.as_slice() else {
            panic!("expected one prove request")
        };
        assert_eq!((*mode, *key, *proof_nonce), (ProofMode::Plonk, [9; 32], [0; 4]));
        assert_eq!(stdin.buffer, [bincode1::serialize(&42u64).unwrap()]);
    }

    #[tokio::test]
    async fn server_error_responses_are_rejected() {
        let (client, server) = UnixStream::pair().unwrap();
        let requests = fake_server(
            server,
            vec![Response::InternalError("oom".into()), Response::ProverError("bad".into())],
        );
        let gpu = connection(client);

        for expected in ["oom", "bad"] {
            let Err(ServerError::Rejected(message)) =
                gpu.call(Request::Destroy { key: [0; 32] }).await
            else {
                panic!("expected a rejection")
            };
            assert_eq!(message, expected);
        }
        assert_eq!(requests.await.unwrap().len(), 2);
    }

    #[tokio::test]
    async fn connect_fails_when_the_server_exits() {
        let dir = tempfile::tempdir().unwrap();
        let mut child =
            Command::new("sh").args(["-c", "exit 3"]).kill_on_drop(true).spawn().unwrap();

        let err = connect(&dir.path().join("server.sock"), &mut child).await.unwrap_err();

        assert!(err.to_string().contains("exited"), "{err}");
    }

    #[tokio::test]
    async fn connect_waits_for_a_late_socket() {
        let dir = tempfile::tempdir().unwrap();
        let socket = dir.path().join("server.sock");
        let mut child =
            Command::new("sh").args(["-c", "sleep 30"]).kill_on_drop(true).spawn().unwrap();
        let listener = {
            let socket = socket.clone();
            tokio::spawn(async move {
                tokio::time::sleep(Duration::from_millis(300)).await;
                let listener = tokio::net::UnixListener::bind(&socket).unwrap();
                listener.accept().await.unwrap();
            })
        };

        connect(&socket, &mut child).await.unwrap();

        listener.await.unwrap();
    }

    #[tokio::test]
    async fn closed_connection_is_io_failure() {
        let (client, server) = UnixStream::pair().unwrap();
        drop(server);

        let result = connection(client).call(Request::Destroy { key: [0; 32] }).await;

        assert!(matches!(result, Err(ServerError::Io(_))));
    }
}
