use alloy_consensus::BlockHeader;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ExecutionPayloadEnvelopeV2, ExecutionPayloadV1};
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4,
    OpPayloadAttributes,
};
use reth_consensus::ConsensusError;
use reth_node_api::{
    BuiltPayload, EngineApiValidator, EngineTypes, NodePrimitives, PayloadValidator,
    payload::{
        EngineApiMessageVersion, EngineObjectValidationError, MessageValidationKind,
        NewPayloadError, PayloadOrAttributes, PayloadTypes, VersionSpecificValidationError,
        validate_parent_beacon_block_root_presence,
    },
    validate_version_specific_fields,
};
use reth_optimism_consensus::isthmus;
use reth_optimism_forks::OpHardforks;
use reth_optimism_payload_builder::{OpExecutionPayloadValidator, OpPayloadTypes};
use reth_optimism_primitives::{L2_TO_L1_MESSAGE_PASSER_ADDRESS, OpBlock};
use reth_primitives_traits::{Block, RecoveredBlock, SealedBlock, SignedTransaction};
use reth_provider::StateProviderFactory;
use reth_trie_common::{HashedPostState, KeyHasher};
use std::{marker::PhantomData, sync::Arc};

/// The types used in the optimism beacon consensus engine.
#[derive(Debug, Default, Clone, serde::Deserialize, serde::Serialize)]
#[non_exhaustive]
pub struct OpEngineTypes<T: PayloadTypes = OpPayloadTypes> {
    _marker: PhantomData<T>,
}

impl<T: PayloadTypes<ExecutionData = OpExecutionData>> PayloadTypes for OpEngineTypes<T> {
    type ExecutionData = T::ExecutionData;
    type BuiltPayload = T::BuiltPayload;
    type PayloadAttributes = T::PayloadAttributes;
    type PayloadBuilderAttributes = T::PayloadBuilderAttributes;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
    ) -> <T as PayloadTypes>::ExecutionData {
        OpExecutionData::from_block_unchecked(
            block.hash(),
            &block.into_block().into_ethereum_block(),
        )
    }
}

impl<T: PayloadTypes<ExecutionData = OpExecutionData>> EngineTypes for OpEngineTypes<T>
where
    T::BuiltPayload: BuiltPayload<Primitives: NodePrimitives<Block = OpBlock>>
        + TryInto<ExecutionPayloadV1>
        + TryInto<ExecutionPayloadEnvelopeV2>
        + TryInto<OpExecutionPayloadEnvelopeV3>
        + TryInto<OpExecutionPayloadEnvelopeV4>,
{
    type ExecutionPayloadEnvelopeV1 = ExecutionPayloadV1;
    type ExecutionPayloadEnvelopeV2 = ExecutionPayloadEnvelopeV2;
    type ExecutionPayloadEnvelopeV3 = OpExecutionPayloadEnvelopeV3;
    type ExecutionPayloadEnvelopeV4 = OpExecutionPayloadEnvelopeV4;
    type ExecutionPayloadEnvelopeV5 = OpExecutionPayloadEnvelopeV4;
    type ExecutionPayloadEnvelopeV6 = OpExecutionPayloadEnvelopeV4;
}

/// Validator for Optimism engine API.
#[derive(Debug)]
pub struct OpEngineValidator<P, Tx, ChainSpec> {
    inner: OpExecutionPayloadValidator<ChainSpec>,
    provider: P,
    hashed_addr_l2tol1_msg_passer: B256,
    phantom: PhantomData<Tx>,
}

impl<P, Tx, ChainSpec> OpEngineValidator<P, Tx, ChainSpec> {
    /// Instantiates a new validator.
    pub fn new<KH: KeyHasher>(chain_spec: Arc<ChainSpec>, provider: P) -> Self {
        let hashed_addr_l2tol1_msg_passer = KH::hash_key(L2_TO_L1_MESSAGE_PASSER_ADDRESS);
        Self {
            inner: OpExecutionPayloadValidator::new(chain_spec),
            provider,
            hashed_addr_l2tol1_msg_passer,
            phantom: PhantomData,
        }
    }
}

