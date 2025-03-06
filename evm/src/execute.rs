//! Optimism block execution strategy.

use crate::{
    l1::ensure_create2_deployer, BasicOpReceiptBuilder, OpBlockAssembler, OpBlockExecutionError,
    OpEvmConfig, OpReceiptBuilder, ReceiptBuilderCtx,
};
use alloc::{boxed::Box, sync::Arc, vec::Vec};
use alloy_consensus::{
    transaction::Recovered, BlockHeader, Eip658Value, Header, Receipt, Transaction, TxReceipt,
};
use alloy_eips::Encodable2718;
use alloy_evm::FromRecoveredTx;
use alloy_primitives::Bytes;
use op_alloy_consensus::OpDepositReceipt;
use reth_chainspec::EthChainSpec;
use reth_evm::{
    execute::{
        balance_increment_state, BasicBlockExecutorProvider, BlockExecutionError,
        BlockExecutionStrategy, BlockExecutionStrategyFactory, BlockValidationError,
    },
    state_change::post_block_balance_increments,
    system_calls::{OnStateHook, StateChangePostBlockSource, StateChangeSource, SystemCaller},
    Database, Evm, EvmFor, InspectorFor,
};
use reth_execution_types::BlockExecutionResult;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::{transaction::signed::OpTransaction, DepositReceipt};
use reth_primitives_traits::{NodePrimitives, SealedBlock, SealedHeader, SignedTransaction};
use revm::{context::TxEnv, context_interface::result::ResultAndState, DatabaseCommit};
use revm_database::State;
use revm_primitives::B256;
use tracing::trace;

impl<ChainSpec, N, T> BlockExecutionStrategyFactory for OpEvmConfig<ChainSpec, N>
where
    ChainSpec: EthChainSpec + OpHardforks + 'static,
    T: SignedTransaction + OpTransaction,
    N: NodePrimitives<
        Receipt: DepositReceipt,
        SignedTx = T,
        BlockHeader = Header,
        BlockBody = alloy_consensus::BlockBody<T>,
    >,
    op_revm::OpTransaction<TxEnv>: FromRecoveredTx<T>,
{
    type Primitives = N;
    type Strategy<'a, DB: Database + 'a, I: InspectorFor<&'a mut State<DB>, Self> + 'a> =
        OpExecutionStrategy<
            'a,
            EvmFor<Self, &'a mut State<DB>, I>,
            N::SignedTx,
            N::Receipt,
            &'a ChainSpec,
        >;
    type ExecutionCtx<'a> = OpBlockExecutionCtx;
    type BlockAssembler = OpBlockAssembler<ChainSpec>;

    fn block_assembler(&self) -> &Self::BlockAssembler {
        &self.block_assembler
    }

    fn context_for_block<'a>(&self, block: &'a SealedBlock<N::Block>) -> Self::ExecutionCtx<'a> {
        OpBlockExecutionCtx {
            parent_hash: block.header().parent_hash(),
            parent_beacon_block_root: block.header().parent_beacon_block_root(),
            extra_data: block.header().extra_data().clone(),
        }
    }

    fn context_for_next_block(
        &self,
        parent: &SealedHeader<N::BlockHeader>,
        attributes: Self::NextBlockEnvCtx,
    ) -> Self::ExecutionCtx<'_> {
        OpBlockExecutionCtx {
            parent_hash: parent.hash(),
            parent_beacon_block_root: attributes.parent_beacon_block_root,
            extra_data: attributes.extra_data,
        }
    }

    fn create_strategy<'a, DB, I>(
        &'a self,
        evm: EvmFor<Self, &'a mut State<DB>, I>,
        ctx: Self::ExecutionCtx<'a>,
    ) -> Self::Strategy<'a, DB, I>
    where
        DB: Database,
        I: reth_evm::InspectorFor<&'a mut State<DB>, Self> + 'a,
    {
        OpExecutionStrategy::new(evm, ctx, self.chain_spec.as_ref(), self.receipt_builder.as_ref())
    }
}

