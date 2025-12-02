//! Utilities for the rollup node service, internal to the crate.

/// Spawns a set of parallel actors in a [JoinSet], and cancels all actors if any of them fail. The
/// type of the error in the [NodeActor]s is erased to avoid having to specify a common error type
/// between actors.
///
/// Actors are passed in as optional arguments, in case a given actor is not needed.
///
/// This macro also handles OS shutdown signals (SIGTERM, SIGINT) and triggers graceful shutdown
/// when received.
///
/// [JoinSet]: tokio::task::JoinSet
/// [NodeActor]: crate::NodeActor
macro_rules! spawn_and_wait {
    ($cancellation:expr, actors = [$($actor:expr$(,)?)*]) => {
        let mut task_handles = tokio::task::JoinSet::new();

        // Check if the actor is present, and spawn it if it is.
        $(
            if let Some((actor, context)) = $actor {
                let cancellation = $cancellation.clone();
                task_handles.spawn(async move {
                    // This guard ensures that the cancellation token is cancelled when the actor is
                    // dropped. This ensures that the actor is properly shut down.
                    // Note the underscore prefix: this is to signal that we don't use the guard anywhere, but
                    // *the compiler shouldn't optimize it away*.
                    // Note that using a simple `_` would not work here because it gets optimized away in
                    // release mode.
                    let _guard = cancellation.drop_guard();

                    if let Err(e) = actor.start(context).await {
                        return Err(format!("{e:?}"));
                    }
                    Ok(())
                });
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
