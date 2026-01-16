use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, Bytes, ChainId, keccak256};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_interop::{
    DependencySet, ExecutingDescriptor, InteropValidator, OutputRootWithChain, SUPER_ROOT_VERSION,
    SafetyLevel, SuperRoot,
};
use kona_protocol::BlockInfo;
use kona_supervisor_rpc::{ChainRootInfoRpc, SuperRootOutputRpc};
use kona_supervisor_storage::{
    ChainDb, ChainDbFactory, DerivationStorageReader, FinalizedL1Storage, HeadRefStorageReader,
    LogStorageReader,
};
use kona_supervisor_types::{SuperHead, parse_access_list};
use op_alloy_rpc_types::SuperchainDAError;
use std::{collections::HashMap, sync::Arc};
use tokio::sync::RwLock;
use tracing::{error, warn};

use crate::{
    SpecError, SupervisorError,
    config::Config,
    syncnode::{BlockProvider, ManagedNodeDataProvider},
};

/// Defines the service for the Supervisor core logic.
#[async_trait]
#[auto_impl::auto_impl(&, &mut, Arc, Box)]
pub trait SupervisorService: Debug + Send + Sync {
    /// Returns list of supervised [`ChainId`]s.
    fn chain_ids(&self) -> impl Iterator<Item = ChainId>;

    /// Returns mapping of supervised [`ChainId`]s to their [`ChainDependency`] config.
    ///
    /// [`ChainDependency`]: kona_interop::ChainDependency
    fn dependency_set(&self) -> &DependencySet;

    /// Returns [`SuperHead`] of given supervised chain.
    fn super_head(&self, chain: ChainId) -> Result<SuperHead, SupervisorError>;

    /// Returns latest block derived from given L1 block, for given chain.
    fn latest_block_from(
        &self,
        l1_block: BlockNumHash,
        chain: ChainId,
    ) -> Result<BlockInfo, SupervisorError>;

    /// Returns the L1 source block that the given L2 derived block was based on, for the specified
    /// chain.
    fn derived_to_source_block(
        &self,
        chain: ChainId,
        derived: BlockNumHash,
    ) -> Result<BlockInfo, SupervisorError>;

    /// Returns [`LocalUnsafe`] block for the given chain.
    ///
    /// [`LocalUnsafe`]: SafetyLevel::LocalUnsafe
    fn local_unsafe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;

    /// Returns [`LocalSafe`] block for the given chain.
    ///
    /// [`LocalSafe`]: SafetyLevel::LocalSafe
    fn local_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;

    /// Returns [`CrossSafe`] block for the given chain.
    ///
    /// [`CrossSafe`]: SafetyLevel::CrossSafe
    fn cross_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;

    /// Returns [`Finalized`] block for the given chain.
    ///
    /// [`Finalized`]: SafetyLevel::Finalized
    fn finalized(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;

    /// Returns the finalized L1 block that the supervisor is synced to.
    fn finalized_l1(&self) -> Result<BlockInfo, SupervisorError>;

    /// Returns the [`SuperRootOutput`] at a specified timestamp, which represents the global
    /// state across all monitored chains.
    ///
    /// [`SuperRootOutput`]: kona_interop::SuperRootOutput
    async fn super_root_at_timestamp(
        &self,
        timestamp: u64,
    ) -> Result<SuperRootOutputRpc, SupervisorError>;

    /// Verifies if an access-list references only valid messages
    fn check_access_list(
        &self,
        inbox_entries: Vec<B256>,
        min_safety: SafetyLevel,
        executing_descriptor: ExecutingDescriptor,
    ) -> Result<(), SupervisorError>;
}

/// The core Supervisor component responsible for monitoring and coordinating chain states.
#[derive(Debug)]
pub struct Supervisor<M> {
    config: Arc<Config>,
    database_factory: Arc<ChainDbFactory>,

