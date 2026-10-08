//! Admin RPC Module

use crate::AdminApiServer;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadError;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

/// Sequencer state shared by the actor and admin RPC readers.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SequencerState {
    /// Whether the sequencer is active.
    pub active: bool,
    /// Whether a conductor is configured.
    pub conductor_enabled: bool,
    /// Whether the sequencer is in recovery mode.
    pub recovery_mode: bool,
}

/// State-changing admin commands executed by the sequencer actor.
#[derive(Debug)]
pub enum SequencerAdminCommand {
    /// Start sequencing.
    StartSequencer(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Stop sequencing and return the current unsafe block hash.
    StopSequencer(oneshot::Sender<Result<B256, SequencerAdminAPIError>>),
    /// Set recovery mode.
    SetRecoveryMode(bool, oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Override the conductor leader.
    OverrideLeader(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Reset the derivation pipeline.
    ResetDerivationPipeline(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

/// Published sequencer state and its admin command queue.
#[derive(Debug, Clone)]
pub struct SequencerAdminHandle {
    state: watch::Receiver<SequencerState>,
    commands: mpsc::Sender<SequencerAdminCommand>,
}

impl SequencerAdminHandle {
    /// Construct a handle from the sequencer's published state and command sender.
    pub const fn new(
        state: watch::Receiver<SequencerState>,
        commands: mpsc::Sender<SequencerAdminCommand>,
    ) -> Self {
        Self { state, commands }
    }

    fn snapshot(&self) -> RpcResult<SequencerState> {
        // Fail if the actor exited, even if its final update has not been read yet.
        self.state.has_changed().map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        Ok(*self.state.borrow())
    }
}

/// Errors that can occur when using the sequencer admin API.
#[derive(Debug, Error)]
pub enum SequencerAdminAPIError {
    /// Error sending request.
    #[error("Error sending request: {0}.")]
    RequestError(String),

    /// Sequencer stopped successfully, followed by some error.
    #[error("Sequencer stopped successfully, followed by error: {0}.")]
    ErrorAfterSequencerWasStopped(String),

    /// Error overriding leader.
    #[error("Error overriding leader: {0}.")]
    LeaderOverrideError(String),
}

/// The admin RPC server.
#[derive(Debug)]
pub struct AdminRpc {
    sequencer: Option<SequencerAdminHandle>,
    unsafe_payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

impl AdminRpc {
    /// Construct the admin RPC server from an optional sequencer and a payload sender.
    pub const fn new(
        sequencer: Option<SequencerAdminHandle>,
        unsafe_payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
    ) -> Self {
        Self { sequencer, unsafe_payloads }
    }

    fn sequencer(&self) -> RpcResult<&SequencerAdminHandle> {
        self.sequencer.as_ref().ok_or_else(|| ErrorObject::from(ErrorCode::MethodNotFound))
    }

    async fn command<T>(
        &self,
        command: impl FnOnce(
            oneshot::Sender<Result<T, SequencerAdminAPIError>>,
        ) -> SequencerAdminCommand,
    ) -> RpcResult<T> {
        let sequencer = self.sequencer()?;
        let (tx, rx) = oneshot::channel();
        sequencer
            .commands
            .send(command(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        rx.await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
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
        Ok(self.sequencer()?.snapshot()?.active)
    }

    async fn admin_start_sequencer(&self) -> RpcResult<()> {
        self.command(SequencerAdminCommand::StartSequencer).await
    }

    async fn admin_stop_sequencer(&self) -> RpcResult<B256> {
        self.command(SequencerAdminCommand::StopSequencer).await
    }

    async fn admin_conductor_enabled(&self) -> RpcResult<bool> {
        Ok(self.sequencer()?.snapshot()?.conductor_enabled)
    }

    async fn admin_recover_mode(&self) -> RpcResult<bool> {
        Ok(self.sequencer()?.snapshot()?.recovery_mode)
    }

    async fn admin_set_recover_mode(&self, mode: bool) -> RpcResult<()> {
        self.command(|tx| SequencerAdminCommand::SetRecoveryMode(mode, tx)).await
    }

    async fn admin_override_leader(&self) -> RpcResult<()> {
        self.command(SequencerAdminCommand::OverrideLeader).await
    }

    async fn admin_reset_derivation_pipeline(&self) -> RpcResult<()> {
        self.command(SequencerAdminCommand::ResetDerivationPipeline).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_types_engine::ExecutionPayloadV1;

    #[tokio::test]
    async fn reads_published_state_without_queueing_and_rejects_closed_publisher() {
        let initial =
            SequencerState { active: false, conductor_enabled: true, recovery_mode: false };
        let (state_tx, state_rx) = watch::channel(initial);
        let (commands_tx, _commands_rx) = mpsc::channel(1);
        let (reply, _) = oneshot::channel();
        commands_tx.send(SequencerAdminCommand::StartSequencer(reply)).await.unwrap();
        let (payloads_tx, _) = mpsc::channel(1);
        let rpc =
            AdminRpc::new(Some(SequencerAdminHandle::new(state_rx, commands_tx)), payloads_tx);
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
        let rpc = AdminRpc::new(None, tx);
        for result in [
            rpc.admin_sequencer_active().await.map(|_| ()),
            rpc.admin_conductor_enabled().await.map(|_| ()),
            rpc.admin_recover_mode().await.map(|_| ()),
            rpc.admin_start_sequencer().await,
            rpc.admin_stop_sequencer().await.map(|_| ()),
            rpc.admin_set_recover_mode(true).await,
            rpc.admin_override_leader().await,
            rpc.admin_reset_derivation_pipeline().await,
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
        let rpc =
            AdminRpc::new(Some(SequencerAdminHandle::new(state_rx, commands_tx)), payloads_tx);
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
    async fn validates_payloads_before_forwarding_on_validators() {
        let (tx, mut rx) = mpsc::channel(1);
        let rpc = AdminRpc::new(None, tx);
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
