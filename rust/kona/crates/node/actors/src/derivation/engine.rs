use crate::{
    EngineClientResult,
    engine::{Consolidation, EngineState, FinalizeBlockId},
};
use async_trait::async_trait;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use std::fmt::Debug;

/// The engine, as the derivation actors use it.
#[async_trait]
pub trait Engine: Debug + Send + Sync {
    /// Resets the engine onto a starting point found from the execution layer's heads, and returns
    /// the engine state afterwards. The engine publishes the reset, for [`Self::next_reset`],
    /// before it replies.
    ///
    /// Fails with [`EngineClientError::ElSyncing`] while the execution layer may be syncing.
    ///
    /// [`EngineClientError::ElSyncing`]: crate::EngineClientError::ElSyncing
    async fn reset(&self) -> EngineClientResult<EngineState>;

    /// Consolidates the safe head with attributes derived from L1, and returns the engine state
    /// afterwards.
    async fn consolidate(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<Consolidation>;

    /// Waits for the engine's next reset, by any actor, and returns its state after the reset.
    ///
    /// Each reset is returned once, starting with the first reset after this engine was created.
    /// Cancelling the wait loses no reset.
    async fn next_reset(&mut self) -> EngineClientResult<EngineState>;

    /// Asks the engine to finalize the L2 block identified by the provided [`FinalizeBlockId`],
    /// without waiting for the engine to do so.
    async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()>;

    /// Asks the engine to follow a safe head supplied by the derivation delegate, without waiting
    /// for the engine to do so.
    async fn follow_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()>;
}

#[cfg(test)]
pub(crate) use test_engine::{Request, TestEngine};

#[cfg(test)]
mod test_engine {
    use super::*;
    use crate::EngineClientError;
    use std::{
        collections::VecDeque,
        fmt,
        sync::{Arc, Mutex},
    };
    use tokio::sync::watch;

    /// A request the [`TestEngine`] received.
    #[derive(Debug, Clone, PartialEq)]
    #[allow(clippy::large_enum_variant)]
    pub(crate) enum Request {
        Reset,
        Consolidate(OpAttributesWithParent),
        Finalize(FinalizeBlockId),
        FollowSafeHead(L2BlockInfo),
    }

    /// Replies to a consolidation, given the publisher of the engine's resets.
    type ConsolidationReply = Box<dyn FnOnce(&watch::Sender<EngineState>) -> Consolidation + Send>;

    /// An engine that replies from a script and records the requests it receives.
    ///
    /// Like the engine actor, it publishes each reset before replying. Clones share the script and
    /// the requests. A script that runs out panics.
    #[derive(Debug, Clone)]
    pub(crate) struct TestEngine {
        script: Arc<Script>,
        resets: watch::Receiver<EngineState>,
    }

    #[derive(Default)]
    struct Script {
        resets_tx: watch::Sender<EngineState>,
        resets: Mutex<VecDeque<EngineClientResult<EngineState>>>,
        consolidations: Mutex<VecDeque<ConsolidationReply>>,
        requests: Mutex<Vec<Request>>,
    }

    impl fmt::Debug for Script {
        fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
            f.debug_struct("Script").field("requests", &self.requests).finish_non_exhaustive()
        }
    }

    impl TestEngine {
        /// Creates an engine that replies to resets with the given results, in order.
        pub(crate) fn new(
            resets: impl IntoIterator<Item = EngineClientResult<EngineState>>,
        ) -> Self {
            let script =
                Script { resets: Mutex::new(resets.into_iter().collect()), ..Default::default() };
            let resets = script.resets_tx.subscribe();
            Self { script: Arc::new(script), resets }
        }

        /// Replies to the next unscripted consolidation by running `reply`, which may publish a
        /// reset.
        pub(crate) fn on_consolidation(
            &self,
            reply: impl FnOnce(&watch::Sender<EngineState>) -> Consolidation + Send + 'static,
        ) {
            self.script.consolidations.lock().unwrap().push_back(Box::new(reply));
        }

        /// Publishes a reset another actor made.
        pub(crate) fn publish_reset(&self, state: EngineState) {
            self.script.resets_tx.send_replace(state);
        }

        /// Returns the requests received so far, in order.
        pub(crate) fn requests(&self) -> Vec<Request> {
            self.script.requests.lock().unwrap().clone()
        }

        fn record(&self, request: Request) {
            self.script.requests.lock().unwrap().push(request);
        }
    }

    #[async_trait]
    impl Engine for TestEngine {
        async fn reset(&self) -> EngineClientResult<EngineState> {
            self.record(Request::Reset);
            let result = self.script.resets.lock().unwrap().pop_front().expect("unscripted reset");
            if let Ok(state) = &result {
                self.publish_reset(*state);
            }
            result
        }

        async fn consolidate(
            &self,
            attributes: OpAttributesWithParent,
        ) -> EngineClientResult<Consolidation> {
            self.record(Request::Consolidate(attributes));
            let reply = self
                .script
                .consolidations
                .lock()
                .unwrap()
                .pop_front()
                .expect("unscripted consolidation");
            Ok(reply(&self.script.resets_tx))
        }

        async fn next_reset(&mut self) -> EngineClientResult<EngineState> {
            self.resets
                .changed()
                .await
                .map_err(|_| EngineClientError::RequestError("resets closed".to_string()))?;
            Ok(*self.resets.borrow_and_update())
        }

        async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()> {
            self.record(Request::Finalize(block_id));
            Ok(())
        }

        async fn follow_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()> {
            self.record(Request::FollowSafeHead(safe_head));
            Ok(())
        }
    }
}
