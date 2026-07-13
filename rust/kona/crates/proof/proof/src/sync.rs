//! Sync Start

use crate::{
    BootInfo, HintType, errors::OracleProviderError, l1::OracleL1ChainProvider,
    l2::OracleL2ChainProvider,
};
use alloc::sync::Arc;
use alloy_consensus::{Header, Sealed};
use alloy_primitives::B256;
use core::fmt::Debug;
use kona_derive::ChainProvider;
use kona_driver::{PipelineCursor, TipCursor};
use kona_executor::TrieDBProvider;
use kona_preimage::{CommsClient, PreimageKey};
use kona_protocol::BatchValidationProvider;
use kona_registry::RollupConfig;
use spin::RwLock;
use thiserror::Error;

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
/// [`BootInfo`].
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

/// An error raised by [`prepare_derivation`] while validating a claim and building the derivation
/// inputs during sync start.
///
/// Distinguishes oracle/data failures from claim-validation failures so each caller can map the
/// variant onto its own error type (e.g. an `InvalidClaim` for the native fault-proof program).
#[derive(Error, Debug)]
pub enum SyncStartError {
    /// An oracle-backed provider or preimage lookup failed.
    #[error(transparent)]
    Oracle(#[from] OracleProviderError),
    /// The claimed L2 block number is behind the agreed safe head, so the claim cannot be valid.
    #[error("Claimed L2 block number {claimed} is less than the safe head {safe_head}")]
    ClaimedBlockBeforeSafeHead {
        /// The claimed (disputed) L2 block number.
        claimed: u64,
        /// The agreed safe head L2 block number.
        safe_head: u64,
    },
    /// The claim targets the safe head block itself (a zero-step transition), but the claimed
    /// output root does not match the agreed output root.
    #[error(
        "Claimed output root {claimed} does not match agreed output root {agreed} at safe head {safe_head}"
    )]
    ClaimedRootMismatch {
        /// The claimed (disputed) L2 output root.
        claimed: B256,
        /// The agreed L2 output root.
        agreed: B256,
        /// The safe head L2 block number the claim targets.
        safe_head: u64,
    },
}

/// The outcome of [`prepare_derivation`]: either the claim is already agreed (no derivation needed)
/// or the derivation inputs required to run the pipeline.
#[derive(Debug)]
pub enum DerivationInputs<O>
where
    O: CommsClient,
{
    /// The claim targets the current safe head and matches the agreed output root: a zero-step
    /// "trace extension" transition that requires no derivation.
    TraceExtension,
    /// The claim is ahead of the safe head; derivation must run over these inputs.
    Derive {
        /// The initialized derivation pipeline cursor.
        cursor: Arc<RwLock<PipelineCursor>>,
        /// The oracle-backed L1 chain provider.
        l1_provider: OracleL1ChainProvider<O>,
        /// The oracle-backed L2 chain provider, with its cursor set.
        l2_provider: OracleL2ChainProvider<O>,
    },
}

