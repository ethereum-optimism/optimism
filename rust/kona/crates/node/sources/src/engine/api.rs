//! An Engine API Client.

use super::versions::{EngineForkchoiceVersion, EngineGetPayloadVersion};
use alloy_eips::{BlockId, eip1898::BlockNumberOrTag};
use alloy_network::Network;
use alloy_provider::{EthGetBlock, Provider, RootProvider};
use alloy_rpc_types_engine::{
    ExecutionPayloadFieldV2, ExecutionPayloadInputV2, ExecutionPayloadV1, ForkchoiceState,
    ForkchoiceUpdated, PayloadId, PayloadStatus,
};
use alloy_rpc_types_eth::Block;
use alloy_transport::{RpcError, TransportErrorKind, TransportResult};
use kona_genesis::RollupConfig;
use op_alloy_network::Optimism;
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types::Transaction;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::sync::Arc;
use thiserror::Error;

/// An error that occurred in the [`EngineApi`].
#[derive(Error, Debug)]
pub enum EngineApiError {
    /// An RPC error occurred
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] RpcError<TransportErrorKind>),
}
/// Client for the L2 execution layer's Engine API. Providers erase their transport type, so tests
/// use the same client with a mock transport. [`Self::forkchoice_updated`], [`Self::get_payload`]
/// and [`Self::new_payload`] select the Engine API method version; the versioned methods are not
/// exposed, so callers cannot pick the wrong one.
#[derive(Clone, Debug)]
pub struct EngineApi {
    engine: RootProvider<Optimism>,
    cfg: Arc<RollupConfig>,
}

impl EngineApi {
    /// Creates a client from an existing Engine API provider.
    pub const fn new(engine: RootProvider<Optimism>, cfg: Arc<RollupConfig>) -> Self {
        Self { engine, cfg }
    }

    /// Returns the rollup configuration.
    pub fn cfg(&self) -> &RollupConfig {
        &self.cfg
    }

    /// Sends `engine_exchangeCapabilities`, which fails if the execution layer rejects the client's
    /// JWT.
    pub async fn exchange_capabilities(
        &self,
        capabilities: Vec<String>,
    ) -> TransportResult<Vec<String>> {
        self.engine.exchange_capabilities(capabilities).await
    }

    /// Fetches an L2 block.
    pub fn get_l2_block(
        &self,
        block: BlockId,
    ) -> EthGetBlock<<Optimism as Network>::BlockResponse> {
        self.engine.get_block(block)
    }

    /// Submits a Paris payload, whose method is provided by Alloy's Ethereum Engine API extension.
    async fn new_payload_v1(&self, payload: ExecutionPayloadV1) -> TransportResult<PayloadStatus> {
        // OP Alloy's extension has no V1 method, and importing Alloy's Ethereum extension would
        // make the other versions' methods ambiguous.
        alloy_provider::ext::EngineApi::new_payload_v1(&self.engine, payload).await
    }

    /// Sends `engine_forkchoiceUpdated`, selecting the method version for the payload attributes'
    /// timestamp.
    ///
    /// Without payload attributes, the call does not depend on the version, so V3 is used.
    pub async fn forkchoice_updated(
        &self,
        state: ForkchoiceState,
        attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let version = attributes.as_ref().map_or(EngineForkchoiceVersion::V3, |attributes| {
            EngineForkchoiceVersion::from_cfg(&self.cfg, attributes.payload_attributes.timestamp)
        });
        match version {
            EngineForkchoiceVersion::V3 => {
                self.engine.fork_choice_updated_v3(state, attributes).await
            }
            EngineForkchoiceVersion::V2 => {
                self.engine.fork_choice_updated_v2(state, attributes).await
            }
        }
    }

