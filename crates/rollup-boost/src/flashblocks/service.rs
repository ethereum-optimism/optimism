use super::outbound::WebSocketPublisher;
use crate::flashblocks::metrics::FlashblocksServiceMetrics;
use crate::{
    ClientResult, EngineApiExt, NewPayload, OpExecutionPayloadEnvelope, PayloadVersion, RpcClient,
};
use alloy_primitives::U256;
use alloy_rpc_types_engine::{
    BlobsBundleV1, ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3,
};
use alloy_rpc_types_engine::{ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus};
use alloy_rpc_types_eth::{Block, BlockNumberOrTag};
use core::net::SocketAddr;
use jsonrpsee::core::async_trait;
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes,
};
use op_alloy_rpc_types_engine::{
    OpFlashblockPayload, OpFlashblockPayloadBase, OpFlashblockPayloadDelta,
};
use std::io;
use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};
use thiserror::Error;
use tokio::sync::RwLock;
use tokio::sync::mpsc;
use tracing::{debug, error, info};

#[derive(Debug, Error, PartialEq)]
pub enum FlashblocksError {
    #[error("Missing base payload for initial flashblock")]
    MissingBasePayload,
    #[error("Unexpected base payload for non-initial flashblock")]
    UnexpectedBasePayload,
    #[error("Missing delta for flashblock")]
    MissingDelta,
    #[error("Invalid index for flashblock")]
    InvalidIndex,
    #[error("Missing payload")]
    MissingPayload,
}

// Simplify actor messages to just handle shutdown
#[derive(Debug)]
enum FlashblocksEngineMessage {
    OpFlashblockPayload(OpFlashblockPayload),
}

#[derive(Clone, Debug, Default)]
pub struct FlashblockBuilder {
    base: Option<OpFlashblockPayloadBase>,
    flashblocks: Vec<OpFlashblockPayloadDelta>,
}

impl FlashblockBuilder {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn extend(&mut self, payload: OpFlashblockPayload) -> Result<(), FlashblocksError> {
        tracing::debug!(message = "Extending payload", payload_id = %payload.payload_id, index = payload.index, has_base=payload.base.is_some());

        // Validate the index is contiguous
        if payload.index != self.flashblocks.len() as u64 {
            return Err(FlashblocksError::InvalidIndex);
        }

        // Check base payload rules
        if payload.index == 0 {
            if let Some(base) = payload.base {
                self.base = Some(base)
            } else {
                return Err(FlashblocksError::MissingBasePayload);
            }
        } else if payload.base.is_some() {
            return Err(FlashblocksError::UnexpectedBasePayload);
        }

        // Update latest diff and accumulate transactions and withdrawals
        self.flashblocks.push(payload.diff);

        Ok(())
    }

    pub fn into_envelope(
        self,
        version: PayloadVersion,
    ) -> Result<OpExecutionPayloadEnvelope, FlashblocksError> {
        self.build_envelope(version)
    }

    pub fn build_envelope(
        &self,
        version: PayloadVersion,
    ) -> Result<OpExecutionPayloadEnvelope, FlashblocksError> {
        let base = self.base.as_ref().ok_or(FlashblocksError::MissingPayload)?;

        // There must be at least one delta
        let diff = self
            .flashblocks
            .last()
            .ok_or(FlashblocksError::MissingDelta)?;

        let (transactions, withdrawals) = self.flashblocks.iter().fold(
            (Vec::new(), Vec::new()),
            |(mut transactions, mut withdrawals), delta| {
                transactions.extend(delta.transactions.clone());
                withdrawals.extend(delta.withdrawals.clone());
                (transactions, withdrawals)
            },
        );

        let withdrawals_root = diff.withdrawals_root;

        let execution_payload = ExecutionPayloadV3 {
            blob_gas_used: diff.blob_gas_used.unwrap_or(0),
            excess_blob_gas: 0,
            payload_inner: ExecutionPayloadV2 {
                withdrawals,
                payload_inner: ExecutionPayloadV1 {
                    parent_hash: base.parent_hash,
                    fee_recipient: base.fee_recipient,
                    state_root: diff.state_root,
                    receipts_root: diff.receipts_root,
                    logs_bloom: diff.logs_bloom,
                    prev_randao: base.prev_randao,
                    block_number: base.block_number,
                    gas_limit: base.gas_limit,
                    gas_used: diff.gas_used,
                    timestamp: base.timestamp,
                    extra_data: base.extra_data.clone(),
                    base_fee_per_gas: base.base_fee_per_gas,
                    block_hash: diff.block_hash,
                    transactions,
                },
            },
        };

        match version {
            PayloadVersion::V3 => Ok(OpExecutionPayloadEnvelope::V3(
                OpExecutionPayloadEnvelopeV3 {
                    parent_beacon_block_root: base.parent_beacon_block_root,
                    block_value: U256::ZERO,
                    blobs_bundle: BlobsBundleV1::default(),
                    should_override_builder: false,
                    execution_payload,
                },
            )),
            PayloadVersion::V4 => Ok(OpExecutionPayloadEnvelope::V4(
                OpExecutionPayloadEnvelopeV4 {
                    parent_beacon_block_root: base.parent_beacon_block_root,
                    block_value: U256::ZERO,
                    blobs_bundle: BlobsBundleV1::default(),
                    should_override_builder: false,
                    execution_payload: OpExecutionPayloadV4 {
                        withdrawals_root,
                        payload_inner: execution_payload,
                    },
                    execution_requests: vec![],
                },
            )),
        }
    }
}

