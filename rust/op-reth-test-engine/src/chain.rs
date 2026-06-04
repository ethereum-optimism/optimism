//! Ephemeral, genesis-initialized OP chain backed by a temp-dir reth provider.

use std::sync::Arc;

use alloy_consensus::Header;
use alloy_eips::BlockHashOrNumber;
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use reth_chain_state::{ExecutedBlock, NewCanonicalChain};
use reth_db::{DatabaseEnv, test_utils::TempDatabase};
use reth_db_common::init::init_genesis;
use reth_node_api::NodeTypesWithDBAdapter;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::OpNode;
use reth_optimism_primitives::{OpBlock, OpPrimitives, OpReceipt};
use reth_primitives_traits::SealedHeader;
use reth_provider::{
    ProviderError, providers::BlockchainProvider,
    test_utils::create_test_provider_factory_with_node_types,
};
use reth_storage_api::{
    BlockReader, HeaderProvider, ReceiptProvider, StateProviderBox, StateProviderFactory,
};

type TestNodeTypes = NodeTypesWithDBAdapter<OpNode, Arc<TempDatabase<DatabaseEnv>>>;
type Provider = BlockchainProvider<TestNodeTypes>;

/// An ephemeral, genesis-initialized OP chain answering read-only block/header/receipt
/// queries and accepting executed blocks onto the canonical chain.
///
/// State and indexing are provided by a reth provider, so trie state roots and receipt lookups are
/// real rather than reimplemented here. Genesis lives in an ephemeral (tmpfs-backed) database;
/// blocks built on top are held in the provider's in-memory canonical state and overlay it, so
/// nothing past genesis is written to disk. The backing storage is discarded when the chain drops.
#[derive(Debug)]
pub struct EphemeralChain {
    provider: Provider,
    chain_spec: Arc<OpChainSpec>,
    genesis_hash: B256,
}

impl EphemeralChain {
    /// Build an ephemeral chain from a genesis spec, initializing genesis state.
    pub fn new(genesis: Genesis) -> crate::Result<Self> {
        Self::from_chain_spec(Arc::new(genesis.into()))
    }

    /// Build an ephemeral chain from an already-constructed chain spec.
    ///
    /// Tests use this to activate hardforks via [`OpChainSpecBuilder`][reth_optimism_chainspec]
    /// (which encodes activations directly) rather than round-tripping them through genesis JSON.
    pub(crate) fn from_chain_spec(chain_spec: Arc<OpChainSpec>) -> crate::Result<Self> {
        let factory = create_test_provider_factory_with_node_types::<OpNode>(chain_spec.clone());
        let genesis_hash = init_genesis(&factory)?;
        let provider = BlockchainProvider::new(factory)?;
        Ok(Self { provider, chain_spec, genesis_hash })
    }

    /// The chain spec this chain was initialized with.
    pub(crate) fn chain_spec(&self) -> Arc<OpChainSpec> {
        self.chain_spec.clone()
    }

    /// The hash of the genesis block.
    pub const fn genesis_hash(&self) -> B256 {
        self.genesis_hash
    }

    /// The sealed header of the current canonical head.
    #[cfg(test)]
    pub(crate) fn tip_header(&self) -> SealedHeader {
        self.provider.canonical_in_memory_state().get_canonical_head()
    }

    /// A state provider for the chain tip (a memory overlay over the genesis DB state once
    /// blocks have been built), used as the parent state for executing the next block.
    #[cfg(test)]
    pub(crate) fn latest_state(&self) -> crate::Result<StateProviderBox> {
        Ok(self.provider.latest()?)
    }

    /// A state provider rooted at `hash`, or `None` if that block is unknown to the chain.
    pub(crate) fn state_at(&self, hash: B256) -> crate::Result<Option<StateProviderBox>> {
        match self.provider.state_by_block_hash(hash) {
            Ok(state) => Ok(Some(state)),
            Err(ProviderError::StateForHashNotFound(_)) => Ok(None),
            Err(err) => Err(err.into()),
        }
    }

    /// Commit an executed block as the new canonical head.
    ///
    /// Both calls are required: `update_chain` inserts the block into the in-memory map (making it
    /// queryable), while `set_canonical_head` advances the head pointer that `latest()` and
    /// `best_block_number` read.
    pub(crate) fn commit_block(&self, executed: ExecutedBlock<OpPrimitives>) {
        let head = executed.recovered_block.clone_sealed_header();
        let state = self.provider.canonical_in_memory_state();
        state.update_chain(NewCanonicalChain::Commit { new: vec![executed] });
        state.set_canonical_head(head);
    }

