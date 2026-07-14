//! `ForgeAllocs` state-dump representation in the exact "forge hex" JSON shape that
//! `op-chain-ops/foundry.ForgeAllocs.UnmarshalJSON` accepts, so the Go parity harness can
//! unmarshal a Rust dump straight into its own type.

use std::collections::BTreeMap;

use alloy_primitives::{Address, B256, U256, keccak256};
use serde::{Deserialize, Serialize};

/// One account in a forge allocs dump.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AllocAccount {
    /// hexutil.U256, e.g. "0x0".
    #[serde(default = "hex_zero")]
    pub balance: String,
    /// hexutil.Uint64, e.g. "0x1".
    #[serde(default = "hex_zero")]
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

fn hex_zero() -> String {
    "0x0".to_string()
}

/// Formats an address as a lowercase `0x`-hex string, matching Go's `common.Address` JSON.
pub fn fmt_addr(a: &Address) -> String {
    format!("0x{:x}", a)
}

/// Formats a 256-bit integer the way Go's `hexutil.U256` does: `0x`-prefixed, leading zeros
/// stripped, zero rendered as `0x0`.
pub fn fmt_u256(v: &U256) -> String {
    format!("0x{:x}", v)
}

/// Formats a `u64` nonce/balance as a `0x`-hex string, matching Go's `hexutil.Uint64`.
pub fn fmt_u64(v: u64) -> String {
    format!("0x{:x}", v)
}

/// Formats a 32-byte hash as a lowercase `0x`-hex string, matching Go's `common.Hash` JSON.
pub fn fmt_hash(h: &B256) -> String {
    format!("0x{:x}", h)
}

/// Formats contract code as a `0x`-hex string, matching Go's `hexutil.Bytes`.
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

/// CREATE2 address = keccak256(0xff ++ deployer ++ salt ++ `keccak256(init_code)`)[12..],
/// mirroring `crypto.CreateAddress2`.
pub fn create2_address(deployer: &Address, salt: &B256, init_code: &[u8]) -> Address {
    create2_address_from_hash(deployer, salt, &keccak256(init_code))
}

/// CREATE2 address from a pre-computed init-code hash = keccak256(0xff ++ deployer ++ salt ++
/// `init_hash`)[12..]. Mirrors `crypto.CreateAddress2` and the `vm.computeCreate2Address`
/// cheatcode.
pub fn create2_address_from_hash(deployer: &Address, salt: &B256, init_hash: &B256) -> Address {
    let mut buf = Vec::with_capacity(1 + 20 + 32 + 32);
    buf.push(0xff);
    buf.extend_from_slice(deployer.as_slice());
    buf.extend_from_slice(salt.as_slice());
    buf.extend_from_slice(init_hash.as_slice());
    let hash = keccak256(&buf);
    Address::from_slice(&hash[12..])
}
