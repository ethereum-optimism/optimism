//! Local key identifiers for fault-proof program bootstrap inputs.
//!
//! These values are part of the fault-proof ABI and must not change.

use alloy_primitives::U256;

/// The local key identifier for the L1 head hash.
///
/// This key retrieves the L1 block hash containing the data required to derive the disputed L2
/// blocks.
pub const L1_HEAD_KEY: U256 = U256::from_be_slice(&[1]);

/// The local key identifier for the agreed L2 output root.
///
/// This key retrieves the last known good L2 output root used as the derivation starting point.
///
/// Interop programs use this slot for the agreed superchain pre-state commitment.
pub const L2_OUTPUT_ROOT_KEY: U256 = U256::from_be_slice(&[2]);

/// The local key identifier for the disputed L2 output root claim.
///
/// This key retrieves the claimed L2 output root that the fault proof verifies.
///
/// Interop programs use this slot for the claimed superchain post-state commitment.
pub const L2_CLAIM_KEY: U256 = U256::from_be_slice(&[3]);

/// The local key identifier for the disputed L2 block number.
///
/// This key retrieves the target L2 block number for the disputed output root.
///
/// Interop programs use this slot for the claimed L2 timestamp.
pub const L2_CLAIM_BLOCK_NUMBER_KEY: U256 = U256::from_be_slice(&[4]);

/// The local key identifier for the L2 chain ID used to select network-specific configuration.
pub const L2_CHAIN_ID_KEY: U256 = U256::from_be_slice(&[5]);

/// The local key identifier for the L2 rollup configuration.
///
/// This key is used as an oracle fallback when the requested rollup configuration is not embedded
/// in the program.
pub const L2_ROLLUP_CONFIG_KEY: U256 = U256::from_be_slice(&[6]);

/// The local key identifier for the L1 chain configuration.
///
/// This key is used as an oracle fallback when the requested L1 configuration is not embedded in
/// the program.
pub const L1_CONFIG_KEY: U256 = U256::from_be_slice(&[7]);

/// The local key identifier for the interop dependency set.
///
/// This key is used as an oracle fallback when the requested dependency set is not embedded in the
/// program.
pub const DEPENDENCY_SET_KEY: U256 = U256::from_be_slice(&[8]);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn local_key_values_match_fault_proof_abi() {
        assert_eq!(L1_HEAD_KEY, U256::from(1));
        assert_eq!(L2_OUTPUT_ROOT_KEY, U256::from(2));
        assert_eq!(L2_CLAIM_KEY, U256::from(3));
        assert_eq!(L2_CLAIM_BLOCK_NUMBER_KEY, U256::from(4));
        assert_eq!(L2_CHAIN_ID_KEY, U256::from(5));
        assert_eq!(L2_ROLLUP_CONFIG_KEY, U256::from(6));
        assert_eq!(L1_CONFIG_KEY, U256::from(7));
        assert_eq!(DEPENDENCY_SET_KEY, U256::from(8));
    }
}
