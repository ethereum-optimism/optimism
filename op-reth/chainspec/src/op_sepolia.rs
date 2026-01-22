//! Chain specification for the Optimism Sepolia testnet network.

use crate::{make_op_genesis_header, LazyLock, OpChainSpec};
use alloc::{sync::Arc, vec};
use alloy_chains::{Chain, NamedChain};
use alloy_primitives::{b256, U256};
use reth_chainspec::{BaseFeeParams, BaseFeeParamsKind, ChainSpec, Hardfork};
use reth_ethereum_forks::EthereumHardfork;
use reth_optimism_forks::{OpHardfork, OP_SEPOLIA_HARDFORKS};
use reth_primitives_traits::SealedHeader;

/// The OP Sepolia spec
pub static OP_SEPOLIA: LazyLock<Arc<OpChainSpec>> = LazyLock::new(|| {
    let genesis = serde_json::from_str(include_str!("../res/genesis/sepolia_op.json"))
        .expect("Can't deserialize OP Sepolia genesis json");
    let hardforks = OP_SEPOLIA_HARDFORKS.clone();
    OpChainSpec {
        inner: ChainSpec {
            chain: Chain::from_named(NamedChain::OptimismSepolia),
            genesis_header: SealedHeader::new(
                make_op_genesis_header(&genesis, &hardforks),
                b256!("0x102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887d"),
            ),
            genesis,
            paris_block_and_final_difficulty: Some((0, U256::from(0))),
            hardforks,
            base_fee_params: BaseFeeParamsKind::Variable(
                vec![
                    (EthereumHardfork::London.boxed(), BaseFeeParams::optimism_sepolia()),
                    (OpHardfork::Canyon.boxed(), BaseFeeParams::optimism_sepolia_canyon()),
                ]
                .into(),
            ),
            prune_delete_limit: 10000,
            ..Default::default()
        },
    }
    .into()
});
