//! Contains bedrock-specific L1 block info types.

use ambassador::Delegate;

use crate::info::bedrock_base::ambassador_impl_L1BlockInfoBedrockBaseFields;
use alloc::vec::Vec;
use alloy_primitives::{Address, B256, Bytes, U256};

use crate::{
    DecodeError,
    info::{L1BlockInfoBedrockBaseFields, bedrock_base::L1BlockInfoBedrockBase},
};
/// Represents the fields within a Bedrock L1 block info transaction.
///
/// Bedrock Binary Format
// +---------+--------------------------+
// | Bytes   | Field                    |
// +---------+--------------------------+
// | 4       | Function signature       |
// | 32      | Number                   |
// | 32      | Time                     |
// | 32      | BaseFee                  |
// | 32      | BlockHash                |
// | 32      | SequenceNumber           |
// | 32      | BatcherHash              |
// | 32      | L1FeeOverhead            |
// | 32      | L1FeeScalar              |
// +---------+--------------------------+
#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy, Delegate)]
#[delegate(L1BlockInfoBedrockBaseFields, target = "base")]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct L1BlockInfoBedrock {
    #[cfg_attr(feature = "serde", serde(flatten))]
    base: L1BlockInfoBedrockBase,
    /// The fee overhead for L1 data. Deprecated in Ecotone.
    pub l1_fee_overhead: U256,
    /// The fee scalar for L1 data. Deprecated in Ecotone.
    pub l1_fee_scalar: U256,
}

/// Accessors for fields deprecated after Bedrock.
pub trait L1BlockInfoBedrockOnlyFields {
    /// The fee overhead for L1 data. Deprecated in Ecotone.
    fn l1_fee_overhead(&self) -> U256;

    /// The fee scalar for L1 data. Deprecated in Ecotone.
    fn l1_fee_scalar(&self) -> U256;
}

impl L1BlockInfoBedrockOnlyFields for L1BlockInfoBedrock {
    fn l1_fee_overhead(&self) -> U256 {
        self.l1_fee_overhead
    }

    fn l1_fee_scalar(&self) -> U256 {
        self.l1_fee_scalar
    }
}

/// Accessors trait for all fields on [`L1BlockInfoBedrock`].
pub trait L1BlockInfoBedrockFields:
    L1BlockInfoBedrockBaseFields + L1BlockInfoBedrockOnlyFields
{
}

impl L1BlockInfoBedrockFields for L1BlockInfoBedrock {}

impl L1BlockInfoBedrock {
    /// The length of an L1 info transaction in Bedrock.
    pub const L1_INFO_TX_LEN: usize = 4 + 32 * 8;

    /// The 4 byte selector of the
    /// "setL1BlockValues(uint64,uint64,uint256,bytes32,uint64,bytes32,uint256,uint256)" function
    pub const L1_INFO_TX_SELECTOR: [u8; 4] = [0x01, 0x5d, 0x8e, 0xb9];

    /// Encodes the [`L1BlockInfoBedrock`] object into Ethereum transaction calldata.
    pub fn encode_calldata(&self) -> Bytes {
        let mut buf = Vec::with_capacity(Self::L1_INFO_TX_LEN);
        buf.extend_from_slice(Self::L1_INFO_TX_SELECTOR.as_ref());
        buf.extend_from_slice(U256::from(self.number()).to_be_bytes::<32>().as_slice());
        buf.extend_from_slice(U256::from(self.time()).to_be_bytes::<32>().as_slice());
        buf.extend_from_slice(U256::from(self.base_fee()).to_be_bytes::<32>().as_slice());
        buf.extend_from_slice(self.block_hash().as_slice());
        buf.extend_from_slice(U256::from(self.sequence_number()).to_be_bytes::<32>().as_slice());
        buf.extend_from_slice(self.batcher_address().into_word().as_slice());
        buf.extend_from_slice(self.l1_fee_overhead().to_be_bytes::<32>().as_slice());
        buf.extend_from_slice(self.l1_fee_scalar().to_be_bytes::<32>().as_slice());
        buf.into()
    }

