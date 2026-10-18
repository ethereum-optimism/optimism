//! Admin operations registered only when the node has a sequencer.

use crate::{SequencerAdminAPIClient, SequencerAdminApiServer};
use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};

/// Sequencer admin handler with a required sequencer capability.
#[derive(Debug)]
pub struct SequencerAdminRpc<S> {
    sequencer: S,
}

impl<S: SequencerAdminAPIClient> SequencerAdminRpc<S> {
    /// Constructs the handler for an enabled sequencer.
    pub const fn new(sequencer: S) -> Self {
        Self { sequencer }
    }
}

#[async_trait]
impl<S: SequencerAdminAPIClient + 'static> SequencerAdminApiServer for SequencerAdminRpc<S> {
    async fn admin_sequencer_active(&self) -> RpcResult<bool> {
        self.sequencer
            .is_sequencer_active()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_start_sequencer(&self) -> RpcResult<()> {
        self.sequencer
            .start_sequencer()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_stop_sequencer(&self) -> RpcResult<B256> {
        self.sequencer
            .stop_sequencer()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_conductor_enabled(&self) -> RpcResult<bool> {
        self.sequencer
            .is_conductor_enabled()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_recover_mode(&self) -> RpcResult<bool> {
        self.sequencer
            .is_recovery_mode()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_set_recover_mode(&self, mode: bool) -> RpcResult<()> {
        self.sequencer
            .set_recovery_mode(mode)
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_override_leader(&self) -> RpcResult<()> {
        self.sequencer
            .override_leader()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}
