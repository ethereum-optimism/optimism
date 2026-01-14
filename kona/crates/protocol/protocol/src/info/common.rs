//! Common encoding and decoding utilities for L1 Block Info transactions.
//!
//! This module contains shared logic for encoding and decoding fields that are
//! common across multiple hardfork versions (Ecotone, Isthmus, Interop).

use alloc::vec::Vec;
use alloy_primitives::{Address, B256, U256};

/// Common fields present in post-Ecotone L1 block info transactions.
///
/// These fields are shared across Ecotone, Isthmus, and Interop hardforks.
#[derive(Debug, Clone, Copy)]
pub(crate) struct CommonL1BlockFields {
    /// The fee scalar for L1 data
    pub base_fee_scalar: u32,
    /// The fee scalar for L1 blobspace data
    pub blob_base_fee_scalar: u32,
    /// The current sequence number
    pub sequence_number: u64,
    /// The current L1 origin block's timestamp
    pub time: u64,
    /// The current L1 origin block number
    pub number: u64,
    /// The current L1 origin block's basefee
    pub base_fee: u64,
    /// The current blob base fee on L1
    pub blob_base_fee: u128,
    /// The current L1 origin block's hash
    pub block_hash: B256,
    /// The address of the batch submitter
    pub batcher_address: Address,
}

impl CommonL1BlockFields {
    /// Encodes the common fields into a buffer.
    ///
    /// The encoding follows this format (excluding the 4-byte selector):
    /// - 4 bytes: BaseFeeScalar
    /// - 4 bytes: BlobBaseFeeScalar
    /// - 8 bytes: SequenceNumber
    /// - 8 bytes: Timestamp
    /// - 8 bytes: L1BlockNumber
    /// - 32 bytes: BaseFee
    /// - 32 bytes: BlobBaseFee
    /// - 32 bytes: BlockHash
    /// - 32 bytes: BatcherHash
    pub(crate) fn encode_into(&self, buf: &mut Vec<u8>) {
        buf.extend_from_slice(self.base_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.blob_base_fee_scalar.to_be_bytes().as_ref());
        buf.extend_from_slice(self.sequence_number.to_be_bytes().as_ref());
        buf.extend_from_slice(self.time.to_be_bytes().as_ref());
        buf.extend_from_slice(self.number.to_be_bytes().as_ref());
        buf.extend_from_slice(U256::from(self.base_fee).to_be_bytes::<32>().as_ref());
        buf.extend_from_slice(U256::from(self.blob_base_fee).to_be_bytes::<32>().as_ref());
        buf.extend_from_slice(self.block_hash.as_ref());
        buf.extend_from_slice(self.batcher_address.into_word().as_ref());
    }

    /// Decodes the common fields from a byte slice.
    ///
    /// This assumes the slice starts at byte offset 4 (after the 4-byte selector)
    /// and contains at least 164 bytes of data.
    ///
    /// # Safety
    /// This method assumes the slice is at least 164 bytes long and starts at
    /// the correct offset. Callers must validate the length before calling.
    pub(crate) fn decode_from(r: &[u8]) -> Self {
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

        Self {
            base_fee_scalar,
            blob_base_fee_scalar,
            sequence_number,
            time,
            number,
            base_fee,
            blob_base_fee,
            block_hash,
            batcher_address,
        }
    }
}