#[derive(Clone, Debug)]
pub struct FlashblocksService {
    client: RpcClient,

    /// Current payload ID we're processing. Set from local payload calculation and updated from
    /// external source.
    /// None means rollup-boost has not served FCU with attributes yet.
    current_payload_id: Arc<RwLock<Option<PayloadId>>>,

    /// The current Flashblock's payload being constructed.
    best_payload: Arc<RwLock<FlashblockBuilder>>,

    /// Websocket publisher for sending valid pre-confirmations to clients.
    ws_pub: Arc<WebSocketPublisher>,

    /// Metrics
    metrics: FlashblocksServiceMetrics,

    /// Atomic to track absolute maximum number of flashblocks used is block building.
    /// This used to measures the reduction in flashblocks issued.
    max_flashblocks: Arc<AtomicU64>,
}

impl FlashblocksService {
    pub fn new(client: RpcClient, outbound_addr: SocketAddr) -> io::Result<Self> {
        let ws_pub = WebSocketPublisher::new(outbound_addr)?.into();

        Ok(Self {
            client,
            current_payload_id: Arc::new(RwLock::new(None)),
            best_payload: Arc::new(RwLock::new(FlashblockBuilder::new())),
            ws_pub,
            metrics: Default::default(),
            max_flashblocks: Arc::new(AtomicU64::new(0)),
        })
    }

    pub async fn get_best_payload(
        &self,
        version: PayloadVersion,
        payload_id: PayloadId,
    ) -> Result<OpExecutionPayloadEnvelope, FlashblocksError> {
        // Check that we have flashblocks for correct payload
        if *self.current_payload_id.read().await != Some(payload_id) {
            // We have outdated `current_payload_id` so we should fallback to get_payload
            // Clearing best_payload in here would cause situation when old `get_payload` would clear
            // currently built correct flashblocks.
            // This will self-heal on the next FCU.
            return Err(FlashblocksError::MissingPayload);
        }
        // consume the best payload and reset the builder
        let payload = {
            let mut builder = self.best_payload.write().await;
            let flashblocks_number = builder.flashblocks.len() as u64;
            let max_flashblocks = self
                .max_flashblocks
                .fetch_max(flashblocks_number, Ordering::Relaxed)
                .max(flashblocks_number);
            self.metrics
                .record_flashblocks(flashblocks_number, max_flashblocks);
            tracing::Span::current().record("flashblocks_count", flashblocks_number);
            // Take payload and place new one in its place in one go to avoid double locking
            std::mem::replace(&mut *builder, FlashblockBuilder::new()).into_envelope(version)?
        };

        Ok(payload)
    }

    pub async fn set_current_payload_id(&self, payload_id: PayloadId) {
        tracing::debug!(message = "Setting current payload ID", payload_id = %payload_id);
        *self.current_payload_id.write().await = Some(payload_id);
        // Current state won't be useful anymore because chain progressed
        *self.best_payload.write().await = FlashblockBuilder::new();
    }

