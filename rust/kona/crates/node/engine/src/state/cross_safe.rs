//! The cross-safe head's single promotion entry point.

use kona_protocol::L2BlockInfo;

/// A promotion of the cross-safe head to a specific L2 block.
///
/// The cross-safe head is what the engine reports as `safeBlockHash` in every forkchoice update.
/// It is deliberately *not* a field of [`EngineSyncStateUpdate`]: the ordinary head writers
/// (payload insert, consolidation, engine reset) advance the local-safe head and cannot express a
/// cross-safe move at all. The only way to *advance* it is
/// [`EngineSyncState::apply_cross_safe_promotion`], which requires a [`CrossSafePromotion`] — and
/// that has no public constructor. Those writers can only drag it downwards, by rewinding the
/// local-safe head it is held at. Outside this crate it can only be minted by the holder of an
/// engine's unique [`CrossSafePromoter`], so an unverified promotion does not compile.
///
/// [`EngineSyncStateUpdate`]: crate::EngineSyncStateUpdate
/// [`EngineSyncState::apply_cross_safe_promotion`]: crate::EngineSyncState::apply_cross_safe_promotion
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct CrossSafePromotion {
    /// The block the cross-safe head is promoted to.
    target: L2BlockInfo,
}

impl CrossSafePromotion {
    /// Creates a promotion of the cross-safe head to `target`.
    pub(crate) const fn new(target: L2BlockInfo) -> Self {
        Self { target }
    }

    /// Returns the block the cross-safe head is promoted to.
    pub const fn target(&self) -> L2BlockInfo {
        self.target
    }
}

/// The capability to mint [`CrossSafePromotion`]s for one engine.
///
/// It has no public constructor and is neither [`Clone`] nor [`Copy`], so at most one exists per
/// engine and whoever owns it is by construction the only writer of that engine's cross-safe
/// head. Under interop that owner is the cross-chain verifier; standalone kona-node never mints
/// one and uses the [`CrossSafeSource::LocalSafe`] feed instead.
///
/// Obtained from [`Engine::with_external_cross_safe`].
///
/// [`Engine::with_external_cross_safe`]: crate::Engine::with_external_cross_safe
#[derive(Debug)]
pub struct CrossSafePromoter {
    /// Blocks construction outside this crate.
    _seal: (),
}

impl CrossSafePromoter {
    /// Creates the promoter. Callable only from [`Engine::with_external_cross_safe`], which hands
    /// out exactly one per engine.
    ///
    /// [`Engine::with_external_cross_safe`]: crate::Engine::with_external_cross_safe
    pub(crate) const fn new() -> Self {
        Self { _seal: () }
    }

    /// Mints a promotion of the cross-safe head to `target`.
    pub const fn promote(&self, target: L2BlockInfo) -> CrossSafePromotion {
        CrossSafePromotion::new(target)
    }
}

/// Where an engine's cross-safe promotions come from.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub enum CrossSafeSource {
    /// Standalone kona-node: there is no cross-chain verifier, so every local-safe advance is
    /// trivially cross-safe. The promotion is minted as part of the same state application, so
    /// the forkchoice update that reports the advance still carries local-safe == cross-safe in
    /// a single call.
    #[default]
    LocalSafe,
    /// Interop: the cross-safe head moves only on promotions minted by the engine's
    /// [`CrossSafePromoter`]. A local-safe advance on its own leaves the cross-safe head where it
    /// is, so absence of promotion naturally holds the previous value.
    Promoted,
}

impl CrossSafeSource {
    /// Returns the trivial promotion implied by advancing the local-safe head to `local_safe`, or
    /// [`None`] when this engine's cross-safe head is fed externally.
    pub(crate) const fn trivial_promotion(
        &self,
        local_safe: L2BlockInfo,
    ) -> Option<CrossSafePromotion> {
        match self {
            Self::LocalSafe => Some(CrossSafePromotion::new(local_safe)),
            Self::Promoted => None,
        }
    }
}
