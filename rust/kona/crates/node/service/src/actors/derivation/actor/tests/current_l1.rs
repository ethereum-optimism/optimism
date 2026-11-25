//! The published origin, which `optimism_syncStatus` serves as `current_l1`, against the derived
//! blocks that are not local-safe yet.

use super::*;

/// A pipeline whose every L1 block holds one batch: once it has handed out a block's attributes,
/// the next step advances its origin to the next L1 block and prepares that block's attributes.
#[derive(Debug)]
struct OneBatchPerL1Block {
    origin: BlockInfo,
    prepared: Option<OpAttributesWithParent>,
    rollup_config: Arc<RollupConfig>,
}

impl OneBatchPerL1Block {
    fn at(origin: BlockInfo) -> Self {
        Self {
            origin,
            prepared: Some(Self::attributes_from(origin)),
            rollup_config: Arc::new(RollupConfig::default()),
        }
    }

    fn attributes_from(l1: BlockInfo) -> OpAttributesWithParent {
        OpAttributesWithParent { derived_from: Some(l1), ..derived_attributes() }
    }
}

impl Iterator for OneBatchPerL1Block {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        self.prepared.take()
    }
}

impl kona_derive::OriginProvider for OneBatchPerL1Block {
    fn origin(&self) -> Option<BlockInfo> {
        Some(self.origin)
    }
}

#[async_trait]
impl SignalReceiver for OneBatchPerL1Block {
    async fn signal(&mut self, _: Signal) -> PipelineResult<()> {
        Ok(())
    }
}

#[async_trait]
impl Pipeline for OneBatchPerL1Block {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        self.prepared.as_ref()
    }

    async fn step(&mut self, _: L2BlockInfo) -> StepResult {
        if self.prepared.is_some() {
            return StepResult::PreparedAttributes;
        }
        self.origin = l1_block(self.origin.number + 1);
        self.prepared = Some(Self::attributes_from(self.origin));
        StepResult::AdvancedOrigin
    }

    fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_config
    }

    async fn system_config_by_l2_hash(
        &mut self,
        _: B256,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        Ok(SystemConfig::default())
    }
}

/// An interop verifier takes a node's `current_l1` to mean that every L1 block below it is
/// derived into local-safe blocks. Derivation does not step while attributes await the engine's
/// confirmation, so the origin stays at the L1 block they were derived from however far L1 moves,
/// and passes it only once the engine confirms them. Data of the origin block itself may still be
/// on its way to the engine, as op-node documents for `current_l1`.
#[tokio::test]
async fn origin_does_not_pass_attributes_awaiting_the_engine() {
    let (origin_tx, origin) = watch::channel(None);
    let (request_tx, request_rx) = mpsc::channel(1);
    let mut actor = DerivationActor::new(
        engine_client_accepting_attributes(),
        request_rx,
        OneBatchPerL1Block::at(l1_block(DERIVED_FROM_L1)),
        None,
        l1_block(0).id(),
        origin_tx,
    );
    let mut handle = async |request| {
        request_tx.send(request).await.unwrap();
        actor.step().await.unwrap();
        (actor.derivation_state_machine.current_state(), origin.borrow().map(|block| block.number))
    };

    // The engine's reset at startup establishes the origin.
    let reset = DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
        kona_derive::ResetSignal { l2_safe_head: l2_block(RESUME_HEAD) },
    )));
    assert_eq!(
        handle(reset).await,
        (DerivationState::AwaitingELSyncCompletion, Some(DERIVED_FROM_L1))
    );
    let awaiting = (DerivationState::AwaitingSafeHeadConfirmation, Some(DERIVED_FROM_L1));
    let synced =
        DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(l2_block(RESUME_HEAD)));
    assert_eq!(handle(synced).await, awaiting);
    for l1 in DERIVED_FROM_L1 + 1..=DERIVED_FROM_L1 + 5 {
        let head = DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(l1)));
        assert_eq!(handle(head).await, awaiting, "L1 head {l1}");
    }

    let confirmed = DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::new(l2_block(
        DERIVED_HEAD,
    )));
    assert_eq!(
        handle(confirmed).await,
        (DerivationState::AwaitingSafeHeadConfirmation, Some(DERIVED_FROM_L1 + 1))
    );
}
