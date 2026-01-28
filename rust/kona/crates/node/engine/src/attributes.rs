//! Contains a utility method to check if attributes match a block.

use alloy_eips::{Decodable2718, eip1559::BaseFeeParams};
use alloy_network::TransactionResponse;
use alloy_primitives::{Address, B256, Bytes};
use alloy_rpc_types_eth::{Block, BlockTransactions, Withdrawals};
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use op_alloy_consensus::{
    EIP1559ParamError, OpTxEnvelope, decode_holocene_extra_data, decode_jovian_extra_data,
};
use op_alloy_rpc_types::Transaction;

/// Result of validating payload attributes against an execution layer block.
///
/// Used to verify that proposed payload attributes match the actual executed block,
/// ensuring consistency between the rollup derivation process and execution layer.
/// Validation includes withdrawals, transactions, fees, and other block properties.
///
/// # Examples
///
/// ```rust,ignore
/// use kona_engine::AttributesMatch;
/// use kona_genesis::RollupConfig;
/// use kona_protocol::OpAttributesWithParent;
///
/// let config = RollupConfig::default();
/// let match_result = AttributesMatch::check_withdrawals(&config, &attributes, &block);
///
/// if match_result.is_match() {
///     println!("Attributes are valid for this block");
/// }
/// ```
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AttributesMatch {
    /// The payload attributes are consistent with the block.
    Match,
    /// The attributes do not match the block (contains mismatch details).
    Mismatch(AttributesMismatch),
}

impl AttributesMatch {
    /// Returns true if the attributes match the block.
    pub const fn is_match(&self) -> bool {
        matches!(self, Self::Match)
    }

    /// Returns true if the attributes do not match the block.
    pub const fn is_mismatch(&self) -> bool {
        matches!(self, Self::Mismatch(_))
    }

    /// Checks that withdrawals for a block and attributes match.
    pub fn check_withdrawals(
        config: &RollupConfig,
        attributes: &OpAttributesWithParent,
        block: &Block<Transaction>,
    ) -> Self {
        let attr_withdrawals = attributes.attributes().payload_attributes.withdrawals.as_ref();
        let attr_withdrawals = attr_withdrawals.map(|w| Withdrawals::new(w.to_vec()));
        let block_withdrawals = block.withdrawals.as_ref();

        if config.is_canyon_active(block.header.timestamp) {
            // In canyon, the withdrawals list should be some and empty
            if attr_withdrawals.is_none_or(|w| !w.is_empty()) {
                return Self::Mismatch(AttributesMismatch::CanyonWithdrawalsNotEmpty);
            }
            if block_withdrawals.is_none_or(|w| !w.is_empty()) {
                return Self::Mismatch(AttributesMismatch::CanyonWithdrawalsNotEmpty);
            }
            if !config.is_isthmus_active(block.header.timestamp) {
                // In canyon, the withdrawals root should be set to the empty value
                let empty_hash = alloy_consensus::EMPTY_ROOT_HASH;
                if block.header.inner.withdrawals_root != Some(empty_hash) {
                    return Self::Mismatch(AttributesMismatch::CanyonNotEmptyHash);
                }
            }
        } else {
            // In bedrock, the withdrawals list should be None
            if attr_withdrawals.is_some() {
                return Self::Mismatch(AttributesMismatch::BedrockWithdrawals);
            }
        }

        if config.is_isthmus_active(block.header.timestamp) {
            // In isthmus, the withdrawals root must be set
            if block.header.inner.withdrawals_root.is_none() {
                return Self::Mismatch(AttributesMismatch::IsthmusMissingWithdrawalsRoot);
            }
        }

        Self::Match
    }

