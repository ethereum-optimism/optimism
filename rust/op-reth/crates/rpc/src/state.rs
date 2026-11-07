//! Proof and state provider factories for OP Proofs ExEx.

use alloy_eips::BlockId;
use alloy_primitives::{Address, B256};
use derive_more::Constructor;
use reth_optimism_trie::{
    OpProofsStorage, OpProofsStorageError, OpProofsStore, api::OpProofsProviderRO,
    proof::DatabaseProof, provider::OpProofsStateProviderRef,
};
use reth_provider::{BlockIdReader, ProviderError, ProviderResult, StateProvider};
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_eth_types::EthApiError;
use reth_trie::{AccountProof, proof::Proof};

/// Creates a factory for state providers using OP Proofs external proofs storage.
#[derive(Debug, Constructor)]
pub struct OpStateProviderFactory<Eth, P> {
    eth_api: Eth,
    preimage_store: OpProofsStorage<P>,
}

impl<'a, Eth, P> OpStateProviderFactory<Eth, P>
where
    Eth: FullEthApi + 'static,
    P: OpProofsStore + Clone + 'a,
{
    /// Creates a state provider for the given block id.
    pub async fn state_provider(
        &'a self,
        block_id: BlockId,
    ) -> ProviderResult<Box<dyn StateProvider + 'a>> {
        let block_number = canonical_block_number(self.eth_api.provider(), block_id)?;

        let historical_provider =
            self.eth_api.state_at_block_id(block_id).await.map_err(ProviderError::other)?;

        let provider_ro = self.preimage_store.provider_ro().map_err(ProviderError::from)?;

        // Check the window in the transaction the overlay reads from, so pruning cannot move it
        // in between.
        ensure_in_proof_window(&provider_ro, block_number)?;

        let external_overlay_provider =
            OpProofsStateProviderRef::new(historical_provider, provider_ro, block_number);

        Ok(Box::new(external_overlay_provider))
    }
}

/// Serves account proofs using only block identifiers from reth and state from the proofs DB.
///
/// Unlike witness execution, account proofs need neither bytecode nor a reth historical state
/// provider. Proof availability follows the proofs DB's retained window independently of reth's
/// historical state retention.
#[derive(Debug, Constructor)]
pub(crate) struct OpProofProviderFactory<Provider, P> {
    provider: Provider,
    proofs_storage: OpProofsStorage<P>,
}

impl<Provider: BlockIdReader, P: OpProofsStore> OpProofProviderFactory<Provider, P> {
    pub(crate) fn account_proof(
        &self,
        block_id: BlockId,
        address: Address,
        slots: &[B256],
    ) -> ProviderResult<AccountProof> {
        let block_number = canonical_block_number(&self.provider, block_id)?;
        let provider = self.proofs_storage.provider_ro().map_err(ProviderError::from)?;
        // Validate and read through the same transaction, so pruning cannot move the window
        // between admission and proof generation.
        ensure_in_proof_window(&provider, block_number)?;
        Proof::from_provider(provider, block_number)
            .account_proof(address, slots)
            .map_err(ProviderError::from)
    }
}

/// Resolves `block_id` to a canonical block number inside the proofs storage window.
///
/// UPSTREAM-MIRROR(copy): reth@rev:8d359d0
/// `reth_rpc_eth_api::helpers::state::EthState::ensure_within_proof_window`
///
/// Stands in for upstream's check in the historical `eth_getProof` path. Both resolve the block
/// and reject an unknown one with `HeaderNotFound`. Upstream then bounds the distance from the tip
/// by `max_proof_window`; this instead requires a canonical block inside the proofs storage
/// window. On a bump, diff upstream's body for new conditions that should also apply here.
///
/// Callers that serve a proof check this before taking a proof permit, so out-of-window requests
/// are rejected without holding a permit or a blocking-task slot.
/// Proof generation and [`OpStateProviderFactory::state_provider`] check the window again in
/// the transaction they read from, because pruning can move it in between.
pub fn ensure_within_proofs_window<Eth, P>(
    eth_api: &Eth,
    preimage_store: &OpProofsStorage<P>,
    block_id: BlockId,
) -> ProviderResult<u64>
where
    Eth: FullEthApi,
    P: OpProofsStore,
{
    let block_number = canonical_block_number(eth_api.provider(), block_id)?;
    let provider_ro = preimage_store.provider_ro().map_err(ProviderError::from)?;
    ensure_in_proof_window(&provider_ro, block_number)?;
    Ok(block_number)
}

