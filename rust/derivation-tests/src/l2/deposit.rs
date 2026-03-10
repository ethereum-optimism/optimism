//! L1 info deposit transaction construction.
//!
//! Delegates to `kona_protocol::L1BlockInfoTx::try_new_with_deposit_tx` which handles
//! all hardfork variants (Bedrock, Ecotone, Isthmus, Jovian) automatically.

use kona_genesis::SystemConfig;
use kona_protocol::L1BlockInfoTx;
use op_alloy_consensus::OpTxEnvelope;

use crate::config::DeterministicConfig;
use crate::l1::L1Block;

/// Construct the L1 info deposit transaction for an L2 block.
///
/// This is always the first transaction in every L2 block.
pub fn l1_info_deposit_tx(
    config: &DeterministicConfig,
    l1_block: &L1Block,
    seq_num: u64,
) -> Result<OpTxEnvelope, Box<dyn std::error::Error>> {
    let rollup_config = config.rollup_config();

    // Build the L1 chain config (Ethereum mainnet-like config for tests)
    let l1_config = alloy_genesis::ChainConfig::default();

    // Build a minimal system config
    let system_config = SystemConfig {
        batcher_address: config.batcher,
        gas_limit: 30_000_000,
        ..Default::default()
    };

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
