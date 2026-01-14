//! The [`L2Finalizer`].

use kona_engine::{EngineClient, EngineTask, FinalizeTask};
use kona_protocol::{BlockInfo, OpAttributesWithParent};
use std::collections::BTreeMap;
use tokio::sync::watch;

use crate::actors::engine::actor::EngineActorState;

/// An internal type alias for L1 block numbers.
type L1BlockNumber = u64;

/// An internal type alias for L2 block numbers.
type L2BlockNumber = u64;

/// The [`L2Finalizer`] is responsible for finalizing L2 blocks derived from finalized L1 blocks.
/// It maintains a queue of derived L2 blocks that are awaiting finalization, and finalizes them
/// as new finalized L1 blocks are received.
#[derive(Debug)]
pub struct L2Finalizer {
    /// A channel that receives new finalized L1 blocks intermittently.
    finalized_l1_block_rx: watch::Receiver<Option<BlockInfo>>,
    /// A map of `L1 block number -> highest derived L2 block number` within the L1 epoch, used to
    /// track derived [`OpAttributesWithParent`] awaiting finalization. When a new finalized L1
    /// block is received, the highest L2 block whose inputs are contained within the finalized
    /// L1 chain is finalized.
    awaiting_finalization: BTreeMap<L1BlockNumber, L2BlockNumber>,
}

impl L2Finalizer {
    /// Creates a new [`L2Finalizer`] with the given channel receiver for finalized L1 blocks.
    pub const fn new(finalized_l1_block_rx: watch::Receiver<Option<BlockInfo>>) -> Self {
        Self { finalized_l1_block_rx, awaiting_finalization: BTreeMap::new() }
    }

    /// Enqueues a derived [`OpAttributesWithParent`] for finalization. When a new finalized L1
    /// block is observed that is `>=` the height of [`OpAttributesWithParent::derived_from`], the
    /// L2 block associated with the payload attributes will be finalized.
    pub fn enqueue_for_finalization(&mut self, attributes: &OpAttributesWithParent) {
        self.awaiting_finalization
            .entry(
                attributes.derived_from.map(|b| b.number).expect(
                    "Fatal: Cannot enqueue attributes for finalization that weren't derived",
                ),
            )
            .and_modify(|n| *n = (*n).max(attributes.block_number()))
            .or_insert(attributes.block_number());
    }

    /// Clears the finalization queue.
    pub fn clear(&mut self) {
        self.awaiting_finalization.clear();
    }

    /// Receives a new finalized L1 block from the channel.
    pub async fn new_finalized_block(&mut self) -> Result<(), watch::error::RecvError> {
        self.finalized_l1_block_rx.changed().await
    }

    /// Attempts to finalize any L2 blocks that the finalizer knows about and are contained within
    /// the new finalized L1 chain.
    pub(super) async fn try_finalize_next<EngineClient_: EngineClient>(
        &mut self,
        engine_state: &mut EngineActorState<EngineClient_>,
    ) {
        // If there is no finalized L1 block available in the watch channel, do nothing.
        let Some(new_finalized_l1) = *self.finalized_l1_block_rx.borrow() else {
            return;
        };

        // Find the highest safe L2 block that is contained within the finalized chain,
        // that the finalizer is aware of.
        let highest_safe = self.awaiting_finalization.range(..=new_finalized_l1.number).next_back();

        // If the highest safe block is found, enqueue a finalization task and drain the
        // queue of all L1 blocks not contained in the finalized L1 chain.
        if let Some((_, highest_safe_number)) = highest_safe {
            let task = EngineTask::Finalize(Box::new(FinalizeTask::new(
                engine_state.client.clone(),
                engine_state.rollup.clone(),
                *highest_safe_number,
            )));
            engine_state.engine.enqueue(task);

            self.awaiting_finalization.retain(|&number, _| number > new_finalized_l1.number);
        }
    }
}
