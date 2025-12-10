//! Jovian L1 Block Info transaction types.

use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes, U256};

use crate::DecodeError;

/// Represents the fields within an Jovian L1 block info transaction.
///
/// Jovian Binary Format
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
/// | 4       | OperatorFeeScalar        |
/// | 8       | OperatorFeeConstant      |
/// | 2       | DAFootprintGasScalar     |
/// +---------+--------------------------+
#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct L1BlockInfoJovian {
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
    /// The operator fee scalar
    pub operator_fee_scalar: u32,
    /// The operator fee constant
    pub operator_fee_constant: u64,
    /// The DA footprint gas scalar
    pub da_footprint_gas_scalar: u16,
}

impl L1BlockInfoJovian {
    /// The default DA footprint gas scalar
    /// <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/l1-attributes.md#overview>
    pub const DEFAULT_DA_FOOTPRINT_GAS_SCALAR: u16 = 400;

    /// The type byte identifier for the L1 scalar format in Jovian.
    pub const L1_SCALAR: u8 = 2;

    /// The length of an L1 info transaction in Jovian.
    pub const L1_INFO_TX_LEN: usize = 4 + 32 * 5 + 4 + 8 + 2;

    /// The 4 byte selector of "setL1BlockValuesJovian()"
    /// Those are the first 4 calldata bytes -> `<https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/l1-attributes.md#overview>`
    pub const L1_INFO_TX_SELECTOR: [u8; 4] = [0x3d, 0xb6, 0xbe, 0x2b];

    /// Encodes the [`L1BlockInfoJovian`] object into Ethereum transaction calldata.
    pub fn encode_calldata(&self) -> Bytes {
        let mut buf = Vec::with_capacity(Self::L1_INFO_TX_LEN);
        buf.extend_from_slice(Self::L1_INFO_TX_SELECTOR.as_ref());
        buf.extend_from_slice(self.base_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.blob_base_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.sequence_number.to_be_bytes().as_ref());
        buf.extend_from_slice(self.time.to_be_bytes().as_ref());
        buf.extend_from_slice(self.number.to_be_bytes().as_ref());
        buf.extend_from_slice(U256::from(self.base_fee).to_be_bytes::<32>().as_ref());
        buf.extend_from_slice(U256::from(self.blob_base_fee).to_be_bytes::<32>().as_ref());
        buf.extend_from_slice(self.block_hash.as_ref());
        buf.extend_from_slice(self.batcher_address.into_word().as_ref());
        buf.extend_from_slice(self.operator_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.operator_fee_constant.to_be_bytes().as_ref());
        buf.extend_from_slice(self.da_footprint_gas_scalar.to_be_bytes().as_ref());
        buf.into()
    }

