//! The chain dimension for kona's metrics.
//!
//! kona resolves every metric through the process-wide `metrics` recorder, so a metric's chain is
//! not knowable at the emit site. In a process that runs several chains, every chain's series
//! collapse into one. This crate carries the chain in an ambient scope instead, and adds it as a
//! label in a wrapping [`Recorder`], so no emit site changes.
//!
//! * [`scoped`] and [`sync_scoped`] bind a chain to the work that emits its metrics.
//! * [`ChainLabelRecorder`] wraps the real recorder and reads that binding on every emit.
//!
//! # The scope does not propagate
//!
//! The binding is a [`tokio::task_local`], carried by the future [`scoped`] returns. It survives
//! `.await` and thread migration, but a new task inherits nothing, so every task that emits kona
//! metrics needs its own scope. `spawn_blocking` closures and library-spawned tasks likewise.
//!
//! [`ChainLabelRecorder::new`]'s `fallback` covers what no scope owns: the process's only chain
//! for a single-chain process, `None` for a multi-chain one, where no chain can be attributed.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]

use metrics::{Counter, Gauge, Histogram, Key, KeyName, Metadata, Recorder, SharedString, Unit};
use std::{borrow::Cow, future::Future, sync::Arc};

// So a dependent can hold a label without taking `metrics`, which most kona crates make optional.
pub use metrics::Label;

/// The label key that carries the chain id.
pub const CHAIN_ID_LABEL: &str = "chain_id";

tokio::task_local! {
    /// The chain bound to the current task, if any.
    static CHAIN: Label;
}

/// Builds the [`Label`] for a chain.
///
/// Build once per chain and clone it: the value is shared, so a clone is a refcount bump. The
/// recorder clones it on every emit.
pub fn chain_label(chain_id: u64) -> Label {
    Label::new(CHAIN_ID_LABEL, Arc::<str>::from(chain_id.to_string()))
}

/// Runs `fut` with `chain` labelling every metric emitted while it is polled.
///
/// The scope does not reach a task the future spawns. See the crate docs.
pub fn scoped<F: Future>(chain: Label, fut: F) -> impl Future<Output = F::Output> {
    CHAIN.scope(chain, fut)
}

/// [`scoped`] for synchronous work, such as the `Metrics::init` calls at start-up.
///
/// Needs no runtime, so it is usable before one is built.
pub fn sync_scoped<R>(chain: Label, f: impl FnOnce() -> R) -> R {
    CHAIN.sync_scope(chain, f)
}

/// The chain bound to the current task, if any.
pub fn current_chain() -> Option<Label> {
    CHAIN.try_with(Clone::clone).ok()
}

/// A [`Recorder`] that labels each metric with the chain of the scope that emitted it.
///
/// Every `metrics::counter!`/`gauge!`/`histogram!` resolves its key through the recorder on every
/// emit, so this sees all of them. An emit that already carries a [`CHAIN_ID_LABEL`] is passed
/// through untouched, for a component that serves more than one chain from one task.
#[derive(Debug, Clone)]
pub struct ChainLabelRecorder<R> {
    inner: R,
    fallback: Option<Label>,
}

impl<R> ChainLabelRecorder<R> {
    /// Wraps `inner`.
    ///
    /// `fallback` labels emits that arrive with no scope bound: the process's only chain when it
    /// serves one, `None` when it serves several.
    pub const fn new(inner: R, fallback: Option<Label>) -> Self {
        Self { inner, fallback }
    }

    /// `key`, with the chain label added when the emit did not carry one and a chain is known.
    fn labelled<'k>(&self, key: &'k Key) -> Cow<'k, Key> {
        // An explicit label wins.
        if key.labels().any(|label| label.key() == CHAIN_ID_LABEL) {
            return Cow::Borrowed(key);
        }

        let Some(chain) = current_chain().or_else(|| self.fallback.clone()) else {
            return Cow::Borrowed(key);
        };

        Cow::Owned(key.with_extra_labels(vec![chain]))
    }
}

impl<R: Recorder> Recorder for ChainLabelRecorder<R> {
    fn describe_counter(&self, key: KeyName, unit: Option<Unit>, description: SharedString) {
        self.inner.describe_counter(key, unit, description);
    }

    fn describe_gauge(&self, key: KeyName, unit: Option<Unit>, description: SharedString) {
        self.inner.describe_gauge(key, unit, description);
    }

    fn describe_histogram(&self, key: KeyName, unit: Option<Unit>, description: SharedString) {
        self.inner.describe_histogram(key, unit, description);
    }

    fn register_counter(&self, key: &Key, metadata: &Metadata<'_>) -> Counter {
        self.inner.register_counter(&self.labelled(key), metadata)
    }