/// Context for OP block execution.
#[derive(Debug, Clone)]
pub struct OpBlockExecutionCtx {
    /// Parent block hash.
    pub parent_hash: B256,
    /// Parent beacon block root.
    pub parent_beacon_block_root: Option<B256>,
    /// The block's extra data.
    pub extra_data: Bytes,
}

/// Block execution strategy for Optimism.
#[derive(Debug)]
pub struct OpExecutionStrategy<'a, E: Evm, Tx, R, ChainSpec> {
    /// Chainspec.
    chain_spec: ChainSpec,
    /// Receipt builder.
    receipt_builder: &'a dyn OpReceiptBuilder<Tx, E::HaltReason, Receipt = R>,

    /// Context for block execution.
    ctx: OpBlockExecutionCtx,
    /// The EVM used by strategy.
    evm: E,
    /// Receipts of executed transactions.
    receipts: Vec<R>,
    /// Total gas used by executed transactions.
    gas_used: u64,
    /// Whether Regolith hardfork is active.
    is_regolith: bool,
    /// Utility to call system smart contracts.
    system_caller: SystemCaller<ChainSpec>,
}

impl<'a, E, Tx, R, ChainSpec> OpExecutionStrategy<'a, E, Tx, R, ChainSpec>
where
    E: Evm,
    ChainSpec: OpHardforks + Clone,
{
    /// Creates a new [`OpExecutionStrategy`]
    pub fn new(
        evm: E,
        ctx: OpBlockExecutionCtx,
        chain_spec: ChainSpec,
        receipt_builder: &'a dyn OpReceiptBuilder<Tx, E::HaltReason, Receipt = R>,
    ) -> Self {
        Self {
            is_regolith: chain_spec.is_regolith_active_at_timestamp(evm.block().timestamp),
            evm,
            system_caller: SystemCaller::new(chain_spec.clone()),
            chain_spec,
            receipt_builder,
            receipts: Vec::new(),
            gas_used: 0,
            ctx,
        }
    }
}

impl<'db, DB, E, Tx, R, ChainSpec> BlockExecutionStrategy
    for OpExecutionStrategy<'_, E, Tx, R, ChainSpec>
