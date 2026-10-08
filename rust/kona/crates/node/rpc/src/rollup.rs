//! Implements the rollup client rpc endpoints. These endpoints serve data about the rollup state.
//!
//! Implemented in the op-node in <https://github.com/ethereum-optimism/optimism/blob/174e55f0a1e73b49b80a561fd3fedd4fea5770c6/op-service/sources/rollupclient.go#L16>

use alloy_eips::BlockNumberOrTag;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::{EngineClient, EngineClientError, EngineState};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OutputRoot, SyncStatus};
use std::sync::Arc;
use tokio::sync::{oneshot, watch};

use crate::{
    L1WatcherQueries, OutputResponse, RollupNodeApiServer, l1_watcher::L1WatcherQuerySender,
};

/// The read-only L2 output query needed by [`RollupRpc`].
pub trait OutputProvider: Send + Sync {
    /// An error encountered while reading or computing an output.
    type Error;

    /// Reads the L2 block and computes its output root.
    fn output_at_block(
        &self,
        block: BlockNumberOrTag,
    ) -> impl Future<Output = Result<(L2BlockInfo, OutputRoot), Self::Error>> + Send;
}

impl OutputProvider for EngineClient {
    type Error = EngineClientError;

    async fn output_at_block(
        &self,
        block: BlockNumberOrTag,
    ) -> Result<(L2BlockInfo, OutputRoot), Self::Error> {
        Self::output_at_block(self, block).await
    }
}

/// `RollupRpc`
///
/// This is a server implementation of [`crate::RollupNodeApiServer`].
#[derive(Debug)]
pub struct RollupRpc<L2> {
    /// The application version.
    pub version: String,
    /// The rollup configuration.
    pub config: Arc<RollupConfig>,
    /// The engine state published by the engine task queue.
    pub engine_state: watch::Receiver<EngineState>,
    /// The read-only L2 output query provider.
    l2: L2,
    /// The channel to send [`crate::L1WatcherQueries`]s.
    pub l1_watcher_sender: L1WatcherQuerySender,
}

impl<L2> RollupRpc<L2> {
    /// Constructs a new [`RollupRpc`] from the application version, configuration, state, and
    /// clients.
    pub const fn new(
        version: String,
        config: Arc<RollupConfig>,
        engine_state: watch::Receiver<EngineState>,
        l2: L2,
        l1_watcher_sender: L1WatcherQuerySender,
    ) -> Self {
        Self { version, config, engine_state, l2, l1_watcher_sender }
    }

    async fn sync_status(&self) -> Result<SyncStatus, oneshot::error::RecvError> {
        // Copy the state before awaiting the L1 read; no watch borrow is held across an await.
        let l2_sync_status = *self.engine_state.borrow();
        let (sender, receiver) = oneshot::channel();
        // A failed send drops the reply sender, so the receive below also fails.
        let _ = self.l1_watcher_sender.send(L1WatcherQueries::L1State(sender)).await;
        let l1_sync_status = receiver.await?;

        // Zero-out the fields that can't be derived yet to follow op-node's behaviour.
        Ok(SyncStatus {
            current_l1: l1_sync_status.current_l1.unwrap_or_default(),
            current_l1_finalized: l1_sync_status.current_l1_finalized.unwrap_or_default(),
            head_l1: l1_sync_status.head_l1.unwrap_or_default(),
            safe_l1: l1_sync_status.safe_l1.unwrap_or_default(),
            finalized_l1: l1_sync_status.finalized_l1.unwrap_or_default(),
            unsafe_l2: l2_sync_status.sync_state.unsafe_head(),
            local_safe_l2: l2_sync_status.sync_state.local_safe_head(),
            safe_l2: l2_sync_status.sync_state.safe_head(),
            finalized_l2: l2_sync_status.sync_state.finalized_head(),
        })
    }
}

