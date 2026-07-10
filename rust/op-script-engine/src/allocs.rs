//! `ForgeAllocs` state-dump representation in the exact "forge hex" JSON shape that
//! `op-chain-ops/foundry.ForgeAllocs.UnmarshalJSON` accepts, so the Go parity harness can
//! unmarshal a Rust dump straight into its own type.

use std::collections::BTreeMap;

use alloy_primitives::{Address, B256, U256, keccak256};
use serde::Serialize;

/// One account in a forge allocs dump.
#[derive(Debug, Clone, Serialize)]
pub struct AllocAccount {
    /// hexutil.U256, e.g. "0x0".
    pub balance: String,
    /// hexutil.Uint64, e.g. "0x1".
    pub nonce: String,
    /// hexutil.Bytes, omitted when empty (matches Go `code,omitempty`).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub code: Option<String>,
    /// storage slot -> value, both full 32-byte hex; omitted when empty.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub storage: Option<BTreeMap<String, String>>,
}

/// Map of address -> account, keyed by lowercase 0x-hex address string.
pub type ForgeAllocs = BTreeMap<String, AllocAccount>;

pub fn fmt_addr(a: &Address) -> String {
    format!("0x{:x}", a)
}

pub fn fmt_u256(v: &U256) -> String {
    // hexutil.U256 marshals zero as "0x0" and strips leading zeros.
    format!("0x{:x}", v)
}

pub fn fmt_u64(v: u64) -> String {
    format!("0x{:x}", v)
}

pub fn fmt_hash(h: &B256) -> String {
    format!("0x{:x}", h)
}

pub fn fmt_code(code: &[u8]) -> String {
    format!("0x{}", alloy_primitives::hex::encode(code))
}

/// CREATE address = keccak256(rlp([sender, nonce]))[12..], mirroring
/// `crypto.CreateAddress`. Used to prune the script-deployer nonce range from dumps.
pub fn create_address(sender: &Address, nonce: u64) -> Address {
    let mut nonce_rlp = Vec::new();
    if nonce == 0 {
        nonce_rlp.push(0x80u8);
    } else if nonce < 0x80 {
        nonce_rlp.push(nonce as u8);
    } else {
        let be = nonce.to_be_bytes();
        let start = be.iter().position(|&b| b != 0).unwrap();
        let sig = &be[start..];
        nonce_rlp.push(0x80 + sig.len() as u8);
        nonce_rlp.extend_from_slice(sig);
    }

    let mut payload = Vec::with_capacity(1 + 20 + nonce_rlp.len());
    payload.push(0x80 + 20); // address string header (20 < 56)
    payload.extend_from_slice(sender.as_slice());
    payload.extend_from_slice(&nonce_rlp);

    let mut out = Vec::with_capacity(1 + payload.len());
    // payload length is always < 56 here
    out.push(0xc0 + payload.len() as u8);
    out.extend_from_slice(&payload);

    let hash = keccak256(&out);
    Address::from_slice(&hash[12..])
}
