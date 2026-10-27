//! The `superroot_atTimestamp` server.

use crate::{
    DecimalChainId, L1State, OutputV0Response, OutputWithRequiredL1, SuperRootApiServer,
    SuperRootAtTimestampResponse, SuperRootResponseData, SuperV1Response,
    l1_watcher::L1WatcherQuerySender, safe_db_error::safe_db_error,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, U64};
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::{EngineQueryClient, EngineState, EngineSyncState};
use kona_genesis::RollupConfig;
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_protocol::OutputRoot;
use kona_safedb::{SafeDb, SafeDbError, SafeHeadRecord};
use std::{collections::BTreeMap, sync::Arc};

const MAX_HISTORY_READ_ATTEMPTS: usize = 3;

/// The server for [`SuperRootApiServer`].
#[derive(Debug)]
pub struct SuperRootRpc {
    engine_client: EngineQueryClient,
    engine_state: tokio::sync::watch::Receiver<EngineState>,
    l1_watcher_sender: L1WatcherQuerySender,
    rollup_config: Arc<RollupConfig>,
    safe_db: Option<Arc<dyn SafeDb>>,
}

impl SuperRootRpc {
    /// Constructs a new [`SuperRootRpc`].
    pub const fn new(
        engine_client: EngineQueryClient,
        engine_state: tokio::sync::watch::Receiver<EngineState>,
        l1_watcher_sender: L1WatcherQuerySender,
        rollup_config: Arc<RollupConfig>,
        safe_db: Option<Arc<dyn SafeDb>>,
    ) -> Self {
        Self { engine_client, engine_state, l1_watcher_sender, rollup_config, safe_db }
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

    fn safe_head_record(&self, block_number: u64) -> RpcResult<SafeHeadRecord> {
        if block_number == self.rollup_config.genesis.l2.number {
            return Ok(SafeHeadRecord {
                l1: BlockNumHash { hash: B256::ZERO, number: 0 },
                safe_head: self.rollup_config.genesis.l2,
            });
        }
        self.safe_db
            .as_ref()
            .ok_or(SafeDbError::NotEnabled)
            .and_then(|database| database.l1_at_safe_head(block_number))
            .map_err(|error| safe_db_error(error, format_args!("l1AtSafeHead@{block_number}")))
    }

    async fn verified_output(
        &self,
        block_number: u64,
    ) -> RpcResult<Option<(OutputRoot, BlockNumHash, EngineSyncState)>> {
        for attempt in 1..=MAX_HISTORY_READ_ATTEMPTS {
            let sync = self.engine_state.borrow().sync_state;
            if block_number > sync.local_safe_head().block_info.number {
                return Ok(None);
            }
            let record = self.safe_head_record(block_number)?;
            if record.safe_head.number > sync.local_safe_head().block_info.number {
                tracing::warn!(attempt, block_number, recorded_head = ?record.safe_head, "Recorded history is ahead of the local-safe head; retrying");
                continue;
            }
            let output = self
                .engine_client
                .output_at_ancestor(block_number, record.safe_head)
                .await
                .map_err(|error| {
                    tracing::error!(block_number, ?error, "Failed to read output for super root");
                    ErrorObject::from(ErrorCode::InternalError)
                })?;
            let current_sync = self.engine_state.borrow().sync_state;
            let history_unchanged = self.safe_head_record(block_number)? == record;
            let safe_head_unchanged = current_sync.local_safe_head() == sync.local_safe_head();
            if let Some((block, output)) = output {
                let within_safe_head = block.block_info.number <
                    sync.local_safe_head().block_info.number ||
                    block.block_info.id() == sync.local_safe_head().block_info.id();
                if history_unchanged &&
                    safe_head_unchanged &&
                    within_safe_head &&
                    block.block_info.number == block_number
                {
                    return Ok(Some((output, record.l1, current_sync)));
                }
            }
            tracing::warn!(
                attempt,
                block_number,
                "Safe head history changed during super-root query; retrying"
            );
        }
        tracing::error!(
            attempts = MAX_HISTORY_READ_ATTEMPTS,
            block_number,
            "Unable to read consistent safe head history"
        );
        Err(ErrorObject::owned(-32000, "safe head history changed during query; retry", None::<()>))
    }
}

#[async_trait]
impl SuperRootApiServer for SuperRootRpc {
    async fn superroot_at_timestamp(
        &self,
        timestamp: U64,
    ) -> RpcResult<SuperRootAtTimestampResponse> {
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

        let current_l1 = self.current_l1().await?;
        let Some((output, required_l1, sync)) = self.verified_output(block_number).await? else {
            return Ok(self.progress_only(current_l1, &self.engine_state.borrow().sync_state));
        };
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
