//! Chain specification for the Optimism Sepolia testnet network.

use crate::{LazyLock, OpChainSpec};
use alloc::sync::Arc;
use alloy_primitives::{B256, b256};

#[cfg(not(feature = "superchain-configs"))]
use crate::make_op_genesis_header;
#[cfg(not(feature = "superchain-configs"))]
use alloc::vec;
#[cfg(not(feature = "superchain-configs"))]
use alloy_chains::{Chain, NamedChain};
#[cfg(not(feature = "superchain-configs"))]
use alloy_primitives::U256;
#[cfg(not(feature = "superchain-configs"))]
use reth_chainspec::{BaseFeeParams, BaseFeeParamsKind, ChainSpec, Hardfork};
#[cfg(not(feature = "superchain-configs"))]
use reth_ethereum_forks::EthereumHardfork;
#[cfg(not(feature = "superchain-configs"))]
use reth_optimism_forks::{OP_SEPOLIA_HARDFORKS, OpHardfork};
#[cfg(not(feature = "superchain-configs"))]
use reth_primitives_traits::SealedHeader;

/// The canonical OP Sepolia genesis block hash.
pub(crate) const OP_SEPOLIA_GENESIS_HASH: B256 =
    b256!("0x102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887d");

/// The OP Sepolia spec, built from the embedded superchain registry.
///
/// Forks (including their activation timestamps) come straight from the registry. Unlike OP Mainnet,
/// the OP Sepolia genesis ships a full `alloc`, so `from_genesis` reproduces the genesis header
/// directly; we still verify it against [`OP_SEPOLIA_GENESIS_HASH`].
#[cfg(feature = "superchain-configs")]
pub static OP_SEPOLIA: LazyLock<Arc<OpChainSpec>> = LazyLock::new(|| {
    use reth_chainspec::EthChainSpec;

    let spec = OpChainSpec::from(
        crate::superchain::read_superchain_genesis("op", "sepolia")
            .expect("embedded superchain registry must contain op-sepolia"),
    );
    assert_eq!(
        spec.genesis_hash(),
        OP_SEPOLIA_GENESIS_HASH,
        "op-sepolia genesis derived from the superchain registry does not match the canonical hash",
    );
    Arc::new(spec)
});

/// The OP Sepolia spec.
///
/// Fallback used when the `superchain-configs` feature is disabled (the registry archive is then not
/// compiled in). Hand-coded fork schedule with the genesis hash pinned explicitly.
#[cfg(not(feature = "superchain-configs"))]
pub static OP_SEPOLIA: LazyLock<Arc<OpChainSpec>> = LazyLock::new(|| {
    let genesis = serde_json::from_str(include_str!("../res/genesis/sepolia_op.json"))
        .expect("Can't deserialize OP Sepolia genesis json");
    let hardforks = OP_SEPOLIA_HARDFORKS.clone();
    OpChainSpec {
        inner: ChainSpec {
            chain: Chain::from_named(NamedChain::OptimismSepolia),
            genesis_header: SealedHeader::new(
                make_op_genesis_header(&genesis, &hardforks),
                OP_SEPOLIA_GENESIS_HASH,
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
