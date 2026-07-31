//! SP1 guest scaffold for unified super-root range and consolidation proofs.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use std::sync::Arc;

use anyhow::anyhow;
use kona_sp1_client_utils::{
    BlobStore,
    super_root::{SuperInteropInputs, SuperInteropOutputs},
    witness::{DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
};
use kona_sp1_ethereum_client_utils::{
    super_consolidation::build_consolidation_outputs, super_range::build_range_outputs,
};
use rkyv::rancor::Error as RkyvError;

/// Entrypoint to the unified super-root range program.
pub fn main() {
    let inputs = sp1_zkvm::io::read::<SuperInteropInputs>();
    let outputs = kona_proof::block_on(run(inputs)).expect("super interop failed");
    sp1_zkvm::io::commit(&outputs);
}

async fn run(inputs: SuperInteropInputs) -> anyhow::Result<SuperInteropOutputs> {
    match inputs {
        SuperInteropInputs::Range(inputs) => {
            let (oracle, beacon) = read_witness().await?;
            Ok(SuperInteropOutputs::Range(build_range_outputs(inputs, oracle, beacon).await?))
        }
        SuperInteropInputs::Consolidation(inputs) => {
            let (oracle, _) = read_witness().await?;
            Ok(SuperInteropOutputs::Consolidation(
                build_consolidation_outputs(inputs, oracle).await?,
            ))
        }
    }
}

async fn read_witness() -> anyhow::Result<(Arc<PreimageStore>, BlobStore)> {
    let witness_rkyv_bytes: Vec<u8> = sp1_zkvm::io::read_vec();
    let witness_data = rkyv::from_bytes::<DefaultWitnessData, RkyvError>(&witness_rkyv_bytes)
        .map_err(|err| anyhow!("failed to deserialize super-range witness data: {err}"))?;

    witness_data
        .get_oracle_and_blob_provider()
        .await
        .map_err(|err| anyhow!("failed to load oracle and blob provider: {err}"))
}
