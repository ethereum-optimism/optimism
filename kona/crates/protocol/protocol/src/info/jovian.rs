//! Jovian L1 Block Info transaction types.

use crate::{
    DecodeError, L1BlockInfoIsthmus,
    info::{
        L1BlockInfoBedrockBaseFields, L1BlockInfoEcotoneBaseFields,
        bedrock_base::ambassador_impl_L1BlockInfoBedrockBaseFields,
        ecotone_base::ambassador_impl_L1BlockInfoEcotoneBaseFields,
        isthmus::{L1BlockInfoIsthmusBaseFields, ambassador_impl_L1BlockInfoIsthmusBaseFields},
    },
};
use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes};
use ambassador::{self, Delegate};

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
#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy, Delegate)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[allow(clippy::duplicated_attributes)]
#[delegate(L1BlockInfoBedrockBaseFields, target = "base")]
#[delegate(L1BlockInfoEcotoneBaseFields, target = "base")]
#[delegate(L1BlockInfoIsthmusBaseFields, target = "base")]
pub struct L1BlockInfoJovian {
    /// Fields inherited from Isthmus.
    #[cfg_attr(feature = "serde", serde(flatten))]
    pub base: L1BlockInfoIsthmus,
    /// The DA footprint gas scalar
    pub da_footprint_gas_scalar: u16,
}
/// Accessors to fields available in Jovian and later.
pub trait L1BlockInfoJovianBaseFields: L1BlockInfoIsthmusBaseFields {
    /// The DA footprint gas scalar
    fn da_footprint_gas_scalar(&self) -> u16;
}

impl L1BlockInfoJovianBaseFields for L1BlockInfoJovian {
    fn da_footprint_gas_scalar(&self) -> u16 {
        self.da_footprint_gas_scalar
    }
}

/// Accessors for all Jovian fields.
pub trait L1BlockInfoJovianFields:
    L1BlockInfoIsthmusBaseFields + L1BlockInfoJovianBaseFields
{
}

impl L1BlockInfoJovianFields for L1BlockInfoJovian {}

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
        self.encode_calldata_header(&mut buf);
        self.encode_calldata_body(&mut buf);
        buf.into()
    }

    /// Encodes the header part of the [`L1BlockInfoJovian`] object.
    pub fn encode_calldata_header(&self, buf: &mut Vec<u8>) {
        buf.extend_from_slice(Self::L1_INFO_TX_SELECTOR.as_ref());
    }

    /// Encodes the base part of the [`L1BlockInfoJovian`] object.
    pub fn encode_calldata_body(&self, buf: &mut Vec<u8>) {
        self.base.encode_calldata_body(buf);
        buf.extend_from_slice(self.da_footprint_gas_scalar.to_be_bytes().as_ref());
    }

    /// Decodes the [`L1BlockInfoJovian`] object from ethereum transaction calldata.
    pub fn decode_calldata(r: &[u8]) -> Result<Self, DecodeError> {
        if r.len() != Self::L1_INFO_TX_LEN {
            return Err(DecodeError::InvalidJovianLength(Self::L1_INFO_TX_LEN, r.len()));
        }
        Self::decode_calldata_body(r)
    }

    /// Decodes the body of the [`L1BlockInfoJovian`] object.
    pub fn decode_calldata_body(r: &[u8]) -> Result<Self, DecodeError> {
        // SAFETY: For all below slice operations, the full
        //         length is validated above to be `178`.

        let base = L1BlockInfoIsthmus::decode_calldata_body(r)?;

        // SAFETY: 2 bytes are copied directly into the array
        let mut da_footprint_gas_scalar = [0u8; 2];
        da_footprint_gas_scalar.copy_from_slice(&r[176..178]);
        let mut da_footprint_gas_scalar = u16::from_be_bytes(da_footprint_gas_scalar);

        // If the da footprint gas scalar is 0, use the default value (`https://github.com/ethereum-optimism/specs/blob/664cba65ab9686b0e70ad19fdf2ad054d6295986/specs/protocol/jovian/l1-attributes.md#overview`).
        if da_footprint_gas_scalar == 0 {
            da_footprint_gas_scalar = Self::DEFAULT_DA_FOOTPRINT_GAS_SCALAR;
        }

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
            base.operator_fee_scalar(),
            base.operator_fee_constant(),
            da_footprint_gas_scalar,
        ))
    }

    /// Construct from all values.
    #[allow(clippy::too_many_arguments)]
    pub const fn new(
        number: u64,
        time: u64,
        base_fee: u64,
        block_hash: B256,
        sequence_number: u64,
        batcher_address: Address,
        blob_base_fee: u128,
        blob_base_fee_scalar: u32,
        base_fee_scalar: u32,
        operator_fee_scalar: u32,
        operator_fee_constant: u64,
        da_footprint_gas_scalar: u16,
    ) -> Self {
        Self {
            base: L1BlockInfoIsthmus::new(
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
            ),
            da_footprint_gas_scalar,
        }
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
        let info = L1BlockInfoJovian::new(
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
            12,
        );

        let calldata = info.encode_calldata();
        let decoded_info = L1BlockInfoJovian::decode_calldata(&calldata).unwrap();

        assert_eq!(info, decoded_info);
    }
}
