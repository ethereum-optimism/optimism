//! Reads L1 blocks over RPC.

use alloy_eips::BlockNumberOrTag;
use alloy_provider::Provider;
use alloy_rpc_client::PollerBuilder;
use alloy_rpc_types_eth::Block;
use alloy_transport::{RpcError, TransportError, TransportErrorKind};
use futures::Stream;
use kona_protocol::BlockInfo;
use std::time::Duration;

/// Reads the latest finalized L1 block, to give [`Builder::new`](super::Builder::new) the block
/// the actor's chain starts at.
///
/// An L1 that has finalized no block yet, such as a new devnet, gives its genesis block, which
/// cannot reorg either.
pub async fn finalized_block(provider: &impl Provider) -> Result<BlockInfo, TransportError> {
    let finalized = match provider.get_block_by_number(BlockNumberOrTag::Finalized).await {
        Ok(block) => block,
        // Geth answers with this error, rather than no block, until it finalizes a block.
        Err(RpcError::ErrorResp(error)) if error.message == "finalized block not found" => None,
        Err(error) => return Err(error),
    };
    let block = match finalized {
        Some(block) => block,
        None => provider
            .get_block_by_number(BlockNumberOrTag::Earliest)
            .await?
            .ok_or_else(|| TransportErrorKind::custom_str("L1 has no genesis block"))?,
    };
    Ok(block_info(&block))
}

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::block;
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::mock::Asserter;

    fn rpc_block(info: BlockInfo) -> Block {
        let mut block: Block = Block::default();
        block.header.hash = info.hash;
        block.header.inner.number = info.number;
        block.header.inner.parent_hash = info.parent_hash;
        block.header.inner.timestamp = info.timestamp;
        block
    }

    #[tokio::test]
    async fn reads_the_finalized_block_or_else_genesis() {
        let responses = Asserter::new();
        let provider = RootProvider::new(RpcClient::mocked(responses.clone()));
        responses.push_success(&rpc_block(block(7, 1, 1)));
        assert_eq!(finalized_block(&provider).await.unwrap(), block(7, 1, 1));

        // No finalized block, as some clients answer, or Geth's error.
        responses.push_success(&Option::<Block>::None);
        responses.push_success(&rpc_block(block(0, 1, 1)));
        assert_eq!(finalized_block(&provider).await.unwrap(), block(0, 1, 1));
        responses.push_failure_msg("finalized block not found");
        responses.push_success(&rpc_block(block(0, 1, 1)));
        assert_eq!(finalized_block(&provider).await.unwrap(), block(0, 1, 1));

        // Any other failure is an error, not genesis.
        responses.push_failure_msg("L1 unavailable");
        assert!(finalized_block(&provider).await.is_err());
    }
}
