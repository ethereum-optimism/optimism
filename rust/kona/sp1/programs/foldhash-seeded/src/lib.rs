//! SP1 guest builds patch `foldhash` to this crate so that every hash map is keyed by a secret
//! seed the prover draws for each proof. Unseeded, the seed is a constant of the guest ELF, so
//! attacker-chosen keys can be made to collide (`HashDoS`). A guest must call
//! [`install_global_seed`] before creating any hash map; creating one earlier panics.

#![no_std]

use core::{
    cell::UnsafeCell,
    hash::Hasher,
    sync::atomic::{AtomicU8, AtomicU64, Ordering},
};

const UNINIT: u8 = 0;
const LOCKED: u8 = 1;
const INIT: u8 = 2;

static STATE: AtomicU8 = AtomicU8::new(UNINIT);

struct SeedCell(UnsafeCell<upstream::SharedSeed>);

// SAFETY: the seed is written once, by the thread that moves `STATE` from `UNINIT` to `LOCKED`,
// and read only after `STATE` is `INIT`, which is stored with release ordering after the write.
unsafe impl Sync for SeedCell {}

static SEED: SeedCell = SeedCell(UnsafeCell::new(upstream::SharedSeed::from_u64(0)));

// Not 0: a zero per-hasher seed makes keys whose first and third words are zero collide under
// every global seed.
const FIRST_PER_HASHER_COUNTER: u64 = 1;
static PER_HASHER_COUNTER: AtomicU64 = AtomicU64::new(FIRST_PER_HASHER_COUNTER);

/// Installs the global hash seed. Must be called once, before any hash map is created.
///
/// # Panics
///
/// If a seed is already installed.
pub fn install_global_seed(seed: u64) {
    assert!(
        STATE.compare_exchange(UNINIT, LOCKED, Ordering::Acquire, Ordering::Relaxed).is_ok(),
        "foldhash: global hash seed already installed"
    );
    // SAFETY: winning the exchange above makes this the only writer, and readers wait for `INIT`.
    unsafe { *SEED.0.get() = upstream::SharedSeed::from_u64(seed) };
    STATE.store(INIT, Ordering::Release);
}

#[inline]
fn shared_seed() -> &'static upstream::SharedSeed {
    assert!(
        STATE.load(Ordering::Acquire) == INIT,
        "foldhash: global hash seed not installed; the guest must call \
         foldhash::install_global_seed before creating a hash map"
    );
    // SAFETY: `INIT` is stored after the only write, so the seed is immutable from here on.
    unsafe { &*SEED.0.get() }
}

/// A distinct secret seed for each hasher builder. hashbrown goes quadratic when one table is
/// filled in another's iteration order under the same seed.
fn next_per_hasher_seed() -> u64 {
    per_hasher_seed(PER_HASHER_COUNTER.fetch_add(1, Ordering::Relaxed), shared_seed())
}

/// `folded_multiply(n, s0)` for counter value `n`: secret, because it depends on `s0`.
fn per_hasher_seed(n: u64, shared: &upstream::SharedSeed) -> u64 {
    let mut hasher = upstream::fast::FoldHasher::with_seed(0, shared);
    hasher.write_u64(n);
    hasher.finish()
}

/// The foldhash hasher optimized for speed, keyed by the installed seed.
pub mod fast {
    use core::hash::BuildHasher;

    pub use upstream::fast::FoldHasher;

    /// Builds [`FoldHasher`]s keyed by the installed global seed and a per-builder seed.
    #[derive(Clone, Copy, Debug)]
    pub struct RandomState {
        per_hasher_seed: u64,
    }

    impl Default for RandomState {
        fn default() -> Self {
            Self { per_hasher_seed: crate::next_per_hasher_seed() }
        }
    }

    impl BuildHasher for RandomState {
        type Hasher = FoldHasher<'static>;

        #[inline(always)]
        fn build_hasher(&self) -> Self::Hasher {
            FoldHasher::with_seed(self.per_hasher_seed, crate::shared_seed())
        }
    }
}

#[cfg(test)]
mod tests {
    extern crate std;

    use core::hash::{BuildHasher, Hasher};

    use super::{FIRST_PER_HASHER_COUNTER, fast::RandomState, per_hasher_seed};

    fn install_test_seed() {
        static ONCE: std::sync::Once = std::sync::Once::new();
        ONCE.call_once(|| super::install_global_seed(0x5eed));
    }

    /// Hashes a 32-byte key as alloy's `FbBuildHasher<32>` does on a 64-bit target.
    fn hash_words(mut hasher: impl Hasher, words: [u64; 4]) -> u64 {
        for word in words {
            hasher.write_u64(word);
        }
        hasher.finish()
    }

    #[test]
    fn builders_hash_the_same_key_differently() {
        install_test_seed();
        let key = [1, 2, 3, 4];
        assert_ne!(
            hash_words(RandomState::default().build_hasher(), key),
            hash_words(RandomState::default().build_hasher(), key)
        );
    }

    /// A zero per-hasher seed hashes every key whose first and third words are zero to 0.
    #[test]
    fn keys_with_zero_words_do_not_collide() {
        let shared = upstream::SharedSeed::from_u64(0x5eed);
        let seed = per_hasher_seed(FIRST_PER_HASHER_COUNTER, &shared);
        let hasher = || upstream::fast::FoldHasher::with_seed(seed, &shared);
        let first = hash_words(hasher(), [0, 1, 0, 0]);
        let second = hash_words(hasher(), [0, 2, 0, 0]);
        assert_ne!(first, second);
        assert_ne!(first, 0);
    }

    /// A per-hasher seed that the attacker can compute lets keys cancel it in the first word.
    #[test]
    fn per_hasher_seeds_depend_on_the_shared_seed() {
        let (one, two) = (upstream::SharedSeed::from_u64(1), upstream::SharedSeed::from_u64(2));
        for n in FIRST_PER_HASHER_COUNTER..FIRST_PER_HASHER_COUNTER + 4 {
            assert_ne!(per_hasher_seed(n, &one), per_hasher_seed(n, &two));
            assert_ne!(per_hasher_seed(n, &one), n);
        }
    }
}
