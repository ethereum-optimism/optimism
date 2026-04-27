//! Superchain types

use alloc::{
    format,
    string::{String, ToString},
};
use core::array::TryFromSliceError;

use alloy_primitives::{B64, B256};
use derive_more::derive::{Display, From};

/// Superchain Signal information.
///
/// The execution engine SHOULD warn the user when the recommended version is newer than the current
/// version supported by the execution engine.
///
/// The execution engine SHOULD take safety precautions if it does not meet the required protocol
/// version. This may include halting the engine, with consent of the execution engine operator.
///
/// See also: <https://specs.optimism.io/protocol/exec-engine.html#engine_signalsuperchainv1>
#[derive(Copy, Clone, Debug, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct SuperchainSignal {
    /// The recommended Superchain Protocol Version.
    pub recommended: ProtocolVersion,
    /// The minimum Superchain Protocol Version required.
    pub required: ProtocolVersion,
}

/// Formatted Superchain Protocol Version.
///
/// The Protocol Version documents the progression of the total set of canonical OP-Stack
/// specifications. Components of the OP-Stack implement the subset of their respective protocol
/// component domain, up to a given Protocol Version of the OP-Stack.
///
/// The Protocol Version **is NOT a hardfork identifier**, but rather indicates software-support for
/// a well-defined set of features introduced in past and future hardforks, not the activation of
/// said hardforks.
///
/// The Protocol Version is Semver-compatible. It is encoded as a single 32 bytes long
/// protocol version. The version must be encoded as 32 bytes of DATA in JSON RPC usage.
///
/// See also: <https://specs.optimism.io/protocol/superchain-upgrades.html#protocol-version>
#[derive(Copy, Clone, Debug, PartialEq, Eq)]
#[non_exhaustive]
pub enum ProtocolVersion {
    /// Version-type 0.
    V0(ProtocolVersionFormatV0),
}

impl core::fmt::Display for ProtocolVersion {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::V0(value) => write!(f, "{value}"),
        }
    }
}

/// An error that can occur when encoding or decoding a `ProtocolVersion`.
#[derive(Copy, Clone, thiserror::Error, Debug, Display, From)]
pub enum ProtocolVersionError {
    /// An unsupported version was encountered.
    #[display("Unsupported version: {_0}")]
    UnsupportedVersion(u8),
    /// An invalid length was encountered.
    #[display("Invalid length: got {}, expected {}", got, expected)]
    InvalidLength {
        /// The length that was encountered.
        got: usize,
        /// The expected length.
        expected: usize,
    },
    /// Failed to convert slice to array.
    #[display("Failed to convert slice to array")]
    #[from(TryFromSliceError)]
    TryFromSlice,
}

impl ProtocolVersion {
    /// Version-type 0 byte encoding:
    ///
    /// ```text
    /// <protocol version> ::= <version-type><typed-payload>
    /// <version-type> ::= <uint8>
    /// <typed-payload> ::= <31 bytes>
    /// ```
    pub fn encode(&self) -> B256 {
        let mut bytes = [0u8; 32];

        match self {
            Self::V0(value) => {
                bytes[0] = 0x00; // this is not necessary, but added for clarity
                bytes[1..].copy_from_slice(&value.encode());
                B256::from_slice(&bytes)
            }
        }
    }

    /// Version-type 0 byte decoding:
    ///
    /// ```text
    /// <protocol version> ::= <version-type><typed-payload>
    /// <version-type> ::= <uint8>
    /// <typed-payload> ::= <31 bytes>
    /// ```
    pub fn decode(value: B256) -> Result<Self, ProtocolVersionError> {
        let version_type = value[0];
        let typed_payload = &value[1..];

        match version_type {
            0 => Ok(Self::V0(ProtocolVersionFormatV0::decode(typed_payload)?)),
            other => Err(ProtocolVersionError::UnsupportedVersion(other)),
        }
    }

    /// Returns the inner value of the `ProtocolVersion` enum
    pub const fn inner(&self) -> ProtocolVersionFormatV0 {
        match self {
            Self::V0(value) => *value,
        }
    }

    /// Returns the inner value of the `ProtocolVersion` enum if it is V0, otherwise None
    pub const fn as_v0(&self) -> Option<ProtocolVersionFormatV0> {
        match self {
            Self::V0(value) => Some(*value),
        }
    }

    /// Differentiates forks and custom-builds of standard protocol
    pub const fn build(&self) -> B64 {
        match self {
            Self::V0(value) => value.build,
        }
    }

    /// Incompatible API changes
    pub const fn major(&self) -> u32 {
        match self {
            Self::V0(value) => value.major,
        }
    }

    /// Identifies additional functionality in backwards compatible manner
    pub const fn minor(&self) -> u32 {
        match self {
            Self::V0(value) => value.minor,
        }
    }

    /// Identifies backward-compatible bug-fixes
    pub const fn patch(&self) -> u32 {
        match self {
            Self::V0(value) => value.patch,
        }
    }

    /// Identifies unstable versions that may not satisfy the above
    pub const fn pre_release(&self) -> u32 {
        match self {
            Self::V0(value) => value.pre_release,
        }
    }

