//! Payload related types

use std::{fmt::Debug, sync::Arc};

use alloy_consensus::{Block, BlockHeader};
use alloy_eips::{
    eip1559::BaseFeeParams, eip2718::Decodable2718, eip4895::Withdrawals, eip7685::Requests,
};
use alloy_primitives::{Address, B64, B256, Bytes, U256, keccak256};
use alloy_rlp::Encodable;
use alloy_rpc_types_engine::{
    BlobsBundleV1, ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2, ExecutionPayloadV1,
    ExecutionPayloadV3, PayloadId,
};
use op_alloy_consensus::{EIP1559ParamError, encode_holocene_extra_data, encode_jovian_extra_data};
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
};
use reth_chainspec::EthChainSpec;
use reth_optimism_evm::OpNextBlockEnvAttributes;
use reth_optimism_forks::OpHardforks;
use reth_payload_builder::{EthPayloadBuilderAttributes, PayloadBuilderError};
use reth_payload_primitives::{
    BuildNextEnv, BuiltPayload, BuiltPayloadExecutedBlock, PayloadBuilderAttributes,
};
use reth_primitives_traits::{
    NodePrimitives, SealedBlock, SealedHeader, SignedTransaction, WithEncoded,
};

/// Re-export for use in downstream arguments.
pub use op_alloy_rpc_types_engine::OpPayloadAttributes;
use reth_optimism_primitives::OpPrimitives;

/// Optimism Payload Builder Attributes
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OpPayloadBuilderAttributes<T> {
    /// Inner ethereum payload builder attributes
    pub payload_attributes: EthPayloadBuilderAttributes,
    /// `NoTxPool` option for the generated payload
    pub no_tx_pool: bool,
    /// Decoded transactions and the original EIP-2718 encoded bytes as received in the payload
    /// attributes.
    pub transactions: Vec<WithEncoded<T>>,
    /// The gas limit for the generated payload
    pub gas_limit: Option<u64>,
    /// EIP-1559 parameters for the generated payload
    pub eip_1559_params: Option<B64>,
    /// Min base fee for the generated payload (only available post-Jovian)
    pub min_base_fee: Option<u64>,
}

impl<T> Default for OpPayloadBuilderAttributes<T> {
    fn default() -> Self {
        Self {
            payload_attributes: Default::default(),
            no_tx_pool: Default::default(),
            gas_limit: Default::default(),
            eip_1559_params: Default::default(),
            transactions: Default::default(),
            min_base_fee: Default::default(),
        }
    }
}

impl<T> OpPayloadBuilderAttributes<T> {
    /// Extracts the extra data parameters post-Holocene hardfork.
    /// In Holocene, those parameters are the EIP-1559 base fee parameters.
    pub fn get_holocene_extra_data(
        &self,
        default_base_fee_params: BaseFeeParams,
    ) -> Result<Bytes, EIP1559ParamError> {
        self.eip_1559_params
            .map(|params| encode_holocene_extra_data(params, default_base_fee_params))
            .ok_or(EIP1559ParamError::NoEIP1559Params)?
    }

    /// Extracts the extra data parameters post-Jovian hardfork.
    /// Those parameters are the EIP-1559 parameters from Holocene and the minimum base fee.
    pub fn get_jovian_extra_data(
        &self,
        default_base_fee_params: BaseFeeParams,
    ) -> Result<Bytes, EIP1559ParamError> {
        let min_base_fee = self.min_base_fee.ok_or(EIP1559ParamError::MinBaseFeeNotSet)?;
        self.eip_1559_params
            .map(|params| encode_jovian_extra_data(params, default_base_fee_params, min_base_fee))
            .ok_or(EIP1559ParamError::NoEIP1559Params)?
    }
}

