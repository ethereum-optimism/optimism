use std::time::Duration;

/// Updates the metrics for the provided sequencer control-plane state.
pub(super) fn update_state_metrics(is_active: bool, in_recovery_mode: bool) {
    // no-op if disabled.
    #[cfg(feature = "metrics")]
    {
        let state_flags: [(&str, String); 2] =
            [("active", is_active.to_string()), ("recovery", in_recovery_mode.to_string())];

        let gauge = metrics::gauge!(crate::Metrics::SEQUENCER_STATE, &state_flags);
        gauge.set(1);
    }
}

#[inline]
pub(super) fn update_attributes_build_duration_metrics(duration: Duration) {
    // Log the attributes build duration, if metrics are enabled.
    kona_macros::set!(gauge, crate::Metrics::SEQUENCER_ATTRIBUTES_BUILDER_DURATION, duration);
}

#[inline]
pub(super) fn update_conductor_commitment_duration_metrics(duration: Duration) {
    kona_macros::set!(gauge, crate::Metrics::SEQUENCER_CONDUCTOR_COMMITMENT_DURATION, duration);
}

#[inline]
pub(super) fn update_block_build_duration_metrics(duration: Duration) {
    kona_macros::set!(
        gauge,
        crate::Metrics::SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION,
        duration
    );
}

#[inline]
pub(super) fn update_seal_duration_metrics(duration: Duration) {
    // Log the block building seal task duration, if metrics are enabled.
    kona_macros::set!(gauge, crate::Metrics::SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION, duration);
}

#[inline]
pub(super) fn update_total_transactions_sequenced(transaction_count: u64) {
    #[cfg(feature = "metrics")]
    metrics::counter!(crate::Metrics::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED)
        .increment(transaction_count);
}
