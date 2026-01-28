use alloy_network::Network;
use alloy_primitives::{B256, BlockHash, Bytes};
use alloy_provider::Provider;
use alloy_rpc_types_engine::{
    ClientVersionV1, ExecutionPayloadBodiesV1, ExecutionPayloadEnvelopeV2, ExecutionPayloadInputV2,
    ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus,
};
use alloy_transport::{Transport, TransportResult};
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes, ProtocolVersion,
};

/// Extension trait that gives access to Optimism engine API RPC methods.
///
/// Note:
/// > The provider should use a JWT authentication layer.
///
/// This follows the Optimism specs that can be found at:
/// <https://specs.optimism.io/protocol/exec-engine.html#engine-api>
#[cfg_attr(target_arch = "wasm32", async_trait::async_trait(?Send))]
#[cfg_attr(not(target_arch = "wasm32"), async_trait::async_trait)]
pub trait OpEngineApi<N, T> {
    /// Sends the given payload to the execution layer client, as specified for the Shanghai fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#engine_newpayloadv2>
    ///
    /// No modifications needed for OP compatibility.
    async fn new_payload_v2(
        &self,
        payload: ExecutionPayloadInputV2,
    ) -> TransportResult<PayloadStatus>;

    /// Sends the given payload to the execution layer client, as specified for the Cancun fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/cancun.md#engine_newpayloadv3>
    ///
    /// OP modifications:
    /// - expected versioned hashes MUST be an empty array: therefore the `versioned_hashes`
    ///   parameter is removed.
    /// - parent beacon block root MUST be the parent beacon block root from the L1 origin block of
    ///   the L2 block.
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus>;

    /// Sends the given payload to the execution layer client, as specified for the Prague fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/03911ffc053b8b806123f1fc237184b0092a485a/src/engine/prague.md#engine_newpayloadv4>
    ///
    /// OP modifications: TODO
    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus>;