where
    DB: Database + 'db,
    Tx: Transaction + OpTransaction + Encodable2718,
    R: TxReceipt + Unpin + 'static,
    E: Evm<DB = &'db mut State<DB>, Tx: FromRecoveredTx<Tx>>,
    ChainSpec: OpHardforks,
{
    type Transaction = Tx;
    type Receipt = R;
    type Evm = E;

    fn apply_pre_execution_changes(&mut self) -> Result<(), BlockExecutionError> {
        // Set state clear flag if the block is after the Spurious Dragon hardfork.
        let state_clear_flag =
            self.chain_spec.is_spurious_dragon_active_at_block(self.evm.block().number);
        self.evm.db_mut().set_state_clear_flag(state_clear_flag);

        self.system_caller.apply_blockhashes_contract_call(self.ctx.parent_hash, &mut self.evm)?;
        self.system_caller
            .apply_beacon_root_contract_call(self.ctx.parent_beacon_block_root, &mut self.evm)?;

        // Ensure that the create2deployer is force-deployed at the canyon transition. Optimism
        // blocks will always have at least a single transaction in them (the L1 info transaction),
        // so we can safely assume that this will always be triggered upon the transition and that
        // the above check for empty blocks will never be hit on OP chains.
        ensure_create2_deployer(&self.chain_spec, self.evm.block().timestamp, self.evm.db_mut())
            .map_err(|_| OpBlockExecutionError::ForceCreate2DeployerFail)?;

        Ok(())
    }

    fn execute_transaction_with_result_closure(
        &mut self,
        tx: Recovered<&Tx>,
        f: impl FnOnce(&revm::context::result::ExecutionResult<<Self::Evm as Evm>::HaltReason>),
    ) -> Result<u64, BlockExecutionError> {
        // The sum of the transaction’s gas limit, Tg, and the gas utilized in this block prior,
        // must be no greater than the block’s gasLimit.
        let block_available_gas = self.evm.block().gas_limit - self.gas_used;
        if tx.gas_limit() > block_available_gas && (self.is_regolith || !tx.is_deposit()) {
            return Err(BlockValidationError::TransactionGasLimitMoreThanAvailableBlockGas {
                transaction_gas_limit: tx.gas_limit(),
                block_available_gas,
            }
            .into())
        }

        // Cache the depositor account prior to the state transition for the deposit nonce.
        //
        // Note that this *only* needs to be done post-regolith hardfork, as deposit nonces
        // were not introduced in Bedrock. In addition, regular transactions don't have deposit
        // nonces, so we don't need to touch the DB for those.
        let depositor = (self.is_regolith && tx.is_deposit())
            .then(|| {
                self.evm
                    .db_mut()
                    .load_cache_account(tx.signer())
                    .map(|acc| acc.account_info().unwrap_or_default())
            })
            .transpose()
            .map_err(|_| OpBlockExecutionError::AccountLoadFailed(tx.signer()))?;

        let hash = tx.trie_hash();

        // Execute transaction.
        let result_and_state =
            self.evm.transact(&tx).map_err(move |err| BlockExecutionError::evm(err, hash))?;

        trace!(
            target: "evm",
            ?tx,
            "Executed transaction"
        );
        self.system_caller
            .on_state(StateChangeSource::Transaction(self.receipts.len()), &result_and_state.state);
        let ResultAndState { result, state } = result_and_state;
        self.evm.db_mut().commit(state);

        f(&result);

        let gas_used = result.gas_used();

        // append gas used
        self.gas_used += gas_used;

        self.receipts.push(
            match self.receipt_builder.build_receipt(ReceiptBuilderCtx {
                tx: tx.tx(),
                result,
                cumulative_gas_used: self.gas_used,
            }) {
                Ok(receipt) => receipt,
                Err(ctx) => {
                    let receipt = Receipt {
                        // Success flag was added in `EIP-658: Embedding transaction status code
                        // in receipts`.
                        status: Eip658Value::Eip658(ctx.result.is_success()),
                        cumulative_gas_used: self.gas_used,
                        logs: ctx.result.into_logs(),
                    };

                    self.receipt_builder.build_deposit_receipt(OpDepositReceipt {
                        inner: receipt,
                        deposit_nonce: depositor.map(|account| account.nonce),
                        // The deposit receipt version was introduced in Canyon to indicate an
                        // update to how receipt hashes should be computed
                        // when set. The state transition process ensures
                        // this is only set for post-Canyon deposit
                        // transactions.
                        deposit_receipt_version: (tx.is_deposit() &&
                            self.chain_spec
                                .is_canyon_active_at_timestamp(self.evm.block().timestamp))
                        .then_some(1),
                    })
                }
            },
        );

        Ok(gas_used)
    }

    fn finish(mut self) -> Result<(Self::Evm, BlockExecutionResult<R>), BlockExecutionError> {
        let balance_increments =
            post_block_balance_increments::<Header>(&self.chain_spec, self.evm.block(), &[], None);
        // increment balances
        self.evm
            .db_mut()
            .increment_balances(balance_increments.clone())
            .map_err(|_| BlockValidationError::IncrementBalanceFailed)?;
        // call state hook with changes due to balance increments.
        let balance_state = balance_increment_state(&balance_increments, self.evm.db_mut())?;
        self.system_caller.on_state(
            StateChangeSource::PostBlock(StateChangePostBlockSource::BalanceIncrements),
            &balance_state,
        );

        let gas_used = self.receipts.last().map(|r| r.cumulative_gas_used()).unwrap_or_default();
        Ok((
            self.evm,
            BlockExecutionResult {
                receipts: self.receipts,
                requests: Default::default(),
                gas_used,
            },
        ))
    }

    fn with_state_hook(&mut self, hook: Option<Box<dyn OnStateHook>>) {
        self.system_caller.with_state_hook(hook);
    }

    fn evm_mut(&mut self) -> &mut Self::Evm {
        &mut self.evm
    }
}

