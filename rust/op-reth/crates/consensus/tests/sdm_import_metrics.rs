//! Exposition test for the `reth_op_sdm_*` import counters.
//!
//! Its own integration binary, holding exactly one test, because the metrics recorder is global:
//! every validated block in the process increments these counters and the assertions below read
//! back shared values. A second test here would race with it.

use alloy_consensus::{
    Block, BlockBody, Eip658Value, Header, Receipt, Sealable, TxEip7702, TxReceipt,
};
use alloy_primitives::{Address, Bloom, Bytes, Log, Signature, U256};
use metrics_exporter_prometheus::PrometheusBuilder;
use op_alloy_consensus::{OpTypedTransaction, SDMGasEntry, build_post_exec_tx};
use reth_consensus::FullConsensus;
use reth_execution_types::BlockExecutionResult;
use reth_node_metrics::recorder::{
    install_prometheus_recorder, try_install_prometheus_recorder_with_builder,
};
use reth_optimism_chainspec::{OP_MAINNET, OpChainSpec, OpChainSpecBuilder};
use reth_optimism_consensus::OpBeaconConsensus;
use reth_optimism_primitives::{OpBlock, OpPrimitives, OpReceipt, OpTransactionSigned};
use reth_primitives_traits::{RecoveredBlock, SealedBlock, proofs};
use std::sync::Arc;

const BLOCK_GAS: &str = "reth_op_sdm_block_gas_total";
const REFUND_ENTRIES: &str = "reth_op_sdm_refund_entries_total";
const REFUND_GAS: &str = "reth_op_sdm_refund_gas_total";

const NO_REFUNDS: &[u64] = &[];

fn lagoon_consensus() -> OpBeaconConsensus<OpChainSpec> {
    OpBeaconConsensus::new(Arc::new(
        OpChainSpecBuilder::default()
            .lagoon_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build(),
    ))
}

fn user_tx() -> OpTransactionSigned {
    let tx = TxEip7702 {
        chain_id: 1,
        gas_limit: 10,
        max_fee_per_gas: 0x28f000fff,
        max_priority_fee_per_gas: 0x28f000fff,
        to: Address::default(),
        value: U256::from(3_u64),
        input: Bytes::from(vec![1, 2]),
        ..Default::default()
    };
    let signature = Signature::new(U256::default(), U256::default(), true);
    OpTransactionSigned::new_unhashed(OpTypedTransaction::Eip7702(tx), signature)
}

fn post_exec_tx(block_number: u64, refunds: &[u64]) -> OpTransactionSigned {
    let entries = refunds
        .iter()
        .enumerate()
        .map(|(index, gas_refund)| SDMGasEntry { index: index as u64, gas_refund: *gas_refund })
        .collect();
    OpTransactionSigned::PostExec(build_post_exec_tx(block_number, entries).seal_slow())
}

fn receipt(kind: fn(Receipt<Log>) -> OpReceipt, cumulative_gas_used: u64) -> OpReceipt {
    kind(Receipt { status: Eip658Value::success(), cumulative_gas_used, logs: vec![] })
}

/// A block of one user tx that passes post-execution validation, with a trailing `0x7D` carrying
/// `refunds` when they are non-empty. `gas_used` is the header's canonical gas.
fn validated_block(
    number: u64,
    gas_used: u64,
    refunds: &[u64],
) -> (RecoveredBlock<OpBlock>, BlockExecutionResult<OpReceipt>) {
    let mut transactions = vec![user_tx()];
    let mut receipts = vec![receipt(OpReceipt::Eip7702, gas_used)];
    if !refunds.is_empty() {
        transactions.push(post_exec_tx(number, refunds));
        receipts.push(receipt(OpReceipt::PostExec, gas_used));
    }
    let receipts_with_bloom = receipts.iter().map(TxReceipt::with_bloom_ref).collect::<Vec<_>>();

    let header = Header {
        base_fee_per_gas: Some(1337),
        blob_gas_used: Some(0),
        gas_used,
        logs_bloom: receipts_with_bloom.iter().fold(Bloom::ZERO, |bloom, r| bloom | r.bloom_ref()),
        number,
        receipts_root: proofs::calculate_receipt_root(&receipts_with_bloom),
        timestamp: number,
        transactions_root: proofs::calculate_transaction_root(&transactions),
        withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
        ..Default::default()
    };
    let senders = vec![Address::default(); transactions.len()];
    let body = BlockBody { transactions, ommers: vec![], withdrawals: None };
    let block = RecoveredBlock::new_sealed(SealedBlock::seal_slow(Block { header, body }), senders);
    let result =
        BlockExecutionResult { blob_gas_used: 0, gas_used, receipts, ..Default::default() };

    (block, result)
}

/// Reads one unlabeled counter out of the exposition. `None` distinguishes a series that was never
/// registered from one reporting 0, which a substring match cannot.
fn counter(exposition: &str, name: &str) -> Option<f64> {
    exposition
        .lines()
        .find_map(|line| line.strip_prefix(name)?.strip_prefix(' ')?.trim().parse().ok())
}

/// Every block that passes post-execution validation adds its canonical gas; a block carrying a
/// `0x7D` also adds its refund entries and refund gas; a block that fails validation adds nothing.
///
/// One sequential test rather than parameterized cases, because they share this process's
/// recorder; as separate `#[test]` functions they would race.
#[test]
fn import_counters_follow_validated_blocks() {
    // Install before the first record so every increment below lands on this recorder.
    let recorder = try_install_prometheus_recorder_with_builder(PrometheusBuilder::new())
        .unwrap_or_else(|_| install_prometheus_recorder());
    let read = |name: &str| counter(&recorder.handle().render(), name);
    let consensus = lagoon_consensus();

    let (block, result) = validated_block(1, /* gas_used */ 21_000, NO_REFUNDS);
    <OpBeaconConsensus<OpChainSpec> as FullConsensus<OpPrimitives>>::validate_block_post_execution(
        &consensus, &block, &result, None, None,
    )
    .expect("block without post-exec tx validates");
    assert_eq!(read(BLOCK_GAS), Some(21_000.0), "canonical gas of a block without refunds");
    assert_eq!(read(REFUND_ENTRIES), Some(0.0), "no refund entries without a post-exec tx");
    assert_eq!(read(REFUND_GAS), Some(0.0), "no refund gas without a post-exec tx");

    let (block, result) = validated_block(2, /* gas_used */ 100_000, &[5, 7]);
    <OpBeaconConsensus<OpChainSpec> as FullConsensus<OpPrimitives>>::validate_block_post_execution(
        &consensus, &block, &result, None, None,
    )
    .expect("block with post-exec tx validates");
    assert_eq!(read(BLOCK_GAS), Some(121_000.0), "canonical gas accumulates across blocks");
    assert_eq!(read(REFUND_ENTRIES), Some(2.0), "one entry per refunded tx");
    assert_eq!(read(REFUND_GAS), Some(12.0), "refund gas is the sum of the entries");

    let (block, mut result) = validated_block(3, /* gas_used */ 1, &[9]);
    result.blob_gas_used += 1;
    <OpBeaconConsensus<OpChainSpec> as FullConsensus<OpPrimitives>>::validate_block_post_execution(
        &consensus, &block, &result, None, None,
    )
    .expect_err("blob gas mismatch fails validation");
    assert_eq!(read(BLOCK_GAS), Some(121_000.0), "a rejected block adds no canonical gas");
    assert_eq!(read(REFUND_ENTRIES), Some(2.0), "a rejected block adds no refund entries");
    assert_eq!(read(REFUND_GAS), Some(12.0), "a rejected block adds no refund gas");
}