    /// Fetches a built payload with `engine_getPayload`, selecting the method version for the
    /// payload's timestamp.
    ///
    /// - `engine_getPayloadV2` is used for payloads with a timestamp before the Ecotone fork.
    /// - `engine_getPayloadV3` is used for payloads with a timestamp after the Ecotone fork.
    /// - `engine_getPayloadV4` is used for payloads with a timestamp after the Isthmus fork.
    /// - `engine_getPayloadV5` is used for payloads with a timestamp after the Karst fork.
    pub async fn get_payload(
        &self,
        payload_id: PayloadId,
        timestamp: u64,
    ) -> TransportResult<OpExecutionPayloadEnvelope> {
        Ok(match EngineGetPayloadVersion::from_cfg(&self.cfg, timestamp) {
            EngineGetPayloadVersion::V5 => {
                // Osaka (Karst) reuses the V4-shaped envelope; only the engine method bumps to V5.
                let payload = self.engine.get_payload_v5(payload_id).await?;
                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V4 => {
                let payload = self.engine.get_payload_v4(payload_id).await?;
                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V3 => {
                let payload = self.engine.get_payload_v3(payload_id).await?;
                OpExecutionPayloadEnvelope::V3 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V2 => {
                let payload = self.engine.get_payload_v2(payload_id).await?;
                match payload.execution_payload {
                    ExecutionPayloadFieldV2::V1(payload) => OpExecutionPayloadEnvelope::V1(payload),
                    ExecutionPayloadFieldV2::V2(payload) => OpExecutionPayloadEnvelope::V2(payload),
                }
            }
        })
    }

    /// Submits a payload with `engine_newPayload`, selecting the method version for the payload's
    /// envelope.
    pub async fn new_payload(
        &self,
        envelope: OpExecutionPayloadEnvelope,
    ) -> TransportResult<PayloadStatus> {
        match envelope {
            OpExecutionPayloadEnvelope::V1(payload) => self.new_payload_v1(payload).await,
            OpExecutionPayloadEnvelope::V2(payload) => {
                let payload_input = ExecutionPayloadInputV2 {
                    execution_payload: payload.payload_inner,
                    withdrawals: Some(payload.withdrawals),
                };
                self.engine.new_payload_v2(payload_input).await
            }
            OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root } => {
                self.engine.new_payload_v3(payload, parent_beacon_block_root).await
            }
            OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root } => {
                self.engine.new_payload_v4(payload, parent_beacon_block_root).await
            }
        }
    }

    /// Fetches an L2 block with full transactions.
    pub async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineApiError> {
        Ok(self.engine.get_block_by_number(numtag).full().await?)
    }
}

#[cfg(test)]
mod version_tests {
    use super::*;
    use crate::test_utils::test_engine_api;
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{
        ExecutionPayloadEnvelopeV2, ExecutionPayloadV2, ExecutionPayloadV3, PayloadAttributes,
        PayloadStatusEnum,
    };
    use op_alloy_rpc_types_engine::{
        OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    };
    use rstest::rstest;
    use serde_json::json;

    /// Returns a configuration that activates Ecotone at 10, Isthmus at 20, and Karst at 30.
    fn config() -> Arc<RollupConfig> {
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(10);
        cfg.hardforks.isthmus_time = Some(20);
        cfg.hardforks.karst_time = Some(30);
        Arc::new(cfg)
    }

    fn valid() -> PayloadStatus {
        PayloadStatus::from_status(PayloadStatusEnum::Valid)
    }

    #[rstest]
    #[case::before_ecotone(Some(5), "engine_forkchoiceUpdatedV2")]
    #[case::after_ecotone(Some(15), "engine_forkchoiceUpdatedV3")]
    #[case::without_attributes(None, "engine_forkchoiceUpdatedV3")]
    #[tokio::test]
    async fn forkchoice_update_selects_version(
        #[case] timestamp: Option<u64>,
        #[case] method: &'static str,
    ) {
        let (client, l2) = test_engine_api(config());
        let state = ForkchoiceState::default();
        let attributes = timestamp.map(|timestamp| OpPayloadAttributes {
            payload_attributes: PayloadAttributes { timestamp, ..Default::default() },
            ..Default::default()
        });
        l2.expect_params(method, json!([state, attributes]), ForkchoiceUpdated::new(valid()));

        let response = client.forkchoice_updated(state, attributes).await.unwrap();
        assert_eq!(response.payload_status, valid());
        l2.assert_finished();
    }

    fn block() -> alloy_consensus::Block<op_alloy_consensus::OpTxEnvelope> {
        alloy_consensus::Block::default()
    }

    /// The parent beacon block root, distinct from the withdrawals root.
    const BEACON_ROOT: B256 = B256::repeat_byte(0x42);
    /// The Isthmus withdrawals root, distinct from the parent beacon block root.
    const WITHDRAWALS_ROOT: B256 = B256::repeat_byte(0x43);

