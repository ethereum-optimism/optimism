//! SP1 guest for final super-root aggregation proofs.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use kona_sp1_client_utils::super_root::{
    SuperAggregationInputs, SuperConsolidationOutputs, SuperInteropOutputs, SuperRangeOutputs,
    SuperRootError,
};
use sha2::{Digest, Sha256};

/// Entrypoint to the super-aggregation program.
pub fn main() {
    let inputs = sp1_zkvm::io::read::<SuperAggregationInputs>();
    let public_values = aggregate(&inputs, sp1_lib::verify::verify_sp1_proof)
        .expect("invalid super-aggregation inputs");

    sp1_zkvm::io::commit_slice(&public_values);
}

fn aggregate(
    inputs: &SuperAggregationInputs,
    mut verify_proof: impl FnMut(&[u32; 8], &[u8; 32]),
) -> Result<Vec<u8>, SuperRootError> {
    inputs.validate()?;

    // Range program proofs are read from SP1's proof stream in this exact public-output order.
    for range_output in &inputs.range_outputs {
        verify_proof(
            &kona_sp1_range_vkeys::SUPER_RANGE_VKEY,
            &range_public_values_digest(range_output),
        );
    }

    for consolidation_output in &inputs.consolidation_outputs {
        verify_proof(
            &kona_sp1_range_vkeys::SUPER_RANGE_VKEY,
            &consolidation_public_values_digest(consolidation_output),
        );
    }

    Ok(inputs.zk_dispute_game_public_values())
}

fn range_public_values_digest(output: &SuperRangeOutputs) -> [u8; 32] {
    interop_public_values_digest(&SuperInteropOutputs::Range(output.clone()))
}

fn consolidation_public_values_digest(output: &SuperConsolidationOutputs) -> [u8; 32] {
    interop_public_values_digest(&SuperInteropOutputs::Consolidation(output.clone()))
}

fn interop_public_values_digest(output: &SuperInteropOutputs) -> [u8; 32] {
    let serialized = bincode::serialize(output).expect("super-root interop output serializes");
    Sha256::digest(serialized).into()
}

#[cfg(test)]
mod tests {
    use kona_sp1_client_utils::{super_root::SuperRootError, test_utils::valid_aggregation_inputs};
    use sha2::{Digest, Sha256};

    use super::{aggregate, consolidation_public_values_digest, range_public_values_digest};

    const RANGE_DIGESTS: [[u8; 32]; 2] = [
        [
            0x4e, 0xe8, 0xd3, 0x48, 0x6c, 0x0f, 0xb2, 0xab, 0xee, 0x27, 0xd3, 0xf3, 0x36, 0x73,
            0xe9, 0x69, 0x3f, 0x2d, 0x58, 0x80, 0xac, 0x88, 0x46, 0x02, 0xe0, 0xa9, 0x30, 0x95,
            0xa9, 0xed, 0x06, 0x32,
        ],
        [
            0xf0, 0x6d, 0x1f, 0x97, 0x3a, 0x76, 0x5b, 0x87, 0xc7, 0x3b, 0xe4, 0x05, 0xce, 0xab,
            0x0b, 0x40, 0xfa, 0x64, 0x11, 0x12, 0xd3, 0xbd, 0xc5, 0x50, 0x6d, 0xe0, 0x94, 0x22,
            0xf6, 0x53, 0x18, 0x2b,
        ],
    ];
    const CONSOLIDATION_DIGESTS: [[u8; 32]; 2] = [
        [
            0x28, 0x9f, 0xf7, 0xbc, 0xa7, 0x27, 0xfd, 0xbe, 0x61, 0xd1, 0x77, 0xc8, 0x82, 0x58,
            0x78, 0x58, 0x91, 0xc8, 0x4b, 0x22, 0xd1, 0xc7, 0x37, 0xa6, 0x94, 0x61, 0x67, 0x66,
            0xb4, 0xa8, 0x55, 0x25,
        ],
        [
            0x15, 0x25, 0x08, 0xa9, 0xc3, 0xd9, 0x53, 0x97, 0x70, 0xf0, 0x1f, 0x5a, 0x27, 0x9b,
            0x4e, 0x90, 0xe9, 0x06, 0xae, 0x40, 0xc7, 0x32, 0x8d, 0x73, 0xdb, 0x54, 0xd9, 0xc3,
            0x97, 0x40, 0x6b, 0x47,
        ],
    ];

    #[test]
    fn aggregation_verifies_children_in_proof_stream_order_and_returns_public_values() {
        let inputs = valid_aggregation_inputs();
        let mut calls = Vec::new();

        let public_values = aggregate(&inputs, |vkey, digest| calls.push((*vkey, *digest)))
            .expect("valid aggregation succeeds");

        assert_eq!(public_values, inputs.zk_dispute_game_public_values());
        assert_eq!(
            calls,
            vec![
                (kona_sp1_range_vkeys::SUPER_RANGE_VKEY, RANGE_DIGESTS[0]),
                (kona_sp1_range_vkeys::SUPER_RANGE_VKEY, RANGE_DIGESTS[1]),
                (kona_sp1_range_vkeys::SUPER_RANGE_VKEY, CONSOLIDATION_DIGESTS[0]),
                (kona_sp1_range_vkeys::SUPER_RANGE_VKEY, CONSOLIDATION_DIGESTS[1]),
            ]
        );
    }

    #[test]
    fn public_value_digests_bind_mode_discriminant() {
        let inputs = valid_aggregation_inputs();
        let range_output = &inputs.range_outputs[0];
        let consolidation_output = &inputs.consolidation_outputs[0];

        assert_eq!(range_public_values_digest(range_output), RANGE_DIGESTS[0]);
        assert_eq!(
            consolidation_public_values_digest(consolidation_output),
            CONSOLIDATION_DIGESTS[0]
        );

        let unwrapped_range_digest: [u8; 32] = Sha256::digest(
            bincode::serialize(range_output).expect("unwrapped range output serializes"),
        )
        .into();
        let unwrapped_consolidation_digest: [u8; 32] = Sha256::digest(
            bincode::serialize(consolidation_output)
                .expect("unwrapped consolidation output serializes"),
        )
        .into();
        assert_ne!(RANGE_DIGESTS[0], unwrapped_range_digest);
        assert_ne!(CONSOLIDATION_DIGESTS[0], unwrapped_consolidation_digest);
        assert_ne!(RANGE_DIGESTS[0], CONSOLIDATION_DIGESTS[0]);
    }

    #[test]
    fn aggregation_rejects_invalid_inputs_before_verifying_children() {
        let mut inputs = valid_aggregation_inputs();
        let expected_l1_head = inputs.l1_head;
        let actual_l1_head = [0xee; 32].into();
        inputs.range_outputs[0].l1_head = actual_l1_head;
        let mut verify_calls = 0;

        let result = aggregate(&inputs, |_, _| verify_calls += 1);

        assert_eq!(verify_calls, 0);
        assert_eq!(
            result,
            Err(SuperRootError::RangeL1HeadMismatch {
                expected: expected_l1_head,
                actual: actual_l1_head,
            })
        );
    }
}
