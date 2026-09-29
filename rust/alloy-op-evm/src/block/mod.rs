//! Block executor for Optimism.

use crate::{OpEvmFactory, spec_by_timestamp_after_bedrock};
use alloc::{boxed::Box, format, string::String, vec::Vec};
use alloy_consensus::{Eip658Value, Header, Transaction, TransactionEnvelope, TxReceipt};
use alloy_eips::{Encodable2718, Typed2718, eip7685::Requests};
use alloy_evm::{
    Database, Evm, EvmFactory, FromRecoveredTx, FromTxWithEncoded, IntoTxEnv, RecoveredTx,
    block::{
        BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
        BlockValidationError, CommitChanges, ExecutableTx, GasOutput, StateDB, SystemCaller,
        TxResult, state_changes::post_block_balance_increments,
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
    encoded_tx_da_footprint,
    transaction::deposit::DEPOSIT_TRANSACTION_TYPE,
    tx_da_footprint,
};
pub use receipt_builder::OpAlloyReceiptBuilder;
use receipt_builder::OpReceiptBuilder;
use revm::{
    Database as _, DatabaseCommit, Inspector,
    context::{
        Block, TxEnv,
        result::{ExecutionResult, ResultAndState},
    },
    database::DatabaseCommitExt,
    state::{Account, AccountStatus, EvmState},
};

use crate::post_exec::{
    PostExecEvm, PostExecEvmFactoryAdapter, PostExecEvmFactoryHooks, PostExecExecutedTx,
    PostExecRefundEvent, PostExecRefundInspector, PostExecTxContext, PostExecTxKind,
    noop_post_exec_result,
};

mod canyon;
#[cfg(feature = "parallel")]
mod parallel;
pub mod receipt_builder;
#[cfg(feature = "parallel")]
pub use reth_optimism_parallel::{
    ExecutionMode, ParallelExecutionConfig, ParallelRuntime,
    SpeculationError as ParallelConfigurationError,
};

/// A non-authoritative transaction preview. Inclusion and commit order remain with the caller.
#[derive(Debug, Clone)]
pub struct ParallelCandidate {
    /// Hash used to match this preview with the subsequently selected transaction.
    pub hash: B256,
    /// Complete transaction environment, checked again before a speculative result is reused.
    pub transaction: crate::OpTx,
}

/// Wraps an [`OpBlockExecutionError`] as a block-execution validation error.
fn validation_error(err: OpBlockExecutionError) -> BlockExecutionError {
    BlockExecutionError::Validation(BlockValidationError::Other(Box::new(err)))
}

/// Returns a producer policy's consensus-safe refund for an executed transaction.
///
/// A refund policy is advisory: malformed output must not reject an otherwise valid transaction or
/// abort payload production. A normal-transaction refund that exceeds the gas the EVM actually used
/// is discarded, while deposits are never refundable. Verifiers independently enforce these same
/// bounds on the resulting post-exec payload.
#[cfg_attr(not(feature = "metrics"), allow(clippy::missing_const_for_fn))]
fn sanitize_producer_refund(refund: u64, evm_gas_used: u64, is_deposit: bool) -> u64 {
    let (refund, correction) = if is_deposit && refund > 0 {
        (0, Some("ineligible_transaction"))
    } else if refund > evm_gas_used {
        (0, Some("exceeds_evm_gas"))
    } else {
        (refund, None)
    };

    #[cfg(feature = "metrics")]
    if let Some(reason) = correction {
        metrics::counter!("optimism_sdm.policy_refund_corrections", "reason" => reason)
            .increment(1);
    }
    #[cfg(not(feature = "metrics"))]
    let _ = correction;

    refund
}

/// Trait for OP transaction environments. Allows to recover the transaction encoded bytes if
/// they're available.
pub trait OpTxEnv: Clone {
    /// Returns the encoded bytes of the transaction.
    fn encoded_bytes(&self) -> Option<&Bytes>;

    /// Opts into the standard OP worker execution path. Custom environments default to serial.
    fn parallel_transaction(&self) -> Option<crate::OpTx> {
        None
    }
}

impl<T: revm::context::Transaction + Clone> OpTxEnv for OpTransaction<T> {
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

impl From<bool> for PostExecMode {
    /// `true` opts into local post-exec production; `false` disables it.
    fn from(produce: bool) -> Self {
        if produce { Self::Produce } else { Self::Disabled }
    }
}

/// Per-block post-exec state carried by [`OpBlockExecutor`].
#[derive(Debug)]
pub enum PostExecState {
    /// Execute with legacy gas accounting.
    Disabled,
    /// Produce canonical post-exec refunds locally and append them to the block later.
    Producing {
        /// Accumulated per-tx refunds for post-exec tx assembly.
        entries: Vec<SDMGasEntry>,
    },
    /// Verify canonical gas accounting using a post-exec payload embedded in the block.
    Verifying {
        /// Decoded post-exec payload being verified. Entries are strictly ordered by transaction
        /// index, so verification can consume them with a cursor instead of allocating a map.
        payload: PostExecPayload,
        /// Offset of the next unconsumed payload entry.
        next_entry: usize,
        /// Invalid verifier payload reason, if any.
        invalid_reason: Option<String>,
        /// Whether the block's synthetic post-exec transaction has been seen during execution.
        saw_post_exec_tx: bool,
    },
}

/// Checks the version-1 canonical form of a verifier payload's refund entries: non-empty, no zero
/// refunds, and tx indices strictly increasing. Returns the first violation.
fn validate_verifier_entries(entries: &[SDMGasEntry]) -> Result<(), String> {
    if entries.is_empty() {
        return Err(String::from("empty post-exec payload gas refund entries"));
    }

    let mut previous_index = None;
    for entry in entries {
        if entry.gas_refund == 0 {
            return Err(format!("zero post-exec payload refund for tx index {}", entry.index));
        }
        match previous_index {
            Some(previous) if entry.index == previous => {
                return Err(format!(
                    "duplicate post-exec payload entry for tx index {}",
                    entry.index
                ));
            }
            Some(previous) if entry.index < previous => {
                return Err(format!(
                    "post-exec payload entries not strictly increasing: tx index {} follows {}",
                    entry.index, previous,
                ));
            }
            _ => {}
        }
        previous_index = Some(entry.index);
    }

    Ok(())
}

impl PostExecState {
    fn new(mode: PostExecMode) -> Self {
        match mode {
            PostExecMode::Disabled => Self::Disabled,
            PostExecMode::Produce => Self::Producing { entries: Vec::new() },
            PostExecMode::Verify(payload) => {
                let invalid_reason = validate_verifier_entries(&payload.gas_refund_entries).err();
                Self::Verifying { payload, next_entry: 0, invalid_reason, saw_post_exec_tx: false }
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

    const fn produced_entries(&self) -> &[SDMGasEntry] {
        match self {
            Self::Producing { entries } => entries.as_slice(),
            _ => &[],
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
            Self::Verifying { payload, next_entry, .. } => payload
                .gas_refund_entries
                .get(*next_entry)
                .filter(|entry| entry.index == tx_index)
                .map(|entry| entry.gas_refund),
            _ => None,
        }
    }

    fn consume_verifier_entry(&mut self, tx_index: u64) {
        if let Self::Verifying { payload, next_entry, .. } = self {
            if payload
                .gas_refund_entries
                .get(*next_entry)
                .is_some_and(|entry| entry.index == tx_index)
            {
                *next_entry += 1;
            }
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
            Self::Verifying { payload, next_entry, .. } => {
                payload.gas_refund_entries[*next_entry..].iter().map(|entry| entry.index).collect()
            }
            _ => Vec::new(),
        }
    }

    /// Whether a `Verify` block claims a post-exec payload yet never carried the trailing `0x7D`.
    ///
    /// Per-tx settlement drains the verifier entries as the refunded txs commit, so an absent
    /// `0x7D` is invisible to the unconsumed-entries check — only this flag proves the producer
    /// actually committed the claimed refunds on-chain.
    const fn missing_post_exec_tx(&self) -> bool {
        matches!(self, Self::Verifying { saw_post_exec_tx: false, .. })
    }
}

/// Context for OP block execution.
#[derive(Debug, Default, Clone)]
pub struct OpBlockExecutionCtx {
    /// Parent block hash.
    pub parent_hash: B256,
    /// Whether this block is the activation block of some fork at or after Jovian, which must
    /// contain only deposit transactions.
    ///
    /// Compute via [`OpHardforks::is_no_user_tx_activation_block`] where the parent timestamp is
    /// available; `false` skips the executor's check.
    pub no_user_tx_activation_block: bool,
    /// Parent beacon block root.
    pub parent_beacon_block_root: Option<B256>,
    /// The block's extra data.
    pub extra_data: Bytes,
    /// Canonical post-exec execution mode for this block.
    pub post_exec_mode: PostExecMode,
    /// Candidate previews for validation or builder lookahead; never an inclusion instruction.
    pub parallel_candidates: Vec<ParallelCandidate>,
}

/// Balance patch that reconciles fee distribution with the post-refund gas used.
///
/// The EVM has already paid fees out based on `evm_gas_used`, so just lowering `gas_used` in the
/// receipt isn't enough — the sender, beneficiary, base-fee recipient, and operator-fee recipient
/// all need their balances rolled back to match the canonical gas. This struct carries the
/// per-recipient debits, plus the matching credit to the sender (which equals their sum).
#[derive(Debug, Default, Clone, PartialEq, Eq)]
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
    /// Exact policy-provided attribution events that produced the refund.
    pub refund_events: Vec<PostExecRefundEvent>,
}

/// Isolated EVM output before block admission, ordered policy evaluation, and fee settlement.
///
/// An execution coordinator must validate all state dependencies and the transaction identity
/// before passing this to [`OpBlockExecutor::prepare_transaction_result`]. It must never commit
/// `result.state` directly: deferred protocol fees and canonical SDM refunds are still missing.
#[derive(Debug, Clone)]
pub struct RawTransactionOutput<H> {
    /// EVM execution result and transaction-local state changes.
    pub result: ResultAndState<H>,
    /// Result from a sequential producer policy, if one was run.
    pub producer_result: Option<PostExecExecutedTx>,
    /// Isolated producer observations, evaluated only after dependency validation.
    pub observation: Option<crate::post_exec::ParallelObservation>,
    /// Protocol credits in their original application order.
    pub deferred_fees: Vec<op_revm::handler::DeferredFeeCredit>,
    /// Block-scoped L1 fee cache after execution, carried forward just like sequential execution.
    pub l1_block_info: Option<L1BlockInfo>,
}

/// Canonical transaction output ready for the existing infallible commit operation.
pub type PreparedCommit<H, T> = OpTxResult<H, T>;

/// The result of executing an OP transaction.
#[derive(Debug)]
pub struct OpTxResult<H, T> {
    /// The inner result of the transaction execution.
    pub inner: EthTxResult<H, T>,
    /// Whether the transaction is a deposit transaction.
    pub is_deposit: bool,
    /// Whether the transaction is a post-exec transaction.
    pub is_post_exec: bool,
    /// Gas used by EVM execution before any post-exec (SDM) refund — the "real compute" performed.
    ///
    /// Accumulated and bounded against the block gas limit by both block builders and the executor
    /// (see [`PreRefundGasUsed`]).
    pub evm_gas_used: u64,
    /// Canonical gas used after any post-exec adjustment.
    pub canonical_gas_used: u64,
    /// Canonical post-exec adjustment, if any.
    pub post_exec: Option<PostExecAdjustment>,
    /// Cached depositor nonce — looked up during execute so commit can be infallible.
    /// `Some` only for regolith deposit transactions.
    pub depositor_nonce: Option<u64>,
}

/// Read access to a transaction's pre-refund EVM gas usage.
///
/// Lets block builders read `evm_gas_used` through the generic [`BlockExecutor::Result`] type
/// (otherwise only bounded by `TxResult`) to self-limit a block against the block gas limit.
pub trait PreRefundGasUsed {
    /// Gas used by EVM execution, before any post-exec (SDM) refund.
    fn evm_gas_used(&self) -> u64;
}

impl<H, T> PreRefundGasUsed for OpTxResult<H, T> {
    fn evm_gas_used(&self) -> u64 {
        self.evm_gas_used
    }
}

impl<H, T> TxResult for OpTxResult<H, T>
where
    H: Send + 'static,
    T: Send + 'static,
{
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
pub struct OpBlockExecutor<Evm: alloy_evm::Evm, R: OpReceiptBuilder, Spec> {
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
    /// Total pre-refund EVM gas (real compute) across executed txs, before any post-exec (SDM)
    /// refund. Bounded by the block gas limit; equals [`Self::gas_used`] with SDM off, greater
    /// otherwise.
    pub evm_gas_used: u64,
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
    /// Per-transaction exact policy-provided refund attribution events aligned with receipts.
    pub refund_events_by_tx: Vec<Vec<PostExecRefundEvent>>,
    #[cfg(feature = "parallel")]
    pub(crate) parallel: Option<parallel::ParallelBlockState<Evm::HaltReason>>,
    #[cfg(feature = "parallel")]
    parallel_configuration_error: Option<&'static str>,
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
            evm_gas_used: 0,
            da_footprint_used: 0,
            ctx,
            l1_block_info: None,
            post_exec,
            refund_events_by_tx: Vec::new(),
            #[cfg(feature = "parallel")]
            parallel: None,
            #[cfg(feature = "parallel")]
            parallel_configuration_error: None,
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
        #[cfg(feature = "parallel")]
        if let Some(parallel) = self.parallel.as_mut() {
            parallel.clear();
        }
        self.post_exec = PostExecState::new(post_exec_mode);
    }

    /// Returns the accumulated post-exec entries (sequencer mode) without clearing them.
    pub const fn post_exec_entries(&self) -> &[SDMGasEntry] {
        self.post_exec.produced_entries()
    }

    /// Take the accumulated post-exec entries (sequencer mode).
    /// Returns the entries and clears the internal state.
    pub fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry> {
        self.post_exec.take_entries()
    }

    /// Take the exact per-transaction policy-provided refund events aligned with receipts.
    pub fn take_refund_events_by_tx(&mut self) -> Vec<Vec<PostExecRefundEvent>> {
        core::mem::take(&mut self.refund_events_by_tx)
    }
}

impl<E, R, Spec> OpBlockExecutor<E, R, Spec>
where
    E: PostExecEvm,
    R: OpReceiptBuilder,
{
    /// Snapshot refund state to carry across subblock executors.
    pub fn refund_snapshot(&self) -> E::Snapshot {
        self.evm.refund_snapshot()
    }

    /// Seed refund state captured from a prior subblock.
    pub fn seed_refund_snapshot(&mut self, state: E::Snapshot) {
        self.evm.seed_refund_snapshot(state);
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

    /// A fork-activation block at or after Jovian, which must contain only deposit transactions,
    /// contained a non-deposit (user) transaction.
    #[error("unexpected non-deposit transactions in fork activation block")]
    UnexpectedNonDepositTxInForkActivationBlock,

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
        // Load the L1 block contract into the cache. If the L1 block contract is not pre-loaded the
        // database will panic when trying to fetch the DA footprint gas scalar.
        self.evm.db_mut().basic(L1_BLOCK_CONTRACT).map_err(BlockExecutionError::other)?;

        let da_footprint_gas_scalar = L1BlockInfo::fetch_da_footprint_gas_scalar(self.evm.db_mut())
            .map_err(BlockExecutionError::other)?
            .into();

        // Use the cached enveloped tx bytes if available, otherwise encode the transaction.
        Ok(tx_env.encoded_bytes().map_or_else(
            || tx_da_footprint(tx.tx(), da_footprint_gas_scalar),
            |encoded| encoded_tx_da_footprint(encoded, da_footprint_gas_scalar),
        ))
    }

    fn invalid_post_exec_payload(reason: impl Into<String>) -> BlockExecutionError {
        validation_error(OpBlockExecutionError::InvalidPostExecPayload(reason.into()))
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

    /// Applies the post-exec refund to the result gas so `tx_gas_used()` reports canonical gas.
    ///
    /// The EVM refund counter stays unchanged to avoid subtracting the refund twice. EIP-7623 is
    /// part of `evm_gas_used`, so SDM applies after its floor and may reduce canonical gas below
    /// that floor. Reducing both `total_gas_spent` and `floor_gas` preserves this ordering in the
    /// generic [`revm::context::result::ResultGas`] view used by receipts, tracing, and execution
    /// observers.
    const fn canonicalize_result_gas(
        result: &mut ExecutionResult<E::HaltReason>,
        post_exec_refund: u64,
    ) {
        if post_exec_refund == 0 {
            return;
        }

        match result {
            ExecutionResult::Success { gas, .. } |
            ExecutionResult::Revert { gas, .. } |
            ExecutionResult::Halt { gas, .. } => {
                *gas = gas
                    .with_total_gas_spent(gas.total_gas_spent().saturating_sub(post_exec_refund))
                    .with_floor_gas(gas.floor_gas().saturating_sub(post_exec_refund));
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
                // Account::from sets original_info equal to the current info, which is
                // safe: it is not used by State::commit — the CacheAccount tracks its
                // own previous state for building transitions.
                let mut account = Account::from(info);
                account.status = AccountStatus::Touched;
                Ok(entry.insert(account))
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
    /// `execute_transaction_without_commit` then applies before state is committed.
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
            refund_events: Vec::new(),
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

/// Ensures a transaction's gas limit fits within the gas still available in the block.
///
/// Checks one condition at a time: pre-Regolith deposits are exempt from this check, and any
/// other transaction must not declare more gas than remains in the block.
fn validate_block_gas(
    transaction_gas_limit: u64,
    block_available_gas: u64,
    is_regolith: bool,
    is_deposit: bool,
) -> Result<(), BlockExecutionError> {
    // Pre-Regolith deposits are exempt from the available-block-gas check.
    if is_deposit && !is_regolith {
        return Ok(());
    }

    if transaction_gas_limit > block_available_gas {
        return Err(BlockValidationError::TransactionGasLimitMoreThanAvailableBlockGas {
            transaction_gas_limit,
            block_available_gas,
        }
        .into());
    }

    Ok(())
}

impl<E, R, Spec> OpBlockExecutor<E, R, Spec>
where
    E: PostExecEvm<
            DB: Database + DatabaseCommit + StateDB,
            Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
            HaltReason: Send + 'static,
        >,
    R: OpReceiptBuilder<
            Transaction: Transaction + Encodable2718 + OpConsensusTransaction,
            Receipt: TxReceipt,
        >,
    Spec: OpHardforks,
{
    /// Finalizes dependency-validated worker output against the canonical committed prefix.
    ///
    /// Producer policy state is staged during preparation. Callers that can reject the result
    /// must restore their policy snapshot, including when this method returns an error.
    pub fn prepare_transaction_result(
        &mut self,
        tx: impl ExecutableTx<Self>,
        raw: RawTransactionOutput<E::HaltReason>,
    ) -> Result<
        PreparedCommit<E::HaltReason, <R::Transaction as TransactionEnvelope>::TxType>,
        BlockExecutionError,
    > {
        self.prepare_transaction_with(tx, |_, _, _, _| Ok(raw))
    }

    /// Finalizes and conditionally commits validated worker output, rolling policy state back
    /// whenever admission, settlement, or the caller's inclusion decision rejects the candidate.
    pub fn commit_transaction_result(
        &mut self,
        tx: impl ExecutableTx<Self>,
        raw: RawTransactionOutput<E::HaltReason>,
        commit: impl FnOnce(
            &OpTxResult<E::HaltReason, <R::Transaction as TransactionEnvelope>::TxType>,
        ) -> CommitChanges,
    ) -> Result<Option<GasOutput>, BlockExecutionError> {
        let snapshot = self.post_exec.is_producing().then(|| self.refund_snapshot());
        let prepared = match self.prepare_transaction_result(tx, raw) {
            Ok(prepared) => prepared,
            Err(error) => {
                if let Some(snapshot) = snapshot {
                    self.seed_refund_snapshot(snapshot);
                }
                return Err(error);
            }
        };
        if !commit(&prepared).should_commit() {
            if let Some(snapshot) = snapshot {
                self.seed_refund_snapshot(snapshot);
            }
            return Ok(None);
        }
        Ok(Some(self.commit_transaction(prepared)))
    }

    fn prepare_transaction_with(
        &mut self,
        tx: impl ExecutableTx<Self>,
        execute: impl FnOnce(
            &mut E,
            E::Tx,
            PostExecTxContext,
            bool,
        ) -> Result<RawTransactionOutput<E::HaltReason>, E::Error>,
    ) -> Result<
        OpTxResult<E::HaltReason, <R::Transaction as TransactionEnvelope>::TxType>,
        BlockExecutionError,
    > {
        let (tx_env, tx) = tx.into_parts();
        let is_deposit = tx.tx().ty() == DEPOSIT_TRANSACTION_TYPE;
        let is_post_exec = tx.tx().ty() == POST_EXEC_TX_TYPE_ID;
        let tx_index = self.receipts.len() as u64;

        // Since Jovian, fork-activation blocks must contain only deposit transactions — before,
        // the sequencer skipped user txs there by policy, but it wasn't a consensus rule. The
        // synthetic post-exec SDM tx is not a user tx, so it is allowed.
        if self.ctx.no_user_tx_activation_block && !is_deposit && !is_post_exec {
            return Err(validation_error(
                OpBlockExecutionError::UnexpectedNonDepositTxInForkActivationBlock,
            ));
        }

        let transaction_gas_limit = tx.tx().gas_limit();

        // Bound the block's *pre-refund* `evm_gas_used` (real compute) rather than canonical
        // `gas_used`, so SDM refunds can't admit more compute than the block gas limit allows.
        // Inducting off the declared gas limit (an upper bound on actual `evm_gas_used`) keeps the
        // pre-refund sum within the limit. Since `evm_gas_used >= gas_used`, this subsumes the
        // canonical block-gas-limit check; with SDM off the two are identical.
        let evm_gas_available = self.evm.block().gas_limit().saturating_sub(self.evm_gas_used);
        validate_block_gas(transaction_gas_limit, evm_gas_available, self.is_regolith, is_deposit)?;

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
                    result: noop_post_exec_result(),
                    blob_gas_used: 0,
                    tx_type: tx.tx().tx_type(),
                },
                is_deposit: false,
                is_post_exec: true,
                evm_gas_used: 0,
                canonical_gas_used: 0,
                post_exec: None,
                depositor_nonce: None,
            });
        }

        let da_footprint_used = if self
            .spec
            .is_jovian_active_at_timestamp(self.evm.block().timestamp().saturating_to()) &&
            !is_deposit
        {
            let da_footprint_available =
                self.evm.block().gas_limit().saturating_sub(self.da_footprint_used);

            let tx_da_footprint = self.jovian_da_footprint_estimation(&tx_env, &tx)?;

            if tx_da_footprint > da_footprint_available {
                return Err(validation_error(
                    OpBlockExecutionError::TransactionDaFootprintAboveGasLimit {
                        transaction_da_footprint: tx_da_footprint,
                        available_block_da_footprint: da_footprint_available,
                    },
                ));
            }

            tx_da_footprint
        } else {
            0
        };

        let context = PostExecTxContext {
            tx_index,
            kind: if is_deposit { PostExecTxKind::Deposit } else { PostExecTxKind::Normal },
        };
        let raw = execute(&mut self.evm, tx_env, context, self.post_exec.is_producing())
            .map_err(|err| BlockExecutionError::evm(err, tx.tx().trie_hash()))?;
        #[cfg(feature = "parallel")]
        let finalization_started = std::time::Instant::now();
        let RawTransactionOutput {
            mut result,
            producer_result,
            observation,
            deferred_fees,
            l1_block_info,
        } = raw;
        if let Some(info) = l1_block_info {
            self.evm.seed_execution_l1_block_info(info);
        }
        for credit in deferred_fees {
            use revm::primitives::hash_map::Entry;
            let account = match result.state.entry(credit.recipient) {
                Entry::Occupied(entry) => entry.into_mut(),
                Entry::Vacant(entry) => {
                    let account = self
                        .evm
                        .db_mut()
                        .basic(credit.recipient)
                        .map_err(BlockExecutionError::other)?
                        .map(Account::from)
                        .unwrap_or_else(|| Account::new_not_existing(Default::default()));
                    entry.insert(account)
                }
            };
            account.mark_touch();
            // JournaledAccount::incr_balance touches on overflow but leaves the balance intact.
            if let Some(balance) = account.info.balance.checked_add(credit.amount) {
                account.info.balance = balance;
            }
        }

        let evm_gas_used = result.result.tx_gas_used();
        let (post_exec_refund, refund_events) = if self.post_exec.is_producing() {
            let policy_result = if let Some(observation) = observation {
                self.evm.evaluate_parallel_observation(context, &observation).ok_or_else(|| {
                    BlockExecutionError::msg(
                        "parallel refund observation does not match producer policy",
                    )
                })?
            } else {
                producer_result.ok_or_else(|| {
                    BlockExecutionError::msg("missing producer refund observation")
                })?
            };
            let PostExecExecutedTx { refund_total: refund, refund_events } = policy_result;
            // The policy is advisory. Contain a faulty policy here, before its output changes gas,
            // settlement, receipts, or the trailing payload: excessive normal-tx refunds are
            // discarded, and deposits never receive a refund.
            let refund = sanitize_producer_refund(refund, evm_gas_used, is_deposit);
            (refund, refund_events)
        } else {
            (
                self.verifier_post_exec_refund_for_tx(tx_index, is_deposit, false, evm_gas_used)?,
                Vec::new(),
            )
        };
        let canonical_gas_used = evm_gas_used.saturating_sub(post_exec_refund);
        let mut deltas = self.post_exec_settlement_deltas(
            &tx,
            evm_gas_used,
            post_exec_refund,
            is_deposit,
            false,
        )?;
        deltas.refund_events = refund_events;
        let post_exec =
            (post_exec_refund > 0 || !deltas.refund_events.is_empty()).then_some(deltas);

        // Pre-compute depositor nonce here so `commit_transaction` can be infallible.
        // Only post-regolith deposit transactions need the depositor account from DB.
        let sender = *tx.signer();
        let depositor_nonce = if self.is_regolith && is_deposit {
            let account = self
                .evm
                .db_mut()
                .basic(sender)
                .map_err(BlockExecutionError::other)?
                .unwrap_or_default();
            Some(account.nonce)
        } else {
            None
        };

        // Canonicalize the result gas and apply any post-exec refund to state in-place. Both
        // operations must run before commit so commit_transaction stays infallible.
        Self::canonicalize_result_gas(&mut result.result, post_exec_refund);
        if let Some(deltas) = post_exec.as_ref() {
            self.apply_post_exec_refund_to_state(&mut result.state, sender, deltas)?;
        }

        #[cfg(all(feature = "parallel", feature = "metrics"))]
        metrics::histogram!("optimism_parallel.finalization_seconds")
            .record(finalization_started.elapsed().as_secs_f64());
        #[cfg(all(feature = "parallel", not(feature = "metrics")))]
        let _ = finalization_started;
        Ok(OpTxResult {
            inner: EthTxResult {
                result,
                blob_gas_used: da_footprint_used,
                tx_type: tx.tx().tx_type(),
            },
            is_deposit,
            is_post_exec: false,
            evm_gas_used,
            canonical_gas_used,
            post_exec,
            depositor_nonce,
        })
    }
}

/// UPSTREAM-MIRROR(copy): alloy-evm@0.38.0 `alloy_evm::eth::block::EthBlockExecutor`
///
/// Mirrors upstream's `BlockExecutor` impl, reusing its `EthTxResult` for the inner result
/// but reimplementing every method. Known divergences to re-confirm on each bump: upstream
/// tracks regular and state block gas separately and returns `GasOutput::with_state_gas`,
/// this tracks `gas_used`/`evm_gas_used` and returns `GasOutput::new`; upstream's admission
/// check clamps the transaction gas limit by `cfg.tx_gas_limit_cap`, `validate_block_gas`
/// does not.
impl<E, R, Spec> BlockExecutor for OpBlockExecutor<E, R, Spec>
where
    E: PostExecEvm<
            DB: Database + DatabaseCommit + StateDB,
            Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
            HaltReason: Send + 'static,
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
        #[cfg(feature = "parallel")]
        if let Some(error) = self.parallel_configuration_error {
            return Err(BlockExecutionError::msg(error));
        }
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

    fn execute_transaction_with_commit_condition(
        &mut self,
        tx: impl ExecutableTx<Self>,
        f: impl FnOnce(&Self::Result) -> CommitChanges,
    ) -> Result<Option<GasOutput>, BlockExecutionError> {
        // Producer policy state is updated during EVM execution (before the commit decision) and
        // is not journaled with EVM state. A declined candidate must not affect a later committed
        // transaction, or the producer's payload can diverge from commit-only derivation paths.
        // Snapshot only in Produce mode and restore on decline or execution error.
        let refund_snapshot = self.post_exec.is_producing().then(|| self.refund_snapshot());

        let output = match self.execute_transaction_without_commit(tx) {
            Ok(output) => output,
            Err(err) => {
                if let Some(snapshot) = refund_snapshot {
                    self.seed_refund_snapshot(snapshot);
                }
                return Err(err);
            }
        };

        if !f(&output).should_commit() {
            if let Some(snapshot) = refund_snapshot {
                self.seed_refund_snapshot(snapshot);
            }
            return Ok(None);
        }

        Ok(Some(self.commit_transaction(output)))
    }

    /// In Produce mode, this method does not snapshot or restore producer-policy state. Opaque
    /// policies can mutate during execution, including on failure; adapted parallel policies keep
    /// their updates private until commit.
    /// Callers that may discard a candidate must snapshot and restore the policy themselves or use
    /// [`execute_transaction_with_commit_condition`](Self::execute_transaction_with_commit_condition),
    /// which restores the per-candidate snapshot on execution error and declined commit.
    fn execute_transaction_without_commit(
        &mut self,
        tx: impl ExecutableTx<Self>,
    ) -> Result<Self::Result, BlockExecutionError> {
        #[cfg(feature = "parallel")]
        if let Some(error) = self.parallel_configuration_error {
            return Err(BlockExecutionError::msg(error));
        }
        #[cfg(feature = "parallel")]
        if self.parallel.is_some() &&
            self.post_exec.is_producing() &&
            !self.evm.supports_parallel_observation()
        {
            return Err(BlockExecutionError::msg(
                "parallel SDM production requires an observation/evaluation refund policy",
            ));
        }
        let (tx_env, tx) = tx.into_parts();
        #[cfg(feature = "parallel")]
        let speculative = {
            let context =
                self.evm.parallel_environment().map(|environment| parallel::ExecutionContext {
                    environment,
                    l1_block_info: self.evm.execution_l1_block_info(),
                    producing: self.post_exec.is_producing(),
                });
            if let Some(parallel) = self.parallel.as_mut() {
                if tx.tx().ty() == DEPOSIT_TRANSACTION_TYPE || tx.tx().ty() == POST_EXEC_TX_TYPE_ID
                {
                    parallel.clear();
                    None
                } else if let Some(worker_tx) = tx_env.parallel_transaction() {
                    parallel
                        .take(
                            tx.tx().trie_hash(),
                            &worker_tx,
                            &self.ctx.parallel_candidates,
                            context,
                            self.evm.db_mut(),
                        )
                        .map_err(BlockExecutionError::other)?
                } else {
                    None
                }
            } else {
                None
            }
        };

        #[cfg(feature = "parallel")]
        let speculative = speculative
            .filter(|output| {
                self.evm
                    .parallel_precompiles_compatible(&output.warm_precompiles, &output.call_targets)
            })
            .map(|output| output.raw);

        #[cfg(feature = "parallel")]
        if self
            .parallel
            .as_ref()
            .is_some_and(|state| state.runtime.config().mode == ExecutionMode::Parallel)
        {
            if let Some(raw) = speculative {
                if let Some(parallel) = &self.parallel {
                    parallel.runtime.record_reuse();
                }
                return self.prepare_transaction_result((tx_env, tx), raw);
            }
        }

        #[cfg(feature = "parallel")]
        let shadow = speculative.map(|raw| {
            let policy = self.refund_snapshot();
            let l1_info = self.evm.execution_l1_block_info();
            let result = self.prepare_transaction_result((tx_env.clone(), &tx), raw);
            let prepared_policy = self.evm.prepared_refund_snapshot();
            self.seed_refund_snapshot(policy);
            if let Some(info) = l1_info {
                self.evm.seed_execution_l1_block_info(info);
            }
            (result, prepared_policy)
        });
        #[cfg(all(feature = "parallel", feature = "metrics"))]
        let retry_started = self
            .parallel
            .as_ref()
            .is_some_and(|parallel| {
                parallel.runtime.config().mode == ExecutionMode::Parallel &&
                    parallel.attempted(tx.tx().trie_hash())
            })
            .then(std::time::Instant::now);
        let output =
            self.prepare_transaction_with((tx_env, &tx), |evm, tx_env, context, producing| {
                if producing {
                    evm.begin_post_exec_tx(context);
                }
                let result = evm.transact(tx_env)?;
                Ok(RawTransactionOutput {
                    result,
                    producer_result: producing.then(|| evm.take_last_post_exec_tx_result()),
                    observation: None,
                    deferred_fees: evm.take_deferred_fees(),
                    l1_block_info: evm.execution_l1_block_info(),
                })
            });
        #[cfg(all(feature = "parallel", feature = "metrics"))]
        if let Some(started) = retry_started {
            metrics::histogram!("optimism_parallel.canonical_retry_seconds")
                .record(started.elapsed().as_secs_f64());
            if let Ok(result) = &output {
                metrics::counter!("optimism_parallel.canonical_retry_gas")
                    .increment(result.evm_gas_used);
            }
        }
        #[cfg(feature = "parallel")]
        if let Some((shadow, prepared_policy)) = shadow {
            let equal = match (&output, &shadow) {
                (Ok(reference), Ok(candidate)) => {
                    reference.inner.result.result == candidate.inner.result.result &&
                        parallel::equal_state(
                            &reference.inner.result.state,
                            &candidate.inner.result.state,
                        ) &&
                        reference.inner.blob_gas_used == candidate.inner.blob_gas_used &&
                        reference.evm_gas_used == candidate.evm_gas_used &&
                        reference.canonical_gas_used == candidate.canonical_gas_used &&
                        reference.post_exec == candidate.post_exec &&
                        reference.depositor_nonce == candidate.depositor_nonce &&
                        (!self.post_exec.is_producing() ||
                            self.evm.matches_prepared_refund_snapshot(&prepared_policy))
                }
                (Err(reference), Err(candidate)) => reference.to_string() == candidate.to_string(),
                _ => false,
            };
            if let Some(parallel) = &self.parallel {
                parallel.runtime.record_shadow(equal);
            }
            // A mismatch disables speculation for the rest of this block; the reference output
            // remains authoritative. Never convert a shadow mismatch into a consensus rejection.
            if !equal {
                tracing::error!(target: "optimism::parallel", tx = ?tx.tx().trie_hash(), "shadow execution mismatch; using sequential result and disabling speculation for this block");
                self.parallel = None;
            }
        }
        output
    }

    fn commit_transaction(&mut self, output: Self::Result) -> GasOutput {
        if self.post_exec.is_producing() {
            self.evm.commit_post_exec_tx();
        }
        let tx_index = self.receipts.len() as u64;
        let OpTxResult {
            inner: EthTxResult { result: ResultAndState { result, state }, blob_gas_used, tx_type },
            is_deposit,
            is_post_exec,
            evm_gas_used,
            canonical_gas_used,
            post_exec,
            depositor_nonce,
        } = output;

        let (post_exec_refund, refund_events) = match post_exec {
            Some(deltas) => (deltas.refund, deltas.refund_events),
            None => (0, Vec::new()),
        };

        if !is_deposit && !is_post_exec && post_exec_refund > 0 {
            if let Some(entries) = self.post_exec.produced_entries_mut() {
                entries.push(SDMGasEntry { index: tx_index, gas_refund: post_exec_refund });
            }
        }
        if self.post_exec.is_verifying() && post_exec_refund > 0 {
            self.post_exec.consume_verifier_entry(tx_index);
        }
        if !is_post_exec {
            self.refund_events_by_tx.push(refund_events);
        }

        // add canonical gas used
        self.gas_used += canonical_gas_used;
        // Accumulate pre-refund EVM gas (real compute); bounded against the block gas limit by the
        // admission check in `execute_transaction_without_commit`.
        self.evm_gas_used += evm_gas_used;

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
                        deposit_nonce: depositor_nonce,
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

        GasOutput::new(canonical_gas_used)
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

        if self.post_exec.missing_post_exec_tx() {
            return Err(Self::invalid_post_exec_payload(
                "post-exec payload present but block carries no post-exec tx",
            ));
        }

        let balance_increments =
            post_block_balance_increments::<Header>(&self.spec, self.evm.block(), &[], None);
        // increment balances; the DB-level state hook (if any) fires via the commit inside.
        self.evm
            .db_mut()
            .increment_balances(balance_increments)
            .map_err(|_| BlockValidationError::IncrementBalanceFailed)?;

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
#[derive(Debug, Clone, Default)]
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
    #[cfg(feature = "parallel")]
    parallel_runtime: Option<alloc::sync::Arc<ParallelRuntime>>,
}

impl<R, Spec, EvmFactory> OpBlockExecutorFactory<R, Spec, EvmFactory> {
    /// Creates a new [`OpBlockExecutorFactory`] with the given spec, [`EvmFactory`], and
    /// [`OpReceiptBuilder`].
    pub const fn new(receipt_builder: R, spec: Spec, evm_factory: EvmFactory) -> Self {
        Self {
            receipt_builder,
            spec,
            evm_factory,
            #[cfg(feature = "parallel")]
            parallel_runtime: None,
        }
    }

    /// Installs a node-shared worker pool. No workers are created by default.
    #[cfg(feature = "parallel")]
    pub fn with_parallel_runtime(
        mut self,
        runtime: impl Into<Option<alloc::sync::Arc<ParallelRuntime>>>,
    ) -> Self {
        self.parallel_runtime = runtime.into();
        self
    }

    /// Returns the optional node-shared speculative runtime.
    #[cfg(feature = "parallel")]
    pub const fn parallel_runtime(&self) -> Option<&alloc::sync::Arc<ParallelRuntime>> {
        self.parallel_runtime.as_ref()
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
        > + 'static,
    Spec: OpHardforks + 'static,
    F: PostExecEvmFactoryHooks + 'static,
    F::Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction> + OpTxEnv,
    Self: 'static,
{
    type EvmFactory = PostExecEvmFactoryAdapter<F>;
    type ExecutionCtx<'a> = OpBlockExecutionCtx;
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;
    type TxExecutionResult = OpTxResult<
        <PostExecEvmFactoryAdapter<F> as EvmFactory>::HaltReason,
        <R::Transaction as TransactionEnvelope>::TxType,
    >;
    type Executor<
        'a,
        DB: StateDB,
        I: Inspector<<PostExecEvmFactoryAdapter<F> as EvmFactory>::Context<DB>>,
    > = OpBlockExecutor<<PostExecEvmFactoryAdapter<F> as EvmFactory>::Evm<DB, I>, &'a R, &'a Spec>;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: <PostExecEvmFactoryAdapter<F> as EvmFactory>::Evm<DB, I>,
        ctx: Self::ExecutionCtx<'a>,
    ) -> Self::Executor<'a, DB, I>
    where
        DB: StateDB,
        I: Inspector<<PostExecEvmFactoryAdapter<F> as EvmFactory>::Context<DB>>,
    {
        let executor = OpBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder);
        #[cfg(feature = "parallel")]
        let executor = {
            let mut executor = executor;
            if self
                .parallel_runtime
                .as_ref()
                .is_some_and(|runtime| runtime.config().mode != ExecutionMode::Sequential)
            {
                executor.parallel_configuration_error =
                    Some("this custom EVM factory does not support parallel execution");
            }
            executor
        };
        executor
    }
}

impl<ReceiptBuilder, Spec, Tx, RefundPolicy> BlockExecutorFactory
    for OpBlockExecutorFactory<ReceiptBuilder, Spec, OpEvmFactory<Tx, RefundPolicy>>
where
    ReceiptBuilder: OpReceiptBuilder<
            Transaction: Transaction + Encodable2718 + OpConsensusTransaction,
            Receipt: TxReceipt,
        > + 'static,
    Spec: OpHardforks + 'static,
    Tx: IntoTxEnv<Tx>
        + Into<OpTransaction<TxEnv>>
        + Default
        + Clone
        + core::fmt::Debug
        + FromRecoveredTx<ReceiptBuilder::Transaction>
        + FromTxWithEncoded<ReceiptBuilder::Transaction>
        + OpTxEnv
        + 'static,
    RefundPolicy: Default + PostExecRefundInspector + 'static,
    Self: 'static,
{
    type EvmFactory = OpEvmFactory<Tx, RefundPolicy>;
    type ExecutionCtx<'a> = OpBlockExecutionCtx;
    type Transaction = ReceiptBuilder::Transaction;
    type Receipt = ReceiptBuilder::Receipt;
    type TxExecutionResult = OpTxResult<
        <OpEvmFactory<Tx, RefundPolicy> as EvmFactory>::HaltReason,
        <ReceiptBuilder::Transaction as TransactionEnvelope>::TxType,
    >;
    type Executor<
        'a,
        DB: StateDB,
        I: Inspector<<OpEvmFactory<Tx, RefundPolicy> as EvmFactory>::Context<DB>>,
    > = OpBlockExecutor<
        <OpEvmFactory<Tx, RefundPolicy> as EvmFactory>::Evm<DB, I>,
        &'a ReceiptBuilder,
        &'a Spec,
    >;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: <OpEvmFactory<Tx, RefundPolicy> as EvmFactory>::Evm<DB, I>,
        ctx: Self::ExecutionCtx<'a>,
    ) -> Self::Executor<'a, DB, I>
    where
        DB: StateDB,
        I: Inspector<<OpEvmFactory<Tx, RefundPolicy> as EvmFactory>::Context<DB>>,
    {
        let executor = OpBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder);
        #[cfg(feature = "parallel")]
        let executor = {
            use alloc::{string::ToString, sync::Arc};
            use reth_optimism_parallel::SpeculationError;
            let mut executor = executor;
            if let Some(runtime) = self
                .parallel_runtime
                .as_ref()
                .filter(|runtime| runtime.config().mode != ExecutionMode::Sequential)
            {
                let producing = executor.post_exec.is_producing();
                if producing && !executor.evm.supports_parallel_observation() {
                    executor.parallel_configuration_error = Some(
                        "parallel SDM production requires an observation/evaluation refund policy",
                    );
                } else {
                    let output_limit = runtime.config().max_output_bytes;
                    let worker: Arc<parallel::Worker<op_revm::OpHaltReason>> =
                        Arc::new(move |(tx, context), db| {
                            let parallel::ExecutionContext {
                                l1_block_info: l1_info,
                                producing,
                                environment: env,
                            } = context;
                            let mut evm = OpEvmFactory::<crate::OpTx, RefundPolicy>::default()
                                .create_evm_with_inspector(
                                    db,
                                    env.clone(),
                                    parallel::CallTargets::default(),
                                );
                            let warm_precompiles = evm.precompiles().addresses().copied().collect();
                            evm.set_parallel_execution(true);
                            if let Some(info) = l1_info {
                                evm.seed_execution_l1_block_info(info);
                            }
                            if producing {
                                evm.begin_post_exec_tx(PostExecTxContext {
                                    tx_index: 0,
                                    kind: PostExecTxKind::Normal,
                                });
                            }
                            let result = evm
                                .transact(tx)
                                .map_err(|error| SpeculationError::Worker(error.to_string()))?;
                            let observation =
                                producing.then(|| evm.take_parallel_observation()).flatten();
                            let state_bytes =
                                result.state.values().fold(0usize, |size, account| {
                                    size.saturating_add(512)
                                        .saturating_add(account.storage.len().saturating_mul(128))
                                        .saturating_add(
                                            account.info.code.as_ref().map_or(0, |code| code.len()),
                                        )
                                });
                            let logs_bytes =
                                result.result.logs().iter().fold(0usize, |size, log| {
                                    size.saturating_add(log.data.data.len())
                                        .saturating_add(log.data.topics().len().saturating_mul(32))
                                        .saturating_add(128)
                                });
                            let bytes = state_bytes
                                .saturating_add(logs_bytes)
                                .saturating_add(
                                    evm.components().1.addresses.len().saturating_mul(64),
                                )
                                .saturating_add(1024)
                                .saturating_add(
                                    result.result.output().map_or(0, |bytes| bytes.len()),
                                )
                                .saturating_add(
                                    observation
                                        .as_ref()
                                        .map_or(0, |observation| observation.size_bytes()),
                                );
                            #[cfg(feature = "metrics")]
                            metrics::histogram!("optimism_parallel.output_bytes")
                                .record(bytes as f64);
                            if bytes > output_limit {
                                return Err(SpeculationError::Limit);
                            }
                            if evm.components().1.exceeded {
                                return Err(SpeculationError::Limit);
                            }
                            let call_targets =
                                evm.components().1.addresses.iter().copied().collect();
                            Ok(parallel::WorkerOutput {
                                environment: env,
                                warm_precompiles,
                                call_targets,
                                raw: RawTransactionOutput {
                                    result,
                                    producer_result: None,
                                    observation,
                                    deferred_fees: evm.take_deferred_fees(),
                                    l1_block_info: evm.execution_l1_block_info(),
                                },
                            })
                        });
                    executor.parallel =
                        Some(parallel::ParallelBlockState::new(runtime.clone(), worker));
                }
            }
            executor
        };
        executor
    }
}

#[cfg(test)]
mod tests;
