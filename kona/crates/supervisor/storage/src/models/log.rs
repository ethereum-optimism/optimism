//! Models for storing blockchain logs in the database.
//!
//! This module defines the data structure and table mapping for logs emitted during
//! transaction execution. Each log is uniquely identified by its block number and
//! index within the block.
//!
//! The table is dup-sorted, allowing efficient grouping of multiple logs per block.
//! It supports fast appends, retrieval, and range queries ordered by log index.

use alloy_primitives::B256;
use bytes::{Buf, BufMut};
use kona_supervisor_types::{ExecutingMessage, Log};
use reth_codecs::Compact;
use serde::{Deserialize, Serialize};

/// Metadata associated with a single emitted log.
///
/// This is the value stored in the [`crate::models::LogEntries`] dup-sorted table. Each entry
/// includes:
/// - `index` - Index of the log in a block.
/// - `hash`: The keccak256 hash of the log event.
/// - `executing_message` - An optional field that may contain a cross-domain execution message.
#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct LogEntry {
    /// Index of the log.
    pub index: u32,
    /// The keccak256 hash of the emitted log event.
    pub hash: B256,
    /// Optional cross-domain execution message.
    pub executing_message: Option<ExecutingMessageEntry>,
}
/// Compact encoding and decoding implementation for [`LogEntry`].
///
/// This encoding is used for storing log entries in dup-sorted tables,
/// where the `index` field is treated as the subkey. The layout is optimized
/// for lexicographic ordering by `index`.
///
/// ## Encoding Layout (ordered):
/// - `index: u32` – Log index (subkey), used for ordering within dup table.
/// - `has_msg: u8` – 1 if `executing_message` is present, 0 otherwise.
/// - `hash: B256` – 32-byte Keccak256 hash of the log.
/// - `executing_message: Option<ExecutingMessageEntry>` – Compact-encoded if present.
impl Compact for LogEntry {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: BufMut + AsMut<[u8]>,
    {
        let start_len = buf.remaining_mut();

        buf.put_u32(self.index); // Subkey must be at first
        buf.put_u8(self.executing_message.is_some() as u8);
        buf.put_slice(self.hash.as_slice());

        if let Some(msg) = &self.executing_message {
            msg.to_compact(buf);
        }

        start_len - buf.remaining_mut()
    }

    fn from_compact(mut buf: &[u8], _len: usize) -> (Self, &[u8]) {
        let index = buf.get_u32();
        let has_msg = buf.get_u8() != 0;

        assert!(buf.len() >= 32, "LogEntry::from_compact: buffer too small for hash");
        let hash = B256::from_slice(&buf[..32]);
        buf.advance(32);

        let executing_message = if has_msg {
            let (msg, rest) = ExecutingMessageEntry::from_compact(buf, buf.len());
            buf = rest;
            Some(msg)
        } else {
            None
        };

        (Self { index, hash, executing_message }, buf)
    }
}

/// Conversion from [`Log`] to [`LogEntry`] used for internal storage.
///
/// Maps fields 1:1, converting `executing_message` using `Into`.
impl From<Log> for LogEntry {
    fn from(log: Log) -> Self {
        Self {
            index: log.index,
            hash: log.hash,
            executing_message: log.executing_message.map(Into::into),
        }
    }
}

/// Conversion from [`LogEntry`] to [`Log`] for external API use.
///
/// Mirrors the conversion from `Log`, enabling easy retrieval.
impl From<LogEntry> for Log {
    fn from(log: LogEntry) -> Self {
        Self {
            index: log.index,
            hash: log.hash,
            executing_message: log.executing_message.map(Into::into),
        }
    }
}

/// Represents an entry of an executing message, containing metadata
/// about the message's origin and context within the blockchain.
/// - `chain_id` (`u64`): The unique identifier of the blockchain where the message originated.
/// - `block_number` (`u64`): The block number in the blockchain where the message originated.
/// - `log_index` (`u64`): The index of the log entry within the block where the message was logged.
/// - `timestamp` (`u64`): The timestamp associated with the block where the message was recorded.
/// - `hash` (`B256`): The unique hash identifier of the message.
#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct ExecutingMessageEntry {
    /// Log index within the block.
    pub log_index: u32,
    /// ID of the chain where the message was emitted.
    pub chain_id: u64,
    /// Block number in the source chain.
    pub block_number: u64,
    /// Timestamp of the block.
    pub timestamp: u64,
    /// Hash of the message.
    pub hash: B256,
}

/// Compact encoding for [`ExecutingMessageEntry`] used in log storage.
///
/// This format ensures deterministic encoding and lexicographic ordering by
/// placing `log_index` first, which is used as the subkey in dup-sorted tables.
///
/// ## Encoding Layout (ordered):
/// - `log_index: u32` – Subkey for dup sort ordering.
/// - `chain_id: u64`
/// - `block_number: u64`
/// - `timestamp: u64`
/// - `hash: B256` – 32-byte message hash.
impl Compact for ExecutingMessageEntry {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: BufMut + AsMut<[u8]>,
    {
        let start_len = buf.remaining_mut();

