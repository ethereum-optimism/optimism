use alloy_consensus::{Sealable, Transaction, conditional::BlockConditionalAttributes};
use alloy_eips::{Encodable2718, Typed2718};
use alloy_evm::{
    Database, Evm as AlloyEvm,
    block::{BlockExecutor as AlloyBlockExecutor, CommitChanges, TxResult},
};
use alloy_op_evm::PreRefundGasUsed;
use alloy_primitives::{BlockHash, Bytes, U256};
use alloy_rpc_types_eth::Withdrawals;
use core::fmt::Debug;
use op_alloy_consensus::{SDMGasEntry, build_post_exec_tx};
use op_revm::{L1BlockInfo, OpSpecId};
use reth_basic_payload_builder::PayloadConfig;
use reth_chainspec::{EthChainSpec, EthereumHardforks};
use reth_evm::{
    EvmEnv,
    execute::{BlockBuilder, BlockExecutionError, BlockValidationError},
};
use reth_node_api::PayloadBuilderError;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::{
    ConfigurePostExecEvm, OpEvmConfig, OpNextBlockEnvAttributes, PostExecExecutorExt, PostExecMode,
};
use reth_optimism_forks::OpHardforks;
use reth_optimism_node::OpPayloadBuilderAttributes;
use reth_optimism_payload_builder::{
    config::{OpDAConfig, OpGasLimitConfig},
    error::OpPayloadBuilderError,
};
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
use reth_optimism_txpool::{
    conditional::MaybeConditionalTransaction,
    estimated_da_size::DataAvailabilitySized,
    interop::{InteropFailsafe, MaybeInteropTransaction, is_interop_tx, is_valid_interop},
};
use reth_payload_builder::PayloadId;
use reth_primitives_traits::{InMemorySize, SealedHeader, SignedTransaction};
use reth_revm::{State, context::Block};
use reth_transaction_pool::{BestTransactionsAttributes, PoolTransaction};
use revm::interpreter::as_u64_saturated;
use std::{
    ops::DerefMut,
    sync::{Arc, atomic::Ordering},
    time::Instant,
};
use tokio_util::sync::CancellationToken;
use tracing::{debug, info, trace};

use crate::{
    gas_limiter::AddressGasLimiter,
    metrics::OpRBuilderMetrics,
    primitives::reth::{ExecutionInfo, TxnExecutionResult},
    sdm_admin::SdmPostExecOptInFlag,
    traits::PayloadTxsBounds,
    tx::MaybeRevertingTransaction,
    tx_signer::Signer,
};

pub(super) fn last_receipt_with_cumulative_gas<Executor>(
    executor: &Executor,
    cumulative_gas_used: u64,
) -> Option<OpReceipt>
where
    Executor: AlloyBlockExecutor<Receipt = OpReceipt>,
{
    let mut receipt = executor.receipts().last().cloned()?;
    receipt.as_receipt_mut().cumulative_gas_used = cumulative_gas_used;
    Some(receipt)
}

/// Adds `state_db_mut` to any `BlockBuilder` whose EVM database derefs to a `State<DB>`.
///
/// PostExec materialization needs raw access to the underlying `State<DB>` (to merge transitions
/// and take the bundle), but the `BlockBuilder` trait only exposes the executor/EVM. Rather than
/// wrap every builder in a delegation struct, this trait provides the accessor as a blanket impl.
pub(super) trait BlockBuilderStateDbExt<DB>: BlockBuilder
where
    Self::Executor: AlloyBlockExecutor<Evm: AlloyEvm<DB: DerefMut<Target = State<DB>>>>,
    DB: Database,
{
    fn state_db_mut(&mut self) -> &mut State<DB> {
        self.evm_mut().db_mut().deref_mut()
    }
}

impl<B, DB> BlockBuilderStateDbExt<DB> for B
where
    B: BlockBuilder,
    B::Executor: AlloyBlockExecutor<Evm: AlloyEvm<DB: DerefMut<Target = State<DB>>>>,
    DB: Database,
{
}

/// Snapshot the post-exec mode for an upcoming build.
///
/// Two gates must agree: the protocol gate (chain spec Interop at the given timestamp) and the
/// operator gate (admin RPC). Either being false yields [`PostExecMode::Disabled`].
///
/// Called once per `OpPayloadBuilderCtx` construction so every builder/replay run for the same
/// payload sees the same decision, even if the admin RPC flips the operator opt-in mid-block.
pub(super) fn compute_post_exec_mode(
    evm_config: &OpEvmConfig,
    timestamp: u64,
    opt_in: &SdmPostExecOptInFlag,
) -> PostExecMode {
    let protocol_active = evm_config.is_sdm_active_at_timestamp(timestamp);
    let operator_opted_in = opt_in.load(Ordering::Acquire);
    if protocol_active && operator_opted_in {
        PostExecMode::Produce
    } else {
        PostExecMode::Disabled
    }
}

