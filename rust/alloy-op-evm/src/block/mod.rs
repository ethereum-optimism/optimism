//! Block executor for Optimism.

use crate::{OpEvmFactory, spec_by_timestamp_after_bedrock};
use alloc::{
    borrow::Cow, boxed::Box, collections::BTreeMap, format, string::String, vec, vec::Vec,
};
use alloy_consensus::{Eip658Value, Header, Transaction, TransactionEnvelope, TxReceipt};
use alloy_eips::{Encodable2718, Typed2718, eip7685::Requests};
use alloy_evm::{
    Database, Evm, EvmFactory, FromRecoveredTx, FromTxWithEncoded, IntoTxEnv, RecoveredTx,
    block::{
        BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
        BlockExecutorFor, BlockValidationError, ExecutableTx, GasOutput, OnStateHook,
        StateChangePostBlockSource, StateChangeSource, StateDB, SystemCaller, TxResult,
        state_changes::{balance_increment_state, post_block_balance_increments},
    },
    eth::{EthTxResult, receipt_builder::ReceiptBuilderCtx},
};
use alloy_op_hardforks::{OpChainHardforks, OpHardforks};
use alloy_primitives::{Address, B256, Bytes, U256};
use canyon::ensure_create2_deployer;
use op_alloy::consensus::{
    OpDepositReceipt, OpTransaction as OpConsensusTransaction, POST_EXEC_TX_TYPE_ID,
    PostExecPayload, SDMGasEntry,
};
use op_revm::{
    L1BlockInfo, OpTransaction,
    constants::{BASE_FEE_RECIPIENT, L1_BLOCK_CONTRACT, OPERATOR_FEE_RECIPIENT},
    estimate_tx_compressed_size,
    transaction::deposit::DEPOSIT_TRANSACTION_TYPE,
};
pub use receipt_builder::OpAlloyReceiptBuilder;
use receipt_builder::OpReceiptBuilder;
use revm::{
    Database as _, DatabaseCommit, Inspector,
    context::{
        Block, TxEnv,
        result::{ExecutionResult, Output, ResultAndState, SuccessReason},
    },
    database::DatabaseCommitExt,
    state::{Account, AccountStatus, EvmState},
};

use crate::post_exec::{
    PostExecEvm, PostExecEvmFactoryAdapter, PostExecEvmFactoryHooks, PostExecTxContext,
    PostExecTxKind,
};

mod canyon;
pub mod receipt_builder;

/// Trait for OP transaction environments. Allows to recover the transaction encoded bytes if
/// they're available.
pub trait OpTxEnv {
    /// Returns the encoded bytes of the transaction.
    fn encoded_bytes(&self) -> Option<&Bytes>;
}

impl<T: revm::context::Transaction> OpTxEnv for OpTransaction<T> {
    fn encoded_bytes(&self) -> Option<&Bytes> {
        self.enveloped_tx.as_ref()
    }
}

/// Canonical post-exec execution mode for an OP block.
#[derive(Debug, Default, Clone)]
pub enum PostExecMode {
    /// Execute with legacy gas accounting.
    #[default]
    Disabled,
    /// Produce canonical post-exec refunds locally and append them to the block later.
    Produce,
    /// Verify canonical gas accounting using an post-exec payload embedded in the block.
    Verify(PostExecPayload),
}

/// Per-block post-exec state carried by [`OpBlockExecutor`].
#[derive(Debug)]
pub enum PostExecState {
    /// Execute with legacy gas accounting.
    Disabled,
    /// Produce canonical post-exec refunds locally and append them to the block later.
    Producing {
        /// Accumulated per-tx warming refunds for post-exec tx assembly.
        entries: Vec<SDMGasEntry>,
    },
    /// Verify canonical gas accounting using a post-exec payload embedded in the block.
    ///
    /// `payload` and `remaining` are not redundant: `payload` is the immutable verifier input
    /// (kept for byte-equality comparison against the actual `0x7D` tx and for the block-number
    /// re-check), while `remaining` is the mutable working set drained as txs are matched.
    Verifying {
        /// Decoded post-exec payload being verified.
        payload: PostExecPayload,
        /// Verifier payload entries not yet consumed, indexed by original tx index.
        remaining: BTreeMap<u64, u64>,
        /// Invalid verifier payload reason, if any.
        invalid_reason: Option<String>,
        /// Whether the block's synthetic post-exec transaction has been seen during execution.
        saw_post_exec_tx: bool,
    },
}

impl PostExecState {
    fn new(mode: PostExecMode) -> Self {
        match mode {
            PostExecMode::Disabled => Self::Disabled,
            PostExecMode::Produce => Self::Producing { entries: Vec::new() },
            PostExecMode::Verify(payload) => {
                let mut remaining = BTreeMap::new();
                let mut invalid_reason = None;

                for entry in &payload.gas_refund_entries {
                    if entry.gas_refund == 0 {
                        invalid_reason = Some(format!(
                            "zero post-exec payload refund for tx index {}",
                            entry.index
                        ));
                        break;
                    }
                    if remaining.insert(entry.index, entry.gas_refund).is_some() {
                        invalid_reason = Some(format!(
                            "duplicate post-exec payload entry for tx index {}",
                            entry.index
                        ));
                        break;
                    }
                }

                Self::Verifying { payload, remaining, invalid_reason, saw_post_exec_tx: false }
            }
        }
    }

