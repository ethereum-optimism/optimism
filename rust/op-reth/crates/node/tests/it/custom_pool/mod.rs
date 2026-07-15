//! Tests for the `OpPoolBuilder` extension seams.

mod ordering;
mod validator;

use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_network::{TxSignerSync, eip2718::Encodable2718};
use alloy_primitives::{Address, B256, Bytes, Signature, TxKind};
use alloy_signer_local::PrivateKeySigner;
use reth_db::{
    DatabaseEnv,
    test_utils::{TempDatabase, create_test_rw_db_with_path},
};
use reth_e2e_test_utils::wallet::Wallet;
use reth_node_api::FullNodeTypes;
use reth_node_builder::{
    NodeConfig,
    components::{BasicPayloadServiceBuilder, ComponentsBuilder},
};
use reth_node_core::args::DatadirArgs;
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_node::{
    args::RollupArgs,
    node::{
        OpConsensusBuilder, OpExecutorBuilder, OpNetworkBuilder, OpNodeTypes, OpPayloadBuilder,
        OpPoolBuilder,
    },
};
use reth_optimism_txpool::OpPooledTransaction;
use reth_tasks::Runtime;
use reth_transaction_pool::TransactionOrdering;
use std::sync::Arc;

fn node_test_setup() -> (NodeConfig<OpChainSpec>, Arc<TempDatabase<DatabaseEnv>>, Runtime, u64) {
    reth_tracing::init_test_tracing();

    let genesis = serde_json::from_str(include_str!("../../assets/genesis.json")).unwrap();
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );
    let chain_id: u64 = chain_spec.chain().into();

    let mut config =
        NodeConfig::new(chain_spec).with_unused_ports().with_datadir_args(DatadirArgs {
            datadir: reth_db::test_utils::tempdir_path().into(),
            ..Default::default()
        });
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    let db = create_test_rw_db_with_path(
        config
            .datadir
            .datadir
            .unwrap_or_chain_default(config.chain.chain(), config.datadir.clone())
            .db(),
    );

    (config, db, Runtime::test(), chain_id)
}

/// Builds node components around the given pool builder. Every component other than the pool is the
/// OP default; the pool builder carries whatever ordering (the OP default or a custom one) the
/// caller selected, so a single launch path can be reused across orderings of different concrete
/// types.
fn build_components_with_pool<Node, Ordering>(
    pool: OpPoolBuilder<OpPooledTransaction, Ordering>,
) -> ComponentsBuilder<
    Node,
    OpPoolBuilder<OpPooledTransaction, Ordering>,
    BasicPayloadServiceBuilder<OpPayloadBuilder>,
    OpNetworkBuilder,
    OpExecutorBuilder,
    OpConsensusBuilder,
>
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
    Ordering: TransactionOrdering<Transaction = OpPooledTransaction>,
{
    let RollupArgs { disable_txpool_gossip, compute_pending_block, discovery_v4, .. } =
        RollupArgs::default();
    ComponentsBuilder::default()
        .node_types::<Node>()
        .pool(pool)
        .executor(OpExecutorBuilder::default())
        .payload(BasicPayloadServiceBuilder::new(OpPayloadBuilder::new(compute_pending_block)))
        .network(OpNetworkBuilder::new(disable_txpool_gossip, !discovery_v4))
        .consensus(OpConsensusBuilder::default())
}

/// Signs an EIP-1559 transfer with the given parameters and returns its 2718-encoded raw bytes
/// alongside the resulting transaction hash.
fn signed_transfer_raw(
    signer: &impl TxSignerSync<Signature>,
    chain_id: u64,
    nonce: u64,
    gas_limit: u64,
    max_priority_fee_per_gas: u128,
    max_fee_per_gas: u128,
) -> (Bytes, B256) {
    let mut tx = TxEip1559 {
        chain_id,
        nonce,
        gas_limit,
        max_fee_per_gas,
        max_priority_fee_per_gas,
        to: TxKind::Call(Address::random()),
        value: 0.try_into().unwrap(),
        ..Default::default()
    };
    let signature = signer.sign_transaction_sync(&mut tx).unwrap();
    let signed = tx.into_signed(signature);
    let hash = *signed.hash();
    let envelope = op_alloy_consensus::OpPooledTransaction::Eip1559(signed);
    (envelope.encoded_2718().into(), hash)
}

/// Returns `n` distinct funded signers from the standard test mnemonic (all present in
/// `genesis.json`).
fn funded_signers(n: usize, chain_id: u64) -> Vec<PrivateKeySigner> {
    Wallet::new(n).with_chain_id(chain_id).wallet_gen()
}
