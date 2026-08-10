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
use kona_genesis::RollupConfig;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};
use std::sync::Arc;

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
    /// The rollup configuration used to validate payload versions.
    pub rollup_config: Arc<RollupConfig>,
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
    /// - `rollup_config`: The rollup configuration used to validate payload versions.
    ///
    /// # Returns
    ///
    /// A new [`AdminRpc`] instance.
    pub const fn new(
        sequencer_admin_client: Option<SequencerAdminAPIClient_>,
        network_sender: NetworkAdminQuerySender,
        rollup_config: Arc<RollupConfig>,
    ) -> Self {
        Self { sequencer_admin_client, network_sender, rollup_config }
    }
}

/// Validates that the payload envelope version matches the fork active at its timestamp.
fn validate_payload_version(
    payload: &OpExecutionPayloadEnvelope,
    config: &RollupConfig,
) -> RpcResult<()> {
    let timestamp = payload.timestamp();
    let actual = match payload {
        OpExecutionPayloadEnvelope::V1(_) => 1,
        OpExecutionPayloadEnvelope::V2(_) => 2,
        OpExecutionPayloadEnvelope::V3 { .. } => 3,
        OpExecutionPayloadEnvelope::V4 { .. } => 4,
    };
    let expected = if config.is_isthmus_active(timestamp) {
        4
    } else if config.is_ecotone_active(timestamp) {
        3
    } else if config.is_canyon_active(timestamp) {
        2
    } else {
        1
    };

    if actual == expected {
        return Ok(());
    }

    Err(ErrorObject::owned(
        ErrorCode::InvalidParams.code(),
        format!(
            "payload version V{actual} does not match timestamp {timestamp}; expected V{expected}"
        ),
        None::<()>,
    ))
}

