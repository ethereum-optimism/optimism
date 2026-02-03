//! Contains the Optimism consensus-layer ENR Type.

use alloy_rlp::{Decodable, Encodable};
use discv5::Enr;
use unsigned_varint::{decode, encode};

/// Validates the [`Enr`] for the OP Stack.
#[derive(Debug, derive_more::Display, Clone, Default, PartialEq, Eq)]
pub enum EnrValidation {
    /// Conversion error.
    #[display("Conversion error: {_0}")]
    ConversionError(OpStackEnrError),
    /// Invalid Chain ID.
    #[display("Invalid Chain ID: {_0}")]
    InvalidChainId(u64),
    /// Valid ENR.
    #[default]
    #[display("Valid ENR")]
    Valid,
}

impl EnrValidation {
    /// Validates the [`Enr`] for the OP Stack.
    pub fn validate(enr: &Enr, chain_id: u64) -> Self {
        let opstack_enr = match OpStackEnr::try_from(enr) {
            Ok(opstack_enr) => opstack_enr,
            Err(e) => return Self::ConversionError(e),
        };

        if opstack_enr.chain_id != chain_id {
            return Self::InvalidChainId(opstack_enr.chain_id);
        }

        Self::Valid
    }

    /// Returns `true` if the ENR is valid.
    pub const fn is_valid(&self) -> bool {
        matches!(self, Self::Valid)
    }

    /// Returns `true` if the ENR is invalid.
    pub const fn is_invalid(&self) -> bool {
        !self.is_valid()
    }
}

/// The unique L2 network identifier
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
pub struct OpStackEnr {
    /// Chain ID
    pub chain_id: u64,
    /// The version. Always set to 0.
    pub version: u64,
}

/// The error type that can be returned when trying to convert an [`Enr`] to an [`OpStackEnr`].
#[derive(Debug, thiserror::Error, Clone, PartialEq, Eq)]
pub enum OpStackEnrError {
    /// Missing OP Stack ENR key.
    #[error("Missing OP Stack ENR key")]
    MissingKey,
    /// Failed to decode the OP Stack ENR Value.
    #[error("Failed to decode the OP Stack ENR Value: {0}")]
    DecodeError(String),
    /// Invalid version.
    #[error("Invalid version: {0}")]
    InvalidVersion(u64),
}

impl TryFrom<&Enr> for OpStackEnr {
    type Error = OpStackEnrError;
    fn try_from(enr: &Enr) -> Result<Self, Self::Error> {
        let Some(mut opstack) = enr.get_raw_rlp(Self::OP_CL_KEY) else {
            return Err(OpStackEnrError::MissingKey);
        };
        let opstack_enr =
            Self::decode(&mut opstack).map_err(|e| OpStackEnrError::DecodeError(e.to_string()))?;

        if opstack_enr.version != 0 {
            return Err(OpStackEnrError::InvalidVersion(opstack_enr.version));
        }

        Ok(opstack_enr)
    }
}

impl OpStackEnr {
    /// The [`Enr`] key literal string for the consensus layer.
    pub const OP_CL_KEY: &str = "opstack";

    /// Constructs an [`OpStackEnr`] from a chain id.
    pub const fn from_chain_id(chain_id: u64) -> Self {
        Self { chain_id, version: 0 }
    }
}

impl Encodable for OpStackEnr {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        let mut chain_id_buf = encode::u128_buffer();
        let chain_id_slice = encode::u128(self.chain_id as u128, &mut chain_id_buf);

        let mut version_buf = encode::u128_buffer();
        let version_slice = encode::u128(self.version as u128, &mut version_buf);

        let opstack = [chain_id_slice, version_slice].concat();
        alloy_primitives::Bytes::from(opstack).encode(out);
    }
}

impl Decodable for OpStackEnr {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let bytes = alloy_primitives::Bytes::decode(buf)?;
        let (chain_id, rest) = decode::u64(&bytes)
            .map_err(|_| alloy_rlp::Error::Custom("could not decode chain id"))?;
        let (version, _) =
            decode::u64(rest).map_err(|_| alloy_rlp::Error::Custom("could not decode version"))?;
        Ok(Self { chain_id, version })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Bytes, bytes};
    use discv5::enr::CombinedKey;

    #[test]
    #[cfg(feature = "arbitrary")]
    fn roundtrip_op_stack_enr() {
        arbtest::arbtest(|u| {
            let op_stack_enr = OpStackEnr::from_chain_id(u.arbitrary()?);
            let bytes = alloy_rlp::encode(op_stack_enr).to_vec();
            let decoded = OpStackEnr::decode(&mut &bytes[..]).unwrap();
            assert_eq!(decoded, op_stack_enr);
            Ok(())
        });
    }

    #[test]
    fn test_enr_validation() {
        let key = CombinedKey::generate_secp256k1();
        let mut enr = Enr::builder().build(&key).unwrap();
        let op_stack_enr = OpStackEnr::from_chain_id(10);
        let mut op_stack_bytes = Vec::new();
        op_stack_enr.encode(&mut op_stack_bytes);
        enr.insert_raw_rlp(OpStackEnr::OP_CL_KEY, op_stack_bytes.into(), &key).unwrap();
        assert!(EnrValidation::validate(&enr, 10).is_valid());
        assert!(EnrValidation::validate(&enr, 11).is_invalid());
    }

    #[test]
    fn test_enr_validation_invalid_version() {
        let key = CombinedKey::generate_secp256k1();
        let mut enr = Enr::builder().build(&key).unwrap();
        let mut op_stack_enr = OpStackEnr::from_chain_id(10);
        op_stack_enr.version = 1;
        let mut op_stack_bytes = Vec::new();
        op_stack_enr.encode(&mut op_stack_bytes);
        enr.insert_raw_rlp(OpStackEnr::OP_CL_KEY, op_stack_bytes.into(), &key).unwrap();
        assert!(EnrValidation::validate(&enr, 10).is_invalid());
    }

    #[test]
    fn test_op_mainnet_enr() {
        let op_enr = OpStackEnr::from_chain_id(10);
        let bytes = alloy_rlp::encode(op_enr).to_vec();
        assert_eq!(Bytes::from(bytes.clone()), bytes!("820A00"));
        let decoded = OpStackEnr::decode(&mut &bytes[..]).unwrap();
        assert_eq!(decoded, op_enr);
    }

    #[test]
    fn test_base_mainnet_enr() {
        let base_enr = OpStackEnr::from_chain_id(8453);
        let bytes = alloy_rlp::encode(base_enr).to_vec();
        assert_eq!(Bytes::from(bytes.clone()), bytes!("83854200"));
        let decoded = OpStackEnr::decode(&mut &bytes[..]).unwrap();
        assert_eq!(decoded, base_enr);
    }
}
