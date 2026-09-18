//! SDM refund observation counters for blocks this node executed and accepted, recorded from
//! post-execution validation. They follow imported blocks rather than locally built ones: blocks
//! later reorged out are included, and a block re-executed after an unwind is counted again. The
//! payload builder's `sdm_refund_gas_total` measures the same quantity for blocks this node
//! produced, a different population, so the two must not be summed.
//!
//! Handles are resolved on every record rather than cached, so a record before the recorder is
//! installed drops that one block instead of silencing the counters for the process.

use op_alloy_consensus::{OpTransaction, PostExecPayload};
use reth_metrics::{
    Metrics,
    metrics::{self, Counter},
};

const NO_LABELS: &[(&str, &str)] = &[];

/// One struct for all three counters on purpose: a block without refunds is a real observation, so
/// a node on a chain without SDM publishing zeros for the refund series is correct, and the alert
/// ratio needs `block_gas_total` from every node either way.
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmImportMetrics {
    /// Canonical (post-refund) gas used by validated blocks.
    block_gas_total: Counter,
    /// Warming-refund entries carried by validated blocks' post-exec transactions.
    refund_entries_total: Counter,
    /// Warming-refund gas carried by validated blocks' post-exec transactions.
    refund_gas_total: Counter,
}

/// Records one validated block: its canonical gas always, and its refunds when it carries a
/// post-exec transaction.
pub(crate) fn record_validated_block<T: OpTransaction>(block_gas_used: u64, transactions: &[T]) {
    let metrics = SdmImportMetrics::new_with_labels(NO_LABELS);
    metrics.block_gas_total.increment(block_gas_used);
    if let Some(payload) = post_exec_payload(transactions) {
        metrics.refund_entries_total.increment(payload.gas_refund_entries.len() as u64);
        metrics.refund_gas_total.increment(payload.total_gas_refund());
    }
}

