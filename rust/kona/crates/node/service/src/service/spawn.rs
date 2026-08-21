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

/// An actor whose failures name the chain it belongs to.
///
/// [`NodeActor::Error`] is only `Debug`, and [`run_actors`] renders it with that implementation, so
/// a multi-chain host's supervision line would otherwise read `Critical error in sub-routine:
/// <debug>` with nothing in it to say which of N chains just died — the same rendering whichever
/// chain's derivation, engine or network actor it was. Wrapping a chain's actors in this labels
/// their failures at the seam where the chain is still known, without touching the actors
/// themselves or the error types they define.
///
/// A single-chain host does not wrap anything, so its output is unchanged.
pub struct ChainLabeledActor {
    /// The chain this actor belongs to.
    chain_id: u64,
    /// The actor whose failures are labelled.
    actor: BoxedNodeActor,
}

impl core::fmt::Debug for ChainLabeledActor {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        // The wrapped actor is a boxed trait object and cannot be formatted; the chain it belongs
        // to is the useful part anyway.
        f.debug_struct("ChainLabeledActor")
            .field("chain_id", &self.chain_id)
            .finish_non_exhaustive()
    }
}

impl ChainLabeledActor {
    /// Labels `actor`'s failures with `chain_id`.
    pub const fn new(chain_id: u64, actor: BoxedNodeActor) -> Self {
        Self { chain_id, actor }
    }
}

#[async_trait]
impl ErasedNodeActor for ChainLabeledActor {
    async fn step_erased(&mut self) -> Result<(), String> {
        self.actor.step_erased().await.map_err(|e| format!("chain {}: {e}", self.chain_id))
    }
}

/// Labels every actor of one chain, so a fatal error names the chain that produced it.
///
/// The counterpart of [`ComposedChain::actors`](crate::ComposedChain::actors) for a multi-chain
/// host: composition returns a chain's actors unlabelled, because a single-chain host has no second
/// chain to tell them apart from, and the multi-chain host labels them as it collects them.
pub fn label_chain(chain_id: u64, actors: Vec<BoxedNodeActor>) -> Vec<BoxedNodeActor> {
    actors
        .into_iter()
        .map(|actor| Box::new(ChainLabeledActor::new(chain_id, actor)) as BoxedNodeActor)
        .collect()
}

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

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    };

    /// An actor that never finishes a step, standing in for a healthy peer.
    struct PendingActor;

    #[async_trait]
    impl NodeActor for PendingActor {
        type Error = &'static str;

        async fn step(&mut self) -> Result<(), Self::Error> {
            std::future::pending().await
        }
    }

    /// An actor that steps `steps` times and then reports a fatal error.
    struct FailingActor {
        steps: usize,
        stepped: Arc<AtomicUsize>,
    }

    #[async_trait]
    impl NodeActor for FailingActor {
        type Error = &'static str;

        async fn step(&mut self) -> Result<(), Self::Error> {
            if self.steps == 0 {
                return Err("fatal");
            }
            self.steps -= 1;
            self.stepped.fetch_add(1, Ordering::Relaxed);
            Ok(())
        }
    }

    #[tokio::test]
    async fn fatal_error_tears_down_peer_actors() {
        let cancellation = CancellationToken::new();
        let stepped = Arc::new(AtomicUsize::new(0));

        let err = run_actors(
            cancellation.clone(),
            vec![
                PendingActor.boxed(),
                FailingActor { steps: 3, stepped: stepped.clone() }.boxed(),
                PendingActor.boxed(),
            ],
        )
        .await
        .expect_err("the failing actor must surface its error");

        // The error is the actor's own, rendered through `Debug`.
        assert_eq!(err, format!("{:?}", "fatal"));
        // The actor was stepped repeatedly before failing, and its peers were torn down: this is
        // the process-wide fail-fast a multi-chain host relies on.
        assert_eq!(stepped.load(Ordering::Relaxed), 3);
        assert!(cancellation.is_cancelled(), "a fatal actor error must cancel its peers");
    }

    #[tokio::test]
    async fn external_cancellation_is_a_clean_exit() {
        let cancellation = CancellationToken::new();
        let token = cancellation.clone();
        tokio::spawn(async move { token.cancel() });

        run_actors(cancellation, vec![PendingActor.boxed(), PendingActor.boxed()])
            .await
            .expect("cancellation is not an error");
    }

    /// A multi-chain host's supervision line has to name the chain: with N chains, the actor's own
    /// `Debug` rendering does not say which one died.
    #[tokio::test]
    async fn a_labelled_actor_names_its_chain() {
        let stepped = Arc::new(AtomicUsize::new(0));

        let err = run_actors(
            CancellationToken::new(),
            label_chain(901, vec![FailingActor { steps: 0, stepped }.boxed()]),
        )
        .await
        .expect_err("the failing actor must surface its error");

        assert_eq!(err, format!("chain 901: {:?}", "fatal"));
    }

    /// A single-chain host labels nothing, so its error is exactly what it always was.
    #[tokio::test]
    async fn an_unlabelled_actor_reads_as_before() {
        let stepped = Arc::new(AtomicUsize::new(0));

        let err =
            run_actors(CancellationToken::new(), vec![FailingActor { steps: 0, stepped }.boxed()])
                .await
                .expect_err("the failing actor must surface its error");

        assert_eq!(err, format!("{:?}", "fatal"));
    }

    #[tokio::test]
    async fn no_actors_returns_immediately() {
        // A host with nothing to run must not block on the shutdown signal.
        run_actors(CancellationToken::new(), Vec::new()).await.expect("empty actor set");
    }
}
