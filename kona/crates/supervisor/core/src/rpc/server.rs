//! Server-side implementation of the Supervisor RPC API.

use super::Metrics;
use crate::{SpecError, SupervisorError, SupervisorService};
use alloy_eips::eip1898::BlockNumHash;
use alloy_primitives::{B256, ChainId, map::HashMap};
use async_trait::async_trait;
use jsonrpsee::{core::RpcResult, types::ErrorObject};
use kona_interop::{DependencySet, DerivedIdPair, ExecutingDescriptor, SafetyLevel};
use kona_protocol::BlockInfo;
use kona_supervisor_rpc::{
    SuperRootOutputRpc, SupervisorApiServer, SupervisorChainSyncStatus, SupervisorSyncStatus,
};
use kona_supervisor_types::{HexStringU64, SuperHead};
use std::sync::Arc;
use tracing::{trace, warn};

/// The server-side implementation struct for the [`SupervisorApiServer`].
/// It holds a reference to the core Supervisor logic.
#[derive(Debug)]
pub struct SupervisorRpc<T> {
    /// Reference to the core Supervisor logic.
    /// Using Arc allows sharing the Supervisor instance if needed,
    supervisor: Arc<T>,
}

impl<T> SupervisorRpc<T> {
    /// Creates a new [`SupervisorRpc`] instance.
    pub fn new(supervisor: Arc<T>) -> Self {
        Metrics::init();
        trace!(target: "supervisor::rpc", "Creating new SupervisorRpc handler");
        Self { supervisor }
    }
}

