//! Historical proofs RPC server implementation.

use crate::{metrics::EthApiExtMetrics, state::OpStateProviderFactory};
use alloy_eips::BlockId;
use alloy_primitives::Address;
use alloy_rpc_types_eth::EIP1186AccountProofResponse;
use alloy_serde::JsonStorageKey;
use async_trait::async_trait;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::RpcResult;
use jsonrpsee_types::error::ErrorObject;
use reth_optimism_trie::{OpProofsStorage, OpProofsStore};
use reth_provider::StateProofProvider;
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_eth_types::EthApiError;
use reth_tasks::Runtime;
use std::{sync::Arc, time::Instant};
use tokio::sync::{OwnedSemaphorePermit, oneshot};

/// The `eth_` proof methods served from the historical proofs storage.
///
/// UPSTREAM-MIRROR(set): reth@rev:0fbe428 `reth_rpc_eth_api::EthApi`
///
/// Re-declares the proof methods that are answered from historical proofs rather than live state.
/// A proof method added to upstream's `EthApi` produces no diff here, so diff the two method sets
/// on each bump: every new one must either be declared here or dropped from the served surface in
/// `reth_optimism_node::node`, so that a proofs-history node never answers it from live state.
#[cfg_attr(not(test), rpc(server, namespace = "eth"))]
#[cfg_attr(test, rpc(server, client, namespace = "eth"))]
pub trait EthApiOverride {
    /// Returns the account and storage values of the specified account including the Merkle-proof.
    /// This call can be used to verify that the data you are pulling from is not tampered with.
    #[method(name = "getProof")]
    async fn get_proof(
        &self,
        address: Address,
        keys: Vec<JsonStorageKey>,
        block_number: Option<BlockId>,
    ) -> RpcResult<EIP1186AccountProofResponse>;
}

#[derive(Debug)]
/// Overrides applied to the `eth_` namespace of the RPC API for historical proofs ExEx.
pub struct EthApiExt<Eth, P> {
    state_provider_factory: Arc<OpStateProviderFactory<Eth, P>>,
    metrics: EthApiExtMetrics,
}

impl<Eth, P> EthApiExt<Eth, P>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
{
    /// Creates a new instance of the `EthApiExt`.
    pub fn new(eth_api: Eth, preimage_store: OpProofsStorage<P>) -> Self {
        let metrics = EthApiExtMetrics::default();
        Self {
            state_provider_factory: Arc::new(OpStateProviderFactory::new(eth_api, preimage_store)),
            metrics,
        }
    }
}

#[async_trait]
impl<Eth, P> EthApiOverrideServer for EthApiExt<Eth, P>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
{
    async fn get_proof(
        &self,
        address: Address,
        keys: Vec<JsonStorageKey>,
        block_number: Option<BlockId>,
    ) -> RpcResult<EIP1186AccountProofResponse> {
        let start = Instant::now();
        self.metrics.get_proof_requests.increment(1);

        let result = async {
            let eth_api = self.state_provider_factory.eth_api();
            // Reth's rpc.proof-permits guard is shared with proof methods and eth_simulateV1.
            // Both RPC surfaces use clones of the same eth API and therefore share admission.
            let permit = eth_api.acquire_owned_tracing().await;
            let permit = permit
                .map_err(|_| Eth::Error::from(EthApiError::InternalEthError))
                .map_err(ErrorObject::from)?;
            let factory = Arc::clone(&self.state_provider_factory);
            run_proof_worker(eth_api.io_task_spawner(), permit, async move {
                let state = factory
                    .state_provider(block_number)
                    .await
                    .map_err(Eth::Error::from)
                    .map_err(ErrorObject::from)?;
                let storage_keys = keys.iter().map(|key| key.as_b256()).collect::<Vec<_>>();
                let proof = state
                    .proof(Default::default(), address, &storage_keys)
                    .map_err(Eth::Error::from)
                    .map_err(ErrorObject::from)?;
                Ok(proof.into_eip1186_response(keys))
            })
            .await
        }
        .await;

        match &result {
            Ok(_) => {
                self.metrics.get_proof_latency.record(start.elapsed().as_secs_f64());
                self.metrics.get_proof_successful_responses.increment(1);
            }
            Err(_) => self.metrics.get_proof_failures.increment(1),
        }

        result
    }
}

