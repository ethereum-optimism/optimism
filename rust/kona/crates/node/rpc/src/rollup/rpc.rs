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
    EngineRpcClient, L1State, OutputResponse, RollupNodeApiServer, SafeHeadResponse,
    l1_watcher::L1WatcherQuerySender, safe_db_error::safe_db_error,
};

/// `RollupRpc`
///
/// This is a server implementation of [`crate::RollupNodeApiServer`].
#[derive(Debug)]
pub struct RollupRpc<EngineRpcClient_> {
    /// The running binary's release version.
    pub version: &'static str,
    /// The channel to send [`kona_engine::EngineQueries`]s.
    pub engine_client: EngineRpcClient_,
    /// The channel to send [`crate::L1WatcherQueries`]s.
    pub l1_watcher_sender: L1WatcherQuerySender,
    /// The record of the L2 safe head reached as of each L1 block.
    pub safe_db: Option<Arc<dyn SafeDb>>,
}

impl<EngineRpcClient_: EngineRpcClient> RollupRpc<EngineRpcClient_> {
    /// The identifier for the Metric that tracks rollup RPC calls.
    pub const RPC_IDENT: &'static str = "rollup_rpc";

    /// Constructs a new [`RollupRpc`] given a sender channel.
    pub const fn new(
        version: &'static str,
        engine_client: EngineRpcClient_,
        l1_watcher_sender: L1WatcherQuerySender,
        safe_db: Option<Arc<dyn SafeDb>>,
    ) -> Self {
        Self { engine_client, l1_watcher_sender, safe_db, version }
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

        let ((l2_block_info, output_root, l2_sync_status), l1_sync_status) = tokio::try_join!(
            self.engine_client.output_at_block(block_num),
            L1State::query(&self.l1_watcher_sender)
        )?;

        let sync_status = Self::sync_status_from_actor_queries(l1_sync_status, l2_sync_status);

        Ok(OutputResponse::from_v0(output_root, sync_status, l2_block_info))
    }

    async fn op_safe_head_at_l1_block(
        &self,
        block_num: BlockNumberOrTag,
    ) -> RpcResult<SafeHeadResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_safeHeadAtL1Block");

        // op-node takes a plain number here; `earliest` is the only tag that names one.
        let l1_block_num = match block_num {
            BlockNumberOrTag::Number(number) => number,
            BlockNumberOrTag::Earliest => 0,
            tag => {
                return Err(ErrorObject::owned(
                    ErrorCode::InvalidParams.code(),
                    format!("safeHeadAtL1Block requires an L1 block number, got {tag}"),
                    None::<()>,
                ));
            }
        };

        let record = self
            .safe_db
            .as_ref()
            .ok_or(SafeDbError::NotEnabled)
            .and_then(|database| database.safe_head_at_l1(l1_block_num))
            .map_err(|e| {
                safe_db_error(e, format_args!("failed to get safe head at l1 block {l1_block_num}"))
            })?;
        Ok(SafeHeadResponse { l1_block: record.l1, safe_head: record.safe_head })
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_syncStatus");

        let (l1_sync_status, l2_sync_status) = tokio::try_join!(
            L1State::query(&self.l1_watcher_sender),
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

        Ok(self.version.to_string())
    }
}
