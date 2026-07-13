//! Fork mode: a lazy RPC-backed base state under the engine's copy-on-write overlay.
//!
//! This is the Rust analog of `op-chain-ops/script/forking`. The Go host wraps a remote state
//! source in a fake geth trie/db/reader/diff stack (~600 LOC) so geth's `state.StateDB` can run
//! over a live L1. revm gives us the overlay for free: `CacheDB` already memoizes reads and layers
//! writes, so the entire trie/db/reader machinery collapses to a swappable underlay
//! ([`ForkUnderlay`]) plus a diff write-log ([`ForkDiff`]) folded from each execution's touched set.
//!
//! The underlay is a fixed-typed enum so the fork can be installed at runtime WITHOUT dropping the
//! overlay cache (the direct analog of Go `ForkableState.SubstituteBaseState`): the well-known
//! script/cheatcode/console accounts pre-inserted at spawn survive the `Empty -> Fork` swap.

use std::collections::BTreeMap;

use alloy_primitives::{Address, B256, U256};
use alloy_provider::RootProvider;
use alloy_provider::network::Ethereum;
use revm::database::{AlloyDB, AlloyDBError, EmptyDB, WrapDatabaseAsync};
use revm::database_interface::DatabaseRef;
use revm::primitives::{KECCAK_EMPTY, StorageKey};
use revm::state::{AccountInfo, Bytecode, EvmState};

/// The RPC-backed base state: an `AlloyDB` (nonce/balance/code/storage over an HTTP provider,
/// pinned to a block) bridged from async to the synchronous revm `Database` trait via
/// `WrapDatabaseAsync` (which `block_on`s on a passed-in runtime handle).
pub type ForkDb = WrapDatabaseAsync<AlloyDB<Ethereum, RootProvider<Ethereum>>>;

/// The one-slot swappable underlay beneath the engine's `CacheDB` overlay.
///
/// `Empty` reproduces today's `CacheDB<EmptyDB>` semantics exactly (every uninserted account is
/// non-existent); `Fork` lazily reads through to an L1 archive. The enum keeps `CacheDB`'s type
/// fixed so the underlay can flip `Empty -> Fork` at runtime while the overlay cache is preserved.
pub enum ForkUnderlay {
    Empty(EmptyDB),
    Fork(Box<ForkDb>),
}

impl Default for ForkUnderlay {
    fn default() -> Self {
        ForkUnderlay::Empty(EmptyDB::default())
    }
}

impl DatabaseRef for ForkUnderlay {
    // Unify the arms' error types on AlloyDBError; the Empty arm is infallible.
    type Error = AlloyDBError;

    fn basic_ref(&self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        match self {
            ForkUnderlay::Empty(db) => match db.basic_ref(address) {
                Ok(v) => Ok(v),
                Err(e) => match e {},
            },
            ForkUnderlay::Fork(db) => db.basic_ref(address),
        }
    }

    fn code_by_hash_ref(&self, code_hash: B256) -> Result<Bytecode, Self::Error> {
        match self {
            ForkUnderlay::Empty(db) => match db.code_by_hash_ref(code_hash) {
                Ok(v) => Ok(v),
                Err(e) => match e {},
            },
            ForkUnderlay::Fork(db) => db.code_by_hash_ref(code_hash),
        }
    }

    fn storage_ref(&self, address: Address, index: StorageKey) -> Result<U256, Self::Error> {
        match self {
            ForkUnderlay::Empty(db) => match db.storage_ref(address, index) {
                Ok(v) => Ok(v),
                Err(e) => match e {},
            },
            ForkUnderlay::Fork(db) => db.storage_ref(address, index),
        }
    }

    fn block_hash_ref(&self, number: u64) -> Result<B256, Self::Error> {
        match self {
            ForkUnderlay::Empty(db) => match db.block_hash_ref(number) {
                Ok(v) => Ok(v),
                Err(e) => match e {},
            },
            ForkUnderlay::Fork(db) => db.block_hash_ref(number),
        }
    }
}

impl std::fmt::Debug for ForkUnderlay {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ForkUnderlay::Empty(_) => write!(f, "ForkUnderlay::Empty"),
            ForkUnderlay::Fork(_) => write!(f, "ForkUnderlay::Fork"),
        }
    }
}

/// Metadata about the installed fork, returned to the Go side from `script_createSelectFork`.
#[derive(Debug, Clone, serde::Serialize)]
pub struct ForkMeta {
    #[serde(rename = "blockNumber")]
    pub block_number: u64,
    #[serde(rename = "blockHash")]
    pub block_hash: B256,
    #[serde(rename = "stateRoot")]
    pub state_root: B256,
}

