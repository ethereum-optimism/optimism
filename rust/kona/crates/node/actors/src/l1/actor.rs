//! The L1 actor.

use super::{
    Config,
    chain::{Chain, Join},
    rpc::{block_info, poll},
};
use alloy_eips::BlockNumberOrTag;
use alloy_provider::{Provider, RootProvider};
use futures::StreamExt;
use kona_protocol::BlockInfo;
use thiserror::Error;
use tokio::sync::watch;

/// A fatal error from the L1 actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// A block stream ended.
    #[error("block stream ended")]
    StreamEnded,
}

/// Polls L1 and publishes the canonical chain from the latest finalized block to the head.
#[derive(Debug)]
pub(super) struct Actor {
    pub(super) provider: RootProvider,
    pub(super) config: Config,
    pub(super) chain: watch::Sender<Chain>,
}

impl Actor {
    /// Runs the actor until a block stream ends.
    pub(super) async fn run(self) -> Result<(), ActorError> {
        let Config { head_interval, finalized_interval } = self.config;
        let mut heads = poll(&self.provider, BlockNumberOrTag::Latest, head_interval);
        let mut safe = poll(&self.provider, BlockNumberOrTag::Safe, head_interval);
        let mut finalized = poll(&self.provider, BlockNumberOrTag::Finalized, finalized_interval);
        loop {
            tokio::select! {
                head = heads.next() => self.follow(head.ok_or(ActorError::StreamEnded)?).await,
                block = safe.next() => {
                    let block = block.ok_or(ActorError::StreamEnded)?;
                    self.chain.send_if_modified(|chain| chain.set_safe(block));
                }
                block = finalized.next() => self.finalize(block.ok_or(ActorError::StreamEnded)?),
            }
        }
    }

    /// Makes `head` the chain's head, first fetching the ancestors that link it to the chain. If
    /// one cannot be fetched, the chain keeps its head, and the next head tries again.
    async fn follow(&self, head: BlockInfo) {
        // The head and the ancestors fetched for it, newest first.
        let mut branch = vec![head];
        loop {
            let oldest = branch[branch.len() - 1];
            let join = self.chain.borrow().join(&oldest);
            match join {
                Join::Held | Join::Extends => break,
                Join::Rejected => {
                    warn!(target: "l1", ?head, "L1 head does not descend from the finalized block");
                    return;
                }
                Join::NeedsParent => {
                    let hash = oldest.parent_hash;
                    match self.provider.get_block_by_hash(hash).await {
                        Ok(Some(parent)) => branch.push(block_info(&parent)),
                        Ok(None) => {
                            debug!(target: "l1", ?head, %hash, "An ancestor of the L1 head was reorged out");
                            return;
                        }
                        Err(error) => {
                            warn!(target: "l1", ?error, %hash, "Failed to fetch an ancestor of the L1 head");
                            return;
                        }
                    }
                }
            }
        }
        branch.reverse();
        self.chain.send_if_modified(|chain| {
            let previous = chain.head();
            let orphaned = chain.set_head(branch);
            if orphaned > 0 {
                info!(target: "l1", ?head, orphaned, "L1 reorg");
            }
            chain.head() != previous
        });
    }

