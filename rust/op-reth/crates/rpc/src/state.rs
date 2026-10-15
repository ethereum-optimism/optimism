//! State provider factory for OP Proofs ExEx.

use alloy_eips::BlockId;
use derive_more::Constructor;
use reth_optimism_trie::{
    OpProofsStorage, OpProofsStorageError, OpProofsStore, api::OpProofsProviderRO,
    provider::OpProofsStateProviderRef,
};
use reth_provider::{BlockHashReader, BlockIdReader, ProviderError, ProviderResult, StateProvider};
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_eth_types::EthApiError;

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
        let block_number = canonical_block_number(&self.eth_api, block_id)?;

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
/// [`OpStateProviderFactory::state_provider`] checks the window again in the transaction it
/// reads from, because pruning can move it in between.
pub fn ensure_within_proofs_window<Eth, P>(
    eth_api: &Eth,
    preimage_store: &OpProofsStorage<P>,
    block_id: BlockId,
) -> ProviderResult<u64>
where
    Eth: FullEthApi,
    P: OpProofsStore,
{
    let block_number = canonical_block_number(eth_api, block_id)?;
    let provider_ro = preimage_store.provider_ro().map_err(ProviderError::from)?;
    ensure_in_proof_window(&provider_ro, block_number)?;
    Ok(block_number)
}