/// Builds the canonical PostExec (`0x7D`) tx for the block being built, or `None` when the
/// builder is not producing or there is nothing to refund — producers never emit an
/// empty-entries PostExec tx. Shared by the standard and flashblocks builders so the
/// produce-side rules live in one place.
pub(super) fn build_current_post_exec_tx<ExtraCtx>(
    ctx: &OpPayloadBuilderCtx<ExtraCtx>,
    entries: Vec<SDMGasEntry>,
) -> Option<OpTransactionSigned>
where
    ExtraCtx: Debug + Default,
{
    if !matches!(ctx.post_exec_mode, PostExecMode::Produce) || entries.is_empty() {
        return None;
    }

    Some(OpTransactionSigned::from(
        build_post_exec_tx(ctx.block_number(), entries).seal_slow(),
    ))
}

/// Container type that holds all necessities to build a new payload.
#[derive(Debug)]
pub struct OpPayloadBuilderCtx<ExtraCtx: Debug + Default = ()> {
    /// The type that knows how to perform system calls and configure the evm.
    pub evm_config: OpEvmConfig,
    /// The DA config for the payload builder
    pub da_config: OpDAConfig,
    // Gas limit configuration for the payload builder
    pub gas_limit_config: OpGasLimitConfig,
    /// The chainspec
    pub chain_spec: Arc<OpChainSpec>,
    /// How to build the payload.
    pub config: PayloadConfig<OpPayloadBuilderAttributes<OpTransactionSigned>>,
    /// Evm Settings
    pub evm_env: EvmEnv<OpSpecId>,
    /// Block env attributes for the current block.
    pub block_env_attributes: OpNextBlockEnvAttributes,
    /// Marker to check whether the job has been cancelled.
    pub cancel: CancellationToken,
    /// The builder signer
    pub builder_signer: Option<Signer>,
    /// The metrics for the builder
    pub metrics: Arc<OpRBuilderMetrics>,
    /// Extra context for the payload builder
    pub extra_ctx: ExtraCtx,
    /// Max gas that can be used by a transaction.
    pub max_gas_per_txn: Option<u64>,
    /// Rate limiting based on gas. This is an optional feature.
    pub address_gas_limiter: AddressGasLimiter,
    /// Snapshot of the post-exec mode for this build. Captured once per `OpPayloadBuilderCtx`
    /// construction (see [`compute_post_exec_mode`]) so every builder created from this ctx —
    /// fallback block, each flashblock, canonical replay — observes the same decision even if
    /// the admin RPC flips the operator opt-in mid-block.
    pub post_exec_mode: PostExecMode,
    /// Interop failsafe gate shared with the txpool interop filter.
    pub interop_failsafe: InteropFailsafe,
}

impl<ExtraCtx: Debug + Default> OpPayloadBuilderCtx<ExtraCtx> {
    pub(super) fn with_cancel(self, cancel: CancellationToken) -> Self {
        Self { cancel, ..self }
    }

    pub(super) fn with_extra_ctx(self, extra_ctx: ExtraCtx) -> Self {
        Self { extra_ctx, ..self }
    }

    /// Returns the parent block the payload will be build on.
    pub fn parent(&self) -> &SealedHeader {
        &self.config.parent_header
    }

    /// Returns the parent hash
    pub fn parent_hash(&self) -> BlockHash {
        self.parent().hash()
    }

    /// Returns the timestamp
    pub fn timestamp(&self) -> u64 {
        self.attributes().timestamp()
    }

    /// Returns the builder attributes.
    pub(super) const fn attributes(&self) -> &OpPayloadBuilderAttributes<OpTransactionSigned> {
        &self.config.attributes
    }

    /// Returns true when the tx pool is excluded and the block must be reproduced
    /// deterministically from forced transactions only (i.e. `no_tx_pool = true`).
    ///
    /// Mirrors op-reth's `force_empty` for cross-builder consistency.
    pub fn force_empty(&self) -> bool {
        self.attributes().no_tx_pool
    }

    /// Returns the withdrawals if shanghai is active.
    pub fn withdrawals(&self) -> Option<&Withdrawals> {
        self.chain_spec
            .is_shanghai_active_at_timestamp(self.attributes().timestamp())
            .then(|| self.attributes().withdrawals())
    }

    /// Returns the block gas limit to target.
    pub fn block_gas_limit(&self) -> u64 {
        match self.gas_limit_config.gas_limit() {
            Some(gas_limit) => gas_limit,
            None => self
                .attributes()
                .gas_limit
                .unwrap_or(self.evm_env.block_env.gas_limit),
        }
    }

    /// Returns the block number for the block.
    pub fn block_number(&self) -> u64 {
        as_u64_saturated!(self.evm_env.block_env.number)
    }

    /// Returns the current base fee
    pub fn base_fee(&self) -> u64 {
        self.evm_env.block_env.basefee
    }

