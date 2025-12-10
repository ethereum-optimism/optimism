//! Mock implementations for testing engine client functionality.

use crate::{EngineClient, HyperAuthClient};
use alloy_eips::{BlockId, eip1898::BlockNumberOrTag};
use alloy_network::{Ethereum, Network};
use alloy_primitives::{Address, B256, BlockHash, StorageKey};
use alloy_provider::{EthGetBlock, ProviderCall, RpcWithBlock};
use alloy_rpc_types_engine::{
    ClientVersionV1, ExecutionPayloadBodiesV1, ExecutionPayloadEnvelopeV2, ExecutionPayloadInputV2,
    ExecutionPayloadV1, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId,
    PayloadStatus,
};
use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse, Transaction as EthTransaction};
use alloy_transport::{TransportError, TransportErrorKind, TransportResult};
use alloy_transport_http::Http;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_network::Optimism;
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types::Transaction as OpTransaction;
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes, ProtocolVersion,
};
use std::{collections::HashMap, sync::Arc};
use tokio::sync::RwLock;

use crate::EngineClientError;

/// Builder for creating test MockEngineClient instances with sensible defaults
pub fn test_engine_client_builder() -> MockEngineClientBuilder {
    MockEngineClientBuilder::new().with_config(Arc::new(RollupConfig::default()))
}

/// Mock storage for engine client responses.
///
/// Each API method has version-specific storage to allow tests to verify
/// which specific version was called and return different responses per version.
#[derive(Debug, Clone, Default)]
pub struct MockEngineStorage {
    /// Storage for block responses by tag.
    pub l2_blocks_by_label: HashMap<BlockNumberOrTag, Block<OpTransaction>>,
    /// Storage for block info responses by tag.
    pub block_info_by_tag: HashMap<BlockNumberOrTag, L2BlockInfo>,

    // Version-specific new_payload responses
    /// Storage for new_payload_v1 responses.
    pub new_payload_v1_response: Option<PayloadStatus>,
    /// Storage for new_payload_v2 responses.
    pub new_payload_v2_response: Option<PayloadStatus>,
    /// Storage for new_payload_v3 responses.
    pub new_payload_v3_response: Option<PayloadStatus>,
    /// Storage for new_payload_v4 responses.
    pub new_payload_v4_response: Option<PayloadStatus>,

    // Version-specific fork_choice_updated responses
    /// Storage for fork_choice_updated_v2 responses.
    pub fork_choice_updated_v2_response: Option<ForkchoiceUpdated>,
    /// Storage for fork_choice_updated_v3 responses.
    pub fork_choice_updated_v3_response: Option<ForkchoiceUpdated>,

    // Version-specific get_payload responses
    /// Storage for execution payload envelope v2 responses.
    pub execution_payload_v2: Option<ExecutionPayloadEnvelopeV2>,
    /// Storage for OP execution payload envelope v3 responses.
    pub execution_payload_v3: Option<OpExecutionPayloadEnvelopeV3>,
    /// Storage for OP execution payload envelope v4 responses.
    pub execution_payload_v4: Option<OpExecutionPayloadEnvelopeV4>,

    // Version-specific get_payload_bodies responses
    /// Storage for get_payload_bodies_by_hash_v1 responses.
    pub get_payload_bodies_by_hash_v1_response: Option<ExecutionPayloadBodiesV1>,
    /// Storage for get_payload_bodies_by_range_v1 responses.
    pub get_payload_bodies_by_range_v1_response: Option<ExecutionPayloadBodiesV1>,

    // Non-versioned responses
    /// Storage for client version responses.
    pub client_versions: Option<Vec<ClientVersionV1>>,
    /// Storage for protocol version responses.
    pub protocol_version: Option<ProtocolVersion>,
    /// Storage for capabilities responses.
    pub capabilities: Option<Vec<String>>,

