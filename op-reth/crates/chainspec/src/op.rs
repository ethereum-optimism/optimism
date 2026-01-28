//! Chain specification for the Optimism Mainnet network.

use crate::{make_op_genesis_header, LazyLock, OpChainSpec};
use alloc::{sync::Arc, vec};
use alloy_chains::Chain;
use alloy_primitives::{b256, U256};
use reth_chainspec::{BaseFeeParams, BaseFeeParamsKind, ChainSpec, Hardfork};
use reth_ethereum_forks::EthereumHardfork;
use reth_optimism_forks::{OpHardfork, OP_MAINNET_HARDFORKS};
use reth_primitives_traits::SealedHeader;

/// The Optimism Mainnet spec
pub static OP_MAINNET: LazyLock<Arc<OpChainSpec>> = LazyLock::new(|| {
    // genesis contains empty alloc field because state at first bedrock block is imported
    // manually from trusted source
    let genesis = serde_json::from_str(include_str!("../res/genesis/optimism.json"))
        .expect("Can't deserialize Optimism Mainnet genesis json");
    let hardforks = OP_MAINNET_HARDFORKS.clone();
    OpChainSpec {
        inner: ChainSpec {
            chain: Chain::optimism_mainnet(),
            genesis_header: SealedHeader::new(
                make_op_genesis_header(&genesis, &hardforks),
                b256!("0x7ca38a1916c42007829c55e69d3e9a73265554b586a499015373241b8a3fa48b"),
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
            prune_delete_limit: 10000,
            ..Default::default()
        },
    }
    .into()
});