    /// Returns the current blob gas price.
    pub fn get_blob_gasprice(&self) -> Option<u64> {
        self.evm_env
            .block_env
            .blob_gasprice()
            .map(|gasprice| gasprice as u64)
    }

    /// Returns the blob fields for the header.
    ///
    /// This will return the culmative DA bytes * scalar after Jovian
    /// after Ecotone, this will always return Some(0) as blobs aren't supported
    /// pre Ecotone, these fields aren't used.
    pub fn blob_fields<Extra: Debug + Default>(
        &self,
        info: &ExecutionInfo<Extra>,
    ) -> (Option<u64>, Option<u64>) {
        if self.is_jovian_active() {
            let scalar = info
                .da_footprint_scalar
                .expect("Scalar must be defined for Jovian blocks");
            let result = info.cumulative_da_bytes_used * scalar as u64;
            (Some(0), Some(result))
        } else if self.is_ecotone_active() {
            (Some(0), Some(0))
        } else {
            (None, None)
        }
    }

    /// Returns the extra data for the block.
    ///
    /// After holocene this extracts the extradata from the payload
    pub fn extra_data(&self) -> Result<Bytes, PayloadBuilderError> {
        if self.is_jovian_active() {
            self.attributes()
                .get_jovian_extra_data(
                    self.chain_spec
                        .base_fee_params_at_timestamp(self.attributes().timestamp()),
                )
                .map_err(PayloadBuilderError::other)
        } else if self.is_holocene_active() {
            self.attributes()
                .get_holocene_extra_data(
                    self.chain_spec
                        .base_fee_params_at_timestamp(self.attributes().timestamp()),
                )
                .map_err(PayloadBuilderError::other)
        } else {
            Ok(Default::default())
        }
    }

    /// Returns the current fee settings for transactions from the mempool
    pub fn best_transaction_attributes(&self) -> BestTransactionsAttributes {
        BestTransactionsAttributes::new(self.base_fee(), self.get_blob_gasprice())
    }

    /// Returns the unique id for this payload job.
    pub fn payload_id(&self) -> PayloadId {
        self.attributes().payload_id()
    }

    /// Returns true if regolith is active for the payload.
    pub fn is_regolith_active(&self) -> bool {
        self.chain_spec
            .is_regolith_active_at_timestamp(self.attributes().timestamp())
    }

    /// Returns true if ecotone is active for the payload.
    pub fn is_ecotone_active(&self) -> bool {
        self.chain_spec
            .is_ecotone_active_at_timestamp(self.attributes().timestamp())
    }

    /// Returns true if canyon is active for the payload.
    pub fn is_canyon_active(&self) -> bool {
        self.chain_spec
            .is_canyon_active_at_timestamp(self.attributes().timestamp())
    }

    /// Returns true if holocene is active for the payload.
    pub fn is_holocene_active(&self) -> bool {
        self.chain_spec
            .is_holocene_active_at_timestamp(self.attributes().timestamp())
    }

    /// Returns true if isthmus is active for the payload.
    pub fn is_isthmus_active(&self) -> bool {
        self.chain_spec
            .is_isthmus_active_at_timestamp(self.attributes().timestamp())
    }

    /// Returns true if isthmus is active for the payload.
    pub fn is_jovian_active(&self) -> bool {
        self.chain_spec
            .is_jovian_active_at_timestamp(self.attributes().timestamp())
    }

    /// Returns the chain id
    pub fn chain_id(&self) -> u64 {
        self.chain_spec.chain_id()
    }
}

impl<ExtraCtx: Debug + Default> OpPayloadBuilderCtx<ExtraCtx> {
    /// Returns a post-exec-aware block builder for the next block in the payload.
    pub(super) fn block_builder_for_next_block<'a, DB: Database + 'a>(
        &'a self,
        db: &'a mut State<DB>,
    ) -> Result<
        impl BlockBuilder<
            Primitives = reth_optimism_primitives::OpPrimitives,
            Executor: PostExecExecutorExt
                          + AlloyBlockExecutor<
                Transaction = OpTransactionSigned,
                Receipt = OpReceipt,
                Evm: AlloyEvm<DB: DerefMut<Target = State<DB>>>,
                Result: PreRefundGasUsed,
            >,
        > + 'a,
        PayloadBuilderError,
    > {
        self.evm_config
            .post_exec_builder_for_next_block(
                db,
                self.parent(),
                self.block_env_attributes.clone(),
                self.post_exec_mode.clone(),
            )
            .map_err(PayloadBuilderError::other)
    }

