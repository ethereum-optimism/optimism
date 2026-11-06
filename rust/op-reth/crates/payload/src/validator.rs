//! Validates execution payload wrt Optimism consensus rules

use alloc::sync::Arc;
use alloy_consensus::Block;
use derive_more::{Constructor, Deref};
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayloadEnvelope, OpPayloadError};
use reth_optimism_forks::OpHardforks;
use reth_payload_validator::{amsterdam, cancun, prague, shanghai};
use reth_primitives_traits::{Block as _, SealedBlock, SignedTransaction};

/// Execution payload validator.
#[derive(Clone, Debug, Deref, Constructor)]
pub struct OpExecutionPayloadValidator<ChainSpec> {
    /// Chain spec to validate against.
    #[deref]
    inner: Arc<ChainSpec>,
}

impl<ChainSpec> OpExecutionPayloadValidator<ChainSpec>
where
    ChainSpec: OpHardforks,
{
    /// Returns reference to chain spec.
    pub fn chain_spec(&self) -> &ChainSpec {
        &self.inner
    }

    /// Ensures that the given payload does not violate any consensus rules that concern the block's
    /// layout.
    ///
    /// See also [`ensure_well_formed_payload`].
    pub fn ensure_well_formed_payload<T: SignedTransaction>(
        &self,
        payload: OpExecutionData,
    ) -> Result<SealedBlock<Block<T>>, OpPayloadError> {
        ensure_well_formed_payload(self.chain_spec(), payload)
    }
}

