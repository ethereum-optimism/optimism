use super::handlers::{
    CrossSafeHandler, CrossUnsafeHandler, EventHandler, FinalizedHandler, InvalidationHandler,
    OriginHandler, ReplacementHandler, SafeBlockHandler, UnsafeBlockHandler,
};
use crate::{
    LogIndexer, ProcessorState,
    event::ChainEvent,
    syncnode::{BlockProvider, ManagedNodeCommand},
};
use alloy_primitives::ChainId;
use kona_interop::InteropValidator;
use kona_supervisor_storage::{
    DerivationStorage, HeadRefStorageWriter, LogStorage, StorageRewinder,
};
use std::{fmt::Debug, sync::Arc};
use tokio::sync::mpsc;
use tracing::debug;

/// Represents a task that processes chain events from a managed node.
/// It listens for events emitted by the managed node and handles them accordingly.
#[derive(Debug)]
pub struct ChainProcessor<P, W, V> {
    chain_id: ChainId,
    metrics_enabled: Option<bool>,

    // state
    state: ProcessorState,

    // Handlers for different types of chain events.
    unsafe_handler: UnsafeBlockHandler<P, W, V>,
    safe_handler: SafeBlockHandler<P, W, V>,
    origin_handler: OriginHandler<W>,
    invalidation_handler: InvalidationHandler<W>,
    replacement_handler: ReplacementHandler<P, W>,
    finalized_handler: FinalizedHandler<W>,
    cross_unsafe_handler: CrossUnsafeHandler,
    cross_safe_handler: CrossSafeHandler,
}

impl<P, W, V> ChainProcessor<P, W, V>
where
    P: BlockProvider + 'static,
    V: InteropValidator + 'static,
    W: LogStorage + DerivationStorage + HeadRefStorageWriter + StorageRewinder + 'static,
{
    /// Creates a new [`ChainProcessor`].
    pub fn new(
        validator: Arc<V>,
        chain_id: ChainId,
        log_indexer: Arc<LogIndexer<P, W>>,
        db_provider: Arc<W>,
        managed_node_sender: mpsc::Sender<ManagedNodeCommand>,
    ) -> Self {
        let unsafe_handler = UnsafeBlockHandler::new(
            chain_id,
            validator.clone(),
            db_provider.clone(),
            log_indexer.clone(),
        );

        let safe_handler = SafeBlockHandler::new(
            chain_id,
            managed_node_sender.clone(),
            db_provider.clone(),
            validator,
            log_indexer.clone(),
        );

        let origin_handler =
            OriginHandler::new(chain_id, managed_node_sender.clone(), db_provider.clone());

        let invalidation_handler =
            InvalidationHandler::new(chain_id, managed_node_sender.clone(), db_provider.clone());

        let replacement_handler =
            ReplacementHandler::new(chain_id, log_indexer, db_provider.clone());

        let finalized_handler =
            FinalizedHandler::new(chain_id, managed_node_sender.clone(), db_provider);
        let cross_unsafe_handler = CrossUnsafeHandler::new(chain_id, managed_node_sender.clone());
        let cross_safe_handler = CrossSafeHandler::new(chain_id, managed_node_sender);

        Self {
            chain_id,
            metrics_enabled: None,

            state: ProcessorState::new(),

            // Handlers for different types of chain events.
            unsafe_handler,
            safe_handler,
            origin_handler,
            invalidation_handler,
            replacement_handler,
            finalized_handler,
            cross_unsafe_handler,
            cross_safe_handler,
        }
    }

    /// Enables metrics on the database environment.
    pub fn with_metrics(mut self) -> Self {
        self.metrics_enabled = Some(true);
        super::Metrics::init(self.chain_id);
        self
    }

    /// Handles a chain event by delegating it to the appropriate handler.
    pub async fn handle_event(&mut self, event: ChainEvent) {
        let result = match event {
            ChainEvent::UnsafeBlock { block } => {
                self.unsafe_handler.handle(block, &mut self.state).await
            }
            ChainEvent::DerivedBlock { derived_ref_pair } => {
                self.safe_handler.handle(derived_ref_pair, &mut self.state).await
            }
            ChainEvent::DerivationOriginUpdate { origin } => {
                self.origin_handler.handle(origin, &mut self.state).await
            }
            ChainEvent::InvalidateBlock { block } => {
                self.invalidation_handler.handle(block, &mut self.state).await
            }
            ChainEvent::BlockReplaced { replacement } => {
                self.replacement_handler.handle(replacement, &mut self.state).await
            }
            ChainEvent::FinalizedSourceUpdate { finalized_source_block } => {
                self.finalized_handler.handle(finalized_source_block, &mut self.state).await
            }
            ChainEvent::CrossUnsafeUpdate { block } => {
                self.cross_unsafe_handler.handle(block, &mut self.state).await
            }
            ChainEvent::CrossSafeUpdate { derived_ref_pair } => {
                self.cross_safe_handler.handle(derived_ref_pair, &mut self.state).await
            }
        };

        if let Err(err) = result {
            debug!(
                target: "supervisor::chain_processor",
                chain_id = self.chain_id,
                %err,
                ?event,
                "Failed to process event"
            );
        }
    }
}
