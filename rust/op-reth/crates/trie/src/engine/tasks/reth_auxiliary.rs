//! Bytecode and block-hash reads without constructing reth's historical state provider.

use alloy_primitives::{B256, BlockNumber};
use reth_db::{tables, transaction::DbTx};
use reth_primitives_traits::Bytecode;
use reth_provider::{
    BlockHashReader, BytecodeReader, DBProvider, DatabaseProviderFactory, ProviderResult,
    StateReader,
};

pub(super) struct RethAuxiliaryProvider<'a, P> {
    provider: &'a P,
    parent: BlockNumber,
}

impl<'a, P> RethAuxiliaryProvider<'a, P> {
    pub(super) const fn new(provider: &'a P, parent: BlockNumber) -> Self {
        Self { provider, parent }
    }
}

impl<P: BlockHashReader> BlockHashReader for RethAuxiliaryProvider<'_, P> {
    fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>> {
        self.provider.block_hash(number)
    }

    fn canonical_hashes_range(
        &self,
        start: BlockNumber,
        end: BlockNumber,
    ) -> ProviderResult<Vec<B256>> {
        self.provider.canonical_hashes_range(start, end)
    }
}

impl<P: DatabaseProviderFactory + StateReader> BytecodeReader for RethAuxiliaryProvider<'_, P> {
    fn bytecode_by_hash(&self, code_hash: &B256) -> ProviderResult<Option<Bytecode>> {
        let persisted_tip = {
            let provider = self.provider.database_provider_ro()?;
            if let Some(code) = provider.tx_ref().get::<tables::Bytecodes>(*code_hash)? {
                return Ok(Some(code));
            }
            // Bytecode is persisted with execution data, independently of the trie frontier.
            provider
                .tx_ref()
                .get::<tables::StageCheckpoints>("Finish".into())?
                .map_or(0, |checkpoint| checkpoint.block_number)
        };

        // New deployments may still exist only in the canonical execution buffer. The bytecode
        // hash comes from the proofs DB's parent account, so this lookup does not depend on reth's
        // account/storage/trie state. Only search the unpersisted suffix. Concurrent reorgs may
        // change which outcomes get_state returns, but cannot change the bytes for this code hash.
        let mut read_error = None;
        for number in (persisted_tip.saturating_add(1)..=self.parent).rev() {
            match self.provider.get_state(number) {
                Ok(Some(outcome)) => {
                    if let Some(code) = outcome.bytecode(code_hash) {
                        return Ok(Some(code));
                    }
                }
                Ok(None) => {}
                Err(error) => {
                    read_error = Some(error);
                    break;
                }
            }
        }

        // Persistence can remove an execution result from memory after the first DB read. Open
        // a fresh transaction to cover that handoff, including a failed execution-result lookup.
        let provider = self.provider.database_provider_ro()?;
        let code = provider.tx_ref().get::<tables::Bytecodes>(*code_hash)?;
        if code.is_none() &&
            let Some(error) = read_error
        {
            return Err(error);
        }
        Ok(code)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Bytes, keccak256};
    use reth_db::transaction::DbTxMut;
    use reth_ethereum_primitives::Receipt;
    use reth_provider::{
        ExecutionOutcome, ProviderError, test_utils::create_test_provider_factory,
    };
    use std::sync::atomic::{AtomicUsize, Ordering};

    // Deliberately implements no StateProviderFactory: bytecode lookup must not require one.
    struct Reader<F> {
        factory: F,
        outcome: ExecutionOutcome<Receipt>,
        calls: AtomicUsize,
        persist_on_read: bool,
    }

    impl<F: DatabaseProviderFactory> DatabaseProviderFactory for Reader<F> {
        type DB = F::DB;
        type Provider = F::Provider;
        type ProviderRW = F::ProviderRW;

        fn database_provider_ro(&self) -> ProviderResult<Self::Provider> {
            self.factory.database_provider_ro()
        }

        fn database_provider_rw(&self) -> ProviderResult<Self::ProviderRW> {
            self.factory.database_provider_rw()
        }
    }

    impl<F: DatabaseProviderFactory> StateReader for Reader<F> {
        type Receipt = Receipt;

        fn get_state(
            &self,
            number: BlockNumber,
        ) -> ProviderResult<Option<ExecutionOutcome<Receipt>>> {
            self.calls.fetch_add(1, Ordering::Relaxed);
            if self.persist_on_read {
                // Simulate persistence clearing the in-memory result between the two reads.
                let provider = self.factory.database_provider_rw()?;
                for (hash, code) in &self.outcome.bundle.contracts {
                    provider.tx_ref().put::<tables::Bytecodes>(*hash, Bytecode(code.clone()))?;
                }
                provider.commit()?;
                return Err(ProviderError::StateForNumberNotFound(number));
            }
            Ok((number == self.outcome.first_block).then(|| self.outcome.clone()))
        }
    }

    #[test]
    fn deployed_bytecode_available_without_parent_state() {
        let code = Bytecode::new_raw(Bytes::from_static(&[0x60, 0x01, 0x60, 0x00, 0x55, 0x00]));
        let hash = keccak256(code.original_bytes());
        let mut outcome = ExecutionOutcome::<Receipt> { first_block: 99, ..Default::default() };
        outcome.bundle.contracts.insert(hash, code.0.clone());
        let reader = Reader {
            factory: create_test_provider_factory(),
            outcome,
            calls: AtomicUsize::new(0),
            persist_on_read: false,
        };

        // Downloaded headers can be ahead of execution persistence during pipeline sync.
        let provider = reader.database_provider_rw().unwrap();
        provider.tx_ref().put::<tables::CanonicalHeaders>(1000, B256::repeat_byte(1)).unwrap();
        provider.commit().unwrap();

        let auxiliary = RethAuxiliaryProvider::new(&reader, 99);
        assert_eq!(auxiliary.bytecode_by_hash(&hash).unwrap(), Some(code.clone()));
        assert_eq!(reader.calls.load(Ordering::Relaxed), 1);

        // Execution has reached disk while the trie frontier still lags: no memory read needed.
        let provider = reader.database_provider_rw().unwrap();
        provider.tx_ref().put::<tables::Bytecodes>(hash, code.clone()).unwrap();
        provider.commit().unwrap();
        assert_eq!(auxiliary.bytecode_by_hash(&hash).unwrap(), Some(code));
        assert_eq!(reader.calls.load(Ordering::Relaxed), 1);
    }

    #[test]
    fn unavailable_execution_data_propagates_error() {
        let reader = Reader {
            factory: create_test_provider_factory(),
            outcome: ExecutionOutcome::<Receipt>::default(),
            calls: AtomicUsize::new(0),
            persist_on_read: true,
        };
        assert!(matches!(
            RethAuxiliaryProvider::new(&reader, 99).bytecode_by_hash(&B256::repeat_byte(1)),
            Err(ProviderError::StateForNumberNotFound(99))
        ));
    }

    #[test]
    fn bytecode_lookup_covers_persistence_handoff() {
        let code = Bytecode::new_raw(Bytes::from_static(&[0x00]));
        let hash = keccak256(code.original_bytes());
        let mut outcome = ExecutionOutcome::<Receipt> { first_block: 99, ..Default::default() };
        outcome.bundle.contracts.insert(hash, code.0.clone());
        let reader = Reader {
            factory: create_test_provider_factory(),
            outcome,
            calls: AtomicUsize::new(0),
            persist_on_read: true,
        };
        assert_eq!(
            RethAuxiliaryProvider::new(&reader, 99).bytecode_by_hash(&hash).unwrap(),
            Some(code)
        );
    }
}