    /// Checks the attributes and block transaction list for consolidation.
    /// We start by checking that there are the same number of transactions in both the attribute
    /// payload and the block. Then we compare their contents
    fn check_transactions(attributes_txs: &[Bytes], block: &Block<Transaction>) -> Self {
        // Before checking the number of transactions, we have to make sure that the block
        // has the right transactions format. We need to have access to the
        // full transactions to be able to compare their contents.
        let block_txs = match block.transactions {
            BlockTransactions::Hashes(_) | BlockTransactions::Full(_)
                if attributes_txs.is_empty() && block.transactions.is_empty() =>
            {
                // We early return when both attributes and blocks are empty. This is for ergonomics
                // because the default [`BlockTransactions`] format is
                // [`BlockTransactions::Hash`], which may cause
                // the [`BlockTransactions`] format check to fail right below. We may want to be a
                // bit more flexible and not reject the hash format if both the
                // attributes and the block are empty.
                return Self::Match;
            }
            BlockTransactions::Uncle => {
                // This can never be uncle transactions
                error!(
                    "Invalid format for the block transactions. The `Uncle` transaction format is not relevant in that context and should not get used here. This is a bug"
                );

                return AttributesMismatch::MalformedBlockTransactions.into();
            }
            BlockTransactions::Hashes(_) => {
                // We can't have hash transactions with non empty blocks
                error!(
                    "Invalid format for the block transactions. The `Hash` transaction format is not relevant in that context and should not get used here. This is a bug."
                );

                return AttributesMismatch::MalformedBlockTransactions.into();
            }
            BlockTransactions::Full(ref block_txs) => block_txs,
        };

        let attributes_txs_len = attributes_txs.len();
        let block_txs_len = block_txs.len();

        if attributes_txs_len != block_txs_len {
            return AttributesMismatch::TransactionLen(attributes_txs_len, block_txs_len).into();
        }

        // Then we need to check that the content of the encoded transactions match
        // Note that it is safe to zip both iterators because we checked their length
        // beforehand.
        for (attr_tx_bytes, block_tx) in attributes_txs.iter().zip(block_txs) {
            trace!(
                target: "engine",
                ?attr_tx_bytes,
                block_tx_hash = %block_tx.tx_hash(),
                "Checking attributes transaction against block transaction",
            );
            // Let's try to deserialize the attributes transaction
            let Ok(attr_tx) = OpTxEnvelope::decode_2718(&mut &attr_tx_bytes[..]) else {
                error!(
                    "Impossible to deserialize transaction from attributes. If we have stored these attributes it means the transactions where well formatted. This is a bug"
                );

                return AttributesMismatch::MalformedAttributesTransaction.into();
            };

            if &attr_tx != block_tx.inner.inner.inner() {
                warn!(target: "engine", ?attr_tx, ?block_tx, "Transaction mismatch in derived attributes");
                return AttributesMismatch::TransactionContent(
                    attr_tx.tx_hash(),
                    block_tx.tx_hash(),
                )
                .into();
            }
        }

        Self::Match
    }

    /// Validates and compares EIP1559 parameters for consolidation.
    fn check_eip1559(
        config: &RollupConfig,
        attributes: &OpAttributesWithParent,
        block: &Block<Transaction>,
    ) -> Self {
        // We can assume that the EIP-1559 params are set iff holocene is active.
        // Note here that we don't need to check for the attributes length because of type-safety.
        let (ae, ad): (u128, u128) = match attributes.attributes().decode_eip_1559_params() {
            None => {
                // Holocene is active but the eip1559 are not set. This is a bug!
                // Note: we checked the timestamp match above, so we can assume that both the
                // attributes and the block have the same stamps
                if config.is_holocene_active(block.header.timestamp) {
                    error!(
                        "EIP1559 parameters for attributes not set while holocene is active. This is a bug"
                    );
                    return AttributesMismatch::MissingAttributesEIP1559.into();
                }

                // If the attributes are not specified, that means we can just early return.
                return Self::Match;
            }
            Some((0, e)) if e != 0 => {
                error!(
                    "Holocene EIP1559 params cannot have a 0 denominator unless elasticity is also 0. This is a bug"
                );
                return AttributesMismatch::InvalidEIP1559ParamsCombination.into();
            }
            // We need to translate (0, 0) parameters to pre-holocene protocol constants.
            // Since holocene is supposed to be active, canyon should be as well. We take the canyon
            // base fee params.
            Some((0, 0)) => {
                let BaseFeeParams { max_change_denominator, elasticity_multiplier } =
                    config.chain_op_config.post_canyon_params();

                (elasticity_multiplier, max_change_denominator)
            }
            Some((ae, ad)) => (ae.into(), ad.into()),
        };

        let extra_data_decoded = if config.is_jovian_active(block.header.timestamp) {
            decode_jovian_extra_data(&block.header.extra_data).map(|(be, bd, _)| (be, bd))
        } else if config.is_holocene_active(block.header.timestamp) {
            decode_holocene_extra_data(&block.header.extra_data)
        } else {
            return AttributesMismatch::MissingBlockEIP1559.into();
        };

        // We decode the extra data stemming from the block header.
        let (be, bd): (u128, u128) = match extra_data_decoded {
            Ok((be, bd)) => (be.into(), bd.into()),
            Err(EIP1559ParamError::NoEIP1559Params) => {
                error!(
                    "EIP1559 parameters for the block not set while holocene is active. This is a bug"
                );
                return AttributesMismatch::MissingBlockEIP1559.into();
            }
            Err(EIP1559ParamError::InvalidVersion(v)) => {
                error!(
                    version = v,
                    "The version in the extra data EIP1559 payload is incorrect. Should be 0. This is a bug",
                );
                return AttributesMismatch::InvalidExtraDataVersion.into();
            }
            Err(e) => {
                error!(err = ?e, "An unknown extra data decoding error occurred. This is a bug",);

                return AttributesMismatch::UnknownExtraDataDecodingError(e).into();
            }
        };

        // We now have to check that both parameters match
        if ae != be || ad != bd {
            return AttributesMismatch::EIP1559Parameters(
                BaseFeeParams { max_change_denominator: ad, elasticity_multiplier: ae },
                BaseFeeParams { max_change_denominator: bd, elasticity_multiplier: be },
            )
            .into();
        }

        Self::Match
    }

