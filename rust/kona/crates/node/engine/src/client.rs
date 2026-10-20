//! An Engine API Client.

use crate::{HyperAuthClient, Metrics};
use alloy_eips::BlockId;
use alloy_network::{Ethereum, Network};
use alloy_primitives::{B256, BlockHash};
use alloy_provider::{EthGetBlock, Provider, ext::EngineApi};
use alloy_rpc_types_engine::{
    ClientVersionV1, ExecutionPayloadBodiesV1, ExecutionPayloadEnvelopeV2, ExecutionPayloadInputV2,
    ExecutionPayloadV1, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId,
    PayloadStatus,
};
use alloy_transport::TransportResult;
use alloy_transport_http::Http;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use op_alloy_network::Optimism;
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes,
};
use std::{sync::Arc, time::Instant};

/// Engine API client used to communicate with L1/L2 ELs.
/// `EngineClient` trait that is very coupled to its only implementation.
/// The main reason this exists is for mocking/unit testing.
#[async_trait]
pub trait EngineClient: OpEngineApi<Optimism, Http<HyperAuthClient>> + Send + Sync {
    /// Returns a reference to the inner [`RollupConfig`].
    fn cfg(&self) -> &RollupConfig;

    /// Fetches the L1 block with the provided `BlockId`.
    fn get_l1_block(&self, block: BlockId) -> EthGetBlock<<Ethereum as Network>::BlockResponse>;

    /// Fetches the L2 block with the provided `BlockId`.
    fn get_l2_block(&self, block: BlockId) -> EthGetBlock<<Optimism as Network>::BlockResponse>;

    /// Sends the given payload to the execution layer client, as specified for the Paris fork.
    async fn new_payload_v1(&self, payload: ExecutionPayloadV1) -> TransportResult<PayloadStatus>;
}

/// An Engine API client that provides authenticated HTTP communication with an execution layer.
///
/// The [`OpEngineClient`] handles JWT authentication and manages connections to both L1 and L2
/// execution layers. It automatically selects the appropriate Engine API version based on the
/// rollup configuration and block timestamps.
#[derive(Clone, Debug)]
pub struct OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    /// The L2 engine provider for Engine API calls.
    engine: L2Provider,
    /// The L1 chain provider for reading L1 data.
    l1_provider: L1Provider,
    /// The [`RollupConfig`] for determining Engine API versions based on hardfork activations.
    cfg: Arc<RollupConfig>,
}

impl<L1Provider, L2Provider> OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    /// Uses the caller's L1/L2 providers and the same chain rules as the rest of the node.
    pub const fn new(l1_provider: L1Provider, engine: L2Provider, cfg: Arc<RollupConfig>) -> Self {
        Self { engine, l1_provider, cfg }
    }
}

#[async_trait]
impl<L1Provider, L2Provider> EngineClient for OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    fn cfg(&self) -> &RollupConfig {
        self.cfg.as_ref()
    }

    fn get_l1_block(&self, block: BlockId) -> EthGetBlock<<Ethereum as Network>::BlockResponse> {
        self.l1_provider.get_block(block)
    }

    fn get_l2_block(&self, block: BlockId) -> EthGetBlock<<Optimism as Network>::BlockResponse> {
        self.engine.get_block(block)
    }

    async fn new_payload_v1(&self, payload: ExecutionPayloadV1) -> TransportResult<PayloadStatus> {
        record_call_time(self.engine.new_payload_v1(payload), Metrics::NEW_PAYLOAD_METHOD).await
    }
}

#[async_trait::async_trait]
impl<L1Provider, L2Provider> OpEngineApi<Optimism, Http<HyperAuthClient>>
    for OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    async fn new_payload_v2(
        &self,
        payload: ExecutionPayloadInputV2,
    ) -> TransportResult<PayloadStatus> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::new_payload_v2(
            &self.engine,
            payload,
        );

        record_call_time(call, Metrics::NEW_PAYLOAD_METHOD).await
    }

    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::new_payload_v3(
            &self.engine,
            payload,
            parent_beacon_block_root,
        );

        record_call_time(call, Metrics::NEW_PAYLOAD_METHOD).await
    }

    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::new_payload_v4(
            &self.engine,
            payload,
            parent_beacon_block_root,
        );

        record_call_time(call, Metrics::NEW_PAYLOAD_METHOD).await
    }

    async fn fork_choice_updated_v2(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let call =
            <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::fork_choice_updated_v2(
                &self.engine,
                fork_choice_state,
                payload_attributes,
            );

        record_call_time(call, Metrics::FORKCHOICE_UPDATE_METHOD).await
    }

    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let call =
            <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::fork_choice_updated_v3(
                &self.engine,
                fork_choice_state,
                payload_attributes,
            );

        record_call_time(call, Metrics::FORKCHOICE_UPDATE_METHOD).await
    }

    async fn get_payload_v2(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<ExecutionPayloadEnvelopeV2> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_payload_v2(
            &self.engine,
            payload_id,
        );

        record_call_time(call, Metrics::GET_PAYLOAD_METHOD).await
    }

    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV3> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_payload_v3(
            &self.engine,
            payload_id,
        );

        record_call_time(call, Metrics::GET_PAYLOAD_METHOD).await
    }

    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV4> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_payload_v4(
            &self.engine,
            payload_id,
        );

        record_call_time(call, Metrics::GET_PAYLOAD_METHOD).await
    }

    async fn get_payload_v5(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV4> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_payload_v5(
            &self.engine,
            payload_id,
        );

        record_call_time(call, Metrics::GET_PAYLOAD_METHOD).await
    }

    async fn get_payload_bodies_by_hash_v1(
        &self,
        block_hashes: Vec<BlockHash>,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_payload_bodies_by_hash_v1(
            &self.engine,
            block_hashes,
        )
        .await
    }

    async fn get_payload_bodies_by_range_v1(
        &self,
        start: u64,
        count: u64,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        <L2Provider as OpEngineApi<
            Optimism,
            Http<HyperAuthClient>,
        >>::get_payload_bodies_by_range_v1(&self.engine, start, count).await
    }

    async fn get_client_version_v1(
        &self,
        client_version: ClientVersionV1,
    ) -> TransportResult<Vec<ClientVersionV1>> {
        <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_client_version_v1(
            &self.engine,
            client_version,
        )
        .await
    }

    async fn exchange_capabilities(
        &self,
        capabilities: Vec<String>,
    ) -> TransportResult<Vec<String>> {
        <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::exchange_capabilities(
            &self.engine,
            capabilities,
        )
        .await
    }
}

/// Wrapper to record the time taken for a call to the engine API and log the result as a metric.
async fn record_call_time<T, Err>(
    f: impl Future<Output = Result<T, Err>>,
    metric_label: &'static str,
) -> Result<T, Err> {
    // Await on the future and track its duration.
    let start = Instant::now();
    let result = f.await?;
    let duration = start.elapsed();

    // Record the call duration.
    kona_macros::record!(
        histogram,
        Metrics::ENGINE_METHOD_REQUEST_DURATION,
        "method",
        metric_label,
        duration.as_secs_f64()
    );
    Ok(result)
}
