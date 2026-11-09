use crate::{
    DerivationActorRequest, DerivationEngineClient, Metrics, NodeActor,
    actors::derivation::{DerivationDelegateProvider, DerivationError},
};
use alloy_primitives::BlockHash;
use async_trait::async_trait;
use kona_derive::{ChainProvider, Signal};
use kona_engine::FinalizeBlockId;
use kona_protocol::{BlockInfo, L2BlockInfo, SyncStatus};
use kona_providers_alloy::{AlloyChainProvider, AlloyChainProviderError};
use thiserror::Error;
use tokio::{
    select,
    sync::{mpsc, watch},
    time,
};

/// The [`NodeActor`] for the delegate derivation sub-routine.
///
/// This actor is responsible for receiving messages from [`NodeActor`]s and polls
/// an external derivation delegation provider for derivation state. It validates
/// the canonicality of the L1 information associated with delegated derivation
/// results against the canonical L1 chain before forwarding updates.
///
/// Once validated, the actor sends the derived safe and finalized L2 info
/// to the [`NodeActor`] responsible for the execution sub-routine.
#[derive(Debug)]
pub struct DelegateDerivationActor<DerivationEngineClient_, DelegateProvider>
where
    DerivationEngineClient_: DerivationEngineClient,
    DelegateProvider: DerivationDelegateProvider,
{
    /// The channel on which all inbound requests are received by the [`DelegateDerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// Derivation delegate provider.
    derivation_delegate_provider: DelegateProvider,
    /// L1 provider for validating L1 info for derivation delegation.
    ///
    /// It must not have a confirmation depth: a confirmation-depth miss is also reported as
    /// [`AlloyChainProviderError::BlockNotFound`], which the `current_l1` check tolerates.
    l1_provider: AlloyChainProvider,
    /// Publishes the delegate's `current_l1`: the L1 block that its derivation has reached.
    derivation_origin_tx: watch::Sender<Option<BlockInfo>>,

    /// The engine's L2 safe head, according to updates from the Engine.
    engine_l2_safe_head: L2BlockInfo,
    /// Whether the engine sync has completed. This will only ever go from false -> true.
    has_engine_sync_completed: bool,
    /// Ticker driving periodic polls of the derivation delegate provider.
    delegated_derivation_ticker: time::Interval,
}

impl<DerivationEngineClient_, DelegateProvider>
    DelegateDerivationActor<DerivationEngineClient_, DelegateProvider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
{
    /// Creates a new instance of the [`DelegateDerivationActor`].
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: AlloyChainProvider,
        derivation_origin_tx: watch::Sender<Option<BlockInfo>>,
    ) -> Self {
        let mut delegated_derivation_ticker =
            time::interval(Self::DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            inbound_request_rx,
            engine_client,
            derivation_delegate_provider,
            l1_provider,
            derivation_origin_tx,
            engine_l2_safe_head: L2BlockInfo::default(),
            has_engine_sync_completed: false,
            delegated_derivation_ticker,
        }
    }
}

#[async_trait]
impl<DerivationEngineClient_, DelegateProvider> NodeActor
    for DelegateDerivationActor<DerivationEngineClient_, DelegateProvider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
{
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        select! {
            biased;

            req = self.inbound_request_rx.recv() => {
                let request = req.ok_or_else(|| {
                    error!(
                        target: "derivation",
                        "DerivationActor inbound request receiver closed unexpectedly",
                    );
                    DerivationError::RequestReceiveFailed
                })?;
                self.handle_derivation_delegation_actor_request(request).await
            }
            _ = self.delegated_derivation_ticker.tick(),
            if self.has_engine_sync_completed => {
                self.fetch_and_apply_delegate_safe_head().await
            }
        }
    }
}

