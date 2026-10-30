//! Lagoon-at-genesis fixtures shared by the SDM integration tests.

use alloy_genesis::Genesis;
use alloy_primitives::B64;
use op_alloy_consensus::encode_jovian_extra_data;
use reth_chainspec::BaseFeeParams;
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_node::utils::optimism_payload_attributes;
use reth_optimism_payload_builder::OpPayloadAttrs;
use std::sync::Arc;

/// Builds a Lagoon-at-genesis spec that funds the e2e test wallet.
pub(crate) fn chain_spec() -> eyre::Result<Arc<OpChainSpec>> {
    let mut genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    // Lagoon is active at genesis, so the first child's base fee uses Jovian extra data.
    genesis.extra_data = encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism(), 0)?;
    genesis.base_fee_per_gas = Some(1_000_000_000);
    Ok(Arc::new(OpChainSpecBuilder::optimism_sepolia().genesis(genesis).lagoon_activated().build()))
}

/// Payload attributes carrying the Jovian fields a Lagoon block requires.
pub(crate) const fn payload_attributes(timestamp: u64) -> OpPayloadAttrs {
    let mut attributes = optimism_payload_attributes(timestamp);
    attributes.0.eip_1559_params = Some(B64::ZERO);
    attributes.0.min_base_fee = Some(0);
    attributes
}
