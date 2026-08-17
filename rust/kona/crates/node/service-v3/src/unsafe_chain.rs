//! Unsafe-chain following and local sequencing task scaffold.

use crate::{ControlError, Engine};
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use kona_engine::EngineSyncStateUpdate;
use kona_protocol::{FromBlockError, L2BlockInfo};
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::{Mutex, mpsc, oneshot, watch};

const CONTROL_CAPACITY: usize = 8;

/// Source of unsafe payloads received from the network.
#[async_trait]
pub trait UnsafePayloadSource: Send {
    /// Returns the next network unsafe payload, or `None` when the source has stopped.
    async fn recv_unsafe_payload(&mut self) -> Option<OpExecutionPayloadEnvelope>;
}

#[async_trait]
impl UnsafePayloadSource for mpsc::Receiver<OpExecutionPayloadEnvelope> {
    async fn recv_unsafe_payload(&mut self) -> Option<OpExecutionPayloadEnvelope> {
        self.recv().await
    }
}

/// The unsafe-chain task's current data-plane mode.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub enum UnsafeMode {
    /// Import unsafe payloads received from P2P.
    #[default]
    Following,
    /// Produce and distribute local unsafe payloads.
    Sequencing,
}

#[derive(Debug)]
enum UnsafeChainCommand {
    StartSequencer(oneshot::Sender<Result<(), ControlError>>),
    StopSequencer(oneshot::Sender<Result<(), ControlError>>),
    Shutdown(oneshot::Sender<Result<(), ControlError>>),
}

/// Cloneable control capability for the unsafe-chain task.
#[derive(Debug, Clone)]
pub struct UnsafeChainBuilderHandle {
    control_tx: mpsc::Sender<UnsafeChainCommand>,
    mode_rx: watch::Receiver<UnsafeMode>,
}

impl UnsafeChainBuilderHandle {
    /// Returns the latest published unsafe-chain mode.
    pub fn mode(&self) -> UnsafeMode {
        *self.mode_rx.borrow()
    }

    /// Starts local sequencing at the next safe workflow boundary.
    pub async fn start_sequencer(&self) -> Result<(), ControlError> {
        self.request(UnsafeChainCommand::StartSequencer).await
    }

    /// Stops local sequencing and resumes P2P following.
    pub async fn stop_sequencer(&self) -> Result<(), ControlError> {
        self.request(UnsafeChainCommand::StopSequencer).await
    }

    /// Requests clean task shutdown and waits for acknowledgement.
    pub async fn shutdown(&self) -> Result<(), ControlError> {
        self.request(UnsafeChainCommand::Shutdown).await
    }

    async fn request(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<(), ControlError>>) -> UnsafeChainCommand,
    ) -> Result<(), ControlError> {
        let (response, result) = oneshot::channel();
        self.control_tx.send(command(response)).await.map_err(|_| ControlError::Unavailable)?;
        result.await.map_err(|_| ControlError::ResponseDropped)?
    }
}

/// Long-running owner of P2P following, local sequencing, conductor use, and gossip publication.
#[derive(Debug)]
pub struct UnsafeChainBuilder<L1Client, EngineClient, Network, Conductor> {
    l1: L1Client,
    engine: Arc<Mutex<Engine<EngineClient>>>,
    network: Network,
    conductor: Option<Conductor>,
    control_rx: mpsc::Receiver<UnsafeChainCommand>,
    mode_tx: watch::Sender<UnsafeMode>,
}

