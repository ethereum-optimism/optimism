//! Optimism Consensus implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

extern crate alloc;

use alloc::{format, sync::Arc};
use alloy_consensus::{
    BlockHeader as _, EMPTY_OMMER_ROOT_HASH, constants::MAXIMUM_EXTRA_DATA_SIZE,
};
use alloy_primitives::B64;
use core::fmt::Debug;
use reth_chainspec::EthChainSpec;
use reth_consensus::{Consensus, ConsensusError, FullConsensus, HeaderValidator, ReceiptRootBloom};
use reth_consensus_common::validation::{
    validate_against_parent_eip1559_base_fee, validate_against_parent_hash_number,
    validate_against_parent_timestamp, validate_cancun_gas, validate_header_base_fee,
    validate_header_extra_data, validate_header_gas,
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
pub use validation::{canyon, isthmus, validate_block_post_execution};

pub mod error;
pub use error::OpConsensusError;

/// Optimism consensus implementation.
///
/// Provides basic checks as outlined in the execution specs.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OpBeaconConsensus<ChainSpec> {
    /// Configuration
    chain_spec: Arc<ChainSpec>,
    /// Maximum allowed extra data size in bytes
    max_extra_data_size: usize,
}

impl<ChainSpec> OpBeaconConsensus<ChainSpec> {
    /// Create a new instance of [`OpBeaconConsensus`]
    pub const fn new(chain_spec: Arc<ChainSpec>) -> Self {
        Self { chain_spec, max_extra_data_size: MAXIMUM_EXTRA_DATA_SIZE }
    }

    /// Returns the maximum allowed extra data size.
    pub const fn max_extra_data_size(&self) -> usize {
        self.max_extra_data_size
    }

    /// Sets the maximum allowed extra data size and returns the updated instance.
    pub const fn with_max_extra_data_size(mut self, size: usize) -> Self {
        self.max_extra_data_size = size;
        self
    }
}

impl<N, ChainSpec> FullConsensus<N> for OpBeaconConsensus<ChainSpec>
where
    N: NodePrimitives<Receipt: DepositReceipt>,
    ChainSpec: EthChainSpec<Header = N::BlockHeader> + OpHardforks + Debug + Send + Sync,
{
    fn validate_block_post_execution(
        &self,
        block: &RecoveredBlock<N::Block>,
        result: &BlockExecutionResult<N::Receipt>,
        receipt_root_bloom: Option<ReceiptRootBloom>,
    ) -> Result<(), ConsensusError> {
        validate_block_post_execution(block.header(), &self.chain_spec, result, receipt_root_bloom)
    }
}

