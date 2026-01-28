use alloy_primitives::{B256, keccak256};
use thiserror::Error;

/// A structured representation of a parsed CrossL2Inbox message access entry.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Access {
    /// Full 256-bit chain ID (combined from lookup + extension)
    pub chain_id: [u8; 32],
    /// Block number in the source chain
    pub block_number: u64,
    /// Timestamp of the message's block
    pub timestamp: u64,
    /// Log index of the message within the block
    pub log_index: u32,
    /// Provided checksum entry (prefix 0x03)
    pub checksum: B256,
}

impl Access {
    /// Constructs a new [`Access`] from a `LookupEntry`, optional `ChainIdExtensionEntry`,
    /// and a `ChecksumEntry`. Used internally by the parser.
    fn from_entries(
        lookup: LookupEntry,
        chain_id_ext: Option<ChainIdExtensionEntry>,
        checksum: ChecksumEntry,
    ) -> Self {
        let mut chain_id = [0u8; 32];

        if let Some(ext) = chain_id_ext {
            chain_id[0..24].copy_from_slice(&ext.upper_bytes);
        }

        chain_id[24..32].copy_from_slice(&lookup.chain_id_low);

        Self {
            chain_id,
            block_number: lookup.block_number,
            timestamp: lookup.timestamp,
            log_index: lookup.log_index,
            checksum: checksum.raw,
        }
    }

    /// Recomputes the checksum for this access entry.
    ///
    /// This follows the spec:
    /// - `idPacked = 12 zero bytes ++ block_number ++ timestamp ++ log_index`
    /// - `idLogHash = keccak256(log_hash ++ idPacked)`
    /// - `bareChecksum = keccak256(idLogHash ++ chain_id)`
    /// - Prepend 0x03 to `bareChecksum[1..]`
    ///
    /// Returns the full 32-byte checksum with prefix 0x03.
    ///
    /// Reference: [Checksum Calculation](https://github.com/ethereum-optimism/specs/blob/main/specs/interop/predeploys.md#type-3-checksum)
    pub fn recompute_checksum(&self, log_hash: &B256) -> B256 {
        // Step 1: idPacked = [0u8; 12] ++ block_number ++ timestamp ++ log_index
        let mut id_packed = [0u8; 12 + 8 + 8 + 4]; // 32 bytes
        id_packed[12..20].copy_from_slice(&self.block_number.to_be_bytes());
        id_packed[20..28].copy_from_slice(&self.timestamp.to_be_bytes());
        id_packed[28..32].copy_from_slice(&self.log_index.to_be_bytes());

        // Step 2: keccak256(log_hash ++ id_packed)
        let id_log_hash = keccak256([log_hash.as_slice(), &id_packed].concat());

        // Step 3: keccak256(id_log_hash ++ chain_id)
        let bare_checksum = keccak256([id_log_hash.as_slice(), &self.chain_id].concat());

        // Step 4: Prepend type byte 0x03 (overwrite first byte)
        let mut checksum = bare_checksum;
        checksum.0[0] = 0x03;

        checksum
    }

    /// Verify the checksums after recalculation
    pub fn verify_checksum(&self, log_hash: &B256) -> Result<(), AccessListError> {
        if self.recompute_checksum(log_hash) != self.checksum {
            return Err(AccessListError::MalformedEntry);
        }
        Ok(())
    }
}

/// Represents a single entry in the access list.
#[derive(Debug, Clone)]
enum AccessListEntry {
    Lookup(LookupEntry),
    ChainIdExtension(ChainIdExtensionEntry),
    Checksum(ChecksumEntry),
}

/// Parsed lookup identity entry (type 0x01).
#[derive(Debug, Clone)]
struct LookupEntry {
    pub chain_id_low: [u8; 8],
    pub block_number: u64,
    pub timestamp: u64,
    pub log_index: u32,
}

/// Parsed Chain ID extension entry (type 0x02).
#[derive(Debug, Clone)]
struct ChainIdExtensionEntry {
    pub upper_bytes: [u8; 24],
}

/// Parsed checksum entry (type 0x03).
#[derive(Debug, Clone)]
struct ChecksumEntry {
    pub raw: B256,
}

/// Error returned when access list parsing fails.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum AccessListError {
    /// Input ended before a complete message group was parsed.
    #[error("unexpected end of access list")]
    UnexpectedEnd,

    /// Unexpected entry type found.
    #[error("expected type {expected:#x}, got {found:#x}")]
    UnexpectedType {
        /// The type we expected (e.g. 0x01, 0x02, or 0x03)
        expected: u8,
        /// The actual type byte we found
        found: u8,
    },

    /// Malformed entry sequence or invalid prefix structure.
    #[error("malformed entry")]
    MalformedEntry,

    /// Message expired.
    #[error("message expired")]
    MessageExpired,

    /// Timestamp invariant violated.
    #[error("executing timestamp is earlier than initiating timestamp")]
    InvalidTimestampInvariant,
}

