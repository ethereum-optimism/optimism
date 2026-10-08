//! Admin RPC Module

use crate::AdminApiServer;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadError;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::{EngineActorRequest, ResetRequest};
use kona_node_actors::sequencer::Handle;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};
use tokio::sync::mpsc;

/// The admin RPC server.
#[derive(Debug)]
pub struct AdminRpc {
    sequencer: Option<Handle>,
    engine: mpsc::Sender<EngineActorRequest>,
    unsafe_payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

impl AdminRpc {
    /// Construct the admin RPC server from the sequencer, engine, and payload handles.
    pub const fn new(
        sequencer: Option<Handle>,
        engine: mpsc::Sender<EngineActorRequest>,
        unsafe_payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
    ) -> Self {
        Self { sequencer, engine, unsafe_payloads }
    }

    fn sequencer(&self) -> RpcResult<&Handle> {
        self.sequencer.as_ref().ok_or_else(|| ErrorObject::from(ErrorCode::MethodNotFound))
    }
}

#[async_trait]
impl AdminApiServer for AdminRpc {
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
        self.unsafe_payloads
            .send(payload)
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_sequencer_active(&self) -> RpcResult<bool> {
        Ok(self
            .sequencer()?
            .snapshot()
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?
            .active)
    }

    async fn admin_start_sequencer(&self) -> RpcResult<()> {
        self.sequencer()?.start().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_stop_sequencer(&self) -> RpcResult<B256> {
        self.sequencer()?.stop().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_conductor_enabled(&self) -> RpcResult<bool> {
        Ok(self
            .sequencer()?
            .snapshot()
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?
            .conductor_enabled)
    }

    async fn admin_recover_mode(&self) -> RpcResult<bool> {
        Ok(self
            .sequencer()?
            .snapshot()
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?
            .recovery_mode)
    }

    async fn admin_set_recover_mode(&self, mode: bool) -> RpcResult<()> {
        self.sequencer()?
            .set_recovery_mode(mode)
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_override_leader(&self) -> RpcResult<()> {
        self.sequencer()?
            .override_leader()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_reset_derivation_pipeline(&self) -> RpcResult<()> {
        let (result_tx, mut result_rx) = mpsc::channel(1);
        self.engine
            .send(EngineActorRequest::Reset(Box::new(ResetRequest { result_tx })))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        result_rx
            .recv()
            .await
            .ok_or_else(|| ErrorObject::from(ErrorCode::InternalError))?
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use kona_node_actors::sequencer::{
        SequencerAdminAPIError, SequencerAdminCommand, SequencerState,
    };
    use tokio::sync::{oneshot, watch};

    #[tokio::test]
    async fn reads_published_state_without_queueing_and_rejects_closed_publisher() {
        let initial =
            SequencerState { active: false, conductor_enabled: true, recovery_mode: false };
        let (state_tx, state_rx) = watch::channel(initial);
        let (commands_tx, _commands_rx) = mpsc::channel(1);
        let (reply, _) = oneshot::channel();
        commands_tx.send(SequencerAdminCommand::StartSequencer(reply)).await.unwrap();
        let (payloads_tx, _) = mpsc::channel(1);
        let rpc = AdminRpc::new(
            Some(Handle::new(state_rx, commands_tx)),
            mpsc::channel(1).0,
            payloads_tx,
        );
        assert!(!rpc.admin_sequencer_active().await.unwrap());
        assert!(rpc.admin_conductor_enabled().await.unwrap());
        assert!(!rpc.admin_recover_mode().await.unwrap());

        let updated = SequencerState { active: true, recovery_mode: true, ..initial };
        state_tx.send_replace(updated);
        assert!(rpc.admin_sequencer_active().await.unwrap());
        assert!(rpc.admin_recover_mode().await.unwrap());

        // Closing with an unread final update must not serve stale status.
        drop(state_tx);
        for result in [
            rpc.admin_sequencer_active().await,
            rpc.admin_conductor_enabled().await,
            rpc.admin_recover_mode().await,
        ] {
            assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());
        }
    }

    #[tokio::test]
    async fn sequencer_methods_are_unavailable_on_validators() {
        let (tx, _) = mpsc::channel(1);
        let rpc = AdminRpc::new(None, mpsc::channel(1).0, tx);
        for result in [
            rpc.admin_sequencer_active().await.map(|_| ()),
            rpc.admin_conductor_enabled().await.map(|_| ()),
            rpc.admin_recover_mode().await.map(|_| ()),
            rpc.admin_start_sequencer().await,
            rpc.admin_stop_sequencer().await.map(|_| ()),
            rpc.admin_set_recover_mode(true).await,
            rpc.admin_override_leader().await,
        ] {
            assert_eq!(result.unwrap_err().code(), ErrorCode::MethodNotFound.code());
        }
    }

    #[tokio::test]
    async fn commands_return_actor_results_and_map_failures() {
        let (_state_tx, state_rx) = watch::channel(SequencerState {
            active: true,
            conductor_enabled: false,
            recovery_mode: false,
        });
        let (commands_tx, mut commands_rx) = mpsc::channel(1);
        let (payloads_tx, _) = mpsc::channel(1);
        let rpc = AdminRpc::new(
            Some(Handle::new(state_rx, commands_tx)),
            mpsc::channel(1).0,
            payloads_tx,
        );
        let hash = B256::repeat_byte(42);
        let (result, ()) = tokio::join!(rpc.admin_stop_sequencer(), async {
            let SequencerAdminCommand::StopSequencer(reply) = commands_rx.recv().await.unwrap()
            else {
                panic!("expected stop");
            };
            reply.send(Ok(hash)).unwrap();
        });
        assert_eq!(result.unwrap(), hash);

        let (result, ()) = tokio::join!(rpc.admin_override_leader(), async {
            let SequencerAdminCommand::OverrideLeader(reply) = commands_rx.recv().await.unwrap()
            else {
                panic!("expected leader override");
            };
            reply
                .send(Err(SequencerAdminAPIError::LeaderOverrideError("unavailable".to_owned())))
                .unwrap();
        });
        assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());

        let (result, ()) = tokio::join!(rpc.admin_start_sequencer(), async {
            drop(commands_rx.recv().await.unwrap());
        });
        assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());
        drop(commands_rx);
        assert_eq!(
            rpc.admin_start_sequencer().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }

    #[tokio::test]
    async fn resets_engine_on_sequencers_and_validators_and_waits_for_acknowledgement() {
        for is_sequencer in [true, false] {
            let (_state_tx, state_rx) = watch::channel(SequencerState {
                active: true,
                conductor_enabled: false,
                recovery_mode: false,
            });
            let (commands_tx, mut commands_rx) = mpsc::channel(1);
            // Reset must bypass even a full sequencer command queue.
            let (reply, _) = oneshot::channel();
            commands_tx.send(SequencerAdminCommand::StartSequencer(reply)).await.unwrap();
            let sequencer = is_sequencer.then(|| Handle::new(state_rx, commands_tx));
            let (engine_tx, mut engine_rx) = mpsc::channel(1);
            let rpc = AdminRpc::new(sequencer, engine_tx, mpsc::channel(1).0).into_rpc();
            let reset =
                rpc.call::<_, ()>("admin_resetDerivationPipeline", jsonrpsee::rpc_params![]);
            tokio::pin!(reset);
            let request = tokio::select! {
                result = &mut reset => panic!("reset returned before engine acknowledgement: {result:?}"),
                request = engine_rx.recv() => request.unwrap(),
            };
            let EngineActorRequest::Reset(request) = request else {
                panic!("expected engine reset");
            };
            assert!(matches!(
                commands_rx.try_recv().unwrap(),
                SequencerAdminCommand::StartSequencer(_)
            ));
            assert!(matches!(
                commands_rx.try_recv(),
                Err(mpsc::error::TryRecvError::Empty | mpsc::error::TryRecvError::Disconnected)
            ));
            // Receiving the request is insufficient: the engine must acknowledge it.
            assert!(
                tokio::time::timeout(std::time::Duration::from_millis(10), &mut reset)
                    .await
                    .is_err()
            );
            request.result_tx.send(Ok(())).await.unwrap();
            reset.await.unwrap();
        }
    }

    #[tokio::test]
    async fn maps_engine_reset_failures_and_closed_channels() {
        let (engine_tx, mut engine_rx) = mpsc::channel(1);
        let rpc = AdminRpc::new(None, engine_tx, mpsc::channel(1).0);
        let (result, ()) = tokio::join!(rpc.admin_reset_derivation_pipeline(), async {
            let EngineActorRequest::Reset(request) = engine_rx.recv().await.unwrap() else {
                panic!("expected engine reset");
            };
            request
                .result_tx
                .send(Err(kona_engine::EngineRequestError::ResetForkchoiceError(
                    "reset failed".to_owned(),
                )))
                .await
                .unwrap();
        });
        assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());
        let (result, ()) = tokio::join!(rpc.admin_reset_derivation_pipeline(), async {
            drop(engine_rx.recv().await.unwrap());
        });
        assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());
        drop(engine_rx);
        assert_eq!(
            rpc.admin_reset_derivation_pipeline().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }

    #[tokio::test]
    async fn validates_payloads_before_forwarding_on_validators() {
        let (tx, mut rx) = mpsc::channel(1);
        let rpc = AdminRpc::new(None, mpsc::channel(1).0, tx);
        let mut payload = ExecutionPayloadV1 {
            parent_hash: Default::default(),
            fee_recipient: Default::default(),
            state_root: Default::default(),
            receipts_root: Default::default(),
            logs_bloom: Default::default(),
            prev_randao: Default::default(),
            block_number: 1,
            gas_limit: 30_000_000,
            gas_used: 0,
            timestamp: 1,
            extra_data: Default::default(),
            base_fee_per_gas: Default::default(),
            block_hash: Default::default(),
            transactions: vec![],
        };
        let error = rpc
            .admin_post_unsafe_payload(OpExecutionPayloadEnvelope::V1(payload.clone()))
            .await
            .unwrap_err();
        assert_eq!(error.code(), ErrorCode::InvalidParams.code());
        assert!(error.message().starts_with("payload has bad block hash:"));
        assert!(matches!(rx.try_recv(), Err(mpsc::error::TryRecvError::Empty)));

        payload.block_hash = payload.clone().into_block_raw().unwrap().header.hash_slow();
        let envelope = OpExecutionPayloadEnvelope::V1(payload);
        rpc.admin_post_unsafe_payload(envelope.clone()).await.unwrap();
        assert_eq!(rx.recv().await.unwrap(), envelope);
        drop(rx);
        assert_eq!(
            rpc.admin_post_unsafe_payload(envelope).await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }
}
