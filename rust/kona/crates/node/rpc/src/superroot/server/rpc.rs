//! The `superroot_atTimestamp` server.

use crate::{
    DecimalChainId, EngineRpcClient, L1State, OutputV0Response, OutputWithRequiredL1,
    SuperRootApiServer, SuperRootAtTimestampResponse, SuperRootResponseData, SuperV1Response,
    l1_watcher::L1WatcherQuerySender, safe_db_error::safe_db_error,
};
use alloy_eips::{BlockNumHash, BlockNumberOrTag};
use alloy_primitives::{B256, U64};
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::EngineSyncState;
use kona_genesis::RollupConfig;
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_safedb::{SafeDb, SafeDbError};
use std::{collections::BTreeMap, fmt::Debug, sync::Arc};

/// The server for [`SuperRootApiServer`].
#[derive(Debug)]
pub struct SuperRootRpc<EngineRpcClient_> {
    engine_client: EngineRpcClient_,
    l1_watcher_sender: L1WatcherQuerySender,
    rollup_config: Arc<RollupConfig>,
    safe_db: Option<Arc<dyn SafeDb>>,
}

impl<EngineRpcClient_: EngineRpcClient> SuperRootRpc<EngineRpcClient_> {
    /// The identifier for the metric that tracks super root RPC calls.
    pub const RPC_IDENT: &'static str = "superroot_rpc";

    /// Constructs a new [`SuperRootRpc`].
    pub const fn new(
        engine_client: EngineRpcClient_,
        l1_watcher_sender: L1WatcherQuerySender,
        rollup_config: Arc<RollupConfig>,
        safe_db: Option<Arc<dyn SafeDb>>,
    ) -> Self {
        Self { engine_client, l1_watcher_sender, rollup_config, safe_db }
    }

    fn chain_id(&self) -> DecimalChainId {
        DecimalChainId(self.rollup_config.l2_chain_id.id())
    }

    async fn current_l1(&self) -> RpcResult<BlockNumHash> {
        let l1_state = L1State::query(&self.l1_watcher_sender).await?;
        Ok(l1_state.current_l1.map(|block| block.id()).unwrap_or_default())
    }

    /// The part of the response that describes the node's progress rather than the requested
    /// block. It is all that is returned while the block at the timestamp is not yet local-safe.
    fn progress_only(
        &self,
        current_l1: BlockNumHash,
        sync: &EngineSyncState,
    ) -> SuperRootAtTimestampResponse {
        SuperRootAtTimestampResponse {
            current_l1,
            current_safe_timestamp: sync.safe_head().block_info.timestamp,
            current_local_safe_timestamp: sync.local_safe_head().block_info.timestamp,
            current_finalized_timestamp: sync.finalized_head().block_info.timestamp,
            optimistic_at_timestamp: BTreeMap::new(),
            chain_ids: vec![self.chain_id()],
            data: None,
        }
    }

    /// The L1 block that first made `block` safe. The L2 genesis block needs no L1 data, so it
    /// is reported as safe from L1 block zero rather than from the configured genesis L1 block,
    /// since the chain's contracts may predate it.
    fn required_l1(&self, block: BlockNumHash) -> RpcResult<BlockNumHash> {
        if block == self.rollup_config.genesis.l2 {
            return Ok(BlockNumHash { hash: B256::ZERO, number: 0 });
        }
        let record = self
            .safe_db
            .as_ref()
            .ok_or(SafeDbError::NotEnabled)
            .and_then(|database| database.l1_at_safe_head(block.number))
            .map_err(|e| safe_db_error(e, format_args!("l1AtSafeHead@{}", block.number)))?;
        Ok(record.l1)
    }
}

/// Whether `block` is local-safe in `sync`: at or below the local-safe head, and when it is the
/// head itself, the very block the head names. A reset between two reads can otherwise pair the
/// output of a replacement block with the record of the block it replaced.
fn is_local_safe(sync: &EngineSyncState, block: BlockNumHash) -> bool {
    let local_safe = sync.local_safe_head().block_info;
    block.number < local_safe.number || block == local_safe.id()
}

#[async_trait]
impl<EngineRpcClient_: EngineRpcClient + 'static> SuperRootApiServer
    for SuperRootRpc<EngineRpcClient_>
{
    async fn superroot_at_timestamp(
        &self,
        timestamp: U64,
    ) -> RpcResult<SuperRootAtTimestampResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "superroot_atTimestamp");
        let timestamp = timestamp.to::<u64>();
        let config = &self.rollup_config;
        if timestamp < config.genesis.l2_time {
            return Err(ErrorObject::owned(
                ErrorCode::InvalidParams.code(),
                format!(
                    "timestamp {timestamp} precedes the L2 genesis time {}",
                    config.genesis.l2_time
                ),
                None::<()>,
            ));
        }
        let block_number = config.block_number_from_timestamp(timestamp);

        let (current_l1, engine_state) =
            tokio::try_join!(self.current_l1(), self.engine_client.get_state())?;
        if block_number > engine_state.sync_state.local_safe_head().block_info.number {
            return Ok(self.progress_only(current_l1, &engine_state.sync_state));
        }

        // The output and the local-safe bound must describe the same engine state.
        let (block, output, engine_state) =
            self.engine_client.output_at_block(BlockNumberOrTag::Number(block_number)).await?;
        let sync = engine_state.sync_state;
        if !is_local_safe(&sync, block.block_info.id()) {
            return Ok(self.progress_only(current_l1, &sync));
        }

        let required_l1 = self.required_l1(block.block_info.id())?;
        let output_root = output.hash();
        let chain_id = self.chain_id();
        let super_root =
            SuperRoot::new(timestamp, vec![OutputRootWithChain::new(chain_id.0, output_root)]);

        let mut response = self.progress_only(current_l1, &sync);
        response.optimistic_at_timestamp.insert(
            chain_id,
            OutputWithRequiredL1 {
                output: OutputV0Response::from(&output),
                output_root,
                required_l1,
            },
        );
        response.data = Some(SuperRootResponseData {
            verified_required_l1: required_l1,
            super_v1: SuperV1Response::from(&super_root),
            super_root: super_root.hash(),
        });
        Ok(response)
    }
}
