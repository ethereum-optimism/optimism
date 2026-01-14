//! Contains the [PreimageKey] type, which is used to identify preimages that may be fetched from
//! the preimage oracle.

use alloy_primitives::{B256, Keccak256, U256};
#[cfg(feature = "rkyv")]
use rkyv::{Archive, Deserialize as RkyvDeserialize, Serialize as RkyvSerialize};
#[cfg(feature = "serde")]
use serde::{Deserialize as SerdeDeserialize, Serialize as SerdeSerialize};

use crate::errors::PreimageOracleError;

/// <https://specs.optimism.io/experimental/fault-proof/index.html#pre-image-key-types>
#[derive(Debug, Default, Clone, Copy, Eq, PartialEq, Ord, PartialOrd, Hash)]
#[repr(u8)]
#[cfg_attr(
    feature = "rkyv",
    derive(Archive, RkyvSerialize, RkyvDeserialize),
    rkyv(derive(Eq, PartialEq, Ord, PartialOrd, Hash))
)]
#[cfg_attr(feature = "serde", derive(SerdeSerialize, SerdeDeserialize))]
pub enum PreimageKeyType {
    /// Local key types are local to a given instance of a fault-proof and context dependent.
    /// Commonly these local keys are mapped to bootstrap data for the fault proof program.
    Local = 1,
    /// Keccak256 key types are global and context independent. Preimages are mapped from the
    /// low-order 31 bytes of the preimage's `keccak256` digest to the preimage itself.
    #[default]
    Keccak256 = 2,
    /// GlobalGeneric key types are reserved for future use.
    GlobalGeneric = 3,
    /// Sha256 key types are global and context independent. Preimages are mapped from the
    /// low-order 31 bytes of the preimage's `sha256` digest to the preimage itself.
    Sha256 = 4,
    /// Blob key types are global and context independent. Blob keys are constructed as
    /// `keccak256(commitment ++ z)`, and then the high-order byte of the digest is set to the
    /// type byte.
    Blob = 5,
    /// Precompile key types are global and context independent. Precompile keys are constructed as
    /// `keccak256(precompile_addr ++ input)`, and then the high-order byte of the digest is set to
    /// the type byte.
    Precompile = 6,
}

impl TryFrom<u8> for PreimageKeyType {
    type Error = PreimageOracleError;

    fn try_from(value: u8) -> Result<Self, Self::Error> {
        let key_type = match value {
            1 => Self::Local,
            2 => Self::Keccak256,
            3 => Self::GlobalGeneric,
            4 => Self::Sha256,
            5 => Self::Blob,
            6 => Self::Precompile,
            _ => return Err(PreimageOracleError::InvalidPreimageKey),
        };
        Ok(key_type)
    }
}

/// A preimage key is a 32-byte value that identifies a preimage that may be fetched from the
/// oracle.
///
/// **Layout**:
/// |  Bits   | Description |
/// |---------|-------------|
/// | [0, 1)  | Type byte   |
/// | [1, 32) | Data        |
#[derive(Debug, Default, Clone, Copy, Eq, PartialEq, Ord, PartialOrd, Hash)]
#[cfg_attr(
    feature = "rkyv",
    derive(Archive, RkyvSerialize, RkyvDeserialize),
    rkyv(derive(Eq, PartialEq, Ord, PartialOrd, Hash))
)]
#[cfg_attr(feature = "serde", derive(SerdeSerialize, SerdeDeserialize))]
pub struct PreimageKey {
    data: [u8; 31],
    key_type: PreimageKeyType,
}

impl PreimageKey {
    /// Creates a new [PreimageKey] from a 32-byte value and a [PreimageKeyType]. The 32-byte value
    /// will be truncated to 31 bytes by taking the low-order 31 bytes.
    pub fn new(key: [u8; 32], key_type: PreimageKeyType) -> Self {
        let mut data = [0u8; 31];
        data.copy_from_slice(&key[1..]);
        Self { data, key_type }
    }

    /// Creates a new local [PreimageKey] from a 64-bit local identifier. The local identifier will
    /// be written into the low-order 8 bytes of the big-endian 31-byte data field.
    pub fn new_local(local_ident: u64) -> Self {
        let mut data = [0u8; 31];
        data[23..].copy_from_slice(&local_ident.to_be_bytes());
        Self { data, key_type: PreimageKeyType::Local }
    }

    /// Creates a new keccak256 [PreimageKey] from a 32-byte keccak256 digest. The digest will be
    /// truncated to 31 bytes by taking the low-order 31 bytes.
    pub fn new_keccak256(digest: [u8; 32]) -> Self {
        Self::new(digest, PreimageKeyType::Keccak256)
    }