    /// Updates the execution layer client with the given fork choice, as specified for the Shanghai
    /// fork.
    ///
    /// Caution: This should not accept the `parentBeaconBlockRoot` field in the payload attributes.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6709c2a795b707202e93c4f2867fa0bf2640a84f/src/engine/shanghai.md#engine_forkchoiceupdatedv2>
    ///
    /// OP modifications:
    /// - The `payload_attributes` parameter is extended with the [`OpPayloadAttributes`] type
    ///   as described in <https://specs.optimism.io/protocol/exec-engine.html#extended-payloadattributesv2>
    async fn fork_choice_updated_v2(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated>;

    /// Updates the execution layer client with the given fork choice, as specified for the Cancun
    /// fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/cancun.md#engine_forkchoiceupdatedv3>
    ///
    /// OP modifications:
    /// - Must be called with an Ecotone payload
    /// - Attributes must contain the parent beacon block root field
    /// - The `payload_attributes` parameter is extended with the [`OpPayloadAttributes`] type
    ///   as described in <https://specs.optimism.io/protocol/exec-engine.html#extended-payloadattributesv2>
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated>;

    /// Retrieves an execution payload from a previously started build process, as specified for the
    /// Shanghai fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6709c2a795b707202e93c4f2867fa0bf2640a84f/src/engine/shanghai.md#engine_getpayloadv2>
    ///
    /// Note:
    /// > Provider software MAY stop the corresponding build process after serving this call.
    ///
    /// No modifications needed for OP compatibility.
    async fn get_payload_v2(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<ExecutionPayloadEnvelopeV2>;

    /// Retrieves an execution payload from a previously started build process, as specified for the
    /// Cancun fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/cancun.md#engine_getpayloadv3>
    ///
    /// Note:
    /// > Provider software MAY stop the corresponding build process after serving this call.
    ///
    /// OP modifications:
    /// - the response type is extended to [`OpExecutionPayloadEnvelopeV3`].
    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV3>;

    /// Returns the most recent version of the payload that is available in the corresponding
    /// payload build process at the time of receiving this call.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/prague.md#engine_getpayloadv4>
    ///
    /// Note:
    /// > Provider software MAY stop the corresponding build process after serving this call.
    ///
    /// OP modifications:
    /// - the response type is extended to [`OpExecutionPayloadEnvelopeV4`].
    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV4>;

    /// Returns the execution payload bodies by the given hash.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6452a6b194d7db269bf1dbd087a267251d3cc7f8/src/engine/shanghai.md#engine_getpayloadbodiesbyhashv1>
    async fn get_payload_bodies_by_hash_v1(
        &self,
        block_hashes: Vec<BlockHash>,
    ) -> TransportResult<ExecutionPayloadBodiesV1>;

    /// Returns the execution payload bodies by the range starting at `start`, containing `count`
    /// blocks.
    ///
    /// WARNING: This method is associated with the BeaconBlocksByRange message in the consensus
    /// layer p2p specification, meaning the input should be treated as untrusted or potentially
    /// adversarial.
    ///
    /// Implementers should take care when acting on the input to this method, specifically
    /// ensuring that the range is limited properly, and that the range boundaries are computed
    /// correctly and without panics.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6452a6b194d7db269bf1dbd087a267251d3cc7f8/src/engine/shanghai.md#engine_getpayloadbodiesbyrangev1>
    async fn get_payload_bodies_by_range_v1(
        &self,
        start: u64,
        count: u64,
    ) -> TransportResult<ExecutionPayloadBodiesV1>;

    /// Returns the execution client version information.
    ///
    /// Note:
    /// > The `client_version` parameter identifies the consensus client.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/identification.md#engine_getclientversionv1>
    async fn get_client_version_v1(
        &self,
        client_version: ClientVersionV1,
    ) -> TransportResult<Vec<ClientVersionV1>>;

    /// Optional extension to the Engine API.
    ///
    /// Signals superchain information to the Engine: V1 signals which protocol version is
    /// recommended and required.
    ///
    /// See : <https://specs.optimism.io/protocol/exec-engine.html#engine_signalsuperchainv1>
    async fn signal_superchain_v1(
        &self,
        recommended: ProtocolVersion,
        required: ProtocolVersion,
    ) -> TransportResult<ProtocolVersion>;

    /// Returns the list of Engine API methods supported by the execution layer client software.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6452a6b194d7db269bf1dbd087a267251d3cc7f8/src/engine/common.md#capabilities>
    async fn exchange_capabilities(
        &self,
        capabilities: Vec<String>,
    ) -> TransportResult<Vec<String>>;
}

#[cfg_attr(target_arch = "wasm32", async_trait::async_trait(?Send))]
#[cfg_attr(not(target_arch = "wasm32"), async_trait::async_trait)]
impl<N, T, P> OpEngineApi<N, T> for P
where
    N: Network,
    T: Transport + Clone,
    P: Provider<N>,
{
    async fn new_payload_v2(
        &self,
        payload: ExecutionPayloadInputV2,
    ) -> TransportResult<PayloadStatus> {
        self.client().request("engine_newPayloadV2", (payload,)).await
    }

    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        // Note: The `versioned_hashes` parameter is always an empty array for OP chains.
        let versioned_hashes: Vec<B256> = vec![];

        self.client()
            .request("engine_newPayloadV3", (payload, versioned_hashes, parent_beacon_block_root))
            .await
    }

    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        // Note: The `versioned_hashes`, `execution_requests` parameters are always an empty array
        // for OP chains.
        let versioned_hashes: Vec<B256> = vec![];
        let execution_requests: Vec<Bytes> = vec![];

        self.client()
            .request(
                "engine_newPayloadV4",
                (payload, versioned_hashes, parent_beacon_block_root, execution_requests),
            )
            .await
    }

    async fn fork_choice_updated_v2(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        self.client()
            .request("engine_forkchoiceUpdatedV2", (fork_choice_state, payload_attributes))
            .await
    }

    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        self.client()
            .request("engine_forkchoiceUpdatedV3", (fork_choice_state, payload_attributes))
            .await
    }

    async fn get_payload_v2(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<ExecutionPayloadEnvelopeV2> {
        self.client().request("engine_getPayloadV2", (payload_id,)).await
    }

    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV3> {
        self.client().request("engine_getPayloadV3", (payload_id,)).await
    }

    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV4> {
        self.client().request("engine_getPayloadV4", (payload_id,)).await
    }

    async fn get_payload_bodies_by_hash_v1(
        &self,
        block_hashes: Vec<BlockHash>,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        self.client().request("engine_getPayloadBodiesByHashV1", (block_hashes,)).await
    }

    async fn get_payload_bodies_by_range_v1(
        &self,
        start: u64,
        count: u64,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        self.client().request("engine_getPayloadBodiesByRangeV1", (start, count)).await
    }

    async fn get_client_version_v1(
        &self,
        client_version: ClientVersionV1,
    ) -> TransportResult<Vec<ClientVersionV1>> {
        self.client().request("engine_getClientVersionV1", (client_version,)).await
    }

    async fn signal_superchain_v1(
        &self,
        recommended: ProtocolVersion,
        required: ProtocolVersion,
    ) -> TransportResult<ProtocolVersion> {
        let signal = op_alloy_rpc_types_engine::SuperchainSignal { recommended, required };
        self.client().request("engine_signalSuperchainV1", (signal,)).await
    }

    async fn exchange_capabilities(
        &self,
        capabilities: Vec<String>,
    ) -> TransportResult<Vec<String>> {
        self.client().request("engine_exchangeCapabilities", (capabilities,)).await
    }
}
