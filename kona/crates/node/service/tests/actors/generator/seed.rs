use std::sync::atomic::{AtomicU64, Ordering};

use rand::SeedableRng;

pub(crate) static SEED_GENERATOR_BUILDER: SeedGeneratorBuilder = SeedGeneratorBuilder::new();

pub(crate) struct SeedGeneratorBuilder(AtomicU64);

impl SeedGeneratorBuilder {
    pub(crate) const fn new() -> Self {
        Self(AtomicU64::new(0))
    }

    fn next(&self) -> u64 {
        self.0.fetch_add(1, Ordering::SeqCst)
    }

    pub(crate) fn next_generator(&self) -> SeedGenerator {
        SeedGenerator(rand::rngs::StdRng::seed_from_u64(self.next()))
    }
}

pub(crate) struct SeedGenerator(pub(super) rand::rngs::StdRng);

impl SeedGenerator {
    pub(crate) const fn as_rng(&mut self) -> &mut rand::rngs::StdRng {
        &mut self.0
    }

    pub(crate) fn random_bytes(&mut self, len: usize) -> Vec<u8> {
        let mut data = vec![0; len];
        rand::Rng::fill(self.as_rng(), &mut data[..]);
        data
    }
}
