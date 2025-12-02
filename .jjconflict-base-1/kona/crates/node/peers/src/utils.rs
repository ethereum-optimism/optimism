//! Utilities to translate types.

use discv5::{
    Enr,
    enr::{CombinedPublicKey, EnrPublicKey},
    multiaddr::Protocol,
};
use libp2p::Multiaddr;

use super::PeerId;

/// Converts an [`Enr`] into a [`Multiaddr`].
pub fn enr_to_multiaddr(enr: &Enr) -> Option<Multiaddr> {
    let mut addr = if let Some(socket) = enr.tcp4_socket() {
        let mut addr = Multiaddr::from(*socket.ip());
        addr.push(Protocol::Tcp(socket.port()));
        addr
    } else if let Some(socket) = enr.tcp6_socket() {
        let mut addr = Multiaddr::from(*socket.ip());
        addr.push(Protocol::Tcp(socket.port()));
        addr
    } else {
        return None;
    };

    let CombinedPublicKey::Secp256k1(pub_key) = enr.public_key() else {
        return None;
    };

    let pub_key = libp2p_identity::secp256k1::PublicKey::try_from_bytes(&pub_key.encode()).ok()?;
    let pub_key = libp2p_identity::PublicKey::from(pub_key);

    addr.push(Protocol::P2p(libp2p::PeerId::from_public_key(&pub_key)));

    Some(addr)
}

/// Converts an uncompressed [`PeerId`] to a [`secp256k1::PublicKey`] by prepending the [`PeerId`]
/// bytes with the `SECP256K1_TAG_PUBKEY_UNCOMPRESSED` tag.
pub fn peer_id_to_secp256k1_pubkey(id: PeerId) -> Result<secp256k1::PublicKey, secp256k1::Error> {
    /// Tags the public key as uncompressed.
    ///
    /// See: <https://github.com/bitcoin-core/secp256k1/blob/master/include/secp256k1.h#L211>
    const SECP256K1_TAG_PUBKEY_UNCOMPRESSED: u8 = 4;

    let mut full_pubkey = [0u8; secp256k1::constants::UNCOMPRESSED_PUBLIC_KEY_SIZE];
    full_pubkey[0] = SECP256K1_TAG_PUBKEY_UNCOMPRESSED;
    full_pubkey[1..].copy_from_slice(id.as_slice());
    secp256k1::PublicKey::from_slice(&full_pubkey)
}

/// An error that can occur when converting a [`PeerId`] to a [`libp2p::PeerId`].
#[derive(Debug, thiserror::Error)]
pub enum PeerIdConversionError {
    /// The peer id is not valid and cannot be converted to a secp256k1 public key.
    #[error("Invalid peer id: {0}")]
    InvalidPeerId(secp256k1::Error),
    /// The secp256k1 public key cannot be converted to a libp2p peer id. This is a bug.
    #[error("Invalid conversion from secp256k1 public key to libp2p peer id: {0}. This is a bug.")]
    InvalidPublicKey(#[from] discv5::libp2p_identity::DecodingError),
}

/// Converts an uncoded [`PeerId`] to a [`libp2p::PeerId`]. These two types represent the same
/// underlying concept (secp256k1 public key) but using different encodings (the local [`PeerId`] is
/// the uncompressed representation of the public key, while the "p2plib" [`libp2p::PeerId`] is a
/// more complex representation, involving protobuf encoding and bitcoin encoding,  defined here: <https://github.com/libp2p/specs/blob/master/peer-ids/peer-ids.md>).
pub fn local_id_to_p2p_id(peer_id: PeerId) -> Result<libp2p::PeerId, PeerIdConversionError> {
    // The libp2p library works with compressed public keys.
    let encoded_pk_bytes = peer_id_to_secp256k1_pubkey(peer_id)
        .map_err(PeerIdConversionError::InvalidPeerId)?
        .serialize();
    let pk: discv5::libp2p_identity::PublicKey =
        discv5::libp2p_identity::secp256k1::PublicKey::try_from_bytes(&encoded_pk_bytes)?.into();

    Ok(pk.to_peer_id())
}

#[cfg(test)]
mod tests {
    use std::net::{Ipv4Addr, Ipv6Addr};

    use super::*;
    use crate::PeerId;
    use alloy_primitives::hex::FromHex;
    use discv5::enr::{CombinedKey, Enr, EnrKey};

