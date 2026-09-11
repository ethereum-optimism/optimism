//! OP EVM environment configuration.
//!
//! Provides spec ID mapping and `EvmEnv` constructors for Optimism.

use alloy_consensus::BlockHeader;
use alloy_evm::{EvmEnv, eth::NextEvmEnvAttributes};
use alloy_op_hardforks::OpHardforks;
use alloy_primitives::{Address, B256, BlockNumber, BlockTimestamp, ChainId, U256};
use op_revm::OpSpecId;
use revm::{
    context::{BlockEnv, CfgEnv},
    context_interface::block::BlobExcessGasAndPrice,
    primitives::hardfork::SpecId,
};

/// Map the latest active hardfork at the given header to a revm [`OpSpecId`].
pub fn spec(chain_spec: impl OpHardforks, header: impl BlockHeader) -> OpSpecId {
    spec_by_timestamp_after_bedrock(chain_spec, header.timestamp())
}

/// Returns the revm [`OpSpecId`] at the given timestamp.
///
/// # Note
///
/// This is only intended to be used after the Bedrock, when hardforks are activated by
/// timestamp.
pub fn spec_by_timestamp_after_bedrock(chain_spec: impl OpHardforks, timestamp: u64) -> OpSpecId {
    macro_rules! check_forks {
        ($($check:ident => $spec:ident),+ $(,)?) => {
            $(
                if chain_spec.$check(timestamp) {
                    return OpSpecId::$spec;
                }
            )+
        };
    }
    // Forks must be checked newest-first. Lagoon is newer than Karst, so Lagoon is checked
    // before Karst.
    check_forks! {
        is_lagoon_active_at_timestamp => LAGOON,
        is_karst_active_at_timestamp => KARST,
        is_jovian_active_at_timestamp => JOVIAN,
        is_isthmus_active_at_timestamp => ISTHMUS,
        is_holocene_active_at_timestamp => HOLOCENE,
        is_granite_active_at_timestamp => GRANITE,
        is_fjord_active_at_timestamp => FJORD,
        is_ecotone_active_at_timestamp => ECOTONE,
        is_canyon_active_at_timestamp => CANYON,
        is_regolith_active_at_timestamp => REGOLITH,
    }
    OpSpecId::BEDROCK
}

/// Internal helper for constructing EVM environment from block header fields.
struct EvmEnvInput {
    timestamp: BlockTimestamp,
    number: BlockNumber,
    beneficiary: Address,
    mix_hash: Option<B256>,
    difficulty: U256,
    gas_limit: u64,
    base_fee_per_gas: u64,
}

impl EvmEnvInput {
    fn from_block_header(header: impl BlockHeader) -> Self {
        Self {
            timestamp: header.timestamp(),
            number: header.number(),
            beneficiary: header.beneficiary(),
            mix_hash: header.mix_hash(),
            difficulty: header.difficulty(),
            gas_limit: header.gas_limit(),
            base_fee_per_gas: header.base_fee_per_gas().unwrap_or_default(),
        }
    }

    fn for_next(
        parent: impl BlockHeader,
        attributes: NextEvmEnvAttributes,
        base_fee_per_gas: u64,
    ) -> Self {
        Self {
            timestamp: attributes.timestamp,
            number: parent.number() + 1,
            beneficiary: attributes.suggested_fee_recipient,
            mix_hash: Some(attributes.prev_randao),
            difficulty: U256::ZERO,
            gas_limit: attributes.gas_limit,
            base_fee_per_gas,
        }
    }
}

/// Create a new `EvmEnv` with [`OpSpecId`] from a block `header`, `chain_id` and `chain_spec`.
pub fn evm_env_for_op_block(
    header: impl BlockHeader,
    chain_spec: impl OpHardforks,
    chain_id: ChainId,
) -> EvmEnv<OpSpecId> {
    evm_env_for_op(EvmEnvInput::from_block_header(header), chain_spec, chain_id)
}

/// Create a new `EvmEnv` with [`OpSpecId`] from a parent block `header`, `chain_id` and
/// `chain_spec`.
pub fn evm_env_for_op_next_block(
    header: impl BlockHeader,
    attributes: NextEvmEnvAttributes,
    base_fee_per_gas: u64,
    chain_spec: impl OpHardforks,
    chain_id: ChainId,
) -> EvmEnv<OpSpecId> {
    evm_env_for_op(
        EvmEnvInput::for_next(header, attributes, base_fee_per_gas),
        chain_spec,
        chain_id,
    )
}

