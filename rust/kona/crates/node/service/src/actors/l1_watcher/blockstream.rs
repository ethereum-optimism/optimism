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
    pub fn into_stream(self) -> impl Stream<Item = BlockInfo> + Unpin + Send {
        let Self { l1_provider, label, poll_interval } = self;
        Box::pin(stream! {
            let mut interval = tokio::time::interval(poll_interval);
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
