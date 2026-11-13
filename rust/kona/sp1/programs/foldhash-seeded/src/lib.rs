//! SP1 guest builds patch `foldhash` to this crate so that every hash map is keyed by a seed that
//! the guest takes from a hash in its input. Only on-chain actors must be unable to predict or
//! choose the seed before their keys are fixed; the prover may know it. Unseeded, the seed is a
//! constant of the guest ELF, so attacker-chosen keys can be made to collide. A guest that creates
//! a hash map must call [`install_global_seed`] first; creating one earlier panics.
//!
//! Regression check: `just check-hashdos-zkvm` (`programs/tests/hashdos-check`).

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

// A random nonzero salt. Unsalted, the first per-hasher seeds would be 0, `s0` and rotations of
// `s0`, which make some keys collide whatever the global seed.
const PER_HASHER_SALT: u64 = 0x812f_5a88_c308_1b20;
static PER_HASHER_COUNTER: AtomicU64 = AtomicU64::new(0);

/// Installs the global hash seed (see the crate doc for its source). Must be called once, before
/// any hash map is created.
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

/// A distinct unpredictable seed for each hasher builder. hashbrown goes quadratic when one table
/// is filled in another's iteration order under the same seed.
fn next_per_hasher_seed(shared: &upstream::SharedSeed) -> u64 {
    per_hasher_seed(PER_HASHER_COUNTER.fetch_add(1, Ordering::Relaxed), shared)
}

/// `folded_multiply(n ^ PER_HASHER_SALT, s0)` for counter value `n`: unpredictable, because it
/// depends on `s0`.
fn per_hasher_seed(n: u64, shared: &upstream::SharedSeed) -> u64 {
    let mut hasher = upstream::fast::FoldHasher::with_seed(PER_HASHER_SALT, shared);
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
        shared_seed: &'static upstream::SharedSeed,
    }

    impl Default for RandomState {
        fn default() -> Self {
            let shared_seed = crate::shared_seed();
            Self { per_hasher_seed: crate::next_per_hasher_seed(shared_seed), shared_seed }
        }
    }

    impl BuildHasher for RandomState {
        type Hasher = FoldHasher<'static>;

        #[inline(always)]
        fn build_hasher(&self) -> Self::Hasher {
            FoldHasher::with_seed(self.per_hasher_seed, self.shared_seed)
        }
    }
}

#[cfg(test)]
mod tests {
    extern crate std;

    use core::hash::{BuildHasher, Hasher};

    use super::{fast::RandomState, per_hasher_seed};

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

    /// Hashes `words` as the first builder does under the shared seed `0x5eed`.
    fn first_builder_hash(words: [u64; 4]) -> u64 {
        let shared = upstream::SharedSeed::from_u64(0x5eed);
        hash_words(
            upstream::fast::FoldHasher::with_seed(per_hasher_seed(0, &shared), &shared),
            words,
        )
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

    /// Pins that the first builder does not hash zero-word keys to 0.
    #[test]
    fn keys_with_zero_words_do_not_collide() {
        let first = first_builder_hash([0, 1, 0, 0]);
        assert_ne!(first, first_builder_hash([0, 2, 0, 0]));
        assert_ne!(first, 0);
    }

    /// Pins that the first builder does not hash keys with their first two words swapped alike.
    #[test]
    fn keys_with_swapped_words_do_not_collide() {
        assert_ne!(first_builder_hash([1, 2, 7, 9]), first_builder_hash([2, 1, 7, 9]));
    }

    /// Pins that per-hasher seeds change with the shared seed and differ from the counter.
    #[test]
    fn per_hasher_seeds_depend_on_the_shared_seed() {
        let (one, two) = (upstream::SharedSeed::from_u64(1), upstream::SharedSeed::from_u64(2));
        for n in 0..4 {
            assert_ne!(per_hasher_seed(n, &one), per_hasher_seed(n, &two));
            assert_ne!(per_hasher_seed(n, &one), n);
        }
    }
}
