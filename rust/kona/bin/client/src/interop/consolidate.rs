//! Consolidation phase of the interop proof program.

use super::FaultProofProgramError;
use crate::interop::util::fetch_output_block_hash;
use alloc::sync::Arc;
use alloy_evm::{EvmFactory, FromRecoveredTx, FromTxWithEncoded, block::BlockExecutorFactory};
use alloy_op_evm::{
    OpBlockExecutionCtx, OpBlockExecutorFactory,
    block::{OpAlloyReceiptBuilder, OpTxEnv},
};
use core::fmt::Debug;
use kona_executor::TrieDBProvider;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use kona_proof::{CachingOracle, l2::OracleL2ChainProvider};
use kona_proof_interop::{
    BootInfo, HintType, OracleInteropProvider, PreState, SuperchainConsolidator, TransitionState,
};
use kona_registry::{HashMap, ROLLUP_CONFIGS, RollupConfig};
use op_alloy_consensus::{OpReceiptEnvelope, OpTxEnvelope};
use op_revm::OpSpecId;
use revm::context::BlockEnv;
use tracing::{error, info};

/// Executes the consolidation phase of the interop proof with the given [PreimageOracleClient] and
/// [HintWriterClient].
///
/// This phase is responsible for checking the dependencies between [OptimisticBlock]s in the
/// superchain and ensuring that all dependencies are satisfied.
///
/// Consolidation only applies once the Lagoon (interop) hardfork is active. Before activation the
/// optimistic outputs pass through unchanged and the super root is finalized directly.
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
    OpBlockExecutorFactory<OpAlloyReceiptBuilder, RollupConfig, Evm>: for<'a> BlockExecutorFactory<
            EvmFactory = Evm,
            ExecutionCtx<'a> = OpBlockExecutionCtx,
            Transaction = OpTxEnvelope,
            Receipt = OpReceiptEnvelope,
        >,
{
    // Ensure that the pre-state is a transition state. It is invalid to pass a non-transition state
    // to this function, as it will not have the required information to derive the local-safe
    // headers for the next super root.
    let PreState::TransitionState(transition_state) = &boot.agreed_pre_state else {
        return Err(FaultProofProgramError::StateTransitionFailed);
    };

    let consolidation_timestamp = transition_state.pre_state.timestamp + 1;

    // An optimisation: the CrossL2Inbox predeploy has no code before interop activates, so no
    // executing message can exist. Keeps in-development interop code fully off.
    if !interop_active_for_consolidation(
        &boot.rollup_configs,
        transition_state,
        consolidation_timestamp,
    ) {
        info!(
            target: "client_interop",
            timestamp = consolidation_timestamp,
            chains = transition_state.pending_progress.len(),
            "Interop inactive on every chain in the transition state; skipping consolidation",
        );
        return finalize_super_root(boot);
    }

    info!(target: "client_interop", "Deriving local-safe headers from prestate");

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
        let rollup_config = rollup_config_for(&boot.rollup_configs, cross_safe_output.chain_id)
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

    finalize_super_root(boot)
}

