//! Optimism payload builder implementation.

use crate::{
    config::{OpBuilderConfig, OpDAConfig},
    error::OpPayloadBuilderError,
    payload::{OpBuiltPayload, OpPayloadBuilderAttributes},
    OpPayloadPrimitives,
};
use alloy_consensus::{Transaction, Typed2718};
use alloy_primitives::{Bytes, B256, U256};
use alloy_rlp::Encodable;
use alloy_rpc_types_debug::ExecutionWitness;
use alloy_rpc_types_engine::PayloadId;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use reth_basic_payload_builder::*;
use reth_chain_state::{ExecutedBlock, ExecutedBlockWithTrieUpdates};
use reth_chainspec::{ChainSpecProvider, EthChainSpec};
use reth_evm::{
    execute::{
        BlockBuilder, BlockBuilderOutcome, BlockExecutionError, BlockExecutor, BlockValidationError,
    },
    ConfigureEvm, Database, Evm,
};
use reth_execution_types::ExecutionOutcome;
use reth_optimism_evm::OpNextBlockEnvAttributes;
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::{transaction::OpTransaction, ADDRESS_L2_TO_L1_MESSAGE_PASSER};
use reth_optimism_txpool::{
    interop::{is_valid_interop, MaybeInteropTransaction},
    OpPooledTx,
};
use reth_payload_builder_primitives::PayloadBuilderError;
use reth_payload_primitives::PayloadBuilderAttributes;
use reth_payload_util::{BestPayloadTransactions, NoopPayloadTransactions, PayloadTransactions};
use reth_primitives_traits::{NodePrimitives, SealedHeader, SignedTransaction, TxTy};
use reth_revm::{
    cancelled::CancelOnDrop, database::StateProviderDatabase, db::State,
    witness::ExecutionWitnessRecord,
};
use reth_storage_api::{errors::ProviderError, StateProvider, StateProviderFactory};
use reth_transaction_pool::{BestTransactionsAttributes, PoolTransaction, TransactionPool};
use revm::context::{Block, BlockEnv};
use std::sync::Arc;
use tracing::{debug, trace, warn};

/// Optimism's payload builder
#[derive(Debug, Clone)]
pub struct OpPayloadBuilder<Pool, Client, Evm, Txs = ()> {
    /// The rollup's compute pending block configuration option.
    // TODO(clabby): Implement this feature.
    pub compute_pending_block: bool,
    /// The type responsible for creating the evm.
    pub evm_config: Evm,
    /// Transaction pool.
    pub pool: Pool,
    /// Node client.
    pub client: Client,
    /// Settings for the builder, e.g. DA settings.
    pub config: OpBuilderConfig,
    /// The type responsible for yielding the best transactions for the payload if mempool
    /// transactions are allowed.
    pub best_transactions: Txs,
}

impl<Pool, Client, Evm> OpPayloadBuilder<Pool, Client, Evm> {
    /// `OpPayloadBuilder` constructor.
    ///
    /// Configures the builder with the default settings.
    pub fn new(pool: Pool, client: Client, evm_config: Evm) -> Self {
        Self::with_builder_config(pool, client, evm_config, Default::default())
    }

    /// Configures the builder with the given [`OpBuilderConfig`].
    pub const fn with_builder_config(
        pool: Pool,
        client: Client,
        evm_config: Evm,
        config: OpBuilderConfig,
    ) -> Self {
        Self {
            pool,
            client,
            compute_pending_block: true,
            evm_config,
            config,
            best_transactions: (),
        }
    }
}

impl<Pool, Client, Evm, Txs> OpPayloadBuilder<Pool, Client, Evm, Txs> {
    /// Sets the rollup's compute pending block configuration option.
    pub const fn set_compute_pending_block(mut self, compute_pending_block: bool) -> Self {
        self.compute_pending_block = compute_pending_block;
        self
    }

    /// Configures the type responsible for yielding the transactions that should be included in the
    /// payload.
    pub fn with_transactions<T>(
        self,
        best_transactions: T,
    ) -> OpPayloadBuilder<Pool, Client, Evm, T> {
        let Self { pool, client, compute_pending_block, evm_config, config, .. } = self;
        OpPayloadBuilder {
            pool,
            client,
            compute_pending_block,
            evm_config,
            best_transactions,
            config,
        }
    }

