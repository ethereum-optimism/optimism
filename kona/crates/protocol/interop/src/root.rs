//! The [SuperRoot] type.
//!
//! Represents a snapshot of the state of the superchain at a given integer timestamp.

use crate::{SUPER_ROOT_VERSION, SuperRootError, SuperRootResult};
use alloc::vec::Vec;
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, Bytes, U256, keccak256};
use alloy_rlp::{Buf, BufMut};

/// The [SuperRoot] is the snapshot of the superchain at a given timestamp.
#[derive(Debug, Clone, Eq, PartialEq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct SuperRoot {
    /// The timestamp of the superchain snapshot, in seconds.
    pub timestamp: u64,
    /// The chain IDs and output root commitments of all chains within the dependency set.
    pub output_roots: Vec<OutputRootWithChain>,
}

impl SuperRoot {
    /// Create a new [SuperRoot] with the given timestamp and output roots.
    pub fn new(timestamp: u64, mut output_roots: Vec<OutputRootWithChain>) -> Self {
        // Guarantee that the output roots are sorted by chain ID.
        output_roots.sort_by_key(|r| r.chain_id);
        Self { timestamp, output_roots }
    }

    /// Decodes a [SuperRoot] from the given buffer.
    pub fn decode(buf: &mut &[u8]) -> SuperRootResult<Self> {
        if buf.is_empty() {
            return Err(SuperRootError::UnexpectedLength);
        }

        let version = buf[0];
        if version != SUPER_ROOT_VERSION {
            return Err(SuperRootError::InvalidVersionByte);
        }
        buf.advance(1);

        if buf.len() < 8 {
            return Err(SuperRootError::UnexpectedLength);
        }
        let timestamp = u64::from_be_bytes(buf[0..8].try_into()?);
        buf.advance(8);

        let mut output_roots = Vec::new();
        while !buf.is_empty() {
            if buf.len() < 64 {
                return Err(SuperRootError::UnexpectedLength);
            }

            let chain_id = U256::from_be_bytes::<32>(buf[0..32].try_into()?);
            buf.advance(32);
            let output_root = B256::from_slice(&buf[0..32]);
            buf.advance(32);
            output_roots.push(OutputRootWithChain::new(chain_id.to(), output_root));
        }

        Ok(Self { timestamp, output_roots })
    }

    /// Encode the [SuperRoot] into the given buffer.
    pub fn encode(&self, out: &mut dyn BufMut) {
        out.put_u8(SUPER_ROOT_VERSION);

        out.put_u64(self.timestamp);
        for output_root in &self.output_roots {
            out.put_slice(U256::from(output_root.chain_id).to_be_bytes::<32>().as_slice());
            out.put_slice(output_root.output_root.as_slice());
        }
    }

    /// Returns the encoded length of the [SuperRoot].
    pub const fn encoded_length(&self) -> usize {
        1 + 8 + 64 * self.output_roots.len()
    }

    /// Hashes the encoded [SuperRoot] using [keccak256].
    pub fn hash(&self) -> B256 {
        let mut rlp_buf = Vec::with_capacity(self.encoded_length());
        self.encode(&mut rlp_buf);
        keccak256(&rlp_buf)
    }
}

/// Chain Root Info
#[derive(Debug, Clone, Eq, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct ChainRootInfo {
    /// The chain ID.
    #[cfg_attr(feature = "serde", serde(rename = "chainID"))]
    pub chain_id: u64,
    /// The canonical output root of the latest canonical block at a particular timestamp.
    pub canonical: B256,
    /// The pending output root.
    ///
    /// This is the output root preimage for the latest block at a particular timestamp prior to
    /// validation of executing messages. If the original block was valid, this will be the
    /// preimage of the output root from the `canonical` array. If it was invalid, it will be
    /// the output root preimage from the optimistic block deposited transaction added to the
    /// deposit-only block.
    pub pending: Bytes,
}

/// The super root response type.
#[derive(Debug, Clone, Eq, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct SuperRootOutput {
    /// The Highest L1 Block that is cross-safe among all chains.
    pub cross_safe_derived_from: BlockNumHash,
    /// The timestamp of the super root.
    pub timestamp: u64,
    /// The super root hash.
    pub super_root: B256,
    /// The version of the super root.
    pub version: u8,
    /// The chain root info for each chain in the dependency set.
    /// It represents the state of the chain at or before the timestamp.
    pub chains: Vec<ChainRootInfo>,
}