    /// Executes all sequencer transactions that are included in the payload attributes.
    pub(super) fn execute_sequencer_transactions<E: Debug + Default, Builder>(
        &self,
        builder: &mut Builder,
    ) -> Result<ExecutionInfo<E>, PayloadBuilderError>
    where
        Builder: BlockBuilder<Primitives = reth_optimism_primitives::OpPrimitives>,
        Builder::Executor:
            AlloyBlockExecutor<Transaction = OpTransactionSigned, Receipt = OpReceipt>,
    {
        let mut info = ExecutionInfo::with_capacity(self.attributes().transactions.len());

        for sequencer_tx in &self.attributes().transactions {
            // A sequencer's block should never contain blob transactions.
            if sequencer_tx.value().is_eip4844() {
                return Err(PayloadBuilderError::other(
                    OpPayloadBuilderError::BlobTransactionRejected,
                ));
            }

            // Convert the transaction to a [Recovered<TransactionSigned>]. This is
            // purely for the purposes of utilizing the `evm_config.tx_env`` function.
            // Deposit transactions do not have signatures, so if the tx is a deposit, this
            // will just pull in its `from` address.
            let sequencer_tx = sequencer_tx
                .value()
                .try_clone_into_recovered()
                .map_err(|_| {
                    PayloadBuilderError::other(OpPayloadBuilderError::TransactionEcRecoverFailed)
                })?;

            let gas_used = match builder.execute_transaction(sequencer_tx.clone()) {
                Ok(gas_used) => gas_used,
                Err(BlockExecutionError::Validation(BlockValidationError::InvalidTx {
                    error,
                    ..
                })) => {
                    trace!(target: "payload_builder", %error, ?sequencer_tx, "Error in sequencer transaction, skipping.");
                    continue;
                }
                Err(err) => {
                    // this is an error that we should treat as fatal for this attempt
                    return Err(PayloadBuilderError::EvmExecutionError(Box::new(err)));
                }
            };

            // add gas used by the transaction to cumulative gas used, before creating the receipt
            let tx_gas_used = gas_used.tx_gas_used();
            info.cumulative_gas_used += tx_gas_used;
            // Sequencer txs (deposits/system txs) are never SDM-refunded, so pre-refund gas equals
            // canonical gas; track it to stay in lockstep with the executor's pre-refund accumulator
            // (otherwise we'd admit a tx the executor then fatally rejects).
            info.cumulative_evm_gas_used += tx_gas_used;

            if !sequencer_tx.is_deposit() {
                info.cumulative_da_bytes_used += op_alloy_flz::tx_estimated_size_fjord_bytes(
                    sequencer_tx.encoded_2718().as_slice(),
                );
            }

            info.receipts.push(
                last_receipt_with_cumulative_gas(builder.executor(), info.cumulative_gas_used)
                    .expect("executor must record a receipt for committed tx"),
            );

            // append sender and transaction to the respective lists
            info.executed_senders.push(sequencer_tx.signer());
            info.executed_transactions.push(sequencer_tx.into_inner());
        }

        let da_footprint_gas_scalar = self
            .chain_spec
            .is_jovian_active_at_timestamp(self.attributes().timestamp())
            .then(|| {
                L1BlockInfo::fetch_da_footprint_gas_scalar(builder.evm_mut().db_mut())
                    .expect("DA footprint should always be available from the database post jovian")
            });

        info.da_footprint_scalar = da_footprint_gas_scalar;

        Ok(info)
    }

