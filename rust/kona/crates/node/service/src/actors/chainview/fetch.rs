//! The one L1 read the chain view needs beyond what the derivation pipeline already does,
//! behind a trait so the actor is testable.

use alloy_primitives::{Address, B256, b256};
use alloy_provider::Provider;
use alloy_transport::TransportError;
use async_trait::async_trait;
use kona_genesis::RollupConfig;

/// The storage slot holding the unsafe block signer in the `SystemConfig` contract:
/// `bytes32(uint256(keccak256("systemconfig.unsafeblocksigner")) - 1)`.
pub const UNSAFE_BLOCK_SIGNER_SLOT: B256 =
    b256!("0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08");

/// The L1 read the chain view performs.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait L1Fetcher: std::fmt::Debug + Send + Sync {
    /// The unsafe block signer stored in the `SystemConfig` contract at the block with `hash`.
    async fn unsafe_block_signer_at(&self, hash: B256) -> Result<Address, TransportError>;
}

/// [`L1Fetcher`] over an alloy provider.
#[derive(Debug, Clone)]
pub struct ProviderL1Fetcher<P> {
    provider: P,
    system_config: Address,
}

impl<P: Provider> ProviderL1Fetcher<P> {
    /// A fetcher reading the `SystemConfig` contract at `rollup_config.l1_system_config_address`.
    pub const fn new(provider: P, rollup_config: &RollupConfig) -> Self {
        Self { provider, system_config: rollup_config.l1_system_config_address }
    }
}

#[async_trait]
impl<P: Provider + std::fmt::Debug + Send + Sync> L1Fetcher for ProviderL1Fetcher<P> {
    async fn unsafe_block_signer_at(&self, hash: B256) -> Result<Address, TransportError> {
        let word = self
            .provider
            .get_storage_at(self.system_config, UNSAFE_BLOCK_SIGNER_SLOT.into())
            .hash(hash)
            .await?;
        Ok(Address::from_slice(&word.to_be_bytes::<32>()[12..]))
    }
}
