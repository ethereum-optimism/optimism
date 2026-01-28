use crate::StorageError;
use alloy_eips::eip1898::BlockNumHash;
use alloy_primitives::ChainId;
use kona_interop::DerivedRefPair;
use kona_protocol::BlockInfo;
use kona_supervisor_types::{Log, SuperHead};
use op_alloy_consensus::interop::SafetyLevel;
use std::fmt::Debug;

/// Provides an interface for supervisor storage to manage source and derived blocks.
///
/// Defines methods to retrieve derived block information,
/// enabling the supervisor to track the derivation progress.
///
/// Implementations are expected to provide persistent and thread-safe access to block data.
pub trait DerivationStorageReader: Debug {
    /// Gets the source [`BlockInfo`] for a given derived block [`BlockNumHash`].
    ///
    /// NOTE: [`LocalUnsafe`] block is not pushed to L1 yet, hence it cannot be part of derivation
    /// storage.
    ///
    /// # Arguments
    /// * `derived_block_id` - The identifier (number and hash) of the derived (L2) block.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the source block information if it exists.
    /// * `Err(StorageError)` if there is an issue retrieving the source block.
    ///
    /// [`LocalUnsafe`]: SafetyLevel::LocalUnsafe
    fn derived_to_source(&self, derived_block_id: BlockNumHash) -> Result<BlockInfo, StorageError>;

    /// Gets the latest derived [`BlockInfo`] associated with the given source block
    /// [`BlockNumHash`].
    ///
    /// # Arguments
    /// * `source_block_id` - The identifier (number and hash) of the L1 source block.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the latest derived block information if it exists.
    /// * `Err(StorageError)` if there is an issue retrieving the derived block.
    fn latest_derived_block_at_source(
        &self,
        source_block_id: BlockNumHash,
    ) -> Result<BlockInfo, StorageError>;

    /// Gets the latest derivation state [`DerivedRefPair`] from the storage, which includes the
    /// latest source block and the latest derived block.
    ///
    /// # Returns
    ///
    /// * `Ok(DerivedRefPair)` containing the latest derived block pair if it exists.
    /// * `Err(StorageError)` if there is an issue retrieving the pair.
    fn latest_derivation_state(&self) -> Result<DerivedRefPair, StorageError>;

    /// Gets the source block for the given source block number.
    ///
    /// # Arguments
    /// * `source_block_number` - The number of the source block to retrieve.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the source block information if it exists.
    /// * `Err(StorageError)` if there is an issue retrieving the source block.
    fn get_source_block(&self, source_block_number: u64) -> Result<BlockInfo, StorageError>;

    /// Gets the interop activation [`BlockInfo`].
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the activation block information if it exists.
    /// * `Err(StorageError)` if there is an issue retrieving the activation block.
    fn get_activation_block(&self) -> Result<BlockInfo, StorageError>;
}

/// Provides an interface for supervisor storage to write source and derived blocks.
///
/// Defines methods to persist derived block information,
/// enabling the supervisor to track the derivation progress.
///
/// Implementations are expected to provide persistent and thread-safe access to block data.
pub trait DerivationStorageWriter: Debug {
    /// Initializes the derivation storage with a given [`DerivedRefPair`].
    /// This method is typically called once to set up the storage with the initial pair.
    ///
    /// # Arguments
    /// * `incoming_pair` - The derived block pair to initialize the storage with.
    ///
    /// # Returns
    /// * `Ok(())` if the storage was successfully initialized.
    /// * `Err(StorageError)` if there is an issue initializing the storage.
    fn initialise_derivation_storage(
        &self,
        incoming_pair: DerivedRefPair,
    ) -> Result<(), StorageError>;

