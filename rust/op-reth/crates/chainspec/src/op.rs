//! Chain specification for the Optimism Mainnet network.

use crate::{LazyLock, OpChainSpec};
use alloc::sync::Arc;
use alloy_primitives::{B256, b256};

#[cfg(not(feature = "superchain-configs"))]
use crate::make_op_genesis_header;
#[cfg(not(feature = "superchain-configs"))]
use alloc::vec;
#[cfg(not(feature = "superchain-configs"))]
use alloy_chains::Chain;
#[cfg(not(feature = "superchain-configs"))]
use alloy_primitives::U256;
#[cfg(not(feature = "superchain-configs"))]
use reth_chainspec::{BaseFeeParams, BaseFeeParamsKind, ChainSpec, Hardfork};
#[cfg(not(feature = "superchain-configs"))]
use reth_ethereum_forks::EthereumHardfork;
#[cfg(not(feature = "superchain-configs"))]
use reth_optimism_forks::{OP_MAINNET_HARDFORKS, OpHardfork};
#[cfg(not(feature = "superchain-configs"))]
use reth_primitives_traits::SealedHeader;

/// The canonical OP Mainnet (Bedrock) genesis block hash.
pub(crate) const OP_MAINNET_GENESIS_HASH: B256 =
    b256!("0x7ca38a1916c42007829c55e69d3e9a73265554b586a499015373241b8a3fa48b");

/// OP Mainnet genesis state root.
///
/// The OP Mainnet genesis `alloc` is empty: the state of the first Bedrock block was imported from a
/// trusted source rather than expressed as an allocation, so the state root cannot be recomputed
/// from `alloc`. This is that precomputed root (op-geth carries it as the genesis `stateHash`; see
/// `core/superchain.go`).
#[cfg(feature = "superchain-configs")]
pub(crate) const OP_MAINNET_GENESIS_STATE_ROOT: B256 =
    b256!("0xeddb4c1786789419153a27c4c80ff44a2226b6eda04f7e22ce5bae892ea568eb");

/// The Optimism Mainnet spec, built from the embedded superchain registry.
///
/// Forks (including their activation timestamps) come straight from the registry, so a new hardfork
/// only needs a registry bump rather than a hand-edited fork list. The Bedrock genesis state root is
/// restored via [`OP_MAINNET_GENESIS_STATE_ROOT`] and verified against [`OP_MAINNET_GENESIS_HASH`].
#[cfg(feature = "superchain-configs")]
pub static OP_MAINNET: LazyLock<Arc<OpChainSpec>> = LazyLock::new(|| {
    use crate::make_op_genesis_header;
    use reth_primitives_traits::SealedHeader;

    let genesis = crate::superchain::read_superchain_genesis("op", "mainnet")
        .expect("embedded superchain registry must contain op-mainnet");
    let mut spec = OpChainSpec::from(genesis.clone());

    // `from_genesis` derives the state root from `alloc`, which is empty for OP Mainnet. Restore the
    // imported Bedrock state root and re-seal the genesis header (mirrors op-geth's `stateHash`
    // handling), then verify we reproduced the canonical genesis hash.
    let mut header = make_op_genesis_header(&genesis, &spec.inner.hardforks);
    header.state_root = OP_MAINNET_GENESIS_STATE_ROOT;
    spec.inner.genesis_header = SealedHeader::seal_slow(header);
    assert_eq!(
        spec.inner.genesis_header.hash(),
        OP_MAINNET_GENESIS_HASH,
        "op-mainnet genesis derived from the superchain registry does not match the canonical hash",
    );

    Arc::new(spec)
});

/// The Optimism Mainnet spec.
///
/// Fallback used when the `superchain-configs` feature is disabled (the registry archive is then not
/// compiled in). Hand-coded fork schedule with the genesis hash pinned explicitly.
#[cfg(not(feature = "superchain-configs"))]
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
                OP_MAINNET_GENESIS_HASH,
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
