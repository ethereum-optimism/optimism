//! Validates execution payload wrt Optimism consensus rules

use alloc::sync::Arc;
use alloy_consensus::Block;
use alloy_rpc_types_engine::PayloadError;
use derive_more::{Constructor, Deref};
use op_alloy_rpc_types_engine::{OpExecutionData, OpPayloadError};
use reth_optimism_forks::OpHardforks;
use reth_payload_validator::{cancun, prague, shanghai};
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
    pub fn ensure_well_formed_payload<T: SignedTransaction>(
        &self,
        payload: OpExecutionData,
    ) -> Result<SealedBlock<Block<T>>, OpPayloadError> {
        let OpExecutionData { payload, sidecar } = payload;

        let expected_hash = payload.block_hash();

        // First parse the block
        let sealed_block = payload.try_into_block_with_sidecar(&sidecar)?.seal_slow();

        // Ensure the hash included in the payload matches the block hash
        if expected_hash != sealed_block.hash() {
            return Err(PayloadError::BlockHash {
                execution: sealed_block.hash(),
                consensus: expected_hash,
            })?
        }

        shanghai::ensure_well_formed_fields(
            sealed_block.body(),
            self.is_shanghai_active_at_timestamp(sealed_block.timestamp),
        )?;

        cancun::ensure_well_formed_header_and_sidecar_fields(
            &sealed_block,
            sidecar.canyon(),
            self.is_cancun_active_at_timestamp(sealed_block.timestamp),
        )?;

        prague::ensure_well_formed_fields(
            sealed_block.body(),
            sidecar.isthmus(),
            self.is_prague_active_at_timestamp(sealed_block.timestamp),
        )?;

        Ok(sealed_block)
    }
}
