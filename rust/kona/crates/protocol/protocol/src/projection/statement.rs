//! The 672-byte `PublicValuesV1` statement and the canonical config hashes.
//! Twin of Go `op-private-interop/projection/statement.go`.

use alloc::vec::Vec;
use alloy_primitives::{B256, U256, b256, keccak256};
use kona_genesis::RollupConfig;
use sha2::{Digest, Sha256};

use super::{ProjectionError, Statement};

/// `keccak256("optimism.private-projection.public-values.v1")`.
pub const PUBLIC_VALUES_MAGIC: B256 =
    b256!("0x1c413426a9c9aeca4f7a3c2f874ab46e075130f1599c378781288677018d5e18");
/// Domain of [`config_hash`].
pub const CONFIG_DOMAIN: &[u8] = b"optimism.private-projection-config.v1\0";
/// Domain of [`private_config_hash`].
pub const PRIVATE_CONFIG_DOMAIN: &[u8] = b"optimism.private-config.v1\0";
/// Domain of [`dependency_set_hash`].
pub const DEP_SET_DOMAIN: &[u8] = b"optimism.private-dependency-set.v1\0";
/// Number of 32-byte words in `PublicValuesV1`.
pub const PUBLIC_VALUES_WORDS: usize = 21;
/// Byte length of `PublicValuesV1`.
pub const PUBLIC_VALUES_LEN: usize = PUBLIC_VALUES_WORDS * 32;

/// BN254 scalar field modulus `r`, big-endian.
pub const BN254_SCALAR_MODULUS: [u8; 32] = [
    0x30, 0x64, 0x4e, 0x72, 0xe1, 0x31, 0xa0, 0x29, 0xb8, 0x50, 0x45, 0xb6, 0x81, 0x81, 0x58, 0x5d,
    0x28, 0x33, 0xe8, 0x48, 0x79, 0xb9, 0x70, 0x91, 0x43, 0xe1, 0xf5, 0x93, 0xf0, 0x00, 0x00, 0x01,
];

/// `true` iff the big-endian word is a canonical BN254 scalar (`< r`).
pub fn is_canonical_scalar(word: &[u8; 32]) -> bool {
    word.as_slice() < BN254_SCALAR_MODULUS.as_slice()
}

fn u64_word(n: u64) -> B256 {
    B256::from(U256::from(n))
}

/// The consensus public values of a statement: 21 big-endian 32-byte words (§C.2).
pub fn public_values(s: &Statement) -> [u8; PUBLIC_VALUES_LEN] {
    let c = &s.claim;
    let words: [B256; PUBLIC_VALUES_WORDS] = [
        PUBLIC_VALUES_MAGIC,
        s.chain_id,
        s.projection_config_hash,
        s.private_config_hash,
        c.depSetHash,
        s.parent_hash,
        u64_word(s.continuation.anchor.number),
        s.continuation.anchor.hash,
        s.continuation.output_root,
        s.continuation.recovery_hash,
        u64_word(c.firstBlock),
        u64_word(c.lastBlock),
        c.parentOutputRoot,
        c.privateTerminalBlockHash,
        c.privateTerminalParentHash,
        c.l1Head,
        c.privateDataHash,
        s.projection_hash,
        s.outputs_root,
        s.messages_root,
        s.terminal_output,
    ];
    let mut out = [0u8; PUBLIC_VALUES_LEN];
    for (chunk, word) in out.chunks_exact_mut(32).zip(words.iter()) {
        chunk.copy_from_slice(word.as_slice());
    }
    out
}

/// SP1 committed-values digest: `sha256(public_values)` with the top three bits cleared.
pub fn public_values_digest(public_values: &[u8]) -> B256 {
    let mut digest: [u8; 32] = Sha256::digest(public_values).into();
    digest[0] &= 0x1f;
    B256::from(digest)
}

/// Canonical hash of the projection rollup config (the claim's `rollupConfigHash`).
pub fn config_hash(cfg: &RollupConfig) -> Result<B256, ProjectionError> {
    let p = cfg.private_projection.as_ref().ok_or(ProjectionError("missing projection config"))?;
    let mut buf = Vec::with_capacity(CONFIG_DOMAIN.len() + 32 * 7 + 8 * 3 + 2);
    buf.extend_from_slice(CONFIG_DOMAIN);
    buf.extend_from_slice(u64_word(cfg.l2_chain_id.id()).as_slice());
    buf.extend_from_slice(&cfg.genesis.l2.number.to_be_bytes());
    buf.extend_from_slice(cfg.genesis.l2.hash.as_slice());
    buf.extend_from_slice(&cfg.genesis.l2_time.to_be_bytes());
    buf.extend_from_slice(&cfg.block_time.to_be_bytes());
    buf.extend_from_slice(keccak256(p.verifier.as_bytes()).as_slice());
    buf.extend_from_slice(p.genesis_output_root.as_slice());
    buf.extend_from_slice(p.program_vkey.as_slice());
    buf.extend_from_slice(p.private_config_hash.as_slice());
    buf.extend_from_slice(p.dependency_set_hash.as_slice());
    buf.push(u8::from(p.allow_events));
    buf.push(u8::from(p.mock_proofs));
    Ok(keccak256(buf))
}

/// Hash of the exact deployed private rollup config and L1 chain config bytes.
pub fn private_config_hash(private_rollup_json: &[u8], l1_chain_config_json: &[u8]) -> B256 {
    let mut buf = Vec::with_capacity(PRIVATE_CONFIG_DOMAIN.len() + 64);
    buf.extend_from_slice(PRIVATE_CONFIG_DOMAIN);
    buf.extend_from_slice(keccak256(private_rollup_json).as_slice());
    buf.extend_from_slice(keccak256(l1_chain_config_json).as_slice());
    keccak256(buf)
}

/// A chain ID accepted by [`dependency_set_hash`]: `u64` or full-width `U256`.
pub trait DepSetChainId {
    /// The chain ID as a uint256.
    fn to_u256(self) -> U256;
}
impl DepSetChainId for u64 {
    fn to_u256(self) -> U256 {
        U256::from(self)
    }
}
impl DepSetChainId for U256 {
    fn to_u256(self) -> U256 {
        self
    }
}

/// Canonical hash of the distinct chain IDs of a dependency set, sorted ascending.
pub fn dependency_set_hash<T: DepSetChainId>(ids: impl IntoIterator<Item = T>) -> B256 {
    let mut ids: Vec<U256> = ids.into_iter().map(DepSetChainId::to_u256).collect();
    ids.sort_unstable();
    ids.dedup();
    let mut buf = Vec::with_capacity(DEP_SET_DOMAIN.len() + 8 + 32 * ids.len());
    buf.extend_from_slice(DEP_SET_DOMAIN);
    buf.extend_from_slice(&(ids.len() as u64).to_be_bytes());
    for id in &ids {
        buf.extend_from_slice(&id.to_be_bytes::<32>());
    }
    keccak256(buf)
}
