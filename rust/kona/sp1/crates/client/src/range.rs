//! Shared range-program execution logic.

use std::sync::Arc;

use alloy_primitives::B256;
use anyhow::{Result, anyhow};
use kona_proof::{l1::OracleL1ChainProvider, l2::OracleL2ChainProvider};

use crate::{
    BlobStore,
    boot::BootInfoStruct,
    witness::{
        executor::{WitnessExecutor, get_inputs_for_pipeline},
        preimage_store::PreimageStore,
    },
};

/// Executes the range program with the given [`WitnessExecutor`], [`PreimageStore`], and
/// [`BlobStore`].
pub async fn run_range_program<E>(
    executor: E,
    oracle: Arc<PreimageStore>,
    beacon: BlobStore,
) -> Result<BootInfoStruct>
where
    E: WitnessExecutor<
            O = PreimageStore,
            B = BlobStore,
            L1 = OracleL1ChainProvider<PreimageStore>,
            L2 = OracleL2ChainProvider<PreimageStore>,
        > + Send
        + Sync,
{
    let (boot_info, input) = get_inputs_for_pipeline(oracle.clone()).await?;
    let boot_info = match input {
        Some((cursor, l1_provider, l2_provider)) => {
            let rollup_config = Arc::new(boot_info.rollup_config.clone());
            let l1_config = Arc::new(boot_info.l1_config.clone());

            let pipeline = executor
                .create_pipeline(
                    rollup_config,
                    l1_config,
                    cursor.clone(),
                    oracle,
                    beacon,
                    l1_provider,
                    l2_provider.clone(),
                )
                .await?;

            executor.run(boot_info, pipeline, cursor, l2_provider).await?
        }
        None => boot_info,
    };

    Ok(BootInfoStruct::from(boot_info))
}

/// Ensures a committed range-program output matches the requested claim.
pub fn ensure_committed_boot_info_matches_claim(
    boot: &BootInfoStruct,
    claimed_output_root: B256,
    claimed_block_number: u64,
) -> Result<()> {
    if boot.l2PostRoot != claimed_output_root || boot.l2BlockNumber != claimed_block_number {
        return Err(anyhow!(
            "range program committed L2 root {committed_root} at block #{committed_block}, but requested {claimed_root} at block #{claimed_block}",
            committed_root = boot.l2PostRoot,
            committed_block = boot.l2BlockNumber,
            claimed_root = claimed_output_root,
            claimed_block = claimed_block_number,
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use alloy_consensus::Header;
    use alloy_primitives::{Address, B256, Bytes, keccak256};
    use alloy_rlp::Encodable;
    use anyhow::Result;
    use async_trait::async_trait;
    use kona_derive::{DataAvailabilityProvider, PipelineResult};
    use kona_driver::PipelineCursor;
    use kona_genesis::{L1ChainConfig, RollupConfig};
    use kona_preimage::{
        L1_HEAD_KEY, L2_CHAIN_ID_KEY, L2_CLAIM_BLOCK_NUMBER_KEY, L2_CLAIM_KEY, L2_OUTPUT_ROOT_KEY,
        PreimageKey,
    };
    use kona_proof::{
        block_on,
        l1::{OracleL1ChainProvider, OraclePipeline},
        l2::OracleL2ChainProvider,
    };
    use kona_protocol::BlockInfo;
    use spin::RwLock;

    use super::{ensure_committed_boot_info_matches_claim, run_range_program};
    use crate::{
        BlobStore,
        witness::{executor::WitnessExecutor, preimage_store::PreimageStore},
    };

    #[derive(Clone, Debug)]
    struct UnusedDataAvailabilityProvider;

    #[async_trait]
    impl DataAvailabilityProvider for UnusedDataAvailabilityProvider {
        type Item = Bytes;

        async fn next(
            &mut self,
            _block_ref: &BlockInfo,
            _batcher_addr: Address,
        ) -> PipelineResult<Self::Item> {
            unreachable!("zero-step range tests must not construct a data availability provider")
        }

        fn clear(&mut self) {}
    }

    #[derive(Debug)]
    struct UnusedExecutor;

    #[async_trait]
    impl WitnessExecutor for UnusedExecutor {
        type O = PreimageStore;
        type B = BlobStore;
        type L1 = OracleL1ChainProvider<Self::O>;
        type L2 = OracleL2ChainProvider<Self::O>;
        type DA = UnusedDataAvailabilityProvider;

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
            unreachable!("zero-step range tests must not construct a pipeline")
        }
    }

    fn b256(fill: u8) -> B256 {
        B256::from([fill; 32])
    }

    fn zero_step_oracle(claimed_root: B256) -> (Arc<PreimageStore>, B256, u64) {
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

        (Arc::new(oracle), agreed_root, safe_head.number)
    }

    #[test]
    fn run_range_program_commits_zero_step_claim() {
        let (_, agreed_root, _) = zero_step_oracle(B256::ZERO);
        let (oracle, agreed_root, safe_head_number) = zero_step_oracle(agreed_root);

        let boot = block_on(run_range_program(UnusedExecutor, oracle, BlobStore::default()))
            .expect("zero-step range program should succeed");

        assert_eq!(boot.l2PostRoot, agreed_root);
        assert_eq!(boot.l2BlockNumber, safe_head_number);
    }

    #[test]
    fn run_range_program_rejects_zero_step_root_mismatch() {
        let (oracle, agreed_root, _) = zero_step_oracle(B256::ZERO);
        assert_ne!(agreed_root, B256::ZERO);

        let err = block_on(run_range_program(UnusedExecutor, oracle, BlobStore::default()))
            .expect_err("zero-step root mismatch must fail");

        assert!(err.to_string().contains("does not match agreed output root"));
    }

    #[test]
    fn committed_boot_info_must_match_requested_claim() {
        let root = b256(0x22);
        let block_number = 7;
        let (_, agreed_root, _) = zero_step_oracle(B256::ZERO);
        let (oracle, _, _) = zero_step_oracle(agreed_root);
        let boot = block_on(run_range_program(UnusedExecutor, oracle, BlobStore::default()))
            .expect("zero-step range program should succeed");

        let err = ensure_committed_boot_info_matches_claim(&boot, root, block_number)
            .expect_err("mismatched committed boot info must fail");
        let msg = err.to_string();

        assert!(msg.contains(&root.to_string()), "missing claimed root in: {msg}");
        assert!(msg.contains("#7"), "missing claimed block number in: {msg}");
    }
}
