//! Linear local block-production workflow.

use crate::{
    engine::{BuiltUnsafePayload, ENGINE_RETRY_DELAY, EngineClient, EngineError},
    network::{NetworkClient, NetworkClientError},
    unsafe_chain::{Conductor, OriginSelector, origin::L1OriginSelectorError},
};
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use std::{sync::Arc, time::Duration};
use tokio::time::Instant;

#[cfg(feature = "metrics")]
use crate::Metrics;

const CONDUCTOR_TIMEOUT: Duration = Duration::from_secs(30);
const PUBLICATION_TIMEOUT: Duration = Duration::from_secs(30);
#[cfg(not(test))]
const RETRY_DELAY: Duration = Duration::from_secs(1);
#[cfg(test)]
const RETRY_DELAY: Duration = Duration::from_millis(1);
#[cfg(not(test))]
const REPLAN_DELAY: Duration = Duration::from_millis(200);
#[cfg(test)]
const REPLAN_DELAY: Duration = Duration::from_millis(1);

/// Attributes for one planned local block.
#[derive(Debug)]
struct PlannedCandidate {
    attributes: OpAttributesWithParent,
}

/// A payload retrieved from the EL but not authorized for publication.
#[derive(Debug)]
struct SealedCandidate {
    built: BuiltUnsafePayload,
    attributes: OpAttributesWithParent,
    distribution_started: Instant,
}

/// A sealed candidate authorized by the configured publication gate.
#[derive(Debug)]
struct AuthorizedCandidate {
    sealed: SealedCandidate,
}

/// An authorized candidate whose gossip publication completed or may have completed.
#[derive(Debug)]
struct PublishedCandidate {
    authorized: AuthorizedCandidate,
}

/// Constructs a fresh local producer whenever sequencing starts.
#[derive(Clone)]
pub struct SequencingWorkflowFactory {
    create: Arc<dyn Fn() -> SequencingWorkflow + Send + Sync>,
    conductor: Option<Arc<dyn Conductor>>,
}

impl core::fmt::Debug for SequencingWorkflowFactory {
    fn fmt(&self, formatter: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        formatter
            .debug_struct("SequencingWorkflowFactory")
            .field("conductor_enabled", &self.conductor.is_some())
            .finish_non_exhaustive()
    }
}

impl SequencingWorkflowFactory {
    /// Creates a workflow factory around node-owned dependency construction.
    pub fn new(
        create: impl Fn() -> SequencingWorkflow + Send + Sync + 'static,
        conductor: Option<Arc<dyn Conductor>>,
    ) -> Self {
        Self { create: Arc::new(create), conductor }
    }

    /// Constructs a producer with fresh planning state.
    pub fn create(&self) -> SequencingWorkflow {
        (self.create)()
    }

    /// Returns the configured HA conductor.
    pub fn conductor(&self) -> Option<&Arc<dyn Conductor>> {
        self.conductor.as_ref()
    }
}

/// Dependencies and state for one local-production session.
#[derive(Debug)]
pub struct SequencingWorkflow {
    attributes_builder: Box<dyn AttributesBuilder + Sync>,
    conductor: Option<Arc<dyn Conductor>>,
    engine: EngineClient,
    network: NetworkClient,
    origin_selector: Box<dyn OriginSelector>,
    config: Arc<RollupConfig>,
    last_distribution_duration: Duration,
}

impl SequencingWorkflow {
    /// Creates a local producer. It is not spawned independently from the unsafe-chain service.
    pub fn new(
        attributes_builder: Box<dyn AttributesBuilder + Sync>,
        conductor: Option<Arc<dyn Conductor>>,
        engine: EngineClient,
        network: NetworkClient,
        origin_selector: Box<dyn OriginSelector>,
        config: Arc<RollupConfig>,
    ) -> Self {
        Self {
            attributes_builder,
            conductor,
            engine,
            network,
            origin_selector,
            config,
            last_distribution_duration: Duration::ZERO,
        }
    }

    /// Returns the configured conductor capability.
    pub fn conductor(&self) -> Option<&Arc<dyn Conductor>> {
        self.conductor.as_ref()
    }

