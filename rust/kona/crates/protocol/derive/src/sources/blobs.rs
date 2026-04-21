//! Blob Data Source

use crate::{
    BlobData, BlobProvider, ChainProvider, DataAvailabilityProvider, PipelineError,
    PipelineErrorKind, PipelineResult, ResetError,
};
use alloc::{boxed::Box, vec::Vec};
use alloy_consensus::{
    Transaction, TxEip4844Variant, TxEnvelope, TxType, transaction::SignerRecoverable,
};
use alloy_primitives::{Address, B256, Bytes};
use async_trait::async_trait;
use kona_protocol::BlockInfo;

/// A data iterator that reads from a blob.
#[derive(Debug, Clone)]
pub struct BlobSource<F, B>
where
    F: ChainProvider + Send,
    B: BlobProvider + Send,
{
    /// Chain provider.
    pub chain_provider: F,
    /// Fetches blobs.
    pub blob_fetcher: B,
    /// The address of the batcher contract.
    pub batcher_address: Address,
    /// Data.
    pub data: Vec<BlobData>,
    /// Whether the source is open.
    pub open: bool,
}

impl<F, B> BlobSource<F, B>
where
    F: ChainProvider + Send,
    B: BlobProvider + Send,
{
    /// Creates a new blob source.
    pub const fn new(chain_provider: F, blob_fetcher: B, batcher_address: Address) -> Self {
        Self { chain_provider, blob_fetcher, batcher_address, data: Vec::new(), open: false }
    }

    fn extract_blob_data(
        &self,
        txs: Vec<TxEnvelope>,
        batcher_address: Address,
    ) -> (Vec<BlobData>, Vec<B256>) {
        let mut data = Vec::new();
        let mut hashes = Vec::new();
        for tx in txs {
            let (tx_kind, calldata, blob_hashes) = match &tx {
                TxEnvelope::Legacy(tx) => (tx.tx().to(), tx.tx().input.clone(), None),
                TxEnvelope::Eip2930(tx) => (tx.tx().to(), tx.tx().input.clone(), None),
                TxEnvelope::Eip1559(tx) => (tx.tx().to(), tx.tx().input.clone(), None),
                TxEnvelope::Eip4844(blob_tx_wrapper) => match blob_tx_wrapper.tx() {
                    TxEip4844Variant::TxEip4844(tx) => {
                        (tx.to(), tx.input.clone(), Some(tx.blob_versioned_hashes.clone()))
                    }
                    TxEip4844Variant::TxEip4844WithSidecar(tx) => {
                        let tx = tx.tx();
                        (tx.to(), tx.input.clone(), Some(tx.blob_versioned_hashes.clone()))
                    }
                },
                _ => continue,
            };
            let Some(to) = tx_kind else { continue };

            if to != self.batcher_address {
                continue;
            }
            if tx.recover_signer().unwrap_or_default() != batcher_address {
                continue;
            }
            if tx.tx_type() != TxType::Eip4844 {
                let blob_data = BlobData { data: None, calldata: Some(calldata.to_vec().into()) };
                data.push(blob_data);
                continue;
            }
            if !calldata.is_empty() {
                let hash = match &tx {
                    TxEnvelope::Legacy(tx) => Some(tx.hash()),
                    TxEnvelope::Eip2930(tx) => Some(tx.hash()),
                    TxEnvelope::Eip1559(tx) => Some(tx.hash()),
                    TxEnvelope::Eip4844(blob_tx_wrapper) => Some(blob_tx_wrapper.hash()),
                    _ => None,
                };
                warn!(target: "blob_source", "Blob tx has calldata, which will be ignored: {hash:?}");
            }
            let blob_hashes = if let Some(b) = blob_hashes {
                b
            } else {
                continue;
            };
            for hash in blob_hashes {
                hashes.push(hash);
                data.push(BlobData::default());
            }
        }
        #[cfg(feature = "metrics")]
        metrics::gauge!(
            crate::metrics::Metrics::PIPELINE_DATA_AVAILABILITY_PROVIDER,
            "source" => "blobs",
        )
        .increment(data.len() as f64);
        (data, hashes)
    }

    /// Loads blob data into the source if it is not open.
    async fn load_blobs(
        &mut self,
        block_ref: &BlockInfo,
        batcher_address: Address,
    ) -> Result<(), PipelineErrorKind> {
        if self.open {
            return Ok(());
        }

        let info = self
            .chain_provider
            .block_info_and_transactions_by_hash(block_ref.hash)
            .await
            .map_err(Into::into)?;

        let (mut data, blob_hashes) = self.extract_blob_data(info.1, batcher_address);

        // If there are no hashes, set the calldata and return.
        if blob_hashes.is_empty() {
            self.open = true;
            self.data = data;
            return Ok(());
        }

        // Convert via Into<PipelineErrorKind> which routes:
        //   BlobNotFound  -> PipelineErrorKind::Reset   (missed/orphaned slot)
        //   Backend       -> PipelineErrorKind::Temporary (transient, retry)
        //   others        -> PipelineErrorKind::Critical
        let blobs = self
            .blob_fetcher
            .get_and_validate_blobs(block_ref, &blob_hashes)
            .await
            .map_err(Into::<PipelineErrorKind>::into)
            .inspect_err(|kind| match kind {
                PipelineErrorKind::Reset(_) => {
                    warn!(
                        target: "blob_source",
                        block_hash = %block_ref.hash,
                        block_number = block_ref.number,
                        timestamp = block_ref.timestamp,
                        "Blobs permanently unavailable (missed/orphaned beacon slot); \
                         triggering pipeline reset"
                    );
                }
                _ => {
                    warn!(
                        target: "blob_source",
                        block_hash = %block_ref.hash,
                        block_number = block_ref.number,
                        timestamp = block_ref.timestamp,
                        "Failed to fetch blobs: {kind}"
                    );
                }
            })?;

        // Fill the blob pointers.
        let mut filled_blobs = 0;
        for blob in &mut data {
            let should_increment = blob.fill(&blobs, filled_blobs)?;
            if should_increment {
                filled_blobs += 1;
            }
        }

        // Post-loop over-fill check: if the provider returned more blobs than were
        // requested, the pipeline state is inconsistent. Reset so the pipeline retries
        // from a clean state.
        if filled_blobs < blobs.len() {
            return Err(
                ResetError::BlobsOverFill { filled: filled_blobs, returned: blobs.len() }.reset()
            );
        }

        self.open = true;
        self.data = data;
        Ok(())
    }

    /// Extracts the next data from the source.
    fn next_data(&mut self) -> PipelineResult<BlobData> {
        if self.data.is_empty() {
            return Err(PipelineError::Eof.temp());
        }

        Ok(self.data.remove(0))
    }
}