    /// Executes the given best transactions and updates the execution info.
    ///
    /// Returns `Ok(Some(())` if the job was cancelled.
    pub(super) fn execute_best_transactions<E: Debug + Default, Builder>(
        &self,
        info: &mut ExecutionInfo<E>,
        builder: &mut Builder,
        best_txs: &mut impl PayloadTxsBounds,
        block_gas_limit: u64,
        block_da_limit: Option<u64>,
        block_da_footprint_limit: Option<u64>,
    ) -> Result<Option<()>, PayloadBuilderError>
    where
        Builder: BlockBuilder<Primitives = reth_optimism_primitives::OpPrimitives>,
        Builder::Executor:
            AlloyBlockExecutor<Transaction = OpTransactionSigned, Receipt = OpReceipt>,
        <Builder::Executor as AlloyBlockExecutor>::Result: PreRefundGasUsed,
    {
        let execute_txs_start_time = Instant::now();
        let mut num_txs_considered = 0;
        let mut num_txs_simulated = 0;
        let mut num_txs_simulated_success = 0;
        let mut num_txs_simulated_fail = 0;
        let mut num_bundles_reverted = 0;
        let mut reverted_gas_used = 0;
        let base_fee = self.base_fee();

        let tx_da_limit = self.da_config.max_da_tx_size();

        debug!(
            target: "payload_builder",
            message = "Executing best transactions",
            block_da_limit = ?block_da_limit,
            tx_da_limit = ?tx_da_limit,
            block_gas_limit = ?block_gas_limit,
        );

        let block_attr = BlockConditionalAttributes {
            number: self.block_number(),
            timestamp: self.attributes().timestamp(),
        };
        let interop_failsafe_active = self.interop_failsafe.enabled();

        while let Some(tx) = best_txs.next(()) {
            let interop = tx.interop_deadline();
            let reverted_hashes = tx.reverted_hashes().clone();
            let conditional = tx.conditional().cloned();

            let tx_da_size = tx.estimated_da_size();
            let tx = tx.into_consensus();
            let tx_hash = tx.tx_hash();

            // exclude reverting transaction if:
            // - the transaction comes from a bundle (is_some) and the hash **is not** in reverted hashes
            // Note that we need to use the Option to signal whether the transaction comes from a bundle,
            // otherwise, we would exclude all transactions that are not in the reverted hashes.
            let is_bundle_tx = reverted_hashes.is_some();
            let exclude_reverting_txs =
                is_bundle_tx && !reverted_hashes.unwrap().contains(&tx_hash);

            let log_txn = |result: TxnExecutionResult| {
                debug!(
                    target: "payload_builder",
                    message = "Considering transaction",
                    tx_hash = ?tx_hash,
                    tx_da_size = ?tx_da_size,
                    exclude_reverting_txs = ?exclude_reverting_txs,
                    result = %result,
                );
            };

            num_txs_considered += 1;

            // TODO: ideally we should get this from the txpool stream
            if let Some(conditional) = conditional
                && !conditional.matches_block_attributes(&block_attr)
            {
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue;
            }

            // ensure we still have capacity for this transaction
            if let Err(result) = info.is_tx_over_limits(
                tx_da_size,
                block_gas_limit,
                tx_da_limit,
                block_da_limit,
                tx.gas_limit(),
                info.da_footprint_scalar,
                block_da_footprint_limit,
            ) {
                // we can't fit this transaction into the block, so we need to mark it as
                // invalid which also removes all dependent transaction from
                // the iterator before we can continue
                log_txn(result);
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue;
            }

            // A sequencer's block should never contain blob or deposit transactions from the pool.
            if tx.is_eip4844() || tx.is_deposit() {
                log_txn(TxnExecutionResult::SequencerTransaction);
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue;
            }

            if interop_failsafe_active && is_interop_tx(&*tx) {
                log_txn(TxnExecutionResult::InteropFailed);
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue;
            }

            // We skip invalid cross chain txs, they would be removed on the next block update in
            // the maintenance job
            if let Some(interop) = interop
                && !is_valid_interop(interop, self.config.attributes.timestamp())
            {
                log_txn(TxnExecutionResult::InteropFailed);
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue;
            }

            // check if the job was cancelled, if so we can exit early
            if self.cancel.is_cancelled() {
                return Ok(Some(()));
            }

            let tx_simulation_start_time = Instant::now();
            let mut gas_used = 0;
            let mut evm_gas_used = 0;
            let mut tx_succeeded = false;
            let mut gas_limit_exceeded = false;
            let mut address_limit_exceeded = false;
            // Declining a candidate (CommitChanges::No, below) must not leak SDM block-warming into
            // a later committed tx; that rollback lives in alloy-op-evm's
            // execute_transaction_with_commit_condition override (ethereum-optimism/optimism#21354).
            let committed = match builder.execute_transaction_with_commit_condition(
                tx.clone(),
                |result| {
                    let execution_result = &result.result().result;
                    gas_used = execution_result.tx_gas_used();
                    evm_gas_used = result.evm_gas_used();
                    tx_succeeded = execution_result.is_success();
                    gas_limit_exceeded = self
                        .max_gas_per_txn
                        .is_some_and(|max_gas_per_txn| gas_used > max_gas_per_txn);
                    address_limit_exceeded = self
                        .address_gas_limiter
                        .consume_gas(tx.signer(), gas_used)
                        .is_err();

                    if gas_limit_exceeded
                        || address_limit_exceeded
                        || (!tx_succeeded && exclude_reverting_txs)
                    {
                        CommitChanges::No
                    } else {
                        CommitChanges::Yes
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
                        log_txn(TxnExecutionResult::NonceTooLow);
                        trace!(target: "payload_builder", %error, ?tx, "skipping nonce too low transaction");
                    } else {
                        // if the transaction is invalid, we can skip it and all of its
                        // descendants
                        log_txn(TxnExecutionResult::EvmError);
                        trace!(target: "payload_builder", %error, ?tx, "skipping invalid transaction and its descendants");
                        best_txs.mark_invalid(tx.signer(), tx.nonce());
                    }

                    continue;
                }
                Err(err) => {
                    // this is an error that we should treat as fatal for this attempt
                    log_txn(TxnExecutionResult::EvmError);
                    return Err(PayloadBuilderError::evm(err));
                }
            };

            self.metrics
                .tx_simulation_duration
                .record(tx_simulation_start_time.elapsed());
            self.metrics.tx_byte_size.record(tx.inner().size() as f64);
            num_txs_simulated += 1;

            if tx_succeeded {
                log_txn(TxnExecutionResult::Success);
                num_txs_simulated_success += 1;
                self.metrics.successful_tx_gas_used.record(gas_used as f64);
            } else {
                num_txs_simulated_fail += 1;
                reverted_gas_used += gas_used as i32;
                self.metrics.reverted_tx_gas_used.record(gas_used as f64);
                if is_bundle_tx {
                    num_bundles_reverted += 1;
                }
                if exclude_reverting_txs {
                    log_txn(TxnExecutionResult::RevertedAndExcluded);
                    info!(target: "payload_builder", tx_hash = ?tx.tx_hash(), "skipping reverted transaction");
                    best_txs.mark_invalid(tx.signer(), tx.nonce());
                    continue;
                } else {
                    log_txn(TxnExecutionResult::Reverted);
                }
            }

            if gas_limit_exceeded || address_limit_exceeded {
                log_txn(TxnExecutionResult::MaxGasUsageExceeded);
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue;
            }

            let Some(committed) = committed else {
                continue;
            };

            // Use the canonical (post-SDM-refund) gas from `committed`, not the pre-refund EVM
            // result gas, which would put wrong cumulative gas into receipts and fail op-reth's
            // receipt-root verification. Mirrors `execute_sequencer_transactions`.
            info.cumulative_gas_used += committed.tx_gas_used();
            info.cumulative_evm_gas_used += evm_gas_used;
            info.cumulative_da_bytes_used += tx_da_size;

            info.receipts.push(
                last_receipt_with_cumulative_gas(builder.executor(), info.cumulative_gas_used)
                    .expect("executor must record a receipt for committed tx"),
            );

            // update add to total fees
            let miner_fee = tx
                .effective_tip_per_gas(base_fee)
                .expect("fee is always valid; execution succeeded");
            info.total_fees += U256::from(miner_fee) * U256::from(gas_used);

            // append sender and transaction to the respective lists
            info.executed_senders.push(tx.signer());
            info.executed_transactions.push(tx.into_inner());
        }

        let payload_transaction_simulation_time = execute_txs_start_time.elapsed();
        self.metrics.set_payload_builder_metrics(
            payload_transaction_simulation_time,
            num_txs_considered,
            num_txs_simulated,
            num_txs_simulated_success,
            num_txs_simulated_fail,
            num_bundles_reverted,
            reverted_gas_used,
        );

        debug!(
            target: "payload_builder",
            message = "Completed executing best transactions",
            txs_executed = num_txs_considered,
            txs_applied = num_txs_simulated_success,
            txs_rejected = num_txs_simulated_fail,
            bundles_reverted = num_bundles_reverted,
        );
        Ok(None)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{gas_limiter::args::GasLimiterArgs, tx::FBPooledTransaction};
    use alloy_consensus::{
        Header, SignableTransaction, TxEip1559,
        transaction::{Recovered, TxHashRef},
    };
    use alloy_eips::eip2930::{AccessList, AccessListItem};
    use alloy_primitives::{Address, B256, Bytes, Signature, TxHash, TxKind, U256};
    use reth_evm::ConfigureEvm;
    use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
    use reth_optimism_txpool::{OpPooledTransaction, interop_filter::CROSS_L2_INBOX_ADDRESS};
    use reth_payload_util::PayloadTransactionsFixed;
    use reth_primitives_traits::{Account, SealedHeader};
    use reth_revm::{database::StateProviderDatabase, db::State, test_utils::StateProviderTest};
    use reth_transaction_pool::PoolTransaction;

    fn pooled_tx(
        nonce: u64,
        signer: Address,
        recipient: Address,
        access_list: AccessList,
    ) -> FBPooledTransaction {
        let tx: OpTransactionSigned = TxEip1559 {
            chain_id: 10,
            nonce,
            gas_limit: 100_000,
            max_fee_per_gas: 20_000_000_000,
            max_priority_fee_per_gas: 1,
            to: TxKind::Call(recipient),
            value: U256::ZERO,
            access_list,
            ..Default::default()
        }
        .into_signed(Signature::test_signature())
        .into();
        let encoded_len = tx.encode_2718_len();

        FBPooledTransaction {
            inner: OpPooledTransaction::new(Recovered::new_unchecked(tx, signer), encoded_len),
            reverted_hashes: None,
            flashblock_number_min: None,
            flashblock_number_max: None,
        }
    }

    fn normal_pooled_tx(nonce: u64, signer: Address, recipient: Address) -> FBPooledTransaction {
        pooled_tx(nonce, signer, recipient, AccessList::default())
    }

    fn interop_pooled_tx(nonce: u64, signer: Address, recipient: Address) -> FBPooledTransaction {
        pooled_tx(
            nonce,
            signer,
            recipient,
            AccessList(vec![AccessListItem {
                address: CROSS_L2_INBOX_ADDRESS,
                storage_keys: vec![B256::ZERO],
            }]),
        )
    }

    fn payload_builder_ctx(
        chain_spec: Arc<OpChainSpec>,
        gas_limit: u64,
        interop_failsafe: InteropFailsafe,
        post_exec_mode: PostExecMode,
    ) -> OpPayloadBuilderCtx {
        let parent = SealedHeader::seal_slow(Header {
            gas_limit,
            number: 0,
            timestamp: 0,
            ..Default::default()
        });
        let attributes = OpPayloadBuilderAttributes {
            timestamp: 1,
            gas_limit: Some(gas_limit),
            ..Default::default()
        };
        let block_env_attributes = OpNextBlockEnvAttributes {
            timestamp: attributes.timestamp(),
            suggested_fee_recipient: attributes.suggested_fee_recipient(),
            prev_randao: attributes.prev_randao(),
            gas_limit: attributes.gas_limit.unwrap_or(parent.gas_limit),
            parent_beacon_block_root: attributes.parent_beacon_block_root(),
            extra_data: Default::default(),
        };
        let evm_config = OpEvmConfig::optimism(chain_spec.clone());
        let evm_env = evm_config
            .next_evm_env(&parent, &block_env_attributes)
            .expect("next evm env can be created");

        OpPayloadBuilderCtx {
            evm_config,
            da_config: Default::default(),
            gas_limit_config: Default::default(),
            chain_spec,
            config: PayloadConfig {
                parent_header: Arc::new(parent),
                parent_block_info: None,
                payload_id: attributes.id,
                attributes,
            },
            evm_env,
            block_env_attributes,
            cancel: Default::default(),
            builder_signer: None,
            metrics: Default::default(),
            extra_ctx: (),
            max_gas_per_txn: None,
            address_gas_limiter: AddressGasLimiter::new(GasLimiterArgs::default()),
            post_exec_mode,
            interop_failsafe,
        }
    }

    fn run_execute_best_transactions(
        ctx: OpPayloadBuilderCtx,
        signer: Address,
        txs: Vec<FBPooledTransaction>,
    ) -> Vec<TxHash> {
        let mut state_provider = StateProviderTest::default();
        state_provider.insert_account(
            signer,
            Account {
                balance: U256::MAX,
                ..Default::default()
            },
            None,
            Default::default(),
        );

        let mut best_txs = PayloadTransactionsFixed::new(txs);
        let mut db = State::builder()
            .with_database(StateProviderDatabase::new(&state_provider))
            .with_bundle_update()
            .build();
        let mut builder = ctx
            .block_builder_for_next_block(&mut db)
            .expect("block builder can be created");
        let mut info: ExecutionInfo = ExecutionInfo::default();

        assert!(
            ctx.execute_best_transactions(
                &mut info,
                &mut builder,
                &mut best_txs,
                ctx.block_gas_limit(),
                None,
                None,
            )
            .expect("best transactions execute")
            .is_none()
        );

        info.executed_transactions
            .iter()
            .map(TxHashRef::tx_hash)
            .copied()
            .collect()
    }

    #[test]
    fn execute_best_transactions_excludes_interop_txs_when_failsafe_active() {
        let signer = Address::repeat_byte(0x11);
        let normal = normal_pooled_tx(0, signer, Address::repeat_byte(0x22));
        let interop = interop_pooled_tx(1, signer, Address::repeat_byte(0x33));
        let normal_hash = *normal.hash();
        let interop_hash = *interop.hash();

        let gas_limit = 1_000_000;
        let chain_spec = Arc::new(
            OpChainSpecBuilder::optimism_mainnet()
                .regolith_activated()
                .build(),
        );
        let failsafe = InteropFailsafe::default();
        let build = |failsafe: &InteropFailsafe| {
            run_execute_best_transactions(
                payload_builder_ctx(
                    chain_spec.clone(),
                    gas_limit,
                    failsafe.clone(),
                    PostExecMode::Disabled,
                ),
                signer,
                vec![normal.clone(), interop.clone()],
            )
        };

        failsafe.set(false);
        assert_eq!(build(&failsafe), vec![normal_hash, interop_hash]);

        failsafe.set(true);
        assert_eq!(build(&failsafe), vec![normal_hash]);

        failsafe.set(false);
        assert_eq!(build(&failsafe), vec![normal_hash, interop_hash]);
    }

    /// Runs the tx-selection loop in Produce mode over probe tx `B` (which does a `BALANCE` on
    /// `leak`), optionally preceded by a failing tx `A` from `leak` that the loop attempts, sees
    /// rejected, and skips. `leak_account`/`leak_code` pick the validation error `A` fails with,
    /// after its sender has been loaded and warmed. Returns the block's pre-refund EVM gas, which
    /// for `B`'s account access is 2600 cold or 100 warm — so a warm-set leak from the skipped `A`
    /// shows up as ~2500 gas less.
    fn warm_set_leak_loop_evm_gas(
        chain_spec: Arc<OpChainSpec>,
        gas_limit: u64,
        leak_account: Account,
        leak_code: Option<Bytes>,
        include_failing_a: bool,
    ) -> u64 {
        let leak = Address::repeat_byte(0xAA);
        let probe_sender = Address::repeat_byte(0xBB);
        let probe_contract = Address::repeat_byte(0xCC);

        // Probe runtime: PUSH20 <leak>; BALANCE; POP; STOP.
        let mut probe_code = vec![0x73u8];
        probe_code.extend_from_slice(leak.as_slice());
        probe_code.extend_from_slice(&[0x31, 0x50, 0x00]);

        let mut state_provider = StateProviderTest::default();
        state_provider.insert_account(leak, leak_account, leak_code, Default::default());
        state_provider.insert_account(
            probe_sender,
            Account {
                balance: U256::MAX,
                ..Default::default()
            },
            None,
            Default::default(),
        );
        state_provider.insert_account(
            probe_contract,
            Account {
                balance: U256::ZERO,
                ..Default::default()
            },
            Some(Bytes::from(probe_code)),
            Default::default(),
        );

        let mut txs = Vec::new();
        if include_failing_a {
            txs.push(normal_pooled_tx(0, leak, probe_sender));
        }
        txs.push(normal_pooled_tx(0, probe_sender, probe_contract));

        let ctx = payload_builder_ctx(
            chain_spec,
            gas_limit,
            InteropFailsafe::default(),
            PostExecMode::Produce,
        );
        let mut best_txs = PayloadTransactionsFixed::new(txs);
        let mut db = State::builder()
            .with_database(StateProviderDatabase::new(&state_provider))
            .with_bundle_update()
            .build();
        let mut builder = ctx
            .block_builder_for_next_block(&mut db)
            .expect("block builder can be created");
        let mut info: ExecutionInfo = ExecutionInfo::default();

        assert!(
            ctx.execute_best_transactions(
                &mut info,
                &mut builder,
                &mut best_txs,
                ctx.block_gas_limit(),
                None,
                None,
            )
            .expect("best transactions execute")
            .is_none()
        );

        info.cumulative_evm_gas_used
    }

    /// op-rbuilder mirror of the alloy-op-evm warm-leak regression. In SDM Produce mode the
    /// selection loop skips a failed candidate `A` (NonceTooLow) but must not leave `A`'s sender
    /// EIP-2929-warm for a later included tx `B`. If it does, the builder bills `B` a warm access
    /// for a tx that never entered the block, diverging from a validator that never executed `A`
    /// — a builder-vs-validator gas mismatch that rejects the block. Equal pre-refund EVM gas for
    /// `B` with vs without the skipped `A` means no divergence.
    ///
    /// Fails against an op-revm whose `catch_error` does not discard the journal on non-deposit
    /// errors; passes once it does.
    #[test]
    fn skipped_failed_tx_does_not_warm_later_tx_in_produce_mode() {
        // `leak` at nonce 5, so its nonce-0 tx `A` is rejected NonceTooLow after being warmed.
        let leak_account = Account {
            nonce: 5,
            balance: U256::MAX,
            ..Default::default()
        };
        assert_no_warm_set_leak_divergence(leak_account, None);
    }

    #[test]
    fn skipped_eip3607_failed_tx_does_not_warm_later_tx_in_produce_mode() {
        // `leak` carries code, so its tx `A` is rejected by EIP-3607 (contract may not originate a
        // tx) rather than by a nonce check — still after its sender is loaded and warmed.
        let leak_account = Account {
            balance: U256::MAX,
            ..Default::default()
        };
        assert_no_warm_set_leak_divergence(leak_account, Some(Bytes::from_static(&[0x00]))); // STOP
    }

    /// Asserts the selection loop bills probe tx `B` the same pre-refund EVM gas whether or not a
    /// failing tx `A` (from `leak`) was attempted and skipped before it. A difference means the
    /// skipped `A` left its sender warm, so the builder would diverge from a validator that never
    /// ran `A` — a block-rejecting gas mismatch. Fails against an op-revm whose `catch_error` does
    /// not discard the journal on non-deposit errors; passes once it does.
    fn assert_no_warm_set_leak_divergence(leak_account: Account, leak_code: Option<Bytes>) {
        let gas_limit = 1_000_000;
        // Lagoon-active spec: SDM's protocol gate (`compute_post_exec_mode`) requires it, so the
        // forced Produce mode below runs on a fork combination that can occur in production.
        let chain_spec = Arc::new(
            OpChainSpecBuilder::optimism_mainnet()
                .interop_activated()
                .build(),
        );

        let with_failing_a = warm_set_leak_loop_evm_gas(
            chain_spec.clone(),
            gas_limit,
            leak_account,
            leak_code.clone(),
            true,
        );
        let without =
            warm_set_leak_loop_evm_gas(chain_spec, gas_limit, leak_account, leak_code, false);

        assert_eq!(
            without,
            with_failing_a,
            "a skipped failed candidate warmed a later included tx: {}-gas builder-vs-validator \
             EVM-gas divergence",
            without.abs_diff(with_failing_a),
        );
    }
}
