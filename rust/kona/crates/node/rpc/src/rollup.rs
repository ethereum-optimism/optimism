//! Implements the rollup client rpc endpoints. These endpoints serve data about the rollup state.
//!
//! Implemented in the op-node in <https://github.com/ethereum-optimism/optimism/blob/174e55f0a1e73b49b80a561fd3fedd4fea5770c6/op-service/sources/rollupclient.go#L16>

use alloy_eips::BlockNumberOrTag;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::EngineState;
use kona_genesis::RollupConfig;
use kona_protocol::SyncStatus;
use kona_safedb::{SafeDb, SafeDbError};
use std::{fmt::Debug, sync::Arc};

use crate::{
    EngineRpcClient, L1State, L1WatcherQueries, OutputResponse, RollupNodeApiServer,
    SafeHeadResponse, l1_watcher::L1WatcherQuerySender,
};

/// `RollupRpc`
///
/// This is a server implementation of [`crate::RollupNodeApiServer`].
#[derive(Debug)]
pub struct RollupRpc<EngineRpcClient_> {
    /// The channel to send [`kona_engine::EngineQueries`]s.
    pub engine_client: EngineRpcClient_,
    /// The channel to send [`crate::L1WatcherQueries`]s.
    pub l1_watcher_sender: L1WatcherQuerySender,
    /// The safe-head database backing `optimism_safeHeadAtL1Block`.
    pub safe_db: Arc<dyn SafeDb>,
}

impl<EngineRpcClient_: EngineRpcClient> RollupRpc<EngineRpcClient_> {
    /// The identifier for the Metric that tracks rollup RPC calls.
    pub const RPC_IDENT: &'static str = "rollup_rpc";

    /// Constructs a new [`RollupRpc`] given a sender channel.
    pub const fn new(
        engine_client: EngineRpcClient_,
        l1_watcher_sender: L1WatcherQuerySender,
        safe_db: Arc<dyn SafeDb>,
    ) -> Self {
        Self { engine_client, l1_watcher_sender, safe_db }
    }

    // Important note: we zero-out the fields that can't be derived yet to follow op-node's
    // behaviour.
    fn sync_status_from_actor_queries(
        l1_sync_status: L1State,
        l2_sync_status: EngineState,
    ) -> SyncStatus {
        SyncStatus {
            current_l1: l1_sync_status.current_l1.unwrap_or_default(),
            current_l1_finalized: l1_sync_status.current_l1_finalized.unwrap_or_default(),
            head_l1: l1_sync_status.head_l1.unwrap_or_default(),
            safe_l1: l1_sync_status.safe_l1.unwrap_or_default(),
            finalized_l1: l1_sync_status.finalized_l1.unwrap_or_default(),
            unsafe_l2: l2_sync_status.sync_state.unsafe_head(),
            cross_unsafe_l2: l2_sync_status.sync_state.cross_unsafe_head(),
            local_safe_l2: l2_sync_status.sync_state.local_safe_head(),
            safe_l2: l2_sync_status.sync_state.safe_head(),
            finalized_l2: l2_sync_status.sync_state.finalized_head(),
        }
    }
}

