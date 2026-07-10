//! Behavioral tests that the pool's [`TransactionOrdering`] actually drives block composition.

use super::{build_components_with_pool, funded_signers, node_test_setup, signed_transfer_raw};
use alloy_consensus::Transaction as _;
use alloy_primitives::B256;
use op_alloy_network::Optimism;
use reth_e2e_test_utils::{node::NodeTestContext, transaction::TransactionTestContext};
use reth_node_builder::{EngineNodeLauncher, NodeBuilder, rpc::BasicEngineValidatorBuilder};
use reth_optimism_node::{
    OpEngineApiBuilder, OpNode,
    node::{OpEngineValidatorBuilder, OpPoolBuilder},
    utils::optimism_payload_attributes,
};
use reth_optimism_txpool::OpPooledTransaction;
use reth_provider::providers::BlockchainProvider;
use reth_transaction_pool::{Priority, TransactionOrdering};
use std::collections::HashMap;

#[derive(Debug, Clone, Copy, Default)]
struct GasDescendingOrdering;

impl TransactionOrdering for GasDescendingOrdering {
    type PriorityValue = u128;
    type Transaction = OpPooledTransaction;

    fn priority(
        &self,
        transaction: &Self::Transaction,
        _base_fee: u64,
    ) -> Priority<Self::PriorityValue> {
        Priority::Value(u128::from(transaction.gas_limit()))
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct TxSpec {
    gas_limit: u64,
    max_priority_fee_per_gas: u128,
    max_fee_per_gas: u128,
}

const HIGH_TIP_LOW_GAS: TxSpec = TxSpec {
    gas_limit: 21_000,
    max_priority_fee_per_gas: 50_000_000_000,
    max_fee_per_gas: 60_000_000_000,
};
const LOW_TIP_HIGH_GAS: TxSpec = TxSpec {
    gas_limit: 50_000,
    max_priority_fee_per_gas: 1_000_000_000,
    max_fee_per_gas: 60_000_000_000,
};

async fn run_ordering_test<Ordering>(
    pool: OpPoolBuilder<OpPooledTransaction, Ordering>,
    txs: Vec<TxSpec>,
    expected_order: Vec<TxSpec>,
) where
    Ordering: TransactionOrdering<Transaction = OpPooledTransaction>,
{
    let (config, db, runtime, chain_id) = node_test_setup();
    let node_handle = NodeBuilder::new(config.clone())
        .with_database(db)
        .with_types_and_provider::<OpNode, BlockchainProvider<_>>()
        .with_components(build_components_with_pool(pool))
        // `OpNode::add_ons()` is pre-typed to the default-ordering pool; the add-ons builder is
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
        .expect("node with the configured pool ordering should launch");

    let mut ctx =
        NodeTestContext::new(node_handle.node, optimism_payload_attributes).await.unwrap();

    // One funded signer for the L1-info system tx, plus one per competing transaction, so their
    // nonces and tips are wholly independent of one another.
    let signers = funded_signers(txs.len() + 1, chain_id);
    let l1_info_signer = &signers[0];

    // The L1-info system transaction must be present; inject it first.
    let l1_info_raw =
        TransactionTestContext::optimism_l1_block_info_tx(chain_id, l1_info_signer.clone(), 0)
            .await;
    ctx.rpc.inject_tx(l1_info_raw).await.expect("l1 info tx should be accepted");

    // Inject each competing transaction from its own signer (nonce 0), mapping its hash to its
    // spec. Hashes are unique: each signer is distinct and submits a single nonce-0 transaction.
    let mut spec_by_hash: HashMap<B256, TxSpec> = HashMap::with_capacity(txs.len());
    for (spec, signer) in txs.iter().zip(signers.iter().skip(1)) {
        let (raw, hash) = signed_transfer_raw(
            signer,
            chain_id,
            0,
            spec.gas_limit,
            spec.max_priority_fee_per_gas,
            spec.max_fee_per_gas,
        );
        ctx.rpc.inject_tx(raw).await.expect("competing tx should be accepted");
        spec_by_hash.insert(hash, *spec);
    }

    // Build one block draining the pool. `advance` is unsuitable: it asserts the injected tx lands
    // first, which need not hold under a custom ordering — exactly the behavior under test.
    let payload = ctx.new_payload().await.unwrap();
    let block = payload.block();
    let block_hashes: Vec<B256> =
        block.body().transactions.iter().map(|tx| B256::from(*tx.tx_hash())).collect();

    // Project the block's transaction order onto the specs of the competing transactions, dropping
    // anything else (the L1-info tx), then compare to the expected order.
    let actual_order: Vec<TxSpec> =
        block_hashes.iter().filter_map(|hash| spec_by_hash.get(hash).copied()).collect();

    assert_eq!(
        actual_order, expected_order,
        "competing txs landed in order {actual_order:?}, expected {expected_order:?}",
    );
}

/// The OP default ordering ([`CoinbaseTipOrdering`]) sequences higher tips first.
#[tokio::test]
async fn default_tip_ordering_drives_block_composition() {
    let txs = vec![LOW_TIP_HIGH_GAS, HIGH_TIP_LOW_GAS];
    let expected_order = vec![HIGH_TIP_LOW_GAS, LOW_TIP_HIGH_GAS];
    run_ordering_test(OpPoolBuilder::default(), txs, expected_order).await;
}

/// The custom [`GasDescendingOrdering`], installed via the seam, sequences higher gas limits first
/// — the opposite of the tip ordering for the same transactions.
#[tokio::test]
async fn custom_gas_ordering_drives_block_composition() {
    let txs = vec![HIGH_TIP_LOW_GAS, LOW_TIP_HIGH_GAS];
    let expected_order = vec![LOW_TIP_HIGH_GAS, HIGH_TIP_LOW_GAS];
    run_ordering_test(
        OpPoolBuilder::default().with_ordering(GasDescendingOrdering),
        txs,
        expected_order,
    )
    .await;
}