    const fn is_producing(&self) -> bool {
        matches!(self, Self::Producing { .. })
    }

    const fn is_verifying(&self) -> bool {
        matches!(self, Self::Verifying { .. })
    }

    const fn invalid_reason(&self) -> Option<&str> {
        match self {
            Self::Verifying { invalid_reason: Some(reason), .. } => Some(reason.as_str()),
            _ => None,
        }
    }

    fn verify_block_number(&self, block_number: u64) -> Option<String> {
        match self {
            Self::Verifying { payload, .. } if payload.block_number != block_number => {
                Some(format!(
                    "payload block number {} does not match block number {}",
                    payload.block_number, block_number,
                ))
            }
            _ => None,
        }
    }

    const fn produced_entries_mut(&mut self) -> Option<&mut Vec<SDMGasEntry>> {
        match self {
            Self::Producing { entries } => Some(entries),
            _ => None,
        }
    }

    fn take_entries(&mut self) -> Vec<SDMGasEntry> {
        match self {
            Self::Producing { entries } => core::mem::take(entries),
            _ => Vec::new(),
        }
    }

    fn verifier_refund(&self, tx_index: u64) -> Option<u64> {
        match self {
            Self::Verifying { remaining, .. } => remaining.get(&tx_index).copied(),
            _ => None,
        }
    }

    fn consume_verifier_entry(&mut self, tx_index: u64) {
        if let Self::Verifying { remaining, .. } = self {
            remaining.remove(&tx_index);
        }
    }

    fn verify_post_exec_tx(
        &mut self,
        tx_index: u64,
        payload: &PostExecPayload,
    ) -> Result<(), String> {
        match self {
            Self::Verifying { payload: expected, saw_post_exec_tx, .. } => {
                if *saw_post_exec_tx {
                    return Err(format!("duplicate post-exec tx at index {tx_index}"));
                }
                if payload != expected {
                    return Err(format!("post-exec tx payload mismatch at index {tx_index}"));
                }
                *saw_post_exec_tx = true;
                Ok(())
            }
            Self::Producing { .. } => Ok(()),
            Self::Disabled => Err(format!(
                "unexpected post-exec tx at index {tx_index}: SDM not active for this block"
            )),
        }
    }

    fn remaining_verifier_indexes(&self) -> Vec<u64> {
        match self {
            Self::Verifying { remaining, .. } => remaining.keys().copied().collect(),
            _ => Vec::new(),
        }
    }
}

/// Context for OP block execution.
#[derive(Debug, Default, Clone)]
pub struct OpBlockExecutionCtx {
    /// Parent block hash.
    pub parent_hash: B256,
    /// Parent beacon block root.
    pub parent_beacon_block_root: Option<B256>,
    /// The block's extra data.
    pub extra_data: Bytes,
    /// Canonical post-exec execution mode for this block.
    pub post_exec_mode: PostExecMode,
}

/// Balance patch that reconciles fee distribution with the post-refund gas used.
///
/// The EVM has already paid fees out based on `evm_gas_used`, so just lowering `gas_used` in the
/// receipt isn't enough — the sender, beneficiary, base-fee recipient, and operator-fee recipient
/// all need their balances rolled back to match the canonical gas. This struct carries the
/// per-recipient debits, plus the matching credit to the sender (which equals their sum).
#[derive(Debug, Default, Clone)]
pub struct PostExecAdjustment {
    /// Refund amount subtracted from `evm_gas_used` to produce `canonical_gas_used`.
    pub refund: u64,
    /// Wei to credit back to the sender (sum of the three recipient deltas below).
    pub sender_balance_delta: U256,
    /// Wei to debit from the block beneficiary — priority-fee share of the refund.
    pub beneficiary_balance_delta: U256,
    /// Wei to debit from the base-fee recipient — base-fee share of the refund.
    pub base_fee_balance_delta: U256,
    /// Wei to debit from the operator-fee recipient — operator-fee share of the refund
    /// (post-Isthmus).
    pub operator_fee_balance_delta: U256,
}

/// The result of executing an OP transaction.
#[derive(Debug)]
pub struct OpTxResult<H, T> {
    /// The inner result of the transaction execution.
    pub inner: EthTxResult<H, T>,
    /// Whether the transaction is a deposit transaction.
    pub is_deposit: bool,
    /// Whether the transaction is a post-exec transaction.
    pub is_post_exec: bool,
    /// The sender of the transaction.
    pub sender: Address,
    /// Gas used returned by normal EVM execution, before any canonical post-exec adjustment.
    pub evm_gas_used: u64,
    /// Canonical gas used after any post-exec adjustment.
    pub canonical_gas_used: u64,
    /// Canonical post-exec adjustment, if any.
    pub post_exec: Option<PostExecAdjustment>,
}