    /// Point the canonical/safe/finalized heads at the given hashes (attrs-less forkchoice).
    ///
    /// Returns `Ok(false)` without mutating anything if `head` is unknown to the chain, which the
    /// engine maps to `SYNCING`. A zero `safe`/`finalized` hash is skipped; a non-zero one that is
    /// unknown is an [`Error::UnknownForkchoiceBlock`](crate::Error::UnknownForkchoiceBlock). All
    /// three are resolved before any pointer is moved.
    pub(crate) fn advance_forkchoice(
        &self,
        head: B256,
        safe: B256,
        finalized: B256,
    ) -> crate::Result<bool> {
        let Some(head_header) = self.provider.header(head)? else {
            return Ok(false);
        };
        let safe_header = self.resolve_forkchoice_block("safe", safe)?;
        let finalized_header = self.resolve_forkchoice_block("finalized", finalized)?;

        let state = self.provider.canonical_in_memory_state();
        state.set_canonical_head(SealedHeader::seal_slow(head_header));
        if let Some(header) = safe_header {
            state.set_safe(SealedHeader::seal_slow(header));
        }
        if let Some(header) = finalized_header {
            state.set_finalized(SealedHeader::seal_slow(header));
        }
        Ok(true)
    }

    /// Look up a non-zero `safe`/`finalized` forkchoice block, erroring if it is unknown. A zero
    /// hash means "unset" and resolves to `None`.
    fn resolve_forkchoice_block(
        &self,
        which: &'static str,
        hash: B256,
    ) -> crate::Result<Option<Header>> {
        if hash.is_zero() {
            return Ok(None);
        }
        self.provider
            .header(hash)?
            .map(Some)
            .ok_or(crate::Error::UnknownForkchoiceBlock { which, hash })
    }

    /// Fetch a block by number, or `None` if unknown.
    pub fn block_by_number(&self, number: u64) -> crate::Result<Option<OpBlock>> {
        Ok(self.provider.block_by_number(number)?)
    }

    /// Fetch a block by hash, or `None` if unknown.
    pub fn block_by_hash(&self, hash: B256) -> crate::Result<Option<OpBlock>> {
        Ok(self.provider.block_by_hash(hash)?)
    }

    /// Fetch a header by number, or `None` if unknown.
    pub fn header_by_number(&self, number: u64) -> crate::Result<Option<Header>> {
        Ok(self.provider.header_by_number(number)?)
    }

    /// Fetch the receipts of a block by hash, or `None` if unknown.
    pub fn receipts_by_block_hash(&self, hash: B256) -> crate::Result<Option<Vec<OpReceipt>>> {
        Ok(self.provider.receipts_by_block(BlockHashOrNumber::Hash(hash))?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_genesis::GenesisAccount;
    use alloy_primitives::{Address, U256};

    fn test_genesis() -> Genesis {
        Genesis::default().extend_accounts([(
            Address::with_last_byte(0x42),
            GenesisAccount { balance: U256::from(1_000_000u64), ..Default::default() },
        )])
    }

    #[test]
    fn genesis_roundtrip() {
        let chain = EphemeralChain::new(test_genesis()).expect("build chain");
        let header = chain.header_by_number(0).expect("query").expect("genesis header present");
        // The hash reth recorded for genesis matches the header we read back.
        assert_eq!(header.hash_slow(), chain.genesis_hash());
        // Genesis state root is a real trie root (alloc applied), not the zero hash.
        assert_ne!(header.state_root, B256::ZERO);
        // The full genesis block is queryable and hashes to the same genesis hash.
        let block = chain.block_by_number(0).expect("query").expect("genesis block present");
        assert_eq!(block.header.hash_slow(), chain.genesis_hash());
    }

    #[test]
    fn queries_on_genesis_only_chain() {
        let chain = EphemeralChain::new(test_genesis()).expect("build chain");
        // Genesis is the latest (and only) block.
        assert!(chain.block_by_number(0).expect("query").is_some());
        // Unknown inputs return `None` across all four read methods.
        let unknown = B256::repeat_byte(0xab);
        assert!(chain.block_by_hash(unknown).expect("query").is_none());
        assert!(chain.header_by_number(99).expect("query").is_none());
        assert!(chain.receipts_by_block_hash(unknown).expect("query").is_none());
    }
}