/// Helper type with backwards compatible methods to obtain executor providers.
#[derive(Debug)]
pub struct OpExecutorProvider;

impl OpExecutorProvider {
    /// Creates a new default optimism executor strategy factory.
    pub fn optimism(chain_spec: Arc<OpChainSpec>) -> BasicBlockExecutorProvider<OpEvmConfig> {
        BasicBlockExecutorProvider::new(OpEvmConfig::new(
            chain_spec,
            BasicOpReceiptBuilder::default(),
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpChainSpec;
    use alloy_consensus::{Block, BlockBody, Header, TxEip1559};
    use alloy_primitives::{
        b256, Address, PrimitiveSignature as Signature, StorageKey, StorageValue, U256,
    };
    use op_alloy_consensus::{OpTypedTransaction, TxDeposit};
    use op_revm::constants::L1_BLOCK_CONTRACT;
    use reth_chainspec::MIN_TRANSACTION_GAS;
    use reth_evm::execute::{BasicBlockExecutorProvider, BlockExecutorProvider, Executor};
    use reth_optimism_chainspec::OpChainSpecBuilder;
    use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
    use reth_primitives_traits::{Account, RecoveredBlock};
    use reth_revm::{database::StateProviderDatabase, test_utils::StateProviderTest};
    use std::{collections::HashMap, str::FromStr};

    fn create_op_state_provider() -> StateProviderTest {
        let mut db = StateProviderTest::default();

        let l1_block_contract_account =
            Account { balance: U256::ZERO, bytecode_hash: None, nonce: 1 };

        let mut l1_block_storage = HashMap::default();
        // base fee
        l1_block_storage.insert(StorageKey::with_last_byte(1), StorageValue::from(1000000000));
        // l1 fee overhead
        l1_block_storage.insert(StorageKey::with_last_byte(5), StorageValue::from(188));
        // l1 fee scalar
        l1_block_storage.insert(StorageKey::with_last_byte(6), StorageValue::from(684000));
        // l1 free scalars post ecotone
        l1_block_storage.insert(
            StorageKey::with_last_byte(3),
            StorageValue::from_str(
                "0x0000000000000000000000000000000000001db0000d27300000000000000005",
            )
            .unwrap(),
        );

        db.insert_account(L1_BLOCK_CONTRACT, l1_block_contract_account, None, l1_block_storage);

        db
    }

    fn executor_provider(chain_spec: Arc<OpChainSpec>) -> BasicBlockExecutorProvider<OpEvmConfig> {
        BasicBlockExecutorProvider::new(OpEvmConfig::new(
            chain_spec,
            BasicOpReceiptBuilder::default(),
        ))
    }

    #[test]
    fn op_deposit_fields_pre_canyon() {
        let header = Header {
            timestamp: 1,
            number: 1,
            gas_limit: 1_000_000,
            gas_used: 42_000,
            receipts_root: b256!(
                "83465d1e7d01578c0d609be33570f91242f013e9e295b0879905346abbd63731"
            ),
            ..Default::default()
        };

        let mut db = create_op_state_provider();

        let addr = Address::ZERO;
        let account = Account { balance: U256::MAX, ..Account::default() };
        db.insert_account(addr, account, None, HashMap::default());

        let chain_spec = Arc::new(OpChainSpecBuilder::base_mainnet().regolith_activated().build());

        let tx = OpTransactionSigned::new_unhashed(
            OpTypedTransaction::Eip1559(TxEip1559 {
                chain_id: chain_spec.chain.id(),
                nonce: 0,
                gas_limit: MIN_TRANSACTION_GAS,
                to: addr.into(),
                ..Default::default()
            }),
            Signature::test_signature(),
        );

        let tx_deposit = OpTransactionSigned::new_unhashed(
            OpTypedTransaction::Deposit(op_alloy_consensus::TxDeposit {
                from: addr,
                to: addr.into(),
                gas_limit: MIN_TRANSACTION_GAS,
                ..Default::default()
            }),
            Signature::test_signature(),
        );

        let provider = executor_provider(chain_spec);
        let mut executor = provider.executor(StateProviderDatabase::new(&db));

        // make sure the L1 block contract state is preloaded.
        executor.with_state_mut(|state| {
            state.load_cache_account(L1_BLOCK_CONTRACT).unwrap();
        });

        // Attempt to execute a block with one deposit and one non-deposit transaction
        let output = executor
            .execute(&RecoveredBlock::new_unhashed(
                Block {
                    header,
                    body: BlockBody { transactions: vec![tx, tx_deposit], ..Default::default() },
                },
                vec![addr, addr],
            ))
            .unwrap();

        let receipts = &output.receipts;
        let tx_receipt = &receipts[0];
        let deposit_receipt = &receipts[1];

        assert!(!matches!(tx_receipt, OpReceipt::Deposit(_)));
        // deposit_nonce is present only in deposit transactions
        let OpReceipt::Deposit(deposit_receipt) = deposit_receipt else {
            panic!("expected deposit")
        };
        assert!(deposit_receipt.deposit_nonce.is_some());
        // deposit_receipt_version is not present in pre canyon transactions
        assert!(deposit_receipt.deposit_receipt_version.is_none());
    }

    #[test]
    fn op_deposit_fields_post_canyon() {
        // ensure_create2_deployer will fail if timestamp is set to less than 2
        let header = Header {
            timestamp: 2,
            number: 1,
            gas_limit: 1_000_000,
            gas_used: 42_000,
            receipts_root: b256!(
                "fffc85c4004fd03c7bfbe5491fae98a7473126c099ac11e8286fd0013f15f908"
            ),
            ..Default::default()
        };

        let mut db = create_op_state_provider();
        let addr = Address::ZERO;
        let account = Account { balance: U256::MAX, ..Account::default() };

        db.insert_account(addr, account, None, HashMap::default());

        let chain_spec = Arc::new(OpChainSpecBuilder::base_mainnet().canyon_activated().build());

        let tx = OpTransactionSigned::new_unhashed(
            OpTypedTransaction::Eip1559(TxEip1559 {
                chain_id: chain_spec.chain.id(),
                nonce: 0,
                gas_limit: MIN_TRANSACTION_GAS,
                to: addr.into(),
                ..Default::default()
            }),
            Signature::test_signature(),
        );

        let tx_deposit = OpTransactionSigned::new_unhashed(
            OpTypedTransaction::Deposit(op_alloy_consensus::TxDeposit {
                from: addr,
                to: addr.into(),
                gas_limit: MIN_TRANSACTION_GAS,
                ..Default::default()
            }),
            TxDeposit::signature(),
        );

        let provider = executor_provider(chain_spec);
        let mut executor = provider.executor(StateProviderDatabase::new(&db));

        // make sure the L1 block contract state is preloaded.
        executor.with_state_mut(|state| {
            state.load_cache_account(L1_BLOCK_CONTRACT).unwrap();
        });

        // attempt to execute an empty block with parent beacon block root, this should not fail
        let output = executor
            .execute(&RecoveredBlock::new_unhashed(
                Block {
                    header,
                    body: BlockBody { transactions: vec![tx, tx_deposit], ..Default::default() },
                },
                vec![addr, addr],
            ))
            .expect("Executing a block while canyon is active should not fail");

        let receipts = &output.receipts;
        let tx_receipt = &receipts[0];
        let deposit_receipt = &receipts[1];

        // deposit_receipt_version is set to 1 for post canyon deposit transactions
        assert!(!matches!(tx_receipt, OpReceipt::Deposit(_)));
        let OpReceipt::Deposit(deposit_receipt) = deposit_receipt else {
            panic!("expected deposit")
        };
        assert_eq!(deposit_receipt.deposit_receipt_version, Some(1));

        // deposit_nonce is present only in deposit transactions
        assert!(deposit_receipt.deposit_nonce.is_some());
    }
}