impl<H, T> TxResult for OpTxResult<H, T> {
    type HaltReason = H;

    fn result(&self) -> &ResultAndState<Self::HaltReason> {
        &self.inner.result
    }

    fn into_result(self) -> ResultAndState<Self::HaltReason> {
        self.inner.result
    }
}

/// Block executor for Optimism.
#[derive(Debug)]
pub struct OpBlockExecutor<Evm, R: OpReceiptBuilder, Spec> {
    /// Spec.
    pub spec: Spec,
    /// Receipt builder.
    pub receipt_builder: R,
    /// Context for block execution.
    pub ctx: OpBlockExecutionCtx,
    /// The EVM used by executor.
    pub evm: Evm,
    /// Receipts of executed transactions.
    pub receipts: Vec<R::Receipt>,
    /// Total gas used by executed transactions.
    pub gas_used: u64,
    /// Da footprint.
    ///
    /// This is only set for blocks post-Jovian activation.
    /// See [DA footprint block limit spec](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/exec-engine.md#da-footprint-block-limit)
    pub da_footprint_used: u64,
    /// Whether Regolith hardfork is active.
    pub is_regolith: bool,
    /// Utility to call system smart contracts.
    pub system_caller: SystemCaller<Spec>,
    /// Cached L1 block info for the current block.
    pub l1_block_info: Option<L1BlockInfo>,
    /// Post-exec execution state (mode and producer/verifier working state).
    pub post_exec: PostExecState,
}

impl<E, R, Spec> OpBlockExecutor<E, R, Spec>
where
    E: Evm,
    R: OpReceiptBuilder,
    Spec: OpHardforks + Clone,
{
    /// Creates a new [`OpBlockExecutor`].
    pub fn new(evm: E, ctx: OpBlockExecutionCtx, spec: Spec, receipt_builder: R) -> Self {
        let post_exec = PostExecState::new(ctx.post_exec_mode.clone());
        Self {
            is_regolith: spec
                .is_regolith_active_at_timestamp(evm.block().timestamp().saturating_to()),
            evm,
            system_caller: SystemCaller::new(spec.clone()),
            spec,
            receipt_builder,
            receipts: Vec::new(),
            gas_used: 0,
            da_footprint_used: 0,
            ctx,
            l1_block_info: None,
            post_exec,
        }
    }

    /// Set the post-exec execution mode for the executor.
    #[must_use]
    pub fn with_post_exec_mode(mut self, post_exec_mode: PostExecMode) -> Self {
        self.set_post_exec_mode(post_exec_mode);
        self
    }

    /// Set the post-exec execution mode for the executor.
    ///
    /// This is primarily intended for tests and replay tooling that need to override the
    /// block-context default after construction.
    pub fn set_post_exec_mode(&mut self, post_exec_mode: PostExecMode) {
        self.post_exec = PostExecState::new(post_exec_mode);
    }

    /// Take the accumulated post-exec entries (sequencer mode).
    /// Returns the entries and clears the internal state.
    pub fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry> {
        self.post_exec.take_entries()
    }
}

/// Custom errors that can occur during OP block execution.
#[derive(Debug, thiserror::Error)]
pub enum OpBlockExecutionError {
    /// Failed to load cache account.
    #[error("failed to load cache account")]
    LoadCacheAccount,

    /// Failed to get Jovian da footprint gas scalar from database.
    #[error("failed to get da footprint gas scalar from database: {_0}")]
    GetJovianDaFootprintScalar(Box<dyn core::error::Error + Send + Sync + 'static>),

    /// Transaction DA footprint exceeds available block DA footprint.
    #[error(
        "transaction DA footprint exceeds available block DA footprint. transaction_da_footprint: {transaction_da_footprint}, available_block_da_footprint: {available_block_da_footprint}"
    )]
    TransactionDaFootprintAboveGasLimit {
        /// The DA footprint of the transaction to execute.
        transaction_da_footprint: u64,
        /// The available block DA footprint.
        available_block_da_footprint: u64,
    },

    /// The block contained an invalid post-exec payload.
    #[error("invalid post-exec payload: {0}")]
    InvalidPostExecPayload(String),

    /// Canonical post-exec settlement would underflow an account balance.
    #[error("canonical post-exec settlement underflow for {address}: delta {delta}")]
    PostExecSettlementUnderflow {
        /// Account whose balance would underflow.
        address: Address,
        /// Delta that could not be removed from the account.
        delta: U256,
    },
}

