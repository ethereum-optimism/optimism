//! Defines the `WitnessExecutor` trait for constructing and running the derivation pipeline.

use std::{fmt::Debug, sync::Arc};

use alloy_op_evm::{block::OpAlloyReceiptBuilder, post_exec::PostExecEvmFactoryAdapter};
use anyhow::{Result, anyhow};
use async_trait::async_trait;
use kona_derive::{
    BlobProvider, ChainProvider, DataAvailabilityProvider, L2ChainProvider, Pipeline,
    SignalReceiver,
};
use kona_driver::{Driver, DriverPipeline, PipelineCursor};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_preimage::CommsClient;
use kona_proof::{
    BootInfo, FlushableCache,
    executor::KonaExecutor,
    l1::{OracleL1ChainProvider, OraclePipeline},
    l2::OracleL2ChainProvider,
    sync::{DerivationInputs, prepare_derivation},
};
use spin::RwLock;
use tracing::info;

use crate::{
    metrics::CycleTrackerDriverMetrics,
    precompiles::{CustomCrypto, ZkvmOpEvmFactory},
};

/// Gets the inputs for constructing the derivation pipeline.
pub async fn get_inputs_for_pipeline<O>(
    oracle: Arc<O>,
) -> Result<(
    BootInfo,
    Option<(Arc<RwLock<PipelineCursor>>, OracleL1ChainProvider<O>, OracleL2ChainProvider<O>)>,
)>
where
    O: CommsClient + FlushableCache + Send + Sync + Debug,
{
    let boot = match BootInfo::load(oracle.as_ref()).await {
        Ok(boot) => boot,
        Err(e) => {
            return Err(anyhow!("Failed to load boot info: {e:?}"));
        }
    };

    let rollup_config = Arc::new(boot.rollup_config.clone());

    // Run the shared sync-start prologue. A trace-extension claim needs no derivation; otherwise we
    // get back the cursor and providers to build the pipeline over.
    match prepare_derivation(&boot, rollup_config, oracle).await? {
        DerivationInputs::TraceExtension => Ok((boot, None)),
        DerivationInputs::Derive { cursor, l1_provider, l2_provider } => {
            Ok((boot, Some((cursor, l1_provider, l2_provider))))
        }
    }
}

/// The [`WitnessExecutor`] trait defines an interface for constructing and running the derivation
/// pipeline.
#[async_trait]
pub trait WitnessExecutor {
    /// The [`CommsClient`] type used by the executor.
    type O: CommsClient + FlushableCache + Send + Sync + Debug;

    /// The [`BlobProvider`] type used by the executor.
    type B: BlobProvider + Send + Sync + Debug + Clone;

    /// The L1 [`ChainProvider`] type used by the executor.
    type L1: ChainProvider + Send + Sync + Debug + Clone;

    /// The [`L2ChainProvider`] type used by the executor.
    type L2: L2ChainProvider + Send + Sync + Debug + Clone;

    /// The [`DataAvailabilityProvider`] type used by the executor.
    type DA: DataAvailabilityProvider + Send + Sync + Debug + Clone;

    /// Constructs the derivation pipeline.
    #[allow(clippy::too_many_arguments)]
    async fn create_pipeline(
        &self,
        rollup_config: Arc<RollupConfig>,
        l1_config: Arc<L1ChainConfig>,
        cursor: Arc<RwLock<PipelineCursor>>,
        oracle: Arc<Self::O>,
        beacon: Self::B,
        l1_provider: Self::L1,
        l2_provider: Self::L2,
    ) -> Result<OraclePipeline<Self::O, Self::L1, Self::L2, Self::DA>>;

