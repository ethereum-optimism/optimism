//! L1 info deposit transaction construction.
//!
//! Delegates to `kona_protocol::L1BlockInfoTx::try_new_with_deposit_tx` which handles
//! all hardfork variants (Bedrock, Ecotone, Isthmus, Jovian) automatically.

use kona_genesis::SystemConfig;
use kona_protocol::L1BlockInfoTx;
use op_alloy_consensus::OpTxEnvelope;

use crate::{config::DeterministicConfig, l1::L1Block};

/// Construct the L1 info deposit transaction for an L2 block.
///
/// This is always the first transaction in every L2 block.
pub fn l1_info_deposit_tx(
    config: &DeterministicConfig,
    l1_block: &L1Block,
    seq_num: u64,
) -> Result<OpTxEnvelope, Box<dyn std::error::Error>> {
    let rollup_config = config.rollup_config();

    // Build the L1 chain config
    let l1_config = config.l1_chain_config();

    // Use the system config from the rollup config genesis — must match what
    // op-program derives so that deposit tx calldata (and thus gas) is identical.
    //
    // LIMITATION: This always uses the genesis system config and does not track
    // updates from SystemConfigUpdate events on L1. If the test framework is
    // extended to include L1 blocks that emit SystemConfigUpdate events (e.g.,
    // changing the batcher address, gas limit, or fee scalars), this function
    // must be updated to accept the current system config as a parameter rather
    // than reading it from genesis. Without that change, deposit transactions
    // would be constructed with stale values, causing state root mismatches
    // when verified against op-program/kona-host.
    let system_config = rollup_config
        .genesis
        .system_config
        .clone()
        .unwrap_or_else(|| SystemConfig {
            batcher_address: config.batcher,
            gas_limit: 30_000_000,
            ..Default::default()
        });

    let (_info, sealed_deposit) = L1BlockInfoTx::try_new_with_deposit_tx(
        &rollup_config,
        &l1_config,
        &system_config,
        seq_num,
        l1_block.header.inner(),
        config.l2_block_time,
    )?;

    // Convert the Sealed<TxDeposit> into an OpTxEnvelope
    Ok(OpTxEnvelope::Deposit(sealed_deposit))
}
