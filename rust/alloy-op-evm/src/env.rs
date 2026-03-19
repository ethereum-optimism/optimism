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
    check_forks! {
        is_interop_active_at_timestamp => INTEROP,
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
        OP_MAINNET_REGOLITH_TIMESTAMP, OpChainHardforks, OpHardfork,
    };
    use alloy_primitives::BlockTimestamp;

    struct FakeHardfork {
        fork: OpHardfork,
        cond: ForkCondition,
    }

    macro_rules! fake_hardfork_constructors {
        (timestamp: $($ts_name:ident => $ts_fork:ident),+ $(,)?; block: $($blk_name:ident => $blk_fork:ident),+ $(,)?) => {
            impl FakeHardfork {
                $(
                    fn $ts_name() -> Self {
                        Self { fork: OpHardfork::$ts_fork, cond: ForkCondition::Timestamp(0) }
                    }
                )+
                $(
                    fn $blk_name() -> Self {
                        Self { fork: OpHardfork::$blk_fork, cond: ForkCondition::Block(0) }
                    }
                )+
            }
        };
    }

    fake_hardfork_constructors! {
        timestamp:
            interop => Interop,
            jovian => Jovian,
            isthmus => Isthmus,
            holocene => Holocene,
            granite => Granite,
            fjord => Fjord,
            ecotone => Ecotone,
            canyon => Canyon,
            regolith => Regolith;
        block:
            bedrock => Bedrock,
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

    #[test_case::test_case(FakeHardfork::interop(), OpSpecId::INTEROP; "Interop")]
    #[test_case::test_case(FakeHardfork::jovian(), OpSpecId::JOVIAN; "Jovian")]
    #[test_case::test_case(FakeHardfork::isthmus(), OpSpecId::ISTHMUS; "Isthmus")]
    #[test_case::test_case(FakeHardfork::holocene(), OpSpecId::HOLOCENE; "Holocene")]
    #[test_case::test_case(FakeHardfork::granite(), OpSpecId::GRANITE; "Granite")]
    #[test_case::test_case(FakeHardfork::fjord(), OpSpecId::FJORD; "Fjord")]
    #[test_case::test_case(FakeHardfork::ecotone(), OpSpecId::ECOTONE; "Ecotone")]
    #[test_case::test_case(FakeHardfork::canyon(), OpSpecId::CANYON; "Canyon")]
    #[test_case::test_case(FakeHardfork::regolith(), OpSpecId::REGOLITH; "Regolith")]
    #[test_case::test_case(FakeHardfork::bedrock(), OpSpecId::BEDROCK; "Bedrock")]
    fn test_spec_maps_hardfork_successfully(fork: impl OpHardforks, expected_spec: OpSpecId) {
        let header = Header::default();
        let actual_spec = spec(fork, &header);

        assert_eq!(actual_spec, expected_spec);
    }

    #[test_case::test_case(OP_MAINNET_JOVIAN_TIMESTAMP, OpSpecId::JOVIAN; "Jovian")]
    #[test_case::test_case(OP_MAINNET_ISTHMUS_TIMESTAMP, OpSpecId::ISTHMUS; "Isthmus")]
    #[test_case::test_case(OP_MAINNET_HOLOCENE_TIMESTAMP, OpSpecId::HOLOCENE; "Holocene")]
    #[test_case::test_case(OP_MAINNET_GRANITE_TIMESTAMP, OpSpecId::GRANITE; "Granite")]
    #[test_case::test_case(OP_MAINNET_FJORD_TIMESTAMP, OpSpecId::FJORD; "Fjord")]
    #[test_case::test_case(OP_MAINNET_ECOTONE_TIMESTAMP, OpSpecId::ECOTONE; "Ecotone")]
    #[test_case::test_case(OP_MAINNET_CANYON_TIMESTAMP, OpSpecId::CANYON; "Canyon")]
    #[test_case::test_case(OP_MAINNET_REGOLITH_TIMESTAMP, OpSpecId::REGOLITH; "Regolith")]
    fn test_op_spec_maps_hardfork_successfully(timestamp: BlockTimestamp, expected_spec: OpSpecId) {
        let fork = OpChainHardforks::op_mainnet();
        let header = Header { timestamp, ..Default::default() };
        let actual_spec = spec(&fork, &header);

        assert_eq!(actual_spec, expected_spec);
    }
}