#[async_trait]
impl<T> SupervisorApiServer for SupervisorRpc<T>
where
    T: SupervisorService + 'static,
{
    async fn cross_derived_to_source(
        &self,
        chain_id_hex: HexStringU64,
        derived: BlockNumHash,
    ) -> RpcResult<BlockInfo> {
        let chain_id = ChainId::from(chain_id_hex);
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_CROSS_DERIVED_TO_SOURCE,
            async {
                trace!(
                    target: "supervisor::rpc",
                    %chain_id,
                    ?derived,
                    "Received cross_derived_to_source request"
                );

                let source_block =
                    self.supervisor.derived_to_source_block(chain_id, derived).map_err(|err| {
                        warn!(
                            target: "supervisor::rpc",
                            %chain_id,
                            ?derived,
                            %err,
                            "Failed to get source block for derived block"
                        );
                        ErrorObject::from(err)
                    })?;

                Ok(source_block)
            }
            .await
        )
    }

    async fn local_unsafe(&self, chain_id_hex: HexStringU64) -> RpcResult<BlockNumHash> {
        let chain_id = ChainId::from(chain_id_hex);
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_LOCAL_UNSAFE,
            async {
                trace!(target: "supervisor::rpc",
                    %chain_id,
                    "Received local_unsafe request"
                );

                Ok(self.supervisor.local_unsafe(chain_id)?.id())
            }
            .await
        )
    }

    async fn local_safe(&self, chain_id_hex: HexStringU64) -> RpcResult<DerivedIdPair> {
        let chain_id = ChainId::from(chain_id_hex);
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_LOCAL_SAFE,
            async {
                trace!(target: "supervisor::rpc",
                    %chain_id,
                    "Received local_safe request"
                );

                let derived = self.supervisor.local_safe(chain_id)?.id();
                let source = self.supervisor.derived_to_source_block(chain_id, derived)?.id();

                Ok(DerivedIdPair { source, derived })
            }
            .await
        )
    }

    async fn dependency_set_v1(&self) -> RpcResult<DependencySet> {
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_DEPENDENCY_SET,
            async {
                trace!(target: "supervisor::rpc",
                    "Received the dependency set"
                );

                Ok(self.supervisor.dependency_set().to_owned())
            }
            .await
        )
    }

    async fn cross_safe(&self, chain_id_hex: HexStringU64) -> RpcResult<DerivedIdPair> {
        let chain_id = ChainId::from(chain_id_hex);
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_CROSS_SAFE,
            async {
                trace!(target: "supervisor::rpc",
                    %chain_id,
                    "Received cross_safe request"
                );

                let derived = self.supervisor.cross_safe(chain_id)?.id();
                let source = self.supervisor.derived_to_source_block(chain_id, derived)?.id();

                Ok(DerivedIdPair { source, derived })
            }
            .await
        )
    }

    async fn finalized(&self, chain_id_hex: HexStringU64) -> RpcResult<BlockNumHash> {
        let chain_id = ChainId::from(chain_id_hex);
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_FINALIZED,
            async {
                trace!(target: "supervisor::rpc",
                    %chain_id,
                    "Received finalized request"
                );

                Ok(self.supervisor.finalized(chain_id)?.id())
            }
            .await
        )
    }

    async fn finalized_l1(&self) -> RpcResult<BlockInfo> {
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_FINALIZED_L1,
            async {
                trace!(target: "supervisor::rpc", "Received finalized_l1 request");
                Ok(self.supervisor.finalized_l1()?)
            }
            .await
        )
    }

    async fn super_root_at_timestamp(
        &self,
        timestamp_hex: HexStringU64,
    ) -> RpcResult<SuperRootOutputRpc> {
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_SUPER_ROOT_AT_TIMESTAMP,
            async {
                let timestamp = u64::from(timestamp_hex);
                trace!(target: "supervisor::rpc",
                    %timestamp,
                    "Received super_root_at_timestamp request"
                );

                self.supervisor.super_root_at_timestamp(timestamp)
                    .await
                    .map_err(|err| {
                        warn!(target: "supervisor::rpc", %err, "Error from core supervisor super_root_at_timestamp");
                        ErrorObject::from(err)
                    })
            }.await
        )
    }

    async fn check_access_list(
        &self,
        inbox_entries: Vec<B256>,
        min_safety: SafetyLevel,
        executing_descriptor: ExecutingDescriptor,
    ) -> RpcResult<()> {
        // TODO:: refactor, maybe build proc macro to record metrics
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_CHECK_ACCESS_LIST,
            async {
                trace!(target: "supervisor::rpc", 
                    num_inbox_entries = inbox_entries.len(),
                    ?min_safety,
                    ?executing_descriptor,
                    "Received check_access_list request",
                );
                self.supervisor
                    .check_access_list(inbox_entries, min_safety, executing_descriptor)
                    .map_err(|err| {
                        warn!(target: "supervisor::rpc", %err, "Error from core supervisor check_access_list");
                        ErrorObject::from(err)
                    })
            }.await
        )
    }

    async fn sync_status(&self) -> RpcResult<SupervisorSyncStatus> {
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_SYNC_STATUS,
            async {
                trace!(target: "supervisor::rpc", "Received sync_status request");

                let mut chains = self
                    .supervisor
                    .chain_ids()
                    .map(|id| (id, Default::default()))
                    .collect::<HashMap<_, SupervisorChainSyncStatus>>();

                if chains.is_empty() {
                    // return error if no chains configured
                    //
                    // <https://github.com/ethereum-optimism/optimism/blob/fac40575a8bcefd325c50a52e12b0e93254ac3f8/op-supervisor/supervisor/backend/status/status.go#L100-L104>
                    //
                    // todo: add to spec
                    Err(SupervisorError::EmptyDependencySet)?;
                }

                let mut min_synced_l1 = BlockInfo { number: u64::MAX, ..Default::default() };
                let mut cross_safe_timestamp = u64::MAX;
                let mut finalized_timestamp = u64::MAX;
                let mut uninitialized_chain_db_count = 0;

                for (id, status) in chains.iter_mut() {
                    let head = match self.supervisor.super_head(*id) {
                        Ok(head) => head,
                        Err(SupervisorError::SpecError(SpecError::ErrorNotInSpec)) => {
                            uninitialized_chain_db_count += 1;
                            continue;
                        }
                        Err(err) => return Err(ErrorObject::from(err)),
                    };

                    // uses lowest safe and finalized timestamps, as well as l1 block, of all l2s
                    //
                    // <https://github.com/ethereum-optimism/optimism/blob/fac40575a8bcefd325c50a52e12b0e93254ac3f8/op-supervisor/supervisor/backend/status/status.go#L117-L131>
                    //
                    // todo: add to spec
                    let SuperHead { l1_source, cross_safe, finalized, .. } = &head;

                    let default_block = BlockInfo::default();
                    let l1_source = l1_source.as_ref().unwrap_or(&default_block);
                    let cross_safe = cross_safe.as_ref().unwrap_or(&default_block);
                    let finalized = finalized.as_ref().unwrap_or(&default_block);

                    if l1_source.number < min_synced_l1.number {
                        min_synced_l1 = *l1_source;
                    }
                    if cross_safe.timestamp < cross_safe_timestamp {
                        cross_safe_timestamp = cross_safe.timestamp;
                    }
                    if finalized.timestamp < finalized_timestamp {
                        finalized_timestamp = finalized.timestamp;
                    }

                    *status = head.into();
                }

                if uninitialized_chain_db_count == chains.len() {
                    warn!(target: "supervisor::rpc", "No chain db initialized");
                    return Err(ErrorObject::from(SupervisorError::SpecError(
                        SpecError::ErrorNotInSpec,
                    )));
                }

                Ok(SupervisorSyncStatus {
                    min_synced_l1,
                    cross_safe_timestamp,
                    finalized_timestamp,
                    chains,
                })
            }
            .await
        )
    }

    async fn all_safe_derived_at(
        &self,
        derived_from: BlockNumHash,
    ) -> RpcResult<HashMap<ChainId, BlockNumHash>> {
        crate::observe_rpc_call!(
            Metrics::SUPERVISOR_RPC_METHOD_ALL_SAFE_DERIVED_AT,
            async {
                trace!(target: "supervisor::rpc",
                    ?derived_from,
                    "Received all_safe_derived_at request"
                );

                let mut chains = self
                    .supervisor
                    .chain_ids()
                    .map(|id| (id, Default::default()))
                    .collect::<HashMap<_, BlockNumHash>>();

                for (id, block) in chains.iter_mut() {
                    *block = self.supervisor.latest_block_from(derived_from, *id)?.id();
                }

                Ok(chains)
            }
            .await
        )
    }
}

