use alloc::vec::Vec;
use alloy_primitives::{Address, B256, U256};
use ambassador::{Delegate, delegatable_trait};

use crate::info::{
    L1BlockInfoBedrockBaseFields,
    bedrock_base::{L1BlockInfoBedrockBase, ambassador_impl_L1BlockInfoBedrockBaseFields},
};

#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy, Delegate)]
#[delegate(L1BlockInfoBedrockBaseFields, target = "base")]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub(crate) struct L1BlockInfoEcotoneBase {
    base: L1BlockInfoBedrockBase,
    /// The current blob base fee on L1
    pub blob_base_fee: u128,
    /// The fee scalar for L1 blobspace data
    pub blob_base_fee_scalar: u32,
    /// The fee scalar for L1 data
    pub base_fee_scalar: u32,
}

impl L1BlockInfoEcotoneBase {
    /// Construct new from all values.
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
            blob_base_fee,
            blob_base_fee_scalar,
            base_fee_scalar,
        }
    }
    /// Construct from default values and `base_fee`.
    pub(crate) fn new_from_base_fee(base_fee: u64) -> Self {
        Self { base: L1BlockInfoBedrockBase::new_from_base_fee(base_fee), ..Default::default() }
    }
    /// Construct from default values and `block_hash`.
    pub(crate) fn new_from_block_hash(block_hash: B256) -> Self {
        let base = L1BlockInfoBedrockBase::new_from_block_hash(block_hash);
        Self { base, ..Default::default() }
    }
    /// Construct from default values and `sequence_number`.
    pub(crate) fn new_from_sequence_number(sequence_number: u64) -> Self {
        Self {
            base: L1BlockInfoBedrockBase::new_from_sequence_number(sequence_number),
            ..Default::default()
        }
    }
    /// Construct from default values and `batcher_address`.
    pub(crate) fn new_from_batcher_address(batcher_address: Address) -> Self {
        Self {
            base: L1BlockInfoBedrockBase::new_from_batcher_address(batcher_address),
            ..Default::default()
        }
    }
    /// Construct from default values and `blob_base_fee`.
    pub(crate) fn new_from_blob_base_fee(blob_base_fee: u128) -> Self {
        Self { base: Default::default(), blob_base_fee, ..Default::default() }
    }
    /// Construct from default values and `blob_base_fee_scalar`.
    pub(crate) fn new_from_blob_base_fee_scalar(blob_base_fee_scalar: u32) -> Self {
        Self { base: Default::default(), blob_base_fee_scalar, ..Default::default() }
    }
    /// Construct from default values and `base_fee_scalar`.
    pub(crate) fn new_from_base_fee_scalar(base_fee_scalar: u32) -> Self {
        Self { base: Default::default(), base_fee_scalar, ..Default::default() }
    }
    /// Construct from default values, `number` and `block_hash`.
    pub(crate) fn new_from_number_and_block_hash(number: u64, block_hash: B256) -> Self {
        let base = L1BlockInfoBedrockBase::new_from_number_and_block_hash(number, block_hash);
        Self { base, ..Default::default() }
    }

    pub(crate) fn encode_calldata_body(&self, buf: &mut Vec<u8>) {
        // We cannot `self.base.encode_bedrock_base(buf)` here, because the fields do not match.
        buf.extend_from_slice(self.base_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.blob_base_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.base.sequence_number.to_be_bytes().as_ref());
        buf.extend_from_slice(self.base.time.to_be_bytes().as_ref());
        buf.extend_from_slice(self.base.number.to_be_bytes().as_ref());
        buf.extend_from_slice(U256::from(self.base.base_fee).to_be_bytes::<32>().as_ref());
        buf.extend_from_slice(U256::from(self.blob_base_fee).to_be_bytes::<32>().as_ref());
        buf.extend_from_slice(self.base.block_hash.as_ref());
        buf.extend_from_slice(self.base.batcher_address.into_word().as_ref());
        // Notice: do not include the `empty_scalars` field in the calldata.
        // Notice: do not include the `l1_fee_overhead` field in the calldata.
    }

    /// Decodes the Ecotone base fields from a byte slice.
    ///
    /// This assumes the slice starts at byte offset 4 (after the 4-byte selector)
    /// and contains at least 164 bytes of data.
    ///
    /// # Safety
    /// This method assumes the slice is at least 164 bytes long and starts at
    /// the correct offset. Callers must validate the length before calling.
    pub(crate) fn decode_calldata_body(r: &[u8]) -> Self {
        // SAFETY: All slice operations below assume r is at least 164 bytes.
        // The caller must validate this before calling this method.

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

        Self::new(
            number,
            time,
            base_fee,
            block_hash,
            sequence_number,
            batcher_address,
            blob_base_fee,
            blob_base_fee_scalar,
            base_fee_scalar,
        )
    }
}
/// Accessors to fields in Ecotone and later.
#[delegatable_trait]
pub trait L1BlockInfoEcotoneBaseFields: L1BlockInfoBedrockBaseFields {
    /// The current blob base fee on L1
    fn blob_base_fee(&self) -> u128;
    /// The fee scalar for L1 blobspace data
    fn blob_base_fee_scalar(&self) -> u32;
    /// The fee scalar for L1 data
    fn base_fee_scalar(&self) -> u32;
}

impl L1BlockInfoEcotoneBaseFields for L1BlockInfoEcotoneBase {
    /// The current blob base fee on L1
    fn blob_base_fee(&self) -> u128 {
        self.blob_base_fee
    }
    /// The fee scalar for L1 blobspace data
    fn blob_base_fee_scalar(&self) -> u32 {
        self.blob_base_fee_scalar
    }
    /// The fee scalar for L1 data
    fn base_fee_scalar(&self) -> u32 {
        self.base_fee_scalar
    }
}
