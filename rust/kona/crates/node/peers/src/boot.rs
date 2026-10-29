//! Contains the [`BootNode`] type which is used to represent a boot node in the network.

use crate::{NodeRecord, NodeRecordParseError, enr_to_multiaddr};
use derive_more::{Display, From};
use discv5::Enr;
use libp2p::multiaddr::{Multiaddr, Protocol};
use serde::{Deserialize, Serialize};
use std::{net::IpAddr, str::FromStr};

use super::utils::{PeerIdConversionError, local_id_to_p2p_id};

/// A boot node can be added either as a string in either 'enode' URL scheme or serialized from
/// [`Enr`] type.
#[derive(Clone, Debug, PartialEq, Eq, Hash, From, Display, Serialize, Deserialize)]
pub enum BootNode {
    /// An unsigned node record.
    #[display("{_0}")]
    Enode(Multiaddr),
    /// A signed node record.
    #[display("{_0:?}")]
    Enr(Enr),
}

impl BootNode {
    /// Parses a [`NodeRecord`] and serializes according to CL format. Note: [`discv5`] is
    /// originally a CL library hence needs this format to add the node.
    pub fn from_unsigned(node_record: NodeRecord) -> Result<Self, PeerIdConversionError> {
        let NodeRecord { address, udp_port, id, tcp_port } = node_record;
        let mut multi_address = Multiaddr::empty();
        match address {
            IpAddr::V4(ip) => multi_address.push(Protocol::Ip4(ip)),
            IpAddr::V6(ip) => multi_address.push(Protocol::Ip6(ip)),
        }

        multi_address.push(Protocol::Udp(udp_port));
        multi_address.push(Protocol::Tcp(tcp_port));
        multi_address.push(Protocol::P2p(local_id_to_p2p_id(id)?));

        Ok(Self::Enode(multi_address))
    }

    /// Converts a [`BootNode`] into a [`Multiaddr`].
    pub fn to_multiaddr(&self) -> Option<Multiaddr> {
        match self {
            Self::Enode(addr) => Some(addr.clone()),
            Self::Enr(enr) => enr_to_multiaddr(enr),
        }
    }

    /// Parses a bootnode from an ENR (`enr:...`) or an enode URL (`enode://...`).
    pub fn parse_bootnode(raw: &str) -> Result<Self, BootNodeParseError> {
        // If the string starts with "enr:" it is an ENR record.
        if raw.starts_with("enr:") {
            let enr = Enr::from_str(raw).map_err(BootNodeParseError::InvalidEnr)?;
            return Ok(Self::from(enr));
        }
        // Otherwise, attempt to use the Node Record format.
        let record = NodeRecord::from_str(raw).map_err(BootNodeParseError::InvalidEnode)?;
        Self::from_unsigned(record).map_err(BootNodeParseError::InvalidPeerId)
    }
}

/// An error parsing a [`BootNode`].
#[derive(Debug, thiserror::Error)]
pub enum BootNodeParseError {
    /// The ENR is invalid. The ENR parser only gives its reason as a string.
    #[error("Invalid ENR: {0}")]
    InvalidEnr(String),
    /// The enode URL is invalid.
    #[error("Invalid enode URL")]
    InvalidEnode(#[source] NodeRecordParseError),
    /// The enode URL's node ID is not a valid public key.
    #[error("Invalid enode node ID")]
    InvalidPeerId(#[source] PeerIdConversionError),
}

#[cfg(test)]
mod tests {
    use discv5::NodeContact;
    use enr::{CombinedPublicKey, k256};

    use crate::utils::peer_id_to_secp256k1_pubkey;

    use super::*;
    use std::{net::Ipv4Addr, str::FromStr};

    /// Malformed bootnodes are errors, not panics.
    #[test]
    fn test_parse_bootnode_rejects_invalid_records() {
        let id = "ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec";
        let cases = [
            ("not-a-bootnode".to_string(), "Invalid enode URL"),
            ("enr:-not-base64!".to_string(), "Invalid ENR"),
            // No port.
            (format!("enode://{id}@34.65.175.185"), "Invalid enode URL"),
            (format!("enode://{id}@34.65.175.185:30305?discport=x"), "Invalid enode URL"),
            // 64 bytes, but not a point on the curve.
            (format!("enode://{}@34.65.175.185:30305", "ff".repeat(64)), "Invalid enode node ID"),
        ];
        for (raw, expected) in cases {
            let err = BootNode::parse_bootnode(&raw).unwrap_err();
            assert!(err.to_string().starts_with(expected), "{raw}: {err:?}");
        }
    }

    #[test]
    fn test_derive_bootnode_enode_multiaddr() {
        let hardcoded_enode = "enode://2bd2e657bb3c8efffb8ff6db9071d9eb7be70d7c6d7d980ff80fc93b2629675c5f750bc0a5ef27cd788c2e491b8795a7e9a4a6e72178c14acc6753c0e5d77ae4@34.65.205.244:30305";
        let node_record = NodeRecord::from_str(hardcoded_enode).unwrap();
        let boot_node = BootNode::from_unsigned(node_record).unwrap();

        // Get the substring from hardcoded_enode between the second / and before the @
        let peer_id = hardcoded_enode
            [hardcoded_enode.find('/').unwrap() + 2..hardcoded_enode.find('@').unwrap()]
            .to_string();

        let peer_id = crate::PeerId::from_str(&peer_id).unwrap();
        let p2p_peer_id = local_id_to_p2p_id(peer_id).unwrap();

        let expected_multiaddr = Multiaddr::empty()
            .with(Protocol::Ip4(Ipv4Addr::new(34, 65, 205, 244)))
            .with(Protocol::Udp(30305))
            .with(Protocol::Tcp(30305))
            .with(Protocol::P2p(p2p_peer_id));

        assert_eq!(boot_node.to_multiaddr(), Some(expected_multiaddr));
    }

    #[test]
    fn test_derive_bootnode_enode_multiaddr_back_and_forth() {
        let hardcoded_enode = "enode://2bd2e657bb3c8efffb8ff6db9071d9eb7be70d7c6d7d980ff80fc93b2629675c5f750bc0a5ef27cd788c2e491b8795a7e9a4a6e72178c14acc6753c0e5d77ae4@34.65.205.244:30305";
        let node_record = NodeRecord::from_str(hardcoded_enode).unwrap();
        let boot_node = BootNode::from_unsigned(node_record).unwrap();
        let multiaddr = boot_node.to_multiaddr().unwrap();

        let node_contact = NodeContact::try_from_multiaddr(multiaddr).unwrap();

        // The public key contained in the NodeContact is used to connect to the
        // bootnode.
        let contact_pkey = node_contact.public_key();

        // We get the expected public key from the enode information.
        let peer_id = hardcoded_enode
            [hardcoded_enode.find('/').unwrap() + 2..hardcoded_enode.find('@').unwrap()]
            .to_string();
        let peer_id = crate::PeerId::from_str(&peer_id).unwrap();

        // The public key from the peer id is using the uncompressed form.
        let pkey_secp256k1 = peer_id_to_secp256k1_pubkey(peer_id).unwrap();
        let p2p_public_key: libp2p_identity::secp256k1::PublicKey =
            libp2p_identity::secp256k1::PublicKey::try_from_bytes(&pkey_secp256k1.serialize())
                .unwrap();

        let expected_pkey: CombinedPublicKey =
            k256::ecdsa::VerifyingKey::from_sec1_bytes(&p2p_public_key.to_bytes()).unwrap().into();

        // These two keys should be equal.
        assert_eq!(contact_pkey, expected_pkey);
    }
}
