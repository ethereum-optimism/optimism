//! Types related to the aggregation program.

use alloy_primitives::{Address, B256};
use alloy_sol_types::sol;
use serde::{Deserialize, Serialize};

use crate::boot::BootInfoStruct;

/// The inputs to the aggregation program.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AggregationInputs {
    /// List of [`BootInfoStruct`] for each block being aggregated.
    pub boot_infos: Vec<BootInfoStruct>,

    /// The latest L1 checkpoint head.
    pub latest_l1_checkpoint_head: B256,

    /// The vkey of the range proof program
    pub multi_block_vkey: [u32; 8],

    /// The address of the prover to commit to.
    pub prover_address: Address,
}

sol! {
    #[derive(Debug, Serialize, Deserialize)]
    struct AggregationOutputs {
        bytes32 l1Head;
        bytes32 l2PreRoot;
        bytes32 l2PostRoot;
        uint64 l2BlockNumber;
        bytes32 rollupConfigHash;
        bytes32 multiBlockVKey;
        address proverAddress;
    }
}

/// Convert a u32 array to a u8 array. Useful for converting the range vkey to a B256.
pub fn u32_to_u8(input: [u32; 8]) -> [u8; 32] {
    let mut output = [0u8; 32];
    for (i, &value) in input.iter().enumerate() {
        let bytes = value.to_be_bytes();
        output[i * 4..(i + 1) * 4].copy_from_slice(&bytes);
    }
    output
}
