use super::super::context::OpPayloadBuilderCtx;
use crate::{
    builders::{BuilderConfig, BuilderTransactions, generator::BuildArguments},
    gas_limiter::AddressGasLimiter,
    metrics::OpRBuilderMetrics,
    primitives::reth::ExecutionInfo,
    traits::{ClientBounds, PayloadTxsBounds, PoolBounds},
};
use alloy_consensus::{
    BlockBody, EMPTY_OMMER_ROOT_HASH, Header, constants::EMPTY_WITHDRAWALS, proofs,
};
use alloy_eips::{eip7685::EMPTY_REQUESTS_HASH, merge::BEACON_NONCE};
use alloy_evm::Database;
use alloy_primitives::U256;
use reth::payload::PayloadBuilderAttributes;
use reth_basic_payload_builder::{BuildOutcome, BuildOutcomeKind, MissingPayloadBehaviour};
use reth_chain_state::ExecutedBlock;
use reth_evm::{ConfigureEvm, execute::BlockBuilder};
use reth_node_api::{Block, PayloadBuilderError};
use reth_optimism_consensus::{calculate_receipt_root_no_memo_optimism, isthmus};
use reth_optimism_evm::{OpEvmConfig, OpNextBlockEnvAttributes};
use reth_optimism_forks::OpHardforks;
use reth_optimism_node::{OpBuiltPayload, OpPayloadBuilderAttributes};
use reth_optimism_primitives::OpTransactionSigned;
use reth_payload_util::{BestPayloadTransactions, NoopPayloadTransactions, PayloadTransactions};
use reth_primitives::RecoveredBlock;
use reth_primitives_traits::InMemorySize;
use reth_provider::{ExecutionOutcome, StateProvider};
use reth_revm::{
    State, database::StateProviderDatabase, db::states::bundle_state::BundleRetention,
};
use reth_transaction_pool::{
    BestTransactions, BestTransactionsAttributes, PoolTransaction, TransactionPool,
};
use std::{sync::Arc, time::Instant};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};

/// Optimism's payload builder
#[derive(Debug, Clone)]
pub(super) struct StandardOpPayloadBuilder<Pool, Client, BuilderTx, Txs = ()> {
    /// The type responsible for creating the evm.
    pub evm_config: OpEvmConfig,
    /// The transaction pool
    pub pool: Pool,
    /// Node client
    pub client: Client,
    /// Settings for the builder, e.g. DA settings.
    pub config: BuilderConfig<()>,
    /// The type responsible for yielding the best transactions for the payload if mempool
    /// transactions are allowed.
    pub best_transactions: Txs,
    /// The metrics for the builder
    pub metrics: Arc<OpRBuilderMetrics>,
    /// Rate limiting based on gas. This is an optional feature.
    pub address_gas_limiter: AddressGasLimiter,
    /// The type responsible for creating the builder transactions
    pub builder_tx: BuilderTx,
}

impl<Pool, Client, BuilderTx> StandardOpPayloadBuilder<Pool, Client, BuilderTx> {
    /// `OpPayloadBuilder` constructor.
    pub(super) fn new(
        evm_config: OpEvmConfig,
        pool: Pool,
        client: Client,
        config: BuilderConfig<()>,
        builder_tx: BuilderTx,
    ) -> Self {
        let address_gas_limiter = AddressGasLimiter::new(config.gas_limiter_config.clone());
        Self {
            pool,
            client,
            config,
            evm_config,
            best_transactions: (),
            metrics: Default::default(),
            address_gas_limiter,
            builder_tx,
        }
    }
}

/// A type that returns a the [`PayloadTransactions`] that should be included in the pool.
pub(super) trait OpPayloadTransactions<Transaction>:
    Clone + Send + Sync + Unpin + 'static
{
    /// Returns an iterator that yields the transaction in the order they should get included in the
    /// new payload.
    fn best_transactions<Pool: TransactionPool<Transaction = Transaction>>(
        &self,
        pool: Pool,
        attr: BestTransactionsAttributes,
    ) -> impl PayloadTransactions<Transaction = Transaction>;
}

impl<T: PoolTransaction> OpPayloadTransactions<T> for () {
    fn best_transactions<Pool: TransactionPool<Transaction = T>>(
        &self,
        pool: Pool,
        attr: BestTransactionsAttributes,
    ) -> impl PayloadTransactions<Transaction = T> {
        // TODO: once this issue is fixed we could remove without_updates and rely on regular impl
        // https://github.com/paradigmxyz/reth/issues/17325
        BestPayloadTransactions::new(
            pool.best_transactions_with_attributes(attr)
                .without_updates(),
        )
    }
}