#[async_trait]
impl<L2: OutputProvider + 'static> RollupNodeApiServer for RollupRpc<L2> {
    async fn op_output_at_block(&self, block_num: BlockNumberOrTag) -> RpcResult<OutputResponse> {
        let ((l2_block_info, output_root), sync_status) = tokio::try_join!(
            async {
                self.l2
                    .output_at_block(block_num)
                    .await
                    .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
            },
            async {
                self.sync_status().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
            }
        )?;

        Ok(OutputResponse::from_v0(output_root, sync_status, l2_block_info))
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        self.sync_status().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn op_rollup_config(&self) -> RpcResult<RollupConfig> {
        Ok((*self.config).clone())
    }

    async fn op_version(&self) -> RpcResult<String> {
        Ok(self.version.clone())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::L1State;
    use alloy_primitives::B256;
    use kona_engine::test_utils::{TestEngineStateBuilder, test_engine_client};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use tokio::sync::mpsc;

    struct TestOutputProvider {
        block: BlockNumberOrTag,
        output: Result<(L2BlockInfo, OutputRoot), &'static str>,
    }

    impl OutputProvider for TestOutputProvider {
        type Error = &'static str;

        async fn output_at_block(
            &self,
            block: BlockNumberOrTag,
        ) -> Result<(L2BlockInfo, OutputRoot), Self::Error> {
            assert_eq!(block, self.block);
            self.output
        }
    }

    #[tokio::test]
    async fn queries_outputs_through_a_single_method_provider() {
        let block = BlockNumberOrTag::Number(17);
        let block_info = L2BlockInfo {
            block_info: BlockInfo { number: 17, hash: B256::repeat_byte(1), ..Default::default() },
            ..Default::default()
        };
        let root = OutputRoot::from_parts(
            B256::repeat_byte(2),
            B256::repeat_byte(3),
            block_info.block_info.hash,
        );
        let config = Arc::new(RollupConfig::default());
        let (_, state_rx) = watch::channel(EngineState::default());
        let (l1_tx, mut l1_rx) = mpsc::channel(1);
        let rpc = RollupRpc::new(
            "test".to_owned(),
            config.clone(),
            state_rx.clone(),
            TestOutputProvider { block, output: Ok((block_info, root)) },
            l1_tx.clone(),
        );
        let head_l1 = BlockInfo { number: 42, ..Default::default() };
        let (response, ()) = tokio::join!(rpc.op_output_at_block(block), async {
            let L1WatcherQueries::L1State(reply) = l1_rx.recv().await.unwrap() else {
                panic!("expected L1 state query");
            };
            reply
                .send(L1State {
                    current_l1: None,
                    current_l1_finalized: None,
                    head_l1: Some(head_l1),
                    safe_l1: None,
                    finalized_l1: None,
                })
                .unwrap();
        });
        let response = response.unwrap();
        assert_eq!(response.block_ref, block_info);
        assert_eq!(response.output_root, root.hash());
        assert_eq!(response.state_root, root.state_root);
        assert_eq!(response.withdrawal_storage_root, root.bridge_storage_root);
        assert_eq!(response.sync_status.head_l1, head_l1);

        let rpc = RollupRpc::new(
            "test".to_owned(),
            config,
            state_rx,
            TestOutputProvider { block, output: Err("output unavailable") },
            l1_tx,
        );
        assert_eq!(
            rpc.op_output_at_block(block).await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }

    #[tokio::test]
    async fn reads_config_and_published_engine_state() {
        let config = Arc::new(RollupConfig { block_time: 11, ..Default::default() });
        let (client, l1, l2) = test_engine_client(config.clone());
        let (state_tx, state_rx) = watch::channel(EngineState::default());
        let (l1_tx, mut l1_rx) = mpsc::channel(1);
        let rpc = RollupRpc::new("1.2.3-test".to_owned(), config.clone(), state_rx, client, l1_tx);
        assert_eq!(rpc.op_rollup_config().await.unwrap(), *config);

        let head = L2BlockInfo {
            block_info: BlockInfo { number: 17, ..Default::default() },
            ..Default::default()
        };
        let state = TestEngineStateBuilder::new().with_unsafe_head(head).build();
        state_tx.send_replace(state);
        let (status, ()) = tokio::join!(rpc.op_sync_status(), async {
            let L1WatcherQueries::L1State(reply) = l1_rx.recv().await.unwrap() else {
                panic!("expected L1 state query");
            };
            reply
                .send(L1State {
                    current_l1: None,
                    current_l1_finalized: None,
                    head_l1: Some(BlockInfo { number: 42, ..Default::default() }),
                    safe_l1: None,
                    finalized_l1: None,
                })
                .unwrap();
        });
        let status = status.unwrap();
        assert_eq!(status.unsafe_l2, head);
        assert_eq!(status.head_l1.number, 42);

        drop(l1_rx);
        assert!(rpc.sync_status().await.is_err());
        assert_eq!(rpc.op_sync_status().await.unwrap_err().code(), ErrorCode::InternalError.code());
        l1.assert_finished();
        l2.assert_finished();
    }

    #[tokio::test]
    async fn returns_configured_version() {
        let config = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(config.clone());
        let (_, state_rx) = watch::channel(EngineState::default());
        let (l1_tx, _) = mpsc::channel(1);

        for version in ["1.2.3-test", "0.0.0-dev"] {
            let rpc = RollupRpc::new(
                version.to_owned(),
                config.clone(),
                state_rx.clone(),
                client.clone(),
                l1_tx.clone(),
            );
            assert_eq!(rpc.op_version().await.unwrap(), version);
        }

        l1.assert_finished();
        l2.assert_finished();
    }
}
