//! Defines the `WitnessExecutor` trait for constructing and running the derivation pipeline.

use std::{fmt::Debug, sync::Arc};

use alloy_op_evm::{block::OpAlloyReceiptBuilder, post_exec::PostExecEvmFactoryAdapter};
use alloy_primitives::B256;
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
    BootInfo, FlushableCache, executor::KonaExecutor, l1::OraclePipeline, l2::OracleL2ChainProvider,
};
use spin::RwLock;
use tracing::info;

use crate::{
    metrics::CycleTrackerDriverMetrics,
    precompiles::{CustomCrypto, ZkvmOpEvmFactory},
};

/// The execution target and expected output of a block in a derivation segment.
#[derive(Debug, Clone, Copy)]
pub struct BlockClaim {
    /// The L2 block number that execution must reach.
    pub block_number: u64,
    /// The expected output root at the target block.
    pub output_root: B256,
}

impl From<&BootInfo> for BlockClaim {
    fn from(boot: &BootInfo) -> Self {
        Self {
            block_number: boot.claimed_l2_block_number,
            output_root: boot.claimed_l2_output_root,
        }
    }
}

/// A nonempty sequence of claims sharing one derivation driver.
#[derive(Debug)]
pub struct SegmentClaims {
    /// Initializes the driver and supplies its first claim.
    pub first: BootInfo,
    /// Subsequent claims, in execution order, using the first boot's configuration.
    pub following: Vec<BlockClaim>,
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
        claims: &SegmentClaims,
        pipeline: DP,
        cursor: Arc<RwLock<PipelineCursor>>,
        l2_provider: OracleL2ChainProvider<O>,
    ) -> Result<()>
    where
        O: CommsClient + FlushableCache + Send + Sync + Debug,
        DP: DriverPipeline<P> + Send + Sync + Debug,
        P: Pipeline + SignalReceiver + Send + Sync + Debug,
    {
        let boot = &claims.first;
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
        let first_claim = BlockClaim::from(boot);
        for claim in std::iter::once(first_claim).chain(claims.following.iter().copied()) {
            #[cfg(target_os = "zkvm")]
            println!("cycle-tracker-report-start: block-execution-and-derivation");
            let (safe_head, output_root) = driver
                .advance_to_target_with_metrics(
                    rollup_config.as_ref(),
                    Some(claim.block_number),
                    &CycleTrackerDriverMetrics,
                )
                .await?;
            #[cfg(target_os = "zkvm")]
            println!("cycle-tracker-report-end: block-execution-and-derivation");

            if output_root != claim.output_root {
                return Err(anyhow!(
                    "Failed to validate L2 block #{number} with claimed output root {claimed_output_root}. Got {output_root} instead",
                    number = safe_head.block_info.number,
                    output_root = output_root,
                    claimed_output_root = claim.output_root,
                ));
            }

            // Bind the claimed block number to the actual derived safe-head number. Without this
            // check, a non-interop EndOfSource that triggers the silent target downgrade in
            // advance_to_target can let an adversarial witness pair a claimed output root and
            // block number that refer to different L2 blocks. See GHSA-5jh4-3p33-85xc.
            ensure_derived_block_matches_claim(safe_head.block_info.number, claim.block_number)?;

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
    use alloy_primitives::B256;
    use kona_derive::{
        EthereumDataSource, OriginProvider, PipelineError, PipelineErrorKind, Signal, StepResult,
    };
    use kona_driver::TipCursor;
    use kona_genesis::SystemConfig;
    use kona_proof::{block_on, l1::OracleL1ChainProvider};
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

    fn run_claims(claims: &SegmentClaims) -> Result<()> {
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
            claims,
            ExhaustedPipeline::default(),
            Arc::new(RwLock::new(cursor)),
            provider,
        ))
    }

    fn matching_claims() -> SegmentClaims {
        SegmentClaims {
            first: BootInfo {
                l1_head: B256::ZERO,
                agreed_l2_output_root: b256(1),
                claimed_l2_output_root: b256(1),
                claimed_l2_block_number: 3,
                chain_id: 10,
                rollup_config: RollupConfig::default(),
                l1_config: L1ChainConfig::default(),
            },
            following: vec![BlockClaim { block_number: 3, output_root: b256(1) }; 2],
        }
    }

    #[test]
    fn executor_rejects_incorrect_output_root() {
        for index in 0..3 {
            let mut claims = matching_claims();
            if index == 0 {
                claims.first.claimed_l2_output_root = b256(2);
            } else {
                claims.following[index - 1].output_root = b256(2);
            }
            let err = run_claims(&claims).unwrap_err();
            assert!(err.to_string().contains("Failed to validate L2 block #3"), "{err}");
        }
    }

    #[test]
    fn executor_rejects_exhausted_pipeline_below_claimed_block() {
        for index in 0..3 {
            let mut claims = matching_claims();
            if index == 0 {
                claims.first.claimed_l2_block_number = 4;
            } else {
                claims.following[index - 1].block_number = 4;
            }
            let err = run_claims(&claims).unwrap_err();
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
        let mut claims = matching_claims();
        run_claims(&claims).unwrap();
        claims.following.clear();
        run_claims(&claims).unwrap();
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
}
