//! Single chain sub-transition phase of the interop proof.

use super::FaultProofProgramError;
use crate::interop::util::fetch_l2_safe_head_hash;
use alloc::{boxed::Box, sync::Arc};
use alloy_consensus::Sealed;
use alloy_evm::{EvmFactory, FromRecoveredTx, FromTxWithEncoded};
use alloy_op_evm::block::OpTxEnv;
use alloy_primitives::B256;
use core::fmt::Debug;
use kona_derive::{EthereumDataSource, PipelineError, PipelineErrorKind};
use kona_driver::{Driver, DriverError};
use kona_executor::TrieDBProvider;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use kona_proof::{
    CachingOracle,
    executor::KonaExecutor,
    l1::{OracleBlobProvider, OracleL1ChainProvider, OraclePipeline},
    l2::OracleL2ChainProvider,
    sync::new_oracle_pipeline_cursor,
};
use kona_proof_interop::{BootInfo, INVALID_TRANSITION_HASH, OptimisticBlock, PreState};
use op_alloy_consensus::OpTxEnvelope;
use op_revm::OpSpecId;
use revm::context::BlockEnv;
use tracing::{error, info, warn};

/// Executes a sub-transition of the interop proof with the given [PreimageOracleClient] and
/// [HintWriterClient].
pub(crate) async fn sub_transition<P, H, Evm>(
    oracle: Arc<CachingOracle<P, H>>,
    boot: BootInfo,
    evm_factory: Evm,
) -> Result<(), FaultProofProgramError>
where
    P: PreimageOracleClient + Send + Sync + Debug + Clone,
    H: HintWriterClient + Send + Sync + Debug + Clone,
    Evm: EvmFactory<Spec = OpSpecId, BlockEnv = BlockEnv> + Send + Sync + Debug + Clone + 'static,
    <Evm as EvmFactory>::Tx:
        FromTxWithEncoded<OpTxEnvelope> + FromRecoveredTx<OpTxEnvelope> + OpTxEnv,
{
    // Check if we can short-circuit the transition, if we are within padding.
    if let PreState::TransitionState(ref transition_state) = boot.agreed_pre_state {
        if transition_state.step >= transition_state.pre_state.output_roots.len() as u64 {
            info!(
                target: "interop_client",
                "No derivation/execution required, transition state is already saturated."
            );

            return transition_and_check(boot.agreed_pre_state, None, boot.claimed_post_state);
        }
    }

    // Fetch the L2 block hash of the current safe head.
    let safe_head_hash = fetch_l2_safe_head_hash(oracle.as_ref(), &boot.agreed_pre_state).await?;

    // Determine the active L2 chain ID and fetch the rollup configuration.
    let rollup_config = boot
        .active_rollup_config()
        .map(Arc::new)
        .ok_or(FaultProofProgramError::StateTransitionFailed)?;

    let l1_config = boot.active_l1_config();

    // Instantiate the L1 EL + CL provider and the L2 EL provider.
    let mut l1_provider = OracleL1ChainProvider::new(boot.l1_head, oracle.clone());
    let mut l2_provider =
        OracleL2ChainProvider::new(safe_head_hash, rollup_config.clone(), oracle.clone());
    let beacon = OracleBlobProvider::new(oracle.clone());

    // Set the active L2 chain ID for the L2 provider.
    l2_provider.set_chain_id(boot.agreed_pre_state.active_l2_chain_id());

    // Fetch the safe head's block header.
    let safe_head = l2_provider
        .header_by_hash(safe_head_hash)
        .map(|header| Sealed::new_unchecked(header, safe_head_hash))?;
    let disputed_l2_block_number = safe_head.number + 1;

    // Check if we can no-op the transition. The Superchain STF happens once every second, but
    // chains have a variable block time, meaning there might be no transition to process.
    if safe_head.timestamp + rollup_config.block_time > boot.agreed_pre_state.timestamp() + 1 {
        info!(
            target: "interop_client",
            "No-op transition, short-circuiting."
        );

        let active_root = boot
            .agreed_pre_state
            .active_l2_output_root()
            .ok_or(FaultProofProgramError::StateTransitionFailed)?;
        let optimistic_block = OptimisticBlock::new(safe_head.hash(), active_root.output_root);
        return transition_and_check(
            boot.agreed_pre_state,
            Some(optimistic_block),
            boot.claimed_post_state,
        );
    }

    // Create a new derivation driver with the given boot information and oracle.
    let cursor = new_oracle_pipeline_cursor(
        rollup_config.as_ref(),
        safe_head,
        &mut l1_provider,
        &mut l2_provider,
    )
    .await?;
    l2_provider.set_cursor(cursor.clone());

    let da_provider =
        EthereumDataSource::new_from_parts(l1_provider.clone(), beacon, &rollup_config);
    let pipeline = OraclePipeline::new(
        rollup_config.clone(),
        l1_config.into(),
        cursor.clone(),
        oracle.clone(),
        da_provider,
        l1_provider.clone(),
        l2_provider.clone(),
    )
    .await
    .map_err(Box::new)?;
    let executor = KonaExecutor::new(
        rollup_config.as_ref(),
        l2_provider.clone(),
        l2_provider,
        evm_factory,
        None,
    );
    let mut driver = Driver::new(cursor, executor, pipeline);

    // Run the derivation pipeline until we are able to produce the output root of the claimed
    // L2 block.
    match driver.advance_to_target(rollup_config.as_ref(), Some(disputed_l2_block_number)).await {
        Ok((safe_head, output_root)) => {
            let optimistic_block = OptimisticBlock::new(safe_head.block_info.hash, output_root);
            transition_and_check(
                boot.agreed_pre_state,
                Some(optimistic_block),
                boot.claimed_post_state,
            )?;

            info!(
                target: "interop_client",
                "Successfully validated progressed transition state claim with commitment {post_state_commitment}",
                post_state_commitment = boot.claimed_post_state
            );

            Ok(())
        }
        Err(DriverError::Pipeline(PipelineErrorKind::Critical(PipelineError::EndOfSource))) => {
            warn!(
                target: "interop_client",
                "Exhausted data source; Transitioning to invalid state."
            );

            (boot.claimed_post_state == INVALID_TRANSITION_HASH).then_some(()).ok_or(
                FaultProofProgramError::InvalidClaim(
                    INVALID_TRANSITION_HASH,
                    boot.claimed_post_state,
                ),
            )
        }
        Err(e) => {
            error!(
                target: "interop_client",
                "Failed to advance derivation pipeline: {:?}",
                e
            );
            Err(Box::new(e).into())
        }
    }
}

/// Transitions the [PreState] with the given [OptimisticBlock] and checks if the resulting state
/// commitment matches the expected commitment.
fn transition_and_check(
    pre_state: PreState,
    optimistic_block: Option<OptimisticBlock>,
    expected_post_state: B256,
) -> Result<(), FaultProofProgramError> {
    let did_append = optimistic_block.is_some();
    let post_state = pre_state
        .transition(optimistic_block)
        .ok_or(FaultProofProgramError::StateTransitionFailed)?;
    let post_state_commitment = post_state.hash();

    if did_append {
        info!(
            target: "interop_client",
            "Appended optimistic L2 block to transition state",
        );
    }

    if post_state_commitment != expected_post_state {
        error!(
            target: "interop_client",
            "Failed to validate progressed transition state. Expected post-state commitment: {expected}, actual: {actual}",
            expected = expected_post_state,
            actual = post_state_commitment
        );

        return Err(FaultProofProgramError::InvalidClaim(
            expected_post_state,
            post_state_commitment,
        ));
    }

    info!(
        target: "interop_client",
        "Successfully validated progressed transition state with commitment {post_state_commitment}",
    );

    Ok(())
}
