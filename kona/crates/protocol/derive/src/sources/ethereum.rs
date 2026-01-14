//! Contains the [EthereumDataSource], which is a concrete implementation of the
//! [DataAvailabilityProvider] trait for the Ethereum protocol.

use crate::{
    BlobProvider, BlobSource, CalldataSource, ChainProvider, DataAvailabilityProvider,
    PipelineResult,
};
use alloc::{boxed::Box, fmt::Debug};
use alloy_primitives::{Address, Bytes};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;

/// A factory for creating an Ethereum data source provider.
#[derive(Debug, Clone)]
pub struct EthereumDataSource<C, B>
where
    C: ChainProvider + Send + Clone,
    B: BlobProvider + Send + Clone,
{
    /// The ecotone timestamp.
    pub ecotone_timestamp: Option<u64>,
    /// The blob source.
    pub blob_source: BlobSource<C, B>,
    /// The calldata source.
    pub calldata_source: CalldataSource<C>,
}

impl<C, B> EthereumDataSource<C, B>
where
    C: ChainProvider + Send + Clone + Debug,
    B: BlobProvider + Send + Clone + Debug,
{
    /// Instantiates a new [`EthereumDataSource`].
    pub const fn new(
        blob_source: BlobSource<C, B>,
        calldata_source: CalldataSource<C>,
        cfg: &RollupConfig,
    ) -> Self {
        Self { ecotone_timestamp: cfg.hardforks.ecotone_time, blob_source, calldata_source }
    }

    /// Instantiates a new [`EthereumDataSource`] from parts.
    pub fn new_from_parts(provider: C, blobs: B, cfg: &RollupConfig) -> Self {
        Self {
            ecotone_timestamp: cfg.hardforks.ecotone_time,
            blob_source: BlobSource::new(provider.clone(), blobs, cfg.batch_inbox_address),
            calldata_source: CalldataSource::new(provider, cfg.batch_inbox_address),
        }
    }
}

#[async_trait]
impl<C, B> DataAvailabilityProvider for EthereumDataSource<C, B>
where
    C: ChainProvider + Send + Sync + Clone + Debug,
    B: BlobProvider + Send + Sync + Clone + Debug,
{
    type Item = Bytes;

    async fn next(
        &mut self,
        block_ref: &BlockInfo,
        batcher_address: Address,
    ) -> PipelineResult<Self::Item> {
        let ecotone_enabled =
            self.ecotone_timestamp.map(|e| block_ref.timestamp >= e).unwrap_or(false);
        if ecotone_enabled {
            self.blob_source.next(block_ref, batcher_address).await
        } else {
            self.calldata_source.next(block_ref, batcher_address).await
        }
    }

    fn clear(&mut self) {
        self.blob_source.clear();
        self.calldata_source.clear();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        BlobData,
        test_utils::{TestBlobProvider, TestChainProvider},
    };
    use alloc::vec;
    use alloy_consensus::TxEnvelope;
    use alloy_eips::eip2718::Decodable2718;
    use alloy_primitives::{Address, address};
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::BlockInfo;

    fn default_test_blob_source() -> BlobSource<TestChainProvider, TestBlobProvider> {
        let chain_provider = TestChainProvider::default();
        let blob_fetcher = TestBlobProvider::default();
        let batcher_address = Address::default();
        BlobSource::new(chain_provider, blob_fetcher, batcher_address)
    }

    #[tokio::test]
    async fn test_clear_ethereum_data_source() {
        let chain = TestChainProvider::default();
        let blob = TestBlobProvider::default();
        let cfg = RollupConfig::default();
        let mut calldata = CalldataSource::new(chain.clone(), Address::ZERO);
        calldata.calldata.insert(0, Default::default());
        calldata.open = true;
        let mut blob = BlobSource::new(chain, blob, Address::ZERO);
        blob.data = vec![Default::default()];
        blob.open = true;
        let mut data_source = EthereumDataSource::new(blob, calldata, &cfg);

        data_source.clear();
        assert!(data_source.blob_source.data.is_empty());
        assert!(!data_source.blob_source.open);
        assert!(data_source.calldata_source.calldata.is_empty());
        assert!(!data_source.calldata_source.open);
    }

    #[tokio::test]
    async fn test_open_blob_source() {
        let chain = TestChainProvider::default();
        let mut blob = default_test_blob_source();
        blob.open = true;
        blob.data.push(BlobData { data: None, calldata: Some(Bytes::default()) });
        let calldata = CalldataSource::new(chain.clone(), Address::ZERO);
        let cfg = RollupConfig {
            hardforks: HardForkConfig { ecotone_time: Some(0), ..Default::default() },
            ..Default::default()
        };

        // Should successfully retrieve a blob batch from the block
        let mut data_source = EthereumDataSource::new(blob, calldata, &cfg);
        let data = data_source.next(&BlockInfo::default(), Address::ZERO).await.unwrap();
        assert_eq!(data, Bytes::default());
    }

    #[tokio::test]
    async fn test_open_ethereum_calldata_source_pre_ecotone() {
        let mut chain = TestChainProvider::default();
        let blob = TestBlobProvider::default();
        let batcher_address = address!("6887246668a3b87F54DeB3b94Ba47a6f63F32985");
        let batch_inbox = address!("FF00000000000000000000000000000000000010");
        let block_ref = BlockInfo { number: 10, ..Default::default() };

        let mut cfg = RollupConfig::default();
        cfg.genesis.system_config = Some(SystemConfig { batcher_address, ..Default::default() });
        cfg.batch_inbox_address = batch_inbox;

        // load a test batcher transaction
        let raw_batcher_tx = include_bytes!("../../testdata/raw_batcher_tx.hex");
        let tx = TxEnvelope::decode_2718(&mut raw_batcher_tx.as_ref()).unwrap();
        chain.insert_block_with_transactions(10, block_ref, vec![tx]);

        // Should successfully retrieve a calldata batch from the block
        let mut data_source = EthereumDataSource::new_from_parts(chain, blob, &cfg);
        let calldata_batch = data_source.next(&block_ref, batcher_address).await.unwrap();
        assert_eq!(calldata_batch.len(), 119823);
    }
}