    async fn on_event(&mut self, event: FlashblocksEngineMessage) {
        match event {
            FlashblocksEngineMessage::OpFlashblockPayload(payload) => {
                self.metrics.messages_processed.increment(1);

                tracing::debug!(
                    message = "Received flashblock payload",
                    payload_id = %payload.payload_id,
                    index = payload.index
                );

                // Make sure the payload id matches the current payload id
                // If local payload id is non then boost is not service FCU with attributes
                match *self.current_payload_id.read().await {
                    Some(payload_id) => {
                        if payload_id != payload.payload_id {
                            self.metrics.current_payload_id_mismatch.increment(1);
                            error!(
                                message = "Payload ID mismatch",
                                payload_id = %payload.payload_id,
                                local_payload_id = %payload_id,
                                index = payload.index,
                            );
                            return;
                        }
                    }
                    None => {
                        // We haven't served FCU with attributes yet, just ignore flashblocks
                        debug!(
                            message = "Received flashblocks, but no FCU with attributes was sent",
                            payload_id = %payload.payload_id,
                            index = payload.index,
                        );
                        return;
                    }
                }

                if let Err(e) = self.best_payload.write().await.extend(payload.clone()) {
                    self.metrics.extend_payload_errors.increment(1);
                    error!(
                        message = "Failed to extend payload",
                        error = %e,
                        payload_id = %payload.payload_id,
                        index = payload.index,
                    );
                } else {
                    // Broadcast the valid message
                    if let Err(e) = self.ws_pub.publish(&payload) {
                        error!(
                            message = "Failed to broadcast payload",
                            error = %e,
                            payload_id = %payload.payload_id,
                            index = payload.index,
                        );
                    }
                }
            }
        }
    }

    pub async fn run(&mut self, mut stream: mpsc::Receiver<OpFlashblockPayload>) {
        while let Some(event) = stream.recv().await {
            self.on_event(FlashblocksEngineMessage::OpFlashblockPayload(event))
                .await;
        }
    }
}

#[async_trait]
impl EngineApiExt for FlashblocksService {
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> ClientResult<ForkchoiceUpdated> {
        // Calculate and set expected payload_id
        if let Some(attr) = &payload_attributes {
            let payload_id = attr.payload_id(&fork_choice_state.head_block_hash, 3);
            self.set_current_payload_id(payload_id).await;
        }

        let resp = self
            .client
            .fork_choice_updated_v3(fork_choice_state, payload_attributes)
            .await?;

        if let Some(payload_id) = resp.payload_id {
            let current_payload = *self.current_payload_id.read().await;
            if current_payload != Some(payload_id) {
                tracing::error!(
                    message = "Payload id returned by builder differs from calculated. Using builder payload id",
                    builder_payload_id = %payload_id,
                    calculated_payload_id = %current_payload.unwrap_or_default(),
                );
                self.set_current_payload_id(payload_id).await;
            } else {
                tracing::debug!(message = "Forkchoice updated", payload_id = %payload_id);
            }
        } else {
            tracing::debug!(message = "Forkchoice updated with no payload ID");
        }
        Ok(resp)
    }

    async fn new_payload(&self, new_payload: NewPayload) -> ClientResult<PayloadStatus> {
        self.client.new_payload(new_payload).await
    }

    async fn get_payload(
        &self,
        payload_id: PayloadId,
        version: PayloadVersion,
    ) -> ClientResult<OpExecutionPayloadEnvelope> {
        // First try to get the best flashblocks payload from the builder if it exists

        match self.get_best_payload(version, payload_id).await {
            Ok(payload) => {
                info!(message = "Returning fb payload");
                // This will finalise block building in builder.
                let client = self.client.clone();
                tokio::spawn(async move {
                    if let Err(e) = client.get_payload(payload_id, version).await {
                        error!(
                            message = "Failed to send finalising getPayload to builder",
                            error = %e,
                        );
                    }
                });
                Ok(payload)
            }
            Err(e) => {
                error!(message = "Error getting fb best payload, falling back on client", error = %e);
                info!(message = "Falling back to get_payload on client", payload_id = %payload_id);
                let result = self.client.get_payload(payload_id, version).await?;
                Ok(result)
            }
        }
    }

