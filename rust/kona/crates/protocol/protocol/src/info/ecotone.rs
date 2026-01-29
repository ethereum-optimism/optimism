//! Contains ecotone-specific L1 block info types.

use crate::{
    DecodeError,
    info::{
        L1BlockInfoEcotoneBaseFields,
        bedrock_base::{
            L1BlockInfoBedrockBaseFields, ambassador_impl_L1BlockInfoBedrockBaseFields,
        },
        ecotone_base::{L1BlockInfoEcotoneBase, ambassador_impl_L1BlockInfoEcotoneBaseFields},
    },
};
use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes, U256};
use ambassador::Delegate;

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
#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy, Delegate)]
#[allow(clippy::duplicated_attributes)]
#[delegate(L1BlockInfoBedrockBaseFields, target = "base")]
#[delegate(L1BlockInfoEcotoneBaseFields, target = "base")]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct L1BlockInfoEcotone {
    #[cfg_attr(feature = "serde", serde(flatten))]
    base: L1BlockInfoEcotoneBase,
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

/// Accessors to fields deprecated in later Isthmus.
pub trait L1BlockInfoEcotoneOnlyFields {
    /// Indicates that the scalars are empty.
    /// This is an edge case where the first block in ecotone has no scalars,
    /// so the bedrock tx l1 cost function needs to be used.
    fn empty_scalars(&self) -> bool;

    /// The l1 fee overhead used along with the `empty_scalars` field for the
    /// bedrock tx l1 cost function.
    ///
    /// This field is deprecated in the Ecotone Hardfork.
    fn l1_fee_overhead(&self) -> U256;
}

impl L1BlockInfoEcotoneOnlyFields for L1BlockInfoEcotone {
    fn empty_scalars(&self) -> bool {
        self.empty_scalars
    }

    fn l1_fee_overhead(&self) -> U256 {
        self.l1_fee_overhead
    }
}

/// Accessors for all Ecotone fields.
pub trait L1BlockInfoEcotoneFields:
    L1BlockInfoBedrockBaseFields + L1BlockInfoEcotoneOnlyFields
{
}

impl L1BlockInfoEcotoneFields for L1BlockInfoEcotone {}

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
        self.encode_ecotone_header(&mut buf);
        self.base.encode_calldata_body(&mut buf);
        // Notice: do not include the `empty_scalars` field in the calldata.
        // Notice: do not include the `l1_fee_overhead` field in the calldata.
        buf.into()
    }

    /// Encodes the header part of the [`L1BlockInfoEcotone`] object.
    pub fn encode_ecotone_header(&self, buf: &mut Vec<u8>) {
        buf.extend_from_slice(Self::L1_INFO_TX_SELECTOR.as_ref())
    }

    /// Decodes the [`L1BlockInfoEcotone`] object from ethereum transaction calldata.
    pub fn decode_calldata(r: &[u8]) -> Result<Self, DecodeError> {
        if r.len() != Self::L1_INFO_TX_LEN {
            return Err(DecodeError::InvalidEcotoneLength(Self::L1_INFO_TX_LEN, r.len()));
        }
        // SAFETY: For all below slice operations, the full
        //         length is validated above to be `164`.
        let base = L1BlockInfoEcotoneBase::decode_calldata_body(r);

        Ok(Self::new(
            base.number(),
            base.time(),
            base.base_fee(),
            base.block_hash(),
            base.sequence_number(),
            base.batcher_address(),
            base.blob_base_fee,
            base.blob_base_fee_scalar,
            base.base_fee_scalar,
            // Notice: the `empty_scalars` field is not included in the calldata.
            // This is used by the evm to indicate that the bedrock tx l1 cost function
            // needs to be used.
            false,
            // Notice: the `l1_fee_overhead` field is not included in the calldata.
            U256::ZERO,
        ))
    }

    /// Construct from all values.
    #[allow(clippy::too_many_arguments)]
    pub(crate) const fn new(
        number: u64,
        time: u64,
        base_fee: u64,
        block_hash: B256,
        sequence_number: u64,
        batcher_address: Address,
        blob_base_fee: u128,
        blob_base_fee_scalar: u32,
        base_fee_scalar: u32,
        empty_scalars: bool,
        l1_fee_overhead: U256,
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
            empty_scalars,
            l1_fee_overhead,
        }
    }
    /// Construct from default values and `base_fee`.
    pub fn new_from_base_fee(base_fee: u64) -> Self {
        Self { base: L1BlockInfoEcotoneBase::new_from_base_fee(base_fee), ..Default::default() }
    }
    /// Construct from default values and `block_hash`.
    pub fn new_from_block_hash(block_hash: B256) -> Self {
        let base = L1BlockInfoEcotoneBase::new_from_block_hash(block_hash);
        Self { base, ..Default::default() }
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
    /// Construct from default values and `base_fee_scalar`.
    pub fn new_from_base_fee_scalar(base_fee: u32) -> Self {
        let base = L1BlockInfoEcotoneBase::new_from_base_fee_scalar(base_fee);
        Self { base, ..Default::default() }
    }
    /// Construct from default values and `l1_fee_overhead`.
    pub fn new_from_l1_fee_overhead(l1_fee_overhead: U256) -> Self {
        Self { l1_fee_overhead, ..Default::default() }
    }
    /// Construct from default values and `empty_scalars`.
    pub fn new_from_empty_scalars(empty_scalars: bool) -> Self {
        Self { empty_scalars, ..Default::default() }
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
    fn test_decode_calldata_ecotone_invalid_length() {
        let r = vec![0u8; 1];
        assert_eq!(
            L1BlockInfoEcotone::decode_calldata(&r),
            Err(DecodeError::InvalidEcotoneLength(L1BlockInfoEcotone::L1_INFO_TX_LEN, r.len(),))
        );
    }

    #[test]
    fn test_l1_block_info_ecotone_roundtrip_calldata_encoding() {
        let info = L1BlockInfoEcotone::new(
            1,
            2,
            3,
            B256::from([4u8; 32]),
            5,
            Address::from([6u8; 20]),
            7,
            8,
            9,
            false,
            U256::ZERO,
        );

        let calldata = info.encode_calldata();
        let decoded_info = L1BlockInfoEcotone::decode_calldata(&calldata).unwrap();
        assert_eq!(info, decoded_info);
    }
}