    // Storage for get_l1_block, get_l2_block, and get_proof
    /// Storage for L1 blocks by stringified BlockId.
    /// L1 blocks use standard Ethereum transactions.
    pub l1_blocks_by_id: HashMap<String, Block<EthTransaction>>,
    /// Storage for L2 blocks by stringified BlockId.
    /// L2 blocks use OP Stack transactions.
    pub l2_blocks_by_id: HashMap<String, Block<OpTransaction>>,
    /// Storage for proofs by (address, stringified BlockId) key.
    pub proofs_by_address: HashMap<(Address, String), EIP1186AccountProofResponse>,
}

/// Builder for constructing a [`MockEngineClient`] with pre-configured responses.
///
/// This builder allows you to set up mock responses before creating the client,
/// making it easier to write concise tests.
///
/// # Example
///
/// ```rust,ignore
/// use kona_engine::test_utils::{MockEngineClient};
/// use alloy_rpc_types_engine::{PayloadStatus, PayloadStatusEnum};
/// use std::sync::Arc;
///
/// let mock = MockEngineClient::builder()
///     .with_config(Arc::new(RollupConfig::default()))
///     .with_payload_status(PayloadStatus {
///         status: PayloadStatusEnum::Valid,
///         latest_valid_hash: Some(B256::ZERO),
///     })
///     .build();
/// ```
#[derive(Debug)]
pub struct MockEngineClientBuilder {
    cfg: Option<Arc<RollupConfig>>,
    storage: MockEngineStorage,
}

impl MockEngineClientBuilder {
    /// Creates a new builder with default values.
    pub fn new() -> Self {
        Self { cfg: None, storage: MockEngineStorage::default() }
    }

    /// Sets the rollup configuration.
    pub fn with_config(mut self, cfg: Arc<RollupConfig>) -> Self {
        self.cfg = Some(cfg);
        self
    }

    /// Sets a block response for a specific tag.
    pub fn with_l2_block_by_label(
        mut self,
        tag: BlockNumberOrTag,
        block: Block<OpTransaction>,
    ) -> Self {
        self.storage.l2_blocks_by_label.insert(tag, block);
        self
    }

    /// Sets a block info response for a specific tag.
    pub fn with_block_info_by_tag(mut self, tag: BlockNumberOrTag, info: L2BlockInfo) -> Self {
        self.storage.block_info_by_tag.insert(tag, info);
        self
    }

    /// Sets the new_payload_v1 response.
    pub fn with_new_payload_v1_response(mut self, status: PayloadStatus) -> Self {
        self.storage.new_payload_v1_response = Some(status);
        self
    }

    /// Sets the new_payload_v2 response.
    pub fn with_new_payload_v2_response(mut self, status: PayloadStatus) -> Self {
        self.storage.new_payload_v2_response = Some(status);
        self
    }

    /// Sets the new_payload_v3 response.
    pub fn with_new_payload_v3_response(mut self, status: PayloadStatus) -> Self {
        self.storage.new_payload_v3_response = Some(status);
        self
    }

    /// Sets the new_payload_v4 response.
    pub fn with_new_payload_v4_response(mut self, status: PayloadStatus) -> Self {
        self.storage.new_payload_v4_response = Some(status);
        self
    }

    /// Sets the fork_choice_updated_v2 response.
    pub fn with_fork_choice_updated_v2_response(mut self, response: ForkchoiceUpdated) -> Self {
        self.storage.fork_choice_updated_v2_response = Some(response);
        self
    }

    /// Sets the fork_choice_updated_v3 response.
    pub fn with_fork_choice_updated_v3_response(mut self, response: ForkchoiceUpdated) -> Self {
        self.storage.fork_choice_updated_v3_response = Some(response);
        self
    }

    /// Sets the execution payload v2 response.
    pub fn with_execution_payload_v2(mut self, payload: ExecutionPayloadEnvelopeV2) -> Self {
        self.storage.execution_payload_v2 = Some(payload);
        self
    }

    /// Sets the execution payload v3 response.
    pub fn with_execution_payload_v3(mut self, payload: OpExecutionPayloadEnvelopeV3) -> Self {
        self.storage.execution_payload_v3 = Some(payload);
        self
    }

