use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, b256};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// Builder for creating test OpAttributesWithParent instances with sensible defaults
#[derive(Debug)]
pub struct TestAttributesBuilder {
    timestamp: u64,
    prev_randao: B256,
    suggested_fee_recipient: alloy_primitives::Address,
    withdrawals: Option<Vec<alloy_eips::eip4895::Withdrawal>>,
    parent_beacon_block_root: Option<B256>,
    transactions: Option<Vec<alloy_primitives::Bytes>>,
    no_tx_pool: Option<bool>,
    gas_limit: Option<u64>,
    eip_1559_params: Option<alloy_primitives::B64>,
    min_base_fee: Option<u64>,
    parent: L2BlockInfo,
    derived_from: Option<BlockInfo>,
    is_last_in_span: bool,
}

impl TestAttributesBuilder {
    /// Creates a new builder with default values
    pub fn new() -> Self {
        let parent = L2BlockInfo {
            block_info: BlockInfo {
                number: 0,
                hash: b256!("1111111111111111111111111111111111111111111111111111111111111111"),
                parent_hash: B256::ZERO,
                timestamp: 1000,
            },
            l1_origin: BlockNumHash::default(),
            seq_num: 0,
        };

        Self {
            timestamp: 2000,
            prev_randao: b256!("2222222222222222222222222222222222222222222222222222222222222222"),
            suggested_fee_recipient: alloy_primitives::Address::ZERO,
            withdrawals: None,
            parent_beacon_block_root: Some(B256::ZERO),
            transactions: None,
            no_tx_pool: Some(false),
            gas_limit: Some(30_000_000),
            eip_1559_params: None,
            min_base_fee: None,
            parent,
            derived_from: None,
            is_last_in_span: false,
        }
    }

    /// Sets the timestamp
    pub const fn with_timestamp(mut self, timestamp: u64) -> Self {
        self.timestamp = timestamp;
        self
    }

    /// Sets the parent block
    pub const fn with_parent(mut self, parent: L2BlockInfo) -> Self {
        self.parent = parent;
        self
    }

    /// Sets the transactions
    #[allow(dead_code)]
    pub fn with_transactions(mut self, txs: Vec<alloy_primitives::Bytes>) -> Self {
        self.transactions = Some(txs);
        self
    }

    /// Sets the gas limit
    #[allow(dead_code)]
    pub const fn with_gas_limit(mut self, gas_limit: u64) -> Self {
        self.gas_limit = Some(gas_limit);
        self
    }

    /// Builds the OpAttributesWithParent
    pub fn build(self) -> OpAttributesWithParent {
        let attributes = OpPayloadAttributes {
            payload_attributes: alloy_rpc_types_engine::PayloadAttributes {
                timestamp: self.timestamp,
                prev_randao: self.prev_randao,
                suggested_fee_recipient: self.suggested_fee_recipient,
                withdrawals: self.withdrawals,
                parent_beacon_block_root: self.parent_beacon_block_root,
            },
            transactions: self.transactions,
            no_tx_pool: self.no_tx_pool,
            gas_limit: self.gas_limit,
            eip_1559_params: self.eip_1559_params,
            min_base_fee: self.min_base_fee,
        };

        OpAttributesWithParent::new(
            attributes,
            self.parent,
            self.derived_from,
            self.is_last_in_span,
        )
    }
}

impl Default for TestAttributesBuilder {
    fn default() -> Self {
        Self::new()
    }
}