    /// Validates ordered, contiguous claims for one chain using a single derivation driver.
    async fn run<O, DP, P>(
        &self,
        boots: &[BootInfo],
        pipeline: DP,
        cursor: Arc<RwLock<PipelineCursor>>,
        l2_provider: OracleL2ChainProvider<O>,
    ) -> Result<()>
    where
        O: CommsClient + FlushableCache + Send + Sync + Debug,
        DP: DriverPipeline<P> + Send + Sync + Debug,
        P: Pipeline + SignalReceiver + Send + Sync + Debug,
    {
        let Some(boot) = boots.first() else {
            return Ok(());
        };
        // Install custom crypto provider for KZG point evaluation precompile
        revm::precompile::install_crypto(CustomCrypto::default());

        let rollup_config = Arc::new(boot.rollup_config.clone());

        let evm_factory = PostExecEvmFactoryAdapter::new(ZkvmOpEvmFactory);
        let executor = KonaExecutor::new(
            rollup_config.as_ref(),
            l2_provider.clone(),
            l2_provider,
            evm_factory,
            OpAlloyReceiptBuilder::default(),
            None,
        );
        let mut driver = Driver::new(cursor, executor, pipeline);
        for boot in boots {
            #[cfg(target_os = "zkvm")]
            println!("cycle-tracker-report-start: block-execution-and-derivation");
            let (safe_head, output_root) = driver
                .advance_to_target_with_metrics(
                    rollup_config.as_ref(),
                    Some(boot.claimed_l2_block_number),
                    &CycleTrackerDriverMetrics,
                )
                .await?;
            #[cfg(target_os = "zkvm")]
            println!("cycle-tracker-report-end: block-execution-and-derivation");

            if output_root != boot.claimed_l2_output_root {
                return Err(anyhow!(
                    "Failed to validate L2 block #{number} with claimed output root {claimed_output_root}. Got {output_root} instead",
                    number = safe_head.block_info.number,
                    output_root = output_root,
                    claimed_output_root = boot.claimed_l2_output_root,
                ));
            }

            // Bind the committed l2BlockNumber to the actual derived safe-head number. Without this
            // check, a non-interop EndOfSource that triggers the silent target downgrade in
            // advance_to_target can let an adversarial witness commit (l2PostRoot, l2BlockNumber)
            // pairs that refer to different L2 blocks. See GHSA-5jh4-3p33-85xc.
            ensure_derived_block_matches_claim(
                safe_head.block_info.number,
                boot.claimed_l2_block_number,
            )?;

            info!(
                target: "client",
                "Successfully validated L2 block #{number} with output root {output_root}",
                number = safe_head.block_info.number,
                output_root = output_root
            );
        }

        #[cfg(target_os = "zkvm")]
        {
            std::mem::forget(driver);
            std::mem::forget(rollup_config);
        }

        Ok(())
    }
}

