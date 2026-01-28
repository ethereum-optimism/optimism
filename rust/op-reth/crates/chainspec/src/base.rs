//! Chain specification for the Base Mainnet network.

use alloc::{sync::Arc, vec};

use alloy_chains::Chain;
use alloy_primitives::{b256, U256};
use reth_chainspec::{BaseFeeParams, BaseFeeParamsKind, ChainSpec};
use reth_ethereum_forks::{EthereumHardfork, Hardfork};
use reth_optimism_forks::{OpHardfork, BASE_MAINNET_HARDFORKS};
use reth_primitives_traits::SealedHeader;

use crate::{make_op_genesis_header, LazyLock, OpChainSpec};

/// The Base mainnet spec
pub static BASE_MAINNET: LazyLock<Arc<OpChainSpec>> = LazyLock::new(|| {
    let genesis = serde_json::from_str(include_str!("../res/genesis/base.json"))
        .expect("Can't deserialize Base genesis json");
    let hardforks = BASE_MAINNET_HARDFORKS.clone();
    OpChainSpec {
        inner: ChainSpec {
            chain: Chain::base_mainnet(),
            genesis_header: SealedHeader::new(
                make_op_genesis_header(&genesis, &hardforks),
                b256!("0xf712aa9241cc24369b143cf6dce85f0902a9731e70d66818a3a5845b296c73dd"),
            ),
            genesis,
            paris_block_and_final_difficulty: Some((0, U256::from(0))),
            hardforks,
            base_fee_params: BaseFeeParamsKind::Variable(
                vec![
                    (EthereumHardfork::London.boxed(), BaseFeeParams::optimism()),
                    (OpHardfork::Canyon.boxed(), BaseFeeParams::optimism_canyon()),
                ]
                .into(),
            ),
            ..Default::default()
        },
    }
    .into()
});
