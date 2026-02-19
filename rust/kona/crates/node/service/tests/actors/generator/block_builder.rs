use std::time::SystemTime;

use alloy_consensus::{Block, EMPTY_OMMER_ROOT_HASH};
use alloy_eips::Encodable2718;
use alloy_primitives::Bytes;
use arbitrary::{Arbitrary, Unstructured};
use libp2p::bytes::BufMut;
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types_engine::{OpExecutionPayload, OpExecutionPayloadEnvelope};

use crate::actors::generator::seed::SeedGenerator;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum PayloadVersion {
    V1,
    #[allow(dead_code)]
    V2,
    #[allow(dead_code)]
    V3,
    #[allow(dead_code)]
    V4,
}

impl SeedGenerator {
    /// Generate a random op execution payload.
    pub(crate) fn random_valid_payload(
        &mut self,
        version: PayloadVersion,
    ) -> anyhow::Result<OpExecutionPayloadEnvelope> {
        let block: Block<OpTxEnvelope> = match version {
            PayloadVersion::V1 => self.v1_valid_block(),
            PayloadVersion::V2 => self.v2_valid_block(),
            PayloadVersion::V3 => self.v3_valid_block(),
            PayloadVersion::V4 => self.v4_valid_block(),
        };

        let (payload, _) = OpExecutionPayload::from_block_slow(&block);

        let parent_beacon_block_root = block.header.parent_beacon_block_root;

        let envelope =
            OpExecutionPayloadEnvelope { parent_beacon_block_root, execution_payload: payload };

        Ok(envelope)
    }

    fn valid_block(&mut self) -> Block<OpTxEnvelope> {
        // Simulate some random data
        let data = self.random_bytes(1024 * 1024);

        // Create unstructured data with the random bytes
        let u = Unstructured::new(&data);

        // Generate a random instance of MyStruct
        let mut block: Block<OpTxEnvelope> = Block::arbitrary_take_rest(u).unwrap();

        let transactions: Vec<Bytes> =
            block.body.transactions().map(|tx| tx.encoded_2718().into()).collect();

        let transactions_root =
            alloy_consensus::proofs::ordered_trie_root_with_encoder(&transactions, |item, buf| {
                buf.put_slice(item)
            });

        block.header.transactions_root = transactions_root;

        // We always need to set the base fee per gas to a positive value to ensure the block is
        // valid.
        block.header.base_fee_per_gas =
            Some(block.header.base_fee_per_gas.unwrap_or_default().saturating_add(1));

        let current_timestamp =
            SystemTime::now().duration_since(SystemTime::UNIX_EPOCH).unwrap().as_secs();
        block.header.timestamp = current_timestamp;

        block
    }

    /// Make the block v1 compatible
    fn v1_valid_block(&mut self) -> Block<OpTxEnvelope> {
        let mut block = self.valid_block();
        block.header.withdrawals_root = None;
        block.header.blob_gas_used = None;
        block.header.excess_blob_gas = None;
        block.header.parent_beacon_block_root = None;
        block.header.requests_hash = None;
        block.header.ommers_hash = EMPTY_OMMER_ROOT_HASH;
        block.header.difficulty = Default::default();
        block.header.nonce = Default::default();

        block
    }

    /// Make the block v2 compatible
    pub(crate) fn v2_valid_block(&mut self) -> Block<OpTxEnvelope> {
        let mut block = self.v1_valid_block();

        block.body.withdrawals = Some(vec![].into());
        let withdrawals_root = alloy_consensus::proofs::calculate_withdrawals_root(
            &block.body.withdrawals.clone().unwrap_or_default(),
        );

        block.header.withdrawals_root = Some(withdrawals_root);

        block
    }

    /// Make the block v3 compatible
    pub(crate) fn v3_valid_block(&mut self) -> Block<OpTxEnvelope> {
        let mut block = self.valid_block();

        block.body.withdrawals = Some(vec![].into());
        let withdrawals_root = alloy_consensus::proofs::calculate_withdrawals_root(
            &block.body.withdrawals.clone().unwrap_or_default(),
        );
        block.header.withdrawals_root = Some(withdrawals_root);

        block.header.blob_gas_used = Some(0);
        block.header.excess_blob_gas = Some(0);
        block.header.parent_beacon_block_root =
            Some(block.header.parent_beacon_block_root.unwrap_or_default());

        block.header.requests_hash = None;
        block.header.ommers_hash = EMPTY_OMMER_ROOT_HASH;
        block.header.difficulty = Default::default();
        block.header.nonce = Default::default();

        block
    }

    /// Make the block v4 compatible
    pub(crate) fn v4_valid_block(&mut self) -> Block<OpTxEnvelope> {
        self.v3_valid_block()
    }
}