#[async_trait]
impl<EngineRpcClient_: EngineRpcClient + 'static> RollupNodeApiServer
    for RollupRpc<EngineRpcClient_>
{
    async fn op_output_at_block(&self, block_num: BlockNumberOrTag) -> RpcResult<OutputResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_outputAtBlock");

        let (l1_sync_status_send, l1_sync_status_recv) = tokio::sync::oneshot::channel();

        let ((l2_block_info, output_root, l2_sync_status), l1_sync_status) =
            tokio::try_join!(self.engine_client.output_at_block(block_num), async {
                self.l1_watcher_sender
                    .send(L1WatcherQueries::L1State(l1_sync_status_send))
                    .await
                    .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

                l1_sync_status_recv.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
            })?;

        let sync_status = Self::sync_status_from_actor_queries(l1_sync_status, l2_sync_status);

        Ok(OutputResponse::from_v0(output_root, sync_status, l2_block_info))
    }

    async fn op_safe_head_at_l1_block(
        &self,
        block_num: BlockNumberOrTag,
    ) -> RpcResult<SafeHeadResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_safeHeadAtL1Block");

        // op-node types this parameter as a plain number. Tags that need chain context to
        // resolve have no counterpart there, so only the numeric forms are accepted.
        let l1_block_num = match block_num {
            BlockNumberOrTag::Number(num) => num,
            BlockNumberOrTag::Earliest => 0,
            tag => {
                return Err(ErrorObject::owned(
                    ErrorCode::InvalidParams.code(),
                    format!("safeHeadAtL1Block requires an L1 block number, got {tag}"),
                    None::<()>,
                ));
            }
        };

        let record = self.safe_db.safe_head_at_l1(l1_block_num).map_err(|e| match e {
            SafeDbError::NotEnabled => ErrorObject::from(ErrorCode::MethodNotFound),
            SafeDbError::NotFound => ErrorObject::owned(
                ErrorCode::InvalidRequest.code(),
                format!("no safe head recorded at or below L1 block {l1_block_num}"),
                None::<()>,
            ),
            _ => ErrorObject::from(ErrorCode::InternalError),
        })?;

        Ok(SafeHeadResponse { l1_block: record.l1, safe_head: record.safe_head })
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_syncStatus");

        let (l1_sync_status_send, l1_sync_status_recv) = tokio::sync::oneshot::channel();

        let (l1_sync_status, l2_sync_status) = tokio::try_join!(
            async {
                self.l1_watcher_sender
                    .send(L1WatcherQueries::L1State(l1_sync_status_send))
                    .await
                    .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
                l1_sync_status_recv.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
            },
            self.engine_client.get_state()
        )
        .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        return Ok(Self::sync_status_from_actor_queries(l1_sync_status, l2_sync_status));
    }

    async fn op_rollup_config(&self) -> RpcResult<RollupConfig> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_rollupConfig");

        self.engine_client.get_config().await
    }

    async fn op_version(&self) -> RpcResult<String> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_version");

        const RPC_VERSION: &str = env!("CARGO_PKG_VERSION");

        return Ok(RPC_VERSION.to_string());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use kona_protocol::{BlockInfo, L2BlockInfo, OutputRoot};
    use kona_safedb::{DisabledDatabase, SafeDatabase};
    use tempfile::TempDir;
    use tokio::sync::{mpsc, watch};

    /// Stub engine client: `safeHeadAtL1Block` touches none of these.
    #[derive(Debug, Clone)]
    struct UnusedEngineClient;

    #[async_trait]
    impl EngineRpcClient for UnusedEngineClient {
        async fn get_config(&self) -> RpcResult<RollupConfig> {
            unimplemented!("not used by safeHeadAtL1Block")
        }
        async fn get_state(&self) -> RpcResult<EngineState> {
            unimplemented!("not used by safeHeadAtL1Block")
        }
        async fn output_at_block(
            &self,
            _block: BlockNumberOrTag,
        ) -> RpcResult<(L2BlockInfo, OutputRoot, EngineState)> {
            unimplemented!("not used by safeHeadAtL1Block")
        }
        async fn dev_get_task_queue_length(&self) -> RpcResult<usize> {
            unimplemented!("not used by safeHeadAtL1Block")
        }
        async fn dev_subscribe_to_engine_queue_length(&self) -> RpcResult<watch::Receiver<usize>> {
            unimplemented!("not used by safeHeadAtL1Block")
        }
        async fn dev_subscribe_to_engine_state(&self) -> RpcResult<watch::Receiver<EngineState>> {
            unimplemented!("not used by safeHeadAtL1Block")
        }
    }

    fn rpc(safe_db: Arc<dyn SafeDb>) -> RollupRpc<UnusedEngineClient> {
        let (l1_watcher_sender, _rx) = mpsc::channel(1);
        RollupRpc::new(UnusedEngineClient, l1_watcher_sender, safe_db)
    }

    fn l2(number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::repeat_byte(2),
                number,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            l1_origin: BlockNumHash { hash: B256::ZERO, number: 0 },
            seq_num: 0,
        }
    }

    fn l1(number: u64) -> BlockNumHash {
        BlockNumHash { hash: B256::repeat_byte(1), number }
    }

    fn populated() -> (TempDir, Arc<dyn SafeDb>) {
        let dir = TempDir::new().unwrap();
        let db = SafeDatabase::new(dir.path()).unwrap();
        db.safe_head_updated(l2(20), l1(100)).unwrap();
        (dir, Arc::new(db))
    }

    #[tokio::test]
    async fn returns_the_recorded_entry() {
        let (_dir, db) = populated();
        let response =
            rpc(db).op_safe_head_at_l1_block(BlockNumberOrTag::Number(100)).await.unwrap();
        assert_eq!(response.l1_block, l1(100));
        assert_eq!(response.safe_head, l2(20).block_info.id());
    }

    #[tokio::test]
    async fn resolves_to_the_highest_entry_at_or_below_the_request() {
        let (_dir, db) = populated();
        let response =
            rpc(db).op_safe_head_at_l1_block(BlockNumberOrTag::Number(150)).await.unwrap();
        assert_eq!(response.l1_block, l1(100));
    }

    #[tokio::test]
    async fn reports_a_request_below_all_records() {
        let (_dir, db) = populated();
        let err = rpc(db).op_safe_head_at_l1_block(BlockNumberOrTag::Number(99)).await.unwrap_err();
        assert_eq!(err.code(), ErrorCode::InvalidRequest.code());
    }

    #[tokio::test]
    async fn earliest_resolves_to_block_zero() {
        let (_dir, db) = populated();
        // Block 0 is below the only record, so this must fail the same way an explicit 0 does.
        let err = rpc(db).op_safe_head_at_l1_block(BlockNumberOrTag::Earliest).await.unwrap_err();
        assert_eq!(err.code(), ErrorCode::InvalidRequest.code());
    }

    #[tokio::test]
    async fn rejects_tags_that_need_chain_context() {
        let (_dir, db) = populated();
        let err = rpc(db).op_safe_head_at_l1_block(BlockNumberOrTag::Latest).await.unwrap_err();
        assert_eq!(err.code(), ErrorCode::InvalidParams.code());
    }

    #[tokio::test]
    async fn disabled_database_reports_method_not_found() {
        let err = rpc(Arc::new(DisabledDatabase))
            .op_safe_head_at_l1_block(BlockNumberOrTag::Number(100))
            .await
            .unwrap_err();
        assert_eq!(err.code(), ErrorCode::MethodNotFound.code());
    }
}