impl<T: Decodable2718 + Send + Sync + Debug + Unpin + 'static> PayloadBuilderAttributes
    for OpPayloadBuilderAttributes<T>
{
    type RpcPayloadAttributes = OpPayloadAttributes;
    type Error = alloy_rlp::Error;

    /// Creates a new payload builder for the given parent block and the attributes.
    ///
    /// Derives the unique [`PayloadId`] for the given parent and attributes
    fn try_new(
        parent: B256,
        attributes: OpPayloadAttributes,
        version: u8,
    ) -> Result<Self, Self::Error> {
        let id = payload_id_optimism(&parent, &attributes, version);

        let transactions = attributes
            .transactions
            .unwrap_or_default()
            .into_iter()
            .map(|data| {
                Decodable2718::decode_2718_exact(data.as_ref()).map(|tx| WithEncoded::new(data, tx))
            })
            .collect::<Result<_, _>>()?;

        let payload_attributes = EthPayloadBuilderAttributes {
            id,
            parent,
            timestamp: attributes.payload_attributes.timestamp,
            suggested_fee_recipient: attributes.payload_attributes.suggested_fee_recipient,
            prev_randao: attributes.payload_attributes.prev_randao,
            withdrawals: attributes.payload_attributes.withdrawals.unwrap_or_default().into(),
            parent_beacon_block_root: attributes.payload_attributes.parent_beacon_block_root,
        };

        Ok(Self {
            payload_attributes,
            no_tx_pool: attributes.no_tx_pool.unwrap_or_default(),
            transactions,
            gas_limit: attributes.gas_limit,
            eip_1559_params: attributes.eip_1559_params,
            min_base_fee: attributes.min_base_fee,
        })
    }

    fn payload_id(&self) -> PayloadId {
        self.payload_attributes.id
    }

    fn parent(&self) -> B256 {
        self.payload_attributes.parent
    }

    fn timestamp(&self) -> u64 {
        self.payload_attributes.timestamp
    }

    fn parent_beacon_block_root(&self) -> Option<B256> {
        self.payload_attributes.parent_beacon_block_root
    }

    fn suggested_fee_recipient(&self) -> Address {
        self.payload_attributes.suggested_fee_recipient
    }

    fn prev_randao(&self) -> B256 {
        self.payload_attributes.prev_randao
    }

    fn withdrawals(&self) -> &Withdrawals {
        &self.payload_attributes.withdrawals
    }
}

impl<OpTransactionSigned> From<EthPayloadBuilderAttributes>
    for OpPayloadBuilderAttributes<OpTransactionSigned>
{
    fn from(value: EthPayloadBuilderAttributes) -> Self {
        Self { payload_attributes: value, ..Default::default() }
    }
}

/// Contains the built payload.
#[derive(Debug, Clone)]
pub struct OpBuiltPayload<N: NodePrimitives = OpPrimitives> {
    /// Identifier of the payload
    pub(crate) id: PayloadId,
    /// Sealed block
    pub(crate) block: Arc<SealedBlock<N::Block>>,
    /// Block execution data for the payload, if any.
    pub(crate) executed_block: Option<BuiltPayloadExecutedBlock<N>>,
    /// The fees of the block
    pub(crate) fees: U256,
}

// === impl BuiltPayload ===

impl<N: NodePrimitives> OpBuiltPayload<N> {
    /// Initializes the payload with the given initial block.
    pub const fn new(
        id: PayloadId,
        block: Arc<SealedBlock<N::Block>>,
        fees: U256,
        executed_block: Option<BuiltPayloadExecutedBlock<N>>,
    ) -> Self {
        Self { id, block, fees, executed_block }
    }

    /// Returns the identifier of the payload.
    pub const fn id(&self) -> PayloadId {
        self.id
    }

    /// Returns the built block(sealed)
    pub fn block(&self) -> &SealedBlock<N::Block> {
        &self.block
    }

    /// Fees of the block
    pub const fn fees(&self) -> U256 {
        self.fees
    }

    /// Converts the value into [`SealedBlock`].
    pub fn into_sealed_block(self) -> SealedBlock<N::Block> {
        Arc::unwrap_or_clone(self.block)
    }
}

impl<N: NodePrimitives> BuiltPayload for OpBuiltPayload<N> {
    type Primitives = N;

    fn block(&self) -> &SealedBlock<N::Block> {
        self.block()
    }

    fn fees(&self) -> U256 {
        self.fees
    }

    fn executed_block(&self) -> Option<BuiltPayloadExecutedBlock<N>> {
        self.executed_block.clone()
    }

    fn requests(&self) -> Option<Requests> {
        None
    }
}

// V1 engine_getPayloadV1 response
impl<T, N> From<OpBuiltPayload<N>> for ExecutionPayloadV1
where
    T: SignedTransaction,
    N: NodePrimitives<Block = Block<T>>,
{
    fn from(value: OpBuiltPayload<N>) -> Self {
        Self::from_block_unchecked(
            value.block().hash(),
            &Arc::unwrap_or_clone(value.block).into_block(),
        )
    }
}

// V2 engine_getPayloadV2 response
impl<T, N> From<OpBuiltPayload<N>> for ExecutionPayloadEnvelopeV2
where
    T: SignedTransaction,
    N: NodePrimitives<Block = Block<T>>,
{
    fn from(value: OpBuiltPayload<N>) -> Self {
        let OpBuiltPayload { block, fees, .. } = value;

        Self {
            block_value: fees,
            execution_payload: ExecutionPayloadFieldV2::from_block_unchecked(
                block.hash(),
                &Arc::unwrap_or_clone(block).into_block(),
            ),
        }
    }
}

