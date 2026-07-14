//! SP1 guest scaffold for unified super-root range and consolidation proofs.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use std::sync::Arc;

use anyhow::anyhow;
use kona_sp1_client_utils::{
    BlobStore,
    super_root::{
        SuperConsolidationInputs, SuperConsolidationOutputs, SuperConsolidationTransition,
        SuperInteropInputs, SuperInteropOutputs, SuperOptimisticBlock, SuperOutputRoot,
        SuperRootError, hash_super_root_proof,
    },
    witness::{DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
};
use kona_sp1_ethereum_client_utils::super_range::build_range_outputs;
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
            Ok(SuperInteropOutputs::Consolidation(build_consolidation_outputs(inputs)?))
        }
    }
}

fn build_consolidation_outputs(
    inputs: SuperConsolidationInputs,
) -> Result<SuperConsolidationOutputs, SuperRootError> {
    inputs.validate()?;
    // TODO: replace this with witness-backed interop consolidation over the timestamp span.
    let transitions = inputs
        .transitions
        .into_iter()
        .map(|input| {
            ensure_matching_consolidation_outputs(
                &input.optimistic_blocks,
                &input.claimed_super_root_proof.super_root.output_roots,
            )?;
            let super_root = hash_super_root_proof(&input.claimed_super_root_proof)?;
            Ok(SuperConsolidationTransition {
                timestamp: input.claimed_super_root_proof.super_root.timestamp,
                optimistic_blocks: input.optimistic_blocks,
                super_root,
            })
        })
        .collect::<Result<Vec<_>, _>>()?;

    Ok(SuperConsolidationOutputs {
        span: inputs.span,
        previous_super_root: inputs.previous_super_root,
        transitions,
    })
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

fn ensure_matching_consolidation_outputs(
    optimistic_blocks: &[SuperOptimisticBlock],
    claimed_output_roots: &[SuperOutputRoot],
) -> Result<(), SuperRootError> {
    for (optimistic_block, claimed_output_root) in
        optimistic_blocks.iter().zip(claimed_output_roots)
    {
        let claimed_output_root = claimed_output_root.output_root;
        if optimistic_block.output_root != claimed_output_root {
            return Err(SuperRootError::MismatchedConsolidationOutputRoot {
                chain_id: optimistic_block.chain_id,
                optimistic_output_root: optimistic_block.output_root,
                claimed_output_root,
            });
        }
    }

    Ok(())
}
