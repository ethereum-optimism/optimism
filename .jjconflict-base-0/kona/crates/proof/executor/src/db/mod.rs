//! This module contains an implementation of an in-memory Trie DB for [`revm`], that allows for
//! incremental updates through fetching node preimages on the fly during execution.

use crate::errors::{TrieDBError, TrieDBResult};
use alloc::{string::ToString, vec::Vec};
use alloy_consensus::{EMPTY_ROOT_HASH, Header, Sealed};
use alloy_primitives::{Address, B256, U256, keccak256};
use alloy_rlp::{Decodable, Encodable};
use alloy_trie::TrieAccount;
use kona_mpt::{Nibbles, TrieHinter, TrieNode, TrieNodeError};
use revm::{
    Database,
    database::{BundleState, states::StorageSlot},
    primitives::{BLOCK_HASH_HISTORY, HashMap},
    state::{AccountInfo, Bytecode},
};

mod traits;
pub use traits::{NoopTrieDBProvider, TrieDBProvider};

/// A Trie DB that caches open state in-memory.
///
/// When accounts that don't already exist within the cached [`TrieNode`] are queried, the database
/// fetches the preimages of the trie nodes on the path to the account using the `PreimageFetcher`
/// (`F` generic). This allows for data to be fetched in a verifiable manner given an initial
/// trusted state root as it is needed during execution.
///
/// The [`TrieDB`] is intended to be wrapped by a [`State`], which is then used by [`revm`] to
/// capture state transitions during block execution.
///
/// **Behavior**:
/// - When an account is queried and the trie path has not already been opened by [Self::basic], we
///   fall through to the `PreimageFetcher` to fetch the preimages of the trie nodes on the path to
///   the account. After it has been fetched, the path will be cached until the next call to
///   [Self::state_root].
/// - When querying for the code hash of an account, the [`TrieDBProvider`] is consulted to fetch
///   the code hash of the account.
/// - When a [`BundleState`] changeset is committed to the parent [`State`] database, the changes
///   are first applied to the [`State`]'s cache, then the trie hash is recomputed with
///   [Self::state_root].
/// - When the block hash of a block number is needed via [Self::block_hash], the
///   `HeaderByHashFetcher` is consulted to walk back to the desired block number by revealing the
///   parent hash of block headers until the desired block number is reached, up to a maximum of
///   [BLOCK_HASH_HISTORY] blocks back relative to the current parent block hash.
///
/// **Example Construction**:
/// ```rust
/// use alloy_consensus::{Header, Sealable};
/// use alloy_evm::{EvmEnv, EvmFactory, block::BlockExecutorFactory};
/// use alloy_op_evm::{
///     OpBlockExecutionCtx, OpBlockExecutorFactory, OpEvmFactory, block::OpAlloyReceiptBuilder,
/// };
/// use alloy_op_hardforks::OpChainHardforks;
/// use alloy_primitives::{B256, Bytes};
/// use kona_executor::{NoopTrieDBProvider, TrieDB};
/// use kona_mpt::NoopTrieHinter;
/// use revm::database::{State, states::bundle_state::BundleRetention};
///
/// let mock_parent_block_header = Header::default();
/// let trie_db =
///     TrieDB::new(mock_parent_block_header.seal_slow(), NoopTrieDBProvider, NoopTrieHinter);
/// let executor_factory = OpBlockExecutorFactory::new(
///     OpAlloyReceiptBuilder::default(),
///     OpChainHardforks::op_mainnet(),
///     OpEvmFactory::default(),
/// );
/// let mut state = State::builder().with_database(trie_db).with_bundle_update().build();
/// let evm = executor_factory.evm_factory().create_evm(&mut state, EvmEnv::default());
/// let executor = executor_factory.create_executor(evm, OpBlockExecutionCtx::default());
///
/// // Execute your block's transactions...
/// drop(executor);
///
/// state.merge_transitions(BundleRetention::Reverts);
/// let bundle = state.take_bundle();
/// let state_root = state.database.state_root(&bundle).expect("Failed to compute state root");
/// ```
///
/// [`State`]: revm::database::State
#[derive(Debug, Clone)]
pub struct TrieDB<F, H>
where
    F: TrieDBProvider,
    H: TrieHinter,
{
    /// The [`TrieNode`] representation of the root node.
    root_node: TrieNode,
    /// Storage roots of accounts within the trie.
    storage_roots: HashMap<Address, TrieNode>,
    /// The parent block hash of the current block.
    parent_block_header: Sealed<Header>,
    /// The [`TrieDBProvider`]
    pub fetcher: F,
    /// The [`TrieHinter`]
    pub hinter: H,
}

