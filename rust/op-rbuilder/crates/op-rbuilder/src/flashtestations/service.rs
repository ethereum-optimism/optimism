use alloy_primitives::{B256, Bytes, keccak256};
use std::{
    fs::{self, OpenOptions},
    io::Write,
    os::unix::fs::OpenOptionsExt,
    path::Path,
};
use tracing::{info, warn};

use super::{
    args::FlashtestationsArgs,
    attestation::{AttestationConfig, get_attestation_provider},
    tx_manager::TxManager,
};
use crate::{
    flashtestations::builder_tx::{FlashtestationsBuilderTx, FlashtestationsBuilderTxArgs},
    metrics::record_tee_metrics,
    tx_signer::{Signer, generate_key_from_seed, generate_signer},
};
use std::fmt::Debug;

pub async fn bootstrap_flashtestations<ExtraCtx, Extra>(
    args: FlashtestationsArgs,
    builder_key: Signer,
) -> eyre::Result<FlashtestationsBuilderTx<ExtraCtx, Extra>>
where
    ExtraCtx: Debug + Default,
    Extra: Debug + Default,
{
    let tee_service_signer = load_or_generate_tee_key(
        &args.flashtestations_key_path,
        args.debug,
        &args.debug_tee_key_seed,
    )?;

    info!(
        "Flashtestations TEE address: {}",
        tee_service_signer.address
    );

    let registry_address = args
        .registry_address
        .expect("registry address required when flashtestations enabled");
    let builder_policy_address = args
        .builder_policy_address
        .expect("builder policy address required when flashtestations enabled");

    let attestation_provider = get_attestation_provider(AttestationConfig {
        debug: args.debug,
        quote_provider: args.quote_provider,
    });

    // Prepare report data:
    // - TEE address (20 bytes) at reportData[0:20]
    // - Extended registration data hash (32 bytes) at reportData[20:52]
    // - Total: 52 bytes, padded to 64 bytes with zeros

    // Extract TEE address as 20 bytes
    let tee_address_bytes: [u8; 20] = tee_service_signer.address.into();

    // Calculate keccak256 hash of empty bytes (32 bytes)
    let ext_data = Bytes::from(b"");
    let ext_data_hash = keccak256(&ext_data);

    // Create 64-byte report data array
    let mut report_data = [0u8; 64];

    // Copy TEE address (20 bytes) to positions 0-19
    report_data[0..20].copy_from_slice(&tee_address_bytes);

    // Copy extended registration data hash (32 bytes) to positions 20-51
    report_data[20..52].copy_from_slice(ext_data_hash.as_ref());

    // Request TDX attestation
    info!(target: "flashtestations", "requesting TDX attestation");
    let attestation = attestation_provider.get_attestation(report_data).await?;

    // Record TEE metrics (workload ID, MRTD, RTMR0)
    record_tee_metrics(&attestation, &tee_service_signer.address)?;

    // Use an external rpc when the builder is not the same as the builder actively building blocks onchain
    let registered = if let Some(rpc_url) = args.rpc_url {
        let tx_manager = TxManager::new(
            tee_service_signer,
            builder_key,
            rpc_url.clone(),
            registry_address,
        );
        // Submit report onchain by registering the key of the tee service
        match tx_manager
            .register_tee_service(attestation.clone(), ext_data.clone())
            .await
        {
            Ok(_) => true,
            Err(e) => {
                warn!(error = %e, "Failed to register tee service via rpc");
                false
            }
        }
    } else {
        false
    };

    let flashtestations_builder_tx = FlashtestationsBuilderTx::new(FlashtestationsBuilderTxArgs {
        attestation,
        extra_registration_data: ext_data,
        tee_service_signer,
        registry_address,
        builder_policy_address,
        builder_proof_version: args.builder_proof_version,
        enable_block_proofs: args.enable_block_proofs,
        registered,
        builder_key,
    });

    Ok(flashtestations_builder_tx)
}

/// Load ephemeral TEE key from file, or generate and save a new one
fn load_or_generate_tee_key(key_path: &str, debug: bool, debug_seed: &str) -> eyre::Result<Signer> {
    if debug {
        info!("Flashtestations debug mode enabled, generating debug key from seed");
        return Ok(generate_key_from_seed(debug_seed));
    }

    let path = Path::new(key_path);

    if let Some(signer) = load_tee_key(path) {
        return Ok(signer);
    }

    // Generate new key
    info!("Generating new ephemeral TEE key");
    let signer = generate_signer();

    let key_hex = hex::encode(signer.secret.secret_bytes());

    // Create file with 0600 permissions atomically
    OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .mode(0o600)
        .open(path)
        .and_then(|mut file| file.write_all(key_hex.as_bytes()))
        .inspect_err(|e| warn!("Failed to write key to {}: {:?}", key_path, e))
        .ok();

    Ok(signer)
}

fn load_tee_key(path: &Path) -> Option<Signer> {
    // Try to load existing key
    if !path.exists() {
        return None;
    }

    info!("Loading TEE key from {:?}", path);
    let key_hex = fs::read_to_string(path)
        .inspect_err(|e| warn!("failed to read key file: {:?}", e))
        .ok()?;

    let secret_bytes = B256::try_from(
        hex::decode(key_hex.trim())
            .inspect_err(|e| warn!("failed to decode hex from file {:?}", e))
            .ok()?
            .as_slice(),
    )
    .inspect_err(|e| warn!("failed to parse key from file: {:?}", e))
    .ok()?;

    Signer::try_from_secret(secret_bytes)
        .inspect_err(|e| warn!("failed to create signer from key: {:?}", e))
        .ok()
}