impl<B, ChainSpec> Consensus<B> for OpBeaconConsensus<ChainSpec>
where
    B: Block,
    ChainSpec: EthChainSpec<Header = B::Header> + OpHardforks + Debug + Send + Sync,
{
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
            ));
        }

        // Check transaction root
        if let Err(error) = block.ensure_transaction_root_valid() {
            return Err(ConsensusError::BodyTransactionRootDiff(error.into()));
        }

        // Check empty shanghai-withdrawals
        if self.chain_spec.is_canyon_active_at_timestamp(block.timestamp()) {
            canyon::ensure_empty_shanghai_withdrawals(block.body()).map_err(|err| {
                ConsensusError::Other(format!("failed to verify block {}: {err}", block.number()))
            })?
        } else {
            return Ok(());
        }

        // Blob gas used validation
        // In Jovian, the blob gas used computation has changed. We are moving the blob base fee
        // validation to post-execution since the DA footprint calculation is stateful.
        // Pre-execution we only validate that the blob gas used is present in the header.
        if self.chain_spec.is_jovian_active_at_timestamp(block.timestamp()) {
            block.blob_gas_used().ok_or(ConsensusError::BlobGasUsedMissing)?;
        } else if self.chain_spec.is_ecotone_active_at_timestamp(block.timestamp()) {
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

impl<H, ChainSpec> HeaderValidator<H> for OpBeaconConsensus<ChainSpec>
where
    H: BlockHeader,
    ChainSpec: EthChainSpec<Header = H> + OpHardforks + Debug + Send + Sync,
{
    fn validate_header(&self, header: &SealedHeader<H>) -> Result<(), ConsensusError> {
        let header = header.header();
        // with OP-stack Bedrock activation number determines when TTD (eth Merge) has been reached.
        debug_assert!(
            self.chain_spec.is_bedrock_active_at_block(header.number()),
            "manually import OVM blocks"
        );

        if header.nonce() != Some(B64::ZERO) {
            return Err(ConsensusError::TheMergeNonceIsNotZero);
        }

        if header.ommers_hash() != EMPTY_OMMER_ROOT_HASH {
            return Err(ConsensusError::TheMergeOmmerRootIsNotEmpty);
        }

        // Post-merge, the consensus layer is expected to perform checks such that the block
        // timestamp is a function of the slot. This is different from pre-merge, where blocks
        // are only allowed to be in the future (compared to the system's clock) by a certain
        // threshold.
        //
        // Block validation with respect to the parent should ensure that the block timestamp
        // is greater than its parent timestamp.

        // validate header extra data for all networks post merge
        validate_header_extra_data(header, self.max_extra_data_size)?;
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

        validate_against_parent_eip1559_base_fee(
            header.header(),
            parent.header(),
            &self.chain_spec,
        )?;

        // Ensure that the blob gas fields for this block are correctly set.
        // In the op-stack, the excess blob gas is always 0 for all blocks after ecotone.
        // The blob gas used and the excess blob gas should both be set after ecotone.
        // After Jovian, the blob gas used contains the current DA footprint.
        if self.chain_spec.is_ecotone_active_at_timestamp(header.timestamp()) {
            let blob_gas_used = header.blob_gas_used().ok_or(ConsensusError::BlobGasUsedMissing)?;

            // Before Jovian and after ecotone, the blob gas used should be 0.
            if !self.chain_spec.is_jovian_active_at_timestamp(header.timestamp()) &&
                blob_gas_used != 0
            {
                return Err(ConsensusError::BlobGasUsedDiff(GotExpected {
                    got: blob_gas_used,
                    expected: 0,
                }));
            }

            let excess_blob_gas =
                header.excess_blob_gas().ok_or(ConsensusError::ExcessBlobGasMissing)?;
            if excess_blob_gas != 0 {
                return Err(ConsensusError::ExcessBlobGasDiff {
                    diff: GotExpected { got: excess_blob_gas, expected: 0 },
                    parent_excess_blob_gas: parent.excess_blob_gas().unwrap_or(0),
                    parent_blob_gas_used: parent.blob_gas_used().unwrap_or(0),
                });
            }
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use alloy_consensus::{BlockBody, Eip658Value, Header, Receipt, TxEip7702, TxReceipt};
    use alloy_eips::{eip4895::Withdrawals, eip7685::Requests};
    use alloy_primitives::{Address, Bytes, Log, Signature, U256};
    use op_alloy_consensus::{
        OpTypedTransaction, encode_holocene_extra_data, encode_jovian_extra_data,
    };
    use reth_chainspec::BaseFeeParams;
    use reth_consensus::{Consensus, ConsensusError, FullConsensus, HeaderValidator};
    use reth_optimism_chainspec::{OP_MAINNET, OpChainSpec, OpChainSpecBuilder};
    use reth_optimism_primitives::{OpPrimitives, OpReceipt, OpTransactionSigned};
    use reth_primitives_traits::{RecoveredBlock, SealedBlock, SealedHeader, proofs};
    use reth_provider::BlockExecutionResult;

    use crate::OpBeaconConsensus;

    fn mock_tx(nonce: u64) -> OpTransactionSigned {
        let tx = TxEip7702 {
            chain_id: 1u64,
            nonce,
            max_fee_per_gas: 0x28f000fff,
            max_priority_fee_per_gas: 0x28f000fff,
            gas_limit: 10,
            to: Address::default(),
            value: U256::from(3_u64),
            input: Bytes::from(vec![1, 2]),
            access_list: Default::default(),
            authorization_list: Default::default(),
        };

        let signature = Signature::new(U256::default(), U256::default(), true);

        OpTransactionSigned::new_unhashed(OpTypedTransaction::Eip7702(tx), signature)
    }

    #[test]
    fn test_block_blob_gas_used_validation_isthmus() {
        let chain_spec = OpChainSpecBuilder::default()
            .isthmus_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let header = Header {
            base_fee_per_gas: Some(1337),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            timestamp: u64::MAX,
            ..Default::default()
        };
        let body = BlockBody {
            transactions: vec![transaction],
            ommers: vec![],
            withdrawals: Some(Withdrawals::default()),
        };

        let block = SealedBlock::seal_slow(alloy_consensus::Block { header, body });

        // validate blob, it should pass blob gas used validation
        let pre_execution = beacon_consensus.validate_block_pre_execution(&block);

        assert!(pre_execution.is_ok());
    }

    #[test]
    fn test_block_blob_gas_used_validation_failure_isthmus() {
        let chain_spec = OpChainSpecBuilder::default()
            .isthmus_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let header = Header {
            base_fee_per_gas: Some(1337),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(10),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            timestamp: u64::MAX,
            ..Default::default()
        };
        let body = BlockBody {
            transactions: vec![transaction],
            ommers: vec![],
            withdrawals: Some(Withdrawals::default()),
        };

        let block = SealedBlock::seal_slow(alloy_consensus::Block { header, body });

        // validate blob, it should fail blob gas used validation
        let pre_execution = beacon_consensus.validate_block_pre_execution(&block);

        assert!(matches!(
            pre_execution.unwrap_err(),
            ConsensusError::BlobGasUsedDiff(diff) if diff.got == 10 && diff.expected == 0
        ));
    }

    #[test]
    fn test_block_blob_gas_used_validation_jovian() {
        const BLOB_GAS_USED: u64 = 1000;
        const GAS_USED: u64 = 10;

        let chain_spec = OpChainSpecBuilder::default()
            .jovian_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let receipt = OpReceipt::Eip7702(Receipt::<Log> {
            status: Eip658Value::success(),
            cumulative_gas_used: GAS_USED,
            logs: vec![],
        });

        let header = Header {
            base_fee_per_gas: Some(1337),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(BLOB_GAS_USED),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            timestamp: u64::MAX,
            gas_used: GAS_USED,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            ..Default::default()
        };
        let body = BlockBody {
            transactions: vec![transaction],
            ommers: vec![],
            withdrawals: Some(Withdrawals::default()),
        };

        let block = SealedBlock::seal_slow(alloy_consensus::Block { header, body });

        let result = BlockExecutionResult::<OpReceipt> {
            blob_gas_used: BLOB_GAS_USED,
            receipts: vec![receipt],
            requests: Requests::default(),
            gas_used: GAS_USED,
        };

        // validate blob, it should pass blob gas used validation
        let pre_execution = beacon_consensus.validate_block_pre_execution(&block);

        assert!(pre_execution.is_ok());

        let block = RecoveredBlock::new_sealed(block, vec![Address::default()]);

        let post_execution = <OpBeaconConsensus<OpChainSpec> as FullConsensus<OpPrimitives>>::validate_block_post_execution(
            &beacon_consensus,
            &block,
            &result,
            None,
        );

        // validate blob, it should pass blob gas used validation
        assert!(post_execution.is_ok());
    }

    #[test]
    fn test_block_blob_gas_used_validation_failure_jovian() {
        const BLOB_GAS_USED: u64 = 1000;
        const GAS_USED: u64 = 10;

        let chain_spec = OpChainSpecBuilder::default()
            .jovian_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let receipt = OpReceipt::Eip7702(Receipt::<Log> {
            status: Eip658Value::success(),
            cumulative_gas_used: GAS_USED,
            logs: vec![],
        });

        let header = Header {
            base_fee_per_gas: Some(1337),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(BLOB_GAS_USED),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: GAS_USED,
            timestamp: u64::MAX,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            ..Default::default()
        };
        let body = BlockBody {
            transactions: vec![transaction],
            ommers: vec![],
            withdrawals: Some(Withdrawals::default()),
        };

        let block = SealedBlock::seal_slow(alloy_consensus::Block { header, body });

        let result = BlockExecutionResult::<OpReceipt> {
            blob_gas_used: BLOB_GAS_USED + 1,
            receipts: vec![receipt],
            requests: Requests::default(),
            gas_used: GAS_USED,
        };

        // validate blob, it should pass blob gas used validation
        let pre_execution = beacon_consensus.validate_block_pre_execution(&block);

        assert!(pre_execution.is_ok());

        let block = RecoveredBlock::new_sealed(block, vec![Address::default()]);

        let post_execution = <OpBeaconConsensus<OpChainSpec> as FullConsensus<OpPrimitives>>::validate_block_post_execution(
            &beacon_consensus,
            &block,
            &result,
            None,
        );

        // validate blob, it should fail blob gas used validation post execution.
        assert!(matches!(
            post_execution.unwrap_err(),
            ConsensusError::BlobGasUsedDiff(diff)
                if diff.got == BLOB_GAS_USED + 1 && diff.expected == BLOB_GAS_USED
        ));
    }

    #[test]
    fn test_header_min_base_fee_validation() {
        const MIN_BASE_FEE: u64 = 1000;

        let chain_spec = OpChainSpecBuilder::default()
            .jovian_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let receipt = OpReceipt::Eip7702(Receipt::<Log> {
            status: Eip658Value::success(),
            cumulative_gas_used: 0,
            logs: vec![],
        });

        let parent = Header {
            number: 0,
            base_fee_per_gas: Some(MIN_BASE_FEE / 10),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX - 1,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            extra_data: encode_jovian_extra_data(
                Default::default(),
                BaseFeeParams::optimism(),
                MIN_BASE_FEE,
            )
            .unwrap(),
            ..Default::default()
        };
        let parent = SealedHeader::seal_slow(parent);

        let header = Header {
            number: 1,
            base_fee_per_gas: Some(MIN_BASE_FEE),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            parent_hash: parent.hash(),
            ..Default::default()
        };
        let header = SealedHeader::seal_slow(header);

        let result = beacon_consensus.validate_header_against_parent(&header, &parent);

        assert!(result.is_ok());
    }

    #[test]
    fn test_header_min_base_fee_validation_failure() {
        const MIN_BASE_FEE: u64 = 1000;

        let chain_spec = OpChainSpecBuilder::default()
            .jovian_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let receipt = OpReceipt::Eip7702(Receipt::<Log> {
            status: Eip658Value::success(),
            cumulative_gas_used: 0,
            logs: vec![],
        });

        let parent = Header {
            number: 0,
            base_fee_per_gas: Some(MIN_BASE_FEE / 10),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX - 1,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            extra_data: encode_jovian_extra_data(
                Default::default(),
                BaseFeeParams::optimism(),
                MIN_BASE_FEE,
            )
            .unwrap(),
            ..Default::default()
        };
        let parent = SealedHeader::seal_slow(parent);

        let header = Header {
            number: 1,
            base_fee_per_gas: Some(MIN_BASE_FEE - 1),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            parent_hash: parent.hash(),
            ..Default::default()
        };
        let header = SealedHeader::seal_slow(header);

        let result = beacon_consensus.validate_header_against_parent(&header, &parent);

        assert!(matches!(
            result.unwrap_err(),
            ConsensusError::BaseFeeDiff(diff)
                if diff.got == MIN_BASE_FEE - 1 && diff.expected == MIN_BASE_FEE
        ));
    }

    #[test]
    fn test_header_da_footprint_validation() {
        const MIN_BASE_FEE: u64 = 100_000;
        const DA_FOOTPRINT: u64 = GAS_LIMIT - 1;
        const GAS_LIMIT: u64 = 100_000_000;

        let chain_spec = OpChainSpecBuilder::default()
            .jovian_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let receipt = OpReceipt::Eip7702(Receipt::<Log> {
            status: Eip658Value::success(),
            cumulative_gas_used: 0,
            logs: vec![],
        });

        let parent = Header {
            number: 0,
            base_fee_per_gas: Some(MIN_BASE_FEE),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(DA_FOOTPRINT),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX - 1,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            extra_data: encode_jovian_extra_data(
                Default::default(),
                BaseFeeParams::optimism(),
                MIN_BASE_FEE,
            )
            .unwrap(),
            gas_limit: GAS_LIMIT,
            ..Default::default()
        };
        let parent = SealedHeader::seal_slow(parent);

        let header = Header {
            number: 1,
            base_fee_per_gas: Some(MIN_BASE_FEE + MIN_BASE_FEE / 10),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(DA_FOOTPRINT),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            parent_hash: parent.hash(),
            ..Default::default()
        };
        let header = SealedHeader::seal_slow(header);

        let result = beacon_consensus.validate_header_against_parent(&header, &parent);

        assert!(result.is_ok());
    }

    #[test]
    fn test_header_isthmus_validation() {
        const MIN_BASE_FEE: u64 = 100_000;
        const DA_FOOTPRINT: u64 = GAS_LIMIT - 1;
        const GAS_LIMIT: u64 = 100_000_000;

        let chain_spec = OpChainSpecBuilder::default()
            .isthmus_activated()
            .genesis(OP_MAINNET.genesis.clone())
            .chain(OP_MAINNET.chain)
            .build();

        // create a tx
        let transaction = mock_tx(0);

        let beacon_consensus = OpBeaconConsensus::new(Arc::new(chain_spec));

        let receipt = OpReceipt::Eip7702(Receipt::<Log> {
            status: Eip658Value::success(),
            cumulative_gas_used: 0,
            logs: vec![],
        });

        let parent = Header {
            number: 0,
            base_fee_per_gas: Some(MIN_BASE_FEE),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(DA_FOOTPRINT),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX - 1,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            extra_data: encode_holocene_extra_data(Default::default(), BaseFeeParams::optimism())
                .unwrap(),
            gas_limit: GAS_LIMIT,
            ..Default::default()
        };
        let parent = SealedHeader::seal_slow(parent);

        let header = Header {
            number: 1,
            base_fee_per_gas: Some(MIN_BASE_FEE - 2 * MIN_BASE_FEE / 100),
            withdrawals_root: Some(proofs::calculate_withdrawals_root(&[])),
            blob_gas_used: Some(DA_FOOTPRINT),
            excess_blob_gas: Some(0),
            transactions_root: proofs::calculate_transaction_root(std::slice::from_ref(
                &transaction,
            )),
            gas_used: 0,
            timestamp: u64::MAX,
            receipts_root: proofs::calculate_receipt_root(std::slice::from_ref(
                &receipt.with_bloom_ref(),
            )),
            logs_bloom: receipt.bloom(),
            parent_hash: parent.hash(),
            ..Default::default()
        };
        let header = SealedHeader::seal_slow(header);

        let result = beacon_consensus.validate_header_against_parent(&header, &parent);

        assert!(matches!(
            result.unwrap_err(),
            ConsensusError::BlobGasUsedDiff(diff)
                if diff.got == DA_FOOTPRINT && diff.expected == 0
        ));
    }
}
