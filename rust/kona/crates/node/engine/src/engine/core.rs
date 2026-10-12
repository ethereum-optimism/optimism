//! The [`Engine`] performs Engine API operations against the engine state and publishes the
//! state.

use super::BuildSealCoupling;
use crate::{
    BuildTaskError, ConsolidateInput, ConsolidateTaskError, EngineClient, EngineState,
    EngineSyncStateUpdate, EngineTaskError, EngineTaskErrorSeverity, FinalizeBlockId,
    FinalizeTaskError, ImportedBlockSink, InsertTaskError, Metrics, SealTaskError, SyncStartError,
    SynchronizeTaskError, find_starting_forkchoice,
};
use alloy_rpc_types_engine::{PayloadId, PayloadStatusEnum};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::watch::Sender;

/// Owns the engine state, performs Engine API operations against it, and publishes it.
///
/// Operations run one at a time with exclusive access to the [`EngineState`], so each is atomic
/// over it. After every operation, successful or not, the state is published to the watch
/// channel's readers. A failed operation returns its error; the caller decides whether to retry
/// it.
#[derive(Debug)]
pub struct Engine {
    /// The Engine API client.
    pub(super) client: Arc<EngineClient>,
    /// The rollup configuration.
    pub(super) config: Arc<RollupConfig>,
    /// Where to hand every block the engine canonicalizes.
    pub(super) block_sink: Arc<dyn ImportedBlockSink>,
    /// The state of the engine. Operations update it; only the public methods publish it.
    pub(super) state: EngineState,
    /// Publishes the state to readers.
    state_sender: Sender<EngineState>,
}

impl Engine {
    /// Creates a new [`Engine`] with the passed initial [`EngineState`].
    pub const fn new(
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
        block_sink: Arc<dyn ImportedBlockSink>,
        initial_state: EngineState,
        state_sender: Sender<EngineState>,
    ) -> Self {
        Self { client, config, block_sink, state: initial_state, state_sender }
    }

    /// Returns a reference to the inner [`EngineState`].
    pub const fn state(&self) -> &EngineState {
        &self.state
    }

    /// Returns a receiver that can be used to listen to engine state updates.
    pub fn state_subscribe(&self) -> tokio::sync::watch::Receiver<EngineState> {
        self.state_sender.subscribe()
    }

    /// Resets the engine by finding a plausible sync starting point via
    /// [`find_starting_forkchoice`]. The state is updated to the starting point, and a forkchoice
    /// update reorgs the execution layer to it.
    pub async fn reset(&mut self) -> Result<L2BlockInfo, EngineResetError> {
        let start = find_starting_forkchoice(&self.config, self.client.as_ref()).await?;

        // One attempt. The caller retries transient failures.
        let result = self
            .synchronize(EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            })
            .await;
        self.state_sender.send_replace(self.state);
        result?;

        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        Ok(start.safe)
    }

    /// Inserts an unsafe payload received from gossip or the admin API.
    ///
    /// A payload the execution layer reports as `INVALID` or `ACCEPTED` is dropped: retrying
    /// returns the same verdict.
    pub async fn insert_unsafe(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), InsertTaskError> {
        let result = match self.insert(payload, false).await {
            Err(InsertTaskError::UnexpectedPayloadStatus(
                status @ (PayloadStatusEnum::Invalid { .. } | PayloadStatusEnum::Accepted),
            )) => {
                warn!(target: "engine", %status, "Dropping rejected unsafe payload");
                Ok(())
            }
            result => result.map(|_| ()),
        };
        self.finish(Metrics::INSERT_TASK_LABEL, result)
    }

    /// Starts building a block on the execution layer and returns its payload ID.
    pub async fn build(
        &mut self,
        attributes: OpAttributesWithParent,
    ) -> Result<PayloadId, BuildTaskError> {
        let result = self.start_build(attributes).await;
        self.finish(Metrics::BUILD_TASK_LABEL, result)
    }

    /// Seals a block started with [`Self::build`] and makes it the unsafe head.
    ///
    /// Fails with [`SealTaskError::UnsafeHeadChangedSinceBuild`] if the unsafe head moved since
    /// the build started.
    pub async fn seal(
        &mut self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        let result =
            self.seal_block(payload_id, attributes, false, BuildSealCoupling::Detached).await;
        self.finish(Metrics::SEAL_TASK_LABEL, result)
    }

    /// Advances the safe head to a derived or delegated block, reorging the unsafe chain to it
    /// if necessary.
    pub async fn consolidate(
        &mut self,
        input: ConsolidateInput,
    ) -> Result<(), ConsolidateTaskError> {
        let result = self.consolidate_input(&input).await;
        self.finish(Metrics::CONSOLIDATE_TASK_LABEL, result)
    }

    /// Finalizes an L2 block.
    pub async fn finalize(&mut self, block_id: FinalizeBlockId) -> Result<(), FinalizeTaskError> {
        let result = self.finalize_block(block_id).await;
        self.finish(Metrics::FINALIZE_TASK_LABEL, result)
    }

    /// Publishes the state and records the outcome of an operation.
    ///
    /// The state is published even if the operation failed: it can update the state before a
    /// later step fails.
    fn finish<T, E: EngineTaskError>(
        &self,
        label: &'static str,
        result: Result<T, E>,
    ) -> Result<T, E> {
        self.state_sender.send_replace(self.state);
        match &result {
            Ok(_) => {
                metrics::counter!(Metrics::ENGINE_TASK_SUCCESS, "type" => label).increment(1);
            }
            Err(err) => {
                metrics::counter!(Metrics::ENGINE_TASK_FAILURE, label => err.severity().to_string())
                    .increment(1);
            }
        }
        result
    }
}