    /// Sets the execution payload v4 response.
    pub fn with_execution_payload_v4(mut self, payload: OpExecutionPayloadEnvelopeV4) -> Self {
        self.storage.execution_payload_v4 = Some(payload);
        self
    }

    /// Sets the get_payload_bodies_by_hash_v1 response.
    pub fn with_payload_bodies_by_hash_response(
        mut self,
        bodies: ExecutionPayloadBodiesV1,
    ) -> Self {
        self.storage.get_payload_bodies_by_hash_v1_response = Some(bodies);
        self
    }

    /// Sets the get_payload_bodies_by_range_v1 response.
    pub fn with_payload_bodies_by_range_response(
        mut self,
        bodies: ExecutionPayloadBodiesV1,
    ) -> Self {
        self.storage.get_payload_bodies_by_range_v1_response = Some(bodies);
        self
    }

    /// Sets the client versions response.
    pub fn with_client_versions(mut self, versions: Vec<ClientVersionV1>) -> Self {
        self.storage.client_versions = Some(versions);
        self
    }

    /// Sets the protocol version response.
    pub const fn with_protocol_version(mut self, version: ProtocolVersion) -> Self {
        self.storage.protocol_version = Some(version);
        self
    }

    /// Sets the capabilities response.
    pub fn with_capabilities(mut self, capabilities: Vec<String>) -> Self {
        self.storage.capabilities = Some(capabilities);
        self
    }

    /// Sets an L1 block response for a specific BlockId.
    pub fn with_l1_block(mut self, block_id: BlockId, block: Block<EthTransaction>) -> Self {
        let key = block_id_to_key(&block_id);
        self.storage.l1_blocks_by_id.insert(key, block);
        self
    }

    /// Sets an L2 block response for a specific BlockId.
    pub fn with_l2_block(mut self, block_id: BlockId, block: Block<OpTransaction>) -> Self {
        let key = block_id_to_key(&block_id);
        self.storage.l2_blocks_by_id.insert(key, block);
        self
    }

    /// Sets a proof response for a specific address and BlockId.
    pub fn with_proof(
        mut self,
        address: Address,
        block_id: BlockId,
        proof: EIP1186AccountProofResponse,
    ) -> Self {
        let key = block_id_to_key(&block_id);
        self.storage.proofs_by_address.insert((address, key), proof);
        self
    }

    /// Builds the [`MockEngineClient`] with the configured values.
    ///
    /// # Panics
    ///
    /// Panics if any required fields (cfg) are not set.
    pub fn build(self) -> MockEngineClient {
        let cfg = self.cfg.expect("cfg must be set");

        MockEngineClient { cfg, storage: Arc::new(RwLock::new(self.storage)) }
    }
}

impl Default for MockEngineClientBuilder {
    fn default() -> Self {
        Self::new()
    }
}

/// Mock implementation of the EngineClient trait for testing.
///
/// This mock allows tests to configure expected responses for all EngineClient
/// and OpEngineApi methods. All responses are stored in a shared [`MockEngineStorage`]
/// protected by an RwLock for thread-safe access.
#[derive(Debug, Clone)]
pub struct MockEngineClient {
    /// The rollup configuration.
    cfg: Arc<RollupConfig>,
    /// Shared storage for mock responses.
    storage: Arc<RwLock<MockEngineStorage>>,
}

impl MockEngineClient {
    /// Creates a new mock engine client with the given config.
    pub fn new(cfg: Arc<RollupConfig>) -> Self {
        Self { cfg, storage: Arc::new(RwLock::new(MockEngineStorage::default())) }
    }

    /// Creates a builder for constructing a mock engine client.
    pub fn builder() -> MockEngineClientBuilder {
        MockEngineClientBuilder::new()
    }

    /// Returns a reference to the mock storage for configuring responses.
    pub fn storage(&self) -> Arc<RwLock<MockEngineStorage>> {
        Arc::clone(&self.storage)
    }

