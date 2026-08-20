//! Contains various traversal stages for kona's derivation pipeline.
//!
//! The traversal stage sits at the bottom of the pipeline, and is responsible for
//! providing the next block to the next stage in the pipeline.
//!
//! ## Types
//!
//! - [`PollingTraversal`]: An active traversal stage that polls for the next block through its
//!   provider.

use alloc::sync::Arc;
use alloy_consensus::Receipt;
use alloy_primitives::Address;
use kona_genesis::SystemConfig;

mod polling;
pub use polling::PollingTraversal;

/// Updates the system config with receipts, logging each applied update and any errors,
/// and setting the appropriate metrics gauges.
///
/// `_chain_id_label` is the caller's cached `chain_id` metric label value; this stage has no
/// [`RollupConfig`][kona_genesis::RollupConfig] of its own to derive it from.
fn update_system_config_with_receipts(
    system_config: &mut SystemConfig,
    receipts: &[Receipt],
    l1_system_config_address: Address,
    ecotone_active: bool,
    block_number: u64,
    _chain_id_label: &Arc<str>,
) {
    let (updates, errors) =
        system_config.update_with_receipts(receipts, l1_system_config_address, ecotone_active);
    for kind in &updates {
        info!(target: "traversal", %kind, block_number, "Applied system config update");
    }
    if !updates.is_empty() {
        kona_macros::set!(
            gauge,
            crate::Metrics::PIPELINE_LATEST_SYS_CONFIG_UPDATE,
            block_number as f64,
            crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
        );
    }
    for err in &errors {
        warn!(target: "traversal", ?err, "Malformed system config update at block {block_number} (skipped)");
        kona_macros::set!(
            gauge,
            crate::Metrics::PIPELINE_SYS_CONFIG_UPDATE_ERROR,
            block_number as f64,
            crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
        );
    }
}