// Access list entry type byte constants
const PREFIX_LOOKUP: u8 = 0x01;
const PREFIX_CHAIN_ID_EXTENSION: u8 = 0x02;
const PREFIX_CHECKSUM: u8 = 0x03;

/// Parses a vector of raw `B256` access list entries into structured [`Access`] objects.
///
/// Each `Access` group must follow the pattern:
/// - One `Lookup` entry (prefix `0x01`)
/// - Optionally one `ChainIdExtension` entry (prefix `0x02`)
/// - One `Checksum` entry (prefix `0x03`)
///
/// Entries are consumed in order. If any group is malformed, this function returns a
/// [`AccessListError`].
///
/// # Arguments
///
/// * `entries` - A `Vec<B256>` representing the raw access list entries.
///
/// # Returns
///
/// A vector of fully parsed [`Access`] items if all entries are valid.
///
/// # Errors
///
/// Returns [`AccessListError`] if entries are out-of-order, malformed, or incomplete.
pub fn parse_access_list(entries: Vec<B256>) -> Result<Vec<Access>, AccessListError> {
    let mut list = Vec::with_capacity(entries.len() / 2);
    let mut lookup_entry: Option<LookupEntry> = None;
    let mut chain_id_ext: Option<ChainIdExtensionEntry> = None;

    for entry in entries {
        let parsed = parse_entry(&entry)?;

        match parsed {
            AccessListEntry::Lookup(lookup) => {
                if lookup_entry.is_some() {
                    return Err(AccessListError::MalformedEntry);
                }
                lookup_entry = Some(lookup);
            }

            AccessListEntry::ChainIdExtension(ext) => {
                if lookup_entry.is_none() || chain_id_ext.is_some() {
                    return Err(AccessListError::MalformedEntry);
                }
                chain_id_ext = Some(ext);
            }

            AccessListEntry::Checksum(checksum) => {
                let lookup = lookup_entry.take().ok_or(AccessListError::MalformedEntry)?;
                let access = Access::from_entries(lookup, chain_id_ext.take(), checksum);
                list.push(access);
            }
        }
    }

    if lookup_entry.is_some() {
        return Err(AccessListError::UnexpectedEnd);
    }

    Ok(list)
}