impl<L1Client, EngineClient, Network, Conductor>
    UnsafeChainBuilder<L1Client, EngineClient, Network, Conductor>
{
    /// Creates the unsafe-chain task and its control handle without spawning it.
    pub(crate) fn new(
        l1: L1Client,
        engine: Arc<Mutex<Engine<EngineClient>>>,
        network: Network,
        conductor: Option<Conductor>,
    ) -> (Self, UnsafeChainBuilderHandle) {
        let (control_tx, control_rx) = mpsc::channel(CONTROL_CAPACITY);
        let (mode_tx, mode_rx) = watch::channel(UnsafeMode::Following);
        (
            Self { l1, engine, network, conductor, control_rx, mode_tx },
            UnsafeChainBuilderHandle { control_tx, mode_rx },
        )
    }

    /// Runs the unsafe-chain task until explicitly shut down.
    pub async fn run(mut self) -> Result<(), UnsafeChainBuilderError>
    where
        EngineClient: kona_engine::EngineClient,
        Network: UnsafePayloadSource,
    {
        let _ = (&self.l1, &self.conductor);
        let mut mode = UnsafeMode::Following;

        loop {
            tokio::select! {
                biased;
                command = self.control_rx.recv() => {
                    let command = command.ok_or(UnsafeChainBuilderError::ControlChannelClosed)?;
                    match command {
                        UnsafeChainCommand::StartSequencer(response) => {
                            mode = UnsafeMode::Sequencing;
                            self.mode_tx.send_replace(mode);
                            let _ = response.send(Ok(()));
                        }
                        UnsafeChainCommand::StopSequencer(response) => {
                            mode = UnsafeMode::Following;
                            self.mode_tx.send_replace(mode);
                            let _ = response.send(Ok(()));
                        }
                        UnsafeChainCommand::Shutdown(response) => {
                            self.mode_tx.send_replace(UnsafeMode::Following);
                            let _ = response.send(Ok(()));
                            return Ok(());
                        }
                    }
                }
                payload = self.network.recv_unsafe_payload() => {
                    let payload = payload.ok_or(UnsafeChainBuilderError::PayloadSourceClosed)?;
                    if mode == UnsafeMode::Sequencing {
                        tracing::debug!(
                            target: "unsafe_chain_v3",
                            hash = %payload.block_hash(),
                            "Dropping network unsafe payload while sequencing"
                        );
                        continue;
                    }

                    if let Err(error) = self.accept_unsafe_from_network(payload).await {
                        if error.is_invalid_payload() {
                            tracing::warn!(
                                target: "unsafe_chain_v3",
                                %error,
                                "Dropping invalid network unsafe payload"
                            );
                            continue;
                        }
                        return Err(error.into());
                    }
                }
            }
        }
    }

    // The mutable receiver keeps this workflow exclusive to the running service without requiring
    // every builder-owned dependency to be `Sync` across the Engine API awaits.
    #[allow(clippy::needless_pass_by_ref_mut)]
    async fn accept_unsafe_from_network(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<L2BlockInfo, AcceptUnsafeFromNetworkError>
    where
        EngineClient: kona_engine::EngineClient,
    {
        let mut engine = self.engine.lock().await;

        let response = engine
            .new_payload(payload.clone())
            .await
            .map_err(AcceptUnsafeFromNetworkError::NewPayload)?;
        match response.status {
            PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing => {}
            status @ PayloadStatusEnum::Invalid { .. } => {
                return Err(AcceptUnsafeFromNetworkError::InvalidPayload(status.to_string()));
            }
            status => return Err(AcceptUnsafeFromNetworkError::NewPayloadStatus(status)),
        }

        let block: OpBlock = payload.try_into_block()?;
        let new_head = L2BlockInfo::from_block_and_genesis(&block, &engine.config().genesis)?;
        let state_update = EngineSyncStateUpdate {
            unsafe_head: Some(new_head),
            cross_unsafe_head: Some(new_head),
            ..Default::default()
        };
        let mut forkchoice = engine.state().sync_state.create_forkchoice_state();
        forkchoice.head_block_hash = new_head.block_info.hash;

        let response = engine
            .forkchoice_updated(forkchoice)
            .await
            .map_err(AcceptUnsafeFromNetworkError::ForkchoiceUpdated)?;
        if !matches!(
            response.payload_status.status,
            PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing
        ) {
            return Err(AcceptUnsafeFromNetworkError::ForkchoiceUpdatedStatus(
                response.payload_status.status,
            ));
        }

        if response.payload_status.status.is_valid() {
            engine.state_mut().el_sync_finished = true;
        }
        let state = engine.state_mut();
        state.sync_state = state.sync_state.apply_update(state_update);
        Ok(new_head)
    }
}

/// An error returned while accepting a network unsafe payload.
#[derive(Debug, Error)]
pub enum AcceptUnsafeFromNetworkError {
    /// The `newPayload` request failed.
    #[error("newPayload failed: {0}")]
    NewPayload(RpcError<TransportErrorKind>),
    /// The execution layer rejected the payload as invalid.
    #[error("invalid unsafe payload: {0}")]
    InvalidPayload(String),
    /// The execution layer returned an unexpected `newPayload` status.
    #[error("unexpected newPayload status: {0}")]
    NewPayloadStatus(PayloadStatusEnum),
    /// The payload could not be converted into a block.
    #[error(transparent)]
    Payload(#[from] OpPayloadError),
    /// L2 block information could not be extracted from the payload.
    #[error(transparent)]
    BlockInfo(#[from] FromBlockError),
    /// The `forkchoiceUpdated` request failed.
    #[error("forkchoiceUpdated failed: {0}")]
    ForkchoiceUpdated(RpcError<TransportErrorKind>),
    /// The execution layer returned an unexpected `forkchoiceUpdated` status.
    #[error("unexpected forkchoiceUpdated status: {0}")]
    ForkchoiceUpdatedStatus(PayloadStatusEnum),
}

impl AcceptUnsafeFromNetworkError {
    const fn is_invalid_payload(&self) -> bool {
        matches!(self, Self::InvalidPayload(_) | Self::Payload(_) | Self::BlockInfo(_))
    }
}

/// A terminal unsafe-chain task failure.
#[derive(Debug, Error)]
pub enum UnsafeChainBuilderError {
    /// Every unsafe-chain control handle was dropped unexpectedly.
    #[error("unsafe-chain control channel closed")]
    ControlChannelClosed,
    /// The network unsafe-payload source stopped unexpectedly.
    #[error("network unsafe-payload source closed")]
    PayloadSourceClosed,
    /// Accepting a network unsafe payload failed.
    #[error(transparent)]
    AcceptUnsafePayload(#[from] AcceptUnsafeFromNetworkError),
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Engine;
    use alloy_consensus::Block;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus};
    use kona_engine::test_utils::MockEngineClient;
    use kona_genesis::RollupConfig;
    use op_alloy_consensus::OpTxEnvelope;
    use std::sync::Arc;
    use tokio::time::{Duration, timeout};

    fn payload_fixture() -> (Arc<RollupConfig>, OpExecutionPayloadEnvelope, L2BlockInfo) {
        let block = Block::<OpTxEnvelope>::default();
        let mut payload = ExecutionPayloadV1::from_block_slow(&block);
        let reconstructed: OpBlock =
            OpExecutionPayloadEnvelope::V1(payload.clone()).try_into_block().unwrap();
        payload.block_hash = reconstructed.header.hash_slow();
        let envelope = OpExecutionPayloadEnvelope::V1(payload);
        let block: OpBlock = envelope.clone().try_into_block().unwrap();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = envelope.block_hash();
        config.genesis.l2.number = envelope.block_number();
        config.genesis.l2_time = envelope.timestamp();
        let expected = L2BlockInfo::from_block_and_genesis(&block, &config.genesis).unwrap();
        (Arc::new(config), envelope, expected)
    }

    fn mock_engine(config: Arc<RollupConfig>) -> Arc<Mutex<Engine<MockEngineClient>>> {
        mock_engine_with_new_payload_status(config, PayloadStatusEnum::Valid)
    }

    fn mock_engine_with_new_payload_status(
        config: Arc<RollupConfig>,
        status: PayloadStatusEnum,
    ) -> Arc<Mutex<Engine<MockEngineClient>>> {
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(PayloadStatus::from_status(status))
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated {
                    payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
                    payload_id: None,
                })
                .build(),
        );
        Arc::new(Mutex::new(Engine::new(client, config)))
    }

    #[tokio::test]
    async fn unsafe_chain_starts_in_following_mode() {
        let engine = mock_engine(Arc::new(RollupConfig::default()));
        let (_payload_tx, payload_rx) = mpsc::channel(1);
        let (service, handle) = UnsafeChainBuilder::new((), engine, payload_rx, None::<()>);
        let task = tokio::spawn(service.run());

        assert_eq!(handle.mode(), UnsafeMode::Following);
        handle.start_sequencer().await.unwrap();
        assert_eq!(handle.mode(), UnsafeMode::Sequencing);
        handle.stop_sequencer().await.unwrap();
        assert_eq!(handle.mode(), UnsafeMode::Following);
        handle.shutdown().await.unwrap();
        task.await.unwrap().unwrap();
    }

    #[tokio::test]
    async fn following_accepts_network_unsafe_payload() {
        let (config, payload, expected) = payload_fixture();
        let engine = mock_engine(config);
        let observed_engine = engine.clone();
        let (payload_tx, payload_rx) = mpsc::channel(1);
        let (service, handle) = UnsafeChainBuilder::new((), engine, payload_rx, None::<()>);
        let task = tokio::spawn(service.run());

        handle.start_sequencer().await.unwrap();
        handle.stop_sequencer().await.unwrap();
        payload_tx.send(payload).await.unwrap();
        timeout(Duration::from_secs(1), async {
            loop {
                if observed_engine.lock().await.state().sync_state.unsafe_head() == expected {
                    break;
                }
                tokio::task::yield_now().await;
            }
        })
        .await
        .unwrap();

        handle.shutdown().await.unwrap();
        task.await.unwrap().unwrap();
    }

    #[tokio::test]
    async fn invalid_network_unsafe_payload_is_dropped() {
        let (config, payload, _) = payload_fixture();
        let engine = mock_engine_with_new_payload_status(
            config,
            PayloadStatusEnum::Invalid { validation_error: "invalid transaction".to_string() },
        );
        let observed_engine = engine.clone();
        let (payload_tx, payload_rx) = mpsc::channel(1);
        let (service, _handle) = UnsafeChainBuilder::new((), engine, payload_rx, None::<()>);
        let task = tokio::spawn(service.run());

        payload_tx.send(payload).await.unwrap();
        drop(payload_tx);

        assert!(matches!(task.await.unwrap(), Err(UnsafeChainBuilderError::PayloadSourceClosed)));
        assert_eq!(observed_engine.lock().await.state(), &Default::default());
    }

    #[tokio::test]
    async fn sequencing_drops_network_unsafe_payload() {
        let (config, payload, _) = payload_fixture();
        let engine = mock_engine(config);
        let observed_engine = engine.clone();
        let (payload_tx, payload_rx) = mpsc::channel(1);
        let (service, handle) = UnsafeChainBuilder::new((), engine, payload_rx, None::<()>);
        let task = tokio::spawn(service.run());

        handle.start_sequencer().await.unwrap();
        payload_tx.send(payload).await.unwrap();
        let permit = timeout(Duration::from_secs(1), payload_tx.reserve()).await.unwrap().unwrap();
        drop(permit);

        assert_eq!(observed_engine.lock().await.state(), &Default::default());
        handle.shutdown().await.unwrap();
        task.await.unwrap().unwrap();
    }
}
