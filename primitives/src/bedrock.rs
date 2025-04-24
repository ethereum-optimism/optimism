//! OP mainnet bedrock related data.

use alloy_consensus::{Header, EMPTY_OMMER_ROOT_HASH, EMPTY_ROOT_HASH};
use alloy_primitives::{address, b256, bloom, bytes, B256, B64, U256};

/// Transaction 0x9ed8f713b2cc6439657db52dcd2fdb9cc944915428f3c6e2a7703e242b259cb9 in block 985,
/// replayed in blocks:
///
/// 19 022
/// 45 036
pub const TX_BLOCK_985: [u64; 2] = [19_022, 45_036];

/// Transaction 0xc033250c5a45f9d104fc28640071a776d146d48403cf5e95ed0015c712e26cb6 in block
/// 123 322, replayed in block:
///
/// 123 542
pub const TX_BLOCK_123_322: u64 = 123_542;

/// Transaction 0x86f8c77cfa2b439e9b4e92a10f6c17b99fce1220edf4001e4158b57f41c576e5 in block
/// 1 133 328, replayed in blocks:
///
/// 1 135 391
/// 1 144 468
pub const TX_BLOCK_1_133_328: [u64; 2] = [1_135_391, 1_144_468];

/// Transaction 0x3cc27e7cc8b7a9380b2b2f6c224ea5ef06ade62a6af564a9dd0bcca92131cd4e in block
/// 1 244 152, replayed in block:
///
/// 1 272 994
pub const TX_BLOCK_1_244_152: u64 = 1_272_994;

/// The six blocks with replayed transactions.
pub const BLOCK_NUMS_REPLAYED_TX: [u64; 6] = [
    TX_BLOCK_985[0],
    TX_BLOCK_985[1],
    TX_BLOCK_123_322,
    TX_BLOCK_1_133_328[0],
    TX_BLOCK_1_133_328[1],
    TX_BLOCK_1_244_152,
];

/// Returns `true` if transaction is the second or third appearance of the transaction. The blocks
/// with replayed transaction happen to only contain the single transaction.
pub fn is_dup_tx(block_number: u64) -> bool {
    if block_number > BLOCK_NUMS_REPLAYED_TX[5] {
        return false
    }

    // these blocks just have one transaction!
    if BLOCK_NUMS_REPLAYED_TX.contains(&block_number) {
        return true
    }

    false
}

/// OVM Header #1 hash.
///
/// <https://optimistic.etherscan.io/block/0xbee7192e575af30420cae0c7776304ac196077ee72b048970549e4f08e875453>
pub const OVM_HEADER_1_HASH: B256 =
    b256!("0xbee7192e575af30420cae0c7776304ac196077ee72b048970549e4f08e875453");

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
};

/// Bedrock total difficulty on Optimism Mainnet.
pub const BEDROCK_HEADER_TTD: U256 = U256::ZERO;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bedrock_header() {
        assert_eq!(BEDROCK_HEADER.hash_slow(), BEDROCK_HEADER_HASH);
    }
}
