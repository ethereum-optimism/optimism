//! This module contains the prologue phase of the client program, pulling in the boot
//! information, which is passed to the zkVM a public inputs to be verified on chain.

use alloy_primitives::B256;
use alloy_sol_types::{SolValue, sol};
use kona_genesis::RollupConfig;
use kona_proof::BootInfo;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

/// ABI encoding of `AggregationOutputs` is 7 * 32 bytes.
pub const AGGREGATION_OUTPUTS_SIZE: usize = 7 * 32;

/// Hash the serialized rollup config using SHA256. Note: The rollup config is never unrolled
/// on-chain, so switching to a different hash function is not a concern, as long as the config hash
/// is consistent with the one on the contract.
pub fn hash_rollup_config(config: &RollupConfig) -> B256 {
    let serialized_config = serde_json::to_string_pretty(config).unwrap();

    // Create a SHA256 hasher
    let mut hasher = Sha256::new();

    // Hash the serialized config
    hasher.update(serialized_config.as_bytes());

    // Finalize and convert to B256
    let hash = hasher.finalize();
    B256::from_slice(hash.as_ref())
}

sol! {
    #[derive(Debug, Serialize, Deserialize)]
    struct BootInfoStruct {
        bytes32 l1Head;
        bytes32 l2PreRoot;
        bytes32 l2PostRoot;
        uint64 l2BlockNumber;
        bytes32 rollupConfigHash;
    }
}

impl BootInfoStruct {
    /// Returns the public values bytes committed by the range program.
    ///
    /// Keep aggregation verification on the same explicit ABI encoding instead of relying on the
    /// serializer used by `sp1_zkvm::io::commit`.
    pub fn public_values(&self) -> Vec<u8> {
        self.abi_encode()
    }
}

impl From<BootInfo> for BootInfoStruct {
    fn from(boot_info: BootInfo) -> Self {
        Self {
            l1Head: boot_info.l1_head,
            l2PreRoot: boot_info.agreed_l2_output_root,
            l2PostRoot: boot_info.claimed_l2_output_root,
            l2BlockNumber: boot_info.claimed_l2_block_number,
            rollupConfigHash: hash_rollup_config(&boot_info.rollup_config),
        }
    }
}

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;

    use super::BootInfoStruct;

    #[test]
    fn boot_info_public_values_are_abi_encoded() {
        let boot_info = BootInfoStruct {
            l1Head: B256::from([0x11; 32]),
            l2PreRoot: B256::from([0x22; 32]),
            l2PostRoot: B256::from([0x33; 32]),
            l2BlockNumber: 0x0102_0304_0506_0708,
            rollupConfigHash: B256::from([0x44; 32]),
        };

        let public_values = boot_info.public_values();

        assert_eq!(public_values.len(), 5 * 32);
        assert_eq!(&public_values[..32], boot_info.l1Head.as_slice());
        assert_eq!(&public_values[3 * 32 + 24..4 * 32], &boot_info.l2BlockNumber.to_be_bytes());
        assert_eq!(&public_values[4 * 32..5 * 32], boot_info.rollupConfigHash.as_slice());
    }
}
