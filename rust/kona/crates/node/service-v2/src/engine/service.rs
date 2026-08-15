//! Long-running execution-engine service.

use crate::engine::{
    BuiltUnsafePayload, EngineClient as SemanticEngineClient, EngineError, EngineResult,
    EngineServiceError, SafeChainUpdate, api::EngineRequest,
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
    sync::Arc,
    time::{Duration, Instant},
};
use tokio::sync::{mpsc, watch};
use tokio_util::sync::CancellationToken;

/// Default number of semantic operations that may wait for the engine.
pub const DEFAULT_ENGINE_REQUEST_CAPACITY: usize = 64;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum LocalBuildTransition {
    Unchanged,
    Started,
    Finished,
}

/// The sole owner of raw Engine API calls and authoritative forkchoice state.
#[derive(Debug)]
pub struct EngineService<Client> {
    client: Arc<Client>,
    config: Arc<RollupConfig>,
    state: EngineState,
    state_tx: watch::Sender<EngineState>,
    queue_length_tx: watch::Sender<usize>,
    readiness_tx: watch::Sender<bool>,
    request_rx: mpsc::Receiver<EngineRequest>,
    local_build_rx: mpsc::Receiver<EngineRequest>,
    query_rx: mpsc::Receiver<EngineRequest>,
    local_sequencing_active: bool,
}

impl<Client> EngineService<Client>
where
    Client: RawEngineClient + 'static,
{
    /// Creates an engine service and its cloneable semantic client.
    pub fn new(client: Arc<Client>, config: Arc<RollupConfig>) -> (Self, SemanticEngineClient) {
        Self::with_capacity(client, config, DEFAULT_ENGINE_REQUEST_CAPACITY)
    }

    /// Creates an engine service with bounded request capacity.
    pub fn with_capacity(
        client: Arc<Client>,
        config: Arc<RollupConfig>,
        capacity: usize,
    ) -> (Self, SemanticEngineClient) {
        let (request_tx, request_rx) = mpsc::channel(capacity);
        let (local_build_tx, local_build_rx) = mpsc::channel(capacity);
        let (query_tx, query_rx) = mpsc::channel(capacity);
        let (state_tx, _) = watch::channel(EngineState::default());
        let (queue_length_tx, _) = watch::channel(0);
        let (readiness_tx, readiness_rx) = watch::channel(false);
        (
            Self {
                client,
                config,
                state: EngineState::default(),
                state_tx,
                queue_length_tx,
                readiness_tx,
                request_rx,
                local_build_rx,
                query_rx,
                local_sequencing_active: false,
            },
            SemanticEngineClient::new(request_tx, local_build_tx, query_tx, readiness_rx),
        )
    }

    /// Initializes forkchoice and then serves semantic operations until shutdown.
    ///
    /// Cancellation is observed only between operations. Once a request starts, it is allowed to
    /// finish so a remote Engine API side effect is never abandoned ambiguously.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), EngineServiceError> {
        self.recover_from_execution_layer()
            .await
            .map_err(|error| EngineServiceError::Startup(error.to_string()))?;
        self.publish_state();
        self.readiness_tx.send_replace(true);

        let mut local_build_active = false;
        loop {
            let request = if local_build_active {
                tokio::select! {
                    biased;
                    _ = shutdown.cancelled() => return Ok(()),
                    request = self.local_build_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed)?
                    }
                    request = self.query_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed)?
                    }
                }
            } else {
                tokio::select! {
                    biased;
                    _ = shutdown.cancelled() => return Ok(()),
                    request = self.local_build_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed)?
                    }
                    request = self.request_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed)?
                    }
                    request = self.query_rx.recv() => {
                        request.ok_or(EngineServiceError::RequestChannelClosed)?
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
                let result = self.update_safe(update).await;
                let _ = response.send(result);
                LocalBuildTransition::Unchanged
            }
            EngineRequest::UpdateFinalized { block, response } => {
                let result = self.update_finalized(block).await;
                let _ = response.send(result);
                LocalBuildTransition::Unchanged
            }
            EngineRequest::Recover { response } => {
                let result = if self.local_sequencing_active {
                    Err(EngineError::RecoveryWhileSequencing)
                } else {
                    self.recover_from_execution_layer().await
                };
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
        let build_started = Instant::now();
        let payload_id =
            BuildTask::new(self.client.clone(), self.config.clone(), attributes.clone(), None)
                .execute(&mut self.state)
                .await
                .map_err(Self::map_task_error);
        kona_macros::set!(
            gauge,
            crate::Metrics::SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION,
            build_started.elapsed()
        );
        let payload_id = payload_id?;

        if self.state.sync_state.unsafe_head() != attributes.parent {
            return Err(EngineError::StaleBuild);
        }

        let seal_started = Instant::now();
        let payload = self.fetch_payload(payload_id, &attributes).await;
        kona_macros::set!(
            gauge,
            crate::Metrics::SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION,
            seal_started.elapsed()
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

    async fn update_safe(&mut self, update: SafeChainUpdate) -> EngineResult<L2BlockInfo> {
        let input = match update {
            SafeChainUpdate::Attributes(attributes) => ConsolidateInput::Attributes(attributes),
            SafeChainUpdate::Block(block) => ConsolidateInput::BlockInfo(block),
        };

        ConsolidateTask::new(self.client.clone(), self.config.clone(), input)
            .execute(&mut self.state)
            .await
            .map_err(Self::map_task_error)?;

        Ok(self.state.sync_state.safe_head())
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
                    warn!(target: "engine", ?error, "Startup forkchoice synchronization yielded; retrying");
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
            warn!(target: "engine", ?error, "Failed to handle engine RPC query");
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
            EngineTaskErrorSeverity::Reset => EngineError::ResetRequired(error.to_string()),
            EngineTaskErrorSeverity::Flush => EngineError::FlushRequired(error.to_string()),
        }
    }
}

/// Delay used by workflows before retrying transient engine operations.
pub(crate) const ENGINE_RETRY_DELAY: Duration = Duration::from_millis(200);

#[cfg(test)]
mod tests {
    use super::*;
    use kona_engine::test_utils::test_engine_client_builder;
    use tokio::sync::oneshot;

    fn service() -> EngineService<kona_engine::test_utils::MockEngineClient> {
        let raw = Arc::new(test_engine_client_builder().build());
        EngineService::new(raw, Arc::new(RollupConfig::default())).0
    }

    #[tokio::test]
    async fn engine_recovery_is_rejected_while_local_sequencing_is_active() {
        let mut service = service();
        let (response, result) = oneshot::channel();
        service
            .handle_request(EngineRequest::SetLocalSequencing { active: true, response }, false)
            .await;
        result.await.unwrap().unwrap();

        let (response, result) = oneshot::channel();
        service.handle_request(EngineRequest::Recover { response }, false).await;
        assert_eq!(result.await.unwrap(), Err(EngineError::RecoveryWhileSequencing));
    }

    #[tokio::test]
    async fn local_build_reservation_cannot_be_released_mid_build() {
        let mut service = service();
        let (response, result) = oneshot::channel();
        service
            .handle_request(EngineRequest::SetLocalSequencing { active: false, response }, true)
            .await;

        assert!(matches!(result.await.unwrap(), Err(EngineError::Critical(_))));
    }
}
