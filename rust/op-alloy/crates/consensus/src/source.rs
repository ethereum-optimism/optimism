//! Deposit transaction source hashing.
//!
//! Deposit transactions are identified by a `source_hash` derived from a domain identifier and
//! domain-specific data. The shared pattern is:
//!
//! ```text
//! source_hash = keccak256(pad32(domain_id) || keccak256(inner_data))
//! ```
//!
//! [`DepositSourceHasher`] is an incremental builder for computing these hashes without heap
//! allocation. Typed helper functions ([`upgrade_source_hash`], [`user_deposit_source_hash`], etc.)
//! provide ergonomic shortcuts for each domain.

use alloy_primitives::{B256, Keccak256, keccak256};

/// Source domain identifiers for deposit transactions.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Copy)]
#[repr(u8)]
pub enum DepositSourceDomainIdentifier {
    /// A user deposit source.
    User = 0,
    /// A L1 info deposit source.
    L1Info = 1,
    /// An upgrade deposit source.
    Upgrade = 2,
    /// An interop block replacement source.
    InteropBlockReplacement = 4,
}

mod sealed {
    #[allow(unnameable_types)]
    pub trait Sealed {}

    macro_rules! impl_sealed {
        ($($t:ty),*) => { $( impl Sealed for $t {} )* };
    }

    impl_sealed!(u8, u16, u32, u64, u128, i8, i16, i32, i64, i128);
}

/// Trait for integer types that can be hashed as big-endian bytes.
///
/// Each type hashes at its native width (e.g., `u32` hashes 4 bytes, `u64` hashes 8 bytes).
/// This is a sealed trait — only standard integer types implement it.
///
/// `usize`/`isize` are intentionally excluded because their width is platform-dependent,
/// which would produce different hashes on different architectures.
pub trait IntoBigEndianBytes: sealed::Sealed {
    /// Write the big-endian representation of this integer into the hasher.
    fn hash_into(self, hasher: &mut Keccak256);
}

macro_rules! impl_into_big_endian_bytes {
    ($($t:ty),*) => {
        $(
            impl IntoBigEndianBytes for $t {
                fn hash_into(self, hasher: &mut Keccak256) {
                    hasher.update(&self.to_be_bytes());
                }
            }
        )*
    };
}

impl_into_big_endian_bytes!(u8, u16, u32, u64, u128, i8, i16, i32, i64, i128);

/// Incremental builder for computing deposit source hashes.
///
/// Wraps an incremental keccak hasher and a domain identifier. Feed data in with the `hash_*`
/// methods, then call [`finish`](Self::finish) to produce the final `source_hash`.
///
/// # Example
///
/// ```
/// use op_alloy_consensus::{DepositSourceDomainIdentifier, deposit_source_hash};
///
/// let hash = deposit_source_hash(DepositSourceDomainIdentifier::Upgrade)
///     .hash_str("Ecotone: L1 Block Deployment")
///     .finish();
/// ```
pub struct DepositSourceHasher {
    domain: DepositSourceDomainIdentifier,
    inner: Keccak256,
}

impl core::fmt::Debug for DepositSourceHasher {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("DepositSourceHasher").field("domain", &self.domain).finish_non_exhaustive()
    }
}

impl DepositSourceHasher {
    /// Creates a new hasher for the given domain.
    pub fn new(domain: DepositSourceDomainIdentifier) -> Self {
        Self { domain, inner: Keccak256::new() }
    }

    /// Feed raw bytes into the inner hash.
    pub fn hash_bytes(mut self, data: &[u8]) -> Self {
        self.inner.update(data);
        self
    }

    /// Feed a string's UTF-8 bytes into the inner hash.
    pub fn hash_str(self, data: &str) -> Self {
        self.hash_bytes(data.as_bytes())
    }

    /// Feed a 32-byte value into the inner hash.
    pub fn hash_b256(self, data: B256) -> Self {
        self.hash_bytes(data.as_ref())
    }

    /// Feed an integer's big-endian bytes into the inner hash.
    ///
    /// The integer is hashed at its native width — `u32` produces 4 bytes, `u64` produces 8.
    pub fn hash_int(mut self, value: impl IntoBigEndianBytes) -> Self {
        value.hash_into(&mut self.inner);
        self
    }

    /// Finalize the inner hash and compute the source hash.
    ///
    /// ```text
    /// source_hash = keccak256(pad32(domain_id) || keccak256(accumulated_inner_data))
    /// ```
    pub fn finish(self) -> B256 {
        let inner_hash = self.inner.finalize();
        domain_hash(self.domain, inner_hash)
    }
}

/// Creates a new [`DepositSourceHasher`] for the given domain.
pub fn deposit_source_hash(domain: DepositSourceDomainIdentifier) -> DepositSourceHasher {
    DepositSourceHasher::new(domain)
}

/// Compute a source hash from a domain identifier and a pre-computed inner value.
///
/// Use this when the inner 32 bytes are already known and should NOT be hashed again. This is
/// the case for [`DepositSourceDomainIdentifier::InteropBlockReplacement`], where the inner
/// value is the output root passed through directly.
///
/// ```text
/// source_hash = keccak256(pad32(domain_id) || inner)
/// ```
pub fn deposit_source_hash_raw(domain: DepositSourceDomainIdentifier, inner: B256) -> B256 {
    domain_hash(domain, inner)
}

/// Shared domain hash computation.
fn domain_hash(domain: DepositSourceDomainIdentifier, inner: B256) -> B256 {
    let mut buf = [0u8; 64];
    buf[24..32].copy_from_slice(&(domain as u64).to_be_bytes());
    buf[32..].copy_from_slice(inner.as_ref());
    keccak256(buf)
}