    /// Enables the rollup's compute pending block configuration option.
    pub const fn compute_pending_block(self) -> Self {
        self.set_compute_pending_block(true)
    }

    /// Returns the rollup's compute pending block configuration option.
    pub const fn is_compute_pending_block(&self) -> bool {
        self.compute_pending_block
    }
}

impl<Pool, Client, Evm, N, T> OpPayloadBuilder<Pool, Client, Evm, T>
where
    Pool: TransactionPool<Transaction: OpPooledTx<Consensus = N::SignedTx>>,
    Client: StateProviderFactory + ChainSpecProvider<ChainSpec: EthChainSpec + OpHardforks>,
    N: OpPayloadPrimitives,
    Evm: ConfigureEvm<Primitives = N, NextBlockEnvCtx = OpNextBlockEnvAttributes>,
{
    /// Constructs an Optimism payload from the transactions sent via the
    /// Payload attributes by the sequencer. If the `no_tx_pool` argument is passed in
    /// the payload attributes, the transaction pool will be ignored and the only transactions
    /// included in the payload will be those sent through the attributes.
    ///
    /// Given build arguments including an Optimism client, transaction pool,
    /// and configuration, this function creates a transaction payload. Returns
    /// a result indicating success with the payload or an error in case of failure.
    fn build_payload<'a, Txs>(
        &self,
        args: BuildArguments<OpPayloadBuilderAttributes<N::SignedTx>, OpBuiltPayload<N>>,
        best: impl FnOnce(BestTransactionsAttributes) -> Txs + Send + Sync + 'a,
    ) -> Result<BuildOutcome<OpBuiltPayload<N>>, PayloadBuilderError>
    where
        Txs: PayloadTransactions<
            Transaction: PoolTransaction<Consensus = N::SignedTx> + MaybeInteropTransaction,
        >,
    {
        let BuildArguments { mut cached_reads, config, cancel, best_payload } = args;

        let ctx = OpPayloadBuilderCtx {
            evm_config: self.evm_config.clone(),
            da_config: self.config.da_config.clone(),
            chain_spec: self.client.chain_spec(),
            config,
            cancel,
            best_payload,
        };

        let builder = OpBuilder::new(best);

        let state_provider = self.client.state_by_block_hash(ctx.parent().hash())?;
        let state = StateProviderDatabase::new(&state_provider);

        if ctx.attributes().no_tx_pool {
            builder.build(state, &state_provider, ctx)
        } else {
            // sequencer mode we can reuse cachedreads from previous runs
            builder.build(cached_reads.as_db_mut(state), &state_provider, ctx)
        }
        .map(|out| out.with_cached_reads(cached_reads))
    }

    /// Computes the witness for the payload.
    pub fn payload_witness(
        &self,
        parent: SealedHeader,
        attributes: OpPayloadAttributes,
    ) -> Result<ExecutionWitness, PayloadBuilderError> {
        let attributes = OpPayloadBuilderAttributes::try_new(parent.hash(), attributes, 3)
            .map_err(PayloadBuilderError::other)?;

        let config = PayloadConfig { parent_header: Arc::new(parent), attributes };
        let ctx = OpPayloadBuilderCtx {
            evm_config: self.evm_config.clone(),
            da_config: self.config.da_config.clone(),
            chain_spec: self.client.chain_spec(),
            config,
            cancel: Default::default(),
            best_payload: Default::default(),
        };

        let state_provider = self.client.state_by_block_hash(ctx.parent().hash())?;

        let builder = OpBuilder::new(|_| NoopPayloadTransactions::<Pool::Transaction>::default());
        builder.witness(state_provider, &ctx)
    }
}

