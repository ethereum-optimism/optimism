//! The L1 actor.

use super::{
    Config,
    follower::Follower,
    handle::{ChangeWaiter, Message, Status, Update},
    rpc::poll,
    view::View,
};
use alloy_eips::BlockNumberOrTag;
use alloy_provider::Provider;
use futures::StreamExt;
use kona_protocol::BlockInfo;
use std::num::NonZeroUsize;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot};

/// A fatal error from the L1 actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// A block stream ended.
    #[error("block stream ended")]
    StreamEnded,
}

/// Runs the L1 actor until a block stream ends.
///
/// The view starts at the finalized block, so requests wait until L1 reports one.
pub(super) async fn run<L1Provider: Provider>(
    provider: L1Provider,
    config: Config,
    mut messages: mpsc::Receiver<Message>,
) -> Result<(), ActorError> {
    let mut finalized = poll(&provider, BlockNumberOrTag::Finalized, config.finalized_interval);
    let mut safe = poll(&provider, BlockNumberOrTag::Safe, config.head_interval);
    let mut heads = poll(&provider, BlockNumberOrTag::Latest, config.head_interval);
    let view = View::new(finalized.next().await.ok_or(ActorError::StreamEnded)?);
    let mut follower = Follower::new(provider, view.head());
    let mut actor = Actor::new(view);
    loop {
        tokio::select! {
            head = heads.next() => {
                follower.on_head(head.ok_or(ActorError::StreamEnded)?, &mut actor.view);
                actor.answer_waiters();
            }
            Some(fetched) = follower.next_fetched() => {
                follower.on_fetched(fetched, &mut actor.view);
                actor.answer_waiters();
            }
            block = safe.next() => actor.safe = block.ok_or(ActorError::StreamEnded)?,
            block = finalized.next() => actor.finalize(block.ok_or(ActorError::StreamEnded)?),
            Some(message) = messages.recv() => actor.handle(message),
        }
    }
}

/// The actor's view of L1, and the requests waiting for it to change.
#[derive(Debug)]
struct Actor {
    view: View,
    safe: BlockInfo,
    update_waiters: Vec<UpdateWaiter>,
    head_waiters: Vec<ChangeWaiter>,
    finalized_waiters: Vec<ChangeWaiter>,
}

/// A request waiting for the view to change.
#[derive(Debug)]
struct UpdateWaiter {
    cursor: BlockInfo,
    max: NonZeroUsize,
    reply: oneshot::Sender<Update>,
}

impl Actor {
    fn new(view: View) -> Self {
        Self {
            view,
            safe: BlockInfo::default(),
            update_waiters: Vec::new(),
            head_waiters: Vec::new(),
            finalized_waiters: Vec::new(),
        }
    }

    /// Moves the view's start to a newly finalized block.
    fn finalize(&mut self, finalized: BlockInfo) {
        if self.view.finalize(finalized) {
            info!(target: "l1", ?finalized, "L1 view restarted at a finalized block it did not hold");
        }
        self.answer_waiters();
    }

    fn handle(&mut self, message: Message) {
        match message {
            Message::UpdatesSince { cursor, max, reply } => {
                match self.view.updates_since(cursor, max) {
                    Some(update) => {
                        let _ = reply.send(update);
                    }
                    None => park(
                        &mut self.update_waiters,
                        UpdateWaiter { cursor, max, reply },
                        |waiter| waiter.reply.is_closed(),
                    ),
                }
            }
            Message::NextHead(waiter) => {
                park(&mut self.head_waiters, waiter, |waiter| waiter.reply.is_closed());
                answer_changes(&mut self.head_waiters, self.view.head());
            }
            Message::NextFinalized(waiter) => {
                park(&mut self.finalized_waiters, waiter, |waiter| waiter.reply.is_closed());
                answer_changes(&mut self.finalized_waiters, self.view.finalized());
            }
            Message::Status(reply) => {
                let _ = reply.send(Status {
                    head: self.view.head(),
                    safe: self.safe,
                    finalized: self.view.finalized(),
                });
            }
        }
    }

