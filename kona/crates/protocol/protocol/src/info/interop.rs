//! Contains interop-specific L1 block info types.

use crate::{info::CommonL1BlockFields, DecodeError};
use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes, U256};

/// Represents the fields within an Interop L1 block info transaction.
///
/// Interop Binary Format
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
pub struct L1BlockInfoInterop {
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
}

impl L1BlockInfoInterop {
    /// The type byte identifier for the L1 scalar format in Interop.
    pub const L1_SCALAR: u8 = 1;

    /// The length of an L1 info transaction in Interop.
    pub const L1_INFO_TX_LEN: usize = 4 + 32 * 5;

    /// The 4 byte selector of "setL1BlockValuesInterop()"
    pub const L1_INFO_TX_SELECTOR: [u8; 4] = [0x76, 0x0e, 0xe0, 0x4d];

    /// Encodes the [`L1BlockInfoInterop`] object into Ethereum transaction calldata.
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
        
        buf.into()
    }

    /// Decodes the [`L1BlockInfoInterop`] object from ethereum transaction calldata.
    pub fn decode_calldata(r: &[u8]) -> Result<Self, DecodeError> {
        if r.len() != Self::L1_INFO_TX_LEN {
            return Err(DecodeError::InvalidInteropLength(Self::L1_INFO_TX_LEN, r.len()));
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
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;

    #[test]
    fn test_decode_calldata_interop_invalid_length() {
        let r = vec![0u8; 1];
        assert_eq!(
            L1BlockInfoInterop::decode_calldata(&r),
            Err(DecodeError::InvalidInteropLength(L1BlockInfoInterop::L1_INFO_TX_LEN, r.len(),))
        );
    }

    #[test]
    fn test_l1_block_info_interop_roundtrip_calldata_encoding() {
        let info = L1BlockInfoInterop {
            number: 1,
            time: 2,
            base_fee: 3,
            block_hash: B256::from([4u8; 32]),
            sequence_number: 5,
            batcher_address: Address::from([6u8; 20]),
            blob_base_fee: 7,
            blob_base_fee_scalar: 8,
            base_fee_scalar: 9,
        };

        let calldata = info.encode_calldata();
        let decoded_info = L1BlockInfoInterop::decode_calldata(&calldata).unwrap();
        assert_eq!(info, decoded_info);
    }
}