fn evm_env_for_op(
    input: EvmEnvInput,
    chain_spec: impl OpHardforks,
    chain_id: ChainId,
) -> EvmEnv<OpSpecId> {
    let spec = spec_by_timestamp_after_bedrock(&chain_spec, input.timestamp);
    let cfg_env = CfgEnv::new().with_chain_id(chain_id).with_spec_and_mainnet_gas_params(spec);

    let blob_excess_gas_and_price = spec
        .into_eth_spec()
        .is_enabled_in(SpecId::CANCUN)
        .then_some(BlobExcessGasAndPrice { excess_blob_gas: 0, blob_gasprice: 1 });

    let is_merge_active = spec.into_eth_spec() >= SpecId::MERGE;

    let block_env = BlockEnv {
        number: U256::from(input.number),
        beneficiary: input.beneficiary,
        timestamp: U256::from(input.timestamp),
        difficulty: if is_merge_active { U256::ZERO } else { input.difficulty },
        prevrandao: if is_merge_active { input.mix_hash } else { None },
        gas_limit: input.gas_limit,
        basefee: input.base_fee_per_gas,
        // EIP-4844 excess blob gas of this block, introduced in Cancun
        blob_excess_gas_and_price,
        // EIP-7843 slot number. Not yet used in the OP Stack.
        // TODO(optimism#19853): populate from block header once EIP-7843 is enabled.
        slot_num: 0,
    };

    EvmEnv::new(cfg_env, block_env)
}

