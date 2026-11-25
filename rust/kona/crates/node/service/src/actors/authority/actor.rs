use crate::NodeActor;
use async_trait::async_trait;
use kona_engine::{EngineActorRequest, EngineState, FinalizeBlockId, VerifierHead};
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

/// The [`NodeActor`] following a cross-chain verification authority.
///
/// It polls the authority for this chain's verified and finalized heads and hands them to the
/// engine, which owns what the safe label becomes ([`kona_engine::PromoteTask`]) and whether a
/// finalization applies ([`kona_engine::FinalizeTask`]). An unreachable authority moves nothing:
/// the labels hold where they are.
#[derive(Debug)]
pub struct AuthorityActor<Authority> {
    /// The authority's client.
    authority: Authority,
    /// The rollup config of this chain.
    rollup: Arc<RollupConfig>,
    /// Where the heads go.
    engine_request_tx: mpsc::Sender<EngineActorRequest>,
    /// The engine's state, to skip finalization targets it has already passed.
    engine_state_rx: watch::Receiver<EngineState>,
    /// Ticker driving the polls.
    ticker: time::Interval,
    /// The last poll failure, so an authority that stays down is warned about once.
    last_failure: Option<String>,
}

impl<Authority> AuthorityActor<Authority>
where
    Authority: InteropAuthorityApiClient + Send + Sync + 'static,
{
    /// Creates an [`AuthorityActor`] polling `authority` every `poll_interval`.
    pub fn new(
        authority: Authority,
        rollup: Arc<RollupConfig>,
        engine_request_tx: mpsc::Sender<EngineActorRequest>,
        engine_state_rx: watch::Receiver<EngineState>,
        poll_interval: Duration,
    ) -> Self {
        let mut ticker = time::interval(poll_interval);
        ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self { authority, rollup, engine_request_tx, engine_state_rx, ticker, last_failure: None }
    }

    fn chain_id(&self) -> u64 {
        self.rollup.l2_chain_id.id()
    }

    /// Polls the authority once and forwards what it reports.
    async fn poll(&mut self) -> Result<(), AuthorityActorError> {
        let chain_id = self.chain_id();
        let (verified, finalized) = match (
            self.authority.verified_head(chain_id).await,
            self.authority.finalized_head(chain_id).await,
        ) {
            (Ok(verified), Ok(finalized)) => (verified, finalized),
            (Err(err), _) | (_, Err(err)) => {
                self.log_failure(&err.to_string());
                return Ok(());
            }
        };
        if self.last_failure.take().is_some() {
            info!(target: "authority", "Interop authority reachable again");
        }
        debug!(target: "authority", ?verified, ?finalized, "Polled the interop authority");

        self.send(EngineActorRequest::ProcessVerifiedSafeHead(Box::new(verified))).await?;
        if let Some(target) = self.finality_target(finalized) {
            self.send(EngineActorRequest::ProcessFinalizedL2Block(Box::new(target))).await?;
        }
        Ok(())
    }

    /// The block to finalize for the authority's finalized head, if it is above the engine's.
    ///
    /// Before activation the authority finalizes nothing; the L1-driven finalizer is off while
    /// an authority is followed, so finality holds until the authority is active.
    fn finality_target(&self, finalized: VerifierHead) -> Option<FinalizeBlockId> {
        let engine_finalized =
            self.engine_state_rx.borrow().sync_state.finalized_head().block_info.number;
        let target = match finalized {
            VerifierHead::PreActivation => return None,
            VerifierHead::Anchor { timestamp } => {
                FinalizeBlockId::ByNumber(self.rollup.block_number_from_timestamp(timestamp))
            }
            VerifierHead::Verified { block, .. } => FinalizeBlockId::ByHash(block),
        };
        (target.number() > engine_finalized).then_some(target)
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
