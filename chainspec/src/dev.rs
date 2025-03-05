//! Chain specification in dev mode for custom chain.

use alloc::sync::Arc;

use alloy_chains::Chain;
use alloy_primitives::U256;
use reth_chainspec::{BaseFeeParams, BaseFeeParamsKind, ChainSpec};
use reth_optimism_forks::DEV_HARDFORKS;
use reth_primitives_traits::SealedHeader;

use crate::{make_op_genesis_header, LazyLock, OpChainSpec};

/// OP dev testnet specification
///
/// Includes 20 prefunded accounts with `10_000` ETH each derived from mnemonic "test test test test
/// test test test test test test test junk".
pub static OP_DEV: LazyLock<Arc<OpChainSpec>> = LazyLock::new(|| {
    let genesis = serde_json::from_str(include_str!("../res/genesis/dev.json"))
        .expect("Can't deserialize Dev testnet genesis json");
    let hardforks = DEV_HARDFORKS.clone();
    let genesis_header = SealedHeader::seal_slow(make_op_genesis_header(&genesis, &hardforks));
    OpChainSpec {
        inner: ChainSpec {
            chain: Chain::dev(),
            genesis_header,
            genesis,
            paris_block_and_final_difficulty: Some((0, U256::from(0))),
            hardforks,
            base_fee_params: BaseFeeParamsKind::Constant(BaseFeeParams::ethereum()),
            deposit_contract: None, // TODO: do we even have?
            ..Default::default()
        },
    }
    .into()
});
