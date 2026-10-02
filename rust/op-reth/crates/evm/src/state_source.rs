//! Adapts reth's provider contract without imposing Send on worker-owned readers.
use alloy_op_evm::block::{ParallelConfigurationError as Error, StateReadFactory};
use alloy_primitives::{Address, B256, U256};
use reth_evm::state_source::ExecutionStateSource;
use revm::{
    Database,
    state::{AccountInfo, Bytecode},
};

pub(crate) struct SourceAdapter(pub(crate) ExecutionStateSource);
struct Reader(Box<dyn Database<Error = reth_storage_errors::provider::ProviderError>>);

impl StateReadFactory for SourceAdapter {
    fn open(&self) -> Result<Box<dyn Database<Error = Error>>, Error> {
        self.0
            .0
            .open()
            .map(|reader| Box::new(Reader(reader)) as Box<dyn Database<Error = Error>>)
            .map_err(|error| Error::Source(error.to_string()))
    }
}

impl Database for Reader {
    type Error = Error;
    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Error> {
        self.0.basic(address).map_err(|error| Error::Source(error.to_string()))
    }
    fn storage(&mut self, address: Address, index: U256) -> Result<U256, Error> {
        self.0.storage(address, index).map_err(|error| Error::Source(error.to_string()))
    }
    fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Error> {
        self.0.code_by_hash(hash).map_err(|error| Error::Source(error.to_string()))
    }
    fn block_hash(&mut self, number: u64) -> Result<B256, Error> {
        self.0.block_hash(number).map_err(|error| Error::Source(error.to_string()))
    }
}