    /// Moves the chain's start to a newly finalized block.
    fn finalize(&self, finalized: BlockInfo) {
        self.chain.send_if_modified(|chain| {
            let previous = chain.finalized();
            if chain.finalize(finalized) {
                info!(target: "l1", ?finalized, "L1 chain restarted at a finalized block it did not hold");
            }
            chain.finalized() != previous
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::{
        Builder, Child, Handle, HandleError, Status,
        test_utils::{block, chain},
    };
    use alloy_json_rpc::{ErrorPayload, RequestPacket, Response, ResponsePacket, ResponsePayload};
    use alloy_primitives::B256;
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

        /// Returns the block at `number` on the chain that ends at the head.
        fn canonical(&self, number: u64) -> Option<BlockInfo> {
            let tree = self.blocks.lock().unwrap();
            let mut block = self.tags.lock().unwrap().get("latest").copied()?;
            while block.number > number {
                block = *tree.get(&block.parent_hash)?;
            }
            (block.number == number).then_some(block)
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
                            let tag = params[0].as_str().unwrap();
                            match tag.strip_prefix("0x") {
                                Some(number) => {
                                    state.canonical(u64::from_str_radix(number, 16).unwrap())
                                }
                                None => {
                                    if state.fail_polls.load(Ordering::SeqCst) {
                                        return Err(TransportErrorKind::custom_str(
                                            "L1 unavailable",
                                        ));
                                    }
                                    state.tags.lock().unwrap().get(tag).copied()
                                }
                            }
                        }
                        "eth_getBlockByHash" => {
                            state.by_hash_calls.fetch_add(1, Ordering::SeqCst);
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

    /// Runs the actor with its chain starting at L1's finalized block.
    fn spawn(state: Arc<RpcState>) -> (Handle, JoinHandle<Result<(), ActorError>>) {
        let finalized = state.tags.lock().unwrap()["finalized"];
        let builder = Builder::new(provider(state), finalized);
        let handle = builder.handle();
        (handle, tokio::spawn(builder.build(CONFIG)))
    }

    /// Waits until the actor's head is `target`.
    async fn head_reaches(handle: &Handle, target: BlockInfo) {
        let mut head = BlockInfo::default();
        while head != target {
            head = handle.next_head(head).await.unwrap();
        }
    }

    #[tokio::test(start_paused = true)]
    async fn follows_new_heads_fetching_the_blocks_between() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        assert_eq!(handle.next_head(BlockInfo::default()).await.unwrap(), block(10, 1, 1));

        // Nothing after the head yet.
        assert!(timeout(INTERVAL * 3, handle.child_of(block(10, 1, 1))).await.is_err());

        // The head jumps two blocks; the actor fetches the one between.
        state.add([block(11, 1, 1)]);
        state.set_head(block(12, 1, 1));
        assert_eq!(handle.child_of(block(10, 1, 1)).await.unwrap(), Child::Block(block(11, 1, 1)));
        assert_eq!(handle.child_of(block(11, 1, 1)).await.unwrap(), Child::Block(block(12, 1, 1)));
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn reports_reorgs_to_waiting_and_late_consumers_alike() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        state.set_head(block(11, 1, 1));
        head_reaches(&handle, block(11, 1, 1)).await;

        // A waiting consumer and one that asks later both see the reorg of block 11.
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.child_of(block(11, 1, 1)).await.unwrap() }
        });
        state.add([block(11, 2, 1)]);
        state.set_head(block(12, 2, 2));
        assert_eq!(waiting.await.unwrap(), Child::Reorged);
        assert_eq!(handle.child_of(block(11, 1, 1)).await.unwrap(), Child::Reorged);
        assert_eq!(handle.child_of(block(10, 1, 1)).await.unwrap(), Child::Block(block(11, 2, 1)));
    }