    /// Saves a [`DerivedRefPair`] to the storage.
    ///
    /// This method is **append-only**: it does not overwrite existing pairs.
    /// - If a pair with the same block number already exists and is identical to the incoming pair,
    ///   the request is silently ignored (idempotent).
    /// - If a pair with the same block number exists but differs from the incoming pair, an error
    ///   is returned to indicate a data inconsistency.
    /// - If the pair is new and consistent, it is appended to the storage.
    ///
    /// Ensures that the latest stored pair is the parent of the incoming pair before saving.
    ///
    /// # Arguments
    /// * `incoming_pair` - The derived block pair to save.
    ///
    /// # Returns
    /// * `Ok(())` if the pair was successfully saved.
    /// * `Err(StorageError)` if there is an issue saving the pair.
    fn save_derived_block(&self, incoming_pair: DerivedRefPair) -> Result<(), StorageError>;

    /// Saves the latest incoming source [`BlockInfo`] to the storage.
    ///
    /// This method is **append-only**: it does not overwrite existing source blocks.
    /// - If a source block with the same number already exists and is identical to the incoming
    ///   block, the request is silently ignored (idempotent).
    /// - If a source block with the same number exists but differs from the incoming block, an
    ///   error is returned to indicate a data inconsistency.
    /// - If the block is new and consistent, it is appended to the storage.
    ///
    /// Ensures that the latest stored source block is the parent of the incoming block before
    /// saving.
    ///
    /// # Arguments
    /// * `source` - The source block to save.
    ///
    /// # Returns
    /// * `Ok(())` if the source block was successfully saved.
    /// * `Err(StorageError)` if there is an issue saving the source block.
    fn save_source_block(&self, source: BlockInfo) -> Result<(), StorageError>;
}

/// Combines both reading and writing capabilities for derivation storage.
///
/// Any type that implements both [`DerivationStorageReader`] and [`DerivationStorageWriter`]
/// automatically implements this trait.
pub trait DerivationStorage: DerivationStorageReader + DerivationStorageWriter {}

impl<T: DerivationStorageReader + DerivationStorageWriter> DerivationStorage for T {}

/// Provides an interface for retrieving logs associated with blocks.
///
/// This trait defines methods to retrieve the latest block,
/// find a block by a specific log, and retrieve logs for a given block number.
///
/// Implementations are expected to provide persistent and thread-safe access to block logs.
pub trait LogStorageReader: Debug {
    /// Retrieves the latest [`BlockInfo`] from the storage.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the latest block information.
    /// * `Err(StorageError)` if there is an issue retrieving the latest block.
    fn get_latest_block(&self) -> Result<BlockInfo, StorageError>;

    /// Retrieves the [`BlockInfo`] from the storage for a given block number
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the block information.
    /// * `Err(StorageError)` if there is an issue retrieving the block.
    fn get_block(&self, block_number: u64) -> Result<BlockInfo, StorageError>;

    /// Finds a [`Log`] by block_number and log_index
    ///
    /// # Arguments
    /// * `block_number` - The block number to search for the log.
    /// * `log_index` - The index of the log within the block.
    ///
    /// # Returns
    /// * `Ok(Log)` containing the [`Log`] object.
    /// * `Err(StorageError)` if there is an issue retrieving the log or if the log is not found.
    fn get_log(&self, block_number: u64, log_index: u32) -> Result<Log, StorageError>;

    /// Retrieves all [`Log`]s associated with a specific block number.
    ///
    /// # Arguments
    /// * `block_number` - The block number for which to retrieve logs.
    ///
    /// # Returns
    /// * `Ok(Vec<Log>)` containing the logs associated with the block number.
    /// * `Err(StorageError)` if there is an issue retrieving the logs or if no logs are found.
    fn get_logs(&self, block_number: u64) -> Result<Vec<Log>, StorageError>;
}

/// Provides an interface for storing blocks and  logs associated with blocks.
///
/// Implementations are expected to provide persistent and thread-safe access to block logs.
pub trait LogStorageWriter: Send + Sync + Debug {
    /// Initializes the log storage with a given [`BlockInfo`].
    /// This method is typically called once to set up the storage with the initial block.
    ///
    /// # Arguments
    /// * `block` - The [`BlockInfo`] to initialize the storage with.
    ///
    /// # Returns
    /// * `Ok(())` if the storage was successfully initialized.
    /// * `Err(StorageError)` if there is an issue initializing the storage.
    fn initialise_log_storage(&self, block: BlockInfo) -> Result<(), StorageError>;