/// Implementation of the [`PayloadBuilder`] trait for [`OpPayloadBuilder`].
impl<Pool, Client, Evm, N, Txs> PayloadBuilder for OpPayloadBuilder<Pool, Client, Evm, Txs>
where
    Client: StateProviderFactory + ChainSpecProvider<ChainSpec: EthChainSpec + OpHardforks> + Clone,
    N: OpPayloadPrimitives,
    Pool: TransactionPool<Transaction: OpPooledTx<Consensus = N::SignedTx>>,
    Evm: ConfigureEvm<Primitives = N, NextBlockEnvCtx = OpNextBlockEnvAttributes>,
    Txs: OpPayloadTransactions<Pool::Transaction>,
{
    type Attributes = OpPayloadBuilderAttributes<N::SignedTx>;
    type BuiltPayload = OpBuiltPayload<N>;

    fn try_build(
        &self,
        args: BuildArguments<Self::Attributes, Self::BuiltPayload>,
    ) -> Result<BuildOutcome<Self::BuiltPayload>, PayloadBuilderError> {
        let pool = self.pool.clone();
        self.build_payload(args, |attrs| self.best_transactions.best_transactions(pool, attrs))
    }

    fn on_missing_payload(
        &self,
        _args: BuildArguments<Self::Attributes, Self::BuiltPayload>,
    ) -> MissingPayloadBehaviour<Self::BuiltPayload> {
        // we want to await the job that's already in progress because that should be returned as
        // is, there's no benefit in racing another job
        MissingPayloadBehaviour::AwaitInProgress
    }

    // NOTE: this should only be used for testing purposes because this doesn't have access to L1
    // system txs, hence on_missing_payload we return [MissingPayloadBehaviour::AwaitInProgress].
    fn build_empty_payload(
        &self,
        config: PayloadConfig<Self::Attributes>,
    ) -> Result<Self::BuiltPayload, PayloadBuilderError> {
        let args = BuildArguments {
            config,
            cached_reads: Default::default(),
            cancel: Default::default(),
            best_payload: None,
        };
        self.build_payload(args, |_| NoopPayloadTransactions::<Pool::Transaction>::default())?
            .into_payload()
            .ok_or_else(|| PayloadBuilderError::MissingPayload)
    }
}

/// The type that builds the payload.
///
/// Payload building for optimism is composed of several steps.
/// The first steps are mandatory and defined by the protocol.
///
/// 1. first all System calls are applied.
/// 2. After canyon the forced deployed `create2deployer` must be loaded
/// 3. all sequencer transactions are executed (part of the payload attributes)
///
/// Depending on whether the node acts as a sequencer and is allowed to include additional
/// transactions (`no_tx_pool == false`):
/// 4. include additional transactions
///
/// And finally
/// 5. build the block: compute all roots (txs, state)
#[derive(derive_more::Debug)]
pub struct OpBuilder<'a, Txs> {
    /// Yields the best transaction to include if transactions from the mempool are allowed.
    #[debug(skip)]
    best: Box<dyn FnOnce(BestTransactionsAttributes) -> Txs + 'a>,
}

impl<'a, Txs> OpBuilder<'a, Txs> {
    /// Creates a new [`OpBuilder`].
    pub fn new(best: impl FnOnce(BestTransactionsAttributes) -> Txs + Send + Sync + 'a) -> Self {
        Self { best: Box::new(best) }
    }
}