/// Ensures the derived L2 safe-head block number matches the boot's claimed L2 block number.
///
/// This is the postcondition that closes GHSA-5jh4-3p33-85xc: a non-interop `EndOfSource` inside
/// `advance_to_target` silently downgrades the local target to the current safe head, so a
/// successful return does not by itself prove the requested target was reached.
fn ensure_derived_block_matches_claim(
    safe_head_number: u64,
    claimed_block_number: u64,
) -> Result<()> {
    if safe_head_number != claimed_block_number {
        return Err(anyhow!(
            "Derived safe head L2 block #{derived} does not match claimed L2 block number #{claimed}",
            derived = safe_head_number,
            claimed = claimed_block_number,
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use alloy_consensus::{Header, Sealed};
    use alloy_primitives::{B256, keccak256};
    use alloy_rlp::Encodable;
    use kona_derive::{
        EthereumDataSource, OriginProvider, PipelineError, PipelineErrorKind, Signal, StepResult,
    };
    use kona_driver::TipCursor;
    use kona_genesis::SystemConfig;
    use kona_preimage::{
        L1_HEAD_KEY, L2_CHAIN_ID_KEY, L2_CLAIM_BLOCK_NUMBER_KEY, L2_CLAIM_KEY, L2_OUTPUT_ROOT_KEY,
        PreimageKey,
    };
    use kona_proof::block_on;
    use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};

    use super::*;
    use crate::{BlobStore, witness::preimage_store::PreimageStore};

    struct TestWitnessExecutor;

    #[async_trait]
    impl WitnessExecutor for TestWitnessExecutor {
        type O = PreimageStore;
        type B = BlobStore;
        type L1 = OracleL1ChainProvider<PreimageStore>;
        type L2 = OracleL2ChainProvider<PreimageStore>;
        type DA = EthereumDataSource<Self::L1, Self::B>;

        async fn create_pipeline(
            &self,
            _rollup_config: Arc<RollupConfig>,
            _l1_config: Arc<L1ChainConfig>,
            _cursor: Arc<RwLock<PipelineCursor>>,
            _oracle: Arc<Self::O>,
            _beacon: Self::B,
            _l1_provider: Self::L1,
            _l2_provider: Self::L2,
        ) -> Result<OraclePipeline<Self::O, Self::L1, Self::L2, Self::DA>> {
            unreachable!("validation tests supply their pipeline directly")
        }
    }

    #[derive(Debug, Default)]
    struct ExhaustedPipeline {
        config: RollupConfig,
    }

    impl Iterator for ExhaustedPipeline {
        type Item = OpAttributesWithParent;

        fn next(&mut self) -> Option<Self::Item> {
            None
        }
    }

    impl OriginProvider for ExhaustedPipeline {
        fn origin(&self) -> Option<BlockInfo> {
            None
        }
    }

    #[async_trait]
    impl SignalReceiver for ExhaustedPipeline {
        async fn signal(&mut self, _signal: Signal) -> Result<(), PipelineErrorKind> {
            unreachable!("exhaustion does not reset the pipeline")
        }
    }

    #[async_trait]
    impl Pipeline for ExhaustedPipeline {
        fn peek(&self) -> Option<&OpAttributesWithParent> {
            None
        }

        async fn step(&mut self, _cursor: L2BlockInfo) -> StepResult {
            StepResult::StepFailed(PipelineError::EndOfSource.crit())
        }

        fn rollup_config(&self) -> &RollupConfig {
            &self.config
        }

        async fn system_config_by_l2_hash(
            &mut self,
            _hash: B256,
        ) -> Result<SystemConfig, PipelineErrorKind> {
            unreachable!("exhaustion does not load system config")
        }
    }

    impl DriverPipeline<Self> for ExhaustedPipeline {
        fn flush(&mut self) {
            unreachable!("exhaustion does not flush the pipeline")
        }
    }

    fn run_claims(boots: &[BootInfo]) -> Result<()> {
        let header = Header { number: 3, ..Default::default() };
        let head = L2BlockInfo {
            block_info: BlockInfo { number: 3, ..Default::default() },
            ..Default::default()
        };
        let mut cursor = PipelineCursor::new(1, BlockInfo::default());
        cursor.advance(BlockInfo::default(), TipCursor::new(head, Sealed::new(header), b256(1)));
        let provider = OracleL2ChainProvider::new(
            B256::ZERO,
            Arc::new(RollupConfig::default()),
            Arc::new(PreimageStore::default()),
        );
        block_on(TestWitnessExecutor.run(
            boots,
            ExhaustedPipeline::default(),
            Arc::new(RwLock::new(cursor)),
            provider,
        ))
    }

    fn claim(block_number: u64, output_root: B256) -> BootInfo {
        BootInfo {
            l1_head: B256::ZERO,
            agreed_l2_output_root: b256(1),
            claimed_l2_output_root: output_root,
            claimed_l2_block_number: block_number,
            chain_id: 10,
            rollup_config: RollupConfig::default(),
            l1_config: L1ChainConfig::default(),
        }
    }

    #[test]
    fn executor_rejects_incorrect_output_root() {
        for index in 0..3 {
            let mut boots = vec![claim(3, b256(1)); 3];
            boots[index].claimed_l2_output_root = b256(2);
            let err = run_claims(&boots).unwrap_err();
            assert!(err.to_string().contains("Failed to validate L2 block #3"), "{err}");
        }
    }

    #[test]
    fn executor_rejects_exhausted_pipeline_below_claimed_block() {
        for index in 0..3 {
            let mut boots = vec![claim(3, b256(1)); 3];
            boots[index].claimed_l2_block_number = 4;
            let err = run_claims(&boots).unwrap_err();
            assert!(
                err.to_string().contains(
                    "Derived safe head L2 block #3 does not match claimed L2 block number #4"
                ),
                "{err}"
            );
        }
    }

    #[test]
    fn executor_accepts_matching_claims() {
        run_claims(&vec![claim(3, b256(1)); 3]).unwrap();
    }

    #[test]
    fn derived_block_must_match_claimed_block() {
        assert!(super::ensure_derived_block_matches_claim(0, 0).is_ok());
        assert!(super::ensure_derived_block_matches_claim(123_456, 123_456).is_ok());
        assert!(super::ensure_derived_block_matches_claim(u64::MAX, u64::MAX).is_ok());

        let err = super::ensure_derived_block_matches_claim(50, 100)
            .expect_err("mismatched derived and claimed block numbers must fail");
        let msg = err.to_string();
        assert!(msg.contains("#50"), "missing derived block number in: {msg}");
        assert!(msg.contains("#100"), "missing claimed block number in: {msg}");

        let err = super::ensure_derived_block_matches_claim(150, 100)
            .expect_err("mismatched derived and claimed block numbers must fail");
        let msg = err.to_string();
        assert!(msg.contains("#150"), "missing derived block number in: {msg}");
        assert!(msg.contains("#100"), "missing claimed block number in: {msg}");
    }

    fn b256(fill: u8) -> B256 {
        B256::from([fill; 32])
    }

    fn zero_step_oracle(claimed_root: B256) -> (Arc<PreimageStore>, B256) {
        let safe_head = Header { number: 3, ..Default::default() };
        let safe_head_hash = safe_head.hash_slow();

        let mut output_preimage = [0u8; 128];
        output_preimage[96..128].copy_from_slice(safe_head_hash.as_slice());
        let agreed_root = B256::from(keccak256(output_preimage));

        let mut oracle = PreimageStore::default();
        oracle
            .save_preimage(PreimageKey::new_local(L1_HEAD_KEY.to()), b256(0x11).as_slice().to_vec())
            .unwrap();
        oracle
            .save_preimage(
                PreimageKey::new_local(L2_OUTPUT_ROOT_KEY.to()),
                agreed_root.as_slice().to_vec(),
            )
            .unwrap();
        oracle
            .save_preimage(
                PreimageKey::new_local(L2_CLAIM_KEY.to()),
                claimed_root.as_slice().to_vec(),
            )
            .unwrap();
        oracle
            .save_preimage(
                PreimageKey::new_local(L2_CLAIM_BLOCK_NUMBER_KEY.to()),
                safe_head.number.to_be_bytes().to_vec(),
            )
            .unwrap();
        oracle
            .save_preimage(
                PreimageKey::new_local(L2_CHAIN_ID_KEY.to()),
                10u64.to_be_bytes().to_vec(),
            )
            .unwrap();
        oracle
            .save_preimage(PreimageKey::new_keccak256(*agreed_root), output_preimage.to_vec())
            .unwrap();

        let mut header_rlp = Vec::new();
        safe_head.encode(&mut header_rlp);
        oracle.save_preimage(PreimageKey::new_keccak256(*safe_head_hash), header_rlp).unwrap();

        (Arc::new(oracle), agreed_root)
    }

    #[test]
    fn get_inputs_for_pipeline_short_circuits_zero_step_claim() {
        let (_, agreed_root) = zero_step_oracle(B256::ZERO);
        let (oracle, _) = zero_step_oracle(agreed_root);

        let (boot, input) = block_on(get_inputs_for_pipeline(oracle)).unwrap();

        assert_eq!(boot.claimed_l2_output_root, agreed_root);
        assert!(input.is_none());
    }

    #[test]
    fn get_inputs_for_pipeline_rejects_zero_step_root_mismatch() {
        let (oracle, agreed_root) = zero_step_oracle(B256::ZERO);
        assert_ne!(agreed_root, B256::ZERO);

        let err = block_on(get_inputs_for_pipeline(oracle)).unwrap_err();
        assert!(err.to_string().contains("does not match agreed output root"));
    }
}