/// Runs the complete provider operation on reth's blocking runtime, returning only owned data.
async fn run_proof_worker<T: Send + 'static>(
    runtime: &Runtime,
    permit: OwnedSemaphorePermit,
    work: impl Future<Output = RpcResult<T>> + Send + 'static,
) -> RpcResult<T> {
    let (tx, rx) = oneshot::channel();
    runtime.spawn_blocking_task(async move {
        // A cancelled waiter cannot release admission while synchronous work is still running.
        // The work future (and its providers) is dropped before this permit.
        let _permit = permit;
        if tx.is_closed() {
            tracing::debug!(target: "rpc::eth", "Skipping historical proof for disconnected waiter");
            return;
        }
        let mut outcome = WorkerOutcome { completed: false };
        let result = work.await;
        outcome.completed = true;
        if tx.send(result).is_err() {
            tracing::debug!(target: "rpc::eth", "Historical proof finished after waiter disconnected");
        }
    });
    rx.await.map_err(|_| ErrorObject::from(EthApiError::InternalEthError))?
}

/// Reports worker failures even when the RPC waiter has disconnected.
struct WorkerOutcome {
    completed: bool,
}

impl Drop for WorkerOutcome {
    fn drop(&mut self) {
        if !self.completed {
            tracing::warn!(target: "rpc::eth", "Historical proof worker exited without a result");
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use futures::poll;
    use std::{
        sync::{
            atomic::{AtomicBool, Ordering},
            mpsc,
        },
        task::Poll,
        time::Duration,
    };
    use tokio::sync::Semaphore;

    const DEADLINE: Duration = Duration::from_secs(10);

    /// A current-thread runtime has just one async worker; synchronous work must run elsewhere.
    #[tokio::test]
    async fn blocked_proof_keeps_async_runtime_responsive() {
        let runtime = Runtime::test();
        let semaphore = Arc::new(Semaphore::new(1));
        let permit = semaphore.clone().acquire_owned().await.unwrap();
        let (started_tx, started_rx) = oneshot::channel();
        let (release_tx, release_rx) = mpsc::channel();
        let mut proof = Box::pin(run_proof_worker(&runtime, permit, async move {
            started_tx.send(()).unwrap();
            release_rx.recv_timeout(DEADLINE).unwrap();
            Ok(42)
        }));
        assert!(poll!(&mut proof).is_pending());
        tokio::time::timeout(DEADLINE, started_rx).await.unwrap().unwrap();
        let unrelated = tokio::spawn(async { 7 });
        assert_eq!(tokio::time::timeout(DEADLINE, unrelated).await.unwrap().unwrap(), 7);
        assert_eq!(semaphore.available_permits(), 0);
        release_tx.send(()).unwrap();
        assert_eq!(tokio::time::timeout(DEADLINE, proof).await.unwrap().unwrap(), 42);
        // Receiving the response can race with the worker releasing admission.
        let _permit = tokio::time::timeout(DEADLINE, semaphore.acquire()).await.unwrap().unwrap();
    }

    #[tokio::test]
    async fn cancelled_waiter_holds_permit_through_provider_cleanup() {
        struct ProviderCleanup {
            semaphore: Arc<Semaphore>,
            cleaned: Arc<AtomicBool>,
        }
        impl Drop for ProviderCleanup {
            fn drop(&mut self) {
                assert_eq!(self.semaphore.available_permits(), 0);
                self.cleaned.store(true, Ordering::SeqCst);
            }
        }

        let runtime = Runtime::test();
        let semaphore = Arc::new(Semaphore::new(1));
        let cleaned = Arc::new(AtomicBool::new(false));
        let cleanup = ProviderCleanup { semaphore: semaphore.clone(), cleaned: cleaned.clone() };
        let (started_tx, started_rx) = oneshot::channel();
        let (release_tx, release_rx) = mpsc::channel();
        let mut proof = Box::pin(run_proof_worker(
            &runtime,
            semaphore.clone().acquire_owned().await.unwrap(),
            async move {
                let _provider = cleanup;
                started_tx.send(()).unwrap();
                release_rx.recv_timeout(DEADLINE).unwrap();
                Ok(())
            },
        ));
        assert!(poll!(&mut proof).is_pending());
        tokio::time::timeout(DEADLINE, started_rx).await.unwrap().unwrap();
        drop(proof);
        assert!(semaphore.try_acquire().is_err());
        let mut next = Box::pin(semaphore.acquire());
        assert!(poll!(&mut next).is_pending());
        assert!(!cleaned.load(Ordering::SeqCst));
        release_tx.send(()).unwrap();
        let _next_permit = tokio::time::timeout(DEADLINE, next).await.unwrap().unwrap();
        assert!(cleaned.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn provider_error_and_panic_release_admission() {
        let runtime = Runtime::test();
        let semaphore = Arc::new(Semaphore::new(1));
        let expected = ErrorObject::owned(-32000, "proof unavailable", None::<()>);
        let result = run_proof_worker(
            &runtime,
            semaphore.clone().acquire_owned().await.unwrap(),
            std::future::ready(Err::<(), _>(expected.clone())),
        )
        .await;
        assert_eq!(result.unwrap_err(), expected);

        let permit = tokio::time::timeout(DEADLINE, semaphore.clone().acquire_owned())
            .await
            .unwrap()
            .unwrap();
        let result = tokio::time::timeout(
            DEADLINE,
            run_proof_worker::<()>(&runtime, permit, async { panic!("proof worker panic") }),
        )
        .await
        .unwrap();
        assert_eq!(result.unwrap_err(), ErrorObject::from(EthApiError::InternalEthError));
        let _permit = tokio::time::timeout(DEADLINE, semaphore.acquire()).await.unwrap().unwrap();
    }

    #[tokio::test]
    async fn cloned_proof_guards_share_concurrency_budget() {
        use reth_tasks::pool::BlockingTaskGuard;
        use std::sync::atomic::AtomicUsize;

        let runtime = Runtime::test();
        let public_guard = BlockingTaskGuard::new(2);
        let auth_guard = public_guard.clone();
        let active = Arc::new(AtomicUsize::new(0));
        let maximum = Arc::new(AtomicUsize::new(0));
        let (started_tx, mut started_rx) = tokio::sync::mpsc::unbounded_channel();
        let mut releases = Vec::new();
        let mut requests = Vec::new();
        for i in 0..4 {
            let guard = if i % 2 == 0 { public_guard.clone() } else { auth_guard.clone() };
            let runtime = runtime.clone();
            let active = active.clone();
            let maximum = maximum.clone();
            let started_tx = started_tx.clone();
            let (release_tx, release_rx) = mpsc::channel();
            releases.push(release_tx);
            requests.push(tokio::spawn(async move {
                let permit = guard.acquire_owned().await.unwrap();
                run_proof_worker(&runtime, permit, async move {
                    let count = active.fetch_add(1, Ordering::SeqCst) + 1;
                    maximum.fetch_max(count, Ordering::SeqCst);
                    started_tx.send(i).unwrap();
                    release_rx.recv_timeout(DEADLINE).unwrap();
                    active.fetch_sub(1, Ordering::SeqCst);
                    Ok(())
                })
                .await
            }));
        }

        let first = tokio::time::timeout(DEADLINE, started_rx.recv()).await.unwrap().unwrap();
        let second = tokio::time::timeout(DEADLINE, started_rx.recv()).await.unwrap().unwrap();
        assert_eq!(active.load(Ordering::SeqCst), 2);
        assert!(started_rx.try_recv().is_err());
        releases[first].send(()).unwrap();
        let third = tokio::time::timeout(DEADLINE, started_rx.recv()).await.unwrap().unwrap();
        assert_eq!(active.load(Ordering::SeqCst), 2);
        releases[second].send(()).unwrap();
        let fourth = tokio::time::timeout(DEADLINE, started_rx.recv()).await.unwrap().unwrap();
        releases[third].send(()).unwrap();
        releases[fourth].send(()).unwrap();
        for request in requests {
            tokio::time::timeout(DEADLINE, request).await.unwrap().unwrap().unwrap();
        }
        assert_eq!(maximum.load(Ordering::SeqCst), 2);
        assert_eq!(active.load(Ordering::SeqCst), 0);
        let _permit = tokio::time::timeout(DEADLINE, public_guard.acquire_many_owned(2))
            .await
            .unwrap()
            .unwrap();
    }

    #[test]
    fn cancelled_queued_worker_skips_provider_setup() {
        let tokio_runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .max_blocking_threads(1)
            .build()
            .unwrap();
        tokio_runtime.block_on(async {
            let runtime = Runtime::test();
            let (started_tx, started_rx) = oneshot::channel();
            let (release_tx, release_rx) = mpsc::channel();
            let blocker = tokio::task::spawn_blocking(move || {
                started_tx.send(()).unwrap();
                release_rx.recv_timeout(DEADLINE).unwrap();
            });
            tokio::time::timeout(DEADLINE, started_rx).await.unwrap().unwrap();
            let semaphore = Arc::new(Semaphore::new(1));
            let ran = Arc::new(AtomicBool::new(false));
            let worker_ran = ran.clone();
            let mut proof = Box::pin(run_proof_worker(
                &runtime,
                semaphore.clone().acquire_owned().await.unwrap(),
                async move {
                    worker_ran.store(true, Ordering::SeqCst);
                    Ok(())
                },
            ));
            assert!(matches!(poll!(&mut proof), Poll::Pending));
            drop(proof);
            assert_eq!(semaphore.available_permits(), 0);
            release_tx.send(()).unwrap();
            tokio::time::timeout(DEADLINE, blocker).await.unwrap().unwrap();
            let _permit =
                tokio::time::timeout(DEADLINE, semaphore.acquire()).await.unwrap().unwrap();
            assert!(!ran.load(Ordering::SeqCst));
        });
    }
}
