//! Single-chain fault proof program entrypoint.

use crate::fpvm_evm::FpvmOpEvmFactory;
use alloc::sync::Arc;
use alloy_consensus::Sealed;
use alloy_primitives::B256;
use core::fmt::Debug;
use kona_derive::{EthereumDataSource, PipelineErrorKind};
use kona_driver::{Driver, DriverError};
use kona_executor::{ExecutorError, TrieDBProvider};
use kona_preimage::{CommsClient, HintWriterClient, PreimageKey, PreimageOracleClient};
use kona_proof::{
    BootInfo, CachingOracle, HintType,
    errors::OracleProviderError,
    executor::KonaExecutor,
    l1::{OracleBlobProvider, OracleL1ChainProvider, OraclePipeline},
    l2::OracleL2ChainProvider,
    sync::new_oracle_pipeline_cursor,
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
    let l1_config = boot.l1_config;
    let rollup_config = Arc::new(boot.rollup_config);
    let safe_head_hash = fetch_safe_head_hash(oracle.as_ref(), boot.agreed_l2_output_root).await?;

    let mut l1_provider = OracleL1ChainProvider::new(boot.l1_head, oracle.clone());
    let mut l2_provider =
        OracleL2ChainProvider::new(safe_head_hash, rollup_config.clone(), oracle.clone());
    let beacon = OracleBlobProvider::new(oracle.clone());

    // Fetch the safe head's block header.
    let safe_head = l2_provider
        .header_by_hash(safe_head_hash)
        .map(|header| Sealed::new_unchecked(header, safe_head_hash))?;

    // If the claimed L2 block number is less than the safe head of the L2 chain, the claim is
    // invalid.
    if boot.claimed_l2_block_number < safe_head.number {
        error!(
            target: "client",
            claimed = boot.claimed_l2_block_number,
            safe = safe_head.number,
            "Claimed L2 block number is less than the safe head",
        );
        return Err(FaultProofProgramError::InvalidClaim(
            boot.agreed_l2_output_root,
            boot.claimed_l2_output_root,
        ));
    }

    // In the case where the agreed upon L2 output root is the same as the claimed L2 output root,
    // trace extension is detected and we can skip the derivation and execution steps.
    if boot.agreed_l2_output_root == boot.claimed_l2_output_root {
        info!(
            target: "client",
            "Trace extension detected. State transition is already agreed upon.",
        );
        return Ok(());
    }

    ////////////////////////////////////////////////////////////////
    //                   DERIVATION & EXECUTION                   //
    ////////////////////////////////////////////////////////////////

    // Create a new derivation driver with the given boot information and oracle.
    let cursor = new_oracle_pipeline_cursor(
        rollup_config.as_ref(),
        safe_head,
        &mut l1_provider,
        &mut l2_provider,
    )
    .await
    .map_err(|e| {
        error!(target: "client", "Failed to create pipeline cursor: {:?}", e);
        e
    })?;
    l2_provider.set_cursor(cursor.clone());

    let evm_factory = FpvmOpEvmFactory::new(hint_client, oracle_client);
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
    .await?;

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

/// Fetches the safe head hash of the L2 chain based on the agreed upon L2 output root in the
/// [BootInfo].
pub async fn fetch_safe_head_hash<O>(
    caching_oracle: &O,
    agreed_l2_output_root: B256,
) -> Result<B256, OracleProviderError>
where
    O: CommsClient,
{
    let mut output_preimage = [0u8; 128];
    HintType::StartingL2Output
        .with_data(&[agreed_l2_output_root.as_ref()])
        .send(caching_oracle)
        .await?;
    caching_oracle
        .get_exact(PreimageKey::new_keccak256(*agreed_l2_output_root), output_preimage.as_mut())
        .await?;

    output_preimage[96..128].try_into().map_err(OracleProviderError::SliceConversion)
}