        buf.put_u32(self.log_index);
        buf.put_u64(self.chain_id);
        buf.put_u64(self.block_number);
        buf.put_u64(self.timestamp);
        buf.put_slice(self.hash.as_slice());

        start_len - buf.remaining_mut()
    }

    fn from_compact(mut buf: &[u8], _len: usize) -> (Self, &[u8]) {
        let log_index = buf.get_u32();
        let chain_id = buf.get_u64();
        let block_number = buf.get_u64();
        let timestamp = buf.get_u64();

        assert!(buf.len() >= 32, "ExecutingMessageEntry::from_compact: buffer too small for hash");
        let hash = B256::from_slice(&buf[..32]);
        buf.advance(32);

        (Self { chain_id, block_number, timestamp, hash, log_index }, buf)
    }
}

/// Converts from [`ExecutingMessage`] (external API format) to [`ExecutingMessageEntry`] (storage
/// format).
///
/// Performs a direct field mapping.
impl From<ExecutingMessage> for ExecutingMessageEntry {
    fn from(msg: ExecutingMessage) -> Self {
        Self {
            chain_id: msg.chain_id,
            block_number: msg.block_number,
            log_index: msg.log_index,
            timestamp: msg.timestamp,
            hash: msg.hash,
        }
    }
}

/// Converts from [`ExecutingMessageEntry`] (storage format) to [`ExecutingMessage`] (external API
/// format).
///
/// This enables decoding values stored in a compact format for use in application logic.
impl From<ExecutingMessageEntry> for ExecutingMessage {
    fn from(msg: ExecutingMessageEntry) -> Self {
        Self {
            chain_id: msg.chain_id,
            block_number: msg.block_number,
            log_index: msg.log_index,
            timestamp: msg.timestamp,
            hash: msg.hash,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*; // Imports LogEntry, ExecutingMessageEntry
    use alloy_primitives::B256;
    use reth_codecs::Compact; // For the Compact trait methods

    // Helper to create somewhat unique B256 values for testing.
    // Assumes the "rand" feature for alloy-primitives is enabled for tests.
    fn test_b256(val: u8) -> B256 {
        let mut val_bytes = [0u8; 32];
        val_bytes[0] = val;
        let b256_from_val = B256::from(val_bytes);
        B256::random() ^ b256_from_val
    }

    #[test]
    fn test_log_entry_compact_roundtrip_with_message() {
        let original_log_entry = LogEntry {
            index: 100,
            hash: test_b256(1),
            executing_message: Some(ExecutingMessageEntry {
                chain_id: 10,
                block_number: 1001,
                log_index: 5,
                timestamp: 1234567890,
                hash: test_b256(2),
            }),
        };

        let mut buffer = Vec::new();
        let bytes_written = original_log_entry.to_compact(&mut buffer);

        assert_eq!(bytes_written, buffer.len(), "Bytes written should match buffer length");
        assert!(!buffer.is_empty(), "Buffer should not be empty after compression");

        let (deserialized_log_entry, remaining_buf) =
            LogEntry::from_compact(&buffer, bytes_written);

        assert_eq!(
            original_log_entry, deserialized_log_entry,
            "Original and deserialized log entries should be equal"
        );
        assert!(remaining_buf.is_empty(), "Remaining buffer should be empty after deserialization");
    }

    #[test]
    fn test_log_entry_compact_roundtrip_without_message() {
        let original_log_entry =
            LogEntry { index: 100, hash: test_b256(3), executing_message: None };

        let mut buffer = Vec::new();
        let bytes_written = original_log_entry.to_compact(&mut buffer);

        assert_eq!(bytes_written, buffer.len(), "Bytes written should match buffer length");
        assert!(!buffer.is_empty(), "Buffer should not be empty after compression");

        let (deserialized_log_entry, remaining_buf) =
            LogEntry::from_compact(&buffer, bytes_written);

        assert_eq!(
            original_log_entry, deserialized_log_entry,
            "Original and deserialized log entries should be equal"
        );
        assert!(remaining_buf.is_empty(), "Remaining buffer should be empty after deserialization");
    }

    #[test]
    fn test_executing_message_entry_compact_roundtrip() {
        let original_entry = ExecutingMessageEntry {
            log_index: 42,
            chain_id: 1,
            block_number: 123456,
            timestamp: 1_654_321_000,
            hash: test_b256(77),
        };

        let mut buffer = Vec::new();
        let bytes_written = original_entry.to_compact(&mut buffer);

        assert_eq!(bytes_written, buffer.len(), "Bytes written should match buffer length");
        assert!(!buffer.is_empty(), "Buffer should not be empty after serialization");

        let (decoded_entry, remaining_buf) =
            ExecutingMessageEntry::from_compact(&buffer, bytes_written);

        assert_eq!(
            original_entry, decoded_entry,
            "Original and decoded ExecutingMessageEntry should be equal"
        );
        assert!(remaining_buf.is_empty(), "Remaining buffer should be empty after decoding");
    }
}