impl<Txs> OpBuilder<'_, Txs> {
    /// Builds the payload on top of the state.
    pub fn build<EvmConfig, ChainSpec, N>(
        self,
        db: impl Database<Error = ProviderError>,
        state_provider: impl StateProvider,
        ctx: OpPayloadBuilderCtx<EvmConfig, ChainSpec>,
    ) -> Result<BuildOutcomeKind<OpBuiltPayload<N>>, PayloadBuilderError>
    where
        EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx = OpNextBlockEnvAttributes>,
        ChainSpec: EthChainSpec + OpHardforks,
        N: OpPayloadPrimitives,
        Txs: PayloadTransactions<
            Transaction: PoolTransaction<Consensus = N::SignedTx> + MaybeInteropTransaction,
        >,
    {
        let Self { best } = self;
        debug!(target: "payload_builder", id=%ctx.payload_id(), parent_header = ?ctx.parent().hash(), parent_number = ctx.parent().number, "building new payload");

        let mut db = State::builder().with_database(db).with_bundle_update().build();

        let mut builder = ctx.block_builder(&mut db)?;

        // 1. apply pre-execution changes
        builder.apply_pre_execution_changes().map_err(|err| {
            warn!(target: "payload_builder", %err, "failed to apply pre-execution changes");
            PayloadBuilderError::Internal(err.into())
        })?;

        // 2. execute sequencer transactions
        let mut info = ctx.execute_sequencer_transactions(&mut builder)?;

        // 3. if mem pool transactions are requested we execute them
        if !ctx.attributes().no_tx_pool {
            let best_txs = best(ctx.best_transaction_attributes(builder.evm_mut().block()));
            if ctx.execute_best_transactions(&mut info, &mut builder, best_txs)?.is_some() {
                return Ok(BuildOutcomeKind::Cancelled)
            }

            // check if the new payload is even more valuable
            if !ctx.is_better_payload(info.total_fees) {
                // can skip building the block
                return Ok(BuildOutcomeKind::Aborted { fees: info.total_fees })
            }
        }

        let BlockBuilderOutcome { execution_result, hashed_state, trie_updates, block } =
            builder.finish(state_provider)?;

        let sealed_block = Arc::new(block.sealed_block().clone());
        debug!(target: "payload_builder", id=%ctx.attributes().payload_id(), sealed_block_header = ?sealed_block.header(), "sealed built block");

        let execution_outcome = ExecutionOutcome::new(
            db.take_bundle(),
            vec![execution_result.receipts],
            block.number,
            Vec::new(),
        );

        // create the executed block data
        let executed: ExecutedBlockWithTrieUpdates<N> = ExecutedBlockWithTrieUpdates {
            block: ExecutedBlock {
                recovered_block: Arc::new(block),
                execution_output: Arc::new(execution_outcome),
                hashed_state: Arc::new(hashed_state),
            },
            trie: Arc::new(trie_updates),
        };

        let no_tx_pool = ctx.attributes().no_tx_pool;

        let payload =
            OpBuiltPayload::new(ctx.payload_id(), sealed_block, info.total_fees, Some(executed));

        if no_tx_pool {
            // if `no_tx_pool` is set only transactions from the payload attributes will be included
            // in the payload. In other words, the payload is deterministic and we can
            // freeze it once we've successfully built it.
            Ok(BuildOutcomeKind::Freeze(payload))
        } else {
            Ok(BuildOutcomeKind::Better { payload })
        }
    }

    /// Builds the payload and returns its [`ExecutionWitness`] based on the state after execution.
    pub fn witness<Evm, ChainSpec, N>(
        self,
        state_provider: impl StateProvider,
        ctx: &OpPayloadBuilderCtx<Evm, ChainSpec>,
    ) -> Result<ExecutionWitness, PayloadBuilderError>
    where
        Evm: ConfigureEvm<Primitives = N, NextBlockEnvCtx = OpNextBlockEnvAttributes>,
        ChainSpec: EthChainSpec + OpHardforks,
        N: OpPayloadPrimitives,
        Txs: PayloadTransactions<Transaction: PoolTransaction<Consensus = N::SignedTx>>,
    {
        let mut db = State::builder()
            .with_database(StateProviderDatabase::new(&state_provider))
            .with_bundle_update()
            .build();
        let mut builder = ctx.block_builder(&mut db)?;

        builder.apply_pre_execution_changes()?;
        ctx.execute_sequencer_transactions(&mut builder)?;
        builder.into_executor().apply_post_execution_changes()?;

        if ctx.chain_spec.is_isthmus_active_at_timestamp(ctx.attributes().timestamp()) {
            // force load `L2ToL1MessagePasser.sol` so l2 withdrawals root can be computed even if
            // no l2 withdrawals in block
            _ = db.load_cache_account(ADDRESS_L2_TO_L1_MESSAGE_PASSER)?;
        }

        let ExecutionWitnessRecord { hashed_state, codes, keys, lowest_block_number: _ } =
            ExecutionWitnessRecord::from_executed_state(&db);
        let state = state_provider.witness(Default::default(), hashed_state)?;
        Ok(ExecutionWitness {
            state: state.into_iter().collect(),
            codes,
            keys,
            ..Default::default()
        })
    }
}