    /// Produces one complete block action.
    ///
    /// Once a payload has been retrieved, this function retains that exact payload through
    /// conductor, gossip, and canonicalization retries. It intentionally does not observe stop or
    /// shutdown signals inside the action.
    pub async fn sequence_one(
        &mut self,
        recovery_mode: bool,
    ) -> Result<Option<L2BlockInfo>, SequencingError> {
        let Some(planned) = self.plan(recovery_mode).await? else {
            tokio::time::sleep(REPLAN_DELAY).await;
            return Ok(None);
        };

        self.wait_until_block_time(&planned).await;

        let distribution_started = Instant::now();
        let built = match self.engine.build_unsafe(planned.attributes.clone()).await {
            Ok(built) => built,
            Err(EngineError::Temporary(_) | EngineError::StaleBuild) => return Ok(None),
            Err(error) => return Err(SequencingError::Engine(error)),
        };
        let sealed =
            SealedCandidate { built, attributes: planned.attributes, distribution_started };

        let authorized = self.authorize_until_done(sealed).await;
        let published = self.publish_until_done(authorized).await?;
        let block = self.import_until_done(&published).await?;

        self.last_distribution_duration =
            published.authorized.sealed.distribution_started.elapsed();
        update_transaction_metrics(published.authorized.sealed.attributes.count_transactions());
        Ok(Some(block))
    }

    async fn plan(
        &mut self,
        recovery_mode: bool,
    ) -> Result<Option<PlannedCandidate>, SequencingError> {
        let unsafe_head = self.engine.state().await.map_err(SequencingError::Engine)?.unsafe_head();
        let l1_origin = match self.origin_selector.next_l1_origin(unsafe_head, recovery_mode).await
        {
            Ok(origin) => origin,
            Err(L1OriginSelectorError::OriginNotFound(hash)) => {
                warn!(target: "sequencer", %hash, "Current L1 origin is unavailable; replanning");
                return Ok(None);
            }
            Err(error) => {
                debug!(target: "sequencer", ?error, "L1 origin selection yielded");
                return Ok(None);
            }
        };

        if unsafe_head.l1_origin.hash != l1_origin.hash &&
            unsafe_head.l1_origin.hash != l1_origin.parent_hash
        {
            warn!(target: "sequencer", ?l1_origin, "Selected L1 origin is inconsistent with unsafe head");
            return Ok(None);
        }

        let attributes_started = Instant::now();
        let attributes =
            self.attributes_builder.prepare_payload_attributes(unsafe_head, l1_origin.id()).await;
        kona_macros::set!(
            gauge,
            Metrics::SEQUENCER_ATTRIBUTES_BUILDER_DURATION,
            attributes_started.elapsed()
        );
        let mut attributes = match attributes {
            Ok(attributes) => attributes,
            Err(PipelineErrorKind::Temporary(_)) => return Ok(None),
            Err(PipelineErrorKind::Reset(error)) => {
                warn!(target: "sequencer", ?error, "Sequencing plan requires reset; replanning without rewinding unsafe");
                return Ok(None);
            }
            Err(error @ PipelineErrorKind::Critical(_)) => {
                return Err(SequencingError::Attributes(error.to_string()));
            }
        };

        attributes.no_tx_pool =
            Some(!self.should_use_tx_pool(l1_origin, &attributes, recovery_mode));
        Ok(Some(PlannedCandidate {
            attributes: OpAttributesWithParent::new(attributes, unsafe_head, None, false),
        }))
    }

    async fn wait_until_block_time(&self, planned: &PlannedCandidate) {
        let timestamp =
            planned.attributes.parent().block_info.timestamp.saturating_add(self.config.block_time);
        let target = std::time::UNIX_EPOCH + Duration::from_secs(timestamp);
        let adjusted = target.checked_sub(self.last_distribution_duration).unwrap_or(target);
        let delay = adjusted.duration_since(std::time::SystemTime::now()).unwrap_or_default();
        tokio::time::sleep(delay).await;
    }

