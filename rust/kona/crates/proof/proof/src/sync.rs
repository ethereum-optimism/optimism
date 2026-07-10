//! Sync Start

use crate::{HintType, errors::OracleProviderError};
use alloc::sync::Arc;
use alloy_consensus::{Header, Sealed};
use alloy_primitives::B256;
use core::fmt::Debug;
use kona_derive::ChainProvider;
use kona_driver::{PipelineCursor, TipCursor};
use kona_preimage::{CommsClient, PreimageKey};
use kona_protocol::BatchValidationProvider;
use kona_registry::RollupConfig;
use spin::RwLock;

/// Constructs a [`PipelineCursor`] from the caching oracle, boot info, and providers.
pub async fn new_oracle_pipeline_cursor<L1, L2>(
    rollup_config: &RollupConfig,
    safe_header: Sealed<Header>,
    agreed_l2_output_root: B256,
    chain_provider: &mut L1,
    l2_chain_provider: &mut L2,
) -> Result<Arc<RwLock<PipelineCursor>>, OracleProviderError>
where
    L1: ChainProvider + Send + Sync + Debug + Clone,
    L2: BatchValidationProvider + Send + Sync + Debug + Clone,
    OracleProviderError:
        From<<L1 as ChainProvider>::Error> + From<<L2 as BatchValidationProvider>::Error>,
{
    let safe_head_info = l2_chain_provider.l2_block_info_by_number(safe_header.number).await?;
    let l1_origin = chain_provider.block_info_by_number(safe_head_info.l1_origin.number).await?;

    // Walk back the starting L1 block by `channel_timeout` to ensure that the full channel is
    // captured.
    let channel_timeout = rollup_config.channel_timeout(safe_head_info.block_info.timestamp);
    let mut l1_origin_number = l1_origin.number.saturating_sub(channel_timeout);
    if l1_origin_number < rollup_config.genesis.l1.number {
        l1_origin_number = rollup_config.genesis.l1.number;
    }
    let origin = chain_provider.block_info_by_number(l1_origin_number).await?;

    // Construct the cursor.
    let mut cursor = PipelineCursor::new(channel_timeout, origin);
    let tip = TipCursor::new(safe_head_info, safe_header, agreed_l2_output_root);
    cursor.advance(origin, tip);

    // Wrap the cursor in a shared read-write lock
    Ok(Arc::new(RwLock::new(cursor)))
}

/// Fetches the safe head hash of the L2 chain based on the agreed upon L2 output root in the
/// [`BootInfo`](crate::BootInfo).
///
/// Decodes the agreed L2 output root's preimage into its safe head block hash. The preimage is the
/// 128-byte output-root V0 layout; the safe head hash occupies bytes `[96..128]`. Only version `0`
/// (all-zero version word in bytes `[0..32]`) is accepted.
pub async fn fetch_safe_head_hash<O>(
    caching_oracle: &O,
    agreed_l2_output_root: B256,
) -> Result<B256, OracleProviderError>
where
    O: CommsClient,
{
    let mut output_preimage = [0u8; 128];
    HintType::StartingL2Output
        .with_data(&[agreed_l2_output_root.as_ref()])
        .send(caching_oracle)
        .await?;
    caching_oracle
        .get_exact(PreimageKey::new_keccak256(*agreed_l2_output_root), output_preimage.as_mut())
        .await?;

    if output_preimage[..32] != [0u8; 32] {
        return Err(OracleProviderError::UnknownOutputVersion(B256::from_slice(
            &output_preimage[..32],
        )));
    }

    output_preimage[96..128].try_into().map_err(OracleProviderError::SliceConversion)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::{boxed::Box, collections::BTreeMap, vec, vec::Vec};
    use async_trait::async_trait;
    use kona_preimage::{
        HintWriterClient, PreimageKeyType, PreimageOracleClient,
        errors::{PreimageOracleError, PreimageOracleResult},
    };

    /// A minimal in-memory [`CommsClient`] serving a single preimage, used to drive
    /// [`fetch_safe_head_hash`] in tests.
    #[derive(Clone, Default)]
    struct MockOracle {
        preimages: BTreeMap<PreimageKey, Vec<u8>>,
    }

    impl MockOracle {
        fn single(key: PreimageKey, value: Vec<u8>) -> Self {
            let mut preimages = BTreeMap::new();
            preimages.insert(key, value);
            Self { preimages }
        }
    }

    #[async_trait]
    impl PreimageOracleClient for MockOracle {
        async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            Ok(self.preimages.get(&key).expect("missing preimage in mock").clone())
        }

        async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
            let v = self.get(key).await?;
            if v.len() != buf.len() {
                return Err(PreimageOracleError::BufferLengthMismatch(buf.len(), v.len()));
            }
            buf.copy_from_slice(&v);
            Ok(())
        }
    }

    #[async_trait]
    impl HintWriterClient for MockOracle {
        async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
            Ok(())
        }
    }

    fn b256(fill: u8) -> B256 {
        B256::from([fill; 32])
    }

    #[tokio::test]
    async fn fetch_safe_head_hash_returns_safe_head_from_v0_preimage() {
        let agreed_root = b256(0xAA);
        let safe_head = b256(0xBB);
        let mut preimage = [0u8; 128];
        preimage[96..128].copy_from_slice(safe_head.as_slice());
        let oracle =
            MockOracle::single(PreimageKey::new(*agreed_root, PreimageKeyType::Keccak256), preimage.to_vec());

        let got = fetch_safe_head_hash(&oracle, agreed_root).await.unwrap();
        assert_eq!(got, safe_head);
    }

    #[tokio::test]
    async fn fetch_safe_head_hash_rejects_unknown_output_version() {
        let agreed_root = b256(0xAA);
        let mut bad_preimage = [0u8; 128];
        bad_preimage[0] = 0x01; // non-V0 version word; [96..128] stays zero
        let oracle = MockOracle::single(
            PreimageKey::new(*agreed_root, PreimageKeyType::Keccak256),
            bad_preimage.to_vec(),
        );

        let err = fetch_safe_head_hash(&oracle, agreed_root).await.unwrap_err();
        match err {
            OracleProviderError::UnknownOutputVersion(version) => assert_eq!(version[0], 0x01),
            other => panic!("unexpected error: {other:?}"),
        }
    }

    #[tokio::test]
    async fn fetch_safe_head_hash_rejects_wrong_preimage_length() {
        // `get_exact` with a `[u8; 128]` buffer is what enforces the length here; this test guards
        // against a future refactor swapping to `get`.
        let agreed_root = b256(0xAA);
        let oracle =
            MockOracle::single(PreimageKey::new(*agreed_root, PreimageKeyType::Keccak256), vec![0u8; 127]);

        let err = fetch_safe_head_hash(&oracle, agreed_root).await.unwrap_err();
        match err {
            OracleProviderError::Preimage(PreimageOracleError::BufferLengthMismatch(
                expected,
                actual,
            )) => {
                assert_eq!(expected, 128);
                assert_eq!(actual, 127);
            }
            other => panic!("unexpected error: {other:?}"),
        }
    }
}