    fn register_gauge(&self, key: &Key, metadata: &Metadata<'_>) -> Gauge {
        self.inner.register_gauge(&self.labelled(key), metadata)
    }

    fn register_histogram(&self, key: &Key, metadata: &Metadata<'_>) -> Histogram {
        self.inner.register_histogram(&self.labelled(key), metadata)
    }
}

#[cfg(test)]
mod tests {
    use super::{CHAIN_ID_LABEL, ChainLabelRecorder, chain_label, scoped, sync_scoped};
    use metrics_util::debugging::DebuggingRecorder;
    use std::collections::BTreeSet;

    /// The `chain_id` label of every series `f` emits, or `None` where a series has none.
    fn chains_of(
        fallback: Option<u64>,
        f: impl FnOnce(&ChainLabelRecorder<DebuggingRecorder>),
    ) -> BTreeSet<Option<String>> {
        let inner = DebuggingRecorder::new();
        let snapshotter = inner.snapshotter();
        let recorder = ChainLabelRecorder::new(inner, fallback.map(chain_label));

        f(&recorder);

        snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .map(|(ckey, ..)| {
                ckey.key()
                    .labels()
                    .find(|label| label.key() == CHAIN_ID_LABEL)
                    .map(|label| label.value().to_string())
            })
            .collect()
    }

    fn emit() {
        metrics::counter!("kona_test_metric").increment(1);
    }

    // Every test drives its runtime on one thread inside `with_local_recorder`, which binds a
    // *thread*-local recorder that a second worker could not see. A harness constraint, not one on
    // the scope, which rides the future.
    #[test]
    fn concurrent_chains_get_one_series_each() {
        let chains = chains_of(None, |recorder| {
            metrics::with_local_recorder(recorder, || {
                // The same metric with the same labels: without the scope, one series.
                let runtime = tokio::runtime::Builder::new_current_thread().build().unwrap();
                runtime.block_on(async {
                    tokio::join!(
                        scoped(chain_label(901), async {
                            tokio::task::yield_now().await;
                            emit();
                        }),
                        scoped(chain_label(902), async {
                            tokio::task::yield_now().await;
                            emit();
                        }),
                    );
                });
            });
        });

        assert_eq!(
            chains,
            BTreeSet::from([Some("901".to_string()), Some("902".to_string())]),
            "each chain must get its own series"
        );
    }

    #[test]
    fn a_scope_survives_an_await() {
        let chains = chains_of(None, |recorder| {
            metrics::with_local_recorder(recorder, || {
                let runtime = tokio::runtime::Builder::new_current_thread().build().unwrap();
                runtime.block_on(scoped(chain_label(10), async {
                    tokio::task::yield_now().await;
                    emit();
                }));
            });
        });

        assert_eq!(chains, BTreeSet::from([Some("10".to_string())]));
    }

    #[test]
    fn sync_work_can_be_scoped_without_a_runtime() {
        let chains = chains_of(None, |recorder| {
            metrics::with_local_recorder(recorder, || sync_scoped(chain_label(10), emit));
        });

        assert_eq!(chains, BTreeSet::from([Some("10".to_string())]));
    }

    #[test]
    fn an_unscoped_emit_takes_the_fallback() {
        let chains = chains_of(Some(10), |recorder| {
            metrics::with_local_recorder(recorder, emit);
        });

        assert_eq!(chains, BTreeSet::from([Some("10".to_string())]));
    }

    #[test]
    fn an_unscoped_emit_stays_unlabelled_without_a_fallback() {
        let chains = chains_of(None, |recorder| {
            metrics::with_local_recorder(recorder, emit);
        });

        assert_eq!(chains, BTreeSet::from([None]), "a process-wide emit must not claim a chain");
    }

    #[test]
    fn an_explicit_label_wins() {
        let chains = chains_of(Some(10), |recorder| {
            metrics::with_local_recorder(recorder, || {
                sync_scoped(chain_label(901), || {
                    metrics::counter!("kona_test_metric", CHAIN_ID_LABEL => "902").increment(1);
                });
            });
        });

        assert_eq!(
            chains,
            BTreeSet::from([Some("902".to_string())]),
            "a component that serves several chains labels its own emits"
        );
    }

    #[test]
    fn a_spawned_task_does_not_inherit_the_scope() {
        // Pins the limit the crate docs describe, so a tokio change is caught here.
        let chains = chains_of(None, |recorder| {
            metrics::with_local_recorder(recorder, || {
                let runtime = tokio::runtime::Builder::new_current_thread().build().unwrap();
                runtime.block_on(scoped(chain_label(10), async {
                    tokio::spawn(async { emit() }).await.unwrap();
                }));
            });
        });

        assert_eq!(chains, BTreeSet::from([None]));
    }
}