impl<P, Tx, ChainSpec> Clone for OpEngineValidator<P, Tx, ChainSpec>
where
    P: Clone,
    ChainSpec: OpHardforks,
{
    fn clone(&self) -> Self {
        Self {
            inner: OpExecutionPayloadValidator::new(self.inner.clone()),
            provider: self.provider.clone(),
            hashed_addr_l2tol1_msg_passer: self.hashed_addr_l2tol1_msg_passer,
            phantom: Default::default(),
        }
    }
}

impl<P, Tx, ChainSpec> OpEngineValidator<P, Tx, ChainSpec>
where
    ChainSpec: OpHardforks,
{
    /// Returns the chain spec used by the validator.
    #[inline]
    pub fn chain_spec(&self) -> &ChainSpec {
        self.inner.chain_spec()
    }
}

impl<P, Tx, ChainSpec, Types> PayloadValidator<Types> for OpEngineValidator<P, Tx, ChainSpec>
where
    P: StateProviderFactory + Unpin + 'static,
    Tx: SignedTransaction + Unpin + 'static,
    ChainSpec: OpHardforks + Send + Sync + 'static,
    Types: PayloadTypes<ExecutionData = OpExecutionData>,
{
    type Block = alloy_consensus::Block<Tx>;

    fn validate_block_post_execution_with_hashed_state(
        &self,
        state_updates: &HashedPostState,
        block: &RecoveredBlock<Self::Block>,
    ) -> Result<(), ConsensusError> {
        if self.chain_spec().is_isthmus_active_at_timestamp(block.timestamp()) {
            let Ok(state) = self.provider.state_by_block_hash(block.parent_hash()) else {
                // FIXME: we don't necessarily have access to the parent block here because the
                // parent block isn't necessarily part of the canonical chain yet. Instead this
                // function should receive the list of in memory blocks as input
                return Ok(());
            };
            let predeploy_storage_updates = state_updates
                .storages
                .get(&self.hashed_addr_l2tol1_msg_passer)
                .cloned()
                .unwrap_or_default();
            isthmus::verify_withdrawals_root_prehashed(
                predeploy_storage_updates,
                state,
                block.header(),
            )
            .map_err(|err| {
                ConsensusError::Other(format!("failed to verify block post-execution: {err}"))
            })?
        }

        Ok(())
    }

    fn convert_payload_to_block(
        &self,
        payload: OpExecutionData,
    ) -> Result<SealedBlock<Self::Block>, NewPayloadError> {
        self.inner.ensure_well_formed_payload(payload).map_err(NewPayloadError::other)
    }
}

