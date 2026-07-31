//! Ephemeral, genesis-initialized OP chain backed by a temp-dir reth provider.

use std::sync::Arc;

use alloy_consensus::Header;
use alloy_eips::BlockHashOrNumber;
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use reth_db::{DatabaseEnv, test_utils::TempDatabase};
use reth_db_common::init::init_genesis;
use reth_node_api::NodeTypesWithDBAdapter;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::OpNode;
use reth_optimism_primitives::{OpBlock, OpReceipt};
use reth_provider::{
    providers::BlockchainProvider, test_utils::create_test_provider_factory_with_node_types,
};
use reth_storage_api::{BlockReader, HeaderProvider, ReceiptProvider};

type TestNodeTypes = NodeTypesWithDBAdapter<OpNode, Arc<TempDatabase<DatabaseEnv>>>;
type Provider = BlockchainProvider<TestNodeTypes>;

/// An ephemeral, genesis-initialized OP chain answering read-only block/header/receipt
/// queries.
///
/// State and indexing are provided by a reth provider, so trie state roots and receipt
/// lookups are real rather than reimplemented here. The backing storage is ephemeral and
/// discarded when the chain is dropped.
#[derive(Debug)]
pub struct EphemeralChain {
    provider: Provider,
    genesis_hash: B256,
}

impl EphemeralChain {
    /// Build an ephemeral chain from a genesis spec, initializing genesis state.
    pub fn new(genesis: Genesis) -> crate::Result<Self> {
        let chain_spec: OpChainSpec = genesis.into();
        let factory = create_test_provider_factory_with_node_types::<OpNode>(Arc::new(chain_spec));
        let genesis_hash = init_genesis(&factory)?;
        let provider = BlockchainProvider::new(factory)?;
        Ok(Self { provider, genesis_hash })
    }

    /// The hash of the genesis block.
    pub const fn genesis_hash(&self) -> B256 {
        self.genesis_hash
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
