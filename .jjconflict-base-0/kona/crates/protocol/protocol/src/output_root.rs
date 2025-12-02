//! The [`OutputRoot`] type.

use alloy_primitives::{B256, keccak256};
use derive_more::Display;

/// The [`OutputRoot`] is a high-level commitment to an L2 block. It lifts the state root from the
/// block header as well as the storage root of the [Predeploys::L2_TO_L1_MESSAGE_PASSER] account
/// into the top-level commitment construction.
///
/// <https://specs.optimism.io/protocol/proposals.html#l2-output-commitment-construction>
///
/// [Predeploys::L2_TO_L1_MESSAGE_PASSER]: crate::Predeploys::L2_TO_L1_MESSAGE_PASSER
#[derive(Debug, Display, Clone, Copy, PartialEq, Eq, Hash)]
#[display("OutputRootV0({}, {}, {})", state_root, bridge_storage_root, block_hash)]
pub struct OutputRoot {
    /// The state root of the block corresponding to the output root.
    pub state_root: B256,
    /// The storage root of the `L2ToL1MessagePasser` predeploy at the block corresponding to the
    /// output root.
    pub bridge_storage_root: B256,
    /// The block hash that the output root represents.
    pub block_hash: B256,
}

impl OutputRoot {
    /// The encoded length of a V0 output root.
    pub const ENCODED_LENGTH: usize = 128;

    /// The version of the [`OutputRoot`]. Currently, the protocol only supports one version of this
    /// commitment.
    pub const VERSION: u8 = 0;

    /// Returns the version of the [`OutputRoot`]. Currently, the protocol only supports the version
    /// number 0.
    pub const fn version(&self) -> B256 {
        B256::ZERO
    }

    /// Constructs a V0 [`OutputRoot`] from its parts.
    pub const fn from_parts(state_root: B256, bridge_storage_root: B256, block_hash: B256) -> Self {
        Self { state_root, bridge_storage_root, block_hash }
    }

    /// Encodes the [`OutputRoot`].
    pub fn encode(&self) -> [u8; Self::ENCODED_LENGTH] {
        let mut encoded = [0u8; Self::ENCODED_LENGTH];
        encoded[31] = Self::VERSION;
        encoded[32..64].copy_from_slice(self.state_root.as_slice());
        encoded[64..96].copy_from_slice(self.bridge_storage_root.as_slice());
        encoded[96..128].copy_from_slice(self.block_hash.as_slice());
        encoded
    }

    /// Encodes and hashes the [`OutputRoot`].
    pub fn hash(&self) -> B256 {
        keccak256(self.encode())
    }
}

#[cfg(test)]
mod test {
    use super::OutputRoot;
    use alloy_primitives::{B256, Bytes, b256, bytes};

    fn test_or() -> OutputRoot {
        OutputRoot::from_parts(
            B256::left_padding_from(&[0xbe, 0xef]),
            B256::left_padding_from(&[0xba, 0xbe]),
            B256::left_padding_from(&[0xc0, 0xde]),
        )
    }

    #[test]
    fn test_hash_output_root() {
        const EXPECTED_HASH: B256 =
            b256!("0c39fb6b07cf6694b13e63e59f7b15255be1c93a4d6d3e0da6c99729647c0d11");

        let root = test_or();
        assert_eq!(root.hash(), EXPECTED_HASH);
    }

    #[test]
    fn test_encode_output_root() {
        const EXPECTED_ENCODING: Bytes = bytes!(
            "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000beef000000000000000000000000000000000000000000000000000000000000babe000000000000000000000000000000000000000000000000000000000000c0de"
        );

        let root = OutputRoot::from_parts(
            B256::left_padding_from(&[0xbe, 0xef]),
            B256::left_padding_from(&[0xba, 0xbe]),
            B256::left_padding_from(&[0xc0, 0xde]),
        );

        assert_eq!(root.encode().as_ref(), EXPECTED_ENCODING.as_ref());
    }
}