    #[tokio::test(start_paused = true)]
    async fn reports_status_and_reads_finalized_blocks_from_l1() {
        let state = RpcState::new(block(10, 1, 1));
        state.add(chain(6..=9, 1));
        state.set("safe", block(8, 1, 1));
        state.set("finalized", block(6, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        head_reaches(&handle, block(10, 1, 1)).await;
        assert_eq!(
            handle.status().unwrap(),
            Status { head: block(10, 1, 1), safe: block(8, 1, 1), finalized: block(6, 1, 1) }
        );

        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_finalized(block(6, 1, 1)).await.unwrap() }
        });
        state.set("finalized", block(10, 1, 1));
        assert_eq!(next.await.unwrap(), block(10, 1, 1));
        // The chain now starts at the finalized block, and the handle reads older blocks from L1.
        assert_eq!(handle.canonical(7).await.unwrap(), Some(block(7, 1, 1)));
        assert_eq!(handle.child_of(block(7, 1, 1)).await.unwrap(), Child::Block(block(8, 1, 1)));
    }

    #[tokio::test(start_paused = true)]
    async fn fills_a_large_gap_one_parent_at_a_time() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        state.add(chain(11..=39, 1));
        state.set_head(block(40, 1, 1));
        head_reaches(&handle, block(40, 1, 1)).await;
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 29);
        assert_eq!(handle.canonical(25).await.unwrap(), Some(block(25, 1, 1)));
    }

    #[rstest::rstest]
    #[case::missing(false)]
    #[case::failed(true)]
    #[tokio::test(start_paused = true)]
    async fn an_unavailable_parent_keeps_the_head_until_a_later_head(#[case] fail: bool) {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        state.fail_by_hash.store(fail, Ordering::SeqCst);
        // Block 11 is missing from the RPC, or fetching it fails.
        state.set_head(block(12, 1, 1));
        time::sleep(INTERVAL * 3).await;
        assert!(state.by_hash_calls.load(Ordering::SeqCst) > 0);
        assert_eq!(handle.status().unwrap().head, block(10, 1, 1));

        state.fail_by_hash.store(false, Ordering::SeqCst);
        state.add([block(11, 1, 1)]);
        assert_eq!(handle.next_head(block(10, 1, 1)).await.unwrap(), block(12, 1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn a_head_that_moves_back_reorgs_out_the_blocks_after_it() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        state.set_head(block(11, 1, 1));
        head_reaches(&handle, block(11, 1, 1)).await;
        state.set_head(block(10, 1, 1));
        head_reaches(&handle, block(10, 1, 1)).await;
        assert_eq!(handle.try_child_of(block(11, 1, 1), 0).await.unwrap(), None);
        assert_eq!(handle.canonical(11).await.unwrap(), None);
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn ignores_heads_that_do_not_descend_from_the_finalized_block() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        state.add([block(10, 9, 9)]);
        state.set_head(block(11, 9, 9));
        time::sleep(INTERVAL * 3).await;
        assert_eq!(handle.status().unwrap().head, block(10, 1, 1));
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn retries_failed_polls() {
        let state = RpcState::new(block(10, 1, 1));
        state.fail_polls.store(true, Ordering::SeqCst);
        state.set_head(block(11, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_head(block(10, 1, 1)).await.unwrap() }
        });
        time::sleep(INTERVAL * 3).await;
        assert!(!next.is_finished());
        state.fail_polls.store(false, Ordering::SeqCst);
        assert_eq!(next.await.unwrap(), block(11, 1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn a_finalized_block_ahead_of_the_head_restarts_the_chain_from_it() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        state.set("finalized", block(11, 1, 1));
        assert_eq!(handle.next_finalized(block(10, 1, 1)).await.unwrap(), block(11, 1, 1));
        assert_eq!(handle.status().unwrap().head, block(11, 1, 1));
    }

    #[tokio::test]
    async fn dropping_the_lifetime_cancels_rpcs_and_fails_questions() {
        let state = RpcState::new(block(1, 1, 1));
        state.stall.store(true, Ordering::SeqCst);
        let builder = Builder::new(provider(state.clone()), block(1, 1, 1));
        let handle = builder.handle();
        let mut lifetime = Box::pin(builder.build(CONFIG));
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        drop(lifetime);
        state.cancelled.notified().await;
        assert!(matches!(handle.status(), Err(HandleError::Stopped)));
    }

    #[tokio::test]
    async fn a_terminal_stream_error_stops_the_actor() {
        let state = RpcState::new(block(1, 1, 1));
        state.terminal.store(true, Ordering::SeqCst);
        let (handle, actor) = spawn(state);
        assert!(matches!(actor.await.unwrap(), Err(ActorError::StreamEnded)));
        assert!(matches!(handle.status(), Err(HandleError::Stopped)));
    }

    #[test]
    fn builds_without_a_runtime() {
        let state = RpcState::new(block(1, 1, 1));
        drop(Builder::new(provider(state), block(1, 1, 1)).build(CONFIG));
    }
}