/// Validates an unsafe payload before it enters the network and engine queues.
fn validate_unsafe_payload(
    payload: &OpExecutionPayloadEnvelope,
    config: &RollupConfig,
) -> RpcResult<()> {
    validate_payload_version(payload, config)?;
    payload.check_block_hash().map_err(|err| {
        tracing::warn!(
            target: "rpc",
            %err,
            "admin_postUnsafePayload: rejecting payload"
        );
        let message = match &err {
            OpPayloadError::Eth(PayloadError::BlockHash { execution, consensus }) => {
                format!(
                    "payload has bad block hash: {consensus}, actual block hash is: {execution}"
                )
            }
            _ => format!("invalid payload: {err}"),
        };
        ErrorObject::owned(ErrorCode::InvalidParams.code(), message, None::<()>)
    })
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
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "admin_postUnsafePayload");

        validate_unsafe_payload(&payload, &self.rollup_config)?;

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
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3};
    use op_alloy_rpc_types_engine::OpExecutionPayloadV4;

    #[derive(Debug)]
    struct TestSequencerAdminClient;

    #[async_trait]
    impl SequencerAdminAPIClient for TestSequencerAdminClient {
        async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
            unreachable!()
        }

        async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
            unreachable!()
        }

        async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
            unreachable!()
        }

        async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
            unreachable!()
        }

        async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
            unreachable!()
        }

        async fn set_recovery_mode(&self, _mode: bool) -> Result<(), SequencerAdminAPIError> {
            unreachable!()
        }

        async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
            unreachable!()
        }

        async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
            unreachable!()
        }
    }

    fn v1_payload(timestamp: u64) -> ExecutionPayloadV1 {
        ExecutionPayloadV1 {
            parent_hash: Default::default(),
            fee_recipient: Default::default(),
            state_root: Default::default(),
            receipts_root: Default::default(),
            logs_bloom: Default::default(),
            prev_randao: Default::default(),
            block_number: 1,
            gas_limit: 0,
            gas_used: 0,
            timestamp,
            extra_data: Default::default(),
            base_fee_per_gas: Default::default(),
            block_hash: Default::default(),
            transactions: Vec::new(),
        }
    }

    fn envelope(version: u8, timestamp: u64) -> OpExecutionPayloadEnvelope {
        let v2 =
            ExecutionPayloadV2 { payload_inner: v1_payload(timestamp), withdrawals: Vec::new() };
        match version {
            1 => OpExecutionPayloadEnvelope::V1(v2.payload_inner),
            2 => OpExecutionPayloadEnvelope::V2(v2),
            3 => OpExecutionPayloadEnvelope::V3 {
                payload: ExecutionPayloadV3 {
                    payload_inner: v2,
                    blob_gas_used: 0,
                    excess_blob_gas: 0,
                },
                parent_beacon_block_root: Default::default(),
            },
            4 => OpExecutionPayloadEnvelope::V4 {
                payload: OpExecutionPayloadV4 {
                    payload_inner: ExecutionPayloadV3 {
                        payload_inner: v2,
                        blob_gas_used: 0,
                        excess_blob_gas: 0,
                    },
                    withdrawals_root: Default::default(),
                },
                parent_beacon_block_root: Default::default(),
            },
            _ => panic!("unsupported test payload version"),
        }
    }

    fn valid_envelope(version: u8, timestamp: u64) -> OpExecutionPayloadEnvelope {
        let mut envelope = envelope(version, timestamp);
        let execution = match envelope.check_block_hash().unwrap_err() {
            OpPayloadError::Eth(PayloadError::BlockHash { execution, .. }) => execution,
            error => panic!("unexpected block hash error: {error}"),
        };
        envelope.as_v1_mut().block_hash = execution;
        envelope.check_block_hash().unwrap();
        envelope
    }

    fn rollup_config() -> RollupConfig {
        let mut config = RollupConfig::default();
        config.hardforks.canyon_time = Some(10);
        config.hardforks.ecotone_time = Some(20);
        config.hardforks.isthmus_time = Some(30);
        config
    }

    #[test]
    fn validates_payload_version_at_fork_boundaries() {
        let config = rollup_config();
        for (version, timestamp) in [(1, 9), (2, 10), (2, 19), (3, 20), (3, 29), (4, 30)] {
            validate_payload_version(&envelope(version, timestamp), &config).unwrap();
        }

        for (version, timestamp, expected) in [
            (2, 9, "expected V1"),
            (1, 10, "expected V2"),
            (3, 19, "expected V2"),
            (2, 20, "expected V3"),
            (4, 29, "expected V3"),
            (3, 30, "expected V4"),
        ] {
            let error = validate_payload_version(&envelope(version, timestamp), &config)
                .expect_err("version mismatch accepted");
            assert_eq!(error.code(), ErrorCode::InvalidParams.code());
            assert!(error.message().contains(expected), "{error}");
        }
    }

    #[test]
    fn maps_block_hash_mismatch_to_invalid_params() {
        let error = validate_unsafe_payload(&envelope(1, 9), &rollup_config())
            .expect_err("zero block hash unexpectedly valid");

        assert_eq!(error.code(), ErrorCode::InvalidParams.code());
        assert!(error.message().contains("payload has bad block hash"));
        assert!(error.message().contains("actual block hash is"));
    }

    #[tokio::test]
    async fn only_enqueues_valid_unsafe_payloads() {
        let (network_sender, mut network_rx) = tokio::sync::mpsc::channel(1);
        let rpc = AdminRpc::<TestSequencerAdminClient>::new(
            None,
            network_sender,
            Arc::new(rollup_config()),
        );

        rpc.admin_post_unsafe_payload(envelope(2, 9)).await.unwrap_err();
        assert!(matches!(
            network_rx.try_recv(),
            Err(tokio::sync::mpsc::error::TryRecvError::Empty)
        ));

        rpc.admin_post_unsafe_payload(envelope(1, 9)).await.unwrap_err();
        assert!(matches!(
            network_rx.try_recv(),
            Err(tokio::sync::mpsc::error::TryRecvError::Empty)
        ));

        let payload = valid_envelope(1, 9);
        rpc.admin_post_unsafe_payload(payload.clone()).await.unwrap();
        assert!(matches!(
            network_rx.recv().await,
            Some(NetworkAdminQuery::PostUnsafePayload { payload: received }) if received == payload
        ));
    }
}
