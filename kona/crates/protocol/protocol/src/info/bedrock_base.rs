use alloy_primitives::{Address, B256};
use ambassador::delegatable_trait;

#[derive(Debug, Clone, Hash, Eq, PartialEq, Default, Copy)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub(crate) struct L1BlockInfoBedrockBase {
    /// The current L1 origin block number
    pub number: u64,
    /// The current L1 origin block's timestamp
    pub time: u64,
    /// The current L1 origin block's basefee
    pub base_fee: u64,
    /// The current L1 origin block's hash
    pub block_hash: B256,
    /// The current sequence number
    pub sequence_number: u64,
    /// The address of the batch submitter
    pub batcher_address: Address,
}

/// Accessors for Bedrock fields that still are available in latest hardfork.
#[delegatable_trait]
pub trait L1BlockInfoBedrockBaseFields {
    /// The current L1 origin block number
    fn number(&self) -> u64;

    /// The current L1 origin block's timestamp
    fn time(&self) -> u64;

    /// The current L1 origin block's basefee
    fn base_fee(&self) -> u64;

    /// The current L1 origin block's hash
    fn block_hash(&self) -> B256;

    /// The current sequence number
    fn sequence_number(&self) -> u64;

    /// The address of the batch new_from_l1_base_feesubmitter
    fn batcher_address(&self) -> Address;
}

impl L1BlockInfoBedrockBaseFields for L1BlockInfoBedrockBase {
    fn number(&self) -> u64 {
        self.number
    }

    fn time(&self) -> u64 {
        self.time
    }

    fn base_fee(&self) -> u64 {
        self.base_fee
    }

    fn block_hash(&self) -> B256 {
        self.block_hash
    }

    fn sequence_number(&self) -> u64 {
        self.sequence_number
    }

    fn batcher_address(&self) -> Address {
        self.batcher_address
    }
}

impl L1BlockInfoBedrockBase {
    /// Construct from all values.
    #[allow(clippy::too_many_arguments)]
    pub(crate) const fn new(
        number: u64,
        time: u64,
        base_fee: u64,
        block_hash: B256,
        sequence_number: u64,
        batcher_address: Address,
    ) -> Self {
        Self { number, time, base_fee, block_hash, sequence_number, batcher_address }
    }
    /// Construct from default values and `base_fee`.
    pub(crate) fn new_from_base_fee(base_fee: u64) -> Self {
        Self { base_fee, ..Default::default() }
    }
    /// Construct from default values and `block_hash`.
    pub(crate) fn new_from_block_hash(block_hash: B256) -> Self {
        Self { block_hash, ..Default::default() }
    }
    /// Construct from default values and `sequence_number`.
    pub(crate) fn new_from_sequence_number(sequence_number: u64) -> Self {
        Self { sequence_number, ..Default::default() }
    }
    /// Construct from default values and `batcher_address`.
    pub(crate) fn new_from_batcher_address(batcher_address: Address) -> Self {
        Self { batcher_address, ..Default::default() }
    }
    /// Construct from default values, `number` and `block_hash`.
    pub(crate) fn new_from_number_and_block_hash(number: u64, block_hash: B256) -> Self {
        Self { number, block_hash, ..Default::default() }
    }
}
