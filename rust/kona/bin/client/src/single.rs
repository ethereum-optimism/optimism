//! Single-chain fault proof program entrypoint.

use crate::fpvm_evm::FpvmOpEvmFactory;
use alloc::sync::Arc;
use alloy_op_evm::post_exec::PostExecEvmFactoryAdapter;
use alloy_primitives::B256;
use core::fmt::Debug;
use kona_derive::{EthereumDataSource, PipelineErrorKind};
use kona_driver::{Driver, DriverError};
use kona_executor::ExecutorError;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use kona_proof::{
    BootInfo, CachingOracle,
    errors::OracleProviderError,
    executor::KonaExecutor,
    l1::{OracleBlobProvider, OraclePipeline},
    sync::{DerivationInputs, SyncStartError, prepare_derivation},
};
use kona_registry::{DEPENDENCY_SETS, DependencySet, HashMap};
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
    /// The registry's dependency set for the chain has members other than the chain itself.
    /// The single-chain program does not validate executing messages, so it only runs a chain
    /// whose dependency set is self-only.
    #[error(
        "dependency set for chain {0} is not self-only; prove this chain with the interop program"
    )]
    NotSelfOnlyDependencySet(u64),
}

/// Returns the superchain-registry dependency set for `chain_id`, the one op-node loads, only
/// when the chain is its only member. A chain the registry does not know has none.
#[allow(clippy::result_large_err)]
fn self_only_dependency_set(
    chain_id: u64,
    embedded: &HashMap<u64, DependencySet>,
) -> Result<Option<DependencySet>, FaultProofProgramError> {
    match embedded.get(&chain_id) {
        None => Ok(None),
        Some(ds) if ds.dependencies.len() == 1 && ds.dependencies.contains_key(&chain_id) => {
            Ok(Some(ds.clone()))
        }
        Some(_) => Err(FaultProofProgramError::NotSelfOnlyDependencySet(chain_id)),
    }
}

/// Executes the fault proof program with the given [PreimageOracleClient] and [HintWriterClient].
#[inline]
#[allow(clippy::result_large_err)]
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
    let dependency_set = self_only_dependency_set(boot.chain_id, &DEPENDENCY_SETS)?.map(Arc::new);
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
        // The registry's self-only dependency set for this chain, the one op-node loads. A chain
        // the registry does not know has none, so the StatefulAttributesBuilder constructor
        // panics if it schedules Lagoon.
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

#[cfg(test)]
mod tests {
    use super::*;
    use kona_registry::ChainDependency;

    fn depset(ids: &[u64]) -> DependencySet {
        DependencySet {
            dependencies: ids.iter().map(|id| (*id, ChainDependency {})).collect(),
            override_message_expiry_window: None,
        }
    }

    /// Keys the set under every member id, as `DEPENDENCY_SETS` does.
    fn embedded(ids: &[u64]) -> HashMap<u64, DependencySet> {
        ids.iter().map(|id| (*id, depset(ids))).collect()
    }

    #[test]
    fn self_only_set_is_used() {
        assert_eq!(self_only_dependency_set(10, &embedded(&[10])).unwrap(), Some(depset(&[10])));
    }

    #[test]
    fn set_with_another_member_is_rejected() {
        assert!(matches!(
            self_only_dependency_set(10, &embedded(&[10, 11])),
            Err(FaultProofProgramError::NotSelfOnlyDependencySet(10))
        ));
    }

    #[test]
    fn unknown_chain_has_no_set() {
        assert_eq!(self_only_dependency_set(10, &HashMap::default()).unwrap(), None);
    }
}