impl<Types, P, Tx, ChainSpec> EngineApiValidator<Types> for OpEngineValidator<P, Tx, ChainSpec>
where
    Types: PayloadTypes<
            PayloadAttributes = OpPayloadAttributes,
            ExecutionData = OpExecutionData,
            BuiltPayload: BuiltPayload<Primitives: NodePrimitives<SignedTx = Tx>>,
        >,
    P: StateProviderFactory + Unpin + 'static,
    Tx: SignedTransaction + Unpin + 'static,
    ChainSpec: OpHardforks + Send + Sync + 'static,
{
    fn validate_version_specific_fields(
        &self,
        version: EngineApiMessageVersion,
        payload_or_attrs: PayloadOrAttributes<
            '_,
            Types::ExecutionData,
            <Types as PayloadTypes>::PayloadAttributes,
        >,
    ) -> Result<(), EngineObjectValidationError> {
        validate_withdrawals_presence(
            self.chain_spec(),
            version,
            payload_or_attrs.message_validation_kind(),
            payload_or_attrs.timestamp(),
            payload_or_attrs.withdrawals().is_some(),
        )?;
        validate_parent_beacon_block_root_presence(
            self.chain_spec(),
            version,
            payload_or_attrs.message_validation_kind(),
            payload_or_attrs.timestamp(),
            payload_or_attrs.parent_beacon_block_root().is_some(),
        )
    }

    fn ensure_well_formed_attributes(
        &self,
        version: EngineApiMessageVersion,
        attributes: &<Types as PayloadTypes>::PayloadAttributes,
    ) -> Result<(), EngineObjectValidationError> {
        validate_version_specific_fields(
            self.chain_spec(),
            version,
            PayloadOrAttributes::<OpExecutionData, OpPayloadAttributes>::PayloadAttributes(
                attributes,
            ),
        )?;

        if attributes.gas_limit.is_none() {
            return Err(EngineObjectValidationError::InvalidParams(
                "MissingGasLimitInPayloadAttributes".to_string().into(),
            ));
        }

        if self
            .chain_spec()
            .is_holocene_active_at_timestamp(attributes.payload_attributes.timestamp)
        {
            let (elasticity, denominator) =
                attributes.decode_eip_1559_params().ok_or_else(|| {
                    EngineObjectValidationError::InvalidParams(
                        "MissingEip1559ParamsInPayloadAttributes".to_string().into(),
                    )
                })?;

            if elasticity != 0 && denominator == 0 {
                return Err(EngineObjectValidationError::InvalidParams(
                    "Eip1559ParamsDenominatorZero".to_string().into(),
                ));
            } else if denominator != 0 && elasticity == 0 {
                return Err(EngineObjectValidationError::InvalidParams(
                    "Eip1559ParamsElasticityZero".to_string().into(),
                ));
            }
        }

        if self.chain_spec().is_jovian_active_at_timestamp(attributes.payload_attributes.timestamp)
        {
            if attributes.min_base_fee.is_none() {
                return Err(EngineObjectValidationError::InvalidParams(
                    "MissingMinBaseFeeInPayloadAttributes".to_string().into(),
                ));
            }
        } else if attributes.min_base_fee.is_some() {
            return Err(EngineObjectValidationError::InvalidParams(
                "MinBaseFeeNotAllowedBeforeJovian".to_string().into(),
            ));
        }

        Ok(())
    }
}

/// Validates the presence of the `withdrawals` field according to the payload timestamp.
///
/// After Canyon, withdrawals field must be [Some].
/// Before Canyon, withdrawals field must be [None];
///
/// Canyon activates the Shanghai EIPs, see the Canyon specs for more details:
/// <https://github.com/ethereum-optimism/optimism/blob/ab926c5fd1e55b5c864341c44842d6d1ca679d99/specs/superchain-upgrades.md#canyon>
pub fn validate_withdrawals_presence(
    chain_spec: impl OpHardforks,
    version: EngineApiMessageVersion,
    message_validation_kind: MessageValidationKind,
    timestamp: u64,
    has_withdrawals: bool,
) -> Result<(), EngineObjectValidationError> {
    let is_shanghai = chain_spec.is_canyon_active_at_timestamp(timestamp);

    match version {
        EngineApiMessageVersion::V1 => {
            if has_withdrawals {
                return Err(message_validation_kind
                    .to_error(VersionSpecificValidationError::WithdrawalsNotSupportedInV1));
            }
            if is_shanghai {
                return Err(message_validation_kind
                    .to_error(VersionSpecificValidationError::NoWithdrawalsPostShanghai));
            }
        }
        EngineApiMessageVersion::V2 |
        EngineApiMessageVersion::V3 |
        EngineApiMessageVersion::V4 |
        EngineApiMessageVersion::V5 => {
            if is_shanghai && !has_withdrawals {
                return Err(message_validation_kind
                    .to_error(VersionSpecificValidationError::NoWithdrawalsPostShanghai));
            }
            if !is_shanghai && has_withdrawals {
                return Err(message_validation_kind
                    .to_error(VersionSpecificValidationError::HasWithdrawalsPreShanghai));
            }
        }
    };

    Ok(())
}

#[cfg(test)]
mod test {
    use super::*;

