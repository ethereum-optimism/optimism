//! Narrow capabilities for the execution-engine service.

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
use tokio::sync::{mpsc, oneshot};

/// A safe-chain update supplied by Derivation.
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

/// The complete semantic Engine capability available to Derivation.
///
/// Unsafe construction, payload import, state queries, readiness, and recovery are deliberately
/// absent. Engine publishes this handle only after startup forkchoice synchronization succeeds.
#[derive(Debug, Clone)]
pub struct EngineHandle {
    request_tx: mpsc::Sender<EngineRequest>,
}

impl EngineHandle {
    pub(super) const fn new(request_tx: mpsc::Sender<EngineRequest>) -> Self {
        Self { request_tx }
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

    async fn request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        send_request(&self.request_tx, request).await
    }

    #[cfg(test)]
    pub(crate) fn test_pair(capacity: usize) -> (Self, mpsc::Receiver<EngineRequest>) {
        let (request_tx, request_rx) = mpsc::channel(capacity);
        (Self { request_tx }, request_rx)
    }
}

/// Read-only Engine capability supplied only to RPC routing.
#[derive(Debug, Clone)]
pub struct EngineRpcAdapter {
    query_tx: mpsc::Sender<EngineRequest>,
}

impl EngineRpcAdapter {
    pub(super) const fn new(query_tx: mpsc::Sender<EngineRequest>) -> Self {
        Self { query_tx }
    }

    async fn query(&self, query: EngineQueries) -> RpcResult<()> {
        self.query_tx
            .send(EngineRequest::Query(Box::new(query)))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[async_trait]
impl EngineRpcClient for EngineRpcAdapter {
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

    async fn dev_subscribe_to_engine_queue_length(
        &self,
    ) -> RpcResult<tokio::sync::watch::Receiver<usize>> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::QueueLengthReceiver(response)).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn dev_subscribe_to_engine_state(
        &self,
    ) -> RpcResult<tokio::sync::watch::Receiver<EngineState>> {
        let (response, result) = oneshot::channel();
        self.query(EngineQueries::StateReceiver(response)).await?;
        result.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

/// A locally built payload retained privately by Engine until publication is complete.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct BuiltUnsafePayload {
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

    pub(super) const fn payload(&self) -> &OpExecutionPayloadEnvelope {
        &self.payload
    }

    pub(super) const fn parent(&self) -> L2BlockInfo {
        self.parent
    }
}

/// Private unsafe-processing capability shared only with Engine-owned workflows.
#[derive(Debug, Clone)]
pub(super) struct EngineInternalHandle {
    request_tx: mpsc::Sender<EngineRequest>,
    local_build_tx: mpsc::Sender<EngineRequest>,
}

impl EngineInternalHandle {
    pub(super) const fn new(
        request_tx: mpsc::Sender<EngineRequest>,
        local_build_tx: mpsc::Sender<EngineRequest>,
    ) -> Self {
        Self { request_tx, local_build_tx }
    }

    pub(super) async fn set_local_sequencing_active(&self, active: bool) -> EngineResult<()> {
        self.local_request(|response| EngineRequest::SetLocalSequencing { active, response }).await
    }

    pub(super) async fn build_unsafe(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineResult<BuiltUnsafePayload> {
        self.local_request(|response| EngineRequest::BuildUnsafe {
            attributes: Box::new(attributes),
            response,
        })
        .await
    }

    pub(super) async fn canonicalize_unsafe(
        &self,
        candidate: BuiltUnsafePayload,
    ) -> EngineResult<L2BlockInfo> {
        self.local_request(|response| EngineRequest::CanonicalizeUnsafe {
            candidate: Box::new(candidate),
            response,
        })
        .await
    }

    pub(super) async fn import_unsafe(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        self.request(|response| EngineRequest::ImportUnsafe {
            payload: Box::new(payload),
            response,
        })
        .await
    }

    pub(super) async fn state(&self) -> EngineResult<EngineSyncState> {
        self.request(|response| EngineRequest::State { response }).await
    }

    async fn request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        send_request(&self.request_tx, request).await
    }

    async fn local_request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        send_request(&self.local_build_tx, request).await
    }

    #[cfg(test)]
    pub(crate) fn test_pair(capacity: usize) -> (Self, mpsc::Receiver<EngineRequest>) {
        let (request_tx, request_rx) = mpsc::channel(capacity);
        (Self { request_tx: request_tx.clone(), local_build_tx: request_tx }, request_rx)
    }
}

async fn send_request<T>(
    sender: &mpsc::Sender<EngineRequest>,
    request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
) -> EngineResult<T> {
    let (response, result) = oneshot::channel();
    sender.send(request(response)).await.map_err(|_| EngineError::Unavailable)?;
    result.await.map_err(|_| EngineError::ResponseDropped)?
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
    SetLocalSequencing {
        active: bool,
        response: oneshot::Sender<EngineResult<()>>,
    },
    State {
        response: oneshot::Sender<EngineResult<EngineSyncState>>,
    },
    Query(Box<EngineQueries>),
}