impl<E, R, Spec> OpBlockExecutor<E, R, Spec>
where
    E: Evm<
            DB: Database + DatabaseCommit + StateDB,
            Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
        >,
    R: OpReceiptBuilder<
            Transaction: Transaction + Encodable2718 + OpConsensusTransaction,
            Receipt: TxReceipt,
        >,
    Spec: OpHardforks,
{
    fn jovian_da_footprint_estimation(
        &mut self,
        tx_env: &E::Tx,
        tx: impl RecoveredTx<R::Transaction>,
    ) -> Result<u64, BlockExecutionError> {
        // Try to use the enveloped tx if it exists, otherwise use the encoded 2718 bytes
        let encoded = tx_env
            .encoded_bytes()
            .map_or_else(
                || estimate_tx_compressed_size(tx.tx().encoded_2718().as_ref()),
                |encoded| estimate_tx_compressed_size(encoded),
            )
            .saturating_div(1_000_000);

        // Load the L1 block contract into the cache. If the L1 block contract is not pre-loaded the
        // database will panic when trying to fetch the DA footprint gas scalar.
        self.evm.db_mut().basic(L1_BLOCK_CONTRACT).map_err(BlockExecutionError::other)?;

        let da_footprint_gas_scalar = L1BlockInfo::fetch_da_footprint_gas_scalar(self.evm.db_mut())
            .map_err(BlockExecutionError::other)?
            .into();

        Ok(encoded.saturating_mul(da_footprint_gas_scalar))
    }

    fn invalid_post_exec_payload(reason: impl Into<String>) -> BlockExecutionError {
        BlockExecutionError::Validation(BlockValidationError::Other(Box::new(
            OpBlockExecutionError::InvalidPostExecPayload(reason.into()),
        )))
    }

    fn verifier_post_exec_refund_for_tx(
        &self,
        tx_index: u64,
        is_deposit: bool,
        is_post_exec: bool,
        evm_gas_used: u64,
    ) -> Result<u64, BlockExecutionError> {
        // Entry-existence first: deposit and post-exec txs are called with this helper
        // unconditionally to validate their tx index against the payload, so we can only
        // raise the deposit/post-exec error when the payload actually targets them.
        let Some(refund) = self.post_exec.verifier_refund(tx_index) else {
            return Ok(0);
        };

        if is_deposit {
            return Err(Self::invalid_post_exec_payload(format!(
                "payload entry targets deposit tx index {tx_index}"
            )));
        }

        if is_post_exec {
            return Err(Self::invalid_post_exec_payload(format!(
                "payload entry targets post-exec tx index {tx_index}"
            )));
        }

        if refund > evm_gas_used {
            return Err(Self::invalid_post_exec_payload(format!(
                "payload refund {refund} exceeds evm_gas_used {evm_gas_used} for tx index {tx_index}"
            )));
        }

        Ok(refund)
    }

    const fn canonicalize_result_gas(
        result: &mut ExecutionResult<E::HaltReason>,
        post_exec_refund: u64,
    ) {
        if post_exec_refund == 0 {
            return;
        }

        match result {
            ExecutionResult::Success { gas, .. } => {
                *gas = gas
                    .with_total_gas_spent(gas.total_gas_spent().saturating_sub(post_exec_refund))
                    .with_refunded(gas.inner_refunded().saturating_add(post_exec_refund));
            }
            ExecutionResult::Revert { gas, .. } | ExecutionResult::Halt { gas, .. } => {
                *gas = gas
                    .with_total_gas_spent(gas.total_gas_spent().saturating_sub(post_exec_refund));
            }
        }
    }

    fn state_account_mut<'a>(
        db: &mut E::DB,
        state: &'a mut EvmState,
        address: Address,
    ) -> Result<&'a mut Account, BlockExecutionError> {
        use revm::primitives::hash_map::Entry;

        match state.entry(address) {
            Entry::Occupied(entry) => Ok(entry.into_mut()),
            Entry::Vacant(entry) => {
                let info =
                    db.basic(address).map_err(BlockExecutionError::other)?.unwrap_or_default();
                let original_info = info.clone();
                Ok(entry.insert(Account {
                    info,
                    // The original_info is not used by State::commit — the
                    // CacheAccount tracks its own previous state for building
                    // transitions. Setting it equal to current info is safe.
                    original_info: Box::new(original_info),
                    status: AccountStatus::Touched,
                    ..Default::default()
                }))
            }
        }
    }

    fn add_state_balance(
        db: &mut E::DB,
        state: &mut EvmState,
        address: Address,
        delta: U256,
    ) -> Result<(), BlockExecutionError> {
        if delta.is_zero() {
            return Ok(());
        }

        let account = Self::state_account_mut(db, state, address)?;
        account.mark_touch();
        account.info.balance = account.info.balance.saturating_add(delta);
        Ok(())
    }

    fn sub_state_balance(
        db: &mut E::DB,
        state: &mut EvmState,
        address: Address,
        delta: U256,
    ) -> Result<(), BlockExecutionError> {
        if delta.is_zero() {
            return Ok(());
        }

        let account = Self::state_account_mut(db, state, address)?;
        account.mark_touch();
        account.info.balance = account.info.balance.checked_sub(delta).ok_or_else(|| {
            BlockExecutionError::Validation(BlockValidationError::Other(Box::new(
                OpBlockExecutionError::PostExecSettlementUnderflow { address, delta },
            )))
        })?;
        Ok(())
    }

    fn l1_block_info(
        &mut self,
        spec_id: op_revm::OpSpecId,
    ) -> Result<L1BlockInfo, BlockExecutionError> {
        if let Some(l1_block_info) = &self.l1_block_info {
            return Ok(l1_block_info.clone());
        }

        let block_number = self.evm.block().number();
        let l1_block_info = L1BlockInfo::try_fetch(self.evm.db_mut(), block_number, spec_id)
            .map_err(BlockExecutionError::other)?;
        self.l1_block_info = Some(l1_block_info.clone());
        Ok(l1_block_info)
    }

    /// Computes the fee-settlement patch required after canonicalizing post-exec gas.
    ///
    /// `evm.transact` has already charged the sender and paid fee recipients according to
    /// `evm_gas_used`. Lowering only the receipt's `gas_used` would leave those balance changes
    /// in place. This translates the refunded gas back into the exact per-recipient deltas
    /// `commit_transaction` then applies before state is committed.
    fn post_exec_settlement_deltas(
        &mut self,
        tx: impl RecoveredTx<R::Transaction>,
        evm_gas_used: u64,
        post_exec_refund: u64,
        is_deposit: bool,
        is_post_exec: bool,
    ) -> Result<PostExecAdjustment, BlockExecutionError> {
        if is_deposit || is_post_exec || post_exec_refund == 0 {
            return Ok(PostExecAdjustment::default());
        }

        let gas_delta_u256 = U256::from(post_exec_refund);
        let basefee = u128::from(self.evm.block().basefee());
        let spec_id = spec_by_timestamp_after_bedrock(
            &self.spec,
            self.evm.block().timestamp().saturating_to(),
        );
        let effective_gas_price = tx.tx().effective_gas_price(Some(self.evm.block().basefee()));
        // SDM/PostExec is only enabled on forks after Karst, which is already post-London.
        // A saturating_sub landing at zero is intentional and consensus-valid: a legacy tx
        // with a gas price equal to the basefee pays zero priority fee, so the beneficiary
        // delta below must be zero as well — we credit back only what the beneficiary
        // actually received for the refunded gas, which is the (effective_price - basefee)
        // component.
        let beneficiary_gas_price = effective_gas_price.saturating_sub(basefee);

        let base_fee_balance_delta = gas_delta_u256.saturating_mul(U256::from(basefee));
        let beneficiary_balance_delta =
            gas_delta_u256.saturating_mul(U256::from(beneficiary_gas_price));

        let canonical_gas_used = evm_gas_used.saturating_sub(post_exec_refund);
        let l1_block_info = self.l1_block_info(spec_id)?;
        let encoded = tx.tx().encoded_2718();
        let raw_fee =
            l1_block_info.operator_fee_charge(encoded.as_ref(), U256::from(evm_gas_used), spec_id);
        let canonical_fee = l1_block_info.operator_fee_charge(
            encoded.as_ref(),
            U256::from(canonical_gas_used),
            spec_id,
        );
        let operator_fee_balance_delta = raw_fee.saturating_sub(canonical_fee);

        let sender_balance_delta = gas_delta_u256
            .saturating_mul(U256::from(effective_gas_price))
            .saturating_add(operator_fee_balance_delta);

        Ok(PostExecAdjustment {
            refund: post_exec_refund,
            sender_balance_delta,
            beneficiary_balance_delta,
            base_fee_balance_delta,
            operator_fee_balance_delta,
        })
    }

    fn apply_post_exec_refund_to_state(
        &mut self,
        state: &mut EvmState,
        sender: Address,
        deltas: &PostExecAdjustment,
    ) -> Result<(), BlockExecutionError> {
        let beneficiary = self.evm.block().beneficiary();
        Self::add_state_balance(self.evm.db_mut(), state, sender, deltas.sender_balance_delta)?;
        Self::sub_state_balance(
            self.evm.db_mut(),
            state,
            beneficiary,
            deltas.beneficiary_balance_delta,
        )?;
        Self::sub_state_balance(
            self.evm.db_mut(),
            state,
            BASE_FEE_RECIPIENT,
            deltas.base_fee_balance_delta,
        )?;
        Self::sub_state_balance(
            self.evm.db_mut(),
            state,
            OPERATOR_FEE_RECIPIENT,
            deltas.operator_fee_balance_delta,
        )?;

        Ok(())
    }
}