    /// Returns a human-readable string representation of the `ProtocolVersion`
    pub fn display(&self) -> String {
        match self {
            Self::V0(value) => format!("{value}"),
        }
    }
}

#[cfg(feature = "serde")]
impl serde::Serialize for ProtocolVersion {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        self.encode().serialize(serializer)
    }
}

#[cfg(feature = "serde")]
impl<'de> serde::Deserialize<'de> for ProtocolVersion {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let value = alloy_primitives::B256::deserialize(deserializer)?;
        Self::decode(value).map_err(serde::de::Error::custom)
    }
}

/// The Protocol Version V0 format.
/// Encoded as 31 bytes with the following structure:
///
/// ```text
/// <reserved><build><major><minor><patch><pre-release>
/// <reserved> ::= <7 zeroed bytes>
/// <build> ::= <8 bytes>
/// <major> ::= <big-endian uint32>
/// <minor> ::= <big-endian uint32>
/// <patch> ::= <big-endian uint32>
/// <pre-release> ::= <big-endian uint32>
/// ```
#[derive(Copy, Clone, Default, Debug, PartialEq, Eq)]
pub struct ProtocolVersionFormatV0 {
    /// Differentiates forks and custom-builds of standard protocol
    pub build: B64,
    /// Incompatible API changes
    pub major: u32,
    /// Identifies additional functionality in backwards compatible manner
    pub minor: u32,
    /// Identifies backward-compatible bug-fixes
    pub patch: u32,
    /// Identifies unstable versions that may not satisfy the above
    pub pre_release: u32,
}

impl core::fmt::Display for ProtocolVersionFormatV0 {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        let build_tag = if self.build.0.iter().any(|&byte| byte != 0) {
            if self.is_readable_build_tag() {
                let full = format!("+{}", String::from_utf8_lossy(&self.build.0));
                full.trim_end_matches('\0').to_string()
            } else {
                format!("+{}", self.build)
            }
        } else {
            String::new()
        };

        let pre_release_tag =
            if self.pre_release != 0 { format!("-{}", self.pre_release) } else { String::new() };

        write!(f, "v{}.{}.{}{}{}", self.major, self.minor, self.patch, pre_release_tag, build_tag)
    }
}

impl ProtocolVersionFormatV0 {
    /// Returns true if the build tag is human-readable, false otherwise.
    pub fn is_readable_build_tag(&self) -> bool {
        for (i, &c) in self.build.iter().enumerate() {
            if c == 0 {
                // Trailing zeros are allowed
                if self.build[i..].iter().any(|&d| d != 0) {
                    return false;
                }
                return true;
            }

            // following semver.org advertised regex, alphanumeric with '-' and '.', except leading
            // '.'.
            if !(c.is_ascii_alphanumeric() || c == b'-' || (c == b'.' && i > 0)) {
                return false;
            }
        }
        true
    }

    /// Version-type 0 byte encoding:
    ///
    /// ```text
    /// <reserved><build><major><minor><patch><pre-release>
    /// <reserved> ::= <7 zeroed bytes>
    /// <build> ::= <8 bytes>
    /// <major> ::= <big-endian uint32>
    /// <minor> ::= <big-endian uint32>
    /// <patch> ::= <big-endian uint32>
    /// <pre-release> ::= <big-endian uint32>
    /// ```
    pub fn encode(&self) -> [u8; 31] {
        let mut bytes = [0u8; 31];
        bytes[0..7].copy_from_slice(&[0u8; 7]);
        bytes[7..15].copy_from_slice(&self.build.0);
        bytes[15..19].copy_from_slice(&self.major.to_be_bytes());
        bytes[19..23].copy_from_slice(&self.minor.to_be_bytes());
        bytes[23..27].copy_from_slice(&self.patch.to_be_bytes());
        bytes[27..31].copy_from_slice(&self.pre_release.to_be_bytes());
        bytes
    }

    /// Version-type 0 byte encoding:
    ///
    /// ```text
    /// <reserved><build><major><minor><patch><pre-release>
    /// <reserved> ::= <7 zeroed bytes>
    /// <build> ::= <8 bytes>
    /// <major> ::= <big-endian uint32>
    /// <minor> ::= <big-endian uint32>
    /// <patch> ::= <big-endian uint32>
    /// <pre-release> ::= <big-endian uint32>
    /// ```
    fn decode(value: &[u8]) -> Result<Self, ProtocolVersionError> {
        if value.len() != 31 {
            return Err(ProtocolVersionError::InvalidLength { got: value.len(), expected: 31 });
        }

        Ok(Self {
            build: B64::from_slice(&value[7..15]),
            major: u32::from_be_bytes(value[15..19].try_into()?),
            minor: u32::from_be_bytes(value[19..23].try_into()?),
            patch: u32::from_be_bytes(value[23..27].try_into()?),
            pre_release: u32::from_be_bytes(value[27..31].try_into()?),
        })
    }
}

#[cfg(test)]
mod tests {
    use alloy_primitives::b256;

    use super::*;

