//! Optimism Consensus implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

extern crate alloc;

use alloc::{format, sync::Arc};
use alloy_consensus::{BlockHeader as _, EMPTY_OMMER_ROOT_HASH};
use alloy_primitives::B64;
use core::fmt::Debug;
use reth_chainspec::EthChainSpec;
use reth_consensus::{Consensus, ConsensusError, FullConsensus, HeaderValidator};
use reth_consensus_common::validation::{
    validate_against_parent_4844, validate_against_parent_eip1559_base_fee,
    validate_against_parent_hash_number, validate_against_parent_timestamp, validate_cancun_gas,
    validate_header_base_fee, validate_header_extra_data, validate_header_gas,
};
use reth_execution_types::BlockExecutionResult;
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::DepositReceipt;
use reth_primitives_traits::{
    Block, BlockBody, BlockHeader, GotExpected, NodePrimitives, RecoveredBlock, SealedBlock,
    SealedHeader,
};

mod proof;
pub use proof::calculate_receipt_root_no_memo_optimism;

pub mod validation;
pub use validation::{
    canyon, decode_holocene_base_fee, isthmus, next_block_base_fee, validate_block_post_execution,
};

pub mod error;
pub use error::OpConsensusError;

/// Optimism consensus implementation.
///
/// Provides basic checks as outlined in the execution specs.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OpBeaconConsensus<ChainSpec> {
    /// Configuration
    chain_spec: Arc<ChainSpec>,
}

impl<ChainSpec> OpBeaconConsensus<ChainSpec> {
    /// Create a new instance of [`OpBeaconConsensus`]
    pub const fn new(chain_spec: Arc<ChainSpec>) -> Self {
        Self { chain_spec }
    }
}

impl<ChainSpec: EthChainSpec + OpHardforks, N: NodePrimitives<Receipt: DepositReceipt>>
    FullConsensus<N> for OpBeaconConsensus<ChainSpec>
{
    fn validate_block_post_execution(
        &self,
        block: &RecoveredBlock<N::Block>,
        result: &BlockExecutionResult<N::Receipt>,
    ) -> Result<(), ConsensusError> {
        validate_block_post_execution(block.header(), &self.chain_spec, &result.receipts)
    }
}

impl<ChainSpec: EthChainSpec + OpHardforks, B: Block> Consensus<B>
    for OpBeaconConsensus<ChainSpec>
{
    type Error = ConsensusError;

    fn validate_body_against_header(
        &self,
        body: &B::Body,
        header: &SealedHeader<B::Header>,
    ) -> Result<(), ConsensusError> {
        validation::validate_body_against_header_op(&self.chain_spec, body, header.header())
    }

    fn validate_block_pre_execution(&self, block: &SealedBlock<B>) -> Result<(), ConsensusError> {
        // Check ommers hash
        let ommers_hash = block.body().calculate_ommers_root();
        if Some(block.ommers_hash()) != ommers_hash {
            return Err(ConsensusError::BodyOmmersHashDiff(
                GotExpected {
                    got: ommers_hash.unwrap_or(EMPTY_OMMER_ROOT_HASH),
                    expected: block.ommers_hash(),
                }
                .into(),
            ))
        }

        // Check transaction root
        if let Err(error) = block.ensure_transaction_root_valid() {
            return Err(ConsensusError::BodyTransactionRootDiff(error.into()))
        }

        // Check empty shanghai-withdrawals
        if self.chain_spec.is_canyon_active_at_timestamp(block.timestamp()) {
            canyon::ensure_empty_shanghai_withdrawals(block.body()).map_err(|err| {
                ConsensusError::Other(format!("failed to verify block {}: {err}", block.number()))
            })?
        } else {
            return Ok(())
        }

        if self.chain_spec.is_ecotone_active_at_timestamp(block.timestamp()) {
            validate_cancun_gas(block)?;
        }

        // Check withdrawals root field in header
        if self.chain_spec.is_isthmus_active_at_timestamp(block.timestamp()) {
            // storage root of withdrawals pre-deploy is verified post-execution
            isthmus::ensure_withdrawals_storage_root_is_some(block.header()).map_err(|err| {
                ConsensusError::Other(format!("failed to verify block {}: {err}", block.number()))
            })?
        } else {
            // canyon is active, else would have returned already
            canyon::ensure_empty_withdrawals_root(block.header())?
        }

        Ok(())
    }
}

impl<ChainSpec: EthChainSpec + OpHardforks, H: BlockHeader> HeaderValidator<H>
    for OpBeaconConsensus<ChainSpec>
{
    fn validate_header(&self, header: &SealedHeader<H>) -> Result<(), ConsensusError> {
        let header = header.header();
        // with OP-stack Bedrock activation number determines when TTD (eth Merge) has been reached.
        debug_assert!(
            self.chain_spec.is_bedrock_active_at_block(header.number()),
            "manually import OVM blocks"
        );

        if header.nonce() != Some(B64::ZERO) {
            return Err(ConsensusError::TheMergeNonceIsNotZero)
        }

        if header.ommers_hash() != EMPTY_OMMER_ROOT_HASH {
            return Err(ConsensusError::TheMergeOmmerRootIsNotEmpty)
        }

        // Post-merge, the consensus layer is expected to perform checks such that the block
        // timestamp is a function of the slot. This is different from pre-merge, where blocks
        // are only allowed to be in the future (compared to the system's clock) by a certain
        // threshold.
        //
        // Block validation with respect to the parent should ensure that the block timestamp
        // is greater than its parent timestamp.

        // validate header extra data for all networks post merge
        validate_header_extra_data(header)?;
        validate_header_gas(header)?;
        validate_header_base_fee(header, &self.chain_spec)
    }

    fn validate_header_against_parent(
        &self,
        header: &SealedHeader<H>,
        parent: &SealedHeader<H>,
    ) -> Result<(), ConsensusError> {
        validate_against_parent_hash_number(header.header(), parent)?;

        if self.chain_spec.is_bedrock_active_at_block(header.number()) {
            validate_against_parent_timestamp(header.header(), parent.header())?;
        }

        // EIP1559 base fee validation
        // <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#base-fee-computation>
        // > if Holocene is active in parent_header.timestamp, then the parameters from
        // > parent_header.extraData are used.
        if self.chain_spec.is_holocene_active_at_timestamp(parent.timestamp()) {
            let header_base_fee =
                header.base_fee_per_gas().ok_or(ConsensusError::BaseFeeMissing)?;
            let expected_base_fee =
                decode_holocene_base_fee(&self.chain_spec, parent.header(), header.timestamp())
                    .map_err(|_| ConsensusError::BaseFeeMissing)?;
            if expected_base_fee != header_base_fee {
                return Err(ConsensusError::BaseFeeDiff(GotExpected {
                    expected: expected_base_fee,
                    got: header_base_fee,
                }))
            }
        } else {
            validate_against_parent_eip1559_base_fee(
                header.header(),
                parent.header(),
                &self.chain_spec,
            )?;
        }

        // ensure that the blob gas fields for this block
        if let Some(blob_params) = self.chain_spec.blob_params_at_timestamp(header.timestamp()) {
            validate_against_parent_4844(header.header(), parent.header(), blob_params)?;
        }

        Ok(())
    }
}
