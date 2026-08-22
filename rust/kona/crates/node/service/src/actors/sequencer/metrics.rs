use std::{sync::Arc, time::Duration};

use crate::{
    Conductor, OriginSelector, SequencerActor, SequencerEngineClient, UnsafePayloadGossipClient,
};
use kona_derive::AttributesBuilder;

/// `SequencerActor` metrics-related method implementations.
impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    SequencerActor<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Updates the metrics for the sequencer actor.
    pub(super) fn update_metrics(&self) {
        // no-op if disabled.
        #[cfg(feature = "metrics")]
        {
            let state_flags: [(&str, String); 3] = [
                ("active", self.is_active.to_string()),
                ("recovery", self.in_recovery_mode.to_string()),
                (crate::Metrics::CHAIN_ID_LABEL, self.chain_id_label.to_string()),
            ];

            let gauge = metrics::gauge!(crate::Metrics::SEQUENCER_STATE, &state_flags);
            gauge.set(1);
        }
    }
}

#[inline]
pub(super) fn update_attributes_build_duration_metrics(
    duration: Duration,
    _chain_id_label: &Arc<str>,
) {
    // Log the attributes build duration, if metrics are enabled.
    kona_macros::set!(
        gauge,
        crate::Metrics::SEQUENCER_ATTRIBUTES_BUILDER_DURATION,
        duration,
        crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
    );
}

#[inline]
pub(super) fn update_conductor_commitment_duration_metrics(
    duration: Duration,
    _chain_id_label: &Arc<str>,
) {
    kona_macros::set!(
        gauge,
        crate::Metrics::SEQUENCER_CONDUCTOR_COMMITMENT_DURATION,
        duration,
        crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
    );
}

#[inline]
pub(super) fn update_block_build_duration_metrics(duration: Duration, _chain_id_label: &Arc<str>) {
    kona_macros::set!(
        gauge,
        crate::Metrics::SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION,
        duration,
        crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
    );
}

#[inline]
pub(super) fn update_seal_duration_metrics(duration: Duration, _chain_id_label: &Arc<str>) {
    // Log the block building seal task duration, if metrics are enabled.
    kona_macros::set!(
        gauge,
        crate::Metrics::SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION,
        duration,
        crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
    );
}

#[inline]
pub(super) fn update_total_transactions_sequenced(
    transaction_count: u64,
    _chain_id_label: &Arc<str>,
) {
    #[cfg(feature = "metrics")]
    metrics::counter!(
        crate::Metrics::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED,
        crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
    )
    .increment(transaction_count);
}