#[async_trait]
impl<F, B> DataAvailabilityProvider for BlobSource<F, B>
where
    F: ChainProvider + Sync + Send,
    B: BlobProvider + Sync + Send,
{
    type Item = Bytes;

    async fn next(
        &mut self,
        block_ref: &BlockInfo,
        batcher_address: Address,
    ) -> PipelineResult<Self::Item> {
        self.load_blobs(block_ref, batcher_address).await?;

        let next_data = self.next_data()?;
        if let Some(c) = next_data.calldata {
            return Ok(c);
        }

        // Decode the blob data to raw bytes.
        // Otherwise, ignore blob and recurse next.
        match next_data.decode() {
            Ok(d) => Ok(d),
            Err(_) => {
                warn!(target: "blob_source", "Failed to decode blob data, skipping");
                self.next(block_ref, batcher_address).await
            }
        }
    }

    fn clear(&mut self) {
        self.data.clear();
        self.open = false;
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::{
        errors::PipelineErrorKind,
        test_utils::{TestBlobProvider, TestChainProvider},
    };
    use alloc::vec;
    use alloy_rlp::Decodable;

    pub(crate) fn default_test_blob_source() -> BlobSource<TestChainProvider, TestBlobProvider> {
        let chain_provider = TestChainProvider::default();
        let blob_fetcher = TestBlobProvider::default();
        let batcher_address = Address::default();
        BlobSource::new(chain_provider, blob_fetcher, batcher_address)
    }

    pub(crate) fn valid_blob_txs() -> Vec<TxEnvelope> {
        // https://sepolia.etherscan.io/getRawTx?tx=0x9a22ccb0029bc8b0ddd073be1a1d923b7ae2b2ea52100bae0db4424f9107e9c0
        let raw_tx = alloy_primitives::hex::decode("0x03f9011d83aa36a7820fa28477359400852e90edd0008252089411e9ca82a3a762b4b5bd264d4173a242e7a770648080c08504a817c800f8a5a0012ec3d6f66766bedb002a190126b3549fce0047de0d4c25cffce0dc1c57921aa00152d8e24762ff22b1cfd9f8c0683786a7ca63ba49973818b3d1e9512cd2cec4a0013b98c6c83e066d5b14af2b85199e3d4fc7d1e778dd53130d180f5077e2d1c7a001148b495d6e859114e670ca54fb6e2657f0cbae5b08063605093a4b3dc9f8f1a0011ac212f13c5dff2b2c6b600a79635103d6f580a4221079951181b25c7e654901a0c8de4cced43169f9aa3d36506363b2d2c44f6c49fc1fd91ea114c86f3757077ea01e11fdd0d1934eda0492606ee0bb80a7bf8f35cc5f86ec60fe5031ba48bfd544").unwrap();
        let eip4844 = TxEnvelope::decode(&mut raw_tx.as_slice()).unwrap();
        vec![eip4844]
    }

    #[tokio::test]
    async fn test_load_blobs_open() {
        let mut source = default_test_blob_source();
        source.open = true;
        assert!(source.load_blobs(&BlockInfo::default(), Address::ZERO).await.is_ok());
    }

    #[tokio::test]
    async fn test_load_blobs_chain_provider_err() {
        let mut source = default_test_blob_source();
        assert!(matches!(
            source.load_blobs(&BlockInfo::default(), Address::ZERO).await,
            Err(PipelineErrorKind::Temporary(_))
        ));
    }

    #[tokio::test]
    async fn test_load_blobs_chain_provider_empty_txs() {
        let mut source = default_test_blob_source();
        let block_info = BlockInfo::default();
        source.chain_provider.insert_block_with_transactions(0, block_info, Vec::new());
        assert!(!source.open); // Source is not open by default.
        assert!(source.load_blobs(&BlockInfo::default(), Address::ZERO).await.is_ok());
        assert!(source.data.is_empty());
        assert!(source.open);
    }

    #[tokio::test]
    async fn test_load_blobs_chain_provider_4844_txs_blob_fetch_error() {
        let mut source = default_test_blob_source();
        let block_info = BlockInfo::default();
        let batcher_address =
            alloy_primitives::address!("A83C816D4f9b2783761a22BA6FADB0eB0606D7B2");
        source.batcher_address =
            alloy_primitives::address!("11E9CA82A3a762b4B5bd264d4173a242e7a77064");
        let txs = valid_blob_txs();
        source.blob_fetcher.should_error = true;
        source.chain_provider.insert_block_with_transactions(1, block_info, txs);
        assert!(matches!(
            source.load_blobs(&BlockInfo::default(), batcher_address).await,
            Err(PipelineErrorKind::Critical(_))
        ));
    }

    #[tokio::test]
    async fn test_load_blobs_chain_provider_4844_txs_succeeds() {
        use alloy_consensus::Blob;

        let mut source = default_test_blob_source();
        let block_info = BlockInfo::default();
        let batcher_address =
            alloy_primitives::address!("A83C816D4f9b2783761a22BA6FADB0eB0606D7B2");
        source.batcher_address =
            alloy_primitives::address!("11E9CA82A3a762b4B5bd264d4173a242e7a77064");
        let txs = valid_blob_txs();
        source.chain_provider.insert_block_with_transactions(1, block_info, txs);
        let hashes = [
            alloy_primitives::b256!(
                "012ec3d6f66766bedb002a190126b3549fce0047de0d4c25cffce0dc1c57921a"
            ),
            alloy_primitives::b256!(
                "0152d8e24762ff22b1cfd9f8c0683786a7ca63ba49973818b3d1e9512cd2cec4"
            ),
            alloy_primitives::b256!(
                "013b98c6c83e066d5b14af2b85199e3d4fc7d1e778dd53130d180f5077e2d1c7"
            ),
            alloy_primitives::b256!(
                "01148b495d6e859114e670ca54fb6e2657f0cbae5b08063605093a4b3dc9f8f1"
            ),
            alloy_primitives::b256!(
                "011ac212f13c5dff2b2c6b600a79635103d6f580a4221079951181b25c7e6549"
            ),
        ];
        for hash in hashes {
            source.blob_fetcher.insert_blob(hash, Blob::with_last_byte(1u8));
        }
        source.load_blobs(&BlockInfo::default(), batcher_address).await.unwrap();
        assert!(source.open);
        assert!(!source.data.is_empty());
    }

    #[tokio::test]
    async fn test_open_empty_data_eof() {
        let mut source = default_test_blob_source();
        source.open = true;

        let err = source.next(&BlockInfo::default(), Address::ZERO).await.unwrap_err();
        assert!(matches!(err, PipelineErrorKind::Temporary(PipelineError::Eof)));
    }

    #[tokio::test]
    async fn test_open_calldata() {
        let mut source = default_test_blob_source();
        source.open = true;
        source.data.push(BlobData { data: None, calldata: Some(Bytes::default()) });

        let data = source.next(&BlockInfo::default(), Address::ZERO).await.unwrap();
        assert_eq!(data, Bytes::default());
    }

    #[tokio::test]
    async fn test_open_blob_data_decode_missing_data() {
        let mut source = default_test_blob_source();
        source.open = true;
        source.data.push(BlobData { data: Some(Bytes::from(&[1; 32])), calldata: None });

        let err = source.next(&BlockInfo::default(), Address::ZERO).await.unwrap_err();
        assert!(matches!(err, PipelineErrorKind::Temporary(PipelineError::Eof)));
    }

    #[tokio::test]
    async fn test_blob_source_pipeline_error() {
        let mut source = default_test_blob_source();
        let err = source.next(&BlockInfo::default(), Address::ZERO).await.unwrap_err();
        assert!(matches!(err, PipelineErrorKind::Temporary(PipelineError::Provider(_))));
    }

    /// Regression test: a beacon node 404 (missed/orphaned slot) must propagate through
    /// `load_blobs` as `PipelineErrorKind::Reset`, not as a temporary retryable error.
    #[tokio::test]
    async fn test_load_blobs_not_found_triggers_reset() {
        let mut source = default_test_blob_source();
        let block_info = BlockInfo::default();
        let batcher_address =
            alloy_primitives::address!("A83C816D4f9b2783761a22BA6FADB0eB0606D7B2");
        source.batcher_address =
            alloy_primitives::address!("11E9CA82A3a762b4B5bd264d4173a242e7a77064");
        source.chain_provider.insert_block_with_transactions(1, block_info, valid_blob_txs());
        source.blob_fetcher.should_return_not_found = true;

        let err = source.load_blobs(&BlockInfo::default(), batcher_address).await.unwrap_err();
        assert!(
            matches!(err, PipelineErrorKind::Reset(_)),
            "expected Reset for missed beacon slot, got {err:?}"
        );
    }

    /// Regression test: `BlobProviderError::BlobNotFound` from the blob fetcher must surface
    /// through `next()` as `PipelineErrorKind::Reset`, triggering a pipeline reset.
    /// Without this, a missed beacon slot causes an infinite retry loop and safe head stall.
    #[tokio::test]
    async fn test_missed_beacon_slot_triggers_pipeline_reset() {
        let mut source = default_test_blob_source();
        let block_info = BlockInfo::default();
        let batcher_address =
            alloy_primitives::address!("A83C816D4f9b2783761a22BA6FADB0eB0606D7B2");
        source.batcher_address =
            alloy_primitives::address!("11E9CA82A3a762b4B5bd264d4173a242e7a77064");
        source.chain_provider.insert_block_with_transactions(1, block_info, valid_blob_txs());
        source.blob_fetcher.should_return_not_found = true;

        let err = source.next(&BlockInfo::default(), batcher_address).await.unwrap_err();
        assert!(
            matches!(err, PipelineErrorKind::Reset(_)),
            "expected Reset for missed beacon slot, got {err:?}"
        );
    }

    /// A minimal [`ChainProvider`] that always returns a "block not found" error which maps to
    /// [`PipelineErrorKind::Reset`].  Used to verify that [`BlobSource::load_blobs`] preserves
    /// the `Reset` kind when the underlying chain provider signals that a block is missing (e.g.
    /// after an L1 reorg removes the block whose hash was referenced).
    #[derive(Debug, Clone, Default)]
    struct BlockNotFoundChainProvider;

    /// Error type for [`BlockNotFoundChainProvider`] that converts to
    /// [`PipelineErrorKind::Reset`], matching what `AlloyChainProvider` emits for 404 responses.
    #[derive(Debug)]
    struct BlockNotFoundError;

    impl core::fmt::Display for BlockNotFoundError {
        fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
            write!(f, "block not found")
        }
    }

    impl From<BlockNotFoundError> for PipelineErrorKind {
        fn from(_: BlockNotFoundError) -> Self {
            use crate::ResetError;
            ResetError::BlockNotFound(alloy_primitives::B256::default().into()).reset()
        }
    }

    #[async_trait::async_trait]
    impl ChainProvider for BlockNotFoundChainProvider {
        type Error = BlockNotFoundError;

        async fn header_by_hash(
            &mut self,
            _: alloy_primitives::B256,
        ) -> Result<alloy_consensus::Header, Self::Error> {
            Err(BlockNotFoundError)
        }

        async fn block_info_by_number(
            &mut self,
            _: u64,
        ) -> Result<kona_protocol::BlockInfo, Self::Error> {
            Err(BlockNotFoundError)
        }

        async fn receipts_by_hash(
            &mut self,
            _: alloy_primitives::B256,
        ) -> Result<alloc::vec::Vec<alloy_consensus::Receipt>, Self::Error> {
            Err(BlockNotFoundError)
        }

        async fn block_info_and_transactions_by_hash(
            &mut self,
            _: alloy_primitives::B256,
        ) -> Result<(kona_protocol::BlockInfo, alloc::vec::Vec<TxEnvelope>), Self::Error> {
            Err(BlockNotFoundError)
        }
    }

    /// Regression test: when `block_info_and_transactions_by_hash` returns an error that maps to
    /// `PipelineErrorKind::Reset` (e.g. because an L1 reorg removed the block), `load_blobs`
    /// must propagate the `Reset` kind unchanged.
    ///
    /// Before the fix, `BlobSource` wrapped every chain-provider error as
    /// `BlobProviderError::Backend(e.to_string()).into()`, which unconditionally produces
    /// `PipelineErrorKind::Temporary`.  The fix uses `.map_err(Into::into)` so the `Reset` kind
    /// set by the underlying provider is preserved, allowing the pipeline to recover via reset
    /// rather than spinning in a retry loop.
    #[tokio::test]
    async fn test_load_blobs_block_not_found_triggers_reset() {
        let chain_provider = BlockNotFoundChainProvider;
        let blob_fetcher = crate::test_utils::TestBlobProvider::default();
        let mut source = BlobSource::new(chain_provider, blob_fetcher, Address::ZERO);

        let err = source.load_blobs(&BlockInfo::default(), Address::ZERO).await.unwrap_err();
        assert!(
            matches!(err, PipelineErrorKind::Reset(_)),
            "expected Reset when block_info_and_transactions_by_hash returns BlockNotFound, \
             got {err:?}"
        );
    }

    /// Regression test: when the blob provider returns more blobs than were requested
    /// (over-fill), `load_blobs` must return `PipelineErrorKind::Reset` rather than
    /// silently discarding the extra blobs.
    /// Over-fill can occur with buggy providers or in rare L1 reorg scenarios.
    #[tokio::test]
    async fn test_load_blobs_overfill_triggers_reset() {
        use alloy_consensus::Blob;

        let mut source = default_test_blob_source();
        let block_info = BlockInfo::default();
        let batcher_address =
            alloy_primitives::address!("A83C816D4f9b2783761a22BA6FADB0eB0606D7B2");
        source.batcher_address =
            alloy_primitives::address!("11E9CA82A3a762b4B5bd264d4173a242e7a77064");
        let txs = valid_blob_txs();
        source.chain_provider.insert_block_with_transactions(1, block_info, txs);
        // Insert blobs for all the real hashes so fill does not under-fill first.
        let hashes = [
            alloy_primitives::b256!(
                "012ec3d6f66766bedb002a190126b3549fce0047de0d4c25cffce0dc1c57921a"
            ),
            alloy_primitives::b256!(
                "0152d8e24762ff22b1cfd9f8c0683786a7ca63ba49973818b3d1e9512cd2cec4"
            ),
            alloy_primitives::b256!(
                "013b98c6c83e066d5b14af2b85199e3d4fc7d1e778dd53130d180f5077e2d1c7"
            ),
            alloy_primitives::b256!(
                "01148b495d6e859114e670ca54fb6e2657f0cbae5b08063605093a4b3dc9f8f1"
            ),
            alloy_primitives::b256!(
                "011ac212f13c5dff2b2c6b600a79635103d6f580a4221079951181b25c7e6549"
            ),
        ];
        for hash in hashes {
            source.blob_fetcher.insert_blob(hash, Blob::with_last_byte(1u8));
        }
        // Instruct the mock provider to return one extra blob beyond what was requested.
        source.blob_fetcher.should_return_extra_blob = true;

        let err = source.load_blobs(&BlockInfo::default(), batcher_address).await.unwrap_err();
        assert!(
            matches!(err, PipelineErrorKind::Reset(_)),
            "expected Reset for blob over-fill, got {err:?}"
        );
    }
}