/// A wrapper around an output root hash with the chain ID it belongs to.
#[derive(Debug, Clone, Eq, PartialEq, Hash)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct OutputRootWithChain {
    /// The chain ID of the output root.
    pub chain_id: u64,
    /// The output root hash.
    pub output_root: B256,
}

impl OutputRootWithChain {
    /// Create a new [OutputRootWithChain] with the given chain ID and output root hash.
    pub const fn new(chain_id: u64, output_root: B256) -> Self {
        Self { chain_id, output_root }
    }
}

#[cfg(test)]
mod test {
    use crate::{SUPER_ROOT_VERSION, errors::SuperRootError};

    use super::{OutputRootWithChain, SuperRoot};
    use alloy_primitives::{B256, b256};

    #[test]
    fn test_super_root_sorts_outputs() {
        let super_root = SuperRoot::new(
            10,
            vec![
                (OutputRootWithChain::new(3, B256::default())),
                (OutputRootWithChain::new(2, B256::default())),
                (OutputRootWithChain::new(1, B256::default())),
            ],
        );

        assert!(super_root.output_roots.windows(2).all(|w| w[0].chain_id <= w[1].chain_id));
    }

    #[test]
    fn test_super_root_empty_buf() {
        let buf: Vec<u8> = Vec::new();
        assert!(matches!(
            SuperRoot::decode(&mut buf.as_slice()).unwrap_err(),
            SuperRootError::UnexpectedLength
        ));
    }

    #[test]
    fn test_super_root_invalid_version() {
        let buf = vec![0xFF];
        assert!(matches!(
            SuperRoot::decode(&mut buf.as_slice()).unwrap_err(),
            SuperRootError::InvalidVersionByte
        ));
    }

    #[test]
    fn test_super_root_invalid_length_at_timestamp() {
        let buf = vec![SUPER_ROOT_VERSION, 0x00];
        assert!(matches!(
            SuperRoot::decode(&mut buf.as_slice()).unwrap_err(),
            SuperRootError::UnexpectedLength
        ));
    }

    #[test]
    fn test_super_root_invalid_length_malformed_output_roots() {
        let buf = [&[SUPER_ROOT_VERSION], 64u64.to_be_bytes().as_ref(), &[0xbe, 0xef]].concat();
        assert!(matches!(
            SuperRoot::decode(&mut buf.as_slice()).unwrap_err(),
            SuperRootError::UnexpectedLength
        ));
    }

    #[test]
    fn test_static_hash_super_root() {
        const EXPECTED: B256 =
            b256!("0980033cbf4337f614a2401ab7efbfdc66ab647812f1c98d891d92ddfb376541");

        let super_root = SuperRoot::new(
            10,
            vec![
                (OutputRootWithChain::new(1, B256::default())),
                (OutputRootWithChain::new(2, B256::default())),
            ],
        );
        assert_eq!(super_root.hash(), EXPECTED);
    }

    #[test]
    fn test_static_super_root_roundtrip() {
        let super_root = SuperRoot::new(
            10,
            vec![
                (OutputRootWithChain::new(1, B256::default())),
                (OutputRootWithChain::new(2, B256::default())),
            ],
        );

        let mut rlp_buf = Vec::with_capacity(super_root.encoded_length());
        super_root.encode(&mut rlp_buf);
        assert_eq!(super_root, SuperRoot::decode(&mut rlp_buf.as_slice()).unwrap());
    }

    #[test]
    fn test_arbitrary_super_root_roundtrip() {
        use arbitrary::Arbitrary;
        use rand::Rng;

        let mut bytes = [0u8; 1024];
        rand::rng().fill(bytes.as_mut_slice());
        let super_root = SuperRoot::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap();

        let mut rlp_buf = Vec::with_capacity(super_root.encoded_length());
        super_root.encode(&mut rlp_buf);
        assert_eq!(super_root, SuperRoot::decode(&mut rlp_buf.as_slice()).unwrap());
    }
}
