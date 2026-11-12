use std::time::Duration;

use async_stream::stream;
use futures::Stream;
use kona_protocol::BlockInfo;
use kona_providers_alloy::{AlloyChainProvider, BlockLabel};
use tokio::time::MissedTickBehavior;

/// Polls the L1 block at a [`BlockLabel`] and yields it whenever it changes.
///
/// Note that this stream is not guaranteed to be contiguous. It may miss certain blocks, and
/// yielded items should only be considered to be the latest block at the given [`BlockLabel`].
#[derive(Debug, Clone)]
pub struct BlockStream {
    /// The L1 provider.
    l1_provider: AlloyChainProvider,
    /// The block label to poll for.
    label: BlockLabel,
    /// The poll interval.
    poll_interval: Duration,
}

impl BlockStream {
    /// Creates a new [`Stream<Item = BlockInfo>`] of the block at `label`.
    pub fn new_as_stream(
        l1_provider: AlloyChainProvider,
        label: BlockLabel,
        poll_interval: Duration,
    ) -> impl Stream<Item = BlockInfo> + Unpin + Send {
        Self { l1_provider, label, poll_interval }.into_stream()
    }

    /// Creates a [`Stream`] of [`BlockInfo`]. A failed poll is logged and retried at the next
    /// interval. Polls are at least one interval apart, even after a slow poll.
    pub fn into_stream(self) -> impl Stream<Item = BlockInfo> + Unpin + Send {
        let Self { l1_provider, label, poll_interval } = self;
        Box::pin(stream! {
            let mut interval = tokio::time::interval(poll_interval);
            // The default, `Burst`, would follow a poll that outlasts several intervals with one
            // poll per missed tick, back to back, while the endpoint is already slow.
            interval.set_missed_tick_behavior(MissedTickBehavior::Delay);
            let mut last_block = None;
            loop {
                interval.tick().await;
                match l1_provider.block_info_by_label(label).await {
                    Ok(info) if last_block != Some(info) => {
                        last_block = Some(info);
                        yield info;
                    }
                    Ok(_) => {}
                    Err(error) => {
                        warn!(target: "l1_watcher", ?label, %error, "Failed to poll L1 block");
                    }
                }
            }
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use futures::StreamExt;
    use kona_engine::test_utils::RpcMock;
    use tokio::time::Instant;

    #[tokio::test(start_paused = true)]
    async fn a_slow_poll_does_not_cause_a_burst_of_polls() {
        const POLL_INTERVAL: Duration = Duration::from_secs(4);
        const SLOW_POLL: Duration = Duration::from_secs(10);

        // Each poll answers with a new block. The first outlasts two intervals; the others answer
        // at once, so each later block is yielded when its poll starts.
        let rpc = RpcMock::default();
        for (number, delay) in [(1, SLOW_POLL), (2, Duration::ZERO), (3, Duration::ZERO)] {
            let mut block =
                alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
            block.header.inner.number = number;
            rpc.expect_with_delay("eth_getBlockByNumber", block, delay);
        }
        let mut stream =
            BlockStream::new_as_stream(rpc.l1_provider(), BlockLabel::Latest, POLL_INTERVAL);

        let mut yielded = Vec::new();
        for _ in 0..3 {
            stream.next().await.unwrap();
            yielded.push(Instant::now());
        }

        // The tick missed during the slow poll fires once, as soon as it ends. Another poll must
        // then wait a full interval rather than fire for the second missed tick.
        assert!(yielded[2] - yielded[1] >= POLL_INTERVAL, "blocks yielded at {yielded:?}");
        rpc.assert_finished();
    }
}