impl<Pool, Client, BuilderTx, Txs> reth_basic_payload_builder::PayloadBuilder
    for StandardOpPayloadBuilder<Pool, Client, BuilderTx, Txs>
where
    Pool: PoolBounds,
    Client: ClientBounds,
    BuilderTx: BuilderTransactions + Clone + Send + Sync,
    Txs: OpPayloadTransactions<Pool::Transaction>,
{
    type Attributes = OpPayloadBuilderAttributes<OpTransactionSigned>;
    type BuiltPayload = OpBuiltPayload;

    fn try_build(
        &self,
        args: reth_basic_payload_builder::BuildArguments<Self::Attributes, Self::BuiltPayload>,
    ) -> Result<BuildOutcome<Self::BuiltPayload>, PayloadBuilderError> {
        let pool = self.pool.clone();

        let reth_basic_payload_builder::BuildArguments {
            cached_reads,
            config,
            cancel: _, // TODO
            best_payload: _,
        } = args;

        let args = BuildArguments {
            cached_reads,
            config,
            cancel: CancellationToken::new(),
        };

        self.build_payload(args, |attrs| {
            #[allow(clippy::unit_arg)]
            self.best_transactions
                .best_transactions(pool.clone(), attrs)
        })
    }

    fn on_missing_payload(
        &self,
        _args: reth_basic_payload_builder::BuildArguments<Self::Attributes, Self::BuiltPayload>,
    ) -> MissingPayloadBehaviour<Self::BuiltPayload> {
        MissingPayloadBehaviour::AwaitInProgress
    }

    fn build_empty_payload(
        &self,
        config: reth_basic_payload_builder::PayloadConfig<
            Self::Attributes,
            reth_basic_payload_builder::HeaderForPayload<Self::BuiltPayload>,
        >,
    ) -> Result<Self::BuiltPayload, PayloadBuilderError> {
        let args = BuildArguments {
            config,
            cached_reads: Default::default(),
            cancel: Default::default(),
        };
        self.build_payload(args, |_| {
            NoopPayloadTransactions::<Pool::Transaction>::default()
        })?
        .into_payload()
        .ok_or_else(|| PayloadBuilderError::MissingPayload)
    }
}

