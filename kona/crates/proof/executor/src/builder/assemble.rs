//! [Header] assembly logic for the [StatelessL2Builder].

use super::StatelessL2Builder;
use crate::{
    ExecutorError, ExecutorResult, TrieDBError, TrieDBProvider,
    util::{encode_holocene_eip_1559_params, encode_jovian_eip_1559_params},
};
use alloc::vec::Vec;
use alloy_consensus::{EMPTY_OMMER_ROOT_HASH, Header, Sealed};
use alloy_eips::{Encodable2718, eip7685::EMPTY_REQUESTS_HASH};
use alloy_evm::{EvmFactory, block::BlockExecutionResult};
use alloy_primitives::{B256, Sealable, U256, logs_bloom};
use alloy_trie::EMPTY_ROOT_HASH;
use kona_genesis::RollupConfig;
use kona_mpt::{TrieHinter, ordered_trie_with_encoder};
use kona_protocol::{OutputRoot, Predeploys};
use op_alloy_consensus::OpReceiptEnvelope;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use revm::{context::BlockEnv, database::BundleState};

impl<P, H, Evm> StatelessL2Builder<'_, P, H, Evm>
where
    P: TrieDBProvider,
    H: TrieHinter,
    Evm: EvmFactory,
{
    /// Seals the block executed from the given [OpPayloadAttributes] and [BlockEnv], returning the
    /// computed [Header].
    pub(crate) fn seal_block(
        &mut self,
        attrs: &OpPayloadAttributes,
        parent_hash: B256,
        block_env: &BlockEnv,
        ex_result: &BlockExecutionResult<OpReceiptEnvelope>,
        bundle: BundleState,
    ) -> ExecutorResult<Sealed<Header>> {
        let timestamp = block_env.timestamp.saturating_to::<u64>();

        // Compute the roots for the block header.
        let state_root = self.trie_db.state_root(&bundle)?;
        let transactions_root = ordered_trie_with_encoder(
            // SAFETY: The OP Stack protocol will never generate a payload attributes with an empty
            // transactions field. Panicking here is the desired behavior, as it indicates a severe
            // protocol violation.
            attrs.transactions.as_ref().expect("Transactions must be non-empty"),
            |tx, buf| buf.put_slice(tx.as_ref()),
        )
        .root();
        let receipts_root = compute_receipts_root(&ex_result.receipts, self.config, timestamp);
        let withdrawals_root = if self.config.is_isthmus_active(timestamp) {
            Some(self.message_passer_account(block_env.number.saturating_to::<u64>())?)
        } else if self.config.is_canyon_active(timestamp) {
            Some(EMPTY_ROOT_HASH)
        } else {
            None
        };

        // Compute the logs bloom from the receipts generated during block execution.
        let logs_bloom = logs_bloom(ex_result.receipts.iter().flat_map(|r| r.logs()));

        // Compute Cancun fields, if active.
        let (blob_gas_used, excess_blob_gas) = if self.config.is_jovian_active(timestamp) {
            (Some(ex_result.blob_gas_used), Some(0))
        } else if self.config.is_ecotone_active(timestamp) {
            (Some(0), Some(0))
        } else {
            Default::default()
        };

        // At holocene activation, the base fee parameters from the payload are placed
        // into the Header's `extra_data` field.
        //
        // If the payload's `eip_1559_params` are equal to `0`, then the header's `extraData`
        // field is set to the encoded canyon base fee parameters.
        let encoded_base_fee_params = match self.config {
            config if config.is_jovian_active(timestamp) => {
                let extra_data = encode_jovian_eip_1559_params(self.config, attrs)?;
                Ok(extra_data)
            }
            config if config.is_holocene_active(timestamp) => {
                encode_holocene_eip_1559_params(self.config, attrs)
            }
            _ => Ok(Default::default()),
        }?;

        // The requests hash on the OP Stack, if Isthmus is active, is always the empty SHA256 hash.
        let requests_hash = self.config.is_isthmus_active(timestamp).then_some(EMPTY_REQUESTS_HASH);

        // Construct the new header.
        let header = Header {
            parent_hash,
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            beneficiary: attrs.payload_attributes.suggested_fee_recipient,
            state_root,
            transactions_root,
            receipts_root,
            withdrawals_root,
            requests_hash,
            logs_bloom,
            difficulty: U256::ZERO,
            number: block_env.number.saturating_to::<u64>(),
            gas_limit: attrs.gas_limit.ok_or(ExecutorError::MissingGasLimit)?,
            gas_used: ex_result.gas_used,
            timestamp,
            mix_hash: attrs.payload_attributes.prev_randao,
            nonce: Default::default(),
            base_fee_per_gas: Some(block_env.basefee),
            blob_gas_used,
            excess_blob_gas: excess_blob_gas.and_then(|x| x.try_into().ok()),
            parent_beacon_block_root: attrs.payload_attributes.parent_beacon_block_root,
            extra_data: encoded_base_fee_params,
        }
        .seal_slow();

        Ok(header)
    }

    /// Computes the current output root of the latest executed block, based on the parent header
    /// and the underlying state trie.
    ///
    /// **CONSTRUCTION:**
    /// ```text
    /// output_root = keccak256(version_byte .. payload)
    /// payload = state_root .. withdrawal_storage_root .. latest_block_hash
    /// ```
    pub fn compute_output_root(&mut self) -> ExecutorResult<B256> {
        let parent_number = self.trie_db.parent_block_header().number;

        info!(
            target: "block_builder",
            parent_state_root = ?self.trie_db.parent_block_header().state_root,
            parent_block_number = parent_number,
            "Computing output root",
        );

        let storage_root = self.message_passer_account(parent_number)?;
        let parent_header = self.trie_db.parent_block_header();

        // Construct the raw output and hash it.
        let output_root_hash =
            OutputRoot::from_parts(parent_header.state_root, storage_root, parent_header.seal())
                .hash();

        info!(
            target: "block_builder",
            parent_block_number = parent_number,
            output_root = ?output_root_hash,
            "Computed output root",
        );

        // Hash the output and return
        Ok(output_root_hash)
    }

    /// Fetches the L2 to L1 message passer account from the cache or underlying trie.
    fn message_passer_account(&mut self, block_number: u64) -> Result<B256, TrieDBError> {
        match self.trie_db.storage_roots().get(&Predeploys::L2_TO_L1_MESSAGE_PASSER) {
            Some(storage_root) => Ok(storage_root.blind()),
            None => Ok(self
                .trie_db
                .get_trie_account(&Predeploys::L2_TO_L1_MESSAGE_PASSER, block_number)?
                .ok_or(TrieDBError::MissingAccountInfo)?
                .storage_root),
        }
    }
}

/// Computes the receipts root from the given set of receipts.
pub fn compute_receipts_root(
    receipts: &[OpReceiptEnvelope],
    config: &RollupConfig,
    timestamp: u64,
) -> B256 {
    // There is a minor bug in op-geth and op-erigon where in the Regolith hardfork,
    // the receipt root calculation does not include the deposit nonce in the
    // receipt encoding. In the Regolith hardfork, we must strip the deposit nonce
    // from the receipt encoding to match the receipt root calculation.
    if config.is_regolith_active(timestamp) && !config.is_canyon_active(timestamp) {
        let receipts = receipts
            .iter()
            .cloned()
            .map(|receipt| match receipt {
                OpReceiptEnvelope::Deposit(mut deposit_receipt) => {
                    deposit_receipt.receipt.deposit_nonce = None;
                    OpReceiptEnvelope::Deposit(deposit_receipt)
                }
                _ => receipt,
            })
            .collect::<Vec<_>>();

        ordered_trie_with_encoder(receipts.as_ref(), |receipt, mut buf| {
            receipt.encode_2718(&mut buf)
        })
        .root()
    } else {
        ordered_trie_with_encoder(receipts, |receipt, mut buf| receipt.encode_2718(&mut buf)).root()
    }
}
