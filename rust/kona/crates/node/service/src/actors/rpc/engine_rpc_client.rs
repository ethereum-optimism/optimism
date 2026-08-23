use crate::ChainControllerRpcRequest;
use alloy_eips::BlockNumberOrTag;
use async_trait::async_trait;
use derive_more::Constructor;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::{EngineQueries, EngineState, LocalSafeSnapshot};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OutputRoot};
use kona_rpc::EngineRpcClient;
use std::fmt::Debug;
use tokio::sync::{mpsc, oneshot, watch};

/// Queue-based implementation of the [`EngineRpcClient`] trait. This handles all channel-based
/// operations, providing a nice facade for callers.
#[derive(Clone, Constructor, Debug)]
pub struct QueuedEngineRpcClient {
    /// A channel to use to send [`ChainControllerRpcRequest`]s to the
    /// [`crate::ChainControllerRpcActor`].
    pub controller_rpc_request_tx: mpsc::Sender<ChainControllerRpcRequest>,
}

#[async_trait]
impl EngineRpcClient for QueuedEngineRpcClient {
    async fn get_config(&self) -> RpcResult<RollupConfig> {
        let (config_tx, config_rx) = oneshot::channel();

        self.controller_rpc_request_tx
            .send(ChainControllerRpcRequest(Box::new(EngineQueries::Config(config_tx))))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        config_rx.await.map_err(|_| {
            error!(target: "block_engine", "Failed to receive config from engine rpc");
            ErrorObject::from(ErrorCode::InternalError)
        })
    }

    async fn get_state(&self) -> RpcResult<EngineState> {
        let (state_tx, state_rx) = oneshot::channel();

        self.controller_rpc_request_tx
            .send(ChainControllerRpcRequest(Box::new(EngineQueries::State(state_tx))))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        state_rx.await.map_err(|_| {
            error!(target: "block_engine", "Failed to receive state from engine rpc");
            ErrorObject::from(ErrorCode::InternalError)
        })
    }

    async fn output_at_block(
        &self,
        block: BlockNumberOrTag,
    ) -> RpcResult<(L2BlockInfo, OutputRoot, EngineState)> {
        let (output_tx, output_rx) = oneshot::channel();

        self.controller_rpc_request_tx
            .send(ChainControllerRpcRequest(Box::new(EngineQueries::OutputAtBlock {
                block,
                sender: output_tx,
            })))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        output_rx.await.map_err(|_| {
            error!(target: "block_engine", "Failed to receive output at block from engine rpc");
            ErrorObject::from(ErrorCode::InternalError)
        })
    }

    async fn local_safe_snapshot_at(&self, timestamp: u64) -> RpcResult<LocalSafeSnapshot> {
        let (snapshot_tx, snapshot_rx) = oneshot::channel();

        self.controller_rpc_request_tx
            .send(ChainControllerRpcRequest(Box::new(EngineQueries::LocalSafeSnapshotAt {
                timestamp,
                sender: snapshot_tx,
            })))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        snapshot_rx.await.map_err(|_| {
            error!(target: "block_engine", "Failed to receive local-safe snapshot from engine rpc");
            ErrorObject::from(ErrorCode::InternalError)
        })
    }

    async fn dev_get_task_queue_length(&self) -> RpcResult<usize> {
        let (length_tx, length_rx) = oneshot::channel();

        self.controller_rpc_request_tx
            .send(ChainControllerRpcRequest(Box::new(EngineQueries::TaskQueueLength(length_tx))))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        length_rx.await.map_err(|_| {
            error!(target: "block_engine", "Failed to receive task queue length from engine rpc");
            ErrorObject::from(ErrorCode::InternalError)
        })
    }

    async fn dev_subscribe_to_engine_queue_length(&self) -> RpcResult<watch::Receiver<usize>> {
        let (sub_tx, sub_rx) = oneshot::channel();

        self.controller_rpc_request_tx
            .send(ChainControllerRpcRequest(Box::new(EngineQueries::QueueLengthReceiver(sub_tx))))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        sub_rx.await.map_err(|_| {
            error!(target: "block_engine", "Failed to receive queue length receiver from engine rpc");
            ErrorObject::from(ErrorCode::InternalError)
        })
    }
    async fn dev_subscribe_to_engine_state(&self) -> RpcResult<watch::Receiver<EngineState>> {
        let (sub_tx, sub_rx) = oneshot::channel();

        self.controller_rpc_request_tx
            .send(ChainControllerRpcRequest(Box::new(EngineQueries::StateReceiver(sub_tx))))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        sub_rx.await.map_err(|_| {
            error!(target: "block_engine", "Failed to receive state receiver from engine rpc");
            ErrorObject::from(ErrorCode::InternalError)
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{ChainControllerRpcActor, NodeActor};
    use kona_engine::{
        LocalSafeAtTimestamp, LocalSafeHead, LocalSafeOrigin,
        test_utils::{MockEngineClient, TestEngineStateBuilder},
    };
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use std::sync::Arc;

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    fn l2(number: u64, timestamp: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { number, timestamp, ..Default::default() },
            ..Default::default()
        }
    }

    /// The read-only query has to be reachable through the actual request channel, not just from
    /// inside the engine crate: the client sends it, the read-only peer actor answers it, and the
    /// pairing survives the round trip.
    #[tokio::test]
    async fn the_client_reaches_the_local_safe_snapshot_through_the_rpc_actor() {
        // Genesis is block 0 at timestamp 80 with a two-second block time, so the head fixture —
        // block 10 at timestamp 100 — sits exactly where the config arithmetic puts it.
        let cfg = Arc::new(RollupConfig {
            block_time: 2,
            genesis: kona_genesis::ChainGenesis { l2_time: 80, ..Default::default() },
            ..Default::default()
        });
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(l2(12, 104))
            .with_local_safe_head(l2(10, 100))
            .with_local_safe_origin(LocalSafeOrigin::DerivedFrom(l1(5)))
            .with_finalized_head(l2(8, 96))
            .build();

        let (request_tx, request_rx) = mpsc::channel(1);
        let (_state_tx, state_rx) = watch::channel(state);
        let (_length_tx, length_rx) = watch::channel(0usize);
        let mut actor = ChainControllerRpcActor::new(
            Arc::new(MockEngineClient::builder().with_config(cfg.clone()).build()),
            cfg,
            state_rx,
            length_rx,
            request_rx,
        );

        let client = QueuedEngineRpcClient::new(request_tx);
        let query = tokio::spawn(async move { client.local_safe_snapshot_at(100).await });
        actor.step().await.expect("the actor answers the query");
        let snapshot = query.await.expect("the query task finished").expect("the query succeeded");

        assert_eq!(
            snapshot.local_safe_at,
            LocalSafeAtTimestamp::Head(LocalSafeHead::derived_from(l2(10, 100), l1(5)))
        );
        assert_eq!(snapshot.sync_state, state.sync_state);
    }
}