impl<E, R, Spec> BlockExecutor for OpBlockExecutor<E, R, Spec>
where
    E: PostExecEvm<
            DB: Database + DatabaseCommit + StateDB,
            Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
        >,
    R: OpReceiptBuilder<
            Transaction: Transaction + Encodable2718 + OpConsensusTransaction,
            Receipt: TxReceipt,
        >,
    Spec: OpHardforks,
{
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;
    type Evm = E;
    type Result = OpTxResult<E::HaltReason, <R::Transaction as TransactionEnvelope>::TxType>;

    fn apply_pre_execution_changes(&mut self) -> Result<(), BlockExecutionError> {
        if let Some(reason) = self.post_exec.invalid_reason() {
            return Err(Self::invalid_post_exec_payload(String::from(reason)));
        }
        let block_number = self.evm.block().number().saturating_to::<u64>();
        if let Some(reason) = self.post_exec.verify_block_number(block_number) {
            return Err(Self::invalid_post_exec_payload(reason));
        }

        self.system_caller.apply_blockhashes_contract_call(self.ctx.parent_hash, &mut self.evm)?;
        self.system_caller
            .apply_beacon_root_contract_call(self.ctx.parent_beacon_block_root, &mut self.evm)?;

        // Ensure that the create2deployer is force-deployed at the canyon transition. Optimism
        // blocks will always have at least a single transaction in them (the L1 info transaction),
        // so we can safely assume that this will always be triggered upon the transition and that
        // the above check for empty blocks will never be hit on OP chains.
        ensure_create2_deployer(
            &self.spec,
            self.evm.block().timestamp().saturating_to(),
            self.evm.db_mut(),
        )
        .map_err(BlockExecutionError::other)?;

        Ok(())
    }

    fn execute_transaction_without_commit(
        &mut self,
        tx: impl ExecutableTx<Self>,
    ) -> Result<Self::Result, BlockExecutionError> {
        let (tx_env, tx) = tx.into_parts();
        let is_deposit = tx.tx().ty() == DEPOSIT_TRANSACTION_TYPE;
        let is_post_exec = tx.tx().ty() == POST_EXEC_TX_TYPE_ID;
        let tx_index = self.receipts.len() as u64;

        // The sum of the transaction's gas limit, Tg, and the gas utilized in this block prior,
        // must be no greater than the block's gasLimit.
        let block_available_gas = self.evm.block().gas_limit() - self.gas_used;
        if tx.tx().gas_limit() > block_available_gas && (self.is_regolith || !is_deposit) {
            return Err(BlockValidationError::TransactionGasLimitMoreThanAvailableBlockGas {
                transaction_gas_limit: tx.tx().gas_limit(),
                block_available_gas,
            }
            .into());
        }

        if is_post_exec {
            let payload =
                tx.tx().as_post_exec().map(|tx| &tx.inner().payload).ok_or_else(|| {
                    Self::invalid_post_exec_payload(format!(
                    "transaction at index {tx_index} has post-exec type but no post-exec payload",
                ))
                })?;
            if let Err(reason) = self.post_exec.verify_post_exec_tx(tx_index, payload) {
                return Err(Self::invalid_post_exec_payload(reason));
            }
            // Validates that no Verify payload entry targets this tx index; refund is always 0.
            self.verifier_post_exec_refund_for_tx(tx_index, false, true, 0)?;
            return Ok(OpTxResult {
                inner: EthTxResult {
                    result: ResultAndState::new(
                        ExecutionResult::Success {
                            reason: SuccessReason::Stop,
                            gas: revm::context::result::ResultGas::default(),
                            logs: vec![],
                            output: Output::Call(Bytes::default()),
                        },
                        EvmState::default(),
                    ),
                    blob_gas_used: 0,
                    tx_type: tx.tx().tx_type(),
                },
                is_deposit: false,
                is_post_exec: true,
                sender: *tx.signer(),
                evm_gas_used: 0,
                canonical_gas_used: 0,
                post_exec: None,
            });
        }

        let da_footprint_used = if self
            .spec
            .is_jovian_active_at_timestamp(self.evm.block().timestamp().saturating_to()) &&
            !is_deposit
        {
            let da_footprint_available = self.evm.block().gas_limit() - self.da_footprint_used;

            let tx_da_footprint = self.jovian_da_footprint_estimation(&tx_env, &tx)?;

            if tx_da_footprint > da_footprint_available {
                return Err(BlockExecutionError::Validation(BlockValidationError::Other(
                    Box::new(OpBlockExecutionError::TransactionDaFootprintAboveGasLimit {
                        transaction_da_footprint: tx_da_footprint,
                        available_block_da_footprint: da_footprint_available,
                    }),
                )));
            }

            tx_da_footprint
        } else {
            0
        };

        if self.post_exec.is_producing() {
            self.evm.begin_post_exec_tx(PostExecTxContext {
                tx_index,
                kind: if is_deposit { PostExecTxKind::Deposit } else { PostExecTxKind::Normal },
            });
        }

        // Execute transaction and return the result
        let result = self.evm.transact(tx_env).map_err(|err| {
            let hash = tx.tx().trie_hash();
            BlockExecutionError::evm(err, hash)
        })?;

        let evm_gas_used = result.result.tx_gas_used();
        let post_exec_refund = if self.post_exec.is_producing() {
            let refund = self.evm.take_last_post_exec_tx_result().refund_total;
            // The inspector's accumulated refund must never exceed the tx's evm_gas_used. If
            // it does, we'd emit an `SDMGasEntry` that any honest verifier would reject
            // at pre-execution ("payload refund exceeds evm_gas_used"), so the sequencer
            // would ship a block it can't verify itself. Fail here with a loud error
            // instead of letting `saturating_sub` mask the discrepancy.
            if refund > evm_gas_used {
                return Err(Self::invalid_post_exec_payload(format!(
                    "produced refund {refund} exceeds evm_gas_used {evm_gas_used} for tx index {tx_index}",
                )));
            }
            refund
        } else {
            self.verifier_post_exec_refund_for_tx(tx_index, is_deposit, false, evm_gas_used)?
        };
        let canonical_gas_used = evm_gas_used.saturating_sub(post_exec_refund);
        let deltas = self.post_exec_settlement_deltas(
            &tx,
            evm_gas_used,
            post_exec_refund,
            is_deposit,
            false,
        )?;
        let post_exec = (post_exec_refund > 0).then_some(deltas);

        Ok(OpTxResult {
            inner: EthTxResult {
                result,
                blob_gas_used: da_footprint_used,
                tx_type: tx.tx().tx_type(),
            },
            is_deposit,
            is_post_exec: false,
            sender: *tx.signer(),
            evm_gas_used,
            canonical_gas_used,
            post_exec,
        })
    }

    fn commit_transaction(
        &mut self,
        output: Self::Result,
    ) -> Result<GasOutput, BlockExecutionError> {
        let tx_index = self.receipts.len() as u64;
        let OpTxResult {
            inner:
                EthTxResult { result: ResultAndState { mut result, mut state }, blob_gas_used, tx_type },
            is_deposit,
            is_post_exec,
            sender,
            evm_gas_used: _,
            canonical_gas_used,
            post_exec,
        } = output;

        let deltas = post_exec.unwrap_or_default();
        let post_exec_refund = deltas.refund;

        if !is_deposit && !is_post_exec && post_exec_refund > 0 {
            if let Some(entries) = self.post_exec.produced_entries_mut() {
                entries.push(SDMGasEntry { index: tx_index, gas_refund: post_exec_refund });
            }
        }
        if self.post_exec.is_verifying() && post_exec_refund > 0 {
            self.post_exec.consume_verifier_entry(tx_index);
        }

        // Fetch the depositor account from the database for the deposit nonce.
        // Note that this *only* needs to be done post-regolith hardfork, as deposit nonces
        // were not introduced in Bedrock. In addition, regular transactions don't have deposit
        // nonces, so we don't need to touch the DB for those.
        let depositor = (self.is_regolith && is_deposit)
            .then(|| self.evm.db_mut().basic(sender).map(|acc| acc.unwrap_or_default()))
            .transpose()
            .map_err(BlockExecutionError::other)?;

        // Canonicalizing the gas in the execution result updates receipt/block gas accounting.
        // The separate state patch below is still required because the EVM state diff already
        // contains the EVM-level fee settlement from execution.
        Self::canonicalize_result_gas(&mut result, post_exec_refund);
        self.apply_post_exec_refund_to_state(&mut state, sender, &deltas)?;

        self.system_caller.on_state(StateChangeSource::Transaction(self.receipts.len()), &state);

        // add canonical gas used
        self.gas_used += canonical_gas_used;

        // Update DA footprint if Jovian is active
        if self.spec.is_jovian_active_at_timestamp(self.evm.block().timestamp().saturating_to()) &&
            !is_deposit &&
            !is_post_exec
        {
            // Add to DA footprint used
            self.da_footprint_used = self.da_footprint_used.saturating_add(blob_gas_used);
        }

        self.receipts.push(
            match self.receipt_builder.build_receipt(ReceiptBuilderCtx {
                tx_type,
                result,
                cumulative_gas_used: self.gas_used,
                evm: &self.evm,
                state: &state,
            }) {
                Ok(receipt) => receipt,
                Err(ctx) => {
                    let receipt = alloy_consensus::Receipt {
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
                        deposit_receipt_version: (is_deposit &&
                            self.spec.is_canyon_active_at_timestamp(
                                self.evm.block().timestamp().saturating_to(),
                            ))
                        .then_some(1),
                    })
                }
            },
        );

        self.evm.db_mut().commit(state);

        Ok(GasOutput::new(canonical_gas_used))
    }

    fn finish(
        mut self,
    ) -> Result<(Self::Evm, BlockExecutionResult<R::Receipt>), BlockExecutionError> {
        let indexes = self.post_exec.remaining_verifier_indexes();
        if !indexes.is_empty() {
            return Err(Self::invalid_post_exec_payload(format!(
                "{} unconsumed post-exec payload entries for tx indexes {:?}",
                indexes.len(),
                indexes,
            )));
        }

        let balance_increments =
            post_block_balance_increments::<Header>(&self.spec, self.evm.block(), &[], None);
        // increment balances
        self.evm
            .db_mut()
            .increment_balances(balance_increments.clone())
            .map_err(|_| BlockValidationError::IncrementBalanceFailed)?;
        // call state hook with changes due to balance increments.
        self.system_caller.try_on_state_with(|| {
            balance_increment_state(&balance_increments, self.evm.db_mut()).map(|state| {
                (
                    StateChangeSource::PostBlock(StateChangePostBlockSource::BalanceIncrements),
                    Cow::Owned(state),
                )
            })
        })?;

        Ok((
            self.evm,
            BlockExecutionResult {
                receipts: self.receipts,
                requests: Requests::default(),
                gas_used: self.gas_used,
                blob_gas_used: self.da_footprint_used,
            },
        ))
    }

    fn set_state_hook(&mut self, hook: Option<Box<dyn OnStateHook>>) {
        self.system_caller.with_state_hook(hook);
    }

    fn evm_mut(&mut self) -> &mut Self::Evm {
        &mut self.evm
    }

    fn evm(&self) -> &Self::Evm {
        &self.evm
    }

    fn receipts(&self) -> &[Self::Receipt] {
        &self.receipts
    }
}

