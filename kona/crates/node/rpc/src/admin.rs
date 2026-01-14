//! Admin RPC Module

use crate::AdminApiServer;
use alloy_primitives::B256;
use async_trait::async_trait;
use core::fmt::Debug;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use rollup_boost::{
    ExecutionMode, GetExecutionModeResponse, SetExecutionModeRequest, SetExecutionModeResponse,
};
use thiserror::Error;
use tokio::sync::oneshot;

/// The query types to the network actor for the admin api.
#[derive(Debug)]
pub enum NetworkAdminQuery {
    /// An admin rpc request to post an unsafe payload.
    PostUnsafePayload {
        /// The payload to post.
        payload: OpExecutionPayloadEnvelope,
    },
}

/// The query types to the rollup boost component of the engine actor.
/// Only set when rollup boost is enabled.
#[derive(Debug)]
pub enum RollupBoostAdminQuery {
    /// An admin rpc request to set the execution mode.
    SetExecutionMode {
        /// The execution mode to set.
        execution_mode: ExecutionMode,
    },
    /// An admin rpc request to get the execution mode.
    GetExecutionMode {
        /// The sender to send the execution mode to.
        sender: oneshot::Sender<ExecutionMode>,
    },
}

type NetworkAdminQuerySender = tokio::sync::mpsc::Sender<NetworkAdminQuery>;
type RollupBoostAdminQuerySender = tokio::sync::mpsc::Sender<RollupBoostAdminQuery>;

/// The admin rpc server.
#[derive(Debug)]
pub struct AdminRpc<SequencerAdminAPIClient> {
    /// The sequencer admin API client.
    pub sequencer_admin_client: Option<SequencerAdminAPIClient>,
    /// The sender to the network actor.
    pub network_sender: NetworkAdminQuerySender,
    /// The sender to the rollup boost component of the engine actor.
    /// Only set when rollup boost is enabled.
    pub rollup_boost_sender: Option<RollupBoostAdminQuerySender>,
}

impl<S: SequencerAdminAPIClient> AdminRpc<S> {
    /// Constructs a new [`AdminRpc`] given the sequencer sender, network sender, and execution
    /// mode.
    ///
    /// # Parameters
    ///
    /// - `sequencer_sender`: The sender to the sequencer actor.
    /// - `network_sender`: The sender to the network actor.
    /// - `rollup_boost_sender`: Sender of admin queries to the rollup boost component of the engine
    ///   actor.
    ///
    /// # Returns
    ///
    /// A new [`AdminRpc`] instance.
    pub const fn new(
        sequencer_admin_client: Option<S>,
        network_sender: NetworkAdminQuerySender,
        rollup_boost_sender: Option<RollupBoostAdminQuerySender>,
    ) -> Self {
        Self { sequencer_admin_client, network_sender, rollup_boost_sender }
    }
}

#[async_trait]
impl<S: SequencerAdminAPIClient + 'static> AdminApiServer for AdminRpc<S> {
    async fn admin_post_unsafe_payload(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "admin_postUnsafePayload");
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

    async fn set_execution_mode(
        &self,
        request: SetExecutionModeRequest,
    ) -> RpcResult<SetExecutionModeResponse> {
        let Some(ref rollup_boost_sender) = self.rollup_boost_sender else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        rollup_boost_sender
            .send(RollupBoostAdminQuery::SetExecutionMode {
                execution_mode: request.execution_mode,
            })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        Ok(SetExecutionModeResponse { execution_mode: request.execution_mode })
    }

    async fn get_execution_mode(&self) -> RpcResult<GetExecutionModeResponse> {
        let Some(ref rollup_boost_sender) = self.rollup_boost_sender else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        let (tx, rx) = oneshot::channel();

        rollup_boost_sender
            .send(RollupBoostAdminQuery::GetExecutionMode { sender: tx })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        rx.await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
            .map(|execution_mode| GetExecutionModeResponse { execution_mode })
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

/// The admin API client for the sequencer actor.
#[async_trait]
pub trait SequencerAdminAPIClient: Send + Sync + Debug {
    /// Check if the sequencer is active.
    async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError>;

    /// Check if the conductor is enabled.
    async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError>;

    /// Check if in recovery mode.
    async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError>;

    /// Start the sequencer.
    async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError>;

    /// Stop the sequencer.
    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError>;

    /// Set recovery mode.
    async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError>;

    /// Override the leader.
    async fn override_leader(&self) -> Result<(), SequencerAdminAPIError>;

    /// Reset the derivation pipeline.
    async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError>;
}

/// Errors that can occur when using the sequencer admin API.
#[derive(Debug, Error)]
pub enum SequencerAdminAPIError {
    /// Error sending request.
    #[error("Error sending request: {0}.")]
    RequestError(String),

    /// Error receiving response.
    #[error("Error receiving response: {0}.")]
    ResponseError(String),

    /// Error stopping sequencer.
    #[error("Error stopping sequencer: {0}.")]
    StopError(#[from] StopSequencerError),

    /// Error overriding leader.
    #[error("Error overriding leader: {0}.")]
    LeaderOverrideError(String),
}

/// Errors that can occur when using the sequencer admin API.
#[derive(Debug, Error)]
pub enum StopSequencerError {
    /// Sequencer stopped successfully, followed by some error.
    #[error("Sequencer stopped successfully, followed by error: {0}.")]
    ErrorAfterSequencerWasStopped(String),
}
