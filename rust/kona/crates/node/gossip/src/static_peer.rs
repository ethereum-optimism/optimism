//! Peers the node keeps connected to, whatever discovery finds.

use crate::StaticPeerError;
use libp2p::{Multiaddr, PeerId, multiaddr::Protocol};
use std::{collections::HashMap, fmt, str::FromStr};

/// A peer to dial at startup and keep connected, given by an address that ends in its
/// `/p2p/<peer id>`, such as `/dns4/node.example.com/tcp/9222/p2p/16Uiu2...`.
///
/// The peer ID is required to recognise the peer once it is connected, whichever address the
/// connection came in on.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StaticPeer {
    /// The address to dial, including the `/p2p/<peer id>` it ends in.
    addr: Multiaddr,
    /// The peer ID at the end of `addr`.
    peer_id: PeerId,
}

impl StaticPeer {
    /// Returns the address to dial, including the `/p2p/<peer id>` it ends in.
    pub const fn addr(&self) -> &Multiaddr {
        &self.addr
    }

    /// Returns the peer's ID.
    pub const fn peer_id(&self) -> PeerId {
        self.peer_id
    }
}

impl TryFrom<Multiaddr> for StaticPeer {
    type Error = StaticPeerError;

    fn try_from(addr: Multiaddr) -> Result<Self, Self::Error> {
        let mut transport = addr.clone();
        let Some(Protocol::P2p(peer_id)) = transport.pop() else {
            return Err(StaticPeerError::MissingPeerId);
        };
        if transport.is_empty() {
            return Err(StaticPeerError::MissingAddress);
        }
        Ok(Self { addr, peer_id })
    }
}

impl FromStr for StaticPeer {
    type Err = StaticPeerError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        s.parse::<Multiaddr>().map_err(StaticPeerError::InvalidMultiaddr)?.try_into()
    }
}

/// Groups static peers by peer ID, so several addresses for one peer make one peer. An address
/// listed twice is kept once.
pub(crate) fn group_by_peer(
    peers: impl IntoIterator<Item = StaticPeer>,
) -> HashMap<PeerId, Vec<Multiaddr>> {
    let mut grouped: HashMap<PeerId, Vec<Multiaddr>> = HashMap::new();
    for peer in peers {
        let addrs = grouped.entry(peer.peer_id).or_default();
        if !addrs.contains(&peer.addr) {
            addrs.push(peer.addr);
        }
    }
    grouped
}

impl fmt::Display for StaticPeer {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.addr.fmt(f)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const PEER_ID: &str = "16Uiv2HAmLFhCjjNUXiZA3Hdd3j3qp6rnD6tHkFtsSZmhpV8axfGX";

    #[test]
    fn test_group_by_peer() {
        let (peer_1, peer_2) = (PeerId::random(), PeerId::random());
        let peer = |ip: &str, id: PeerId| -> StaticPeer {
            format!("/ip4/{ip}/tcp/9222/p2p/{id}").parse().unwrap()
        };
        let peers = [
            peer("10.0.0.1", peer_1),
            peer("10.0.0.2", peer_1),
            // Listed twice.
            peer("10.0.0.1", peer_1),
            peer("10.0.0.3", peer_2),
        ];

        let grouped = group_by_peer(peers.clone());

        let expected = HashMap::from([
            (peer_1, vec![peers[0].addr.clone(), peers[1].addr.clone()]),
            (peer_2, vec![peers[3].addr.clone()]),
        ]);
        assert_eq!(grouped, expected);
    }

    #[test]
    fn test_parses_address_with_peer_id() {
        let addr = format!("/ip4/127.0.0.1/tcp/9222/p2p/{PEER_ID}");
        let peer: StaticPeer = addr.parse().unwrap();
        assert_eq!(peer.peer_id(), PEER_ID.parse().unwrap());
        assert_eq!(peer.addr().to_string(), addr);
        assert_eq!(peer.to_string(), addr);
    }

    #[test]
    fn test_rejects_invalid_multiaddr() {
        let err = "127.0.0.1:9222".parse::<StaticPeer>().unwrap_err();
        assert!(matches!(err, StaticPeerError::InvalidMultiaddr(_)), "{err:?}");
    }

    #[test]
    fn test_rejects_invalid_peer_id() {
        // Not base58 at all, and base58 that does not decode to a multihash.
        for peer_id in ["not-a-peer-id", "abc"] {
            let err =
                format!("/ip4/127.0.0.1/tcp/9222/p2p/{peer_id}").parse::<StaticPeer>().unwrap_err();
            assert!(matches!(err, StaticPeerError::InvalidMultiaddr(_)), "{peer_id}: {err:?}");
        }
    }

    #[test]
    fn test_rejects_address_without_peer_id() {
        let err = "/ip4/127.0.0.1/tcp/9222".parse::<StaticPeer>().unwrap_err();
        assert!(matches!(err, StaticPeerError::MissingPeerId), "{err:?}");
    }

    #[test]
    fn test_rejects_peer_id_not_at_the_end() {
        let err =
            format!("/p2p/{PEER_ID}/ip4/127.0.0.1/tcp/9222").parse::<StaticPeer>().unwrap_err();
        assert!(matches!(err, StaticPeerError::MissingPeerId), "{err:?}");
    }

    #[test]
    fn test_rejects_peer_id_without_address() {
        let err = format!("/p2p/{PEER_ID}").parse::<StaticPeer>().unwrap_err();
        assert!(matches!(err, StaticPeerError::MissingAddress), "{err:?}");
    }
}
