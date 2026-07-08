//! Behavioral tests that a pool validator wrapper, installed via
//! `OpPoolBuilder::with_validator_wrapper`, gates transaction admission at ingress: a custom
//! filtering wrapper rejects matching transactions, while the default identity wrapper admits them.

use super::{build_components_with_pool, funded_signers, node_test_setup, signed_transfer_raw};
use alloy_consensus::Transaction as _;
use alloy_primitives::B256;
use op_alloy_network::Optimism;
use reth_e2e_test_utils::{node::NodeTestContext, transaction::TransactionTestContext};
use reth_node_api::FullNodeTypes;
use reth_node_builder::{
    EngineNodeLauncher, NodeBuilder,
    components::{BasicPayloadServiceBuilder, ComponentsBuilder},
    rpc::BasicEngineValidatorBuilder,
};
use reth_optimism_node::{
    OpEngineApiBuilder, OpNode,
    args::RollupArgs,
    node::{
        OpConsensusBuilder, OpEngineValidatorBuilder, OpExecutorBuilder, OpNetworkBuilder,
        OpNodeTypes, OpPayloadBuilder, OpPoolBuilder, OpValidatorWrapper,
    },
    txpool::OpTransactionValidator,
    utils::optimism_payload_attributes,
};
use reth_optimism_txpool::OpPooledTransaction;
use reth_primitives_traits::SealedBlock;
use reth_transaction_pool::{
    CoinbaseTipOrdering, TransactionOrigin, TransactionValidationOutcome, TransactionValidator,
    error::InvalidPoolTransactionError,
};

const GAS_LIMIT_REJECT_THRESHOLD: u64 = 250_000;

#[derive(Debug, Clone)]
struct GasLimitFilterValidator<V> {
    inner: V,
}

impl<V> TransactionValidator for GasLimitFilterValidator<V>
where
    V: TransactionValidator<Transaction = OpPooledTransaction>,
{
    type Transaction = V::Transaction;
    type Block = V::Block;

    async fn validate_transaction(
        &self,
        origin: TransactionOrigin,
        transaction: Self::Transaction,
    ) -> TransactionValidationOutcome<Self::Transaction> {
        let gas_limit = transaction.gas_limit();
        let filtered = gas_limit >= GAS_LIMIT_REJECT_THRESHOLD;
        let outcome = self.inner.validate_transaction(origin, transaction).await;

        // Only reject transactions the inner validator would otherwise have accepted, so we never
        // mask its checks.
        match outcome {
            TransactionValidationOutcome::Valid { transaction, .. } if filtered => {
                TransactionValidationOutcome::Invalid(
                    transaction.into_transaction(),
                    InvalidPoolTransactionError::MaxTxGasLimitExceeded(
                        gas_limit,
                        GAS_LIMIT_REJECT_THRESHOLD,
                    ),
                )
            }
            other => other,
        }
    }

    fn on_new_head_block(&self, new_tip_block: &SealedBlock<Self::Block>) {
        self.inner.on_new_head_block(new_tip_block);
    }
}

/// Installs a [`GasLimitFilterValidator`] around the standard [`OpTransactionValidator`].
#[derive(Debug, Default, Clone, Copy)]
struct FilteringWrapper;

impl<Provider, T, Evm> OpValidatorWrapper<Provider, T, Evm> for FilteringWrapper
where
    OpTransactionValidator<Provider, T, Evm>:
        TransactionValidator<Transaction = OpPooledTransaction>,
{
    type Validator = GasLimitFilterValidator<OpTransactionValidator<Provider, T, Evm>>;

    fn wrap(
        self,
        validator: OpTransactionValidator<Provider, T, Evm>,
        _provider: Provider,
    ) -> Self::Validator {
        GasLimitFilterValidator { inner: validator }
    }
}

/// The pool builder used by the filtering test: the OP default ordering plus the
/// [`FilteringWrapper`].
type FilteringPoolBuilder =
    OpPoolBuilder<OpPooledTransaction, CoinbaseTipOrdering<OpPooledTransaction>, FilteringWrapper>;

/// The components used by the filtering test: OP defaults except for the pool, which carries the
/// filtering validator wrapper.
type FilteringComponents<Node> = ComponentsBuilder<
    Node,
    FilteringPoolBuilder,
    BasicPayloadServiceBuilder<OpPayloadBuilder>,
    OpNetworkBuilder,
    OpExecutorBuilder,
    OpConsensusBuilder,
>;

/// Builds node components with the filtering validator wrapper installed via the `OpPoolBuilder`
/// seam. Every other component, including the ordering, is the OP default.
///
/// The default-wrapper case reuses the shared [`super::build_components_with_pool`]; this concrete
/// helper exists because the deep payload/EVM/RPC bound chain a generic-over-wrapper builder would
/// have to restate is intractable, so the filtering wrapper is wired in concretely instead.
fn build_components_with_filter<Node>() -> FilteringComponents<Node>
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
{
    let RollupArgs { disable_txpool_gossip, compute_pending_block, discovery_v4, .. } =
        RollupArgs::default();
    ComponentsBuilder::default()
        .node_types::<Node>()
        .pool(OpPoolBuilder::default().with_validator_wrapper(FilteringWrapper))
        .executor(OpExecutorBuilder::default())
        .payload(BasicPayloadServiceBuilder::new(OpPayloadBuilder::new(compute_pending_block)))
        .network(OpNetworkBuilder::new(disable_txpool_gossip, !discovery_v4))
        .consensus(OpConsensusBuilder::default())
}

