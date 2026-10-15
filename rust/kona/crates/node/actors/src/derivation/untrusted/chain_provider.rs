//! The derivation pipeline's view of L1.

use crate::l1::{self, HandleError};
use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_primitives::B256;
use async_trait::async_trait;
use kona_derive::{ChainProvider, PipelineError, PipelineErrorKind};
use kona_protocol::BlockInfo;
use kona_providers_alloy::{AlloyChainProvider, AlloyChainProviderError};
use thiserror::Error;

/// Reads L1 for the derivation pipeline.
///
/// The canonical block at each height comes from the L1 actor, so the pipeline follows the same
/// chain as every other actor, and finds no block after the actor's head. Block contents come from
/// `contents`, by hash.
#[derive(Debug, Clone)]
pub struct L1ChainProvider {
    l1: l1::Handle,
    contents: AlloyChainProvider,
}

impl L1ChainProvider {
    /// Creates the provider, which reads heights from `l1` and block contents from `contents`.
    pub const fn new(l1: l1::Handle, contents: AlloyChainProvider) -> Self {
        Self { l1, contents }
    }
}

/// A failed read of L1 for the derivation pipeline.
#[derive(Debug, Error)]
pub enum L1ChainProviderError {
    /// L1 has no block at the height yet.
    #[error("L1 has no block {0} yet")]
    Ahead(u64),
    /// The L1 actor could not answer.
    #[error(transparent)]
    L1(#[from] HandleError),
    /// A block's contents could not be read.
    #[error(transparent)]
    Contents(#[from] AlloyChainProviderError),
}

impl From<L1ChainProviderError> for PipelineErrorKind {
    fn from(error: L1ChainProviderError) -> Self {
        match error {
            // Not `PipelineError::Eof`, which a pipeline reset takes for success.
            error @ (L1ChainProviderError::Ahead(_) | L1ChainProviderError::L1(_)) => {
                PipelineError::Provider(error.to_string()).temp()
            }
            L1ChainProviderError::Contents(error) => error.into(),
        }
    }
}

#[async_trait]
impl ChainProvider for L1ChainProvider {
    type Error = L1ChainProviderError;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        Ok(self.contents.header_by_hash(hash).await?)
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        self.l1.canonical(number).await?.ok_or(L1ChainProviderError::Ahead(number))
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        Ok(self.contents.receipts_by_hash(hash).await?)
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        Ok(self.contents.block_info_and_transactions_by_hash(hash).await?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::{TestChain, chain};
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::mock::Asserter;

    #[tokio::test]
    async fn reads_heights_from_the_l1_actor() {
        let l1 = TestChain::new(&chain(3..=5, 1));
        let contents =
            AlloyChainProvider::new(RootProvider::new(RpcClient::mocked(Asserter::new())), 1);
        let mut provider = L1ChainProvider::new(l1.handle(), contents);
        assert_eq!(provider.block_info_by_number(4).await.unwrap(), chain(4..=4, 1)[0]);
        let ahead = provider.block_info_by_number(6).await.unwrap_err();
        assert!(matches!(
            PipelineErrorKind::from(ahead),
            PipelineErrorKind::Temporary(PipelineError::Provider(_))
        ));
        drop(l1);
        let stopped = provider.block_info_by_number(4).await.unwrap_err();
        assert!(matches!(
            PipelineErrorKind::from(stopped),
            PipelineErrorKind::Temporary(PipelineError::Provider(_))
        ));
    }
}
