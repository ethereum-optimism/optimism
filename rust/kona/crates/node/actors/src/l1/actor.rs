//! The L1 actor.

use super::{
    Config,
    chain::{Chain, Join},
    rpc::{block_info, poll},
};
use alloy_eips::BlockNumberOrTag;
use futures::StreamExt;
use kona_protocol::BlockInfo;
use kona_sources::RetryingRpc;
use std::convert::Infallible;
use tokio::sync::watch;

/// Polls L1 and publishes the canonical chain from the latest finalized block to the head.
#[derive(Debug)]
pub(super) struct Actor {
    pub(super) rpc: RetryingRpc,
    pub(super) config: Config,
    pub(super) chain: watch::Sender<Chain>,
}

impl Actor {
    /// Runs the actor. It has no fatal errors: it waits for as long as L1 cannot be read.
    pub(super) async fn run(self) -> Infallible {
        let Config { head_interval, finalized_interval } = self.config;
        let rpc = self.rpc.untrusted();
        let mut heads = poll(rpc.clone(), BlockNumberOrTag::Latest, head_interval);
        let mut safe = poll(rpc.clone(), BlockNumberOrTag::Safe, head_interval);
        let mut finalized = poll(rpc.clone(), BlockNumberOrTag::Finalized, finalized_interval);
        loop {
            // The polls never end.
            tokio::select! {
                Some(head) = heads.next() => self.follow(head).await,
                Some(block) = safe.next() => {
                    self.chain.send_if_modified(|chain| chain.set_safe(block));
                }
                Some(block) = finalized.next() => self.finalize(block),
            }
        }
    }

    /// Makes `head` the chain's head, first fetching the ancestors that link it to the chain. If
    /// one was reorged out, the chain keeps its head, and the next head tries again.
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
                    match self.rpc.header_by_hash(hash).await {
                        Some(parent) => branch.push(block_info(&parent)),
                        None => {
                            debug!(target: "l1", ?head, %hash, "An ancestor of the L1 head was reorged out");
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
        test_utils::{RETRY_DELAY, child, descendants, genesis, rpc_block},
    };
    use alloy_json_rpc::{RequestPacket, Response, ResponsePacket, ResponsePayload};
    use alloy_primitives::B256;
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::TransportErrorKind;
    use kona_sources::UntrustedRpc;
    use std::{
        collections::HashMap,
        sync::{
            Arc, Mutex,
            atomic::{AtomicBool, Ordering},
        },
        time::Duration,
    };
    use tokio::{
        sync::Notify,
        task::JoinHandle,
        time::{self, timeout},
    };

    const INTERVAL: Duration = Duration::from_secs(4);

    const CONFIG: Config = Config { head_interval: INTERVAL, finalized_interval: INTERVAL };

    /// Returns block `number` of the test L1 chain, which an RPC can serve.
    fn block(number: u64) -> BlockInfo {
        descendants(&genesis(), 40, 1).into_iter().nth(number as usize - 1).unwrap()
    }

    /// Returns blocks `numbers` of the test L1 chain.
    fn blocks(numbers: std::ops::RangeInclusive<u64>) -> Vec<BlockInfo> {
        numbers.map(block).collect()
    }

    /// An L1 RPC serving a block tree and the blocks its tags point at.
    #[derive(Default)]
    struct RpcState {
        blocks: Mutex<HashMap<B256, BlockInfo>>,
        tags: Mutex<HashMap<&'static str, BlockInfo>>,
        /// How often each block was fetched by hash.
        fetches: Mutex<HashMap<B256, usize>>,
        /// Holds every request until it is cancelled.
        stall: AtomicBool,
        stalled: Notify,
        cancelled: Notify,
        /// Fails polls for blocks by tag.
        fail_polls: AtomicBool,
        /// Fails fetches of the block with this hash.
        fail_fetch: Mutex<Option<B256>>,
        /// Reports a hash other than the head's.
        lie_about_head: AtomicBool,
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

        /// Returns how often the block was fetched by hash.
        fn fetches_of(&self, block: BlockInfo) -> usize {
            self.fetches.lock().unwrap().get(&block.hash).copied().unwrap_or_default()
        }

        /// Returns how often any block was fetched by hash.
        fn all_fetches(&self) -> usize {
            self.fetches.lock().unwrap().values().sum()
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

        /// Answers a JSON-RPC request.
        async fn answer(
            &self,
            method: &str,
            params: serde_json::Value,
        ) -> Result<serde_json::Value, alloy_transport::TransportError> {
            if self.stall.load(Ordering::SeqCst) {
                struct OnDrop<'a>(&'a Notify);
                impl Drop for OnDrop<'_> {
                    fn drop(&mut self) {
                        self.0.notify_one();
                    }
                }
                let _on_drop = OnDrop(&self.cancelled);
                self.stalled.notify_one();
                std::future::pending::<()>().await;
            }
            let block = match method {
                "eth_getBlockByNumber" => {
                    let tag = params[0].as_str().unwrap();
                    if let Some(number) = tag.strip_prefix("0x") {
                        self.canonical(u64::from_str_radix(number, 16).unwrap())
                            .map(|b| rpc_block(&b))
                    } else {
                        if self.fail_polls.load(Ordering::SeqCst) {
                            return Err(TransportErrorKind::custom_str("L1 unavailable"));
                        }
                        let block = self.tags.lock().unwrap().get(tag).copied();
                        block.map(|block| {
                            let mut rpc = rpc_block(&block);
                            if tag == "latest" && self.lie_about_head.load(Ordering::SeqCst) {
                                rpc.header.hash = B256::repeat_byte(0xff);
                            }
                            rpc
                        })
                    }
                }
                "eth_getBlockByHash" => {
                    let hash: B256 = serde_json::from_value(params[0].clone()).unwrap();
                    *self.fetches.lock().unwrap().entry(hash).or_default() += 1;
                    if *self.fail_fetch.lock().unwrap() == Some(hash) {
                        return Err(TransportErrorKind::custom_str("L1 unavailable"));
                    }
                    self.blocks.lock().unwrap().get(&hash).map(rpc_block)
                }
                method => panic!("unexpected method {method}"),
            };
            Ok(serde_json::to_value(block).unwrap())
        }
    }