    /// Returns a payload envelope, its `engine_newPayload` method, and the parameters it sends.
    fn v1() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = ExecutionPayloadV1::from_block_slow(&block());
        (OpExecutionPayloadEnvelope::V1(payload.clone()), "engine_newPayloadV1", json!([payload]))
    }

    fn v2() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = ExecutionPayloadV2::from_block_slow(&block());
        (OpExecutionPayloadEnvelope::V2(payload.clone()), "engine_newPayloadV2", json!([payload]))
    }

    fn v3() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = ExecutionPayloadV3::from_block_slow(&block());
        (
            OpExecutionPayloadEnvelope::V3 {
                payload: payload.clone(),
                parent_beacon_block_root: BEACON_ROOT,
            },
            "engine_newPayloadV3",
            json!([payload, [], BEACON_ROOT]),
        )
    }

    fn v4() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
            ExecutionPayloadV3::from_block_slow(&block()),
            WITHDRAWALS_ROOT,
        );
        (
            OpExecutionPayloadEnvelope::V4 {
                payload: payload.clone(),
                parent_beacon_block_root: BEACON_ROOT,
            },
            "engine_newPayloadV4",
            json!([payload, [], BEACON_ROOT, []]),
        )
    }

    #[rstest]
    #[case::v1(v1())]
    #[case::v2(v2())]
    #[case::v3(v3())]
    #[case::v4(v4())]
    #[tokio::test]
    async fn new_payload_selects_version_by_envelope(
        #[case] call: (OpExecutionPayloadEnvelope, &'static str, serde_json::Value),
    ) {
        let (envelope, method, params) = call;
        let (client, l2) = test_engine_api(config());
        l2.expect_params(method, params, valid());

        assert_eq!(client.new_payload(envelope).await.unwrap(), valid());
        l2.assert_finished();
    }

    /// Exercise version selection and decoding through the production client and OP Alloy
    /// extension.
    #[rstest]
    #[case(4, "engine_getPayloadV2")]
    #[case(5, "engine_getPayloadV2")]
    #[case(15, "engine_getPayloadV3")]
    #[case(25, "engine_getPayloadV4")]
    #[case(35, "engine_getPayloadV5")]
    #[tokio::test]
    async fn payload_fetch_selects_version_and_decodes_reply(
        #[case] timestamp: u64,
        #[case] method: &'static str,
    ) {
        let (client, l2) = test_engine_api(config());
        let id = PayloadId::new([1; 8]);
        let params = json!([id]);
        let root = B256::repeat_byte(0x42);
        let block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
        let expected = if timestamp < 5 {
            // A reply without withdrawals is a pre-Canyon V1 payload.
            let payload = ExecutionPayloadV1::from_block_slow(&block);
            l2.expect_params(
                method,
                params,
                ExecutionPayloadEnvelopeV2 {
                    execution_payload: ExecutionPayloadFieldV2::V1(payload.clone()),
                    block_value: Default::default(),
                },
            );
            OpExecutionPayloadEnvelope::V1(payload)
        } else if timestamp < 10 {
            let payload = ExecutionPayloadV2::from_block_slow(&block);
            l2.expect_params(
                method,
                params,
                ExecutionPayloadEnvelopeV2 {
                    execution_payload: ExecutionPayloadFieldV2::V2(payload.clone()),
                    block_value: Default::default(),
                },
            );
            OpExecutionPayloadEnvelope::V2(payload)
        } else if timestamp < 20 {
            let payload = ExecutionPayloadV3::from_block_slow(&block);
            l2.expect_params(
                method,
                params,
                OpExecutionPayloadEnvelopeV3 {
                    execution_payload: payload.clone(),
                    block_value: Default::default(),
                    blobs_bundle: Default::default(),
                    should_override_builder: false,
                    parent_beacon_block_root: root,
                },
            );
            OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root: root }
        } else {
            let payload = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
                ExecutionPayloadV3::from_block_slow(&block),
                root,
            );
            l2.expect_params(
                method,
                params,
                OpExecutionPayloadEnvelopeV4 {
                    execution_payload: payload.clone(),
                    block_value: Default::default(),
                    blobs_bundle: Default::default(),
                    should_override_builder: false,
                    parent_beacon_block_root: root,
                    execution_requests: vec![],
                },
            );
            OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root: root }
        };
        assert_eq!(client.get_payload(id, timestamp).await.unwrap(), expected);
        l2.assert_finished();
    }
}
