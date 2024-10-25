use alloy::primitives::B256;
use alloy_primitives::Bytes;
use alloy_rpc_types_engine::{
    ExecutionPayload, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId,
    PayloadStatus,
};
use jsonrpsee::core::{async_trait, ClientError, RpcResult};
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::HttpClient;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee::types::error::INVALID_REQUEST_CODE;
use jsonrpsee::types::{ErrorCode, ErrorObject};
use op_alloy_rpc_types_engine::{
    AsInnerPayload, OptimismExecutionPayloadEnvelopeV3, OptimismPayloadAttributes,
};
use reth_rpc_layer::AuthClientService;
use std::sync::Arc;
use tracing::{debug, error, info};

use crate::metrics::ServerMetrics;

#[rpc(server, client, namespace = "engine")]
pub trait EngineApi {
    #[method(name = "forkchoiceUpdatedV3")]
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OptimismPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    #[method(name = "getPayloadV3")]
    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OptimismExecutionPayloadEnvelopeV3>;

    #[method(name = "newPayloadV3")]
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus>;
}

#[rpc(server, client, namespace = "eth")]
pub trait EthApi {
    #[method(name = "sendRawTransaction")]
    async fn send_raw_transaction(&self, bytes: Bytes) -> RpcResult<B256>;
}

pub struct EthEngineApi<S = AuthClientService<HttpBackend>> {
    l2_client: Arc<HttpClient<S>>,
    builder_client: Arc<HttpClient<S>>,
    boost_sync: bool,
    metrics: Option<Arc<ServerMetrics>>,
}

impl<S> EthEngineApi<S> {
    pub fn new(
        l2_client: Arc<HttpClient<S>>,
        builder_client: Arc<HttpClient<S>>,
        boost_sync: bool,
        metrics: Option<Arc<ServerMetrics>>,
    ) -> Self {
        Self {
            l2_client,
            builder_client,
            boost_sync,
            metrics,
        }
    }
}

