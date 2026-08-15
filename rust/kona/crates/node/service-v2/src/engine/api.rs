//! Cloneable semantic client for the execution-engine service.

use crate::engine::{EngineError, EngineResult};
use alloy_eips::BlockNumberOrTag;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::{EngineQueries, EngineState, EngineSyncState};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent, OutputRoot};
use kona_rpc::EngineRpcClient;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::{mpsc, oneshot, watch};

/// A safe-chain update supplied by L1 derivation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SafeChainUpdate {
    /// Derived attributes that must be matched against or executed over the unsafe chain.
    Attributes(Box<OpAttributesWithParent>),
    /// An existing L2 block identified by delegated derivation.
    Block(L2BlockInfo),
}

impl From<OpAttributesWithParent> for SafeChainUpdate {
    fn from(value: OpAttributesWithParent) -> Self {
        Self::Attributes(Box::new(value))
    }
}

impl From<L2BlockInfo> for SafeChainUpdate {
    fn from(value: L2BlockInfo) -> Self {
        Self::Block(value)
    }
}

/// A locally built payload that has not been published or made canonical.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BuiltUnsafePayload {
    payload: OpExecutionPayloadEnvelope,
    parent: L2BlockInfo,
}

impl BuiltUnsafePayload {
    pub(super) const fn new(payload: OpExecutionPayloadEnvelope, parent: L2BlockInfo) -> Self {
        Self { payload, parent }
    }

    #[cfg(test)]
    pub(crate) const fn test_new(payload: OpExecutionPayloadEnvelope, parent: L2BlockInfo) -> Self {
        Self::new(payload, parent)
    }

    /// Returns the complete payload to authorize and publish.
    pub const fn payload(&self) -> &OpExecutionPayloadEnvelope {
        &self.payload
    }

    pub(super) const fn parent(&self) -> L2BlockInfo {
        self.parent
    }
}

/// A cloneable capability for semantic engine operations.
///
/// Raw Engine API methods and payload identifiers are deliberately absent. Every request is
/// executed by the single [`super::EngineService`] task.
#[derive(Debug, Clone)]
pub struct EngineClient {
    request_tx: mpsc::Sender<EngineRequest>,
    local_build_tx: mpsc::Sender<EngineRequest>,
    query_tx: mpsc::Sender<EngineRequest>,
    readiness: watch::Receiver<bool>,
}

impl EngineClient {
    pub(super) const fn new(
        request_tx: mpsc::Sender<EngineRequest>,
        local_build_tx: mpsc::Sender<EngineRequest>,
        query_tx: mpsc::Sender<EngineRequest>,
        readiness: watch::Receiver<bool>,
    ) -> Self {
        Self { request_tx, local_build_tx, query_tx, readiness }
    }

    #[cfg(test)]
    pub(crate) fn test_pair(capacity: usize) -> (Self, mpsc::Receiver<EngineRequest>) {
        let (request_tx, request_rx) = mpsc::channel(capacity);
        let (_, readiness) = watch::channel(true);
        (
            Self {
                request_tx: request_tx.clone(),
                local_build_tx: request_tx.clone(),
                query_tx: request_tx,
                readiness,
            },
            request_rx,
        )
    }

    pub(crate) async fn set_local_sequencing_active(&self, active: bool) -> EngineResult<()> {
        self.local_request(|response| EngineRequest::SetLocalSequencing { active, response }).await
    }

    /// Waits until startup forkchoice reconciliation has completed.
    pub async fn wait_ready(&self) -> EngineResult<()> {
        let mut readiness = self.readiness.clone();
        loop {
            if *readiness.borrow() {
                return Ok(());
            }
            readiness.changed().await.map_err(|_| EngineError::Unavailable)?;
        }
    }