    /// Decodes the [`L1BlockInfoBedrock`] object from ethereum transaction calldata.
    pub fn decode_calldata(r: &[u8]) -> Result<Self, DecodeError> {
        if r.len() != Self::L1_INFO_TX_LEN {
            return Err(DecodeError::InvalidBedrockLength(Self::L1_INFO_TX_LEN, r.len()));
        }

        // SAFETY: For all below slice operations, the full
        //         length is validated above to be `260`.

        // SAFETY: 8 bytes are copied directly into the array
        let mut number = [0u8; 8];
        number.copy_from_slice(&r[28..36]);
        let number = u64::from_be_bytes(number);

        // SAFETY: 8 bytes are copied directly into the array
        let mut time = [0u8; 8];
        time.copy_from_slice(&r[60..68]);
        let time = u64::from_be_bytes(time);

        // SAFETY: 8 bytes are copied directly into the array
        let mut base_fee = [0u8; 8];
        base_fee.copy_from_slice(&r[92..100]);
        let base_fee = u64::from_be_bytes(base_fee);

        let block_hash = B256::from_slice(r[100..132].as_ref());

        // SAFETY: 8 bytes are copied directly into the array
        let mut sequence_number = [0u8; 8];
        sequence_number.copy_from_slice(&r[156..164]);
        let sequence_number = u64::from_be_bytes(sequence_number);

        let batcher_address = Address::from_slice(r[176..196].as_ref());
        let l1_fee_overhead = U256::from_be_slice(r[196..228].as_ref());
        let l1_fee_scalar = U256::from_be_slice(r[228..260].as_ref());

        Ok(Self::new(
            number,
            time,
            base_fee,
            block_hash,
            sequence_number,
            batcher_address,
            l1_fee_overhead,
            l1_fee_scalar,
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
        l1_fee_overhead: U256,
        l1_fee_scalar: U256,
    ) -> Self {
        Self {
            base: L1BlockInfoBedrockBase::new(
                number,
                time,
                base_fee,
                block_hash,
                sequence_number,
                batcher_address,
            ),
            l1_fee_overhead,
            l1_fee_scalar,
        }
    }
    /// Construct from default values and `base_fee`.
    pub fn new_from_base_fee(base_fee: u64) -> Self {
        Self { base: L1BlockInfoBedrockBase::new_from_base_fee(base_fee), ..Default::default() }
    }
    /// Construct from default values and `block_hash`.
    pub fn new_from_block_hash(block_hash: B256) -> Self {
        Self { base: L1BlockInfoBedrockBase::new_from_block_hash(block_hash), ..Default::default() }
    }
    /// Construct from default values and `sequence_number`.
    pub fn new_from_sequence_number(sequence_number: u64) -> Self {
        Self {
            base: L1BlockInfoBedrockBase::new_from_sequence_number(sequence_number),
            ..Default::default()
        }
    }
    /// Construct from default values and `batcher_address`.
    pub fn new_from_batcher_address(batcher_address: Address) -> Self {
        Self {
            base: L1BlockInfoBedrockBase::new_from_batcher_address(batcher_address),
            ..Default::default()
        }
    }
    /// Construct from default values and `l1_fee_scalar`.
    pub fn new_from_l1_fee_scalar(l1_fee_scalar: U256) -> Self {
        Self { l1_fee_scalar, ..Default::default() }
    }
    /// Construct from default values and `l1_fee_overhead`.
    pub fn new_from_l1_fee_overhead(l1_fee_overhead: U256) -> Self {
        Self { l1_fee_overhead, ..Default::default() }
    }
    /// Construct from default values, `number` and `block_hash`.
    pub fn new_from_number_and_block_hash(number: u64, block_hash: B256) -> Self {
        Self {
            base: L1BlockInfoBedrockBase::new_from_number_and_block_hash(number, block_hash),
            ..Default::default()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;

    #[test]
    fn test_decode_calldata_bedrock_invalid_length() {
        let r = vec![0u8; 1];
        assert_eq!(
            L1BlockInfoBedrock::decode_calldata(&r),
            Err(DecodeError::InvalidBedrockLength(L1BlockInfoBedrock::L1_INFO_TX_LEN, r.len(),))
        );
    }

    #[test]
    fn test_l1_block_info_bedrock_roundtrip_calldata_encoding() {
        let info = L1BlockInfoBedrock::new(
            1,
            2,
            3,
            B256::from([4u8; 32]),
            5,
            Address::from([6u8; 20]),
            U256::from(7),
            U256::from(8),
        );

        let calldata = info.encode_calldata();
        let decoded_info = L1BlockInfoBedrock::decode_calldata(&calldata).unwrap();
        assert_eq!(info, decoded_info);
    }
}
