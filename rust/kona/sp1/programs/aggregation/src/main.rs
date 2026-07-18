//! A program that aggregates the proofs of the range program.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use alloy_consensus::Header;
use alloy_primitives::B256;
use alloy_sol_types::SolValue;
use kona_sp1_client_utils::{
    boot::BootInfoStruct,
    types::{AggregationInputs, AggregationOutputs, u32_to_u8},
};
use sha2::{Digest, Sha256};
use std::collections::HashMap;

/// Entrypoint to the aggregation program.
pub fn main() {
    // Read in the public values corresponding to each range proof.
    let agg_inputs = sp1_zkvm::io::read::<AggregationInputs>();
    // Note: The headers are in order from start to end. We use serde_cbor as bincode serialization
    // causes issues with the zkVM.
    let headers_bytes = sp1_zkvm::io::read_vec();
    let headers: Vec<Header> = serde_cbor::from_slice(&headers_bytes).unwrap();
    assert!(!agg_inputs.boot_infos.is_empty());

    // Confirm that the boot infos are sequential.
    agg_inputs.boot_infos.windows(2).for_each(|pair| {
        let (prev_boot_info, boot_info) = (&pair[0], &pair[1]);

        // The claimed block of the previous boot info must be the L2 output root of the current
        // boot.
        assert_eq!(prev_boot_info.l2PostRoot, boot_info.l2PreRoot);

        // The rollup config must be the same for all the boot infos, to ensure they're
        // from the same chain and span batch range.
        assert_eq!(prev_boot_info.rollupConfigHash, boot_info.rollupConfigHash);
    });

    // Verify each range program proof.
    agg_inputs.boot_infos.iter().for_each(|boot_info| {
        // In the range program, the public values digest is just the hash of the ABI encoded
        // boot info.
        let serialized_boot_info = bincode::serialize(&boot_info).unwrap();
        let pv_digest = Sha256::digest(serialized_boot_info);

        sp1_lib::verify::verify_sp1_proof(&kona_sp1_range_vkeys::RANGE_VKEY, &pv_digest.into());
    });

    // Create a map of each l1 head in the [`BootInfoStruct`]'s to booleans
    let mut l1_heads_map: HashMap<B256, bool> =
        agg_inputs.boot_infos.iter().map(|boot_info| (boot_info.l1Head, false)).collect();

    // Iterate through the headers in reverse order. The headers should be sequentially linked and
    // include the l1 head of each boot info.
    let mut current_hash = agg_inputs.latest_l1_checkpoint_head;
    for header in headers.iter().rev() {
        assert_eq!(current_hash, header.hash_slow());

        // Mark the l1 head as found if it's in our map.
        if let Some(found) = l1_heads_map.get_mut(&current_hash) {
            *found = true;
        }

        current_hash = header.parent_hash;
    }

    // Check if all l1 heads were found in the chain.
    for (l1_head, found) in &l1_heads_map {
        assert!(*found, "l1 head {l1_head:?} not found in the provided header chain");
    }

    let first_boot_info = &agg_inputs.boot_infos[0];
    let last_boot_info = &agg_inputs.boot_infos[agg_inputs.boot_infos.len() - 1];
    // Consolidate the boot info into a single BootInfo struct that represents the range proven.
    let final_boot_info = BootInfoStruct {
        // The first boot info's L2 output root is the L2 output root of the range.
        l2PreRoot: first_boot_info.l2PreRoot,
        l2BlockNumber: last_boot_info.l2BlockNumber,
        l2PostRoot: last_boot_info.l2PostRoot,
        l1Head: agg_inputs.latest_l1_checkpoint_head,
        rollupConfigHash: last_boot_info.rollupConfigHash,
    };

    // Convert the range vkey to a B256.
    let multi_block_vkey_b256 = B256::from(u32_to_u8(kona_sp1_range_vkeys::RANGE_VKEY));

    let agg_outputs = AggregationOutputs {
        l1Head: final_boot_info.l1Head,
        l2PreRoot: final_boot_info.l2PreRoot,
        l2PostRoot: final_boot_info.l2PostRoot,
        l2BlockNumber: final_boot_info.l2BlockNumber,
        rollupConfigHash: final_boot_info.rollupConfigHash,
        multiBlockVKey: multi_block_vkey_b256,
        proverAddress: agg_inputs.prover_address,
    };

    // Commit to the aggregated [`AggregationOutputs`].
    sp1_zkvm::io::commit_slice(&agg_outputs.abi_encode());
}
