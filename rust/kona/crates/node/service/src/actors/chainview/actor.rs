//! The chain view actor: feeds the circuit what the derivation actor does not, and publishes
//! what the circuit derives.
//!
//! It mirrors the L1 tags the watcher polls and the engine's heads into the circuit, reads the
//! unsafe-block signer from the `SystemConfig` contract at every new L1 head, forwards the
//! signer the circuit derives to the network actor, and turns the circuit thread's exit into an
//! actor error. The L1 blocks themselves come from the derivation actor, which pushes every
//! origin its pipeline advances through.

use super::L1Fetcher;
use crate::{L1Watches, NodeActor};
use alloy_primitives::Address;
use async_trait::async_trait;
use kona_chainview::{
    ChainViewClient, ChainViewError, ChainViewHandle, ChainViewSnapshot, Fact, L1StatusKind,
    L2Heads,
};
use kona_engine::EngineState;
use kona_protocol::BlockInfo;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

/// Errors of the [`ChainViewActor`].
#[derive(Error, Debug)]
pub enum ChainViewActorError {
    /// The circuit thread exited.
    #[error("chain view exited: {0}")]
    CircuitExited(ChainViewError),
    /// The circuit thread exited without reporting a result.
    #[error("chain view exited without a result")]
    CircuitVanished,
    /// The engine state watch closed.
    #[error("engine state watch closed")]
    EngineStateClosed,
    /// The L1 watcher's tag watches closed.
    #[error("L1 watcher closed")]
    L1WatcherClosed,
    /// Pushing a fact failed.
    #[error("chain view push failed: {0}")]
    Push(ChainViewError),
}

/// What woke the actor.
enum Event {
    Exit(Result<Result<(), ChainViewError>, oneshot::error::RecvError>),
    Engine(Result<(), watch::error::RecvError>),
    Snapshot(Result<(), watch::error::RecvError>),
    Head(Result<(), watch::error::RecvError>),
    Finalized(Result<(), watch::error::RecvError>),
    Safe(Result<(), watch::error::RecvError>),
}

/// Feeds L1 tags and engine heads to the chain view and publishes what it derives.
#[derive(Debug)]
pub struct ChainViewActor<F> {
    client: ChainViewClient,
    exit: oneshot::Receiver<Result<(), ChainViewError>>,
    engine_state: watch::Receiver<EngineState>,
    snapshots: watch::Receiver<ChainViewSnapshot>,
    l1: L1Watches,
    fetcher: F,
    signer_tx: mpsc::Sender<Address>,
    last_signer: Option<Address>,
}

impl<F: L1Fetcher> ChainViewActor<F> {
    /// Wraps a running chain view.
    ///
    /// `fetcher` reads the unsafe-block signer at each new L1 head in `l1`; `signer_tx` is the
    /// network actor's unsafe-block-signer channel, fed with the circuit's `current_signer`
    /// whenever it changes.
    pub fn new(
        handle: ChainViewHandle,
        fetcher: F,
        l1: L1Watches,
        engine_state: watch::Receiver<EngineState>,
        signer_tx: mpsc::Sender<Address>,
    ) -> Self {
        let snapshots = handle.client.subscribe();
        let ChainViewHandle { client, exit, .. } = handle;
        Self { client, exit, engine_state, snapshots, l1, fetcher, signer_tx, last_signer: None }
    }

    /// The engine's four labels as a chain view fact.
    pub const fn heads_of(state: &EngineState) -> L2Heads {
        let sync = state.sync_state;
        L2Heads {
            unsafe_head: sync.unsafe_head(),
            local_safe_head: sync.local_safe_head(),
            safe_head: sync.safe_head(),
            finalized_head: sync.finalized_head(),
        }
    }

    async fn push(&self, fact: Fact) -> Result<(), ChainViewActorError> {
        self.client.push(fact).await.map_err(ChainViewActorError::Push)
    }

    async fn push_engine_state(&mut self) -> Result<(), ChainViewActorError> {
        let heads = Self::heads_of(&self.engine_state.borrow_and_update());
        self.push(Fact::L2Status(Box::new(heads))).await
    }

    /// Publishes a new L1 head, and the unsafe-block signer the contract holds at it. A failed
    /// signer read is retried at the next head; the previous read stays in force until then.
    async fn on_head(&self, head: BlockInfo) -> Result<(), ChainViewActorError> {
        self.push(Fact::L1Status { kind: L1StatusKind::Head, block: head }).await?;
        match self.fetcher.unsafe_block_signer_at(head.hash).await {
            Ok(signer) => self.push(Fact::UnsafeBlockSigner { l1: head, signer }).await,
            Err(e) => {
                warn!(target: "chainview", head = head.number, error = ?e, "could not read the unsafe block signer at the L1 head; retrying at the next head");
                Ok(())
            }
        }
    }

    async fn forward_signer(&mut self) {
        let signer = self.snapshots.borrow_and_update().signer;
        let Some(signer) = signer else { return };
        if self.last_signer == Some(signer) {
            return;
        }
        info!(target: "chainview", %signer, "unsafe block signer from the chain view");
        self.last_signer = Some(signer);
        if let Err(e) = self.signer_tx.send(signer).await {
            error!(target: "chainview", "failed to forward the unsafe block signer: {e}");
        }
    }

    /// The latest value of an L1 tag watch, marking it seen.
    fn latest(rx: &mut watch::Receiver<Option<BlockInfo>>) -> Option<BlockInfo> {
        *rx.borrow_and_update()
    }
}