    /// Starts and retrieves a locally constructed unsafe payload without importing it.
    pub async fn build_unsafe(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineResult<BuiltUnsafePayload> {
        self.local_request(|response| EngineRequest::BuildUnsafe {
            attributes: Box::new(attributes),
            response,
        })
        .await
    }

    /// Imports a locally built payload only after its publication gate has completed.
    pub async fn canonicalize_unsafe(
        &self,
        candidate: BuiltUnsafePayload,
    ) -> EngineResult<L2BlockInfo> {
        self.local_request(|response| EngineRequest::CanonicalizeUnsafe {
            candidate: Box::new(candidate),
            response,
        })
        .await
    }

    /// Imports a complete payload and advances the unsafe head.
    pub async fn import_unsafe(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        self.request(|response| EngineRequest::ImportUnsafe {
            payload: Box::new(payload),
            response,
        })
        .await
    }

    /// Reconciles a derived safe-chain update with the current unsafe chain.
    pub async fn update_safe(
        &self,
        update: impl Into<SafeChainUpdate>,
    ) -> EngineResult<L2BlockInfo> {
        self.request(|response| EngineRequest::UpdateSafe { update: update.into(), response }).await
    }

    /// Advances finality to an existing safe block.
    pub async fn update_finalized(&self, block: L2BlockInfo) -> EngineResult<()> {
        self.request(|response| EngineRequest::UpdateFinalized { block, response }).await
    }

    /// Reconciles the engine with trusted L2 labels discovered from the execution layer.
    ///
    /// This is destructive to unsafe state and must be prohibited by the caller while local
    /// sequencing is active.
    pub async fn recover(&self) -> EngineResult<L2BlockInfo> {
        self.request(|response| EngineRequest::Recover { response }).await
    }

    /// Returns the current synchronization state.
    pub async fn state(&self) -> EngineResult<EngineSyncState> {
        self.query_request(|response| EngineRequest::State { response }).await
    }

    async fn request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        Self::send_request(&self.request_tx, request).await
    }

    async fn local_request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        Self::send_request(&self.local_build_tx, request).await
    }

    async fn query_request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        Self::send_request(&self.query_tx, request).await
    }

    async fn send_request<T>(
        sender: &mpsc::Sender<EngineRequest>,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        let (response, result) = oneshot::channel();
        sender.send(request(response)).await.map_err(|_| EngineError::Unavailable)?;
        result.await.map_err(|_| EngineError::ResponseDropped)?
    }

    async fn query(&self, query: EngineQueries) -> RpcResult<()> {
        self.query_tx
            .send(EngineRequest::Query(Box::new(query)))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[async_trait]
impl EngineRpcClient for EngineClient {
    async fn get_config(&self) -> RpcResult<RollupConfig> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::Config(response)).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn get_state(&self) -> RpcResult<EngineState> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::State(response)).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn output_at_block(
        &self,
        block: BlockNumberOrTag,
    ) -> RpcResult<(L2BlockInfo, OutputRoot, EngineState)> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::OutputAtBlock { block, sender: response }).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn dev_get_task_queue_length(&self) -> RpcResult<usize> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::TaskQueueLength(response)).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn dev_subscribe_to_engine_queue_length(&self) -> RpcResult<watch::Receiver<usize>> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::QueueLengthReceiver(response)).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn dev_subscribe_to_engine_state(&self) -> RpcResult<watch::Receiver<EngineState>> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::StateReceiver(response)).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[derive(Debug)]
pub(crate) enum EngineRequest {
    BuildUnsafe {
        attributes: Box<OpAttributesWithParent>,
        response: oneshot::Sender<EngineResult<BuiltUnsafePayload>>,
    },
    CanonicalizeUnsafe {
        candidate: Box<BuiltUnsafePayload>,
        response: oneshot::Sender<EngineResult<L2BlockInfo>>,
    },
    ImportUnsafe {
        payload: Box<OpExecutionPayloadEnvelope>,
        response: oneshot::Sender<EngineResult<L2BlockInfo>>,
    },
    UpdateSafe {
        update: SafeChainUpdate,
        response: oneshot::Sender<EngineResult<L2BlockInfo>>,
    },
    UpdateFinalized {
        block: L2BlockInfo,
        response: oneshot::Sender<EngineResult<()>>,
    },
    Recover {
        response: oneshot::Sender<EngineResult<L2BlockInfo>>,
    },
    SetLocalSequencing {
        active: bool,
        response: oneshot::Sender<EngineResult<()>>,
    },
    State {
        response: oneshot::Sender<EngineResult<EngineSyncState>>,
    },
    Query(Box<EngineQueries>),
}