    // As of now supervisor only supports a single managed node per chain.
    // This is a limitation of the current implementation, but it will be extended in the future.
    managed_nodes: RwLock<HashMap<ChainId, Arc<M>>>,
}

impl<M> Supervisor<M>
where
    M: ManagedNodeDataProvider + BlockProvider + Send + Sync + Debug,
{
    /// Creates a new [`Supervisor`] instance.
    #[allow(clippy::new_without_default, clippy::missing_const_for_fn)]
    pub fn new(config: Arc<Config>, database_factory: Arc<ChainDbFactory>) -> Self {
        Self { config, database_factory, managed_nodes: RwLock::new(HashMap::new()) }
    }

    /// Adds a new managed node to the [`Supervisor`].
    pub async fn add_managed_node(
        &self,
        chain_id: ChainId,
        managed_node: Arc<M>,
    ) -> Result<(), SupervisorError> {
        // todo: instead of passing the chain ID, we should get it from the managed node
        if !self.config.dependency_set.dependencies.contains_key(&chain_id) {
            warn!(target: "supervisor::service", %chain_id, "Unsupported chain ID");
            return Err(SupervisorError::UnsupportedChainId);
        }

        let mut managed_nodes = self.managed_nodes.write().await;
        if managed_nodes.contains_key(&chain_id) {
            warn!(target: "supervisor::service", %chain_id, "Managed node already exists for chain");
            return Ok(());
        }

        managed_nodes.insert(chain_id, managed_node.clone());
        Ok(())
    }

    fn verify_safety_level(
        &self,
        chain_id: ChainId,
        block: &BlockInfo,
        safety: SafetyLevel,
    ) -> Result<(), SupervisorError> {
        let head_ref = self.database_factory.get_db(chain_id)?.get_safety_head_ref(safety)?;

        if head_ref.number < block.number {
            return Err(SpecError::SuperchainDAError(SuperchainDAError::ConflictingData).into());
        }

        Ok(())
    }

    fn get_db(&self, chain: ChainId) -> Result<Arc<ChainDb>, SupervisorError> {
        self.database_factory.get_db(chain).map_err(|err| {
            error!(target: "supervisor::service", %chain, %err, "Failed to get database for chain");
            SpecError::from(err).into()
        })
    }
}

#[async_trait]
impl<M> SupervisorService for Supervisor<M>
where
    M: ManagedNodeDataProvider + BlockProvider + Send + Sync + Debug,
{
    fn chain_ids(&self) -> impl Iterator<Item = ChainId> {
        self.config.dependency_set.dependencies.keys().copied()
    }

    fn dependency_set(&self) -> &DependencySet {
        &self.config.dependency_set
    }

    fn super_head(&self, chain: ChainId) -> Result<SuperHead, SupervisorError> {
        Ok(self.get_db(chain)?.get_super_head().map_err(|err| {
            error!(target: "supervisor::service", %chain, %err, "Failed to get super head for chain");
            SpecError::from(err)
        })?)
    }

    fn latest_block_from(
        &self,
        l1_block: BlockNumHash,
        chain: ChainId,
    ) -> Result<BlockInfo, SupervisorError> {
        Ok(self
            .get_db(chain)?
            .latest_derived_block_at_source(l1_block)
            .map_err(|err| {
                error!(target: "supervisor::service", %chain, %err, "Failed to get latest derived block at source for chain");
                SpecError::from(err)
            })?
        )
    }

    fn derived_to_source_block(
        &self,
        chain: ChainId,
        derived: BlockNumHash,
    ) -> Result<BlockInfo, SupervisorError> {
        Ok(self.get_db(chain)?.derived_to_source(derived).map_err(|err| {
            error!(target: "supervisor::service", %chain, %err, "Failed to get derived to source block for chain");
            SpecError::from(err)
        })?)
    }

    fn local_unsafe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError> {
        Ok(self.get_db(chain)?.get_safety_head_ref(SafetyLevel::LocalUnsafe).map_err(|err| {
            error!(target: "supervisor::service", %chain, %err, "Failed to get local unsafe head ref for chain");
            SpecError::from(err)
        })?)
    }

    fn local_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError> {
        Ok(self.get_db(chain)?.get_safety_head_ref(SafetyLevel::LocalSafe).map_err(|err| {
            error!(target: "supervisor::service", %chain, %err, "Failed to get local safe head ref for chain");
            SpecError::from(err)
        })?)
    }

    fn cross_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError> {
        Ok(self.get_db(chain)?.get_safety_head_ref(SafetyLevel::CrossSafe).map_err(|err| {
            error!(target: "supervisor::service", %chain, %err, "Failed to get cross safe head ref for chain");
            SpecError::from(err)
        })?)
    }

    fn finalized(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError> {
        Ok(self.get_db(chain)?.get_safety_head_ref(SafetyLevel::Finalized).map_err(|err| {
            error!(target: "supervisor::service", %chain, %err, "Failed to get finalized head ref for chain");
            SpecError::from(err)
        })?)
    }

    fn finalized_l1(&self) -> Result<BlockInfo, SupervisorError> {
        Ok(self.database_factory.get_finalized_l1().map_err(|err| {
            error!(target: "supervisor::service", %err, "Failed to get finalized L1");
            SpecError::from(err)
        })?)
    }

    async fn super_root_at_timestamp(
        &self,
        timestamp: u64,
    ) -> Result<SuperRootOutputRpc, SupervisorError> {
        let mut chain_ids = self.config.dependency_set.dependencies.keys().collect::<Vec<_>>();
        // Sorting chain ids for deterministic super root hash
        chain_ids.sort();

        let mut chain_infos = Vec::<ChainRootInfoRpc>::with_capacity(chain_ids.len());
        let mut super_root_chains = Vec::<OutputRootWithChain>::with_capacity(chain_ids.len());
        let mut cross_safe_source = BlockNumHash::default();

        for id in chain_ids {
            let managed_node = {
                let guard = self.managed_nodes.read().await;
                match guard.get(id) {
                    Some(m) => m.clone(),
                    None => {
                        error!(target: "supervisor::service", chain_id = %id, "Managed node not found for chain");
                        return Err(SupervisorError::ManagedNodeMissing(*id));
                    }
                }
            };
            let output_v0 = managed_node.output_v0_at_timestamp(timestamp).await?;
            let output_v0_string = serde_json::to_string(&output_v0)
                .inspect_err(|err| {
                    error!(target: "supervisor::service", chain_id = %id, %err, "Failed to serialize output_v0 for chain");
                })?;
            let canonical_root = keccak256(output_v0_string.as_bytes());

            let pending_output_v0 = managed_node.pending_output_v0_at_timestamp(timestamp).await?;
            let pending_output_v0_string = serde_json::to_string(&pending_output_v0)
                .inspect_err(|err| {
                    error!(target: "supervisor::service", chain_id = %id, %err, "Failed to serialize pending_output_v0 for chain");
                })?;
            let pending_output_v0_bytes =
                Bytes::copy_from_slice(pending_output_v0_string.as_bytes());

            chain_infos.push(ChainRootInfoRpc {
                chain_id: *id,
                canonical: canonical_root,
                pending: pending_output_v0_bytes,
            });

            super_root_chains
                .push(OutputRootWithChain { chain_id: *id, output_root: canonical_root });

            let l2_block = managed_node.l2_block_ref_by_timestamp(timestamp).await?;
            let source = self
                .derived_to_source_block(*id, l2_block.id())
                .inspect_err(|err| {
                    error!(target: "supervisor::service", %id, %err, "Failed to get derived to source block for chain");
                })?;

            if cross_safe_source.number == 0 || cross_safe_source.number < source.number {
                cross_safe_source = source.id();
            }
        }

        let super_root = SuperRoot { timestamp, output_roots: super_root_chains };
        let super_root_hash = super_root.hash();

        Ok(SuperRootOutputRpc {
            cross_safe_derived_from: cross_safe_source,
            timestamp,
            super_root: super_root_hash,
            chains: chain_infos,
            version: SUPER_ROOT_VERSION,
        })
    }

    fn check_access_list(
        &self,
        inbox_entries: Vec<B256>,
        min_safety: SafetyLevel,
        executing_descriptor: ExecutingDescriptor,
    ) -> Result<(), SupervisorError> {
        let access_list = parse_access_list(inbox_entries)?;

        for access in &access_list {
            // Check all the invariants for each message
            // Ref: https://github.com/ethereum-optimism/specs/blob/main/specs/interop/derivation.md#invariants

            // TODO: support 32 bytes chain id and convert to u64 via dependency set to be usable
            // across services
            let initiating_chain_id = access.chain_id[24..32]
                .try_into()
                .map(u64::from_be_bytes)
                .map_err(|err| {
                    error!(target: "supervisor::service", %err, "Failed to parse initiating chain id from access list");
                    SupervisorError::ChainIdParseError()
                })?;

            let executing_chain_id = executing_descriptor.chain_id.unwrap_or(initiating_chain_id);

            // Message must be valid at the time of execution.
            self.config.validate_interop_timestamps(
                initiating_chain_id,
                access.timestamp,
                executing_chain_id,
                executing_descriptor.timestamp,
                executing_descriptor.timeout,
            ).map_err(|err| {
                warn!(target: "supervisor::service", %err, "Failed to validate interop timestamps");
                SpecError::SuperchainDAError(SuperchainDAError::ConflictingData)
            })?;

            // Verify the initiating message exists and valid for corresponding executing message.
            let db = self.get_db(initiating_chain_id)?;

            let block = db.get_block(access.block_number).map_err(|err| {
                warn!(target: "supervisor::service", %initiating_chain_id, %err, "Failed to get block for chain");
                SpecError::from(err)
            })?;
            if block.timestamp != access.timestamp {
                return Err(SupervisorError::from(SpecError::SuperchainDAError(
                    SuperchainDAError::ConflictingData,
                )))
            }

            let log = db.get_log(access.block_number, access.log_index).map_err(|err| {
                warn!(target: "supervisor::service", %initiating_chain_id, %err, "Failed to get log for chain");
                SpecError::from(err)
            })?;
            access.verify_checksum(&log.hash).map_err(|err| {
                warn!(target: "supervisor::service", %initiating_chain_id, %err, "Failed to verify checksum for access list");
                SpecError::SuperchainDAError(SuperchainDAError::ConflictingData)
            })?;

            // The message must be included in a block that is at least as safe as required
            // by the `min_safety` level
            if min_safety != SafetyLevel::LocalUnsafe {
                // The block is already unsafe as it is found in log db
                self.verify_safety_level(initiating_chain_id, &block, min_safety)?;
            }
        }

        Ok(())
    }
}