impl<DerivationEngineClient_, DelegateProvider>
    DelegateDerivationActor<DerivationEngineClient_, DelegateProvider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
{
    /// Hardcoded poll interval for Derivation Delegation
    const DERIVATION_DELEGATE_POLL_INTERVAL: std::time::Duration =
        std::time::Duration::from_secs(4);

    /// Validates a single L1 block height and hash against the canonical L1 chain.
    async fn validate_l1_block(
        &mut self,
        context: &str,
        l1_block_number: u64,
        expected_hash: BlockHash,
    ) -> Result<(), DerivationDelegationError> {
        let block =
            self.l1_provider.block_info_by_number(l1_block_number).await.map_err(|e| match e {
                AlloyChainProviderError::BlockNotFound(_) => {
                    DerivationDelegationError::BlockNotFound {
                        context: context.to_string(),
                        number: l1_block_number,
                    }
                }
                e => DerivationDelegationError::Provider(e),
            })?;

        if block.hash != expected_hash {
            return Err(DerivationDelegationError::HashMismatch {
                context: context.to_string(),
                number: l1_block_number,
                expected: expected_hash,
                actual: block.hash,
            });
        }

        Ok(())
    }

    /// Verifies that the L1 info reported by the derivation delegate is consistent with the
    /// canonical L1 chain.
    ///
    /// Returns an error if the sync status must not be applied. If only `current_l1` is not on
    /// our L1 chain yet, the heads can still be applied.
    async fn validate_sync_status(
        &mut self,
        v: &SyncStatus,
    ) -> Result<CurrentL1Check, DerivationDelegationError> {
        self.validate_l1_block(
            "L1 Origin of Safe L2",
            v.safe_l2.l1_origin.number,
            v.safe_l2.l1_origin.hash,
        )
        .await?;
        self.validate_l1_block(
            "L1 Origin of Finalized L2",
            v.finalized_l2.l1_origin.number,
            v.finalized_l2.l1_origin.hash,
        )
        .await?;

        match self.validate_l1_block("Current L1", v.current_l1.number, v.current_l1.hash).await {
            Ok(()) => Ok(CurrentL1Check::Verified),
            // Our L1 provider is behind the delegate's: it does not have this block yet, for
            // example while a load-balanced L1 endpoint serves from a lagging backend. The L1
            // origins of the heads are verified, so the heads can be applied. The L1 blocks
            // between the safe head's origin and `current_l1` stay unverified until a later poll
            // finds a `current_l1` on our chain. If the delegate is on an L1 fork, that poll
            // rejects the status with a hash mismatch, but heads already applied are not
            // reverted.
            Err(err @ DerivationDelegationError::BlockNotFound { .. }) => {
                info!(
                    target: "derivation",
                    error = %err,
                    "Delegate current L1 not found on our L1 chain yet, not updating derivation origin"
                );
                metrics::counter!(Metrics::DERIVATION_DELEGATE_CURRENT_L1_NOT_FOUND).increment(1);
                Ok(CurrentL1Check::NotFound)
            }
            Err(err) => Err(err),
        }
    }

    /// Fetches, validates, and applies sync status from the derivation delegate.
    async fn fetch_and_apply_delegate_safe_head(&mut self) -> Result<(), DerivationError> {
        let sync_status = match self.derivation_delegate_provider.fetch_sync_status().await {
            Ok(status) => status,
            Err(_) => {
                warn!(target: "derivation", "Failed to fetch sync status from delegate");
                return Ok(());
            }
        };

        match self.validate_sync_status(&sync_status).await {
            Ok(CurrentL1Check::Verified) => {
                self.derivation_origin_tx.send_replace(Some(sync_status.current_l1));
            }
            Ok(CurrentL1Check::NotFound) => {}
            Err(err) => {
                // Validation failures here are expected to be transient, so we skip processing
                // this sync status and continue delegating derivation instead of treating it as
                // fatal.
                warn!(
                    target: "derivation",
                    error = %err,
                    "L1 inconsistency detected at sync status from delegate"
                );
                return Ok(());
            }
        }

        self.engine_client
            .send_safe_l2_signal(sync_status.safe_l2.into())
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        // Delegated polling supplies `(number, hash)`. Carry the hash through so the engine
        // finalizes the specific block we were asked to, not whatever it happens to have at the
        // same height.
        self.engine_client
            .send_finalized_l2_block(FinalizeBlockId::ByHash(
                sync_status.finalized_l2.block_info.id(),
            ))
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        debug!(
            target: "derivation",
            safe_l2 = ?sync_status.safe_l2,
            finalized_l2 = ?sync_status.finalized_l2,
            "Processed sync status from delegate"
        );

        Ok(())
    }

    async fn handle_derivation_delegation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(safe_head) => {
                debug!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
                self.engine_l2_safe_head = *safe_head;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.engine_l2_safe_head = *safe_head;
                self.has_engine_sync_completed = true;
            }
            DerivationActorRequest::ProcessEngineSignalRequest(signal) => match *signal {
                Signal::Reset(reset) => self.engine_l2_safe_head = reset.l2_safe_head,
                signal => {
                    debug!(target: "derivation", ?signal, "Ignoring signal while derivation delegation");
                }
            },
            DerivationActorRequest::ProcessFinalizedL1Block(_) |
            DerivationActorRequest::ProcessL1HeadUpdateRequest(_) => {
                debug!(target: "derivation", "Ignoring request while derivation delegation: {:?}", request_type);
            }
        }
        Ok(())
    }
}

