//! Utilities for the rollup node service, internal to the crate.

/// Spawns a set of parallel actors in a [JoinSet], and cancels all actors if any of them fail. The
/// type of the error in the [NodeActor]s is erased to avoid having to specify a common error type
/// between actors.
///
/// Actors are passed in as optional arguments, in case a given actor is not needed.
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

        while let Some(result) = task_handles.join_next().await {
            match result {
                Ok(Ok(())) => { /* Actor completed successfully */ }
                Ok(Err(e)) => {
                    tracing::error!(target: "rollup_node", "Critical error in sub-routine: {e}");
                    // Cancel all tasks and gracefully shutdown.
                    $cancellation.cancel();
                    return Err(e);
                }
                Err(e) => {
                    let error_msg = format!("Task join error: {e}");
                    // Log the error and cancel all tasks.
                    tracing::error!(target: "rollup_node", "Task join error: {e}");
                    // Cancel all tasks and gracefully shutdown.
                    $cancellation.cancel();
                    return Err(error_msg);
                }
            }
        }
    };
}

// Export the `spawn_and_wait` macro for use in other modules.
pub(crate) use spawn_and_wait;