/// Ethereum block executor factory.
#[derive(Debug, Clone, Default, Copy)]
pub struct OpBlockExecutorFactory<
    R = OpAlloyReceiptBuilder,
    Spec = OpChainHardforks,
    EvmFactory = OpEvmFactory,
> {
    /// Receipt builder.
    receipt_builder: R,
    /// Chain specification.
    spec: Spec,
    /// EVM factory.
    evm_factory: EvmFactory,
}

impl<R, Spec, EvmFactory> OpBlockExecutorFactory<R, Spec, EvmFactory> {
    /// Creates a new [`OpBlockExecutorFactory`] with the given spec, [`EvmFactory`], and
    /// [`OpReceiptBuilder`].
    pub const fn new(receipt_builder: R, spec: Spec, evm_factory: EvmFactory) -> Self {
        Self { receipt_builder, spec, evm_factory }
    }

    /// Exposes the receipt builder.
    pub const fn receipt_builder(&self) -> &R {
        &self.receipt_builder
    }

    /// Exposes the chain specification.
    pub const fn spec(&self) -> &Spec {
        &self.spec
    }

    /// Exposes the EVM factory.
    pub const fn evm_factory(&self) -> &EvmFactory {
        &self.evm_factory
    }
}

impl<R, Spec, F> BlockExecutorFactory
    for OpBlockExecutorFactory<R, Spec, PostExecEvmFactoryAdapter<F>>