/// Whether a delegate's `current_l1` is on our L1 chain, for a sync status whose heads'
/// L1 origins are.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum CurrentL1Check {
    /// `current_l1` is on our L1 chain.
    Verified,
    /// Our L1 provider does not have `current_l1` yet. The heads can be applied, but the
    /// derivation origin must not move.
    NotFound,
}

#[derive(Error, Debug)]
enum DerivationDelegationError {
    /// The L1 provider returned an error (network, RPC, etc.)
    #[error("L1 provider error: {0}")]
    Provider(#[source] AlloyChainProviderError),

    /// The L1 provider does not have a block at this height.
    #[error("L1 block not found in {context} at block {number}")]
    BlockNotFound { context: String, number: u64 },

    /// The hash provided by the derivation delegation does not match the canonical chain.
    #[error("L1 inconsistency in {context} at block {number}: expected {expected}, got {actual}")]
    HashMismatch { context: String, number: u64, expected: BlockHash, actual: BlockHash },
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        DerivationDelegateClientError,
        actors::derivation::engine_client::MockDerivationEngineClient,
    };
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
    use alloy_transport::mock::Asserter;
    use rstest::rstest;

    /// A delegate that always reports the same sync status.
    struct FixedDelegate(SyncStatus);

    #[async_trait]
    impl DerivationDelegateProvider for FixedDelegate {
        async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError> {
            Ok(self.0.clone())
        }
    }

    /// What our L1 provider answers when asked for a block by number.
    enum L1Answer {
        Block(Box<Block>),
        NotFound,
        Error,
    }

    fn l1_block(number: u64, timestamp: u64) -> Block {
        Block::new(
            Header::new(alloy_consensus::Header { number, timestamp, ..Default::default() }),
            BlockTransactions::Hashes(vec![]),
        )
    }