/// A type that returns a the [`PayloadTransactions`] that should be included in the pool.
pub trait OpPayloadTransactions<Transaction>: Clone + Send + Sync + Unpin + 'static {
    /// Returns an iterator that yields the transaction in the order they should get included in the
    /// new payload.
    fn best_transactions<Pool: TransactionPool<Transaction = Transaction>>(
        &self,
        pool: Pool,
        attr: BestTransactionsAttributes,
    ) -> impl PayloadTransactions<Transaction = Transaction>;
}

impl<T: PoolTransaction + MaybeInteropTransaction> OpPayloadTransactions<T> for () {
    fn best_transactions<Pool: TransactionPool<Transaction = T>>(
        &self,
        pool: Pool,
        attr: BestTransactionsAttributes,
    ) -> impl PayloadTransactions<Transaction = T> {
        BestPayloadTransactions::new(pool.best_transactions_with_attributes(attr))
    }
}

/// Holds the state after execution
#[derive(Debug)]
pub struct ExecutedPayload<N: NodePrimitives> {
    /// Tracked execution info
    pub info: ExecutionInfo,
    /// Withdrawal hash.
    pub withdrawals_root: Option<B256>,
    /// The transaction receipts.
    pub receipts: Vec<N::Receipt>,
    /// The block env used during execution.
    pub block_env: BlockEnv,
}

/// This acts as the container for executed transactions and its byproducts (receipts, gas used)
#[derive(Default, Debug)]
pub struct ExecutionInfo {
    /// All gas used so far
    pub cumulative_gas_used: u64,
    /// Estimated DA size
    pub cumulative_da_bytes_used: u64,
    /// Tracks fees from executed mempool transactions
    pub total_fees: U256,
}

impl ExecutionInfo {
    /// Create a new instance with allocated slots.
    pub const fn new() -> Self {
        Self { cumulative_gas_used: 0, cumulative_da_bytes_used: 0, total_fees: U256::ZERO }
    }

    /// Returns true if the transaction would exceed the block limits:
    /// - block gas limit: ensures the transaction still fits into the block.
    /// - tx DA limit: if configured, ensures the tx does not exceed the maximum allowed DA limit
    ///   per tx.
    /// - block DA limit: if configured, ensures the transaction's DA size does not exceed the
    ///   maximum allowed DA limit per block.
    pub fn is_tx_over_limits(
        &self,
        tx: &(impl Encodable + Transaction),
        block_gas_limit: u64,
        tx_data_limit: Option<u64>,
        block_data_limit: Option<u64>,
    ) -> bool {
        if tx_data_limit.is_some_and(|da_limit| tx.length() as u64 > da_limit) {
            return true;
        }

        if block_data_limit
            .is_some_and(|da_limit| self.cumulative_da_bytes_used + (tx.length() as u64) > da_limit)
        {
            return true;
        }

        self.cumulative_gas_used + tx.gas_limit() > block_gas_limit
    }
}

/// Container type that holds all necessities to build a new payload.
#[derive(derive_more::Debug)]
pub struct OpPayloadBuilderCtx<Evm: ConfigureEvm, ChainSpec> {
    /// The type that knows how to perform system calls and configure the evm.
    pub evm_config: Evm,
    /// The DA config for the payload builder
    pub da_config: OpDAConfig,
    /// The chainspec
    pub chain_spec: Arc<ChainSpec>,
    /// How to build the payload.
    pub config: PayloadConfig<OpPayloadBuilderAttributes<TxTy<Evm::Primitives>>>,
    /// Marker to check whether the job has been cancelled.
    pub cancel: CancelOnDrop,
    /// The currently best payload.
    pub best_payload: Option<OpBuiltPayload<Evm::Primitives>>,
}