    #[test]
    fn test_protocol_version_display() {
        assert_eq!(
            ProtocolVersion::V0(ProtocolVersionFormatV0 {
                build: B64::from_slice(&[0x61, 0x62, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00]),
                major: 42,
                minor: 0,
                patch: 2,
                pre_release: 0,
            })
            .display(),
            "v42.0.2+0x6162010000000000"
        );
    }

    #[test]
    fn test_protocol_version_accessors() {
        let inner = ProtocolVersionFormatV0 {
            build: B64::from_slice(&[0x61, 0x62, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00]),
            major: 42,
            minor: 0,
            patch: 2,
            pre_release: 0,
        };
        let protocol_version = ProtocolVersion::V0(inner);

        assert_eq!(
            protocol_version.build(),
            B64::from_slice(&[0x61, 0x62, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00])
        );
        assert_eq!(protocol_version.major(), 42);
        assert_eq!(protocol_version.minor(), 0);
        assert_eq!(protocol_version.patch(), 2);
        assert_eq!(protocol_version.pre_release(), 0);
        assert_eq!(protocol_version.inner(), inner);
        assert_eq!(protocol_version.as_v0(), Some(inner));
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_protocol_version_serde() {
        let raw_protocol_version = r#"
            "0x000000000000000061620100000000000000002a000000000000000200000000"
        "#;
        let protocol_version = ProtocolVersion::V0(ProtocolVersionFormatV0 {
            build: B64::from_slice(&[0x61, 0x62, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00]),
            major: 42,
            minor: 0,
            patch: 2,
            pre_release: 0,
        });

        let encoded = serde_json::to_string(&protocol_version).unwrap();
        assert_eq!(encoded, raw_protocol_version.trim());
    }

    #[test]
    fn test_protocol_version_encode_decode() {
        let test_cases = [
            (
                ProtocolVersion::V0(ProtocolVersionFormatV0 {
                    build: B64::from_slice(&[0x61, 0x62, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00]),
                    major: 42,
                    minor: 0,
                    patch: 2,
                    pre_release: 0,
                }),
                "v42.0.2+0x6162010000000000",
                b256!("000000000000000061620100000000000000002a000000000000000200000000"),
            ),
            (
                ProtocolVersion::V0(ProtocolVersionFormatV0 {
                    build: B64::from_slice(&[0x61, 0x62, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00]),
                    major: 42,
                    minor: 0,
                    patch: 2,
                    pre_release: 1,
                }),
                "v42.0.2-1+0x6162010000000000",
                b256!("000000000000000061620100000000000000002a000000000000000200000001"),
            ),
            (
                ProtocolVersion::V0(ProtocolVersionFormatV0 {
                    build: B64::from_slice(&[0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08]),
                    major: 42,
                    minor: 0,
                    patch: 2,
                    pre_release: 0,
                }),
                "v42.0.2+0x0102030405060708",
                b256!("000000000000000001020304050607080000002a000000000000000200000000"),
            ),
            (
                ProtocolVersion::V0(ProtocolVersionFormatV0 {
                    build: B64::from_slice(&[0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00]),
                    major: 0,
                    minor: 100,
                    patch: 2,
                    pre_release: 0,
                }),
                "v0.100.2",
                b256!("0000000000000000000000000000000000000000000000640000000200000000"),
            ),
            (
                ProtocolVersion::V0(ProtocolVersionFormatV0 {
                    build: B64::from_slice(&[b'O', b'P', b'-', b'm', b'o', b'd', 0x00, 0x00]),
                    major: 42,
                    minor: 0,
                    patch: 2,
                    pre_release: 1,
                }),
                "v42.0.2-1+OP-mod",
                b256!("00000000000000004f502d6d6f6400000000002a000000000000000200000001"),
            ),
            (
                ProtocolVersion::V0(ProtocolVersionFormatV0 {
                    build: B64::from_slice(&[b'a', b'b', 0x01, 0x00, 0x00, 0x00, 0x00, 0x00]),
                    major: 42,
                    minor: 0,
                    patch: 2,
                    pre_release: 0,
                }),
                "v42.0.2+0x6162010000000000", // do not render invalid alpha numeric
                b256!("000000000000000061620100000000000000002a000000000000000200000000"),
            ),
            (
                ProtocolVersion::V0(ProtocolVersionFormatV0 {
                    build: B64::from_slice(b"beta.123"),
                    major: 1,
                    minor: 0,
                    patch: 0,
                    pre_release: 0,
                }),
                "v1.0.0+beta.123",
                b256!("0000000000000000626574612e31323300000001000000000000000000000000"),
            ),
        ]
        .to_vec();

        for (decoded_exp, formatted_exp, encoded_exp) in test_cases {
            encode_decode_v0(encoded_exp, formatted_exp, decoded_exp);
        }
    }

    fn encode_decode_v0(encoded_exp: B256, formatted_exp: &str, decoded_exp: ProtocolVersion) {
        let decoded = ProtocolVersion::decode(encoded_exp).unwrap();
        assert_eq!(decoded, decoded_exp);

        let encoded = decoded.encode();
        assert_eq!(encoded, encoded_exp);

        let formatted = decoded.display();
        assert_eq!(formatted, formatted_exp);
    }
}