/// Per-account overlay diff, mirroring `forking.AccountDiff`. An entry set to `None` in
/// [`ForkDiff::accounts`] is a deleted account (geth `DeleteAccount` -> JSON `null`).
#[derive(Debug, Clone, Default)]
struct AccountDiff {
    nonce: Option<u64>,
    balance: Option<U256>,
    storage: Option<BTreeMap<B256, B256>>,
    code_hash: Option<B256>,
}

/// The accumulated fork-overlay diff, in the exact `forking.ExportDiff` JSON shape:
/// `{"account": {addr: {nonce,balance,storage,codeHash} | null}, "code": {hash: base64}}`.
///
/// Built as a write-log folded at each execution's finalized touched set (plus direct host
/// mutators) — NOT by scraping `CacheDB`, which cannot distinguish read-cached from written
/// entries. This mirrors Go, where the diff records exactly the committed dirty accounts/slots.
#[derive(Debug, Default)]
pub struct ForkDiff {
    accounts: BTreeMap<Address, Option<AccountDiff>>,
    code: BTreeMap<B256, Vec<u8>>,
}

impl ForkDiff {
    /// Fold one execution's finalized state (revm `EvmState`) into the running diff, replicating
    /// geth's dirty-object flush semantics (`Finalise` + trie `UpdateAccount`/`UpdateStorage`/
    /// `UpdateContractCode`/`DeleteAccount`):
    ///  - only accounts that ACTUALLY CHANGED are flushed. revm's "touched" flag is broader than
    ///    geth's "dirty" (it fires on plain reads, the 0-reward coinbase, etc.), so a touched
    ///    account is kept only when it is created/selfdestructed, has ≥1 changed storage slot, or
    ///    its nonce/balance/codeHash differs from the pre-commit fork-loaded base;
    ///  - selfdestructed accounts are deleted (`null`);
    ///  - a kept account records nonce+balance+codeHash and its changed storage slots;
    ///  - newly-created contracts contribute their code to the code map.
    ///
    /// `excluded` is the persistent/excluded set (deployer, VM, console, script infra): in Go these
    /// route to the fallback state, so their writes never appear in the fork diff. `base` maps each
    /// touched account to its pre-commit `(nonce, balance, codeHash)` (the value loaded from the
    /// fork), so a mere read/touch is distinguishable from a real write; a missing entry means the
    /// account did not exist before (newly created).
    pub fn record_evm_state(
        &mut self,
        state: &EvmState,
        excluded: impl Fn(&Address) -> bool,
        base: &std::collections::HashMap<Address, (u64, U256, B256)>,
    ) {
        for (addr, account) in state.iter() {
            if !account.is_touched() || excluded(addr) {
                continue;
            }
            if account.is_selfdestructed() {
                self.accounts.insert(*addr, None);
                continue;
            }
            let mut changed_storage: BTreeMap<B256, B256> = BTreeMap::new();
            for (k, slot) in account.storage.iter() {
                if slot.is_changed() {
                    changed_storage.insert(B256::from(*k), B256::from(slot.present_value));
                }
            }
            let info_changed = match base.get(addr) {
                Some((n, b, ch)) => {
                    account.info.nonce != *n
                        || account.info.balance != *b
                        || account.info.code_hash != *ch
                }
                // No pre-commit base -> the account is new this run.
                None => true,
            };
            if !account.is_created() && !info_changed && changed_storage.is_empty() {
                continue; // pure read/touch — geth would not record it
            }

            let entry = self.accounts.entry(*addr).or_insert_with(|| Some(AccountDiff::default()));
            let ad = entry.get_or_insert_with(AccountDiff::default);
            ad.nonce = Some(account.info.nonce);
            ad.balance = Some(account.info.balance);
            ad.code_hash = Some(account.info.code_hash);
            if !changed_storage.is_empty() {
                let storage = ad.storage.get_or_insert_with(BTreeMap::new);
                storage.extend(changed_storage);
            }
            if account.is_created() {
                if let Some(code) = &account.info.code {
                    if !code.is_empty() && account.info.code_hash != KECCAK_EMPTY {
                        self.code
                            .insert(account.info.code_hash, code.original_bytes().to_vec());
                    }
                }
            }
        }
    }

    /// Record a direct host-side account write (set_balance / set_nonce / set_code): these bypass
    /// EVM execution, so they are not in any finalized state, but in Go the equivalent cheatcode
    /// writes land in the fork diff. `code` is `Some` only when the code changed.
    pub fn record_account_write(
        &mut self,
        addr: Address,
        nonce: u64,
        balance: U256,
        code_hash: B256,
        code: Option<&Bytecode>,
    ) {
        let entry = self.accounts.entry(addr).or_insert_with(|| Some(AccountDiff::default()));
        let ad = entry.get_or_insert_with(AccountDiff::default);
        ad.nonce = Some(nonce);
        ad.balance = Some(balance);
        ad.code_hash = Some(code_hash);
        if let Some(code) = code {
            if !code.is_empty() && code_hash != KECCAK_EMPTY {
                self.code.insert(code_hash, code.original_bytes().to_vec());
            }
        }
    }

