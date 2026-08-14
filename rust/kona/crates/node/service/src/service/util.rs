//! Utilities for the rollup node service, internal to the crate.

/// Spawns step-driven actors and long-running services, and supervises them as one node.
///
/// Actors are passed as `Option<actor>` and have their [`NodeActor::step`](crate::NodeActor::step)
/// method called in a loop. Services are passed as `Option<service>` and run one owned `run`
/// future.
///
/// Shutdown is ordered. Long-running services receive cancellation first and are allowed to reach a
/// safe point while their actor dependencies remain available. Once every service exits, the actor
/// cancellation token fires and the remaining actor tasks are drained.
macro_rules! spawn_and_wait {
    (
        $actor_cancellation:expr,
        services = [$($service:expr),* $(,)?],
        actors = [$($actor:expr),* $(,)?]
    ) => {
        let service_cancellation = tokio_util::sync::CancellationToken::new();
        let mut service_handles = tokio::task::JoinSet::new();
        let mut actor_handles = tokio::task::JoinSet::new();

        $(
            if let Some(service) = $service {
                let cancellation = service_cancellation.clone();
                service_handles.spawn(async move {
                    service.run(cancellation).await.map_err(|err| format!("{err:?}"))
                });
            }
        )*

        $(
            if let Some(mut actor) = $actor {
                let cancellation = $actor_cancellation.clone();
                actor_handles.spawn(async move {
                    loop {
                        tokio::select! {
                            biased;
                            _ = cancellation.cancelled() => return Ok(()),
                            result = actor.step() => {
                                result.map_err(|err| format!("{err:?}"))?;
                            }
                        }
                    }
                });
            }
        )*

        let shutdown = $crate::service::shutdown_signal();
        tokio::pin!(shutdown);

        let mut first_error = None::<String>;
        let mut stopping = false;

        while !service_handles.is_empty() || !actor_handles.is_empty() {
            tokio::select! {
                _ = &mut shutdown, if !stopping => {
                    tracing::info!(
                        target: "rollup_node",
                        "Received shutdown signal, stopping long-running services..."
                    );
                    stopping = true;
                    service_cancellation.cancel();
                    if service_handles.is_empty() {
                        $actor_cancellation.cancel();
                    }
                }
                result = service_handles.join_next(), if !service_handles.is_empty() => {
                    match result {
                        Some(Ok(Ok(()))) => {}
                        Some(Ok(Err(err))) => {
                            tracing::error!(target: "rollup_node", "Critical error in service: {err}");
                            if first_error.is_none() {
                                first_error = Some(err);
                            }
                        }
                        Some(Err(err)) => {
                            let error = format!("Service task join error: {err}");
                            tracing::error!(target: "rollup_node", "{error}");
                            if first_error.is_none() {
                                first_error = Some(error);
                            }
                        }
                        None => {}
                    }

                    if !stopping {
                        stopping = true;
                        service_cancellation.cancel();
                    }
                    if service_handles.is_empty() {
                        $actor_cancellation.cancel();
                    }
                }
                result = actor_handles.join_next(), if !actor_handles.is_empty() => {
                    match result {
                        Some(Ok(Ok(()))) => {}
                        Some(Ok(Err(err))) => {
                            tracing::error!(target: "rollup_node", "Critical error in actor: {err}");
                            if first_error.is_none() {
                                first_error = Some(err);
                            }
                        }
                        Some(Err(err)) => {
                            let error = format!("Actor task join error: {err}");
                            tracing::error!(target: "rollup_node", "{error}");
                            if first_error.is_none() {
                                first_error = Some(error);
                            }
                        }
                        None => {}
                    }

                    // Keep surviving actor dependencies alive until long-running services reach a
                    // safe point. The service will observe closed request channels if the actor that
                    // exited was one of its required dependencies.
                    if !stopping {
                        stopping = true;
                        service_cancellation.cancel();
                    }
                    if service_handles.is_empty() {
                        $actor_cancellation.cancel();
                    }
                }
            }
        }

        if let Some(error) = first_error {
            return Err(error);
        }
    };
}

// Export the `spawn_and_wait` macro for use in other modules.
pub(crate) use spawn_and_wait;

/// Listens for OS shutdown signals.
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
        }
        _ = terminate => {
            tracing::info!(target: "rollup_node", "Received SIGTERM");
        }
    }
}