    /// Sets a block response for a specific tag.
    pub async fn set_l2_block_by_label(&self, tag: BlockNumberOrTag, block: Block<OpTransaction>) {
        self.storage.write().await.l2_blocks_by_label.insert(tag, block);
    }

    /// Sets a block info response for a specific tag.
    pub async fn set_block_info_by_tag(&self, tag: BlockNumberOrTag, info: L2BlockInfo) {
        self.storage.write().await.block_info_by_tag.insert(tag, info);
    }

    /// Sets the new_payload_v1 response.
    pub async fn set_new_payload_v1_response(&self, status: PayloadStatus) {
        self.storage.write().await.new_payload_v1_response = Some(status);
    }

    /// Sets the new_payload_v2 response.
    pub async fn set_new_payload_v2_response(&self, status: PayloadStatus) {
        self.storage.write().await.new_payload_v2_response = Some(status);
    }

    /// Sets the new_payload_v3 response.
    pub async fn set_new_payload_v3_response(&self, status: PayloadStatus) {
        self.storage.write().await.new_payload_v3_response = Some(status);
    }

    /// Sets the new_payload_v4 response.
    pub async fn set_new_payload_v4_response(&self, status: PayloadStatus) {
        self.storage.write().await.new_payload_v4_response = Some(status);
    }

    /// Sets the fork_choice_updated_v2 response.
    pub async fn set_fork_choice_updated_v2_response(&self, response: ForkchoiceUpdated) {
        self.storage.write().await.fork_choice_updated_v2_response = Some(response);
    }

    /// Sets the fork_choice_updated_v3 response.
    pub async fn set_fork_choice_updated_v3_response(&self, response: ForkchoiceUpdated) {
        self.storage.write().await.fork_choice_updated_v3_response = Some(response);
    }

    /// Sets the execution payload v2 response.
    pub async fn set_execution_payload_v2(&self, payload: ExecutionPayloadEnvelopeV2) {
        self.storage.write().await.execution_payload_v2 = Some(payload);
    }

    /// Sets the execution payload v3 response.
    pub async fn set_execution_payload_v3(&self, payload: OpExecutionPayloadEnvelopeV3) {
        self.storage.write().await.execution_payload_v3 = Some(payload);
    }

    /// Sets the execution payload v4 response.
    pub async fn set_execution_payload_v4(&self, payload: OpExecutionPayloadEnvelopeV4) {
        self.storage.write().await.execution_payload_v4 = Some(payload);
    }

    /// Sets the get_payload_bodies_by_hash_v1 response.
    pub async fn set_payload_bodies_by_hash_response(&self, bodies: ExecutionPayloadBodiesV1) {
        self.storage.write().await.get_payload_bodies_by_hash_v1_response = Some(bodies);
    }

    /// Sets the get_payload_bodies_by_range_v1 response.
    pub async fn set_payload_bodies_by_range_response(&self, bodies: ExecutionPayloadBodiesV1) {
        self.storage.write().await.get_payload_bodies_by_range_v1_response = Some(bodies);
    }

    /// Sets the client versions response.
    pub async fn set_client_versions(&self, versions: Vec<ClientVersionV1>) {
        self.storage.write().await.client_versions = Some(versions);
    }

    /// Sets the protocol version response.
    pub async fn set_protocol_version(&self, version: ProtocolVersion) {
        self.storage.write().await.protocol_version = Some(version);
    }

    /// Sets the capabilities response.
    pub async fn set_capabilities(&self, capabilities: Vec<String>) {
        self.storage.write().await.capabilities = Some(capabilities);
    }

    /// Sets an L1 block response for a specific BlockId.
    pub async fn set_l1_block(&self, block_id: BlockId, block: Block<EthTransaction>) {
        let key = block_id_to_key(&block_id);
        self.storage.write().await.l1_blocks_by_id.insert(key, block);
    }

    /// Sets an L2 block response for a specific BlockId.
    pub async fn set_l2_block(&self, block_id: BlockId, block: Block<OpTransaction>) {
        let key = block_id_to_key(&block_id);
        self.storage.write().await.l2_blocks_by_id.insert(key, block);
    }