#[cfg(test)]
impl Engine {
    /// Returns an engine with the initial state, a block sink that discards blocks, and no state
    /// readers.
    pub(crate) fn for_test(
        client: EngineClient,
        config: Arc<RollupConfig>,
        initial_state: EngineState,
    ) -> Self {
        Self::new(
            Arc::new(client),
            config,
            Arc::new(crate::NoopBlockSink),
            initial_state,
            tokio::sync::watch::channel(initial_state).0,
        )
    }
}

/// An error occurred while attempting to reset the [`Engine`].
#[derive(Debug, Error)]
pub enum EngineResetError {
    /// An error that occurred while updating the forkchoice state.
    #[error(transparent)]
    Forkchoice(#[from] SynchronizeTaskError),
    /// An error occurred while traversing the L1 for the sync starting point.
    #[error(transparent)]
    SyncStart(#[from] SyncStartError),
}

impl EngineResetError {
    /// Whether retrying the reset can recover without changing the node configuration.
    pub fn is_temporary(&self) -> bool {
        match self {
            Self::SyncStart(SyncStartError::RpcError(_)) => true,
            Self::SyncStart(_) => false,
            Self::Forkchoice(err) => err.severity() != EngineTaskErrorSeverity::Critical,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        NoopBlockSink,
        test_utils::{
            TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
        },
    };
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

    fn engine(
        client: EngineClient,
        config: Arc<RollupConfig>,
        state: EngineState,
        state_tx: Sender<EngineState>,
    ) -> Engine {
        Engine::new(Arc::new(client), config, Arc::new(NoopBlockSink), state, state_tx)
    }

    #[tokio::test]
    async fn reset_publishes_the_new_state() {
        let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.inner.hash_slow();
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        for _ in 0..3 {
            l2.expect("eth_getBlockByNumber", &block);
        }
        l1.expect(
            "eth_getBlockByHash",
            alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default(),
        );
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        let (state_tx, mut state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = engine(client, config.clone(), EngineState::default(), state_tx);
        engine.reset().await.unwrap();
        // Readers observe the reset without waiting for another operation.
        assert!(state_rx.has_changed().unwrap());
        let state = *state_rx.borrow_and_update();
        assert_eq!(state, *engine.state());
        assert_eq!(state.sync_state.unsafe_head().block_info.hash, config.genesis.l2.hash);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Returns an unsafe payload, and a configuration whose genesis is the payload's block.
    fn genesis_payload() -> (OpExecutionPayloadEnvelope, Arc<RollupConfig>) {
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
        ));
        let block: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.hash_slow();
        (payload, Arc::new(config))
    }

    #[tokio::test]
    async fn operations_publish_the_state_after_every_attempt() {
        let (payload, config) = genesis_payload();
        let (client, l1, l2) = test_engine_client(config.clone());
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        l2.expect_error("engine_newPayloadV1");
        l2.expect("engine_newPayloadV1", valid());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
        let (state_tx, mut state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = engine(client, config.clone(), EngineState::default(), state_tx);

        // A failed attempt returns its error for the caller to retry, and is still published.
        let error = engine.insert_unsafe(payload.clone()).await.unwrap_err();
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Temporary);
        assert!(state_rx.has_changed().unwrap());
        state_rx.mark_unchanged();

        engine.insert_unsafe(payload).await.unwrap();
        assert!(state_rx.has_changed().unwrap());
        assert_eq!(
            state_rx.borrow().sync_state.unsafe_head().block_info.hash,
            config.genesis.l2.hash
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Records the blocks the engine hands over after a successful import.
    #[derive(Debug, Default)]
    struct RecordingSink(std::sync::Mutex<Vec<(alloy_primitives::B256, u64)>>);

    impl ImportedBlockSink for RecordingSink {
        fn block_imported(&self, block: op_alloy_consensus::OpBlock, info: L2BlockInfo) {
            self.0.lock().unwrap().push((info.block_info.hash, block.header.number));
        }
    }

    #[tokio::test]
    async fn inserted_blocks_are_handed_to_the_block_sink() {
        let (payload, config) = genesis_payload();
        let (client, l1, l2) = test_engine_client(config.clone());
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        l2.expect("engine_newPayloadV1", valid());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
        let sink = Arc::new(RecordingSink::default());
        let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = Engine::new(
            Arc::new(client),
            config.clone(),
            sink.clone(),
            EngineState::default(),
            state_tx,
        );

        engine.insert_unsafe(payload).await.unwrap();
        assert_eq!(
            sink.0.lock().unwrap().as_slice(),
            &[(config.genesis.l2.hash, 0)],
            "a successfully imported block must reach the sink"
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    #[rstest::rstest]
    #[case::invalid(PayloadStatusEnum::Invalid { validation_error: "invalid transaction".into() })]
    #[case::accepted(PayloadStatusEnum::Accepted)]
    #[tokio::test]
    async fn insert_unsafe_drops_rejected_payloads(#[case] status: PayloadStatusEnum) {
        let (payload, config) = genesis_payload();
        let (client, l1, l2) = test_engine_client(config.clone());
        l2.expect("engine_newPayloadV1", PayloadStatus::from_status(status));
        let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = engine(client, config, EngineState::default(), state_tx);

        engine.insert_unsafe(payload).await.unwrap();
        assert_eq!(*engine.state(), EngineState::default());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Builds with attributes the execution layer answers with `status` and a payload ID.
    async fn build(status: PayloadStatusEnum) -> Result<PayloadId, BuildTaskError> {
        let mut config = RollupConfig::default();
        config.hardforks.ecotone_time = Some(0);
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(status))
                .with_payload_id(PayloadId::new([1; 8])),
        );
        let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = engine(client, config, EngineState::default(), state_tx);

        let result = engine.build(TestAttributesBuilder::new().build()).await;
        l1.assert_finished();
        l2.assert_finished();
        result
    }

    #[tokio::test]
    async fn build_returns_the_payload_id() {
        assert_eq!(build(PayloadStatusEnum::Valid).await.unwrap(), PayloadId::new([1; 8]));
    }

    #[tokio::test]
    async fn build_returns_a_rejection_to_the_caller() {
        let result = build(PayloadStatusEnum::Invalid { validation_error: String::new() }).await;
        assert!(matches!(
            result,
            Err(BuildTaskError::EngineBuildError(crate::EngineBuildError::InvalidPayload(_)))
        ));
    }

    #[tokio::test]
    async fn seal_rejects_a_build_whose_parent_is_no_longer_the_unsafe_head() {
        let config = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(config.clone());
        let unsafe_head = test_block_info(2);
        let state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();
        let (state_tx, _state_rx) = tokio::sync::watch::channel(state);
        let mut engine = engine(client, config, state, state_tx);

        let attributes = TestAttributesBuilder::new().with_parent(test_block_info(1)).build();
        let result = engine.seal(PayloadId::new([1; 8]), attributes).await;
        assert!(matches!(result, Err(SealTaskError::UnsafeHeadChangedSinceBuild)));
        l1.assert_finished();
        l2.assert_finished();
    }
}