#[async_trait]
impl<F: L1Fetcher + 'static> NodeActor for ChainViewActor<F> {
    type Error = ChainViewActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        // The arms only classify the wake-up; handling happens after the borrows they hold on
        // the receivers are released.
        let event = tokio::select! {
            biased;
            exit = &mut self.exit => Event::Exit(exit),
            changed = self.engine_state.changed() => Event::Engine(changed),
            changed = self.snapshots.changed() => Event::Snapshot(changed),
            changed = self.l1.head.changed() => Event::Head(changed),
            changed = self.l1.finalized.changed() => Event::Finalized(changed),
            changed = self.l1.safe.changed() => Event::Safe(changed),
        };

        match event {
            Event::Exit(Ok(Ok(()))) => {
                Err(ChainViewActorError::CircuitExited(ChainViewError::Closed))
            }
            Event::Exit(Ok(Err(e))) => Err(ChainViewActorError::CircuitExited(e)),
            Event::Exit(Err(_)) => Err(ChainViewActorError::CircuitVanished),
            Event::Engine(changed) => {
                changed.map_err(|_| ChainViewActorError::EngineStateClosed)?;
                self.push_engine_state().await
            }
            Event::Snapshot(changed) => {
                changed.map_err(|_| ChainViewActorError::CircuitExited(ChainViewError::Closed))?;
                self.forward_signer().await;
                Ok(())
            }
            Event::Head(changed) => {
                changed.map_err(|_| ChainViewActorError::L1WatcherClosed)?;
                match Self::latest(&mut self.l1.head) {
                    Some(head) => self.on_head(head).await,
                    None => Ok(()),
                }
            }
            Event::Finalized(changed) => {
                changed.map_err(|_| ChainViewActorError::L1WatcherClosed)?;
                match Self::latest(&mut self.l1.finalized) {
                    Some(block) => {
                        self.push(Fact::L1Status { kind: L1StatusKind::Finalized, block }).await
                    }
                    None => Ok(()),
                }
            }
            Event::Safe(changed) => {
                changed.map_err(|_| ChainViewActorError::L1WatcherClosed)?;
                match Self::latest(&mut self.l1.safe) {
                    Some(block) => {
                        self.push(Fact::L1Status { kind: L1StatusKind::Safe, block }).await
                    }
                    None => Ok(()),
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actors::chainview::MockL1Fetcher;
    use alloy_primitives::B256;
    use kona_chainview::{ChainViewConfig, spawn};

    /// The actor plus the senders of its L1 watches.
    struct Fixture {
        actor: ChainViewActor<MockL1Fetcher>,
        client: ChainViewClient,
        head_tx: watch::Sender<Option<BlockInfo>>,
        finalized_tx: watch::Sender<Option<BlockInfo>>,
        engine_tx: watch::Sender<EngineState>,
    }

    fn fixture(fetcher: MockL1Fetcher) -> Fixture {
        let handle = spawn(ChainViewConfig::default()).expect("circuit");
        let client = handle.client.clone();
        let (engine_tx, engine_rx) = watch::channel(EngineState::default());
        let (signer_tx, _signer_rx) = mpsc::channel(4);
        let (head_tx, head) = watch::channel(None);
        let (finalized_tx, finalized) = watch::channel(None);
        let (_safe_tx, safe) = watch::channel(None);
        let actor = ChainViewActor::new(
            handle,
            fetcher,
            L1Watches { head, finalized, safe },
            engine_rx,
            signer_tx,
        );
        Fixture { actor, client, head_tx, finalized_tx, engine_tx }
    }

    #[tokio::test]
    async fn engine_state_changes_reach_the_snapshot() {
        let mut f = fixture(MockL1Fetcher::new());

        let mut state = EngineState::default();
        state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
            safe_head: Some(kona_protocol::L2BlockInfo {
                block_info: kona_protocol::BlockInfo { number: 7, ..Default::default() },
                ..Default::default()
            }),
            ..Default::default()
        });
        f.engine_tx.send_replace(state);
        f.actor.step().await.expect("push engine state");
        f.client.sync().await.expect("sync");
        assert_eq!(f.client.snapshot().l2.expect("heads").safe_head.block_info.number, 7);
        f.client.try_shutdown().expect("shutdown");
    }

    /// A new L1 head becomes the head status and the signer read at it; a new finalized block
    /// becomes the finalized status.
    #[tokio::test]
    async fn l1_tags_become_statuses_and_the_head_seeds_the_signer() {
        let signer = Address::repeat_byte(0x42);
        let mut fetcher = MockL1Fetcher::new();
        fetcher.expect_unsafe_block_signer_at().times(1).returning(move |_| Ok(signer));
        let mut f = fixture(fetcher);

        let head = BlockInfo { number: 100, hash: B256::repeat_byte(1), ..Default::default() };
        let finalized = BlockInfo { number: 90, hash: B256::repeat_byte(2), ..Default::default() };
        f.head_tx.send_replace(Some(head));
        f.finalized_tx.send_replace(Some(finalized));
        f.actor.step().await.expect("head");
        f.actor.step().await.expect("finalized");
        f.client.sync().await.expect("sync");

        let snapshot = f.client.snapshot();
        assert_eq!(snapshot.l1.head, Some(head));
        assert_eq!(snapshot.l1.finalized, Some(finalized));
        assert_eq!(snapshot.signer, Some(signer));
        f.client.try_shutdown().expect("shutdown");
    }
}