/// Create a new `EvmEnv` with [`OpSpecId`] from a `payload`, `chain_id` and `chain_spec`.
#[cfg(feature = "engine")]
pub fn evm_env_for_op_payload(
    payload: &op_alloy::rpc_types_engine::OpExecutionPayload,
    chain_spec: impl OpHardforks,
    chain_id: ChainId,
) -> EvmEnv<OpSpecId> {
    let input = EvmEnvInput {
        timestamp: payload.timestamp(),
        number: payload.block_number(),
        beneficiary: payload.as_v1().fee_recipient,
        mix_hash: Some(payload.as_v1().prev_randao),
        difficulty: payload.as_v1().prev_randao.into(),
        gas_limit: payload.as_v1().gas_limit,
        base_fee_per_gas: payload.as_v1().base_fee_per_gas.saturating_to(),
    };
    evm_env_for_op(input, chain_spec, chain_id)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_hardforks::EthereumHardfork;
    use alloy_op_hardforks::{
        EthereumHardforks, ForkCondition, OP_MAINNET_CANYON_TIMESTAMP,
        OP_MAINNET_ECOTONE_TIMESTAMP, OP_MAINNET_FJORD_TIMESTAMP, OP_MAINNET_GRANITE_TIMESTAMP,
        OP_MAINNET_HOLOCENE_TIMESTAMP, OP_MAINNET_ISTHMUS_TIMESTAMP, OP_MAINNET_JOVIAN_TIMESTAMP,
        OP_MAINNET_KARST_TIMESTAMP, OP_MAINNET_REGOLITH_TIMESTAMP, OpChainHardforks, OpHardfork,
    };
    use alloy_primitives::BlockTimestamp;

    struct FakeHardfork {
        fork: OpHardfork,
        cond: ForkCondition,
    }

    /// Expands to the [`ForkCondition`] for a chronology row's activation kind.
    macro_rules! fork_condition {
        (block) => {
            ForkCondition::Block(0)
        };
        (timestamp) => {
            ForkCondition::Timestamp(0)
        };
    }

    /// Single source of truth for the OP fork chronology, oldest first, driving every
    /// fork-parameterized item in this module. Invokes the given callback macro with an optional
    /// parenthesized argument group followed by one row per fork:
    ///
    /// `case_name => OpHardfork / OpSpecId @ activation, mainnet: TIMESTAMP;`
    ///
    /// - `case_name`: snake-case ident naming the `FakeHardfork` constructor and the generated test
    ///   cases.
    /// - `activation`: `block` or `timestamp` — how the fork activates on a chain.
    /// - `mainnet:`: the OP mainnet activation timestamp constant, or `_` while the fork has no
    ///   scheduled OP mainnet activation.
    ///
    /// Adding a fork row here automatically extends [`FORK_CHRONOLOGY`], the `FakeHardfork`
    /// constructors, and every test generated with `fake_fork_cases!` / `mainnet_fork_cases!`,
    /// so a new fork only needs to be added in this one place.
    macro_rules! with_fork_chronology {
        ($cb:ident) => {
            with_fork_chronology!($cb());
        };
        ($cb:ident($($args:tt)*)) => {
            $cb! {
                ($($args)*)
                bedrock => Bedrock / BEDROCK @ block, mainnet: _;
                regolith => Regolith / REGOLITH @ timestamp, mainnet: OP_MAINNET_REGOLITH_TIMESTAMP;
                canyon => Canyon / CANYON @ timestamp, mainnet: OP_MAINNET_CANYON_TIMESTAMP;
                ecotone => Ecotone / ECOTONE @ timestamp, mainnet: OP_MAINNET_ECOTONE_TIMESTAMP;
                fjord => Fjord / FJORD @ timestamp, mainnet: OP_MAINNET_FJORD_TIMESTAMP;
                granite => Granite / GRANITE @ timestamp, mainnet: OP_MAINNET_GRANITE_TIMESTAMP;
                holocene => Holocene / HOLOCENE @ timestamp, mainnet: OP_MAINNET_HOLOCENE_TIMESTAMP;
                isthmus => Isthmus / ISTHMUS @ timestamp, mainnet: OP_MAINNET_ISTHMUS_TIMESTAMP;
                jovian => Jovian / JOVIAN @ timestamp, mainnet: OP_MAINNET_JOVIAN_TIMESTAMP;
                karst => Karst / KARST @ timestamp, mainnet: OP_MAINNET_KARST_TIMESTAMP;
                lagoon => Lagoon / LAGOON @ timestamp, mainnet: _;
            }
        };
    }

    /// [`with_fork_chronology!`] consumer: defines [`FORK_CHRONOLOGY`] and the per-fork
    /// `FakeHardfork` constructors from the chronology rows.
    macro_rules! define_fork_fixtures {
        (() $($name:ident => $hf:ident / $spec:ident @ $act:ident, mainnet: $ts:tt;)+) => {
            /// The OP fork chronology, oldest first. `Lagoon` is the newest fork.
            const FORK_CHRONOLOGY: [(OpHardfork, OpSpecId); [$(OpSpecId::$spec),+].len()] = [
                $((OpHardfork::$hf, OpSpecId::$spec),)+
            ];

            impl FakeHardfork {
                $(
                    fn $name() -> Self {
                        Self { fork: OpHardfork::$hf, cond: fork_condition!($act) }
                    }
                )+
            }
        };
    }
    with_fork_chronology!(define_fork_fixtures);

    /// [`with_fork_chronology!`] consumer: generates one `#[test]` per fork in the chronology,
    /// named `<module>::<case_name>` exactly like the `test_case` stacks this replaces, each
    /// invoking `body(FakeHardfork::<case_name>(), OpSpecId::<spec>)`.
    macro_rules! fake_fork_cases {
        (($mod_name:ident, $body:expr) $($name:ident => $hf:ident / $spec:ident @ $act:ident, mainnet: $ts:tt;)+) => {
            mod $mod_name {
                use super::*;

                $(
                    #[test]
                    fn $name() {
                        let body: fn(FakeHardfork, OpSpecId) = $body;
                        body(FakeHardfork::$name(), OpSpecId::$spec);
                    }
                )+
            }
        };
    }

    /// [`with_fork_chronology!`] consumer: like `fake_fork_cases!`, but only for forks with a
    /// scheduled OP mainnet activation, each case invoking
    /// `body(<mainnet timestamp>, OpSpecId::<spec>)`. Rows with `mainnet: _` are skipped, so
    /// scheduling a fork's mainnet activation in the chronology automatically adds its case.
    macro_rules! mainnet_fork_cases {
        (($mod_name:ident, $body:expr) $($rows:tt)*) => {
            mod $mod_name {
                use super::*;

                mainnet_fork_cases!(@case $body; $($rows)*);
            }
        };
        (@case $body:expr;) => {};
        (@case $body:expr; $name:ident => $hf:ident / $spec:ident @ $act:ident, mainnet: _; $($rest:tt)*) => {
            mainnet_fork_cases!(@case $body; $($rest)*);
        };
        (@case $body:expr; $name:ident => $hf:ident / $spec:ident @ $act:ident, mainnet: $ts:ident; $($rest:tt)*) => {
            #[test]
            fn $name() {
                let body: fn(BlockTimestamp, OpSpecId) = $body;
                body($ts, OpSpecId::$spec);
            }
            mainnet_fork_cases!(@case $body; $($rest)*);
        };
    }

    impl EthereumHardforks for FakeHardfork {
        fn ethereum_fork_activation(&self, _: EthereumHardfork) -> ForkCondition {
            unimplemented!()
        }
    }

    impl OpHardforks for FakeHardfork {
        fn op_fork_activation(&self, fork: OpHardfork) -> ForkCondition {
            if fork == self.fork { self.cond } else { ForkCondition::Never }
        }
    }

    // The `BLOBBASEFEE` opcode must always be 1 on the OP Stack from Ecotone onward, and no blob
    // env must be present before Ecotone. Post-Jovian the header's `blobGasUsed` carries the
    // block's DA footprint, which must not influence the blob env. The footprint here is from
    // op-mainnet block 152635937.
    with_fork_chronology!(fake_fork_cases(
        evm_env_for_op_next_block_pins_blob_gasprice_to_one,
        |fork, op_spec| {
            let parent_header = Header {
                number: 100,
                timestamp: 1_000_000,
                gas_limit: 60_000_000,
                base_fee_per_gas: Some(1_000_000_000),
                blob_gas_used: Some(30_406_400),
                excess_blob_gas: Some(0),
                ..Default::default()
            };

            let evm_env = evm_env_for_op_next_block(
                &parent_header,
                NextEvmEnvAttributes {
                    timestamp: parent_header.timestamp + 2,
                    suggested_fee_recipient: Default::default(),
                    prev_randao: Default::default(),
                    gas_limit: parent_header.gas_limit,
                    slot_number: None,
                },
                1_000_000_000,
                fork,
                10,
            );

            let blob = evm_env.block_env.blob_excess_gas_and_price;
            if op_spec < OpSpecId::ECOTONE {
                assert!(blob.is_none(), "no blob env must be present before Ecotone");
                return;
            }
            let blob = blob.expect("blob env should be present from Ecotone onward");
            assert_eq!(
                blob.blob_gasprice, 1,
                "BLOBBASEFEE must be pinned to 1 on the OP Stack regardless of the parent DA footprint"
            );
            assert_eq!(
                blob.excess_blob_gas, 0,
                "excess blob gas must be pinned to 0 regardless of the parent DA footprint"
            );
        }
    ));

    // A chain whose newest active fork is the given one must resolve to the expected spec.
    with_fork_chronology!(fake_fork_cases(
        test_spec_maps_hardfork_successfully,
        |fork, expected_spec| {
            let header = Header::default();
            let actual_spec = spec(fork, &header);

            assert_eq!(actual_spec, expected_spec);
        }
    ));

    // OP mainnet's scheduled activation timestamps must resolve to the matching spec.
    with_fork_chronology!(mainnet_fork_cases(
        test_op_spec_maps_hardfork_successfully,
        |timestamp, expected_spec| {
            let fork = OpChainHardforks::op_mainnet();
            let header = Header { timestamp, ..Default::default() };
            let actual_spec = spec(&fork, &header);

            assert_eq!(actual_spec, expected_spec);
        }
    ));

    /// The revm [`SpecId`] equivalent of the given L1 hardfork, for the L1 forks that OP forks
    /// imply. The enums use different names for the merge fork (`Paris`/`MERGE`).
    fn eth_spec_id(l1_fork: EthereumHardfork) -> SpecId {
        match l1_fork {
            EthereumHardfork::Paris => SpecId::MERGE,
            EthereumHardfork::Shanghai => SpecId::SHANGHAI,
            EthereumHardfork::Cancun => SpecId::CANCUN,
            EthereumHardfork::Prague => SpecId::PRAGUE,
            EthereumHardfork::Osaka => SpecId::OSAKA,
            _ => panic!("not an L1 fork implied by an OP fork: {l1_fork}"),
        }
    }

    /// Pins op-revm's hand-written `OpSpecId::into_eth_spec` to the canonical
    /// OP fork → implied L1 fork mapping in alloy-op-hardforks. op-revm deliberately has no
    /// alloy dependencies, so divergence between the two is caught here, where both crates
    /// meet.
    #[test]
    fn op_spec_eth_base_matches_canonical_implied_l1_fork() {
        for (hardfork, spec_id) in FORK_CHRONOLOGY {
            assert_eq!(
                spec_id.into_eth_spec(),
                eth_spec_id(hardfork.implied_l1_fork()),
                "{hardfork}: OpSpecId::into_eth_spec diverges from OpHardfork::implied_l1_fork"
            );
        }
    }

    /// Builds a chain config where every fork up to and including `forks[idx]` is active at
    /// timestamp 1, mirroring a real chain that has progressed through the schedule. This is what
    /// exercises the newest-first precedence in `spec_by_timestamp_after_bedrock`: with several
    /// forks simultaneously active, the resolver must return the newest one.
    fn cumulative_hardforks(idx: usize) -> OpChainHardforks {
        let activations = FORK_CHRONOLOGY.iter().take(idx + 1).map(|(fork, _)| {
            let cond = if *fork == OpHardfork::Bedrock {
                ForkCondition::Block(0)
            } else {
                ForkCondition::Timestamp(1)
            };
            (*fork, cond)
        });
        OpChainHardforks::new(activations)
    }

    /// Locks in the corrected newest-first fork ordering: when forks are activated cumulatively
    /// (as on a real chain), the resolver must return the newest active spec, not an older one.
    /// In particular, a post-Lagoon block resolves to `LAGOON`, not `KARST`.
    #[test]
    fn test_spec_resolves_newest_active_fork() {
        for (idx, (fork, expected_spec)) in FORK_CHRONOLOGY.iter().enumerate() {
            let chain = cumulative_hardforks(idx);
            let header = Header { timestamp: 1, ..Default::default() };
            let actual_spec = spec(&chain, &header);
            assert_eq!(
                actual_spec, *expected_spec,
                "with forks active up to {fork:?}, expected newest spec {expected_spec:?}",
            );
        }
    }

    /// Conformance guard: the eth base spec must be non-decreasing across the OP fork chronology.
    /// A newer OP fork must never map to an older eth base (e.g. LAGOON must not downgrade Karst's
    /// OSAKA base back to PRAGUE).
    #[test]
    fn test_eth_base_is_monotonic_across_chronology() {
        let mut prev_eth = SpecId::default();
        let mut first = true;
        for (fork, op_spec) in FORK_CHRONOLOGY {
            let eth = op_spec.into_eth_spec();
            if !first {
                assert!(
                    eth >= prev_eth,
                    "{fork:?} ({op_spec:?}) eth base {eth:?} is older than the previous fork's {prev_eth:?}",
                );
            }
            prev_eth = eth;
            first = false;
        }
        // Sanity-check the specific pairing the bug was about.
        assert_eq!(OpSpecId::KARST.into_eth_spec(), SpecId::OSAKA);
        assert_eq!(OpSpecId::LAGOON.into_eth_spec(), SpecId::OSAKA);
        assert!(OpSpecId::LAGOON.into_eth_spec() >= OpSpecId::KARST.into_eth_spec());
    }

    /// The EIP-7825 tx gas limit cap must apply from Osaka-based forks (Karst) onward via revm's
    /// effective cap, without the env builder overriding the raw config field (the estimate path
    /// uses the effective cap directly since paradigmxyz/reth#25612; see issue #21583).
    #[test]
    fn osaka_forks_effective_tx_gas_limit_cap() {
        use revm::context_interface::Cfg;

        for (idx, (fork, op_spec)) in FORK_CHRONOLOGY.iter().enumerate() {
            let chain = cumulative_hardforks(idx);
            let header = Header { timestamp: 1, gas_limit: 60_000_000, ..Default::default() };
            let env = evm_env_for_op_block(&header, &chain, 10);

            assert_eq!(
                env.cfg_env.tx_gas_limit_cap, None,
                "with forks active up to {fork:?} ({op_spec:?}), the raw cap must stay unset",
            );
            let expected = if op_spec.into_eth_spec().is_enabled_in(SpecId::OSAKA) {
                revm::primitives::eip7825::TX_GAS_LIMIT_CAP
            } else {
                u64::MAX
            };
            assert_eq!(
                env.cfg_env.tx_gas_limit_cap(),
                expected,
                "with forks active up to {fork:?} ({op_spec:?}), effective tx_gas_limit_cap mismatch",
            );
        }
    }
}