    /// Record a direct host-side storage write (set_storage).
    pub fn record_storage_write(&mut self, addr: Address, key: U256, value: U256) {
        let entry = self.accounts.entry(addr).or_insert_with(|| Some(AccountDiff::default()));
        let ad = entry.get_or_insert_with(AccountDiff::default);
        ad.storage
            .get_or_insert_with(BTreeMap::new)
            .insert(B256::from(key), B256::from(value));
    }

    /// True when the diff has recorded any account or code — the non-vacuity guard for the gate.
    pub fn any(&self) -> bool {
        !self.accounts.is_empty() || !self.code.is_empty()
    }

    /// Serialize into the `forking.ExportDiff` JSON shape (for the A/B parity gate).
    pub fn to_json(&self) -> serde_json::Value {
        let mut account = serde_json::Map::new();
        for (addr, diff) in &self.accounts {
            let key = format!("0x{addr:x}");
            match diff {
                None => {
                    account.insert(key, serde_json::Value::Null);
                }
                Some(ad) => {
                    // Assemble here so the code-hash field name matches Go (`codeHash`).
                    let storage = ad.storage.as_ref().map(|s| {
                        let mut m = serde_json::Map::new();
                        for (k, v) in s {
                            m.insert(
                                format!("0x{k:x}"),
                                serde_json::Value::String(format!("0x{v:x}")),
                            );
                        }
                        serde_json::Value::Object(m)
                    });
                    let val = serde_json::json!({
                        "nonce": ad.nonce,
                        "balance": ad.balance.map(|b| b.to_string()),
                        "storage": storage,
                        "codeHash": ad.code_hash.map(|h| format!("0x{h:x}")),
                    });
                    account.insert(key, val);
                }
            }
        }
        let mut code = serde_json::Map::new();
        for (hash, bytes) in &self.code {
            code.insert(format!("0x{hash:x}"), serde_json::Value::String(base64_std(bytes)));
        }
        serde_json::json!({ "account": account, "code": code })
    }
}

/// Standard base64 (RFC 4648, with padding), matching Go `encoding/json`'s `[]byte` marshaling
/// used for the `code` map values in `forking.ExportDiff`.
fn base64_std(input: &[u8]) -> String {
    const TABLE: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(input.len().div_ceil(3) * 4);
    for chunk in input.chunks(3) {
        let b0 = chunk[0] as u32;
        let b1 = *chunk.get(1).unwrap_or(&0) as u32;
        let b2 = *chunk.get(2).unwrap_or(&0) as u32;
        let n = (b0 << 16) | (b1 << 8) | b2;
        out.push(TABLE[((n >> 18) & 63) as usize] as char);
        out.push(TABLE[((n >> 12) & 63) as usize] as char);
        if chunk.len() > 1 {
            out.push(TABLE[((n >> 6) & 63) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(TABLE[(n & 63) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn base64_matches_go_encoding_json() {
        // Go: json.Marshal([]byte{...}) == base64.StdEncoding.
        assert_eq!(base64_std(b""), "");
        assert_eq!(base64_std(b"f"), "Zg==");
        assert_eq!(base64_std(b"fo"), "Zm8=");
        assert_eq!(base64_std(b"foo"), "Zm9v");
        assert_eq!(base64_std(b"foob"), "Zm9vYg==");
        assert_eq!(base64_std(&[0x60, 0x00, 0x54]), "YABU");
    }

    #[test]
    fn empty_diff_is_not_vacuous_guarded() {
        let d = ForkDiff::default();
        assert!(!d.any());
        assert_eq!(d.to_json(), serde_json::json!({"account": {}, "code": {}}));
    }

    #[test]
    fn account_diff_shapes_match_go() {
        let mut d = ForkDiff::default();
        let a = Address::with_last_byte(0xAB);
        d.record_storage_write(a, U256::from(1), U256::from(0x2a));
        d.record_account_write(a, 7, U256::from(1000u64), KECCAK_EMPTY, None);
        let v = d.to_json();
        let acc = &v["account"][format!("0x{a:x}")];
        assert_eq!(acc["nonce"], serde_json::json!(7));
        // decimal string, not hex
        assert_eq!(acc["balance"], serde_json::json!("1000"));
        assert_eq!(
            acc["codeHash"],
            serde_json::json!(format!("0x{:x}", KECCAK_EMPTY))
        );
        assert_eq!(
            acc["storage"]
                [format!("0x{:x}", B256::from(U256::from(1)))],
            serde_json::json!(format!("0x{:x}", B256::from(U256::from(0x2a))))
        );
    }
}
