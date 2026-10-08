//! Implements the rollup client rpc endpoints. These endpoints serve data about the rollup state.
//!
//! Implemented in the op-node in <https://github.com/ethereum-optimism/optimism/blob/174e55f0a1e73b49b80a561fd3fedd4fea5770c6/op-service/sources/rollupclient.go#L16>

use alloy_eips::BlockNumberOrTag;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::{EngineQueryClient, EngineState};
use kona_genesis::RollupConfig;
use kona_protocol::SyncStatus;
use std::sync::Arc;
use tokio::sync::{oneshot, watch};

use crate::{
    L1WatcherQueries, OutputResponse, RollupNodeApiServer, l1_watcher::L1WatcherQuerySender,
};

/// `RollupRpc`
///
/// This is a server implementation of [`crate::RollupNodeApiServer`].
#[derive(Debug)]
pub struct RollupRpc {
    /// The rollup configuration.
    pub config: Arc<RollupConfig>,
    /// The engine state published by the engine task queue.
    pub engine_state: watch::Receiver<EngineState>,
    /// The read-only L2 connection shared with the engine.
    pub l2: EngineQueryClient,
    /// The channel to send [`crate::L1WatcherQueries`]s.
    pub l1_watcher_sender: L1WatcherQuerySender,
}

impl RollupRpc {
    /// Constructs a new [`RollupRpc`] from the node configuration, state, and clients.
    pub const fn new(
        config: Arc<RollupConfig>,
        engine_state: watch::Receiver<EngineState>,
        l2: EngineQueryClient,
        l1_watcher_sender: L1WatcherQuerySender,
    ) -> Self {
        Self { config, engine_state, l2, l1_watcher_sender }
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
impl RollupNodeApiServer for RollupRpc {
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
        const RPC_VERSION: &str = env!("CARGO_PKG_VERSION");

        return Ok(RPC_VERSION.to_string());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::L1State;
    use kona_engine::test_utils::{TestEngineStateBuilder, test_engine_client};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use tokio::sync::mpsc;

    #[tokio::test]
    async fn reads_config_and_published_engine_state() {
        let config = Arc::new(RollupConfig { block_time: 11, ..Default::default() });
        let (client, l1, l2) = test_engine_client(config.clone());
        let (state_tx, state_rx) = watch::channel(EngineState::default());
        let (l1_tx, mut l1_rx) = mpsc::channel(1);
        let rpc = RollupRpc::new(config.clone(), state_rx, client.query_client(), l1_tx);
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
}
