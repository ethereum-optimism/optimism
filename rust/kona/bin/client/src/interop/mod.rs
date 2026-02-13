//! Multi-chain, interoperable fault proof program entrypoint.

use alloc::{boxed::Box, sync::Arc};
use alloy_primitives::B256;
use consolidate::consolidate_dependencies;
use core::{cmp::Ordering, fmt::Debug};
use kona_derive::PipelineErrorKind;
use kona_driver::DriverError;
use kona_executor::ExecutorError;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use kona_proof::{CachingOracle, errors::OracleProviderError};
use kona_proof_interop::{
    BootInfo, ConsolidationError, PreState, TRANSITION_STATE_MAX_STEPS, boot::BootstrapError,
};
use thiserror::Error;
use tracing::{error, info};
use transition::sub_transition;

use crate::fpvm_evm::FpvmOpEvmFactory;

pub(crate) mod consolidate;
pub(crate) mod transition;
pub(crate) mod util;

/// An error that can occur when running the fault proof program.
#[derive(Error, Debug)]
pub enum FaultProofProgramError {
    /// The claim is invalid.
    #[error("Invalid claim. Expected {0}, actual {1}")]
    InvalidClaim(B256, B256),
    /// An error occurred in the Oracle provider.
    #[error(transparent)]
    OracleProvider(#[from] OracleProviderError),
    /// An error occurred in the driver.
    #[error(transparent)]
    Driver(#[from] Box<DriverError<ExecutorError>>),
    /// An error occurred in the derivation pipeline.
    #[error(transparent)]
    PipelineError(#[from] Box<PipelineErrorKind>),
    /// Consolidation error.
    #[error(transparent)]
    Consolidation(#[from] ConsolidationError),
    /// Bootstrap error
    #[error(transparent)]
    Bootstrap(#[from] BootstrapError),
    /// State transition failed.
    #[error("Critical state transition failure")]
    StateTransitionFailed,
    /// Missing a rollup configuration.
    #[error("Missing rollup configuration for chain ID {0}")]
    MissingRollupConfig(u64),
}

/// Executes the interop fault proof program with the given [PreimageOracleClient] and
/// [HintWriterClient].
#[inline]
pub async fn run<P, H>(oracle_client: P, hint_client: H) -> Result<(), FaultProofProgramError>
where
    P: PreimageOracleClient + Send + Sync + Debug + Clone + 'static,
    H: HintWriterClient + Send + Sync + Debug + Clone + 'static,
{
    const ORACLE_LRU_SIZE: usize = 1024;

    // Instantiate the oracle and bootstrap the program from local inputs.
    let oracle =
        Arc::new(CachingOracle::new(ORACLE_LRU_SIZE, oracle_client.clone(), hint_client.clone()));
    let boot = match BootInfo::load(oracle.as_ref()).await {
        Ok(boot) => boot,
        Err(BootstrapError::InvalidToInvalid) => {
            info!(target: "client_interop", "No-op transition, short-circuiting.");
            return Ok(());
        }
        Err(e) => {
            error!(target: "client_interop", "Failed to load boot info: {}", e);
            return Err(e.into());
        }
    };

    let evm_factory = FpvmOpEvmFactory::new(hint_client, oracle_client);

    // Load in the agreed pre-state from the preimage oracle in order to determine the active
    // sub-problem.
    match boot.agreed_pre_state {
        PreState::SuperRoot(ref super_root) => {
            // If the claimed L2 block timestamp is less than the super root timestamp, the
            // post-state must be the agreed pre-state to accommodate trace extension.
            if super_root.timestamp >= boot.claimed_l2_timestamp {
                if boot.agreed_pre_state_commitment == boot.claimed_post_state {
                    return Ok(());
                } else {
                    return Err(FaultProofProgramError::InvalidClaim(
                        boot.agreed_pre_state_commitment,
                        boot.claimed_post_state,
                    ));
                }
            }

            // If the pre-state is a super root, the first sub-problem is always selected.
            sub_transition(oracle, boot, evm_factory).await
        }
        PreState::TransitionState(ref transition_state) => {
            // If the claimed L2 block timestamp is less than the prestate timestamp, the
            // claim must be invalid.
            if transition_state.pre_state.timestamp >= boot.claimed_l2_timestamp {
                return Err(FaultProofProgramError::InvalidClaim(
                    boot.agreed_pre_state_commitment,
                    boot.claimed_post_state,
                ));
            }

            // If the pre-state is a transition state, the sub-problem is selected based on the
            // current step.
            match transition_state.step.cmp(&TRANSITION_STATE_MAX_STEPS) {
                Ordering::Equal => consolidate_dependencies(oracle, boot, evm_factory).await,
                Ordering::Less => sub_transition(oracle, boot, evm_factory).await,
                Ordering::Greater => {
                    error!(
                        target: "client_interop",
                        transition_state_step = transition_state.step,
                        transition_state_max_steps = TRANSITION_STATE_MAX_STEPS,
                        "Invalid transition state step; 'step' is greater than maximum value."
                    );
                    Err(FaultProofProgramError::StateTransitionFailed)
                }
            }
        }
    }
}
