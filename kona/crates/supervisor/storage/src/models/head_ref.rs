use derive_more::TryFrom;
use op_alloy_consensus::interop::SafetyLevel;
use reth_db::DatabaseError;
use reth_db_api::table;
use serde::{Deserialize, Serialize};

/// Key representing a particular head reference type.
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord, Serialize, Deserialize, TryFrom,
)]
#[try_from(repr)]
#[repr(u8)]
pub enum SafetyHeadRefKey {
    /// Latest unverified or unsafe head.
    Unsafe = 0,

    /// Head block considered safe via local verification.
    LocalSafe = 1,

    /// Head block considered unsafe via cross-chain sync.
    CrossUnsafe = 2,

    /// Head block considered safe.
    Safe = 3,

    /// Finalized head block.
    Finalized = 4,

    /// Invalid head reference.
    Invalid = u8::MAX,
}

/// Implementation of [`table::Encode`] for [`SafetyHeadRefKey`].
impl table::Encode for SafetyHeadRefKey {
    type Encoded = [u8; 1];

    fn encode(self) -> Self::Encoded {
        [self as u8]
    }
}

/// Implementation of [`table::Decode`] for [`SafetyHeadRefKey`].
impl table::Decode for SafetyHeadRefKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.is_empty() {
            return Err(DatabaseError::Decode)
        }

        value[0].try_into().map_err(|_| DatabaseError::Decode)
    }
}

/// Converts from [`SafetyHeadRefKey`] (internal storage reference) to [`SafetyLevel`] (public API
/// format).
///
/// Performs a lossless and direct mapping from head reference level to safety level.
impl From<SafetyHeadRefKey> for SafetyLevel {
    fn from(key: SafetyHeadRefKey) -> Self {
        match key {
            SafetyHeadRefKey::Unsafe => Self::LocalUnsafe,
            SafetyHeadRefKey::LocalSafe => Self::LocalSafe,
            SafetyHeadRefKey::CrossUnsafe => Self::CrossUnsafe,
            SafetyHeadRefKey::Safe => Self::CrossSafe,
            SafetyHeadRefKey::Finalized => Self::Finalized,
            SafetyHeadRefKey::Invalid => Self::Invalid,
        }
    }
}

/// Converts from [`SafetyLevel`] (public API format) to [`SafetyHeadRefKey`] (internal storage
/// reference).
///
/// Performs a direct  mapping from safety level to head reference key.
impl From<SafetyLevel> for SafetyHeadRefKey {
    fn from(key: SafetyLevel) -> Self {
        match key {
            SafetyLevel::LocalUnsafe => Self::Unsafe,
            SafetyLevel::LocalSafe => Self::LocalSafe,
            SafetyLevel::CrossUnsafe => Self::CrossUnsafe,
            SafetyLevel::CrossSafe => Self::Safe,
            SafetyLevel::Finalized => Self::Finalized,
            SafetyLevel::Invalid => Self::Invalid,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_db_api::table::{Decode, Encode};
    #[test]
    fn test_head_ref_key_encode_decode() {
        let cases = vec![
            (SafetyHeadRefKey::Unsafe, [0]),
            (SafetyHeadRefKey::LocalSafe, [1]),
            (SafetyHeadRefKey::CrossUnsafe, [2]),
            (SafetyHeadRefKey::Safe, [3]),
            (SafetyHeadRefKey::Finalized, [4]),
            (SafetyHeadRefKey::Invalid, [255]),
        ];

        for (key, expected_encoding) in &cases {
            // Test encoding
            let encoded = key.encode();
            assert_eq!(encoded, *expected_encoding, "Encoding failed for {key:?}");

            // Test decoding
            let decoded = SafetyHeadRefKey::decode(&encoded).expect("Decoding should succeed");
            assert_eq!(decoded, *key, "Decoding mismatch for {key:?}");
        }
    }
    #[test]
    fn test_round_trip_conversion() {
        for level in [
            SafetyLevel::LocalUnsafe,
            SafetyLevel::LocalSafe,
            SafetyLevel::CrossUnsafe,
            SafetyLevel::CrossSafe,
            SafetyLevel::Finalized,
            SafetyLevel::Invalid,
        ] {
            let round_trip = SafetyLevel::from(SafetyHeadRefKey::from(level));
            assert_eq!(round_trip, level, "Round-trip failed for {level:?}");
        }

        for key in [
            SafetyHeadRefKey::Unsafe,
            SafetyHeadRefKey::LocalSafe,
            SafetyHeadRefKey::CrossUnsafe,
            SafetyHeadRefKey::Safe,
            SafetyHeadRefKey::Finalized,
            SafetyHeadRefKey::Invalid,
        ] {
            let round_trip = SafetyHeadRefKey::from(SafetyLevel::from(key));
            assert_eq!(round_trip, key, "Round-trip failed for {key:?}");
        }
    }
}
