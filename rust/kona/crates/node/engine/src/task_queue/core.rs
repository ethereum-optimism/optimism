//! The [`Engine`] executes [`EngineTask`]s against the engine state and publishes the state.

use super::EngineTaskExt;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, EngineTaskErrors, Metrics, SyncStartError, SynchronizeTask,
    SynchronizeTaskError, find_starting_forkchoice,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::watch::Sender;

/// Owns the engine state, executes [`EngineTask`]s against it, and publishes it.
///
/// Tasks execute one at a time with exclusive access to the [`EngineState`], so each is an atomic
/// operation over it. After every task and reset, successful or not, the state is published to
/// the watch channel's readers. A failed task returns its error; the caller decides whether to
/// retry it.
#[derive(Debug)]
pub struct Engine {
    /// The state of the engine.
    state: EngineState,
    /// Publishes the state to readers.
    state_sender: Sender<EngineState>,
}

impl Engine {
    /// Creates a new [`Engine`] with the passed initial [`EngineState`].
    pub const fn new(initial_state: EngineState, state_sender: Sender<EngineState>) -> Self {
        Self { state: initial_state, state_sender }
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
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let start = find_starting_forkchoice(&config, client.as_ref()).await?;

        // One attempt. The caller retries transient failures.
        let synchronize = SynchronizeTask::new(
            client,
            config,
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
        );
        let result = synchronize.execute(&mut self.state).await;
        self.state_sender.send_replace(self.state);
        result?;

        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        Ok(start.safe)
    }

    /// Executes a task against the engine state, then publishes the state.
    ///
    /// The state is published even if the task fails: a task can update it before a later step
    /// fails.
    pub async fn execute(&mut self, task: &EngineTask) -> Result<(), EngineTaskErrors> {
        let result = task.execute(&mut self.state).await;
        self.state_sender.send_replace(self.state);
        result
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
    use crate::{InsertTask, NoopBlockSink, test_utils::test_engine_client};
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

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
        let mut engine = Engine::new(EngineState::default(), state_tx);
        engine.reset(Arc::new(client), config.clone()).await.unwrap();
        // Readers observe the reset without waiting for a task to drain.
        assert!(state_rx.has_changed().unwrap());
        let state = *state_rx.borrow_and_update();
        assert_eq!(state, *engine.state());
        assert_eq!(state.sync_state.unsafe_head().block_info.hash, config.genesis.l2.hash);
        l1.assert_finished();
        l2.assert_finished();
    }

    #[tokio::test]
    async fn execute_publishes_the_state_after_every_attempt() {
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
        ));
        let block: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.hash_slow();
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        l2.expect_error("engine_newPayloadV1");
        l2.expect("engine_newPayloadV1", valid());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
        let (state_tx, mut state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = Engine::new(EngineState::default(), state_tx);
        let task = EngineTask::Insert(Box::new(InsertTask::new(
            Arc::new(client),
            config.clone(),
            payload,
            false,
            Arc::new(NoopBlockSink),
        )));

        // A failed attempt returns its error for the caller to retry, and is still published.
        let error = engine.execute(&task).await.unwrap_err();
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Temporary);
        assert!(state_rx.has_changed().unwrap());
        state_rx.mark_unchanged();

        engine.execute(&task).await.unwrap();
        assert!(state_rx.has_changed().unwrap());
        assert_eq!(
            state_rx.borrow().sync_state.unsafe_head().block_info.hash,
            config.genesis.l2.hash
        );
        l1.assert_finished();
        l2.assert_finished();
    }
}