impl<F, H> TrieDB<F, H>
where
    F: TrieDBProvider,
    H: TrieHinter,
{
    /// Creates a new [TrieDB] with the given root node.
    pub fn new(parent_block_header: Sealed<Header>, fetcher: F, hinter: H) -> Self {
        Self {
            root_node: TrieNode::new_blinded(parent_block_header.state_root),
            storage_roots: Default::default(),
            parent_block_header,
            fetcher,
            hinter,
        }
    }

    /// Consumes `Self` and takes the current state root of the trie DB.
    pub fn take_root_node(self) -> TrieNode {
        self.root_node
    }

    /// Returns a shared reference to the root [TrieNode] of the trie DB.
    pub const fn root(&self) -> &TrieNode {
        &self.root_node
    }

    /// Returns the mapping of [Address]es to storage roots.
    pub const fn storage_roots(&self) -> &HashMap<Address, TrieNode> {
        &self.storage_roots
    }

    /// Returns a reference to the current parent block header of the trie DB.
    pub const fn parent_block_header(&self) -> &Sealed<Header> {
        &self.parent_block_header
    }

    /// Sets the parent block header of the trie DB. Should be called after a block has been
    /// executed and the Header has been created.
    ///
    /// ## Takes
    /// - `parent_block_header`: The parent block header of the current block.
    pub fn set_parent_block_header(&mut self, parent_block_header: Sealed<Header>) {
        self.parent_block_header = parent_block_header;
    }

    /// Applies a [BundleState] changeset to the [TrieNode] and recomputes the state root hash.
    ///
    /// ## Takes
    /// - `bundle`: The [BundleState] changeset to apply to the trie DB.
    ///
    /// ## Returns
    /// - `Ok(B256)`: The new state root hash of the trie DB.
    /// - `Err(_)`: If the state root hash could not be computed.
    pub fn state_root(&mut self, bundle: &BundleState) -> TrieDBResult<B256> {
        debug!(target: "client_executor", "Recomputing state root");

        // Update the accounts in the trie with the changeset.
        self.update_accounts(bundle)?;

        // Recompute the root hash of the trie.
        let root = self.root_node.blind();

        debug!(
            target: "client_executor",
            "Recomputed state root: {root}",
        );

        // Extract the new state root from the root node.
        Ok(root)
    }

    /// Fetches the [TrieAccount] of an account from the trie DB.
    ///
    /// ## Takes
    /// - `address`: The address of the account.
    ///
    /// ## Returns
    /// - `Ok(Some(TrieAccount))`: The [TrieAccount] of the account.
    /// - `Ok(None)`: If the account does not exist in the trie.
    /// - `Err(_)`: If the account could not be fetched.
    pub fn get_trie_account(
        &mut self,
        address: &Address,
        block_number: u64,
    ) -> TrieDBResult<Option<TrieAccount>> {
        // Send a hint to the host to fetch the account proof.
        self.hinter
            .hint_account_proof(*address, block_number)
            .map_err(|e| TrieDBError::Provider(e.to_string()))?;

        // Fetch the account from the trie.
        let hashed_address_nibbles = Nibbles::unpack(keccak256(address.as_slice()));
        let Some(trie_account_rlp) = self.root_node.open(&hashed_address_nibbles, &self.fetcher)?
        else {
            return Ok(None);
        };

        // Decode the trie account from the RLP bytes.
        TrieAccount::decode(&mut trie_account_rlp.as_ref())
            .map_err(TrieNodeError::RLPError)
            .map_err(Into::into)
            .map(Some)
    }

    /// Modifies the accounts in the storage trie with the given [BundleState] changeset.
    ///
    /// ## Takes
    /// - `bundle`: The [BundleState] changeset to apply to the trie DB.
    ///
    /// ## Returns
    /// - `Ok(())` if the accounts were successfully updated.
    /// - `Err(_)` if the accounts could not be updated.
    fn update_accounts(&mut self, bundle: &BundleState) -> TrieDBResult<()> {
        // Sort the storage keys prior to applying the changeset, to ensure that the order of
        // application is deterministic between runs.
        let mut sorted_state =
            bundle.state().iter().map(|(k, v)| (k, keccak256(*k), v)).collect::<Vec<_>>();
        sorted_state.sort_by_key(|(_, hashed_addr, _)| *hashed_addr);

        for (address, hashed_address, bundle_account) in sorted_state {
            if bundle_account.status.is_not_modified() {
                continue;
            }

            // Compute the path to the account in the trie.
            let account_path = Nibbles::unpack(hashed_address.as_slice());

            // If the account was destroyed, delete it from the trie.
            if bundle_account.was_destroyed() {
                self.root_node.delete(&account_path, &self.fetcher, &self.hinter)?;
                self.storage_roots.remove(address);
                continue;
            }

            let account_info =
                bundle_account.account_info().ok_or(TrieDBError::MissingAccountInfo)?;

            let mut trie_account = TrieAccount {
                balance: account_info.balance,
                nonce: account_info.nonce,
                code_hash: account_info.code_hash,
                ..Default::default()
            };

            // Update the account's storage root
            let acc_storage_root = self
                .storage_roots
                .entry(*address)
                .or_insert_with(|| TrieNode::new_blinded(EMPTY_ROOT_HASH));

            // Sort the hashed storage keys prior to applying the changeset, to ensure that the
            // order of application is deterministic between runs.
            let mut sorted_storage = bundle_account
                .storage
                .iter()
                .map(|(k, v)| (keccak256(k.to_be_bytes::<32>()), v))
                .collect::<Vec<_>>();
            sorted_storage.sort_by_key(|(slot, _)| *slot);

            sorted_storage.into_iter().try_for_each(|(hashed_key, value)| {
                Self::change_storage(
                    acc_storage_root,
                    hashed_key,
                    value,
                    &self.fetcher,
                    &self.hinter,
                )
            })?;

            // Recompute the account storage root.
            let root = acc_storage_root.blind();
            trie_account.storage_root = root;

            // RLP encode the trie account for insertion.
            let mut account_buf = Vec::with_capacity(trie_account.length());
            trie_account.encode(&mut account_buf);

            // Insert or update the account in the trie.
            self.root_node.insert(&account_path, account_buf.into(), &self.fetcher)?;
        }

        Ok(())
    }

    /// Modifies a storage slot of an account in the Merkle Patricia Trie.
    ///
    /// ## Takes
    /// - `storage_root`: The storage root of the account.
    /// - `hashed_key`: The hashed storage slot key.
    /// - `value`: The new value of the storage slot.
    /// - `fetcher`: The trie node fetcher.
    /// - `hinter`: The trie hinter.
    ///
    /// ## Returns
    /// - `Ok(())` if the storage slot was successfully modified.
    /// - `Err(_)` if the storage slot could not be modified.
    fn change_storage(
        storage_root: &mut TrieNode,
        hashed_key: B256,
        value: &StorageSlot,
        fetcher: &F,
        hinter: &H,
    ) -> TrieDBResult<()> {
        if !value.is_changed() {
            return Ok(());
        }

        // RLP encode the storage slot value.
        let mut rlp_buf = Vec::with_capacity(value.present_value.length());
        value.present_value.encode(&mut rlp_buf);

        // Insert or update the storage slot in the trie.
        let hashed_slot_key = Nibbles::unpack(hashed_key.as_slice());
        if value.present_value.is_zero() {
            // If the storage slot is being set to zero, prune it from the trie.
            storage_root.delete(&hashed_slot_key, fetcher, hinter)?;
        } else {
            // Otherwise, update the storage slot.
            storage_root.insert(&hashed_slot_key, rlp_buf.into(), fetcher)?;
        }

        Ok(())
    }
}

