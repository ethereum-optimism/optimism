//! SDM refund observation counters, recorded for every block that passes post-execution
//! validation. That point runs on every node for every canonically imported block, whether it
//! arrived through the engine API or pipeline sync, and not for RPC replays, so the counters
//! describe the chain rather than whichever node produced a block.

use op_alloy_consensus::PostExecPayload;
use reth_metrics::{
    Metrics,
    metrics::{self, Counter},
};

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
pub(crate) fn record_validated_block(block_gas_used: u64, post_exec: Option<&PostExecPayload>) {
    let metrics = SdmImportMetrics::default();
    metrics.block_gas_total.increment(block_gas_used);
    if let Some(payload) = post_exec {
        metrics.refund_entries_total.increment(payload.gas_refund_entries.len() as u64);
        metrics.refund_gas_total.increment(refund_gas(payload));
    }
}

fn refund_gas(payload: &PostExecPayload) -> u64 {
    payload
        .gas_refund_entries
        .iter()
        .fold(0u64, |total, entry| total.saturating_add(entry.gas_refund))
}

#[cfg(test)]
mod tests {
    use super::*;
    use op_alloy_consensus::SDMGasEntry;
    use rstest::rstest;

    fn payload_with_refunds(refunds: &[u64]) -> PostExecPayload {
        PostExecPayload {
            gas_refund_entries: refunds
                .iter()
                .enumerate()
                .map(|(index, gas_refund)| SDMGasEntry {
                    index: index as u64,
                    gas_refund: *gas_refund,
                })
                .collect(),
            ..Default::default()
        }
    }

    #[rstest]
    #[case::no_entries(&[], 0)]
    #[case::single_entry(&[5], 5)]
    #[case::sums_entries(&[5, 7], 12)]
    #[case::saturates(&[u64::MAX, 1], u64::MAX)]
    fn refund_gas_sums_entries(#[case] refunds: &[u64], #[case] expected: u64) {
        assert_eq!(refund_gas(&payload_with_refunds(refunds)), expected);
    }
}
