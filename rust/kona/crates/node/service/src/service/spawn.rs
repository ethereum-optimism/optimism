//! Running a set of [`NodeActor`]s to completion under a single cancellation token.

use crate::NodeActor;
use async_trait::async_trait;
use tokio_util::sync::CancellationToken;

/// A [`NodeActor`] with its error type erased.
///
/// Actors have unrelated error types, so the set of actors a host runs cannot be held in one
/// collection as `NodeActor`s. Rendering the error with its [`Debug`](std::fmt::Debug)
/// implementation at this boundary makes them uniform, so a host that composes a *dynamic* number
/// of actor groups — one per chain — can collect all of them into a single
/// `Vec<`[`BoxedNodeActor`]`>` and hand it to [`run_actors`].
///
/// This trait is implemented for every [`NodeActor`]; it is not meant to be implemented by hand.
#[async_trait]
pub trait ErasedNodeActor: Send + 'static {
    /// Steps the underlying actor, rendering a fatal error as a string.
    async fn step_erased(&mut self) -> Result<(), String>;
}

#[async_trait]
impl<A> ErasedNodeActor for A
where
    A: NodeActor,
{
    async fn step_erased(&mut self) -> Result<(), String> {
        self.step().await.map_err(|e| format!("{e:?}"))
    }
}

/// A boxed [`NodeActor`] with its error type erased, ready for [`run_actors`].
pub type BoxedNodeActor = Box<dyn ErasedNodeActor>;

/// Boxes a [`NodeActor`] into a [`BoxedNodeActor`].
pub trait IntoBoxedNodeActor {
    /// Erases this actor's error type and boxes it.
    fn boxed(self) -> BoxedNodeActor;
}

impl<A> IntoBoxedNodeActor for A
where
    A: NodeActor,
{
    fn boxed(self) -> BoxedNodeActor {
        Box::new(self)
    }
}

/// Steps `actor` in a loop until `cancellation` fires or the actor reports a fatal error.
///
/// The drop guard cancels the token when this task exits for *any* reason — success, error, or
/// panic — so peer actors observe shutdown on their next `select!`.
async fn drive_actor(
    mut actor: BoxedNodeActor,
    cancellation: CancellationToken,
) -> Result<(), String> {
    // Note the underscore prefix: this is to signal that we don't use the guard anywhere, but
    // *the compiler shouldn't optimize it away*. Note that using a simple `_` would not work here
    // because it gets optimized away in release mode.
    let _guard = cancellation.clone().drop_guard();
    loop {
        tokio::select! {
            biased;
            _ = cancellation.cancelled() => return Ok(()),
            result = actor.step_erased() => {
                result?;
            }
        }
    }
}

/// Runs `actors` in parallel in a [`JoinSet`](tokio::task::JoinSet), tearing the whole set down as
/// soon as any one of them exits.
///
/// This is the single place actors are spawned, and therefore the single definition of the node's
/// failure behaviour: whether a host runs one chain's actors or the actors of every chain in a
/// multi-chain process, a fatal error anywhere cancels `cancellation` and returns the error, rather
/// than leaving the process running with a dead actor. OS shutdown signals (SIGTERM, SIGINT) cancel
/// the same token.
///
/// ## Shutdown
///
/// Shutdown is unordered: when any actor exits (success, error, or panic) or an OS signal is
/// received, the cancellation token fires and all peer actors observe it on their next `select!`.
/// Actors may log channel-closed errors while peers are torn down concurrently; this is expected
/// and not a sign of an unclean exit.
pub async fn run_actors(
    cancellation: CancellationToken,
    actors: Vec<BoxedNodeActor>,
) -> Result<(), String> {
    let mut task_handles = tokio::task::JoinSet::new();
    for actor in actors {
        task_handles.spawn(drive_actor(actor, cancellation.clone()));
    }

    // Create the shutdown signal future
    let shutdown = shutdown_signal();
    tokio::pin!(shutdown);

    loop {
        tokio::select! {
            _ = &mut shutdown => {
                info!(target: "rollup_node", "Received shutdown signal, initiating graceful shutdown...");
                cancellation.cancel();
                break;
            }
            result = task_handles.join_next() => {
                match result {
                    Some(Ok(Ok(()))) => { /* Actor completed successfully */ }
                    Some(Ok(Err(e))) => {
                        error!(target: "rollup_node", "Critical error in sub-routine: {e}");
                        // Cancel all tasks and gracefully shutdown.
                        cancellation.cancel();
                        return Err(e);
                    }
                    Some(Err(e)) => {
                        let error_msg = format!("Task join error: {e}");
                        // Log the error and cancel all tasks.
                        error!(target: "rollup_node", "Task join error: {e}");
                        // Cancel all tasks and gracefully shutdown.
                        cancellation.cancel();
                        return Err(error_msg);
                    }
                    None => break, // All tasks completed
                }
            }
        }
    }

    Ok(())
}

/// Listens for OS shutdown signals (SIGTERM, SIGINT)
async fn shutdown_signal() {
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
