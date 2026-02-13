use std::time::Duration;

use alloy_eips::BlockNumberOrTag;
use alloy_provider::Provider;
use alloy_rpc_client::PollerBuilder;
use alloy_rpc_types_eth::Block;
use async_stream::stream;
use futures::{Stream, StreamExt};
use kona_protocol::BlockInfo;

/// A wrapper around a [`PollerBuilder`] that observes [`BlockInfo`] updates on a [`Provider`].
///
/// Note that this stream is not guaranteed to be contiguous. It may miss certain blocks, and
/// yielded items should only be considered to be the latest block matching the given
/// [`BlockNumberOrTag`].
#[derive(Debug, Clone)]
pub struct BlockStream<L1P>
where
    L1P: Provider,
{
    /// The inner [`Provider`].
    l1_provider: L1P,
    /// The block tag to poll for.
    tag: BlockNumberOrTag,
    /// The poll interval (in seconds).
    poll_interval: Duration,
}

impl<L1P: Provider> BlockStream<L1P> {
    /// Creates a new [`Stream<Item = BlockInfo>`] instance.
    ///
    /// # Returns
    /// Returns error if the passed [`BlockNumberOrTag`] is of the [`BlockNumberOrTag::Number`]
    /// variant.
    pub fn new_as_stream(
        l1_provider: L1P,
        tag: BlockNumberOrTag,
        poll_interval: Duration,
    ) -> Result<impl Stream<Item = BlockInfo> + Unpin + Send, String> {
        if matches!(tag, BlockNumberOrTag::Number(_)) {
            error!("Invalid BlockNumberOrTag variant - Must be a tag");
        }
        Ok(Self { l1_provider, tag, poll_interval }.into_stream())
    }

    /// Creates a [`Stream`] of [`BlockInfo`].
    pub fn into_stream(self) -> impl Stream<Item = BlockInfo> + Unpin + Send {
        let mut poll_stream = PollerBuilder::<(BlockNumberOrTag, bool), Block>::new(
            self.l1_provider.weak_client(),
            "eth_getBlockByNumber",
            (self.tag, false),
        )
        .with_poll_interval(self.poll_interval)
        .into_stream();

        Box::pin(stream! {
            let mut last_block = None;
            while let Some(next) = poll_stream.next().await {
                let info: BlockInfo = next.into_consensus().into();

                if last_block.map(|b| b != info).unwrap_or(true) {
                    last_block = Some(info);
                    yield info;
                }
            }
        })
    }
}