/// Parses a single 32-byte access list entry into a typed [`AccessListEntry`].
///
/// This function performs a prefix-based decoding of the input hash:
///
/// ### Entry Type Encoding
///
/// | Prefix Byte | Type                   | Description                                                       |
/// |-------------|------------------------|-------------------------------------------------------------------|
/// | `0x01`      | `LookupEntry`          | Contains chain ID (low bits), block number, timestamp, log index. |
/// | `0x02`      | `ChainIdExtensionEntry`| Contains upper 24 bytes of a 256-bit chain ID.                    |
/// | `0x03`      | `ChecksumEntry`        | Contains the checksum hash used for message validation.           |
///
/// ### Spec References
///
/// - [Optimism Access List Format](https://github.com/ethereum-optimism/specs/blob/main/specs/interop/predeploys.md#access-list)
/// - Entry format and layout based on CrossL2Inbox access-list encoding.
fn parse_entry(entry: &B256) -> Result<AccessListEntry, AccessListError> {
    match entry[0] {
        PREFIX_LOOKUP => {
            if entry[1..4] != [0; 3] {
                return Err(AccessListError::MalformedEntry);
            }
            Ok(AccessListEntry::Lookup(LookupEntry {
                chain_id_low: entry[4..12].try_into().unwrap(),
                block_number: u64::from_be_bytes(entry[12..20].try_into().unwrap()),
                timestamp: u64::from_be_bytes(entry[20..28].try_into().unwrap()),
                log_index: u32::from_be_bytes(entry[28..32].try_into().unwrap()),
            }))
        }

        PREFIX_CHAIN_ID_EXTENSION => {
            if entry[1..8] != [0; 7] {
                return Err(AccessListError::MalformedEntry);
            }
            Ok(AccessListEntry::ChainIdExtension(ChainIdExtensionEntry {
                upper_bytes: entry[8..32].try_into().unwrap(),
            }))
        }

        PREFIX_CHECKSUM => Ok(AccessListEntry::Checksum(ChecksumEntry { raw: *entry })),

        other => Err(AccessListError::UnexpectedType { expected: PREFIX_LOOKUP, found: other }),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{B256, U256, b256};

    fn make_lookup_entry(
        block_number: u64,
        timestamp: u64,
        log_index: u32,
        chain_id_low: [u8; 8],
    ) -> B256 {
        let mut buf = [0u8; 32];
        buf[0] = PREFIX_LOOKUP;
        // 3 zero padding
        buf[4..12].copy_from_slice(&chain_id_low);
        buf[12..20].copy_from_slice(&block_number.to_be_bytes());
        buf[20..28].copy_from_slice(&timestamp.to_be_bytes());
        buf[28..32].copy_from_slice(&log_index.to_be_bytes());
        B256::from(buf)
    }

    fn make_chain_id_ext(upper: [u8; 24]) -> B256 {
        let mut buf = [0u8; 32];
        buf[0] = PREFIX_CHAIN_ID_EXTENSION;
        // 7 zero padding
        buf[8..32].copy_from_slice(&upper);
        B256::from(buf)
    }

    fn make_checksum(access: &Access, log_hash: &B256) -> B256 {
        access.recompute_checksum(log_hash)
    }

    #[test]
    fn test_parse_valid_access_list_with_chain_id_ext() {
        let block_number = 1234;
        let timestamp = 9999;
        let log_index = 5;
        let chain_id_low = [1u8; 8];
        let upper_bytes = [2u8; 24];
        let log_hash = keccak256([0u8; 32]);

        let lookup = make_lookup_entry(block_number, timestamp, log_index, chain_id_low);
        let chain_ext = make_chain_id_ext(upper_bytes);

        let access = Access::from_entries(
            LookupEntry { chain_id_low, block_number, timestamp, log_index },
            Some(ChainIdExtensionEntry { upper_bytes }),
            ChecksumEntry {
                raw: B256::default(), // will override later
            },
        );

        let checksum = make_checksum(&access, &log_hash);

        let access = Access::from_entries(
            LookupEntry { chain_id_low, block_number, timestamp, log_index },
            Some(ChainIdExtensionEntry { upper_bytes }),
            ChecksumEntry { raw: checksum },
        );

        let list = vec![lookup, chain_ext, checksum];
        let parsed = parse_access_list(list).unwrap();
        assert_eq!(parsed.len(), 1);
        assert_eq!(parsed[0], access);
        assert!(parsed[0].verify_checksum(&log_hash).is_ok());
    }

    #[test]
    fn test_parse_access_list_without_chain_id_ext() {
        let block_number = 1;
        let timestamp = 2;
        let log_index = 3;
        let chain_id_low = [0xaa; 8];
        let log_hash = keccak256([1u8; 32]);

        let lookup = make_lookup_entry(block_number, timestamp, log_index, chain_id_low);
        let access = Access::from_entries(
            LookupEntry { chain_id_low, block_number, timestamp, log_index },
            None,
            ChecksumEntry { raw: B256::default() },
        );
        let checksum = make_checksum(&access, &log_hash);
        let access = Access::from_entries(
            LookupEntry { chain_id_low, block_number, timestamp, log_index },
            None,
            ChecksumEntry { raw: checksum },
        );

        let list = vec![lookup, checksum];
        let parsed = parse_access_list(list).unwrap();
        assert_eq!(parsed.len(), 1);
        assert_eq!(parsed[0], access);
        assert!(parsed[0].verify_checksum(&log_hash).is_ok());
    }

    #[test]
    fn test_recompute_checksum_against_known_value() {
        // Input data
        let access = Access {
            chain_id: U256::from(3).to_be_bytes(),
            block_number: 2587,
            timestamp: 4660,
            log_index: 66,
            checksum: B256::default(), // not used in this test
        };

        let log_hash = b256!("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef");

        // Expected checksum computed previously using spec logic
        let expected = b256!("0x03ca886771056d8ea647bb809b888ba14986f57daaf28954d40408321717716a");

        let computed = access.recompute_checksum(&log_hash);
        assert_eq!(computed, expected, "Checksum does not match expected value");
    }

    #[test]
    fn test_checksum_mismatch() {
        let block_number = 1;
        let timestamp = 2;
        let log_index = 3;
        let chain_id_low = [0xaa; 8];
        let log_hash = keccak256([1u8; 32]);

        let lookup = make_lookup_entry(block_number, timestamp, log_index, chain_id_low);
        let fake_checksum =
            b256!("0x03ca886771056d8ea647bb809b888ba14986f57daaf28954d40408321717716a");
        let list = vec![lookup, fake_checksum];

        let parsed = parse_access_list(list).unwrap();
        let err = parsed[0].verify_checksum(&log_hash);
        assert_eq!(err, Err(AccessListError::MalformedEntry));
    }

    #[test]
    fn test_invalid_entry_order_should_fail() {
        let mut raw = [0u8; 32];
        raw[0] = PREFIX_CHECKSUM;
        let checksum = B256::from(raw);

        let lookup = make_lookup_entry(0, 0, 0, [0u8; 8]);
        let entries = vec![checksum, lookup];

        assert!(matches!(parse_access_list(entries), Err(AccessListError::MalformedEntry)));
    }
}
