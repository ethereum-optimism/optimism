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
use kona_genesis::{DependencySet, RollupConfig};
use kona_protocol::SyncStatus;
use std::fmt::Debug;

use crate::{
    DependencySetResponse, EngineRpcClient, L1State, L1WatcherQueries, OutputResponse,
    RollupNodeApiServer, SafeHeadResponse, l1_watcher::L1WatcherQuerySender,
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
    /// The L2 chain ID, pre-rendered as the `chain_id` metric label value.
    pub chain_id_label: std::sync::Arc<str>,
    /// The interop dependency set this chain was configured with, when it has one.
    ///
    /// [`None`] is the answer, not a missing input: a chain that schedules no interop has no
    /// dependency set to serve, and op-node reports that as not-found rather than as an empty set.
    pub dependency_set: Option<std::sync::Arc<DependencySet>>,
}

impl<EngineRpcClient_: EngineRpcClient> RollupRpc<EngineRpcClient_> {
    /// The identifier for the Metric that tracks rollup RPC calls.
    pub const RPC_IDENT: &'static str = "rollup_rpc";

    /// Constructs a new [`RollupRpc`] given a sender channel and the L2 chain ID.
    pub fn new(
        engine_client: EngineRpcClient_,
        l1_watcher_sender: L1WatcherQuerySender,
        chain_id: u64,
    ) -> Self {
        Self {
            engine_client,
            l1_watcher_sender,
            chain_id_label: kona_macros::chain_id_label(chain_id),
            dependency_set: None,
        }
    }

    /// Attaches the dependency set `optimism_dependencySet` answers from.
    ///
    /// Separate from [`Self::new`] so that a caller which only reads sync status — the supernode
    /// query API does — is not made to supply one.
    pub fn with_dependency_set(
        self,
        dependency_set: Option<std::sync::Arc<DependencySet>>,
    ) -> Self {
        Self { dependency_set, ..self }
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
            // The `safe_l2` wire field is the forkchoice safe label, i.e. the cross-safe head.
            safe_l2: l2_sync_status.sync_state.cross_safe_head(),
            finalized_l2: l2_sync_status.sync_state.finalized_head(),
        }
    }
}

#[async_trait]
impl<EngineRpcClient_: EngineRpcClient + 'static> RollupNodeApiServer
    for RollupRpc<EngineRpcClient_>
{
    async fn op_output_at_block(&self, block_num: BlockNumberOrTag) -> RpcResult<OutputResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_outputAtBlock", kona_macros::CHAIN_ID_LABEL => self.chain_id_label.clone());

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

    /// This RPC endpoint is not supported. It is not necessary to track the safe head for every L1
    /// block post-interop anymore so we can remove this method from the rpc interface.
    async fn op_safe_head_at_l1_block(
        &self,
        _block_num: BlockNumberOrTag,
    ) -> RpcResult<SafeHeadResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_safeHeadAtL1Block", kona_macros::CHAIN_ID_LABEL => self.chain_id_label.clone());
        return Err(ErrorObject::from(ErrorCode::MethodNotFound));
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_syncStatus", kona_macros::CHAIN_ID_LABEL => self.chain_id_label.clone());

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
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_rollupConfig", kona_macros::CHAIN_ID_LABEL => self.chain_id_label.clone());

        self.engine_client.get_config().await
    }

    async fn op_dependency_set(&self) -> RpcResult<DependencySetResponse> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_dependencySet", kona_macros::CHAIN_ID_LABEL => self.chain_id_label.clone());

        // op-node returns `ethereum.NotFound` when no set is configured, and the Go callers that
        // ask for one only ask on a chain that schedules Lagoon — so the error is the answer for a
        // chain that does not, rather than a failure to serve the method.
        self.dependency_set.as_deref().map(DependencySetResponse::from).ok_or_else(|| {
            ErrorObject::owned(ErrorCode::InternalError.code(), "not found", None::<()>)
        })
    }

    async fn op_version(&self) -> RpcResult<String> {
        kona_macros::inc!(gauge, Self::RPC_IDENT, "method" => "op_version", kona_macros::CHAIN_ID_LABEL => self.chain_id_label.clone());

        const RPC_VERSION: &str = env!("CARGO_PKG_VERSION");

        return Ok(RPC_VERSION.to_string());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_engine::LocalSafeSnapshot;
    use kona_genesis::ChainDependency;
    use kona_protocol::{L2BlockInfo, OutputRoot};
    use std::{collections::BTreeMap, sync::Arc};
    use tokio::sync::watch;

    /// An engine client that answers nothing.
    ///
    /// `optimism_dependencySet` is a statement about how the chain was configured, not about what
    /// it has derived, so it must be answerable without reaching the engine at all — which is what
    /// this stub asserts by making any such reach panic.
    #[derive(Debug, Clone)]
    struct NoEngine;

    #[async_trait]
    impl EngineRpcClient for NoEngine {
        async fn get_config(&self) -> RpcResult<RollupConfig> {
            unimplemented!("the dependency set is not read from the engine")
        }
        async fn get_state(&self) -> RpcResult<EngineState> {
            unimplemented!("the dependency set is not read from the engine")
        }
        async fn output_at_block(
            &self,
            _block: BlockNumberOrTag,
        ) -> RpcResult<(L2BlockInfo, OutputRoot, EngineState)> {
            unimplemented!("the dependency set is not read from the engine")
        }
        async fn local_safe_snapshot_at(&self, _timestamp: u64) -> RpcResult<LocalSafeSnapshot> {
            unimplemented!("the dependency set is not read from the engine")
        }
        async fn dev_get_task_queue_length(&self) -> RpcResult<usize> {
            unimplemented!("the dependency set is not read from the engine")
        }
        async fn dev_subscribe_to_engine_queue_length(&self) -> RpcResult<watch::Receiver<usize>> {
            unimplemented!("the dependency set is not read from the engine")
        }
        async fn dev_subscribe_to_engine_state(&self) -> RpcResult<watch::Receiver<EngineState>> {
            unimplemented!("the dependency set is not read from the engine")
        }
    }

    fn rollup_rpc(dependency_set: Option<DependencySet>) -> RollupRpc<NoEngine> {
        let (l1_watcher_sender, _rx) = tokio::sync::mpsc::channel(1);
        RollupRpc::new(NoEngine, l1_watcher_sender, 901)
            .with_dependency_set(dependency_set.map(Arc::new))
    }

    /// A configured chain serves its set, in op-node's wire shape.
    #[tokio::test]
    async fn serves_the_configured_dependency_set() {
        let rpc = rollup_rpc(Some(DependencySet {
            dependencies: BTreeMap::from([(901, ChainDependency {}), (902, ChainDependency {})]),
            override_message_expiry_window: None,
        }));

        let response = rpc.op_dependency_set().await.expect("a configured chain answers");

        assert_eq!(
            serde_json::to_value(&response).unwrap(),
            serde_json::json!({"dependencies": {"901": {}, "902": {}}}),
        );
    }

    /// A chain with no set configured reports not-found, which is what op-node's
    /// `nodeAPI.DependencySet` returns through `ethereum.NotFound`.
    #[tokio::test]
    async fn reports_not_found_without_a_dependency_set() {
        let err = rollup_rpc(None)
            .op_dependency_set()
            .await
            .expect_err("a chain with no set configured has nothing to serve");

        assert_eq!(err.message(), "not found");
    }
}
