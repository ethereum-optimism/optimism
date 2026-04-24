use std::time::Duration;

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
    // `self` is read only when the `metrics` feature is enabled.
    #[cfg_attr(not(feature = "metrics"), allow(clippy::unused_self, clippy::missing_const_for_fn))]
    pub(super) fn update_metrics(&self) {
        // no-op if disabled.
        #[cfg(feature = "metrics")]
        {
            let state_flags: [(&str, String); 2] = [
                ("active", self.is_active.to_string()),
                ("recovery", self.in_recovery_mode.to_string()),
            ];

            let gauge = metrics::gauge!(crate::Metrics::SEQUENCER_STATE, &state_flags);
            gauge.set(1);
        }
    }
}

// Parameters are only read inside `kona_macros::set!` expansions gated on the
// `metrics` feature; the helpers keep unified signatures across build modes.
#[inline]
#[cfg_attr(not(feature = "metrics"), allow(unused_variables, clippy::missing_const_for_fn))]
pub(super) fn update_attributes_build_duration_metrics(duration: Duration) {
    // Log the attributes build duration, if metrics are enabled.
    kona_macros::set!(gauge, crate::Metrics::SEQUENCER_ATTRIBUTES_BUILDER_DURATION, duration);
}

#[inline]
#[cfg_attr(not(feature = "metrics"), allow(unused_variables, clippy::missing_const_for_fn))]
pub(super) fn update_conductor_commitment_duration_metrics(duration: Duration) {
    kona_macros::set!(gauge, crate::Metrics::SEQUENCER_CONDUCTOR_COMMITMENT_DURATION, duration);
}

#[inline]
#[cfg_attr(not(feature = "metrics"), allow(unused_variables, clippy::missing_const_for_fn))]
pub(super) fn update_block_build_duration_metrics(duration: Duration) {
    kona_macros::set!(
        gauge,
        crate::Metrics::SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION,
        duration
    );
}

#[inline]
#[cfg_attr(not(feature = "metrics"), allow(unused_variables, clippy::missing_const_for_fn))]
pub(super) fn update_seal_duration_metrics(duration: Duration) {
    // Log the block building seal task duration, if metrics are enabled.
    kona_macros::set!(gauge, crate::Metrics::SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION, duration);
}

#[inline]
#[cfg_attr(not(feature = "metrics"), allow(unused_variables, clippy::missing_const_for_fn))]
pub(super) fn update_total_transactions_sequenced(transaction_count: u64) {
    #[cfg(feature = "metrics")]
    metrics::counter!(crate::Metrics::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED)
        .increment(transaction_count);
}