impl<T, N> From<OpBuiltPayload<N>> for OpExecutionPayloadEnvelopeV3
where
    T: SignedTransaction,
    N: NodePrimitives<Block = Block<T>>,
{
    fn from(value: OpBuiltPayload<N>) -> Self {
        let OpBuiltPayload { block, fees, .. } = value;

        let parent_beacon_block_root = block.parent_beacon_block_root.unwrap_or_default();

        Self {
            execution_payload: ExecutionPayloadV3::from_block_unchecked(
                block.hash(),
                &Arc::unwrap_or_clone(block).into_block(),
            ),
            block_value: fees,
            // From the engine API spec:
            //
            // > Client software **MAY** use any heuristics to decide whether to set
            // `shouldOverrideBuilder` flag or not. If client software does not implement any
            // heuristic this flag **SHOULD** be set to `false`.
            //
            // Spec:
            // <https://github.com/ethereum/execution-apis/blob/fe8e13c288c592ec154ce25c534e26cb7ce0530d/src/engine/cancun.md#specification-2>
            should_override_builder: false,
            // No blobs for OP.
            blobs_bundle: BlobsBundleV1 { blobs: vec![], commitments: vec![], proofs: vec![] },
            parent_beacon_block_root,
        }
    }
}

impl<T, N> From<OpBuiltPayload<N>> for OpExecutionPayloadEnvelopeV4
where
    T: SignedTransaction,
    N: NodePrimitives<Block = Block<T>>,
{
    fn from(value: OpBuiltPayload<N>) -> Self {
        let OpBuiltPayload { block, fees, .. } = value;

        let parent_beacon_block_root = block.parent_beacon_block_root.unwrap_or_default();

        let l2_withdrawals_root = block.withdrawals_root.unwrap_or_default();
        let payload_v3 = ExecutionPayloadV3::from_block_unchecked(
            block.hash(),
            &Arc::unwrap_or_clone(block).into_block(),
        );

        Self {
            execution_payload: OpExecutionPayloadV4::from_v3_with_withdrawals_root(
                payload_v3,
                l2_withdrawals_root,
            ),
            block_value: fees,
            // From the engine API spec:
            //
            // > Client software **MAY** use any heuristics to decide whether to set
            // `shouldOverrideBuilder` flag or not. If client software does not implement any
            // heuristic this flag **SHOULD** be set to `false`.
            //
            // Spec:
            // <https://github.com/ethereum/execution-apis/blob/fe8e13c288c592ec154ce25c534e26cb7ce0530d/src/engine/cancun.md#specification-2>
            should_override_builder: false,
            // No blobs for OP.
            blobs_bundle: BlobsBundleV1 { blobs: vec![], commitments: vec![], proofs: vec![] },
            parent_beacon_block_root,
            execution_requests: vec![],
        }
    }
}

/// Generates the payload id for the configured payload from the [`OpPayloadAttributes`].
///
/// Returns an 8-byte identifier by hashing the payload components with sha256 hash.
///
/// Note: This must be updated whenever the [`OpPayloadAttributes`] changes for a hardfork.
/// See also <https://github.com/ethereum-optimism/op-geth/blob/d401af16f2dd94b010a72eaef10e07ac10b31931/miner/payload_building.go#L59-L59>
pub fn payload_id_optimism(
    parent: &B256,
    attributes: &OpPayloadAttributes,
    payload_version: u8,
) -> PayloadId {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    hasher.update(parent.as_slice());
    hasher.update(&attributes.payload_attributes.timestamp.to_be_bytes()[..]);
    hasher.update(attributes.payload_attributes.prev_randao.as_slice());
    hasher.update(attributes.payload_attributes.suggested_fee_recipient.as_slice());
    if let Some(withdrawals) = &attributes.payload_attributes.withdrawals {
        let mut buf = Vec::new();
        withdrawals.encode(&mut buf);
        hasher.update(buf);
    }

    if let Some(parent_beacon_block) = attributes.payload_attributes.parent_beacon_block_root {
        hasher.update(parent_beacon_block);
    }

    let no_tx_pool = attributes.no_tx_pool.unwrap_or_default();
    if no_tx_pool || attributes.transactions.as_ref().is_some_and(|txs| !txs.is_empty()) {
        hasher.update([no_tx_pool as u8]);
        let txs_len = attributes.transactions.as_ref().map(|txs| txs.len()).unwrap_or_default();
        hasher.update(&txs_len.to_be_bytes()[..]);
        if let Some(txs) = &attributes.transactions {
            for tx in txs {
                // we have to just hash the bytes here because otherwise we would need to decode
                // the transactions here which really isn't ideal
                let tx_hash = keccak256(tx);
                // maybe we can try just taking the hash and not decoding
                hasher.update(tx_hash)
            }
        }
    }

    if let Some(gas_limit) = attributes.gas_limit {
        hasher.update(gas_limit.to_be_bytes());
    }

    if let Some(eip_1559_params) = attributes.eip_1559_params {
        hasher.update(eip_1559_params.as_slice());
    }

    if let Some(min_base_fee) = attributes.min_base_fee {
        hasher.update(min_base_fee.to_be_bytes());
    }

    let mut out = hasher.finalize();
    out[0] = payload_version;

    #[allow(deprecated)] // generic-array 0.14 deprecated
    PayloadId::new(out.as_slice()[..8].try_into().expect("sufficient length"))
}

