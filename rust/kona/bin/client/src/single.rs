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
}

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
        // Single-chain fault proof: no interop dependency set. If this chain
        // ever schedules interop, the StatefulAttributesBuilder constructor
        // will panic.
        None,
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
