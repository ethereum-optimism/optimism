//! Single-chain fault proof program entrypoint.

use crate::fpvm_evm::FpvmOpEvmFactory;
use alloc::sync::Arc;
use alloy_op_evm::post_exec::PostExecEvmFactoryAdapter;
use alloy_primitives::B256;
use core::fmt::Debug;
use kona_derive::{EthereumDataSource, PipelineErrorKind};
use kona_driver::{Driver, DriverError};
use kona_executor::ExecutorError;
use kona_genesis::RollupConfig;
use kona_interop::DependencySet;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use kona_proof::{
    BootInfo, CachingOracle,
    errors::OracleProviderError,
    executor::KonaExecutor,
    l1::{OracleBlobProvider, OraclePipeline},
    sync::{DerivationInputs, SyncStartError, prepare_derivation},
};
use kona_registry::DEPENDENCY_SETS;
use thiserror::Error;
use tracing::{error, info};

/// An error that can occur when running the fault proof program.
#[derive(Error, Debug)]
pub enum FaultProofProgramError {
    /// The claim is invalid.
    #[error("Invalid claim. Expected {0}, actual {1}")]
    InvalidClaim(B256, B256),
    /// An error occurred in the Oracle provider.
    #[error(transparent)]
    OracleProviderError(#[from] OracleProviderError),
    /// An error occurred in the derivation pipeline.
    #[error(transparent)]
    PipelineError(#[from] PipelineErrorKind),
    /// An error occurred in the driver.
    #[error(transparent)]
    Driver(#[from] DriverError<ExecutorError>),
    /// The chain has no interop dependency set.
    #[error(transparent)]
    MissingDependencySet(#[from] MissingDependencySet),
}

/// The chain schedules Lagoon, but the registry has no interop dependency set for it.
#[derive(Error, Debug, PartialEq, Eq)]
#[error("chain {0} schedules Lagoon but has no embedded interop dependency set")]
pub struct MissingDependencySet(pub u64);

/// Executes the fault proof program with the given [PreimageOracleClient] and [HintWriterClient].
#[inline]
pub async fn run<P, H>(oracle_client: P, hint_client: H) -> Result<(), FaultProofProgramError>
where
    P: PreimageOracleClient + Send + Sync + Debug + Clone + 'static,
    H: HintWriterClient + Send + Sync + Debug + Clone + 'static,
{
    const ORACLE_LRU_SIZE: usize = 1024;

    ////////////////////////////////////////////////////////////////
    //                          PROLOGUE                          //
    ////////////////////////////////////////////////////////////////

    let oracle =
        Arc::new(CachingOracle::new(ORACLE_LRU_SIZE, oracle_client.clone(), hint_client.clone()));
    let boot = BootInfo::load(oracle.as_ref()).await?;
    let rollup_config = Arc::new(boot.rollup_config.clone());
    let beacon = OracleBlobProvider::new(oracle.clone());

    // Run the shared sync-start prologue: validate the claim against the safe head and either
    // detect a zero-step trace extension or build the derivation inputs.
    let inputs = match prepare_derivation(&boot, rollup_config.clone(), oracle.clone()).await {
        Ok(inputs) => inputs,
        Err(SyncStartError::Oracle(e)) => return Err(e.into()),
        Err(SyncStartError::ClaimedBlockBeforeSafeHead { claimed, safe_head }) => {
            error!(
                target: "client",
                claimed,
                safe = safe_head,
                "Claimed L2 block number is less than the safe head",
            );
            return Err(FaultProofProgramError::InvalidClaim(
                boot.agreed_l2_output_root,
                boot.claimed_l2_output_root,
            ));
        }
        Err(SyncStartError::ClaimedRootMismatch { safe_head, .. }) => {
            error!(
                target: "client",
                claimed = boot.claimed_l2_block_number,
                safe = safe_head,
                expected_output_root = ?boot.agreed_l2_output_root,
                claimed_output_root = ?boot.claimed_l2_output_root,
                "Claimed output root does not match agreed output root at safe head",
            );
            return Err(FaultProofProgramError::InvalidClaim(
                boot.agreed_l2_output_root,
                boot.claimed_l2_output_root,
            ));
        }
    };

    let (cursor, l1_provider, l2_provider) = match inputs {
        // The claim targets the safe head and matches the agreed output root: nothing to derive.
        DerivationInputs::TraceExtension => return Ok(()),
        DerivationInputs::Derive { cursor, l1_provider, l2_provider } => {
            (cursor, l1_provider, l2_provider)
        }
    };

    let l1_config = boot.l1_config;
    let dependency_set = dependency_set(boot.chain_id, &rollup_config)?;

    let evm_factory =
        PostExecEvmFactoryAdapter::new(FpvmOpEvmFactory::new(hint_client, oracle_client));
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
        dependency_set,
    )
    .await?;

    let executor = KonaExecutor::new(
        rollup_config.as_ref(),
        l2_provider.clone(),
        l2_provider,
        evm_factory,
        alloy_op_evm::block::OpAlloyReceiptBuilder::default(),
        None,
    );
    let mut driver = Driver::new(cursor, executor, pipeline);

    // Run the derivation pipeline until we are able to produce the output root of the claimed
    // L2 block.
    let (safe_head, output_root) = driver
        .advance_to_target(rollup_config.as_ref(), Some(boot.claimed_l2_block_number))
        .await?;

    ////////////////////////////////////////////////////////////////
    //                          EPILOGUE                          //
    ////////////////////////////////////////////////////////////////

    if output_root != boot.claimed_l2_output_root {
        error!(
            target: "client",
            number = safe_head.block_info.number,
            output_root = ?output_root,
            claimed_output_root = ?boot.claimed_l2_output_root,
            "Failed to validate L2 block",
        );
        return Err(FaultProofProgramError::InvalidClaim(output_root, boot.claimed_l2_output_root));
    }

    info!(
        target: "client",
        number = safe_head.block_info.number,
        output_root = ?output_root,
        "Successfully validated L2 block",
    );

    Ok(())
}

/// Returns the interop dependency set that derivation needs for the chain.
///
/// Derivation needs a dependency set once the chain schedules Lagoon, even for the blocks before
/// it activates. A registry chain uses the set embedded from the registry, as op-node does. A chain
/// outside the registry has no embedded set, and the single-chain host does not serve one, so it
/// cannot be proven once it schedules Lagoon.
fn dependency_set(
    chain_id: u64,
    rollup_config: &RollupConfig,
) -> Result<Option<Arc<DependencySet>>, MissingDependencySet> {
    if rollup_config.hardforks.lagoon_time.is_none() {
        return Ok(None);
    }
    DEPENDENCY_SETS
        .get(&chain_id)
        .map(|dependency_set| Some(Arc::new(dependency_set.clone())))
        .ok_or(MissingDependencySet(chain_id))
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_registry::ROLLUP_CONFIGS;

    const OP_MAINNET: u64 = 10;

    #[test]
    fn no_dependency_set_without_lagoon() {
        let mut config = ROLLUP_CONFIGS[&OP_MAINNET].clone();
        config.hardforks.lagoon_time = None;
        assert!(dependency_set(OP_MAINNET, &config).unwrap().is_none());
    }

    #[test]
    fn registry_dependency_set_with_lagoon() {
        let mut config = ROLLUP_CONFIGS[&OP_MAINNET].clone();
        config.hardforks.lagoon_time = Some(u64::MAX);
        let dependency_set = dependency_set(OP_MAINNET, &config).unwrap().unwrap();
        assert_eq!(*dependency_set, DEPENDENCY_SETS[&OP_MAINNET]);
    }

    #[test]
    fn missing_dependency_set_with_lagoon() {
        const UNKNOWN_CHAIN: u64 = u64::MAX;
        assert!(!DEPENDENCY_SETS.contains_key(&UNKNOWN_CHAIN));
        let mut config = ROLLUP_CONFIGS[&OP_MAINNET].clone();
        config.hardforks.lagoon_time = Some(u64::MAX);
        assert_eq!(
            dependency_set(UNKNOWN_CHAIN, &config),
            Err(MissingDependencySet(UNKNOWN_CHAIN))
        );
    }
}
