//! Building and sealing a block in one operation.

use super::{BuildSealCoupling, Engine};
use crate::engine::{BuildTaskError, SealTaskError};
use kona_protocol::OpAttributesWithParent;

/// Error type for build and seal operations.
#[derive(Debug, thiserror::Error)]
pub(in crate::engine) enum BuildAndSealError {
    /// An error occurred during the build phase.
    #[error(transparent)]
    Build(#[from] BuildTaskError),
    /// An error occurred during the seal phase.
    #[error(transparent)]
    Seal(#[from] SealTaskError),
}

impl Engine {
    /// Builds and seals a payload in sequence, so no other operation can advance the unsafe head
    /// between them.
    ///
    /// Consolidation uses this to build derived attributes that do not match the unsafe chain, and
    /// sealing uses it for the Holocene deposits-only fallback.
    ///
    /// `is_attributes_derived` says whether the attributes were derived or created by the
    /// sequencer.
    pub(in crate::engine) async fn build_and_seal(
        &mut self,
        attributes: OpAttributesWithParent,
        is_attributes_derived: bool,
    ) -> Result<(), BuildAndSealError> {
        let payload_id = self.start_build(attributes.clone()).await?;

        // Sealing can fall back to building and sealing a deposits-only block, so box the
        // recursive call.
        Box::pin(self.seal_block(
            payload_id,
            attributes,
            is_attributes_derived,
            BuildSealCoupling::Atomic,
        ))
        .await?;

        Ok(())
    }
}
