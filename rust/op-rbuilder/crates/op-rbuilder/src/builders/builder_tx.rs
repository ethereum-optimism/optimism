use alloy_consensus::{TxEip1559, transaction::Recovered};
use alloy_eips::{Encodable2718, eip7623::TOTAL_COST_FLOOR_PER_TOKEN};
use alloy_evm::{
    Database,
    block::{BlockExecutor as AlloyBlockExecutor, CommitChanges, TxResult},
    rpc::TryIntoTxEnv,
};
use alloy_op_evm::{OpEvm, OpTx};
use alloy_primitives::{
    Address, B256, Bytes, TxKind, U256,
    map::{AddressMap, HashSet},
};
use alloy_sol_types::{ContractError, Revert, SolCall, SolError, SolInterface};
use core::fmt::Debug;
use op_alloy_consensus::OpTypedTransaction;
use op_alloy_rpc_types::OpTransactionRequest;
use op_revm::{OpHaltReason, OpTransactionError};
use reth_evm::{
    ConfigureEvm, Evm, EvmError,
    execute::{BlockBuilder, BlockExecutionError, BlockValidationError},
    precompiles::PrecompilesMap,
};
use reth_node_api::PayloadBuilderError;
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
use reth_provider::{ProviderError, StateProvider};
use reth_revm::{State, database::StateProviderDatabase};
use reth_rpc_api::eth::EthTxEnvError;
use revm::{
    DatabaseCommit, DatabaseRef,
    context::result::{EVMError, ExecutionResult, ResultAndState},
    inspector::NoOpInspector,
    state::Account,
};
use tracing::{trace, warn};

use crate::{
    builders::context::{OpPayloadBuilderCtx, last_receipt_with_cumulative_gas},
    primitives::reth::ExecutionInfo,
    tx_signer::Signer,
};

#[derive(Debug, Default)]
pub struct SimulationSuccessResult<T: SolCall> {
    pub gas_used: u64,
    pub output: T::Return,
    pub state_changes: AddressMap<Account>,
}

#[derive(Debug, Clone)]
pub struct BuilderTransactionCtx {
    pub gas_used: u64,
    pub da_size: u64,
    pub signed_tx: Recovered<OpTransactionSigned>,
    // whether the transaction should be a top of block or
    // bottom of block transaction
    pub is_top_of_block: bool,
}

impl BuilderTransactionCtx {
    pub fn set_top_of_block(mut self) -> Self {
        self.is_top_of_block = true;
        self
    }

    pub fn set_bottom_of_block(mut self) -> Self {
        self.is_top_of_block = false;
        self
    }
}

#[derive(Debug, thiserror::Error)]
pub enum InvalidContractDataError {
    #[error("did not find expected logs expected {0:?} but got {1:?}")]
    InvalidLogs(Vec<B256>, Vec<B256>),
    #[error("could not decode output from contract call")]
    OutputAbiDecodeError,
}

/// Possible error variants during construction of builder txs.
#[derive(Debug, thiserror::Error)]
pub enum BuilderTransactionError {
    /// Builder account load fails to get builder nonce
    #[error("failed to load account {0}")]
    AccountLoadFailed(Address),
    /// Signature signing fails
    #[error("failed to sign transaction: {0}")]
    SigningError(secp256k1::Error),
    /// Invalid contract errors indicating the contract is incorrect
    #[error("contract {0} may be incorrect, invalid contract data: {1}")]
    InvalidContract(Address, InvalidContractDataError),
    /// Transaction halted execution
    #[error("transaction to {0} halted {1:?}")]
    TransactionHalted(Address, OpHaltReason),
    /// Transaction reverted
    #[error("transaction to {0} reverted {1}")]
    TransactionReverted(Address, Revert),
    /// Invalid tx errors during evm execution.
    #[error("invalid transaction error {0}")]
    InvalidTransactionError(Box<dyn core::error::Error + Send + Sync>),
    /// Unrecoverable error during evm execution.
    #[error("evm execution error {0}")]
    EvmExecutionError(Box<dyn core::error::Error + Send + Sync>),
    /// Any other builder transaction errors.
    #[error(transparent)]
    Other(Box<dyn core::error::Error + Send + Sync>),
}

impl From<secp256k1::Error> for BuilderTransactionError {
    fn from(error: secp256k1::Error) -> Self {
        BuilderTransactionError::SigningError(error)
    }
}

impl From<EVMError<ProviderError, OpTransactionError>> for BuilderTransactionError {
    fn from(error: EVMError<ProviderError, OpTransactionError>) -> Self {
        BuilderTransactionError::EvmExecutionError(Box::new(error))
    }
}