/// Resolves `block_id` to a block number, rejecting missing or non-canonical headers.
fn canonical_block_number<Provider: BlockIdReader>(
    provider: &Provider,
    block_id: BlockId,
) -> ProviderResult<u64> {
    let block_number = provider
        .block_number_for_id(block_id)?
        .ok_or(EthApiError::HeaderNotFound(block_id))
        .map_err(ProviderError::other)?;

    // A numeric ID resolves without checking header existence. After a rewind the proofs DB
    // can still cover a height reth no longer has; reject it rather than serving stale state.
    let canonical_hash = provider
        .block_hash(block_number)?
        .ok_or(EthApiError::HeaderNotFound(block_id))
        .map_err(ProviderError::other)?;

    // The proofs storage is canonical-only and is overlaid by block number, so a hash that
    // names a non-canonical block would be served canonical state for its height.
    if let BlockId::Hash(hash) = block_id &&
        canonical_hash != hash.block_hash
    {
        return Err(ProviderError::other(EthApiError::HeaderNotFound(block_id)));
    }

    Ok(block_number)
}

/// Rejects a block outside the proofs storage window visible to `provider`.
fn ensure_in_proof_window(
    provider: &impl OpProofsProviderRO,
    block_number: u64,
) -> ProviderResult<()> {
    // An empty proof window is semantically equivalent to an out-of-window lookup for the proofs
    // RPC: there's no proof for that block. Any other storage error is real and propagates up.
    let window = match provider.get_proof_window() {
        Ok(w) => w,
        Err(OpProofsStorageError::NoBlocksFound) => {
            return Err(ProviderError::StateForNumberNotFound(block_number));
        }
        Err(err) => return Err(ProviderError::from(err)),
    };

    if block_number < window.earliest.number || block_number > window.latest.number {
        return Err(ProviderError::StateForNumberNotFound(block_number));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::{BlockNumHash, BlockNumberOrTag};
    use alloy_primitives::{U256, keccak256, map::B256Map};
    use alloy_trie::root::{state_root_unhashed, storage_root_unhashed};
    use reth_chainspec::ChainInfo;
    use reth_optimism_trie::{
        BlockStateDiff, InMemoryProofsStorage, MdbxProofsStorage, OpProofsInitProvider,
        OpProofsProviderRw,
    };
    use reth_primitives_traits::Account;
    use reth_provider::{BlockHashReader, BlockNumReader};
    use std::sync::Arc;

    /// Intentionally implements no state-provider traits: proof serving must need only headers.
    #[derive(Debug)]
    struct HeadersOnly {
        tip: BlockNumHash,
    }

    const ANCHOR: BlockNumHash = BlockNumHash::new(10, B256::repeat_byte(1));
    const SIDE_HASH: B256 = B256::repeat_byte(2);
    const ADDRESS: Address = Address::repeat_byte(3);
    const SLOT: B256 = B256::repeat_byte(4);

    impl BlockHashReader for HeadersOnly {
        fn block_hash(&self, number: u64) -> ProviderResult<Option<B256>> {
            Ok(if number > self.tip.number {
                None
            } else if number == ANCHOR.number {
                Some(ANCHOR.hash)
            } else if number == self.tip.number {
                Some(self.tip.hash)
            } else {
                Some(B256::repeat_byte(u8::try_from(number).unwrap()))
            })
        }

        fn canonical_hashes_range(&self, start: u64, end: u64) -> ProviderResult<Vec<B256>> {
            Ok((start..end).filter_map(|number| self.block_hash(number).unwrap()).collect())
        }
    }

    impl BlockNumReader for HeadersOnly {
        fn chain_info(&self) -> ProviderResult<ChainInfo> {
            Ok(ChainInfo { best_number: self.tip.number, best_hash: self.tip.hash })
        }

        fn best_block_number(&self) -> ProviderResult<u64> {
            Ok(self.tip.number)
        }

        fn last_block_number(&self) -> ProviderResult<u64> {
            Ok(self.tip.number)
        }

        fn block_number(&self, hash: B256) -> ProviderResult<Option<u64>> {
            Ok(if [ANCHOR.hash, SIDE_HASH].contains(&hash) {
                Some(ANCHOR.number)
            } else {
                (hash == self.tip.hash).then_some(self.tip.number)
            })
        }
    }

    impl BlockIdReader for HeadersOnly {
        fn pending_block_num_hash(&self) -> ProviderResult<Option<BlockNumHash>> {
            Ok(None)
        }

        fn safe_block_num_hash(&self) -> ProviderResult<Option<BlockNumHash>> {
            Ok(Some(ANCHOR))
        }

        fn finalized_block_num_hash(&self) -> ProviderResult<Option<BlockNumHash>> {
            Ok(Some(ANCHOR))
        }
    }

    impl Default for HeadersOnly {
        fn default() -> Self {
            Self { tip: ANCHOR }
        }
    }

    impl HeadersOnly {
        fn account() -> Account {
            Account { nonce: 3, balance: U256::from(42), bytecode_hash: None }
        }

        fn seed(storage: &impl OpProofsStore) {
            let init = storage.initialization_provider().unwrap();
            init.set_initial_state_anchor(ANCHOR).unwrap();
            init.store_hashed_accounts(vec![(keccak256(ADDRESS), Some(Self::account()))]).unwrap();
            init.store_hashed_storages(keccak256(ADDRESS), vec![(keccak256(SLOT), U256::from(11))])
                .unwrap();
            init.commit_initial_state().unwrap();
            init.commit().unwrap();
        }

        fn state_root() -> B256 {
            Self::state_root_at(42, 11)
        }

        fn state_root_at(balance: u64, value: u64) -> B256 {
            let storage_root = storage_root_unhashed([(SLOT, U256::from(value))]);
            let account = Account { balance: U256::from(balance), ..Self::account() };
            state_root_unhashed([(ADDRESS, account.into_trie_account(storage_root))])
        }

        fn diff(balance: u64, value: u64) -> BlockStateDiff {
            BlockStateDiff {
                sorted_post_state: reth_trie::HashedPostState {
                    accounts: B256Map::from_iter([(
                        keccak256(ADDRESS),
                        Some(Account { balance: U256::from(balance), ..Self::account() }),
                    )]),
                    storages: B256Map::from_iter([(
                        keccak256(ADDRESS),
                        reth_trie::HashedStorage::from_iter([(keccak256(SLOT), U256::from(value))]),
                    )]),
                }
                .into_sorted(),
                ..Default::default()
            }
        }
    }

    #[test]
    fn account_proofs_need_only_headers_and_proofs_storage() {
        let storage = InMemoryProofsStorage::new();
        HeadersOnly::seed(&storage);
        let proofs = OpProofProviderFactory::<_, InMemoryProofsStorage>::new(
            HeadersOnly::default(),
            storage.into(),
        );
        let missing_slot = B256::repeat_byte(5);

        for block_id in [
            BlockId::number(ANCHOR.number),
            BlockId::Hash(ANCHOR.hash.into()),
            BlockNumberOrTag::Latest.into(),
            BlockNumberOrTag::Safe.into(),
            BlockNumberOrTag::Finalized.into(),
        ] {
            let proof = proofs.account_proof(block_id, ADDRESS, &[SLOT, missing_slot]).unwrap();
            assert_eq!(proof.info, Some(HeadersOnly::account()));
            assert_eq!(proof.storage_proofs[0].value, U256::from(11));
            assert_eq!(proof.storage_proofs[1].value, U256::ZERO);
            proof.verify(HeadersOnly::state_root()).unwrap();
        }

        let absent = proofs.account_proof(BlockId::number(10), Address::ZERO, &[]).unwrap();
        assert!(absent.info.is_none());
        absent.verify(HeadersOnly::state_root()).unwrap();
    }

    #[test]
    fn account_proofs_reject_unavailable_and_noncanonical_blocks() {
        let storage = InMemoryProofsStorage::new();
        let proofs = OpProofProviderFactory::<_, InMemoryProofsStorage>::new(
            HeadersOnly { tip: BlockNumHash::new(11, B256::repeat_byte(11)) },
            storage.clone().into(),
        );
        assert!(matches!(
            proofs.account_proof(BlockId::number(10), ADDRESS, &[]),
            Err(ProviderError::StateForNumberNotFound(10))
        ));
        HeadersOnly::seed(&storage);
        for number in [9, 11] {
            assert!(matches!(
                proofs.account_proof(BlockId::number(number), ADDRESS, &[]),
                Err(ProviderError::StateForNumberNotFound(n)) if n == number
            ));
        }
        for hash in [SIDE_HASH, B256::repeat_byte(99)] {
            let err = proofs.account_proof(BlockId::Hash(hash.into()), ADDRESS, &[]).unwrap_err();
            assert!(matches!(
                err.as_other().and_then(|err| err.downcast_ref::<EthApiError>()),
                Some(EthApiError::HeaderNotFound(_))
            ));
        }
    }

    #[test]
    fn account_proofs_reject_retained_state_above_the_rewound_chain() {
        let storage = InMemoryProofsStorage::new();
        HeadersOnly::seed(&storage);
        let proofs = OpProofProviderFactory::<_, InMemoryProofsStorage>::new(
            HeadersOnly { tip: BlockNumHash::new(9, B256::repeat_byte(9)) },
            storage.into(),
        );
        let err = proofs.account_proof(BlockId::number(10), ADDRESS, &[]).unwrap_err();
        assert!(matches!(
            err.as_other().and_then(|err| err.downcast_ref::<EthApiError>()),
            Some(EthApiError::HeaderNotFound(_))
        ));
    }

    #[test]
    fn reopened_proofs_storage_serves_last_committed_state() {
        let dir = tempfile::tempdir().unwrap();
        let committed = BlockNumHash::new(11, B256::repeat_byte(11));
        {
            let storage = MdbxProofsStorage::new(dir.path()).unwrap();
            HeadersOnly::seed(&storage);
            let writer = storage.provider_rw().unwrap();
            writer
                .store_trie_updates(
                    alloy_eips::eip1898::BlockWithParent::new(ANCHOR.hash, committed),
                    HeadersOnly::diff(43, 22),
                )
                .unwrap();
            OpProofsProviderRw::commit(writer).unwrap();

            // Drop a write transaction after changing account and storage data, before commit.
            let interrupted = storage.provider_rw().unwrap();
            interrupted
                .store_trie_updates(
                    alloy_eips::eip1898::BlockWithParent::new(
                        committed.hash,
                        BlockNumHash::new(12, B256::repeat_byte(12)),
                    ),
                    HeadersOnly::diff(99, 99),
                )
                .unwrap();
        }
        let reopened = Arc::new(MdbxProofsStorage::new(dir.path()).unwrap());
        let proofs = OpProofProviderFactory::<_, Arc<MdbxProofsStorage>>::new(
            HeadersOnly { tip: BlockNumHash::new(12, B256::repeat_byte(12)) },
            reopened.into(),
        );
        for (number, balance, value) in [(10, 42, 11), (11, 43, 22)] {
            let proof = proofs.account_proof(BlockId::number(number), ADDRESS, &[SLOT]).unwrap();
            assert_eq!(proof.info.unwrap().balance, U256::from(balance));
            assert_eq!(proof.storage_proofs[0].value, U256::from(value));
            proof.verify(HeadersOnly::state_root_at(balance, value)).unwrap();
        }
        assert!(matches!(
            proofs.account_proof(BlockId::number(12), ADDRESS, &[]),
            Err(ProviderError::StateForNumberNotFound(12))
        ));
    }

    #[test]
    fn proof_window_check_rejects_blocks_outside_the_stored_window() {
        let storage = InMemoryProofsStorage::new();
        assert!(matches!(
            ensure_in_proof_window(&storage.provider_ro().unwrap(), 10),
            Err(ProviderError::StateForNumberNotFound(10))
        ));

        let init = storage.initialization_provider().unwrap();
        init.set_initial_state_anchor(BlockNumHash::new(10, B256::repeat_byte(1))).unwrap();
        init.commit_initial_state().unwrap();

        let provider = storage.provider_ro().unwrap();
        ensure_in_proof_window(&provider, 10).unwrap();
        for block_number in [9, 11] {
            assert!(matches!(
                ensure_in_proof_window(&provider, block_number),
                Err(ProviderError::StateForNumberNotFound(n)) if n == block_number
            ));
        }
    }
}
