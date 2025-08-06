//! OP-Reth chain specs.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(not(feature = "std"), no_std)]

// About the provided chain specs from `res/superchain-configs.tar`:
// The provided `OpChainSpec` structs are built from config files read from
// `superchain-configs.tar`. This `superchain-configs.tar` file contains the chain configs and
// genesis files for all chains. It is created by the `fetch_superchain_config.sh` script in
// the `res` directory. Where all configs are where initial loaded from
// <https://github.com/ethereum-optimism/superchain-registry>. See the script for more details.
//
// The file is a tar archive containing the following files:
// - `genesis/<environment>/<chain_name>.json.zz`: The genesis file compressed with deflate. It
//   contains the initial accounts, etc.
// - `configs/<environment>/<chain_name>.json`: The chain metadata file containing the chain id,
//   hard forks, etc.
//
// For example, for `UNICHAIN_MAINNET`, the `genesis/mainnet/unichain.json.zz` and
// `configs/mainnet/base.json` is loaded and combined into the `OpChainSpec` struct.
// See `read_superchain_genesis` in `configs.rs` for more details.
//
// To update the chain specs, run the `fetch_superchain_config.sh` script in the `res` directory.
// This will fetch the latest chain configs from the superchain registry and create a new
// `superchain-configs.tar` file. See the script for more details.

extern crate alloc;

mod base;
mod base_sepolia;
mod basefee;

pub mod constants;
mod dev;
mod op;
mod op_sepolia;

#[cfg(feature = "superchain-configs")]
mod superchain;
#[cfg(feature = "superchain-configs")]
pub use superchain::*;

pub use base::BASE_MAINNET;
pub use base_sepolia::BASE_SEPOLIA;
pub use basefee::*;
pub use dev::OP_DEV;
pub use op::OP_MAINNET;
pub use op_sepolia::OP_SEPOLIA;

/// Re-export for convenience
pub use reth_optimism_forks::*;

use alloc::{boxed::Box, vec, vec::Vec};
use alloy_chains::Chain;
use alloy_consensus::{proofs::storage_root_unhashed, BlockHeader, Header};
use alloy_eips::eip7840::BlobParams;
use alloy_genesis::Genesis;
use alloy_hardforks::Hardfork;
use alloy_primitives::{B256, U256};
use derive_more::{Constructor, Deref, From, Into};
use reth_chainspec::{
    BaseFeeParams, BaseFeeParamsKind, ChainSpec, ChainSpecBuilder, DepositContract,
    DisplayHardforks, EthChainSpec, EthereumHardforks, ForkFilter, ForkId, Hardforks, Head,
};
use reth_ethereum_forks::{ChainHardforks, EthereumHardfork, ForkCondition};
use reth_network_peers::NodeRecord;
use reth_optimism_primitives::ADDRESS_L2_TO_L1_MESSAGE_PASSER;
use reth_primitives_traits::{sync::LazyLock, SealedHeader};

/// Chain spec builder for a OP stack chain.
#[derive(Debug, Default, From)]
pub struct OpChainSpecBuilder {
    /// [`ChainSpecBuilder`]
    inner: ChainSpecBuilder,
}

impl OpChainSpecBuilder {
    /// Construct a new builder from the base mainnet chain spec.
    pub fn base_mainnet() -> Self {
        let mut inner = ChainSpecBuilder::default()
            .chain(BASE_MAINNET.chain)
            .genesis(BASE_MAINNET.genesis.clone());
        let forks = BASE_MAINNET.hardforks.clone();
        inner = inner.with_forks(forks);

        Self { inner }
    }

    /// Construct a new builder from the optimism mainnet chain spec.
    pub fn optimism_mainnet() -> Self {
        let mut inner =
            ChainSpecBuilder::default().chain(OP_MAINNET.chain).genesis(OP_MAINNET.genesis.clone());
        let forks = OP_MAINNET.hardforks.clone();
        inner = inner.with_forks(forks);

        Self { inner }
    }
}

impl OpChainSpecBuilder {
    /// Set the chain ID
    pub fn chain(mut self, chain: Chain) -> Self {
        self.inner = self.inner.chain(chain);
        self
    }

    /// Set the genesis block.
    pub fn genesis(mut self, genesis: Genesis) -> Self {
        self.inner = self.inner.genesis(genesis);
        self
    }

    /// Add the given fork with the given activation condition to the spec.
    pub fn with_fork<H: Hardfork>(mut self, fork: H, condition: ForkCondition) -> Self {
        self.inner = self.inner.with_fork(fork, condition);
        self
    }

    /// Add the given forks with the given activation condition to the spec.
    pub fn with_forks(mut self, forks: ChainHardforks) -> Self {
        self.inner = self.inner.with_forks(forks);
        self
    }

    /// Remove the given fork from the spec.
    pub fn without_fork(mut self, fork: OpHardfork) -> Self {
        self.inner = self.inner.without_fork(fork);
        self
    }

    /// Enable Bedrock at genesis
    pub fn bedrock_activated(mut self) -> Self {
        self.inner = self.inner.paris_activated();
        self.inner = self.inner.with_fork(OpHardfork::Bedrock, ForkCondition::Block(0));
        self
    }