    /// Stores [`BlockInfo`] and [`Log`]s in the storage.
    /// This method is append-only and does not overwrite existing logs.
    /// Ensures that the latest stored block is the parent of the incoming block before saving.
    ///
    /// # Arguments
    /// * `block` - [`BlockInfo`] to associate with the logs.
    /// * `logs` - The [`Log`] events associated with the block.
    ///
    /// # Returns
    /// * `Ok(())` if the logs were successfully stored.
    /// * `Err(StorageError)` if there is an issue storing the logs.
    fn store_block_logs(&self, block: &BlockInfo, logs: Vec<Log>) -> Result<(), StorageError>;
}

/// Combines both reading and writing capabilities for log storage.
///
/// Any type that implements both [`LogStorageReader`] and [`LogStorageWriter`]
/// automatically implements this trait.
pub trait LogStorage: LogStorageReader + LogStorageWriter {}

impl<T: LogStorageReader + LogStorageWriter> LogStorage for T {}

/// Provides an interface for retrieving head references.
///
/// This trait defines methods to manage safety head references for different safety levels.
/// Each safety level maintains a reference to a block.
///
/// Implementations are expected to provide persistent and thread-safe access to safety head
/// references.
pub trait HeadRefStorageReader: Debug {
    /// Retrieves the current [`BlockInfo`] for a given [`SafetyLevel`].
    ///
    /// # Arguments
    /// * `safety_level` - The safety level for which to retrieve the head reference.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the current safety head reference.
    /// * `Err(StorageError)` if there is an issue retrieving the reference.
    fn get_safety_head_ref(&self, safety_level: SafetyLevel) -> Result<BlockInfo, StorageError>;

    /// Retrieves the super head reference from the storage.
    ///
    /// # Returns
    /// * `Ok(SuperHead)` containing the super head reference.
    /// * `Err(StorageError)` if there is an issue retrieving the super head reference.
    fn get_super_head(&self) -> Result<SuperHead, StorageError>;
}

/// Provides an interface for storing head references.
///
/// This trait defines methods to manage safety head references for different safety levels.
/// Each safety level maintains a reference to a block.
///
/// Implementations are expected to provide persistent and thread-safe access to safety head
/// references.
pub trait HeadRefStorageWriter: Debug {
    /// Updates the finalized head reference using a finalized source(l1) block.
    ///
    /// # Arguments
    /// * `source_block` - The [`BlockInfo`] of the source block to use for the update.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the updated finalized derived(l2) block information.
    /// * `Err(StorageError)` if there is an issue updating the finalized head reference.
    fn update_finalized_using_source(
        &self,
        finalized_source_block: BlockInfo,
    ) -> Result<BlockInfo, StorageError>;

    /// Updates the current [`CrossUnsafe`](SafetyLevel::CrossUnsafe) head reference in storage.
    ///
    /// Ensures the provided block still exists in log storage and was not removed due to a re-org.
    /// If the stored block's hash does not match the provided block, the update is aborted.
    /// # Arguments
    /// * `block` - The [`BlockInfo`] to set as the head reference
    ///
    /// # Returns
    /// * `Ok(())` if the reference was successfully updated.
    /// * `Err(StorageError)` if there is an issue updating the reference.
    fn update_current_cross_unsafe(&self, block: &BlockInfo) -> Result<(), StorageError>;

    /// Updates the current [`CrossSafe`](SafetyLevel::CrossSafe) head reference in storage and
    /// returns the corresponding derived pair.
    ///
    /// Ensures the provided block still exists in derivation storage and was not removed due to a
    /// re-org. # Arguments
    /// * `block` - The [`BlockInfo`] to set as the head reference
    ///
    /// # Returns
    /// * `Ok(DerivedRefPair)` if the reference was successfully updated.
    /// * `Err(StorageError)` if there is an issue updating the reference.
    fn update_current_cross_safe(&self, block: &BlockInfo) -> Result<DerivedRefPair, StorageError>;
}