impl<F, H> Database for TrieDB<F, H>
where
    F: TrieDBProvider,
    H: TrieHinter,
{
    type Error = TrieDBError;

    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        // Fetch the account from the trie.
        let Some(trie_account) =
            self.get_trie_account(&address, self.parent_block_header.number)?
        else {
            // If the account does not exist in the trie, return `Ok(None)`.
            return Ok(None);
        };

        // Insert the account's storage root into the cache.
        self.storage_roots.insert(address, TrieNode::new_blinded(trie_account.storage_root));

        // Return a partial DB account. The storage and code are not loaded out-right, and are
        // loaded optimistically in the `Database` + `DatabaseRef` trait implementations.
        Ok(Some(AccountInfo {
            balance: trie_account.balance,
            nonce: trie_account.nonce,
            code_hash: trie_account.code_hash,
            code: None,
        }))
    }

    fn code_by_hash(&mut self, code_hash: B256) -> Result<Bytecode, Self::Error> {
        self.fetcher
            .bytecode_by_hash(code_hash)
            .map(Bytecode::new_raw)
            .map_err(|e| TrieDBError::Provider(e.to_string()))
    }

    fn storage(&mut self, address: Address, index: U256) -> Result<U256, Self::Error> {
        // Send a hint to the host to fetch the storage proof.
        self.hinter
            .hint_storage_proof(address, index, self.parent_block_header.number)
            .map_err(|e| TrieDBError::Provider(e.to_string()))?;

        // Fetch the account's storage root from the cache. If storage is being accessed, the
        // account should have been loaded into the cache by the `basic` method. If the account was
        // non-existing, the storage root will not be present.
        match self.storage_roots.get_mut(&address) {
            None => {
                // If the storage root for the account does not exist, return zero.
                Ok(U256::ZERO)
            }
            Some(storage_root) => {
                // Fetch the storage slot from the trie.
                let hashed_slot_key = keccak256(index.to_be_bytes::<32>().as_slice());
                match storage_root.open(&Nibbles::unpack(hashed_slot_key), &self.fetcher)? {
                    Some(slot_value) => {
                        // Decode the storage slot value.
                        let int_slot = U256::decode(&mut slot_value.as_ref())
                            .map_err(TrieNodeError::RLPError)?;
                        Ok(int_slot)
                    }
                    None => {
                        // If the storage slot does not exist, return zero.
                        Ok(U256::ZERO)
                    }
                }
            }
        }
    }

    fn block_hash(&mut self, block_number: u64) -> Result<B256, Self::Error> {
        // Copy the current header
        let mut header = self.parent_block_header.inner().clone();

        // Check if the block number is in range. If not, we can fail early.
        if block_number > header.number ||
            header.number.saturating_sub(block_number) > BLOCK_HASH_HISTORY
        {
            return Ok(B256::default());
        }

        // Walk back the block headers to the desired block number.
        while header.number > block_number {
            header = self
                .fetcher
                .header_by_hash(header.parent_hash)
                .map_err(|e| TrieDBError::Provider(e.to_string()))?;
        }

        Ok(header.hash_slow())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Sealable;
    use alloy_primitives::b256;
    use kona_mpt::NoopTrieHinter;

    fn new_test_db() -> TrieDB<NoopTrieDBProvider, NoopTrieHinter> {
        TrieDB::new(Header::default().seal_slow(), NoopTrieDBProvider, NoopTrieHinter)
    }

    #[test]
    fn test_trie_db_take_root_node() {
        let db = new_test_db();
        let root_node = db.take_root_node();
        assert_eq!(root_node.blind(), EMPTY_ROOT_HASH);
    }

    #[test]
    fn test_trie_db_root_node_ref() {
        let db = new_test_db();
        let root_node = db.root();
        assert_eq!(root_node.blind(), EMPTY_ROOT_HASH);
    }

    #[test]
    fn test_trie_db_storage_roots() {
        let db = new_test_db();
        let storage_roots = db.storage_roots();
        assert!(storage_roots.is_empty());
    }

    #[test]
    fn test_block_hash_above_range() {
        let mut db = new_test_db();
        db.parent_block_header = Header { number: 10, ..Default::default() }.seal_slow();
        let block_number = 11;
        let block_hash = db.block_hash(block_number).unwrap();
        assert_eq!(block_hash, B256::default());
    }

    #[test]
    fn test_block_hash_below_range() {
        let mut db = new_test_db();
        db.parent_block_header =
            Header { number: BLOCK_HASH_HISTORY + 10, ..Default::default() }.seal_slow();
        let block_number = 0;
        let block_hash = db.block_hash(block_number).unwrap();
        assert_eq!(block_hash, B256::default());
    }

    #[test]
    fn test_block_hash_provider_missing_hash() {
        let mut db = new_test_db();
        db.parent_block_header = Header { number: 10, ..Default::default() }.seal_slow();
        let block_number = 5;
        let block_hash = db.block_hash(block_number).unwrap();
        assert_eq!(
            block_hash,
            b256!("78dec18c6d7da925bbe773c315653cdc70f6444ed6c1de9ac30bdb36cff74c3b")
        );
    }
}
