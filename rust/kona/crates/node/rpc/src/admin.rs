//! Admin RPC Module

use crate::{AdminApiServer, SequencerAdminAPIClient};
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadError;
use async_trait::async_trait;
use core::fmt::Debug;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};

/// The query types to the network actor for the admin api.
#[derive(Debug)]
pub enum NetworkAdminQuery {
    /// An admin rpc request to post an unsafe payload.
    PostUnsafePayload {
        /// The payload to post.
        payload: OpExecutionPayloadEnvelope,
    },
}

type NetworkAdminQuerySender = tokio::sync::mpsc::Sender<NetworkAdminQuery>;

/// The admin rpc server.
#[derive(Debug)]
pub struct AdminRpc<SequencerAdminAPIClient> {
    /// The sequencer admin API client.
    pub sequencer_admin_client: Option<SequencerAdminAPIClient>,
    /// The sender to the network actor.
    pub network_sender: NetworkAdminQuerySender,
}

impl<SequencerAdminAPIClient_> AdminRpc<SequencerAdminAPIClient_>
where
    SequencerAdminAPIClient_: SequencerAdminAPIClient,
{
    /// Constructs a new [`AdminRpc`] given the sequencer sender and network sender.
    ///
    /// # Parameters
    ///
    /// - `sequencer_sender`: The [`SequencerAdminAPIClient`] used to fulfill sequencer admin
    ///   queries.
    /// - `network_sender`: The sender to the network actor.
    ///
    /// # Returns
    ///
    /// A new [`AdminRpc`] instance.
    pub const fn new(
        sequencer_admin_client: Option<SequencerAdminAPIClient_>,
        network_sender: NetworkAdminQuerySender,
    ) -> Self {
        Self { sequencer_admin_client, network_sender }
    }
}

#[async_trait]
impl<SequencerAdminAPIClient_> AdminApiServer for AdminRpc<SequencerAdminAPIClient_>
where
    SequencerAdminAPIClient_: SequencerAdminAPIClient + 'static + Send + Sync,
{
    async fn admin_post_unsafe_payload(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> RpcResult<()> {
        payload.check_block_hash().map_err(|err| {
            let message = match &err {
                OpPayloadError::Eth(PayloadError::BlockHash { execution, consensus }) => {
                    format!(
                        "payload has bad block hash: {consensus}, actual block hash is: {execution}"
                    )
                }
                _ => format!("invalid payload: {err}"),
            };
            ErrorObject::owned(ErrorCode::InvalidParams.code(), message, None::<()>)
        })?;
        self.network_sender
            .send(NetworkAdminQuery::PostUnsafePayload { payload })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_sequencer_active(&self) -> RpcResult<bool> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .is_sequencer_active()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_start_sequencer(&self) -> RpcResult<()> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .start_sequencer()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_stop_sequencer(&self) -> RpcResult<B256> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .stop_sequencer()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_conductor_enabled(&self) -> RpcResult<bool> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .is_conductor_enabled()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_recover_mode(&self) -> RpcResult<bool> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .is_recovery_mode()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_set_recover_mode(&self, mode: bool) -> RpcResult<()> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .set_recovery_mode(mode)
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_override_leader(&self) -> RpcResult<()> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .override_leader()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_reset_derivation_pipeline(&self) -> RpcResult<()> {
        // If the sequencer is not enabled (mode runs in validator mode), return an error.
        let Some(ref sequencer_client) = self.sequencer_admin_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        sequencer_client
            .reset_derivation_pipeline()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::SequencerAdminAPIError;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use jsonrpsee::core::server::MethodsError;
    use op_alloy_consensus::OpBlock;
    use tokio::sync::mpsc;

    /// Admin payload submission must work without a sequencer.
    #[derive(Debug)]
    struct NoSequencer;

    #[async_trait]
    impl SequencerAdminAPIClient for NoSequencer {
        async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
            Ok(false)
        }

        async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
            unreachable!("validator has no sequencer")
        }

        async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
            unreachable!("validator has no sequencer")
        }

        async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
            unreachable!("validator has no sequencer")
        }

        async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
            unreachable!("validator has no sequencer")
        }

        async fn set_recovery_mode(&self, _mode: bool) -> Result<(), SequencerAdminAPIError> {
            unreachable!("validator has no sequencer")
        }

        async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
            unreachable!("validator has no sequencer")
        }

        async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
            unreachable!("validator has no sequencer")
        }
    }

    #[tokio::test]
    async fn unsafe_payload_validation_precedes_admission() {
        let (sender, mut receiver) = mpsc::channel(1);
        let rpc = AdminRpc::<NoSequencer>::new(None, sender).into_rpc();
        let mut payload = ExecutionPayloadV1::from_block_slow(&OpBlock::default());
        payload.block_hash = payload.clone().into_block_raw().unwrap().header.hash_slow();
        let valid = OpExecutionPayloadEnvelope::V1(payload.clone());
        let actual_hash = payload.block_hash;
        payload.block_hash = B256::repeat_byte(0xaa);
        let invalid_hash = payload.block_hash;

        let error = rpc
            .call::<_, ()>(
                "admin_postUnsafePayload",
                jsonrpsee::rpc_params![OpExecutionPayloadEnvelope::V1(payload)],
            )
            .await
            .unwrap_err();
        let MethodsError::JsonRpc(error) = error else {
            panic!("expected a JSON-RPC error, got {error}");
        };
        assert_eq!(error.code(), ErrorCode::InvalidParams.code());
        assert_eq!(
            error.message(),
            format!(
                "payload has bad block hash: {invalid_hash}, actual block hash is: {actual_hash}"
            )
        );
        assert!(matches!(receiver.try_recv(), Err(mpsc::error::TryRecvError::Empty)));

        rpc.call::<_, ()>("admin_postUnsafePayload", jsonrpsee::rpc_params![valid.clone()])
            .await
            .unwrap();
        let NetworkAdminQuery::PostUnsafePayload { payload } = receiver.try_recv().unwrap();
        assert_eq!(payload, valid);
    }
}