/// Combines both reading and writing capabilities for safety head ref storage.
///
/// Any type that implements both [`HeadRefStorageReader`] and [`HeadRefStorageWriter`]
/// automatically implements this trait.
pub trait HeadRefStorage: HeadRefStorageReader + HeadRefStorageWriter {}

impl<T: HeadRefStorageReader + HeadRefStorageWriter> HeadRefStorage for T {}

/// Provides an interface for managing the finalized L1 block reference in the storage.
///
/// This trait defines methods to update and retrieve the finalized L1 block reference.
pub trait FinalizedL1Storage {
    /// Updates the finalized L1 block reference in the storage.
    ///
    /// # Arguments
    /// * `block` - The new [`BlockInfo`] to set as the finalized L1 block reference.
    ///
    /// # Returns
    /// * `Ok(())` if the reference was successfully updated.
    /// * `Err(StorageError)` if there is an issue updating the reference.
    fn update_finalized_l1(&self, block: BlockInfo) -> Result<(), StorageError>;

    /// Retrieves the finalized L1 block reference from the storage.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the finalized L1 block reference.
    /// * `Err(StorageError)` if there is an issue retrieving the reference.
    fn get_finalized_l1(&self) -> Result<BlockInfo, StorageError>;
}

/// Provides an interface for retrieving block and safety information across multiple chains.
///
/// This trait defines methods required by the cross-chain safety checker to access
/// block metadata, logs, and safe head references for various chains.
pub trait CrossChainSafetyProvider {
    /// Retrieves the [`BlockInfo`] for a given block number on the specified chain.
    ///
    /// # Arguments
    /// * `chain_id` - The [`ChainId`] of the target chain.
    /// * `block_number` - The number of the block to retrieve.
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` containing the block metadata if available.
    /// * `Err(StorageError)` if there is an issue fetching the block.
    fn get_block(&self, chain_id: ChainId, block_number: u64) -> Result<BlockInfo, StorageError>;

    /// Retrieves a [`Log`] by block_number and log_index
    ///
    /// # Arguments
    /// * `chain_id` - The [`ChainId`] of the target chain.
    /// * `block_number` - The block number to search for the log.
    /// * `log_index` - The index of the log within the block.
    ///
    /// # Returns
    /// * `Ok(Log)` containing the [`Log`] object.
    /// * `Err(StorageError)` if there is an issue retrieving the log or if the log is not found.
    fn get_log(
        &self,
        chain_id: ChainId,
        block_number: u64,
        log_index: u32,
    ) -> Result<Log, StorageError>;

    /// Retrieves all logs associated with the specified block on the given chain.
    ///
    /// # Arguments
    /// * `chain_id` - The [`ChainId`] of the target chain.
    /// * `block_number` - The number of the block whose logs should be retrieved.
    ///
    /// # Returns
    /// * `Ok(Vec<Log>)` containing all logs for the block.
    /// * `Err(StorageError)` if there is an issue fetching the logs.
    fn get_block_logs(
        &self,
        chain_id: ChainId,
        block_number: u64,
    ) -> Result<Vec<Log>, StorageError>;

    /// Retrieves the latest known safe head reference for a given chain at the specified safety
    /// level.
    ///
    /// # Arguments
    /// * `chain_id` - The [`ChainId`] of the target chain.
    /// * `level` - The desired [`SafetyLevel`] (e.g., `CrossSafe`, `LocalSafe`).
    ///
    /// # Returns
    /// * `Ok(BlockInfo)` representing the safe head block at the requested safety level.
    /// * `Err(StorageError)` if the safe head cannot be retrieved.
    fn get_safety_head_ref(
        &self,
        chain_id: ChainId,
        level: SafetyLevel,
    ) -> Result<BlockInfo, StorageError>;

    /// Updates the current [`CrossUnsafe`](SafetyLevel::CrossUnsafe) head reference in storage.
    ///
    /// Ensures the provided block still exists in log storage and was not removed due to a re-org.
    /// If the stored block's hash does not match the provided block, the update is aborted.
    /// # Arguments
    /// * `chain_id` - The [`ChainId`] of the target chain.
    /// * `block` - The [`BlockInfo`] to set as the head reference
    ///
    /// # Returns
    /// * `Ok(())` if the reference was successfully updated.
    /// * `Err(StorageError)` if there is an issue updating the reference.
    fn update_current_cross_unsafe(
        &self,
        chain_id: ChainId,
        block: &BlockInfo,
    ) -> Result<(), StorageError>;

    /// Updates the current [`CrossSafe`](SafetyLevel::CrossSafe) head reference in storage and
    /// returns the corresponding derived pair.
    ///
    /// Ensures the provided block still exists in derivation storage and was not removed due to a
    /// re-org. # Arguments
    /// * `chain_id` - The [`ChainId`] of the target chain.
    /// * `block` - The [`BlockInfo`] to set as the head reference
    ///
    /// # Returns
    /// * `Ok(DerivedRefPair)` if the reference was successfully updated.
    /// * `Err(StorageError)` if there is an issue updating the reference.
    fn update_current_cross_safe(
        &self,
        chain_id: ChainId,
        block: &BlockInfo,
    ) -> Result<DerivedRefPair, StorageError>;
}

/// Trait for rewinding supervisor-related state in the database.
///
/// This trait provides an interface to revert persisted log data, derivation records,
/// and safety head references from the latest block back to a specified block number (inclusive).
/// It is typically used during chain reorganizations or when invalid blocks are detected and need
/// to be rolled back.
pub trait StorageRewinder {
    /// Rewinds the log storage from the latest block down to the specified block (inclusive).
    /// This method ensures that log storage is never rewound to(since it's inclusive) and beyond
    /// the local safe head. If the target block is beyond the local safe head, an error is
    /// returned. Use [`StorageRewinder::rewind`] to rewind to and beyond the local safe head.
    ///
    /// # Arguments
    /// * `to` - The block id to rewind to.
    ///
    /// # Errors
    /// Returns a [`StorageError`] if any database operation fails during the rewind.
    fn rewind_log_storage(&self, to: &BlockNumHash) -> Result<(), StorageError>;

    /// Rewinds all supervisor-managed state (log storage, derivation, and safety head refs)
    /// from the latest block back to the given block (inclusive).
    ///
    /// This method performs a coordinated rewind across all components, ensuring consistency
    /// of supervisor state after chain reorganizations or rollback of invalid blocks.
    ///
    /// # Arguments
    /// * `to` - The target block id to rewind to. Rewind is performed from the latest block down to
    ///   this block.
    ///
    /// # Errors
    /// Returns a [`StorageError`] if any part of the rewind process fails.
    fn rewind(&self, to: &BlockNumHash) -> Result<(), StorageError>;

    /// Rewinds the storage to a specific source block (inclusive), ensuring that all derived blocks
    /// and logs associated with that source blocks are also reverted.
    ///
    /// # Arguments
    /// * `to` - The source block [`BlockNumHash`] to rewind to.
    ///
    /// # Returns
    /// * [`BlockInfo`] of the derived block that was rewound to, or `None` if no derived blocks
    ///   were found.
    /// * `Err(StorageError)` if there is an issue during the rewind operation.
    fn rewind_to_source(&self, to: &BlockNumHash) -> Result<Option<BlockInfo>, StorageError>;
}

/// Combines the reader traits for the database.
///
/// Any type that implements [`DerivationStorageReader`], [`HeadRefStorageReader`], and
/// [`LogStorageReader`] automatically implements this trait.
pub trait DbReader: DerivationStorageReader + HeadRefStorageReader + LogStorageReader {}

impl<T: DerivationStorageReader + HeadRefStorageReader + LogStorageReader> DbReader for T {}
