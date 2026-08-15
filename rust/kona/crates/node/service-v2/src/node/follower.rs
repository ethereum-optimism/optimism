//! Structured supervision for the initial V2 follower vertical slice.

use crate::{
    engine::{EngineClient, EngineDriver, EngineService},
    node::NodeError,
    unsafe_chain::{FollowerService, UnsafePayloadIngress},
};
use tokio_util::sync::CancellationToken;

/// A V2 follower composed from the engine and network unsafe-chain workflows.
///
/// This is intentionally a small vertical slice. Safe-chain, L1, network transport, and RPC tasks
/// will be added without changing the engine or follower task lifecycle.
#[derive(Debug)]
pub struct FollowerNode<Driver> {
    engine: EngineService<Driver>,
    engine_guard: EngineClient,
    follower: FollowerService,
}

impl<Driver> FollowerNode<Driver>
where
    Driver: EngineDriver,
{
    /// Creates a follower node, its semantic engine client, and network payload ingress.
    pub fn new(driver: Driver) -> (Self, EngineClient, UnsafePayloadIngress) {
        let (engine_service, engine) = EngineService::new(driver);
        let (follower, ingress) = FollowerService::new(engine.clone());
        (Self { engine: engine_service, engine_guard: engine.clone(), follower }, engine, ingress)
    }

    /// Runs the follower until shutdown or until a supervised task fails.
    ///
    /// Orderly shutdown first drains the unsafe-chain workflow while the engine remains available,
    /// then stops the engine service.
    pub async fn run(self, shutdown: CancellationToken) -> Result<(), NodeError> {
        let Self { engine, engine_guard: _engine_guard, follower } = self;
        let engine_shutdown = CancellationToken::new();
        let follower_shutdown = CancellationToken::new();
        let mut engine_task = tokio::spawn(engine.run(engine_shutdown.clone()));
        let mut follower_task = tokio::spawn(follower.run(follower_shutdown.clone()));

        tokio::select! {
            biased;
            _ = shutdown.cancelled() => {
                follower_shutdown.cancel();
                follower_task.await??;
                engine_shutdown.cancel();
                engine_task.await??;
                Ok(())
            }
            result = &mut engine_task => {
                follower_shutdown.cancel();
                let _ = follower_task.await;
                result??;
                Err(NodeError::Task("engine service exited without node shutdown".into()))
            }
            result = &mut follower_task => {
                engine_shutdown.cancel();
                let _ = engine_task.await;
                result??;
                Err(NodeError::Task("unsafe-chain service exited without node shutdown".into()))
            }
        }
    }
}