    /// Decodes the [`L1BlockInfoJovian`] object from ethereum transaction calldata.
    pub fn decode_calldata(r: &[u8]) -> Result<Self, DecodeError> {
        if r.len() != Self::L1_INFO_TX_LEN {
            return Err(DecodeError::InvalidJovianLength(Self::L1_INFO_TX_LEN, r.len()));
        }

        // SAFETY: For all below slice operations, the full
        //         length is validated above to be `178`.

        // SAFETY: 4 bytes are copied directly into the array
        let mut base_fee_scalar = [0u8; 4];
        base_fee_scalar.copy_from_slice(&r[4..8]);
        let base_fee_scalar = u32::from_be_bytes(base_fee_scalar);

        // SAFETY: 4 bytes are copied directly into the array
        let mut blob_base_fee_scalar = [0u8; 4];
        blob_base_fee_scalar.copy_from_slice(&r[8..12]);
        let blob_base_fee_scalar = u32::from_be_bytes(blob_base_fee_scalar);

        // SAFETY: 8 bytes are copied directly into the array
        let mut sequence_number = [0u8; 8];
        sequence_number.copy_from_slice(&r[12..20]);
        let sequence_number = u64::from_be_bytes(sequence_number);

        // SAFETY: 8 bytes are copied directly into the array
        let mut time = [0u8; 8];
        time.copy_from_slice(&r[20..28]);
        let time = u64::from_be_bytes(time);

        // SAFETY: 8 bytes are copied directly into the array
        let mut number = [0u8; 8];
        number.copy_from_slice(&r[28..36]);
        let number = u64::from_be_bytes(number);

        // SAFETY: 8 bytes are copied directly into the array
        let mut base_fee = [0u8; 8];
        base_fee.copy_from_slice(&r[60..68]);
        let base_fee = u64::from_be_bytes(base_fee);

        // SAFETY: 16 bytes are copied directly into the array
        let mut blob_base_fee = [0u8; 16];
        blob_base_fee.copy_from_slice(&r[84..100]);
        let blob_base_fee = u128::from_be_bytes(blob_base_fee);

        let block_hash = B256::from_slice(r[100..132].as_ref());
        let batcher_address = Address::from_slice(r[144..164].as_ref());

        // SAFETY: 4 bytes are copied directly into the array
        let mut operator_fee_scalar = [0u8; 4];
        operator_fee_scalar.copy_from_slice(&r[164..168]);
        let operator_fee_scalar = u32::from_be_bytes(operator_fee_scalar);

        // SAFETY: 8 bytes are copied directly into the array
        let mut operator_fee_constant = [0u8; 8];
        operator_fee_constant.copy_from_slice(&r[168..176]);
        let operator_fee_constant = u64::from_be_bytes(operator_fee_constant);

        // SAFETY: 2 bytes are copied directly into the array
        let mut da_footprint_gas_scalar = [0u8; 2];
        da_footprint_gas_scalar.copy_from_slice(&r[176..178]);
        let mut da_footprint_gas_scalar = u16::from_be_bytes(da_footprint_gas_scalar);

        // If the da footprint gas scalar is 0, use the default value (`https://github.com/ethereum-optimism/specs/blob/664cba65ab9686b0e70ad19fdf2ad054d6295986/specs/protocol/jovian/l1-attributes.md#overview`).
        if da_footprint_gas_scalar == 0 {
            da_footprint_gas_scalar = Self::DEFAULT_DA_FOOTPRINT_GAS_SCALAR;
        }

        Ok(Self {
            number,
            time,
            base_fee,
            block_hash,
            sequence_number,
            batcher_address,
            blob_base_fee,
            blob_base_fee_scalar,
            base_fee_scalar,
            operator_fee_scalar,
            operator_fee_constant,
            da_footprint_gas_scalar,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;
    use alloy_primitives::keccak256;

    #[test]
    fn test_decode_calldata_jovian_invalid_length() {
        let r = vec![0u8; 1];
        assert_eq!(
            L1BlockInfoJovian::decode_calldata(&r),
            Err(DecodeError::InvalidJovianLength(L1BlockInfoJovian::L1_INFO_TX_LEN, r.len()))
        );
    }

    #[test]
    fn test_function_selector() {
        assert_eq!(
            keccak256("setL1BlockValuesJovian()")[..4].to_vec(),
            L1BlockInfoJovian::L1_INFO_TX_SELECTOR
        );
    }

    #[test]
    fn test_l1_block_info_jovian_roundtrip_calldata_encoding() {
        let info = L1BlockInfoJovian {
            number: 1,
            time: 2,
            base_fee: 3,
            block_hash: B256::from([4; 32]),
            sequence_number: 5,
            batcher_address: Address::from_slice(&[6; 20]),
            blob_base_fee: 7,
            blob_base_fee_scalar: 8,
            base_fee_scalar: 9,
            operator_fee_scalar: 10,
            operator_fee_constant: 11,
            da_footprint_gas_scalar: 12,
        };

        let calldata = info.encode_calldata();
        let decoded_info = L1BlockInfoJovian::decode_calldata(&calldata).unwrap();

        assert_eq!(info, decoded_info);
    }
}