    /// Creates a new precompile [PreimageKey] from a precompile address and input. The key will be
    /// constructed as `keccak256(precompile_addr ++ input)`, and then the high-order byte of the
    /// digest will be set to the type byte.
    pub fn new_precompile(precompile_addr: [u8; 20], input: &[u8]) -> Self {
        let mut data = [0u8; 31];

        let mut hasher = Keccak256::new();
        hasher.update(precompile_addr);
        hasher.update(input);

        data.copy_from_slice(&hasher.finalize()[1..]);
        Self { data, key_type: PreimageKeyType::Precompile }
    }

    /// Returns the [PreimageKeyType] for the [PreimageKey].
    pub const fn key_type(&self) -> PreimageKeyType {
        self.key_type
    }

    /// Returns the value of the [PreimageKey] as a [U256].
    pub const fn key_value(&self) -> U256 {
        U256::from_be_slice(self.data.as_slice())
    }
}

impl From<PreimageKey> for [u8; 32] {
    fn from(key: PreimageKey) -> Self {
        let mut rendered_key = [0u8; 32];
        rendered_key[0] = key.key_type as u8;
        rendered_key[1..].copy_from_slice(&key.data);
        rendered_key
    }
}

impl From<PreimageKey> for B256 {
    fn from(value: PreimageKey) -> Self {
        let raw: [u8; 32] = value.into();
        Self::from(raw)
    }
}

impl TryFrom<[u8; 32]> for PreimageKey {
    type Error = PreimageOracleError;

    fn try_from(value: [u8; 32]) -> Result<Self, Self::Error> {
        let key_type = PreimageKeyType::try_from(value[0])?;
        Ok(Self::new(value, key_type))
    }
}

impl core::fmt::Display for PreimageKey {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        let raw: [u8; 32] = (*self).into();
        write!(f, "{}", B256::from(raw))
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn test_preimage_key_from_u8() {
        assert_eq!(PreimageKeyType::try_from(1).unwrap(), PreimageKeyType::Local);
        assert_eq!(PreimageKeyType::try_from(2).unwrap(), PreimageKeyType::Keccak256);
        assert_eq!(PreimageKeyType::try_from(3).unwrap(), PreimageKeyType::GlobalGeneric);
        assert_eq!(PreimageKeyType::try_from(4).unwrap(), PreimageKeyType::Sha256);
        assert_eq!(PreimageKeyType::try_from(5).unwrap(), PreimageKeyType::Blob);
        assert_eq!(PreimageKeyType::try_from(6).unwrap(), PreimageKeyType::Precompile);
        assert!(PreimageKeyType::try_from(0).is_err());
        assert!(PreimageKeyType::try_from(7).is_err());
    }

    #[test]
    fn test_preimage_key_new_local() {
        let key = PreimageKey::new_local(0xFFu64);
        assert_eq!(key.key_type(), PreimageKeyType::Local);
        assert_eq!(key.key_value(), U256::from(0xFFu64));
    }

    #[test]
    fn test_preimage_key_value() {
        let key = PreimageKey::new([0xFFu8; 32], PreimageKeyType::Local);
        assert_eq!(
            key.key_value(),
            alloy_primitives::uint!(
                452312848583266388373324160190187140051835877600158453279131187530910662655_U256
            )
        );
    }

    #[test]
    fn test_preimage_key_roundtrip_b256() {
        let key = PreimageKey::new([0xFFu8; 32], PreimageKeyType::Local);
        let b256: B256 = key.into();
        let key2 = PreimageKey::try_from(<[u8; 32]>::from(b256)).unwrap();
        assert_eq!(key, key2);
    }

    #[test]
    fn test_preimage_key_display() {
        let key = PreimageKey::new([0xFFu8; 32], PreimageKeyType::Local);
        assert_eq!(
            key.to_string(),
            "0x01ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
        );
    }

    #[test]
    fn test_preimage_keys() {
        let types = [
            PreimageKeyType::Local,
            PreimageKeyType::Keccak256,
            PreimageKeyType::GlobalGeneric,
            PreimageKeyType::Sha256,
            PreimageKeyType::Blob,
            PreimageKeyType::Precompile,
        ];

        for key_type in types {
            let key = PreimageKey::new([0xFFu8; 32], key_type);
            assert_eq!(key.key_type(), key_type);

            let mut rendered_key = [0xFFu8; 32];
            rendered_key[0] = key_type as u8;
            let actual: [u8; 32] = key.into();
            assert_eq!(actual, rendered_key);
        }
    }
}
