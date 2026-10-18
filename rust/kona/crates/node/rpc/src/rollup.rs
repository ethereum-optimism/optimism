//! Implements the rollup client rpc endpoints. These endpoints serve data about the rollup state.
//!
//! Implemented in the op-node in <https://github.com/ethereum-optimism/optimism/blob/174e55f0a1e73b49b80a561fd3fedd4fea5770c6/op-service/sources/rollupclient.go#L16>

use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_provider::Provider;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::EngineState;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OutputRoot, Predeploys, SyncStatus};
use op_alloy_network::Optimism;
use std::sync::Arc;
use tokio::sync::watch;

use crate::{
    L1State, L1WatcherQueries, OutputResponse, RollupNodeApiServer, SafeHeadResponse,
    l1_watcher::L1WatcherQuerySender,
};

/// Rollup RPC reads use the node's provider, chain rules and published engine state.
#[derive(Debug)]
pub struct RollupRpc<P> {
    provider: P,
    config: Arc<RollupConfig>,
    state: watch::Receiver<EngineState>,
    version: &'static str,
    l1_watcher_sender: L1WatcherQuerySender,
}

impl<P: Provider<Optimism>> RollupRpc<P> {
    /// The identifier for the metric that tracks rollup RPC calls.
    pub const RPC_IDENT: &'static str = "rollup_rpc";

    /// Constructs the handler from its read dependencies and the binary's release version.
    pub const fn new(
        provider: P,
        config: Arc<RollupConfig>,
        state: watch::Receiver<EngineState>,
        version: &'static str,
        l1_watcher_sender: L1WatcherQuerySender,
    ) -> Self {
        Self { provider, config, state, version, l1_watcher_sender }
    }

    async fn l1_state(&self) -> RpcResult<L1State> {
        let (sender, receiver) = tokio::sync::oneshot::channel();
        self.l1_watcher_sender
            .send(L1WatcherQueries::L1State(sender))
            .await
            .map_err(internal_error)?;
        receiver.await.map_err(internal_error)
    }

    fn engine_state(&self) -> RpcResult<EngineState> {
        self.state.has_changed().map_err(internal_error)?;
        Ok(*self.state.borrow())
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
impl<P: Provider<Optimism> + 'static> RollupNodeApiServer for RollupRpc<P> {
    async fn op_output_at_block(&self, block_num: BlockNumberOrTag) -> RpcResult<OutputResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_outputAtBlock");
        let state = self.engine_state()?;
        let ((block_info, output), l1_state) = tokio::try_join!(
            async {
                let block = self
                    .provider
                    .get_block_by_number(block_num)
                    .full()
                    .await
                    .map_err(internal_error)?
                    .ok_or_else(|| {
                        internal_error(format_args!("No L2 block found for {block_num}"))
                    })?;
                let header = block.header.clone();
                let block_info =
                    L2BlockInfo::from_block_and_genesis::<op_alloy_consensus::OpTxEnvelope>(
                        &block.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner()),
                        &self.config.genesis,
                    )
                    .map_err(internal_error)?;
                let storage_root = if self.config.is_isthmus_active(header.timestamp) {
                    header
                        .withdrawals_root
                        .ok_or_else(|| internal_error("No withdrawals root after Isthmus"))?
                } else {
                    // The tag may move or reorg while this request is in flight. Both roots must
                    // describe the exact block returned above; reject a reorged-out hash.
                    self.provider
                        .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Vec::new())
                        .block_id(BlockId::hash_canonical(header.hash))
                        .await
                        .map_err(internal_error)?
                        .storage_hash
                };
                Ok((
                    block_info,
                    OutputRoot::from_parts(header.state_root, storage_root, header.hash),
                ))
            },
            self.l1_state()
        )?;
        let sync_status = Self::sync_status_from_actor_queries(l1_state, state);
        Ok(OutputResponse::from_v0(output, sync_status, block_info))
    }

    /// This RPC endpoint is not supported. It is not necessary to track the safe head for every L1
    /// block post-interop anymore so we can remove this method from the rpc interface.
    async fn op_safe_head_at_l1_block(
        &self,
        _block_num: BlockNumberOrTag,
    ) -> RpcResult<SafeHeadResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_safeHeadAtL1Block");
        return Err(ErrorObject::from(ErrorCode::MethodNotFound));
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_syncStatus");

        let state = self.engine_state()?;
        Ok(Self::sync_status_from_actor_queries(self.l1_state().await?, state))
    }

    async fn op_rollup_config(&self) -> RpcResult<RollupConfig> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_rollupConfig");

        Ok((*self.config).clone())
    }

    async fn op_version(&self) -> RpcResult<String> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_version");

        Ok(self.version.to_string())
    }
}

fn internal_error(error: impl std::fmt::Display) -> ErrorObject<'static> {
    warn!(target: "rpc::rollup", %error, "Rollup RPC read failed");
    ErrorObject::from(ErrorCode::InternalError)
}