/// The post-exec payload of a block that passed execution. The executor has already enforced that
/// a post-exec transaction is unique and final, so only the last transaction is inspected.
fn post_exec_payload<T: OpTransaction>(transactions: &[T]) -> Option<&PostExecPayload> {
    transactions.last().and_then(OpTransaction::as_post_exec).map(|tx| &tx.inner().payload)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpBeaconConsensus;
    use alloy_consensus::{
        Block, BlockBody, Eip658Value, Header, Receipt, Sealable, TxEip7702, TxReceipt,
    };
    use alloy_primitives::{Address, Bloom, Bytes, Log, Signature, U256};
    use metrics_exporter_prometheus::{PrometheusBuilder, PrometheusHandle, PrometheusRecorder};
    use metrics_util::layers::{Layer, Prefix, PrefixLayer};
    use op_alloy_consensus::{OpTypedTransaction, SDMGasEntry, TxDeposit, build_post_exec_tx};
    use reth_consensus::{ConsensusError, FullConsensus};
    use reth_execution_types::BlockExecutionResult;
    use reth_metrics::metrics::with_local_recorder;
    use reth_optimism_chainspec::{OP_MAINNET, OpChainSpec, OpChainSpecBuilder};
    use reth_optimism_primitives::{OpBlock, OpPrimitives, OpReceipt, OpTransactionSigned};
    use reth_primitives_traits::{RecoveredBlock, SealedBlock, proofs};
    use rstest::rstest;
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

    fn deposit_tx() -> OpTransactionSigned {
        OpTransactionSigned::Deposit(TxDeposit::default().seal_slow())
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

    struct ValidatedBlock {
        block: RecoveredBlock<OpBlock>,
        result: BlockExecutionResult<OpReceipt>,
    }

    /// A block of one user tx that passes post-execution validation, with a trailing `0x7D`
    /// carrying `refunds` when they are non-empty. `gas_used` is the header's canonical gas.
    fn validated_block(number: u64, gas_used: u64, refunds: &[u64]) -> ValidatedBlock {
        let mut transactions = vec![user_tx()];
        let mut receipts = vec![receipt(OpReceipt::Eip7702, gas_used)];
        if !refunds.is_empty() {
            transactions.push(post_exec_tx(number, refunds));
            receipts.push(receipt(OpReceipt::PostExec, gas_used));
        }
        let receipts_with_bloom =
            receipts.iter().map(TxReceipt::with_bloom_ref).collect::<Vec<_>>();

        let header = Header {
            base_fee_per_gas: Some(1337),
            blob_gas_used: Some(0),
            gas_used,
            logs_bloom: receipts_with_bloom
                .iter()
                .fold(Bloom::ZERO, |bloom, receipt| bloom | receipt.bloom_ref()),
            number,
            receipts_root: proofs::calculate_receipt_root(&receipts_with_bloom),
            timestamp: number,
            transactions_root: proofs::calculate_transaction_root(&transactions),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            ..Default::default()
        };
        let senders = vec![Address::default(); transactions.len()];
        let body = BlockBody { transactions, ommers: vec![], withdrawals: None };
        let block =
            RecoveredBlock::new_sealed(SealedBlock::seal_slow(Block { header, body }), senders);
        let result =
            BlockExecutionResult { blob_gas_used: 0, gas_used, receipts, ..Default::default() };

        ValidatedBlock { block, result }
    }

    /// A recorder private to one test, prefixed like the node's recorder stack so the rendered
    /// series names are the ones operators query.
    fn recorder() -> (Prefix<PrometheusRecorder>, PrometheusHandle) {
        let recorder = PrometheusBuilder::new().build_recorder();
        let handle = recorder.handle();
        (PrefixLayer::new("reth").layer(recorder), handle)
    }

    /// Runs post-execution validation against a recorder private to this test and returns the
    /// rendered exposition alongside the result.
    fn validate(validated: &ValidatedBlock) -> (Result<(), ConsensusError>, String) {
        let (recorder, handle) = recorder();
        let consensus = lagoon_consensus();
        let result = with_local_recorder(&recorder, || {
            FullConsensus::<OpPrimitives>::validate_block_post_execution(
                &consensus,
                &validated.block,
                &validated.result,
                None,
                None,
            )
        });
        (result, handle.render())
    }

    /// Reads one unlabeled counter out of the exposition. `None` distinguishes a series that was
    /// never registered from one reporting 0, which a substring match cannot.
    fn counter(exposition: &str, name: &str) -> Option<f64> {
        exposition
            .lines()
            .find_map(|line| line.strip_prefix(name)?.strip_prefix(' ')?.trim().parse().ok())
    }

    #[rstest]
    #[case::without_post_exec_tx(
        /* gas_used */ 21_000,
        /* refunds */ NO_REFUNDS,
        /* block_gas */ 21_000.0,
        /* entries */ 0.0,
        /* refund_gas */ 0.0
    )]
    #[case::with_post_exec_tx(
        /* gas_used */ 100_000,
        /* refunds */ &[5, 7],
        /* block_gas */ 100_000.0,
        /* entries */ 2.0,
        /* refund_gas */ 12.0
    )]
    fn validated_block_adds_its_gas_and_refunds(
        #[case] gas_used: u64,
        #[case] refunds: &[u64],
        #[case] block_gas: f64,
        #[case] entries: f64,
        #[case] refund_gas: f64,
    ) {
        let (result, exposition) = validate(&validated_block(1, gas_used, refunds));

        result.expect("block validates");
        assert_eq!(counter(&exposition, BLOCK_GAS), Some(block_gas), "{exposition}");
        assert_eq!(counter(&exposition, REFUND_ENTRIES), Some(entries), "{exposition}");
        assert_eq!(counter(&exposition, REFUND_GAS), Some(refund_gas), "{exposition}");
    }

    #[test]
    fn rejected_block_registers_nothing() {
        let mut validated = validated_block(1, /* gas_used */ 1, &[9]);
        validated.result.blob_gas_used += 1;

        let (result, exposition) = validate(&validated);

        result.expect_err("blob gas mismatch fails validation");
        for name in [BLOCK_GAS, REFUND_ENTRIES, REFUND_GAS] {
            assert_eq!(counter(&exposition, name), None, "{name} registered: {exposition}");
        }
    }

    #[rstest]
    #[case::trailing(vec![deposit_tx(), post_exec_tx(1, &[3])], Some(3))]
    #[case::absent(vec![deposit_tx()], None)]
    #[case::not_last(vec![post_exec_tx(1, &[3]), deposit_tx()], None)]
    fn post_exec_payload_reads_the_final_transaction(
        #[case] transactions: Vec<OpTransactionSigned>,
        #[case] expected_refund: Option<u64>,
    ) {
        let refund = post_exec_payload(&transactions).map(PostExecPayload::total_gas_refund);

        assert_eq!(refund, expected_refund);
    }
}