    async fn authorize_until_done(&self, sealed: SealedCandidate) -> AuthorizedCandidate {
        let Some(conductor) = self.conductor.as_ref() else {
            return AuthorizedCandidate { sealed };
        };

        loop {
            let commitment_started = Instant::now();
            let commitment = tokio::time::timeout(
                CONDUCTOR_TIMEOUT,
                conductor.commit_unsafe_payload(sealed.built.payload()),
            )
            .await;
            kona_macros::set!(
                gauge,
                Metrics::SEQUENCER_CONDUCTOR_COMMITMENT_DURATION,
                commitment_started.elapsed()
            );
            match commitment {
                Ok(Ok(())) => return AuthorizedCandidate { sealed },
                Ok(Err(error)) => {
                    error!(target: "sequencer", ?error, "Conductor rejected or failed to commit payload; retaining candidate");
                }
                Err(_) => {
                    error!(target: "sequencer", "Conductor commit timed out ambiguously; retaining candidate");
                }
            }
            tokio::time::sleep(RETRY_DELAY).await;
        }
    }

    async fn publish_until_done(
        &self,
        authorized: AuthorizedCandidate,
    ) -> Result<PublishedCandidate, SequencingError> {
        loop {
            match tokio::time::timeout(
                PUBLICATION_TIMEOUT,
                self.network.publish_unsafe(authorized.sealed.built.payload().clone()),
            )
            .await
            {
                Ok(Ok(())) => return Ok(PublishedCandidate { authorized }),
                Ok(Err(
                    error @ (NetworkClientError::Unavailable |
                    NetworkClientError::SignerUnavailable |
                    NetworkClientError::Signing(_)),
                )) => {
                    return Err(SequencingError::Network(error));
                }
                Ok(Err(error)) => {
                    warn!(target: "sequencer", ?error, "Gossip publication failed; retaining authorized candidate");
                }
                Err(_) => {
                    warn!(target: "sequencer", "Gossip acknowledgement timed out ambiguously; retaining authorized candidate");
                }
            }
            tokio::time::sleep(RETRY_DELAY).await;
        }
    }

    async fn import_until_done(
        &self,
        published: &PublishedCandidate,
    ) -> Result<L2BlockInfo, SequencingError> {
        loop {
            match self.engine.canonicalize_unsafe(published.authorized.sealed.built.clone()).await {
                Ok(block) => return Ok(block),
                Err(EngineError::Temporary(_) | EngineError::ResponseDropped) => {
                    warn!(target: "sequencer", "Local canonicalization failed ambiguously; retaining published candidate");
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(error) => return Err(SequencingError::Engine(error)),
            }
        }
    }

    fn should_use_tx_pool(
        &self,
        l1_origin: BlockInfo,
        attributes: &OpPayloadAttributes,
        recovery_mode: bool,
    ) -> bool {
        if recovery_mode {
            return false;
        }
        let timestamp = attributes.payload_attributes.timestamp;
        if timestamp > l1_origin.timestamp + self.config.max_sequencer_drift(l1_origin.timestamp) {
            return false;
        }
        !(self.config.is_first_ecotone_block(timestamp) ||
            self.config.is_first_fjord_block(timestamp) ||
            self.config.is_first_granite_block(timestamp) ||
            self.config.is_first_holocene_block(timestamp) ||
            self.config.is_first_isthmus_block(timestamp) ||
            self.config.is_first_jovian_block(timestamp) ||
            self.config.is_first_karst_block(timestamp) ||
            self.config.is_first_interop_block(timestamp))
    }
}

/// Terminal local-production error.
#[derive(Debug, thiserror::Error)]
pub enum SequencingError {
    /// Semantic engine operation failed.
    #[error(transparent)]
    Engine(#[from] EngineError),
    /// Network publication cannot make progress.
    #[error(transparent)]
    Network(#[from] NetworkClientError),
    /// Payload-attribute construction failed critically.
    #[error("payload attribute construction failed: {0}")]
    Attributes(String),
}

#[inline]
fn update_transaction_metrics(transaction_count: u64) {
    #[cfg(feature = "metrics")]
    metrics::counter!(Metrics::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED).increment(transaction_count);
    #[cfg(not(feature = "metrics"))]
    let _ = transaction_count;
}