impl<Evm, ChainSpec> OpPayloadBuilderCtx<Evm, ChainSpec>
where
    Evm: ConfigureEvm<Primitives: OpPayloadPrimitives, NextBlockEnvCtx = OpNextBlockEnvAttributes>,
    ChainSpec: EthChainSpec + OpHardforks,
{
    /// Returns the parent block the payload will be build on.
    pub fn parent(&self) -> &SealedHeader {
        &self.config.parent_header
    }

    /// Returns the builder attributes.
    pub const fn attributes(&self) -> &OpPayloadBuilderAttributes<TxTy<Evm::Primitives>> {
        &self.config.attributes
    }

    /// Returns the extra data for the block.
    ///
    /// After holocene this extracts the extra data from the payload
    pub fn extra_data(&self) -> Result<Bytes, PayloadBuilderError> {
        if self.is_holocene_active() {
            self.attributes()
                .get_holocene_extra_data(
                    self.chain_spec.base_fee_params_at_timestamp(
                        self.attributes().payload_attributes.timestamp,
                    ),
                )
                .map_err(PayloadBuilderError::other)
        } else {
            Ok(Default::default())
        }
    }

    /// Returns the current fee settings for transactions from the mempool
    pub fn best_transaction_attributes(&self, block_env: &BlockEnv) -> BestTransactionsAttributes {
        BestTransactionsAttributes::new(
            block_env.basefee,
            block_env.blob_gasprice().map(|p| p as u64),
        )
    }

    /// Returns the unique id for this payload job.
    pub fn payload_id(&self) -> PayloadId {
        self.attributes().payload_id()
    }

    /// Returns true if holocene is active for the payload.
    pub fn is_holocene_active(&self) -> bool {
        self.chain_spec.is_holocene_active_at_timestamp(self.attributes().timestamp())
    }

    /// Returns true if the fees are higher than the previous payload.
    pub fn is_better_payload(&self, total_fees: U256) -> bool {
        is_better_payload(self.best_payload.as_ref(), total_fees)
    }

    /// Prepares a [`BlockBuilder`] for the next block.
    pub fn block_builder<'a, DB: Database>(
        &'a self,
        db: &'a mut State<DB>,
    ) -> Result<impl BlockBuilder<Primitives = Evm::Primitives> + 'a, PayloadBuilderError> {
        self.evm_config
            .builder_for_next_block(
                db,
                self.parent(),
                OpNextBlockEnvAttributes {
                    timestamp: self.attributes().timestamp(),
                    suggested_fee_recipient: self.attributes().suggested_fee_recipient(),
                    prev_randao: self.attributes().prev_randao(),
                    gas_limit: self.attributes().gas_limit.unwrap_or(self.parent().gas_limit),
                    parent_beacon_block_root: self.attributes().parent_beacon_block_root(),
                    extra_data: self.extra_data()?,
                },
            )
            .map_err(PayloadBuilderError::other)
    }
}