/// Compute the source hash for an upgrade deposit transaction.
///
/// This is the primary API for the 31+ hardfork upgrade transactions that use string literal
/// intents.
pub fn upgrade_source_hash(intent: &str) -> B256 {
    deposit_source_hash(DepositSourceDomainIdentifier::Upgrade).hash_str(intent).finish()
}

/// Compute the source hash for an upgrade deposit from its component parts.
///
/// Equivalent to `upgrade_source_hash(&format!("{fork} {index}: {intent}"))` but without the
/// heap allocation — the index is formatted as decimal ASCII on the stack.
pub fn upgrade_source_hash_parts(fork: &str, index: u64, intent: &str) -> B256 {
    let mut buf = [0u8; 20]; // u64::MAX is 20 decimal digits
    let ascii = format_u64_decimal(index, &mut buf);
    deposit_source_hash(DepositSourceDomainIdentifier::Upgrade)
        .hash_str(fork)
        .hash_str(" ")
        .hash_bytes(ascii)
        .hash_str(": ")
        .hash_str(intent)
        .finish()
}

/// Format a u64 as decimal ASCII into a stack buffer, returning the written slice.
fn format_u64_decimal(mut n: u64, buf: &mut [u8; 20]) -> &[u8] {
    if n == 0 {
        buf[19] = b'0';
        return &buf[19..];
    }
    let mut i = 20;
    while n > 0 {
        i -= 1;
        buf[i] = b'0' + (n % 10) as u8;
        n /= 10;
    }
    &buf[i..]
}

/// Compute the source hash for a user deposit transaction.
pub fn user_deposit_source_hash(l1_block_hash: B256, log_index: u64) -> B256 {
    let mut input = [0u8; 64];
    input[..32].copy_from_slice(l1_block_hash.as_ref());
    input[56..].copy_from_slice(&log_index.to_be_bytes());
    let inner = keccak256(input);
    domain_hash(DepositSourceDomainIdentifier::User, inner)
}

/// Compute the source hash for an L1 info deposit transaction.
pub fn l1_info_deposit_source_hash(l1_block_hash: B256, seq_number: u64) -> B256 {
    let mut input = [0u8; 64];
    input[..32].copy_from_slice(l1_block_hash.as_ref());
    input[56..].copy_from_slice(&seq_number.to_be_bytes());
    let inner = keccak256(input);
    domain_hash(DepositSourceDomainIdentifier::L1Info, inner)
}

/// Compute the source hash for an interop block replacement deposit transaction.
///
/// Unlike other deposit sources, the output root is used directly as the inner value
/// without an additional keccak hash.
pub fn interop_block_replacement_source_hash(output_root: B256) -> B256 {
    deposit_source_hash_raw(DepositSourceDomainIdentifier::InteropBlockReplacement, output_root)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::hex;

    #[test]
    fn upgrade_source_hash_ecotone() {
        assert_eq!(
            upgrade_source_hash("Ecotone: L1 Block Deployment"),
            hex!("877a6077205782ea15a6dc8699fa5ebcec5e0f4389f09cb8eda09488231346f8"),
        );
    }

    #[test]
    fn user_deposit_source_hash_roundtrip() {
        let block_hash = B256::repeat_byte(0xaa);
        let log_index = 42u64;
        let hash = user_deposit_source_hash(block_hash, log_index);
        assert_ne!(hash, B256::ZERO);
    }

    #[test]
    fn l1_info_deposit_source_hash_roundtrip() {
        let block_hash = B256::repeat_byte(0xbb);
        let seq_number = 7u64;
        let hash = l1_info_deposit_source_hash(block_hash, seq_number);
        assert_ne!(hash, B256::ZERO);
    }

    #[test]
    fn interop_block_replacement_no_inner_hash() {
        let output_root = B256::repeat_byte(0xcc);
        let hash = interop_block_replacement_source_hash(output_root);
        let expected = deposit_source_hash_raw(
            DepositSourceDomainIdentifier::InteropBlockReplacement,
            output_root,
        );
        assert_eq!(hash, expected);
    }

    #[test]
    fn upgrade_source_hash_parts_matches_format() {
        let formatted = alloc::format!("{} {}: {}", "Ecotone", 3, "L1 Block Deployment");
        assert_eq!(
            upgrade_source_hash_parts("Ecotone", 3, "L1 Block Deployment"),
            upgrade_source_hash(&formatted),
        );
    }

    #[test]
    fn upgrade_source_hash_parts_zero_index() {
        let formatted = alloc::format!("{} {}: {}", "Fjord", 0, "Deploy");
        assert_eq!(
            upgrade_source_hash_parts("Fjord", 0, "Deploy"),
            upgrade_source_hash(&formatted),
        );
    }

    #[test]
    fn format_u64_decimal_edge_cases() {
        let check = |n: u64| {
            let expected = alloc::format!("{n}");
            let mut buf = [0u8; 20];
            let result = super::format_u64_decimal(n, &mut buf);
            assert_eq!(
                core::str::from_utf8(result).unwrap(),
                expected,
                "format_u64_decimal({n}) failed"
            );
        };

        check(0);
        check(u64::MAX);

        // Cover every output length boundary: 9/10, 99/100, ..., up to 20 digits
        let mut power = 1u64;
        for _ in 0..19 {
            check(power - 1);
            check(power);
            power = power.saturating_mul(10);
        }
    }

    #[test]
    fn hash_int_native_width() {
        let h32 =
            deposit_source_hash(DepositSourceDomainIdentifier::Upgrade).hash_int(42u32).finish();
        let h64 =
            deposit_source_hash(DepositSourceDomainIdentifier::Upgrade).hash_int(42u64).finish();
        assert_ne!(h32, h64, "u32 and u64 should hash different widths");
    }
}
