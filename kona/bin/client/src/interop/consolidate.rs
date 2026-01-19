//! Consolidation phase of the interop proof program.

use super::FaultProofProgramError;
use crate::interop::util::fetch_output_block_hash;
use alloc::sync::Arc;
use alloy_evm::{EvmFactory, FromRecoveredTx, FromTxWithEncoded};
use alloy_op_evm::block::OpTxEnv;
use core::fmt::Debug;
use kona_executor::TrieDBProvider;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use kona_proof::{CachingOracle, l2::OracleL2ChainProvider};
use kona_proof_interop::{
    BootInfo, HintType, OracleInteropProvider, PreState, SuperchainConsolidator,
};
use kona_registry::{HashMap, ROLLUP_CONFIGS};
use op_alloy_consensus::OpTxEnvelope;
use op_revm::OpSpecId;
use revm::context::BlockEnv;
use tracing::{error, info};

/// Executes the consolidation phase of the interop proof with the given [PreimageOracleClient] and
/// [HintWriterClient].
///
/// This phase is responsible for checking the dependencies between [OptimisticBlock]s in the
/// superchain and ensuring that all dependencies are satisfied.
///
/// [OptimisticBlock]: kona_proof_interop::OptimisticBlock
pub(crate) async fn consolidate_dependencies<P, H, Evm>(
    oracle: Arc<CachingOracle<P, H>>,
    mut boot: BootInfo,
    evm_factory: Evm,
) -> Result<(), FaultProofProgramError>
where
    P: PreimageOracleClient + Send + Sync + Debug + Clone,
    H: HintWriterClient + Send + Sync + Debug + Clone,
    Evm: EvmFactory<Spec = OpSpecId, BlockEnv = BlockEnv> + Send + Sync + Debug + Clone + 'static,
    <Evm as EvmFactory>::Tx:
        FromTxWithEncoded<OpTxEnvelope> + FromRecoveredTx<OpTxEnvelope> + OpTxEnv,
{
    info!(target: "client_interop", "Deriving local-safe headers from prestate");

    // Ensure that the pre-state is a transition state. It is invalid to pass a non-transition state
    // to this function, as it will not have the required information to derive the local-safe
    // headers for the next super root.
    let PreState::TransitionState(ref transition_state) = boot.agreed_pre_state else {
        return Err(FaultProofProgramError::StateTransitionFailed);
    };

    // Collect the cross-safe output roots and local-safe block hashes from the transition state.
    let transition_meta = transition_state
        .pending_progress
        .iter()
        .zip(transition_state.pre_state.output_roots.iter())
        .map(|(optimistic_block, pre_state)| (pre_state, optimistic_block.block_hash))
        .collect::<HashMap<_, _>>();

    let mut headers = HashMap::default();
    let mut l2_providers = HashMap::default();
    for (cross_safe_output, local_safe_block_hash) in transition_meta {
        // Fetch the cross-safe head's block hash for the given L2 chain ID.
        let cross_safe_head_hash = fetch_output_block_hash(
            oracle.as_ref(),
            cross_safe_output.output_root,
            cross_safe_output.chain_id,
        )
        .await?;

        // Fetch the rollup config for the given L2 chain ID.
        let rollup_config = ROLLUP_CONFIGS
            .get(&cross_safe_output.chain_id)
            .or_else(|| boot.rollup_configs.get(&cross_safe_output.chain_id))
            .ok_or(FaultProofProgramError::MissingRollupConfig(cross_safe_output.chain_id))?;

        // Initialize the local provider for the current L2 chain.
        let mut local_provider = OracleL2ChainProvider::new(
            cross_safe_head_hash,
            Arc::new(rollup_config.clone()),
            oracle.clone(),
        );
        local_provider.set_chain_id(Some(cross_safe_output.chain_id));

        // Send hints for the L2 block data in the pending progress. This is an important step,
        // because non-canonical blocks within the pending progress will not be able to be fetched
        // by the host through traditional means. If the block is determined to not be canonical
        // by the host, it will derive + build it and store the required preimages to complete
        // deposit-only re-execution. If the block is determined to be canonical, the host will
        // no-op, and preimages will be fetched through the traditional route as needed.
        HintType::L2BlockData
            .with_data(&[
                cross_safe_head_hash.as_slice(),
                local_safe_block_hash.as_slice(),
                cross_safe_output.chain_id.to_be_bytes().as_slice(),
            ])
            .send(oracle.as_ref())
            .await?;

        // Fetch the header for the local-safe head of the current L2 chain.
        let header = local_provider.header_by_hash(local_safe_block_hash)?;

        headers.insert(cross_safe_output.chain_id, header.seal(local_safe_block_hash));
        l2_providers.insert(cross_safe_output.chain_id, local_provider);
    }

    info!(
        target: "client_interop",
        num_blocks = headers.len(),
        "Loaded local-safe headers",
    );

    // Consolidate the superchain
    let global_provider = OracleInteropProvider::new(oracle.clone(), boot.clone(), headers);
    SuperchainConsolidator::new(&mut boot, global_provider, l2_providers, evm_factory)
        .consolidate()
        .await?;

    // Transition to the Super Root at the next timestamp.
    let post = boot
        .agreed_pre_state
        .transition(None)
        .ok_or(FaultProofProgramError::StateTransitionFailed)?;
    let post_commitment = post.hash();

    // Ensure that the post-state matches the claimed post-state.
    if post_commitment != boot.claimed_post_state {
        error!(
            target: "client_interop",
            claimed = ?boot.claimed_post_state,
            actual = ?post_commitment,
            "Post state validation failed",
        );
        return Err(FaultProofProgramError::InvalidClaim(boot.claimed_post_state, post_commitment));
    }

    info!(
        target: "client_interop",
        root = ?boot.claimed_post_state,
        "Super root validation succeeded"
    );
    Ok(())
}