    fn l2_block(number: u64, l1_origin: &Block) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { number, ..Default::default() },
            l1_origin: BlockInfo::from(l1_origin.clone()).id(),
            seq_num: 0,
        }
    }

    fn sync_status(
        safe_origin: &Block,
        finalized_origin: &Block,
        current_l1: &Block,
    ) -> SyncStatus {
        SyncStatus {
            current_l1: current_l1.clone().into(),
            current_l1_finalized: BlockInfo::default(),
            head_l1: BlockInfo::default(),
            safe_l1: BlockInfo::default(),
            finalized_l1: BlockInfo::default(),
            unsafe_l2: l2_block(3, safe_origin),
            safe_l2: l2_block(2, safe_origin),
            finalized_l2: l2_block(1, finalized_origin),
            local_safe_l2: l2_block(2, safe_origin),
        }
    }

    /// Runs one poll of the delegate. Our L1 provider answers the lookups in order: the safe
    /// head's L1 origin, the finalized head's L1 origin, then `current_l1`. Expects the heads to
    /// be sent to the engine exactly once if `heads_applied`, and never otherwise. Returns the
    /// published derivation origin and the number of answers left unread.
    async fn poll_once(
        status: SyncStatus,
        answers: Vec<L1Answer>,
        heads_applied: bool,
    ) -> (Option<BlockInfo>, usize) {
        let asserter = Asserter::new();
        for answer in answers {
            match answer {
                L1Answer::Block(block) => asserter.push_success(&*block),
                L1Answer::NotFound => asserter.push_success(&Option::<Block>::None),
                L1Answer::Error => asserter.push_failure_msg("connection refused"),
            }
        }
        let l1_provider =
            AlloyChainProvider::new(RootProvider::new(RpcClient::mocked(asserter.clone())), 16);

        let mut engine_client = MockDerivationEngineClient::new();
        engine_client
            .expect_send_safe_l2_signal()
            .times(usize::from(heads_applied))
            .returning(|_| Ok(()));
        engine_client
            .expect_send_finalized_l2_block()
            .times(usize::from(heads_applied))
            .returning(|_| Ok(()));

        let (_request_tx, request_rx) = mpsc::channel(1);
        let (origin_tx, origin_rx) = watch::channel(None);
        let mut actor = DelegateDerivationActor::new(
            engine_client,
            request_rx,
            FixedDelegate(status),
            l1_provider,
            origin_tx,
        );

        actor.fetch_and_apply_delegate_safe_head().await.unwrap();
        let origin = *origin_rx.borrow();
        (origin, asserter.read_q().len())
    }

    /// What our L1 provider has at the height of the delegate's `current_l1`.
    enum OurCurrentL1 {
        Same,
        OtherBlock,
        NotFound,
        Error,
    }

    #[rstest]
    #[case::verified(OurCurrentL1::Same, true, true)]
    #[case::not_found_on_our_l1_yet(OurCurrentL1::NotFound, true, false)]
    #[case::lookup_fails(OurCurrentL1::Error, false, false)]
    #[case::not_on_our_l1_chain(OurCurrentL1::OtherBlock, false, false)]
    #[tokio::test]
    async fn current_l1_check_only_soft_fails_when_not_found(
        #[case] ours: OurCurrentL1,
        #[case] heads_applied: bool,
        #[case] origin_published: bool,
    ) {
        let safe_origin = l1_block(2, 0);
        let finalized_origin = l1_block(1, 0);
        let current_l1 = l1_block(4, 0);
        let current_l1_answer = match ours {
            OurCurrentL1::Same => L1Answer::Block(Box::new(current_l1.clone())),
            OurCurrentL1::OtherBlock => L1Answer::Block(Box::new(l1_block(4, 1))),
            OurCurrentL1::NotFound => L1Answer::NotFound,
            OurCurrentL1::Error => L1Answer::Error,
        };
        let answers = vec![
            L1Answer::Block(Box::new(safe_origin.clone())),
            L1Answer::Block(Box::new(finalized_origin.clone())),
            current_l1_answer,
        ];

        let (origin, unread) = poll_once(
            sync_status(&safe_origin, &finalized_origin, &current_l1),
            answers,
            heads_applied,
        )
        .await;

        assert_eq!(origin, origin_published.then(|| current_l1.into()));
        assert_eq!(unread, 0);
    }

    #[tokio::test]
    async fn head_origin_not_found_rejects_the_sync_status() {
        let safe_origin = l1_block(2, 0);
        let finalized_origin = l1_block(1, 0);
        let current_l1 = l1_block(4, 0);
        let answers = vec![
            L1Answer::NotFound,
            L1Answer::Block(Box::new(finalized_origin.clone())),
            L1Answer::Block(Box::new(current_l1.clone())),
        ];

        let (origin, unread) =
            poll_once(sync_status(&safe_origin, &finalized_origin, &current_l1), answers, false)
                .await;

        assert_eq!(origin, None);
        assert_eq!(unread, 2, "nothing past the safe head's L1 origin is looked up");
    }
}
