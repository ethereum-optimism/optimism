//! Facts the host pushes into the circuit, and their conversions to circuit rows.
//!
//! The circuit never observes the world directly: actors decode what they see (L1 blocks and
//! receipts, engine state, engine-confirmed derived blocks) into these plain types and push
//! them through a [`ChainViewClient`](crate::ChainViewClient). Every conversion is total
//! except `u64 -> BIGINT`, which fails above `i64::MAX`.

use std::sync::Arc;

use alloy_primitives::{Address, B256};
use dbsp::utils::{Tup3, Tup4, Tup5, Tup8, Tup10};
use feldera_sqllib::{ByteArray, SqlString};
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_consensus::OpBlock;

use crate::handles::{L1BlockRow, L1StatusRow, L2SafeRow, L2StatusRow, UnsafeBlockSignerRow};

/// Error converting a fact into a circuit row.
#[derive(Debug, Clone, Copy, PartialEq, Eq, thiserror::Error)]
pub enum FactError {
    /// A `u64` did not fit the circuit's `BIGINT`.
    #[error("{field} = {value} does not fit a BIGINT")]
    OutOfRange {
        /// The column that overflowed.
        field: &'static str,
        /// The offending value.
        value: u64,
    },
}

fn bigint(field: &'static str, value: u64) -> Result<i64, FactError> {
    i64::try_from(value).map_err(|_| FactError::OutOfRange { field, value })
}

/// Encodes a 32-byte hash as a circuit `BINARY(32)`.
pub fn bytes32(hash: B256) -> ByteArray {
    ByteArray::new(hash.as_slice())
}

/// Encodes an address as a circuit `BINARY(20)`.
pub fn bytes20(address: Address) -> ByteArray {
    ByteArray::new(address.as_slice())
}

/// Decodes a circuit `BINARY(32)` back into a hash; `None` if the length is wrong.
pub fn hash_from_bytes(bytes: &ByteArray) -> Option<B256> {
    let slice = bytes.as_slice();
    (slice.len() == 32).then(|| B256::from_slice(slice))
}

/// Decodes a circuit `BINARY(20)` back into an address; `None` if the length is wrong.
pub fn address_from_bytes(bytes: &ByteArray) -> Option<Address> {
    let slice = bytes.as_slice();
    (slice.len() == 20).then(|| Address::from_slice(slice))
}

/// Which `l1_status` row a fact replaces.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum L1StatusKind {
    /// The canonical tracker's tip (the L1 head as far as the chain view knows).
    Head,
    /// The L1 `safe` tag.
    Safe,
    /// The L1 `finalized` tag.
    Finalized,
    /// The derivation pipeline's current L1 origin.
    Current,
}

impl L1StatusKind {
    /// The `kind` column value.
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Head => "head",
            Self::Safe => "safe",
            Self::Finalized => "finalized",
            Self::Current => "current",
        }
    }

    /// Every kind, in a fixed order.
    pub const ALL: [Self; 4] = [Self::Head, Self::Safe, Self::Finalized, Self::Current];
}

/// Which `l2_status` row a head label occupies.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum L2StatusKind {
    /// The unsafe head.
    Unsafe,
    /// The local-safe head.
    LocalSafe,
    /// The safe head.
    Safe,
    /// The finalized head.
    Finalized,
}

impl L2StatusKind {
    /// The `kind` column value.
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Unsafe => "unsafe",
            Self::LocalSafe => "local_safe",
            Self::Safe => "safe",
            Self::Finalized => "finalized",
        }
    }

    /// Every kind, in a fixed order.
    pub const ALL: [Self; 4] = [Self::Unsafe, Self::LocalSafe, Self::Safe, Self::Finalized];
}

/// The engine's four head labels, as published by its state watch.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct L2Heads {
    /// The unsafe head.
    pub unsafe_head: L2BlockInfo,
    /// The local-safe head.
    pub local_safe_head: L2BlockInfo,
    /// The safe head.
    pub safe_head: L2BlockInfo,
    /// The finalized head.
    pub finalized_head: L2BlockInfo,
}

impl L2Heads {
    /// The head stored under `kind`.
    pub const fn get(&self, kind: L2StatusKind) -> L2BlockInfo {
        match kind {
            L2StatusKind::Unsafe => self.unsafe_head,
            L2StatusKind::LocalSafe => self.local_safe_head,
            L2StatusKind::Safe => self.safe_head,
            L2StatusKind::Finalized => self.finalized_head,
        }
    }
}

/// An engine-confirmed derived L2 block together with the L1 block it was derived from.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct L2SafeFact {
    /// Host-assigned, monotone over the life of the process; the tie-breaker in every view.
    pub seq: u64,
    /// The confirmed block.
    pub block: L2BlockInfo,
    /// The pipeline's L1 origin when the block's attributes were produced.
    pub derived_from: BlockInfo,
}

/// An L2 block the engine imported, with its derivation summary.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ImportedL2Block {
    /// The block's number, hash, L1 origin and sequence number.
    pub info: L2BlockInfo,
    /// The full block.
    pub block: Arc<OpBlock>,
}

