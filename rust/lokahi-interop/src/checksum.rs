//! Log hashes and message checksums: the space-efficient way the log store answers "does this
//! initiating message exist?".
//!
//! A log store that kept whole logs would have to keep whole logs. Instead it keeps one 32-byte
//! hash per log, and an executing message's access-list entry carries a checksum over
//! `(chain, block, log index, timestamp, log hash)`. Comparing checksums answers the existence
//! question from the stored hash alone.
//!
//! This is an encoding, not a validity rule — the consensus-level rules live in
//! [`kona_interop`]. The computation must nevertheless match the Go implementation
//! (`op-core/interop/messages`) byte for byte, or a lokahi node and an op-supernode node
//! disagree about which messages exist; [`ChecksumArgs::checksum`] is pinned against Go by a
//! golden vector.

use alloy_primitives::{Address, B256, Log, U256, keccak256};
use kona_interop::RawMessagePayload;

/// The checksum of an interop message, as it appears in a type-3 access-list entry.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord, Default)]
pub struct MessageChecksum(pub B256);

impl MessageChecksum {
    /// The leading byte every checksum carries, identifying the access-list entry type.
    pub const TYPE_BYTE: u8 = 0x03;

    /// Returns the checksum's bytes.
    pub const fn as_b256(&self) -> B256 {
        self.0
    }
}

impl From<B256> for MessageChecksum {
    fn from(value: B256) -> Self {
        Self(value)
    }
}

/// The hash of an initiating message's log: `keccak256(origin || keccak256(payload))`.
///
/// `payload` is the log's topics concatenated with its data, per the messaging spec.
pub fn log_hash(origin: Address, payload_hash: B256) -> B256 {
    let mut buf = [0u8; 52];
    buf[..20].copy_from_slice(origin.as_slice());
    buf[20..].copy_from_slice(payload_hash.as_slice());
    keccak256(buf)
}

/// Returns the [`log_hash`] of `log`, deriving the payload hash from its topics and data.
pub fn log_to_log_hash(log: &Log) -> B256 {
    let payload = RawMessagePayload::from(log);
    log_hash(log.address, keccak256(payload.as_ref()))
}

/// The inputs to a [`MessageChecksum`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ChecksumArgs {
    /// The initiating message's block number on its own chain.
    pub block_number: u64,
    /// The initiating message's log index within that block.
    pub log_index: u32,
    /// The timestamp of that block.
    pub timestamp: u64,
    /// The chain the initiating message was emitted on.
    pub chain_id: U256,
    /// The [`log_hash`] of the initiating message.
    pub log_hash: B256,
}

impl ChecksumArgs {
    /// Computes the checksum.
    ///
    /// The layout is fixed by the Go implementation and by the `CrossL2Inbox` predeploy:
    ///
    /// ```text
    /// id_packed = 12 zero bytes || block_number (BE u64) || timestamp (BE u64) || log_index (BE u32)
    /// checksum  = keccak256(keccak256(log_hash || id_packed) || chain_id (BE u256))
    /// checksum[0] = 0x03
    /// ```
    pub fn checksum(&self) -> MessageChecksum {
        let mut id_packed = [0u8; 32];
        id_packed[12..20].copy_from_slice(&self.block_number.to_be_bytes());
        id_packed[20..28].copy_from_slice(&self.timestamp.to_be_bytes());
        id_packed[28..32].copy_from_slice(&self.log_index.to_be_bytes());

        let mut id_log_hash_input = [0u8; 64];
        id_log_hash_input[..32].copy_from_slice(self.log_hash.as_slice());
        id_log_hash_input[32..].copy_from_slice(&id_packed);
        let id_log_hash = keccak256(id_log_hash_input);

        let mut outer = [0u8; 64];
        outer[..32].copy_from_slice(id_log_hash.as_slice());
        outer[32..].copy_from_slice(&self.chain_id.to_be_bytes::<32>());
        let mut out = keccak256(outer);
        out[0] = MessageChecksum::TYPE_BYTE;
        MessageChecksum(out)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{LogData, address, b256, bytes};

    #[test]
    fn checksum_matches_go_golden_vectors() {
        // Golden vectors pinning `ChecksumArgs::checksum` against Go's
        // `messages.ChecksumArgs.Checksum()`. Regenerate by printing
        // `ChecksumArgs{...}.Checksum()` for the same inputs from a test in
        // `op-core/interop/messages`.
        let cases = [
            (
                ChecksumArgs {
                    block_number: 1,
                    log_index: 2,
                    timestamp: 3,
                    chain_id: U256::from(4),
                    log_hash: B256::repeat_byte(5),
                },
                b256!("03a0b2c62b5ccbd0696110563eca09f4871f14e81e372a76a6851a6bc6c7d3c8"),
            ),
            (
                ChecksumArgs {
                    block_number: 0xdead_beef,
                    log_index: 7,
                    timestamp: 0x0012_3456_7890,
                    chain_id: U256::from(901),
                    log_hash: B256::repeat_byte(0xaa),
                },
                b256!("037850369207ffd34a0b5fb486dfb562fce0f328c1211855f328124fd8f9f36c"),
            ),
        ];
        for (args, expected) in cases {
            assert_eq!(args.checksum().as_b256(), expected);
        }
    }

    #[test]
    fn log_hash_matches_go_golden_vector() {
        // Go: `messages.PayloadHashToLogHash(0x11..11, 0x4200..23)`.
        assert_eq!(
            log_hash(address!("4200000000000000000000000000000000000023"), B256::repeat_byte(0x11)),
            b256!("bf556a7286807fba9667ab92fa18118440066530c7c2521d649e91c078bc3891")
        );
    }

    #[test]
    fn checksum_leads_with_the_type_byte() {
        let args = ChecksumArgs {
            block_number: 9,
            log_index: 0,
            timestamp: 100,
            chain_id: U256::from(901),
            log_hash: B256::repeat_byte(0xaa),
        };
        assert_eq!(args.checksum().as_b256()[0], MessageChecksum::TYPE_BYTE);
    }

    #[test]
    fn checksum_is_sensitive_to_every_field() {
        let base = ChecksumArgs {
            block_number: 9,
            log_index: 1,
            timestamp: 100,
            chain_id: U256::from(901),
            log_hash: B256::repeat_byte(0xaa),
        };
        let mutations = [
            ChecksumArgs { block_number: 10, ..base },
            ChecksumArgs { log_index: 2, ..base },
            ChecksumArgs { timestamp: 101, ..base },
            ChecksumArgs { chain_id: U256::from(902), ..base },
            ChecksumArgs { log_hash: B256::repeat_byte(0xab), ..base },
        ];
        for mutated in mutations {
            assert_ne!(base.checksum(), mutated.checksum());
        }
    }

    #[test]
    fn log_hash_covers_topics_then_data() {
        let log = Log {
            address: address!("4200000000000000000000000000000000000023"),
            data: LogData::new_unchecked(vec![B256::repeat_byte(1)], bytes!("beef")),
        };
        let expected = {
            let mut payload = Vec::new();
            payload.extend_from_slice(B256::repeat_byte(1).as_slice());
            payload.extend_from_slice(&[0xbe, 0xef]);
            log_hash(log.address, keccak256(payload))
        };
        assert_eq!(log_to_log_hash(&log), expected);
    }
}