impl<T> Clone for SupervisorRpc<T> {
    fn clone(&self) -> Self {
        Self { supervisor: self.supervisor.clone() }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::ChainId;
    use kona_protocol::BlockInfo;
    use kona_supervisor_storage::{EntryNotFoundError, StorageError};
    use mockall::*;
    use std::sync::Arc;

    mock!(
        #[derive(Debug)]
        pub SupervisorService {}

        #[async_trait]
        impl SupervisorService for SupervisorService {
            fn chain_ids(&self) -> impl Iterator<Item = ChainId>;
            fn dependency_set(&self) -> &DependencySet;
            fn super_head(&self, chain: ChainId) -> Result<SuperHead, SupervisorError>;
            fn latest_block_from(&self, l1_block: BlockNumHash, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn derived_to_source_block(&self, chain: ChainId, derived: BlockNumHash) -> Result<BlockInfo, SupervisorError>;
            fn local_unsafe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn local_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn cross_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn finalized(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn finalized_l1(&self) -> Result<BlockInfo, SupervisorError>;
            fn check_access_list(&self, inbox_entries: Vec<B256>, min_safety: SafetyLevel, executing_descriptor: ExecutingDescriptor) -> Result<(), SupervisorError>;
            async fn super_root_at_timestamp(&self, timestamp: u64) -> Result<SuperRootOutputRpc, SupervisorError>;
        }
    );

    #[tokio::test]
    async fn test_sync_status_empty_chains() {
        let mut mock_service = MockSupervisorService::new();
        mock_service.expect_chain_ids().returning(|| Box::new(vec![].into_iter()));

        let rpc = SupervisorRpc::new(Arc::new(mock_service));
        let result = rpc.sync_status().await;

        assert!(result.is_err());
        assert_eq!(result.unwrap_err(), ErrorObject::from(SupervisorError::EmptyDependencySet));
    }

    #[tokio::test]
    async fn test_sync_status_single_chain() {
        let chain_id = ChainId::from(1u64);

        let block_info = BlockInfo { number: 42, ..Default::default() };
        let super_head = SuperHead {
            l1_source: Some(block_info),
            cross_safe: Some(BlockInfo { timestamp: 100, ..Default::default() }),
            finalized: Some(BlockInfo { timestamp: 50, ..Default::default() }),
            ..Default::default()
        };

        let mut mock_service = MockSupervisorService::new();
        mock_service.expect_chain_ids().returning(move || Box::new(vec![chain_id].into_iter()));
        mock_service.expect_super_head().returning(move |_| Ok(super_head));

        let rpc = SupervisorRpc::new(Arc::new(mock_service));
        let result = rpc.sync_status().await.unwrap();

        assert_eq!(result.min_synced_l1.number, 42);
        assert_eq!(result.cross_safe_timestamp, 100);
        assert_eq!(result.finalized_timestamp, 50);
        assert_eq!(result.chains.len(), 1);
    }

    #[tokio::test]
    async fn test_sync_status_missing_super_head() {
        let chain_id_1 = ChainId::from(1u64);
        let chain_id_2 = ChainId::from(2u64);

        // Only chain_id_1 has a SuperHead, chain_id_2 is missing
        let block_info = BlockInfo { number: 42, ..Default::default() };
        let super_head = SuperHead {
            l1_source: Some(block_info),
            cross_safe: Some(BlockInfo { timestamp: 100, ..Default::default() }),
            finalized: Some(BlockInfo { timestamp: 50, ..Default::default() }),
            ..Default::default()
        };

        let mut mock_service = MockSupervisorService::new();
        mock_service
            .expect_chain_ids()
            .returning(move || Box::new(vec![chain_id_1, chain_id_2].into_iter()));
        mock_service.expect_super_head().returning(move |chain_id| {
            if chain_id == chain_id_1 {
                Ok(super_head)
            } else {
                Err(SupervisorError::StorageError(StorageError::EntryNotFound(
                    EntryNotFoundError::DerivedBlockNotFound(1),
                )))
            }
        });

        let rpc = SupervisorRpc::new(Arc::new(mock_service));
        let result = rpc.sync_status().await;

        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_sync_status_uninitialized_chain_db() {
        let chain_id_1 = ChainId::from(1u64);
        let chain_id_2 = ChainId::from(2u64);

        // Case 1: No chain db is initialized
        let mut mock_service = MockSupervisorService::new();
        mock_service
            .expect_chain_ids()
            .returning(move || Box::new(vec![chain_id_1, chain_id_2].into_iter()));
        mock_service
            .expect_super_head()
            .times(2)
            .returning(move |_| Err(SupervisorError::SpecError(SpecError::ErrorNotInSpec)));

        let rpc = SupervisorRpc::new(Arc::new(mock_service));
        let result = rpc.sync_status().await;
        assert!(result.is_err());
        assert_eq!(
            result.unwrap_err(),
            ErrorObject::from(SupervisorError::SpecError(SpecError::ErrorNotInSpec,))
        );

        // Case 2: Only one chain db is initialized
        let mut super_head_map = std::collections::HashMap::new();
        let block_info = BlockInfo { number: 42, ..Default::default() };
        let super_head = SuperHead {
            l1_source: Some(block_info),
            cross_safe: Some(BlockInfo { timestamp: 100, ..Default::default() }),
            finalized: Some(BlockInfo { timestamp: 50, ..Default::default() }),
            ..Default::default()
        };
        super_head_map.insert(chain_id_1, super_head);

        let mut mock_service = MockSupervisorService::new();
        mock_service
            .expect_chain_ids()
            .returning(move || Box::new(vec![chain_id_1, chain_id_2].into_iter()));
        mock_service.expect_super_head().times(2).returning(move |chain_id| {
            if chain_id == chain_id_1 {
                Ok(super_head)
            } else {
                Err(SupervisorError::SpecError(SpecError::ErrorNotInSpec))
            }
        });

        let rpc = SupervisorRpc::new(Arc::new(mock_service));
        let result = rpc.sync_status().await;
        assert!(result.is_ok());

        // Case 3: Both chain dbs are initialized
        let mut super_head_map = std::collections::HashMap::new();
        let block_info_1 = BlockInfo { number: 42, ..Default::default() };
        let super_head_1 = SuperHead {
            l1_source: Some(block_info_1),
            cross_safe: Some(BlockInfo { timestamp: 100, ..Default::default() }),
            finalized: Some(BlockInfo { timestamp: 50, ..Default::default() }),
            ..Default::default()
        };
        let block_info_2 = BlockInfo { number: 43, ..Default::default() };
        let super_head_2 = SuperHead {
            l1_source: Some(block_info_2),
            cross_safe: Some(BlockInfo { timestamp: 110, ..Default::default() }),
            finalized: Some(BlockInfo { timestamp: 60, ..Default::default() }),
            ..Default::default()
        };
        super_head_map.insert(chain_id_1, super_head_1);
        super_head_map.insert(chain_id_2, super_head_2);

        let mut mock_service = MockSupervisorService::new();
        mock_service
            .expect_chain_ids()
            .returning(move || Box::new(vec![chain_id_1, chain_id_2].into_iter()));
        mock_service.expect_super_head().times(2).returning(move |chain_id| {
            if chain_id == chain_id_1 { Ok(super_head_1) } else { Ok(super_head_2) }
        });

        let rpc = SupervisorRpc::new(Arc::new(mock_service));
        let result = rpc.sync_status().await;
        assert!(result.is_ok());
        let status = result.unwrap();
        assert_eq!(status.min_synced_l1.number, 42);
        assert_eq!(status.cross_safe_timestamp, 100);
        assert_eq!(status.finalized_timestamp, 50);
        assert_eq!(status.chains.len(), 2);
    }
}
