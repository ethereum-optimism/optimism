//! SP1 guest scaffold for final super-root aggregation proofs.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use kona_sp1_client_utils::super_root::{
    SuperAggregationInputs, SuperConsolidationOutputs, SuperInteropOutputs, SuperRangeOutputs,
};
use sha2::{Digest, Sha256};

/// Entrypoint to the super-aggregation program.
pub fn main() {
    let inputs = sp1_zkvm::io::read::<SuperAggregationInputs>();
    inputs.validate().expect("invalid super-aggregation inputs");

    // Range program proofs are read from SP1's proof stream in this exact public-output order.
    for range_output in &inputs.range_outputs {
        sp1_lib::verify::verify_sp1_proof(
            &kona_sp1_range_vkeys::SUPER_RANGE_VKEY,
            &range_public_values_digest(range_output),
        );
    }

    for consolidation_output in &inputs.consolidation_outputs {
        sp1_lib::verify::verify_sp1_proof(
            &kona_sp1_range_vkeys::SUPER_RANGE_VKEY,
            &consolidation_public_values_digest(consolidation_output),
        );
    }

    // TODO: check for sequential consistency and other checks

    sp1_zkvm::io::commit_slice(&inputs.zk_dispute_game_public_values());
}

fn range_public_values_digest(output: &SuperRangeOutputs) -> [u8; 32] {
    interop_public_values_digest(&SuperInteropOutputs::Range(output.clone()))
}

fn consolidation_public_values_digest(output: &SuperConsolidationOutputs) -> [u8; 32] {
    interop_public_values_digest(&SuperInteropOutputs::Consolidation(output.clone()))
}

fn interop_public_values_digest(output: &SuperInteropOutputs) -> [u8; 32] {
    let serialized = bincode::serialize(output).expect("super-root range output serializes");
    Sha256::digest(serialized).into()
}
