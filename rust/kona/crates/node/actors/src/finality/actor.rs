//! The finality actor.

use super::{ActorError, Engine};
use crate::engine::{EngineState, FinalizeBlockId};
use kona_protocol::BlockInfo;
use std::{collections::BTreeMap, future::Future};
use tokio::sync::watch;

/// Supplies the finality actor's dependencies, and produces its lifetime future without spawning
/// it.
///
/// The actor records the safe heads in `engine_state` with the L1 blocks they were derived from,
/// and asks `engine` to finalize them as `finalized` L1 blocks arrive. Initial snapshots do not
/// trigger processing. Runtime work begins when the future is polled. Dropping the future stops
/// the actor.
pub fn build<Engine_>(
    engine: Engine_,
    engine_state: watch::Receiver<EngineState>,
    finalized: watch::Receiver<BlockInfo>,
) -> impl Future<Output = Result<(), ActorError>> + Send + 'static
where
    Engine_: Engine + 'static,
{
    Actor { engine, engine_state, finalized, resets: 0, safe_heads: BTreeMap::new() }.run()
}

/// An L1 block number.
type L1BlockNumber = u64;

/// An L2 block number.
type L2BlockNumber = u64;

/// Finalizes L2 blocks once the L1 blocks they were derived from finalize.
///
/// The engine's published state coalesces, so the actor may miss some safe heads. It then
/// finalizes the highest one it saw, and catches up when the next L1 block finalizes.
#[derive(Debug)]
struct Actor<Engine_> {
    /// The engine, which finalizes blocks.
    engine: Engine_,
    /// The engine's published state, whose safe head and its source the actor records.
    engine_state: watch::Receiver<EngineState>,
    /// The latest published finalized L1 block.
    finalized: watch::Receiver<BlockInfo>,
    /// The engine's reset count the actor last saw.
    resets: u64,
    /// The highest safe L2 block derived from each L1 block, awaiting finalization.
    safe_heads: BTreeMap<L1BlockNumber, L2BlockNumber>,
}

