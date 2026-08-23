//! Canonical head metrics mirroring the `chain/head/*`
//!
//! The block height counterparts (`chain_head_block`, `chain_head_safe`, `chain_head_finalized`)
//! already exist upstream as `reth_blockchain_tree_canonical_chain_height`,
//! `reth_blockchain_tree_safe_block_height` and `reth_blockchain_tree_finalized_block_height`.

use alloy_consensus::BlockHeader;
use futures_util::StreamExt;
use reth_chain_state::CanonStateNotificationStream;
use reth_metrics::{
    Metrics,
    metrics::{Gauge, Histogram},
};
use reth_primitives_traits::NodePrimitives;

/// Gas and fee attributes of the canonical head.
#[derive(Metrics)]
#[metrics(scope = "chain.head")]
pub struct OpChainHeadMetrics {
    /// The base fee per gas of the canonical head.
    pub basefee: Gauge,
    /// The gas used by the canonical head.
    pub gas_used: Gauge,
    /// The blob gas used by the canonical head.
    pub blob_gas_used: Gauge,
    /// Histogram of the gas used by canonical heads.
    pub gas_used_hist: Histogram,
    /// Histogram of the blob gas used by canonical heads.
    pub blob_gas_used_hist: Histogram,
}

impl OpChainHeadMetrics {
    /// Records the gas and fee attributes of a new canonical head.
    ///
    /// Attributes the header does not carry are left unrecorded rather than reported as zero, so a
    /// pre-1559 chain does not publish a 0 base fee and a pre-4844 chain does not skew the blob gas
    /// histogram with zeroes. This matches op-geth, whose `TryUpdate` helpers skip absent values.
    pub fn record(&self, header: &impl BlockHeader) {
        if let Some(basefee) = header.base_fee_per_gas() {
            self.basefee.set(basefee as f64);
        }

        self.gas_used.set(header.gas_used() as f64);
        self.gas_used_hist.record(header.gas_used() as f64);

        if let Some(blob_gas_used) = header.blob_gas_used() {
            self.blob_gas_used.set(blob_gas_used as f64);
            self.blob_gas_used_hist.record(blob_gas_used as f64);
        }
    }
}

/// Records [`OpChainHeadMetrics`] for every canonical chain update until `stream` ends.
///
/// Reorgs are covered as well as plain commits: both carry the new canonical tip. A notification
/// that commits no new block (a pure revert) is skipped, since it has no new head to describe.
pub async fn maintain_chain_head_metrics<N>(mut stream: CanonStateNotificationStream<N>)
where
    N: NodePrimitives,
{
    let metrics = OpChainHeadMetrics::default();

    while let Some(notification) = stream.next().await {
        if let Some(tip) = notification.tip_checked() {
            metrics.record(tip.header());
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use metrics_util::debugging::{DebugValue, DebuggingRecorder};

    fn find<'a>(snapshot: &'a [(String, DebugValue)], name: &str) -> Option<&'a DebugValue> {
        snapshot.iter().find(|(key, _)| key == name).map(|(_, value)| value)
    }

    #[test]
    fn records_canonical_head_attributes() {
        // A local recorder keeps this off any process-wide recorder another test installs. Note
        // that `Snapshotter::snapshot` drains what it reads, so each snapshot below contains only
        // what was recorded since the previous one.
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        let snapshot = || -> Vec<(String, DebugValue)> {
            snapshotter
                .snapshot()
                .into_vec()
                .into_iter()
                .map(|(key, _unit, _desc, value)| (key.key().name().to_string(), value))
                .collect()
        };

        metrics::with_local_recorder(&recorder, || {
            // `Default` caches its handles in a static `OnceLock`, so it would hand back handles
            // bound to whichever recorder initialized it first. Register fresh handles instead.
            let metrics = OpChainHeadMetrics::new_with_labels(Vec::<metrics::Label>::new());

            // A pre-Cancun head carries no blob gas, so nothing blob related may be observed.
            metrics.record(&Header {
                base_fee_per_gas: Some(7),
                gas_used: 21_000,
                blob_gas_used: None,
                ..Default::default()
            });

            let without_blob_gas = snapshot();
            assert_eq!(
                find(&without_blob_gas, "chain.head.basefee"),
                Some(&DebugValue::Gauge(7.0.into()))
            );
            assert_eq!(
                find(&without_blob_gas, "chain.head.gas_used"),
                Some(&DebugValue::Gauge(21_000.0.into()))
            );
            assert_eq!(
                find(&without_blob_gas, "chain.head.gas_used_hist"),
                Some(&DebugValue::Histogram(vec![21_000.0.into()]))
            );
            assert_eq!(
                find(&without_blob_gas, "chain.head.blob_gas_used_hist"),
                Some(&DebugValue::Histogram(vec![])),
                "expected no blob gas observation for a pre-Cancun head"
            );

            metrics.record(&Header {
                base_fee_per_gas: Some(9),
                gas_used: 42_000,
                blob_gas_used: Some(131_072),
                ..Default::default()
            });

            let with_blob_gas = snapshot();
            assert_eq!(
                find(&with_blob_gas, "chain.head.basefee"),
                Some(&DebugValue::Gauge(9.0.into()))
            );
            assert_eq!(
                find(&with_blob_gas, "chain.head.gas_used"),
                Some(&DebugValue::Gauge(42_000.0.into()))
            );
            assert_eq!(
                find(&with_blob_gas, "chain.head.blob_gas_used"),
                Some(&DebugValue::Gauge(131_072.0.into()))
            );
            assert_eq!(
                find(&with_blob_gas, "chain.head.gas_used_hist"),
                Some(&DebugValue::Histogram(vec![42_000.0.into()]))
            );
            assert_eq!(
                find(&with_blob_gas, "chain.head.blob_gas_used_hist"),
                Some(&DebugValue::Histogram(vec![131_072.0.into()]))
            );
        });
    }
}