impl<Evm, ChainSpec> OpPayloadBuilderCtx<Evm, ChainSpec>
where
    Evm: ConfigureEvm<Primitives: OpPayloadPrimitives, NextBlockEnvCtx = OpNextBlockEnvAttributes>,
    ChainSpec: EthChainSpec + OpHardforks,
{
    /// Executes all sequencer transactions that are included in the payload attributes.
    pub fn execute_sequencer_transactions(
        &self,
        builder: &mut impl BlockBuilder<Primitives = Evm::Primitives>,
    ) -> Result<ExecutionInfo, PayloadBuilderError> {
        let mut info = ExecutionInfo::new();

        for sequencer_tx in &self.attributes().transactions {
            // A sequencer's block should never contain blob transactions.
            if sequencer_tx.value().is_eip4844() {
                return Err(PayloadBuilderError::other(
                    OpPayloadBuilderError::BlobTransactionRejected,
                ))
            }

            // Convert the transaction to a [RecoveredTx]. This is
            // purely for the purposes of utilizing the `evm_config.tx_env`` function.
            // Deposit transactions do not have signatures, so if the tx is a deposit, this
            // will just pull in its `from` address.
            let sequencer_tx = sequencer_tx.value().try_clone_into_recovered().map_err(|_| {
                PayloadBuilderError::other(OpPayloadBuilderError::TransactionEcRecoverFailed)
            })?;

            let gas_used = match builder.execute_transaction(sequencer_tx.clone()) {
                Ok(gas_used) => gas_used,
                Err(BlockExecutionError::Validation(BlockValidationError::InvalidTx {
                    error,
                    ..
                })) => {
                    trace!(target: "payload_builder", %error, ?sequencer_tx, "Error in sequencer transaction, skipping.");
                    continue
                }
                Err(err) => {
                    // this is an error that we should treat as fatal for this attempt
                    return Err(PayloadBuilderError::EvmExecutionError(Box::new(err)))
                }
            };

            // add gas used by the transaction to cumulative gas used, before creating the receipt
            info.cumulative_gas_used += gas_used;
        }

        Ok(info)
    }

    /// Executes the given best transactions and updates the execution info.
    ///
    /// Returns `Ok(Some(())` if the job was cancelled.
    pub fn execute_best_transactions(
        &self,
        info: &mut ExecutionInfo,
        builder: &mut impl BlockBuilder<Primitives = Evm::Primitives>,
        mut best_txs: impl PayloadTransactions<
            Transaction: PoolTransaction<Consensus = TxTy<Evm::Primitives>>
                             + MaybeInteropTransaction,
        >,
    ) -> Result<Option<()>, PayloadBuilderError> {
        let block_gas_limit = builder.evm_mut().block().gas_limit;
        let block_da_limit = self.da_config.max_da_block_size();
        let tx_da_limit = self.da_config.max_da_tx_size();
        let base_fee = builder.evm_mut().block().basefee;

        while let Some(tx) = best_txs.next(()) {
            let interop = tx.interop_deadline();
            let tx = tx.into_consensus();
            if info.is_tx_over_limits(tx.inner(), block_gas_limit, tx_da_limit, block_da_limit) {
                // we can't fit this transaction into the block, so we need to mark it as
                // invalid which also removes all dependent transaction from
                // the iterator before we can continue
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue
            }

            // A sequencer's block should never contain blob or deposit transactions from the pool.
            if tx.is_eip4844() || tx.is_deposit() {
                best_txs.mark_invalid(tx.signer(), tx.nonce());
                continue
            }

            // We skip invalid cross chain txs, they would be removed on the next block update in
            // the maintenance job
            if let Some(interop) = interop {
                if !is_valid_interop(interop, self.config.attributes.timestamp()) {
                    best_txs.mark_invalid(tx.signer(), tx.nonce());
                    continue
                }
            }
            // check if the job was cancelled, if so we can exit early
            if self.cancel.is_cancelled() {
                return Ok(Some(()))
            }

            let gas_used = match builder.execute_transaction(tx.clone()) {
                Ok(gas_used) => gas_used,
                Err(BlockExecutionError::Validation(BlockValidationError::InvalidTx {
                    error,
                    ..
                })) => {
                    if error.is_nonce_too_low() {
                        // if the nonce is too low, we can skip this transaction
                        trace!(target: "payload_builder", %error, ?tx, "skipping nonce too low transaction");
                    } else {
                        // if the transaction is invalid, we can skip it and all of its
                        // descendants
                        trace!(target: "payload_builder", %error, ?tx, "skipping invalid transaction and its descendants");
                        best_txs.mark_invalid(tx.signer(), tx.nonce());
                    }
                    continue
                }
                Err(err) => {
                    // this is an error that we should treat as fatal for this attempt
                    return Err(PayloadBuilderError::EvmExecutionError(Box::new(err)))
                }
            };

            // add gas used by the transaction to cumulative gas used, before creating the
            // receipt
            info.cumulative_gas_used += gas_used;
            info.cumulative_da_bytes_used += tx.length() as u64;

            // update add to total fees
            let miner_fee = tx
                .effective_tip_per_gas(base_fee)
                .expect("fee is always valid; execution succeeded");
            info.total_fees += U256::from(miner_fee) * U256::from(gas_used);
        }

        Ok(None)
    }
}
