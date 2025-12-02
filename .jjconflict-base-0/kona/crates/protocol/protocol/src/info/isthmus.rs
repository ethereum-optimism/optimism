//! Isthmus L1 Block Info transaction types.

use crate::info::{
    bedrock_base::ambassador_impl_L1BlockInfoBedrockBaseFields,
    ecotone_base::ambassador_impl_L1BlockInfoEcotoneBaseFields,
};
use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes};
use ambassador::{Delegate, delegatable_trait};

use crate::{
    DecodeError,
    info::{
        bedrock_base::L1BlockInfoBedrockBaseFields,
        ecotone_base::{L1BlockInfoEcotoneBase, L1BlockInfoEcotoneBaseFields},
    },
};

/// Represents the fields within an Isthmus L1 block info transaction.
///
/// Isthmus Binary Format
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
/// +---------+--------------------------+
#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy, Delegate)]
#[allow(clippy::duplicated_attributes)]
#[delegate(L1BlockInfoBedrockBaseFields, target = "base")]
#[delegate(L1BlockInfoEcotoneBaseFields, target = "base")]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct L1BlockInfoIsthmus {
    #[cfg_attr(feature = "serde", serde(flatten))]
    base: L1BlockInfoEcotoneBase,
    /// The operator fee scalar
    pub operator_fee_scalar: u32,
    /// The operator fee constant
    pub operator_fee_constant: u64,
}

/// Accessors for fields in Isthmus and later.
#[delegatable_trait]
pub trait L1BlockInfoIsthmusBaseFields: L1BlockInfoEcotoneBaseFields {
    /// The operator fee scalar
    fn operator_fee_scalar(&self) -> u32;
    /// The operator fee constant
    fn operator_fee_constant(&self) -> u64;
}

impl L1BlockInfoIsthmusBaseFields for L1BlockInfoIsthmus {
    /// The operator fee scalar
    fn operator_fee_scalar(&self) -> u32 {
        self.operator_fee_scalar
    }
    /// The operator fee constant
    fn operator_fee_constant(&self) -> u64 {
        self.operator_fee_constant
    }
}

/// Accessors for all Isthmus fields.
pub trait L1BlockInfoIsthmusFields:
    L1BlockInfoEcotoneBaseFields + L1BlockInfoIsthmusBaseFields
{
}

impl L1BlockInfoIsthmusFields for L1BlockInfoIsthmus {}

impl L1BlockInfoIsthmus {
    /// The type byte identifier for the L1 scalar format in Isthmus.
    pub const L1_SCALAR: u8 = 2;

    /// The length of an L1 info transaction in Isthmus.
    pub const L1_INFO_TX_LEN: usize = 4 + 32 * 5 + 4 + 8;

    /// The 4 byte selector of "setL1BlockValuesIsthmus()"
    pub const L1_INFO_TX_SELECTOR: [u8; 4] = [0x09, 0x89, 0x99, 0xbe];

    /// Encodes the [`L1BlockInfoIsthmus`] object into Ethereum transaction calldata.
    pub fn encode_calldata(&self) -> Bytes {
        let mut buf = Vec::with_capacity(Self::L1_INFO_TX_LEN);
        self.encode_calldata_header(&mut buf);
        self.encode_calldata_body(&mut buf);
        buf.into()
    }

    /// Encodes the header of the [`L1BlockInfoIsthmus`] object.
    pub fn encode_calldata_header(&self, buf: &mut Vec<u8>) {
        buf.extend_from_slice(Self::L1_INFO_TX_SELECTOR.as_ref());
    }

    /// Encodes the base of the [`L1BlockInfoIsthmus`] object.
    pub fn encode_calldata_body(&self, buf: &mut Vec<u8>) {
        self.base.encode_calldata_body(buf);

        // Encode Isthmus-specific fields
        buf.extend_from_slice(self.operator_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.operator_fee_constant.to_be_bytes().as_ref());
    }

    /// Decodes the [`L1BlockInfoIsthmus`] object from ethereum transaction calldata.
    pub fn decode_calldata(r: &[u8]) -> Result<Self, DecodeError> {
        if r.len() != Self::L1_INFO_TX_LEN {
            return Err(DecodeError::InvalidIsthmusLength(Self::L1_INFO_TX_LEN, r.len()));
        }
        // SAFETY: For all below slice operations, the full
        //         length is validated above to be `176`.
        Self::decode_calldata_body(r)
    }

