//! Block executor for Optimism.

use crate::{OpEvmFactory, spec_by_timestamp_after_bedrock};
use alloc::{
    borrow::Cow, boxed::Box, collections::BTreeMap, format, string::String, vec, vec::Vec,
};
use alloy_consensus::{Eip658Value, Header, Transaction, TransactionEnvelope, TxReceipt};
use alloy_eips::{Encodable2718, Typed2718};
use alloy_evm::{
    Database, Evm, EvmFactory, FromRecoveredTx, FromTxWithEncoded, RecoveredTx,
    block::{
        BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
        BlockExecutorFor, BlockValidationError, ExecutableTx, OnStateHook,
        StateChangePostBlockSource, StateChangeSource, StateDB, SystemCaller, TxResult,
        state_changes::{balance_increment_state, post_block_balance_increments},
    },
    eth::{EthTxResult, receipt_builder::ReceiptBuilderCtx},
};
use alloy_op_hardforks::{OpChainHardforks, OpHardforks};
use alloy_primitives::{Address, B256, Bytes, U256};
use canyon::ensure_create2_deployer;
use op_alloy::consensus::{OpDepositReceipt, POST_EXEC_TX_TYPE_ID, PostExecPayload, SDMGasEntry};
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
        Block,
        result::{ExecutionResult, Output, ResultAndState, SuccessReason},
    },
    database::DatabaseCommitExt,
    state::{Account, AccountStatus, EvmState},
};

use crate::post_exec::{PostExecExecutedTx, PostExecTxContext, PostExecTxKind, WarmingRefundEvent};

mod canyon;
pub mod receipt_builder;

/// Default no-op hook installed by [`OpBlockExecutor::new`] for Produce-mode tracking.
///
/// Kept as a named fn item (not a closure) so `apply_pre_execution_changes` can identity-
/// compare the installed hook against this default via [`core::ptr::fn_addr_eq`] and
/// `debug_assert!` that callers wired the real inspector before driving execution in
/// `PostExecMode::Produce`.
const fn default_begin_post_exec_tx<E: Evm>(_: &mut E, _: PostExecTxContext) {}

/// Default no-op hook installed by [`OpBlockExecutor::new`] for Produce-mode result take.
///
/// See [`default_begin_post_exec_tx`] — paired with it for the same identity-compare.
const fn default_take_last_post_exec_tx_result<E: Evm>(_: &mut E) -> PostExecExecutedTx {
    PostExecExecutedTx { refund_total: 0, refund_events: Vec::new() }
}

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
    /// An post-exec tx was present but invalid, so block execution must fail.
    Invalid,
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

/// Canonical gas adjustment applied to a transaction when a post-exec refund reduces its gas
/// cost below the raw EVM result.
#[derive(Debug, Default, Clone)]
pub struct PostExecAdjustment {
    /// Refund amount subtracted from raw gas to produce canonical gas.
    pub refund: u64,
    /// Sender balance delta to credit back.
    pub sender_refund: U256,
    /// Beneficiary balance delta to debit.
    pub beneficiary_delta: U256,
    /// Base fee recipient balance delta to debit.
    pub base_fee_delta: U256,
    /// Operator fee recipient balance delta to debit.
    pub operator_fee_delta: U256,
    /// Exact warming refund attribution events that produced `refund` (populated in Produce
    /// mode; empty in Verify mode where the refund comes from the embedded payload).
    pub warming_events: Vec<WarmingRefundEvent>,
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
    /// Raw gas used returned by normal EVM execution before any canonical post-exec adjustment.
    pub raw_gas_used: u64,
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

    // -- post-exec Block-Level Warming fields --
    /// Accumulated per-tx warming refunds for post-exec tx assembly (sequencer mode).
    pub post_exec_entries: Vec<SDMGasEntry>,
    /// Active post-exec execution mode.
    pub post_exec_mode: PostExecMode,
    /// Verifier payload indexed by original tx index.
    pub post_exec_verify_entries: BTreeMap<u64, u64>,
    /// Invalid verifier payload reason, if any.
    pub post_exec_invalid_reason: Option<String>,
    /// Begin post-exec tracking for the next transaction.
    pub begin_post_exec_tx: fn(&mut Evm, PostExecTxContext),
    /// Extractor for the most recent transaction's exact warming result.
    pub take_last_post_exec_tx_result: fn(&mut Evm) -> PostExecExecutedTx,
    /// Per-transaction exact warming refund attribution events aligned with receipts.
    pub warming_events_by_tx: Vec<Vec<WarmingRefundEvent>>,
}

