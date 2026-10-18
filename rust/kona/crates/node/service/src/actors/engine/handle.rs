use super::{EngineClientError, EngineClientResult};
use alloy_eips::BlockNumHash;
use alloy_rpc_types_engine::PayloadId;
use kona_engine::{Consolidation, EngineState, FinalizationSource};
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::{mpsc, oneshot, watch};

/// The epoch and parent against which a producer prepared its work.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct EngineContext {
    /// Reset generation, including resets which return the same heads.
    pub epoch: u64,
    /// Accepted parent before the producer began asynchronous preparation.
    pub parent: L2BlockInfo,
}

/// An engine-owned build job. Callers cannot manufacture or restamp one after a reset.
#[derive(Debug, Clone)]
pub struct BuildToken {
    pub(super) context: EngineContext,
    pub(super) id: PayloadId,
    pub(super) timestamp: u64,
}

/// A retrieved payload tied to its original engine context, reusable for an exact import retry.
#[derive(Debug, Clone)]
pub struct Sealed {
    pub(super) context: EngineContext,
    pub(super) payload: OpExecutionPayloadEnvelope,
}

/// The actual accepted state and derivation outcome, published before this reply is sent.
#[derive(Debug)]
pub struct Consolidated {
    /// Accepted engine state, including pending progress and completed safe heads.
    pub state: EngineState,
    /// Whether the original batch applied, was replaced, or was dropped.
    pub outcome: Consolidation,
}

pub(crate) type Reply<T> = oneshot::Sender<EngineClientResult<T>>;

#[derive(Debug)]
pub(crate) enum Message {
    Start(EngineContext, OpAttributesWithParent, Reply<BuildToken>),
    Seal(BuildToken, Reply<Sealed>),
    Canonicalize(Box<Sealed>, Reply<OpExecutionPayloadEnvelope>),
    Consolidate(EngineContext, OpAttributesWithParent, Reply<Consolidated>),
    FollowSafe(EngineContext, L2BlockInfo, Reply<Consolidated>),
    Finalize(u64, BlockNumHash, FinalizationSource, Reply<()>),
}

/// Bounded unsafe admission did not accept the payload.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UnsafeAdmissionError {
    /// Best-effort gossip may be dropped when the engine is busy.
    Full,
    /// The node's engine owner stopped.
    Closed,
}

/// Cloneable access to the single engine owner. Channels and messages stay inside the service.
#[derive(Debug, Clone)]
pub struct EngineHandle {
    pub(super) commands: mpsc::Sender<Message>,
    pub(super) unsafe_payloads: mpsc::Sender<(u64, OpExecutionPayloadEnvelope)>,
    pub(super) resets: mpsc::Sender<Reply<()>>,
    pub(super) state: watch::Receiver<EngineState>,
    pub(super) pending: watch::Sender<usize>,
}

impl EngineHandle {
    /// Reads one atomic engine snapshot.
    pub fn state(&self) -> EngineState {
        *self.state.borrow()
    }

    /// Subscribes to accepted state and reset epochs.
    pub fn subscribe(&self) -> watch::Receiver<EngineState> {
        self.state.clone()
    }

    /// Counts all admitted work, including queued and currently executing operations.
    pub fn queue_length_subscribe(&self) -> watch::Receiver<usize> {
        self.pending.subscribe()
    }

    async fn send(&self, message: Message) -> EngineClientResult<()> {
        let permit = self.commands.reserve().await.map_err(|_| EngineClientError::Closed)?;
        self.pending.send_modify(|n| *n += 1);
        permit.send(message);
        Ok(())
    }

    /// Starts a sequencer job only if its captured context is still current.
    pub async fn start_build(
        &self,
        context: EngineContext,
        attrs: OpAttributesWithParent,
    ) -> EngineClientResult<BuildToken> {
        let (tx, rx) = oneshot::channel();
        self.send(Message::Start(context, attrs, tx)).await?;
        rx.await.map_err(|_| EngineClientError::Closed)?
    }

    /// Retrieves the exact job's payload, without importing it.
    pub async fn seal(&self, token: BuildToken) -> EngineClientResult<Sealed> {
        let (tx, rx) = oneshot::channel();
        self.send(Message::Seal(token, tx)).await?;
        rx.await.map_err(|_| EngineClientError::Closed)?
    }