    #[test]
    fn test_resolve_multiaddr() {
        let ip = Ipv4Addr::new(132, 145, 16, 10);
        let tcp_port = 9000;
        let udp_port = 9001;
        let private_key = CombinedKey::generate_secp256k1();

        let public_key = private_key.public().encode();
        let public_key =
            libp2p_identity::secp256k1::PublicKey::try_from_bytes(&public_key).unwrap();
        let peer_id = libp2p::PeerId::from_public_key(&public_key.into());

        let enr = Enr::builder().ip4(ip).tcp4(tcp_port).udp4(udp_port).build(&private_key).unwrap();

        let multiaddr = enr_to_multiaddr(&enr).unwrap();

        let mut received_ip = None;
        let mut received_tcp_port = None;
        let mut received_p2p_id = None;

        for protocol in multiaddr.iter() {
            match protocol {
                Protocol::Ip4(ip) => {
                    received_ip = Some(ip);
                }
                Protocol::Tcp(port) => {
                    received_tcp_port = Some(port);
                }
                Protocol::P2p(id) => {
                    received_p2p_id = Some(id);
                }
                _ => {
                    panic!("Unexpected protocol: {protocol:?}");
                }
            }
        }
        assert_eq!(received_ip, Some(ip));
        assert_eq!(received_tcp_port, Some(tcp_port));
        assert_eq!(received_p2p_id, Some(peer_id));
    }

    #[test]
    fn test_resolve_multiaddr_ipv6() {
        let ip = Ipv6Addr::new(0x2001, 0xdb8, 0x0a, 0x11, 0x1e, 0x8a, 0x2e, 0x3a);
        let tcp_port = 9000;
        let udp_port = 9001;
        let private_key = CombinedKey::generate_secp256k1();

        let public_key = private_key.public().encode();
        let public_key =
            libp2p_identity::secp256k1::PublicKey::try_from_bytes(&public_key).unwrap();
        let peer_id = libp2p::PeerId::from_public_key(&public_key.into());

        let enr = Enr::builder().ip6(ip).tcp6(tcp_port).udp6(udp_port).build(&private_key).unwrap();

        let multiaddr = enr_to_multiaddr(&enr).unwrap();

        let mut received_ip = None;
        let mut received_tcp_port = None;
        let mut received_p2p_id = None;

        for protocol in multiaddr.iter() {
            match protocol {
                Protocol::Ip6(ip) => {
                    received_ip = Some(ip);
                }
                Protocol::Tcp(port) => {
                    received_tcp_port = Some(port);
                }
                Protocol::P2p(id) => {
                    received_p2p_id = Some(id);
                }
                _ => {
                    panic!("Unexpected protocol: {protocol:?}");
                }
            }
        }
        assert_eq!(received_ip, Some(ip));
        assert_eq!(received_tcp_port, Some(tcp_port));
        assert_eq!(received_p2p_id, Some(peer_id));
    }

    #[test]
    fn test_convert_local_peer_id_to_multi_peer_id() {
        let p2p_keypair = discv5::libp2p_identity::secp256k1::Keypair::generate();
        let uncompressed = p2p_keypair.public().to_bytes_uncompressed();
        let local_peer_id = PeerId::from_slice(&uncompressed[1..]);

        // We need to convert the local peer id (uncompressed secp256k1 public key) to a libp2p
        // peer id (protocol buffer encoded public key).
        let peer_id = local_id_to_p2p_id(local_peer_id).unwrap();

        let p2p_public_key: discv5::libp2p_identity::PublicKey =
            p2p_keypair.public().clone().into();

        assert_eq!(peer_id, p2p_public_key.to_peer_id());
    }

    #[test]
    fn test_hardcoded_peer_id() {
        const PUB_KEY_STR: &str = "548f715f3fc388a7c917ba644a2f16270f1ede48a5d88a4d14ea287cc916068363f3092e39936f1a3e7885198bef0e5af951f1d7b1041ce8ba4010917777e71f";
        let pub_key = PeerId::from_hex(PUB_KEY_STR).unwrap();

        // We need to convert the local peer id (uncompressed secp256k1 public key) to a libp2p
        // peer id (protocol buffer encoded public key).
        let peer_id = local_id_to_p2p_id(pub_key).unwrap();

        let uncompressed_pub_key = peer_id_to_secp256k1_pubkey(pub_key).unwrap();

        let p2p_public_key: discv5::libp2p_identity::PublicKey =
            discv5::libp2p_identity::secp256k1::PublicKey::try_from_bytes(
                &uncompressed_pub_key.serialize(),
            )
            .unwrap()
            .into();

        assert_eq!(peer_id, p2p_public_key.to_peer_id());
    }
}
