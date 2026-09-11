use alloy_consensus::BlockHeader;
use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_engine::{ExecutionPayloadEnvelopeV2, ExecutionPayloadV1};
use op_alloy_consensus::{POST_EXEC_TX_TYPE_ID, PostExecPayload};
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3,
    OpExecutionPayloadEnvelopeV4,
};
use reth_consensus::ConsensusError;
use reth_node_api::{
    BuiltPayload, EngineApiValidator, EngineTypes, InsertBlockErrorKind, NodePrimitives,
    PayloadValidator,
    payload::{
        EngineApiMessageVersion, EngineObjectValidationError, MessageValidationKind,
        NewPayloadError, PayloadOrAttributes, PayloadTypes, VersionSpecificValidationError,
        validate_parent_beacon_block_root_presence,
    },
    validate_version_specific_fields,
};
use reth_optimism_consensus::isthmus;
use reth_optimism_evm::metrics::{
    PostExecValidationFailureReason, record_post_exec_validation_failure,
};
use reth_optimism_forks::OpHardforks;
use reth_optimism_payload_builder::{
    OpExecData, OpExecutionPayloadValidator, OpPayloadAttrs, OpPayloadTypes,
};
use reth_optimism_primitives::{L2_TO_L1_MESSAGE_PASSER_ADDRESS, OpBlock};
use reth_primitives_traits::{Block, RecoveredBlock, SealedBlock, SignedTransaction};
use reth_provider::{ProviderResult, StateProviderBox, StateProviderFactory};
use reth_trie_common::{HashedPostState, KeyHasher};
use std::{marker::PhantomData, sync::Arc};

/// The types used in the optimism beacon consensus engine.
#[derive(Debug, Default, Clone, serde::Deserialize, serde::Serialize)]
#[non_exhaustive]
pub struct OpEngineTypes<T: PayloadTypes = OpPayloadTypes> {
    _marker: PhantomData<T>,
}

impl<T: PayloadTypes<ExecutionData = OpExecData>> PayloadTypes for OpEngineTypes<T>
where
    OpExecData: From<<T as PayloadTypes>::BuiltPayload>,
{
    type ExecutionData = T::ExecutionData;
    type BuiltPayload = T::BuiltPayload;
    type PayloadAttributes = T::PayloadAttributes;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
        _bal: Option<alloy_primitives::Bytes>,
    ) -> <T as PayloadTypes>::ExecutionData {
        OpExecData(OpExecutionData::from(
            OpExecutionPayloadEnvelope::from_block_unchecked(
                block.hash(),
                &block.into_block().into_ethereum_block(),
            )
            .expect("built OP blocks must normalize"),
        ))
    }
}

impl<T: PayloadTypes<ExecutionData = OpExecData>> EngineTypes for OpEngineTypes<T>
where
    OpExecData: From<<T as PayloadTypes>::BuiltPayload>,
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

fn has_invalid_post_exec_encoding(transactions: &[Bytes]) -> bool {
    transactions.iter().any(|encoded| {
        let bytes = encoded.as_ref();
        bytes.first() == Some(&POST_EXEC_TX_TYPE_ID) &&
            PostExecPayload::from_rlp_bytes(&bytes[1..]).is_err()
    })
}

impl<P, Tx, ChainSpec, Types> PayloadValidator<Types> for OpEngineValidator<P, Tx, ChainSpec>
where
    P: StateProviderFactory + Unpin + 'static,
    Tx: SignedTransaction + Unpin + 'static,
    ChainSpec: OpHardforks + Send + Sync + 'static,
    Types: PayloadTypes<ExecutionData = OpExecData>,
{
    type Block = alloy_consensus::Block<Tx>;

    fn validate_block_post_execution_with_hashed_state<'a>(
        &self,
        state_updates: impl FnOnce() -> &'a HashedPostState,
        block: &RecoveredBlock<Self::Block>,
        parent_state: impl FnOnce() -> ProviderResult<StateProviderBox>,
    ) -> Result<(), InsertBlockErrorKind> {
        if self.chain_spec().is_isthmus_active_at_timestamp(block.timestamp()) {
            let state = parent_state()?;
            let predeploy_storage_updates = state_updates()
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
                ConsensusError::msg(format!("failed to verify block post-execution: {err}"))
            })?
        }

        Ok(())
    }

    fn convert_payload_to_block(
        &self,
        payload: OpExecData,
    ) -> Result<SealedBlock<Self::Block>, NewPayloadError> {
        if has_invalid_post_exec_encoding(payload.0.payload.transactions()) {
            record_post_exec_validation_failure(
                PostExecValidationFailureReason::InvalidEncodingOrSchema,
            );
        }
        self.inner.ensure_well_formed_payload(payload.0).map_err(NewPayloadError::other)
    }
}

