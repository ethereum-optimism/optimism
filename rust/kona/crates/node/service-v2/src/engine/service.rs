//! Top-level Engine service and its single raw Engine API core.

use crate::engine::{
    EngineAdminAdapter, EngineError, EngineHandle, EngineResult, EngineRpcAdapter,
    EngineServiceError, SafeChainUpdate,
    api::{BuiltUnsafePayload, EngineInternalHandle, EngineRequest},
    runtime::{EngineRuntime, EngineRuntimeConfig},
    unsafe_chain::UnsafeLifecycleCommand,
};
use alloy_rpc_types_engine::{ExecutionPayload, PayloadId, PayloadStatusEnum};
use kona_engine::{
    BuildTask, ConsolidateInput, ConsolidateTask, EngineClient as RawEngineClient,
    EngineGetPayloadVersion, EngineQueries, EngineState, EngineSyncStateUpdate, EngineTaskError,
    EngineTaskErrorSeverity, EngineTaskExt, FinalizeBlockId, FinalizeTask, InsertTask,
    InsertTaskError, SyncStartError, SynchronizeTask, find_starting_forkchoice,
};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    future::pending,
    sync::Arc,
    time::{Duration, Instant},
};
use tokio::{
    sync::{mpsc, oneshot, watch},
    task::{JoinError, JoinHandle},
};

/// Default number of semantic operations that may wait for Engine.
pub const DEFAULT_ENGINE_REQUEST_CAPACITY: usize = 64;
const LIFECYCLE_CAPACITY: usize = 4;

/// Capabilities published only after startup forkchoice synchronization succeeds.
#[derive(Debug)]
pub(crate) struct EngineStarted {
    /// The only Engine capability supplied to Derivation.
    pub handle: EngineHandle,
    /// Read-only Engine RPC capability.
    pub rpc: EngineRpcAdapter,
    /// Unsafe-chain administration capability supplied only to RPC.
    pub admin: EngineAdminAdapter,
    /// Initial authoritative safe head used to initialize Derivation.
    pub safe_head: L2BlockInfo,
}

#[derive(Debug)]
enum EngineLifecycleCommand {
    Quiesce(oneshot::Sender<Result<(), String>>),
    Shutdown(oneshot::Sender<Result<(), String>>),
}

/// Node-owned lifecycle capability. This is intentionally distinct from all domain APIs.
#[derive(Debug)]
pub(crate) struct EngineController {
    tx: mpsc::Sender<EngineLifecycleCommand>,
}

impl EngineController {
    /// Stops local production and inbound unsafe processing at a block boundary while retaining
    /// the raw Engine core for Derivation drain.
    pub(crate) async fn quiesce_unsafe(&self) -> Result<(), String> {
        self.request(EngineLifecycleCommand::Quiesce).await
    }

    /// Requests complete Engine shutdown and waits for all private children to be joined.
    pub(crate) async fn shutdown(&self) -> Result<(), String> {
        self.request(EngineLifecycleCommand::Shutdown).await
    }

    async fn request(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<(), String>>) -> EngineLifecycleCommand,
    ) -> Result<(), String> {
        let (response, result) = oneshot::channel();
        self.tx
            .send(command(response))
            .await
            .map_err(|_| "Engine task is unavailable".to_string())?;
        result.await.map_err(|_| "Engine lifecycle response was dropped".to_string())?
    }
}

/// Owns the raw Engine core plus private network, follower, signer, and sequencer workflows.
#[derive(Debug)]
pub struct EngineService<Client> {
    core: EngineCore<Client>,
    runtime: EngineRuntime,
    lifecycle_rx: mpsc::Receiver<EngineLifecycleCommand>,
    derivation_handle: EngineHandle,
    rpc_adapter: EngineRpcAdapter,
}