    /// Imports and canonicalizes a retrieved payload while its context is current.
    pub async fn canonicalize(
        &self,
        sealed: Sealed,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        let (tx, rx) = oneshot::channel();
        self.send(Message::Canonicalize(Box::new(sealed), tx)).await?;
        rx.await.map_err(|_| EngineClientError::Closed)?
    }

    /// Applies derived attributes and returns the actual accepted result.
    pub async fn consolidate(
        &self,
        context: EngineContext,
        attrs: OpAttributesWithParent,
    ) -> EngineClientResult<Consolidated> {
        let (tx, rx) = oneshot::channel();
        self.send(Message::Consolidate(context, attrs, tx)).await?;
        rx.await.map_err(|_| EngineClientError::Closed)?
    }

    /// Follows an explicitly trusted derivation delegate's safe head.
    pub async fn follow_safe(
        &self,
        context: EngineContext,
        block: L2BlockInfo,
    ) -> EngineClientResult<Consolidated> {
        let (tx, rx) = oneshot::channel();
        self.send(Message::FollowSafe(context, block, tx)).await?;
        rx.await.map_err(|_| EngineClientError::Closed)?
    }

    /// Finalizes a recorded hash after the owner revalidates its epoch and provenance.
    pub async fn finalize(
        &self,
        epoch: u64,
        block: BlockNumHash,
        source: FinalizationSource,
    ) -> EngineClientResult<()> {
        let (tx, rx) = oneshot::channel();
        self.send(Message::Finalize(epoch, block, source, tx)).await?;
        rx.await.map_err(|_| EngineClientError::Closed)?
    }

    /// Requests a reset and waits until its accepted heads have been published.
    pub async fn reset(&self) -> EngineClientResult<()> {
        let (tx, rx) = oneshot::channel();
        let permit = self.resets.reserve().await.map_err(|_| EngineClientError::Closed)?;
        self.pending.send_modify(|n| *n += 1);
        permit.send(tx);
        rx.await.map_err(|_| EngineClientError::Closed)?
    }

    /// Waits for bounded reliable admission; success does not acknowledge payload validity.
    pub async fn insert_unsafe(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> EngineClientResult<()> {
        let permit = self.unsafe_payloads.reserve().await.map_err(|_| EngineClientError::Closed)?;
        self.pending.send_modify(|n| *n += 1);
        permit.send((self.state().epoch, payload));
        Ok(())
    }

    /// Attempts best-effort admission without stalling the network event loop.
    pub fn try_insert_unsafe(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafeAdmissionError> {
        let permit = self.unsafe_payloads.try_reserve().map_err(|error| match error {
            mpsc::error::TrySendError::Full(_) => UnsafeAdmissionError::Full,
            mpsc::error::TrySendError::Closed(_) => UnsafeAdmissionError::Closed,
        })?;
        self.pending.send_modify(|n| *n += 1);
        permit.send((self.state().epoch, payload));
        Ok(())
    }

    #[cfg(test)]
    pub(crate) fn test_channel(state: EngineState) -> (Self, TestEngine) {
        let (commands, command_rx) = mpsc::channel(32);
        let (unsafe_payloads, unsafe_rx) = mpsc::channel(1024);
        let (resets, reset_rx) = mpsc::channel(1);
        let (state_tx, state_rx) = watch::channel(state);
        let (pending, _) = watch::channel(0);
        (
            Self { commands, unsafe_payloads, resets, state: state_rx, pending },
            TestEngine {
                commands: command_rx,
                _unsafe_payloads: unsafe_rx,
                resets: reset_rx,
                state: state_tx,
            },
        )
    }
}

#[cfg(test)]
#[derive(Debug)]
pub(crate) struct TestEngine {
    pub commands: mpsc::Receiver<Message>,
    pub _unsafe_payloads: mpsc::Receiver<(u64, OpExecutionPayloadEnvelope)>,
    pub resets: mpsc::Receiver<Reply<()>>,
    pub state: watch::Sender<EngineState>,
}