    /// Checks if the specified [`OpAttributesWithParent`] matches the specified [`Block`].
    /// Returns [`AttributesMatch::Match`] if they match, otherwise returns
    /// [`AttributesMatch::Mismatch`].
    pub fn check(
        config: &RollupConfig,
        attributes: &OpAttributesWithParent,
        block: &Block<Transaction>,
    ) -> Self {
        if attributes.parent.block_info.hash != block.header.inner.parent_hash {
            return AttributesMismatch::ParentHash(
                attributes.parent.block_info.hash,
                block.header.inner.parent_hash,
            )
            .into();
        }

        if attributes.attributes().payload_attributes.timestamp != block.header.inner.timestamp {
            return AttributesMismatch::Timestamp(
                attributes.attributes().payload_attributes.timestamp,
                block.header.inner.timestamp,
            )
            .into();
        }

        let mix_hash = block.header.inner.mix_hash;
        if attributes.attributes().payload_attributes.prev_randao != mix_hash {
            return AttributesMismatch::PrevRandao(
                attributes.attributes().payload_attributes.prev_randao,
                mix_hash,
            )
            .into();
        }

        // Let's extract the list of attribute transactions
        let default_vec = vec![];
        let attributes_txs = attributes
            .attributes()
            .transactions
            .as_ref()
            .map_or_else(|| &default_vec, |attrs| attrs);

        // Check transactions
        if let mismatch @ Self::Mismatch(_) = Self::check_transactions(attributes_txs, block) {
            return mismatch;
        }

        let Some(gas_limit) = attributes.attributes().gas_limit else {
            return AttributesMismatch::MissingAttributesGasLimit.into();
        };

        if gas_limit != block.header.inner.gas_limit {
            return AttributesMismatch::GasLimit(gas_limit, block.header.inner.gas_limit).into();
        }

        if let m @ Self::Mismatch(_) = Self::check_withdrawals(config, attributes, block) {
            return m;
        }

        if attributes.attributes().payload_attributes.parent_beacon_block_root !=
            block.header.inner.parent_beacon_block_root
        {
            return AttributesMismatch::ParentBeaconBlockRoot(
                attributes.attributes().payload_attributes.parent_beacon_block_root,
                block.header.inner.parent_beacon_block_root,
            )
            .into();
        }

        if attributes.attributes().payload_attributes.suggested_fee_recipient !=
            block.header.inner.beneficiary
        {
            return AttributesMismatch::FeeRecipient(
                attributes.attributes().payload_attributes.suggested_fee_recipient,
                block.header.inner.beneficiary,
            )
            .into();
        }

        // Check the EIP-1559 parameters in a separate helper method
        if let m @ Self::Mismatch(_) = Self::check_eip1559(config, attributes, block) {
            return m;
        }

        Self::Match
    }
}

