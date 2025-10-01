use alloy_consensus::BlockHeader;
use alloy_eips::eip4895::Withdrawal;
use alloy_primitives::{bytes, Address, Bloom, Bytes, B256, U256};
use alloy_rpc_types_engine::PayloadId;
use derive_more::Deref;
use reth_node_api::NodePrimitives;
use reth_optimism_evm::OpNextBlockEnvAttributes;
use reth_optimism_primitives::OpReceipt;
use reth_rpc_eth_types::PendingBlock;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// Represents a Flashblock, a real-time block-like structure emitted by the Base L2 chain.
///
/// A Flashblock provides a snapshot of a block’s effects before finalization,
/// allowing faster insight into state transitions, balance changes, and logs.
/// It includes a diff of the block’s execution and associated metadata.
///
/// See: [Base Flashblocks Documentation](https://docs.base.org/chain/flashblocks)
#[derive(Default, Debug, Clone, Eq, PartialEq, Serialize, Deserialize)]
pub struct FlashBlock {
    /// The unique payload ID as assigned by the execution engine for this block.
    pub payload_id: PayloadId,
    /// A sequential index that identifies the order of this Flashblock.
    pub index: u64,
    /// A subset of block header fields.
    pub base: Option<ExecutionPayloadBaseV1>,
    /// The execution diff representing state transitions and transactions.
    pub diff: ExecutionPayloadFlashblockDeltaV1,
    /// Additional metadata about the block such as receipts and balances.
    pub metadata: Metadata,
}

impl FlashBlock {
    /// Returns the block number of this flashblock.
    pub const fn block_number(&self) -> u64 {
        self.metadata.block_number
    }

    /// Returns the first parent hash of this flashblock.
    pub fn parent_hash(&self) -> Option<B256> {
        Some(self.base.as_ref()?.parent_hash)
    }
}

/// A trait for decoding flashblocks from bytes.
pub trait FlashBlockDecoder: Send + 'static {
    /// Decodes `bytes` into a [`FlashBlock`].
    fn decode(&self, bytes: bytes::Bytes) -> eyre::Result<FlashBlock>;
}

/// Default implementation of the decoder.
impl FlashBlockDecoder for () {
    fn decode(&self, bytes: bytes::Bytes) -> eyre::Result<FlashBlock> {
        FlashBlock::decode(bytes)
    }
}

/// Provides metadata about the block that may be useful for indexing or analysis.
#[derive(Default, Debug, Clone, Eq, PartialEq, Serialize, Deserialize)]
pub struct Metadata {
    /// The number of the block in the L2 chain.
    pub block_number: u64,
    /// A map of addresses to their updated balances after the block execution.
    /// This represents balance changes due to transactions, rewards, or system transfers.
    pub new_account_balances: BTreeMap<Address, U256>,
    /// Execution receipts for all transactions in the block.
    /// Contains logs, gas usage, and other EVM-level metadata.
    pub receipts: BTreeMap<B256, OpReceipt>,
}

/// Represents the base configuration of an execution payload that remains constant
/// throughout block construction. This includes fundamental block properties like
/// parent hash, block number, and other header fields that are determined at
/// block creation and cannot be modified.
#[derive(Clone, Debug, Eq, PartialEq, Default, Deserialize, Serialize)]
pub struct ExecutionPayloadBaseV1 {
    /// Ecotone parent beacon block root
    pub parent_beacon_block_root: B256,
    /// The parent hash of the block.
    pub parent_hash: B256,
    /// The fee recipient of the block.
    pub fee_recipient: Address,
    /// The previous randao of the block.
    pub prev_randao: B256,
    /// The block number.
    #[serde(with = "alloy_serde::quantity")]
    pub block_number: u64,
    /// The gas limit of the block.
    #[serde(with = "alloy_serde::quantity")]
    pub gas_limit: u64,
    /// The timestamp of the block.
    #[serde(with = "alloy_serde::quantity")]
    pub timestamp: u64,
    /// The extra data of the block.
    pub extra_data: Bytes,
    /// The base fee per gas of the block.
    pub base_fee_per_gas: U256,
}

/// Represents the modified portions of an execution payload within a flashblock.
/// This structure contains only the fields that can be updated during block construction,
/// such as state root, receipts, logs, and new transactions. Other immutable block fields
/// like parent hash and block number are excluded since they remain constant throughout
/// the block's construction.
#[derive(Clone, Debug, Eq, PartialEq, Default, Deserialize, Serialize)]
pub struct ExecutionPayloadFlashblockDeltaV1 {
    /// The state root of the block.
    pub state_root: B256,
    /// The receipts root of the block.
    pub receipts_root: B256,
    /// The logs bloom of the block.
    pub logs_bloom: Bloom,
    /// The gas used of the block.
    #[serde(with = "alloy_serde::quantity")]
    pub gas_used: u64,
    /// The block hash of the block.
    pub block_hash: B256,
    /// The transactions of the block.
    pub transactions: Vec<Bytes>,
    /// Array of [`Withdrawal`] enabled with V2
    pub withdrawals: Vec<Withdrawal>,
    /// The withdrawals root of the block.
    pub withdrawals_root: B256,
}

impl From<ExecutionPayloadBaseV1> for OpNextBlockEnvAttributes {
    fn from(value: ExecutionPayloadBaseV1) -> Self {
        Self {
            timestamp: value.timestamp,
            suggested_fee_recipient: value.fee_recipient,
            prev_randao: value.prev_randao,
            gas_limit: value.gas_limit,
            parent_beacon_block_root: Some(value.parent_beacon_block_root),
            extra_data: value.extra_data,
        }
    }
}

/// The pending block built with all received Flashblocks alongside the metadata for the last added
/// Flashblock.
#[derive(Debug, Clone, Deref)]
pub struct PendingFlashBlock<N: NodePrimitives> {
    /// The complete pending block built out of all received Flashblocks.
    #[deref]
    pub pending: PendingBlock<N>,
    /// A sequential index that identifies the last Flashblock added to this block.
    pub last_flashblock_index: u64,
    /// The last Flashblock block hash,
    pub last_flashblock_hash: B256,
    /// Whether the [`PendingBlock`] has a properly computed stateroot.
    pub has_computed_state_root: bool,
}

impl<N: NodePrimitives> PendingFlashBlock<N> {
    /// Create new pending flashblock.
    pub const fn new(
        pending: PendingBlock<N>,
        last_flashblock_index: u64,
        last_flashblock_hash: B256,
        has_computed_state_root: bool,
    ) -> Self {
        Self { pending, last_flashblock_index, last_flashblock_hash, has_computed_state_root }
    }

    /// Returns the properly calculated state root for that block if it was computed.
    pub fn computed_state_root(&self) -> Option<B256> {
        self.has_computed_state_root.then_some(self.pending.block().state_root())
    }
}
