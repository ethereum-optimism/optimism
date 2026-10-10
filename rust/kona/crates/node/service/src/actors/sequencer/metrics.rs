use std::time::Duration;

#[inline]
pub(super) fn update_attributes_build_duration_metrics(duration: Duration) {
    // Log the attributes build duration, if metrics are enabled.
    metrics::gauge!(crate::Metrics::SEQUENCER_ATTRIBUTES_BUILDER_DURATION).set(duration);
}

#[inline]
pub(super) fn update_conductor_commitment_duration_metrics(duration: Duration) {
    metrics::gauge!(crate::Metrics::SEQUENCER_CONDUCTOR_COMMITMENT_DURATION).set(duration);
}

#[inline]
pub(super) fn update_block_build_duration_metrics(duration: Duration) {
    metrics::gauge!(crate::Metrics::SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION).set(duration);
}

#[inline]
pub(super) fn update_seal_duration_metrics(duration: Duration) {
    // Log the block building seal task duration, if metrics are enabled.
    metrics::gauge!(crate::Metrics::SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION).set(duration);
}

#[inline]
pub(super) fn update_total_transactions_sequenced(transaction_count: u64) {
    metrics::counter!(crate::Metrics::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED)
        .increment(transaction_count);
}