impl<Engine_: Engine> Actor<Engine_> {
    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            // The engine publishes at most once per request, so its state cannot keep the
            // finalized L1 block waiting.
            tokio::select! {
                biased;
                result = self.engine_state.changed() => {
                    result.map_err(|_| ActorError::EngineStateClosed)?;
                    let state = *self.engine_state.borrow_and_update();
                    self.record(state);
                }
                result = self.finalized.changed() => {
                    result.map_err(|_| ActorError::FinalizedClosed)?;
                    let finalized = *self.finalized.borrow_and_update();
                    self.finalize(finalized).await?;
                }
            }
        }
    }

    /// Records the safe head and the L1 block it was derived from. A reset first drops the safe
    /// heads recorded before it, whose sources an L1 reorg may have replaced.
    fn record(&mut self, state: EngineState) {
        if state.resets() != self.resets {
            self.resets = state.resets();
            self.safe_heads.clear();
        }
        if let Some(source) = state.safe_head_source() {
            let safe_head = state.safe_head().block_info.number;
            let highest = self.safe_heads.entry(source.number).or_insert(safe_head);
            *highest = (*highest).max(safe_head);
        }
    }

    /// Finalizes the highest recorded safe head whose source the finalized L1 block covers, and
    /// forgets the safe heads it covers.
    async fn finalize(&mut self, finalized: BlockInfo) -> Result<(), ActorError> {
        let covered = self.safe_heads.range(..=finalized.number).next_back();
        let Some((_, &safe_head)) = covered else {
            return Ok(());
        };
        self.safe_heads.retain(|&source, _| source > finalized.number);
        // Finalize by number against the engine's canonical chain.
        self.engine.finalize(FinalizeBlockId::ByNumber(safe_head)).await?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{EngineClientError, EngineClientResult, engine::HeadsUpdate};
    use async_trait::async_trait;
    use kona_protocol::L2BlockInfo;
    use std::sync::{Arc, Mutex};
    use tokio::time::{self, Duration};

    /// An engine that records the blocks it is asked to finalize, or fails if asked to.
    #[derive(Debug, Default)]
    struct TestEngine {
        finalized: Mutex<Vec<FinalizeBlockId>>,
        fails: bool,
    }

    #[async_trait]
    impl Engine for Arc<TestEngine> {
        async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()> {
            if self.fails {
                return Err(EngineClientError::RequestError("closed".to_string()));
            }
            self.finalized.lock().unwrap().push(block_id);
            Ok(())
        }
    }

    /// The publishers the actor reads from.
    struct Publishers {
        engine_state: watch::Sender<EngineState>,
        finalized: watch::Sender<BlockInfo>,
    }

    /// Runs the actor against `engine` while `script` publishes to it. Returns the actor's error if
    /// it stops first.
    async fn run_during(
        engine: Arc<TestEngine>,
        script: impl AsyncFnOnce(&Publishers),
    ) -> Option<ActorError> {
        let (engine_state, engine_state_rx) = watch::channel(EngineState::default());
        let (finalized, finalized_rx) = watch::channel(BlockInfo::default());
        let publishers = Publishers { engine_state, finalized };
        tokio::select! {
            result = build(engine, engine_state_rx, finalized_rx) => Some(result.unwrap_err()),
            () = async {
                script(&publishers).await;
                settle().await;
            } => None,
        }
    }

    /// Runs the actor while `script` publishes to it, and returns the blocks it finalized.
    async fn finalized_during(script: impl AsyncFnOnce(&Publishers)) -> Vec<FinalizeBlockId> {
        let engine = Arc::new(TestEngine::default());
        assert!(run_during(engine.clone(), script).await.is_none(), "the actor stopped");
        engine.finalized.lock().unwrap().clone()
    }

    /// Lets the actor handle what was published.
    async fn settle() {
        time::sleep(Duration::from_secs(1)).await;
    }

    fn l1_block(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    /// Returns an engine state after `resets` resets, whose safe head is the L2 block with the
    /// given number, derived from the L1 block with the given number.
    fn state(safe_head: u64, source: Option<u64>, resets: u64) -> EngineState {
        let safe_head = L2BlockInfo { block_info: l1_block(safe_head), ..Default::default() };
        let state = EngineState::default()
            .apply_update(HeadsUpdate { safe_head: Some(safe_head), ..Default::default() })
            .with_safe_head_source(source.map(l1_block));
        (0..resets).fold(state, |state, _| state.count_reset())
    }

    /// Each finalized L1 block finalizes the highest safe head derived from it or an earlier L1
    /// block, once.
    #[tokio::test(start_paused = true)]
    async fn finalizes_the_highest_safe_head_a_finalized_l1_block_covers() {
        let finalized = finalized_during(async |publishers| {
            for (safe_head, source) in [(10, 5), (12, 6), (13, 6), (15, 8)] {
                publishers.engine_state.send_replace(state(safe_head, Some(source), 0));
                settle().await;
            }
            for l1 in [4, 6, 7, 8] {
                publishers.finalized.send_replace(l1_block(l1));
                settle().await;
            }
        })
        .await;

        assert_eq!(finalized, [FinalizeBlockId::ByNumber(13), FinalizeBlockId::ByNumber(15)]);
    }

    /// Safe heads published faster than the actor reads them coalesce, so only the latest is
    /// recorded.
    #[tokio::test(start_paused = true)]
    async fn coalesced_safe_heads_finalize_the_latest() {
        let finalized = finalized_during(async |publishers| {
            publishers.engine_state.send_replace(state(10, Some(5), 0));
            publishers.engine_state.send_replace(state(12, Some(6), 0));
            settle().await;
            for l1 in [5, 6] {
                publishers.finalized.send_replace(l1_block(l1));
                settle().await;
            }
        })
        .await;

        assert_eq!(finalized, [FinalizeBlockId::ByNumber(12)]);
    }

    /// Safe heads without a known source, such as a reset's, are not recorded.
    #[tokio::test(start_paused = true)]
    async fn ignores_safe_heads_without_a_source() {
        let finalized = finalized_during(async |publishers| {
            publishers.engine_state.send_replace(state(10, None, 0));
            settle().await;
            publishers.finalized.send_replace(l1_block(9));
        })
        .await;

        assert_eq!(finalized, []);
    }

    /// A reset drops the safe heads recorded before it, even when it is published together with a
    /// finalized L1 block.
    #[tokio::test(start_paused = true)]
    async fn reset_drops_recorded_safe_heads() {
        let finalized = finalized_during(async |publishers| {
            publishers.engine_state.send_replace(state(10, Some(5), 0));
            settle().await;
            publishers.engine_state.send_replace(state(8, None, 1));
            publishers.finalized.send_replace(l1_block(5));
        })
        .await;

        assert_eq!(finalized, []);
    }

    #[tokio::test(start_paused = true)]
    async fn failed_finalization_stops_the_actor() {
        let engine = Arc::new(TestEngine { fails: true, ..Default::default() });
        let error = run_during(engine, async |publishers| {
            publishers.engine_state.send_replace(state(10, Some(5), 0));
            settle().await;
            publishers.finalized.send_replace(l1_block(5));
            settle().await;
        })
        .await;

        assert!(matches!(error, Some(ActorError::Engine(_))));
    }

    #[rstest::rstest]
    #[case::engine_state(true, |error: &ActorError| matches!(error, ActorError::EngineStateClosed))]
    #[case::finalized(false, |error: &ActorError| matches!(error, ActorError::FinalizedClosed))]
    #[tokio::test]
    async fn closed_publisher_stops_the_actor(
        #[case] close_engine_state: bool,
        #[case] expected: fn(&ActorError) -> bool,
    ) {
        let (engine_state, engine_state_rx) = watch::channel(EngineState::default());
        let (finalized, finalized_rx) = watch::channel(BlockInfo::default());
        if close_engine_state {
            drop(engine_state);
        } else {
            drop(finalized);
        }

        let result = build(Arc::new(TestEngine::default()), engine_state_rx, finalized_rx).await;

        assert!(expected(&result.unwrap_err()));
    }
}
