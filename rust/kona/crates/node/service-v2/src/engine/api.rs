//! Cloneable semantic engine client.

use crate::engine::{EngineError, EngineResult};
use kona_engine::EngineSyncState;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::{mpsc, oneshot};

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

/// A cloneable client for the semantic engine service.
///
/// This client deliberately does not expose raw Engine API methods, payload IDs, or forkchoice
/// mutation. All operations are serialized by the engine service task.
#[derive(Debug, Clone)]
pub struct EngineClient {
    request_tx: mpsc::Sender<EngineRequest>,
}

impl EngineClient {
    pub(super) const fn new(request_tx: mpsc::Sender<EngineRequest>) -> Self {
        Self { request_tx }
    }

    /// Builds and retrieves an unsafe payload without making it canonical.
    pub async fn build_unsafe(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope> {
        self.request(|response| EngineRequest::BuildUnsafe {
            attributes: Box::new(attributes),
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

    /// Returns the engine service's current synchronization state.
    pub async fn state(&self) -> EngineResult<EngineSyncState> {
        self.request(|response| EngineRequest::State { response }).await
    }

    async fn request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<EngineResult<T>>) -> EngineRequest,
    ) -> EngineResult<T> {
        let (response, result) = oneshot::channel();
        self.request_tx.send(request(response)).await.map_err(|_| EngineError::Unavailable)?;
        result.await.map_err(|_| EngineError::ResponseDropped)?
    }
}

#[derive(Debug)]
pub(super) enum EngineRequest {
    BuildUnsafe {
        attributes: Box<OpAttributesWithParent>,
        response: oneshot::Sender<EngineResult<OpExecutionPayloadEnvelope>>,
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
    State {
        response: oneshot::Sender<EngineResult<EngineSyncState>>,
    },
}
