//! One process-global metrics recorder, shared by the tests that need one.
//!
//! Both seams this crate scopes run on spawned tasks, which a thread-local recorder cannot reach.
//! `metrics::set_global_recorder` succeeds once per process, hence the single install.

use metrics_util::debugging::{DebuggingRecorder, Snapshotter};
use std::sync::LazyLock;

static SNAPSHOTTER: LazyLock<Snapshotter> = LazyLock::new(|| {
    let inner = DebuggingRecorder::new();
    let snapshotter = inner.snapshotter();

    // `None`, so an unscoped emit stays unlabelled and the tests can tell a scope from a fallback.
    metrics::set_global_recorder(kona_metrics::ChainLabelRecorder::new(inner, None))
        .expect("no other test installs a global recorder");

    snapshotter
});

/// The `chain_id` label of every series recorded so far for `metric`, or `None` where a series
/// carries none. Call it first in a test too, to install the recorder.
pub(crate) fn chains_of(metric: &str) -> Vec<Option<String>> {
    SNAPSHOTTER
        .snapshot()
        .into_vec()
        .into_iter()
        .filter(|(ckey, ..)| ckey.key().name() == metric)
        .map(|(ckey, ..)| {
            ckey.key()
                .labels()
                .find(|label| label.key() == kona_metrics::CHAIN_ID_LABEL)
                .map(|label| label.value().to_string())
        })
        .collect()
}
