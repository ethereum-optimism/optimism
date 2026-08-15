//! Polling streams for canonical L1 labels.

use alloy_eips::BlockNumberOrTag;
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_client::PollerBuilder;
use alloy_rpc_types_eth::Block;
use async_stream::stream;
use futures::{Stream, StreamExt};
use kona_protocol::BlockInfo;
use std::time::Duration;

/// Creates a deduplicated stream for an L1 block tag.
pub(super) fn block_stream(
    provider: RootProvider,
    tag: BlockNumberOrTag,
    poll_interval: Duration,
) -> impl Stream<Item = BlockInfo> + Unpin + Send {
    let mut poll_stream = PollerBuilder::<(BlockNumberOrTag, bool), Block>::new(
        provider.weak_client(),
        "eth_getBlockByNumber",
        (tag, false),
    )
    .with_poll_interval(poll_interval)
    .into_stream();

    Box::pin(stream! {
        let mut last_block = None;
        while let Some(next) = poll_stream.next().await {
            let info: BlockInfo = next.into_consensus().into();
            if last_block != Some(info) {
                last_block = Some(info);
                yield info;
            }
        }
    })
}