impl<H, T, ChainSpec> BuildNextEnv<OpPayloadBuilderAttributes<T>, H, ChainSpec>
    for OpNextBlockEnvAttributes
where
    H: BlockHeader,
    T: SignedTransaction,
    ChainSpec: EthChainSpec + OpHardforks,
{
    fn build_next_env(
        attributes: &OpPayloadBuilderAttributes<T>,
        parent: &SealedHeader<H>,
        chain_spec: &ChainSpec,
    ) -> Result<Self, PayloadBuilderError> {
        let extra_data = if chain_spec.is_jovian_active_at_timestamp(attributes.timestamp()) {
            attributes
                .get_jovian_extra_data(
                    chain_spec.base_fee_params_at_timestamp(attributes.timestamp()),
                )
                .map_err(PayloadBuilderError::other)?
        } else if chain_spec.is_holocene_active_at_timestamp(attributes.timestamp()) {
            attributes
                .get_holocene_extra_data(
                    chain_spec.base_fee_params_at_timestamp(attributes.timestamp()),
                )
                .map_err(PayloadBuilderError::other)?
        } else {
            Default::default()
        };

        Ok(Self {
            timestamp: attributes.timestamp(),
            suggested_fee_recipient: attributes.suggested_fee_recipient(),
            prev_randao: attributes.prev_randao(),
            gas_limit: attributes.gas_limit.unwrap_or_else(|| parent.gas_limit()),
            parent_beacon_block_root: attributes.parent_beacon_block_root(),
            extra_data,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpPayloadAttributes;
    use alloy_primitives::{FixedBytes, address, b256, bytes};
    use alloy_rpc_types_engine::PayloadAttributes;
    use reth_optimism_primitives::OpTransactionSigned;
    use reth_payload_primitives::EngineApiMessageVersion;
    use std::str::FromStr;

    #[test]
    fn test_payload_id_parity_op_geth() {
        // INFO rollup_boost::server:received fork_choice_updated_v3 from builder and l2_client
        // payload_id_builder="0x6ef26ca02318dcf9" payload_id_l2="0x03d2dae446d2a86a"
        let expected =
            PayloadId::new(FixedBytes::<8>::from_str("0x03d2dae446d2a86a").unwrap().into());
        let attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 1728933301,
                prev_randao: b256!("0x9158595abbdab2c90635087619aa7042bbebe47642dfab3c9bfb934f6b082765"),
                suggested_fee_recipient: address!("0x4200000000000000000000000000000000000011"),
                withdrawals: Some([].into()),
                parent_beacon_block_root: b256!("0x8fe0193b9bf83cb7e5a08538e494fecc23046aab9a497af3704f4afdae3250ff").into(),
            },
            transactions: Some([bytes!("7ef8f8a0dc19cfa777d90980e4875d0a548a881baaa3f83f14d1bc0d3038bc329350e54194deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000f424000000000000000000000000300000000670d6d890000000000000125000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000014bf9181db6e381d4384bbf69c48b0ee0eed23c6ca26143c6d2544f9d39997a590000000000000000000000007f83d659683caf2767fd3c720981d51f5bc365bc")].into()),
            no_tx_pool: None,
            gas_limit: Some(30000000),
            eip_1559_params: None,
            min_base_fee: None,
        };

        // Reth's `PayloadId` should match op-geth's `PayloadId`. This fails
        assert_eq!(
            expected,
            payload_id_optimism(
                &b256!("0x3533bf30edaf9505d0810bf475cbe4e5f4b9889904b9845e83efdeab4e92eb1e"),
                &attrs,
                EngineApiMessageVersion::V3 as u8
            )
        );
    }

    #[test]
    fn test_payload_id_parity_op_geth_jovian() {
        // <https://github.com/ethereum-optimism/op-geth/compare/optimism...mattsse:op-geth:matt/check-payload-id-equality>
        let expected =
            PayloadId::new(FixedBytes::<8>::from_str("0x046c65ffc4d659ec").unwrap().into());
        let attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 1728933301,
                prev_randao: b256!("0x9158595abbdab2c90635087619aa7042bbebe47642dfab3c9bfb934f6b082765"),
                suggested_fee_recipient: address!("0x4200000000000000000000000000000000000011"),
                withdrawals: Some([].into()),
                parent_beacon_block_root: b256!("0x8fe0193b9bf83cb7e5a08538e494fecc23046aab9a497af3704f4afdae3250ff").into(),
            },
            transactions: Some([bytes!("7ef8f8a0dc19cfa777d90980e4875d0a548a881baaa3f83f14d1bc0d3038bc329350e54194deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000f424000000000000000000000000300000000670d6d890000000000000125000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000014bf9181db6e381d4384bbf69c48b0ee0eed23c6ca26143c6d2544f9d39997a590000000000000000000000007f83d659683caf2767fd3c720981d51f5bc365bc")].into()),
            no_tx_pool: None,
            gas_limit: Some(30000000),
            eip_1559_params: None,
            min_base_fee: Some(100),
        };

        // Reth's `PayloadId` should match op-geth's `PayloadId`. This fails
        assert_eq!(
            expected,
            payload_id_optimism(
                &b256!("0x3533bf30edaf9505d0810bf475cbe4e5f4b9889904b9845e83efdeab4e92eb1e"),
                &attrs,
                EngineApiMessageVersion::V4 as u8
            )
        );
    }

    #[test]
    fn test_get_extra_data_post_holocene() {
        let attributes: OpPayloadBuilderAttributes<OpTransactionSigned> =
            OpPayloadBuilderAttributes {
                eip_1559_params: Some(B64::from_str("0x0000000800000008").unwrap()),
                ..Default::default()
            };
        let extra_data = attributes.get_holocene_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(extra_data.unwrap(), Bytes::copy_from_slice(&[0, 0, 0, 0, 8, 0, 0, 0, 8]));
    }

    #[test]
    fn test_get_extra_data_post_holocene_default() {
        let attributes: OpPayloadBuilderAttributes<OpTransactionSigned> =
            OpPayloadBuilderAttributes { eip_1559_params: Some(B64::ZERO), ..Default::default() };
        let extra_data = attributes.get_holocene_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(extra_data.unwrap(), Bytes::copy_from_slice(&[0, 0, 0, 0, 80, 0, 0, 0, 60]));
    }

    #[test]
    fn test_get_extra_data_post_jovian() {
        let attributes: OpPayloadBuilderAttributes<OpTransactionSigned> =
            OpPayloadBuilderAttributes {
                eip_1559_params: Some(B64::from_str("0x0000000800000008").unwrap()),
                min_base_fee: Some(10),
                ..Default::default()
            };
        let extra_data = attributes.get_jovian_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(
            extra_data.unwrap(),
            // Version byte is 1 for Jovian, then holocene payload followed by 8 bytes for the
            // minimum base fee
            Bytes::copy_from_slice(&[1, 0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 10])
        );
    }

    #[test]
    fn test_get_extra_data_post_jovian_default() {
        let attributes: OpPayloadBuilderAttributes<OpTransactionSigned> =
            OpPayloadBuilderAttributes {
                eip_1559_params: Some(B64::ZERO),
                min_base_fee: Some(10),
                ..Default::default()
            };
        let extra_data = attributes.get_jovian_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(
            extra_data.unwrap(),
            // Version byte is 1 for Jovian, then holocene payload followed by 8 bytes for the
            // minimum base fee
            Bytes::copy_from_slice(&[1, 0, 0, 0, 80, 0, 0, 0, 60, 0, 0, 0, 0, 0, 0, 0, 10])
        );
    }

    #[test]
    fn test_get_extra_data_post_jovian_no_base_fee() {
        let attributes: OpPayloadBuilderAttributes<OpTransactionSigned> =
            OpPayloadBuilderAttributes {
                eip_1559_params: Some(B64::ZERO),
                min_base_fee: None,
                ..Default::default()
            };
        let extra_data = attributes.get_jovian_extra_data(BaseFeeParams::new(80, 60));
        assert_eq!(extra_data.unwrap_err(), EIP1559ParamError::MinBaseFeeNotSet);
    }
}
