use super::AppliedEpoch;
use crate::NodeActor;
use async_trait::async_trait;
use kona_engine::{EngineActorRequest, EngineState, UndoRequest, VerifierStatus};
use kona_genesis::RollupConfig;
use kona_rpc::InteropAuthorityApiClient;
use std::{sync::Arc, time::Duration};
use thiserror::Error;
use tokio::{
    sync::{mpsc, watch},
    time,
};

/// How often the authority is polled for the chain's heads.
pub const AUTHORITY_POLL_INTERVAL: Duration = Duration::from_secs(1);

/// How long a poll waits for the engine to move its labels back on a new epoch before it tries
/// again on a later poll.
const UNDO_TIMEOUT: Duration = Duration::from_secs(30);

/// The [`NodeActor`] following a cross-chain verification authority.
///
/// It polls the authority for this chain's status and hands it to the engine, which owns what the
/// safe label becomes ([`kona_engine::PromoteTask`]), what is final
/// ([`kona_engine::VerifierFinality`]) and how a denied block is replaced
/// ([`kona_engine::DenyTask`]). An authority that is unreachable or answers no heads moves no
/// label from the authority: the labels hold where they are, and the engine keeps the last denied
/// blocks it heard.
///
/// The authority owns undoing its decisions. When its epoch differs from the one this node
/// applied, it deleted a denial the node may have replaced a block on, or lost its store, so the
/// node derives again from its finalized head with the new denied blocks
/// ([`kona_engine::UndoTask`]), and only then records the new epoch. The node follows nothing
/// else of an answer on another epoch. The epoch is held in memory only, see [`AppliedEpoch`]:
/// the first answer after a start always undoes, which moves nothing once the start derives
/// from the finalized head.
#[derive(Debug)]
pub struct AuthorityActor<Authority> {
    /// The authority's client.
    authority: Authority,
    /// The rollup config of this chain.
    rollup: Arc<RollupConfig>,
    /// Where the heads go.
    engine_request_tx: mpsc::Sender<EngineActorRequest>,
    /// The engine's state, to ask only for denials above its finalized head.
    engine_state_rx: watch::Receiver<EngineState>,
    /// The authority epoch the node derives under.
    pub(super) applied: AppliedEpoch,
    /// Ticker driving the polls.
    ticker: time::Interval,
    /// The last poll failure, so an authority that stays down is warned about once.
    last_failure: Option<String>,
}

impl<Authority> AuthorityActor<Authority>
where
    Authority: InteropAuthorityApiClient + Send + Sync + 'static,
{
    /// Creates an [`AuthorityActor`] polling `authority` every `poll_interval`, deriving under the
    /// `applied` epoch.
    pub fn new(
        authority: Authority,
        rollup: Arc<RollupConfig>,
        engine_request_tx: mpsc::Sender<EngineActorRequest>,
        engine_state_rx: watch::Receiver<EngineState>,
        poll_interval: Duration,
        applied: AppliedEpoch,
    ) -> Self {
        let mut ticker = time::interval(poll_interval);
        ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            authority,
            rollup,
            engine_request_tx,
            engine_state_rx,
            applied,
            ticker,
            last_failure: None,
        }
    }

    fn chain_id(&self) -> u64 {
        self.rollup.l2_chain_id.id()
    }

    /// Polls the authority once and forwards what it reports.
    ///
    /// The denied blocks go first, so the engine never promotes a block the same answer denies.
    /// They are forwarded also while the authority answers no heads, as it does until it has
    /// caught up after a start: a denial holds whatever the heads.
    async fn poll(&mut self) -> Result<(), AuthorityActorError> {
        // A block at or below the finalized head is never replaced, so older denials are not
        // asked for.
        let denied_from =
            self.engine_state_rx.borrow().sync_state.finalized_head().block_info.number + 1;
        let status = match self.authority.chain_status(self.chain_id(), denied_from).await {
            Ok(status) => status,
            Err(err) => {
                self.log_failure(&err.to_string());
                return Ok(());
            }
        };
        if self.last_failure.take().is_some() {
            info!(target: "authority", "Interop authority answers again");
        }
        if self.applied.get() != Some(status.epoch) {
            return self.undo(status).await;
        }
        debug!(target: "authority", ?status, "Polled the interop authority");

        self.send(EngineActorRequest::ProcessDeniedBlocks(status.denied)).await?;
        let (Some(verified), Some(finalized)) = (status.verified, status.finalized) else {
            debug!(target: "authority", "Interop authority answers no heads yet; holding the labels");
            return Ok(());
        };
        self.send(EngineActorRequest::ProcessVerifiedSafeHead(Box::new(verified))).await?;
        self.send(EngineActorRequest::ProcessVerifiedFinalizedHead(Box::new(finalized))).await?;
        Ok(())
    }

    /// Derives again from the finalized head under `status`'s epoch, and records the epoch once
    /// the engine moved its labels back.
    async fn undo(&self, status: VerifierStatus) -> Result<(), AuthorityActorError> {
        warn!(target: "authority", applied = ?self.applied.get(), epoch = %status.epoch, denied = ?status.denied, "Interop authority is on another epoch; deriving again from the finalized head");
        let (done, mut heard) = mpsc::channel(1);
        let request = UndoRequest { denied: status.denied, done };
        self.send(EngineActorRequest::UndoVerifications(Box::new(request))).await?;
        if !matches!(time::timeout(UNDO_TIMEOUT, heard.recv()).await, Ok(Some(()))) {
            warn!(target: "authority", epoch = %status.epoch, "The engine did not move its labels back yet; trying again on the next poll");
            return Ok(());
        }
        self.applied.apply(status.epoch);
        info!(target: "authority", epoch = %status.epoch, "Deriving under the interop authority's epoch");
        Ok(())
    }

    async fn send(&self, request: EngineActorRequest) -> Result<(), AuthorityActorError> {
        self.engine_request_tx.send(request).await.map_err(|_| AuthorityActorError::EngineClosed)
    }

    fn log_failure(&mut self, reason: &str) {
        if self.last_failure.as_deref() == Some(reason) {
            debug!(target: "authority", reason, "Interop authority still unreachable; holding the labels");
            return;
        }
        warn!(target: "authority", reason, "Interop authority unreachable; holding the labels");
        self.last_failure = Some(reason.to_owned());
    }
}

#[async_trait]
impl<Authority> NodeActor for AuthorityActor<Authority>
where
    Authority: InteropAuthorityApiClient + Send + Sync + 'static,
{
    type Error = AuthorityActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        self.ticker.tick().await;
        self.poll().await
    }
}

/// An error from the [`AuthorityActor`].
#[derive(Debug, Error)]
pub enum AuthorityActorError {
    /// The engine actor's request channel closed.
    #[error("the engine actor's request channel closed")]
    EngineClosed,
}