    /// Enable Regolith at genesis
    pub fn regolith_activated(mut self) -> Self {
        self = self.bedrock_activated();
        self.inner = self.inner.with_fork(OpHardfork::Regolith, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Canyon at genesis
    pub fn canyon_activated(mut self) -> Self {
        self = self.regolith_activated();
        // Canyon also activates changes from L1's Shanghai hardfork
        self.inner = self.inner.with_fork(EthereumHardfork::Shanghai, ForkCondition::Timestamp(0));
        self.inner = self.inner.with_fork(OpHardfork::Canyon, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Ecotone at genesis
    pub fn ecotone_activated(mut self) -> Self {
        self = self.canyon_activated();
        self.inner = self.inner.with_fork(EthereumHardfork::Cancun, ForkCondition::Timestamp(0));
        self.inner = self.inner.with_fork(OpHardfork::Ecotone, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Fjord at genesis
    pub fn fjord_activated(mut self) -> Self {
        self = self.ecotone_activated();
        self.inner = self.inner.with_fork(OpHardfork::Fjord, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Granite at genesis
    pub fn granite_activated(mut self) -> Self {
        self = self.fjord_activated();
        self.inner = self.inner.with_fork(OpHardfork::Granite, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Holocene at genesis
    pub fn holocene_activated(mut self) -> Self {
        self = self.granite_activated();
        self.inner = self.inner.with_fork(OpHardfork::Holocene, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Isthmus at genesis
    pub fn isthmus_activated(mut self) -> Self {
        self = self.holocene_activated();
        self.inner = self.inner.with_fork(OpHardfork::Isthmus, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Jovian at genesis
    pub fn jovian_activated(mut self) -> Self {
        self = self.isthmus_activated();
        self.inner = self.inner.with_fork(OpHardfork::Jovian, ForkCondition::Timestamp(0));
        self
    }

    /// Enable Interop at genesis
    pub fn interop_activated(mut self) -> Self {
        self = self.jovian_activated();
        self.inner = self.inner.with_fork(OpHardfork::Interop, ForkCondition::Timestamp(0));
        self
    }

    /// Build the resulting [`OpChainSpec`].
    ///
    /// # Panics
    ///
    /// This function panics if the chain ID and genesis is not set ([`Self::chain`] and
    /// [`Self::genesis`])
    pub fn build(self) -> OpChainSpec {
        let mut inner = self.inner.build();
        inner.genesis_header =
            SealedHeader::seal_slow(make_op_genesis_header(&inner.genesis, &inner.hardforks));

        OpChainSpec { inner }
    }
}

/// OP stack chain spec type.
#[derive(Debug, Clone, Deref, Into, Constructor, PartialEq, Eq)]
pub struct OpChainSpec {
    /// [`ChainSpec`].
    pub inner: ChainSpec,
}

impl OpChainSpec {
    /// Converts the given [`Genesis`] into a [`OpChainSpec`].
    pub fn from_genesis(genesis: Genesis) -> Self {
        genesis.into()
    }
}

impl EthChainSpec for OpChainSpec {
    type Header = Header;

    fn chain(&self) -> Chain {
        self.inner.chain()
    }

    fn base_fee_params_at_block(&self, block_number: u64) -> BaseFeeParams {
        self.inner.base_fee_params_at_block(block_number)
    }

    fn base_fee_params_at_timestamp(&self, timestamp: u64) -> BaseFeeParams {
        self.inner.base_fee_params_at_timestamp(timestamp)
    }

    fn blob_params_at_timestamp(&self, timestamp: u64) -> Option<BlobParams> {
        self.inner.blob_params_at_timestamp(timestamp)
    }

    fn deposit_contract(&self) -> Option<&DepositContract> {
        self.inner.deposit_contract()
    }

    fn genesis_hash(&self) -> B256 {
        self.inner.genesis_hash()
    }

    fn prune_delete_limit(&self) -> usize {
        self.inner.prune_delete_limit()
    }

    fn display_hardforks(&self) -> Box<dyn core::fmt::Display> {
        // filter only op hardforks
        let op_forks = self.inner.hardforks.forks_iter().filter(|(fork, _)| {
            !EthereumHardfork::VARIANTS.iter().any(|h| h.name() == (*fork).name())
        });

        Box::new(DisplayHardforks::new(op_forks))
    }

    fn genesis_header(&self) -> &Self::Header {
        self.inner.genesis_header()
    }

    fn genesis(&self) -> &Genesis {
        self.inner.genesis()
    }

    fn bootnodes(&self) -> Option<Vec<NodeRecord>> {
        self.inner.bootnodes()
    }

    fn is_optimism(&self) -> bool {
        true
    }

    fn final_paris_total_difficulty(&self) -> Option<U256> {
        self.inner.final_paris_total_difficulty()
    }

    fn next_block_base_fee(&self, parent: &Header, target_timestamp: u64) -> Option<u64> {
        if self.is_holocene_active_at_timestamp(parent.timestamp()) {
            decode_holocene_base_fee(self, parent, target_timestamp).ok()
        } else {
            self.inner.next_block_base_fee(parent, target_timestamp)
        }
    }
}

impl Hardforks for OpChainSpec {
    fn fork<H: Hardfork>(&self, fork: H) -> ForkCondition {
        self.inner.fork(fork)
    }

    fn forks_iter(&self) -> impl Iterator<Item = (&dyn Hardfork, ForkCondition)> {
        self.inner.forks_iter()
    }

    fn fork_id(&self, head: &Head) -> ForkId {
        self.inner.fork_id(head)
    }

    fn latest_fork_id(&self) -> ForkId {
        self.inner.latest_fork_id()
    }

    fn fork_filter(&self, head: Head) -> ForkFilter {
        self.inner.fork_filter(head)
    }
}

impl EthereumHardforks for OpChainSpec {
    fn ethereum_fork_activation(&self, fork: EthereumHardfork) -> ForkCondition {
        self.fork(fork)
    }
}

impl OpHardforks for OpChainSpec {
    fn op_fork_activation(&self, fork: OpHardfork) -> ForkCondition {
        self.fork(fork)
    }
}

impl From<Genesis> for OpChainSpec {
    fn from(genesis: Genesis) -> Self {
        use reth_optimism_forks::OpHardfork;
        let optimism_genesis_info = OpGenesisInfo::extract_from(&genesis);
        let genesis_info =
            optimism_genesis_info.optimism_chain_info.genesis_info.unwrap_or_default();

        // Block-based hardforks
        let hardfork_opts = [
            (EthereumHardfork::Frontier.boxed(), Some(0)),
            (EthereumHardfork::Homestead.boxed(), genesis.config.homestead_block),
            (EthereumHardfork::Tangerine.boxed(), genesis.config.eip150_block),
            (EthereumHardfork::SpuriousDragon.boxed(), genesis.config.eip155_block),
            (EthereumHardfork::Byzantium.boxed(), genesis.config.byzantium_block),
            (EthereumHardfork::Constantinople.boxed(), genesis.config.constantinople_block),
            (EthereumHardfork::Petersburg.boxed(), genesis.config.petersburg_block),
            (EthereumHardfork::Istanbul.boxed(), genesis.config.istanbul_block),
            (EthereumHardfork::MuirGlacier.boxed(), genesis.config.muir_glacier_block),
            (EthereumHardfork::Berlin.boxed(), genesis.config.berlin_block),
            (EthereumHardfork::London.boxed(), genesis.config.london_block),
            (EthereumHardfork::ArrowGlacier.boxed(), genesis.config.arrow_glacier_block),
            (EthereumHardfork::GrayGlacier.boxed(), genesis.config.gray_glacier_block),
            (OpHardfork::Bedrock.boxed(), genesis_info.bedrock_block),
        ];
        let mut block_hardforks = hardfork_opts
            .into_iter()
            .filter_map(|(hardfork, opt)| opt.map(|block| (hardfork, ForkCondition::Block(block))))
            .collect::<Vec<_>>();

        // We set the paris hardfork for OP networks to zero
        block_hardforks.push((
            EthereumHardfork::Paris.boxed(),
            ForkCondition::TTD {
                activation_block_number: 0,
                total_difficulty: U256::ZERO,
                fork_block: genesis.config.merge_netsplit_block,
            },
        ));

        // Time-based hardforks
        let time_hardfork_opts = [
            // L1
            // we need to map the L1 hardforks to the activation timestamps of the correspondong op
            // hardforks
            (EthereumHardfork::Shanghai.boxed(), genesis_info.canyon_time),
            (EthereumHardfork::Cancun.boxed(), genesis_info.ecotone_time),
            (EthereumHardfork::Prague.boxed(), genesis_info.isthmus_time),
            // OP
            (OpHardfork::Regolith.boxed(), genesis_info.regolith_time),
            (OpHardfork::Canyon.boxed(), genesis_info.canyon_time),
            (OpHardfork::Ecotone.boxed(), genesis_info.ecotone_time),
            (OpHardfork::Fjord.boxed(), genesis_info.fjord_time),
            (OpHardfork::Granite.boxed(), genesis_info.granite_time),
            (OpHardfork::Holocene.boxed(), genesis_info.holocene_time),
            (OpHardfork::Isthmus.boxed(), genesis_info.isthmus_time),
            (OpHardfork::Jovian.boxed(), genesis_info.jovian_time),
            (OpHardfork::Interop.boxed(), genesis_info.interop_time),
        ];

        let mut time_hardforks = time_hardfork_opts
            .into_iter()
            .filter_map(|(hardfork, opt)| {
                opt.map(|time| (hardfork, ForkCondition::Timestamp(time)))
            })
            .collect::<Vec<_>>();

        block_hardforks.append(&mut time_hardforks);

        // Ordered Hardforks
        let mainnet_hardforks = OP_MAINNET_HARDFORKS.clone();
        let mainnet_order = mainnet_hardforks.forks_iter();

        let mut ordered_hardforks = Vec::with_capacity(block_hardforks.len());
        for (hardfork, _) in mainnet_order {
            if let Some(pos) = block_hardforks.iter().position(|(e, _)| **e == *hardfork) {
                ordered_hardforks.push(block_hardforks.remove(pos));
            }
        }

        // append the remaining unknown hardforks to ensure we don't filter any out
        ordered_hardforks.append(&mut block_hardforks);

        let hardforks = ChainHardforks::new(ordered_hardforks);
        let genesis_header = SealedHeader::seal_slow(make_op_genesis_header(&genesis, &hardforks));

        Self {
            inner: ChainSpec {
                chain: genesis.config.chain_id.into(),
                genesis_header,
                genesis,
                hardforks,
                // We assume no OP network merges, and set the paris block and total difficulty to
                // zero
                paris_block_and_final_difficulty: Some((0, U256::ZERO)),
                base_fee_params: optimism_genesis_info.base_fee_params,
                ..Default::default()
            },
        }
    }
}

impl From<ChainSpec> for OpChainSpec {
    fn from(value: ChainSpec) -> Self {
        Self { inner: value }
    }
}

#[derive(Default, Debug)]
struct OpGenesisInfo {
    optimism_chain_info: op_alloy_rpc_types::OpChainInfo,
    base_fee_params: BaseFeeParamsKind,
}

impl OpGenesisInfo {
    fn extract_from(genesis: &Genesis) -> Self {
        let mut info = Self {
            optimism_chain_info: op_alloy_rpc_types::OpChainInfo::extract_from(
                &genesis.config.extra_fields,
            )
            .unwrap_or_default(),
            ..Default::default()
        };
        if let Some(optimism_base_fee_info) = &info.optimism_chain_info.base_fee_info {
            if let (Some(elasticity), Some(denominator)) = (
                optimism_base_fee_info.eip1559_elasticity,
                optimism_base_fee_info.eip1559_denominator,
            ) {
                let base_fee_params = if let Some(canyon_denominator) =
                    optimism_base_fee_info.eip1559_denominator_canyon
                {
                    BaseFeeParamsKind::Variable(
                        vec![
                            (
                                EthereumHardfork::London.boxed(),
                                BaseFeeParams::new(denominator as u128, elasticity as u128),
                            ),
                            (
                                OpHardfork::Canyon.boxed(),
                                BaseFeeParams::new(canyon_denominator as u128, elasticity as u128),
                            ),
                        ]
                        .into(),
                    )
                } else {
                    BaseFeeParams::new(denominator as u128, elasticity as u128).into()
                };

                info.base_fee_params = base_fee_params;
            }
        }

        info
    }
}

/// Helper method building a [`Header`] given [`Genesis`] and [`ChainHardforks`].
pub fn make_op_genesis_header(genesis: &Genesis, hardforks: &ChainHardforks) -> Header {
    let mut header = reth_chainspec::make_genesis_header(genesis, hardforks);

    // If Isthmus is active, overwrite the withdrawals root with the storage root of predeploy
    // `L2ToL1MessagePasser.sol`
    if hardforks.fork(OpHardfork::Isthmus).active_at_timestamp(header.timestamp) {
        if let Some(predeploy) = genesis.alloc.get(&ADDRESS_L2_TO_L1_MESSAGE_PASSER) {
            if let Some(storage) = &predeploy.storage {
                header.withdrawals_root =
                    Some(storage_root_unhashed(storage.iter().map(|(k, v)| (*k, (*v).into()))))
            }
        }
    }

    header
}

#[cfg(test)]
mod tests {
    use alloc::string::String;
    use alloy_genesis::{ChainConfig, Genesis};
    use alloy_primitives::b256;
    use reth_chainspec::{test_fork_ids, BaseFeeParams, BaseFeeParamsKind};
    use reth_ethereum_forks::{EthereumHardfork, ForkCondition, ForkHash, ForkId, Head};
    use reth_optimism_forks::{OpHardfork, OpHardforks};

    use crate::*;

    #[test]
    fn base_mainnet_forkids() {
        let mut base_mainnet = OpChainSpecBuilder::base_mainnet().build();
        base_mainnet.inner.genesis_header.set_hash(BASE_MAINNET.genesis_hash());
        test_fork_ids(
            &BASE_MAINNET,
            &[
                (
                    Head { number: 0, ..Default::default() },
                    ForkId { hash: ForkHash([0x67, 0xda, 0x02, 0x60]), next: 1704992401 },
                ),
                (
                    Head { number: 0, timestamp: 1704992400, ..Default::default() },
                    ForkId { hash: ForkHash([0x67, 0xda, 0x02, 0x60]), next: 1704992401 },
                ),
                (
                    Head { number: 0, timestamp: 1704992401, ..Default::default() },
                    ForkId { hash: ForkHash([0x3c, 0x28, 0x3c, 0xb3]), next: 1710374401 },
                ),
                (
                    Head { number: 0, timestamp: 1710374400, ..Default::default() },
                    ForkId { hash: ForkHash([0x3c, 0x28, 0x3c, 0xb3]), next: 1710374401 },
                ),
                (
                    Head { number: 0, timestamp: 1710374401, ..Default::default() },
                    ForkId { hash: ForkHash([0x51, 0xcc, 0x98, 0xb3]), next: 1720627201 },
                ),
                (
                    Head { number: 0, timestamp: 1720627200, ..Default::default() },
                    ForkId { hash: ForkHash([0x51, 0xcc, 0x98, 0xb3]), next: 1720627201 },
                ),
                (
                    Head { number: 0, timestamp: 1720627201, ..Default::default() },
                    ForkId { hash: ForkHash([0xe4, 0x01, 0x0e, 0xb9]), next: 1726070401 },
                ),
                (
                    Head { number: 0, timestamp: 1726070401, ..Default::default() },
                    ForkId { hash: ForkHash([0xbc, 0x38, 0xf9, 0xca]), next: 1736445601 },
                ),
                (
                    Head { number: 0, timestamp: 1736445601, ..Default::default() },
                    ForkId { hash: ForkHash([0x3a, 0x2a, 0xf1, 0x83]), next: 1746806401 },
                ),
                // Isthmus
                (
                    Head { number: 0, timestamp: 1746806401, ..Default::default() },
                    ForkId { hash: ForkHash([0x86, 0x72, 0x8b, 0x4e]), next: 0 }, /* TODO: update timestamp when Jovian is planned */
                ),
                // // Jovian
                // (
                //     Head { number: 0, timestamp: u64::MAX, ..Default::default() }, /* TODO:
                // update timestamp when Jovian is planned */     ForkId { hash:
                // ForkHash([0xef, 0x0e, 0x58, 0x33]), next: 0 }, ),
            ],
        );
    }

    #[test]
    fn op_sepolia_forkids() {
        test_fork_ids(
            &OP_SEPOLIA,
            &[
                (
                    Head { number: 0, ..Default::default() },
                    ForkId { hash: ForkHash([0x67, 0xa4, 0x03, 0x28]), next: 1699981200 },
                ),
                (
                    Head { number: 0, timestamp: 1699981199, ..Default::default() },
                    ForkId { hash: ForkHash([0x67, 0xa4, 0x03, 0x28]), next: 1699981200 },
                ),
                (
                    Head { number: 0, timestamp: 1699981200, ..Default::default() },
                    ForkId { hash: ForkHash([0xa4, 0x8d, 0x6a, 0x00]), next: 1708534800 },
                ),
                (
                    Head { number: 0, timestamp: 1708534799, ..Default::default() },
                    ForkId { hash: ForkHash([0xa4, 0x8d, 0x6a, 0x00]), next: 1708534800 },
                ),
                (
                    Head { number: 0, timestamp: 1708534800, ..Default::default() },
                    ForkId { hash: ForkHash([0xcc, 0x17, 0xc7, 0xeb]), next: 1716998400 },
                ),
                (
                    Head { number: 0, timestamp: 1716998399, ..Default::default() },
                    ForkId { hash: ForkHash([0xcc, 0x17, 0xc7, 0xeb]), next: 1716998400 },
                ),
                (
                    Head { number: 0, timestamp: 1716998400, ..Default::default() },
                    ForkId { hash: ForkHash([0x54, 0x0a, 0x8c, 0x5d]), next: 1723478400 },
                ),
                (
                    Head { number: 0, timestamp: 1723478399, ..Default::default() },
                    ForkId { hash: ForkHash([0x54, 0x0a, 0x8c, 0x5d]), next: 1723478400 },
                ),
                (
                    Head { number: 0, timestamp: 1723478400, ..Default::default() },
                    ForkId { hash: ForkHash([0x75, 0xde, 0xa4, 0x1e]), next: 1732633200 },
                ),
                (
                    Head { number: 0, timestamp: 1732633200, ..Default::default() },
                    ForkId { hash: ForkHash([0x4a, 0x1c, 0x79, 0x2e]), next: 1744905600 },
                ),
                // Isthmus
                (
                    Head { number: 0, timestamp: 1744905600, ..Default::default() },
                    ForkId { hash: ForkHash([0x6c, 0x62, 0x5e, 0xe1]), next: 0 }, /* TODO: update timestamp when Jovian is planned */
                ),
                // // Jovian
                // (
                //     Head { number: 0, timestamp: u64::MAX, ..Default::default() }, /* TODO:
                // update timestamp when Jovian is planned */     ForkId { hash:
                // ForkHash([0x04, 0x2a, 0x5c, 0x14]), next: 0 }, ),
            ],
        );
    }

    #[test]
    fn op_mainnet_forkids() {
        let mut op_mainnet = OpChainSpecBuilder::optimism_mainnet().build();
        // for OP mainnet we have to do this because the genesis header can't be properly computed
        // from the genesis.json file
        op_mainnet.inner.genesis_header.set_hash(OP_MAINNET.genesis_hash());
        test_fork_ids(
            &op_mainnet,
            &[
                (
                    Head { number: 0, ..Default::default() },
                    ForkId { hash: ForkHash([0xca, 0xf5, 0x17, 0xed]), next: 3950000 },
                ),
                // London
                (
                    Head { number: 105235063, ..Default::default() },
                    ForkId { hash: ForkHash([0xe3, 0x39, 0x8d, 0x7c]), next: 1704992401 },
                ),
                // Bedrock
                (
                    Head { number: 105235063, ..Default::default() },
                    ForkId { hash: ForkHash([0xe3, 0x39, 0x8d, 0x7c]), next: 1704992401 },
                ),
                // Shanghai
                (
                    Head { number: 105235063, timestamp: 1704992401, ..Default::default() },
                    ForkId { hash: ForkHash([0xbd, 0xd4, 0xfd, 0xb2]), next: 1710374401 },
                ),
                // OP activation timestamps
                // https://specs.optimism.io/protocol/superchain-upgrades.html#activation-timestamps
                // Canyon
                (
                    Head { number: 105235063, timestamp: 1704992401, ..Default::default() },
                    ForkId { hash: ForkHash([0xbd, 0xd4, 0xfd, 0xb2]), next: 1710374401 },
                ),
                // Ecotone
                (
                    Head { number: 105235063, timestamp: 1710374401, ..Default::default() },
                    ForkId { hash: ForkHash([0x19, 0xda, 0x4c, 0x52]), next: 1720627201 },
                ),
                // Fjord
                (
                    Head { number: 105235063, timestamp: 1720627201, ..Default::default() },
                    ForkId { hash: ForkHash([0x49, 0xfb, 0xfe, 0x1e]), next: 1726070401 },
                ),
                // Granite
                (
                    Head { number: 105235063, timestamp: 1726070401, ..Default::default() },
                    ForkId { hash: ForkHash([0x44, 0x70, 0x4c, 0xde]), next: 1736445601 },
                ),
                // Holocene
                (
                    Head { number: 105235063, timestamp: 1736445601, ..Default::default() },
                    ForkId { hash: ForkHash([0x2b, 0xd9, 0x3d, 0xc8]), next: 1746806401 },
                ),
                // Isthmus
                (
                    Head { number: 105235063, timestamp: 1746806401, ..Default::default() },
                    ForkId { hash: ForkHash([0x37, 0xbe, 0x75, 0x8f]), next: 0 }, /* TODO: update timestamp when Jovian is planned */
                ),
                // Jovian
                // (
                //     Head { number: 105235063, timestamp: u64::MAX, ..Default::default() }, /*
                // TODO: update timestamp when Jovian is planned */     ForkId {
                // hash: ForkHash([0x26, 0xce, 0xa1, 0x75]), next: 0 }, ),
            ],
        );
    }

    #[test]
    fn base_sepolia_forkids() {
        test_fork_ids(
            &BASE_SEPOLIA,
            &[
                (
                    Head { number: 0, ..Default::default() },
                    ForkId { hash: ForkHash([0xb9, 0x59, 0xb9, 0xf7]), next: 1699981200 },
                ),
                (
                    Head { number: 0, timestamp: 1699981199, ..Default::default() },
                    ForkId { hash: ForkHash([0xb9, 0x59, 0xb9, 0xf7]), next: 1699981200 },
                ),
                (
                    Head { number: 0, timestamp: 1699981200, ..Default::default() },
                    ForkId { hash: ForkHash([0x60, 0x7c, 0xd5, 0xa1]), next: 1708534800 },
                ),
                (
                    Head { number: 0, timestamp: 1708534799, ..Default::default() },
                    ForkId { hash: ForkHash([0x60, 0x7c, 0xd5, 0xa1]), next: 1708534800 },
                ),
                (
                    Head { number: 0, timestamp: 1708534800, ..Default::default() },
                    ForkId { hash: ForkHash([0xbe, 0x96, 0x9b, 0x17]), next: 1716998400 },
                ),
                (
                    Head { number: 0, timestamp: 1716998399, ..Default::default() },
                    ForkId { hash: ForkHash([0xbe, 0x96, 0x9b, 0x17]), next: 1716998400 },
                ),
                (
                    Head { number: 0, timestamp: 1716998400, ..Default::default() },
                    ForkId { hash: ForkHash([0x4e, 0x45, 0x7a, 0x49]), next: 1723478400 },
                ),
                (
                    Head { number: 0, timestamp: 1723478399, ..Default::default() },
                    ForkId { hash: ForkHash([0x4e, 0x45, 0x7a, 0x49]), next: 1723478400 },
                ),
                (
                    Head { number: 0, timestamp: 1723478400, ..Default::default() },
                    ForkId { hash: ForkHash([0x5e, 0xdf, 0xa3, 0xb6]), next: 1732633200 },
                ),
                (
                    Head { number: 0, timestamp: 1732633200, ..Default::default() },
                    ForkId { hash: ForkHash([0x8b, 0x5e, 0x76, 0x29]), next: 1744905600 },
                ),
                // Isthmus
                (
                    Head { number: 0, timestamp: 1744905600, ..Default::default() },
                    ForkId { hash: ForkHash([0x06, 0x0a, 0x4d, 0x1d]), next: 0 }, /* TODO: update timestamp when Jovian is planned */
                ),
                // // Jovian
                // (
                //     Head { number: 0, timestamp: u64::MAX, ..Default::default() }, /* TODO:
                // update timestamp when Jovian is planned */     ForkId { hash:
                // ForkHash([0xcd, 0xfd, 0x39, 0x99]), next: 0 }, ),
            ],
        );
    }

    #[test]
    fn base_mainnet_genesis() {
        let genesis = BASE_MAINNET.genesis_header();
        assert_eq!(
            genesis.hash_slow(),
            b256!("0xf712aa9241cc24369b143cf6dce85f0902a9731e70d66818a3a5845b296c73dd")
        );
        let base_fee = BASE_MAINNET.next_block_base_fee(genesis, genesis.timestamp).unwrap();
        // <https://base.blockscout.com/block/1>
        assert_eq!(base_fee, 980000000);
    }

    #[test]
    fn base_sepolia_genesis() {
        let genesis = BASE_SEPOLIA.genesis_header();
        assert_eq!(
            genesis.hash_slow(),
            b256!("0x0dcc9e089e30b90ddfc55be9a37dd15bc551aeee999d2e2b51414c54eaf934e4")
        );
        let base_fee = BASE_SEPOLIA.next_block_base_fee(genesis, genesis.timestamp).unwrap();
        // <https://base-sepolia.blockscout.com/block/1>
        assert_eq!(base_fee, 980000000);
    }

    #[test]
    fn op_sepolia_genesis() {
        let genesis = OP_SEPOLIA.genesis_header();
        assert_eq!(
            genesis.hash_slow(),
            b256!("0x102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887d")
        );
        let base_fee = OP_SEPOLIA.next_block_base_fee(genesis, genesis.timestamp).unwrap();
        // <https://optimism-sepolia.blockscout.com/block/1>
        assert_eq!(base_fee, 980000000);
    }

    #[test]
    fn latest_base_mainnet_fork_id() {
        assert_eq!(
            ForkId { hash: ForkHash([0x86, 0x72, 0x8b, 0x4e]), next: 0 },
            BASE_MAINNET.latest_fork_id()
        )
    }

    #[test]
    fn latest_base_mainnet_fork_id_with_builder() {
        let base_mainnet = OpChainSpecBuilder::base_mainnet().build();
        assert_eq!(
            ForkId { hash: ForkHash([0x86, 0x72, 0x8b, 0x4e]), next: 0 },
            base_mainnet.latest_fork_id()
        )
    }

    #[test]
    fn is_bedrock_active() {
        let op_mainnet = OpChainSpecBuilder::optimism_mainnet().build();
        assert!(!op_mainnet.is_bedrock_active_at_block(1))
    }

    #[test]
    fn parse_optimism_hardforks() {
        let geth_genesis = r#"
    {
      "config": {
        "bedrockBlock": 10,
        "regolithTime": 20,
        "canyonTime": 30,
        "ecotoneTime": 40,
        "fjordTime": 50,
        "graniteTime": 51,
        "holoceneTime": 52,
        "isthmusTime": 53,
        "optimism": {
          "eip1559Elasticity": 60,
          "eip1559Denominator": 70
        }
      }
    }
    "#;
        let genesis: Genesis = serde_json::from_str(geth_genesis).unwrap();

        let actual_bedrock_block = genesis.config.extra_fields.get("bedrockBlock");
        assert_eq!(actual_bedrock_block, Some(serde_json::Value::from(10)).as_ref());
        let actual_regolith_timestamp = genesis.config.extra_fields.get("regolithTime");
        assert_eq!(actual_regolith_timestamp, Some(serde_json::Value::from(20)).as_ref());
        let actual_canyon_timestamp = genesis.config.extra_fields.get("canyonTime");
        assert_eq!(actual_canyon_timestamp, Some(serde_json::Value::from(30)).as_ref());
        let actual_ecotone_timestamp = genesis.config.extra_fields.get("ecotoneTime");
        assert_eq!(actual_ecotone_timestamp, Some(serde_json::Value::from(40)).as_ref());
        let actual_fjord_timestamp = genesis.config.extra_fields.get("fjordTime");
        assert_eq!(actual_fjord_timestamp, Some(serde_json::Value::from(50)).as_ref());
        let actual_granite_timestamp = genesis.config.extra_fields.get("graniteTime");
        assert_eq!(actual_granite_timestamp, Some(serde_json::Value::from(51)).as_ref());
        let actual_holocene_timestamp = genesis.config.extra_fields.get("holoceneTime");
        assert_eq!(actual_holocene_timestamp, Some(serde_json::Value::from(52)).as_ref());
        let actual_isthmus_timestamp = genesis.config.extra_fields.get("isthmusTime");
        assert_eq!(actual_isthmus_timestamp, Some(serde_json::Value::from(53)).as_ref());

        let optimism_object = genesis.config.extra_fields.get("optimism").unwrap();
        assert_eq!(
            optimism_object,
            &serde_json::json!({
                "eip1559Elasticity": 60,
                "eip1559Denominator": 70,
            })
        );

        let chain_spec: OpChainSpec = genesis.into();

        assert_eq!(
            chain_spec.base_fee_params,
            BaseFeeParamsKind::Constant(BaseFeeParams::new(70, 60))
        );

        assert!(!chain_spec.is_fork_active_at_block(OpHardfork::Bedrock, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Regolith, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Canyon, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Ecotone, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Fjord, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Granite, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Holocene, 0));

        assert!(chain_spec.is_fork_active_at_block(OpHardfork::Bedrock, 10));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Regolith, 20));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Canyon, 30));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Ecotone, 40));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Fjord, 50));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Granite, 51));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Holocene, 52));
    }

    #[test]
    fn parse_optimism_hardforks_variable_base_fee_params() {
        let geth_genesis = r#"
    {
      "config": {
        "bedrockBlock": 10,
        "regolithTime": 20,
        "canyonTime": 30,
        "ecotoneTime": 40,
        "fjordTime": 50,
        "graniteTime": 51,
        "holoceneTime": 52,
        "isthmusTime": 53,
        "optimism": {
          "eip1559Elasticity": 60,
          "eip1559Denominator": 70,
          "eip1559DenominatorCanyon": 80
        }
      }
    }
    "#;
        let genesis: Genesis = serde_json::from_str(geth_genesis).unwrap();

        let actual_bedrock_block = genesis.config.extra_fields.get("bedrockBlock");
        assert_eq!(actual_bedrock_block, Some(serde_json::Value::from(10)).as_ref());
        let actual_regolith_timestamp = genesis.config.extra_fields.get("regolithTime");
        assert_eq!(actual_regolith_timestamp, Some(serde_json::Value::from(20)).as_ref());
        let actual_canyon_timestamp = genesis.config.extra_fields.get("canyonTime");
        assert_eq!(actual_canyon_timestamp, Some(serde_json::Value::from(30)).as_ref());
        let actual_ecotone_timestamp = genesis.config.extra_fields.get("ecotoneTime");
        assert_eq!(actual_ecotone_timestamp, Some(serde_json::Value::from(40)).as_ref());
        let actual_fjord_timestamp = genesis.config.extra_fields.get("fjordTime");
        assert_eq!(actual_fjord_timestamp, Some(serde_json::Value::from(50)).as_ref());
        let actual_granite_timestamp = genesis.config.extra_fields.get("graniteTime");
        assert_eq!(actual_granite_timestamp, Some(serde_json::Value::from(51)).as_ref());
        let actual_holocene_timestamp = genesis.config.extra_fields.get("holoceneTime");
        assert_eq!(actual_holocene_timestamp, Some(serde_json::Value::from(52)).as_ref());
        let actual_isthmus_timestamp = genesis.config.extra_fields.get("isthmusTime");
        assert_eq!(actual_isthmus_timestamp, Some(serde_json::Value::from(53)).as_ref());

        let optimism_object = genesis.config.extra_fields.get("optimism").unwrap();
        assert_eq!(
            optimism_object,
            &serde_json::json!({
                "eip1559Elasticity": 60,
                "eip1559Denominator": 70,
                "eip1559DenominatorCanyon": 80
            })
        );

        let chain_spec: OpChainSpec = genesis.into();

        assert_eq!(
            chain_spec.base_fee_params,
            BaseFeeParamsKind::Variable(
                vec![
                    (EthereumHardfork::London.boxed(), BaseFeeParams::new(70, 60)),
                    (OpHardfork::Canyon.boxed(), BaseFeeParams::new(80, 60)),
                ]
                .into()
            )
        );

        assert!(!chain_spec.is_fork_active_at_block(OpHardfork::Bedrock, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Regolith, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Canyon, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Ecotone, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Fjord, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Granite, 0));
        assert!(!chain_spec.is_fork_active_at_timestamp(OpHardfork::Holocene, 0));

        assert!(chain_spec.is_fork_active_at_block(OpHardfork::Bedrock, 10));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Regolith, 20));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Canyon, 30));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Ecotone, 40));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Fjord, 50));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Granite, 51));
        assert!(chain_spec.is_fork_active_at_timestamp(OpHardfork::Holocene, 52));
    }

    #[test]
    fn parse_genesis_optimism_with_variable_base_fee_params() {
        use op_alloy_rpc_types::OpBaseFeeInfo;

        let geth_genesis = r#"
    {
      "config": {
        "chainId": 8453,
        "homesteadBlock": 0,
        "eip150Block": 0,
        "eip155Block": 0,
        "eip158Block": 0,
        "byzantiumBlock": 0,
        "constantinopleBlock": 0,
        "petersburgBlock": 0,
        "istanbulBlock": 0,
        "muirGlacierBlock": 0,
        "berlinBlock": 0,
        "londonBlock": 0,
        "arrowGlacierBlock": 0,
        "grayGlacierBlock": 0,
        "mergeNetsplitBlock": 0,
        "bedrockBlock": 0,
        "regolithTime": 15,
        "terminalTotalDifficulty": 0,
        "terminalTotalDifficultyPassed": true,
        "optimism": {
          "eip1559Elasticity": 6,
          "eip1559Denominator": 50
        }
      }
    }
    "#;
        let genesis: Genesis = serde_json::from_str(geth_genesis).unwrap();
        let chainspec = OpChainSpec::from(genesis.clone());

        let actual_chain_id = genesis.config.chain_id;
        assert_eq!(actual_chain_id, 8453);

        assert_eq!(
            chainspec.hardforks.get(EthereumHardfork::Istanbul),
            Some(ForkCondition::Block(0))
        );

        let actual_bedrock_block = genesis.config.extra_fields.get("bedrockBlock");
        assert_eq!(actual_bedrock_block, Some(serde_json::Value::from(0)).as_ref());
        let actual_canyon_timestamp = genesis.config.extra_fields.get("canyonTime");
        assert_eq!(actual_canyon_timestamp, None);

        assert!(genesis.config.terminal_total_difficulty_passed);

        let optimism_object = genesis.config.extra_fields.get("optimism").unwrap();
        let optimism_base_fee_info =
            serde_json::from_value::<OpBaseFeeInfo>(optimism_object.clone()).unwrap();

        assert_eq!(
            optimism_base_fee_info,
            OpBaseFeeInfo {
                eip1559_elasticity: Some(6),
                eip1559_denominator: Some(50),
                eip1559_denominator_canyon: None,
            }
        );
        assert_eq!(
            chainspec.base_fee_params,
            BaseFeeParamsKind::Constant(BaseFeeParams {
                max_change_denominator: 50,
                elasticity_multiplier: 6,
            })
        );

        assert!(chainspec.is_fork_active_at_block(OpHardfork::Bedrock, 0));

        assert!(chainspec.is_fork_active_at_timestamp(OpHardfork::Regolith, 20));
    }

    #[test]
    fn test_fork_order_optimism_mainnet() {
        use reth_optimism_forks::OpHardfork;

        let genesis = Genesis {
            config: ChainConfig {
                chain_id: 0,
                homestead_block: Some(0),
                dao_fork_block: Some(0),
                dao_fork_support: false,
                eip150_block: Some(0),
                eip155_block: Some(0),
                eip158_block: Some(0),
                byzantium_block: Some(0),
                constantinople_block: Some(0),
                petersburg_block: Some(0),
                istanbul_block: Some(0),
                muir_glacier_block: Some(0),
                berlin_block: Some(0),
                london_block: Some(0),
                arrow_glacier_block: Some(0),
                gray_glacier_block: Some(0),
                merge_netsplit_block: Some(0),
                shanghai_time: Some(0),
                cancun_time: Some(0),
                prague_time: Some(0),
                terminal_total_difficulty: Some(U256::ZERO),
                extra_fields: [
                    (String::from("bedrockBlock"), 0.into()),
                    (String::from("regolithTime"), 0.into()),
                    (String::from("canyonTime"), 0.into()),
                    (String::from("ecotoneTime"), 0.into()),
                    (String::from("fjordTime"), 0.into()),
                    (String::from("graniteTime"), 0.into()),
                    (String::from("holoceneTime"), 0.into()),
                    (String::from("isthmusTime"), 0.into()),
                    (String::from("jovianTime"), 0.into()),
                ]
                .into_iter()
                .collect(),
                ..Default::default()
            },
            ..Default::default()
        };

        let chain_spec: OpChainSpec = genesis.into();

        let hardforks: Vec<_> = chain_spec.hardforks.forks_iter().map(|(h, _)| h).collect();
        let expected_hardforks = vec![
            EthereumHardfork::Frontier.boxed(),
            EthereumHardfork::Homestead.boxed(),
            EthereumHardfork::Tangerine.boxed(),
            EthereumHardfork::SpuriousDragon.boxed(),
            EthereumHardfork::Byzantium.boxed(),
            EthereumHardfork::Constantinople.boxed(),
            EthereumHardfork::Petersburg.boxed(),
            EthereumHardfork::Istanbul.boxed(),
            EthereumHardfork::MuirGlacier.boxed(),
            EthereumHardfork::Berlin.boxed(),
            EthereumHardfork::London.boxed(),
            EthereumHardfork::ArrowGlacier.boxed(),
            EthereumHardfork::GrayGlacier.boxed(),
            EthereumHardfork::Paris.boxed(),
            OpHardfork::Bedrock.boxed(),
            OpHardfork::Regolith.boxed(),
            EthereumHardfork::Shanghai.boxed(),
            OpHardfork::Canyon.boxed(),
            EthereumHardfork::Cancun.boxed(),
            OpHardfork::Ecotone.boxed(),
            OpHardfork::Fjord.boxed(),
            OpHardfork::Granite.boxed(),
            OpHardfork::Holocene.boxed(),
            EthereumHardfork::Prague.boxed(),
            OpHardfork::Isthmus.boxed(),
            OpHardfork::Jovian.boxed(),
            // OpHardfork::Interop.boxed(),
        ];

        for (expected, actual) in expected_hardforks.iter().zip(hardforks.iter()) {
            assert_eq!(&**expected, &**actual);
        }
        assert_eq!(expected_hardforks.len(), hardforks.len());
    }

    #[test]
    fn json_genesis() {
        let geth_genesis = r#"
{
    "config": {
        "chainId": 1301,
        "homesteadBlock": 0,
        "eip150Block": 0,
        "eip155Block": 0,
        "eip158Block": 0,
        "byzantiumBlock": 0,
        "constantinopleBlock": 0,
        "petersburgBlock": 0,
        "istanbulBlock": 0,
        "muirGlacierBlock": 0,
        "berlinBlock": 0,
        "londonBlock": 0,
        "arrowGlacierBlock": 0,
        "grayGlacierBlock": 0,
        "mergeNetsplitBlock": 0,
        "shanghaiTime": 0,
        "cancunTime": 0,
        "bedrockBlock": 0,
        "regolithTime": 0,
        "canyonTime": 0,
        "ecotoneTime": 0,
        "fjordTime": 0,
        "graniteTime": 0,
        "holoceneTime": 1732633200,
        "terminalTotalDifficulty": 0,
        "terminalTotalDifficultyPassed": true,
        "optimism": {
            "eip1559Elasticity": 6,
            "eip1559Denominator": 50,
            "eip1559DenominatorCanyon": 250
        }
    },
    "nonce": "0x0",
    "timestamp": "0x66edad4c",
    "extraData": "0x424544524f434b",
    "gasLimit": "0x1c9c380",
    "difficulty": "0x0",
    "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "coinbase": "0x4200000000000000000000000000000000000011",
    "alloc": {},
    "number": "0x0",
    "gasUsed": "0x0",
    "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "baseFeePerGas": "0x3b9aca00",
    "excessBlobGas": "0x0",
    "blobGasUsed": "0x0"
}
        "#;

        let genesis: Genesis = serde_json::from_str(geth_genesis).unwrap();
        let chainspec = OpChainSpec::from_genesis(genesis);
        assert!(chainspec.is_holocene_active_at_timestamp(1732633200));
    }

    #[test]
    fn json_genesis_mapped_l1_timestamps() {
        let geth_genesis = r#"
{
    "config": {
        "chainId": 1301,
        "homesteadBlock": 0,
        "eip150Block": 0,
        "eip155Block": 0,
        "eip158Block": 0,
        "byzantiumBlock": 0,
        "constantinopleBlock": 0,
        "petersburgBlock": 0,
        "istanbulBlock": 0,
        "muirGlacierBlock": 0,
        "berlinBlock": 0,
        "londonBlock": 0,
        "arrowGlacierBlock": 0,
        "grayGlacierBlock": 0,
        "mergeNetsplitBlock": 0,
        "bedrockBlock": 0,
        "regolithTime": 0,
        "canyonTime": 0,
        "ecotoneTime": 1712633200,
        "fjordTime": 0,
        "graniteTime": 0,
        "holoceneTime": 1732633200,
        "isthmusTime": 1742633200,
        "terminalTotalDifficulty": 0,
        "terminalTotalDifficultyPassed": true,
        "optimism": {
            "eip1559Elasticity": 6,
            "eip1559Denominator": 50,
            "eip1559DenominatorCanyon": 250
        }
    },
    "nonce": "0x0",
    "timestamp": "0x66edad4c",
    "extraData": "0x424544524f434b",
    "gasLimit": "0x1c9c380",
    "difficulty": "0x0",
    "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "coinbase": "0x4200000000000000000000000000000000000011",
    "alloc": {},
    "number": "0x0",
    "gasUsed": "0x0",
    "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "baseFeePerGas": "0x3b9aca00",
    "excessBlobGas": "0x0",
    "blobGasUsed": "0x0"
}
        "#;

        let genesis: Genesis = serde_json::from_str(geth_genesis).unwrap();
        let chainspec = OpChainSpec::from_genesis(genesis);
        assert!(chainspec.is_holocene_active_at_timestamp(1732633200));

        assert!(chainspec.is_shanghai_active_at_timestamp(0));
        assert!(chainspec.is_canyon_active_at_timestamp(0));

        assert!(chainspec.is_ecotone_active_at_timestamp(1712633200));
        assert!(chainspec.is_cancun_active_at_timestamp(1712633200));

        assert!(chainspec.is_prague_active_at_timestamp(1742633200));
        assert!(chainspec.is_isthmus_active_at_timestamp(1742633200));
    }

    #[test]
    fn display_hardorks() {
        let content = BASE_MAINNET.display_hardforks().to_string();
        for eth_hf in EthereumHardfork::VARIANTS {
            assert!(!content.contains(eth_hf.name()));
        }
    }
}