impl From<EthTxEnvError> for BuilderTransactionError {
    fn from(error: EthTxEnvError) -> Self {
        BuilderTransactionError::EvmExecutionError(Box::new(error))
    }
}

impl From<BuilderTransactionError> for PayloadBuilderError {
    fn from(error: BuilderTransactionError) -> Self {
        match error {
            BuilderTransactionError::EvmExecutionError(e) => {
                PayloadBuilderError::EvmExecutionError(e)
            }
            _ => PayloadBuilderError::other(error),
        }
    }
}

impl BuilderTransactionError {
    pub fn other(error: impl core::error::Error + Send + Sync + 'static) -> Self {
        BuilderTransactionError::Other(Box::new(error))
    }

    pub fn msg(msg: impl core::fmt::Display) -> Self {
        Self::Other(msg.to_string().into())
    }
}

pub trait BuilderTransactions<ExtraCtx: Debug + Default = (), Extra: Debug + Default = ()> {
    // Simulates and returns the signed builder transactions. The simulation modifies and commit
    // changes to the db so call new_simulation_state to simulate on a new copy of the state
    fn simulate_builder_txs(
        &self,
        state_provider: impl StateProvider + Clone,
        info: &mut ExecutionInfo<Extra>,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        db: &mut State<impl Database + DatabaseRef>,
        top_of_block: bool,
    ) -> Result<Vec<BuilderTransactionCtx>, BuilderTransactionError>;

    fn simulate_builder_txs_with_state_copy(
        &self,
        state_provider: impl StateProvider + Clone,
        info: &mut ExecutionInfo<Extra>,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        db: &State<impl Database>,
        top_of_block: bool,
    ) -> Result<Vec<BuilderTransactionCtx>, BuilderTransactionError> {
        let mut simulation_state = self.new_simulation_state(state_provider.clone(), db);
        self.simulate_builder_txs(
            state_provider,
            info,
            ctx,
            &mut simulation_state,
            top_of_block,
        )
    }

    fn add_builder_txs<Builder, DB>(
        &self,
        state_provider: impl StateProvider + Clone,
        info: &mut ExecutionInfo<Extra>,
        builder_ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        builder: &mut Builder,
        top_of_block: bool,
    ) -> Result<Vec<BuilderTransactionCtx>, BuilderTransactionError>
    where
        Builder: BlockBuilder<Primitives = reth_optimism_primitives::OpPrimitives>,
        Builder::Executor: AlloyBlockExecutor<
                Transaction = OpTransactionSigned,
                Receipt = OpReceipt,
                Evm: alloy_evm::Evm<DB: core::ops::Deref<Target = State<DB>>>,
            >,
        DB: Database,
    {
        let builder_txs = self.simulate_builder_txs_with_state_copy(
            state_provider,
            info,
            builder_ctx,
            builder.evm().db(),
            top_of_block,
        )?;

        let mut invalid = HashSet::new();

        for builder_tx in builder_txs.iter() {
            if builder_tx.is_top_of_block != top_of_block {
                // don't commit tx if the builder tx is not being added in the intended
                // position in the block
                continue;
            }
            if invalid.contains(&builder_tx.signed_tx.signer()) {
                warn!(target: "payload_builder", tx_hash = ?builder_tx.signed_tx.tx_hash(), "builder signer invalid as previous builder tx reverted");
                continue;
            }

            let mut gas_used = 0;
            let committed = match builder.execute_transaction_with_commit_condition(
                builder_tx.signed_tx.clone(),
                |result| {
                    gas_used = result.result().result.tx_gas_used();
                    if result.result().result.is_success() {
                        CommitChanges::Yes
                    } else {
                        warn!(target: "payload_builder", tx_hash = ?builder_tx.signed_tx.tx_hash(), "builder tx reverted");
                        invalid.insert(builder_tx.signed_tx.signer());
                        CommitChanges::No
                    }
                },
            ) {
                Ok(committed) => committed,
                Err(BlockExecutionError::Validation(BlockValidationError::InvalidTx {
                    error,
                    ..
                })) => {
                    if error.is_nonce_too_low() {
                        // if the nonce is too low, we can skip this transaction
                        trace!(target: "payload_builder", %error, ?builder_tx.signed_tx, "skipping nonce too low builder transaction");
                    } else {
                        // if the transaction is invalid, we can skip it and all of its
                        // descendants
                        trace!(target: "payload_builder", %error, ?builder_tx.signed_tx, "skipping invalid builder transaction and its descendants");
                        invalid.insert(builder_tx.signed_tx.signer());
                    }

                    continue;
                }
                Err(err) => {
                    // this is an error that we should treat as fatal for this attempt
                    return Err(BuilderTransactionError::EvmExecutionError(Box::new(err)));
                }
            };

            if committed.is_none() {
                continue;
            }

            info.cumulative_gas_used += gas_used;
            info.cumulative_da_bytes_used += builder_tx.da_size;
            info.receipts.push(
                last_receipt_with_cumulative_gas(builder.executor(), info.cumulative_gas_used)
                    .expect("executor must record a receipt for committed tx"),
            );

            // Append sender and transaction to the respective lists
            info.executed_senders.push(builder_tx.signed_tx.signer());
            info.executed_transactions
                .push(builder_tx.signed_tx.clone().into_inner());
        }

        Ok(builder_txs)
    }

