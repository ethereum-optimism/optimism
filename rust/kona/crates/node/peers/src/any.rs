//! Contains the `AnyNode` enum, which can represent a peer in any form.

use crate::{NodeRecord, PeerId};
use derive_more::From;
use discv5::{Enr, enr::EnrPublicKey};
use libp2p::swarm::dial_opts::DialOpts;

use super::utils::peer_id_to_secp256k1_pubkey;

/// A peer that can come in [`Enr`] or [`NodeRecord`] form.
#[derive(Debug, Clone, From, Eq, PartialEq, Hash)]
pub enum AnyNode {
    /// An "enode:" peer with full ip
    NodeRecord(NodeRecord),
    /// An "enr:" peer
    Enr(Enr),
    /// An incomplete "enode" with only a peer id
    PeerId(PeerId),
}

/// An error that can occur when converting an [`AnyNode`] to [`DialOpts`].
#[derive(Debug, thiserror::Error)]
pub enum DialOptsError {
    /// The peer id is not valid and cannot be converted to a secp256k1 public key.
    #[error("Invalid peer id. Error: {0}")]
    InvalidPeerId(secp256k1::Error),
    /// The secp256k1 public key cannot be converted to a libp2p peer id.
    #[error("Invalid public key. Error: {0}")]
    InvalidPublicKey(#[from] discv5::libp2p_identity::DecodingError),
}

impl AnyNode {
    /// Returns the local peer id of the node.
    pub fn peer_id(&self) -> PeerId {
        match self {
            Self::NodeRecord(record) => record.id,
            Self::Enr(enr) => PeerId::from_slice(&enr.public_key().encode_uncompressed()),
            Self::PeerId(peer_id) => *peer_id,
        }
    }

    /// Converts the [`AnyNode`] into [`DialOpts`].
    pub fn as_dial_opts(&self) -> Result<DialOpts, DialOptsError> {
        let pub_key = &peer_id_to_secp256k1_pubkey(self.peer_id())
            .map_err(DialOptsError::InvalidPeerId)?
            .serialize();

        // codecov:ignore-start
        // We ignore the code coverage because in theory, the serialization of the public key
        // should never fail, but we don't want to panic in case of a bug.
        let pub_key: discv5::libp2p_identity::PublicKey =
            discv5::libp2p_identity::secp256k1::PublicKey::try_from_bytes(pub_key)?.into();
        // codecov:ignore-end

        let libp2p_id = libp2p::PeerId::from_public_key(&pub_key);
        Ok(libp2p_id.into())
    }
}

#[allow(clippy::from_over_into)]
impl TryInto<DialOpts> for AnyNode {
    type Error = DialOptsError;

    fn try_into(self) -> Result<DialOpts, Self::Error> {
        self.as_dial_opts()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::b512;
    use std::str::FromStr;

    #[test]
    fn test_into_dial_opts() {
        let peer_id: PeerId = b512!(
            "ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec"
        );
        let any_node = AnyNode::from(peer_id);
        let _: DialOpts = any_node.try_into().unwrap();
    }

    #[test]
    fn test_into_dial_ops_error() {
        let peer_id: PeerId = PeerId::ZERO;
        let any_node = AnyNode::from(peer_id);
        let res: DialOptsError = any_node.as_dial_opts().unwrap_err();
        assert!(matches!(res, DialOptsError::InvalidPeerId(_)));
    }

    #[test]
    fn test_peer_id_node_record() {
        let raw = crate::OP_RAW_BOOTNODES[8];
        let record = NodeRecord::from_str(raw).unwrap();
        let any_node = AnyNode::from(record);
        let peer_id = any_node.peer_id();
        let expected = b512!(
            "ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec"
        );
        assert_eq!(peer_id, expected);
    }

    #[test]
    fn test_peer_id_enr() {
        let enr = Enr::from_str("enr:-J64QBbwPjPLZ6IOOToOLsSjtFUjjzN66qmBZdUexpO32Klrc458Q24kbty2PdRaLacHM5z-cZQr8mjeQu3pik6jPSOGAYYFIqBfgmlkgnY0gmlwhDaRWFWHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECmeSnJh7zjKrDSPoNMGXoopeDF4hhpj5I0OsQUUt4u8uDdGNwgiQGg3VkcIIkBg").unwrap();
        let any_node = AnyNode::from(enr);
        let peer_id = any_node.peer_id();
        let expected = b512!(
            "99e4a7261ef38caac348fa0d3065e8a29783178861a63e48d0eb10514b78bbcb03c757c6bd6db80e8560bd176ce44781ca161dcacd49b54ee02b45d1a7135b18"
        );
        assert_eq!(peer_id, expected);
    }

    #[test]
    fn test_peer_id_from_peer_id() {
        let peer_id: PeerId = b512!(
            "ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec"
        );
        let any_node = AnyNode::from(peer_id);
        let parsed = any_node.peer_id();
        assert_eq!(parsed, peer_id);
    }
}