/// UPSTREAM-MIRROR(copy): reth@rev:8d359d0
/// `reth_ethereum_payload_builder::validator::ensure_well_formed_payload`
///
/// Copies upstream's ordered payload checks with OP payload fields and fork activation.
/// Ensures that the given payload does not violate any consensus rules that concern the block's
/// layout, like:
///    - missing or invalid base fee
///    - invalid extra data
///    - invalid transactions
///    - incorrect hash
///    - block contains blob transactions or blob versioned hashes
///    - block contains l1 withdrawals
///
/// The checks are done in the order that conforms with the engine-API specification.
///
/// This is intended to be invoked after receiving the payload from the CLI.
/// The additional fields, starting with [`MaybeCancunPayloadFields`](alloy_rpc_types_engine::MaybeCancunPayloadFields), are not part of the payload, but are additional fields starting in the `engine_newPayloadV3` RPC call, See also <https://specs.optimism.io/protocol/exec-engine.html#engine_newpayloadv3>
///
/// If the cancun fields are provided this also validates that the versioned hashes in the block
/// are empty as well as those passed in the sidecar. If the payload fields are not provided.
///
/// Validation according to specs <https://specs.optimism.io/protocol/exec-engine.html#engine-api>.
pub fn ensure_well_formed_payload<ChainSpec, T>(
    chain_spec: ChainSpec,
    payload: OpExecutionData,
) -> Result<SealedBlock<Block<T>>, OpPayloadError>
where
    ChainSpec: OpHardforks,
    T: SignedTransaction,
{
    let sidecar = payload.sidecar.clone();
    let payload = OpExecutionPayloadEnvelope::try_from(payload)?;

    // First parse the block and ensure its hash matches the payload's claim.
    let sealed_block = payload.try_into_checked_block()?.seal_slow();

    shanghai::ensure_well_formed_fields(
        sealed_block.body(),
        chain_spec.is_shanghai_active_at_timestamp(sealed_block.timestamp),
    )?;

    cancun::ensure_well_formed_header_and_sidecar_fields(
        &sealed_block,
        sidecar.ecotone(),
        chain_spec.is_cancun_active_at_timestamp(sealed_block.timestamp),
    )?;

    prague::ensure_well_formed_fields(
        sealed_block.body(),
        sidecar.isthmus(),
        chain_spec.is_prague_active_at_timestamp(sealed_block.timestamp),
    )?;

    amsterdam::ensure_well_formed_fields(
        &sealed_block,
        chain_spec.is_amsterdam_active_at_timestamp(sealed_block.timestamp),
    )?;

    Ok(sealed_block)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{BlockBody, Header, constants::EMPTY_ROOT_HASH};
    use alloy_eips::{eip4895::Withdrawals, eip7685::EMPTY_REQUESTS_HASH};
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::PayloadError;
    use op_alloy_rpc_types_engine::OpExecutionPayload;
    use reth_chainspec::{EthereumHardfork, ForkCondition};
    use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
    use reth_optimism_primitives::OpBlock;
    use rstest::rstest;

    fn isthmus_chain_spec() -> OpChainSpec {
        OpChainSpecBuilder::optimism_mainnet().isthmus_activated().build()
    }

    /// Isthmus plus an L1 Amsterdam activation, which no OP fork implies.
    fn amsterdam_chain_spec() -> OpChainSpec {
        OpChainSpecBuilder::optimism_mainnet()
            .isthmus_activated()
            .with_fork(EthereumHardfork::Amsterdam, ForkCondition::Timestamp(0))
            .build()
    }

    /// The header of an empty post-Isthmus block, with every field an `engine_newPayloadV4` round
    /// trip fills in already set so the payload's claimed hash matches the block it decodes to.
    fn isthmus_header() -> Header {
        Header {
            base_fee_per_gas: Some(0),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            parent_beacon_block_root: Some(B256::ZERO),
            requests_hash: Some(EMPTY_REQUESTS_HASH),
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            ..Default::default()
        }
    }

    fn isthmus_block(header: Header) -> OpBlock {
        let body = BlockBody { withdrawals: Some(Withdrawals::default()), ..Default::default() };
        OpBlock { header, body }
    }

    /// The payload `engine_newPayload` would carry for `block`, claiming the block's own hash.
    fn payload_for(block: &OpBlock) -> OpExecutionData {
        let (payload, sidecar) = OpExecutionPayload::from_block_slow(block);
        OpExecutionData::new(payload, sidecar)
    }

    fn validate(
        chain_spec: OpChainSpec,
        block: &OpBlock,
    ) -> Result<SealedBlock<OpBlock>, OpPayloadError> {
        ensure_well_formed_payload(chain_spec, payload_for(block))
    }

    #[test]
    fn plain_isthmus_payload_is_well_formed() {
        let block = isthmus_block(isthmus_header());

        let sealed_block = validate(isthmus_chain_spec(), &block).unwrap();

        assert_eq!(sealed_block.hash(), block.header.hash_slow());
    }

    /// No OP payload version carries the Amsterdam header fields, so a block that has one cannot
    /// be represented: the payload drops the field, its claimed hash no longer matches the block
    /// it decodes to, and the hash check rejects it before the Amsterdam check runs.
    #[rstest]
    #[case::block_access_list_hash(Some(B256::ZERO), None)]
    #[case::slot_number(None, Some(1))]
    fn amsterdam_header_fields_fail_the_block_hash_check(
        #[case] block_access_list_hash: Option<B256>,
        #[case] slot_number: Option<u64>,
    ) {
        let header = Header { block_access_list_hash, slot_number, ..isthmus_header() };

        let err = validate(isthmus_chain_spec(), &isthmus_block(header)).unwrap_err();

        assert!(matches!(err, OpPayloadError::Eth(PayloadError::BlockHash { .. })), "{err}");
    }

    /// An OP payload can never satisfy the post-Amsterdam check, so scheduling Amsterdam on an OP
    /// chain spec rejects every payload from the activation timestamp on.
    #[test]
    fn amsterdam_activation_rejects_every_op_payload() {
        let block = isthmus_block(isthmus_header());

        let err = validate(amsterdam_chain_spec(), &block).unwrap_err();

        assert!(
            matches!(
                err,
                OpPayloadError::Eth(PayloadError::PostAmsterdamBlockWithoutBlockAccessList)
            ),
            "{err}"
        );
    }
}
