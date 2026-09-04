//! Implements the rollup client rpc endpoints. These endpoints serve data about the rollup state.
//!
//! Implemented in the op-node in <https://github.com/ethereum-optimism/optimism/blob/174e55f0a1e73b49b80a561fd3fedd4fea5770c6/op-service/sources/rollupclient.go#L16>

use alloy_eips::BlockNumberOrTag;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_chainview::{ChainViewClient, L1Statuses};
use kona_engine::EngineState;
use kona_genesis::RollupConfig;
use kona_protocol::SyncStatus;
use std::fmt::Debug;

use crate::{
    EngineRpcClient, OutputResponse, RollupNodeApiServer, SafeHeadResponse,
    l1_watcher::L1WatcherQuerySender,
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
    /// The chain view: the L1 side of the sync status comes from its snapshot, with no L1 RPC
    /// on the request path, and `optimism_safeHeadAtL1Block` from its safe-head history.
    chainview: ChainViewClient,
}

impl<EngineRpcClient_: EngineRpcClient> RollupRpc<EngineRpcClient_> {
    /// The identifier for the Metric that tracks rollup RPC calls.
    pub const RPC_IDENT: &'static str = "rollup_rpc";

    /// Constructs a new [`RollupRpc`].
    pub const fn new(
        engine_client: EngineRpcClient_,
        l1_watcher_sender: L1WatcherQuerySender,
        chainview: ChainViewClient,
    ) -> Self {
        Self { engine_client, l1_watcher_sender, chainview }
    }

    /// The sync status from the chain view's L1 statuses and the engine's L2 heads. Fields
    /// not known yet are zeroed, as op-node does.
    fn sync_status(l1: &L1Statuses, l2_sync_status: EngineState) -> SyncStatus {
        SyncStatus {
            // op-node reports the derivation origin here; fall back to the head until
            // derivation has advanced an origin.
            current_l1: l1.current.or(l1.head).unwrap_or_default(),
            // A legacy attribute that op-node documents as matching `finalized_l1`.
            current_l1_finalized: l1.finalized.unwrap_or_default(),
            head_l1: l1.head.unwrap_or_default(),
            safe_l1: l1.safe.unwrap_or_default(),
            finalized_l1: l1.finalized.unwrap_or_default(),
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

        let (l2_block_info, output_root, l2_sync_status) =
            self.engine_client.output_at_block(block_num).await?;
        let sync_status = Self::sync_status(&self.chainview.snapshot().l1, l2_sync_status);

        Ok(OutputResponse::from_v0(output_root, sync_status, l2_block_info))
    }

    /// The safe L2 head after derivation consumed the nearest L1 block at or below `block_num`
    /// (op-node's `SafeDB` semantics), from the chain view's safe-head history.
    async fn op_safe_head_at_l1_block(
        &self,
        block_num: BlockNumberOrTag,
    ) -> RpcResult<SafeHeadResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_safeHeadAtL1Block");
        let BlockNumberOrTag::Number(number) = block_num else {
            return Err(ErrorObject::owned(
                ErrorCode::InvalidParams.code(),
                "safeHeadAtL1Block takes an L1 block number",
                None::<()>,
            ));
        };
        let entry = self
            .chainview
            .safe_head_at_l1(number)
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        // op-node's client maps any error mentioning "not found" to its safedb.ErrNotFound;
        // keep that wording.
        entry.map_or_else(
            || {
                Err(ErrorObject::owned(
                    ErrorCode::ServerError(-32000).code(),
                    format!("safe head not found for L1 block {number}"),
                    None::<()>,
                ))
            },
            |entry| Ok(SafeHeadResponse { l1_block: entry.l1, safe_head: entry.l2 }),
        )
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_syncStatus");

        let l2_sync_status = self
            .engine_client
            .get_state()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        Ok(Self::sync_status(&self.chainview.snapshot().l1, l2_sync_status))
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