impl<Pool, Client, BuilderTx, T> StandardOpPayloadBuilder<Pool, Client, BuilderTx, T>
where
    Pool: PoolBounds,
    Client: ClientBounds,
    BuilderTx: BuilderTransactions + Clone,
{
    /// Constructs an Optimism payload from the transactions sent via the
    /// Payload attributes by the sequencer. If the `no_tx_pool` argument is passed in
    /// the payload attributes, the transaction pool will be ignored and the only transactions
    /// included in the payload will be those sent through the attributes.
    ///
    /// Given build arguments including an Optimism client, transaction pool,
    /// and configuration, this function creates a transaction payload. Returns
    /// a result indicating success with the payload or an error in case of failure.
    fn build_payload<'a, Txs: PayloadTxsBounds>(
        &self,
        args: BuildArguments<OpPayloadBuilderAttributes<OpTransactionSigned>, OpBuiltPayload>,
        best: impl FnOnce(BestTransactionsAttributes) -> Txs + Send + Sync + 'a,
    ) -> Result<BuildOutcome<OpBuiltPayload>, PayloadBuilderError> {
        let block_build_start_time = Instant::now();

        let BuildArguments {
            mut cached_reads,
            config,
            cancel,
        } = args;

        let chain_spec = self.client.chain_spec();
        let timestamp = config.attributes.timestamp();

        let extra_data = if chain_spec.is_jovian_active_at_timestamp(timestamp) {
            config
                .attributes
                .get_jovian_extra_data(chain_spec.base_fee_params_at_timestamp(timestamp))
                .map_err(PayloadBuilderError::other)?
        } else if chain_spec.is_holocene_active_at_timestamp(timestamp) {
            config
                .attributes
                .get_holocene_extra_data(chain_spec.base_fee_params_at_timestamp(timestamp))
                .map_err(PayloadBuilderError::other)?
        } else {
            Default::default()
        };

        let block_env_attributes = OpNextBlockEnvAttributes {
            timestamp,
            suggested_fee_recipient: config.attributes.suggested_fee_recipient(),
            prev_randao: config.attributes.prev_randao(),
            gas_limit: config
                .attributes
                .gas_limit
                .unwrap_or(config.parent_header.gas_limit),
            parent_beacon_block_root: config
                .attributes
                .payload_attributes
                .parent_beacon_block_root,
            extra_data,
        };

        let evm_env = self
            .evm_config
            .next_evm_env(&config.parent_header, &block_env_attributes)
            .map_err(PayloadBuilderError::other)?;

        let ctx = OpPayloadBuilderCtx {
            evm_config: self.evm_config.clone(),
            da_config: self.config.da_config.clone(),
            gas_limit_config: self.config.gas_limit_config.clone(),
            chain_spec,
            config,
            evm_env,
            block_env_attributes,
            cancel,
            builder_signer: self.config.builder_signer,
            metrics: self.metrics.clone(),
            extra_ctx: Default::default(),
            max_gas_per_txn: self.config.max_gas_per_txn,
            address_gas_limiter: self.address_gas_limiter.clone(),
        };

        let builder = OpBuilder::new(best);

        self.address_gas_limiter.refresh(ctx.block_number());

        let state_provider = self.client.state_by_block_hash(ctx.parent().hash())?;
        let db = StateProviderDatabase::new(&state_provider);
        let metrics = ctx.metrics.clone();
        if ctx.attributes().no_tx_pool {
            let state = State::builder()
                .with_database(db)
                .with_bundle_update()
                .build();
            builder.build(state, &state_provider, ctx, self.builder_tx.clone())
        } else {
            // sequencer mode we can reuse cachedreads from previous runs
            let state = State::builder()
                .with_database(cached_reads.as_db_mut(db))
                .with_bundle_update()
                .build();
            builder.build(state, &state_provider, ctx, self.builder_tx.clone())
        }
        .map(|out| {
            let total_block_building_time = block_build_start_time.elapsed();
            metrics
                .total_block_built_duration
                .record(total_block_building_time);
            metrics
                .total_block_built_gauge
                .set(total_block_building_time);

            out.with_cached_reads(cached_reads)
        })
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
pub(super) struct OpBuilder<'a, Txs> {
    /// Yields the best transaction to include if transactions from the mempool are allowed.
    best: Box<dyn FnOnce(BestTransactionsAttributes) -> Txs + 'a>,
}

impl<'a, Txs> OpBuilder<'a, Txs> {
    fn new(best: impl FnOnce(BestTransactionsAttributes) -> Txs + Send + Sync + 'a) -> Self {
        Self {
            best: Box::new(best),
        }
    }
}

/// Holds the state after execution
#[derive(Debug)]
pub(super) struct ExecutedPayload {
    /// Tracked execution info
    pub info: ExecutionInfo,
}