/// A fact pushed into the circuit.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Fact {
    /// The derivation pipeline's L1 origin advanced to this block. The driver keeps one block
    /// per height: a block it already holds with the same hash is a no-op, and one with another
    /// hash (a reorg the pipeline re-walked) is retracted before the new one is inserted.
    L1Origin(BlockInfo),
    /// The unsafe-block signer read from the `SystemConfig` contract at L1 block `l1`; the
    /// driver retracts the previous row.
    UnsafeBlockSigner {
        /// The L1 block the contract was read at.
        l1: BlockInfo,
        /// The signer it held.
        signer: Address,
    },
    /// A current L1 status changed; the driver retracts the previous row of the same kind.
    L1Status {
        /// Which status.
        kind: L1StatusKind,
        /// Its new value.
        block: BlockInfo,
    },
    /// The engine's head labels changed; the driver retracts the previous rows.
    L2Status(Box<L2Heads>),
    /// The engine confirmed a derived L2 block; the driver retracts any earlier row at the
    /// same height.
    L2Safe(L2SafeFact),
    /// The pipeline was reset to the safe head `(l2_number, l2_hash)`: derived blocks above it
    /// are no longer facts, and neither is a block at that height with another hash.
    L2SafeRetractAbove {
        /// The number of the safe head the pipeline was reset to.
        l2_number: u64,
        /// Its hash.
        l2_hash: B256,
    },
    /// The engine imported an L2 block. No table holds it: the driver keeps the block for
    /// derivation's hash-keyed lookups, dropping it once it is below the engine's finalized
    /// head or the newest `imported_limit` blocks no longer include it.
    L2Imported(ImportedL2Block),
}

/// The `l1_blocks` row for a canonical block.
pub fn l1_block_row(block: &BlockInfo) -> Result<L1BlockRow, FactError> {
    Ok(Tup4::new(
        bigint("l1_blocks.number", block.number)?,
        bytes32(block.hash),
        bytes32(block.parent_hash),
        bigint("l1_blocks.ts", block.timestamp)?,
    ))
}

/// The `unsafe_block_signer` row for the signer read at `l1`.
pub fn unsafe_block_signer_row(
    l1: &BlockInfo,
    signer: Address,
) -> Result<UnsafeBlockSignerRow, FactError> {
    Ok(Tup3::new(
        bigint("unsafe_block_signer.l1_number", l1.number)?,
        bytes32(l1.hash),
        bytes20(signer),
    ))
}

/// The `l1_status` row for `kind`.
pub fn l1_status_row(kind: L1StatusKind, block: &BlockInfo) -> Result<L1StatusRow, FactError> {
    Ok(Tup5::new(
        SqlString::from_ref(kind.as_str()),
        bigint("l1_status.number", block.number)?,
        bytes32(block.hash),
        bytes32(block.parent_hash),
        bigint("l1_status.ts", block.timestamp)?,
    ))
}

/// The `l2_status` row for `kind`.
pub fn l2_status_row(kind: L2StatusKind, block: &L2BlockInfo) -> Result<L2StatusRow, FactError> {
    Ok(Tup8::new(
        SqlString::from_ref(kind.as_str()),
        bigint("l2_status.number", block.block_info.number)?,
        bytes32(block.block_info.hash),
        bytes32(block.block_info.parent_hash),
        bigint("l2_status.ts", block.block_info.timestamp)?,
        bigint("l2_status.l1_origin_number", block.l1_origin.number)?,
        bytes32(block.l1_origin.hash),
        bigint("l2_status.seq_num", block.seq_num)?,
    ))
}

/// The `l2_safe_blocks` row for a confirmed derived block.
pub fn l2_safe_row(fact: &L2SafeFact) -> Result<L2SafeRow, FactError> {
    Ok(Tup10::new(
        bigint("l2_safe_blocks.seq", fact.seq)?,
        bigint("l2_safe_blocks.l2_number", fact.block.block_info.number)?,
        bytes32(fact.block.block_info.hash),
        bytes32(fact.block.block_info.parent_hash),
        bigint("l2_safe_blocks.l2_ts", fact.block.block_info.timestamp)?,
        bigint("l2_safe_blocks.l1_origin_number", fact.block.l1_origin.number)?,
        bytes32(fact.block.l1_origin.hash),
        bigint("l2_safe_blocks.seq_num", fact.block.seq_num)?,
        bigint("l2_safe_blocks.derived_from_number", fact.derived_from.number)?,
        bytes32(fact.derived_from.hash),
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn l1(number: u64) -> BlockInfo {
        BlockInfo {
            hash: B256::repeat_byte(number as u8),
            number,
            parent_hash: B256::ZERO,
            timestamp: 12 * number,
        }
    }

    #[test]
    fn bigint_conversion_rejects_values_above_i64_max() {
        let block = BlockInfo { number: u64::MAX, ..l1(1) };
        assert_eq!(
            l1_block_row(&block),
            Err(FactError::OutOfRange { field: "l1_blocks.number", value: u64::MAX })
        );
    }

    #[test]
    fn byte_round_trips() {
        let hash = B256::repeat_byte(0xab);
        assert_eq!(hash_from_bytes(&bytes32(hash)), Some(hash));
        let address = Address::repeat_byte(0xcd);
        assert_eq!(address_from_bytes(&bytes20(address)), Some(address));
        assert_eq!(hash_from_bytes(&bytes20(address)), None);
    }
}