    async fn get_block_by_number(
        &self,
        number: BlockNumberOrTag,
        full: bool,
    ) -> ClientResult<Block> {
        self.client.get_block_by_number(number, full).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::payload::PayloadSource;
    use crate::server::tests::{MockEngineServer, spawn_server};
    use http::Uri;
    use reth_rpc_layer::JwtSecret;
    use std::str::FromStr;

    /// Test that we fallback to the getPayload method if the flashblocks payload is not available
    #[tokio::test]
    async fn test_flashblocks_fallback_to_get_payload() -> eyre::Result<()> {
        let builder_mock: MockEngineServer = MockEngineServer::new();
        let (_fallback_server, fallback_server_addr) = spawn_server(builder_mock.clone()).await;
        let jwt_secret = JwtSecret::random();

        let builder_auth_rpc = Uri::from_str(&format!("http://{fallback_server_addr}")).unwrap();
        let builder_client = RpcClient::new(
            builder_auth_rpc.clone(),
            jwt_secret,
            2000,
            PayloadSource::Builder,
        )?;

        let service =
            FlashblocksService::new(builder_client, "127.0.0.1:8000".parse().unwrap()).unwrap();

        // by default, builder_mock returns a valid payload always
        service
            .get_payload(PayloadId::default(), PayloadVersion::V3)
            .await?;

        let get_payload_requests_builder = builder_mock.get_payload_requests.clone();
        assert_eq!(get_payload_requests_builder.lock().len(), 1);

        Ok(())
    }

    /// Test that we don't return block from flashblocks if payload_id is different
    #[tokio::test]
    async fn test_flashblocks_different_payload_id() -> eyre::Result<()> {
        let builder_mock: MockEngineServer = MockEngineServer::new();
        let (_fallback_server, fallback_server_addr) = spawn_server(builder_mock.clone()).await;
        let jwt_secret = JwtSecret::random();

        let builder_auth_rpc = Uri::from_str(&format!("http://{fallback_server_addr}")).unwrap();
        let builder_client = RpcClient::new(
            builder_auth_rpc.clone(),
            jwt_secret,
            2000,
            PayloadSource::Builder,
        )?;

        let service =
            FlashblocksService::new(builder_client, "127.0.0.1:8001".parse().unwrap()).unwrap();

        // Some "random" payload id
        *service.current_payload_id.write().await = Some(PayloadId::new([1, 1, 1, 1, 1, 1, 1, 1]));

        // We ensure that request will skip rollup-boost and serve payload from backup if payload id
        // don't match
        service
            .get_payload(PayloadId::default(), PayloadVersion::V3)
            .await?;

        let get_payload_requests_builder = builder_mock.get_payload_requests.clone();
        assert_eq!(get_payload_requests_builder.lock().len(), 1);

        Ok(())
    }

    #[tokio::test]
    async fn test_flashblocks_builder() -> eyre::Result<()> {
        let mut builder = FlashblockBuilder::new();

        // Error: First payload must have a base
        let result = builder.extend(OpFlashblockPayload {
            payload_id: PayloadId::default(),
            index: 0,
            ..Default::default()
        });
        assert!(result.is_err());
        assert_eq!(result.unwrap_err(), FlashblocksError::MissingBasePayload);

        // Ok: First payload is correct if it has base and index 0
        let result = builder.extend(OpFlashblockPayload {
            payload_id: PayloadId::default(),
            index: 0,
            base: Some(OpFlashblockPayloadBase {
                ..Default::default()
            }),
            ..Default::default()
        });
        assert!(result.is_ok());

        // Error: First payload must have index 0
        let result = builder.extend(OpFlashblockPayload {
            payload_id: PayloadId::default(),
            index: 1,
            base: Some(OpFlashblockPayloadBase {
                ..Default::default()
            }),
            ..Default::default()
        });
        assert!(result.is_err());
        assert_eq!(result.unwrap_err(), FlashblocksError::UnexpectedBasePayload);

        // Error: Second payload must have a follow-up index
        let result = builder.extend(OpFlashblockPayload {
            payload_id: PayloadId::default(),
            index: 2,
            base: None,
            ..Default::default()
        });
        assert!(result.is_err());
        assert_eq!(result.unwrap_err(), FlashblocksError::InvalidIndex);

        // Ok: Second payload has the correct index
        let result = builder.extend(OpFlashblockPayload {
            payload_id: PayloadId::default(),
            index: 1,
            base: None,
            ..Default::default()
        });
        assert!(result.is_ok());

        Ok(())
    }
}
