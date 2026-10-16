//! The L2 execution layer, as derivation from L1 reads it.

use alloy_eips::BlockNumberOrTag;
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_types_eth::Block;
use alloy_transport::TransportError;
use async_trait::async_trait;
use op_alloy_network::Optimism;
use op_alloy_rpc_types::Transaction;
use std::fmt::Debug;

/// The L2 execution layer, which derivation reads the unsafe chain from. It only reads.
#[async_trait]
pub trait L2: Debug + Send + Sync {
    /// Returns the execution layer's canonical block at `number`, with its transactions, or `None`
    /// if it has none there.
    async fn block(&self, number: u64) -> Result<Option<Block<Transaction>>, TransportError>;
}

#[async_trait]
impl L2 for RootProvider<Optimism> {
    async fn block(&self, number: u64) -> Result<Option<Block<Transaction>>, TransportError> {
        self.get_block_by_number(BlockNumberOrTag::Number(number)).full().await
    }
}