/// Resolves `block_id` to a block number, rejecting a hash that names a non-canonical block.
fn canonical_block_number<Eth: FullEthApi>(
    eth_api: &Eth,
    block_id: BlockId,
) -> ProviderResult<u64> {
    let block_number = eth_api
        .provider()
        .block_number_for_id(block_id)?
        .ok_or(EthApiError::HeaderNotFound(block_id))
        .map_err(ProviderError::other)?;

    // The proofs storage is canonical-only and is overlaid by block number, so a hash that
    // names a non-canonical block would be served canonical state for its height.
    if let BlockId::Hash(hash) = block_id &&
        eth_api.provider().block_hash(block_number)? != Some(hash.block_hash)
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
    use alloy_consensus::Header;
    use alloy_eips::{BlockNumHash, eip1898::BlockWithParent};
    use alloy_primitives::{Address, B256, keccak256};
    use reth_chainspec::{ChainSpec, ChainSpecProvider};
    use reth_evm_ethereum::EthEvmConfig;
    use reth_network_api::noop::NoopNetwork;
    use reth_optimism_trie::{
        BlockStateDiff, InMemoryProofsStorage, OpProofsInitProvider, OpProofsProviderRw,
    };
    use reth_primitives_traits::Account;
    use reth_provider::test_utils::MockEthProvider;
    use reth_rpc::{EthApi, eth::helpers::types::EthRpcConverter};
    use reth_rpc_eth_api::node::RpcNodeCoreAdapter;
    use reth_transaction_pool::test_utils::{TestPool, testing_pool};
    use reth_trie_common::HashedPostState;
    use rstest::rstest;


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

    type MockEthApi = EthApi<
        RpcNodeCoreAdapter<MockEthProvider, TestPool, NoopNetwork, EthEvmConfig>,
        EthRpcConverter<ChainSpec>,
    >;

    /// Every block from genesis to this height is canonical on the mock chain.
    const CHAIN_HEAD: u64 = 20;
    /// The block every store in these tests is anchored at.
    const STORE_EARLIEST: u64 = 10;
    /// The latest block a store holds while it trails the chain head.
    const STORE_LATEST: u64 = 18;
    /// An account whose nonce the store records as the height of every block it holds, so a test
    /// can tell which block's state a provider serves.
    const HEIGHT_MARKER: Address = Address::repeat_byte(0x11);

    fn canonical_hash(number: u64) -> B256 {
        keccak256(number.to_be_bytes())
    }

    fn by_hash(number: u64) -> BlockId {
        BlockId::from(canonical_hash(number))
    }

    fn height_marker_at(height: u64) -> (B256, Option<Account>) {
        (keccak256(HEIGHT_MARKER), Some(Account { nonce: height, ..Default::default() }))
    }

    fn mock_eth_api() -> MockEthApi {
        let provider = MockEthProvider::default();
        provider.extend_headers(
            (0..=CHAIN_HEAD)
                .map(|number| (canonical_hash(number), Header { number, ..Default::default() })),
        );
        let evm_config = EthEvmConfig::new(provider.chain_spec());
        EthApi::builder(provider, testing_pool(), NoopNetwork::default(), evm_config).build()
    }

    /// A store anchored at `STORE_EARLIEST` holding every block through `latest`, each recording
    /// its height in the marker account.
    fn store_through(latest: u64) -> InMemoryProofsStorage {
        let store = InMemoryProofsStorage::new();
        let init = store.initialization_provider().expect("initialization provider");
        init.set_initial_state_anchor(BlockNumHash::new(
            STORE_EARLIEST,
            canonical_hash(STORE_EARLIEST),
        ))
        .expect("anchor");
        init.store_hashed_accounts(vec![height_marker_at(STORE_EARLIEST)]).expect("anchor marker");
        init.commit_initial_state().expect("commit initial state");
        OpProofsInitProvider::commit(init).expect("commit");

        let rw = store.provider_rw().expect("rw provider");
        for number in STORE_EARLIEST + 1..=latest {
            let block_ref = BlockWithParent {
                parent: canonical_hash(number - 1),
                block: BlockNumHash::new(number, canonical_hash(number)),
            };
            let diff = BlockStateDiff {
                sorted_post_state: HashedPostState::default()
                    .with_accounts([height_marker_at(number)])
                    .into_sorted(),
                ..Default::default()
            };
            rw.store_trie_updates(block_ref, diff).expect("store block");
        }
        OpProofsProviderRw::commit(rw).expect("commit");
        store
    }

    fn factory_over(
        store: InMemoryProofsStorage,
    ) -> OpStateProviderFactory<MockEthApi, InMemoryProofsStorage> {
        OpStateProviderFactory::new(mock_eth_api(), store.into())
    }

    fn served_height(state: &dyn StateProvider) -> u64 {
        state.basic_account(&HEIGHT_MARKER).expect("marker lookup").expect("marker account").nonce
    }

    fn assert_state_not_found(err: &ProviderError, block: u64) {
        assert!(
            matches!(err, ProviderError::StateForNumberNotFound(number) if *number == block),
            "expected no state for block {block}, got: {err}"
        );
    }

    #[rstest]
    #[case::earliest_block(BlockId::from(STORE_EARLIEST), STORE_EARLIEST)]
    #[case::earliest_block_by_hash(by_hash(STORE_EARLIEST), STORE_EARLIEST)]
    #[case::inside_the_window(BlockId::from(STORE_EARLIEST + 3), STORE_EARLIEST + 3)]
    #[case::latest_stored_block(BlockId::from(STORE_LATEST), STORE_LATEST)]
    #[case::latest_stored_block_by_hash(by_hash(STORE_LATEST), STORE_LATEST)]
    #[tokio::test]
    async fn serves_every_block_inside_the_window_at_its_own_height(
        #[case] block_id: BlockId,
        #[case] height: u64,
    ) {
        let factory = factory_over(store_through(STORE_LATEST));

        let state = factory
            .state_provider((block_id))
            .await
            .expect("block inside the window is served");

        assert_eq!(served_height(state.as_ref()), height);
    }

    #[rstest]
    #[case::one_below_the_earliest(BlockId::from(STORE_EARLIEST - 1), STORE_EARLIEST - 1)]
    #[case::one_below_the_earliest_by_hash(by_hash(STORE_EARLIEST - 1), STORE_EARLIEST - 1)]
    #[case::genesis(BlockId::from(0), 0)]
    #[case::one_above_the_store_tip(BlockId::from(STORE_LATEST + 1), STORE_LATEST + 1)]
    #[case::chain_head_the_store_has_not_flushed(BlockId::from(CHAIN_HEAD), CHAIN_HEAD)]
    #[case::latest_tag_while_the_store_trails(BlockId::latest(), CHAIN_HEAD)]
    #[tokio::test]
    async fn refuses_every_block_outside_the_window(
        #[case] block_id: BlockId,
        #[case] refused_block: u64,
    ) {
        let factory = factory_over(store_through(STORE_LATEST));

        let err = factory
            .state_provider((block_id))
            .await
            .err()
            .expect("block outside the window is refused");

        assert_state_not_found(&err, refused_block);
    }

    #[rstest]
    #[case::latest_tag(BlockId::latest())]
    #[case::chain_head_by_number(BlockId::from(CHAIN_HEAD))]
    #[tokio::test]
    async fn serves_the_chain_head_once_the_store_caught_up(#[case] block_id: BlockId) {
        let factory = factory_over(store_through(CHAIN_HEAD));

        let state =
            factory.state_provider(block_id).await.expect("chain head is served once stored");

        assert_eq!(served_height(state.as_ref()), CHAIN_HEAD);
    }

    #[tokio::test]
    async fn a_freshly_initialized_store_serves_only_its_anchor() {
        let factory = factory_over(store_through(STORE_EARLIEST));

        let anchor = factory
            .state_provider((BlockId::from(STORE_EARLIEST)))
            .await
            .expect("anchor block is served");
        let err = factory
            .state_provider((BlockId::from(STORE_EARLIEST + 1)))
            .await
            .err()
            .expect("block after the anchor is refused");

        assert_eq!(served_height(anchor.as_ref()), STORE_EARLIEST);
        assert_state_not_found(&err, STORE_EARLIEST + 1);
    }

    #[tokio::test]
    async fn reports_an_unknown_hash_as_header_not_found() {
        let factory = factory_over(store_through(STORE_LATEST));
        let unknown_block = BlockId::from(B256::repeat_byte(0xab));

        let err = factory
            .state_provider((unknown_block))
            .await
            .err()
            .expect("unknown hash is refused");

        assert!(
            matches!(
                err.downcast_other_ref::<EthApiError>(),
                Some(EthApiError::HeaderNotFound(id)) if *id == unknown_block
            ),
            "expected header not found for {unknown_block}, got: {err}"
        );
    }

    #[tokio::test]
    async fn refuses_every_block_while_the_store_is_empty() {
        let factory = factory_over(InMemoryProofsStorage::new());

        let err = factory
            .state_provider((BlockId::from(STORE_EARLIEST)))
            .await
            .err()
            .expect("empty store serves nothing");

        assert_state_not_found(&err, STORE_EARLIEST);
    }
}