impl<Txs: PayloadTxsBounds> OpBuilder<'_, Txs> {
    /// Executes the payload and returns the outcome.
    pub(crate) fn execute<BuilderTx>(
        self,
        state_provider: impl StateProvider,
        db: &mut State<impl Database>,
        ctx: &OpPayloadBuilderCtx,
        builder_tx: BuilderTx,
    ) -> Result<BuildOutcomeKind<ExecutedPayload>, PayloadBuilderError>
    where
        BuilderTx: BuilderTransactions,
    {
        let Self { best } = self;
        info!(target: "payload_builder", id=%ctx.payload_id(), parent_header = ?ctx.parent().hash(), parent_number = ctx.parent().number, "building new payload");

        // 1. apply pre-execution changes
        ctx.evm_config
            .builder_for_next_block(db, ctx.parent(), ctx.block_env_attributes.clone())
            .map_err(PayloadBuilderError::other)?
            .apply_pre_execution_changes()?;

        let sequencer_tx_start_time = Instant::now();

        // 3. execute sequencer transactions
        let mut info = ctx.execute_sequencer_transactions(db)?;

        let sequencer_tx_time = sequencer_tx_start_time.elapsed();
        ctx.metrics.sequencer_tx_duration.record(sequencer_tx_time);
        ctx.metrics.sequencer_tx_gauge.set(sequencer_tx_time);

        // 4. if mem pool transactions are requested we execute them

        // gas reserved for builder tx
        let builder_txs =
            match builder_tx.add_builder_txs(&state_provider, &mut info, ctx, db, true) {
                Ok(builder_txs) => builder_txs,
                Err(e) => {
                    error!(target: "payload_builder", "Error adding builder txs to block: {}", e);
                    vec![]
                }
            };

        let builder_tx_gas = builder_txs.iter().fold(0, |acc, tx| acc + tx.gas_used);

        let block_gas_limit = ctx.block_gas_limit().saturating_sub(builder_tx_gas);
        if block_gas_limit == 0 {
            error!(
                "Builder tx gas subtraction resulted in block gas limit to be 0. No transactions would be included"
            );
        }
        // Save some space in the block_da_limit for builder tx
        let builder_tx_da_size = builder_txs.iter().fold(0, |acc, tx| acc + tx.da_size);
        let block_da_limit = ctx
            .da_config
            .max_da_block_size()
            .map(|da_limit| {
                let da_limit = da_limit.saturating_sub(builder_tx_da_size);
                if da_limit == 0 {
                    error!("Builder tx da size subtraction caused max_da_block_size to be 0. No transaction would be included.");
                }
                da_limit
            });
        let block_da_footprint = info.da_footprint_scalar
        .map(|da_footprint_scalar| {
            let da_footprint_limit = ctx.block_gas_limit().saturating_sub(builder_tx_da_size.saturating_mul(da_footprint_scalar as u64));
            if da_footprint_limit == 0 {
                error!("Builder tx da size subtraction caused max_da_footprint to be 0. No transaction would be included.");
            }
            da_footprint_limit
        });

        if !ctx.attributes().no_tx_pool {
            let best_txs_start_time = Instant::now();
            let mut best_txs = best(ctx.best_transaction_attributes());
            let transaction_pool_fetch_time = best_txs_start_time.elapsed();
            ctx.metrics
                .transaction_pool_fetch_duration
                .record(transaction_pool_fetch_time);
            ctx.metrics
                .transaction_pool_fetch_gauge
                .set(transaction_pool_fetch_time);

            if ctx
                .execute_best_transactions(
                    &mut info,
                    db,
                    &mut best_txs,
                    block_gas_limit,
                    block_da_limit,
                    block_da_footprint,
                )?
                .is_some()
            {
                return Ok(BuildOutcomeKind::Cancelled);
            }
        }

        // Add builder tx to the block
        if let Err(e) = builder_tx.add_builder_txs(&state_provider, &mut info, ctx, db, false) {
            error!(target: "payload_builder", "Error adding builder txs to fallback block: {}", e);
        };

        let state_merge_start_time = Instant::now();

        // merge all transitions into bundle state, this would apply the withdrawal balance changes
        // and 4788 contract call
        db.merge_transitions(BundleRetention::Reverts);

        let state_transition_merge_time = state_merge_start_time.elapsed();
        ctx.metrics
            .state_transition_merge_duration
            .record(state_transition_merge_time);
        ctx.metrics
            .state_transition_merge_gauge
            .set(state_transition_merge_time);

        ctx.metrics
            .payload_num_tx
            .record(info.executed_transactions.len() as f64);
        ctx.metrics
            .payload_num_tx_gauge
            .set(info.executed_transactions.len() as f64);

        let payload = ExecutedPayload { info };

        ctx.metrics.block_built_success.increment(1);
        Ok(BuildOutcomeKind::Better { payload })
    }

    /// Builds the payload on top of the state.
    pub(super) fn build<BuilderTx>(
        self,
        state: impl Database,
        state_provider: impl StateProvider,
        ctx: OpPayloadBuilderCtx,
        builder_tx: BuilderTx,
    ) -> Result<BuildOutcomeKind<OpBuiltPayload>, PayloadBuilderError>
    where
        BuilderTx: BuilderTransactions,
    {
        let mut db = State::builder()
            .with_database(state)
            .with_bundle_update()
            .build();
        let ExecutedPayload { info } =
            match self.execute(&state_provider, &mut db, &ctx, builder_tx)? {
                BuildOutcomeKind::Better { payload } | BuildOutcomeKind::Freeze(payload) => payload,
                BuildOutcomeKind::Cancelled => return Ok(BuildOutcomeKind::Cancelled),
                BuildOutcomeKind::Aborted { fees } => {
                    return Ok(BuildOutcomeKind::Aborted { fees });
                }
            };

        let block_number = ctx.block_number();
        // OP doesn't support blobs/EIP-4844.
        // https://specs.optimism.io/protocol/exec-engine.html#ecotone-disable-blob-transactions
        // Need [Some] or [None] based on hardfork to match block hash.
        let (excess_blob_gas, blob_gas_used) = ctx.blob_fields(&info);

        let execution_outcome = ExecutionOutcome::new(
            db.take_bundle(),
            vec![info.receipts],
            block_number,
            Vec::new(),
        );
        let receipts_root = execution_outcome
            .generic_receipts_root_slow(block_number, |receipts| {
                calculate_receipt_root_no_memo_optimism(
                    receipts,
                    &ctx.chain_spec,
                    ctx.attributes().timestamp(),
                )
            })
            .expect("Number is in range");
        let logs_bloom = execution_outcome
            .block_logs_bloom(block_number)
            .expect("Number is in range");

        // calculate the state root
        let state_root_start_time = Instant::now();

        let hashed_state = state_provider.hashed_post_state(execution_outcome.state());
        let (state_root, trie_output) = {
            state_provider
                .state_root_with_updates(hashed_state.clone())
                .inspect_err(|err| {
                    warn!(target: "payload_builder",
                    parent_header=%ctx.parent().hash(),
                        %err,
                        "failed to calculate state root for payload"
                    );
                })?
        };

        let state_root_calculation_time = state_root_start_time.elapsed();
        ctx.metrics
            .state_root_calculation_duration
            .record(state_root_calculation_time);
        ctx.metrics
            .state_root_calculation_gauge
            .set(state_root_calculation_time);

        let (withdrawals_root, requests_hash) = if ctx.is_isthmus_active() {
            // withdrawals root field in block header is used for storage root of L2 predeploy
            // `l2tol1-message-passer`
            (
                Some(
                    isthmus::withdrawals_root(execution_outcome.state(), state_provider)
                        .map_err(PayloadBuilderError::other)?,
                ),
                Some(EMPTY_REQUESTS_HASH),
            )
        } else if ctx.is_canyon_active() {
            (Some(EMPTY_WITHDRAWALS), None)
        } else {
            (None, None)
        };

        // create the block header
        let transactions_root = proofs::calculate_transaction_root(&info.executed_transactions);

        let extra_data = ctx.extra_data()?;

        let header = Header {
            parent_hash: ctx.parent().hash(),
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            beneficiary: ctx.evm_env.block_env.beneficiary,
            state_root,
            transactions_root,
            receipts_root,
            withdrawals_root,
            logs_bloom,
            timestamp: ctx.attributes().payload_attributes.timestamp,
            mix_hash: ctx.attributes().payload_attributes.prev_randao,
            nonce: BEACON_NONCE.into(),
            base_fee_per_gas: Some(ctx.base_fee()),
            number: ctx.parent().number + 1,
            gas_limit: ctx.block_gas_limit(),
            difficulty: U256::ZERO,
            gas_used: info.cumulative_gas_used,
            extra_data,
            parent_beacon_block_root: ctx.attributes().payload_attributes.parent_beacon_block_root,
            blob_gas_used,
            excess_blob_gas,
            requests_hash,
        };

        // seal the block
        let block = alloy_consensus::Block::<OpTransactionSigned>::new(
            header,
            BlockBody {
                transactions: info.executed_transactions,
                ommers: vec![],
                withdrawals: ctx.withdrawals().cloned(),
            },
        );

        let sealed_block = Arc::new(block.seal_slow());
        info!(target: "payload_builder", id=%ctx.attributes().payload_id(), "sealed built block");

        // create the executed block data
        let executed = ExecutedBlock {
            recovered_block: Arc::new(
                RecoveredBlock::<alloy_consensus::Block<OpTransactionSigned>>::new_sealed(
                    sealed_block.as_ref().clone(),
                    info.executed_senders,
                ),
            ),
            execution_output: Arc::new(execution_outcome),
            hashed_state: Arc::new(hashed_state),
            trie_updates: Arc::new(trie_output),
        };

        let no_tx_pool = ctx.attributes().no_tx_pool;

        let payload = OpBuiltPayload::new(
            ctx.payload_id(),
            sealed_block,
            info.total_fees,
            Some(executed),
        );

        ctx.metrics
            .payload_byte_size
            .record(InMemorySize::size(payload.block()) as f64);
        ctx.metrics
            .payload_byte_size_gauge
            .set(InMemorySize::size(payload.block()) as f64);

        if no_tx_pool {
            // if `no_tx_pool` is set only transactions from the payload attributes will be included
            // in the payload. In other words, the payload is deterministic and we can
            // freeze it once we've successfully built it.
            Ok(BuildOutcomeKind::Freeze(payload))
        } else {
            Ok(BuildOutcomeKind::Better { payload })
        }
    }
}
