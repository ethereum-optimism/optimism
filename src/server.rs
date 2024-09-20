use std::sync::Arc;

use alloy::eips::{BlockId, BlockNumberOrTag};
use alloy::primitives::{Address, B256};
use alloy::rpc::types::serde_helpers::JsonStorageKey;
use alloy_primitives::U64;
use alloy_rpc_types_engine::{
    ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadAttributes, PayloadId,
    PayloadStatus,
};
use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse};
use jsonrpsee::core::{async_trait, ClientError, RpcResult};
use jsonrpsee::http_client::HttpClient;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee::types::ErrorCode;
use tracing::{error, info};

#[rpc(server, client, namespace = "eth")]
pub trait EthApi {
    /// Returns the chain ID of the current network.
    #[method(name = "chainId")]
    async fn chain_id(&self) -> RpcResult<Option<U64>>;

    /// Returns information about a block by number.
    #[method(name = "getBlockByNumber")]
    async fn block_by_number(
        &self,
        number: BlockNumberOrTag,
        full: bool,
    ) -> RpcResult<Option<Block>>;

    /// Returns information about a block by hash.
    #[method(name = "getBlockByHash")]
    async fn block_by_hash(&self, hash: B256, full: bool) -> RpcResult<Option<Block>>;

    /// Returns the account and storage values of the specified account including the Merkle-proof.
    /// This call can be used to verify that the data you are pulling from is not tampered with.
    #[method(name = "getProof")]
    async fn get_proof(
        &self,
        address: Address,
        keys: Vec<JsonStorageKey>,
        block_number: Option<BlockId>,
    ) -> RpcResult<EIP1186AccountProofResponse>;
}

#[rpc(server, client, namespace = "engine")]
pub trait EngineApi {
    #[method(name = "forkchoiceUpdatedV3")]
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<PayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    #[method(name = "getPayloadV3")]
    async fn get_payload_v3(&self, payload_id: PayloadId) -> RpcResult<ExecutionPayloadV3>;

    #[method(name = "newPayloadV3")]
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus>;
}

pub struct EthEngineApi {
    l2_client: Arc<HttpClient>,
    builder_client: Arc<HttpClient>,
}

impl EthEngineApi {
    pub fn new(l2_client: Arc<HttpClient>, builder_client: Arc<HttpClient>) -> Self {
        Self {
            l2_client,
            builder_client,
        }
    }
}

#[async_trait]
impl EthApiServer for EthEngineApi {
    async fn chain_id(&self) -> RpcResult<Option<U64>> {
        self.l2_client.chain_id().await.map_err(|e| match e {
            ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
            other_error => {
                error!(
                    "Error calling chain_id from execution engine: {:?}",
                    other_error
                );
                ErrorCode::InternalError.into()
            }
        })
    }

    async fn block_by_number(
        &self,
        number: BlockNumberOrTag,
        full: bool,
    ) -> RpcResult<Option<Block>> {
        self.l2_client
            .block_by_number(number, full)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        "Error calling block_by_number from execution engine: {:?}",
                        other_error
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }

    async fn block_by_hash(&self, hash: B256, full: bool) -> RpcResult<Option<Block>> {
        self.l2_client
            .block_by_hash(hash, full)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        "Error calling block_by_hash from execution engine: {:?}",
                        other_error
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }

    async fn get_proof(
        &self,
        address: Address,
        keys: Vec<JsonStorageKey>,
        block_number: Option<BlockId>,
    ) -> RpcResult<EIP1186AccountProofResponse> {
        self.l2_client
            .get_proof(address, keys, block_number)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        "Error calling get_proof from execution engine: {:?}",
                        other_error
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }
}

#[async_trait]
impl EngineApiServer for EthEngineApi {
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<PayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        info!("fork_choice_updated_v3");
        self.l2_client
            .fork_choice_updated_v3(fork_choice_state, payload_attributes)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        "Error calling fork_choice_updated_v3 from execution engine: {:?}",
                        other_error
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }

    async fn get_payload_v3(&self, payload_id: PayloadId) -> RpcResult<ExecutionPayloadV3> {
        info!("get_payload_v3");
        self.l2_client
            .get_payload_v3(payload_id)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        "Error calling get_payload_v3 from execution engine: {:?}",
                        other_error
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }

    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus> {
        info!("new_payload_v3");
        self.l2_client
            .new_payload_v3(payload, versioned_hashes, parent_beacon_block_root)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        "Error calling new_payload_v3 from execution engine: {:?}",
                        other_error
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }
}
