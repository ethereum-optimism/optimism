//! Node composition root and structured task supervision.

use crate::{
    Engine, Rpc, RpcHandle, SafeChainBuilder, SafeChainBuilderHandle, UnsafeChainBuilder,
    UnsafeChainBuilderHandle,
};
use std::{future::Future, sync::Arc};
use thiserror::Error;
use tokio::{
    sync::Mutex,
    task::{JoinError, JoinHandle},
};

/// The result produced by each erased top-level service task.
type ServiceResult = Result<(), String>;
type ServiceTask = JoinHandle<ServiceResult>;

/// Fully assembled service-v3 task graph.
#[derive(Debug)]
pub struct RollupNode<SafeL1, L2, UnsafeL1, EngineClient, Network, Conductor> {
    safe_chain: SafeChainBuilder<SafeL1, L2, EngineClient>,
    unsafe_chain: UnsafeChainBuilder<UnsafeL1, EngineClient, Network, Conductor>,
    rpc: Rpc,
    safe_chain_handle: SafeChainBuilderHandle,
    unsafe_chain_handle: UnsafeChainBuilderHandle,
    rpc_handle: RpcHandle,
}

impl<SafeL1, L2, UnsafeL1, EngineClient, Network, Conductor>
    RollupNode<SafeL1, L2, UnsafeL1, EngineClient, Network, Conductor>
{
    /// Constructs the complete three-task graph without spawning any task.
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        safe_l1: SafeL1,
        l2_el: L2,
        unsafe_l1: UnsafeL1,
        engine: Engine<EngineClient>,
        network: Network,
        conductor: Option<Conductor>,
    ) -> Self {
        let engine = Arc::new(Mutex::new(engine));
        let (safe_chain, safe_chain_handle) = SafeChainBuilder::new(safe_l1, l2_el, engine.clone());
        let (unsafe_chain, unsafe_chain_handle) =
            UnsafeChainBuilder::new(unsafe_l1, engine, network, conductor);
        let (rpc, rpc_handle) = Rpc::new(safe_chain_handle.clone(), unsafe_chain_handle.clone());
        Self { safe_chain, unsafe_chain, rpc, safe_chain_handle, unsafe_chain_handle, rpc_handle }
    }

    /// Returns a control handle for the safe-chain task.
    pub const fn safe_chain_handle(&self) -> &SafeChainBuilderHandle {
        &self.safe_chain_handle
    }

    /// Returns a control handle for the unsafe-chain task.
    pub const fn unsafe_chain_handle(&self) -> &UnsafeChainBuilderHandle {
        &self.unsafe_chain_handle
    }
}

impl<SafeL1, L2, UnsafeL1, EngineClient, Network, Conductor>
    RollupNode<SafeL1, L2, UnsafeL1, EngineClient, Network, Conductor>
where
    SafeL1: Send + 'static,
    L2: Send + 'static,
    UnsafeL1: Send + 'static,
    EngineClient: Send + Sync + 'static,
    Network: Send + 'static,
    Conductor: Send + 'static,
{
    /// Starts all core tasks, handles process shutdown signals, and joins every task.
    pub async fn run(self) -> Result<(), NodeError> {
        self.run_until(shutdown_signal()).await
    }

    /// Starts all core tasks and shuts them down when the supplied future completes.
    ///
    /// This entry point makes supervision deterministic in tests and embedding applications.
    pub async fn run_until<Shutdown>(self, shutdown: Shutdown) -> Result<(), NodeError>
    where
        Shutdown: Future<Output = Result<(), String>>,
    {
        let Self {
            safe_chain,
            unsafe_chain,
            rpc,
            safe_chain_handle,
            unsafe_chain_handle,
            rpc_handle,
        } = self;

        let safe_task =
            tokio::spawn(async move { safe_chain.run().await.map_err(|error| error.to_string()) });
        let unsafe_task =
            tokio::spawn(
                async move { unsafe_chain.run().await.map_err(|error| error.to_string()) },
            );
        let rpc_task =
            tokio::spawn(async move { rpc.run().await.map_err(|error| error.to_string()) });

        let mut running = RunningNode {
            safe_task: Some(safe_task),
            unsafe_task: Some(unsafe_task),
            rpc_task: Some(rpc_task),
            safe_chain_handle,
            unsafe_chain_handle,
            rpc_handle,
        };

        let cause = tokio::select! {
            signal = shutdown => signal.err().map(NodeError::Signal),
            failure = running.wait_for_critical_exit() => Some(failure),
        };

        let cleanup = running.shutdown().await.err();
        cause.or(cleanup).map_or(Ok(()), Err)
    }
}

struct RunningNode {
    safe_task: Option<ServiceTask>,
    unsafe_task: Option<ServiceTask>,
    rpc_task: Option<ServiceTask>,
    safe_chain_handle: SafeChainBuilderHandle,
    unsafe_chain_handle: UnsafeChainBuilderHandle,
    rpc_handle: RpcHandle,
}