    /// Decodes the body of the [`L1BlockInfoIsthmus`] object.
    pub fn decode_calldata_body(r: &[u8]) -> Result<Self, DecodeError> {
        let base = L1BlockInfoEcotoneBase::decode_calldata_body(r);

        // Decode Isthmus-specific fields
        // SAFETY: 4 bytes are copied directly into the array
        let mut operator_fee_scalar = [0u8; 4];
        operator_fee_scalar.copy_from_slice(&r[164..168]);
        let operator_fee_scalar = u32::from_be_bytes(operator_fee_scalar);

        // SAFETY: 8 bytes are copied directly into the array
        let mut operator_fee_constant = [0u8; 8];
        operator_fee_constant.copy_from_slice(&r[168..176]);
        let operator_fee_constant = u64::from_be_bytes(operator_fee_constant);

        Ok(Self::new(
            base.number(),
            base.time(),
            base.base_fee(),
            base.block_hash(),
            base.sequence_number(),
            base.batcher_address(),
            base.blob_base_fee(),
            base.blob_base_fee_scalar(),
            base.base_fee_scalar(),
            operator_fee_scalar,
            operator_fee_constant,
        ))
    }
    /// Construct from all values.
    #[allow(clippy::too_many_arguments)]
    pub const fn new(
        number: u64,
        time: u64,
        base_fee: u64,
        block_hash: alloy_primitives::FixedBytes<32>,
        sequence_number: u64,
        batcher_address: Address,
        blob_base_fee: u128,
        blob_base_fee_scalar: u32,
        base_fee_scalar: u32,
        operator_fee_scalar: u32,
        operator_fee_constant: u64,
    ) -> Self {
        Self {
            base: L1BlockInfoEcotoneBase::new(
                number,
                time,
                base_fee,
                block_hash,
                sequence_number,
                batcher_address,
                blob_base_fee,
                blob_base_fee_scalar,
                base_fee_scalar,
            ),
            operator_fee_scalar,
            operator_fee_constant,
        }
    }
    /// Construct from default values and `base_fee`.
    pub fn new_from_base_fee(base_fee: u64) -> Self {
        Self { base: L1BlockInfoEcotoneBase::new_from_base_fee(base_fee), ..Default::default() }
    }
    /// Construct from default values and `sequence_number`.
    pub fn new_from_sequence_number(sequence_number: u64) -> Self {
        Self {
            base: L1BlockInfoEcotoneBase::new_from_sequence_number(sequence_number),
            ..Default::default()
        }
    }
    /// Construct from default values and `batcher_address`.
    pub fn new_from_batcher_address(batcher_address: Address) -> Self {
        Self {
            base: L1BlockInfoEcotoneBase::new_from_batcher_address(batcher_address),
            ..Default::default()
        }
    }
    /// Construct from default values and `base_fee_scalar`.
    pub fn new_from_base_fee_scalar(base_fee: u32) -> Self {
        let base = L1BlockInfoEcotoneBase::new_from_base_fee_scalar(base_fee);
        Self { base, ..Default::default() }
    }
    /// Construct from default values and `blob_base_fee`.
    pub fn new_from_blob_base_fee(blob_base_fee: u128) -> Self {
        let base = L1BlockInfoEcotoneBase::new_from_blob_base_fee(blob_base_fee);
        Self { base, ..Default::default() }
    }
    /// Construct from default values and `blob_base_fee_scalar`.
    pub fn new_from_blob_base_fee_scalar(base_fee_scalar: u32) -> Self {
        let base = L1BlockInfoEcotoneBase::new_from_blob_base_fee_scalar(base_fee_scalar);
        Self { base, ..Default::default() }
    }
    /// Construct from default values and `operator_fee_scalar`.
    pub fn new_from_operator_fee_scalar(operator_fee_scalar: u32) -> Self {
        Self { operator_fee_scalar, ..Default::default() }
    }
    /// Construct from default values and `operator_fee_constant`.
    pub fn new_from_operator_fee_constant(operator_fee_constant: u64) -> Self {
        Self { operator_fee_constant, ..Default::default() }
    }
    /// Construct from default values, `number` and `block_hash`.
    pub fn new_from_number_and_block_hash(number: u64, block_hash: B256) -> Self {
        let base = L1BlockInfoEcotoneBase::new_from_number_and_block_hash(number, block_hash);
        Self { base, ..Default::default() }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;

    #[test]
    fn test_decode_calldata_isthmus_invalid_length() {
        let r = vec![0u8; 1];
        assert_eq!(
            L1BlockInfoIsthmus::decode_calldata(&r),
            Err(DecodeError::InvalidIsthmusLength(L1BlockInfoIsthmus::L1_INFO_TX_LEN, r.len()))
        );
    }

    #[test]
    fn test_l1_block_info_isthmus_roundtrip_calldata_encoding() {
        let info = L1BlockInfoIsthmus::new(
            1,
            2,
            3,
            B256::from([4; 32]),
            5,
            Address::from_slice(&[6; 20]),
            7,
            8,
            9,
            10,
            11,
        );

        let calldata = info.encode_calldata();
        let decoded_info = L1BlockInfoIsthmus::decode_calldata(&calldata).unwrap();

        assert_eq!(info, decoded_info);
    }
}
