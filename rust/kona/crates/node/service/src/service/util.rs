//! Utilities for the rollup node service, internal to the crate.

/// Spawns a set of parallel actors in a [`JoinSet`](tokio::task::JoinSet), and cancels all actors
/// if any of them fail. The type of the error in the [`NodeActor`](crate::NodeActor)s is erased to
/// avoid having to specify a common error type between actors.
///
/// Actors are passed in as `Option<actor>`. Each actor's [`step`](crate::NodeActor::step) method is
/// called in a loop, with external cancellation via the provided
/// [`CancellationToken`](tokio_util::sync::CancellationToken).
///
/// This macro also handles OS shutdown signals (SIGTERM, SIGINT) and triggers graceful shutdown
/// when received.
macro_rules! spawn_and_wait {
    ($cancellation:expr, chain = $chain:expr, actors = [$($actor:expr),* $(,)?]) => {
        let mut task_handles = tokio::task::JoinSet::new();
        let chain = $chain;

        $(
            if let Some(mut actor) = $actor {
                let cancellation = $cancellation.clone();
                // A task the actor spawns for itself does not inherit this scope.
                task_handles.spawn(kona_metrics::scoped(chain.clone(), async move {
                    // This guard ensures that the cancellation token is cancelled when the actor
                    // task exits for any reason. This ensures peer actors observe shutdown on
                    // their next macro-level `select!`.
                    // Note the underscore prefix: this is to signal that we don't use the guard
                    // anywhere, but *the compiler shouldn't optimize it away*. Note that using a
                    // simple `_` would not work here because it gets optimized away in release
                    // mode.
                    let _guard = cancellation.clone().drop_guard();
                    loop {
                        tokio::select! {
                            biased;
                            _ = cancellation.cancelled() => return Ok(()),
                            result = actor.step() => {
                                result.map_err(|e| format!("{e:?}"))?;
                            }
                        }
                    }
                }));
            }
        )*

        // Create the shutdown signal future
        let shutdown = $crate::service::shutdown_signal();
        tokio::pin!(shutdown);

        loop {
            tokio::select! {
                _ = &mut shutdown => {
                    tracing::info!(target: "rollup_node", "Received shutdown signal, initiating graceful shutdown...");
                    $cancellation.cancel();
                    break;
                }
                result = task_handles.join_next() => {
                    match result {
                        Some(Ok(Ok(()))) => { /* Actor completed successfully */ }
                        Some(Ok(Err(e))) => {
                            tracing::error!(target: "rollup_node", "Critical error in sub-routine: {e}");
                            // Cancel all tasks and gracefully shutdown.
                            $cancellation.cancel();
                            return Err(e);
                        }
                        Some(Err(e)) => {
                            let error_msg = format!("Task join error: {e}");
                            // Log the error and cancel all tasks.
                            tracing::error!(target: "rollup_node", "Task join error: {e}");
                            // Cancel all tasks and gracefully shutdown.
                            $cancellation.cancel();
                            return Err(error_msg);
                        }
                        None => break, // All tasks completed
                    }
                }
            }
        }
    };
}

// Export the `spawn_and_wait` macro for use in other modules.
pub(crate) use spawn_and_wait;

/// Listens for OS shutdown signals (SIGTERM, SIGINT)
pub(crate) async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c().await.expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            tracing::info!(target: "rollup_node", "Received SIGINT (Ctrl+C)");
        },
        _ = terminate => {
            tracing::info!(target: "rollup_node", "Received SIGTERM");
        },
    }
}

#[cfg(all(test, feature = "metrics"))]
mod tests {
    use crate::{NodeActor, test_metrics::chains_of};
    use async_trait::async_trait;
    use tokio_util::sync::CancellationToken;

    const METRIC: &str = "kona_test_actor_steps";

    /// Emits `METRIC` on its first step, then stops the supervision loop.
    struct EmittingActor;

    #[async_trait]
    impl NodeActor for EmittingActor {
        type Error = &'static str;

        async fn step(&mut self) -> Result<(), Self::Error> {
            // Await first, so the emit cannot have run on the spawning task.
            tokio::task::yield_now().await;
            metrics::counter!(METRIC).increment(1);
            Err("done")
        }
    }

    async fn run_actor(chain_id: u64) -> Result<(), String> {
        let cancellation = CancellationToken::new();
        crate::service::spawn_and_wait!(
            cancellation,
            chain = kona_metrics::chain_label(chain_id),
            actors = [Some(EmittingActor)]
        );
        Ok(())
    }

    /// Actors are spawned, not awaited inline, so `spawn_and_wait!` is the one place that can
    /// scope all seven of them.
    #[tokio::test]
    async fn actor_metrics_carry_the_chain_the_supervisor_was_given() {
        // Installs the recorder before anything registers.
        assert!(chains_of(METRIC).is_empty());

        let error = run_actor(11155420).await.expect_err("the actor stops the loop");
        assert_eq!(error, "\"done\"");

        assert_eq!(chains_of(METRIC), vec![Some("11155420".to_string())]);
    }
}