    /// Answers the waiting requests that now have something to report.
    fn answer_waiters(&mut self) {
        for waiter in std::mem::take(&mut self.update_waiters) {
            if waiter.reply.is_closed() {
                continue;
            }
            match self.view.updates_since(waiter.cursor, waiter.max) {
                Some(update) => {
                    let _ = waiter.reply.send(update);
                }
                None => self.update_waiters.push(waiter),
            }
        }
        answer_changes(&mut self.head_waiters, self.view.head());
        answer_changes(&mut self.finalized_waiters, self.view.finalized());
    }
}

/// Replies to the waiters whose block changed to `current`, and keeps the others.
fn answer_changes(waiters: &mut Vec<ChangeWaiter>, current: BlockInfo) {
    for waiter in std::mem::take(waiters) {
        if waiter.reply.is_closed() {
            continue;
        }
        if current == waiter.seen {
            waiters.push(waiter);
        } else {
            let _ = waiter.reply.send(current);
        }
    }
}

/// Parks a waiter, dropping the ones whose requester stopped waiting.
fn park<T>(waiters: &mut Vec<T>, waiter: T, abandoned: impl Fn(&T) -> bool) {
    waiters.retain(|waiter| !abandoned(waiter));
    waiters.push(waiter);
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::{
        Builder, Capacity, Handle,
        test_utils::{MAX, block, chain},
    };
    use alloy_json_rpc::{ErrorPayload, RequestPacket, Response, ResponsePacket, ResponsePayload};
    use alloy_primitives::B256;
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::Block;
    use alloy_transport::TransportErrorKind;
    use std::{
        collections::HashMap,
        sync::{
            Arc, Mutex,
            atomic::{AtomicBool, AtomicUsize, Ordering},
        },
        time::Duration,
    };
    use tokio::{
        sync::Notify,
        task::JoinHandle,
        time::{self, timeout},
    };

    const INTERVAL: Duration = Duration::from_secs(4);

    /// An L1 RPC serving a block tree and the blocks its tags point at.
    #[derive(Default)]
    struct RpcState {
        blocks: Mutex<HashMap<B256, BlockInfo>>,
        tags: Mutex<HashMap<&'static str, BlockInfo>>,
        by_hash_calls: AtomicUsize,
        terminal: AtomicBool,
        stall: AtomicBool,
        cancelled: Notify,
        /// Fails polls for blocks by tag.
        fail_polls: AtomicBool,
        /// Fails fetches of blocks by hash.
        fail_by_hash: AtomicBool,
        /// Holds fetches of blocks by hash until `release`.
        hold_by_hash: AtomicBool,
        release: Notify,
    }

    impl RpcState {
        fn new(head: BlockInfo) -> Arc<Self> {
            let state = Arc::new(Self::default());
            state.set_head(head);
            state.set("safe", head);
            state.set("finalized", head);
            state
        }

        /// Makes `block` the head, and adds it to the tree.
        fn set_head(&self, block: BlockInfo) {
            self.set("latest", block);
        }

        fn set(&self, tag: &'static str, block: BlockInfo) {
            self.add([block]);
            self.tags.lock().unwrap().insert(tag, block);
        }

        fn add(&self, blocks: impl IntoIterator<Item = BlockInfo>) {
            let mut tree = self.blocks.lock().unwrap();
            tree.extend(blocks.into_iter().map(|block| (block.hash, block)));
        }
    }

    fn rpc_block(info: BlockInfo) -> Block {
        let mut block: Block = Block::default();
        block.header.hash = info.hash;
        block.header.inner.number = info.number;
        block.header.inner.parent_hash = info.parent_hash;
        block.header.inner.timestamp = info.timestamp;
        block
    }

    fn provider(state: Arc<RpcState>) -> RootProvider {
        let transport = tower::service_fn(
            move |request: RequestPacket| -> alloy_transport::TransportFut<'static> {
                let state = state.clone();
                Box::pin(async move {
                    let RequestPacket::Single(request) = request else {
                        panic!("unexpected batch")
                    };
                    let params: serde_json::Value =
                        serde_json::from_str(request.params().unwrap().get()).unwrap();
                    if state.stall.load(Ordering::SeqCst) {
                        struct OnDrop<'a>(&'a Notify);
                        impl Drop for OnDrop<'_> {
                            fn drop(&mut self) {
                                self.0.notify_one();
                            }
                        }
                        let _on_drop = OnDrop(&state.cancelled);
                        std::future::pending::<()>().await;
                    }
                    let block = match request.method() {
                        "eth_getBlockByNumber" => {
                            if state.fail_polls.load(Ordering::SeqCst) {
                                return Err(TransportErrorKind::custom_str("L1 unavailable"));
                            }
                            let tag = params[0].as_str().unwrap();
                            state.tags.lock().unwrap().get(tag).copied()
                        }
                        "eth_getBlockByHash" => {
                            state.by_hash_calls.fetch_add(1, Ordering::SeqCst);
                            if state.hold_by_hash.load(Ordering::SeqCst) {
                                state.release.notified().await;
                            }
                            if state.fail_by_hash.load(Ordering::SeqCst) {
                                return Err(TransportErrorKind::custom_str("L1 unavailable"));
                            }
                            let hash: B256 = serde_json::from_value(params[0].clone()).unwrap();
                            state.blocks.lock().unwrap().get(&hash).copied()
                        }
                        method => panic!("unexpected method {method}"),
                    };
                    let payload = if state.terminal.load(Ordering::SeqCst) {
                        ResponsePayload::Failure(ErrorPayload {
                            code: -32000,
                            message: "filter not found".into(),
                            data: None,
                        })
                    } else {
                        ResponsePayload::Success(
                            serde_json::value::to_raw_value(&block.map(rpc_block)).unwrap(),
                        )
                    };
                    Ok::<_, alloy_transport::TransportError>(ResponsePacket::Single(Response {
                        id: request.id().clone(),
                        payload,
                    }))
                })
            },
        );
        RootProvider::new(RpcClient::new(transport, false))
    }

    const CONFIG: Config = Config { head_interval: INTERVAL, finalized_interval: INTERVAL };

    fn spawn(state: Arc<RpcState>) -> (Handle, JoinHandle<Result<(), ActorError>>) {
        let builder = Builder::new(Capacity::try_from(16).unwrap());
        let handle = builder.handle();
        (handle, tokio::spawn(builder.build(provider(state), CONFIG)))
    }

    /// Waits until the actor's head is `target`.
    async fn head_reaches(handle: &Handle, target: BlockInfo) {
        let mut head = BlockInfo::default();
        while head != target {
            head = handle.next_head(head).await.unwrap();
        }
    }

    #[tokio::test(start_paused = true)]
    async fn reports_advances_including_gaps_and_waits_for_new_heads() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        assert_eq!(head, block(10, 1, 1));

        // Nothing after the head yet.
        let pending = handle.updates_since(head, MAX);
        assert!(timeout(INTERVAL * 3, pending).await.is_err());

        // The head jumps two blocks; the actor fetches the one between.
        state.add([block(11, 1, 1)]);
        state.set_head(block(12, 1, 1));
        assert_eq!(
            handle.updates_since(head, MAX).await.unwrap(),
            Update::Advanced(chain(11..=12, 1))
        );
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn reports_reorgs_to_waiting_and_late_consumers_alike() {
        let state = RpcState::new(block(10, 1, 1));
        state.add([block(9, 1, 1)]);
        let (handle, _actor) = spawn(state.clone());
        let mut cursor = handle.next_head(BlockInfo::default()).await.unwrap();
        state.set_head(block(11, 1, 1));
        cursor = match handle.updates_since(cursor, MAX).await.unwrap() {
            Update::Advanced(blocks) => *blocks.last().unwrap(),
            update => panic!("unexpected {update:?}"),
        };
        assert_eq!(cursor, block(11, 1, 1));

        // A waiting consumer and one that asks later both see the reorg of block 11.
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.updates_since(cursor, MAX).await.unwrap() }
        });
        state.add([block(11, 2, 1)]);
        state.set_head(block(12, 2, 2));
        let expected = Update::Reorged {
            ancestor: block(10, 1, 1),
            blocks: vec![block(11, 2, 1), block(12, 2, 2)],
        };
        assert_eq!(waiting.await.unwrap(), expected);
        assert_eq!(handle.updates_since(cursor, MAX).await.unwrap(), expected);
    }

    #[tokio::test(start_paused = true)]
    async fn reports_status_and_finalized_changes_and_prunes_finalized_blocks() {
        let state = RpcState::new(block(10, 1, 1));
        state.add(chain(6..=9, 1));
        state.set("safe", block(8, 1, 1));
        state.set("finalized", block(6, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let finalized = handle.next_finalized(BlockInfo::default()).await.unwrap();
        assert_eq!(finalized, block(6, 1, 1));
        head_reaches(&handle, block(10, 1, 1)).await;
        assert_eq!(
            handle.status().await.unwrap(),
            Status { head: block(10, 1, 1), safe: block(8, 1, 1), finalized: block(6, 1, 1) }
        );

        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_finalized(finalized).await.unwrap() }
        });
        state.set("finalized", block(10, 1, 1));
        assert_eq!(next.await.unwrap(), block(10, 1, 1));
        // The view now starts at the finalized block.
        assert_eq!(
            handle.updates_since(block(6, 1, 1), MAX).await.unwrap(),
            Update::Behind { base: block(10, 1, 1) }
        );
    }

    #[tokio::test(start_paused = true)]
    async fn answers_requests_while_fetching_blocks() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        state.hold_by_hash.store(true, Ordering::SeqCst);
        state.add([block(11, 1, 1)]);
        state.set_head(block(12, 1, 1));
        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_head(head).await.unwrap() }
        });
        // The actor is waiting for block 11, but still answers.
        time::sleep(INTERVAL * 3).await;
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 1);
        assert_eq!(handle.status().await.unwrap().head, head);
        assert!(!next.is_finished());

        state.release.notify_one();
        assert_eq!(next.await.unwrap(), block(12, 1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn follows_a_head_that_changes_while_fetching() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        handle.next_head(BlockInfo::default()).await.unwrap();
        state.hold_by_hash.store(true, Ordering::SeqCst);
        // Head 12 needs block 11; before it arrives, head 12' on another fork needs block 11'.
        state.add([block(11, 1, 1), block(11, 2, 1)]);
        state.set_head(block(12, 1, 1));
        time::sleep(INTERVAL * 3).await;
        state.set_head(block(12, 2, 2));
        time::sleep(INTERVAL * 3).await;
        // Each block is fetched once, however many heads asked for it.
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 2);

        state.hold_by_hash.store(false, Ordering::SeqCst);
        state.release.notify_waiters();
        head_reaches(&handle, block(12, 2, 2)).await;
    }

    #[tokio::test(start_paused = true)]
    async fn fills_a_large_gap_one_parent_at_a_time() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        state.add(chain(11..=39, 1));
        state.set_head(block(40, 1, 1));
        head_reaches(&handle, block(40, 1, 1)).await;
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 29);
        let all = NonZeroUsize::new(64).unwrap();
        assert_eq!(
            handle.updates_since(head, all).await.unwrap(),
            Update::Advanced(chain(11..=40, 1))
        );
    }

    #[tokio::test(start_paused = true)]
    async fn waits_for_a_finalized_block_before_answering() {
        let state = RpcState::new(block(10, 1, 1));
        state.tags.lock().unwrap().remove("finalized");
        let (handle, _actor) = spawn(state.clone());
        assert!(timeout(INTERVAL * 3, handle.status()).await.is_err());
        state.set("finalized", block(10, 1, 1));
        assert_eq!(handle.status().await.unwrap().finalized, block(10, 1, 1));
    }

    #[rstest::rstest]
    #[case::missing(false)]
    #[case::failed(true)]
    #[tokio::test(start_paused = true)]
    async fn an_unavailable_parent_keeps_the_held_chain_until_a_later_head(#[case] fail: bool) {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        state.fail_by_hash.store(fail, Ordering::SeqCst);
        // Block 11 is missing from the RPC, or fetching it fails.
        state.set_head(block(12, 1, 1));
        time::sleep(INTERVAL * 3).await;
        assert!(state.by_hash_calls.load(Ordering::SeqCst) > 0);
        assert_eq!(handle.status().await.unwrap().head, head);

        state.fail_by_hash.store(false, Ordering::SeqCst);
        state.add([block(11, 1, 1)]);
        assert_eq!(
            handle.updates_since(head, MAX).await.unwrap(),
            Update::Advanced(chain(11..=12, 1))
        );
    }

    #[tokio::test(start_paused = true)]
    async fn a_polled_head_that_moves_back_reorgs_later_cursors() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        state.set_head(block(11, 1, 1));
        let cursor = handle.next_head(head).await.unwrap();
        state.set_head(head);
        assert_eq!(
            handle.updates_since(cursor, MAX).await.unwrap(),
            Update::Reorged { ancestor: head, blocks: vec![] }
        );
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn retries_failed_polls() {
        let state = RpcState::new(block(10, 1, 1));
        state.fail_polls.store(true, Ordering::SeqCst);
        let (handle, _actor) = spawn(state.clone());
        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_head(BlockInfo::default()).await.unwrap() }
        });
        time::sleep(INTERVAL * 3).await;
        assert!(!next.is_finished());
        state.fail_polls.store(false, Ordering::SeqCst);
        assert_eq!(next.await.unwrap(), block(10, 1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn a_cancelled_wait_loses_no_block() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let finalized = handle.next_finalized(BlockInfo::default()).await.unwrap();
        tokio::select! {
            _ = handle.next_finalized(finalized) => panic!("the finalized block did not change"),
            () = time::sleep(INTERVAL * 3) => {}
        }
        // A finalized block ahead of the head restarts the view from it.
        state.set("finalized", block(11, 1, 1));
        assert_eq!(handle.next_finalized(finalized).await.unwrap(), block(11, 1, 1));
        assert_eq!(handle.status().await.unwrap().head, block(11, 1, 1));
    }

    #[tokio::test]
    async fn dropping_the_lifetime_cancels_rpcs_and_fails_requests() {
        let state = RpcState::new(block(1, 1, 1));
        state.stall.store(true, Ordering::SeqCst);
        let builder = Builder::new(Capacity::try_from(1).unwrap());
        let handle = builder.handle();
        let mut lifetime = Box::pin(builder.build(provider(state.clone()), CONFIG));
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        drop(lifetime);
        state.cancelled.notified().await;
        assert!(handle.status().await.is_err());
    }

    #[tokio::test]
    async fn a_terminal_stream_error_stops_the_actor() {
        let state = RpcState::new(block(1, 1, 1));
        state.terminal.store(true, Ordering::SeqCst);
        let (handle, actor) = spawn(state);
        assert!(matches!(actor.await.unwrap(), Err(ActorError::StreamEnded)));
        assert!(handle.status().await.is_err());
    }

    #[test]
    fn builds_without_a_runtime() {
        let builder = Builder::new(Capacity::try_from(1).unwrap());
        let state = RpcState::new(block(1, 1, 1));
        drop(builder.build(provider(state), CONFIG));
    }
}
