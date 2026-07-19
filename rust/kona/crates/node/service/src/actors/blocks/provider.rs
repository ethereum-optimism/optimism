use alloy_eips::{BlockNumHash, BlockNumberOrTag};
use alloy_provider::{Provider, RootProvider};
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_network::Optimism;
use thiserror::Error;

/// Read-only local L2 provider used to anchor the sequencer blocks stream.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait BlocksClientLocalProvider: std::fmt::Debug + Send + Sync {
    /// Returns the local canonical block at the requested number.
    async fn block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockNumHash>, BlocksClientLocalProviderError>;

    /// Returns the local canonical unsafe head.
    async fn latest_block(&self) -> Result<BlockNumHash, BlocksClientLocalProviderError>;

    /// Returns the local safe head, if one is available.
    async fn safe_block(&self) -> Result<Option<BlockNumHash>, BlocksClientLocalProviderError>;
}

#[async_trait]
impl BlocksClientLocalProvider for RootProvider<Optimism> {
    async fn block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockNumHash>, BlocksClientLocalProviderError> {
        block_by_label(self, number.into()).await
    }

    async fn latest_block(&self) -> Result<BlockNumHash, BlocksClientLocalProviderError> {
        block_by_label(self, BlockNumberOrTag::Latest)
            .await?
            .ok_or(BlocksClientLocalProviderError::LatestBlockUnavailable)
    }

    async fn safe_block(&self) -> Result<Option<BlockNumHash>, BlocksClientLocalProviderError> {
        match block_by_label(self, BlockNumberOrTag::Safe).await {
            Err(BlocksClientLocalProviderError::Transport(error))
                if is_block_not_found_error(&error) =>
            {
                Ok(None)
            }
            result => result,
        }
    }
}

fn is_block_not_found_error(error: &RpcError<TransportErrorKind>) -> bool {
    let message = error.to_string();
    message.contains("block not found") || message.contains("Unknown block")
}

async fn block_by_label(
    provider: &RootProvider<Optimism>,
    label: BlockNumberOrTag,
) -> Result<Option<BlockNumHash>, BlocksClientLocalProviderError> {
    let block = provider.get_block_by_number(label).await?;
    Ok(block.map(|block| BlockNumHash { number: block.header.number, hash: block.header.hash }))
}

/// Error reading local L2 chain state for the blocks client.
#[derive(Debug, Error)]
pub enum BlocksClientLocalProviderError {
    /// The local L2 RPC request failed.
    #[error("local L2 RPC request failed: {0}")]
    Transport(#[from] RpcError<TransportErrorKind>),
    /// The local L2 RPC did not return a latest block.
    #[error("local L2 latest block is unavailable")]
    LatestBlockUnavailable,
}
