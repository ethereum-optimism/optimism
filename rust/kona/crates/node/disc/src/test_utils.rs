//! Helpers shared by the crate's tests.

use std::net::{IpAddr, Ipv4Addr};

use discv5::Enr;
use enr::CombinedKey;

use crate::LocalNode;

/// A record of a new node on `ip`, for `chain_id`.
pub(crate) fn enr_at(ip: IpAddr, chain_id: u64) -> Enr {
    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };
    LocalNode::new(secret_key, ip, 9222, 9223).build_enr(chain_id).expect("a node's own ENR builds")
}

/// A record of a new node on `127.0.0.1`, for `chain_id`.
pub(crate) fn localhost_enr(chain_id: u64) -> Enr {
    enr_at(IpAddr::V4(Ipv4Addr::LOCALHOST), chain_id)
}