where
    R: OpReceiptBuilder<
            Transaction: Transaction + Encodable2718 + OpConsensusTransaction,
            Receipt: TxReceipt,
        >,
    Spec: OpHardforks,
    F: PostExecEvmFactoryHooks,
    F::Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
    Self: 'static,
{
    type EvmFactory = PostExecEvmFactoryAdapter<F>;
    type ExecutionCtx<'a> = OpBlockExecutionCtx;
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: <PostExecEvmFactoryAdapter<F> as EvmFactory>::Evm<DB, I>,
        ctx: Self::ExecutionCtx<'a>,
    ) -> impl BlockExecutorFor<'a, Self, DB, I>
    where
        DB: StateDB + 'a,
        I: Inspector<<PostExecEvmFactoryAdapter<F> as EvmFactory>::Context<DB>> + 'a,
    {
        OpBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder)
    }
}

impl<R, Spec, Tx> BlockExecutorFactory for OpBlockExecutorFactory<R, Spec, OpEvmFactory<Tx>>
where
    R: OpReceiptBuilder<
            Transaction: Transaction + Encodable2718 + OpConsensusTransaction,
            Receipt: TxReceipt,
        >,
    Spec: OpHardforks,
    Tx: IntoTxEnv<Tx>
        + Into<OpTransaction<TxEnv>>
        + Default
        + Clone
        + core::fmt::Debug
        + FromRecoveredTx<R::Transaction>
        + FromTxWithEncoded<R::Transaction>
        + OpTxEnv,
    Self: 'static,
{
    type EvmFactory = OpEvmFactory<Tx>;
    type ExecutionCtx<'a> = OpBlockExecutionCtx;
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: <OpEvmFactory<Tx> as EvmFactory>::Evm<DB, I>,
        ctx: Self::ExecutionCtx<'a>,
    ) -> impl BlockExecutorFor<'a, Self, DB, I>
    where
        DB: StateDB + 'a,
        I: Inspector<<OpEvmFactory<Tx> as EvmFactory>::Context<DB>> + 'a,
    {
        OpBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder)
    }
}

#[cfg(test)]
mod tests;
