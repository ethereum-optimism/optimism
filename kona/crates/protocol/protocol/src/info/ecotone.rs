//! Contains ecotone-specific L1 block info types.

use crate::{DecodeError, info::CommonL1BlockFields};
use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes, U256};

/// Represents the fields within an Ecotone L1 block info transaction.
///
/// Ecotone Binary Format
/// +---------+--------------------------+
/// | Bytes   | Field                    |
/// +---------+--------------------------+
/// | 4       | Function signature       |
/// | 4       | BaseFeeScalar            |
/// | 4       | BlobBaseFeeScalar        |
/// | 8       | SequenceNumber           |
/// | 8       | Timestamp                |
/// | 8       | L1BlockNumber            |
/// | 32      | BaseFee                  |
/// | 32      | BlobBaseFee              |
/// | 32      | BlockHash                |
/// | 32      | BatcherHash              |
/// +---------+--------------------------+
#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct L1BlockInfoEcotone {
    /// The current L1 origin block number
    pub number: u64,
    /// The current L1 origin block's timestamp
    pub time: u64,
    /// The current L1 origin block's basefee
    pub base_fee: u64,
    /// The current L1 origin block's hash
    pub block_hash: B256,
    /// The current sequence number
    pub sequence_number: u64,
    /// The address of the batch submitter
    pub batcher_address: Address,
    /// The current blob base fee on L1
    pub blob_base_fee: u128,
    /// The fee scalar for L1 blobspace data
    pub blob_base_fee_scalar: u32,
    /// The fee scalar for L1 data
    pub base_fee_scalar: u32,
    /// Indicates that the scalars are empty.
    /// This is an edge case where the first block in ecotone has no scalars,
    /// so the bedrock tx l1 cost function needs to be used.
    pub empty_scalars: bool,
    /// The l1 fee overhead used along with the `empty_scalars` field for the
    /// bedrock tx l1 cost function.
    ///
    /// This field is deprecated in the Ecotone Hardfork.
    pub l1_fee_overhead: U256,
}

impl L1BlockInfoEcotone {
    /// The type byte identifier for the L1 scalar format in Ecotone.
    pub const L1_SCALAR: u8 = 1;

    /// The length of an L1 info transaction in Ecotone.
    pub const L1_INFO_TX_LEN: usize = 4 + 32 * 5;

    /// The 4 byte selector of "setL1BlockValuesEcotone()"
    pub const L1_INFO_TX_SELECTOR: [u8; 4] = [0x44, 0x0a, 0x5e, 0x20];

    /// Encodes the [`L1BlockInfoEcotone`] object into Ethereum transaction calldata.
    pub fn encode_calldata(&self) -> Bytes {
        let mut buf = Vec::with_capacity(Self::L1_INFO_TX_LEN);
        buf.extend_from_slice(Self::L1_INFO_TX_SELECTOR.as_ref());

        let common = CommonL1BlockFields {
            base_fee_scalar: self.base_fee_scalar,
            blob_base_fee_scalar: self.blob_base_fee_scalar,
            sequence_number: self.sequence_number,
            time: self.time,
            number: self.number,
            base_fee: self.base_fee,
            blob_base_fee: self.blob_base_fee,
            block_hash: self.block_hash,
            batcher_address: self.batcher_address,
        };
        common.encode_into(&mut buf);

        // Notice: do not include the `empty_scalars` field in the calldata.
        // Notice: do not include the `l1_fee_overhead` field in the calldata.
        buf.into()
    }

    /// Decodes the [`L1BlockInfoEcotone`] object from ethereum transaction calldata.
    pub fn decode_calldata(r: &[u8]) -> Result<Self, DecodeError> {
        if r.len() != Self::L1_INFO_TX_LEN {
            return Err(DecodeError::InvalidEcotoneLength(Self::L1_INFO_TX_LEN, r.len()));
        }

        // SAFETY: For all below slice operations, the full
        //         length is validated above to be `164`.

        let common = CommonL1BlockFields::decode_from(r);

        Ok(Self {
            number: common.number,
            time: common.time,
            base_fee: common.base_fee,
            block_hash: common.block_hash,
            sequence_number: common.sequence_number,
            batcher_address: common.batcher_address,
            blob_base_fee: common.blob_base_fee,
            blob_base_fee_scalar: common.blob_base_fee_scalar,
            base_fee_scalar: common.base_fee_scalar,
            // Notice: the `empty_scalars` field is not included in the calldata.
            // This is used by the evm to indicate that the bedrock tx l1 cost function
            // needs to be used.
            empty_scalars: false,
            // Notice: the `l1_fee_overhead` field is not included in the calldata.
            l1_fee_overhead: U256::ZERO,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;

    #[test]
    fn test_decode_calldata_ecotone_invalid_length() {
        let r = vec![0u8; 1];
        assert_eq!(
            L1BlockInfoEcotone::decode_calldata(&r),
            Err(DecodeError::InvalidEcotoneLength(L1BlockInfoEcotone::L1_INFO_TX_LEN, r.len(),))
        );
    }

    #[test]
    fn test_l1_block_info_ecotone_roundtrip_calldata_encoding() {
        let info = L1BlockInfoEcotone {
            number: 1,
            time: 2,
            base_fee: 3,
            block_hash: B256::from([4u8; 32]),
            sequence_number: 5,
            batcher_address: Address::from([6u8; 20]),
            blob_base_fee: 7,
            blob_base_fee_scalar: 8,
            base_fee_scalar: 9,
            empty_scalars: false,
            l1_fee_overhead: U256::ZERO,
        };

        let calldata = info.encode_calldata();
        let decoded_info = L1BlockInfoEcotone::decode_calldata(&calldata).unwrap();
        assert_eq!(info, decoded_info);
    }
}
