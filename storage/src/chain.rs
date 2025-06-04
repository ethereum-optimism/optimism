use alloc::{vec, vec::Vec};
use alloy_consensus::Header;
use alloy_primitives::BlockNumber;
use core::marker::PhantomData;
use reth_chainspec::{ChainSpecProvider, EthChainSpec, EthereumHardforks};
use reth_db_api::transaction::{DbTx, DbTxMut};
use reth_node_api::{FullNodePrimitives, FullSignedTx};
use reth_optimism_primitives::OpTransactionSigned;
use reth_primitives_traits::{Block, FullBlockHeader, SignedTransaction};
use reth_provider::{
    providers::{ChainStorage, NodeTypesForProvider},
    DatabaseProvider,
};
use reth_storage_api::{
    errors::ProviderResult, BlockBodyReader, BlockBodyWriter, ChainStorageReader,
    ChainStorageWriter, DBProvider, ReadBodyInput, StorageLocation,
};

/// Optimism storage implementation.
#[derive(Debug, Clone, Copy)]
pub struct OpStorage<T = OpTransactionSigned, H = Header>(PhantomData<(T, H)>);

impl<T, H> Default for OpStorage<T, H> {
    fn default() -> Self {
        Self(Default::default())
    }
}

impl<N, T, H> ChainStorage<N> for OpStorage<T, H>
where
    T: FullSignedTx,
    H: FullBlockHeader,
    N: FullNodePrimitives<
        Block = alloy_consensus::Block<T, H>,
        BlockHeader = H,
        BlockBody = alloy_consensus::BlockBody<T, H>,
        SignedTx = T,
    >,
{
    fn reader<TX, Types>(&self) -> impl ChainStorageReader<DatabaseProvider<TX, Types>, N>
    where
        TX: DbTx + 'static,
        Types: NodeTypesForProvider<Primitives = N>,
    {
        self
    }

    fn writer<TX, Types>(&self) -> impl ChainStorageWriter<DatabaseProvider<TX, Types>, N>
    where
        TX: DbTxMut + DbTx + 'static,
        Types: NodeTypesForProvider<Primitives = N>,
    {
        self
    }
}

impl<Provider, T, H> BlockBodyWriter<Provider, alloy_consensus::BlockBody<T, H>> for OpStorage<T, H>
where
    Provider: DBProvider<Tx: DbTxMut>,
    T: SignedTransaction,
    H: FullBlockHeader,
{
    fn write_block_bodies(
        &self,
        _provider: &Provider,
        _bodies: Vec<(u64, Option<alloy_consensus::BlockBody<T, H>>)>,
        _write_to: StorageLocation,
    ) -> ProviderResult<()> {
        // noop
        Ok(())
    }

    fn remove_block_bodies_above(
        &self,
        _provider: &Provider,
        _block: BlockNumber,
        _remove_from: StorageLocation,
    ) -> ProviderResult<()> {
        // noop
        Ok(())
    }
}

impl<Provider, T, H> BlockBodyReader<Provider> for OpStorage<T, H>
where
    Provider: ChainSpecProvider<ChainSpec: EthChainSpec + EthereumHardforks> + DBProvider,
    T: SignedTransaction,
    H: FullBlockHeader,
{
    type Block = alloy_consensus::Block<T, H>;

    fn read_block_bodies(
        &self,
        provider: &Provider,
        inputs: Vec<ReadBodyInput<'_, Self::Block>>,
    ) -> ProviderResult<Vec<<Self::Block as Block>::Body>> {
        let chain_spec = provider.chain_spec();

        let mut bodies = Vec::with_capacity(inputs.len());

        for (header, transactions) in inputs {
            let mut withdrawals = None;
            if chain_spec.is_shanghai_active_at_timestamp(header.timestamp()) {
                // after shanghai the body should have an empty withdrawals list
                withdrawals.replace(vec![].into());
            }

            bodies.push(alloy_consensus::BlockBody::<T, H> {
                transactions,
                ommers: vec![],
                withdrawals,
            });
        }

        Ok(bodies)
    }
}