impl<Client> EngineService<Client>
where
    Client: RawEngineClient + 'static,
{
    /// Constructs the complete Engine domain without spawning any task.
    pub(crate) fn new(
        client: Arc<Client>,
        config: Arc<RollupConfig>,
        runtime: EngineRuntimeConfig,
    ) -> (Self, EngineController) {
        let (core, derivation_handle, rpc_adapter, internal) =
            EngineCore::new(client, config.clone(), DEFAULT_ENGINE_REQUEST_CAPACITY);
        let runtime = runtime.build(config, internal);
        let (lifecycle_tx, lifecycle_rx) = mpsc::channel(LIFECYCLE_CAPACITY);
        (
            Self { core, runtime, lifecycle_rx, derivation_handle, rpc_adapter },
            EngineController { tx: lifecycle_tx },
        )
    }

    /// Synchronizes initial forkchoice, publishes capabilities, and supervises private children.
    pub(crate) async fn run(
        mut self,
        started: oneshot::Sender<EngineStarted>,
    ) -> Result<(), EngineServiceError> {
        let Some(safe_head) = self
            .core
            .recover_for_startup(&mut self.lifecycle_rx)
            .await
            .map_err(|error| EngineServiceError::Startup(error.to_string()))?
        else {
            return Ok(());
        };
        self.core.publish_state();

        let EngineRuntime { network, signer, unsafe_chain, unsafe_lifecycle, admin } = self.runtime;
        let (network_shutdown_tx, network_shutdown_rx) = oneshot::channel();
        let (signer_shutdown_tx, signer_shutdown_rx) = oneshot::channel();
        let (core_shutdown_tx, core_shutdown_rx) = oneshot::channel();

        let mut network_task = Some(tokio::spawn(network.run(network_shutdown_rx)));
        let mut signer_task = Some(tokio::spawn(signer.run(signer_shutdown_rx)));
        let mut unsafe_task = Some(tokio::spawn(unsafe_chain.run()));
        let mut core_task = Some(tokio::spawn(self.core.run(core_shutdown_rx)));
        let mut network_shutdown_tx = Some(network_shutdown_tx);
        let mut signer_shutdown_tx = Some(signer_shutdown_tx);
        let mut core_shutdown_tx = Some(core_shutdown_tx);

        if started
            .send(EngineStarted {
                handle: self.derivation_handle,
                rpc: self.rpc_adapter,
                admin,
                safe_head,
            })
            .is_err()
        {
            let _ = Self::shutdown_tasks(
                &unsafe_lifecycle,
                &mut unsafe_task,
                &mut signer_shutdown_tx,
                &mut signer_task,
                &mut network_shutdown_tx,
                &mut network_task,
                &mut core_shutdown_tx,
                &mut core_task,
            )
            .await;
            return Err(EngineServiceError::StartupReceiverDropped);
        }

        enum Exit {
            Requested(Option<oneshot::Sender<Result<(), String>>>),
            Failed(EngineServiceError),
        }

        let exit = loop {
            tokio::select! {
                biased;
                command = self.lifecycle_rx.recv() => {
                    match command {
                        Some(EngineLifecycleCommand::Quiesce(response)) => {
                            let result = request_unsafe_lifecycle(
                                &unsafe_lifecycle,
                                UnsafeLifecycleCommand::Quiesce,
                            ).await;
                            let _ = response.send(result.clone());
                            if let Err(error) = result {
                                break Exit::Failed(EngineServiceError::Lifecycle(error));
                            }
                        }
                        Some(EngineLifecycleCommand::Shutdown(response)) => {
                            break Exit::Requested(Some(response));
                        }
                        None => break Exit::Requested(None),
                    }
                }
                result = poll_task(&mut unsafe_task) => {
                    unsafe_task.take();
                    break Exit::Failed(child_result("unsafe", result));
                }
                result = poll_task(&mut signer_task) => {
                    signer_task.take();
                    break Exit::Failed(child_result("signer", result));
                }
                result = poll_task(&mut network_task) => {
                    network_task.take();
                    break Exit::Failed(child_result("network", result));
                }
                result = poll_task(&mut core_task) => {
                    core_task.take();
                    break Exit::Failed(child_result("core", result));
                }
            }
        };

        let cleanup = Self::shutdown_tasks(
            &unsafe_lifecycle,
            &mut unsafe_task,
            &mut signer_shutdown_tx,
            &mut signer_task,
            &mut network_shutdown_tx,
            &mut network_task,
            &mut core_shutdown_tx,
            &mut core_task,
        )
        .await;

        match exit {
            Exit::Requested(response) => {
                let response_result =
                    cleanup.as_ref().map_or(Ok(()), |error| Err(error.to_string()));
                if let Some(response) = response {
                    let _ = response.send(response_result);
                }
                cleanup.map_or(Ok(()), Err)
            }
            Exit::Failed(error) => Err(error),
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn shutdown_tasks(
        unsafe_lifecycle: &mpsc::Sender<UnsafeLifecycleCommand>,
        unsafe_task: &mut Option<
            JoinHandle<Result<(), super::unsafe_chain::UnsafeChainServiceError>>,
        >,
        signer_shutdown_tx: &mut Option<oneshot::Sender<()>>,
        signer_task: &mut Option<JoinHandle<Result<(), super::signer::SignerTrackerError>>>,
        network_shutdown_tx: &mut Option<oneshot::Sender<()>>,
        network_task: &mut Option<JoinHandle<Result<(), super::network::NetworkServiceError>>>,
        core_shutdown_tx: &mut Option<oneshot::Sender<()>>,
        core_task: &mut Option<JoinHandle<Result<(), EngineServiceError>>>,
    ) -> Option<EngineServiceError> {
        let mut first_error = if unsafe_task.is_some() &&
            let Err(error) =
                request_unsafe_lifecycle(unsafe_lifecycle, UnsafeLifecycleCommand::Shutdown)
                    .await
        {
            Some(EngineServiceError::Lifecycle(error))
        } else {
            None
        };
        record_join("unsafe", unsafe_task.take(), &mut first_error).await;

        // Signal both transport-side children before joining either one. Otherwise dropping the
        // signer sender can look like an unexpected network failure before network sees shutdown.
        if let Some(tx) = signer_shutdown_tx.take() {
            let _ = tx.send(());
        }
        if let Some(tx) = network_shutdown_tx.take() {
            let _ = tx.send(());
        }
        record_join("signer", signer_task.take(), &mut first_error).await;
        record_join("network", network_task.take(), &mut first_error).await;

        if let Some(tx) = core_shutdown_tx.take() {
            let _ = tx.send(());
        }
        record_join("core", core_task.take(), &mut first_error).await;
        first_error
    }
}

async fn request_unsafe_lifecycle(
    sender: &mpsc::Sender<UnsafeLifecycleCommand>,
    command: impl FnOnce(oneshot::Sender<()>) -> UnsafeLifecycleCommand,
) -> Result<(), String> {
    let (done, result) = oneshot::channel();
    sender
        .send(command(done))
        .await
        .map_err(|_| "Engine unsafe workflow is unavailable".to_string())?;
    result.await.map_err(|_| "Engine unsafe lifecycle response was dropped".to_string())
}

async fn poll_task<E>(
    task: &mut Option<JoinHandle<Result<(), E>>>,
) -> Result<Result<(), E>, JoinError> {
    match task {
        Some(task) => task.await,
        None => pending().await,
    }
}

fn child_result<E>(
    name: &'static str,
    result: Result<Result<(), E>, JoinError>,
) -> EngineServiceError
where
    E: core::fmt::Display,
{
    match result {
        Ok(Ok(())) => {
            EngineServiceError::Child { name, error: "terminated unexpectedly".to_string() }
        }
        Ok(Err(error)) => EngineServiceError::Child { name, error: error.to_string() },
        Err(error) => EngineServiceError::ChildPanic { name, error: error.to_string() },
    }
}

async fn record_join<E>(
    name: &'static str,
    task: Option<JoinHandle<Result<(), E>>>,
    first_error: &mut Option<EngineServiceError>,
) where
    E: core::fmt::Display,
{
    let Some(task) = task else { return };
    let error = match task.await {
        Ok(Ok(())) => return,
        result => child_result(name, result),
    };
    if first_error.is_none() {
        *first_error = Some(error);
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum LocalBuildTransition {
    Unchanged,
    Started,
    Finished,
}

/// The single owner of raw Engine API calls and authoritative forkchoice state.
#[derive(Debug)]
struct EngineCore<Client> {
    client: Arc<Client>,
    config: Arc<RollupConfig>,
    state: EngineState,
    state_tx: watch::Sender<EngineState>,
    queue_length_tx: watch::Sender<usize>,
    request_rx: mpsc::Receiver<EngineRequest>,
    local_build_rx: mpsc::Receiver<EngineRequest>,
    query_rx: mpsc::Receiver<EngineRequest>,
    local_sequencing_active: bool,
}

impl<Client> EngineCore<Client>
where
    Client: RawEngineClient + 'static,
{
    fn new(
        client: Arc<Client>,
        config: Arc<RollupConfig>,
        capacity: usize,
    ) -> (Self, EngineHandle, EngineRpcAdapter, EngineInternalHandle) {
        let (request_tx, request_rx) = mpsc::channel(capacity);
        let (local_build_tx, local_build_rx) = mpsc::channel(capacity);
        let (query_tx, query_rx) = mpsc::channel(capacity);
        let (state_tx, _) = watch::channel(EngineState::default());
        let (queue_length_tx, _) = watch::channel(0);
        (
            Self {
                client,
                config,
                state: EngineState::default(),
                state_tx,
                queue_length_tx,
                request_rx,
                local_build_rx,
                query_rx,
                local_sequencing_active: false,
            },
            EngineHandle::new(request_tx.clone()),
            EngineRpcAdapter::new(query_tx),
            EngineInternalHandle::new(request_tx, local_build_tx),
        )
    }

    /// Serves operations until its owner requests shutdown. Shutdown is observed only between
    /// operations, so an accepted remote side effect is never abandoned ambiguously.
    async fn run(mut self, mut shutdown: oneshot::Receiver<()>) -> Result<(), EngineServiceError> {
        let mut local_build_active = false;
        loop {
            let request = if local_build_active {
                tokio::select! {
                    biased;
                    _ = &mut shutdown => return Ok(()),
                    request = self.local_build_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed("local build"))?
                    }
                    request = self.query_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed("query"))?
                    }
                }
            } else {
                tokio::select! {
                    biased;
                    _ = &mut shutdown => return Ok(()),
                    request = self.local_build_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed("local build"))?
                    }
                    request = self.request_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed("semantic"))?
                    }
                    request = self.query_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed("query"))?
                    }
                }
            };

            self.publish_queue_length(1);
            match self.handle_request(request, local_build_active).await {
                LocalBuildTransition::Unchanged => {}
                LocalBuildTransition::Started => local_build_active = true,
                LocalBuildTransition::Finished => local_build_active = false,
            }
            self.publish_state();
            self.publish_queue_length(0);
        }
    }

    async fn handle_request(
        &mut self,
        request: EngineRequest,
        local_build_active: bool,
    ) -> LocalBuildTransition {
        match request {
            EngineRequest::BuildUnsafe { attributes, response } => {
                if local_build_active {
                    let _ = response.send(Err(EngineError::Critical(
                        "a local unsafe build is already active".to_string(),
                    )));
                    return LocalBuildTransition::Unchanged;
                }
                let result = self.build_unsafe(*attributes).await;
                let started = result.is_ok();
                let _ = response.send(result);
                if started {
                    LocalBuildTransition::Started
                } else {
                    LocalBuildTransition::Unchanged
                }
            }
            EngineRequest::CanonicalizeUnsafe { candidate, response } => {
                if !local_build_active {
                    let _ = response.send(Err(EngineError::Critical(
                        "no local unsafe build is active".to_string(),
                    )));
                    return LocalBuildTransition::Unchanged;
                }
                let result = self.canonicalize_unsafe(*candidate).await;
                let _ = response.send(result);
                LocalBuildTransition::Finished
            }
            EngineRequest::ImportUnsafe { payload, response } => {
                let result = self.import_unsafe(*payload).await;
                let _ = response.send(result);
                LocalBuildTransition::Unchanged
            }
            EngineRequest::UpdateSafe { update, response } => {
                let result = self.update_safe_owned(update).await;
                let _ = response.send(result);
                LocalBuildTransition::Unchanged
            }
            EngineRequest::UpdateFinalized { block, response } => {
                let result = self.update_finalized(block).await;
                let _ = response.send(result);
                LocalBuildTransition::Unchanged
            }
            EngineRequest::SetLocalSequencing { active, response } => {
                if local_build_active && !active {
                    let _ = response.send(Err(EngineError::Critical(
                        "cannot deactivate local sequencing during an active build".to_string(),
                    )));
                } else {
                    self.local_sequencing_active = active;
                    let _ = response.send(Ok(()));
                }
                LocalBuildTransition::Unchanged
            }
            EngineRequest::State { response } => {
                let _ = response.send(Ok(self.state.sync_state));
                LocalBuildTransition::Unchanged
            }
            EngineRequest::Query(query) => {
                self.handle_query(*query).await;
                LocalBuildTransition::Unchanged
            }
        }
    }

    async fn build_unsafe(
        &mut self,
        attributes: OpAttributesWithParent,
    ) -> EngineResult<BuiltUnsafePayload> {
        let _build_started = Instant::now();
        let payload_id =
            BuildTask::new(self.client.clone(), self.config.clone(), attributes.clone(), None)
                .execute(&mut self.state)
                .await
                .map_err(Self::map_task_error);
        kona_macros::set!(
            gauge,
            crate::Metrics::SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION,
            _build_started.elapsed()
        );
        let payload_id = payload_id?;

        if self.state.sync_state.unsafe_head() != attributes.parent {
            return Err(EngineError::StaleBuild);
        }

        let _seal_started = Instant::now();
        let payload = self.fetch_payload(payload_id, &attributes).await;
        kona_macros::set!(
            gauge,
            crate::Metrics::SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION,
            _seal_started.elapsed()
        );
        Ok(BuiltUnsafePayload::new(payload?, attributes.parent))
    }

    async fn canonicalize_unsafe(
        &mut self,
        candidate: BuiltUnsafePayload,
    ) -> EngineResult<L2BlockInfo> {
        if self.state.sync_state.unsafe_head() != candidate.parent() {
            return Err(EngineError::StaleBuild);
        }

        loop {
            match self.import_unsafe(candidate.payload().clone()).await {
                Ok(block) => return Ok(block),
                Err(EngineError::Temporary(_)) => {
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(EngineError::InvalidUnsafePayload(error)) => {
                    return Err(EngineError::Critical(format!(
                        "locally built payload was rejected after publication: {error}"
                    )));
                }
                Err(error) => return Err(error),
            }
        }
    }

    async fn fetch_payload(
        &self,
        payload_id: PayloadId,
        attributes: &OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope> {
        let timestamp = attributes.attributes().payload_attributes.timestamp;
        let result = match EngineGetPayloadVersion::from_cfg(&self.config, timestamp) {
            EngineGetPayloadVersion::V5 => {
                self.client.get_payload_v5(payload_id).await.map(|payload| {
                    OpExecutionPayloadEnvelope::V4 {
                        parent_beacon_block_root: payload.parent_beacon_block_root,
                        payload: payload.execution_payload,
                    }
                })
            }
            EngineGetPayloadVersion::V4 => {
                self.client.get_payload_v4(payload_id).await.map(|payload| {
                    OpExecutionPayloadEnvelope::V4 {
                        parent_beacon_block_root: payload.parent_beacon_block_root,
                        payload: payload.execution_payload,
                    }
                })
            }
            EngineGetPayloadVersion::V3 => {
                self.client.get_payload_v3(payload_id).await.map(|payload| {
                    OpExecutionPayloadEnvelope::V3 {
                        parent_beacon_block_root: payload.parent_beacon_block_root,
                        payload: payload.execution_payload,
                    }
                })
            }
            EngineGetPayloadVersion::V2 => {
                self.client.get_payload_v2(payload_id).await.map(|payload| {
                    match payload.execution_payload.into_payload() {
                        ExecutionPayload::V1(payload) => OpExecutionPayloadEnvelope::V1(payload),
                        ExecutionPayload::V2(payload) => OpExecutionPayloadEnvelope::V2(payload),
                        _ => unreachable!("getPayloadV2 returned a post-V2 execution payload"),
                    }
                })
            }
        };

        result.map_err(|error| EngineError::Temporary(error.to_string()))
    }

    async fn import_unsafe(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        match InsertTask::new(self.client.clone(), self.config.clone(), payload, false)
            .execute(&mut self.state)
            .await
        {
            Err(InsertTaskError::UnexpectedPayloadStatus(
                status @ PayloadStatusEnum::Invalid { .. },
            )) => Err(EngineError::InvalidUnsafePayload(status.to_string())),
            Err(error) => Err(Self::map_task_error(error)),
            Ok(block) => Ok(block),
        }
    }

    async fn update_safe_owned(&mut self, update: SafeChainUpdate) -> EngineResult<L2BlockInfo> {
        let input = match update {
            SafeChainUpdate::Attributes(attributes) => ConsolidateInput::Attributes(attributes),
            SafeChainUpdate::Block(block) => ConsolidateInput::BlockInfo(block),
        };

        match ConsolidateTask::new(self.client.clone(), self.config.clone(), input)
            .execute(&mut self.state)
            .await
        {
            Ok(()) => Ok(self.state.sync_state.safe_head()),
            Err(error) => match error.severity() {
                EngineTaskErrorSeverity::Reset => {
                    self.ensure_recovery_allowed()?;
                    let reason = error.to_string();
                    self.recover_from_execution_layer().await?;
                    Err(EngineError::ResetRequired {
                        reason,
                        safe_head: Some(self.state.sync_state.safe_head()),
                    })
                }
                EngineTaskErrorSeverity::Flush => Err(EngineError::FlushRequired {
                    reason: error.to_string(),
                    safe_head: Some(self.state.sync_state.safe_head()),
                }),
                _ => Err(Self::map_task_error(error)),
            },
        }
    }

    const fn ensure_recovery_allowed(&self) -> EngineResult<()> {
        if self.local_sequencing_active {
            Err(EngineError::RecoveryWhileSequencing)
        } else {
            Ok(())
        }
    }

    async fn update_finalized(&mut self, block: L2BlockInfo) -> EngineResult<()> {
        FinalizeTask::new(
            self.client.clone(),
            self.config.clone(),
            FinalizeBlockId::ByHash(block.block_info.id()),
        )
        .execute(&mut self.state)
        .await
        .map_err(Self::map_task_error)
    }

    async fn recover_for_startup(
        &mut self,
        lifecycle: &mut mpsc::Receiver<EngineLifecycleCommand>,
    ) -> EngineResult<Option<L2BlockInfo>> {
        loop {
            if startup_should_stop(lifecycle) {
                return Ok(None);
            }
            let start = match find_starting_forkchoice(&self.config, self.client.as_ref()).await {
                Ok(start) => start,
                Err(error @ (SyncStartError::RpcError(_) | SyncStartError::BlockNotFound(_))) => {
                    warn!(target: "engine", ?error, "Startup forkchoice discovery yielded; retrying");
                    if startup_should_stop(lifecycle) {
                        return Ok(None);
                    }
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                    continue;
                }
                Err(error) => {
                    if startup_should_stop(lifecycle) {
                        return Ok(None);
                    }
                    return Err(EngineError::Critical(error.to_string()));
                }
            };

            let result = SynchronizeTask::new(
                self.client.clone(),
                self.config.clone(),
                EngineSyncStateUpdate {
                    unsafe_head: Some(start.un_safe),
                    cross_unsafe_head: Some(start.un_safe),
                    local_safe_head: Some(start.safe),
                    safe_head: Some(start.safe),
                    finalized_head: Some(start.finalized),
                },
            )
            .execute(&mut self.state)
            .await;

            if startup_should_stop(lifecycle) {
                return Ok(None);
            }
            match result {
                Ok(()) => return Ok(Some(start.safe)),
                Err(error) if error.severity() != EngineTaskErrorSeverity::Critical => {
                    warn!(target: "engine", ?error, "Startup forkchoice synchronization yielded; retrying");
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(error) => return Err(EngineError::Critical(error.to_string())),
            }
        }
    }

    async fn recover_from_execution_layer(&mut self) -> EngineResult<L2BlockInfo> {
        loop {
            let start = match find_starting_forkchoice(&self.config, self.client.as_ref()).await {
                Ok(start) => start,
                Err(error @ (SyncStartError::RpcError(_) | SyncStartError::BlockNotFound(_))) => {
                    warn!(target: "engine", ?error, "Startup forkchoice discovery yielded; retrying");
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                    continue;
                }
                Err(error) => return Err(EngineError::Critical(error.to_string())),
            };

            let result = SynchronizeTask::new(
                self.client.clone(),
                self.config.clone(),
                EngineSyncStateUpdate {
                    unsafe_head: Some(start.un_safe),
                    cross_unsafe_head: Some(start.un_safe),
                    local_safe_head: Some(start.safe),
                    safe_head: Some(start.safe),
                    finalized_head: Some(start.finalized),
                },
            )
            .execute(&mut self.state)
            .await;

            match result {
                Ok(()) => return Ok(start.safe),
                Err(error) if error.severity() != EngineTaskErrorSeverity::Critical => {
                    warn!(target: "engine", ?error, "Forkchoice synchronization yielded; retrying");
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(error) => return Err(EngineError::Critical(error.to_string())),
            }
        }
    }

    async fn handle_query(&self, query: EngineQueries) {
        let state_rx = self.state_tx.subscribe();
        let queue_length_rx = self.queue_length_tx.subscribe();
        if let Err(error) =
            query.handle(&state_rx, &queue_length_rx, &self.client, &self.config).await
        {
            warn!(target: "engine", ?error, "Failed to handle Engine RPC query");
        }
    }

    fn publish_state(&self) {
        self.state_tx.send_replace(self.state);
    }

    fn publish_queue_length(&self, active: usize) {
        self.queue_length_tx.send_replace(
            self.request_rx
                .len()
                .saturating_add(self.local_build_rx.len())
                .saturating_add(self.query_rx.len())
                .saturating_add(active),
        );
    }

    fn map_task_error<E>(error: E) -> EngineError
    where
        E: EngineTaskError + core::fmt::Display,
    {
        match error.severity() {
            EngineTaskErrorSeverity::Temporary => EngineError::Temporary(error.to_string()),
            EngineTaskErrorSeverity::Critical => EngineError::Critical(error.to_string()),
            EngineTaskErrorSeverity::Reset => {
                EngineError::ResetRequired { reason: error.to_string(), safe_head: None }
            }
            EngineTaskErrorSeverity::Flush => {
                EngineError::FlushRequired { reason: error.to_string(), safe_head: None }
            }
        }
    }
}

fn startup_should_stop(lifecycle: &mut mpsc::Receiver<EngineLifecycleCommand>) -> bool {
    loop {
        match lifecycle.try_recv() {
            Ok(EngineLifecycleCommand::Quiesce(response)) => {
                let _ = response.send(Ok(()));
            }
            Ok(EngineLifecycleCommand::Shutdown(response)) => {
                let _ = response.send(Ok(()));
                return true;
            }
            Err(mpsc::error::TryRecvError::Empty) => return false,
            Err(mpsc::error::TryRecvError::Disconnected) => return true,
        }
    }
}

/// Delay used before retrying transient Engine operations.
pub(crate) const ENGINE_RETRY_DELAY: Duration = Duration::from_millis(200);

#[cfg(test)]
mod tests {
    use super::*;
    use kona_engine::test_utils::test_engine_client_builder;

    fn core() -> EngineCore<kona_engine::test_utils::MockEngineClient> {
        let raw = Arc::new(test_engine_client_builder().build());
        EngineCore::new(raw, Arc::new(RollupConfig::default()), 8).0
    }

    #[test]
    fn engine_recovery_is_rejected_while_local_sequencing_is_active() {
        let mut core = core();
        core.local_sequencing_active = true;
        assert_eq!(core.ensure_recovery_allowed(), Err(EngineError::RecoveryWhileSequencing));
    }

    #[tokio::test]
    async fn startup_shutdown_is_acknowledged_between_engine_operations() {
        let (lifecycle_tx, mut lifecycle_rx) = mpsc::channel(1);
        let (response, result) = oneshot::channel();
        lifecycle_tx.send(EngineLifecycleCommand::Shutdown(response)).await.unwrap();

        assert!(startup_should_stop(&mut lifecycle_rx));
        assert_eq!(result.await.unwrap(), Ok(()));
    }

    #[tokio::test]
    async fn local_build_reservation_cannot_be_released_mid_build() {
        let mut core = core();
        let (response, result) = oneshot::channel();
        core.handle_request(EngineRequest::SetLocalSequencing { active: false, response }, true)
            .await;

        assert!(matches!(result.await.unwrap(), Err(EngineError::Critical(_))));
    }
}