impl<E, R, Spec> OpBlockExecutor<E, R, Spec>
where
    E: Evm,
    R: OpReceiptBuilder,
    Spec: OpHardforks + Clone,
{
    /// Creates a new [`OpBlockExecutor`].
    pub fn new(evm: E, ctx: OpBlockExecutionCtx, spec: Spec, receipt_builder: R) -> Self {
        let (post_exec_mode, post_exec_verify_entries, post_exec_invalid_reason) =
            Self::init_post_exec_state(ctx.post_exec_mode.clone());
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
            post_exec_entries: Vec::new(),
            post_exec_mode,
            post_exec_verify_entries,
            post_exec_invalid_reason,
            begin_post_exec_tx: default_begin_post_exec_tx::<E>,
            take_last_post_exec_tx_result: default_take_last_post_exec_tx_result::<E>,
            warming_events_by_tx: Vec::new(),
        }
    }

    /// Configure how the executor should begin inspector-backed post-exec tracking.
    pub fn with_post_exec_begin(
        mut self,
        begin_post_exec_tx: fn(&mut E, PostExecTxContext),
    ) -> Self {
        self.begin_post_exec_tx = begin_post_exec_tx;
        self
    }

    /// Configure how the executor should read the most recent inspector-backed post-exec result.
    pub fn with_post_exec_result(
        mut self,
        take_last_post_exec_tx_result: fn(&mut E) -> PostExecExecutedTx,
    ) -> Self {
        self.take_last_post_exec_tx_result = take_last_post_exec_tx_result;
        self
    }

    /// Set the post-exec execution mode for the executor.
    pub fn with_post_exec_mode(mut self, post_exec_mode: PostExecMode) -> Self {
        self.set_post_exec_mode(post_exec_mode);
        self
    }

    fn init_post_exec_state(
        post_exec_mode: PostExecMode,
    ) -> (PostExecMode, BTreeMap<u64, u64>, Option<String>) {
        let mut post_exec_verify_entries = BTreeMap::new();
        let mut post_exec_invalid_reason = None;

        if let PostExecMode::Verify(payload) = &post_exec_mode {
            for entry in &payload.gas_refund_entries {
                if post_exec_verify_entries.insert(entry.index, entry.gas_refund).is_some() {
                    post_exec_invalid_reason = Some(format!(
                        "duplicate post-exec payload entry for tx index {}",
                        entry.index
                    ));
                    break;
                }
            }
        }

        (post_exec_mode, post_exec_verify_entries, post_exec_invalid_reason)
    }

    /// Set the post-exec execution mode for the executor.
    ///
    /// This is primarily intended for tests and replay tooling that need to override the
    /// block-context default after construction.
    pub fn set_post_exec_mode(&mut self, post_exec_mode: PostExecMode) {
        let (post_exec_mode, post_exec_verify_entries, post_exec_invalid_reason) =
            Self::init_post_exec_state(post_exec_mode);
        self.post_exec_mode = post_exec_mode;
        self.post_exec_verify_entries = post_exec_verify_entries;
        self.post_exec_invalid_reason = post_exec_invalid_reason;
    }

    /// Take the accumulated post-exec entries (sequencer mode).
    /// Returns the entries and clears the internal state.
    pub fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry> {
        core::mem::take(&mut self.post_exec_entries)
    }

    /// Take the exact per-transaction warming refund attribution events aligned with receipts.
    pub fn take_warming_events_by_tx(&mut self) -> Vec<Vec<WarmingRefundEvent>> {
        core::mem::take(&mut self.warming_events_by_tx)
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
    R: OpReceiptBuilder<Transaction: Transaction + Encodable2718, Receipt: TxReceipt>,
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

    fn invalid_post_exec_payload(&self, reason: impl Into<String>) -> BlockExecutionError {
        BlockExecutionError::Validation(BlockValidationError::Other(Box::new(
            OpBlockExecutionError::InvalidPostExecPayload(reason.into()),
        )))
    }

    fn verifier_post_exec_refund_for_tx(
        &self,
        tx_index: u64,
        is_deposit: bool,
        is_post_exec: bool,
        raw_gas_used: u64,
    ) -> Result<u64, BlockExecutionError> {
        if !matches!(self.post_exec_mode, PostExecMode::Verify(_)) {
            return Ok(0);
        }

        let Some(refund) = self.post_exec_verify_entries.get(&tx_index).copied() else {
            return Ok(0);
        };

        if is_deposit {
            return Err(self.invalid_post_exec_payload(format!(
                "payload entry targets deposit tx index {tx_index}"
            )));
        }

        if is_post_exec {
            return Err(self.invalid_post_exec_payload(format!(
                "payload entry targets post-exec tx index {tx_index}"
            )));
        }

        if refund > raw_gas_used {
            return Err(self.invalid_post_exec_payload(format!(
                "payload refund {refund} exceeds raw gas used {raw_gas_used} for tx index {tx_index}"
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
                    .with_spent(gas.spent().saturating_sub(post_exec_refund))
                    .with_refunded(gas.inner_refunded().saturating_add(post_exec_refund));
            }
            ExecutionResult::Revert { gas, .. } | ExecutionResult::Halt { gas, .. } => {
                *gas = gas.with_spent(gas.spent().saturating_sub(post_exec_refund));
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

    fn post_exec_settlement_deltas(
        &mut self,
        tx: impl RecoveredTx<R::Transaction>,
        raw_gas_used: u64,
        canonical_gas_used: u64,
        is_deposit: bool,
        is_post_exec: bool,
    ) -> Result<(U256, U256, U256, U256), BlockExecutionError> {
        if is_deposit || is_post_exec || canonical_gas_used >= raw_gas_used {
            return Ok((U256::ZERO, U256::ZERO, U256::ZERO, U256::ZERO));
        }

        let gas_delta = raw_gas_used.saturating_sub(canonical_gas_used);
        let gas_delta_u256 = U256::from(gas_delta);
        let basefee = self.evm.block().basefee() as u128;
        let spec_id = spec_by_timestamp_after_bedrock(
            &self.spec,
            self.evm.block().timestamp().saturating_to(),
        );
        let effective_gas_price = tx.tx().effective_gas_price(Some(self.evm.block().basefee()));
        // SDM/PostExec is only enabled on forks after Isthmus, which is already post-London.
        // A saturating_sub landing at zero is intentional and consensus-valid: a legacy tx
        // with a gas price equal to the basefee pays zero priority fee, so the beneficiary
        // delta below must be zero as well — we credit back only what the beneficiary
        // actually received for the refunded gas, which is the (effective_price - basefee)
        // component.
        let beneficiary_gas_price = effective_gas_price.saturating_sub(basefee);

        let base_fee_delta = gas_delta_u256.saturating_mul(U256::from(basefee));
        let beneficiary_delta = gas_delta_u256.saturating_mul(U256::from(beneficiary_gas_price));

        let l1_block_info = self.l1_block_info(spec_id)?;
        let encoded = tx.tx().encoded_2718();
        let raw_fee =
            l1_block_info.operator_fee_charge(encoded.as_ref(), U256::from(raw_gas_used), spec_id);
        let canonical_fee = l1_block_info.operator_fee_charge(
            encoded.as_ref(),
            U256::from(canonical_gas_used),
            spec_id,
        );
        let operator_fee_delta = raw_fee.saturating_sub(canonical_fee);

        let sender_refund = gas_delta_u256
            .saturating_mul(U256::from(effective_gas_price))
            .saturating_add(operator_fee_delta);

        Ok((sender_refund, beneficiary_delta, base_fee_delta, operator_fee_delta))
    }

    fn apply_post_exec_refund_to_state(
        &mut self,
        state: &mut EvmState,
        sender: Address,
        sender_refund: U256,
        beneficiary_delta: U256,
        base_fee_delta: U256,
        operator_fee_delta: U256,
    ) -> Result<(), BlockExecutionError> {
        let beneficiary = self.evm.block().beneficiary();
        Self::add_state_balance(self.evm.db_mut(), state, sender, sender_refund)?;
        Self::sub_state_balance(self.evm.db_mut(), state, beneficiary, beneficiary_delta)?;
        Self::sub_state_balance(self.evm.db_mut(), state, BASE_FEE_RECIPIENT, base_fee_delta)?;
        Self::sub_state_balance(
            self.evm.db_mut(),
            state,
            OPERATOR_FEE_RECIPIENT,
            operator_fee_delta,
        )?;

        Ok(())
    }
}

impl<E, R, Spec> BlockExecutor for OpBlockExecutor<E, R, Spec>
where
    E: Evm<
            DB: Database + DatabaseCommit + StateDB,
            Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
        >,
    R: OpReceiptBuilder<Transaction: Transaction + Encodable2718, Receipt: TxReceipt>,
    Spec: OpHardforks,
{
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;
    type Evm = E;
    type Result = OpTxResult<E::HaltReason, <R::Transaction as TransactionEnvelope>::TxType>;

    fn apply_pre_execution_changes(&mut self) -> Result<(), BlockExecutionError> {
        if matches!(self.post_exec_mode, PostExecMode::Invalid) {
            return Err(self.invalid_post_exec_payload("post-exec tx payload could not be decoded"));
        }
        if let Some(reason) = &self.post_exec_invalid_reason {
            return Err(self.invalid_post_exec_payload(String::from(reason.as_str())));
        }
        if let PostExecMode::Verify(payload) = &self.post_exec_mode {
            let block_number = self.evm.block().number().saturating_to::<u64>();
            if payload.block_number != block_number {
                let reason = format!(
                    "payload block number {} does not match block number {}",
                    payload.block_number, block_number,
                );
                return Err(self.invalid_post_exec_payload(reason));
            }
        }

        // Produce mode drives refund accounting through the begin/take hooks; if a caller
        // forgets to wire them the executor silently drops all refunds, which would diverge
        // this node from any peer that *did* wire them. OpEvm auto-wires in-tree (see
        // `ConfigurePostExecEvm` in lib.rs); this guard catches downstream forks that
        // bypass the builder.
        if matches!(self.post_exec_mode, PostExecMode::Produce) {
            debug_assert!(
                !core::ptr::fn_addr_eq(
                    self.begin_post_exec_tx,
                    default_begin_post_exec_tx::<E> as fn(&mut E, PostExecTxContext),
                ),
                "PostExecMode::Produce requires begin_post_exec_tx to be wired via \
                 with_post_exec_begin; the default no-op would silently drop refunds",
            );
            debug_assert!(
                !core::ptr::fn_addr_eq(
                    self.take_last_post_exec_tx_result,
                    default_take_last_post_exec_tx_result::<E> as fn(&mut E) -> PostExecExecutedTx,
                ),
                "PostExecMode::Produce requires take_last_post_exec_tx_result to be wired \
                 via with_post_exec_result; the default no-op would silently drop refunds",
            );
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

        if is_post_exec {
            // Validates that no Verify payload entry targets this tx index; refund is always 0.
            self.verifier_post_exec_refund_for_tx(tx_index, false, true, 0)?;
            return Ok(OpTxResult {
                inner: EthTxResult {
                    result: ResultAndState::new(
                        ExecutionResult::Success {
                            reason: SuccessReason::Stop,
                            gas: revm::context::result::ResultGas::new(0, 0, 0, 0, 0),
                            logs: vec![],
                            output: Output::Call(Bytes::default()),
                        },
                        Default::default(),
                    ),
                    blob_gas_used: 0,
                    tx_type: tx.tx().tx_type(),
                },
                is_deposit: false,
                is_post_exec: true,
                sender: *tx.signer(),
                raw_gas_used: 0,
                post_exec: None,
            });
        }

        if matches!(self.post_exec_mode, PostExecMode::Produce) {
            (self.begin_post_exec_tx)(
                &mut self.evm,
                PostExecTxContext {
                    tx_index,
                    kind: if is_deposit { PostExecTxKind::Deposit } else { PostExecTxKind::Normal },
                },
            );
        }

        // Execute transaction and return the result
        let result = self.evm.transact(tx_env).map_err(|err| {
            let hash = tx.tx().trie_hash();
            BlockExecutionError::evm(err, hash)
        })?;

        let raw_gas_used = result.result.gas_used();
        let (post_exec_refund, warming_events) = match &self.post_exec_mode {
            PostExecMode::Produce => {
                let PostExecExecutedTx { refund_total, refund_events } =
                    (self.take_last_post_exec_tx_result)(&mut self.evm);
                (refund_total, refund_events)
            }
            PostExecMode::Verify(_) => (
                self.verifier_post_exec_refund_for_tx(tx_index, is_deposit, false, raw_gas_used)?,
                Vec::new(),
            ),
            PostExecMode::Disabled | PostExecMode::Invalid => (0, Vec::new()),
        };
        let canonical_gas_used = raw_gas_used.saturating_sub(post_exec_refund);
        let (sender_refund, beneficiary_delta, base_fee_delta, operator_fee_delta) = self
            .post_exec_settlement_deltas(
                &tx,
                raw_gas_used,
                canonical_gas_used,
                is_deposit,
                false,
            )?;

        let post_exec =
            (post_exec_refund > 0 || !warming_events.is_empty()).then_some(PostExecAdjustment {
                refund: post_exec_refund,
                sender_refund,
                beneficiary_delta,
                base_fee_delta,
                operator_fee_delta,
                warming_events,
            });

        Ok(OpTxResult {
            inner: EthTxResult {
                result,
                blob_gas_used: da_footprint_used,
                tx_type: tx.tx().tx_type(),
            },
            is_deposit,
            is_post_exec: false,
            sender: *tx.signer(),
            raw_gas_used,
            post_exec,
        })
    }

    fn commit_transaction(&mut self, output: Self::Result) -> Result<u64, BlockExecutionError> {
        let tx_index = self.receipts.len() as u64;
        let OpTxResult {
            inner:
                EthTxResult { result: ResultAndState { mut result, mut state }, blob_gas_used, tx_type },
            is_deposit,
            is_post_exec,
            sender,
            raw_gas_used,
            post_exec,
        } = output;

        let PostExecAdjustment {
            refund: post_exec_refund,
            sender_refund,
            beneficiary_delta,
            base_fee_delta,
            operator_fee_delta,
            warming_events,
        } = post_exec.unwrap_or_default();

        if !is_deposit &&
            !is_post_exec &&
            matches!(self.post_exec_mode, PostExecMode::Produce) &&
            post_exec_refund > 0
        {
            let entry = SDMGasEntry { index: tx_index, gas_refund: post_exec_refund };
            self.post_exec_entries.push(entry);
        }
        if matches!(self.post_exec_mode, PostExecMode::Verify(_)) && post_exec_refund > 0 {
            self.post_exec_verify_entries.remove(&tx_index);
        }
        // Skip push for the synthetic 0x7D tx: its execute path returns early with an empty
        // `warming_events`, and the replay consumer (`post-exec-replay::replay_block`) runs
        // against the stripped block so this index is never addressed. Deposit pushes stay
        // because replay relies on positional alignment between the stripped block's
        // transactions and `warming_events_by_tx`.
        if !is_post_exec {
            self.warming_events_by_tx.push(warming_events);
        }

        // Fetch the depositor account from the database for the deposit nonce.
        // Note that this *only* needs to be done post-regolith hardfork, as deposit nonces
        // were not introduced in Bedrock. In addition, regular transactions don't have deposit
        // nonces, so we don't need to touch the DB for those.
        let depositor = (self.is_regolith && is_deposit)
            .then(|| self.evm.db_mut().basic(sender).map(|acc| acc.unwrap_or_default()))
            .transpose()
            .map_err(BlockExecutionError::other)?;

        let canonical_gas_used = raw_gas_used.saturating_sub(post_exec_refund);
        Self::canonicalize_result_gas(&mut result, post_exec_refund);
        self.apply_post_exec_refund_to_state(
            &mut state,
            sender,
            sender_refund,
            beneficiary_delta,
            base_fee_delta,
            operator_fee_delta,
        )?;

        self.system_caller.on_state(StateChangeSource::Transaction(self.receipts.len()), &state);

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

        Ok(canonical_gas_used)
    }

    fn finish(
        mut self,
    ) -> Result<(Self::Evm, BlockExecutionResult<R::Receipt>), BlockExecutionError> {
        if !self.post_exec_verify_entries.is_empty() {
            let indexes: Vec<u64> = self.post_exec_verify_entries.keys().copied().collect();
            return Err(self.invalid_post_exec_payload(format!(
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
                requests: Default::default(),
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

impl<R, Spec, EvmF> BlockExecutorFactory for OpBlockExecutorFactory<R, Spec, EvmF>
where
    R: OpReceiptBuilder<Transaction: Transaction + Encodable2718, Receipt: TxReceipt>,
    Spec: OpHardforks,
    EvmF: EvmFactory<
        Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
    >,
    Self: 'static,
{
    type EvmFactory = EvmF;
    type ExecutionCtx<'a> = OpBlockExecutionCtx;
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: EvmF::Evm<DB, I>,
        ctx: Self::ExecutionCtx<'a>,
    ) -> impl BlockExecutorFor<'a, Self, DB, I>
    where
        DB: StateDB + 'a,
        I: Inspector<EvmF::Context<DB>> + 'a,
    {
        OpBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder)
    }
}

#[cfg(test)]
mod tests {
    use alloc::{string::ToString, vec};
    use alloy_consensus::{SignableTransaction, TxLegacy, transaction::Recovered};
    use alloy_eips::eip2718::WithEncoded;
    use alloy_evm::{EvmEnv, ToTxEnv};
    use alloy_hardforks::ForkCondition;
    use alloy_op_hardforks::OpHardfork;
    use alloy_primitives::{Address, Signature, U256, uint};
    use op_alloy::consensus::OpTxEnvelope;
    use op_revm::{
        DefaultOp, L1BlockInfo, OpBuilder, OpSpecId,
        constants::{
            BASE_FEE_SCALAR_OFFSET, ECOTONE_L1_BLOB_BASE_FEE_SLOT, ECOTONE_L1_FEE_SCALARS_SLOT,
            L1_BASE_FEE_SLOT, L1_BLOCK_CONTRACT, OPERATOR_FEE_SCALARS_SLOT,
        },
    };
    use revm::{
        Context,
        context::BlockEnv,
        database::{CacheDB, EmptyDB, InMemoryDB, State},
        inspector::NoOpInspector,
        primitives::HashMap,
        state::AccountInfo,
    };

    use crate::OpEvm;

    use super::*;

    #[test]
    fn test_with_encoded() {
        let executor_factory = OpBlockExecutorFactory::new(
            OpAlloyReceiptBuilder::default(),
            OpChainHardforks::op_mainnet(),
            OpEvmFactory::<crate::OpTx>::default(),
        );
        let mut db = State::builder().with_database(CacheDB::<EmptyDB>::default()).build();
        let evm = executor_factory.evm_factory.create_evm(&mut db, EvmEnv::default());
        let mut executor = executor_factory.create_executor(evm, OpBlockExecutionCtx::default());
        let tx = Recovered::new_unchecked(
            OpTxEnvelope::Legacy(TxLegacy::default().into_signed(Signature::new(
                Default::default(),
                Default::default(),
                Default::default(),
            ))),
            Address::ZERO,
        );
        let tx_with_encoded = WithEncoded::new(tx.encoded_2718().into(), tx.clone());

        // make sure we can use both `WithEncoded` and transaction itself as inputs.
        let _ = executor.execute_transaction(&tx);
        let _ = executor.execute_transaction(&tx_with_encoded);
    }

    #[test]
    fn test_settlement_state_account_preserves_original_info() {
        type TestExecutor<'a> = OpBlockExecutor<
            OpEvm<&'a mut State<InMemoryDB>, NoOpInspector>,
            &'a OpAlloyReceiptBuilder,
            &'a OpChainHardforks,
        >;

        let mut backing_db = InMemoryDB::default();
        backing_db.insert_account_info(
            BASE_FEE_RECIPIENT,
            AccountInfo { balance: U256::from(10), ..Default::default() },
        );
        let mut db = State::builder().with_database(backing_db).with_bundle_update().build();
        revm::Database::basic(&mut db, BASE_FEE_RECIPIENT)
            .expect("failed to load base fee recipient into cache");

        let mut credited_account =
            Account::from(AccountInfo { balance: U256::from(15), ..Default::default() });
        credited_account.mark_touch();
        revm::DatabaseCommit::commit(
            &mut db,
            HashMap::from_iter([(BASE_FEE_RECIPIENT, credited_account)]),
        );

        let mut state = EvmState::default();
        let mut db_ref = &mut db;
        let account = TestExecutor::state_account_mut(&mut db_ref, &mut state, BASE_FEE_RECIPIENT)
            .expect("failed to materialize settlement account");
        assert_eq!(account.info.balance, U256::from(15));
        // original_info mirrors current info here — State::commit computes the
        // true previous value from its own cache, so the bundle stays correct.
        assert_eq!(account.original_info.balance, U256::from(15));

        account.info.balance = account.info.balance.saturating_sub(U256::from(3));
        revm::DatabaseCommit::commit(&mut db, state);
        db.merge_transitions(revm::database::states::bundle_state::BundleRetention::Reverts);

        let bundle = db.take_bundle();
        let bundle_account = bundle
            .account(&BASE_FEE_RECIPIENT)
            .expect("bundle must contain the base fee recipient");
        assert_eq!(bundle_account.original_info.as_ref().unwrap().balance, U256::from(10));
        assert_eq!(bundle_account.info.as_ref().unwrap().balance, U256::from(12));
    }

    fn prepare_jovian_db(da_footprint_gas_scalar: u16) -> State<InMemoryDB> {
        const L1_BASE_FEE: U256 = uint!(1_U256);
        const L1_BLOB_BASE_FEE: U256 = uint!(2_U256);
        const L1_BASE_FEE_SCALAR: u64 = 3;
        const L1_BLOB_BASE_FEE_SCALAR: u64 = 4;
        const L1_FEE_SCALARS: U256 = U256::from_limbs([
            0,
            (L1_BASE_FEE_SCALAR << (64 - BASE_FEE_SCALAR_OFFSET * 2)) | L1_BLOB_BASE_FEE_SCALAR,
            0,
            0,
        ]);
        const OPERATOR_FEE_SCALAR: u8 = 5;
        const OPERATOR_FEE_CONST: u8 = 6;
        let da_footprint_gas_scalar_bytes = da_footprint_gas_scalar.to_be_bytes();
        let mut operator_fee_and_da_footprint = [0u8; 32];
        operator_fee_and_da_footprint[31] = OPERATOR_FEE_CONST;
        operator_fee_and_da_footprint[23] = OPERATOR_FEE_SCALAR;
        operator_fee_and_da_footprint[19] = da_footprint_gas_scalar_bytes[1];
        operator_fee_and_da_footprint[18] = da_footprint_gas_scalar_bytes[0];
        let operator_fee_and_da_footprint_u256 = U256::from_be_bytes(operator_fee_and_da_footprint);

        let mut db = State::builder().with_database(InMemoryDB::default()).build();

        db.insert_account_with_storage(
            L1_BLOCK_CONTRACT,
            Default::default(),
            HashMap::from_iter([
                (L1_BASE_FEE_SLOT, L1_BASE_FEE),
                (ECOTONE_L1_FEE_SCALARS_SLOT, L1_FEE_SCALARS),
                (ECOTONE_L1_BLOB_BASE_FEE_SLOT, L1_BLOB_BASE_FEE),
                (OPERATOR_FEE_SCALARS_SLOT, operator_fee_and_da_footprint_u256),
            ]),
        );

        db.insert_account(
            Address::ZERO,
            AccountInfo { balance: U256::from(400_000_000), ..Default::default() },
        );

        db
    }

    fn build_executor<'a>(
        db: &'a mut State<InMemoryDB>,
        receipt_builder: &'a OpAlloyReceiptBuilder,
        op_chain_hardforks: &'a OpChainHardforks,
        gas_limit: u64,
        jovian_timestamp: u64,
    ) -> OpBlockExecutor<
        OpEvm<
            &'a mut State<InMemoryDB>,
            NoOpInspector,
            op_revm::precompiles::OpPrecompiles,
            crate::OpTx,
        >,
        &'a OpAlloyReceiptBuilder,
        &'a OpChainHardforks,
    > {
        let ctx = Context::op()
            .with_db(db)
            .with_chain(L1BlockInfo {
                operator_fee_scalar: Some(U256::from(2)),
                operator_fee_constant: Some(U256::from(50)),
                ..Default::default()
            })
            .with_block(BlockEnv {
                timestamp: U256::from(jovian_timestamp),
                gas_limit,
                ..Default::default()
            })
            .modify_cfg_chained(|cfg| cfg.spec = OpSpecId::JOVIAN);

        let evm = OpEvm::new(ctx.build_op_with_inspector(NoOpInspector {}), true);

        OpBlockExecutor::new(
            evm,
            OpBlockExecutionCtx::default(),
            op_chain_hardforks,
            receipt_builder,
        )
    }

    #[test]
    fn test_jovian_da_footprint_estimation() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );

        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );

        let tx_inner = TxLegacy { gas_limit: GAS_LIMIT, ..Default::default() };

        let tx = Recovered::new_unchecked(
            OpTxEnvelope::Legacy(tx_inner.into_signed(Signature::new(
                Default::default(),
                Default::default(),
                Default::default(),
            ))),
            Address::ZERO,
        );
        let tx_env = tx.to_tx_env();

        assert!(executor.da_footprint_used == 0);

        let expected_da_footprint = executor.jovian_da_footprint_estimation(&tx_env, &tx).unwrap();

        // make sure we can use both `WithEncoded` and transaction itself as inputs.
        let res = executor.execute_transaction(&tx);
        assert!(res.is_ok());

        assert!(executor.da_footprint_used == expected_da_footprint);
    }

    #[test]
    fn test_jovian_da_footprint_estimation_out_of_gas() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;
        const GAS_LIMIT: u64 = 100;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );

        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );

        let tx_inner = TxLegacy { gas_limit: GAS_LIMIT, ..Default::default() };

        let tx = Recovered::new_unchecked(
            OpTxEnvelope::Legacy(tx_inner.into_signed(Signature::new(
                Default::default(),
                Default::default(),
                Default::default(),
            ))),
            Address::ZERO,
        );
        let tx_env = tx.to_tx_env();

        assert!(executor.da_footprint_used == 0);

        let expected_da_footprint = executor.jovian_da_footprint_estimation(&tx_env, &tx).unwrap();

        // make sure we can use both `WithEncoded` and transaction itself as inputs.
        let res = executor.execute_transaction(&tx);
        assert!(res.is_err());
        let err = res.unwrap_err();
        match err {
            BlockExecutionError::Validation(BlockValidationError::Other(err)) => {
                assert_eq!(
                    err.to_string(),
                    OpBlockExecutionError::TransactionDaFootprintAboveGasLimit {
                        transaction_da_footprint: expected_da_footprint,
                        available_block_da_footprint: GAS_LIMIT,
                    }
                    .to_string(),
                );
            }
            _ => panic!("expected TransactionDaFootprintAboveGasLimit error"),
        }
    }

    #[test]
    fn test_jovian_da_footprint_estimation_maxed_out_da_footprint() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 2000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;
        const GAS_LIMIT: u64 = 200_000;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );

        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );

        let tx_inner = TxLegacy { gas_limit: GAS_LIMIT, ..Default::default() };

        let tx = Recovered::new_unchecked(
            OpTxEnvelope::Legacy(tx_inner.into_signed(Signature::new(
                Default::default(),
                Default::default(),
                Default::default(),
            ))),
            Address::ZERO,
        );
        let tx_env = tx.to_tx_env();

        assert!(executor.da_footprint_used == 0);

        let expected_da_footprint = executor.jovian_da_footprint_estimation(&tx_env, &tx).unwrap();

        // make sure we can use both `WithEncoded` and transaction itself as inputs.
        let gas_used_tx = executor.execute_transaction(&tx).expect("failed to execute transaction");

        // The gas used when executing the transaction should be the legacy value...
        assert!(gas_used_tx < expected_da_footprint);

        // The gas used when finishing the executor should be the DA footprint since this is higher
        // than the legacy gas used and jovian is active...
        let (_, result) = executor.finish().expect("failed to finish executor");
        assert_eq!(result.blob_gas_used, expected_da_footprint);
        assert_eq!(result.gas_used, gas_used_tx);
        assert!(result.blob_gas_used > result.gas_used);
    }

    #[test]
    fn test_invalid_post_exec_mode_fails_pre_execution() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        executor.set_post_exec_mode(PostExecMode::Invalid);

        let err =
            executor.apply_pre_execution_changes().expect_err("invalid post-exec mode must fail");
        match err {
            BlockExecutionError::Validation(BlockValidationError::Other(err)) => {
                assert_eq!(
                    err.to_string(),
                    OpBlockExecutionError::InvalidPostExecPayload(
                        "post-exec tx payload could not be decoded".to_string(),
                    )
                    .to_string(),
                );
            }
            _ => panic!("expected invalid post-exec payload error"),
        }
    }

    #[test]
    #[cfg(debug_assertions)]
    #[should_panic(expected = "PostExecMode::Produce requires begin_post_exec_tx")]
    fn test_produce_mode_without_wired_hooks_debug_asserts() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        // build_executor does not call with_post_exec_begin / with_post_exec_result, so
        // the fn-pointer fields stay pinned to the default_* no-ops.
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        executor.set_post_exec_mode(PostExecMode::Produce);

        // Release builds skip the assert and would silently drop refunds — document that
        // too so anyone removing the assert sees the expected behavior.
        let _ = executor.apply_pre_execution_changes();
    }

    #[test]
    fn test_mismatched_payload_block_number_fails_pre_execution() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        // build_executor configures BlockEnv with block number 0; a payload anchored to a
        // different block must be rejected before any tx runs.
        executor.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 42,
            gas_refund_entries: vec![],
        }));

        let err =
            executor.apply_pre_execution_changes().expect_err("mismatched block number must fail");
        match err {
            BlockExecutionError::Validation(BlockValidationError::Other(err)) => {
                assert_eq!(
                    err.to_string(),
                    OpBlockExecutionError::InvalidPostExecPayload(
                        "payload block number 42 does not match block number 0".to_string(),
                    )
                    .to_string(),
                );
            }
            _ => panic!("expected invalid post-exec payload error"),
        }
    }

    fn assert_invalid_post_exec(err: BlockExecutionError, expected_reason: &str) {
        match err {
            BlockExecutionError::Validation(BlockValidationError::Other(err)) => {
                assert_eq!(
                    err.to_string(),
                    OpBlockExecutionError::InvalidPostExecPayload(expected_reason.to_string())
                        .to_string(),
                );
            }
            other => panic!("expected invalid post-exec payload error, got: {other:?}"),
        }
    }

    #[test]
    fn test_duplicate_payload_index_fails_pre_execution() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        // Two entries colliding on tx index 3 — the second insert must be flagged at construction
        // and surface as a pre-execution failure.
        executor.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 0,
            gas_refund_entries: vec![
                SDMGasEntry { index: 3, gas_refund: 10 },
                SDMGasEntry { index: 3, gas_refund: 20 },
            ],
        }));

        let err = executor
            .apply_pre_execution_changes()
            .expect_err("duplicate payload index must fail pre-execution");
        assert_invalid_post_exec(err, "duplicate post-exec payload entry for tx index 3");
    }

    #[test]
    fn test_verifier_rejects_payload_targeting_deposit_tx() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        executor.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 0,
            gas_refund_entries: vec![SDMGasEntry { index: 0, gas_refund: 1 }],
        }));

        let err = executor
            .verifier_post_exec_refund_for_tx(0, true, false, 21_000)
            .expect_err("payload entries must not target deposit txs");
        assert_invalid_post_exec(err, "payload entry targets deposit tx index 0");
    }

    #[test]
    fn test_verifier_rejects_payload_targeting_post_exec_tx() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        executor.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 0,
            gas_refund_entries: vec![SDMGasEntry { index: 4, gas_refund: 1 }],
        }));

        // A 0x7D tx reaching this helper with an entry at its own index would mean the payload
        // is attributing a refund to the synthetic tx itself — refunds are per-normal-tx only.
        let err = executor
            .verifier_post_exec_refund_for_tx(4, false, true, 0)
            .expect_err("payload entries must not target the post-exec tx itself");
        assert_invalid_post_exec(err, "payload entry targets post-exec tx index 4");
    }

    #[test]
    fn test_verifier_rejects_refund_exceeding_raw_gas() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        executor.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 0,
            gas_refund_entries: vec![SDMGasEntry { index: 2, gas_refund: 50_000 }],
        }));

        // raw_gas_used < payload refund — a refund that exceeds the tx's raw cost is
        // impossible under SDM semantics and must be rejected, otherwise canonical gas
        // would underflow to a bogus value via saturating_sub.
        let err = executor
            .verifier_post_exec_refund_for_tx(2, false, false, 40_000)
            .expect_err("refund greater than raw gas must be rejected");
        assert_invalid_post_exec(
            err,
            "payload refund 50000 exceeds raw gas used 40000 for tx index 2",
        );

        // Boundary: refund == raw_gas_used is permitted (canonical gas ends up at zero).
        let ok = executor
            .verifier_post_exec_refund_for_tx(2, false, false, 50_000)
            .expect("refund equal to raw gas is permitted");
        assert_eq!(ok, 50_000);
    }

    #[test]
    fn test_verifier_returns_zero_when_no_entry_for_tx() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        executor.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 0,
            gas_refund_entries: vec![SDMGasEntry { index: 7, gas_refund: 42 }],
        }));

        // Normal tx that has no entry in the payload — the deposit/post-exec guards must NOT
        // fire, the helper must return 0 so execution proceeds with raw gas unchanged.
        let refund = executor
            .verifier_post_exec_refund_for_tx(3, false, false, 21_000)
            .expect("no entry for this tx index means no refund");
        assert_eq!(refund, 0);
    }

    #[test]
    fn test_finish_reports_all_unconsumed_post_exec_entries() {
        const DA_FOOTPRINT_GAS_SCALAR: u16 = 7;
        const GAS_LIMIT: u64 = 100_000;
        const JOVIAN_TIMESTAMP: u64 = 1746806402;

        let mut db = prepare_jovian_db(DA_FOOTPRINT_GAS_SCALAR);
        let op_chain_hardforks = OpChainHardforks::new(
            OpHardfork::op_mainnet()
                .into_iter()
                .chain(vec![(OpHardfork::Jovian, ForkCondition::Timestamp(JOVIAN_TIMESTAMP))]),
        );
        let receipt_builder = OpAlloyReceiptBuilder::default();
        let mut executor = build_executor(
            &mut db,
            &receipt_builder,
            &op_chain_hardforks,
            GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        executor.set_post_exec_mode(PostExecMode::Verify(PostExecPayload {
            version: 1,
            block_number: 0,
            gas_refund_entries: vec![
                SDMGasEntry { index: 2, gas_refund: 7 },
                SDMGasEntry { index: 5, gas_refund: 11 },
            ],
        }));

        let err = match executor.finish() {
            Ok(_) => panic!("unconsumed verifier entries must fail"),
            Err(err) => err,
        };
        match err {
            BlockExecutionError::Validation(BlockValidationError::Other(err)) => {
                assert_eq!(
                    err.to_string(),
                    OpBlockExecutionError::InvalidPostExecPayload(
                        "2 unconsumed post-exec payload entries for tx indexes [2, 5]".to_string(),
                    )
                    .to_string(),
                );
            }
            _ => panic!("expected invalid post-exec payload error"),
        }
    }
}
