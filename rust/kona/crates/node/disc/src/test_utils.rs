//! Helpers shared by the crate's tests.

use std::net::{IpAddr, Ipv4Addr};

use alloy_rlp::Encodable;
use discv5::Enr;
use enr::CombinedKey;
use kona_peers::OpStackEnr;

use crate::LocalNode;

/// The chain ID of the offline tests: not in the registry, so a node on it has no bootnodes of
/// its own.
pub(crate) const TEST_CHAIN_ID: u64 = 0xdead_beef;

/// A record of a new node on `ip`, for `chain_id`.
pub(crate) fn enr_at(ip: IpAddr, chain_id: u64) -> Enr {
    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };
    LocalNode::new(secret_key, ip, 9222, 9223).build_enr(chain_id).expect("a node's own ENR builds")
}

/// A record of a new node on `127.0.0.1`, for `chain_id`.
pub(crate) fn enr_at_localhost(chain_id: u64) -> Enr {
    enr_at(IpAddr::V4(Ipv4Addr::LOCALHOST), chain_id)
}

/// A record of a new node for `chain_id` with the addresses and ports `build` sets, which a
/// node's own record, built by [`LocalNode`], can't have.
pub(crate) fn enr_with(chain_id: u64, build: impl FnOnce(&mut enr::Builder<CombinedKey>)) -> Enr {
    enr_signed_with(&CombinedKey::generate_secp256k1(), chain_id, build)
}

/// Like [`enr_with`], signed with `key`, for a test that signs another record of the node, or
/// runs it.
pub(crate) fn enr_signed_with(
    key: &CombinedKey,
    chain_id: u64,
    build: impl FnOnce(&mut enr::Builder<CombinedKey>),
) -> Enr {
    let mut opstack = Vec::new();
    OpStackEnr::from_chain_id(chain_id).encode(&mut opstack);
    let mut builder = Enr::builder();
    builder.add_value_rlp(OpStackEnr::OP_CL_KEY, opstack.into());
    build(&mut builder);
    builder.build(key).expect("the record builds")
}
