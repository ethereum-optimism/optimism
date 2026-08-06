//! OP mainnet bedrock related data.

use alloy_consensus::{EMPTY_OMMER_ROOT_HASH, EMPTY_ROOT_HASH, Header};
use alloy_primitives::{B64, B256, U256, address, b256, bloom, bytes};

/// Bedrock hash on Optimism Mainnet.
///
/// <https://optimistic.etherscan.io/block/0xdbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3>
pub const BEDROCK_HEADER_HASH: B256 =
    b256!("0xdbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3");

/// Bedrock on Optimism Mainnet. (`105_235_063`)
pub const BEDROCK_HEADER: Header = Header {
    difficulty: U256::ZERO,
    extra_data: bytes!("424544524f434b"),
    gas_limit: 30000000,
    gas_used: 0,
    logs_bloom: bloom!(
        "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
    ),
    nonce: B64::ZERO,
    number: 105235063,
    parent_hash: b256!("0x21a168dfa5e727926063a28ba16fd5ee84c814e847c81a699c7a0ea551e4ca50"),
    receipts_root: EMPTY_ROOT_HASH,
    state_root: b256!("0x920314c198da844a041d63bf6cbe8b59583165fd2229d1b3f599da812fd424cb"),
    timestamp: 1686068903,
    transactions_root: EMPTY_ROOT_HASH,
    ommers_hash: EMPTY_OMMER_ROOT_HASH,
    beneficiary: address!("0x4200000000000000000000000000000000000011"),
    withdrawals_root: None,
    mix_hash: B256::ZERO,
    base_fee_per_gas: Some(0x3b9aca00),
    blob_gas_used: None,
    excess_blob_gas: None,
    parent_beacon_block_root: None,
    requests_hash: None,
    block_access_list_hash: None,
    slot_number: None,
};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bedrock_header() {
        assert_eq!(BEDROCK_HEADER.hash_slow(), BEDROCK_HEADER_HASH);
    }
}