    /// Sets a proof response for a specific address and BlockId.
    pub async fn set_proof(
        &self,
        address: Address,
        block_id: BlockId,
        proof: EIP1186AccountProofResponse,
    ) {
        let key = block_id_to_key(&block_id);
        self.storage.write().await.proofs_by_address.insert((address, key), proof);
    }
}

#[async_trait]
impl EngineClient for MockEngineClient {
    fn cfg(&self) -> &RollupConfig {
        self.cfg.as_ref()
    }

    fn get_l1_block(&self, block: BlockId) -> EthGetBlock<<Ethereum as Network>::BlockResponse> {
        let storage = Arc::clone(&self.storage);
        let block_key = block_id_to_key(&block);

        EthGetBlock::new_provider(
            block,
            Box::new(move |_kind| {
                let storage = Arc::clone(&storage);
                let block_key = block_key.clone();

                ProviderCall::BoxedFuture(Box::pin(async move {
                    let storage_guard = storage.read().await;
                    Ok(storage_guard.l1_blocks_by_id.get(&block_key).cloned())
                }))
            }),
        )
    }

    fn get_l2_block(&self, block: BlockId) -> EthGetBlock<<Optimism as Network>::BlockResponse> {
        let storage = Arc::clone(&self.storage);
        let block_key = block_id_to_key(&block);

        EthGetBlock::new_provider(
            block,
            Box::new(move |_kind| {
                let storage = Arc::clone(&storage);
                let block_key = block_key.clone();

                ProviderCall::BoxedFuture(Box::pin(async move {
                    let storage_guard = storage.read().await;
                    Ok(storage_guard.l2_blocks_by_id.get(&block_key).cloned())
                }))
            }),
        )
    }

    fn get_proof(
        &self,
        address: Address,
        _keys: Vec<StorageKey>,
    ) -> RpcWithBlock<(Address, Vec<StorageKey>), EIP1186AccountProofResponse> {
        let storage = Arc::clone(&self.storage);

        RpcWithBlock::new_provider(move |block_id| {
            let storage = Arc::clone(&storage);
            let block_key = block_id_to_key(&block_id);
            let address = address;

            ProviderCall::BoxedFuture(Box::pin(async move {
                let storage_guard = storage.read().await;
                storage_guard.proofs_by_address.get(&(address, block_key)).cloned().ok_or_else(
                    || {
                        TransportError::from(TransportErrorKind::custom_str(
                            "No proof configured for this address and block. \
                             Use with_proof() or set_proof() to set a response.",
                        ))
                    },
                )
            }))
        })
    }

    async fn new_payload_v1(&self, _payload: ExecutionPayloadV1) -> TransportResult<PayloadStatus> {
        let storage = self.storage.read().await;
        storage.new_payload_v1_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "new_payload_v1 was called but no v1 response configured. \
                 Use with_new_payload_v1_response() or set_new_payload_v1_response() to set a response."
            ))
        })
    }

    async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<OpTransaction>>, EngineClientError> {
        let storage = self.storage.read().await;
        Ok(storage.l2_blocks_by_label.get(&numtag).cloned())
    }

    async fn l2_block_info_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<L2BlockInfo>, EngineClientError> {
        let storage = self.storage.read().await;
        Ok(storage.block_info_by_tag.get(&numtag).cloned())
    }
}