/// An enum over the type of mismatch between [`OpAttributesWithParent`]
/// and a [`Block`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AttributesMismatch {
    /// The parent hash of the block does not match the parent hash of the attributes.
    ParentHash(B256, B256),
    /// The timestamp of the block does not match the timestamp of the attributes.
    Timestamp(u64, u64),
    /// The prev randao of the block does not match the prev randao of the attributes.
    PrevRandao(B256, B256),
    /// The block contains malformed transactions. This is a bug - the transaction format
    /// should be checked before the consolidation step.
    MalformedBlockTransactions,
    /// There is a malformed transaction inside the attributes. This is a bug - the transaction
    /// format should be checked before the consolidation step.
    MalformedAttributesTransaction,
    /// A mismatch in the number of transactions contained in the attributes and the block.
    TransactionLen(usize, usize),
    /// A mismatch in the content of some transactions contained in the attributes and the block.
    TransactionContent(B256, B256),
    /// The EIP1559 payload for the [`OpAttributesWithParent`] is missing when holocene is active.
    MissingAttributesEIP1559,
    /// The EIP1559 payload for the block is missing when holocene is active.
    MissingBlockEIP1559,
    /// The version in the extra data EIP1559 payload is incorrect. Should be 0.
    InvalidExtraDataVersion,
    /// An unknown extra data decoding error occurred.
    UnknownExtraDataDecodingError(EIP1559ParamError),
    /// Holocene EIP1559 params cannot have a 0 denominator unless elasticity is also 0
    InvalidEIP1559ParamsCombination,
    /// The EIP1559 base fee parameters of the attributes and the block don't match
    EIP1559Parameters(BaseFeeParams, BaseFeeParams),
    /// Transactions mismatch.
    Transactions(u64, u64),
    /// The gas limit of the block does not match the gas limit of the attributes.
    GasLimit(u64, u64),
    /// The gas limit for the [`OpAttributesWithParent`] is missing.
    MissingAttributesGasLimit,
    /// The fee recipient of the block does not match the fee recipient of the attributes.
    FeeRecipient(Address, Address),
    /// A mismatch in the parent beacon block root.
    ParentBeaconBlockRoot(Option<B256>, Option<B256>),
    /// After the canyon hardfork, withdrawals cannot be empty.
    CanyonWithdrawalsNotEmpty,
    /// After the canyon hardfork, the withdrawals root must be the empty hash.
    CanyonNotEmptyHash,
    /// In the bedrock hardfork, the attributes must has empty withdrawals.
    BedrockWithdrawals,
    /// In the isthmus hardfork, the withdrawals root must be set.
    IsthmusMissingWithdrawalsRoot,
}

impl From<AttributesMismatch> for AttributesMatch {
    fn from(mismatch: AttributesMismatch) -> Self {
        Self::Mismatch(mismatch)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::AttributesMismatch::EIP1559Parameters;
    use alloy_consensus::EMPTY_ROOT_HASH;
    use alloy_primitives::{Bytes, FixedBytes, address, b256};
    use alloy_rpc_types_eth::BlockTransactions;
    use arbitrary::{Arbitrary, Unstructured};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use kona_registry::ROLLUP_CONFIGS;
    use op_alloy_consensus::encode_holocene_extra_data;
    use op_alloy_rpc_types_engine::OpPayloadAttributes;

    fn default_attributes() -> OpAttributesWithParent {
        OpAttributesWithParent {
            attributes: OpPayloadAttributes::default(),
            parent: L2BlockInfo::default(),
            derived_from: Some(BlockInfo::default()),
            is_last_in_span: true,
        }
    }

    fn default_rollup_config() -> &'static RollupConfig {
        let opm = 10;
        ROLLUP_CONFIGS.get(&opm).expect("default rollup config should exist")
    }