#[tokio::test]
async fn custom_validator_wrapper_rejects_filtered_txs() {
    let (config, db, runtime, chain_id) = node_test_setup();
    let node_handle = NodeBuilder::new(config.clone())
        .with_database(db)
        .with_types::<OpNode>()
        .with_components(build_components_with_filter())
        // `OpNode::add_ons()` is pre-typed to the default-wrapper pool; the add-ons builder is
        // generic over the node's components, so build it with the OP-default generics so it
        // specializes to this node's pool.
        .with_add_ons(
            OpNode::new(Default::default()).add_ons_builder::<Optimism>().build::<
                _,
                OpEngineValidatorBuilder,
                OpEngineApiBuilder<OpEngineValidatorBuilder>,
                BasicEngineValidatorBuilder<OpEngineValidatorBuilder>,
            >(),
        )
        .launch_with_fn(|builder| {
            let launcher =
                EngineNodeLauncher::new(runtime.clone(), builder.config.datadir(), Default::default());
            builder.launch_with(launcher)
        })
        .await
        .expect("node with a custom validator wrapper should launch");

    let mut ctx =
        NodeTestContext::new(node_handle.node, optimism_payload_attributes).await.unwrap();

    // Signers: L1-info, the above-threshold tx, and the below-threshold tx.
    let signers = funded_signers(3, chain_id);

    let l1_info_raw =
        TransactionTestContext::optimism_l1_block_info_tx(chain_id, signers[0].clone(), 0).await;
    ctx.rpc.inject_tx(l1_info_raw).await.expect("l1 info tx should be accepted");

    let (above_raw, above_hash) = signed_transfer_raw(
        &signers[1],
        chain_id,
        0,
        GAS_LIMIT_REJECT_THRESHOLD,
        1_000_000_000,
        60_000_000_000,
    );
    let (below_raw, below_hash) = signed_transfer_raw(
        &signers[2],
        chain_id,
        0,
        GAS_LIMIT_REJECT_THRESHOLD - 1,
        1_000_000_000,
        60_000_000_000,
    );

    assert!(
        ctx.rpc.inject_tx(above_raw).await.is_err(),
        "above-threshold tx must be rejected at ingress, but injection succeeded",
    );
    ctx.rpc.inject_tx(below_raw).await.expect("below-threshold tx should be accepted");

    let payload = ctx.new_payload().await.unwrap();
    let block = payload.block();
    let block_hashes: Vec<B256> =
        block.body().transactions.iter().map(|tx| B256::from(*tx.tx_hash())).collect();

    assert!(!block_hashes.contains(&above_hash), "above-threshold tx should not be included");
    assert!(block_hashes.contains(&below_hash), "below-threshold tx should be included");
}

#[tokio::test]
async fn default_validator_admits_all_txs() {
    let (config, db, runtime, chain_id) = node_test_setup();
    let node_handle = NodeBuilder::new(config.clone())
        .with_database(db)
        .with_types::<OpNode>()
        .with_components(build_components_with_pool(OpPoolBuilder::default()))
        .with_add_ons(
            OpNode::new(Default::default()).add_ons_builder::<Optimism>().build::<
                _,
                OpEngineValidatorBuilder,
                OpEngineApiBuilder<OpEngineValidatorBuilder>,
                BasicEngineValidatorBuilder<OpEngineValidatorBuilder>,
            >(),
        )
        .launch_with_fn(|builder| {
            let launcher =
                EngineNodeLauncher::new(runtime.clone(), builder.config.datadir(), Default::default());
            builder.launch_with(launcher)
        })
        .await
        .expect("node with the default validator wrapper should launch");

    let mut ctx =
        NodeTestContext::new(node_handle.node, optimism_payload_attributes).await.unwrap();

    // Signers: L1-info, the above-threshold tx, and the below-threshold tx.
    let signers = funded_signers(3, chain_id);

    let l1_info_raw =
        TransactionTestContext::optimism_l1_block_info_tx(chain_id, signers[0].clone(), 0).await;
    ctx.rpc.inject_tx(l1_info_raw).await.expect("l1 info tx should be accepted");

    let (above_raw, above_hash) = signed_transfer_raw(
        &signers[1],
        chain_id,
        0,
        GAS_LIMIT_REJECT_THRESHOLD,
        1_000_000_000,
        60_000_000_000,
    );
    let (below_raw, below_hash) = signed_transfer_raw(
        &signers[2],
        chain_id,
        0,
        GAS_LIMIT_REJECT_THRESHOLD - 1,
        1_000_000_000,
        60_000_000_000,
    );

    ctx.rpc.inject_tx(above_raw).await.expect("above-threshold tx should be accepted by default");
    ctx.rpc.inject_tx(below_raw).await.expect("below-threshold tx should be accepted");

    let payload = ctx.new_payload().await.unwrap();
    let block = payload.block();
    let block_hashes: Vec<B256> =
        block.body().transactions.iter().map(|tx| B256::from(*tx.tx_hash())).collect();

    assert!(block_hashes.contains(&above_hash), "above-threshold tx should be included");
    assert!(block_hashes.contains(&below_hash), "below-threshold tx should be included");
}