#[async_trait]
impl OpEngineApi<Optimism, Http<HyperAuthClient>> for MockEngineClient {
    async fn new_payload_v2(
        &self,
        _payload: ExecutionPayloadInputV2,
    ) -> TransportResult<PayloadStatus> {
        let storage = self.storage.read().await;
        storage.new_payload_v2_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "new_payload_v2 was called but no v2 response configured. \
                 Use with_new_payload_v2_response() or set_new_payload_v2_response() to set a response."
            ))
        })
    }

    async fn new_payload_v3(
        &self,
        _payload: ExecutionPayloadV3,
        _parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        let storage = self.storage.read().await;
        storage.new_payload_v3_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "new_payload_v3 was called but no v3 response configured. \
                 Use with_new_payload_v3_response() or set_new_payload_v3_response() to set a response."
            ))
        })
    }

    async fn new_payload_v4(
        &self,
        _payload: OpExecutionPayloadV4,
        _parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        let storage = self.storage.read().await;
        storage.new_payload_v4_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "new_payload_v4 was called but no v4 response configured. \
                 Use with_new_payload_v4_response() or set_new_payload_v4_response() to set a response."
            ))
        })
    }

    async fn fork_choice_updated_v2(
        &self,
        _fork_choice_state: ForkchoiceState,
        _payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let storage = self.storage.read().await;
        storage.fork_choice_updated_v2_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "fork_choice_updated_v2 was called but no v2 response configured. \
                 Use with_fork_choice_updated_v2_response() or set_fork_choice_updated_v2_response() to set a response."
            ))
        })
    }

    async fn fork_choice_updated_v3(
        &self,
        _fork_choice_state: ForkchoiceState,
        _payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let storage = self.storage.read().await;
        storage.fork_choice_updated_v3_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "fork_choice_updated_v3 was called but no v3 response configured. \
                 Use with_fork_choice_updated_v3_response() or set_fork_choice_updated_v3_response() to set a response."
            ))
        })
    }

    async fn get_payload_v2(
        &self,
        _payload_id: PayloadId,
    ) -> TransportResult<ExecutionPayloadEnvelopeV2> {
        let storage = self.storage.read().await;
        storage.execution_payload_v2.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "No execution payload v2 set in mock",
            ))
        })
    }

    async fn get_payload_v3(
        &self,
        _payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV3> {
        let storage = self.storage.read().await;
        storage.execution_payload_v3.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "No execution payload v3 set in mock",
            ))
        })
    }

    async fn get_payload_v4(
        &self,
        _payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV4> {
        let storage = self.storage.read().await;
        storage.execution_payload_v4.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "No execution payload v4 set in mock",
            ))
        })
    }

    async fn get_payload_bodies_by_hash_v1(
        &self,
        _block_hashes: Vec<BlockHash>,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        let storage = self.storage.read().await;
        storage.get_payload_bodies_by_hash_v1_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "get_payload_bodies_by_hash_v1 was called but no response configured. \
                 Use with_payload_bodies_by_hash_response() or set_payload_bodies_by_hash_response() to set a response."
            ))
        })
    }

    async fn get_payload_bodies_by_range_v1(
        &self,
        _start: u64,
        _count: u64,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        let storage = self.storage.read().await;
        storage.get_payload_bodies_by_range_v1_response.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str(
                "get_payload_bodies_by_range_v1 was called but no response configured. \
                 Use with_payload_bodies_by_range_response() or set_payload_bodies_by_range_response() to set a response."
            ))
        })
    }

    async fn get_client_version_v1(
        &self,
        _client_version: ClientVersionV1,
    ) -> TransportResult<Vec<ClientVersionV1>> {
        let storage = self.storage.read().await;
        storage.client_versions.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str("No client versions set in mock"))
        })
    }

    async fn signal_superchain_v1(
        &self,
        _recommended: ProtocolVersion,
        _required: ProtocolVersion,
    ) -> TransportResult<ProtocolVersion> {
        let storage = self.storage.read().await;
        storage.protocol_version.ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str("No protocol version set in mock"))
        })
    }

    async fn exchange_capabilities(
        &self,
        _capabilities: Vec<String>,
    ) -> TransportResult<Vec<String>> {
        let storage = self.storage.read().await;
        storage.capabilities.clone().ok_or_else(|| {
            TransportError::from(TransportErrorKind::custom_str("No capabilities set in mock"))
        })
    }
}

