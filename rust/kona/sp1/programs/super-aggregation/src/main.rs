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

    // TODO(#21412): Embed the super-range verification key in this guest instead of accepting it
    // from the input. Range program proofs are read from SP1's proof stream in this exact
    // public-output order.
    for range_output in &inputs.range_outputs {
        verify_proof(&inputs.range_vkey, &range_public_values_digest(range_output));
    }

    for consolidation_output in &inputs.consolidation_outputs {
        verify_proof(&inputs.range_vkey, &consolidation_public_values_digest(consolidation_output));
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
    let serialized = bincode::serialize(output).expect("super-root range output serializes");
    Sha256::digest(serialized).into()
}

#[cfg(test)]
mod tests {
    use kona_sp1_client_utils::super_root::{
        SuperAggregationInputs, SuperConsolidationOutputs, SuperConsolidationTransition,
        SuperOptimisticBlock, SuperOutputRoot, SuperRangeOutputs, SuperRangeTransition,
        SuperRootError, SuperRootProof, TimestampSpan, hash_super_root_proof,
    };
    use sha2::{Digest, Sha256};

    use super::{aggregate, consolidation_public_values_digest, range_public_values_digest};

    const RANGE_DIGESTS: [[u8; 32]; 2] = [
        [
            0xdb, 0x92, 0x1d, 0xe3, 0x9f, 0xe2, 0xd7, 0xf6, 0x70, 0x01, 0xd9, 0x0a, 0x97, 0x67,
            0xd2, 0xc5, 0x6a, 0xc5, 0xf6, 0xbe, 0x45, 0xaa, 0x38, 0xac, 0x08, 0x84, 0xac, 0xad,
            0x69, 0x86, 0x72, 0x00,
        ],
        [
            0x50, 0xc4, 0x29, 0x09, 0x93, 0xcb, 0x22, 0x19, 0xe4, 0x89, 0x18, 0xd0, 0xd1, 0xbe,
            0x4b, 0x45, 0x69, 0xfc, 0xb6, 0x24, 0xf6, 0x7e, 0x3c, 0x1b, 0x71, 0xce, 0xf7, 0xec,
            0xfb, 0x7e, 0xe1, 0x74,
        ],
    ];
    const CONSOLIDATION_DIGESTS: [[u8; 32]; 2] = [
        [
            0xbc, 0xd9, 0x85, 0x0b, 0xff, 0xf7, 0x71, 0x91, 0x9b, 0x65, 0x71, 0x45, 0x15, 0xbf,
            0x27, 0xc3, 0x51, 0xbe, 0x67, 0xd4, 0xbb, 0x07, 0x41, 0x78, 0x08, 0x2a, 0xf1, 0x58,
            0x93, 0x50, 0x5d, 0x60,
        ],
        [
            0x84, 0x1a, 0x78, 0xe4, 0x79, 0x81, 0x31, 0xc0, 0xc5, 0xec, 0xf8, 0xd2, 0x7a, 0xc1,
            0xc4, 0x26, 0x64, 0x1c, 0x1e, 0x7c, 0xda, 0x61, 0x21, 0x03, 0x90, 0x64, 0xa9, 0xcf,
            0x2b, 0x10, 0xca, 0xc4,
        ],
    ];

    fn valid_inputs() -> SuperAggregationInputs {
        let l1_head = [0x99; 32].into();
        let starting_super_root_proof = SuperRootProof::new(
            99,
            vec![SuperOutputRoot { chain_id: 0, output_root: [0x01; 32].into() }],
        );
        let starting_root_hash =
            hash_super_root_proof(&starting_super_root_proof).expect("starting root hashes");
        let first_optimistic_block = SuperOptimisticBlock {
            chain_id: Default::default(),
            block_hash: [0x11; 32].into(),
            output_root: [0x12; 32].into(),
        };
        let second_optimistic_block = SuperOptimisticBlock {
            chain_id: Default::default(),
            block_hash: [0x21; 32].into(),
            output_root: [0x22; 32].into(),
        };
        let intermediate_root = [0x23; 32].into();
        let root_claim = [0x33; 32].into();

        SuperAggregationInputs {
            l1_head,
            starting_root_hash,
            starting_super_root_proof,
            root_claim,
            l2_sequence_number: 101,
            prover: [0x12; 20].into(),
            range_outputs: vec![
                SuperRangeOutputs {
                    span: TimestampSpan::new(100, 100).expect("valid span"),
                    l1_head,
                    previous_super_roots: vec![starting_root_hash],
                    transitions: vec![SuperRangeTransition {
                        timestamp: 100,
                        optimistic_block: first_optimistic_block,
                    }],
                },
                SuperRangeOutputs {
                    span: TimestampSpan::new(101, 101).expect("valid span"),
                    l1_head,
                    previous_super_roots: vec![intermediate_root],
                    transitions: vec![SuperRangeTransition {
                        timestamp: 101,
                        optimistic_block: second_optimistic_block,
                    }],
                },
            ],
            consolidation_outputs: vec![
                SuperConsolidationOutputs {
                    span: TimestampSpan::new(100, 100).expect("valid span"),
                    previous_super_root: starting_root_hash,
                    transitions: vec![SuperConsolidationTransition {
                        timestamp: 100,
                        optimistic_blocks: vec![first_optimistic_block],
                        super_root: intermediate_root,
                    }],
                },
                SuperConsolidationOutputs {
                    span: TimestampSpan::new(101, 101).expect("valid span"),
                    previous_super_root: intermediate_root,
                    transitions: vec![SuperConsolidationTransition {
                        timestamp: 101,
                        optimistic_blocks: vec![second_optimistic_block],
                        super_root: root_claim,
                    }],
                },
            ],
            range_vkey: [1, 2, 3, 4, 5, 6, 7, 8],
        }
    }

    #[test]
    fn aggregation_verifies_children_in_proof_stream_order_and_returns_public_values() {
        let inputs = valid_inputs();
        let mut calls = Vec::new();

        let public_values = aggregate(&inputs, |vkey, digest| calls.push((*vkey, *digest)))
            .expect("valid aggregation succeeds");

        assert_eq!(public_values, inputs.zk_dispute_game_public_values());
        assert_eq!(
            calls,
            vec![
                (inputs.range_vkey, RANGE_DIGESTS[0]),
                (inputs.range_vkey, RANGE_DIGESTS[1]),
                (inputs.range_vkey, CONSOLIDATION_DIGESTS[0]),
                (inputs.range_vkey, CONSOLIDATION_DIGESTS[1]),
            ]
        );
    }

    #[test]
    fn public_value_digests_bind_mode_discriminant() {
        let inputs = valid_inputs();
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
    }

    #[test]
    fn aggregation_rejects_invalid_inputs_before_verifying_children() {
        let mut inputs = valid_inputs();
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