/// Runs the sync-start prologue shared by the native fault-proof program and kona-sp1.
///
/// Given loaded boot info and an oracle, this:
/// 1. resolves the agreed safe head hash and header,
/// 2. rejects a claim whose block number is behind the safe head,
/// 3. short-circuits a zero-step "trace extension" claim (claim == safe head), and
/// 4. otherwise builds the pipeline cursor and returns the L1/L2 providers ready for derivation.
///
/// Callers map [`SyncStartError`] onto their own error type. This is the single source of truth for
/// the claim-validation and trace-extension logic that both derivation paths depend on.
pub async fn prepare_derivation<O>(
    boot: &BootInfo,
    rollup_config: Arc<RollupConfig>,
    oracle: Arc<O>,
) -> Result<DerivationInputs<O>, SyncStartError>
where
    O: CommsClient + Send + Sync + Debug,
{
    let safe_head_hash = fetch_safe_head_hash(oracle.as_ref(), boot.agreed_l2_output_root).await?;

    let mut l1_provider = OracleL1ChainProvider::new(boot.l1_head, oracle.clone());
    let mut l2_provider =
        OracleL2ChainProvider::new(safe_head_hash, rollup_config.clone(), oracle.clone());

    // Fetch the safe head's block header.
    let safe_head = l2_provider
        .header_by_hash(safe_head_hash)
        .map(|header| Sealed::new_unchecked(header, safe_head_hash))?;

    // If the claimed L2 block number is less than the safe head of the L2 chain, the claim is
    // invalid.
    if boot.claimed_l2_block_number < safe_head.number {
        return Err(SyncStartError::ClaimedBlockBeforeSafeHead {
            claimed: boot.claimed_l2_block_number,
            safe_head: safe_head.number,
        });
    }

    // If the claim targets the safe head block itself, then no derivation is required. This can
    // happen at trace-extension leaves where the trace is capped at the root-claim block number.
    // In this case, the only valid output root is the agreed output root (a zero-step transition).
    if boot.claimed_l2_block_number == safe_head.number {
        if boot.claimed_l2_output_root != boot.agreed_l2_output_root {
            return Err(SyncStartError::ClaimedRootMismatch {
                claimed: boot.claimed_l2_output_root,
                agreed: boot.agreed_l2_output_root,
                safe_head: safe_head.number,
            });
        }

        info!(
            target: "client",
            "Trace extension detected. State transition is already agreed upon.",
        );
        return Ok(DerivationInputs::TraceExtension);
    }

    // Create a new derivation pipeline cursor from the agreed safe head.
    let cursor = new_oracle_pipeline_cursor(
        rollup_config.as_ref(),
        safe_head,
        boot.agreed_l2_output_root,
        &mut l1_provider,
        &mut l2_provider,
    )
    .await?;
    l2_provider.set_cursor(cursor.clone());

    Ok(DerivationInputs::Derive { cursor, l1_provider, l2_provider })
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::{boxed::Box, collections::BTreeMap, string::ToString, vec, vec::Vec};
    use alloy_primitives::keccak256;
    use alloy_rlp::Encodable;
    use async_trait::async_trait;
    use kona_genesis::L1ChainConfig;
    use kona_preimage::{
        HintWriterClient, PreimageKeyType, PreimageOracleClient,
        errors::{PreimageOracleError, PreimageOracleResult},
    };

    /// A minimal in-memory [`CommsClient`] serving a single preimage, used to drive
    /// [`fetch_safe_head_hash`] in tests.
    #[derive(Clone, Debug, Default)]
    struct MockOracle {
        preimages: BTreeMap<PreimageKey, Vec<u8>>,
    }

    impl MockOracle {
        fn single(key: PreimageKey, value: Vec<u8>) -> Self {
            let mut preimages = BTreeMap::new();
            preimages.insert(key, value);
            Self { preimages }
        }

        fn insert(&mut self, key: PreimageKey, value: Vec<u8>) {
            self.preimages.insert(key, value);
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
        let oracle = MockOracle::single(
            PreimageKey::new(*agreed_root, PreimageKeyType::Keccak256),
            preimage.to_vec(),
        );

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
        let oracle = MockOracle::single(
            PreimageKey::new(*agreed_root, PreimageKeyType::Keccak256),
            vec![0u8; 127],
        );

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

    /// Builds a [`MockOracle`] that serves the agreed output-root preimage (encoding `safe_head`'s
    /// hash) and the safe-head header RLP, plus the matching [`BootInfo`]. `claimed_block` and
    /// `claimed_root` drive which [`prepare_derivation`] branch is exercised.
    fn prologue_fixture(
        safe_head_number: u64,
        claimed_block: u64,
        claimed_root: B256,
    ) -> (MockOracle, BootInfo) {
        let safe_head = Header { number: safe_head_number, ..Default::default() };
        let safe_head_hash = safe_head.hash_slow();

        let mut output_preimage = [0u8; 128];
        output_preimage[96..128].copy_from_slice(safe_head_hash.as_slice());
        let agreed_root = B256::from(keccak256(output_preimage));

        let mut oracle =
            MockOracle::single(PreimageKey::new_keccak256(*agreed_root), output_preimage.to_vec());
        let mut header_rlp = Vec::new();
        safe_head.encode(&mut header_rlp);
        oracle.insert(PreimageKey::new_keccak256(*safe_head_hash), header_rlp);

        let boot = BootInfo {
            l1_head: b256(0x11),
            agreed_l2_output_root: agreed_root,
            claimed_l2_output_root: claimed_root,
            claimed_l2_block_number: claimed_block,
            chain_id: 10,
            rollup_config: RollupConfig::default(),
            l1_config: L1ChainConfig::default(),
        };

        (oracle, boot)
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn prepare_derivation_short_circuits_trace_extension() {
        // Claim targets the safe head itself and matches the agreed root: zero-step transition.
        let (oracle, boot) = prologue_fixture(3, 3, B256::ZERO);
        let boot = BootInfo { claimed_l2_output_root: boot.agreed_l2_output_root, ..boot };
        let rollup_config = Arc::new(boot.rollup_config.clone());

        let out = prepare_derivation(&boot, rollup_config, Arc::new(oracle)).await.unwrap();
        assert!(matches!(out, DerivationInputs::TraceExtension));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn prepare_derivation_rejects_trace_extension_root_mismatch() {
        // Claim targets the safe head but the claimed root differs from the agreed root.
        let (oracle, boot) = prologue_fixture(3, 3, b256(0xCC));
        assert_ne!(boot.claimed_l2_output_root, boot.agreed_l2_output_root);
        let rollup_config = Arc::new(boot.rollup_config.clone());

        let err = prepare_derivation(&boot, rollup_config, Arc::new(oracle)).await.unwrap_err();
        assert!(matches!(err, SyncStartError::ClaimedRootMismatch { safe_head: 3, .. }));
        assert!(err.to_string().contains("does not match agreed output root"));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn prepare_derivation_rejects_claim_before_safe_head() {
        // Claim targets a block behind the safe head: the claim cannot be valid.
        let (oracle, boot) = prologue_fixture(3, 2, b256(0xCC));
        let rollup_config = Arc::new(boot.rollup_config.clone());

        let err = prepare_derivation(&boot, rollup_config, Arc::new(oracle)).await.unwrap_err();
        assert!(matches!(
            err,
            SyncStartError::ClaimedBlockBeforeSafeHead { claimed: 2, safe_head: 3 }
        ));
    }
}
