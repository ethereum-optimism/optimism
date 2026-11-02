use std::time::Duration;

use async_stream::stream;
use futures::Stream;
use kona_protocol::BlockInfo;
use kona_providers_alloy::{AlloyChainProvider, BlockLabel};

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
    /// interval.
    ///
    /// Each poll is followed by a full interval, so a slow endpoint is not polled again as soon as
    /// it answers.
    pub fn into_stream(self) -> impl Stream<Item = BlockInfo> + Unpin + Send {
        let Self { l1_provider, label, poll_interval } = self;
        Box::pin(stream! {
            let mut last_block = None;
            loop {
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
                tokio::time::sleep(poll_interval).await;
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

    /// An L1 block at `number`, as `eth_getBlockByNumber` returns it.
    fn block(number: u64) -> alloy_rpc_types_eth::Block {
        alloy_rpc_types_eth::Block::new(
            alloy_rpc_types_eth::Header::new(alloy_consensus::Header {
                number,
                ..Default::default()
            }),
            alloy_rpc_types_eth::BlockTransactions::Hashes(Vec::new()),
        )
    }

    #[tokio::test(start_paused = true)]
    async fn a_slow_poll_is_followed_by_a_full_interval() {
        let l1 = RpcMock::default();
        l1.expect_with_delay("eth_getBlockByNumber", block(1), Duration::from_secs(10));
        l1.expect("eth_getBlockByNumber", block(2));
        let mut stream = BlockStream::new_as_stream(
            l1.l1_provider(),
            BlockLabel::Latest,
            Duration::from_secs(4),
        );

        let start = Instant::now();
        assert_eq!(stream.next().await.unwrap().number, 1);
        let slow_poll_done = start.elapsed();
        assert_eq!(stream.next().await.unwrap().number, 2);

        // The next poll waits a full interval after the 10s poll rather than starting at once.
        assert_eq!(start.elapsed() - slow_poll_done, Duration::from_secs(4));
        l1.assert_finished();
    }
}