    // Creates a copy of the state to simulate against
    fn new_simulation_state(
        &self,
        state_provider: impl StateProvider,
        db: &State<impl Database>,
    ) -> State<StateProviderDatabase<impl StateProvider>> {
        let state = StateProviderDatabase::new(state_provider);

        State::builder()
            .with_database(state)
            .with_cached_prestate(db.cache.clone())
            .with_bundle_update()
            .build()
    }

    fn sign_tx(
        &self,
        to: Address,
        from: Signer,
        gas_used: u64,
        calldata: Bytes,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        db: impl DatabaseRef,
    ) -> Result<Recovered<OpTransactionSigned>, BuilderTransactionError> {
        let nonce = get_nonce(db, from.address)?;
        // Create the EIP-1559 transaction
        let tx = OpTypedTransaction::Eip1559(TxEip1559 {
            chain_id: ctx.chain_id(),
            nonce,
            // Due to EIP-150, 63/64 of available gas is forwarded to external calls so need to add a buffer
            gas_limit: gas_used * 64 / 63,
            max_fee_per_gas: ctx.base_fee().into(),
            to: TxKind::Call(to),
            input: calldata,
            ..Default::default()
        });
        Ok(from.sign_tx(tx)?)
    }

    fn commit_txs(
        &self,
        signed_txs: Vec<Recovered<OpTransactionSigned>>,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        db: &mut State<impl Database>,
    ) -> Result<(), BuilderTransactionError> {
        let mut evm = ctx.evm_config.evm_with_env(&mut *db, ctx.evm_env.clone());
        for signed_tx in signed_txs {
            let ResultAndState { state, .. } = evm
                .transact(&signed_tx)
                .map_err(|err| BuilderTransactionError::EvmExecutionError(Box::new(err)))?;
            evm.db_mut().commit(state)
        }
        Ok(())
    }

    fn simulate_call<T: SolCall, E: SolInterface + Debug>(
        &self,
        tx: OpTransactionRequest,
        expected_logs: Vec<B256>,
        evm: &mut OpEvm<impl Database, NoOpInspector, PrecompilesMap>,
    ) -> Result<SimulationSuccessResult<T>, BuilderTransactionError> {
        let evm_env = alloy_evm::EvmEnv::new(evm.cfg_env().clone(), evm.block().clone());
        let tx_env: revm::context::TxEnv = tx.as_ref().clone().try_into_tx_env(&evm_env)?;
        let to = tx_env.kind.into_to().unwrap_or_default();
        let op_tx = OpTx(op_revm::OpTransaction {
            base: tx_env,
            enveloped_tx: Some(Bytes::new()),
            deposit: Default::default(),
        });

        let ResultAndState { result, state } = evm.transact(op_tx).map_err(|err| {
            if err.is_invalid_tx_err() {
                BuilderTransactionError::InvalidTransactionError(Box::new(err))
            } else {
                BuilderTransactionError::EvmExecutionError(Box::new(err))
            }
        })?;
        let gas_used = result.tx_gas_used();

        match result {
            ExecutionResult::Success { output, logs, .. } => {
                let topics: HashSet<B256> = logs
                    .into_iter()
                    .flat_map(|log| log.topics().to_vec())
                    .collect();
                if !expected_logs
                    .iter()
                    .all(|expected_topic| topics.contains(expected_topic))
                {
                    return Err(BuilderTransactionError::InvalidContract(
                        to,
                        InvalidContractDataError::InvalidLogs(
                            expected_logs,
                            topics.into_iter().collect(),
                        ),
                    ));
                }
                let return_output = T::abi_decode_returns(&output.into_data()).map_err(|_| {
                    BuilderTransactionError::InvalidContract(
                        to,
                        InvalidContractDataError::OutputAbiDecodeError,
                    )
                })?;
                Ok(SimulationSuccessResult::<T> {
                    gas_used,
                    output: return_output,
                    state_changes: state,
                })
            }
            ExecutionResult::Revert { output, .. } => {
                let revert = ContractError::<E>::abi_decode(&output)
                    .map(|reason| Revert::from(format!("{reason:?}")))
                    .or_else(|_| Revert::abi_decode(&output))
                    .unwrap_or_else(|_| {
                        Revert::from(format!("unknown revert: {}", hex::encode(&output)))
                    });
                Err(BuilderTransactionError::TransactionReverted(to, revert))
            }
            ExecutionResult::Halt { reason, .. } => {
                Err(BuilderTransactionError::TransactionHalted(to, reason))
            }
        }
    }
}

