//! SP1 guest scaffold for unified super-root range and consolidation proofs.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use kona_sp1_client_utils::super_root::{
    SuperConsolidationOutputs, SuperConsolidationTransition, SuperInteropInputs,
    SuperInteropOutputs, SuperRangeOutputs, hash_super_root_proof,
};

/// Entrypoint to the unified super-root range program.
pub fn main() {
    let inputs = sp1_zkvm::io::read::<SuperInteropInputs>();
    inputs.validate().expect("invalid super interop inputs");

    let outputs = match inputs {
        SuperInteropInputs::Range(inputs) => {
            // TODO: run optimistic output root stf
            let previous_super_roots = inputs
                .previous_super_root_proofs
                .iter()
                .map(hash_super_root_proof)
                .collect::<Result<Vec<_>, _>>()
                .expect("invalid previous super-root proof");
            SuperInteropOutputs::Range(SuperRangeOutputs {
                span: inputs.span,
                l1_head: inputs.l1_head,
                previous_super_roots,
                transitions: inputs.claimed_transitions,
            })
        }
        SuperInteropInputs::Consolidation(inputs) => {
            // TODO: run consolidation
            let transitions = inputs
                .transitions
                .into_iter()
                .map(|input| {
                    let super_root = hash_super_root_proof(&input.claimed_super_root_proof)
                        .expect("invalid super-root proof");
                    SuperConsolidationTransition {
                        timestamp: input.claimed_super_root_proof.super_root.timestamp,
                        optimistic_blocks: input.optimistic_blocks,
                        super_root,
                    }
                })
                .collect();

            SuperInteropOutputs::Consolidation(SuperConsolidationOutputs {
                span: inputs.span,
                previous_super_root: inputs.previous_super_root,
                transitions,
            })
        }
    };
    sp1_zkvm::io::commit(&outputs);
}