#[async_trait]
impl EthApiServer for EthEngineApi {
    async fn send_raw_transaction(&self, bytes: Bytes) -> RpcResult<B256> {
        debug!(
            message = "received send_raw_transaction",
            "bytes_len" = bytes.len()
        );

        if let Some(metrics) = &self.metrics {
            metrics.send_raw_tx_count.increment(1);
        }

        let builder = self.builder_client.clone();
        let tx_bytes = bytes.clone();
        tokio::spawn(async move {
            builder.send_raw_transaction(tx_bytes).await.map_err(|e| {
                error!(message = "error calling send_raw_transaction for builder", "error" = %e);
            })
        });

        self.l2_client
            .send_raw_transaction(bytes)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        message = "error calling send_raw_transaction for l2 client",
                        "error" = %other_error,
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
        payload_attributes: Option<OptimismPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        info!(
            message = "received fork_choice_updated_v3",
            "head_block_hash" = %fork_choice_state.head_block_hash,
            "has_attributes" = payload_attributes.is_some(),
        );

        let use_tx_pool = payload_attributes
            .as_ref()
            .map(|attr| !attr.no_tx_pool.unwrap_or_default());
        let should_send_to_builder = if self.boost_sync {
            // don't send to builder only if no_tx_pool is set
            use_tx_pool.unwrap_or(true)
        } else {
            // send to builder if there are payload attributes and no_tx_pool is not set
            use_tx_pool.is_some()
        };

        if should_send_to_builder {
            // async call to builder to trigger payload building and sync
            if let Some(metrics) = &self.metrics {
                metrics.fcu_count.increment(1);
            }
            let builder = self.builder_client.clone();
            let attr = payload_attributes.clone();
            tokio::spawn(async move {
                builder.fork_choice_updated_v3(fork_choice_state, attr).await.map(|response| {
                    let payload_id_str = response.payload_id.map(|id| id.to_string()).unwrap_or_default();
                    if response.is_invalid() {
                        error!(message = "builder rejected fork_choice_updated_v3 with attributes", "payload_id" = payload_id_str, "validation_error" = %response.payload_status.status);
                    } else {
                        info!(message = "called fork_choice_updated_v3 to builder with payload attributes", "payload_status" = %response.payload_status.status, "payload_id" = payload_id_str);
                    }
                }).map_err(|e| {
                    error!(message = "error calling fork_choice_updated_v3 to builder", "error" = %e, "head_block_hash" = %fork_choice_state.head_block_hash);
                })
            });
        } else {
            info!(message = "no payload attributes provided or no_tx_pool is set", "head_block_hash" = %fork_choice_state.head_block_hash);
        }

        self.l2_client
            .fork_choice_updated_v3(fork_choice_state, payload_attributes)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        message = "error calling fork_choice_updated_v3 for l2 client",
                        "error" = %other_error,
                        "head_block_hash" = %fork_choice_state.head_block_hash,
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }

    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OptimismExecutionPayloadEnvelopeV3> {
        info!(message = "received get_payload_v3", "payload_id" = %payload_id);
        let l2_client_future = self.l2_client.get_payload_v3(payload_id);
        let builder_client_future = Box::pin(async move {
            if let Some(metrics) = &self.metrics {
                metrics.get_payload_count.increment(1);
            }
            let builder = self.builder_client.clone();
            let payload = builder.get_payload_v3(payload_id).await.map_err(|e| {
                error!(message = "error calling get_payload_v3 from builder", "error" = %e, "payload_id" = %payload_id);
                e
                })?;

            info!(message = "received payload from builder", "payload_id" = %payload_id, "block_hash" = %payload.as_v1_payload().block_hash);

            // Send the payload to the local execution engine with engine_newPayload to validate the block from the builder.
            // Otherwise, we do not want to risk the network to a halt since op-node will not be able to propose the block.
            // If validation fails, return the local block since that one has already been validated.
            if let Some(metrics) = &self.metrics {
                metrics.new_payload_count.increment(1);
            }
            let payload_status = self.l2_client.new_payload_v3(payload.execution_payload.clone(), vec![], payload.parent_beacon_block_root).await.map_err(|e| {
                error!(message = "error calling new_payload_v3 to validate builder payload", "error" = %e, "payload_id" = %payload_id);
                e
            })?;
            if payload_status.is_invalid() {
                error!(message = "builder payload was not valid", "payload_status" = %payload_status.status, "payload_id" = %payload_id);
                Err(ClientError::Call(ErrorObject::owned(
                    INVALID_REQUEST_CODE,
                    "Builder payload was not valid",
                    None::<String>,
                )))
            } else {
                info!(message = "received payload status from local execution engine validating builder payload", "payload_id" = %payload_id);
                Ok(payload)
            }
        });

        let (l2_payload, builder_payload) = tokio::join!(l2_client_future, builder_client_future);

        builder_payload.or(l2_payload).map_err(|e| match e {
            ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
            other_error => {
                error!(
                    message = "error calling get_payload_v3",
                    "error" = %other_error,
                    "payload_id" = %payload_id
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
        let block_hash = ExecutionPayload::from(payload.clone()).block_hash();
        info!(message = "received new_payload_v3", "block_hash" = %block_hash);

        // async call to builder to sync the builder node
        if self.boost_sync {
            if let Some(metrics) = &self.metrics {
                metrics.new_payload_count.increment(1);
            }

            let builder = self.builder_client.clone();
            let builder_payload = payload.clone();
            let builder_versioned_hashes = versioned_hashes.clone();
            tokio::spawn(async move {
                builder.new_payload_v3(builder_payload, builder_versioned_hashes, parent_beacon_block_root).await
                .map(|response: PayloadStatus| {
                    if response.is_invalid() {
                        error!(message = "builder rejected new_payload_v3", "block_hash" = %block_hash);
                    } else {
                        info!(message = "called new_payload_v3 to builder", "payload_status" = %response.status, "block_hash" = %block_hash);
                    }
                }).map_err(|e| {
                    error!(message = "error calling new_payload_v3 to builder", "error" = %e, "block_hash" = %block_hash);
                    e
                })
            });
        }

        self.l2_client
            .new_payload_v3(payload, versioned_hashes, parent_beacon_block_root)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err, // Already an ErrorObjectOwned, so just return it
                other_error => {
                    error!(
                        message = "error calling new_payload_v3",
                        "error" = %other_error,
                        "block_hash" = %block_hash
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }
}