impl<Types, P, Tx, ChainSpec> EngineApiValidator<Types> for OpEngineValidator<P, Tx, ChainSpec>
where
    Types: PayloadTypes<
            PayloadAttributes = OpPayloadAttrs,
            ExecutionData = OpExecData,
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
            PayloadOrAttributes::<OpExecData, OpPayloadAttrs>::PayloadAttributes(attributes),
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
        EngineApiMessageVersion::V5 |
        EngineApiMessageVersion::V6 => {
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

    use crate::{OpNode, engine};
    use alloy_consensus::{BlockBody, Header};
    use alloy_op_hardforks::OP_SEPOLIA_JOVIAN_TIMESTAMP;
    use alloy_primitives::{Address, B64, B256, b64};
    use alloy_rpc_types_engine::PayloadAttributes;
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use reth_db_common::init::init_genesis;
    use reth_optimism_chainspec::OP_SEPOLIA;
    use reth_optimism_primitives::OpTransactionSigned;
    use reth_provider::{
        noop::NoopProvider, providers::BlockchainProvider,
        test_utils::create_test_provider_factory_with_node_types,
    };
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

    #[test]
    fn detects_invalid_post_exec_encoding_or_schema() {
        let valid_payload = PostExecPayload {
            version: op_alloy_consensus::POST_EXEC_PAYLOAD_VERSION,
            block_number: 1,
            gas_refund_entries: vec![],
        };
        let mut valid = vec![POST_EXEC_TX_TYPE_ID];
        valid.extend_from_slice(&valid_payload.to_rlp_bytes());

        let mut invalid_schema = vec![POST_EXEC_TX_TYPE_ID];
        invalid_schema
            .extend_from_slice(&PostExecPayload { version: 2, ..valid_payload }.to_rlp_bytes());

        assert!(!has_invalid_post_exec_encoding(&[Bytes::from(valid)]));
        assert!(has_invalid_post_exec_encoding(&[Bytes::from(invalid_schema)]));
        assert!(!has_invalid_post_exec_encoding(&[Bytes::from_static(&[0x02, 0x01])]));
    }

    fn get_attributes(
        eip_1559_params: Option<B64>,
        min_base_fee: Option<u64>,
        timestamp: u64,
    ) -> OpPayloadAttrs {
        OpPayloadAttrs(OpPayloadAttributes {
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
                slot_number: None,
                target_gas_limit: None,
            },
        })
    }

    #[test]
    fn test_well_formed_attributes_pre_holocene() {
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
        let attributes =
            get_attributes(Some(b64!("0000000000000000")), Some(1), OP_SEPOLIA_JOVIAN_TIMESTAMP);

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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
        let attributes = get_attributes(None, Some(1), OP_SEPOLIA_JOVIAN_TIMESTAMP);

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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
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
        let validator =
            OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), NoopProvider::default());
        let attributes =
            get_attributes(Some(b64!("0000000000000000")), None, OP_SEPOLIA_JOVIAN_TIMESTAMP);

        let result = <engine::OpEngineValidator<_, _, _> as EngineApiValidator<
            OpEngineTypes,
        >>::ensure_well_formed_attributes(
            &validator, EngineApiMessageVersion::V3, &attributes,
        );
        assert_invalid_params_error!(result, "MissingMinBaseFeeInPayloadAttributes");
    }

    fn isthmus_block(
        parent_hash: B256,
        withdrawals_root: B256,
    ) -> RecoveredBlock<alloy_consensus::Block<OpTransactionSigned>> {
        let header = Header {
            parent_hash,
            timestamp: OP_SEPOLIA_JOVIAN_TIMESTAMP,
            withdrawals_root: Some(withdrawals_root),
            ..Default::default()
        };
        let body = BlockBody::<OpTransactionSigned> { transactions: vec![], ..Default::default() };
        RecoveredBlock::new_sealed(
            SealedBlock::seal_slow(alloy_consensus::Block { header, body }),
            vec![],
        )
    }

    #[test]
    fn isthmus_uses_engine_parent_state_for_withdrawals_root() {
        let provider_factory =
            create_test_provider_factory_with_node_types::<OpNode>(OP_SEPOLIA.clone());
        init_genesis(&provider_factory).unwrap();
        let provider = BlockchainProvider::new(provider_factory).unwrap();
        let unavailable_parent = B256::repeat_byte(0x11);
        assert!(
            provider.state_by_block_hash(unavailable_parent).is_err(),
            "fixture parent must be unavailable from canonical state"
        );

        let validator = OpEngineValidator::new::<KeccakKeyHasher>(OP_SEPOLIA.clone(), provider);
        let block = isthmus_block(unavailable_parent, B256::repeat_byte(0xab));
        let hashed_state = HashedPostState::default();
        let result = <OpEngineValidator<_, OpTransactionSigned, _> as PayloadValidator<
            OpPayloadTypes,
        >>::validate_block_post_execution_with_hashed_state(
            &validator,
            || &hashed_state,
            &block,
            || NoopProvider::default().latest(),
        );

        assert!(
            matches!(result, Err(InsertBlockErrorKind::Consensus(_))),
            "mismatched withdrawals root must be a consensus error, got {result:?}"
        );
    }
}