/// Finalizes a saturated transition state into the super root at the next timestamp, and validates
/// it against the claimed post-state.
fn finalize_super_root(boot: BootInfo) -> Result<(), FaultProofProgramError> {
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

/// Resolves the [`RollupConfig`] for `chain_id`, preferring the bundled superchain registry.
///
/// Matches the order the message validity rules use, so the gate reads the config they read.
fn rollup_config_for(
    local_configs: &HashMap<u64, RollupConfig>,
    chain_id: u64,
) -> Option<&RollupConfig> {
    ROLLUP_CONFIGS.get(&chain_id).or_else(|| local_configs.get(&chain_id))
}

/// Returns `true` when the Lagoon (interop) hardfork is active at `timestamp` on any chain in the
/// transition state.
///
/// A chain with no resolvable config counts as active, so consolidation proceeds and reports the
/// missing config exactly where it did before.
fn interop_active_for_consolidation(
    local_configs: &HashMap<u64, RollupConfig>,
    transition_state: &TransitionState,
    timestamp: u64,
) -> bool {
    transition_state.pre_state.output_roots.iter().any(|output_root| {
        rollup_config_for(local_configs, output_root.chain_id)
            .is_none_or(|config| config.is_interop_active(timestamp))
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec::Vec;
    use alloy_primitives::B256;
    use kona_genesis::HardForkConfig;
    use kona_interop::{OutputRootWithChain, SuperRoot};
    use kona_proof_interop::{OptimisticBlock, TRANSITION_STATE_MAX_STEPS};

    /// Chain IDs absent from the bundled superchain registry, so the tests set their own
    /// activation times.
    const CHAIN_A: u64 = 0xdead_0001;
    const CHAIN_B: u64 = 0xdead_0002;

    /// The pre-state timestamp. Consolidation finalizes the root at `PRE_TIMESTAMP + 1`.
    const PRE_TIMESTAMP: u64 = 99;

    fn configs(
        entries: impl IntoIterator<Item = (u64, Option<u64>)>,
    ) -> HashMap<u64, RollupConfig> {
        entries
            .into_iter()
            .map(|(chain_id, lagoon_time)| {
                let config = RollupConfig {
                    hardforks: HardForkConfig { lagoon_time, ..Default::default() },
                    ..Default::default()
                };
                (chain_id, config)
            })
            .collect()
    }

    /// Builds the saturated transition state that consolidation always receives.
    fn saturated_transition_state(chain_ids: impl IntoIterator<Item = u64>) -> TransitionState {
        let output_roots = chain_ids
            .into_iter()
            .map(|chain_id| OutputRootWithChain::new(chain_id, B256::ZERO))
            .collect::<Vec<_>>();
        let pending_progress =
            output_roots.iter().map(|_| OptimisticBlock::new(B256::ZERO, B256::ZERO)).collect();
        TransitionState::new(
            SuperRoot::new(PRE_TIMESTAMP, output_roots),
            pending_progress,
            TRANSITION_STATE_MAX_STEPS,
        )
    }

    fn interop_active(
        local: &HashMap<u64, RollupConfig>,
        chain_ids: impl IntoIterator<Item = u64>,
        timestamp: u64,
    ) -> bool {
        interop_active_for_consolidation(local, &saturated_transition_state(chain_ids), timestamp)
    }

    #[test]
    fn test_inactive_when_lagoon_unscheduled() {
        // `lagoon_time = None` is never active, at any timestamp.
        let local = configs([(CHAIN_A, None), (CHAIN_B, None)]);
        assert!(!interop_active(&local, [CHAIN_A, CHAIN_B], u64::MAX));
    }

    #[test]
    fn test_inactive_before_activation() {
        let local = configs([(CHAIN_A, Some(PRE_TIMESTAMP + 2))]);
        assert!(!interop_active(&local, [CHAIN_A], PRE_TIMESTAMP + 1));
    }

    #[test]
    fn test_active_at_activation_timestamp() {
        // The activation timestamp must still consolidate, so the per-message rules can reject it
        // via `is_first_interop_block`.
        let local = configs([(CHAIN_A, Some(PRE_TIMESTAMP + 1))]);
        assert!(interop_active(&local, [CHAIN_A], PRE_TIMESTAMP + 1));
    }

    #[test]
    fn test_active_after_activation() {
        let local = configs([(CHAIN_A, Some(PRE_TIMESTAMP))]);
        assert!(interop_active(&local, [CHAIN_A], PRE_TIMESTAMP + 1));
    }

    #[test]
    fn test_active_when_any_chain_is_active() {
        // An active chain can hold a message referencing the inactive one. Both orderings assert
        // the result does not depend on which chain is read first.
        let local = configs([(CHAIN_A, None), (CHAIN_B, Some(PRE_TIMESTAMP))]);
        assert!(interop_active(&local, [CHAIN_A, CHAIN_B], PRE_TIMESTAMP + 1));
        assert!(interop_active(&local, [CHAIN_B, CHAIN_A], PRE_TIMESTAMP + 1));
    }

    #[test]
    fn test_unresolvable_config_counts_as_active() {
        // Consolidating raises `MissingRollupConfig`; skipping would swallow it.
        let local = configs([(CHAIN_A, None)]);
        assert!(interop_active(&local, [CHAIN_A, CHAIN_B], PRE_TIMESTAMP + 1));
    }

    #[test]
    fn test_empty_transition_state_is_inactive() {
        assert!(!interop_active(&configs([]), [], PRE_TIMESTAMP + 1));
    }

    #[test]
    fn test_rollup_config_prefers_the_bundled_registry() {
        // The registry entry must win over a conflicting oracle-supplied config.
        let registry_chain = *ROLLUP_CONFIGS.keys().next().expect("registry is not empty");
        let local = configs([(registry_chain, Some(1))]);

        let resolved = rollup_config_for(&local, registry_chain).unwrap();
        assert_eq!(resolved, ROLLUP_CONFIGS.get(&registry_chain).unwrap());
    }

    #[test]
    fn test_rollup_config_falls_back_to_oracle_supplied_config() {
        let local = configs([(CHAIN_A, Some(42))]);
        let config = rollup_config_for(&local, CHAIN_A).unwrap();
        assert_eq!(config.hardforks.lagoon_time, Some(42));
    }
}
