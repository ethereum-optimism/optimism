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
    /// the engine state afterwards.
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

    /// Returns the engine's latest published state.
    fn state(&self) -> EngineState;

    /// Waits until the engine has reset more than `resets` times, and returns its state then.
    /// Returns at once if the engine already has.
    async fn reset_after(&self, resets: u64) -> EngineClientResult<EngineState>;

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

    /// Replies to a consolidation, given the engine's state publisher.
    type ConsolidationReply = Box<dyn FnOnce(&watch::Sender<EngineState>) -> Consolidation + Send>;

    /// An engine that replies from a script and records the requests it receives.
    ///
    /// Like the engine actor, it publishes the state it replies with. A script runs out with a
    /// panic.
    #[derive(Default)]
    pub(crate) struct TestEngine {
        /// The engine's published state.
        pub(crate) published: watch::Sender<EngineState>,
        resets: Mutex<VecDeque<EngineClientResult<EngineState>>>,
        consolidations: Mutex<VecDeque<ConsolidationReply>>,
        requests: Mutex<Vec<Request>>,
    }

    impl TestEngine {
        /// Creates an engine that replies to resets with the given results, in order.
        pub(crate) fn new(
            resets: impl IntoIterator<Item = EngineClientResult<EngineState>>,
        ) -> Arc<Self> {
            Arc::new(Self {
                resets: Mutex::new(resets.into_iter().collect()),
                ..Default::default()
            })
        }

        /// Replies to the next unscripted consolidation with the given reply.
        pub(crate) fn consolidates(&self, consolidation: Consolidation) {
            self.on_consolidation(move |state| {
                state.send_replace(consolidation.state);
                consolidation
            });
        }

        /// Replies to the next unscripted consolidation by running `reply`, which publishes the
        /// engine's state itself.
        pub(crate) fn on_consolidation(
            &self,
            reply: impl FnOnce(&watch::Sender<EngineState>) -> Consolidation + Send + 'static,
        ) {
            self.consolidations.lock().unwrap().push_back(Box::new(reply));
        }

        /// Returns the requests received so far, in order.
        pub(crate) fn requests(&self) -> Vec<Request> {
            self.requests.lock().unwrap().clone()
        }

        fn record(&self, request: Request) {
            self.requests.lock().unwrap().push(request);
        }
    }

    impl fmt::Debug for TestEngine {
        fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
            f.debug_struct("TestEngine").field("requests", &self.requests).finish_non_exhaustive()
        }
    }

    #[async_trait]
    impl Engine for Arc<TestEngine> {
        async fn reset(&self) -> EngineClientResult<EngineState> {
            self.record(Request::Reset);
            let result = self.resets.lock().unwrap().pop_front().expect("unscripted reset");
            if let Ok(state) = &result {
                self.published.send_replace(*state);
            }
            result
        }

        async fn consolidate(
            &self,
            attributes: OpAttributesWithParent,
        ) -> EngineClientResult<Consolidation> {
            self.record(Request::Consolidate(attributes));
            let reply =
                self.consolidations.lock().unwrap().pop_front().expect("unscripted consolidation");
            Ok(reply(&self.published))
        }

        fn state(&self) -> EngineState {
            *self.published.borrow()
        }

        async fn reset_after(&self, resets: u64) -> EngineClientResult<EngineState> {
            let mut state = self.published.subscribe();
            let state = state.wait_for(|state| state.resets() > resets).await.unwrap();
            Ok(*state)
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