#[derive(Debug, Clone)]
pub(super) struct BuilderTxBase<ExtraCtx = ()> {
    pub signer: Option<Signer>,
    _marker: std::marker::PhantomData<ExtraCtx>,
}

impl<ExtraCtx: Debug + Default> BuilderTxBase<ExtraCtx> {
    pub(super) fn new(signer: Option<Signer>) -> Self {
        Self {
            signer,
            _marker: std::marker::PhantomData,
        }
    }

    pub(super) fn simulate_builder_tx(
        &self,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        db: impl DatabaseRef,
    ) -> Result<Option<BuilderTransactionCtx>, BuilderTransactionError> {
        match self.signer {
            Some(signer) => {
                let message: Vec<u8> = format!("Block Number: {}", ctx.block_number()).into_bytes();
                let gas_used = self.estimate_builder_tx_gas(&message);
                let signed_tx = self.signed_builder_tx(ctx, db, signer, gas_used, message)?;
                let da_size = op_alloy_flz::tx_estimated_size_fjord_bytes(
                    signed_tx.encoded_2718().as_slice(),
                );
                Ok(Some(BuilderTransactionCtx {
                    gas_used,
                    da_size,
                    signed_tx,
                    is_top_of_block: false,
                }))
            }
            None => Ok(None),
        }
    }

    fn estimate_builder_tx_gas(&self, input: &[u8]) -> u64 {
        // Count zero and non-zero bytes
        let (zero_bytes, nonzero_bytes) = input.iter().fold((0, 0), |(zeros, nonzeros), &byte| {
            if byte == 0 {
                (zeros + 1, nonzeros)
            } else {
                (zeros, nonzeros + 1)
            }
        });

        // Calculate gas cost (4 gas per zero byte, 16 gas per non-zero byte)
        let zero_cost = zero_bytes * 4;
        let nonzero_cost = nonzero_bytes * 16;

        // Tx gas should be not less than floor gas https://eips.ethereum.org/EIPS/eip-7623
        let tokens_in_calldata = zero_bytes + nonzero_bytes * 4;
        let floor_gas = 21_000 + tokens_in_calldata * TOTAL_COST_FLOOR_PER_TOKEN;

        std::cmp::max(zero_cost + nonzero_cost + 21_000, floor_gas)
    }

    fn signed_builder_tx(
        &self,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        db: impl DatabaseRef,
        signer: Signer,
        gas_used: u64,
        message: Vec<u8>,
    ) -> Result<Recovered<OpTransactionSigned>, BuilderTransactionError> {
        let nonce = get_nonce(db, signer.address)?;

        // Create the EIP-1559 transaction
        let tx = OpTypedTransaction::Eip1559(TxEip1559 {
            chain_id: ctx.chain_id(),
            nonce,
            gas_limit: gas_used,
            max_fee_per_gas: ctx.base_fee().into(),
            max_priority_fee_per_gas: 0,
            to: TxKind::Call(Address::ZERO),
            // Include the message as part of the transaction data
            input: message.into(),
            ..Default::default()
        });
        // Sign the transaction
        let builder_tx = signer
            .sign_tx(tx)
            .map_err(BuilderTransactionError::SigningError)?;

        Ok(builder_tx)
    }
}

pub fn get_nonce(db: impl DatabaseRef, address: Address) -> Result<u64, BuilderTransactionError> {
    db.basic_ref(address)
        .map(|acc| acc.unwrap_or_default().nonce)
        .map_err(|_| BuilderTransactionError::AccountLoadFailed(address))
}

pub fn get_balance(
    db: impl DatabaseRef,
    address: Address,
) -> Result<U256, BuilderTransactionError> {
    db.basic_ref(address)
        .map(|acc| acc.unwrap_or_default().balance)
        .map_err(|_| BuilderTransactionError::AccountLoadFailed(address))
}
