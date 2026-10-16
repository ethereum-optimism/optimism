//! Reads L1 blocks through an untrusted RPC.

use alloy_consensus::{Header, Sealed};
use alloy_eips::BlockNumberOrTag;
use alloy_transport::TransportErrorKind;
use futures::{Stream, stream};
use kona_protocol::BlockInfo;
use kona_sources::{UntrustedRpc, UntrustedRpcError};
use std::time::Duration;
use tokio::time::{self, MissedTickBehavior};

/// Reads the latest finalized L1 block, to give [`Builder::new`](super::Builder::new) the block
/// the actor's chain starts at.
///
/// An L1 that has finalized no block yet, such as a new devnet, gives its genesis block, which
/// cannot reorg either.
pub async fn finalized_block(rpc: &UntrustedRpc) -> Result<BlockInfo, UntrustedRpcError> {
    let header = match rpc.header_by_number(BlockNumberOrTag::Finalized).await? {
        Some(header) => header,
        None => rpc.header_by_number(BlockNumberOrTag::Earliest).await?.ok_or_else(|| {
            UntrustedRpcError::Transport(TransportErrorKind::custom_str("L1 has no genesis block"))
        })?,
    };
    Ok(block_info(&header))
}

/// Polls the block with the given tag every `interval`, starting at once. A failed poll is tried
/// again at the next interval.
pub(super) fn poll(
    rpc: UntrustedRpc,
    tag: BlockNumberOrTag,
    interval: Duration,
) -> impl Stream<Item = BlockInfo> + Send + Unpin + 'static {
    let mut ticks = time::interval(interval);
    ticks.set_missed_tick_behavior(MissedTickBehavior::Delay);
    Box::pin(stream::unfold((rpc, ticks), move |(rpc, mut ticks)| async move {
        loop {
            ticks.tick().await;
            match rpc.header_by_number(tag).await {
                Ok(Some(header)) => return Some((block_info(&header), (rpc, ticks))),
                Ok(None) => {}
                Err(error) => warn!(target: "l1", %error, %tag, "Failed to poll an L1 block"),
            }
        }
    }))
}

/// Returns the identity of a checked L1 header.
pub(super) fn block_info(header: &Sealed<Header>) -> BlockInfo {
    BlockInfo::new(header.hash(), header.number, header.parent_hash, header.timestamp)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::{child, genesis, rpc_block};
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::Block;
    use alloy_transport::mock::Asserter;

    #[tokio::test]
    async fn reads_the_finalized_block_or_else_genesis() {
        let responses = Asserter::new();
        let rpc = UntrustedRpc::new(RootProvider::new(RpcClient::mocked(responses.clone())));
        let finalized = child(&genesis(), 1);
        responses.push_success(&rpc_block(&finalized));
        assert_eq!(finalized_block(&rpc).await.unwrap(), finalized);

        // No finalized block, as some clients answer, or Geth's error.
        responses.push_success(&Option::<Block>::None);
        responses.push_success(&rpc_block(&genesis()));
        assert_eq!(finalized_block(&rpc).await.unwrap(), genesis());
        responses.push_failure_msg("finalized block not found");
        responses.push_success(&rpc_block(&genesis()));
        assert_eq!(finalized_block(&rpc).await.unwrap(), genesis());

        // Any other failure is an error, not genesis.
        responses.push_failure_msg("L1 unavailable");
        assert!(finalized_block(&rpc).await.is_err());
    }
}
