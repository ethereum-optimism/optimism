use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
use alloy_primitives::B256;
use reth_chainspec::MAINNET;
use reth_db_common::init::init_genesis;
use reth_ethereum_primitives::Block;
use reth_evm_ethereum::EthEvmConfig;
use reth_optimism_trie::{
    MdbxProofsStorageV2, OpProofStoragePruner, OpProofsInitProvider, OpProofsProviderRO,
    OpProofsStore, engine::EngineHandle,
};
use reth_provider::{
    providers::BlockchainProvider, test_utils::create_test_provider_factory_with_chain_spec,
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use std::sync::Arc;
use tempfile::TempDir;

#[test]
fn buffered_tail_is_persisted_on_shutdown() -> eyre::Result<()> {
    let chain_spec = MAINNET.clone();
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory)?;
    let blockchain_db = BlockchainProvider::new(provider_factory)?;

    let proofs_dir = TempDir::new()?;
    let storage = Arc::new(MdbxProofsStorageV2::new(proofs_dir.path())?);
    let genesis = BlockNumHash::new(0, chain_spec.genesis_hash());
    let init = storage.initialization_provider()?;
    init.set_initial_state_anchor(genesis)?;
    init.commit_initial_state()?;
    OpProofsInitProvider::commit(init)?;

    let pruner = OpProofStoragePruner::new(storage.clone(), blockchain_db.clone(), 1_000);
    let engine = EngineHandle::<Block>::spawn(
        EthEvmConfig::ethereum(chain_spec),
        blockchain_db,
        storage.clone(),
        pruner,
    );

    // One block remains below the default five-block persistence threshold.
    let tail = NumHash::new(1, B256::repeat_byte(0x01));
    engine.index_block(
        BlockWithParent::new(genesis.hash, tail),
        TrieUpdatesSorted::default(),
        HashedPostStateSorted::default(),
    )?;

    assert_eq!(storage.provider_ro()?.get_latest_block()?, genesis);

    // Dropping the last handle disconnects and joins the collector engine.
    drop(engine);
    assert_eq!(storage.provider_ro()?.get_latest_block()?, tail);

    Ok(())
}