    #[test]
    fn test_attributes_match_parent_hash_mismatch() {
        let cfg = default_rollup_config();
        let attributes = default_attributes();
        let mut block = Block::<Transaction>::default();
        block.header.inner.parent_hash =
            b256!("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef");
        let check = AttributesMatch::check(cfg, &attributes, &block);
        let expected: AttributesMatch = AttributesMismatch::ParentHash(
            attributes.parent.block_info.hash,
            block.header.inner.parent_hash,
        )
        .into();
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_match_check_timestamp() {
        let cfg = default_rollup_config();
        let attributes = default_attributes();
        let mut block = Block::<Transaction>::default();
        block.header.inner.timestamp = 1234567890;
        let check = AttributesMatch::check(cfg, &attributes, &block);
        let expected: AttributesMatch = AttributesMismatch::Timestamp(
            attributes.attributes().payload_attributes.timestamp,
            block.header.inner.timestamp,
        )
        .into();
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_match_check_prev_randao() {
        let cfg = default_rollup_config();
        let attributes = default_attributes();
        let mut block = Block::<Transaction>::default();
        block.header.inner.mix_hash =
            b256!("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef");
        let check = AttributesMatch::check(cfg, &attributes, &block);
        let expected: AttributesMatch = AttributesMismatch::PrevRandao(
            attributes.attributes().payload_attributes.prev_randao,
            block.header.inner.mix_hash,
        )
        .into();
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_match_missing_gas_limit() {
        let cfg = default_rollup_config();
        let attributes = default_attributes();
        let mut block = Block::<Transaction>::default();
        block.header.inner.gas_limit = 123456;
        let check = AttributesMatch::check(cfg, &attributes, &block);
        let expected: AttributesMatch = AttributesMismatch::MissingAttributesGasLimit.into();
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_match_check_gas_limit() {
        let cfg = default_rollup_config();
        let mut attributes = default_attributes();
        attributes.attributes.gas_limit = Some(123457);
        let mut block = Block::<Transaction>::default();
        block.header.inner.gas_limit = 123456;
        let check = AttributesMatch::check(cfg, &attributes, &block);
        let expected: AttributesMatch = AttributesMismatch::GasLimit(
            attributes.attributes().gas_limit.unwrap_or_default(),
            block.header.inner.gas_limit,
        )
        .into();
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_match_check_parent_beacon_block_root() {
        let cfg = default_rollup_config();
        let mut attributes = default_attributes();
        attributes.attributes.gas_limit = Some(0);
        attributes.attributes.payload_attributes.parent_beacon_block_root =
            Some(b256!("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"));
        let block = Block::<Transaction>::default();
        let check = AttributesMatch::check(cfg, &attributes, &block);
        let expected: AttributesMatch = AttributesMismatch::ParentBeaconBlockRoot(
            attributes.attributes().payload_attributes.parent_beacon_block_root,
            block.header.inner.parent_beacon_block_root,
        )
        .into();
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_match_check_fee_recipient() {
        let cfg = default_rollup_config();
        let mut attributes = default_attributes();
        attributes.attributes.gas_limit = Some(0);
        let mut block = Block::<Transaction>::default();
        block.header.inner.beneficiary = address!("1234567890abcdef1234567890abcdef12345678");
        let check = AttributesMatch::check(cfg, &attributes, &block);
        let expected: AttributesMatch = AttributesMismatch::FeeRecipient(
            attributes.attributes().payload_attributes.suggested_fee_recipient,
            block.header.inner.beneficiary,
        )
        .into();
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    fn generate_txs(num_txs: usize) -> Vec<Transaction> {
        // Simulate some random data
        let mut data = vec![0; 1024];
        let mut rng = rand::rng();

        (0..num_txs)
            .map(|_| {
                rand::Rng::fill(&mut rng, &mut data[..]);

                // Create unstructured data with the random bytes
                let u = Unstructured::new(&data);

                // Generate a random instance of MyStruct
                Transaction::arbitrary_take_rest(u).expect("Impossible to generate arbitrary tx")
            })
            .collect()
    }

    fn test_transactions_match_helper() -> (OpAttributesWithParent, Block<Transaction>) {
        const NUM_TXS: usize = 10;

        let transactions = generate_txs(NUM_TXS);
        let mut attributes = default_attributes();
        attributes.attributes.gas_limit = Some(0);
        attributes.attributes.transactions = Some(
            transactions
                .iter()
                .map(|tx| {
                    use alloy_eips::Encodable2718;
                    let mut buf = vec![];
                    tx.inner.inner.inner().encode_2718(&mut buf);
                    Bytes::from(buf)
                })
                .collect::<Vec<_>>(),
        );

        let block = Block::<Transaction> {
            transactions: BlockTransactions::Full(transactions),
            ..Default::default()
        };

        (attributes, block)
    }

    #[test]
    fn test_attributes_match_check_transactions() {
        let cfg = default_rollup_config();
        let (attributes, block) = test_transactions_match_helper();
        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Match);
    }

    #[test]
    fn test_attributes_mismatch_check_transactions_len() {
        let cfg = default_rollup_config();
        let (mut attributes, block) = test_transactions_match_helper();
        attributes.attributes = OpPayloadAttributes {
            transactions: attributes.attributes.transactions.map(|mut txs| {
                txs.pop();
                txs
            }),
            ..attributes.attributes
        };

        let block_txs_len = block.transactions.len();

        let expected: AttributesMatch =
            AttributesMismatch::TransactionLen(block_txs_len - 1, block_txs_len).into();

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_mismatch_check_transaction_content() {
        let cfg = default_rollup_config();
        let (attributes, mut block) = test_transactions_match_helper();
        let BlockTransactions::Full(block_txs) = &mut block.transactions else {
            unreachable!("The helper should build a full list of transactions")
        };

        let first_tx = block_txs.last().unwrap().clone();
        let first_tx_hash = first_tx.tx_hash();

        // We set the last tx to be the same as the first transaction.
        // Since the transactions are generated randomly and there are more than one transaction,
        // there is a very high likelihood that any pair of transactions is distinct.
        let last_tx = block_txs.first_mut().unwrap();
        let last_tx_hash = last_tx.tx_hash();
        *last_tx = first_tx;

        let expected: AttributesMatch =
            AttributesMismatch::TransactionContent(last_tx_hash, first_tx_hash).into();

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    /// Checks the edge case where the attributes array is empty.
    #[test]
    fn test_attributes_mismatch_empty_tx_attributes() {
        let cfg = default_rollup_config();
        let (mut attributes, block) = test_transactions_match_helper();
        attributes.attributes = OpPayloadAttributes { transactions: None, ..attributes.attributes };

        let block_txs_len = block.transactions.len();

        let expected: AttributesMatch = AttributesMismatch::TransactionLen(0, block_txs_len).into();

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    /// Checks the edge case where the transactions contained in the block have the wrong
    /// format.
    #[test]
    fn test_block_transactions_wrong_format() {
        let cfg = default_rollup_config();
        let (attributes, mut block) = test_transactions_match_helper();
        block.transactions = BlockTransactions::Uncle;

        let expected: AttributesMatch = AttributesMismatch::MalformedBlockTransactions.into();

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    /// Checks the edge case where the transactions contained in the attributes have the wrong
    /// format.
    #[test]
    fn test_attributes_transactions_wrong_format() {
        let cfg = default_rollup_config();
        let (mut attributes, block) = test_transactions_match_helper();
        let txs = attributes.attributes.transactions.as_mut().unwrap();
        let first_tx_bytes = txs.first_mut().unwrap();
        *first_tx_bytes = Bytes::copy_from_slice(&[0, 1, 2]);

        let expected: AttributesMatch = AttributesMismatch::MalformedAttributesTransaction.into();

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, expected);
        assert!(check.is_mismatch());
    }

    // Test that the check pass if the transactions obtained from the attributes have the format
    // `Some(vec![])`, ie an empty vector inside a `Some` option.
    #[test]
    fn test_attributes_and_block_transactions_empty() {
        let cfg = default_rollup_config();
        let (mut attributes, mut block) = test_transactions_match_helper();

        attributes.attributes =
            OpPayloadAttributes { transactions: Some(vec![]), ..attributes.attributes };

        block.transactions = BlockTransactions::Full(vec![]);

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Match);

        // Edge case: if the block transactions and the payload attributes are empty, we can also
        // use the hash format (this is the default value of `BlockTransactions`).
        attributes.attributes = OpPayloadAttributes { transactions: None, ..attributes.attributes };
        block.transactions = BlockTransactions::Hashes(vec![]);

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Match);
    }

    // Edge case: if the payload attributes has the format `Some(vec![])`, we can still
    // use the hash format.
    #[test]
    fn test_attributes_and_block_transactions_empty_hash_format() {
        let cfg = default_rollup_config();
        let (mut attributes, mut block) = test_transactions_match_helper();

        attributes.attributes =
            OpPayloadAttributes { transactions: Some(vec![]), ..attributes.attributes };

        block.transactions = BlockTransactions::Hashes(vec![]);

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Match);
    }

    // Test that the check fails if the block format is incorrect and the attributes are empty
    #[test]
    fn test_attributes_empty_and_block_uncle() {
        let cfg = default_rollup_config();
        let (mut attributes, mut block) = test_transactions_match_helper();

        attributes.attributes =
            OpPayloadAttributes { transactions: Some(vec![]), ..attributes.attributes };

        block.transactions = BlockTransactions::Uncle;

        let expected: AttributesMatch = AttributesMismatch::MalformedBlockTransactions.into();

        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, expected);
    }

    fn eip1559_test_setup() -> (RollupConfig, OpAttributesWithParent, Block<Transaction>) {
        let mut cfg = default_rollup_config().clone();

        // We need to activate holocene to make sure it works! We set the activation time to zero to
        // make sure that it is activated by default.
        cfg.hardforks.holocene_time = Some(0);

        let mut attributes = default_attributes();
        attributes.attributes.gas_limit = Some(0);
        // For canyon and above we need to specify the withdrawals
        attributes.attributes.payload_attributes.withdrawals = Some(vec![]);

        // For canyon and above we also need to specify the withdrawal headers
        let block = Block {
            withdrawals: Some(Withdrawals(vec![])),
            header: alloy_rpc_types_eth::Header {
                inner: alloy_consensus::Header {
                    withdrawals_root: Some(EMPTY_ROOT_HASH),
                    ..Default::default()
                },
                ..Default::default()
            },
            ..Default::default()
        };

        (cfg, attributes, block)
    }

    /// Ensures that we have to set the EIP1559 parameters for holocene and above.
    #[test]
    fn test_eip1559_parameters_not_specified_holocene() {
        let (cfg, attributes, block) = eip1559_test_setup();

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Mismatch(AttributesMismatch::MissingAttributesEIP1559));
        assert!(check.is_mismatch());
    }

    /// Ensures that we have to set the EIP1559 parameters for holocene and above.
    #[test]
    fn test_eip1559_parameters_specified_attributes_but_not_block() {
        let (cfg, mut attributes, block) = eip1559_test_setup();

        attributes.attributes.eip_1559_params = Some(Default::default());

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(
            check,
            AttributesMatch::Mismatch(AttributesMismatch::UnknownExtraDataDecodingError(
                EIP1559ParamError::InvalidExtraDataLength
            ))
        );
        assert!(check.is_mismatch());
    }

    /// Check that, when the eip1559 params are specified and empty, the check fails because we
    /// fallback on canyon params for the attributes but not for the block (edge case).
    #[test]
    fn test_eip1559_parameters_specified_both_and_empty() {
        let (cfg, mut attributes, mut block) = eip1559_test_setup();

        attributes.attributes.eip_1559_params = Some(Default::default());
        block.header.extra_data = vec![0; 9].into();

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(
            check,
            AttributesMatch::Mismatch(EIP1559Parameters(
                BaseFeeParams { max_change_denominator: 250, elasticity_multiplier: 6 },
                BaseFeeParams { max_change_denominator: 0, elasticity_multiplier: 0 }
            ))
        );
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_eip1559_parameters_empty_for_attr_only() {
        let (cfg, mut attributes, mut block) = eip1559_test_setup();

        attributes.attributes.eip_1559_params = Some(Default::default());
        block.header.extra_data = encode_holocene_extra_data(
            Default::default(),
            BaseFeeParams { max_change_denominator: 250, elasticity_multiplier: 6 },
        )
        .unwrap();

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Match);
        assert!(check.is_match());
    }

    #[test]
    fn test_eip1559_parameters_custom_values_match() {
        let (cfg, mut attributes, mut block) = eip1559_test_setup();

        let eip1559_extra_params = encode_holocene_extra_data(
            Default::default(),
            BaseFeeParams { max_change_denominator: 100, elasticity_multiplier: 2 },
        )
        .unwrap();
        let eip1559_params: FixedBytes<8> =
            eip1559_extra_params.clone().split_off(1).as_ref().try_into().unwrap();

        attributes.attributes.eip_1559_params = Some(eip1559_params);
        block.header.extra_data = eip1559_extra_params;

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Match);
        assert!(check.is_match());
    }

    #[test]
    fn test_eip1559_parameters_custom_values_mismatch() {
        let (cfg, mut attributes, mut block) = eip1559_test_setup();

        let eip1559_extra_params = encode_holocene_extra_data(
            Default::default(),
            BaseFeeParams { max_change_denominator: 100, elasticity_multiplier: 2 },
        )
        .unwrap();

        let eip1559_params: FixedBytes<8> = encode_holocene_extra_data(
            Default::default(),
            BaseFeeParams { max_change_denominator: 99, elasticity_multiplier: 2 },
        )
        .unwrap()
        .split_off(1)
        .as_ref()
        .try_into()
        .unwrap();

        attributes.attributes.eip_1559_params = Some(eip1559_params);
        block.header.extra_data = eip1559_extra_params;

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(
            check,
            AttributesMatch::Mismatch(AttributesMismatch::EIP1559Parameters(
                BaseFeeParams { max_change_denominator: 99, elasticity_multiplier: 2 },
                BaseFeeParams { max_change_denominator: 100, elasticity_multiplier: 2 }
            ))
        );
        assert!(check.is_mismatch());
    }

    /// Edge case: if the elasticity multiplier is 0, the max change denominator cannot be 0 as well
    #[test]
    fn test_eip1559_parameters_combination_mismatch() {
        let (cfg, mut attributes, mut block) = eip1559_test_setup();

        let eip1559_extra_params = encode_holocene_extra_data(
            Default::default(),
            BaseFeeParams { max_change_denominator: 5, elasticity_multiplier: 0 },
        )
        .unwrap();
        let eip1559_params: FixedBytes<8> =
            eip1559_extra_params.clone().split_off(1).as_ref().try_into().unwrap();

        attributes.attributes.eip_1559_params = Some(eip1559_params);
        block.header.extra_data = eip1559_extra_params;

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(
            check,
            AttributesMatch::Mismatch(AttributesMismatch::InvalidEIP1559ParamsCombination)
        );
        assert!(check.is_mismatch());
    }

    /// Check that the version of the extra block data must be zero.
    #[test]
    fn test_eip1559_parameters_invalid_version() {
        let (cfg, mut attributes, mut block) = eip1559_test_setup();

        let eip1559_extra_params = encode_holocene_extra_data(
            Default::default(),
            BaseFeeParams { max_change_denominator: 100, elasticity_multiplier: 2 },
        )
        .unwrap();
        let eip1559_params: FixedBytes<8> =
            eip1559_extra_params.clone().split_off(1).as_ref().try_into().unwrap();

        let mut raw_extra_params_bytes = eip1559_extra_params.to_vec();
        raw_extra_params_bytes[0] = 10;

        attributes.attributes.eip_1559_params = Some(eip1559_params);
        block.header.extra_data = raw_extra_params_bytes.into();

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Mismatch(AttributesMismatch::InvalidExtraDataVersion));
        assert!(check.is_mismatch());
    }

    /// Try to encode jovian extra data with the holocene encoding function.
    #[test]
    fn test_eip1559_parameters_invalid_jovian_encoding() {
        let (mut cfg, mut attributes, mut block) = eip1559_test_setup();

        cfg.hardforks.jovian_time = Some(0);

        let eip1559_extra_params = encode_holocene_extra_data(
            Default::default(),
            BaseFeeParams { max_change_denominator: 100, elasticity_multiplier: 2 },
        )
        .unwrap();
        let eip1559_params: FixedBytes<8> =
            eip1559_extra_params.clone().split_off(1).as_ref().try_into().unwrap();

        let raw_extra_params_bytes = eip1559_extra_params.to_vec();

        attributes.attributes.eip_1559_params = Some(eip1559_params);
        block.header.extra_data = raw_extra_params_bytes.into();

        let check = AttributesMatch::check(&cfg, &attributes, &block);
        assert_eq!(
            check,
            AttributesMatch::Mismatch(AttributesMismatch::UnknownExtraDataDecodingError(
                EIP1559ParamError::InvalidExtraDataLength
            ))
        );
        assert!(check.is_mismatch());
    }

    /// The default parameters can't overflow the u32 byte representation of the base fee params!
    #[test]
    fn test_eip1559_default_param_cant_overflow() {
        let (mut cfg, mut attributes, mut block) = eip1559_test_setup();
        cfg.chain_op_config.eip1559_denominator_canyon = u64::MAX;
        cfg.chain_op_config.eip1559_elasticity = u64::MAX;

        attributes.attributes.eip_1559_params = Some(Default::default());
        block.header.extra_data = vec![0; 9].into();

        let check = AttributesMatch::check(&cfg, &attributes, &block);

        // Note that in this case we *always* have a mismatch because there isn't enough bytes in
        // the default representation of the extra params to represent a u128
        assert_eq!(
            check,
            AttributesMatch::Mismatch(EIP1559Parameters(
                BaseFeeParams {
                    max_change_denominator: u64::MAX as u128,
                    elasticity_multiplier: u64::MAX as u128
                },
                BaseFeeParams { max_change_denominator: 0, elasticity_multiplier: 0 }
            ))
        );
        assert!(check.is_mismatch());
    }

    #[test]
    fn test_attributes_match() {
        let cfg = default_rollup_config();
        let mut attributes = default_attributes();
        attributes.attributes.gas_limit = Some(0);
        let block = Block::<Transaction>::default();
        let check = AttributesMatch::check(cfg, &attributes, &block);
        assert_eq!(check, AttributesMatch::Match);
        assert!(check.is_match());
    }
}
