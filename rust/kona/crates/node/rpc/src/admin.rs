//! Admin RPC Module

use crate::{AdminApiServer, RollupBoostAdminClient, SequencerAdminAPIClient};
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
        /// The sender to send the response confirming the execution mode was updated.
        sender: oneshot::Sender<()>,
    },
    /// An admin rpc request to get the execution mode.
    GetExecutionMode {
        /// The sender to send the execution mode to.
        sender: oneshot::Sender<ExecutionMode>,
    },
}

type NetworkAdminQuerySender = tokio::sync::mpsc::Sender<NetworkAdminQuery>;

/// The admin rpc server.
#[derive(Debug)]
pub struct AdminRpc<RollupBoostAdminClient, SequencerAdminAPIClient> {
    /// The sequencer admin API client.
    pub sequencer_admin_client: Option<SequencerAdminAPIClient>,
    /// The sender to the network actor.
    pub network_sender: NetworkAdminQuerySender,
    /// The sender to the rollup boost component of the engine actor.
    /// Only set when rollup boost is enabled.
    pub rollup_boost_client: Option<RollupBoostAdminClient>,
}

impl<RollupBoostAdminClient_, SequencerAdminAPIClient_>
    AdminRpc<RollupBoostAdminClient_, SequencerAdminAPIClient_>
where
    RollupBoostAdminClient_: RollupBoostAdminClient,
    SequencerAdminAPIClient_: SequencerAdminAPIClient,
{
    /// Constructs a new [`AdminRpc`] given the sequencer sender, network sender, and execution
    /// mode.
    ///
    /// # Parameters
    ///
    /// - `sequencer_sender`: The [`SequencerAdminAPIClient`] used to fulfill sequencer admin
    ///   queries.
    /// - `network_sender`: The sender to the network actor.
    /// - `rollup_boost_sender`: The [`RollupBoostAdminClient`] used to fulfill rollup boost admin
    ///   queries.
    /// # Returns
    ///
    /// A new [`AdminRpc`] instance.
    pub const fn new(
        sequencer_admin_client: Option<SequencerAdminAPIClient_>,
        network_sender: NetworkAdminQuerySender,
        rollup_boost_client: Option<RollupBoostAdminClient_>,
    ) -> Self {
        Self { sequencer_admin_client, network_sender, rollup_boost_client }
    }
}

#[async_trait]
impl<RollupBoostAdminClient_, SequencerAdminAPIClient_> AdminApiServer
    for AdminRpc<RollupBoostAdminClient_, SequencerAdminAPIClient_>
where
    RollupBoostAdminClient_: RollupBoostAdminClient + 'static + Send + Sync,
    SequencerAdminAPIClient_: SequencerAdminAPIClient + 'static + Send + Sync,
{
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
        let Some(ref client) = self.rollup_boost_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        client.set_execution_mode(request).await
    }

    async fn get_execution_mode(&self) -> RpcResult<GetExecutionModeResponse> {
        let Some(ref client) = self.rollup_boost_client else {
            return Err(ErrorObject::from(ErrorCode::MethodNotFound));
        };

        client.get_execution_mode().await
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