    use crate::engine;
    use alloy_op_hardforks::BASE_SEPOLIA_JOVIAN_TIMESTAMP;
    use alloy_primitives::{Address, B64, B256, b64};
    use alloy_rpc_types_engine::PayloadAttributes;
    use reth_optimism_chainspec::BASE_SEPOLIA;
    use reth_provider::noop::NoopProvider;
    use reth_trie_common::KeccakKeyHasher;

    macro_rules! assert_invalid_params_error {
        ($result:expr, $msg:expr) => {{
            let err = $result.expect_err("expected InvalidParams error");
            match err {
                EngineObjectValidationError::InvalidParams(inner) => {
                    assert_eq!(inner.to_string(), $msg);
                }
                other => panic!("expected InvalidParams, got {other:?}"),
            }
        }};
    }

    const fn get_attributes(
        eip_1559_params: Option<B64>,
        min_base_fee: Option<u64>,
        timestamp: u64,
    ) -> OpPayloadAttributes {
        OpPayloadAttributes {
            gas_limit: Some(1000),
            eip_1559_params,
            min_base_fee,
            transactions: None,
            no_tx_pool: None,
            payload_attributes: PayloadAttributes {
                timestamp,
                prev_randao: B256::ZERO,
                suggested_fee_recipient: Address::ZERO,
                withdrawals: Some(vec![]),
                parent_beacon_block_root: Some(B256::ZERO),
            },
        }
    }

    #[test]
    fn test_well_formed_attributes_pre_holocene() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(None, None, 1732633199);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert!(result.is_ok());
    }

    #[test]
    fn test_well_formed_attributes_holocene_no_eip1559_params() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(None, None, 1732633200);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert_invalid_params_error!(result, "MissingEip1559ParamsInPayloadAttributes");
    }

    #[test]
    fn test_well_formed_attributes_holocene_eip1559_params_zero_denominator() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(Some(b64!("0000000000000008")), None, 1732633200);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert_invalid_params_error!(result, "Eip1559ParamsDenominatorZero");
    }

    #[test]
    fn test_well_formed_attributes_holocene_eip1559_params_zero_elasticity() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(Some(b64!("0000000800000000")), None, 1732633200);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert_invalid_params_error!(result, "Eip1559ParamsElasticityZero");
    }

    #[test]
    fn test_well_formed_attributes_holocene_valid() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(Some(b64!("0000000800000008")), None, 1732633200);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert!(result.is_ok());
    }

    #[test]
    fn test_well_formed_attributes_holocene_valid_all_zero() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(Some(b64!("0000000000000000")), None, 1732633200);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert!(result.is_ok());
    }

    #[test]
    fn test_well_formed_attributes_jovian_valid() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes =
            get_attributes(Some(b64!("0000000000000000")), Some(1), BASE_SEPOLIA_JOVIAN_TIMESTAMP);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert!(result.is_ok());
    }

    /// After Jovian (and holocene), eip1559 params must be Some
    #[test]
    fn test_malformed_attributes_jovian_with_eip_1559_params_none() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(None, Some(1), BASE_SEPOLIA_JOVIAN_TIMESTAMP);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert_invalid_params_error!(result, "MissingEip1559ParamsInPayloadAttributes");
    }

    /// Before Jovian, min base fee must be None
    #[test]
    fn test_malformed_attributes_pre_jovian_with_min_base_fee() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes = get_attributes(Some(b64!("0000000000000000")), Some(1), 1732633200);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert_invalid_params_error!(result, "MinBaseFeeNotAllowedBeforeJovian");
    }

    /// After Jovian, min base fee must be Some
    #[test]
    fn test_malformed_attributes_post_jovian_with_min_base_fee_none() {
        let validator = OpEngineValidator::new::<KeccakKeyHasher>(
            BASE_SEPOLIA.clone(),
            NoopProvider::default(),
        );
        let attributes =
            get_attributes(Some(b64!("0000000000000000")), None, BASE_SEPOLIA_JOVIAN_TIMESTAMP);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert_invalid_params_error!(result, "MissingMinBaseFeeInPayloadAttributes");
    }
}