/// Helper function to convert BlockId to a string key for HashMap storage.
/// This is necessary because BlockId doesn't implement Hash.
fn block_id_to_key(block_id: &BlockId) -> String {
    match block_id {
        BlockId::Hash(hash) => format!("hash:{}", hash.block_hash),
        BlockId::Number(num) => format!("number:{num}"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_types_engine::PayloadStatusEnum;

    #[tokio::test]
    async fn test_mock_engine_client_creation() {
        let cfg = Arc::new(RollupConfig::default());

        let mock = MockEngineClient::new(cfg.clone());

        // Verify the config was set correctly
        assert_eq!(mock.cfg().block_time, cfg.block_time);
    }

    #[tokio::test]
    async fn test_mock_payload_status() {
        let cfg = Arc::new(RollupConfig::default());

        let mock = MockEngineClient::new(cfg);

        let status =
            PayloadStatus { status: PayloadStatusEnum::Valid, latest_valid_hash: Some(B256::ZERO) };

        mock.set_new_payload_v2_response(status.clone()).await;

        // Create a minimal ExecutionPayloadInputV2 for testing
        use alloy_primitives::{Bytes, U256};
        use alloy_rpc_types_engine::ExecutionPayloadV1;
        let payload = ExecutionPayloadInputV2 {
            execution_payload: ExecutionPayloadV1 {
                parent_hash: B256::ZERO,
                fee_recipient: Default::default(),
                state_root: B256::ZERO,
                receipts_root: B256::ZERO,
                logs_bloom: Default::default(),
                prev_randao: B256::ZERO,
                block_number: 0,
                gas_limit: 0,
                gas_used: 0,
                timestamp: 0,
                extra_data: Bytes::new(),
                base_fee_per_gas: U256::ZERO,
                block_hash: B256::ZERO,
                transactions: vec![],
            },
            withdrawals: None,
        };

        let result = mock.new_payload_v2(payload).await.unwrap();

        assert_eq!(result.status, status.status);
    }

    #[tokio::test]
    async fn test_mock_forkchoice_updated() {
        let cfg = Arc::new(RollupConfig::default());

        let mock = MockEngineClient::new(cfg);

        let fcu = ForkchoiceUpdated {
            payload_status: PayloadStatus {
                status: PayloadStatusEnum::Valid,
                latest_valid_hash: Some(B256::ZERO),
            },
            payload_id: None,
        };

        mock.set_fork_choice_updated_v2_response(fcu.clone()).await;

        let result = mock.fork_choice_updated_v2(ForkchoiceState::default(), None).await.unwrap();

        assert_eq!(result.payload_status.status, fcu.payload_status.status);
    }

    #[tokio::test]
    async fn test_builder_pattern() {
        let cfg = Arc::new(RollupConfig::default());
        let status =
            PayloadStatus { status: PayloadStatusEnum::Valid, latest_valid_hash: Some(B256::ZERO) };

        let mock = MockEngineClient::builder()
            .with_config(cfg.clone())
            .with_new_payload_v2_response(status.clone())
            .build();

        // Verify the config was set
        assert_eq!(mock.cfg().block_time, cfg.block_time);

        // Create a minimal ExecutionPayloadInputV2 for testing
        use alloy_primitives::{Bytes, U256};
        use alloy_rpc_types_engine::ExecutionPayloadV1;
        let payload = ExecutionPayloadInputV2 {
            execution_payload: ExecutionPayloadV1 {
                parent_hash: B256::ZERO,
                fee_recipient: Default::default(),
                state_root: B256::ZERO,
                receipts_root: B256::ZERO,
                logs_bloom: Default::default(),
                prev_randao: B256::ZERO,
                block_number: 0,
                gas_limit: 0,
                gas_used: 0,
                timestamp: 0,
                extra_data: Bytes::new(),
                base_fee_per_gas: U256::ZERO,
                block_hash: B256::ZERO,
                transactions: vec![],
            },
            withdrawals: None,
        };

        // Verify the pre-configured response is returned
        let result = mock.new_payload_v2(payload).await.unwrap();
        assert_eq!(result.status, status.status);
    }
}