impl RunningNode {
    async fn wait_for_critical_exit(&mut self) -> NodeError {
        enum Service {
            SafeChain,
            UnsafeChain,
            Rpc,
        }

        let (service, result) = tokio::select! {
            result = poll_task(&mut self.safe_task) => (Service::SafeChain, result),
            result = poll_task(&mut self.unsafe_task) => (Service::UnsafeChain, result),
            result = poll_task(&mut self.rpc_task) => (Service::Rpc, result),
        };

        let name = match service {
            Service::SafeChain => {
                self.safe_task.take();
                "safe-chain"
            }
            Service::UnsafeChain => {
                self.unsafe_task.take();
                "unsafe-chain"
            }
            Service::Rpc => {
                self.rpc_task.take();
                "RPC"
            }
        };
        unexpected_exit(name, result)
    }

    async fn shutdown(mut self) -> Result<(), NodeError> {
        let mut first_error = None;

        shutdown_service("RPC", &mut self.rpc_task, self.rpc_handle.shutdown(), &mut first_error)
            .await;
        shutdown_service(
            "unsafe-chain",
            &mut self.unsafe_task,
            self.unsafe_chain_handle.shutdown(),
            &mut first_error,
        )
        .await;
        shutdown_service(
            "safe-chain",
            &mut self.safe_task,
            self.safe_chain_handle.shutdown(),
            &mut first_error,
        )
        .await;

        first_error.map_or(Ok(()), Err)
    }
}

async fn shutdown_service<ShutdownFuture>(
    name: &'static str,
    task: &mut Option<ServiceTask>,
    shutdown: ShutdownFuture,
    first_error: &mut Option<NodeError>,
) where
    ShutdownFuture: Future<Output = Result<(), crate::ControlError>>,
{
    let Some(service_task) = task.take() else { return };

    if let Err(error) = shutdown.await {
        record_error(NodeError::Shutdown { service: name, error: error.to_string() }, first_error);
    }
    match service_task.await {
        Ok(Ok(())) => {}
        Ok(Err(error)) => {
            record_error(NodeError::Shutdown { service: name, error }, first_error);
        }
        Err(error) => {
            record_error(
                NodeError::TaskPanic { service: name, error: error.to_string() },
                first_error,
            );
        }
    }
}

async fn poll_task(task: &mut Option<ServiceTask>) -> Result<ServiceResult, JoinError> {
    match task {
        Some(task) => task.await,
        None => std::future::pending().await,
    }
}

fn unexpected_exit(service: &'static str, result: Result<ServiceResult, JoinError>) -> NodeError {
    match result {
        Ok(Ok(())) => NodeError::UnexpectedExit { service },
        Ok(Err(error)) => NodeError::ServiceFailure { service, error },
        Err(error) => NodeError::TaskPanic { service, error: error.to_string() },
    }
}

fn record_error(error: NodeError, first_error: &mut Option<NodeError>) {
    tracing::error!(target: "rollup_node_v3", %error, "Service shutdown failed");
    if first_error.is_none() {
        *first_error = Some(error);
    }
}

async fn shutdown_signal() -> Result<(), String> {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .map_err(|error| format!("failed to install Ctrl-C handler: {error}"))
    };
    #[cfg(unix)]
    let terminate = async {
        let mut signal = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .map_err(|error| format!("failed to install SIGTERM handler: {error}"))?;
        signal.recv().await;
        Ok(())
    };
    #[cfg(not(unix))]
    let terminate = std::future::pending::<Result<(), String>>();

    tokio::select! {
        result = ctrl_c => result,
        result = terminate => result,
    }
}

/// A top-level node lifecycle or task failure.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum NodeError {
    /// Installing or receiving the process shutdown signal failed.
    #[error("shutdown signal failed: {0}")]
    Signal(String),
    /// A critical service returned successfully without a shutdown request.
    #[error("{service} service terminated unexpectedly")]
    UnexpectedExit {
        /// Name of the service.
        service: &'static str,
    },
    /// A critical service returned an error.
    #[error("{service} service failed: {error}")]
    ServiceFailure {
        /// Name of the service.
        service: &'static str,
        /// Service failure description.
        error: String,
    },
    /// A critical service task panicked.
    #[error("{service} service task panicked: {error}")]
    TaskPanic {
        /// Name of the service.
        service: &'static str,
        /// Join failure description.
        error: String,
    },
    /// A service failed during ordered shutdown.
    #[error("failed to shut down {service} service: {error}")]
    Shutdown {
        /// Name of the service.
        service: &'static str,
        /// Shutdown failure description.
        error: String,
    },
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Engine;
    use kona_genesis::RollupConfig;
    use std::sync::Arc;
    use tokio::sync::oneshot;

    #[tokio::test]
    async fn node_starts_and_joins_all_three_tasks() {
        let engine = Engine::new(Arc::new(()), Arc::new(RollupConfig::default()));
        let node = RollupNode::new((), (), (), engine, (), None::<()>);
        let (shutdown_tx, shutdown_rx) = oneshot::channel();

        let running = tokio::spawn(
            node.run_until(async move { shutdown_rx.await.map_err(|error| error.to_string()) }),
        );
        shutdown_tx.send(()).unwrap();

        assert_eq!(running.await.unwrap(), Ok(()));
    }
}