    fn provider(state: Arc<RpcState>) -> RootProvider {
        let transport = tower::service_fn(
            move |request: RequestPacket| -> alloy_transport::TransportFut<'static> {
                let state = state.clone();
                Box::pin(async move {
                    let RequestPacket::Single(request) = request else {
                        panic!("unexpected batch")
                    };
                    let params = serde_json::from_str(request.params().unwrap().get()).unwrap();
                    let result = state.answer(request.method(), params).await?;
                    Ok(ResponsePacket::Single(Response {
                        id: request.id().clone(),
                        payload: ResponsePayload::Success(
                            serde_json::value::to_raw_value(&result).unwrap(),
                        ),
                    }))
                })
            },
        );
        RootProvider::new(RpcClient::new(transport, false))
    }

    /// Returns the builder of an actor whose chain starts at `finalized`.
    fn builder(state: &Arc<RpcState>, finalized: BlockInfo) -> Builder {
        let rpc = UntrustedRpc::new(provider(state.clone())).retrying(RETRY_DELAY);
        Builder::new(rpc, finalized)
    }

    /// Runs the actor with its chain starting at L1's finalized block.
    fn spawn(state: &Arc<RpcState>) -> (Handle, JoinHandle<Infallible>) {
        let finalized = state.tags.lock().unwrap()["finalized"];
        let builder = builder(state, finalized);
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
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        assert_eq!(handle.next_head(BlockInfo::default()).await.unwrap(), block(10));

        // Nothing after the head yet.
        assert!(timeout(INTERVAL * 3, handle.child_of(block(10).id())).await.is_err());

        // The head jumps two blocks; the actor fetches the one between.
        state.add([block(11)]);
        state.set_head(block(12));
        assert_eq!(handle.child_of(block(10).id()).await.unwrap(), Child::Block(block(11)));
        assert_eq!(handle.child_of(block(11).id()).await.unwrap(), Child::Block(block(12)));
        assert_eq!(state.all_fetches(), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn reports_reorgs_to_waiting_and_late_consumers_alike() {
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        state.set_head(block(11));
        head_reaches(&handle, block(11)).await;

        // A waiting consumer and one that asks later both see the reorg of block 11.
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.child_of(block(11).id()).await.unwrap() }
        });
        let fork = descendants(&block(10), 2, 2);
        state.add([fork[0]]);
        state.set_head(fork[1]);
        assert_eq!(waiting.await.unwrap(), Child::Reorged);
        assert_eq!(handle.child_of(block(11).id()).await.unwrap(), Child::Reorged);
        assert_eq!(handle.child_of(block(10).id()).await.unwrap(), Child::Block(fork[0]));
    }

    #[tokio::test(start_paused = true)]
    async fn reports_status_and_reads_finalized_blocks_from_l1() {
        let state = RpcState::new(block(10));
        state.add(blocks(6..=9));
        state.set("safe", block(8));
        state.set("finalized", block(6));
        let (handle, _actor) = spawn(&state);
        head_reaches(&handle, block(10)).await;
        assert_eq!(
            handle.status().unwrap(),
            Status { head: block(10), safe: block(8), finalized: block(6) }
        );

        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_finalized(block(6)).await.unwrap() }
        });
        state.set("finalized", block(10));
        assert_eq!(next.await.unwrap(), block(10));
        // The chain now starts at the finalized block, and the handle reads older blocks from L1.
        assert_eq!(handle.canonical(7).await.unwrap(), Some(block(7)));
        assert_eq!(handle.child_of(block(7).id()).await.unwrap(), Child::Block(block(8)));
    }

    #[tokio::test(start_paused = true)]
    async fn fills_a_large_gap_one_parent_at_a_time() {
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        state.add(blocks(11..=39));
        state.set_head(block(40));
        head_reaches(&handle, block(40)).await;
        assert_eq!(state.all_fetches(), 29);
        assert_eq!(handle.canonical(25).await.unwrap(), Some(block(25)));
    }

    #[tokio::test(start_paused = true)]
    async fn keeps_fetching_an_ancestor_until_l1_serves_it() {
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        state.add(blocks(11..=13));
        *state.fail_fetch.lock().unwrap() = Some(block(11).hash);
        state.set_head(block(14));
        time::sleep(INTERVAL * 3).await;
        assert!(state.fetches_of(block(11)) > 1);
        assert_eq!(handle.status().unwrap().head, block(10));

        *state.fail_fetch.lock().unwrap() = None;
        assert_eq!(handle.next_head(block(10)).await.unwrap(), block(14));
        // The ancestors fetched before the failure were kept.
        assert_eq!((state.fetches_of(block(13)), state.fetches_of(block(12))), (1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn a_reorged_out_ancestor_keeps_the_head_until_a_later_head() {
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        // Block 11 is missing from the RPC.
        state.set_head(block(12));
        time::sleep(INTERVAL * 3).await;
        assert!(state.fetches_of(block(11)) > 0);
        assert_eq!(handle.status().unwrap().head, block(10));

        state.add([block(11)]);
        assert_eq!(handle.next_head(block(10)).await.unwrap(), block(12));
    }

    #[tokio::test(start_paused = true)]
    async fn a_head_that_moves_back_reorgs_out_the_blocks_after_it() {
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        state.set_head(block(11));
        head_reaches(&handle, block(11)).await;
        state.set_head(block(10));
        head_reaches(&handle, block(10)).await;
        assert_eq!(handle.try_child_of(block(11).id(), 0).await.unwrap(), None);
        assert_eq!(handle.canonical(11).await.unwrap(), None);
        assert_eq!(state.all_fetches(), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn ignores_heads_that_do_not_descend_from_the_finalized_block() {
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        let fork = descendants(&block(9), 2, 9);
        state.add([fork[0]]);
        state.set_head(fork[1]);
        time::sleep(INTERVAL * 3).await;
        assert_eq!(handle.status().unwrap().head, block(10));
        assert_eq!(state.all_fetches(), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn ignores_heads_whose_hash_is_not_their_headers() {
        let state = RpcState::new(block(10));
        state.lie_about_head.store(true, Ordering::SeqCst);
        let (handle, _actor) = spawn(&state);
        state.set_head(block(11));
        time::sleep(INTERVAL * 3).await;
        assert_eq!(handle.status().unwrap().head, block(10));

        state.lie_about_head.store(false, Ordering::SeqCst);
        assert_eq!(handle.next_head(block(10)).await.unwrap(), block(11));
    }

    #[tokio::test(start_paused = true)]
    async fn retries_failed_polls() {
        let state = RpcState::new(block(10));
        state.fail_polls.store(true, Ordering::SeqCst);
        state.set_head(block(11));
        let (handle, _actor) = spawn(&state);
        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_head(block(10)).await.unwrap() }
        });
        time::sleep(INTERVAL * 3).await;
        assert!(!next.is_finished());
        state.fail_polls.store(false, Ordering::SeqCst);
        assert_eq!(next.await.unwrap(), block(11));
    }

    #[tokio::test(start_paused = true)]
    async fn a_finalized_block_ahead_of_the_head_restarts_the_chain_from_it() {
        let state = RpcState::new(block(10));
        let (handle, _actor) = spawn(&state);
        state.set("finalized", block(11));
        assert_eq!(handle.next_finalized(block(10)).await.unwrap(), block(11));
        assert_eq!(handle.status().unwrap().head, block(11));
    }

    #[tokio::test]
    async fn dropping_the_lifetime_cancels_rpcs_and_fails_questions() {
        let state = RpcState::new(block(1));
        state.stall.store(true, Ordering::SeqCst);
        let builder = builder(&state, block(1));
        let handle = builder.handle();
        let lifetime = tokio::spawn(builder.build(CONFIG));
        state.stalled.notified().await;
        lifetime.abort();
        state.cancelled.notified().await;
        assert!(matches!(handle.status(), Err(HandleError)));
    }

    #[test]
    fn builds_without_a_runtime() {
        let state = RpcState::new(block(1));
        drop(builder(&state, block(1)).build(CONFIG));
    }

    #[test]
    fn test_blocks_link_by_their_hashes() {
        assert_eq!(block(1).parent_hash, genesis().hash);
        assert_eq!(block(2).parent_hash, block(1).hash);
        assert_ne!(child(&block(1), 2), block(2));
    }
}
