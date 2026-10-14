//! Reads L1 blocks over RPC.

use alloy_eips::BlockNumberOrTag;
use alloy_provider::Provider;
use alloy_rpc_client::PollerBuilder;
use alloy_rpc_types_eth::Block;
use futures::Stream;
use kona_protocol::BlockInfo;
use std::time::Duration;

/// Polls the block with the given tag every `interval`. Failed polls are retried at the next
/// interval.
pub(super) fn poll(
    provider: &impl Provider,
    tag: BlockNumberOrTag,
    interval: Duration,
) -> impl Stream<Item = BlockInfo> + Send + Unpin + 'static {
    PollerBuilder::<(BlockNumberOrTag, bool), Block>::new(
        provider.weak_client(),
        "eth_getBlockByNumber",
        (tag, false),
    )
    .with_poll_interval(interval)
    .into_stream()
    .map(|block| block_info(&block))
}

/// Returns the identity of an RPC block.
pub(super) fn block_info(block: &Block) -> BlockInfo {
    let header = &block.header;
    BlockInfo::new(header.hash, header.number, header.parent_hash, header.timestamp)
}
